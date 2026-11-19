package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"sync"
	"time"
)

// Client 封装 OpenIDCS-Client REST API 调用
type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
	logFn   func(context.Context, HTTPLogEntry)
}

// HTTPLogEntry 记录一次 HTTP 请求的日志
type HTTPLogEntry struct {
	Action   string
	Request  map[string]any
	Response map[string]any
	Success  bool
	Message  string
}

// NewClient 创建新的 OpenIDCS 客户端
func NewClient(baseURL, apiKey string, timeout time.Duration) *Client {
	if timeout <= 0 {
		// 兜底 1800s：覆盖备份/快照等长耗时操作。
		timeout = 1800 * time.Second
	}
	return &Client{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		apiKey:  apiKey,
		http: &http.Client{
			Timeout: timeout,
		},
	}
}

// WithLogger 设置日志回调
func (c *Client) WithLogger(fn func(context.Context, HTTPLogEntry)) *Client {
	c.logFn = fn
	return c
}

// ---- CSRF Token ----

// errCSRFExpired 标记上游拒绝了当前 CSRF Token，刷新后可重试。
var errCSRFExpired = errors.New("csrf token rejected by upstream")

// csrfTokenCache 按主控地址缓存 CSRF Token。
//
// 背景：CloudHost 主控的 /api 分组启用了 auth.CSRFMiddleware，
// 所有 POST/PUT/DELETE 请求都必须携带 X-CSRF-Token，否则返回
// 403 {"error":"CSRF token 无效或已过期"}；GET 等安全方法豁免。
//
// 插件每个 RPC 都会 NewClient，token 挂在 Client 实例上命中率极低，
// 因此改用包级缓存。Token 由主控内存保存（24h 有效），主控重启后会全部
// 失效，由 doRequest 的 403 重试兜底。
var (
	csrfCacheMu sync.RWMutex
	csrfCache   = make(map[string]string)
)

// ---- 响应结构 ----

type apiResponse struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
	// Timestamp 与 HostAgent 协议保持一致。
	// 使用 json.Number 兼容两种格式：
	//   - 新版 HostAgent 返回 int(time.time())，即 Unix 秒级整数
	//   - 老版 HostAgent 可能返回字符串格式的时间戳
	// 当前代码不读取此字段，仅保留协议完整性。
	Timestamp json.Number `json:"timestamp"`
}

// ---- 数据结构 ----

// ServerInfo 主机信息
// HostAgent /api/server/detail 返回结构为 {hs_name: {name, status, server_area, server_type, addr, config: HSConfig, ...}}；
// 其中 config 承载整份 HSConfig（含套餐、端口段、价格等），由 ServerDetailConfig 解析。
type ServerInfo struct {
	ServerName  string  `json:"server_name"`
	ServerType  string  `json:"server_type"`
	ServerAddr  string  `json:"server_addr"`
	Status      string  `json:"status"`
	ServerArea  string  `json:"server_area"` // 主机所属区域（来自 HSConfig.server_area）
	VMsCount    int     `json:"vms_count"`
	RunningVMs  int     `json:"running_vms"`
	StoppedVMs  int     `json:"stopped_vms"`
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	DiskUsage   float64 `json:"disk_usage"`
	LastCheck   string  `json:"last_check"`
	// Config 对应 HostAgent HSConfig 的完整子对象；
	// - server_plan 里是 VMConfig 格式的套餐（key=plan_name, value=VMConfig）
	// - ports_start/ports_close 是主机端口段
	// - n_cpu_price/n_mem_price/n_hdd_price/n_net_price 是四项单价（若 HostAgent 尚未升级则为 0）
	Config ServerDetailConfig `json:"config"`
}

// ServerDetailConfig 对应 HostAgent MainObject/Config/HSConfig.py 的完整配置
// 这里只解析对插件有用的字段，其余未列出的 HSConfig 字段由 json 默认忽略。
type ServerDetailConfig struct {
	ServerArea string `json:"server_area"`
	PortsStart int    `json:"ports_start"`
	PortsClose int    `json:"ports_close"`
	LimitsNums int    `json:"limits_nums"`
	// 单价字段：HostAgent HSConfig.n_cpu_price/n_mem_price/n_hdd_price/n_net_price
	// 旧版 HostAgent 无此字段时，json 解析为 0，下游视为"未配置"。
	// 注意：HostAgent Python 端可能把价格写成 float（如 1.0），故用 float64 以兼容整数/小数两种写法，
	// 否则 encoding/json 在严格模式下无法把 1.0 解析进 int，会报 "cannot unmarshal number 1.0 into Go struct field"。
	NCPUPrice float64 `json:"n_cpu_price"`
	NMemPrice float64 `json:"n_mem_price"`
	NHDDPrice float64 `json:"n_hdd_price"`
	NNetPrice float64 `json:"n_net_price"`
	// PublicAddr: 主机公网 IP 地址列表（来自 HSConfig.public_addr）
	PublicAddr []string `json:"public_addr"`
	// ServerPlan: key=plan_name, value=VMPlan（即 HostAgent VMConfig 的关键子字段）
	ServerPlan map[string]VMPlan `json:"server_plan"`
}

// VMPlan 从 HostAgent VMConfig 中抽取与套餐相关的字段
// 参考: MainObject/Config/VMConfig.py
//   - cpu_num: 处理器核心数
//   - mem_num: 内存 MB（小黑云侧转 GB 存储）
//   - hdd_num: 硬盘 MB（小黑云侧转 GB 存储）
//   - speed_u/speed_d: 上/下行带宽 Mbps（取 max 作为 BandwidthMbps）
//   - nat_num:  VM 分配的端口数，对应小黑云套餐 port_num
//   - flu_num:  VM 分配的流量 MB（后续可扩展）
//   - gpu_mem:  GPU 显存 MB（后续可扩展）
type VMPlan struct {
	CPUNum int `json:"cpu_num"`
	MemNum int `json:"mem_num"`
	HDDNum int `json:"hdd_num"`
	SpeedU int `json:"speed_u"`
	SpeedD int `json:"speed_d"`
	NATNum int `json:"nat_num"`
	FluNum int `json:"flu_num"`
	GPUMem int `json:"gpu_mem"`
}

// VMInfo 虚拟机信息
// HostAgent VMConfig 序列化后的字段名与本结构体一一对应：
//   - vm_flag: VMPowers 枚举名（"STARTED", "STOPPED", "SUSPEND", "UNKNOWN" 等）
//   - status / power_state: 插件内部使用的归一化字段，HostAgent 不直接返回
type VMInfo struct {
	VMUUID      string `json:"vm_uuid"`
	VMName      string `json:"vm_name"`
	DisplayName string `json:"display_name"`
	OSName      string `json:"os_name"`
	CPUNum      int    `json:"cpu_num"`
	MemNum      int    `json:"mem_num"`
	HDDNum      int    `json:"hdd_num"`
	GPUNum      int    `json:"gpu_num"`
	VMFlag      string `json:"vm_flag"`     // HostAgent VMPowers 枚举名：STARTED/STOPPED/SUSPEND/UNKNOWN 等
	Status      string `json:"status"`      // 归一化状态（由插件填充，HostAgent 不直接返回此字段）
	PowerState  string `json:"power_state"` // 归一化电源状态（由插件填充）
	IPAddress   string `json:"ip_address"`
	CreatedTime string `json:"created_time"`
	Owner       string `json:"owner"`
	VCPass      string `json:"vc_pass"` // VNC 密码
	OSPass      string `json:"os_pass"` // 系统密码
}

// normalizeStatus 将 HostAgent 返回的 vm_flag（VMPowers 枚举名）归一化为
// 插件内部使用的 Status 和 PowerState 字段。
func (v *VMInfo) normalizeStatus() {
	switch strings.ToUpper(v.VMFlag) {
	case "STARTED":
		v.Status = "running"
		v.PowerState = "on"
	case "STOPPED":
		v.Status = "stopped"
		v.PowerState = "off"
	case "SUSPEND":
		v.Status = "suspended"
		v.PowerState = "suspended"
	default:
		v.Status = "unknown"
		v.PowerState = "unknown"
	}
}

// VMStatus 虚拟机状态
type VMStatus struct {
	VMUUID        string   `json:"vm_uuid"`
	Status        string   `json:"status"`
	PowerState    string   `json:"power_state"`
	CPUUsage      float64  `json:"cpu_usage"`
	MemoryUsage   int64    `json:"memory_usage"`
	MemoryTotal   int64    `json:"memory_total"`
	DiskReadRate  int64    `json:"disk_read_rate"`
	DiskWriteRate int64    `json:"disk_write_rate"`
	NetworkRxRate float64  `json:"network_rx_rate"`
	NetworkTxRate float64  `json:"network_tx_rate"`
	UptimeSeconds int64    `json:"uptime_seconds"`
	IPAddresses   []string `json:"ip_addresses"`
}

// OSImage OS 镜像
// Category: "iso"（光驱镜像）/ "system"（系统磁盘模板）
// Type: "windows" / "linux" / "macos"（由 HostAgent sys_type 推导或文件名启发）
// Enabled: HostAgent OSConfig.sys_flag，false 表示该镜像在上游被手动停用
type OSImage struct {
	Name     string `json:"name"`
	File     string `json:"file"`
	SizeMB   int    `json:"size_mb"`
	SizeGB   int    `json:"size_gb"`
	Version  string `json:"version"`
	Arch     string `json:"architecture"`
	Type     string `json:"type"`
	Enabled  bool   `json:"enabled"`
	Category string `json:"category"`
}

// OSConfigItem 对应 HostAgent MainObject/Config/OSConfig.py 的新结构
// 参考：class OSConfig: sys_name / sys_file / sys_size / sys_type / sys_flag
type OSConfigItem struct {
	SysName string `json:"sys_name"`
	SysFile string `json:"sys_file"`
	SysSize string `json:"sys_size"`           // 允许最低磁盘值-GB（字符串）
	SysType string `json:"sys_type"`           // WinNT / Linux / macOS
	SysFlag *bool  `json:"sys_flag,omitempty"` // 是否启用此镜像（兼容老 HostAgent 无此字段时默认 true）
}

// HSConfig 主机配置（/api/client/os-images/{hsName} 返回的原始结构）
// images_maps / system_maps 同时兼容新旧两种格式，具体解析由 ListOSImages 处理：
//   - 新：list[OSConfig]（每项含 sys_name/sys_file/sys_size/sys_type/sys_flag）
//   - 旧：dict[name]=file（images_maps）/ dict[name]=[file,size]（system_maps）
type HSConfig struct {
	HostName   string          `json:"host_name"`
	ServerType string          `json:"server_type"`
	FilterName string          `json:"filter_name"`
	ImagesMaps json.RawMessage `json:"images_maps"`
	SystemMaps json.RawMessage `json:"system_maps"`
}

// NATRule NAT 端口转发规则
type NATRule struct {
	RuleIndex int    `json:"rule_index"`
	WanPort   int    `json:"wan_port"` // 宿主机端口（外部端口）
	LanPort   int    `json:"lan_port"` // 虚拟机端口（内部端口）
	LanAddr   string `json:"lan_addr"` // 虚拟机内网地址
	NatTips   string `json:"nat_tips"` // 备注说明
}

// BackupInfo 备份信息
type BackupInfo struct {
	BackupName    string `json:"backup_name"`
	Description   string `json:"description"`
	SizeMB        int    `json:"size_mb"`
	CreatedTime   string `json:"created_time"`
	CreatedBy     string `json:"created_by"`
	VMState       string `json:"vm_state"`
	IncludeMemory bool   `json:"include_memory"`
	Compressed    bool   `json:"compressed"`
}

// RemoteAccess 控制台访问信息
type RemoteAccess struct {
	ConsoleURL  string `json:"console_url"`
	TerminalURL string `json:"terminal_url"`
	ExpiresAt   string `json:"expires_at"`
}

// AreaInfo 区域信息（来自 OpenIDCS server_area）
// Name 承载完整的 server_area 原始字符串（如 "CN,成都"），
// 仅在内部生成稳定 ID 时通过 parseServerArea 提取 code 部分。
// 上游未填写时统一归并为 Name="default"。
type AreaInfo struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	State int    `json:"state"`
}

// defaultAreaCode / defaultAreaName 作为上游 server_area 为空时的兜底区域标识
const (
	defaultAreaCode = "default"
	defaultAreaName = "default"
)

// defaultAreaID 稳定的 default 区域 ID（与具体名字解耦，避免与旧数据冲突）
var defaultAreaID = fnv64("area/__default__")

// parseServerArea 解析 HostAgent server_area 字符串
// 约定格式："code,name"，例如 "CN,成都"；单值时作为 code，name 回退到 code；
// 完全为空时回退到 default/default。
func parseServerArea(raw string) (code, name string) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return defaultAreaCode, defaultAreaName
	}
	parts := strings.SplitN(s, ",", 2)
	code = strings.TrimSpace(parts[0])
	if len(parts) == 2 {
		name = strings.TrimSpace(parts[1])
	}
	if code == "" {
		code = defaultAreaCode
	}
	if name == "" {
		name = code
	}
	return code, name
}

// areaIDFromCode 由 code 生成稳定的 area_id
func areaIDFromCode(code string) int64 {
	c := strings.TrimSpace(code)
	if c == "" || c == defaultAreaCode {
		return defaultAreaID
	}
	return fnv64("area/" + c)
}

// PlanInfo 套餐规格信息（来自 OpenIDCS server_plan）
type PlanInfo struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	CPU           int    `json:"cpu"`
	MemoryGB      int    `json:"memory_gb"`
	DiskGB        int    `json:"disk_gb"`
	GPUMemGB      int    `json:"gpu_mem_gb"`
	BandwidthMbps int    `json:"bandwidth_mbps"`
	TrafficGB     int    `json:"traffic_gb"`
}

// AvailablePorts 可分配端口信息
type AvailablePorts struct {
	HostName       string  `json:"host_name"`
	PortsStart     int     `json:"ports_start"`
	PortsClose     int     `json:"ports_close"`
	AvailableCount int     `json:"available_count"`
	AvailablePorts []int64 `json:"available_ports"`
}

// ---- 主机管理 ----

// ListServers 获取所有主机列表
func (c *Client) ListServers(ctx context.Context) (map[string]ServerInfo, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/server/detail", nil)
	if err != nil {
		return nil, err
	}
	var data map[string]ServerInfo
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, fmt.Errorf("decode server list: %w", err)
	}
	return data, nil
}

// ListAreas 从主机列表中提取区域信息（按原始 server_area 去重）
// OpenIDCS 没有独立的区域接口，区域信息来自主机的 server_area 字段（格式 "code,name"）。
// Name 承载完整的 server_area 原始字符串（如 "CN,成都"），仅在生成稳定 ID 时内部提取 code 部分。
// 任意一台主机的 server_area 为空时，会合并到统一的 default 区域。
func (c *Client) ListAreas(ctx context.Context) ([]AreaInfo, error) {
	servers, err := c.ListServers(ctx)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var areas []AreaInfo
	hasDefault := false
	for _, info := range servers {
		raw := strings.TrimSpace(info.ServerArea)
		if raw == "" {
			hasDefault = true
			continue
		}
		if seen[raw] {
			continue
		}
		seen[raw] = true
		code, _ := parseServerArea(raw)
		areas = append(areas, AreaInfo{
			ID:    areaIDFromCode(code),
			Name:  raw,
			State: 1,
		})
	}
	if hasDefault {
		areas = append(areas, AreaInfo{
			ID:    defaultAreaID,
			Name:  defaultAreaName,
			State: 1,
		})
	}
	return areas, nil
}

// ListPlans 获取主机套餐列表
// 调用 /api/server/plans/{hs_name}，若接口不存在则返回空列表
func (c *Client) ListPlans(ctx context.Context, hsName string) ([]PlanInfo, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/server/plans/"+hsName, nil)
	if err != nil {
		// 接口不存在或主机不支持时，降级返回空列表
		pluginLog.Printf("ListPlans: %s not supported or failed: %v", hsName, err)
		return []PlanInfo{}, nil
	}
	var items []PlanInfo
	if err := json.Unmarshal(resp.Data, &items); err != nil {
		return []PlanInfo{}, nil
	}
	return items, nil
}

// UpsertPlanRequest 对应 HostAgent POST /api/server/plan/<hs_name> 的 plan_config 部分
// 字段单位对齐 HostAgent VMConfig：mem_num/hdd_num/flu_num 为 MB；speed_d/speed_u 为 Mbps
type UpsertPlanRequest struct {
	CPU         int `json:"cpu_num"`           // 处理器核心数
	MemoryMB    int `json:"mem_num"`           // 内存 MB
	DiskMB      int `json:"hdd_num"`           // 硬盘 MB
	BandwidthDn int `json:"speed_d"`           // 下行 Mbps
	BandwidthUp int `json:"speed_u"`           // 上行 Mbps
	TrafficMB   int `json:"flu_num,omitempty"` // 流量 MB
	GPUMemMB    int `json:"gpu_mem,omitempty"` // GPU 显存 MB
}

// UpsertPlan 新增或更新主机套餐（HostAgent POST /api/server/plan/<hs_name>）
// 强一致：失败会直接冒泡 error，供上游回滚本地事务
func (c *Client) UpsertPlan(ctx context.Context, hsName, planName string, cfg UpsertPlanRequest) error {
	_, err := c.doRequest(ctx, http.MethodPost, "/api/server/plan/"+hsName, map[string]any{
		"plan_name":   planName,
		"plan_config": cfg,
	})
	return err
}

// DeletePlan 删除主机套餐（HostAgent DELETE /api/server/plan/<hs_name>/<plan_name>）
func (c *Client) DeletePlan(ctx context.Context, hsName, planName string) error {
	_, err := c.doRequest(ctx, http.MethodDelete, "/api/server/plan/"+hsName+"/"+planName, nil)
	return err
}

// SetHostEnable 主机启用/禁用控制（HostAgent POST /api/server/powers/<hs_name>）
// enable=true 启用主机，false 禁用主机
// 对应小黑云侧"线路启用/禁用"的反向联动
func (c *Client) SetHostEnable(ctx context.Context, hsName string, enable bool) error {
	_, err := c.doRequest(ctx, http.MethodPost, "/api/server/powers/"+hsName, map[string]any{
		"enable": enable,
	})
	return err
}

// GetAvailablePorts 获取主机可分配端口列表
// 调用 /api/server/ports/{hs_name}，若接口不存在则返回空结构
func (c *Client) GetAvailablePorts(ctx context.Context, hsName string) (AvailablePorts, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/server/ports/"+hsName, nil)
	if err != nil {
		// 接口不存在或主机不支持时，降级返回空列表
		pluginLog.Printf("GetAvailablePorts: %s not supported or failed: %v", hsName, err)
		return AvailablePorts{HostName: hsName}, nil
	}
	var data AvailablePorts
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return AvailablePorts{HostName: hsName}, nil
	}
	return data, nil
}

// PickAvailablePort 从主机端口段中挑一个未被占用的 WAN 端口。
//
// 上游 CloudHost 的 /api/server/ports/{hs} 只返回 ports_start/ports_close 区间，
// 不提供 available_ports 列表，因此需要插件侧结合各 VM 已有的 NAT 规则自行计算；
// 若上游将来返回 available_ports，则优先使用该列表。
func (c *Client) PickAvailablePort(ctx context.Context, hsName string, exclude map[int]bool) (int, error) {
	avail, err := c.GetAvailablePorts(ctx, hsName)
	if err != nil {
		return 0, err
	}
	if len(avail.AvailablePorts) > 0 {
		for _, p := range avail.AvailablePorts {
			if p > 0 && !exclude[int(p)] {
				return int(p), nil
			}
		}
	}
	start, end := avail.PortsStart, avail.PortsClose
	if start <= 0 || end < start {
		return 0, fmt.Errorf("主机 %s 未配置可用端口段（ports_start/ports_close）", hsName)
	}
	used, err := c.usedWanPorts(ctx, hsName)
	if err != nil {
		return 0, err
	}
	for p := start; p <= end; p++ {
		if used[p] || exclude[p] {
			continue
		}
		return p, nil
	}
	return 0, fmt.Errorf("主机 %s 端口段 %d-%d 已无可用端口", hsName, start, end)
}

// usedWanPorts 汇总主机上所有虚拟机已占用的 WAN 端口。
func (c *Client) usedWanPorts(ctx context.Context, hsName string) (map[int]bool, error) {
	used := make(map[int]bool)
	vms, err := c.ListVMs(ctx, hsName)
	if err != nil {
		return used, err
	}
	for _, vm := range vms {
		if strings.TrimSpace(vm.VMUUID) == "" {
			continue
		}
		rules, err := c.ListNATRules(ctx, hsName, vm.VMUUID)
		if err != nil {
			continue
		}
		for _, r := range rules {
			if r.WanPort > 0 {
				used[r.WanPort] = true
			}
		}
	}
	return used, nil
}

// ---- 虚拟机管理 ----

// VMTask 上游异步任务摘要（CloudHost /api/system/async_task/list）
// 仅保留判断创建结果所需字段。
type VMTask struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Status      string `json:"status"` // pending / running / completed / failed / stopped / cancelled
	Progress    int    `json:"progress"`
	Message     string `json:"message"`
	Error       string `json:"error"`
	HsName      string `json:"hs_name"`
	VmUUID      string `json:"vm_uuid"`
	CreatedAt   string `json:"created_at"`
	CompletedAt string `json:"completed_at"`
}

// ListVMTasks 查询指定主机 + 虚拟机的创建任务，按创建时间倒序返回。
//
// 背景：CloudHost 的 /api/client/create/{hs} 是异步接口，只返回 task_id；
// 创建失败（例如镜像文件不存在）只会体现在任务状态里，VM 侧根本不会出现。
// 不查任务的话，插件只能一直返回 state=0，平台要等到 watchdog 超时才失败。
func (c *Client) ListVMTasks(ctx context.Context, hsName, vmUUID string) ([]VMTask, error) {
	q := url.Values{}
	q.Set("hs_name", hsName)
	q.Set("vm_uuid", vmUUID)
	q.Set("task_type", "create_vm")
	q.Set("page_size", "5")

	resp, err := c.doRequest(ctx, http.MethodGet, "/api/system/async_task/list?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Items []VMTask `json:"items"`
	}
	if err := json.Unmarshal(resp.Data, &payload); err != nil {
		return nil, fmt.Errorf("decode task list: %w", err)
	}
	return payload.Items, nil
}

// ListVMs 获取指定主机的虚拟机列表
// 兼容 HostAgent 两种返回格式：
//   - array 形态（新版）：[ { "vm_uuid": "...", "vm_name": "...", ... } ]
//   - dict 形态（旧版）：{ "<vm_uuid>": { "config": { "vm_uuid": "...", ... }, ... } }
func (c *Client) ListVMs(ctx context.Context, hsName string) ([]VMInfo, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/client/detail/"+hsName, nil)
	if err != nil {
		return nil, err
	}
	trimmed := bytes.TrimSpace(resp.Data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil, nil
	}
	// 优先尝试 array 形态
	if trimmed[0] == '[' {
		var items []VMInfo
		if err := json.Unmarshal(trimmed, &items); err != nil {
			return nil, fmt.Errorf("decode vm list (array): %w", err)
		}
		return items, nil
	}
	// 兼容 dict 形态：key 是 vm_uuid，value 包含 config 等信息
	if trimmed[0] == '{' {
		var m map[string]json.RawMessage
		if err := json.Unmarshal(trimmed, &m); err != nil {
			return nil, fmt.Errorf("decode vm list (dict): %w", err)
		}
		items := make([]VMInfo, 0, len(m))
		for key, raw := range m {
			var info VMInfo
			// 尝试直接解析为 VMInfo
			if err := json.Unmarshal(raw, &info); err == nil && info.VMUUID != "" {
				items = append(items, info)
				continue
			}
			// 尝试从 config 子对象解析
			var wrapper struct {
				Config json.RawMessage `json:"config"`
			}
			if err := json.Unmarshal(raw, &wrapper); err == nil && len(wrapper.Config) > 0 {
				if err := json.Unmarshal(wrapper.Config, &info); err == nil {
					if info.VMUUID == "" {
						info.VMUUID = key // dict 的 key 就是 vm_uuid
					}
					items = append(items, info)
					continue
				}
			}
			// 最后兜底：至少用 key 作为 vm_uuid
			items = append(items, VMInfo{VMUUID: key})
		}
		return items, nil
	}
	return nil, fmt.Errorf("decode vm list: unexpected format (starts with %q)", string(trimmed[:1]))
}

// GetVM 获取虚拟机详情
func (c *Client) GetVM(ctx context.Context, hsName, vmUUID string) (VMInfo, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/client/detail/"+hsName+"/"+vmUUID, nil)
	if err != nil {
		return VMInfo{}, err
	}
	// HostAgent /api/client/detail/{hs}/{vm} 返回结构：
	//   { "uuid": "...", "config": { VMConfig.__save__() }, "user_permissions": ..., ... }
	// VMConfig 的字段（vm_uuid, vm_flag, cpu_num, mem_num 等）嵌套在 config 子对象中。
	var wrapper struct {
		UUID   string          `json:"uuid"`
		Config json.RawMessage `json:"config"`
	}
	if err := json.Unmarshal(resp.Data, &wrapper); err != nil {
		return VMInfo{}, fmt.Errorf("decode vm detail wrapper: %w", err)
	}
	var info VMInfo
	if len(wrapper.Config) > 0 {
		if err := json.Unmarshal(wrapper.Config, &info); err != nil {
			return VMInfo{}, fmt.Errorf("decode vm config: %w", err)
		}
	}
	if info.VMUUID == "" {
		info.VMUUID = wrapper.UUID
	}
	if info.VMUUID == "" {
		info.VMUUID = vmUUID
	}
	return info, nil
}

// CreateVM 创建虚拟机
//
// HostAgent 的 /api/client/create/{hs} 在不同版本下返回体差异很大：
//   - 老版本：data = null，仅靠 code=200 + msg="虚拟机创建成功" 表示成功
//   - 新版本：data = { "vm_uuid": "<uuid>", ... }
//
// 因此这里的策略是：
//  1. 如果响应 data 里能直接解析出 vm_uuid，走快速路径直接返回；
//  2. 否则认为 HostAgent 未把 UUID 回写给我们，但 VM 已经创建成功，
//     通过 vm_name 回查 /api/client/detail/{hs} 反查 UUID（短时轮询，
//     应对上游索引刷新延迟）；
//  3. 回查仍失败再报错，保留原始 http_trace，便于上层看到"其实已经成功"。
//
// 注意：HostAgent 的 VMConfig 只有 vm_uuid 字段同时承担"名称"语义，所以
// 插件侧会把 req.Name 同时作为 vm_uuid 下发；这样老版本 HostAgent 即便
// 不回 data.vm_uuid，我们也能用这个 name 反向命中已创建的 VM。
func (c *Client) CreateVM(ctx context.Context, hsName, vmName string, req map[string]any) (string, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "/api/client/create/"+hsName, req)
	if err != nil {
		return "", err
	}

	// 快速路径：响应体带 vm_uuid（新版 HostAgent，本次已修复）
	isAsync := false
	if len(resp.Data) > 0 && !bytes.Equal(bytes.TrimSpace(resp.Data), []byte("null")) {
		var data map[string]any
		if err := json.Unmarshal(resp.Data, &data); err == nil {
			if uuid, ok := data["vm_uuid"].(string); ok && uuid != "" {
				return uuid, nil
			}
			// 异步模式：HostAgent 返回 task_id，VM 在后台创建，需要延长轮询窗口
			if tid, ok := data["task_id"].(string); ok && tid != "" {
				isAsync = true
			}
		}
	}

	// 兼容路径：HostAgent 未回 vm_uuid
	if strings.TrimSpace(vmName) == "" {
		return "", fmt.Errorf("vm_uuid not found in response and vm_name is empty, cannot recover")
	}
	// 异步模式：立即以 vmName 作临时 vmUUID 返回，不阻塞。
	// GetInstance 轮询中若 VM 尚未出现则返回 state=0，provision_worker 持续重试。
	if isAsync {
		return vmName, nil
	}
	// 同步兼容路径：短时轮询确认 VM 已注册
	uuid, lookupErr := c.lookupVMUUIDByName(ctx, hsName, vmName)
	if lookupErr == nil && uuid != "" {
		return uuid, nil
	}
	if lookupErr != nil {
		return "", fmt.Errorf("vm_uuid not found in response, fallback lookup by vm_name=%q failed: %w", vmName, lookupErr)
	}
	return "", fmt.Errorf("vm_uuid not found in response, fallback lookup by vm_name=%q returned no match", vmName)
}

// lookupVMUUIDByName 在指定主机上按 vm_name 短时反查 vm_uuid（同步兼容路径）。
// 兼容 HostAgent 新旧两种返回：
//   - 老：data = { "<vm_uuid>": { "config": { "vm_uuid": "..." } } }
//   - 新：data = [ { "vm_uuid": "...", "vm_name": "..." }, ... ]
func (c *Client) lookupVMUUIDByName(ctx context.Context, hsName, vmName string) (string, error) {
	const (
		maxAttempts = 2
		interval    = 500 * time.Millisecond
	)
	var lastErr error
	for i := 0; i < maxAttempts; i++ {
		if err := ctx.Err(); err != nil {
			if lastErr != nil {
				return "", fmt.Errorf("%v (after context done: %w)", lastErr, err)
			}
			return "", err
		}

		resp, err := c.doRequest(ctx, http.MethodGet, "/api/client/detail/"+hsName, nil)
		if err != nil {
			lastErr = fmt.Errorf("list vms: %w", err)
		} else if uuid, ok := scanVMListForName(resp.Data, vmName); ok {
			return uuid, nil
		} else {
			lastErr = fmt.Errorf("vm_name=%q not in list on host=%s", vmName, hsName)
		}

		if i < maxAttempts-1 {
			select {
			case <-time.After(interval):
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("vm_name=%q not found on host=%s", vmName, hsName)
	}
	return "", lastErr
}

// scanVMListForName 解析 HostAgent /api/client/detail/{hs} 的 data 字段，
// 查找 vm_name 对应的 vm_uuid。兼容两种形态：
//   - dict 形态：{ "<uuid>": { "uuid": "...", "config": { "vm_uuid": "..." } } }
//     → key 本身就是 vm_uuid（同时承担名字），直接比对 key
//   - array 形态：[ { "vm_uuid": "...", "vm_name": "..." } ]
//     → 比对 vm_name / vm_uuid 任一字段
//
// 任一字段匹配即视为命中；都没命中返回 ("", false)。
func scanVMListForName(raw []byte, vmName string) (string, bool) {
	if len(raw) == 0 {
		return "", false
	}
	trimmed := bytes.TrimSpace(raw)
	if bytes.Equal(trimmed, []byte("null")) {
		return "", false
	}

	// 尝试按 dict 解析
	if len(trimmed) > 0 && trimmed[0] == '{' {
		var m map[string]json.RawMessage
		if err := json.Unmarshal(trimmed, &m); err == nil {
			// 先看 key 是否直接等于 vmName（HostAgent 的 vm_uuid 就是 name）
			if _, ok := m[vmName]; ok {
				return vmName, true
			}
			// 再看每个 value.config.vm_uuid / value.uuid
			for k, v := range m {
				if uuid, ok := extractUUIDFromItem(v); ok && uuid == vmName {
					return k, true
				}
			}
		}
	}

	// 尝试按数组解析
	if len(trimmed) > 0 && trimmed[0] == '[' {
		var arr []json.RawMessage
		if err := json.Unmarshal(trimmed, &arr); err == nil {
			for _, item := range arr {
				if uuid, ok := extractUUIDFromItem(item); ok && uuid == vmName {
					return uuid, true
				}
				// 兼容显式 vm_name 字段
				var probe struct {
					VMName string `json:"vm_name"`
					VMUUID string `json:"vm_uuid"`
				}
				if err := json.Unmarshal(item, &probe); err == nil {
					if probe.VMName == vmName && probe.VMUUID != "" {
						return probe.VMUUID, true
					}
				}
			}
		}
	}

	return "", false
}

// extractUUIDFromItem 从单个 VM 条目里尽量解析出 vm_uuid。
// 条目可能是：
//   - { "uuid": "...", "config": { "vm_uuid": "..." } }
//   - { "vm_uuid": "..." }
//   - 纯 VMConfig 字典（带 vm_uuid）
func extractUUIDFromItem(raw json.RawMessage) (string, bool) {
	if len(raw) == 0 {
		return "", false
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return "", false
	}
	// 优先 config.vm_uuid
	if cfgRaw, ok := m["config"]; ok {
		var cfg struct {
			VMUUID string `json:"vm_uuid"`
		}
		if err := json.Unmarshal(cfgRaw, &cfg); err == nil && cfg.VMUUID != "" {
			return cfg.VMUUID, true
		}
	}
	// 顶层 vm_uuid / uuid
	var top struct {
		VMUUID string `json:"vm_uuid"`
		UUID   string `json:"uuid"`
	}
	if err := json.Unmarshal(raw, &top); err == nil {
		if top.VMUUID != "" {
			return top.VMUUID, true
		}
		if top.UUID != "" {
			return top.UUID, true
		}
	}
	return "", false
}

// UpdateVM 更新虚拟机配置
func (c *Client) UpdateVM(ctx context.Context, hsName, vmUUID string, req map[string]any) error {
	_, err := c.doRequest(ctx, http.MethodPut, "/api/client/update/"+hsName+"/"+vmUUID, req)
	return err
}

// ResetOSPassword 修改虚拟机操作系统密码（专用接口，不会强制重启）
// 调用 HostAgent POST /api/client/password/{hs_name}/{vm_uuid}
// HostAgent 会通过 QEMU Guest Agent 热修改密码（无需重启）
func (c *Client) ResetOSPassword(ctx context.Context, hsName, vmUUID string, password string) error {
	body := map[string]any{
		"type":     "os_password",
		"password": password,
	}
	_, err := c.doRequest(ctx, http.MethodPost, "/api/client/password/"+hsName+"/"+vmUUID, body)
	return err
}

// DeleteVM 删除虚拟机
func (c *Client) DeleteVM(ctx context.Context, hsName, vmUUID string) error {
	_, err := c.doRequest(ctx, http.MethodDelete, "/api/client/delete/"+hsName+"/"+vmUUID, nil)
	return err
}

// PowerVM 虚拟机电源控制
// action: S_START / H_CLOSE / S_RESET / S_PAUSE / S_RESUME
func (c *Client) PowerVM(ctx context.Context, hsName, vmUUID, action string) error {
	_, err := c.doRequest(ctx, http.MethodPost, "/api/client/powers/"+hsName+"/"+vmUUID, map[string]any{
		"action": action,
	})
	return err
}

// GetVMStatus 获取虚拟机状态（监控数据）
//
// HostAgent /api/client/status/{hs}/{vm} 返回的字段名在不同版本间可能不一致，
// 因此先用 map[string]any 灵活解析，再通过多候选字段名提取值，确保兼容性。
func (c *Client) GetVMStatus(ctx context.Context, hsName, vmUUID string) (map[string]any, error) {
	// limit 为最近 N 分钟的历史采样；上游前端同样使用 30/180/360。
	// 不传时部分版本只返回 power_status，history 恒为空。
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/client/status/"+hsName+"/"+vmUUID+"?limit=30", nil)
	if err != nil {
		return nil, err
	}
	var raw map[string]any
	if err := json.Unmarshal(resp.Data, &raw); err != nil {
		return nil, fmt.Errorf("decode vm status: %w", err)
	}
	if pluginLog != nil {
		preview := string(resp.Data)
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		pluginLog.Printf("[monitor] GetVMStatus hs=%s vm=%s raw=%s", hsName, vmUUID, preview)
	}
	return raw, nil
}

// GetRemoteAccess 获取控制台访问地址（VNC/SSH）
// HostAgent /api/client/remote 返回的 data 可能是：
//   - 字符串：直接是 VNC 控制台 URL
//   - JSON 对象：包含 console_url / terminal_url 等字段
func (c *Client) GetRemoteAccess(ctx context.Context, hsName, vmUUID string) (RemoteAccess, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/client/remote/"+hsName+"/"+vmUUID, nil)
	if err != nil {
		return RemoteAccess{}, err
	}
	// 尝试作为字符串解析（HostAgent 直接返回 URL 字符串的情况）
	var urlStr string
	if err := json.Unmarshal(resp.Data, &urlStr); err == nil && urlStr != "" {
		return RemoteAccess{ConsoleURL: urlStr}, nil
	}
	// 尝试作为 JSON 对象解析
	var access RemoteAccess
	if err := json.Unmarshal(resp.Data, &access); err != nil {
		return RemoteAccess{}, fmt.Errorf("decode remote access: %w", err)
	}
	return access, nil
}

// GetTempLoginURL 获取 OpenIDC 面板「一键登录」URL。
// 内部先调用 /api/client/temptoken/{hs_name}/{vm_uuid} 取得临时 token，
// 再拼接 {baseURL}/api/client/templogin?token={temp_token}，
// 与 FSPlugins/OpenIDC-SwapIDC/openidc.php 中 openidc_ClientArea() 行为一致。
// 若未能拿到 temp_token，则返回 baseURL 作为兜底（与 PHP 逻辑保持一致）。
func (c *Client) GetTempLoginURL(ctx context.Context, hsName, vmUUID string) (string, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/client/temptoken/"+hsName+"/"+vmUUID, nil)
	if err != nil {
		return "", err
	}
	var payload struct {
		TempToken string `json:"temp_token"`
	}
	if len(resp.Data) > 0 && string(resp.Data) != "null" {
		if err := json.Unmarshal(resp.Data, &payload); err != nil {
			return "", fmt.Errorf("decode temp token: %w", err)
		}
	}
	if strings.TrimSpace(payload.TempToken) == "" {
		// 降级：没有临时 token 时，直接给出管理地址，用户仍可自行登录。
		return c.baseURL, nil
	}
	return c.baseURL + "/api/client/templogin?token=" + url.QueryEscape(payload.TempToken), nil
}

// ---- OS 镜像 ----

// ListOSImages 获取主机 OS 镜像列表
// 返回 map[os_category][]OSImage，category ∈ {"iso", "system"}
// 同时兼容两种 HostAgent 数据格式：
//   - 新：list[OSConfig]（推荐；携带 sys_flag 启用状态）
//   - 旧：dict[name]=file（images_maps）或 dict[name]=[file,size]（system_maps）
func (c *Client) ListOSImages(ctx context.Context, hsName string) (map[string][]OSImage, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/client/os-images/"+hsName, nil)
	if err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 || string(resp.Data) == "null" {
		// 上游返回空 data（例如主机不存在却未返回错误 code）；给出明确错误
		return nil, fmt.Errorf("empty os-images data for host %q", hsName)
	}
	var cfg HSConfig
	if err := json.Unmarshal(resp.Data, &cfg); err != nil {
		// 错误信息里带上 raw data 前缀，方便排查字段类型不匹配的问题
		preview := string(resp.Data)
		if len(preview) > 512 {
			preview = preview[:512] + "..."
		}
		return nil, fmt.Errorf("decode os images config: %w (raw=%s)", err, preview)
	}
	// 调试日志：落地 HostAgent 原始 images_maps / system_maps 长度与前缀，
	// 部署后若仍看到 ISO 泄漏，可直接对比日志确定是"HostAgent 数据源里混入"还是"插件过滤 bug"。
	if pluginLog != nil {
		imgPreview := string(cfg.ImagesMaps)
		sysPreview := string(cfg.SystemMaps)
		if len(imgPreview) > 300 {
			imgPreview = imgPreview[:300] + "..."
		}
		if len(sysPreview) > 300 {
			sysPreview = sysPreview[:300] + "..."
		}
		pluginLog.Printf("[v2-iso-filter] ListOSImages hs=%s images_maps=%s system_maps=%s",
			hsName, imgPreview, sysPreview)
	}
	result := make(map[string][]OSImage)
	result["iso"] = append(result["iso"], parseOSImageField(cfg.ImagesMaps, "iso")...)
	result["system"] = append(result["system"], parseOSImageField(cfg.SystemMaps, "system")...)
	return result, nil
}

// ResolveOSImageFile 把平台下发的镜像显示名（sys_name）解析为上游真实文件名（sys_file）。
//
// 平台可能只下发镜像显示名（AutomationCreateHostRequest.OS），但上游 CreateVM 的
// os_name 必须是 system_maps 里的 sys_file。直接把显示名传下去会得到
// 「镜像文件不存在: /var/lib/vz/template/import/Debian Server 13.3 LTS」。
func (c *Client) ResolveOSImageFile(ctx context.Context, hsName, osName string) string {
	name := strings.TrimSpace(osName)
	if name == "" {
		return ""
	}
	// 已经是文件名（带常见扩展名）时直接使用，避免多余的上游请求
	if looksLikeImageFile(name) {
		return name
	}
	images, err := c.ListOSImages(ctx, hsName)
	if err != nil {
		pluginLog.Printf("ResolveOSImageFile: list images for %s failed: %v", hsName, err)
		return name
	}
	// 优先系统模板（CreateVM 只接受 system_maps），再退到 ISO 分类
	for _, category := range []string{"system", "iso"} {
		for _, img := range images[category] {
			if strings.EqualFold(strings.TrimSpace(img.Name), name) && strings.TrimSpace(img.File) != "" {
				return strings.TrimSpace(img.File)
			}
		}
	}
	pluginLog.Printf("ResolveOSImageFile: hs=%s 未找到显示名 %q 对应的 sys_file，按原值下发", hsName, name)
	return name
}

// looksLikeImageFile 判断名称是否已经是镜像文件名（而非显示名）。
func looksLikeImageFile(name string) bool {
	switch strings.ToLower(path.Ext(strings.TrimSpace(name))) {
	case ".qcow2", ".vmdk", ".img", ".raw", ".iso", ".template", ".vhd", ".vhdx", ".ova":
		return true
	}
	return false
}

// parseOSImageField 统一解析 images_maps / system_maps 两种历史格式
// category: "iso" 或 "system"，仅用于回填 OSImage.Category
func parseOSImageField(raw json.RawMessage, category string) []OSImage {
	if len(raw) == 0 {
		return nil
	}
	// 优先尝试新格式：list[OSConfig]
	var list []OSConfigItem
	if err := json.Unmarshal(raw, &list); err == nil {
		out := make([]OSImage, 0, len(list))
		for _, item := range list {
			name := strings.TrimSpace(item.SysName)
			file := strings.TrimSpace(item.SysFile)
			if name == "" && file == "" {
				continue
			}
			if name == "" {
				name = file
			}
			enabled := true
			if item.SysFlag != nil {
				enabled = *item.SysFlag
			}
			sizeGB := 0
			if s := strings.TrimSpace(item.SysSize); s != "" {
				_, _ = fmt.Sscanf(s, "%d", &sizeGB)
			}
			out = append(out, OSImage{
				Name:     name,
				File:     file,
				SizeGB:   sizeGB,
				Type:     normalizeSysType(item.SysType, name),
				Enabled:  enabled,
				Category: category,
			})
		}
		return out
	}
	// 回退：旧 dict 格式
	// images_maps 旧结构: map[name]=file（字符串值）
	var strMap map[string]string
	if err := json.Unmarshal(raw, &strMap); err == nil {
		out := make([]OSImage, 0, len(strMap))
		for name, file := range strMap {
			out = append(out, OSImage{
				Name:     name,
				File:     file,
				Type:     normalizeSysType("", name),
				Enabled:  true,
				Category: category,
			})
		}
		return out
	}
	// system_maps 旧结构: map[name]=[file, size_str]
	var arrMap map[string][]string
	if err := json.Unmarshal(raw, &arrMap); err == nil {
		out := make([]OSImage, 0, len(arrMap))
		for name, parts := range arrMap {
			img := OSImage{
				Name:     name,
				Type:     normalizeSysType("", name),
				Enabled:  true,
				Category: category,
			}
			if len(parts) > 0 {
				img.File = parts[0]
			}
			if len(parts) > 1 {
				_, _ = fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &img.SizeGB)
			}
			out = append(out, img)
		}
		return out
	}
	if pluginLog != nil {
		pluginLog.Printf("parseOSImageField: unknown format for category=%s, raw=%s", category, string(raw))
	}
	return nil
}

// normalizeSysType 将 HostAgent sys_type（WinNT/Linux/macOS）或名称推导为 linux/windows/macos
func normalizeSysType(sysType, fallbackName string) string {
	t := strings.ToLower(strings.TrimSpace(sysType))
	switch {
	case strings.Contains(t, "win"):
		return "windows"
	case strings.Contains(t, "mac") || strings.Contains(t, "osx") || strings.Contains(t, "darwin"):
		return "macos"
	case t != "":
		return "linux"
	}
	// 未提供 sys_type 时，回退按名称启发
	n := strings.ToLower(fallbackName)
	if strings.Contains(n, "win") {
		return "windows"
	}
	if strings.Contains(n, "mac") || strings.Contains(n, "osx") || strings.Contains(n, "darwin") {
		return "macos"
	}
	return "linux"
}

// MountISO 挂载 ISO 镜像
func (c *Client) MountISO(ctx context.Context, hsName, vmUUID, isoName string) error {
	_, err := c.doRequest(ctx, http.MethodPost, "/api/client/iso/mount/"+hsName+"/"+vmUUID, map[string]any{
		"iso_name": isoName,
	})
	return err
}

// ReinstallVM 重装系统（HostAgent POST /api/client/reinstall/{hs}/{vm}）
// osName 为系统模板文件名（来自 system_maps），password 为新系统密码（os_pass）。
func (c *Client) ReinstallVM(ctx context.Context, hsName, vmUUID, osName, password string) error {
	body := map[string]any{"os_name": osName}
	if password != "" {
		body["os_pass"] = password
	}
	_, err := c.doRequest(ctx, http.MethodPost, "/api/client/reinstall/"+hsName+"/"+vmUUID, body)
	return err
}

// ---- NAT 端口映射 ----

// ListNATRules 获取 NAT 规则列表
func (c *Client) ListNATRules(ctx context.Context, hsName, vmUUID string) ([]NATRule, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/client/natget/"+hsName+"/"+vmUUID, nil)
	if err != nil {
		return nil, err
	}
	var items []NATRule
	if err := json.Unmarshal(resp.Data, &items); err != nil {
		return nil, fmt.Errorf("decode nat rules: %w", err)
	}
	return items, nil
}

// AddNATRule 添加 NAT 规则
func (c *Client) AddNATRule(ctx context.Context, hsName, vmUUID string, hostPort, vmPort int, protocol, description string) error {
	_, err := c.doRequest(ctx, http.MethodPost, "/api/client/natadd/"+hsName+"/"+vmUUID, map[string]any{
		"wan_port": hostPort,
		"lan_port": vmPort,
		"protocol": protocol,
		"nat_tips": description,
	})
	return err
}

// DeleteNATRule 删除 NAT 规则
func (c *Client) DeleteNATRule(ctx context.Context, hsName, vmUUID string, ruleIndex int) error {
	_, err := c.doRequest(ctx, http.MethodDelete,
		fmt.Sprintf("/api/client/natdel/%s/%s/%d", hsName, vmUUID, ruleIndex), nil)
	return err
}

// ---- 备份管理 ----

// ListBackups 获取备份列表
func (c *Client) ListBackups(ctx context.Context, hsName, vmUUID string) ([]BackupInfo, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/client/backup/detail/"+hsName+"/"+vmUUID, nil)
	if err != nil {
		return nil, err
	}
	var items []BackupInfo
	if err := json.Unmarshal(resp.Data, &items); err != nil {
		return nil, fmt.Errorf("decode backups: %w", err)
	}
	return items, nil
}

// CreateBackup 创建备份
func (c *Client) CreateBackup(ctx context.Context, hsName, vmUUID, backupName, description string) error {
	body := map[string]any{}
	// HostAgent 创建 backup 使用 vm_tips 作为备份说明
	tips := description
	if tips == "" {
		tips = backupName
	}
	if tips == "" {
		tips = "备份 " + time.Now().Format("2006-01-02 15:04:05")
	}
	body["vm_tips"] = tips
	_, err := c.doRequest(ctx, http.MethodPost, "/api/client/backup/create/"+hsName+"/"+vmUUID, body)
	return err
}

// RestoreBackup 恢复备份
func (c *Client) RestoreBackup(ctx context.Context, hsName, vmUUID, backupName string) error {
	_, err := c.doRequest(ctx, http.MethodPost, "/api/client/backup/restore/"+hsName+"/"+vmUUID, map[string]any{
		"backup_name":            backupName,
		"power_on_after_restore": true,
	})
	return err
}

// DeleteBackup 删除备份
func (c *Client) DeleteBackup(ctx context.Context, hsName, vmUUID, backupName string) error {
	_, err := c.doRequest(ctx, http.MethodDelete, "/api/client/backup/delete/"+hsName+"/"+vmUUID, map[string]any{
		"backup_name": backupName,
	})
	return err
}

// ---- 快照管理 ----
//
// 直接对接主控的虚拟机快照 API（/api/client/snapshot/*），
// 与备份（/api/client/backup/*）是两套独立语义：
//   - 快照：平台原生快照树，可秒级回滚，不产生备份文件
//   - 备份：vzdump / 导出产物，独立存储介质
// 平台不具备快照能力时主控返回 400「当前虚拟化平台不支持虚拟机快照」，
// 上层据 errUnsupportedPlatform 转成 gRPC Unimplemented，便于宿主下发 501。

// SnapshotInfo 快照信息
type SnapshotInfo struct {
	SnapshotName string `json:"snapshot_name"`
	Description  string `json:"description"`
	Parent       string `json:"parent"`
	CreatedTime  string `json:"created_time"`
	State        string `json:"state"`
	Current      bool   `json:"current"`
}

// snapshotRawInfo 主控 /api/client/snapshot 返回的原始字段
type snapshotRawInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parent      string `json:"parent"`
	CreatedAt   int64  `json:"created_at"`
	State       string `json:"state"`
	Current     bool   `json:"current"`
}

// ListSnapshots 获取快照列表
func (c *Client) ListSnapshots(ctx context.Context, hsName, vmUUID string) ([]SnapshotInfo, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/client/snapshot/"+hsName+"/"+vmUUID, nil)
	if err != nil {
		return nil, err
	}
	var raws []snapshotRawInfo
	if len(resp.Data) > 0 && string(resp.Data) != "null" {
		if err := json.Unmarshal(resp.Data, &raws); err != nil {
			return nil, fmt.Errorf("decode snapshots: %w", err)
		}
	}
	out := make([]SnapshotInfo, 0, len(raws))
	for _, r := range raws {
		created := ""
		if r.CreatedAt > 0 {
			created = time.Unix(r.CreatedAt, 0).Format(time.RFC3339)
		}
		out = append(out, SnapshotInfo{
			SnapshotName: r.Name,
			Description:  r.Description,
			Parent:       r.Parent,
			CreatedTime:  created,
			State:        r.State,
			Current:      r.Current,
		})
	}
	return out, nil
}

// CreateSnapshot 创建快照
func (c *Client) CreateSnapshot(ctx context.Context, hsName, vmUUID, snapshotName, description string) error {
	body := map[string]any{
		"name":        strings.TrimSpace(snapshotName),
		"description": strings.TrimSpace(description),
	}
	_, err := c.doRequest(ctx, http.MethodPost, "/api/client/snapshot/"+hsName+"/"+vmUUID, body)
	return err
}

// RestoreSnapshot 回滚到指定快照
func (c *Client) RestoreSnapshot(ctx context.Context, hsName, vmUUID, snapshotName string) error {
	_, err := c.doRequest(ctx, http.MethodPost, "/api/client/snapshot/"+hsName+"/"+vmUUID+"/restore", map[string]any{
		"name": snapshotName,
	})
	return err
}

// DeleteSnapshot 删除指定快照（快照名走 query，主控侧避免路径字符冲突）
func (c *Client) DeleteSnapshot(ctx context.Context, hsName, vmUUID, snapshotName string) error {
	path := "/api/client/snapshot/" + hsName + "/" + vmUUID + "?name=" + url.QueryEscape(snapshotName)
	_, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	return err
}

// ---- 防火墙管理 ----
//
// 对接主控的虚拟机防火墙 API（/api/client/firewall/*），
// 规则语义与 PVE firewall/rules 对齐：
//   direction = type（in/out），method = action（ACCEPT/DROP/REJECT），
//   protocol = proto，ip = source，port = dport，priority = pos（平台位序）。
// 平台支持情况由主控判定（PVE / Hyper-V 支持，其余返回不支持）。

// FirewallRule 防火墙规则
type FirewallRule struct {
	RuleID    int    `json:"rule_id"`   // 平台位序 pos
	Direction string `json:"direction"` // in / out
	Protocol  string `json:"protocol"`  // tcp / udp / icmp / 空=任意
	Method    string `json:"method"`    // ACCEPT / DROP / REJECT
	Port      string `json:"port"`      // 目的端口，支持 "80" / "8000:9000"
	IP        string `json:"ip"`        // 来源地址/CIDR，空=任意
	Priority  int    `json:"priority"`  // 位序（与 RuleID 一致，便于前端排序展示）
	Enable    bool   `json:"enable"`
	Dest      string `json:"dest"`
	Sport     string `json:"sport"`
	Comment   string `json:"comment"`
	Macro     string `json:"macro"`
	Iface     string `json:"iface"`
}

// FirewallState 虚拟机防火墙整体状态
type FirewallState struct {
	Enabled        bool  `json:"enabled"`
	PolicyIn       string `json:"policy_in"`
	PolicyOut      string `json:"policy_out"`
	ClusterEnabled *bool `json:"cluster_enabled"`
	NodeEnabled    *bool `json:"node_enabled"`
	Effective      bool  `json:"effective"`
	Platform       string `json:"platform"`
}

// firewallRulePayload 主控防火墙规则 JSON
type firewallRulePayload struct {
	Pos     int    `json:"pos"`
	Type    string `json:"type"`
	Action  string `json:"action"`
	Enable  bool   `json:"enable"`
	Proto   string `json:"proto"`
	Source  string `json:"source"`
	Dest    string `json:"dest"`
	Sport   string `json:"sport"`
	Dport   string `json:"dport"`
	Comment string `json:"comment"`
	Macro   string `json:"macro"`
	Iface   string `json:"iface"`
}

// toFirewallRule 把主控规则转换为插件规则
func (r firewallRulePayload) toFirewallRule() FirewallRule {
	return FirewallRule{
		RuleID:    r.Pos,
		Direction: r.Type,
		Protocol:  r.Proto,
		Method:    r.Action,
		Port:      r.Dport,
		IP:        r.Source,
		Priority:  r.Pos,
		Enable:    r.Enable,
		Dest:      r.Dest,
		Sport:     r.Sport,
		Comment:   r.Comment,
		Macro:     r.Macro,
		Iface:     r.Iface,
	}
}

// toPayload 把插件规则转换为主控请求体
func (r FirewallRule) toPayload() map[string]any {
	body := map[string]any{
		"type":    strings.TrimSpace(r.Direction),
		"action":  strings.TrimSpace(r.Method),
		"enable":  r.Enable,
		"proto":   strings.TrimSpace(r.Protocol),
		"source":  strings.TrimSpace(r.IP),
		"dest":    strings.TrimSpace(r.Dest),
		"sport":   strings.TrimSpace(r.Sport),
		"dport":   strings.TrimSpace(r.Port),
		"comment": strings.TrimSpace(r.Comment),
		"macro":   strings.TrimSpace(r.Macro),
		"iface":   strings.TrimSpace(r.Iface),
	}
	return body
}

// ListFirewallRules 获取防火墙状态与规则列表
func (c *Client) ListFirewallRules(ctx context.Context, hsName, vmUUID string) ([]FirewallRule, *FirewallState, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/client/firewall/"+hsName+"/"+vmUUID, nil)
	if err != nil {
		return nil, nil, err
	}
	var payload struct {
		State *FirewallState        `json:"state"`
		Rules []firewallRulePayload `json:"rules"`
	}
	if len(resp.Data) > 0 && string(resp.Data) != "null" {
		if err := json.Unmarshal(resp.Data, &payload); err != nil {
			return nil, nil, fmt.Errorf("decode firewall: %w", err)
		}
	}
	out := make([]FirewallRule, 0, len(payload.Rules))
	for _, r := range payload.Rules {
		out = append(out, r.toFirewallRule())
	}
	return out, payload.State, nil
}

// AddFirewallRule 新增防火墙规则
func (c *Client) AddFirewallRule(ctx context.Context, hsName, vmUUID string, rule FirewallRule) error {
	_, err := c.doRequest(ctx, http.MethodPost, "/api/client/firewall/"+hsName+"/"+vmUUID+"/rules", rule.toPayload())
	return err
}

// UpdateFirewallRule 按平台位序更新防火墙规则
func (c *Client) UpdateFirewallRule(ctx context.Context, hsName, vmUUID string, ruleID int, rule FirewallRule) error {
	path := fmt.Sprintf("/api/client/firewall/%s/%s/rules/%d", hsName, vmUUID, ruleID)
	_, err := c.doRequest(ctx, http.MethodPut, path, rule.toPayload())
	return err
}

// DeleteFirewallRule 按平台位序删除防火墙规则
func (c *Client) DeleteFirewallRule(ctx context.Context, hsName, vmUUID string, ruleID int) error {
	path := fmt.Sprintf("/api/client/firewall/%s/%s/rules/%d", hsName, vmUUID, ruleID)
	_, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	return err
}

// SetFirewallOptions 保存防火墙开关与默认策略
func (c *Client) SetFirewallOptions(ctx context.Context, hsName, vmUUID string, enable bool, policyIn, policyOut string) error {
	body := map[string]any{"enable": enable}
	if v := strings.TrimSpace(policyIn); v != "" {
		body["policy_in"] = v
	}
	if v := strings.TrimSpace(policyOut); v != "" {
		body["policy_out"] = v
	}
	_, err := c.doRequest(ctx, http.MethodPut, "/api/client/firewall/"+hsName+"/"+vmUUID+"/options", body)
	return err
}

// ---- 网卡管理 ----

// NetworkCard 网卡配置
type NetworkCard struct {
	NicName   string   `json:"nic_name"`
	NicType   string   `json:"nic_type"`
	MacAddr   string   `json:"mac_addr"`
	IP4Addr   string   `json:"ip4_addr"`
	IP6Addr   string   `json:"ip6_addr"`
	NicGate   string   `json:"nic_gate"`
	NicMask   string   `json:"nic_mask"`
	DNSAddr   []string `json:"dns_addr"`
	IP6Prefix int      `json:"ip6_prefix"`
}

// ListNetworkCards 获取网卡列表（主控返回 map[nic_name]NCConfig）
func (c *Client) ListNetworkCards(ctx context.Context, hsName, vmUUID string) ([]NetworkCard, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/client/ipaddr/detail/"+hsName+"/"+vmUUID, nil)
	if err != nil {
		return nil, err
	}
	cards := make(map[string]NetworkCard)
	if len(resp.Data) > 0 && string(resp.Data) != "null" {
		if err := json.Unmarshal(resp.Data, &cards); err != nil {
			return nil, fmt.Errorf("decode network cards: %w", err)
		}
	}
	names := make([]string, 0, len(cards))
	for name := range cards {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]NetworkCard, 0, len(names))
	for _, name := range names {
		card := cards[name]
		if card.NicName == "" {
			card.NicName = name
		}
		out = append(out, card)
	}
	return out, nil
}

// CreateNetworkCard 新增网卡
func (c *Client) CreateNetworkCard(ctx context.Context, hsName, vmUUID string, card NetworkCard) error {
	_, err := c.doRequest(ctx, http.MethodPost, "/api/client/ipaddr/create/"+hsName+"/"+vmUUID, card)
	return err
}

// UpdateNetworkCard 更新网卡
func (c *Client) UpdateNetworkCard(ctx context.Context, hsName, vmUUID, nicName string, card NetworkCard) error {
	card.NicName = nicName
	path := "/api/hosts/" + hsName + "/vms/" + vmUUID + "/ip_addresses/" + url.PathEscape(nicName)
	_, err := c.doRequest(ctx, http.MethodPut, path, card)
	return err
}

// DeleteNetworkCard 删除网卡
func (c *Client) DeleteNetworkCard(ctx context.Context, hsName, vmUUID, nicName string) error {
	path := "/api/hosts/" + hsName + "/vms/" + vmUUID + "/ip_addresses/" + url.PathEscape(nicName)
	_, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	return err
}

// ---- 数据盘管理 ----

// DataDisk 数据盘配置
type DataDisk struct {
	HddName string `json:"hdd_name"`
	HddSize int    `json:"hdd_size"`
	HddFlag int    `json:"hdd_flag"`
	HddType int    `json:"hdd_type"`
	HddPath string `json:"hdd_path"`
	HddScsi string `json:"hdd_scsi"`
}

// ListDataDisks 获取数据盘列表（主控返回 map[hdd_name]SDConfig）
func (c *Client) ListDataDisks(ctx context.Context, hsName, vmUUID string) ([]DataDisk, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/client/hdd/detail/"+hsName+"/"+vmUUID, nil)
	if err != nil {
		return nil, err
	}
	disks := make(map[string]DataDisk)
	if len(resp.Data) > 0 && string(resp.Data) != "null" {
		if err := json.Unmarshal(resp.Data, &disks); err != nil {
			return nil, fmt.Errorf("decode data disks: %w", err)
		}
	}
	names := make([]string, 0, len(disks))
	for name := range disks {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]DataDisk, 0, len(names))
	for _, name := range names {
		disk := disks[name]
		if disk.HddName == "" {
			disk.HddName = name
		}
		out = append(out, disk)
	}
	return out, nil
}

// AttachDataDisk 挂载数据盘（hdd_size 单位 MB）
func (c *Client) AttachDataDisk(ctx context.Context, hsName, vmUUID string, sizeMB, hddType int) error {
	_, err := c.doRequest(ctx, http.MethodPost, "/api/client/hdd/mount/"+hsName+"/"+vmUUID, map[string]any{
		"hdd_size": sizeMB,
		"hdd_type": hddType,
	})
	return err
}

// DetachDataDisk 卸载数据盘（保留磁盘）
func (c *Client) DetachDataDisk(ctx context.Context, hsName, vmUUID, hddName string) error {
	_, err := c.doRequest(ctx, http.MethodPost, "/api/client/hdd/unmount/"+hsName+"/"+vmUUID, map[string]any{
		"hdd_name": hddName,
	})
	return err
}

// DeleteDataDisk 删除数据盘
func (c *Client) DeleteDataDisk(ctx context.Context, hsName, vmUUID, hddName string) error {
	_, err := c.doRequest(ctx, http.MethodDelete, "/api/client/hdd/delete/"+hsName+"/"+vmUUID, map[string]any{
		"hdd_name": hddName,
	})
	return err
}

// ---- 光盘管理 ----

// ISOInfo 光盘配置
type ISOInfo struct {
	IsoName string `json:"iso_name"`
	IsoFile string `json:"iso_file"`
	IsoPath string `json:"iso_path"`
	IsoHint string `json:"iso_hint"`
}

// ListISOs 获取已挂载光盘列表（主控返回 map[iso_name]IMConfig）
func (c *Client) ListISOs(ctx context.Context, hsName, vmUUID string) ([]ISOInfo, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/client/isos/detail/"+hsName+"/"+vmUUID, nil)
	if err != nil {
		return nil, err
	}
	isos := make(map[string]ISOInfo)
	if len(resp.Data) > 0 && string(resp.Data) != "null" {
		if err := json.Unmarshal(resp.Data, &isos); err != nil {
			return nil, fmt.Errorf("decode isos: %w", err)
		}
	}
	names := make([]string, 0, len(isos))
	for name := range isos {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]ISOInfo, 0, len(names))
	for _, name := range names {
		iso := isos[name]
		if iso.IsoName == "" {
			iso.IsoName = name
		}
		out = append(out, iso)
	}
	return out, nil
}

// MountISOImage 挂载光盘。
// 与既有 MountISO（只下发 iso_name）不同，这里按主控 IMConfig 语义下发镜像文件，
// 名称由主控自动生成，便于同一虚拟机挂载多张光驱。
func (c *Client) MountISOImage(ctx context.Context, hsName, vmUUID, isoFile, isoPath, isoHint string) error {
	_, err := c.doRequest(ctx, http.MethodPost, "/api/client/iso/mount/"+hsName+"/"+vmUUID, map[string]any{
		"iso_file": isoFile,
		"iso_path": isoPath,
		"iso_hint": isoHint,
	})
	return err
}

// UnmountISO 卸载光盘
func (c *Client) UnmountISO(ctx context.Context, hsName, vmUUID, isoName string) error {
	path := "/api/client/iso/unmount/" + hsName + "/" + vmUUID + "/" + url.PathEscape(isoName)
	_, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	return err
}

// ListHostISOImages 获取主机可挂载的 ISO 镜像名列表（供光盘挂载下拉选择）
func (c *Client) ListHostISOImages(ctx context.Context, hsName string) ([]string, error) {
	images, err := c.ListOSImages(ctx, hsName)
	if err != nil {
		return nil, err
	}
	list := images["iso"]
	out := make([]string, 0, len(list))
	for _, img := range list {
		file := strings.TrimSpace(img.File)
		if file == "" {
			continue
		}
		out = append(out, file)
	}
	return out, nil
}

// ---- 反向代理 ----

// ReverseProxy 反向代理配置
type ReverseProxy struct {
	ProxyIndex  int    `json:"proxy_index"`
	Domain      string `json:"domain"`
	BackendIP   string `json:"backend_ip"`
	BackendPort int    `json:"backend_port"`
	SSLEnabled  bool   `json:"ssl_enabled"`
	Description string `json:"description"`
}

// ListReverseProxies 获取反向代理列表
func (c *Client) ListReverseProxies(ctx context.Context, hsName, vmUUID string) ([]ReverseProxy, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/client/proxys/detail/"+hsName+"/"+vmUUID, nil)
	if err != nil {
		return nil, err
	}
	var raws []ReverseProxy
	if len(resp.Data) > 0 && string(resp.Data) != "null" {
		if err := json.Unmarshal(resp.Data, &raws); err != nil {
			return nil, fmt.Errorf("decode reverse proxies: %w", err)
		}
	}
	return raws, nil
}

// CreateReverseProxy 新增反向代理
func (c *Client) CreateReverseProxy(ctx context.Context, hsName, vmUUID string, proxy ReverseProxy) error {
	_, err := c.doRequest(ctx, http.MethodPost, "/api/client/proxys/create/"+hsName+"/"+vmUUID, map[string]any{
		"domain":       proxy.Domain,
		"backend_ip":   proxy.BackendIP,
		"backend_port": proxy.BackendPort,
		"ssl_enabled":  proxy.SSLEnabled,
		"description":  proxy.Description,
	})
	return err
}

// UpdateReverseProxy 更新反向代理
func (c *Client) UpdateReverseProxy(ctx context.Context, hsName, vmUUID string, proxyIndex int, proxy ReverseProxy) error {
	path := fmt.Sprintf("/api/client/proxys/update/%s/%s/%d", hsName, vmUUID, proxyIndex)
	_, err := c.doRequest(ctx, http.MethodPut, path, map[string]any{
		"domain":       proxy.Domain,
		"backend_ip":   proxy.BackendIP,
		"backend_port": proxy.BackendPort,
		"ssl_enabled":  proxy.SSLEnabled,
		"description":  proxy.Description,
	})
	return err
}

// DeleteReverseProxy 删除反向代理
func (c *Client) DeleteReverseProxy(ctx context.Context, hsName, vmUUID string, proxyIndex int) error {
	path := fmt.Sprintf("/api/client/proxys/delete/%s/%s/%d", hsName, vmUUID, proxyIndex)
	_, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	return err
}

// ---- 启动顺序 ----

// BootOption 启动项
type BootOption struct {
	EfiType bool   `json:"efi_type"`
	EfiName string `json:"efi_name"`
}

// GetBootOrder 获取启动顺序
func (c *Client) GetBootOrder(ctx context.Context, hsName, vmUUID string) ([]BootOption, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/client/efi-list/"+hsName+"/"+vmUUID, nil)
	if err != nil {
		return nil, err
	}
	var items []BootOption
	if len(resp.Data) > 0 && string(resp.Data) != "null" {
		if err := json.Unmarshal(resp.Data, &items); err != nil {
			return nil, fmt.Errorf("decode boot order: %w", err)
		}
	}
	return items, nil
}

// SetBootOrder 保存启动顺序
func (c *Client) SetBootOrder(ctx context.Context, hsName, vmUUID string, items []BootOption) error {
	_, err := c.doRequest(ctx, http.MethodPost, "/api/client/efi/setup/"+hsName+"/"+vmUUID, map[string]any{
		"efi_list": items,
	})
	return err
}

// ---- PCI / USB 直通 ----

// PCIDevice PCI 设备
type PCIDevice struct {
	PciKey  string `json:"pci_key"`
	GpuUUID string `json:"gpu_uuid"`
	GpuMdev string `json:"gpu_mdev"`
	GpuHint string `json:"gpu_hint"`
}

// ListPCIDevices 获取主机可直通 PCI 设备列表
func (c *Client) ListPCIDevices(ctx context.Context, hsName string) ([]PCIDevice, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/client/pci-list/"+hsName, nil)
	if err != nil {
		return nil, err
	}
	var items []PCIDevice
	if len(resp.Data) > 0 && string(resp.Data) != "null" {
		if err := json.Unmarshal(resp.Data, &items); err != nil {
			return nil, fmt.Errorf("decode pci list: %w", err)
		}
	}
	return items, nil
}

// SetupPCI 添加或移除 PCI 直通
func (c *Client) SetupPCI(ctx context.Context, hsName, vmUUID string, device PCIDevice, add bool) error {
	action := "remove"
	if add {
		action = "add"
	}
	_, err := c.doRequest(ctx, http.MethodPost, "/api/client/pci/setup/"+hsName+"/"+vmUUID, map[string]any{
		"pci_key":  device.PciKey,
		"gpu_uuid": device.GpuUUID,
		"gpu_mdev": device.GpuMdev,
		"gpu_hint": device.GpuHint,
		"action":   action,
	})
	return err
}

// USBDevice USB 设备
type USBDevice struct {
	UsbKey  string `json:"usb_key"`
	VidUUID string `json:"vid_uuid"`
	PidUUID string `json:"pid_uuid"`
	UsbHint string `json:"usb_hint"`
}

// ListUSBDevices 获取主机可直通 USB 设备列表
func (c *Client) ListUSBDevices(ctx context.Context, hsName string) ([]USBDevice, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/client/usb-list/"+hsName, nil)
	if err != nil {
		return nil, err
	}
	var items []USBDevice
	if len(resp.Data) > 0 && string(resp.Data) != "null" {
		if err := json.Unmarshal(resp.Data, &items); err != nil {
			return nil, fmt.Errorf("decode usb list: %w", err)
		}
	}
	return items, nil
}

// SetupUSB 添加或移除 USB 直通
func (c *Client) SetupUSB(ctx context.Context, hsName, vmUUID string, device USBDevice, add bool) error {
	action := "remove"
	if add {
		action = "add"
	}
	_, err := c.doRequest(ctx, http.MethodPost, "/api/client/usb/setup/"+hsName+"/"+vmUUID, map[string]any{
		"usb_key":  device.UsbKey,
		"vid_uuid": device.VidUUID,
		"pid_uuid": device.PidUUID,
		"usb_hint": device.UsbHint,
		"action":   action,
	})
	return err
}

// ---- 平台能力 ----

// PlatformCapabilities 主控下发的平台能力（/api/client/os-images/:hs_name）
type PlatformCapabilities struct {
	HotISO      bool `json:"hot_iso"`
	HotNIC      bool `json:"hot_nic"`
	OfflineOnly bool `json:"offline_only"`
	Firewall    bool `json:"firewall"`
	Snapshot    bool `json:"snapshot"`
}

// GetPlatformCapabilities 读取主机侧平台能力；失败时返回空能力（调用方按「不阻断」处理）
func (c *Client) GetPlatformCapabilities(ctx context.Context, hsName string) (PlatformCapabilities, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/client/os-images/"+hsName, nil)
	if err != nil {
		return PlatformCapabilities{}, err
	}
	var payload struct {
		ServerType   string               `json:"server_type"`
		Capabilities PlatformCapabilities `json:"capabilities"`
	}
	if len(resp.Data) > 0 && string(resp.Data) != "null" {
		if err := json.Unmarshal(resp.Data, &payload); err != nil {
			return PlatformCapabilities{}, fmt.Errorf("decode platform capabilities: %w", err)
		}
	}
	return payload.Capabilities, nil
}

// ---- 内部工具方法 ----

// needsCSRF 判断该方法是否需要携带 CSRF Token。
// 与上游 auth.CSRFMiddleware 保持一致：GET/HEAD/OPTIONS/TRACE 属于安全方法。
func needsCSRF(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return false
	}
	return true
}

// ensureCSRFToken 返回缓存的 CSRF Token，未命中时向主控申请。
func (c *Client) ensureCSRFToken(ctx context.Context) (string, error) {
	csrfCacheMu.RLock()
	token := csrfCache[c.baseURL]
	csrfCacheMu.RUnlock()
	if token != "" {
		return token, nil
	}

	endpoint := c.baseURL + "/api/csrf-token"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("create csrf request: %w", err)
	}
	// 该接口在 CloudHost 属于公开路由，带上 Bearer 无副作用；
	// 若对接的旧版本对 /api 全量鉴权，带上更稳妥。
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch csrf token: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read csrf response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch csrf token: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var payload struct {
		CSRF string `json:"csrf_token"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", fmt.Errorf("decode csrf response: %w", err)
	}
	if payload.CSRF == "" {
		return "", fmt.Errorf("fetch csrf token: empty csrf_token")
	}

	csrfCacheMu.Lock()
	csrfCache[c.baseURL] = payload.CSRF
	csrfCacheMu.Unlock()
	return payload.CSRF, nil
}

// invalidateCSRFToken 丢弃缓存中的 CSRF Token（上游重启或过期时调用）。
func (c *Client) invalidateCSRFToken() {
	csrfCacheMu.Lock()
	delete(csrfCache, c.baseURL)
	csrfCacheMu.Unlock()
}

// extractUpstreamError 提取非统一信封的错误原因。
// 例如 CloudHost 的 CSRF 校验失败会返回 {"error":"CSRF token 无效或已过期"}。
func extractUpstreamError(body []byte) string {
	var payload struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	return strings.TrimSpace(payload.Error)
}

// doRequest 执行 HTTP 请求，统一处理认证、CSRF、响应解析和日志。
// 主控的 CSRF Token 保存在内存中，主控重启后全部失效，
// 因此命中 CSRF 403 时刷新 Token 并重试一次（403 发生在业务处理前，重试安全）。
func (c *Client) doRequest(ctx context.Context, method, path string, body any) (apiResponse, error) {
	resp, err := c.doRequestOnce(ctx, method, path, body)
	if err == nil {
		return resp, nil
	}
	if errors.Is(err, errCSRFExpired) {
		pluginLog.Printf("HTTP %s %s CSRF token rejected, refreshing and retrying once", method, c.baseURL+path)
		c.invalidateCSRFToken()
		return c.doRequestOnce(ctx, method, path, body)
	}
	return resp, err
}

// doRequestOnce 执行单次 HTTP 请求，不做重试。
func (c *Client) doRequestOnce(ctx context.Context, method, path string, body any) (apiResponse, error) {
	endpoint := c.baseURL + path

	var reqBody []byte
	var bodyReader io.Reader
	if body != nil {
		var err error
		reqBody, err = json.Marshal(body)
		if err != nil {
			return apiResponse{}, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(reqBody)
	}

	pluginLog.Printf("HTTP %s %s body=%s", method, endpoint, string(reqBody))

	req, err := http.NewRequestWithContext(ctx, method, endpoint, bodyReader)
	if err != nil {
		pluginLog.Printf("HTTP %s %s create request error: %v", method, endpoint, err)
		return apiResponse{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if needsCSRF(method) {
		token, tokenErr := c.ensureCSRFToken(ctx)
		if tokenErr != nil {
			// 老版本主控或直连 HostAgent 可能没有 /api/csrf-token，
			// 此时降级为不带该头继续请求，由上游决定是否校验。
			pluginLog.Printf("HTTP %s %s fetch csrf token failed, fallback without header: %v", method, endpoint, tokenErr)
		} else if token != "" {
			req.Header.Set("X-CSRF-Token", token)
		}
	}

	start := time.Now()
	resp, err := c.http.Do(req)
	duration := time.Since(start)
	if err != nil {
		pluginLog.Printf("HTTP %s %s request failed (%dms): %v", method, endpoint, duration.Milliseconds(), err)
		c.emitLog(ctx, method, endpoint, string(reqBody), nil, nil, duration, err)
		return apiResponse{}, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		pluginLog.Printf("HTTP %s %s read response error (%dms): %v", method, endpoint, duration.Milliseconds(), err)
		c.emitLog(ctx, method, endpoint, string(reqBody), resp, nil, duration, err)
		return apiResponse{}, fmt.Errorf("read response: %w", err)
	}

	var parsed apiResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		pluginLog.Printf("HTTP %s %s decode error (%dms) status=%d body=%s err=%v",
			method, endpoint, duration.Milliseconds(), resp.StatusCode, string(respBody), err)
		c.emitLog(ctx, method, endpoint, string(reqBody), resp, respBody, duration, err)
		return apiResponse{}, fmt.Errorf("decode response: %w", err)
	}

	pluginLog.Printf("HTTP %s %s => status=%d code=%d msg=%q (%dms)",
		method, endpoint, resp.StatusCode, parsed.Code, parsed.Msg, duration.Milliseconds())

	// 非 2xx 响应可能不带统一信封（code/msg 均为空），
	// 例如 CSRF 拦截返回 {"error":"CSRF token 无效或已过期"}，
	// 这里统一把 error 字段纳入错误信息，避免上层只看到 code=0。
	if resp.StatusCode >= 400 {
		msg := extractUpstreamError(respBody)
		if msg == "" {
			msg = strings.TrimSpace(parsed.Msg)
		}
		if msg == "" {
			msg = "upstream returned no error message"
		}
		err = fmt.Errorf("openidc error: %s (HTTP %d)", msg, resp.StatusCode)
		c.emitLog(ctx, method, endpoint, string(reqBody), resp, respBody, duration, err)
		if resp.StatusCode == http.StatusForbidden && strings.Contains(strings.ToUpper(msg), "CSRF") {
			return apiResponse{}, fmt.Errorf("%w: %s", errCSRFExpired, msg)
		}
		return apiResponse{}, err
	}

	c.emitLog(ctx, method, endpoint, string(reqBody), resp, respBody, duration, nil)

	if parsed.Code != 200 {
		msg := strings.TrimSpace(parsed.Msg)
		if msg == "" {
			msg = fmt.Sprintf("upstream error code=%d", parsed.Code)
		}
		return apiResponse{}, fmt.Errorf("openidc error: %s", msg)
	}
	return parsed, nil
}

// emitLog 发送日志事件
func (c *Client) emitLog(ctx context.Context, method, endpoint, reqBody string, resp *http.Response, respBody []byte, duration time.Duration, err error) {
	if c.logFn == nil {
		return
	}
	reqPayload := map[string]any{
		"method": method,
		"url":    endpoint,
		"body":   reqBody,
	}
	var respPayload map[string]any
	if resp != nil {
		var bodyJSON any
		if len(respBody) > 0 {
			_ = json.Unmarshal(respBody, &bodyJSON)
		}
		respPayload = map[string]any{
			"status":      resp.StatusCode,
			"body":        string(respBody),
			"body_json":   bodyJSON,
			"duration_ms": duration.Milliseconds(),
		}
	} else {
		respPayload = map[string]any{
			"status":      0,
			"body":        "",
			"duration_ms": duration.Milliseconds(),
		}
	}
	success := err == nil
	message := "ok"
	if err != nil {
		message = err.Error()
	}
	c.logFn(ctx, HTTPLogEntry{
		Action:   method + " " + endpoint,
		Request:  reqPayload,
		Response: respPayload,
		Success:  success,
		Message:  message,
	})
}
