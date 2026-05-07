package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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
	ServerArea  string  `json:"server_area"`  // 主机所属区域（来自 HSConfig.server_area）
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
	VCPass      string `json:"vc_pass"`     // VNC 密码
	OSPass      string `json:"os_pass"`     // 系统密码
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
	VMUUID       string    `json:"vm_uuid"`
	Status       string    `json:"status"`
	PowerState   string    `json:"power_state"`
	CPUUsage     float64   `json:"cpu_usage"`
	MemoryUsage  int64     `json:"memory_usage"`
	MemoryTotal  int64     `json:"memory_total"`
	DiskReadRate int64     `json:"disk_read_rate"`
	DiskWriteRate int64   `json:"disk_write_rate"`
	NetworkRxRate float64  `json:"network_rx_rate"`
	NetworkTxRate float64  `json:"network_tx_rate"`
	UptimeSeconds int64    `json:"uptime_seconds"`
	IPAddresses  []string  `json:"ip_addresses"`
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
	SysSize string `json:"sys_size"` // 允许最低磁盘值-GB（字符串）
	SysType string `json:"sys_type"` // WinNT / Linux / macOS
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
	RuleIndex   int    `json:"rule_index"`
	WanPort     int    `json:"wan_port"`     // 宿主机端口（外部端口）
	LanPort     int    `json:"lan_port"`     // 虚拟机端口（内部端口）
	LanAddr     string `json:"lan_addr"`     // 虚拟机内网地址
	NatTips     string `json:"nat_tips"`     // 备注说明
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
// HostAgent server_area 的字符串约定格式为 "code,name"（例："CN,成都"）；
// 上游未填写时统一归并为 Code="default", Name="default"。
type AreaInfo struct {
	ID    int64  `json:"id"`
	Code  string `json:"code"`
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

// ListAreas 从主机列表中提取区域信息（按 code 去重）
// OpenIDCS 没有独立的区域接口，区域信息来自主机的 server_area 字段（格式 "code,name"）。
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
		code, name := parseServerArea(info.ServerArea)
		if code == defaultAreaCode {
			hasDefault = true
			continue
		}
		if seen[code] {
			continue
		}
		seen[code] = true
		areas = append(areas, AreaInfo{
			ID:    areaIDFromCode(code),
			Code:  code,
			Name:  name,
			State: 1,
		})
	}
	if hasDefault {
		areas = append(areas, AreaInfo{
			ID:    defaultAreaID,
			Code:  defaultAreaCode,
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

// ---- 虚拟机管理 ----

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
//   1. 如果响应 data 里能直接解析出 vm_uuid，走快速路径直接返回；
//   2. 否则认为 HostAgent 未把 UUID 回写给我们，但 VM 已经创建成功，
//      通过 vm_name 回查 /api/client/detail/{hs} 反查 UUID（短时轮询，
//      应对上游索引刷新延迟）；
//   3. 回查仍失败再报错，保留原始 http_trace，便于上层看到"其实已经成功"。
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
	if len(resp.Data) > 0 && !bytes.Equal(bytes.TrimSpace(resp.Data), []byte("null")) {
		var data map[string]any
		if err := json.Unmarshal(resp.Data, &data); err == nil {
			if uuid, ok := data["vm_uuid"].(string); ok && uuid != "" {
				return uuid, nil
			}
		}
	}

	// 兼容路径：老版本 HostAgent 没回 vm_uuid（或 data=null），按 vm_name 回查
	if strings.TrimSpace(vmName) == "" {
		return "", fmt.Errorf("vm_uuid not found in response and vm_name is empty, cannot recover")
	}
	uuid, lookupErr := c.lookupVMUUIDByName(ctx, hsName, vmName)
	if lookupErr == nil && uuid != "" {
		return uuid, nil
	}
	if lookupErr != nil {
		return "", fmt.Errorf("vm_uuid not found in response, fallback lookup by vm_name=%q failed: %w", vmName, lookupErr)
	}
	return "", fmt.Errorf("vm_uuid not found in response, fallback lookup by vm_name=%q returned no match", vmName)
}

// lookupVMUUIDByName 在指定主机上按 vm_name 反查 vm_uuid。
//
// HostAgent 创建接口返回成功后，新 VM 出现在 /api/client/detail/{hs} 列表里
// 可能有几百毫秒的延迟（尤其在并发创建时），所以这里做短时轮询，总等待时间 ~ 5s。
// 这个时长相对于 VM 创建整体耗时（十几秒起步）是可以接受的常数开销。
//
// 解析上放弃了预先假设数组/字典结构，用一份"双形态兼容"的 decodeVMMap 处理
// HostAgent 新旧两种返回：
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
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/client/status/"+hsName+"/"+vmUUID, nil)
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
	pluginLog.Printf("parseOSImageField: unknown format for category=%s, raw=%s", category, string(raw))
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
// HostAgent 没有独立的快照 API，"快照" 功能通过备份（backup）接口实现。
// ListSnapshots / CreateSnapshot / RestoreSnapshot / DeleteSnapshot 均映射到
// /api/client/backup/* 接口，对外保持 SnapshotInfo 的字段形态不变。

// SnapshotInfo 快照信息（由 backup 数据适配而来）
type SnapshotInfo struct {
	SnapshotName string `json:"snapshot_name"`
	Description  string `json:"description"`
	SizeMB       int    `json:"size_mb"`
	CreatedTime  string `json:"created_time"`
	CreatedBy    string `json:"created_by"`
	VMState      string `json:"vm_state"`
}

// backupRawInfo HostAgent backup 接口返回的原始字段
type backupRawInfo struct {
	BackupName  string `json:"backup_name"`
	BackupHint  string `json:"backup_hint"`
	BackupPath  string `json:"backup_path"`
	BackupTime  int64  `json:"backup_time"`
	CreatedTime string `json:"created_time"`
	SizeMB      int    `json:"size_mb"`
	CreatedBy   string `json:"created_by"`
	VMState     string `json:"vm_state"`
}

// ListSnapshots 获取快照列表（映射 backup 列表）
func (c *Client) ListSnapshots(ctx context.Context, hsName, vmUUID string) ([]SnapshotInfo, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/client/backup/detail/"+hsName+"/"+vmUUID, nil)
	if err != nil {
		return nil, err
	}
	// 兼容两种返回结构：
	//   1) 直接是数组：[{backup_name: ...}, ...]
	//   2) 对象包裹：{"backups": [...]} 或 {"config": {"backups": [...]}}
	var raws []backupRawInfo
	if len(resp.Data) > 0 && resp.Data[0] == '[' {
		if err := json.Unmarshal(resp.Data, &raws); err != nil {
			return nil, fmt.Errorf("decode backups: %w", err)
		}
	} else {
		var wrap struct {
			Backups []backupRawInfo `json:"backups"`
			Config  struct {
				Backups []backupRawInfo `json:"backups"`
			} `json:"config"`
		}
		if err := json.Unmarshal(resp.Data, &wrap); err != nil {
			return nil, fmt.Errorf("decode backups: %w", err)
		}
		if len(wrap.Backups) > 0 {
			raws = wrap.Backups
		} else {
			raws = wrap.Config.Backups
		}
	}
	out := make([]SnapshotInfo, 0, len(raws))
	for _, r := range raws {
		created := r.CreatedTime
		if created == "" && r.BackupTime > 0 {
			created = time.Unix(r.BackupTime, 0).Format(time.RFC3339)
		}
		out = append(out, SnapshotInfo{
			SnapshotName: r.BackupName,
			Description:  r.BackupHint,
			SizeMB:       r.SizeMB,
			CreatedTime:  created,
			CreatedBy:    r.CreatedBy,
			VMState:      r.VMState,
		})
	}
	return out, nil
}

// CreateSnapshot 创建快照（映射为创建 backup）
func (c *Client) CreateSnapshot(ctx context.Context, hsName, vmUUID, snapshotName, description string) error {
	body := map[string]any{}
	// HostAgent 创建 backup 使用 vm_tips 作为备份说明
	tips := description
	if tips == "" {
		tips = snapshotName
	}
	if tips == "" {
		tips = "快照 " + time.Now().Format("2006-01-02 15:04:05")
	}
	body["vm_tips"] = tips
	_, err := c.doRequest(ctx, http.MethodPost, "/api/client/backup/create/"+hsName+"/"+vmUUID, body)
	return err
}

// RestoreSnapshot 恢复快照（映射为恢复 backup）
func (c *Client) RestoreSnapshot(ctx context.Context, hsName, vmUUID, snapshotName string) error {
	_, err := c.doRequest(ctx, http.MethodPost, "/api/client/backup/restore/"+hsName+"/"+vmUUID, map[string]any{
		"vm_back": snapshotName,
	})
	return err
}

// DeleteSnapshot 删除快照（映射为删除 backup）
func (c *Client) DeleteSnapshot(ctx context.Context, hsName, vmUUID, snapshotName string) error {
	_, err := c.doRequest(ctx, http.MethodDelete, "/api/client/backup/delete/"+hsName+"/"+vmUUID, map[string]any{
		"vm_back": snapshotName,
	})
	return err
}

// ---- 防火墙管理 ----
//
// HostAgent 没有独立的防火墙 API，"防火墙" 功能通过 NAT 转发规则实现。
// ListFirewallRules / AddFirewallRule / DeleteFirewallRule 均映射到已有的 NAT API，
// 将 NAT 规则适配为前端防火墙表格期望的字段格式（direction / protocol / method / port / ip / priority）。

// FirewallRule 防火墙规则（由 NAT 规则适配而来）
type FirewallRule struct {
	RuleID    int    `json:"rule_id"`
	Direction string `json:"direction"` // In（NAT 转发默认入站）
	Protocol  string `json:"protocol"`  // tcp / udp
	Method    string `json:"method"`    // allowed（NAT 转发默认允许）
	Port      string `json:"port"`      // 映射端口，格式 "host_port → vm_port"
	IP        string `json:"ip"`        // 0.0.0.0（NAT 转发不限制来源 IP）
	Priority  int    `json:"priority"`  // 优先级（NAT 规则无此概念，默认 100）
}

// ListFirewallRules 获取防火墙规则列表（映射 NAT 转发规则）
// publicIP 为主机公网 IP，用于填充防火墙规则的 IP 字段；为空时回退到 "0.0.0.0"。
func (c *Client) ListFirewallRules(ctx context.Context, hsName, vmUUID, publicIP string) ([]FirewallRule, error) {
	natRules, err := c.ListNATRules(ctx, hsName, vmUUID)
	if err != nil {
		return nil, err
	}
	if publicIP == "" {
		publicIP = "0.0.0.0"
	}
	out := make([]FirewallRule, 0, len(natRules))
	for _, r := range natRules {
		out = append(out, FirewallRule{
			RuleID:    r.RuleIndex,
			Direction: "In",
			Protocol:  "tcp",
			Method:    "allowed",
			Port:      fmt.Sprintf("%d → %d", r.WanPort, r.LanPort),
			IP:        publicIP,
			Priority:  100,
		})
	}
	return out, nil
}

// AddFirewallRule 添加防火墙规则（映射为添加 NAT 转发规则）
// 前端传入的 port 字段作为 vm_port（目标端口），host_port 设为 0 让 HostAgent 自动分配。
func (c *Client) AddFirewallRule(ctx context.Context, hsName, vmUUID string, rule FirewallRule) error {
	vmPort := 0
	fmt.Sscanf(rule.Port, "%d", &vmPort)
	if vmPort <= 0 {
		return fmt.Errorf("invalid port: %s", rule.Port)
	}
	protocol := rule.Protocol
	if protocol == "" || protocol == "all" {
		protocol = "tcp"
	}
	return c.AddNATRule(ctx, hsName, vmUUID, 0, vmPort, protocol, "firewall-rule")
}

// DeleteFirewallRule 删除防火墙规则（映射为删除 NAT 转发规则）
func (c *Client) DeleteFirewallRule(ctx context.Context, hsName, vmUUID string, ruleID int) error {
	return c.DeleteNATRule(ctx, hsName, vmUUID, ruleID)
}

// ---- 内部工具方法 ----

// doRequest 执行 HTTP 请求，统一处理认证、响应解析和日志
func (c *Client) doRequest(ctx context.Context, method, path string, body any) (apiResponse, error) {
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
