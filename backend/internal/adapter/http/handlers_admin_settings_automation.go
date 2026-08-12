package http

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

func (h *Handler) AdminAPIKeys(c *gin.Context) {
	limit, offset := paging(c)
	items, total, err := h.adminSvc.ListAPIKeys(c.Request.Context(), limit, offset)
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
		writeBodyError(c, err)
		return
	}
	raw, key, err := h.adminSvc.CreateAPIKey(c.Request.Context(), getUserID(c), payload.Name, payload.PermissionGroupID, payload.Scopes)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"api_key": raw, "record": toAPIKeyDTO(key)})
}

func (h *Handler) AdminAPIKeyUpdate(c *gin.Context) {
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	var payload struct {
		Status string `json:"status"`
	}
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	status := domain.APIKeyStatus(payload.Status)
	if err := h.adminSvc.UpdateAPIKeyStatus(c.Request.Context(), getUserID(c), uri.ID, status); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// redactedSecretValue is the placeholder returned for secret settings.
const redactedSecretValue = "***"

// maskedSecretSentinel is the value the frontend sends back to signal "keep the
// existing secret". Both the redacted value and the frontend mask placeholder
// must be treated as keep markers.
const maskedSecretSentinel = "********"


// redactSecretValue replaces a credential with a fixed placeholder.
func redactSecretValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return redactedSecretValue
}

// isRedactedSecretValue reports whether the submitted value is a keep-marker
// (the backend's "***" or the frontend's "********") that must not overwrite a
// real stored secret.
func isRedactedSecretValue(value string) bool {
	v := strings.TrimSpace(value)
	return v == "" || v == redactedSecretValue || v == maskedSecretSentinel || v == pluginUploadPasswordMask
}

// redactRobotWebhooksValue masks the per-webhook HMAC secret field of the
// robot_webhooks setting JSON while preserving URLs/names/events for the admin
// UI. A raw value that cannot be parsed is fully redacted so no secret can leak.
func redactRobotWebhooksValue(valueJSON string) string {
	webhooks := appshared.ParseRobotWebhookConfigs(valueJSON)
	if len(webhooks) == 0 {
		return redactSecretValue(valueJSON)
	}
	for i := range webhooks {
		if strings.TrimSpace(webhooks[i].Secret) != "" {
			webhooks[i].Secret = redactedSecretValue
		}
	}
	raw, err := json.Marshal(webhooks)
	if err != nil {
		return redactSecretValue(valueJSON)
	}
	return string(raw)
}

// restoreRobotWebhookSecrets replaces masked per-webhook secrets ("***") in the
// submitted robot_webhooks JSON with the currently stored real secrets, so a
// save from the admin UI (which only ever sees masked secrets) never overwrites
// a signing credential with the placeholder. Matching is by webhook identity
// (URL, else name), the same key the admin robot config update uses.
func (h *Handler) restoreRobotWebhookSecrets(c *gin.Context, submitted string) (string, error) {
	return restoreRobotWebhookSecretsInJSON(h.getSettingValueByKey(c, "robot_webhooks"), submitted)
}

// restoreRobotWebhookSecretsInJSON is the pure form of the restore: it replaces
// masked per-webhook secrets in the submitted JSON with the real secrets from
// the currently stored JSON.
func restoreRobotWebhookSecretsInJSON(existingJSON, submitted string) (string, error) {
	existing := appshared.ParseRobotWebhookConfigs(existingJSON)
	byKey := make(map[string]string, len(existing))
	for _, w := range existing {
		byKey[webhookIdentity(w)] = strings.TrimSpace(w.Secret)
	}
	webhooks := appshared.ParseRobotWebhookConfigs(submitted)
	if webhooks == nil {
		return "", domain.ErrInvalidBody
	}
	for i := range webhooks {
		if !isRedactedSecretValue(webhooks[i].Secret) {
			continue
		}
		if real := byKey[webhookIdentity(webhooks[i])]; real != "" {
			webhooks[i].Secret = real
		}
	}
	raw, err := json.Marshal(webhooks)
	if err != nil {
		return "", domain.ErrInvalidBody
	}
	return string(raw), nil
}

func (h *Handler) AdminSettingsList(c *gin.Context) {
	items, err := h.adminSvc.ListSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrListError.Error()})
		return
	}
	for i := range items {
		switch items[i].Key {
		case "robot_webhooks":
			// Deep-redact the nested per-webhook secret; the whole value would
			// break the admin robot config page that needs URLs/names/events.
			items[i].ValueJSON = redactRobotWebhooksValue(items[i].ValueJSON)
		default:
			if isSecretSettingKey(items[i].Key) {
				items[i].ValueJSON = redactSecretValue(items[i].ValueJSON)
			}
		}
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
		writeBodyError(c, err)
		return
	}

	// 批量更新模式
	if len(payload.Items) > 0 {
		normalizedItems := make([]domain.Setting, 0, len(payload.Items))

		// 验证所有配置项
		for _, item := range payload.Items {
			if strings.TrimSpace(item.Key) == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidKey.Error()})
				return
			}
			if item.Key == "robot_webhooks" {
				// Restore masked per-webhook secrets before persisting so a save
				// never overwrites a signing credential with the placeholder.
				restored, err := h.restoreRobotWebhookSecrets(c, item.Value)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				normalizedItems = append(normalizedItems, domain.Setting{Key: item.Key, ValueJSON: restored})
				continue
			}
			// Masked secrets are keep-markers and must be preserved, not
			// validated as JSON/structural values or written back.
			if isSecretSettingKey(item.Key) && isRedactedSecretValue(item.Value) {
				continue
			}
			normalizedValue, err := validateAndNormalizeSettingValue(item.Key, item.Value)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			normalizedItems = append(normalizedItems, domain.Setting{Key: item.Key, ValueJSON: normalizedValue})
		}

		// 执行更新：所有键在一个事务内写入，任一键失败则全部回滚。
		if err := h.adminSvc.UpdateSettings(c.Request.Context(), getUserID(c), normalizedItems); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	} else {
		// 单个更新模式
		if strings.TrimSpace(payload.Key) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidKey.Error()})
			return
		}
		if isSecretSettingKey(payload.Key) && isRedactedSecretValue(payload.Value) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
			return
		}

		normalizedValue := payload.Value
		if payload.Key == "robot_webhooks" {
			restored, err := h.restoreRobotWebhookSecrets(c, payload.Value)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			normalizedValue = restored
		} else {
			validated, err := validateAndNormalizeSettingValue(payload.Key, payload.Value)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			normalizedValue = validated
		}

		if err := h.adminSvc.UpdateSetting(c.Request.Context(), getUserID(c), payload.Key, normalizedValue); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func validateAndNormalizeSettingValue(key, value string) (string, error) {
	switch key {
	case "copyright_text":
		if strings.TrimSpace(value) == "" {
			return "", domain.ErrCopyrightTextRequired
		}
		if len([]rune(value)) > 200 {
			return "", domain.ErrCopyrightTextTooLong
		}
		return value, nil
	case "beian_info_list":
		if err := validateBeianInfoList(value); err != nil {
			return "", err
		}
		return value, nil
	case "site_nav_items":
		if err := validateSiteNavItems(value); err != nil {
			return "", err
		}
		return value, nil
	case "admin_path":
		normalizedPath := strings.TrimSpace(value)
		if err := ValidateAdminPath(normalizedPath); err != nil {
			return "", err
		}
		return normalizedPath, nil
	case "emergency_renew_window_days":
		// P3-27: a non-positive window would permanently open the emergency renew
		// door, so reject 0/negative values instead of letting them through.
		n, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil || n <= 0 {
			return "", fmt.Errorf("%w: emergency_renew_window_days must be a positive integer", domain.ErrInvalidInput)
		}
		return value, nil
	case "analytics_code":
		// analytics_code has no runtime consumer. Keep the compatibility setting,
		// but reject any payload instead of maintaining an incomplete HTML
		// blacklist that could become stored XSS if a render sink is added later.
		if strings.TrimSpace(value) != "" {
			return "", domain.ErrInvalidInput
		}
		return "", nil
	default:
		// P2-1: previously validateSettingJSONValue was dead code and unknown
		// *_json / task.* / named-JSON keys passed straight through, so malformed
		// or double-encoded JSON was persisted. Route every key that isLikelyJSON
		// flags through the shared validator.
		if isLikelyJSONSettingKey(key) {
			if err := validateSettingJSONValue(key, value); err != nil {
				return "", err
			}
		}
		return value, nil
	}
}

// validateExternalURL 校验会被渲染进 `<a href>` / `<img src>` 的外部 URL 字段。
//
// 站点设置里的导航、备案链接、footer 社交链接等会直接绑定到模板锚点；如果
// 允许 `javascript:`/`data:`/协议相对，管理员写一次就会被任何访客以同源身份
// 执行（存储型 XSS，见 web_url_content F1）。只放行：
//
//   - 空值（可选字段）
//   - 同源相对路径（`/` 开头且非 `//`，无控制字符）
//   - http(s) 绝对地址（有 host、无 userinfo、无控制字符）
//
// 其余（javascript:/data:/file:/blob:/vbscript:/协议相对/userinfo/控制字符）
// 一律拒绝。
func validateExternalURL(raw string) error {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return domain.ErrUnsafeExternalURL
		}
	}
	if strings.HasPrefix(value, "//") {
		return domain.ErrUnsafeExternalURL
	}
	if strings.HasPrefix(value, "/") {
		return nil
	}
	u, err := url.Parse(value)
	if err != nil {
		return domain.ErrUnsafeExternalURL
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return domain.ErrUnsafeExternalURL
	}
	if u.Host == "" {
		return domain.ErrUnsafeExternalURL
	}
	if u.User != nil {
		return domain.ErrUnsafeExternalURL
	}
	return nil
}

// validateSiteNavItems 验证站点导航项：只校验 URL 安全性，不限制字段结构
// （前端 headerNavItems 已做 label/url/enabled 归一化）。
func validateSiteNavItems(value string) error {
	value = strings.TrimSpace(value)
	if value == "" || value == "[]" {
		return nil
	}
	var navList []struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(value), &navList); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrInvalidBeianInfoFormat, err)
	}
	for i, item := range navList {
		if err := validateExternalURL(item.URL); err != nil {
			return fmt.Errorf("导航项 %d: %w", i+1, err)
		}
	}
	return nil
}

// validateBeianInfoList 验证备案信息列表的格式与 URL 安全性
func validateBeianInfoList(value string) error {
	value = strings.TrimSpace(value)
	if value == "" || value == "[]" {
		return nil
	}

	var beianList []struct {
		Number  string `json:"number"`
		IconURL string `json:"icon_url"`
		LinkURL string `json:"link_url"`
	}

	if err := json.Unmarshal([]byte(value), &beianList); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrInvalidBeianInfoFormat, err)
	}

	for i, beian := range beianList {
		// 如果填写了任何字段，备案号必填
		if (beian.IconURL != "" || beian.LinkURL != "") && strings.TrimSpace(beian.Number) == "" {
			return fmt.Errorf("备案信息 %d: %w", i+1, domain.ErrBeianNumberRequired)
		}
		if err := validateExternalURL(beian.LinkURL); err != nil {
			return fmt.Errorf("备案信息 %d link_url: %w", i+1, err)
		}
		if err := validateExternalURL(beian.IconURL); err != nil {
			return fmt.Errorf("备案信息 %d icon_url: %w", i+1, err)
		}
	}

	return nil
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
		writeBodyError(c, err)
		return
	}
	if err := h.pushSvc.RegisterToken(c.Request.Context(), getUserID(c), payload.Platform, payload.Token, payload.DeviceID); err != nil {
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
		writeBodyError(c, err)
		return
	}
	if err := h.pushSvc.RemoveToken(c.Request.Context(), getUserID(c), payload.Token); err != nil {
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
		writeBodyError(c, err)
		return
	}
	if err := h.adminSvc.UpdateSetting(c.Request.Context(), getUserID(c), "debug_enabled", boolToString(payload.Enabled)); err != nil {
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
	var query struct {
		Types   string `form:"types"`
		OrderID *int64 `form:"order_id" binding:"omitempty,gt=0"`
		Target  string `form:"target"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	types := strings.ToLower(strings.TrimSpace(query.Types))
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
		items, total, err := h.adminSvc.ListAuditLogs(c.Request.Context(), appshared.AuditLogFilter{}, limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrListAuditLogsError.Error()})
			return
		}
		resp["audit_logs"] = gin.H{"items": toAdminAuditLogDTOs(items), "total": total}
	}
	if includeType("automation") && h.autoLogSvc != nil {
		orderID := int64(0)
		if query.OrderID != nil {
			orderID = *query.OrderID
		}
		items, total, err := h.autoLogSvc.List(c.Request.Context(), orderID, limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrListAutomationLogsError.Error()})
			return
		}
		resp["automation_logs"] = gin.H{"items": toAutomationLogDTOs(items), "total": total}
	}
	if includeType("sync") && h.integration != nil {
		items, total, err := h.integration.ListSyncLogs(c.Request.Context(), query.Target, limit, offset)
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
			"error":         domain.ErrNoWritableAutomationPluginInstance.Error(),
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
		"api_key":       redactSecretValue(cfg.APIKey),
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
		writeBodyError(c, err)
		return
	}
	binding, err := h.resolveWritableAutomationBinding(c)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error":         domain.ErrNoWritableAutomationPluginInstance.Error(),
			"code":          "no_writable_automation_instance",
			"redirect_path": "/admin/catalog",
		})
		return
	}

	current := appshared.AutomationConfig{}
	cfgJSON, err := h.pluginAdmin.GetConfigInstance(c.Request.Context(), "automation", binding.PluginID, binding.InstanceID)
	if err == nil {
		if cfg, _, perr := parseAutomationConfigJSON(cfgJSON); perr == nil {
			current = cfg
		}
	}
	if payload.BaseURL != nil {
		current.BaseURL = strings.TrimSpace(*payload.BaseURL)
	}
	if payload.APIKey != nil && !isRedactedSecretValue(*payload.APIKey) {
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
	if err := h.pluginAdmin.UpdateConfigInstance(c.Request.Context(), "automation", binding.PluginID, binding.InstanceID, string(rawCfg)); err != nil {
		writePluginHandlerError(c, err)
		return
	}

	if payload.Enabled != nil {
		if *payload.Enabled {
			if err := h.pluginAdmin.EnableInstance(c.Request.Context(), "automation", binding.PluginID, binding.InstanceID); err != nil {
				writePluginHandlerError(c, err)
				return
			}
		} else {
			if err := h.pluginAdmin.DisableInstance(c.Request.Context(), "automation", binding.PluginID, binding.InstanceID); err != nil {
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
	items, err := h.pluginAdmin.List(c.Request.Context())
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
		cfgJSON, err := h.pluginAdmin.GetConfigInstance(c.Request.Context(), "automation", binding.PluginID, binding.InstanceID)
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
		if _, err := h.pluginAdmin.GetConfigInstance(c.Request.Context(), "automation", binding.PluginID, binding.InstanceID); err == nil {
			return binding, nil
		}
	}
	return automationBinding{}, domain.ErrAutomationPluginInstanceNotFound
}

func (h *Handler) collectAutomationBindingCandidates(c *gin.Context) []automationBinding {
	candidates := make([]automationBinding, 0, 4)
	if h.goodsTypes != nil {
		items, err := h.goodsTypes.List(c.Request.Context())
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
	var query struct {
		Mode string `form:"mode" binding:"omitempty,oneof=merge replace"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	result, err := h.integration.SyncAutomation(c.Request.Context(), query.Mode)
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
	var query struct {
		Target string `form:"target"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	items, total, err := h.integration.ListSyncLogs(c.Request.Context(), query.Target, limit, offset)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toIntegrationSyncLogDTOs(items), "total": total})
}
