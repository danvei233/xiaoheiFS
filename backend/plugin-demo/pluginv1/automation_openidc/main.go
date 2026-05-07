package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"math/big"
	"os"
	"strings"
	"sync"
	"time"

	"xiaoheiplay/pkg/pluginsdk"
	pluginv1 "xiaoheiplay/plugin/v1"
)

// pluginLog 插件日志，写入当前目录下的 plugin.log 文件
var pluginLog *log.Logger

func initLogger() {
	f, err := os.OpenFile("plugin.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		// 如果无法写文件，回退到 stderr
		pluginLog = log.New(os.Stderr, "[openidc] ", log.LstdFlags)
		return
	}
	pluginLog = log.New(f, "[openidc] ", log.LstdFlags)
}

// ---- 配置 ----

type config struct {
	BaseURL    string `json:"base_url"`
	APIKey     string `json:"api_key"`
	HsName     string `json:"hs_name"`
	TimeoutSec int    `json:"timeout_sec"`
	Retry      int    `json:"retry"`
	DryRun     bool   `json:"dry_run"`
	// DefaultNicType 创建虚拟机时默认下发的网卡类型。
	// HostAgent 在"管理员/Token 登录"路径不会自动补网卡（只有普通用户+配额模式才会补），
	// 插件又固定走 API Token，所以若不显式下发 nic_all，VM 会没有任何网卡。
	// 取值：
	//   - "" 或 "nat"：默认分配 1 张 NAT 网卡（默认行为，兼容绝大多数内网机房）
	//   - "pub"：默认分配 1 张公网网卡（有独立公网 IP 池的机房使用）
	//   - "none"：不下发任何网卡（需要运维/用户后续手动添加）
	DefaultNicType string `json:"default_nic_type"`
}

// ---- ID 编解码 ----
//
// 财务系统的 instance_id / line_id 均为 int64。
// OpenIDCS 使用字符串 hs_name（主机名）和 vm_uuid（UUID）。
//
// 方案：
//   - line_id   = fnv64(hs_name)，同时在 idStore 中记录 line_id → hs_name
//   - instance_id = fnv64(hs_name + "/" + vm_uuid)，同时在 idStore 中记录 instance_id → "hs_name/vm_uuid"
//
// fnv64 碰撞概率极低（64位），且对同一字符串始终返回相同值，插件重启后仍然有效。
// idStore 作为缓存加速反向查找，缓存未命中时通过遍历 OpenIDCS API 恢复。

func fnv64(s string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	v := int64(h.Sum64())
	if v < 0 {
		v = -v // 保证正数，避免部分系统对负 ID 的处理问题
	}
	return v
}

// generateRandomPassword 生成指定长度的随机密码（字母+数字）
func generateRandomPassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			// 极端情况下回退到固定字符
			b[i] = charset[i%len(charset)]
			continue
		}
		b[i] = charset[n.Int64()]
	}
	return string(b)
}

// idStore 维护 int64 ID → 字符串 的双向映射缓存
type idStore struct {
	mu sync.RWMutex
	m  map[int64]string // id → raw string
}

func newIDStore() *idStore {
	return &idStore{m: make(map[int64]string)}
}

func (s *idStore) put(id int64, raw string) {
	s.mu.Lock()
	s.m[id] = raw
	s.mu.Unlock()
}

func (s *idStore) get(id int64) (string, bool) {
	s.mu.RLock()
	v, ok := s.m[id]
	s.mu.RUnlock()
	return v, ok
}

// lineID 计算 hs_name 对应的 line_id 并缓存
func (s *idStore) lineID(hsName string) int64 {
	id := fnv64(hsName)
	s.put(id, hsName)
	return id
}

// instanceID 计算 {hs_name}/{vm_uuid} 对应的 instance_id 并缓存
func (s *idStore) instanceID(hsName, vmUUID string) int64 {
	raw := hsName + "/" + vmUUID
	id := fnv64(raw)
	s.put(id, raw)
	return id
}

// resolveHsName 通过 line_id 反查 hs_name
func (s *idStore) resolveHsName(lineID int64) (string, bool) {
	return s.get(lineID)
}

// resolveInstance 通过 instance_id 反查 hs_name 和 vm_uuid
func (s *idStore) resolveInstance(instanceID int64) (hsName, vmUUID string, ok bool) {
	raw, found := s.get(instanceID)
	if !found {
		return "", "", false
	}
	parts := strings.SplitN(raw, "/", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// ---- CoreService ----

type coreServer struct {
	pluginv1.UnimplementedCoreServiceServer
	cfg       config
	instance  string
	ids       *idStore
	updatedAt time.Time
}

func (s *coreServer) GetManifest(_ context.Context, _ *pluginv1.Empty) (*pluginv1.Manifest, error) {
	return &pluginv1.Manifest{
		PluginId:    "openidc_default",
		Name:        "OpenIDCS Automation (Built-in)",
		Version:     "0.1.0",
		Description: "Built-in OpenIDCS-Client automation plugin (catalog + lifecycle + port_mapping + backup + catalog_writeback).",
		Automation: &pluginv1.AutomationCapability{
			Features: []pluginv1.AutomationFeature{
				pluginv1.AutomationFeature_AUTOMATION_FEATURE_CATALOG_SYNC,
				pluginv1.AutomationFeature_AUTOMATION_FEATURE_LIFECYCLE,
				pluginv1.AutomationFeature_AUTOMATION_FEATURE_PORT_MAPPING,
				pluginv1.AutomationFeature_AUTOMATION_FEATURE_BACKUP,
				pluginv1.AutomationFeature_AUTOMATION_FEATURE_SNAPSHOT,
				pluginv1.AutomationFeature_AUTOMATION_FEATURE_FIREWALL,
			},
			NotSupportedReasons: map[int32]string{},
			// 套餐支持反向写回 HostAgent（走 HTTP 直通，不依赖 gRPC proto 未生成符号）
			CatalogReadonly: false,
		},
	}, nil
}

func (s *coreServer) GetConfigSchema(_ context.Context, _ *pluginv1.Empty) (*pluginv1.ConfigSchema, error) {
	return &pluginv1.ConfigSchema{
		JsonSchema: `{
  "title": "OpenIDCS Automation",
  "type": "object",
  "properties": {
    "base_url": { "type": "string", "title": "Base URL", "description": "OpenIDCS-Client 服务地址，例如 http://192.168.1.100:1880" },
    "api_key": { "type": "string", "title": "API Key", "format": "password", "description": "OpenIDCS-Client 的 Bearer Token" },
    "hs_name": { "type": "string", "title": "默认主机名（hs_name）", "description": "指定该商品类型对应的 OpenIDCS 主机名，用于镜像同步。留空则使用所有主机。" },
    "timeout_sec": { "type": "integer", "title": "超时时间（秒）", "description": "HTTP 请求超时，会覆盖 CreateVM 等长耗时接口。建议 >= 180 以覆盖虚拟机创建（importdisk + resize + boot）最慢场景。", "default": 60, "minimum": 1, "maximum": 600 },
    "retry": { "type": "integer", "title": "重试次数", "default": 1, "minimum": 0, "maximum": 5 },
    "dry_run": { "type": "boolean", "title": "Dry Run（演练模式）", "default": false },
    "default_nic_type": {
      "type": "string",
      "title": "默认网卡类型",
      "description": "创建虚拟机时默认分配的网卡类型。HostAgent 在 Token 登录路径不会自动补网卡，必须由插件显式下发。nat=内网NAT网卡，pub=公网网卡，none=不分配（运维自行管理）。",
      "enum": ["nat", "pub", "none"],
      "default": "nat"
    }
  },
  "required": ["base_url","api_key"]
}`,
		UiSchema: `{
  "api_key": { "ui:widget": "password", "ui:help": "留空表示不修改（由宿主处理）" }
}`,
	}, nil
}

func (s *coreServer) ValidateConfig(_ context.Context, req *pluginv1.ValidateConfigRequest) (*pluginv1.ValidateConfigResponse, error) {
	var cfg config
	if err := json.Unmarshal([]byte(req.GetConfigJson()), &cfg); err != nil {
		return &pluginv1.ValidateConfigResponse{Ok: false, Error: "invalid json"}, nil
	}
	if strings.TrimSpace(cfg.BaseURL) == "" || strings.TrimSpace(cfg.APIKey) == "" {
		return &pluginv1.ValidateConfigResponse{Ok: false, Error: "base_url/api_key required"}, nil
	}
	if cfg.TimeoutSec < 0 || cfg.TimeoutSec > 600 {
		return &pluginv1.ValidateConfigResponse{Ok: false, Error: "timeout_sec out of range [0,600]"}, nil
	}
	if cfg.Retry < 0 || cfg.Retry > 5 {
		return &pluginv1.ValidateConfigResponse{Ok: false, Error: "retry out of range [0,5]"}, nil
	}
	switch strings.ToLower(strings.TrimSpace(cfg.DefaultNicType)) {
	case "", "nat", "pub", "none":
		// ok
	default:
		return &pluginv1.ValidateConfigResponse{Ok: false, Error: "default_nic_type must be one of: nat, pub, none"}, nil
	}
	return &pluginv1.ValidateConfigResponse{Ok: true}, nil
}

func (s *coreServer) Init(_ context.Context, req *pluginv1.InitRequest) (*pluginv1.InitResponse, error) {
	pluginLog.Printf("Init called, instance_id=%q, config_json=%s", req.GetInstanceId(), req.GetConfigJson())
	if strings.TrimSpace(req.GetConfigJson()) != "" {
		var cfg config
		if err := json.Unmarshal([]byte(req.GetConfigJson()), &cfg); err != nil {
			pluginLog.Printf("Init: unmarshal config failed: %v", err)
			return &pluginv1.InitResponse{Ok: false, Error: "invalid config"}, nil
		}
		s.cfg = cfg
		pluginLog.Printf("Init: config loaded, base_url=%q, api_key_len=%d, dry_run=%v",
			cfg.BaseURL, len(cfg.APIKey), cfg.DryRun)
	} else {
		pluginLog.Printf("Init: config_json is empty, using existing config")
	}
	s.instance = req.GetInstanceId()
	s.updatedAt = time.Now()
	return &pluginv1.InitResponse{Ok: true}, nil
}

func (s *coreServer) ReloadConfig(_ context.Context, req *pluginv1.ReloadConfigRequest) (*pluginv1.ReloadConfigResponse, error) {
	pluginLog.Printf("ReloadConfig called, config_json=%s", req.GetConfigJson())
	var cfg config
	if err := json.Unmarshal([]byte(req.GetConfigJson()), &cfg); err != nil {
		pluginLog.Printf("ReloadConfig: unmarshal failed: %v", err)
		return &pluginv1.ReloadConfigResponse{Ok: false, Error: "invalid config"}, nil
	}
	s.cfg = cfg
	s.updatedAt = time.Now()
	pluginLog.Printf("ReloadConfig: config reloaded, base_url=%q, api_key_len=%d, dry_run=%v",
		cfg.BaseURL, len(cfg.APIKey), cfg.DryRun)
	return &pluginv1.ReloadConfigResponse{Ok: true}, nil
}

func (s *coreServer) Health(_ context.Context, req *pluginv1.HealthCheckRequest) (*pluginv1.HealthCheckResponse, error) {
	status := pluginv1.HealthStatus_HEALTH_STATUS_OK
	msg := "ok"
	if req.GetInstanceId() == "" || s.instance == "" {
		status = pluginv1.HealthStatus_HEALTH_STATUS_ERROR
		msg = "not initialized"
	}
	return &pluginv1.HealthCheckResponse{
		Status:     status,
		Message:    msg,
		UnixMillis: time.Now().UnixMilli(),
	}, nil
}

// newClient 创建 HTTP 客户端
func (s *coreServer) newClient() (*Client, error) {
	pluginLog.Printf("newClient: base_url=%q, api_key_len=%d", s.cfg.BaseURL, len(s.cfg.APIKey))
	if strings.TrimSpace(s.cfg.BaseURL) == "" || strings.TrimSpace(s.cfg.APIKey) == "" {
		pluginLog.Printf("newClient: ERROR - base_url or api_key is empty!")
		return nil, fmt.Errorf("missing config: base_url/api_key required")
	}
	timeout := time.Duration(s.cfg.TimeoutSec) * time.Second
	if timeout <= 0 {
		// 兜底 1800s：覆盖备份/快照等长耗时操作。
		timeout = 1800 * time.Second
	}
	return NewClient(s.cfg.BaseURL, s.cfg.APIKey, timeout), nil
}

// newClientWithTrace 创建带日志追踪的 HTTP 客户端
func (s *coreServer) newClientWithTrace() (*Client, *HTTPLogEntry, error) {
	client, err := s.newClient()
	if err != nil {
		return nil, nil, err
	}
	var last HTTPLogEntry
	client.WithLogger(func(_ context.Context, entry HTTPLogEntry) {
		last = entry
	})
	return client, &last, nil
}

// retry 带重试的执行
func (s *coreServer) retry(fn func() error) error {
	maxRetry := s.cfg.Retry
	if maxRetry < 0 {
		maxRetry = 0
	}
	var err error
	for i := 0; i <= maxRetry; i++ {
		err = fn()
		if err == nil {
			return nil
		}
	}
	return err
}

// resolveInstanceWithFallback 通过 instance_id 反查 hs_name/vm_uuid
// 若缓存未命中，则遍历 OpenIDCS 所有主机恢复缓存
func (s *coreServer) resolveInstanceWithFallback(ctx context.Context, instanceID int64) (hsName, vmUUID string, err error) {
	// 先查缓存
	if hs, vm, ok := s.ids.resolveInstance(instanceID); ok {
		return hs, vm, nil
	}
	// 缓存未命中：遍历所有主机，重建缓存
	c, _, err := s.newClientWithTrace()
	if err != nil {
		return "", "", err
	}
	servers, err := c.ListServers(ctx)
	if err != nil {
		return "", "", fmt.Errorf("resolve instance_id %d: list servers failed: %w", instanceID, err)
	}
	var failedHosts []string
	for hs := range servers {
		s.ids.lineID(hs) // 顺便缓存 line_id
		vms, vmErr := c.ListVMs(ctx, hs)
		if vmErr != nil {
			failedHosts = append(failedHosts, fmt.Sprintf("%s(%v)", hs, vmErr))
			continue
		}
		for _, vm := range vms {
			id := s.ids.instanceID(hs, vm.VMUUID)
			if id == instanceID {
				return hs, vm.VMUUID, nil
			}
		}
	}
	if len(failedHosts) > 0 {
		return "", "", fmt.Errorf("instance_id %d not found in OpenIDCS (failed to list VMs on hosts: %s)", instanceID, strings.Join(failedHosts, ", "))
	}
	return "", "", fmt.Errorf("instance_id %d not found in OpenIDCS", instanceID)
}

// resolveLineWithFallback 通过 line_id 反查 hs_name
func (s *coreServer) resolveLineWithFallback(ctx context.Context, lineID int64) (string, error) {
	if hs, ok := s.ids.resolveHsName(lineID); ok {
		return hs, nil
	}
	// 缓存未命中：重新拉取主机列表
	c, _, err := s.newClientWithTrace()
	if err != nil {
		return "", err
	}
	servers, err := c.ListServers(ctx)
	if err != nil {
		return "", fmt.Errorf("resolve line_id %d: list servers failed: %w", lineID, err)
	}
	for hs := range servers {
		id := s.ids.lineID(hs)
		if id == lineID {
			return hs, nil
		}
	}
	return "", fmt.Errorf("line_id %d not found in OpenIDCS", lineID)
}

// wrapHTTPTraceErr 包装 HTTP 追踪错误信息
//
// 关键点：`last` 仅记录最近一次 HTTP 请求，err 可能来自**本次 HTTP 请求之后**
// 的解析/逻辑阶段（例如 JSON 解码、字段校验）。因此不能仅依赖上游 body.msg
// 构造错误描述——当上游响应本身是成功（success=true / msg ∈ {success, ok}）时，
// 必须退回到原始 err 的文本，否则用户会看到「rpc error desc = success」
// 这种自相矛盾的提示。
func wrapHTTPTraceErr(err error, last *HTTPLogEntry) error {
	if err == nil {
		return nil
	}
	if last == nil || strings.TrimSpace(last.Action) == "" {
		return err
	}
	trace := map[string]any{
		"action":   last.Action,
		"request":  last.Request,
		"response": last.Response,
		"success":  last.Success,
		"message":  last.Message,
	}
	// 当上游 HTTP 本身成功时，msg 代表的是成功提示（例如 "success" / "ok"），
	// 不能作为错误描述，直接用本地 err。
	var msg string
	if isHTTPTraceSuccess(trace) {
		msg = err.Error()
	} else {
		msg = extractTraceMessage(trace)
		if strings.TrimSpace(msg) == "" {
			msg = err.Error()
		}
	}
	raw, marshalErr := json.Marshal(trace)
	if marshalErr != nil {
		return fmt.Errorf("%s", msg)
	}
	return fmt.Errorf("%s | http_trace=%s", msg, string(raw))
}

// isHTTPTraceSuccess 判断上游 HTTP 是否本身就成功了
// 约定：trace.success=true 或 body_json.msg ∈ {success, ok}（忽略大小写）都视为成功
func isHTTPTraceSuccess(trace map[string]any) bool {
	if trace == nil {
		return false
	}
	if ok, isBool := trace["success"].(bool); isBool && ok {
		return true
	}
	if resp, ok := trace["response"].(map[string]any); ok {
		if bodyJSON, ok := resp["body_json"].(map[string]any); ok {
			if msg, ok := bodyJSON["msg"].(string); ok {
				m := strings.ToLower(strings.TrimSpace(msg))
				if m == "success" || m == "ok" {
					return true
				}
			}
			// code==200 也视为上游成功
			if code, ok := bodyJSON["code"]; ok {
				switch v := code.(type) {
				case float64:
					if int(v) == 200 {
						return true
					}
				case int:
					if v == 200 {
						return true
					}
				}
			}
		}
	}
	return false
}

func extractTraceMessage(trace map[string]any) string {
	if trace == nil {
		return ""
	}
	if resp, ok := trace["response"].(map[string]any); ok {
		if bodyJSON, ok := resp["body_json"].(map[string]any); ok {
			if msg, ok := bodyJSON["msg"].(string); ok && strings.TrimSpace(msg) != "" {
				return msg
			}
		}
	}
	if msg, ok := trace["message"].(string); ok {
		return strings.TrimSpace(msg)
	}
	return ""
}

// ---- AutomationService ----

type automationServer struct {
	pluginv1.UnimplementedAutomationServiceServer
	core *coreServer
}

// ---- 目录同步 ----

// ListAreas 地区列表（从 OpenIDCS server_area 字段获取）
func (a *automationServer) ListAreas(ctx context.Context, _ *pluginv1.Empty) (*pluginv1.ListAreasResponse, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	var areas []AreaInfo
	err = a.core.retry(func() error {
		var callErr error
		areas, callErr = c.ListAreas(ctx)
		return callErr
	})
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	out := make([]*pluginv1.AutomationArea, 0, len(areas))
	for _, area := range areas {
		// 缓存 area_id → "area/<code>"，后续按 code 反查上游 server_area
		a.core.ids.put(area.ID, "area/"+area.Code)
		out = append(out, &pluginv1.AutomationArea{
			Id:    area.ID,
			Name:  area.Name,
			State: int32(area.State),
		})
	}
	return &pluginv1.ListAreasResponse{Items: out}, nil
}

// ListLines 线路列表（映射 OpenIDCS 主机列表，每台主机 = 一条线路）
// line_id = fnv64(hs_name)
func (a *automationServer) ListLines(ctx context.Context, _ *pluginv1.Empty) (*pluginv1.ListLinesResponse, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	var servers map[string]ServerInfo
	err = a.core.retry(func() error {
		var callErr error
		servers, callErr = c.ListServers(ctx)
		return callErr
	})
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	out := make([]*pluginv1.AutomationLine, 0, len(servers))
	for name, info := range servers {
		lineID := a.core.ids.lineID(name) // 缓存 line_id → hs_name
		state := int32(1)
		if info.Status != "online" {
			state = 0
		}
		// 根据 HostAgent server_area 解析出 area_code，空值统一归并到 default 区域
		areaCode, _ := parseServerArea(info.ServerArea)
		areaID := areaIDFromCode(areaCode)
		out = append(out, &pluginv1.AutomationLine{
			Id:     lineID,
			Name:   name,
			AreaId: areaID,
			State:  state,
		})
	}
	return &pluginv1.ListLinesResponse{Items: out}, nil
}

// ListPackages 套餐列表
//
// 数据源说明：
//   HostAgent 当前只暴露 GET /api/server/detail，该接口已经返回完整 HSConfig
//   （含 server_plan：key=plan_name, value=VMConfig）。
//   因此这里直接复用 ListServers 的结果，在本地解包出套餐；无需依赖未实现的
//   /api/server/plans/{hs} 接口。将来 HostAgent 若单独提供该接口，只需把 detail
//   换成 ListPlans 即可平滑切换。
//
// 字段映射（VMConfig → AutomationPackage）：
//   cpu_num → Cpu
//   mem_num (MB) → MemoryGb（除 1024 取整）
//   hdd_num (MB) → DiskGb（除 1024 取整）
//   max(speed_u, speed_d) → BandwidthMbps
//   nat_num → PortNum（VM 分配端口数，对应小黑云套餐 port_num）
//   monthly_price 目前保持 0，由小黑云财务侧维护（上游 VMConfig 无价格字段）。
func (a *automationServer) ListPackages(ctx context.Context, req *pluginv1.ListPackagesRequest) (*pluginv1.ListPackagesResponse, error) {
	hsName, err := a.core.resolveLineWithFallback(ctx, req.GetLineId())
	if err != nil {
		// 如果 line_id 无法解析，返回空列表（兼容旧行为）
		return &pluginv1.ListPackagesResponse{Items: []*pluginv1.AutomationPackage{}}, nil
	}
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	var servers map[string]ServerInfo
	err = a.core.retry(func() error {
		var callErr error
		servers, callErr = c.ListServers(ctx)
		return callErr
	})
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	info, ok := servers[hsName]
	if !ok {
		// 目标主机不存在：返回空列表而非报错，避免把"一台主机掉线"放大为整个目录同步失败
		return &pluginv1.ListPackagesResponse{Items: []*pluginv1.AutomationPackage{}}, nil
	}
	plans := info.Config.ServerPlan
	out := make([]*pluginv1.AutomationPackage, 0, len(plans))
	for planName, p := range plans {
		// plan_id = fnv64(hs_name + "/plan/" + plan_name)
		planKey := hsName + "/plan/" + planName
		planID := fnv64(planKey)
		a.core.ids.put(planID, planKey)
		bw := p.SpeedU
		if p.SpeedD > bw {
			bw = p.SpeedD
		}
		out = append(out, &pluginv1.AutomationPackage{
			Id:            planID,
			Name:          planName,
			Cpu:           int32(p.CPUNum),
			MemoryGb:      int32(p.MemNum / 1024),
			DiskGb:        int32(p.HDDNum / 1024),
			BandwidthMbps: int32(bw),
			PortNum:       int32(p.NATNum),
			MonthlyPrice:  0, // 价格由小黑云财务系统管理，上游 VMConfig 不承载
		})
	}
	return &pluginv1.ListPackagesResponse{Items: out}, nil
}

// ListImages 镜像列表（映射 OpenIDCS OS 镜像接口）
// 若 config.HsName 已配置则只查该主机，否则查 line_id 对应主机
// line_id = fnv64(hs_name)，image_id = fnv64(hs_name + "/img/" + img.File)
func (a *automationServer) ListImages(ctx context.Context, req *pluginv1.ListImagesRequest) (*pluginv1.ListImagesResponse, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	// 优先使用 config 中配置的 hs_name，否则通过 line_id 反查
	hsName := strings.TrimSpace(a.core.cfg.HsName)
	if hsName == "" {
		hsName, err = a.core.resolveLineWithFallback(ctx, req.GetLineId())
		if err != nil {
			return nil, err
		}
	}
	var imageMap map[string][]OSImage
	err = a.core.retry(func() error {
		var callErr error
		imageMap, callErr = c.ListOSImages(ctx, hsName)
		return callErr
	})
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	// 【介质分类说明】HostAgent HSConfig 的两个字段承载的业务语义完全不同：
	//   - system_maps → OSImage{Category="system"}：系统磁盘模板 / 装机母盘
	//     * CreateVM 的 os_name 从这里取，"安装系统"唯一合法来源
	//     * 文件名通常是 .vmdk / .qcow2 / .img / .raw / .template 等磁盘格式
	//   - images_maps → OSImage{Category="iso"}：光驱 ISO 镜像
	//     * MountISO 的 iso_name 从这里取，仅用于运维（FirPE/WinPE 等工具盘）
	//     * 文件名基本是 .iso
	//
	// 过往事故：ListImages 把两类合并返回 → 小黑云前台"购买 VPS / 重装系统"
	// 的镜像下拉里混入 FirPE 等 ISO，用户误选后装机失败。
	//
	// 本次彻底修复（双重过滤，保证任何 HostAgent 配置形态下都不泄漏 ISO）：
	//   1) 主过滤：Category == "system" 才入选；
	//   2) 副过滤：Name / File 任一以 .iso 结尾（忽略大小写）一律剔除，
	//      兜住"运维把 ISO 错配到 system_maps 里"的人为失误；
	//   3) iso 分类同时仍写入 a.core.ids 缓存，保留 Rebuild / MountISO 通过
	//      已知 image_id 反查 sys_file 的向后兼容能力。
	//   4) 日志打点 raw 与 filtered 数量，方便部署后直接在 plugin.log 校验生效。
	systemImages := imageMap["system"]
	isoImages := imageMap["iso"]

	// ISO 分类：仅注入缓存（不对外返回），保留 Rebuild/MountISO 老流程
	for _, img := range isoImages {
		if strings.TrimSpace(img.File) == "" {
			continue
		}
		imageKey := hsName + "/img/" + img.File
		imageID := fnv64(imageKey)
		a.core.ids.put(imageID, imageKey)
	}

	out := make([]*pluginv1.AutomationImage, 0, len(systemImages))
	droppedByExt := 0
	for _, img := range systemImages {
		name := img.Name
		if name == "" {
			name = img.File
		}
		// 副过滤：若 system_maps 里混入 .iso 文件（通常是运维误配），也从结果里剔除。
		if hasISOExt(img.File) || hasISOExt(img.Name) {
			droppedByExt++
			// 仍注入缓存以便 Rebuild 兼容
			if strings.TrimSpace(img.File) != "" {
				imageKey := hsName + "/img/" + img.File
				a.core.ids.put(fnv64(imageKey), imageKey)
			}
			continue
		}
		// image_id = fnv64(hs_name + "/img/" + img.File)，缓存反查
		imageKey := hsName + "/img/" + img.File
		imageID := fnv64(imageKey)
		a.core.ids.put(imageID, imageKey)
		// Type 来自 HostAgent sys_type（WinNT/Linux/macOS），在 http_client 已归一化
		// Enabled 来自 HostAgent sys_flag（老版本无此字段时默认 true）
		out = append(out, &pluginv1.AutomationImage{
			Id:      imageID,
			Name:    name,
			Type:    img.Type,
			Enabled: img.Enabled,
		})
	}

	if pluginLog != nil {
		// 强版本戳 "[v2-iso-filter]" 方便部署后一眼辨识新二进制是否加载成功
		pluginLog.Printf("[v2-iso-filter] ListImages hs=%s raw=system:%d,iso:%d dropped_by_ext=%d returned=%d",
			hsName, len(systemImages), len(isoImages), droppedByExt, len(out))
	}
	return &pluginv1.ListImagesResponse{Items: out}, nil
}

// hasISOExt 判定文件名是否以 .iso 结尾（忽略大小写），用于兜底剔除
// 运维误配到 system_maps 的 ISO 文件。
func hasISOExt(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	return n != "" && strings.HasSuffix(n, ".iso")
}

// ---- 实例生命周期 ----

// CreateInstance 创建虚拟机
// line_id = fnv64(hs_name)，image_id = fnv64(hs_name + "/img/" + img.File)
// 返回 instance_id = fnv64(hs_name + "/" + vm_uuid)
func (a *automationServer) CreateInstance(ctx context.Context, req *pluginv1.CreateInstanceRequest) (*pluginv1.CreateInstanceResponse, error) {
	if a.core.cfg.DryRun {
		return &pluginv1.CreateInstanceResponse{InstanceId: fnv64(fmt.Sprintf("dry-run/%d", time.Now().UnixNano()))}, nil
	}
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	hsName, err := a.core.resolveLineWithFallback(ctx, req.GetLineId())
	if err != nil {
		return nil, fmt.Errorf("resolve line_id: %w", err)
	}
	// 解析 image_id → img.File
	isoName := ""
	if req.GetImageId() != 0 {
		if raw, ok := a.core.ids.get(req.GetImageId()); ok {
			// raw 格式：{hs_name}/img/{img.File}
			parts := strings.SplitN(raw, "/img/", 2)
			if len(parts) == 2 {
				isoName = parts[1]
			}
		}
	}
	body := map[string]any{
		// 不传 vm_uuid，让 HostAgent 根据主机配置的 filter_name 前缀 + 随机字符自动生成。
		// 这样 vm_uuid 格式为 "{filter_name}-{random8}"，与 HostAgent 面板一致。
		// vm_name 仅作为显示名传递，HostAgent 当前版本会忽略此字段。
		"vm_name": req.GetName(),
		"cpu_num": req.GetCpu(),
		"mem_num": req.GetMemoryGb() * 1024, // GB → MB
		"hdd_num": req.GetDiskGb() * 1024,   // GB → MB（HostAgent VMConfig.hdd_num 单位为 MB）
	}
	if isoName != "" {
		body["os_name"] = isoName
	} else if req.GetOs() != "" {
		body["os_name"] = req.GetOs()
	}
	if req.GetPassword() != "" {
		body["os_pass"] = req.GetPassword()
	}
	// VNC 密码：优先使用传入值，否则自动生成8位随机密码传入 HostAgent
	if req.GetVncPassword() != "" {
		body["vc_pass"] = req.GetVncPassword()
	} else {
		body["vc_pass"] = generateRandomPassword(8)
	}

	// 默认网卡配置：
	// HostAgent 的 RestManager.create_vm 只有在"普通用户+配额模式"下才会自动补一张默认网卡，
	// 我们走 API Token（is_token_login=true）路径，HostAgent 不会补，
	// 因此必须由插件显式下发 nic_all，否则创建出来的 VM 没有任何网卡，外界无法访问。
	// 由运维通过配置项 default_nic_type 选择走 NAT 还是公网（PUB），或显式关闭（none）。
	nicType := strings.ToLower(strings.TrimSpace(a.core.cfg.DefaultNicType))
	if nicType == "" {
		nicType = "nat" // 未配置时兼容默认：分配一张 NAT 网卡
	}
	if nicType != "none" {
		body["nic_all"] = map[string]any{
			"nic0": map[string]any{"nic_type": nicType},
		}
	}

	vmUUID, err := c.CreateVM(ctx, hsName, "", body)
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	if vmUUID == "" {
		return nil, fmt.Errorf("HostAgent did not return vm_uuid in create response")
	}

	// 创建成功后，根据操作系统类型自动映射远程连接端口（NAT）
	// Linux→22(SSH), Windows→3389(RDP), macOS→5900(VNC)
	osNameLower := strings.ToLower(body["os_name"].(string))
	if osNameLower == "" {
		osNameLower = strings.ToLower(req.GetOs())
	}
	lanPort := 0
	switch {
	case strings.Contains(osNameLower, "win"):
		lanPort = 3389
	case strings.Contains(osNameLower, "mac") || strings.Contains(osNameLower, "darwin"):
		lanPort = 5900
	default:
		// 默认按 Linux 处理
		lanPort = 22
	}
	if lanPort > 0 {
		// 从可用端口列表中获取一个宿主机端口
		wanPort := 0
		avail, portErr := c.GetAvailablePorts(ctx, hsName)
		if portErr == nil && len(avail.AvailablePorts) > 0 {
			wanPort = int(avail.AvailablePorts[0])
		}
		if wanPort > 0 {
			natErr := c.AddNATRule(ctx, hsName, vmUUID, wanPort, lanPort, "tcp", fmt.Sprintf("auto-remote-%d", lanPort))
			if natErr != nil {
				pluginLog.Printf("[CreateInstance] 自动映射端口失败 hs=%s vm=%s lan=%d wan=%d: %v", hsName, vmUUID, lanPort, wanPort, natErr)
			} else {
				pluginLog.Printf("[CreateInstance] 自动映射端口成功 hs=%s vm=%s lan=%d wan=%d", hsName, vmUUID, lanPort, wanPort)
			}
		} else {
			pluginLog.Printf("[CreateInstance] 无可用宿主机端口，跳过自动映射 hs=%s vm=%s", hsName, vmUUID)
		}
	}

	instanceID := a.core.ids.instanceID(hsName, vmUUID)
	return &pluginv1.CreateInstanceResponse{InstanceId: instanceID}, nil
}

// GetInstance 查询虚拟机详情
func (a *automationServer) GetInstance(ctx context.Context, req *pluginv1.GetInstanceRequest) (*pluginv1.GetInstanceResponse, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	hsName, vmUUID, err := a.core.resolveInstanceWithFallback(ctx, req.GetInstanceId())
	if err != nil {
		return nil, err
	}
	var info VMInfo
	err = a.core.retry(func() error {
		var callErr error
		info, callErr = c.GetVM(ctx, hsName, vmUUID)
		return callErr
	})
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	// state 值与 MapAutomationState 约定一致：
	//   2 = 运行中(running)       ← isReadyState
	//   3 = 已关机(stopped)       ← isReadyState
	//  10 = 已锁定(locked)        ← isReadyState
	//
	// 重要：只要 GetVM 能成功返回虚拟机信息，就说明 HostAgent 已完成创建，
	// 默认 state=3（stopped/就绪）。不再依赖虚拟机电源状态来判断是否"开通完成"，
	// 避免因 vm_flag=UNKNOWN/STOPPED 导致 provision_worker 误判为仍在创建中。
	state := int32(3) // 默认：已就绪（stopped）
	switch strings.ToLower(info.Status) {
	case "running", "powered_on", "starting":
		state = 2
	case "suspended":
		state = 10
	}
	instanceID := a.core.ids.instanceID(hsName, info.VMUUID)

	// 构建远程地址：public_addr:映射端口
	// 查询主机 public_addr 和 NAT 规则，拼接为 "公网IP:宿主机端口" 格式
	remoteAddr := info.IPAddress // 兜底：使用虚拟机内网 IP
	if publicIP := resolvePublicIP(ctx, c, hsName); publicIP != "" {
		remoteAddr = publicIP // 有公网 IP 时优先使用
		// 查询 NAT 规则，找到远程连接端口（SSH/RDP/VNC）
		natRules, natErr := c.ListNATRules(ctx, hsName, info.VMUUID)
		if natErr == nil {
			for _, rule := range natRules {
				if rule.LanPort == 22 || rule.LanPort == 3389 || rule.LanPort == 5900 {
					remoteAddr = fmt.Sprintf("%s:%d", publicIP, rule.WanPort)
					break
				}
			}
		}
	}

	return &pluginv1.GetInstanceResponse{
		Instance: &pluginv1.AutomationInstance{
			Id:       instanceID,
			Name:     hsName + "/" + info.VMUUID, // GetPanelURL 需要 "{hs_name}/{vm_uuid}" 格式
			State:    state,
			Cpu:      int32(info.CPUNum),
			MemoryGb: int32(info.MemNum / 1024), // MB → GB
			DiskGb:   int32(info.HDDNum / 1024), // MB → GB（HostAgent VMConfig.hdd_num 单位为 MB）
			RemoteIp: remoteAddr,
		},
	}, nil
}

// ListInstancesSimple 简易实例搜索（遍历所有主机下的虚拟机）
func (a *automationServer) ListInstancesSimple(ctx context.Context, req *pluginv1.ListInstancesSimpleRequest) (*pluginv1.ListInstancesSimpleResponse, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	var servers map[string]ServerInfo
	err = a.core.retry(func() error {
		var callErr error
		servers, callErr = c.ListServers(ctx)
		return callErr
	})
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	out := make([]*pluginv1.AutomationInstanceSimple, 0)
	searchTag := strings.ToLower(strings.TrimSpace(req.GetSearchTag()))
	for hsName := range servers {
		a.core.ids.lineID(hsName) // 顺便缓存 line_id
		vms, vmErr := c.ListVMs(ctx, hsName)
		if vmErr != nil {
			continue
		}
		for _, vm := range vms {
			if searchTag != "" {
				if !strings.Contains(strings.ToLower(vm.VMName), searchTag) &&
					!strings.Contains(strings.ToLower(vm.DisplayName), searchTag) &&
					!strings.Contains(strings.ToLower(vm.IPAddress), searchTag) {
					continue
				}
			}
			name := vm.DisplayName
			if name == "" {
				name = vm.VMName
			}
			instanceID := a.core.ids.instanceID(hsName, vm.VMUUID)
			out = append(out, &pluginv1.AutomationInstanceSimple{
				Id:   instanceID,
				Name: name,
				Ip:   vm.IPAddress,
			})
		}
	}
	return &pluginv1.ListInstancesSimpleResponse{Items: out}, nil
}

// Start 开机
func (a *automationServer) Start(ctx context.Context, req *pluginv1.StartRequest) (*pluginv1.OperationResult, error) {
	if a.core.cfg.DryRun {
		return &pluginv1.OperationResult{Ok: true}, nil
	}
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	hsName, vmUUID, err := a.core.resolveInstanceWithFallback(ctx, req.GetInstanceId())
	if err != nil {
		return nil, err
	}
	if err := c.PowerVM(ctx, hsName, vmUUID, "S_START"); err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.OperationResult{Ok: true}, nil
}

// Shutdown 关机
func (a *automationServer) Shutdown(ctx context.Context, req *pluginv1.ShutdownRequest) (*pluginv1.OperationResult, error) {
	if a.core.cfg.DryRun {
		return &pluginv1.OperationResult{Ok: true}, nil
	}
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	hsName, vmUUID, err := a.core.resolveInstanceWithFallback(ctx, req.GetInstanceId())
	if err != nil {
		return nil, err
	}
	if err := c.PowerVM(ctx, hsName, vmUUID, "H_CLOSE"); err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.OperationResult{Ok: true}, nil
}

// Reboot 重启
func (a *automationServer) Reboot(ctx context.Context, req *pluginv1.RebootRequest) (*pluginv1.OperationResult, error) {
	if a.core.cfg.DryRun {
		return &pluginv1.OperationResult{Ok: true}, nil
	}
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	hsName, vmUUID, err := a.core.resolveInstanceWithFallback(ctx, req.GetInstanceId())
	if err != nil {
		return nil, err
	}
	if err := c.PowerVM(ctx, hsName, vmUUID, "S_RESET"); err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.OperationResult{Ok: true}, nil
}

// Rebuild 重装系统（挂载 ISO + 重启）
// image_id = fnv64(hs_name + "/img/" + img.File)
func (a *automationServer) Rebuild(ctx context.Context, req *pluginv1.RebuildRequest) (*pluginv1.OperationResult, error) {
	if a.core.cfg.DryRun {
		return &pluginv1.OperationResult{Ok: true}, nil
	}
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	hsName, vmUUID, err := a.core.resolveInstanceWithFallback(ctx, req.GetInstanceId())
	if err != nil {
		return nil, err
	}
	// 解析 image_id → ISO 文件名
	isoName := ""
	if req.GetImageId() != 0 {
		if raw, ok := a.core.ids.get(req.GetImageId()); ok {
			parts := strings.SplitN(raw, "/img/", 2)
			if len(parts) == 2 {
				isoName = parts[1]
			}
		}
	}
	if isoName == "" {
		return nil, fmt.Errorf("image_id %d not found in cache, please call ListImages first", req.GetImageId())
	}
	// 步骤1：挂载 ISO
	if mountErr := c.MountISO(ctx, hsName, vmUUID, isoName); mountErr != nil {
		return nil, wrapHTTPTraceErr(mountErr, last)
	}
	// 步骤2：重启虚拟机
	if err := c.PowerVM(ctx, hsName, vmUUID, "S_RESET"); err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.OperationResult{Ok: true}, nil
}

// ResetPassword 重置系统密码
func (a *automationServer) ResetPassword(ctx context.Context, req *pluginv1.ResetPasswordRequest) (*pluginv1.OperationResult, error) {
	if a.core.cfg.DryRun {
		return &pluginv1.OperationResult{Ok: true}, nil
	}
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	hsName, vmUUID, err := a.core.resolveInstanceWithFallback(ctx, req.GetInstanceId())
	if err != nil {
		return nil, err
	}
	if err := c.UpdateVM(ctx, hsName, vmUUID, map[string]any{
		"password": req.GetPassword(),
	}); err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.OperationResult{Ok: true}, nil
}

// ElasticUpdate 弹性变更配置
func (a *automationServer) ElasticUpdate(ctx context.Context, req *pluginv1.ElasticUpdateRequest) (*pluginv1.OperationResult, error) {
	if a.core.cfg.DryRun {
		return &pluginv1.OperationResult{Ok: true}, nil
	}
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	hsName, vmUUID, err := a.core.resolveInstanceWithFallback(ctx, req.GetInstanceId())
	if err != nil {
		return nil, err
	}
	body := map[string]any{}
	if req.Cpu != nil {
		body["cpu_num"] = req.GetCpu()
	}
	if req.MemoryGb != nil {
		body["mem_num"] = req.GetMemoryGb() * 1024 // GB → MB
	}
	if req.DiskGb != nil {
		body["hdd_num"] = req.GetDiskGb() * 1024 // GB → MB（HostAgent VMConfig.hdd_num 单位为 MB）
	}
	if len(body) == 0 {
		return &pluginv1.OperationResult{Ok: true}, nil
	}
	if err := c.UpdateVM(ctx, hsName, vmUUID, body); err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.OperationResult{Ok: true}, nil
}

// Lock 锁定（强制关机，财务系统到期/欠费时调用）
func (a *automationServer) Lock(ctx context.Context, req *pluginv1.LockRequest) (*pluginv1.OperationResult, error) {
	if a.core.cfg.DryRun {
		return &pluginv1.OperationResult{Ok: true}, nil
	}
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	hsName, vmUUID, err := a.core.resolveInstanceWithFallback(ctx, req.GetInstanceId())
	if err != nil {
		return nil, err
	}
	// Lock = 强制关机（H_CLOSE）
	if err := c.PowerVM(ctx, hsName, vmUUID, "H_CLOSE"); err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.OperationResult{Ok: true}, nil
}

// Unlock 解锁（noop，财务系统续费后会主动调用 Start 开机）
func (a *automationServer) Unlock(_ context.Context, _ *pluginv1.UnlockRequest) (*pluginv1.OperationResult, error) {
	// OpenIDCS 没有"禁止开机"的状态，解锁只需返回成功
	// 财务系统续费后会自动调用 Start 开机
	return &pluginv1.OperationResult{Ok: true}, nil
}

// Renew 续费（noop，OpenIDCS 不处理到期概念）
func (a *automationServer) Renew(_ context.Context, _ *pluginv1.RenewRequest) (*pluginv1.OperationResult, error) {
	// OpenIDCS 不处理到期概念，续费由财务系统管理
	// 到期后财务系统发起 Lock（关机），续期后发起 Unlock + Start（开机）
	return &pluginv1.OperationResult{Ok: true}, nil
}

// Destroy 销毁虚拟机
func (a *automationServer) Destroy(ctx context.Context, req *pluginv1.DestroyRequest) (*pluginv1.OperationResult, error) {
	if a.core.cfg.DryRun {
		return &pluginv1.OperationResult{Ok: true}, nil
	}
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	hsName, vmUUID, err := a.core.resolveInstanceWithFallback(ctx, req.GetInstanceId())
	if err != nil {
		return nil, err
	}
	if err := c.DeleteVM(ctx, hsName, vmUUID); err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.OperationResult{Ok: true}, nil
}

// GetVNCURL 获取 VNC 控制台地址
func (a *automationServer) GetVNCURL(ctx context.Context, req *pluginv1.GetVNCURLRequest) (*pluginv1.GetVNCURLResponse, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	hsName, vmUUID, err := a.core.resolveInstanceWithFallback(ctx, req.GetInstanceId())
	if err != nil {
		return nil, err
	}
	var access RemoteAccess
	err = a.core.retry(func() error {
		var callErr error
		access, callErr = c.GetRemoteAccess(ctx, hsName, vmUUID)
		return callErr
	})
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	url := access.ConsoleURL
	if url == "" {
		url = access.TerminalURL
	}
	return &pluginv1.GetVNCURLResponse{Url: url}, nil
}

// GetPanelURL 获取面板「一键登录」URL。
// instance_name 格式：{hs_name}/{vm_uuid}（财务系统传入的是 VPS 的 name 字段）
// 实现参考 FSPlugins/OpenIDC-SwapIDC/openidc.php 的 openidc_ClientArea()：
//  1. 先调用 /api/client/temptoken/{hs_name}/{vm_uuid} 获取临时 token；
//  2. 拼接 {base_url}/api/client/templogin?token={temp_token} 作为跳转地址；
//  3. 若临时 token 缺失则降级返回 base_url，由用户自行登录。
func (a *automationServer) GetPanelURL(ctx context.Context, req *pluginv1.GetPanelURLRequest) (*pluginv1.GetPanelURLResponse, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	// instance_name 可能是 "{hs_name}/{vm_uuid}" 格式
	instanceName := strings.TrimSpace(req.GetInstanceName())
	var hsName, vmUUID string
	parts := strings.SplitN(instanceName, "/", 2)
	if len(parts) == 2 {
		hsName = parts[0]
		vmUUID = parts[1]
		// 顺便缓存
		a.core.ids.instanceID(hsName, vmUUID)
	} else {
		return nil, fmt.Errorf("invalid instance_name format, expected {hs_name}/{vm_uuid}, got: %s", instanceName)
	}
	var url string
	err = a.core.retry(func() error {
		var callErr error
		url, callErr = c.GetTempLoginURL(ctx, hsName, vmUUID)
		return callErr
	})
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.GetPanelURLResponse{Url: url}, nil
}

// GetMonitor 获取监控数据
func (a *automationServer) GetMonitor(ctx context.Context, req *pluginv1.GetMonitorRequest) (*pluginv1.GetMonitorResponse, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	hsName, vmUUID, err := a.core.resolveInstanceWithFallback(ctx, req.GetInstanceId())
	if err != nil {
		return nil, err
	}
	var statusMap map[string]any
	err = a.core.retry(func() error {
		var callErr error
		statusMap, callErr = c.GetVMStatus(ctx, hsName, vmUUID)
		return callErr
	})
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}

	// HostAgent /api/client/status 返回结构为:
	//   {"power_status": "...", "history": [{HWStatus}, ...]}
	// 需要从 history 数组中取最新一条 HWStatus 进行解析。
	// 同时兼容旧版直接返回扁平 map 的情况。
	latestStatus := statusMap
	if historyRaw, ok := statusMap["history"]; ok {
		if historySlice, ok := historyRaw.([]any); ok && len(historySlice) > 0 {
			// 取最后一条（最新的）HWStatus
			if latestEntry, ok := historySlice[len(historySlice)-1].(map[string]any); ok {
				latestStatus = latestEntry
			}
		}
	}

	// 从 HWStatus 中提取监控数据，兼容多种字段名
	cpuUsage := extractFloat(latestStatus, "cpu_usage", "cpu_percent", "cpu", "CpuUsage", "CpuStats")
	cpuTotal := extractFloat(latestStatus, "cpu_total", "cpu_num", "CpuTotal")
	memUsage := extractFloat(latestStatus, "mem_usage", "memory_usage", "mem_used", "MemoryUsage")
	memTotal := extractFloat(latestStatus, "mem_total", "memory_total", "MemoryTotal")
	hddUsage := extractFloat(latestStatus, "hdd_usage", "disk_usage", "DiskUsage")
	hddTotal := extractFloat(latestStatus, "hdd_total", "disk_total", "DiskTotal")
	netRx := extractFloat(latestStatus, "network_d", "network_rx_rate", "net_rx", "rx_rate", "rx_bytes", "NetworkRxRate", "bytes_in")
	netTx := extractFloat(latestStatus, "network_u", "network_tx_rate", "net_tx", "tx_rate", "tx_bytes", "NetworkTxRate", "bytes_out")

	// 计算 CPU 百分比：cpu_usage 是单核百分比（0~100），直接使用
	cpuPercent := 0.0
	if cpuUsage > 0 && cpuUsage <= 100 {
		cpuPercent = cpuUsage
	} else if cpuTotal > 0 && cpuUsage > 100 {
		// 兼容旧版返回多核总百分比的情况（如4核满载=400），需除以核心数
		cpuPercent = cpuUsage / cpuTotal
	}

	// 计算内存百分比：mem_usage 是已用MB，mem_total 是总MB
	memPercent := 0.0
	if mp := extractFloat(latestStatus, "memory_percent", "mem_percent", "memory", "MemoryStats"); mp > 0 {
		memPercent = mp
	} else if memTotal > 0 {
		memPercent = memUsage / memTotal * 100
	}

	// 计算磁盘使用百分比
	storagePercent := 0.0
	if hddTotal > 0 {
		storagePercent = hddUsage / hddTotal * 100
	}

	// 网络速率（HostAgent 的 network_u/network_d 单位为 Mbps）
	// 转换为字节/秒
	bytesIn := int64(netRx)
	bytesOut := int64(netTx)
	if netRx > 0 && netRx < 1000 {
		bytesIn = int64(netRx * 1024 * 1024 / 8)
	}
	if netTx > 0 && netTx < 1000 {
		bytesOut = int64(netTx * 1024 * 1024 / 8)
	}

	raw := map[string]any{
		"CpuStats":     cpuPercent,
		"MemoryStats":  memPercent,
		"StorageStats": storagePercent,
		"NetworkStats": map[string]any{
			"BytesSentPersec":     bytesOut,
			"BytesReceivedPersec": bytesIn,
		},
	}
	b, _ := json.Marshal(raw)
	return &pluginv1.GetMonitorResponse{RawJson: string(b)}, nil
}

// extractFloat 从 map 中按多个候选 key 提取 float64 值，返回第一个非零值。
// 兼容 HostAgent 不同版本返回的字段名差异。
func extractFloat(m map[string]any, keys ...string) float64 {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch val := v.(type) {
			case float64:
				return val
			case int:
				return float64(val)
			case int64:
				return float64(val)
			case json.Number:
				f, _ := val.Float64()
				return f
			case string:
				var f float64
				fmt.Sscanf(val, "%f", &f)
				return f
			}
		}
	}
	return 0
}

// ---- 端口映射 ----

// ListPortMappings 获取 NAT 规则列表
func (a *automationServer) ListPortMappings(ctx context.Context, req *pluginv1.ListPortMappingsRequest) (*pluginv1.ListPortMappingsResponse, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	hsName, vmUUID, err := a.core.resolveInstanceWithFallback(ctx, req.GetInstanceId())
	if err != nil {
		return nil, err
	}
	publicIP := resolvePublicIP(ctx, c, hsName)
	var rules []NATRule
	err = a.core.retry(func() error {
		var callErr error
		rules, callErr = c.ListNATRules(ctx, hsName, vmUUID)
		return callErr
	})
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	out := make([]*pluginv1.AutomationPortMapping, 0, len(rules))
	for _, rule := range rules {
		// sport 格式：若有公网 IP 则为 "public_addr:port"，否则仅端口号
		sport := fmt.Sprintf("%d", rule.WanPort)
		if publicIP != "" {
			sport = fmt.Sprintf("%s:%d", publicIP, rule.WanPort)
		}
		out = append(out, &pluginv1.AutomationPortMapping{
			Id:    int64(rule.RuleIndex),
			Name:  rule.NatTips,
			Sport: sport,
			Dport: int64(rule.LanPort),
		})
	}
	return &pluginv1.ListPortMappingsResponse{Items: out}, nil
}

// AddPortMapping 添加 NAT 规则
// sport 格式："{host_port}/{protocol}" 或 "{host_port}"
func (a *automationServer) AddPortMapping(ctx context.Context, req *pluginv1.AddPortMappingRequest) (*pluginv1.OperationResult, error) {
	if a.core.cfg.DryRun {
		return &pluginv1.OperationResult{Ok: true}, nil
	}
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	hsName, vmUUID, err := a.core.resolveInstanceWithFallback(ctx, req.GetInstanceId())
	if err != nil {
		return nil, err
	}
	// 解析 sport：格式 "8080/tcp" 或 "8080"
	hostPort := 0
	protocol := "tcp"
	sport := strings.TrimSpace(req.GetSport())
	if parts := strings.SplitN(sport, "/", 2); len(parts) == 2 {
		fmt.Sscanf(parts[0], "%d", &hostPort)
		protocol = parts[1]
	} else {
		fmt.Sscanf(sport, "%d", &hostPort)
	}
	if err := c.AddNATRule(ctx, hsName, vmUUID, hostPort, int(req.GetDport()), protocol, req.GetName()); err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.OperationResult{Ok: true}, nil
}

// DeletePortMapping 删除 NAT 规则
// mapping_id = rule_index
func (a *automationServer) DeletePortMapping(ctx context.Context, req *pluginv1.DeletePortMappingRequest) (*pluginv1.OperationResult, error) {
	if a.core.cfg.DryRun {
		return &pluginv1.OperationResult{Ok: true}, nil
	}
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	hsName, vmUUID, err := a.core.resolveInstanceWithFallback(ctx, req.GetInstanceId())
	if err != nil {
		return nil, err
	}
	if err := c.DeleteNATRule(ctx, hsName, vmUUID, int(req.GetMappingId())); err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.OperationResult{Ok: true}, nil
}

// FindPortCandidates 查找可用端口候选（从 OpenIDCS 获取主机可分配端口）
func (a *automationServer) FindPortCandidates(ctx context.Context, req *pluginv1.FindPortCandidatesRequest) (*pluginv1.FindPortCandidatesResponse, error) {
	hsName, vmUUID, err := a.core.resolveInstanceWithFallback(ctx, req.GetInstanceId())
	if err != nil {
		// 如果 instance_id 无法解析，返回空列表
		return &pluginv1.FindPortCandidatesResponse{Ports: []int64{}}, nil
	}
	_ = vmUUID // 端口候选基于主机，不需要 vm_uuid
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	var portData AvailablePorts
	err = a.core.retry(func() error {
		var callErr error
		portData, callErr = c.GetAvailablePorts(ctx, hsName)
		return callErr
	})
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.FindPortCandidatesResponse{Ports: portData.AvailablePorts}, nil
}

// ---- 备份管理 ----

// ListBackups 获取备份列表
func (a *automationServer) ListBackups(ctx context.Context, req *pluginv1.ListBackupsRequest) (*pluginv1.ListBackupsResponse, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	hsName, vmUUID, err := a.core.resolveInstanceWithFallback(ctx, req.GetInstanceId())
	if err != nil {
		return nil, err
	}
	var backups []BackupInfo
	err = a.core.retry(func() error {
		var callErr error
		backups, callErr = c.ListBackups(ctx, hsName, vmUUID)
		return callErr
	})
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	out := make([]*pluginv1.AutomationBackup, 0, len(backups))
	for i, b := range backups {
		createdAt := parseTimeToUnix(b.CreatedTime)
		out = append(out, &pluginv1.AutomationBackup{
			Id:            int64(i), // 用索引作为 ID
			Name:          b.BackupName,
			CreatedAtUnix: createdAt,
			State:         1,
		})
	}
	return &pluginv1.ListBackupsResponse{Items: out}, nil
}

// CreateBackup 创建备份
func (a *automationServer) CreateBackup(ctx context.Context, req *pluginv1.CreateBackupRequest) (*pluginv1.OperationResult, error) {
	if a.core.cfg.DryRun {
		return &pluginv1.OperationResult{Ok: true}, nil
	}
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	hsName, vmUUID, err := a.core.resolveInstanceWithFallback(ctx, req.GetInstanceId())
	if err != nil {
		return nil, err
	}
	if err := c.CreateBackup(ctx, hsName, vmUUID, "", ""); err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.OperationResult{Ok: true}, nil
}

// DeleteBackup 删除备份
// backup_id = 备份索引（对应 ListBackups 返回的 id）
func (a *automationServer) DeleteBackup(ctx context.Context, req *pluginv1.DeleteBackupRequest) (*pluginv1.OperationResult, error) {
	if a.core.cfg.DryRun {
		return &pluginv1.OperationResult{Ok: true}, nil
	}
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	hsName, vmUUID, err := a.core.resolveInstanceWithFallback(ctx, req.GetInstanceId())
	if err != nil {
		return nil, err
	}
	// 先获取备份列表，通过索引找到备份名称
	backups, listErr := c.ListBackups(ctx, hsName, vmUUID)
	if listErr != nil {
		return nil, wrapHTTPTraceErr(listErr, last)
	}
	idx := int(req.GetBackupId())
	if idx < 0 || idx >= len(backups) {
		return nil, fmt.Errorf("backup index %d out of range (total: %d)", idx, len(backups))
	}
	if err := c.DeleteBackup(ctx, hsName, vmUUID, backups[idx].BackupName); err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.OperationResult{Ok: true}, nil
}

// RestoreBackup 恢复备份
func (a *automationServer) RestoreBackup(ctx context.Context, req *pluginv1.RestoreBackupRequest) (*pluginv1.OperationResult, error) {
	if a.core.cfg.DryRun {
		return &pluginv1.OperationResult{Ok: true}, nil
	}
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	hsName, vmUUID, err := a.core.resolveInstanceWithFallback(ctx, req.GetInstanceId())
	if err != nil {
		return nil, err
	}
	// 先获取备份列表，通过索引找到备份名称
	backups, listErr := c.ListBackups(ctx, hsName, vmUUID)
	if listErr != nil {
		return nil, wrapHTTPTraceErr(listErr, last)
	}
	idx := int(req.GetBackupId())
	if idx < 0 || idx >= len(backups) {
		return nil, fmt.Errorf("backup index %d out of range (total: %d)", idx, len(backups))
	}
	if err := c.RestoreBackup(ctx, hsName, vmUUID, backups[idx].BackupName); err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.OperationResult{Ok: true}, nil
}

// ---- 快照管理 ----

// ListSnapshots 获取快照列表
func (a *automationServer) ListSnapshots(ctx context.Context, req *pluginv1.ListSnapshotsRequest) (*pluginv1.ListSnapshotsResponse, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	hsName, vmUUID, err := a.core.resolveInstanceWithFallback(ctx, req.GetInstanceId())
	if err != nil {
		return nil, err
	}
	var snapshots []SnapshotInfo
	err = a.core.retry(func() error {
		var callErr error
		snapshots, callErr = c.ListSnapshots(ctx, hsName, vmUUID)
		return callErr
	})
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	out := make([]*pluginv1.AutomationSnapshot, 0, len(snapshots))
	for i, s := range snapshots {
		createdAt := parseTimeToUnix(s.CreatedTime)
		out = append(out, &pluginv1.AutomationSnapshot{
			Id:            int64(i),
			Name:          s.SnapshotName,
			CreatedAtUnix: createdAt,
			State:         1,
		})
	}
	return &pluginv1.ListSnapshotsResponse{Items: out}, nil
}

// CreateSnapshot 创建快照
func (a *automationServer) CreateSnapshot(ctx context.Context, req *pluginv1.CreateSnapshotRequest) (*pluginv1.OperationResult, error) {
	if a.core.cfg.DryRun {
		return &pluginv1.OperationResult{Ok: true}, nil
	}
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	hsName, vmUUID, err := a.core.resolveInstanceWithFallback(ctx, req.GetInstanceId())
	if err != nil {
		return nil, err
	}
	if err := c.CreateSnapshot(ctx, hsName, vmUUID, "", ""); err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.OperationResult{Ok: true}, nil
}

// DeleteSnapshot 删除快照
// snapshot_id = 快照索引（对应 ListSnapshots 返回的 id）
func (a *automationServer) DeleteSnapshot(ctx context.Context, req *pluginv1.DeleteSnapshotRequest) (*pluginv1.OperationResult, error) {
	if a.core.cfg.DryRun {
		return &pluginv1.OperationResult{Ok: true}, nil
	}
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	hsName, vmUUID, err := a.core.resolveInstanceWithFallback(ctx, req.GetInstanceId())
	if err != nil {
		return nil, err
	}
	// 先获取快照列表，通过索引找到快照名称
	snapshots, listErr := c.ListSnapshots(ctx, hsName, vmUUID)
	if listErr != nil {
		return nil, wrapHTTPTraceErr(listErr, last)
	}
	idx := int(req.GetSnapshotId())
	if idx < 0 || idx >= len(snapshots) {
		return nil, fmt.Errorf("snapshot index %d out of range (total: %d)", idx, len(snapshots))
	}
	if err := c.DeleteSnapshot(ctx, hsName, vmUUID, snapshots[idx].SnapshotName); err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.OperationResult{Ok: true}, nil
}

// RestoreSnapshot 恢复快照
func (a *automationServer) RestoreSnapshot(ctx context.Context, req *pluginv1.RestoreSnapshotRequest) (*pluginv1.OperationResult, error) {
	if a.core.cfg.DryRun {
		return &pluginv1.OperationResult{Ok: true}, nil
	}
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	hsName, vmUUID, err := a.core.resolveInstanceWithFallback(ctx, req.GetInstanceId())
	if err != nil {
		return nil, err
	}
	// 先获取快照列表，通过索引找到快照名称
	snapshots, listErr := c.ListSnapshots(ctx, hsName, vmUUID)
	if listErr != nil {
		return nil, wrapHTTPTraceErr(listErr, last)
	}
	idx := int(req.GetSnapshotId())
	if idx < 0 || idx >= len(snapshots) {
		return nil, fmt.Errorf("snapshot index %d out of range (total: %d)", idx, len(snapshots))
	}
	if err := c.RestoreSnapshot(ctx, hsName, vmUUID, snapshots[idx].SnapshotName); err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.OperationResult{Ok: true}, nil
}

// ---- 防火墙管理 ----

// ListFirewallRules 获取防火墙规则列表
func (a *automationServer) ListFirewallRules(ctx context.Context, req *pluginv1.ListFirewallRulesRequest) (*pluginv1.ListFirewallRulesResponse, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	hsName, vmUUID, err := a.core.resolveInstanceWithFallback(ctx, req.GetInstanceId())
	if err != nil {
		return nil, err
	}
	publicIP := resolvePublicIP(ctx, c, hsName)
	var rules []FirewallRule
	err = a.core.retry(func() error {
		var callErr error
		rules, callErr = c.ListFirewallRules(ctx, hsName, vmUUID, publicIP)
		return callErr
	})
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	out := make([]*pluginv1.AutomationFirewallRule, 0, len(rules))
	for _, r := range rules {
		out = append(out, &pluginv1.AutomationFirewallRule{
			Id:        int64(r.RuleID),
			Direction: r.Direction,
			Protocol:  r.Protocol,
			Method:    r.Method,
			Port:      r.Port,
			Ip:        r.IP,
			Priority:  int32(r.Priority),
		})
	}
	return &pluginv1.ListFirewallRulesResponse{Items: out}, nil
}

// AddFirewallRule 添加防火墙规则
func (a *automationServer) AddFirewallRule(ctx context.Context, req *pluginv1.AddFirewallRuleRequest) (*pluginv1.OperationResult, error) {
	if a.core.cfg.DryRun {
		return &pluginv1.OperationResult{Ok: true}, nil
	}
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	hsName, vmUUID, err := a.core.resolveInstanceWithFallback(ctx, req.GetInstanceId())
	if err != nil {
		return nil, err
	}
	rule := FirewallRule{
		Direction: req.GetDirection(),
		Protocol:  req.GetProtocol(),
		Method:    req.GetMethod(),
		Port:      req.GetPort(),
		IP:        req.GetIp(),
		Priority:  int(req.GetPriority()),
	}
	if err := c.AddFirewallRule(ctx, hsName, vmUUID, rule); err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.OperationResult{Ok: true}, nil
}

// DeleteFirewallRule 删除防火墙规则
func (a *automationServer) DeleteFirewallRule(ctx context.Context, req *pluginv1.DeleteFirewallRuleRequest) (*pluginv1.OperationResult, error) {
	if a.core.cfg.DryRun {
		return &pluginv1.OperationResult{Ok: true}, nil
	}
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	hsName, vmUUID, err := a.core.resolveInstanceWithFallback(ctx, req.GetInstanceId())
	if err != nil {
		return nil, err
	}
	if err := c.DeleteFirewallRule(ctx, hsName, vmUUID, int(req.GetRuleId())); err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	return &pluginv1.OperationResult{Ok: true}, nil
}

// ---- 工具函数 ----

// resolvePublicIP 查询主机的公网 IP（取 public_addr 列表的第一个）。
// 查询失败或 public_addr 为空时返回空字符串。
func resolvePublicIP(ctx context.Context, c *Client, hsName string) string {
	servers, err := c.ListServers(ctx)
	if err != nil {
		return ""
	}
	if srv, ok := servers[hsName]; ok && len(srv.Config.PublicAddr) > 0 {
		return srv.Config.PublicAddr[0]
	}
	return ""
}

// parseTimeToUnix 解析时间字符串为 Unix 时间戳
func parseTimeToUnix(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t.Unix()
		}
	}
	return 0
}

// ---- main ----

func main() {
	initLogger()
	pluginLog.Printf("plugin starting, pid=%d", os.Getpid())
	ids := newIDStore()
	core := &coreServer{ids: ids}
	auto := &automationServer{core: core}
	pluginsdk.Serve(map[string]pluginsdk.Plugin{
		pluginsdk.PluginKeyCore:       &pluginsdk.CoreGRPCPlugin{Impl: core},
		pluginsdk.PluginKeyAutomation: &pluginsdk.AutomationGRPCPlugin{Impl: auto},
	})
}
