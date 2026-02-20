package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	appshared "xiaoheiplay/internal/app/shared"
)

type Client struct {
	baseURL     string
	openAKID    string
	openSecret  string
	adminAPIKey string
	priceRate   float64
	goodsTypeID int64
	http        *http.Client
	logFn       func(context.Context, HTTPLogEntry)
}

type HTTPLogEntry struct {
	Action   string
	Request  map[string]any
	Response map[string]any
	Success  bool
	Message  string
}

func NewClient(baseURL, openAKID, openSecret, adminAPIKey string, priceRate float64, goodsTypeID int64, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 12 * time.Second
	}
	if priceRate <= 0 {
		priceRate = 1.0
	}
	return &Client{
		baseURL:     strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		openAKID:    strings.TrimSpace(openAKID),
		openSecret:  strings.TrimSpace(openSecret),
		adminAPIKey: strings.TrimSpace(adminAPIKey),
		priceRate:   priceRate,
		goodsTypeID: goodsTypeID,
		http:        &http.Client{Timeout: timeout},
	}
}

func (c *Client) WithLogger(fn func(context.Context, HTTPLogEntry)) *Client {
	c.logFn = fn
	return c
}

type regionDTO struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

type planGroupDTO struct {
	ID          int64  `json:"id"`
	RegionID    int64  `json:"region_id"`
	Name        string `json:"name"`
	LineID      int64  `json:"line_id"`
	Active      bool   `json:"active"`
	Visible     bool   `json:"visible"`
	GoodsTypeID int64  `json:"goods_type_id"`
}

type packageDTO struct {
	ID                   int64  `json:"id"`
	PlanGroupID          int64  `json:"plan_group_id"`
	ProductID            int64  `json:"product_id"`
	IntegrationPackageID int64  `json:"integration_package_id"`
	Name                 string `json:"name"`
	Cores                int    `json:"cores"`
	MemoryGB             int    `json:"memory_gb"`
	DiskGB               int    `json:"disk_gb"`
	BandwidthMbps        int    `json:"bandwidth_mbps"`
	MonthlyPrice         any    `json:"monthly_price"`
	PortNum              int    `json:"port_num"`
	CapacityRemaining    int    `json:"capacity_remaining"`
	Active               bool   `json:"active"`
	Visible              bool   `json:"visible"`
}

type imageDTO struct {
	ID      int64  `json:"id"`
	ImageID int64  `json:"image_id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Enabled bool   `json:"enabled"`
}

type vpsDTO struct {
	ID                   int64          `json:"id"`
	Name                 string         `json:"name"`
	Status               string         `json:"status"`
	AutomationState      int            `json:"automation_state"`
	CPU                  int            `json:"cpu"`
	MemoryGB             int            `json:"memory_gb"`
	DiskGB               int            `json:"disk_gb"`
	BandwidthMbps        int            `json:"bandwidth_mbps"`
	PortNum              int            `json:"port_num"`
	PackageID            int64          `json:"package_id"`
	PackageName          string         `json:"package_name"`
	SystemID             int64          `json:"system_id"`
	AutomationInstanceID string         `json:"automation_instance_id"`
	ExpireAt             *time.Time     `json:"expire_at"`
	AccessInfo           map[string]any `json:"access_info"`
	CreatedAt            time.Time      `json:"created_at"`
}

func (c *Client) ListAreas(ctx context.Context) ([]appshared.AutomationArea, error) {
	regions, err := c.listRegions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]appshared.AutomationArea, 0, len(regions))
	for _, r := range regions {
		state := 0
		if r.Active {
			state = 1
		}
		out = append(out, appshared.AutomationArea{ID: r.ID, Name: r.Name, State: state})
	}
	return out, nil
}

func (c *Client) ListLines(ctx context.Context) ([]appshared.AutomationLine, error) {
	plans, err := c.listPlanGroups(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]appshared.AutomationLine, 0, len(plans))
	for _, p := range plans {
		lineID := p.LineID
		if lineID <= 0 {
			lineID = p.ID
		}
		state := 0
		if p.Active {
			state = 1
		}
		out = append(out, appshared.AutomationLine{ID: lineID, Name: p.Name, AreaID: p.RegionID, State: state})
	}
	return out, nil
}

func (c *Client) ListImages(ctx context.Context, lineID int64) ([]appshared.AutomationImage, error) {
	q := map[string]string{}
	if lineID > 0 {
		q["line_id"] = strconv.FormatInt(lineID, 10)
	}
	var resp struct {
		Items []imageDTO `json:"items"`
	}
	if err := c.adminJSON(ctx, http.MethodGet, "/admin/api/v1/system-images", q, nil, &resp); err != nil {
		return nil, err
	}
	out := make([]appshared.AutomationImage, 0, len(resp.Items))
	for _, it := range resp.Items {
		if !it.Enabled {
			continue
		}
		id := it.ImageID
		if id <= 0 {
			id = it.ID
		}
		out = append(out, appshared.AutomationImage{ImageID: id, Name: it.Name, Type: it.Type})
	}
	return out, nil
}

func (c *Client) ListProducts(ctx context.Context, lineID int64) ([]appshared.AutomationProduct, error) {
	plans, err := c.listPlanGroups(ctx)
	if err != nil {
		return nil, err
	}
	planByID := map[int64]planGroupDTO{}
	planIDs := map[int64]struct{}{}
	for _, p := range plans {
		pid := p.ID
		line := p.LineID
		if line <= 0 {
			line = p.ID
		}
		if lineID > 0 && line != lineID {
			continue
		}
		planByID[pid] = p
		planIDs[pid] = struct{}{}
	}
	var resp struct {
		Items []packageDTO `json:"items"`
	}
	query := map[string]string{}
	if c.goodsTypeID > 0 {
		query["goods_type_id"] = strconv.FormatInt(c.goodsTypeID, 10)
	}
	if err := c.adminJSON(ctx, http.MethodGet, "/admin/api/v1/packages", query, nil, &resp); err != nil {
		return nil, err
	}
	out := make([]appshared.AutomationProduct, 0)
	for _, pkg := range resp.Items {
		if _, ok := planIDs[pkg.PlanGroupID]; !ok {
			continue
		}
		if !pkg.Active || !pkg.Visible {
			continue
		}
		extID := pkg.IntegrationPackageID
		if extID <= 0 {
			extID = pkg.ProductID
		}
		if extID <= 0 {
			extID = pkg.ID
		}
		out = append(out, appshared.AutomationProduct{
			ID:                extID,
			Name:              pkg.Name,
			CPU:               pkg.Cores,
			MemoryGB:          pkg.MemoryGB,
			DiskGB:            pkg.DiskGB,
			Bandwidth:         pkg.BandwidthMbps,
			Price:             int64(math.Round(float64(toCents(pkg.MonthlyPrice)) * c.priceRate)),
			PortNum:           pkg.PortNum,
			CapacityRemaining: pkg.CapacityRemaining,
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (c *Client) CreateHost(ctx context.Context, req appshared.AutomationCreateHostRequest) (appshared.AutomationCreateHostResult, error) {
	pkg, spec, err := c.pickPackageForRequest(ctx, req)
	if err != nil {
		return appshared.AutomationCreateHostResult{}, err
	}
	systemID, err := c.pickSystemID(ctx, req.LineID, req.OS)
	if err != nil {
		return appshared.AutomationCreateHostResult{}, err
	}
	months := 1
	if req.ExpireTime.After(time.Now()) {
		months = int(math.Ceil(req.ExpireTime.Sub(time.Now()).Hours() / 24.0 / 30.0))
		if months <= 0 {
			months = 1
		}
	}
	body := map[string]any{
		"items": []map[string]any{{
			"package_id": pkg.ID,
			"system_id":  systemID,
			"spec": map[string]any{
				"add_cores":       spec.AddCores,
				"add_mem_gb":      spec.AddMemGB,
				"add_disk_gb":     spec.AddDiskGB,
				"add_bw_mbps":     spec.AddBWMbps,
				"duration_months": months,
			},
			"qty": 1,
		}},
	}
	var createResp map[string]any
	if err := c.openJSON(ctx, http.MethodPost, "/api/v1/open/orders/instant/create", nil, body, &createResp); err != nil {
		return appshared.AutomationCreateHostResult{}, err
	}
	inst, err := c.findNewestVPS(ctx, pkg.ID, systemID, req.HostName)
	if err != nil {
		return appshared.AutomationCreateHostResult{}, err
	}
	return appshared.AutomationCreateHostResult{HostID: inst.ID, Raw: createResp}, nil
}

func (c *Client) GetHostInfo(ctx context.Context, hostID int64) (appshared.AutomationHostInfo, error) {
	inst, err := c.getVPS(ctx, hostID)
	if err != nil {
		return appshared.AutomationHostInfo{}, err
	}
	state := mapStatusState(inst.Status, inst.AutomationState)
	info := appshared.AutomationHostInfo{
		HostID:        inst.ID,
		HostName:      inst.Name,
		State:         state,
		CPU:           inst.CPU,
		MemoryGB:      inst.MemoryGB,
		DiskGB:        inst.DiskGB,
		Bandwidth:     inst.BandwidthMbps,
		RemoteIP:      toString(inst.AccessInfo["remote_ip"]),
		PanelPassword: toString(inst.AccessInfo["panel_password"]),
		VNCPassword:   toString(inst.AccessInfo["vnc_password"]),
		OSPassword:    toString(inst.AccessInfo["os_password"]),
		ExpireAt:      inst.ExpireAt,
	}
	return info, nil
}

func (c *Client) ListHostSimple(ctx context.Context, searchTag string) ([]appshared.AutomationHostSimple, error) {
	items, err := c.listVPS(ctx)
	if err != nil {
		return nil, err
	}
	tag := strings.ToLower(strings.TrimSpace(searchTag))
	out := make([]appshared.AutomationHostSimple, 0, len(items))
	for _, it := range items {
		ip := toString(it.AccessInfo["remote_ip"])
		if tag != "" {
			if !strings.Contains(strings.ToLower(it.Name), tag) && !strings.Contains(strings.ToLower(ip), tag) {
				continue
			}
		}
		out = append(out, appshared.AutomationHostSimple{ID: it.ID, HostName: it.Name, IP: ip})
	}
	return out, nil
}

func (c *Client) StartHost(ctx context.Context, hostID int64) error {
	return c.openAction(ctx, hostID, "start")
}
func (c *Client) ShutdownHost(ctx context.Context, hostID int64) error {
	return c.openAction(ctx, hostID, "shutdown")
}
func (c *Client) RebootHost(ctx context.Context, hostID int64) error {
	return c.openAction(ctx, hostID, "reboot")
}
func (c *Client) ResetOS(ctx context.Context, hostID int64, templateID int64, password string) error {
	body := map[string]any{"template_id": templateID, "password": password}
	return c.openJSON(ctx, http.MethodPost, fmt.Sprintf("/api/v1/open/vps/%d/reset-os", hostID), nil, body, nil)
}
func (c *Client) ResetOSPassword(ctx context.Context, hostID int64, password string) error {
	body := map[string]any{"password": password}
	return c.openJSON(ctx, http.MethodPost, fmt.Sprintf("/api/v1/open/vps/%d/reset-os-password", hostID), nil, body, nil)
}
func (c *Client) RenewHost(ctx context.Context, hostID int64, nextDueDate time.Time) error {
	months := int(math.Ceil(nextDueDate.Sub(time.Now()).Hours() / 24.0 / 30.0))
	if months <= 0 {
		months = 1
	}
	body := map[string]any{"vps_id": hostID, "duration_months": months}
	return c.openJSON(ctx, http.MethodPost, "/api/v1/open/orders/instant/renew", nil, body, nil)
}

func (c *Client) ElasticUpdate(ctx context.Context, req appshared.AutomationElasticUpdateRequest) error {
	inst, err := c.getVPS(ctx, req.HostID)
	if err != nil {
		return err
	}
	pkg, err := c.findPackageByID(ctx, inst.PackageID)
	if err != nil {
		return err
	}
	targetCPU := inst.CPU
	targetMem := inst.MemoryGB
	targetDisk := inst.DiskGB
	targetBW := inst.BandwidthMbps
	if req.CPU != nil {
		targetCPU = *req.CPU
	}
	if req.MemoryGB != nil {
		targetMem = *req.MemoryGB
	}
	if req.DiskGB != nil {
		targetDisk = *req.DiskGB
	}
	if req.Bandwidth != nil {
		targetBW = *req.Bandwidth
	}
	spec := map[string]any{
		"add_cores":   maxInt(targetCPU-pkg.Cores, 0),
		"add_mem_gb":  maxInt(targetMem-pkg.MemoryGB, 0),
		"add_disk_gb": maxInt(targetDisk-pkg.DiskGB, 0),
		"add_bw_mbps": maxInt(targetBW-pkg.BandwidthMbps, 0),
	}
	body := map[string]any{
		"vps_id":            req.HostID,
		"spec":              spec,
		"target_package_id": pkg.ID,
		"reset_addons":      false,
	}
	return c.openJSON(ctx, http.MethodPost, "/api/v1/open/orders/instant/resize", nil, body, nil)
}

func (c *Client) LockHost(ctx context.Context, hostID int64) error {
	if strings.TrimSpace(c.adminAPIKey) != "" {
		return c.adminJSON(ctx, http.MethodPost, fmt.Sprintf("/admin/api/v1/vps/%d/lock", hostID), nil, nil, nil)
	}
	return c.ShutdownHost(ctx, hostID)
}

func (c *Client) UnlockHost(ctx context.Context, hostID int64) error {
	if strings.TrimSpace(c.adminAPIKey) != "" {
		return c.adminJSON(ctx, http.MethodPost, fmt.Sprintf("/admin/api/v1/vps/%d/unlock", hostID), nil, nil, nil)
	}
	return c.StartHost(ctx, hostID)
}

func (c *Client) DeleteHost(ctx context.Context, hostID int64) error {
	if strings.TrimSpace(c.adminAPIKey) == "" {
		return fmt.Errorf("admin_api_key required for delete")
	}
	return c.adminJSON(ctx, http.MethodPost, fmt.Sprintf("/admin/api/v1/vps/%d/delete", hostID), nil, nil, nil)
}

func (c *Client) GetPanelURL(ctx context.Context, hostName string, panelPassword string) (string, error) {
	_ = panelPassword
	items, err := c.listVPS(ctx)
	if err != nil {
		return "", err
	}
	name := strings.TrimSpace(hostName)
	var hostID int64
	for _, it := range items {
		if strings.EqualFold(strings.TrimSpace(it.Name), name) {
			hostID = it.ID
			break
		}
	}
	if hostID == 0 {
		return "", fmt.Errorf("instance not found by name: %s", hostName)
	}
	return c.openRedirect(ctx, http.MethodGet, fmt.Sprintf("/api/v1/open/vps/%d/panel", hostID), nil)
}

func (c *Client) GetVNCURL(ctx context.Context, hostID int64) (string, error) {
	location, err := c.openRedirect(ctx, http.MethodGet, fmt.Sprintf("/api/v1/open/vps/%d/vnc", hostID), nil)
	if err != nil {
		return "", err
	}
	return location, nil
}

func (c *Client) GetMonitor(ctx context.Context, hostID int64) (appshared.AutomationMonitor, error) {
	var resp map[string]any
	if err := c.openJSON(ctx, http.MethodGet, fmt.Sprintf("/api/v1/open/vps/%d/monitor", hostID), nil, nil, &resp); err != nil {
		return appshared.AutomationMonitor{}, err
	}
	return appshared.AutomationMonitor{
		CPUPercent:     int(toInt64(resp["cpu"])),
		MemoryPercent:  int(toInt64(resp["memory"])),
		BytesIn:        toInt64(resp["bytes_in"]),
		BytesOut:       toInt64(resp["bytes_out"]),
		StoragePercent: int(toInt64(resp["storage"])),
	}, nil
}

func (c *Client) ListPortMappings(ctx context.Context, hostID int64) ([]appshared.AutomationPortMapping, error) {
	var resp struct {
		Data []map[string]any `json:"data"`
	}
	if err := c.openJSON(ctx, http.MethodGet, fmt.Sprintf("/api/v1/open/vps/%d/ports", hostID), nil, nil, &resp); err != nil {
		return nil, err
	}
	out := make([]appshared.AutomationPortMapping, 0, len(resp.Data))
	for _, it := range resp.Data {
		out = append(out, appshared.AutomationPortMapping(it))
	}
	return out, nil
}

func (c *Client) AddPortMapping(ctx context.Context, req appshared.AutomationPortMappingCreate) error {
	body := map[string]any{"name": req.Name, "sport": req.Sport, "dport": req.Dport}
	return c.openJSON(ctx, http.MethodPost, fmt.Sprintf("/api/v1/open/vps/%d/ports", req.HostID), nil, body, nil)
}

func (c *Client) DeletePortMapping(ctx context.Context, hostID int64, mappingID int64) error {
	return c.openJSON(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/open/vps/%d/ports/%d", hostID, mappingID), nil, nil, nil)
}

func (c *Client) FindPortCandidates(ctx context.Context, hostID int64, keywords string) ([]int64, error) {
	q := map[string]string{"keywords": strings.TrimSpace(keywords)}
	var resp struct {
		Data []int64 `json:"data"`
	}
	if err := c.openJSON(ctx, http.MethodGet, fmt.Sprintf("/api/v1/open/vps/%d/ports/candidates", hostID), q, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (c *Client) ListBackups(ctx context.Context, hostID int64) ([]appshared.AutomationBackup, error) {
	var resp struct {
		Data []map[string]any `json:"data"`
	}
	if err := c.openJSON(ctx, http.MethodGet, fmt.Sprintf("/api/v1/open/vps/%d/backups", hostID), nil, nil, &resp); err != nil {
		return nil, err
	}
	out := make([]appshared.AutomationBackup, 0, len(resp.Data))
	for _, it := range resp.Data {
		out = append(out, appshared.AutomationBackup(it))
	}
	return out, nil
}
func (c *Client) CreateBackup(ctx context.Context, hostID int64) error {
	return c.openJSON(ctx, http.MethodPost, fmt.Sprintf("/api/v1/open/vps/%d/backups", hostID), nil, nil, nil)
}
func (c *Client) DeleteBackup(ctx context.Context, hostID int64, backupID int64) error {
	return c.openJSON(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/open/vps/%d/backups/%d", hostID, backupID), nil, nil, nil)
}
func (c *Client) RestoreBackup(ctx context.Context, hostID int64, backupID int64) error {
	return c.openJSON(ctx, http.MethodPost, fmt.Sprintf("/api/v1/open/vps/%d/backups/%d/restore", hostID, backupID), nil, nil, nil)
}

func (c *Client) ListSnapshots(ctx context.Context, hostID int64) ([]appshared.AutomationSnapshot, error) {
	var resp struct {
		Data []map[string]any `json:"data"`
	}
	if err := c.openJSON(ctx, http.MethodGet, fmt.Sprintf("/api/v1/open/vps/%d/snapshots", hostID), nil, nil, &resp); err != nil {
		return nil, err
	}
	out := make([]appshared.AutomationSnapshot, 0, len(resp.Data))
	for _, it := range resp.Data {
		out = append(out, appshared.AutomationSnapshot(it))
	}
	return out, nil
}
func (c *Client) CreateSnapshot(ctx context.Context, hostID int64) error {
	return c.openJSON(ctx, http.MethodPost, fmt.Sprintf("/api/v1/open/vps/%d/snapshots", hostID), nil, nil, nil)
}
func (c *Client) DeleteSnapshot(ctx context.Context, hostID int64, snapshotID int64) error {
	return c.openJSON(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/open/vps/%d/snapshots/%d", hostID, snapshotID), nil, nil, nil)
}
func (c *Client) RestoreSnapshot(ctx context.Context, hostID int64, snapshotID int64) error {
	return c.openJSON(ctx, http.MethodPost, fmt.Sprintf("/api/v1/open/vps/%d/snapshots/%d/restore", hostID, snapshotID), nil, nil, nil)
}

func (c *Client) ListFirewallRules(ctx context.Context, hostID int64) ([]appshared.AutomationFirewallRule, error) {
	var resp struct {
		Data []map[string]any `json:"data"`
	}
	if err := c.openJSON(ctx, http.MethodGet, fmt.Sprintf("/api/v1/open/vps/%d/firewall", hostID), nil, nil, &resp); err != nil {
		return nil, err
	}
	out := make([]appshared.AutomationFirewallRule, 0, len(resp.Data))
	for _, it := range resp.Data {
		out = append(out, appshared.AutomationFirewallRule(it))
	}
	return out, nil
}
func (c *Client) AddFirewallRule(ctx context.Context, req appshared.AutomationFirewallRuleCreate) error {
	body := map[string]any{
		"direction": req.Direction,
		"protocol":  req.Protocol,
		"method":    req.Method,
		"port":      req.Port,
		"ip":        req.IP,
		"priority":  req.Priority,
	}
	return c.openJSON(ctx, http.MethodPost, fmt.Sprintf("/api/v1/open/vps/%d/firewall", req.HostID), nil, body, nil)
}
func (c *Client) DeleteFirewallRule(ctx context.Context, hostID int64, ruleID int64) error {
	return c.openJSON(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/open/vps/%d/firewall/%d", hostID, ruleID), nil, nil, nil)
}

func (c *Client) listRegions(ctx context.Context) ([]regionDTO, error) {
	var resp struct {
		Items []regionDTO `json:"items"`
	}
	query := map[string]string{}
	if c.goodsTypeID > 0 {
		query["goods_type_id"] = strconv.FormatInt(c.goodsTypeID, 10)
	}
	if err := c.adminJSON(ctx, http.MethodGet, "/admin/api/v1/regions", query, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Items, nil
}

func (c *Client) listPlanGroups(ctx context.Context) ([]planGroupDTO, error) {
	var resp struct {
		Items []planGroupDTO `json:"items"`
	}
	query := map[string]string{}
	if c.goodsTypeID > 0 {
		query["goods_type_id"] = strconv.FormatInt(c.goodsTypeID, 10)
	}
	if err := c.adminJSON(ctx, http.MethodGet, "/admin/api/v1/plan-groups", query, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Items, nil
}

func (c *Client) listVPS(ctx context.Context) ([]vpsDTO, error) {
	var resp struct {
		Items []vpsDTO `json:"items"`
	}
	if err := c.openJSON(ctx, http.MethodGet, "/api/v1/open/vps", nil, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Items, nil
}

func (c *Client) getVPS(ctx context.Context, id int64) (vpsDTO, error) {
	var out vpsDTO
	if err := c.openJSON(ctx, http.MethodGet, fmt.Sprintf("/api/v1/open/vps/%d", id), nil, nil, &out); err != nil {
		return vpsDTO{}, err
	}
	return out, nil
}

func (c *Client) findNewestVPS(ctx context.Context, packageID, systemID int64, preferredName string) (vpsDTO, error) {
	items, err := c.listVPS(ctx)
	if err != nil {
		return vpsDTO{}, err
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	for _, it := range items {
		if packageID > 0 && it.PackageID != packageID {
			continue
		}
		if systemID > 0 && it.SystemID != systemID {
			continue
		}
		if preferredName != "" && strings.EqualFold(it.Name, preferredName) {
			return it, nil
		}
		return it, nil
	}
	return vpsDTO{}, fmt.Errorf("no upstream vps found after order")
}

func (c *Client) findPackageByID(ctx context.Context, id int64) (packageDTO, error) {
	var resp struct {
		Items []packageDTO `json:"items"`
	}
	query := map[string]string{}
	if c.goodsTypeID > 0 {
		query["goods_type_id"] = strconv.FormatInt(c.goodsTypeID, 10)
	}
	if err := c.adminJSON(ctx, http.MethodGet, "/admin/api/v1/packages", query, nil, &resp); err != nil {
		return packageDTO{}, err
	}
	for _, it := range resp.Items {
		if it.ID == id {
			return it, nil
		}
	}
	return packageDTO{}, fmt.Errorf("package %d not found", id)
}

func (c *Client) pickPackageForRequest(ctx context.Context, req appshared.AutomationCreateHostRequest) (packageDTO, appshared.CartSpec, error) {
	plans, err := c.listPlanGroups(ctx)
	if err != nil {
		return packageDTO{}, appshared.CartSpec{}, err
	}
	planIDs := map[int64]struct{}{}
	for _, p := range plans {
		lineID := p.LineID
		if lineID <= 0 {
			lineID = p.ID
		}
		if req.LineID > 0 && lineID != req.LineID {
			continue
		}
		if !p.Active || !p.Visible {
			continue
		}
		planIDs[p.ID] = struct{}{}
	}
	var resp struct {
		Items []packageDTO `json:"items"`
	}
	query := map[string]string{}
	if c.goodsTypeID > 0 {
		query["goods_type_id"] = strconv.FormatInt(c.goodsTypeID, 10)
	}
	if err := c.adminJSON(ctx, http.MethodGet, "/admin/api/v1/packages", query, nil, &resp); err != nil {
		return packageDTO{}, appshared.CartSpec{}, err
	}
	bestIdx := -1
	bestScore := int(^uint(0) >> 1)
	bestSpec := appshared.CartSpec{}
	for i, p := range resp.Items {
		if _, ok := planIDs[p.PlanGroupID]; !ok {
			continue
		}
		if !p.Active || !p.Visible {
			continue
		}
		addCore := req.CPU - p.Cores
		addMem := req.MemoryGB - p.MemoryGB
		addDisk := req.DiskGB - p.DiskGB
		addBW := req.Bandwidth - p.BandwidthMbps
		if addCore < 0 || addMem < 0 || addDisk < 0 || addBW < 0 {
			continue
		}
		score := addCore + addMem + addDisk + addBW
		if score < bestScore {
			bestScore = score
			bestIdx = i
			bestSpec = appshared.CartSpec{AddCores: addCore, AddMemGB: addMem, AddDiskGB: addDisk, AddBWMbps: addBW}
		}
	}
	if bestIdx < 0 {
		return packageDTO{}, appshared.CartSpec{}, fmt.Errorf("no upstream package can satisfy requested spec")
	}
	return resp.Items[bestIdx], bestSpec, nil
}

func (c *Client) pickSystemID(ctx context.Context, lineID int64, osName string) (int64, error) {
	q := map[string]string{}
	if lineID > 0 {
		q["line_id"] = strconv.FormatInt(lineID, 10)
	}
	var resp struct {
		Items []imageDTO `json:"items"`
	}
	if err := c.adminJSON(ctx, http.MethodGet, "/admin/api/v1/system-images", q, nil, &resp); err != nil {
		return 0, err
	}
	name := strings.ToLower(strings.TrimSpace(osName))
	for _, it := range resp.Items {
		if !it.Enabled {
			continue
		}
		if name != "" && strings.EqualFold(strings.TrimSpace(it.Name), osName) {
			if it.ID > 0 {
				return it.ID, nil
			}
			return it.ImageID, nil
		}
	}
	for _, it := range resp.Items {
		if !it.Enabled {
			continue
		}
		if name != "" && strings.Contains(strings.ToLower(it.Name), name) {
			if it.ID > 0 {
				return it.ID, nil
			}
			return it.ImageID, nil
		}
	}
	for _, it := range resp.Items {
		if it.Enabled {
			if it.ID > 0 {
				return it.ID, nil
			}
			if it.ImageID > 0 {
				return it.ImageID, nil
			}
		}
	}
	return 0, fmt.Errorf("no upstream image available")
}

func (c *Client) openAction(ctx context.Context, hostID int64, action string) error {
	return c.openJSON(ctx, http.MethodPost, fmt.Sprintf("/api/v1/open/vps/%d/%s", hostID, action), nil, nil, nil)
}

func (c *Client) openRedirect(ctx context.Context, method, path string, query map[string]string) (string, error) {
	full := c.buildURL(path, query)
	req, err := http.NewRequestWithContext(ctx, method, full, nil)
	if err != nil {
		return "", err
	}
	ts := time.Now().UTC().Format(time.RFC3339)
	nonce, _ := randomNonce(8)
	canonical := buildCanonical(method, path, encodeQuery(query), ts, nonce, nil)
	signature := signHex(c.openSecret, canonical)
	req.Header.Set("X-AKID", c.openAKID)
	req.Header.Set("X-Timestamp", ts)
	req.Header.Set("X-Nonce", nonce)
	req.Header.Set("X-Signature", signature)
	client := *c.http
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusTemporaryRedirect {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected redirect status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	loc := strings.TrimSpace(resp.Header.Get("Location"))
	if loc == "" {
		return "", fmt.Errorf("empty redirect location")
	}
	return loc, nil
}

func (c *Client) adminJSON(ctx context.Context, method, path string, query map[string]string, body any, out any) error {
	if strings.TrimSpace(c.adminAPIKey) == "" {
		return fmt.Errorf("admin_api_key required")
	}
	return c.doJSON(ctx, method, path, query, body, false, out)
}

func (c *Client) openJSON(ctx context.Context, method, path string, query map[string]string, body any, out any) error {
	return c.doJSON(ctx, method, path, query, body, true, out)
}

func (c *Client) doJSON(ctx context.Context, method, path string, query map[string]string, body any, signed bool, out any) error {
	var bodyBytes []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyBytes = b
	}
	full := c.buildURL(path, query)
	req, err := http.NewRequestWithContext(ctx, method, full, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	if len(bodyBytes) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	if signed {
		ts := time.Now().UTC().Format(time.RFC3339)
		nonce, _ := randomNonce(8)
		canonical := buildCanonical(method, path, encodeQuery(query), ts, nonce, bodyBytes)
		req.Header.Set("X-AKID", c.openAKID)
		req.Header.Set("X-Timestamp", ts)
		req.Header.Set("X-Nonce", nonce)
		req.Header.Set("X-Signature", signHex(c.openSecret, canonical))
	} else {
		req.Header.Set("X-API-Key", c.adminAPIKey)
	}
	start := time.Now()
	resp, err := c.http.Do(req)
	if err != nil {
		c.emitLog(ctx, method+" "+path, req, nil, nil, time.Since(start), err)
		return err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		c.emitLog(ctx, method+" "+path, req, resp, nil, time.Since(start), err)
		return err
	}
	c.emitLog(ctx, method+" "+path, req, resp, b, time.Since(start), nil)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	if out == nil {
		return nil
	}
	if len(bytes.TrimSpace(b)) == 0 {
		return nil
	}
	return json.Unmarshal(b, out)
}

func (c *Client) buildURL(path string, query map[string]string) string {
	u := c.baseURL + path
	enc := encodeQuery(query)
	if enc != "" {
		u += "?" + enc
	}
	return u
}

func encodeQuery(query map[string]string) string {
	if len(query) == 0 {
		return ""
	}
	v := url.Values{}
	for k, val := range query {
		if strings.TrimSpace(val) == "" {
			continue
		}
		v.Set(k, val)
	}
	return v.Encode()
}

func buildCanonical(method, path, rawQuery, timestamp, nonce string, body []byte) string {
	h := sha256.Sum256(body)
	return strings.Join([]string{
		strings.ToUpper(strings.TrimSpace(method)),
		path,
		rawQuery,
		strings.TrimSpace(timestamp),
		strings.TrimSpace(nonce),
		hex.EncodeToString(h[:]),
	}, "\n")
}

func signHex(secret, canonical string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(canonical))
	return hex.EncodeToString(mac.Sum(nil))
}

func randomNonce(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func mapStatusState(status string, fallback int) int {
	s := strings.ToLower(strings.TrimSpace(status))
	switch s {
	case "running":
		return 2
	case "stopped":
		return 3
	case "provisioning", "pending", "creating":
		return 0
	case "failed", "error":
		return 11
	case "locked", "expired_locked":
		return 10
	default:
		if fallback != 0 {
			return fallback
		}
		return 0
	}
}

func toInt64(v any) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case float32:
		return int64(t)
	case int:
		return int64(t)
	case int32:
		return int64(t)
	case int64:
		return t
	case string:
		i, _ := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		return i
	case json.Number:
		i, _ := t.Int64()
		return i
	default:
		return 0
	}
}

func toCents(v any) int64 {
	switch t := v.(type) {
	case float64:
		return int64(math.Round(t * 100))
	case float32:
		return int64(math.Round(float64(t) * 100))
	case int64:
		return t
	case int:
		return int64(t)
	case json.Number:
		if i, err := t.Int64(); err == nil {
			return i
		}
		if f, err := t.Float64(); err == nil {
			return int64(math.Round(f * 100))
		}
	case string:
		s := strings.TrimSpace(t)
		if strings.Contains(s, ".") {
			if f, err := strconv.ParseFloat(s, 64); err == nil {
				return int64(math.Round(f * 100))
			}
		}
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return i
		}
	}
	return 0
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (c *Client) emitLog(ctx context.Context, action string, req *http.Request, resp *http.Response, body []byte, duration time.Duration, err error) {
	if c.logFn == nil {
		return
	}
	headers := map[string]string{}
	if req != nil {
		for k, v := range req.Header {
			headers[k] = strings.Join(v, ",")
		}
	}
	respHeaders := map[string]string{}
	status := 0
	if resp != nil {
		status = resp.StatusCode
		for k, v := range resp.Header {
			respHeaders[k] = strings.Join(v, ",")
		}
	}
	message := "ok"
	if err != nil {
		message = err.Error()
	}
	c.logFn(ctx, HTTPLogEntry{
		Action: action,
		Request: map[string]any{
			"method":  req.Method,
			"url":     req.URL.String(),
			"headers": headers,
		},
		Response: map[string]any{
			"status":      status,
			"headers":     respHeaders,
			"body":        string(body),
			"duration_ms": duration.Milliseconds(),
		},
		Success: err == nil,
		Message: message,
	})
}
