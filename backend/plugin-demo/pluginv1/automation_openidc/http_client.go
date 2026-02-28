package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
		timeout = 15 * time.Second
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
	Code      int             `json:"code"`
	Msg       string          `json:"msg"`
	Data      json.RawMessage `json:"data"`
	Timestamp string          `json:"timestamp"`
}

// ---- 数据结构 ----

// ServerInfo 主机信息
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
}

// VMInfo 虚拟机信息
type VMInfo struct {
	VMUUID      string `json:"vm_uuid"`
	VMName      string `json:"vm_name"`
	DisplayName string `json:"display_name"`
	OSName      string `json:"os_name"`
	CPUNum      int    `json:"cpu_num"`
	MemNum      int    `json:"mem_num"`
	HDDNum      int    `json:"hdd_num"`
	GPUNum      int    `json:"gpu_num"`
	Status      string `json:"status"`
	PowerState  string `json:"power_state"`
	IPAddress   string `json:"ip_address"`
	CreatedTime string `json:"created_time"`
	Owner       string `json:"owner"`
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
type OSImage struct {
	Name    string `json:"name"`
	File    string `json:"file"`
	SizeMB  int    `json:"size_mb"`
	Version string `json:"version"`
	Arch    string `json:"architecture"`
}

// HSConfig 主机配置（/api/client/os-images/{hsName} 返回的原始结构）
type HSConfig struct {
	HostName   string              `json:"host_name"`
	ServerType string              `json:"server_type"`
	FilterName string              `json:"filter_name"`
	// images_maps: 显示名 → ISO 文件名（光驱镜像）
	ImagesMaps map[string]string   `json:"images_maps"`
	// system_maps: 系统名 → [vmdk文件, 版本号]（磁盘镜像）
	SystemMaps map[string][]string `json:"system_maps"`
}

// NATRule NAT 端口转发规则
type NATRule struct {
	RuleIndex   int    `json:"rule_index"`
	HostPort    int    `json:"host_port"`
	VMPort      int    `json:"vm_port"`
	Protocol    string `json:"protocol"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	CreatedTime string `json:"created_time"`
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
type AreaInfo struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	State int    `json:"state"`
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

// ListAreas 从主机列表中提取区域信息（去重）
// OpenIDCS 没有独立的区域接口，区域信息来自主机的 server_area 字段
func (c *Client) ListAreas(ctx context.Context) ([]AreaInfo, error) {
	servers, err := c.ListServers(ctx)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var areas []AreaInfo
	for _, info := range servers {
		area := strings.TrimSpace(info.ServerArea)
		if area == "" || seen[area] {
			continue
		}
		seen[area] = true
		areas = append(areas, AreaInfo{
			ID:    fnv64("area/" + area),
			Name:  area,
			State: 1,
		})
	}
	return areas, nil
}

// ListPlans OpenIDCS 不提供套餐接口，套餐由财务系统自行管理，返回空列表
func (c *Client) ListPlans(ctx context.Context, hsName string) ([]PlanInfo, error) {
	return []PlanInfo{}, nil
}

// GetAvailablePorts OpenIDCS 不提供端口候选接口，返回空结构
func (c *Client) GetAvailablePorts(ctx context.Context, hsName string) (AvailablePorts, error) {
	return AvailablePorts{HostName: hsName}, nil
}

// ---- 虚拟机管理 ----

// ListVMs 获取指定主机的虚拟机列表
func (c *Client) ListVMs(ctx context.Context, hsName string) ([]VMInfo, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/client/detail/"+hsName, nil)
	if err != nil {
		return nil, err
	}
	var items []VMInfo
	if err := json.Unmarshal(resp.Data, &items); err != nil {
		return nil, fmt.Errorf("decode vm list: %w", err)
	}
	return items, nil
}

// GetVM 获取虚拟机详情
func (c *Client) GetVM(ctx context.Context, hsName, vmUUID string) (VMInfo, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/client/detail/"+hsName+"/"+vmUUID, nil)
	if err != nil {
		return VMInfo{}, err
	}
	var info VMInfo
	if err := json.Unmarshal(resp.Data, &info); err != nil {
		return VMInfo{}, fmt.Errorf("decode vm info: %w", err)
	}
	return info, nil
}

// CreateVM 创建虚拟机
func (c *Client) CreateVM(ctx context.Context, hsName string, req map[string]any) (string, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "/api/client/create/"+hsName, req)
	if err != nil {
		return "", err
	}
	// 返回 vm_uuid
	var data map[string]any
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return "", fmt.Errorf("decode create vm response: %w", err)
	}
	if uuid, ok := data["vm_uuid"].(string); ok && uuid != "" {
		return uuid, nil
	}
	return "", fmt.Errorf("vm_uuid not found in response")
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
func (c *Client) GetVMStatus(ctx context.Context, hsName, vmUUID string) (VMStatus, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/client/status/"+hsName+"/"+vmUUID, nil)
	if err != nil {
		return VMStatus{}, err
	}
	var status VMStatus
	if err := json.Unmarshal(resp.Data, &status); err != nil {
		return VMStatus{}, fmt.Errorf("decode vm status: %w", err)
	}
	return status, nil
}

// GetRemoteAccess 获取控制台访问地址（VNC/SSH）
func (c *Client) GetRemoteAccess(ctx context.Context, hsName, vmUUID string) (RemoteAccess, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/client/remote/"+hsName+"/"+vmUUID, nil)
	if err != nil {
		return RemoteAccess{}, err
	}
	var access RemoteAccess
	if err := json.Unmarshal(resp.Data, &access); err != nil {
		return RemoteAccess{}, fmt.Errorf("decode remote access: %w", err)
	}
	return access, nil
}

// ---- OS 镜像 ----

// ListOSImages 获取主机 OS 镜像列表
// 返回 map[os_category][]OSImage
// API 实际返回 HSConfig 结构，其中 images_maps（ISO光驱镜像）和 system_maps（磁盘镜像）分别归入不同分类
func (c *Client) ListOSImages(ctx context.Context, hsName string) (map[string][]OSImage, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/client/os-images/"+hsName, nil)
	if err != nil {
		return nil, err
	}
	var cfg HSConfig
	if err := json.Unmarshal(resp.Data, &cfg); err != nil {
		return nil, fmt.Errorf("decode os images config: %w", err)
	}
	result := make(map[string][]OSImage)
	// images_maps: 显示名 → ISO 文件名（光驱/LiveCD 镜像）
	for displayName, fileName := range cfg.ImagesMaps {
		result["iso"] = append(result["iso"], OSImage{
			Name: displayName,
			File: fileName,
		})
	}
	// system_maps: 系统名 → [vmdk文件, 版本号]（磁盘镜像）
	for sysName, parts := range cfg.SystemMaps {
		img := OSImage{Name: sysName}
		if len(parts) > 0 {
			img.File = parts[0]
		}
		if len(parts) > 1 {
			img.Version = parts[1]
		}
		result["system"] = append(result["system"], img)
	}
	return result, nil
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
		"host_port":   hostPort,
		"vm_port":     vmPort,
		"protocol":    protocol,
		"description": description,
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
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/client/backup/list/"+hsName+"/"+vmUUID, nil)
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
	if backupName != "" {
		body["backup_name"] = backupName
	}
	if description != "" {
		body["description"] = description
	}
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
