package http

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"sort"
	"strconv"
	"strings"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

func (h *Handler) AdminAPIKeys(c *gin.Context) {
	limit, offset := paging(c)
	items, total, err := h.adminSvc.ListAPIKeys(c, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrListError.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toAPIKeyDTOs(items), "total": total})
}

func (h *Handler) AdminAPIKeyCreate(c *gin.Context) {
	var payload struct {
		Name              string   `json:"name"`
		PermissionGroupID *int64   `json:"permission_group_id"`
		Scopes            []string `json:"scopes"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	raw, key, err := h.adminSvc.CreateAPIKey(c, getUserID(c), payload.Name, payload.PermissionGroupID, payload.Scopes)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"api_key": raw, "record": toAPIKeyDTO(key)})
}

func (h *Handler) AdminAPIKeyUpdate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		Status string `json:"status"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	status := domain.APIKeyStatus(payload.Status)
	if err := h.adminSvc.UpdateAPIKeyStatus(c, getUserID(c), id, status); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminSettingsList(c *gin.Context) {
	items, err := h.adminSvc.ListSettings(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrListError.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toSettingDTOs(items)})
}

func (h *Handler) AdminSettingsUpdate(c *gin.Context) {
	var payload struct {
		Key   string `json:"key"`
		Value string `json:"value"`
		Items []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"items"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	if len(payload.Items) > 0 {
		for _, item := range payload.Items {
			if strings.TrimSpace(item.Key) == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidKey.Error()})
				return
			}
			if err := validateSettingJSONValue(item.Key, item.Value); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if err := h.adminSvc.UpdateSetting(c, getUserID(c), item.Key, item.Value); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		}
	} else {
		if err := validateSettingJSONValue(payload.Key, payload.Value); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := h.adminSvc.UpdateSetting(c, getUserID(c), payload.Key, payload.Value); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPushTokenRegister(c *gin.Context) {
	if h.pushSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	var payload struct {
		Token    string `json:"token"`
		Platform string `json:"platform"`
		DeviceID string `json:"device_id"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	if err := h.pushSvc.RegisterToken(c, getUserID(c), payload.Platform, payload.Token, payload.DeviceID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPushTokenDelete(c *gin.Context) {
	if h.pushSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	var payload struct {
		Token string `json:"token"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	if err := h.pushSvc.RemoveToken(c, getUserID(c), payload.Token); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminDebugStatus(c *gin.Context) {
	if h.adminSvc == nil && h.settingsSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	enabled := strings.ToLower(h.getSettingValueByKey(c, "debug_enabled")) == "true"
	c.JSON(http.StatusOK, gin.H{"enabled": enabled})
}

func (h *Handler) AdminDebugStatusUpdate(c *gin.Context) {
	if h.adminSvc == nil && h.settingsSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	var payload struct {
		Enabled bool `json:"enabled"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	if err := h.adminSvc.UpdateSetting(c, getUserID(c), "debug_enabled", boolToString(payload.Enabled)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminDebugLogs(c *gin.Context) {
	if h.adminSvc == nil && h.settingsSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	if strings.ToLower(h.getSettingValueByKey(c, "debug_enabled")) != "true" {
		c.JSON(http.StatusForbidden, gin.H{"error": domain.ErrDebugDisabled.Error()})
		return
	}
	limit, offset := paging(c)
	types := strings.ToLower(strings.TrimSpace(c.Query("types")))
	includeAll := types == ""
	includeType := func(name string) bool {
		if includeAll {
			return true
		}
		for _, item := range strings.Split(types, ",") {
			if strings.TrimSpace(item) == name {
				return true
			}
		}
		return false
	}

	resp := gin.H{}
	if includeType("audit") && h.adminSvc != nil {
		items, total, err := h.adminSvc.ListAuditLogs(c, limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrListAuditLogsError.Error()})
			return
		}
		resp["audit_logs"] = gin.H{"items": toAdminAuditLogDTOs(items), "total": total}
	}
	if includeType("automation") && h.autoLogSvc != nil {
		orderID, _ := strconv.ParseInt(c.Query("order_id"), 10, 64)
		items, total, err := h.autoLogSvc.List(c, orderID, limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrListAutomationLogsError.Error()})
			return
		}
		resp["automation_logs"] = gin.H{"items": toAutomationLogDTOs(items), "total": total}
	}
	if includeType("sync") && h.integration != nil {
		target := c.Query("target")
		items, total, err := h.integration.ListSyncLogs(c, target, limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrListSyncLogsError.Error()})
			return
		}
		resp["sync_logs"] = gin.H{"items": toIntegrationSyncLogDTOs(items), "total": total}
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) AdminAutomationConfig(c *gin.Context) {
	if h.pluginAdmin == nil {
		c.JSON(http.StatusOK, gin.H{
			"base_url":      "",
			"api_key":       "",
			"enabled":       false,
			"timeout_sec":   12,
			"retry":         0,
			"dry_run":       false,
			"configured":    false,
			"compat_mode":   false,
			"plugins_ready": false,
			"config_source": "goods_type_plugin_instance",
		})
		return
	}
	cfg, present, binding, enabled, err := h.readAutomationPluginConfig(c)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error":         "no writable automation plugin instance found; configure automation plugin instance first",
			"code":          "no_writable_automation_instance",
			"redirect_path": "/admin/catalog",
		})
		return
	}
	if cfg.TimeoutSec <= 0 {
		cfg.TimeoutSec = 12
	}
	if cfg.Retry < 0 {
		cfg.Retry = 0
	}
	configured := present["base_url"] && strings.TrimSpace(cfg.BaseURL) != "" &&
		present["api_key"] && strings.TrimSpace(cfg.APIKey) != ""
	resp := gin.H{
		"base_url":      cfg.BaseURL,
		"api_key":       cfg.APIKey,
		"enabled":       enabled,
		"timeout_sec":   cfg.TimeoutSec,
		"retry":         cfg.Retry,
		"dry_run":       cfg.DryRun,
		"plugin_id":     binding.PluginID,
		"instance_id":   binding.InstanceID,
		"configured":    configured,
		"compat_mode":   false,
		"config_source": "goods_type_plugin_instance",
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) AdminAutomationConfigUpdate(c *gin.Context) {
	if h.pluginAdmin == nil {
		c.JSON(http.StatusOK, gin.H{
			"ok":            true,
			"compat_mode":   false,
			"plugins_ready": false,
		})
		return
	}
	var payload struct {
		BaseURL    *string `json:"base_url"`
		APIKey     *string `json:"api_key"`
		Enabled    *bool   `json:"enabled"`
		TimeoutSec *int    `json:"timeout_sec"`
		Retry      *int    `json:"retry"`
		DryRun     *bool   `json:"dry_run"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	binding, err := h.resolveWritableAutomationBinding(c)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error":         "no writable automation plugin instance found; configure automation plugin instance first",
			"code":          "no_writable_automation_instance",
			"redirect_path": "/admin/catalog",
		})
		return
	}

	current := appshared.AutomationConfig{}
	cfgJSON, err := h.pluginAdmin.GetConfigInstance(c, "automation", binding.PluginID, binding.InstanceID)
	if err == nil {
		if cfg, _, perr := parseAutomationConfigJSON(cfgJSON); perr == nil {
			current = cfg
		}
	}
	if payload.BaseURL != nil {
		current.BaseURL = strings.TrimSpace(*payload.BaseURL)
	}
	if payload.APIKey != nil {
		current.APIKey = strings.TrimSpace(*payload.APIKey)
	}
	if payload.TimeoutSec != nil {
		current.TimeoutSec = *payload.TimeoutSec
	}
	if payload.Retry != nil {
		current.Retry = *payload.Retry
	}
	if payload.DryRun != nil {
		current.DryRun = *payload.DryRun
	}
	if current.TimeoutSec <= 0 {
		current.TimeoutSec = 12
	}
	if current.Retry < 0 {
		current.Retry = 0
	}

	rawCfg, _ := json.Marshal(map[string]any{
		"base_url":    current.BaseURL,
		"api_key":     current.APIKey,
		"timeout_sec": current.TimeoutSec,
		"retry":       current.Retry,
		"dry_run":     current.DryRun,
	})
	if err := h.pluginAdmin.UpdateConfigInstance(c, "automation", binding.PluginID, binding.InstanceID, string(rawCfg)); err != nil {
		writePluginHandlerError(c, err)
		return
	}

	if payload.Enabled != nil {
		if *payload.Enabled {
			if err := h.pluginAdmin.EnableInstance(c, "automation", binding.PluginID, binding.InstanceID); err != nil {
				writePluginHandlerError(c, err)
				return
			}
		} else {
			if err := h.pluginAdmin.DisableInstance(c, "automation", binding.PluginID, binding.InstanceID); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":          true,
		"compat_mode": false,
		"plugin_id":   binding.PluginID,
		"instance_id": binding.InstanceID,
	})
}

type automationBinding struct {
	PluginID   string
	InstanceID string
}

func (h *Handler) readAutomationPluginConfig(c *gin.Context) (appshared.AutomationConfig, map[string]bool, automationBinding, bool, error) {
	if h.pluginAdmin == nil {
		return appshared.AutomationConfig{}, nil, automationBinding{}, false, domain.ErrPluginsDisabled
	}
	items, err := h.pluginAdmin.List(c)
	if err != nil {
		return appshared.AutomationConfig{}, nil, automationBinding{}, false, err
	}
	enabledByBinding := map[string]bool{}
	for _, item := range items {
		if !strings.EqualFold(strings.TrimSpace(item.Category), "automation") {
			continue
		}
		key := strings.TrimSpace(item.PluginID) + ":" + strings.TrimSpace(item.InstanceID)
		enabledByBinding[key] = item.Enabled
	}
	for _, binding := range h.collectAutomationBindingCandidates(c) {
		cfgJSON, err := h.pluginAdmin.GetConfigInstance(c, "automation", binding.PluginID, binding.InstanceID)
		if err != nil {
			continue
		}
		cfg, present, err := parseAutomationConfigJSON(cfgJSON)
		if err != nil {
			continue
		}
		key := binding.PluginID + ":" + binding.InstanceID
		return cfg, present, binding, enabledByBinding[key], nil
	}
	return appshared.AutomationConfig{}, nil, automationBinding{}, false, domain.ErrAutomationPluginInstanceNotFound
}

func (h *Handler) resolveWritableAutomationBinding(c *gin.Context) (automationBinding, error) {
	if h.pluginAdmin == nil {
		return automationBinding{}, domain.ErrPluginsDisabled
	}
	for _, binding := range h.collectAutomationBindingCandidates(c) {
		if _, err := h.pluginAdmin.GetConfigInstance(c, "automation", binding.PluginID, binding.InstanceID); err == nil {
			return binding, nil
		}
	}
	return automationBinding{}, domain.ErrAutomationPluginInstanceNotFound
}

func (h *Handler) collectAutomationBindingCandidates(c *gin.Context) []automationBinding {
	candidates := make([]automationBinding, 0, 4)
	if h.goodsTypes != nil {
		items, err := h.goodsTypes.List(c)
		if err == nil {
			sort.SliceStable(items, func(i, j int) bool {
				if items[i].SortOrder == items[j].SortOrder {
					return items[i].ID < items[j].ID
				}
				return items[i].SortOrder < items[j].SortOrder
			})
			for _, item := range items {
				if !strings.EqualFold(strings.TrimSpace(item.AutomationCategory), "automation") {
					continue
				}
				pluginID := strings.TrimSpace(item.AutomationPluginID)
				instanceID := strings.TrimSpace(item.AutomationInstanceID)
				if pluginID == "" || instanceID == "" {
					continue
				}
				candidates = append(candidates, automationBinding{PluginID: pluginID, InstanceID: instanceID})
			}
		}
	}

	uniq := make(map[string]struct{}, len(candidates))
	out := make([]automationBinding, 0, len(candidates))
	for _, candidate := range candidates {
		key := candidate.PluginID + ":" + candidate.InstanceID
		if _, exists := uniq[key]; exists {
			continue
		}
		uniq[key] = struct{}{}
		out = append(out, candidate)
	}
	return out
}

func parseAutomationConfigJSON(raw string) (appshared.AutomationConfig, map[string]bool, error) {
	cfg := appshared.AutomationConfig{}
	present := map[string]bool{}
	payload := strings.TrimSpace(raw)
	if payload == "" {
		return cfg, present, nil
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(payload), &obj); err != nil {
		return cfg, present, err
	}
	if v, ok := obj["base_url"]; ok {
		cfg.BaseURL = strings.TrimSpace(toString(v))
		present["base_url"] = true
	}
	if v, ok := obj["api_key"]; ok {
		cfg.APIKey = strings.TrimSpace(toString(v))
		present["api_key"] = true
	}
	if v, ok := obj["timeout_sec"]; ok {
		if n, ok := toInt(v); ok {
			cfg.TimeoutSec = n
		}
		present["timeout_sec"] = true
	}
	if v, ok := obj["retry"]; ok {
		if n, ok := toInt(v); ok {
			cfg.Retry = n
		}
		present["retry"] = true
	}
	if v, ok := obj["dry_run"]; ok {
		if b, ok := v.(bool); ok {
			cfg.DryRun = b
		}
		present["dry_run"] = true
	}
	return cfg, present, nil
}

func mergeAutomationConfig(base, override appshared.AutomationConfig, present map[string]bool) appshared.AutomationConfig {
	out := base
	if present["base_url"] && strings.TrimSpace(override.BaseURL) != "" {
		out.BaseURL = override.BaseURL
	}
	if present["api_key"] && strings.TrimSpace(override.APIKey) != "" {
		out.APIKey = override.APIKey
	}
	if present["timeout_sec"] && override.TimeoutSec > 0 {
		out.TimeoutSec = override.TimeoutSec
	}
	if present["retry"] && override.Retry >= 0 {
		out.Retry = override.Retry
	}
	if present["dry_run"] {
		out.DryRun = override.DryRun
	}
	return out
}

func toString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	default:
		return fmt.Sprintf("%v", t)
	}
}

func toInt(v any) (int, bool) {
	switch t := v.(type) {
	case int:
		return t, true
	case int8:
		return int(t), true
	case int16:
		return int(t), true
	case int32:
		return int(t), true
	case int64:
		return int(t), true
	case uint:
		return int(t), true
	case uint8:
		return int(t), true
	case uint16:
		return int(t), true
	case uint32:
		return int(t), true
	case uint64:
		return int(t), true
	case float32:
		return int(t), true
	case float64:
		return int(t), true
	case json.Number:
		n, err := t.Int64()
		if err != nil {
			return 0, false
		}
		return int(n), true
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(t))
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}

func (h *Handler) AdminAutomationSync(c *gin.Context) {
	if h.integration == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	mode := c.Query("mode")
	result, err := h.integration.SyncAutomation(c, mode)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) AdminAutomationSyncLogs(c *gin.Context) {
	if h.integration == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	limit, offset := paging(c)
	target := c.Query("target")
	items, total, err := h.integration.ListSyncLogs(c, target, limit, offset)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toIntegrationSyncLogDTOs(items), "total": total})
}
