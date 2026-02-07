package automation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"xiaoheiplay/internal/pkg/money"
	"xiaoheiplay/internal/usecase"
)

type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
	logFn   func(context.Context, HTTPLogEntry)
}

func NewClient(baseURL, apiKey string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 12 * time.Second
	}
	return &Client{
		baseURL: normalizeBaseURL(baseURL),
		apiKey:  apiKey,
		http: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) WithLogger(fn func(context.Context, HTTPLogEntry)) *Client {
	c.logFn = fn
	return c
}

type apiResponse struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

type findPortResponse struct {
	Code    int     `json:"code"`
	Msg     string  `json:"msg"`
	Content []int64 `json:"content"`
	Type    string  `json:"type"`
}

type HTTPLogEntry struct {
	Action   string
	Request  map[string]any
	Response map[string]any
	Success  bool
	Message  string
}

type hostInfoResp struct {
	ID            int64  `json:"id"`
	HostName      string `json:"host_name"`
	State         int    `json:"state"`
	CPU           int    `json:"cpu"`
	MemoryGB      int    `json:"memory"`
	DiskGB        int    `json:"hard_disks"`
	Bandwidth     int    `json:"bandwidth"`
	PanelPassword string `json:"panel_password"`
	VNCPassword   string `json:"vnc_password"`
	OSPassword    string `json:"os_password"`
	RemoteIP      string `json:"remote_ip"`
	EndTime       string `json:"end_time"`
}

type hostListItem struct {
	ID       int64  `json:"id"`
	HostName string `json:"host_name"`
	IP       string `json:"ip"`
}

type lineItem struct {
	ID       int64  `json:"id"`
	LineName string `json:"line_name"`
	Name     string `json:"name"`
	Remark   string `json:"remark"`
	LineAPI  string `json:"line_api"`
	AreaID   int64  `json:"area_id"`
	State    int    `json:"state"`
}

type imageItem struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type areaItem struct {
	ID    int64  `json:"id"`
	Name  string `json:"area_name"`
	State int    `json:"state"`
}
type productItem struct {
	ID         int64  `json:"id"`
	Name       string `json:"product_name"`
	CPU        int    `json:"host_cpu"`
	MemoryGB   int    `json:"host_ram"`
	DiskGB     int    `json:"host_data"`
	Bandwidth  int    `json:"bandwidth"`
	NatPortNum int    `json:"nat_port_num"`
	Price      string `json:"price"`
}

type monitorResp struct {
	StorageStats float64         `json:"StorageStats"`
	NetworkStats json.RawMessage `json:"NetworkStats"`
	CpuStats     float64         `json:"CpuStats"`
	MemoryStats  float64         `json:"MemoryStats"`
	Traffic      json.RawMessage `json:"Traffic"`
}

func (c *Client) CreateHost(ctx context.Context, req usecase.AutomationCreateHostRequest) (usecase.AutomationCreateHostResult, error) {
	params := url.Values{}
	params.Set("line_id", strconv.FormatInt(req.LineID, 10))
	params.Set("os", req.OS)
	params.Set("cpu", strconv.Itoa(req.CPU))
	params.Set("memory", strconv.Itoa(req.MemoryGB))
	params.Set("hard_disks", strconv.Itoa(req.DiskGB))
	params.Set("bandwidth", strconv.Itoa(req.Bandwidth))
	params.Set("expire_time", req.ExpireTime.Format("2006-01-02 15:04:05"))
	if req.HostName != "" {
		params.Set("host_name", req.HostName)
	}
	if req.SysPwd != "" {
		params.Set("sys_pwd", req.SysPwd)
	}
	if req.VNCPwd != "" {
		params.Set("vnc_pwd", req.VNCPwd)
	}
	if req.PortNum > 0 {
		params.Set("port_num", strconv.Itoa(req.PortNum))
	}
	if req.Snapshot > 0 {
		params.Set("snapshot", strconv.Itoa(req.Snapshot))
	}
	if req.Backups > 0 {
		params.Set("backups", strconv.Itoa(req.Backups))
	}

	endpoint := c.baseURL + "/create_host?" + params.Encode()
	resp, err := c.doRequest(ctx, http.MethodPost, endpoint, nil, "")
	if err != nil {
		return usecase.AutomationCreateHostResult{}, err
	}
	if resp.Code != 1 {
		return usecase.AutomationCreateHostResult{}, fmt.Errorf("automation error: %s", resp.Msg)
	}
	result := usecase.AutomationCreateHostResult{}
	if len(resp.Data) > 0 {
		var data map[string]any
		if err := json.Unmarshal(resp.Data, &data); err == nil {
			if v, ok := data["host_id"]; ok {
				result.HostID = toInt64(v)
			} else if v, ok := data["id"]; ok {
				result.HostID = toInt64(v)
			}
			result.Raw = data
		}
	}
	return result, nil
}

func (c *Client) GetHostInfo(ctx context.Context, hostID int64) (usecase.AutomationHostInfo, error) {
	form := url.Values{}
	// automation endpoints are inconsistent about host_id vs hostid
	form.Set("host_id", strconv.FormatInt(hostID, 10))
	form.Set("hostid", strconv.FormatInt(hostID, 10))
	endpoint := c.baseURL + "/hostinfo"
	resp, err := c.doRequest(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return usecase.AutomationHostInfo{}, err
	}
	if resp.Code != 1 {
		if strings.Contains(resp.Msg, "创建中") {
			return usecase.AutomationHostInfo{HostID: hostID, State: 0}, nil
		}
		return usecase.AutomationHostInfo{}, fmt.Errorf("automation error: %s", resp.Msg)
	}
	var data hostInfoResp
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return usecase.AutomationHostInfo{}, err
	}
	var expire *time.Time
	if data.EndTime != "" {
		if t, err := time.Parse("2006-01-02", data.EndTime); err == nil {
			expire = &t
		}
	}
	return usecase.AutomationHostInfo{
		HostID:        data.ID,
		HostName:      data.HostName,
		State:         data.State,
		CPU:           data.CPU,
		MemoryGB:      data.MemoryGB,
		DiskGB:        data.DiskGB,
		Bandwidth:     data.Bandwidth,
		PanelPassword: data.PanelPassword,
		VNCPassword:   data.VNCPassword,
		OSPassword:    data.OSPassword,
		RemoteIP:      data.RemoteIP,
		ExpireAt:      expire,
	}, nil
}

func (c *Client) ListHostSimple(ctx context.Context, searchTag string) ([]usecase.AutomationHostSimple, error) {
	params := url.Values{}
	params.Set("limit", "50")
	params.Set("pages", "1")
	if searchTag != "" {
		params.Set("search_tag", searchTag)
	}
	endpoint := c.baseURL + "/hostlist?" + params.Encode()
	resp, err := c.doRequest(ctx, http.MethodGet, endpoint, nil, "")
	if err != nil {
		return nil, err
	}
	if resp.Code != 1 {
		return nil, fmt.Errorf("automation error: %s", resp.Msg)
	}
	var items []hostListItem
	if err := json.Unmarshal(resp.Data, &items); err != nil {
		return nil, err
	}
	var out []usecase.AutomationHostSimple
	for _, item := range items {
		out = append(out, usecase.AutomationHostSimple{ID: item.ID, HostName: item.HostName, IP: item.IP})
	}
	return out, nil
}

func (c *Client) ElasticUpdate(ctx context.Context, req usecase.AutomationElasticUpdateRequest) error {
	params := url.Values{}
	params.Set("host_id", strconv.FormatInt(req.HostID, 10))
	if req.CPU != nil {
		params.Set("cpu", strconv.Itoa(*req.CPU))
	}
	if req.MemoryGB != nil {
		params.Set("memory", strconv.Itoa(*req.MemoryGB))
	}
	if req.DiskGB != nil {
		params.Set("hard_disks", strconv.Itoa(*req.DiskGB))
	}
	if req.Bandwidth != nil {
		params.Set("bandwidth", strconv.Itoa(*req.Bandwidth))
	}
	if req.PortNum != nil {
		params.Set("port_num", strconv.Itoa(*req.PortNum))
	}
	endpoint := c.baseURL + "/elastic_update?" + params.Encode()
	resp, err := c.doRequest(ctx, http.MethodPost, endpoint, nil, "")
	if err != nil {
		return err
	}
	if resp.Code != 1 {
		return fmt.Errorf("automation error: %s", resp.Msg)
	}
	return nil
}

func (c *Client) RenewHost(ctx context.Context, hostID int64, nextDueDate time.Time) error {
	form := url.Values{}
	form.Set("host_id", strconv.FormatInt(hostID, 10))
	form.Set("nextduedate", nextDueDate.Format("2006-01-02 15:04:05"))
	endpoint := c.baseURL + "/renew"
	resp, err := c.doRequest(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return err
	}
	if resp.Code != 1 {
		return fmt.Errorf("automation error: %s", resp.Msg)
	}
	return nil
}

func (c *Client) StartHost(ctx context.Context, hostID int64) error {
	return c.simpleHostAction(ctx, "/start", hostID)
}

func (c *Client) ShutdownHost(ctx context.Context, hostID int64) error {
	return c.simpleHostAction(ctx, "/shutdown", hostID)
}

func (c *Client) RebootHost(ctx context.Context, hostID int64) error {
	return c.simpleHostAction(ctx, "/reboot", hostID)
}

func (c *Client) ResetOS(ctx context.Context, hostID int64, templateID int64, password string) error {
	form := url.Values{}
	form.Set("host_id", strconv.FormatInt(hostID, 10))
	form.Set("template_id", strconv.FormatInt(templateID, 10))
	if password != "" {
		form.Set("password", password)
	}
	endpoint := c.baseURL + "/reset_os"
	resp, err := c.doRequest(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return err
	}
	if resp.Code != 1 {
		return fmt.Errorf("automation error: %s", resp.Msg)
	}
	return nil
}

func (c *Client) ResetOSPassword(ctx context.Context, hostID int64, password string) error {
	form := url.Values{}
	form.Set("host_id", strconv.FormatInt(hostID, 10))
	if password != "" {
		form.Set("password", password)
	}
	endpoint := c.baseURL + "/reset_password"
	resp, err := c.doRequest(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return err
	}
	if resp.Code != 1 {
		return fmt.Errorf("automation error: %s", resp.Msg)
	}
	return nil
}

func (c *Client) ListSnapshots(ctx context.Context, hostID int64) ([]usecase.AutomationSnapshot, error) {
	form := url.Values{}
	form.Set("host_id", strconv.FormatInt(hostID, 10))
	endpoint := c.baseURL + "/snapshot_list"
	resp, err := c.doRequest(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return nil, err
	}
	if resp.Code != 1 {
		return nil, fmt.Errorf("automation error: %s", resp.Msg)
	}
	items, err := parseAutomationList(resp.Data)
	if err != nil {
		return nil, err
	}
	out := make([]usecase.AutomationSnapshot, 0, len(items))
	for _, item := range items {
		out = append(out, usecase.AutomationSnapshot(item))
	}
	return out, nil
}

func (c *Client) CreateSnapshot(ctx context.Context, hostID int64) error {
	form := url.Values{}
	form.Set("host_id", strconv.FormatInt(hostID, 10))
	endpoint := c.baseURL + "/snapshot_add"
	resp, err := c.doRequest(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return err
	}
	if resp.Code != 1 {
		return fmt.Errorf("automation error: %s", resp.Msg)
	}
	return nil
}

func (c *Client) DeleteSnapshot(ctx context.Context, hostID int64, snapshotID int64) error {
	form := url.Values{}
	form.Set("host_id", strconv.FormatInt(hostID, 10))
	form.Set("id", strconv.FormatInt(snapshotID, 10))
	endpoint := c.baseURL + "/snapshot_del"
	resp, err := c.doRequest(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return err
	}
	if resp.Code != 1 {
		return fmt.Errorf("automation error: %s", resp.Msg)
	}
	return nil
}

func (c *Client) RestoreSnapshot(ctx context.Context, hostID int64, snapshotID int64) error {
	form := url.Values{}
	form.Set("host_id", strconv.FormatInt(hostID, 10))
	form.Set("id", strconv.FormatInt(snapshotID, 10))
	endpoint := c.baseURL + "/snapshot_restore"
	resp, err := c.doRequest(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return err
	}
	if resp.Code != 1 {
		return fmt.Errorf("automation error: %s", resp.Msg)
	}
	return nil
}

func (c *Client) ListBackups(ctx context.Context, hostID int64) ([]usecase.AutomationBackup, error) {
	form := url.Values{}
	form.Set("host_id", strconv.FormatInt(hostID, 10))
	endpoint := c.baseURL + "/backups_list"
	resp, err := c.doRequest(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return nil, err
	}
	if resp.Code != 1 {
		return nil, fmt.Errorf("automation error: %s", resp.Msg)
	}
	items, err := parseAutomationList(resp.Data)
	if err != nil {
		return nil, err
	}
	out := make([]usecase.AutomationBackup, 0, len(items))
	for _, item := range items {
		out = append(out, usecase.AutomationBackup(item))
	}
	return out, nil
}

func (c *Client) CreateBackup(ctx context.Context, hostID int64) error {
	form := url.Values{}
	form.Set("host_id", strconv.FormatInt(hostID, 10))
	endpoint := c.baseURL + "/backups_add"
	resp, err := c.doRequest(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return err
	}
	if resp.Code != 1 {
		return fmt.Errorf("automation error: %s", resp.Msg)
	}
	return nil
}

func (c *Client) DeleteBackup(ctx context.Context, hostID int64, backupID int64) error {
	form := url.Values{}
	form.Set("host_id", strconv.FormatInt(hostID, 10))
	form.Set("id", strconv.FormatInt(backupID, 10))
	endpoint := c.baseURL + "/backups_del"
	resp, err := c.doRequest(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return err
	}
	if resp.Code != 1 {
		return fmt.Errorf("automation error: %s", resp.Msg)
	}
	return nil
}

func (c *Client) RestoreBackup(ctx context.Context, hostID int64, backupID int64) error {
	form := url.Values{}
	form.Set("host_id", strconv.FormatInt(hostID, 10))
	form.Set("id", strconv.FormatInt(backupID, 10))
	endpoint := c.baseURL + "/backups_restore"
	resp, err := c.doRequest(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return err
	}
	if resp.Code != 1 {
		return fmt.Errorf("automation error: %s", resp.Msg)
	}
	return nil
}

func (c *Client) ListFirewallRules(ctx context.Context, hostID int64) ([]usecase.AutomationFirewallRule, error) {
	form := url.Values{}
	// automation endpoints are inconsistent about host_id vs hostid
	form.Set("host_id", strconv.FormatInt(hostID, 10))
	form.Set("hostid", strconv.FormatInt(hostID, 10))
	endpoint := c.baseURL + "/security_acl_list"
	resp, err := c.doRequest(ctx, http.MethodGet, endpoint+"?"+form.Encode(), nil, "")
	if err != nil {
		return nil, err
	}
	if resp.Code != 1 {
		return nil, fmt.Errorf("automation error: %s", resp.Msg)
	}
	items, err := parseAutomationList(resp.Data)
	if err != nil {
		return nil, err
	}
	out := make([]usecase.AutomationFirewallRule, 0, len(items))
	for _, item := range items {
		out = append(out, usecase.AutomationFirewallRule(item))
	}
	return out, nil
}

func (c *Client) AddFirewallRule(ctx context.Context, req usecase.AutomationFirewallRuleCreate) error {
	form := url.Values{}
	// automation endpoints are inconsistent about host_id vs hostid
	form.Set("host_id", strconv.FormatInt(req.HostID, 10))
	form.Set("hostid", strconv.FormatInt(req.HostID, 10))
	if req.Direction != "" {
		form.Set("direction", req.Direction)
	}
	if req.Protocol != "" {
		form.Set("protocol", req.Protocol)
	}
	if req.Method != "" {
		form.Set("method", req.Method)
	}
	if req.Port != "" {
		form.Set("port", req.Port)
	}
	if req.IP != "" {
		form.Set("ip", req.IP)
	}
	endpoint := c.baseURL + "/security_acl_add"
	resp, err := c.doRequest(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return err
	}
	if resp.Code != 1 {
		return fmt.Errorf("automation error: %s", resp.Msg)
	}
	return nil
}

func (c *Client) DeleteFirewallRule(ctx context.Context, hostID int64, ruleID int64) error {
	form := url.Values{}
	// automation endpoints are inconsistent about host_id vs hostid
	form.Set("host_id", strconv.FormatInt(hostID, 10))
	form.Set("hostid", strconv.FormatInt(hostID, 10))
	form.Set("id", strconv.FormatInt(ruleID, 10))
	endpoint := c.baseURL + "/security_acl_del"
	resp, err := c.doRequest(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return err
	}
	if resp.Code != 1 {
		return fmt.Errorf("automation error: %s", resp.Msg)
	}
	return nil
}

func (c *Client) ListPortMappings(ctx context.Context, hostID int64) ([]usecase.AutomationPortMapping, error) {
	form := url.Values{}
	// automation endpoints are inconsistent about host_id vs hostid
	form.Set("host_id", strconv.FormatInt(hostID, 10))
	form.Set("hostid", strconv.FormatInt(hostID, 10))
	endpoint := c.baseURL + "/nat_acl_list"
	resp, err := c.doRequest(ctx, http.MethodGet, endpoint+"?"+form.Encode(), nil, "")
	if err != nil {
		return nil, err
	}
	if resp.Code != 1 {
		return nil, fmt.Errorf("automation error: %s", resp.Msg)
	}
	items, err := parseAutomationList(resp.Data)
	if err != nil {
		return nil, err
	}
	out := make([]usecase.AutomationPortMapping, 0, len(items))
	for _, item := range items {
		out = append(out, usecase.AutomationPortMapping(item))
	}
	return out, nil
}

func (c *Client) AddPortMapping(ctx context.Context, req usecase.AutomationPortMappingCreate) error {
	form := url.Values{}
	// automation endpoints are inconsistent about host_id vs hostid
	form.Set("host_id", strconv.FormatInt(req.HostID, 10))
	form.Set("hostid", strconv.FormatInt(req.HostID, 10))
	if req.Name != "" {
		form.Set("name", req.Name)
	}
	if req.Sport != "" {
		form.Set("sport", req.Sport)
	}
	if req.Dport > 0 {
		form.Set("dport", strconv.FormatInt(req.Dport, 10))
	}
	endpoint := c.baseURL + "/add_port_host"
	resp, err := c.doRequest(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return err
	}
	if resp.Code != 1 {
		return fmt.Errorf("automation error: %s", resp.Msg)
	}
	return nil
}

func (c *Client) DeletePortMapping(ctx context.Context, hostID int64, mappingID int64) error {
	form := url.Values{}
	// automation endpoints are inconsistent about host_id vs hostid
	form.Set("host_id", strconv.FormatInt(hostID, 10))
	form.Set("hostid", strconv.FormatInt(hostID, 10))
	form.Set("id", strconv.FormatInt(mappingID, 10))
	endpoint := c.baseURL + "/remove_port_host"
	resp, err := c.doRequest(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return err
	}
	if resp.Code != 1 {
		return fmt.Errorf("automation error: %s", resp.Msg)
	}
	return nil
}

func (c *Client) FindPortCandidates(ctx context.Context, hostID int64, keywords string) ([]int64, error) {
	params := url.Values{}
	// automation endpoints are inconsistent about host_id vs hostid
	params.Set("host_id", strconv.FormatInt(hostID, 10))
	params.Set("hostid", strconv.FormatInt(hostID, 10))
	if strings.TrimSpace(keywords) != "" {
		params.Set("keywords", strings.TrimSpace(keywords))
	}
	endpoint := c.baseURL + "/findport?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("apikey", c.apiKey)
	start := time.Now()
	resp, err := c.http.Do(req)
	if err != nil {
		c.emitLog(ctx, http.MethodGet, endpoint, req.Header, "", nil, nil, time.Since(start), err)
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.emitLog(ctx, http.MethodGet, endpoint, req.Header, "", resp, nil, time.Since(start), err)
		return nil, err
	}
	var parsed findPortResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		c.emitLog(ctx, http.MethodGet, endpoint, req.Header, "", resp, body, time.Since(start), err)
		return nil, fmt.Errorf("decode response: %w", err)
	}
	c.emitLog(ctx, http.MethodGet, endpoint, req.Header, "", resp, body, time.Since(start), nil)
	if parsed.Code != 0 {
		msg := parsed.Msg
		if strings.TrimSpace(msg) == "" {
			msg = "findport failed"
		}
		return nil, fmt.Errorf("automation error: %s", msg)
	}
	if parsed.Content == nil {
		return []int64{}, nil
	}
	return parsed.Content, nil
}

func (c *Client) LockHost(ctx context.Context, hostID int64) error {
	return c.simpleHostAction(ctx, "/lock", hostID)
}

func (c *Client) UnlockHost(ctx context.Context, hostID int64) error {
	return c.simpleHostAction(ctx, "/unlock", hostID)
}

func (c *Client) DeleteHost(ctx context.Context, hostID int64) error {
	return c.simpleHostAction(ctx, "/delete", hostID)
}

func (c *Client) simpleHostAction(ctx context.Context, path string, hostID int64) error {
	form := url.Values{}
	form.Set("host_id", strconv.FormatInt(hostID, 10))
	endpoint := c.baseURL + path
	resp, err := c.doRequest(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return err
	}
	if resp.Code != 1 {
		return fmt.Errorf("automation error: %s", resp.Msg)
	}
	return nil
}

func (c *Client) GetPanelURL(ctx context.Context, hostName string, panelPassword string) (string, error) {
	form := url.Values{}
	form.Set("host_name", hostName)
	form.Set("panel_password", panelPassword)
	endpoint := c.baseURL + "/panel"
	client := &http.Client{
		Timeout: c.http.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("apikey", c.apiKey)
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		c.emitLog(ctx, http.MethodPost, endpoint, req.Header, form.Encode(), nil, nil, time.Since(start), err)
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	c.emitLog(ctx, http.MethodPost, endpoint, req.Header, form.Encode(), resp, body, time.Since(start), nil)
	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusTemporaryRedirect {
		return "", fmt.Errorf("automation panel failed: %d", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	if loc == "" {
		return "", fmt.Errorf("missing panel location")
	}
	return resolveRedirectURL(c.baseURL, loc), nil
}

func (c *Client) ListImages(ctx context.Context, lineID int64) ([]usecase.AutomationImage, error) {
	endpoint := c.baseURL + "/mirror_image"
	if lineID > 0 {
		endpoint += "?line_id=" + strconv.FormatInt(lineID, 10)
	}
	resp, err := c.doRequest(ctx, http.MethodGet, endpoint, nil, "")
	if err != nil {
		return nil, err
	}
	if resp.Code != 1 {
		return nil, fmt.Errorf("automation error: %s", resp.Msg)
	}
	var items []imageItem
	if err := json.Unmarshal(resp.Data, &items); err != nil {
		return nil, err
	}
	var out []usecase.AutomationImage
	for _, item := range items {
		out = append(out, usecase.AutomationImage{ImageID: item.ID, Name: item.Name, Type: item.Type})
	}
	return out, nil
}

func (c *Client) ListAreas(ctx context.Context) ([]usecase.AutomationArea, error) {
	endpoint := c.baseURL + "/area_list"
	resp, err := c.doRequest(ctx, http.MethodGet, endpoint, nil, "")
	if err != nil {
		return nil, err
	}
	if resp.Code != 1 {
		return nil, fmt.Errorf("automation error: %s", resp.Msg)
	}
	var items []areaItem
	if err := json.Unmarshal(resp.Data, &items); err != nil {
		return nil, err
	}
	out := make([]usecase.AutomationArea, 0, len(items))
	for _, item := range items {
		out = append(out, usecase.AutomationArea{ID: item.ID, Name: item.Name, State: item.State})
	}
	return out, nil
}

func (c *Client) ListLines(ctx context.Context) ([]usecase.AutomationLine, error) {
	endpoint := c.baseURL + "/line"
	resp, err := c.doRequest(ctx, http.MethodGet, endpoint, nil, "")
	if err != nil {
		return nil, err
	}
	if resp.Code != 1 {
		return nil, fmt.Errorf("automation error: %s", resp.Msg)
	}
	var items []lineItem
	if err := json.Unmarshal(resp.Data, &items); err != nil {
		return nil, err
	}
	out := make([]usecase.AutomationLine, 0, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.LineName)
		if name == "" {
			name = strings.TrimSpace(item.Name)
		}
		if name == "" {
			name = strings.TrimSpace(item.Remark)
		}
		if name == "" {
			name = strings.TrimSpace(item.LineAPI)
		}
		if name == "" {
			name = fmt.Sprintf("line-%d", item.ID)
		}
		out = append(out, usecase.AutomationLine{ID: item.ID, Name: name, AreaID: item.AreaID, State: item.State})
	}
	return out, nil
}

func (c *Client) ListProducts(ctx context.Context, lineID int64) ([]usecase.AutomationProduct, error) {
	endpoint := c.baseURL + "/product"
	if lineID > 0 {
		endpoint += "?line_id=" + strconv.FormatInt(lineID, 10)
	}
	resp, err := c.doRequest(ctx, http.MethodGet, endpoint, nil, "")
	if err != nil {
		return nil, err
	}
	if resp.Code != 1 {
		return nil, fmt.Errorf("automation error: %s", resp.Msg)
	}
	var raw map[string]productItem
	if err := json.Unmarshal(resp.Data, &raw); err != nil {
		return nil, err
	}
	out := make([]usecase.AutomationProduct, 0, len(raw))
	for _, item := range raw {
		price, _ := money.ParseNumberStringToCents(item.Price)
		out = append(out, usecase.AutomationProduct{
			ID:        item.ID,
			Name:      item.Name,
			CPU:       item.CPU,
			MemoryGB:  item.MemoryGB,
			DiskGB:    item.DiskGB,
			Bandwidth: item.Bandwidth,
			Price:     price,
			PortNum:   item.NatPortNum,
		})
	}
	return out, nil
}

func (c *Client) GetMonitor(ctx context.Context, hostID int64) (usecase.AutomationMonitor, error) {
	form := url.Values{}
	form.Set("host_id", strconv.FormatInt(hostID, 10))
	endpoint := c.baseURL + "/monitor"
	resp, err := c.doRequest(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return usecase.AutomationMonitor{}, err
	}
	if resp.Code != 1 {
		return usecase.AutomationMonitor{}, fmt.Errorf("automation error: %s", resp.Msg)
	}
	var data monitorResp
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return usecase.AutomationMonitor{}, err
	}
	bytesIn, bytesOut := parseNetworkStats(data.NetworkStats)
	return usecase.AutomationMonitor{
		CPUPercent:     int(math.Round(data.CpuStats)),
		MemoryPercent:  int(math.Round(data.MemoryStats)),
		StoragePercent: int(math.Round(data.StorageStats)),
		BytesIn:        bytesIn,
		BytesOut:       bytesOut,
	}, nil
}

func parseNetworkStats(raw json.RawMessage) (int64, int64) {
	if len(raw) == 0 {
		return 0, 0
	}
	var legacy struct {
		BytesSentPersec     int64 `json:"BytesSentPersec"`
		BytesReceivedPersec int64 `json:"BytesReceivedPersec"`
	}
	if err := json.Unmarshal(raw, &legacy); err == nil && (legacy.BytesSentPersec != 0 || legacy.BytesReceivedPersec != 0) {
		return legacy.BytesReceivedPersec, legacy.BytesSentPersec
	}
	var series [][]any
	if err := json.Unmarshal(raw, &series); err != nil || len(series) == 0 {
		return 0, 0
	}
	last := series[len(series)-1]
	if len(last) < 3 {
		return 0, 0
	}
	return toInt64(last[1]), toInt64(last[2])
}

func parseAutomationList(raw json.RawMessage) ([]map[string]any, error) {
	if len(raw) == 0 {
		return []map[string]any{}, nil
	}
	var items []map[string]any
	if err := json.Unmarshal(raw, &items); err == nil {
		return items, nil
	}
	var wrapper map[string]any
	if err := json.Unmarshal(raw, &wrapper); err == nil {
		if list, ok := wrapper["list"]; ok {
			if arr, ok := list.([]any); ok {
				out := make([]map[string]any, 0, len(arr))
				for _, item := range arr {
					if m, ok := item.(map[string]any); ok {
						out = append(out, m)
					}
				}
				return out, nil
			}
		}
	}
	return []map[string]any{}, nil
}

func (c *Client) GetVNCURL(ctx context.Context, hostID int64) (string, error) {
	form := url.Values{}
	form.Set("host_id", strconv.FormatInt(hostID, 10))
	endpoint := c.baseURL + "/vnc_view"
	client := &http.Client{
		Timeout: c.http.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("apikey", c.apiKey)
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		c.emitLog(ctx, http.MethodPost, endpoint, req.Header, form.Encode(), nil, nil, time.Since(start), err)
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	c.emitLog(ctx, http.MethodPost, endpoint, req.Header, form.Encode(), resp, body, time.Since(start), nil)
	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusTemporaryRedirect {
		return "", fmt.Errorf("automation vnc failed: %d", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	if loc == "" {
		return "", fmt.Errorf("missing vnc location")
	}
	return resolveRedirectURL(c.baseURL, loc), nil
}

func resolveRedirectURL(baseURL, location string) string {
	trimmed := strings.TrimSpace(location)
	if trimmed == "" {
		return location
	}
	parsed, err := url.Parse(trimmed)
	if err == nil && parsed.IsAbs() {
		return trimmed
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return trimmed
	}
	origin := &url.URL{Scheme: base.Scheme, Host: base.Host}
	if origin.Scheme == "" || origin.Host == "" {
		return trimmed
	}
	if parsed == nil {
		return trimmed
	}
	path := parsed.Path
	if path == "" && parsed.RawQuery != "" {
		path = "/"
	}
	if path != "" && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	origin.Path = path
	origin.RawQuery = parsed.RawQuery
	origin.Fragment = parsed.Fragment
	return origin.String()
}

func normalizeBaseURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return trimmed
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return strings.TrimRight(trimmed, "/")
	}
	if parsed.Path == "" || parsed.Path == "/" {
		parsed.Path = "/index.php/api/cloud"
	}
	return strings.TrimRight(parsed.String(), "/")
}

func (c *Client) doRequest(ctx context.Context, method, endpoint string, body io.Reader, contentType string) (apiResponse, error) {
	var reqBody []byte
	if body != nil {
		if b, err := io.ReadAll(body); err == nil {
			reqBody = b
			body = bytes.NewReader(b)
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return apiResponse{}, err
	}
	req.Header.Set("apikey", c.apiKey)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	start := time.Now()
	resp, err := c.http.Do(req)
	if err != nil {
		c.emitLog(ctx, method, endpoint, req.Header, string(reqBody), nil, nil, time.Since(start), err)
		return apiResponse{}, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		c.emitLog(ctx, method, endpoint, req.Header, string(reqBody), resp, nil, time.Since(start), err)
		return apiResponse{}, err
	}
	var parsed apiResponse
	if err := json.Unmarshal(b, &parsed); err != nil {
		c.emitLog(ctx, method, endpoint, req.Header, string(reqBody), resp, b, time.Since(start), err)
		return apiResponse{}, fmt.Errorf("decode response: %w", err)
	}
	c.emitLog(ctx, method, endpoint, req.Header, string(reqBody), resp, b, time.Since(start), nil)
	return parsed, nil
}

func toInt64(v any) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case int64:
		return t
	case string:
		id, _ := strconv.ParseInt(t, 10, 64)
		return id
	default:
		return 0
	}
}

func (c *Client) emitLog(ctx context.Context, method, endpoint string, headers http.Header, body string, resp *http.Response, respBody []byte, duration time.Duration, err error) {
	if c.logFn == nil {
		return
	}
	action := actionFromEndpoint(method, endpoint)
	reqPayload := map[string]any{
		"method":  method,
		"url":     endpoint,
		"headers": headerMap(headers),
		"body":    body,
	}
	var respPayload map[string]any
	if resp != nil {
		format, bodyJSON := detectBodyFormat(resp.Header, respBody)
		respPayload = map[string]any{
			"status":      resp.StatusCode,
			"headers":     headerMap(resp.Header),
			"body":        string(respBody),
			"format":      format,
			"duration_ms": duration.Milliseconds(),
		}
		if bodyJSON != nil {
			respPayload["body_json"] = bodyJSON
		}
	} else {
		respPayload = map[string]any{
			"status":      0,
			"headers":     map[string]string{},
			"body":        "",
			"format":      "none",
			"duration_ms": duration.Milliseconds(),
		}
	}
	success := err == nil
	message := "ok"
	if err != nil {
		message = err.Error()
	}
	c.logFn(ctx, HTTPLogEntry{
		Action:   action,
		Request:  reqPayload,
		Response: respPayload,
		Success:  success,
		Message:  message,
	})
}

func actionFromEndpoint(method, endpoint string) string {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Path == "" {
		return strings.TrimSpace(method) + " " + endpoint
	}
	return strings.TrimSpace(method) + " " + parsed.Path
}

func headerMap(h http.Header) map[string]string {
	out := make(map[string]string, len(h))
	for k, v := range h {
		if strings.EqualFold(k, "apikey") {
			out[k] = "***"
			continue
		}
		out[k] = strings.Join(v, ",")
	}
	return out
}

func detectBodyFormat(headers http.Header, body []byte) (string, any) {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return "empty", nil
	}
	contentType := strings.ToLower(headers.Get("Content-Type"))
	if strings.Contains(contentType, "application/json") || strings.Contains(contentType, "+json") || isLikelyJSON(trimmed) {
		var parsed any
		if err := json.Unmarshal(trimmed, &parsed); err == nil {
			return "json", parsed
		}
	}
	if strings.Contains(contentType, "text/html") || strings.Contains(contentType, "application/xhtml") || isLikelyHTML(trimmed) {
		return "html", nil
	}
	return "text", nil
}

func isLikelyJSON(body []byte) bool {
	switch body[0] {
	case '{', '[':
		return true
	default:
		return false
	}
}

func isLikelyHTML(body []byte) bool {
	sampleLen := 64
	if len(body) < sampleLen {
		sampleLen = len(body)
	}
	sample := strings.ToLower(string(body[:sampleLen]))
	return strings.HasPrefix(sample, "<!doctype") ||
		strings.HasPrefix(sample, "<html") ||
		strings.HasPrefix(sample, "<head") ||
		strings.HasPrefix(sample, "<body")
}
