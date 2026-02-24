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
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	appshared "xiaoheiplay/internal/app/shared"
)

var errNotSupported = errors.New("not supported by mofang openapi")

type Client struct {
	baseURL          string
	account          string
	apiPassword      string
	productID        int64
	billingCycle     string
	cancelType       string
	cancelReason     string
	configOptionTmpl any
	customFieldTmpl  any
	osTmpl           any
	enableCatalog    bool
	enableCreateHost bool
	http             *http.Client
	logFn            func(context.Context, HTTPLogEntry)

	mu    sync.Mutex
	token string
}

type HTTPLogEntry struct {
	Action   string
	Request  map[string]any
	Response map[string]any
	Success  bool
	Message  string
}

func NewClient(baseURL, account, apiPassword string, productID int64, billingCycle, cancelType, cancelReason string, configOptionTmpl any, customFieldTmpl any, osTmpl any, enableCatalog, enableCreateHost bool, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 12 * time.Second
	}
	billingCycle = strings.TrimSpace(billingCycle)
	if billingCycle == "" {
		billingCycle = "monthly"
	}
	cancelType = strings.TrimSpace(cancelType)
	if cancelType == "" {
		cancelType = "Immediate"
	}
	cancelReason = strings.TrimSpace(cancelReason)
	if cancelReason == "" {
		cancelReason = "automation cancel request"
	}
	return &Client{
		baseURL:          strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		account:          strings.TrimSpace(account),
		apiPassword:      strings.TrimSpace(apiPassword),
		productID:        productID,
		billingCycle:     billingCycle,
		cancelType:       cancelType,
		cancelReason:     cancelReason,
		configOptionTmpl: configOptionTmpl,
		customFieldTmpl:  customFieldTmpl,
		osTmpl:           osTmpl,
		enableCatalog:    enableCatalog,
		enableCreateHost: enableCreateHost,
		http:             &http.Client{Timeout: timeout},
	}
}

func (c *Client) WithLogger(fn func(context.Context, HTTPLogEntry)) *Client {
	c.logFn = fn
	return c
}

type responseEnvelope struct {
	Status any            `json:"status"`
	Msg    string         `json:"msg"`
	Data   map[string]any `json:"data"`
}

func (c *Client) ListAreas(ctx context.Context) ([]appshared.AutomationArea, error) {
	if !c.enableCatalog {
		return []appshared.AutomationArea{}, nil
	}
	var env responseEnvelope
	if err := c.getJSON(ctx, "/v1/products", nil, &env); err != nil {
		return nil, err
	}
	groups := toSlice(toMap(env.Data)["first_group"])
	out := make([]appshared.AutomationArea, 0, len(groups))
	for _, item := range groups {
		m := toMap(item)
		out = append(out, appshared.AutomationArea{ID: toInt64(m["id"]), Name: toString(m["name"]), State: 1})
	}
	return out, nil
}

func (c *Client) ListLines(ctx context.Context) ([]appshared.AutomationLine, error) {
	if !c.enableCatalog {
		return []appshared.AutomationLine{}, nil
	}
	var env responseEnvelope
	if err := c.getJSON(ctx, "/v1/products", nil, &env); err != nil {
		return nil, err
	}
	firstGroups := toSlice(toMap(env.Data)["first_group"])
	out := make([]appshared.AutomationLine, 0)
	for _, fg := range firstGroups {
		fgm := toMap(fg)
		areaID := toInt64(fgm["id"])
		for _, group := range toSlice(fgm["group"]) {
			gm := toMap(group)
			out = append(out, appshared.AutomationLine{ID: toInt64(gm["id"]), Name: toString(gm["name"]), AreaID: areaID, State: 1})
		}
	}
	return out, nil
}

func (c *Client) ListImages(ctx context.Context, lineID int64) ([]appshared.AutomationImage, error) {
	_ = lineID
	return []appshared.AutomationImage{}, nil
}

func (c *Client) ListProducts(ctx context.Context, lineID int64) ([]appshared.AutomationProduct, error) {
	if !c.enableCatalog {
		return []appshared.AutomationProduct{}, nil
	}
	var env responseEnvelope
	if err := c.getJSON(ctx, "/v1/products", nil, &env); err != nil {
		return nil, err
	}
	firstGroups := toSlice(toMap(env.Data)["first_group"])
	out := make([]appshared.AutomationProduct, 0)
	for _, fg := range firstGroups {
		for _, group := range toSlice(toMap(fg)["group"]) {
			gm := toMap(group)
			groupID := toInt64(gm["id"])
			if lineID > 0 && groupID != lineID {
				continue
			}
			for _, p := range toSlice(gm["products"]) {
				pm := toMap(p)
				pid := toInt64(pm["id"])
				if pid <= 0 {
					continue
				}
				price := toCents(pm["sale_price"])
				if price <= 0 {
					price = toCents(pm["monthly"])
				}
				out = append(out, appshared.AutomationProduct{
					ID:                pid,
					Name:              toString(pm["name"]),
					CPU:               parseIntFromSpec(pm, []string{"cpu", "CPU", "cores"}),
					MemoryGB:          parseIntFromSpec(pm, []string{"memory", "ram", "Memory"}),
					DiskGB:            parseIntFromSpec(pm, []string{"disk", "DiskSpace", "storage"}),
					Bandwidth:         parseIntFromSpec(pm, []string{"bandwidth", "bw", "network"}),
					Price:             price,
					PortNum:           1,
					CapacityRemaining: 999999,
				})
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (c *Client) CreateHost(ctx context.Context, req appshared.AutomationCreateHostRequest) (appshared.AutomationCreateHostResult, error) {
	if !c.enableCreateHost {
		return appshared.AutomationCreateHostResult{}, fmt.Errorf("create host disabled by config")
	}
	pid := c.productID
	if pid <= 0 {
		return appshared.AutomationCreateHostResult{}, fmt.Errorf("product_id required for create host")
	}
	before, err := c.listHosts(ctx, map[string]string{"limit": "200", "page": "1"})
	if err != nil {
		return appshared.AutomationCreateHostResult{}, err
	}
	seen := map[int64]struct{}{}
	for _, h := range before {
		seen[toInt64(h["id"])] = struct{}{}
	}
	addPayload := map[string]any{
		"product_id":   pid,
		"billingcycle": c.billingCycle,
		"qty":          1,
	}
	if configoption := renderTemplateMap(c.configOptionTmpl, req); len(configoption) > 0 {
		addPayload["configoption"] = configoption
	} else {
		addPayload["configoption"] = map[string]any{}
	}
	if customfield := renderTemplateMap(c.customFieldTmpl, req); len(customfield) > 0 {
		addPayload["customfield"] = customfield
	} else {
		addPayload["customfield"] = map[string]any{}
	}
	if osMap := renderTemplateMap(c.osTmpl, req); len(osMap) > 0 {
		addPayload["os"] = osMap
	} else if osID := strings.TrimSpace(req.OS); osID != "" {
		addPayload["os"] = map[string]any{osID: osID}
	}
	if strings.TrimSpace(req.HostName) != "" {
		addPayload["host"] = strings.TrimSpace(req.HostName)
	}
	if strings.TrimSpace(req.SysPwd) != "" {
		addPayload["password"] = strings.TrimSpace(req.SysPwd)
	}
	var addResp responseEnvelope
	if err := c.postJSON(ctx, "/v1/cart/products", addPayload, &addResp); err != nil {
		return appshared.AutomationCreateHostResult{}, err
	}
	checkoutPayload := map[string]any{"checkout": 1, "payment": "credit"}
	var checkoutResp responseEnvelope
	if err := c.postJSON(ctx, "/v1/cart/checkout", checkoutPayload, &checkoutResp); err != nil {
		return appshared.AutomationCreateHostResult{}, err
	}
	deadline := time.Now().Add(40 * time.Second)
	for time.Now().Before(deadline) {
		hosts, listErr := c.listHosts(ctx, map[string]string{"limit": "200", "page": "1"})
		if listErr == nil {
			for _, h := range hosts {
				id := toInt64(h["id"])
				if id <= 0 {
					continue
				}
				if _, ok := seen[id]; ok {
					continue
				}
				hostname := strings.TrimSpace(toString(h["domain"]))
				if req.HostName == "" || hostname == "" || strings.EqualFold(hostname, req.HostName) {
					return appshared.AutomationCreateHostResult{HostID: id, Raw: map[string]any{"checkout": checkoutResp.Data}}, nil
				}
			}
		}
		time.Sleep(2 * time.Second)
	}
	return appshared.AutomationCreateHostResult{}, fmt.Errorf("host created but id not discovered in time")
}

func (c *Client) GetHostInfo(ctx context.Context, hostID int64) (appshared.AutomationHostInfo, error) {
	var env responseEnvelope
	if err := c.getJSON(ctx, fmt.Sprintf("/v1/hosts/%d", hostID), nil, &env); err != nil {
		return appshared.AutomationHostInfo{}, err
	}
	host := toMap(toMap(env.Data)["host"])
	state := statusToState(toString(host["domainstatus"]))
	if pstate, err := c.getPowerState(ctx, hostID); err == nil && pstate >= 0 {
		state = pstate
	}
	var expire *time.Time
	if t := parseTimeFlex(host["nextduedate"]); !t.IsZero() {
		tt := t
		expire = &tt
	}
	return appshared.AutomationHostInfo{
		HostID:        toInt64(host["id"]),
		HostName:      toString(host["domain"]),
		State:         state,
		CPU:           parseIntFromSpec(host, []string{"cpu", "CPU"}),
		MemoryGB:      parseIntFromSpec(host, []string{"memory", "ram", "Memory"}),
		DiskGB:        parseIntFromSpec(host, []string{"disk", "DiskSpace"}),
		Bandwidth:     parseIntFromSpec(host, []string{"bandwidth", "bw"}),
		PanelPassword: toString(host["password"]),
		VNCPassword:   "",
		OSPassword:    toString(host["password"]),
		RemoteIP:      pickRemoteIP(host),
		ExpireAt:      expire,
	}, nil
}

func (c *Client) ListHostSimple(ctx context.Context, searchTag string) ([]appshared.AutomationHostSimple, error) {
	query := map[string]string{"limit": "200", "page": "1"}
	if strings.TrimSpace(searchTag) != "" {
		query["keywords"] = searchTag
	}
	hosts, err := c.listHosts(ctx, query)
	if err != nil {
		return nil, err
	}
	out := make([]appshared.AutomationHostSimple, 0, len(hosts))
	for _, h := range hosts {
		out = append(out, appshared.AutomationHostSimple{ID: toInt64(h["id"]), HostName: toString(h["domain"]), IP: pickRemoteIP(h)})
	}
	return out, nil
}

func (c *Client) StartHost(ctx context.Context, hostID int64) error {
	_, err := c.putSimple(ctx, fmt.Sprintf("/v1/hosts/%d/module/on", hostID), nil)
	return err
}

func (c *Client) ShutdownHost(ctx context.Context, hostID int64) error {
	_, err := c.putSimple(ctx, fmt.Sprintf("/v1/hosts/%d/module/off", hostID), nil)
	return err
}

func (c *Client) RebootHost(ctx context.Context, hostID int64) error {
	_, err := c.putSimple(ctx, fmt.Sprintf("/v1/hosts/%d/module/reboot", hostID), nil)
	return err
}

func (c *Client) ResetOS(ctx context.Context, hostID int64, templateID int64, password string) error {
	payload := map[string]any{"password": strings.TrimSpace(password)}
	if templateID > 0 {
		payload["os"] = templateID
		payload["osid"] = templateID
	}
	_, err := c.putSimple(ctx, fmt.Sprintf("/v1/hosts/%d/module/reinstall", hostID), payload)
	return err
}

func (c *Client) ResetOSPassword(ctx context.Context, hostID int64, password string) error {
	payload := map[string]any{"password": strings.TrimSpace(password)}
	_, err := c.putSimple(ctx, fmt.Sprintf("/v1/hosts/%d/module/repassword", hostID), payload)
	return err
}

func (c *Client) RenewHost(ctx context.Context, hostID int64, nextDueDate time.Time) error {
	_ = nextDueDate
	payload := map[string]any{"billingcycle": c.billingCycle}
	_, err := c.postSimple(ctx, fmt.Sprintf("/v1/hosts/%d/renew", hostID), payload)
	return err
}

func (c *Client) ElasticUpdate(ctx context.Context, req appshared.AutomationElasticUpdateRequest) error {
	_ = ctx
	_ = req
	return errNotSupported
}

func (c *Client) LockHost(ctx context.Context, hostID int64) error {
	_ = ctx
	_ = hostID
	return errNotSupported
}

func (c *Client) UnlockHost(ctx context.Context, hostID int64) error {
	_ = ctx
	_ = hostID
	return errNotSupported
}

func (c *Client) DeleteHost(ctx context.Context, hostID int64) error {
	payload := map[string]any{"type": c.cancelType, "reason": c.cancelReason}
	_, err := c.postSimple(ctx, fmt.Sprintf("/v1/hosts/%d/cancel", hostID), payload)
	return err
}

func (c *Client) GetPanelURL(ctx context.Context, hostName string, panelPassword string) (string, error) {
	_ = panelPassword
	hosts, err := c.ListHostSimple(ctx, hostName)
	if err != nil {
		return "", err
	}
	for _, h := range hosts {
		if strings.EqualFold(strings.TrimSpace(h.HostName), strings.TrimSpace(hostName)) {
			return c.GetVNCURL(ctx, h.ID)
		}
	}
	return "", fmt.Errorf("host not found by name")
}

func (c *Client) GetVNCURL(ctx context.Context, hostID int64) (string, error) {
	env, err := c.putSimple(ctx, fmt.Sprintf("/v1/hosts/%d/module/vnc", hostID), nil)
	if err != nil {
		return "", err
	}
	data := toMap(env.Data)
	if u := toString(data["url"]); u != "" {
		return u, nil
	}
	if u := toString(data["vnc_url"]); u != "" {
		return u, nil
	}
	return "", fmt.Errorf("vnc url not found")
}

func (c *Client) GetMonitor(ctx context.Context, hostID int64) (appshared.AutomationMonitor, error) {
	_ = ctx
	_ = hostID
	return appshared.AutomationMonitor{}, errNotSupported
}

func (c *Client) ListPortMappings(ctx context.Context, hostID int64) ([]appshared.AutomationPortMapping, error) {
	_ = ctx
	_ = hostID
	return nil, errNotSupported
}

func (c *Client) AddPortMapping(ctx context.Context, req appshared.AutomationPortMappingCreate) error {
	_ = ctx
	_ = req
	return errNotSupported
}

func (c *Client) DeletePortMapping(ctx context.Context, hostID int64, mappingID int64) error {
	_ = ctx
	_ = hostID
	_ = mappingID
	return errNotSupported
}

func (c *Client) FindPortCandidates(ctx context.Context, hostID int64, keywords string) ([]int64, error) {
	_ = ctx
	_ = hostID
	_ = keywords
	return nil, errNotSupported
}

func (c *Client) ListBackups(ctx context.Context, hostID int64) ([]appshared.AutomationBackup, error) {
	_ = ctx
	_ = hostID
	return nil, errNotSupported
}

func (c *Client) CreateBackup(ctx context.Context, hostID int64) error {
	_ = ctx
	_ = hostID
	return errNotSupported
}

func (c *Client) DeleteBackup(ctx context.Context, hostID int64, backupID int64) error {
	_ = ctx
	_ = hostID
	_ = backupID
	return errNotSupported
}

func (c *Client) RestoreBackup(ctx context.Context, hostID int64, backupID int64) error {
	_ = ctx
	_ = hostID
	_ = backupID
	return errNotSupported
}

func (c *Client) ListSnapshots(ctx context.Context, hostID int64) ([]appshared.AutomationSnapshot, error) {
	_ = ctx
	_ = hostID
	return nil, errNotSupported
}

func (c *Client) CreateSnapshot(ctx context.Context, hostID int64) error {
	_ = ctx
	_ = hostID
	return errNotSupported
}

func (c *Client) DeleteSnapshot(ctx context.Context, hostID int64, snapshotID int64) error {
	_ = ctx
	_ = hostID
	_ = snapshotID
	return errNotSupported
}

func (c *Client) RestoreSnapshot(ctx context.Context, hostID int64, snapshotID int64) error {
	_ = ctx
	_ = hostID
	_ = snapshotID
	return errNotSupported
}

func (c *Client) ListFirewallRules(ctx context.Context, hostID int64) ([]appshared.AutomationFirewallRule, error) {
	_ = ctx
	_ = hostID
	return nil, errNotSupported
}

func (c *Client) AddFirewallRule(ctx context.Context, req appshared.AutomationFirewallRuleCreate) error {
	_ = ctx
	_ = req
	return errNotSupported
}

func (c *Client) DeleteFirewallRule(ctx context.Context, hostID int64, ruleID int64) error {
	_ = ctx
	_ = hostID
	_ = ruleID
	return errNotSupported
}

func (c *Client) listHosts(ctx context.Context, query map[string]string) ([]map[string]any, error) {
	var env responseEnvelope
	if err := c.getJSON(ctx, "/v1/hosts", query, &env); err != nil {
		return nil, err
	}
	arr := toSlice(toMap(env.Data)["host"])
	out := make([]map[string]any, 0, len(arr))
	for _, item := range arr {
		out = append(out, toMap(item))
	}
	return out, nil
}

func (c *Client) getPowerState(ctx context.Context, hostID int64) (int, error) {
	var env responseEnvelope
	if err := c.getJSON(ctx, fmt.Sprintf("/v1/hosts/%d/module/status", hostID), nil, &env); err != nil {
		return -1, err
	}
	m := toMap(env.Data)
	statusText := strings.ToLower(toString(m["status"]))
	if statusText == "" {
		statusText = strings.ToLower(toString(m["power"]))
	}
	switch {
	case strings.Contains(statusText, "on"), strings.Contains(statusText, "running"), strings.Contains(statusText, "active"):
		return 1, nil
	case strings.Contains(statusText, "off"), strings.Contains(statusText, "stop"):
		return 0, nil
	default:
		return -1, nil
	}
}

func (c *Client) ensureToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	t := c.token
	c.mu.Unlock()
	if strings.TrimSpace(t) != "" {
		return t, nil
	}
	return c.login(ctx)
}

func (c *Client) login(ctx context.Context) (string, error) {
	payload := map[string]any{"account": c.account, "password": c.apiPassword}
	reqBody, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.buildURL("/v1/login_api", nil), bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("login parse failed: %w", err)
	}
	if !isSuccess(parsed["status"]) {
		return "", fmt.Errorf("login failed: %s", toString(parsed["msg"]))
	}
	token := strings.TrimSpace(toString(parsed["jwt"]))
	if token == "" {
		return "", fmt.Errorf("login jwt missing")
	}
	c.mu.Lock()
	c.token = token
	c.mu.Unlock()
	return token, nil
}

func (c *Client) clearToken() {
	c.mu.Lock()
	c.token = ""
	c.mu.Unlock()
}

func (c *Client) getJSON(ctx context.Context, path string, query map[string]string, out any) error {
	return c.doJSON(ctx, http.MethodGet, path, query, nil, out)
}

func (c *Client) postJSON(ctx context.Context, path string, body any, out any) error {
	return c.doJSON(ctx, http.MethodPost, path, nil, body, out)
}

func (c *Client) putJSON(ctx context.Context, path string, body any, out any) error {
	return c.doJSON(ctx, http.MethodPut, path, nil, body, out)
}

func (c *Client) postSimple(ctx context.Context, path string, body any) (responseEnvelope, error) {
	var env responseEnvelope
	if err := c.postJSON(ctx, path, body, &env); err != nil {
		return responseEnvelope{}, err
	}
	return env, nil
}

func (c *Client) putSimple(ctx context.Context, path string, body any) (responseEnvelope, error) {
	var env responseEnvelope
	if err := c.putJSON(ctx, path, body, &env); err != nil {
		return responseEnvelope{}, err
	}
	return env, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, query map[string]string, body any, out any) error {
	for attempt := 0; attempt < 2; attempt++ {
		token, err := c.ensureToken(ctx)
		if err != nil {
			return err
		}
		var reqBody []byte
		if body != nil {
			reqBody, err = json.Marshal(body)
			if err != nil {
				return err
			}
		}
		req, err := http.NewRequestWithContext(ctx, method, c.buildURL(path, query), bytes.NewReader(reqBody))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "JWT "+token)
		req.Header.Set("Content-Type", "application/json")
		started := time.Now()
		resp, err := c.http.Do(req)
		if err != nil {
			c.emitLog(ctx, path, req, nil, nil, time.Since(started), err)
			return err
		}
		bodyBytes, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		var parsed map[string]any
		if len(bodyBytes) > 0 {
			_ = json.Unmarshal(bodyBytes, &parsed)
		}
		c.emitLog(ctx, path, req, resp, bodyBytes, time.Since(started), nil)
		if resp.StatusCode >= 400 {
			if attempt == 0 && shouldRelogin(parsed) {
				c.clearToken()
				continue
			}
			return fmt.Errorf("http %d: %s", resp.StatusCode, toString(parsed["msg"]))
		}
		if shouldRelogin(parsed) {
			if attempt == 0 {
				c.clearToken()
				continue
			}
			return fmt.Errorf("auth expired")
		}
		if !isSuccess(parsed["status"]) {
			return fmt.Errorf("mofang error: %s", toString(parsed["msg"]))
		}
		if out != nil {
			if len(bodyBytes) == 0 {
				return nil
			}
			if err := json.Unmarshal(bodyBytes, out); err != nil {
				return err
			}
		}
		return nil
	}
	return fmt.Errorf("request failed after retry")
}

func shouldRelogin(parsed map[string]any) bool {
	if parsed == nil {
		return false
	}
	if toInt64(parsed["status"]) == 405 {
		return true
	}
	msg := strings.ToLower(toString(parsed["msg"]))
	return strings.Contains(msg, "登陆") || strings.Contains(msg, "login")
}

func isSuccess(status any) bool {
	s := strings.TrimSpace(strings.ToLower(fmt.Sprint(status)))
	return s == "200" || s == "success"
}

func statusToState(s string) int {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "active", "running", "on":
		return 1
	case "suspended", "locked":
		return 3
	case "pending", "off", "stopped":
		return 0
	case "cancelled", "terminated", "deleted":
		return 4
	default:
		return 0
	}
}

func pickRemoteIP(host map[string]any) string {
	if ip := strings.TrimSpace(toString(host["dedicatedip"])); ip != "" {
		return ip
	}
	if raw := strings.TrimSpace(toString(host["assignedips"])); raw != "" {
		parts := strings.Split(raw, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	return ""
}

func parseTimeFlex(v any) time.Time {
	s := strings.TrimSpace(toString(v))
	if s == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil && n > 0 {
		if n > 1_000_000_000_000 {
			n /= 1000
		}
		return time.Unix(n, 0)
	}
	return time.Time{}
}

func parseIntFromSpec(m map[string]any, keys []string) int {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if iv := int(toInt64(v)); iv > 0 {
				return iv
			}
		}
	}
	for _, k := range []string{"options", "config_option"} {
		if nested, ok := m[k]; ok {
			nm := toMap(nested)
			for _, key := range keys {
				if iv := int(toInt64(nm[key])); iv > 0 {
					return iv
				}
			}
		}
	}
	return 0
}

func renderTemplateMap(tmpl any, req appshared.AutomationCreateHostRequest) map[string]any {
	root := toMap(tmpl)
	if len(root) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(root))
	for k, v := range root {
		rendered := renderTemplateValue(v, req)
		if rendered == nil {
			continue
		}
		out[k] = rendered
	}
	return out
}

func renderTemplateValue(v any, req appshared.AutomationCreateHostRequest) any {
	switch t := v.(type) {
	case string:
		return replaceCreateTokens(t, req)
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, vv := range t {
			out[k] = renderTemplateValue(vv, req)
		}
		return out
	case []any:
		out := make([]any, 0, len(t))
		for _, vv := range t {
			out = append(out, renderTemplateValue(vv, req))
		}
		return out
	default:
		return v
	}
}

func replaceCreateTokens(s string, req appshared.AutomationCreateHostRequest) any {
	raw := strings.TrimSpace(s)
	if raw == "" {
		return ""
	}
	tokens := map[string]any{
		"{{cpu}}":         req.CPU,
		"{{memory_gb}}":   req.MemoryGB,
		"{{disk_gb}}":     req.DiskGB,
		"{{bandwidth}}":   req.Bandwidth,
		"{{host_name}}":   strings.TrimSpace(req.HostName),
		"{{sys_pwd}}":     strings.TrimSpace(req.SysPwd),
		"{{vnc_pwd}}":     strings.TrimSpace(req.VNCPwd),
		"{{os}}":          strings.TrimSpace(req.OS),
		"{{port_num}}":    req.PortNum,
		"{{snapshot}}":    req.Snapshot,
		"{{backups}}":     req.Backups,
		"{{expire_unix}}": req.ExpireTime.Unix(),
	}
	if v, ok := tokens[raw]; ok {
		switch vv := v.(type) {
		case int:
			if vv <= 0 {
				return ""
			}
			return vv
		case int64:
			if vv <= 0 {
				return ""
			}
			return vv
		case string:
			return vv
		default:
			return v
		}
	}
	replaced := raw
	for k, v := range tokens {
		replaced = strings.ReplaceAll(replaced, k, toString(v))
	}
	return replaced
}

func toString(v any) string {
	if v == nil {
		return ""
	}
	s := strings.TrimSpace(fmt.Sprint(v))
	if s == "<nil>" || s == "null" {
		return ""
	}
	return s
}

func toMap(v any) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	if m, ok := v.(map[string]any); ok {
		return m
	}
	b, _ := json.Marshal(v)
	out := map[string]any{}
	_ = json.Unmarshal(b, &out)
	return out
}

func toSlice(v any) []any {
	if v == nil {
		return []any{}
	}
	if arr, ok := v.([]any); ok {
		return arr
	}
	b, _ := json.Marshal(v)
	var out []any
	_ = json.Unmarshal(b, &out)
	return out
}

func toInt64(v any) int64 {
	s := toString(v)
	if s == "" {
		return 0
	}
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return int64(f)
	}
	return 0
}

func toCents(v any) int64 {
	s := strings.TrimSpace(toString(v))
	if s == "" {
		return 0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return int64(f*100 + 0.5)
}

func (c *Client) buildURL(path string, query map[string]string) string {
	u := strings.TrimRight(c.baseURL, "/") + path
	if len(query) == 0 {
		return u
	}
	values := url.Values{}
	for k, v := range query {
		if strings.TrimSpace(v) == "" {
			continue
		}
		values.Set(k, v)
	}
	enc := values.Encode()
	if enc == "" {
		return u
	}
	return u + "?" + enc
}

func (c *Client) emitLog(ctx context.Context, action string, req *http.Request, resp *http.Response, body []byte, duration time.Duration, err error) {
	if c.logFn == nil {
		return
	}
	entry := HTTPLogEntry{Action: action, Success: err == nil, Message: "ok"}
	if err != nil {
		entry.Success = false
		entry.Message = err.Error()
	}
	entry.Request = map[string]any{"method": req.Method, "url": req.URL.String()}
	entry.Response = map[string]any{"duration_ms": duration.Milliseconds()}
	if resp != nil {
		entry.Response["status_code"] = resp.StatusCode
	}
	if len(body) > 0 {
		var bodyJSON any
		if json.Unmarshal(body, &bodyJSON) == nil {
			entry.Response["body_json"] = bodyJSON
		} else {
			entry.Response["body_text"] = string(body)
		}
	}
	c.logFn(ctx, entry)
}
