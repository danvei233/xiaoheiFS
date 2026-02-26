package main

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
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
	TimeoutSec int    `json:"timeout_sec"`
	Retry      int    `json:"retry"`
	DryRun     bool   `json:"dry_run"`
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
		Description: "Built-in OpenIDCS-Client automation plugin (catalog + lifecycle + port_mapping + backup).",
		Automation: &pluginv1.AutomationCapability{
			Features: []pluginv1.AutomationFeature{
				pluginv1.AutomationFeature_AUTOMATION_FEATURE_CATALOG_SYNC,
				pluginv1.AutomationFeature_AUTOMATION_FEATURE_LIFECYCLE,
				pluginv1.AutomationFeature_AUTOMATION_FEATURE_PORT_MAPPING,
				pluginv1.AutomationFeature_AUTOMATION_FEATURE_BACKUP,
			},
			NotSupportedReasons: map[int32]string{},
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
    "timeout_sec": { "type": "integer", "title": "超时时间（秒）", "default": 15, "minimum": 1, "maximum": 60 },
    "retry": { "type": "integer", "title": "重试次数", "default": 1, "minimum": 0, "maximum": 5 },
    "dry_run": { "type": "boolean", "title": "Dry Run（演练模式）", "default": false }
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
	if cfg.TimeoutSec < 0 || cfg.TimeoutSec > 60 {
		return &pluginv1.ValidateConfigResponse{Ok: false, Error: "timeout_sec out of range [0,60]"}, nil
	}
	if cfg.Retry < 0 || cfg.Retry > 5 {
		return &pluginv1.ValidateConfigResponse{Ok: false, Error: "retry out of range [0,5]"}, nil
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
	msg := "ok"
	if req.GetInstanceId() == "" || s.instance == "" {
		msg = "not initialized"
	}
	return &pluginv1.HealthCheckResponse{
		Status:     pluginv1.HealthStatus_HEALTH_STATUS_OK,
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
		timeout = 15 * time.Second
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
	for hs := range servers {
		s.ids.lineID(hs) // 顺便缓存 line_id
		vms, vmErr := c.ListVMs(ctx, hs)
		if vmErr != nil {
			continue
		}
		for _, vm := range vms {
			id := s.ids.instanceID(hs, vm.VMUUID)
			if id == instanceID {
				return hs, vm.VMUUID, nil
			}
		}
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
	msg := extractTraceMessage(trace)
	if strings.TrimSpace(msg) == "" {
		msg = err.Error()
	}
	raw, marshalErr := json.Marshal(trace)
	if marshalErr != nil {
		return fmt.Errorf("%s", msg)
	}
	return fmt.Errorf("%s | http_trace=%s", msg, string(raw))
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
		// 缓存 area_id → area_name
		a.core.ids.put(area.ID, "area/"+area.Name)
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
		// 尝试从主机 server_area 获取 area_id
		var areaID int64
		if info.ServerArea != "" {
			areaID = fnv64("area/" + info.ServerArea)
		}
		out = append(out, &pluginv1.AutomationLine{
			Id:     lineID,
			Name:   name,
			AreaId: areaID,
			State:  state,
		})
	}
	return &pluginv1.ListLinesResponse{Items: out}, nil
}

// ListPackages 套餐列表（从 OpenIDCS server_plan 字段获取）
// line_id = fnv64(hs_name)
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
	var plans []PlanInfo
	err = a.core.retry(func() error {
		var callErr error
		plans, callErr = c.ListPlans(ctx, hsName)
		return callErr
	})
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	out := make([]*pluginv1.AutomationPackage, 0, len(plans))
	for _, plan := range plans {
		// plan_id = fnv64(hs_name + "/plan/" + plan.ID)
		planKey := hsName + "/plan/" + plan.ID
		planID := fnv64(planKey)
		a.core.ids.put(planID, planKey)
		out = append(out, &pluginv1.AutomationPackage{
			Id:           planID,
			Name:         plan.Name,
			Cpu:          int32(plan.CPU),
			MemoryGb:     int32(plan.MemoryGB),
			DiskGb:       int32(plan.DiskGB),
			MonthlyPrice: 0, // 价格由财务系统管理
		})
	}
	return &pluginv1.ListPackagesResponse{Items: out}, nil
}

// ListImages 镜像列表（映射 OpenIDCS OS 镜像接口）
// line_id = fnv64(hs_name)，image_id = fnv64(hs_name + "/" + img.File)
func (a *automationServer) ListImages(ctx context.Context, req *pluginv1.ListImagesRequest) (*pluginv1.ListImagesResponse, error) {
	c, last, err := a.core.newClientWithTrace()
	if err != nil {
		return nil, err
	}
	hsName, err := a.core.resolveLineWithFallback(ctx, req.GetLineId())
	if err != nil {
		return nil, err
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
	out := make([]*pluginv1.AutomationImage, 0)
	for category, images := range imageMap {
		for _, img := range images {
			name := img.Name
			if name == "" {
				name = img.File
			}
			// image_id = fnv64(hs_name + "/" + img.File)，缓存反查
			imageKey := hsName + "/img/" + img.File
			imageID := fnv64(imageKey)
			a.core.ids.put(imageID, imageKey)
			out = append(out, &pluginv1.AutomationImage{
				Id:   imageID,
				Name: name,
				Type: category,
			})
		}
	}
	return &pluginv1.ListImagesResponse{Items: out}, nil
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
		"vm_name": req.GetName(),
		"cpu_num": req.GetCpu(),
		"mem_num": req.GetMemoryGb() * 1024, // GB → MB
		"hdd_num": req.GetDiskGb(),
	}
	if isoName != "" {
		body["os_name"] = isoName
	} else if req.GetOs() != "" {
		body["os_name"] = req.GetOs()
	}
	if req.GetPassword() != "" {
		body["password"] = req.GetPassword()
	}
	if req.GetVncPassword() != "" {
		body["vnc_password"] = req.GetVncPassword()
	}
	vmUUID, err := c.CreateVM(ctx, hsName, body)
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
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
	state := int32(0)
	switch strings.ToLower(info.Status) {
	case "running", "powered_on":
		state = 1
	case "stopped", "powered_off":
		state = 0
	case "suspended":
		state = 2
	}
	instanceID := a.core.ids.instanceID(hsName, info.VMUUID)
	return &pluginv1.GetInstanceResponse{
		Instance: &pluginv1.AutomationInstance{
			Id:       instanceID,
			Name:     info.VMName,
			State:    state,
			Cpu:      int32(info.CPUNum),
			MemoryGb: int32(info.MemNum / 1024), // MB → GB
			DiskGb:   int32(info.HDDNum),
			RemoteIp: info.IPAddress,
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
	if _, mountErr := c.doRequest(ctx, "POST", "/api/client/iso/mount/"+hsName+"/"+vmUUID, map[string]any{
		"iso_name": isoName,
	}); mountErr != nil {
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
		body["hdd_num"] = req.GetDiskGb()
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

// GetPanelURL 获取面板/终端地址
// instance_name 格式：{hs_name}/{vm_uuid}（财务系统传入的是 VPS 的 name 字段）
// 为兼容性，同时支持 instance_name 为 fnv64 数字字符串的情况
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
	var access RemoteAccess
	err = a.core.retry(func() error {
		var callErr error
		access, callErr = c.GetRemoteAccess(ctx, hsName, vmUUID)
		return callErr
	})
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	url := access.TerminalURL
	if url == "" {
		url = access.ConsoleURL
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
	var status VMStatus
	err = a.core.retry(func() error {
		var callErr error
		status, callErr = c.GetVMStatus(ctx, hsName, vmUUID)
		return callErr
	})
	if err != nil {
		return nil, wrapHTTPTraceErr(err, last)
	}
	// 转换为财务系统期望的监控格式
	memPercent := 0.0
	if status.MemoryTotal > 0 {
		memPercent = float64(status.MemoryUsage) / float64(status.MemoryTotal) * 100
	}
	raw := map[string]any{
		"CpuStats":    status.CPUUsage,
		"MemoryStats": memPercent,
		"StorageStats": 0,
		"NetworkStats": map[string]any{
			"BytesSentPersec":     int64(status.NetworkTxRate * 1024 * 1024 / 8),
			"BytesReceivedPersec": int64(status.NetworkRxRate * 1024 * 1024 / 8),
		},
		"UptimeSeconds": status.UptimeSeconds,
		"PowerState":    status.PowerState,
		"IPAddresses":   status.IPAddresses,
	}
	b, _ := json.Marshal(raw)
	return &pluginv1.GetMonitorResponse{RawJson: string(b)}, nil
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
		out = append(out, &pluginv1.AutomationPortMapping{
			Id:    int64(rule.RuleIndex),
			Name:  rule.Description,
			Sport: fmt.Sprintf("%d/%s", rule.HostPort, rule.Protocol),
			Dport: int64(rule.VMPort),
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

// ---- 工具函数 ----

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
