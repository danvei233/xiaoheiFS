package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"

	"github.com/gin-gonic/gin"
)

func (h *Handler) AdminRobotConfig(c *gin.Context) {
	if h.adminSvc == nil && h.settingsSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	webhooks := appshared.ParseRobotWebhookConfigs(h.getSettingValueByKey(c, "robot_webhooks"))
	// Mask both the legacy top-level secret and every per-webhook secret so the
	// signing credentials are never echoed back to the admin UI.
	for i := range webhooks {
		webhooks[i].Secret = redactSecretValue(webhooks[i].Secret)
	}
	c.JSON(http.StatusOK, gin.H{
		"url":      h.getSettingValueByKey(c, "robot_webhook_url"),
		"secret":   redactSecretValue(h.getSettingValueByKey(c, "robot_webhook_secret")),
		"enabled":  strings.ToLower(h.getSettingValueByKey(c, "robot_webhook_enabled")) == "true",
		"webhooks": webhooks,
	})
}

func webhookIdentity(w appshared.RobotWebhookConfig) string {
	if strings.TrimSpace(w.URL) != "" {
		return strings.ToLower(strings.TrimSpace(w.URL))
	}
	return strings.ToLower(strings.TrimSpace(w.Name))
}

func (h *Handler) AdminRobotConfigUpdate(c *gin.Context) {
	var payload struct {
		URL      string                         `json:"url"`
		Secret   string                         `json:"secret"`
		Enabled  bool                           `json:"enabled"`
		Webhooks []appshared.RobotWebhookConfig `json:"webhooks"`
	}
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	if payload.Webhooks != nil {
		// Preserve masked per-webhook secrets: the admin UI only ever sees the
		// "***" placeholder, so a save must not overwrite real secrets with it.
		existing := appshared.ParseRobotWebhookConfigs(h.getSettingValueByKey(c, "robot_webhooks"))
		byKey := make(map[string]string, len(existing))
		for _, w := range existing {
			byKey[webhookIdentity(w)] = strings.TrimSpace(w.Secret)
		}
		for i := range payload.Webhooks {
			if !isRedactedSecretValue(payload.Webhooks[i].Secret) {
				continue
			}
			if real := byKey[webhookIdentity(payload.Webhooks[i])]; real != "" {
				payload.Webhooks[i].Secret = real
			}
		}
		raw, err := json.Marshal(payload.Webhooks)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
			return
		}
		if err := h.adminSvc.UpdateSetting(c.Request.Context(), getUserID(c), "robot_webhooks", string(raw)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}
	if payload.URL != "" || payload.Secret != "" || payload.Enabled {
		if err := h.adminSvc.UpdateSetting(c.Request.Context(), getUserID(c), "robot_webhook_url", payload.URL); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// A masked placeholder (or absent secret) must never overwrite the real
		// secret; a settings write failure must not be swallowed, otherwise the
		// webhook config reports success while only part of it was persisted.
		if !isRedactedSecretValue(payload.Secret) {
			if err := h.adminSvc.UpdateSetting(c.Request.Context(), getUserID(c), "robot_webhook_secret", payload.Secret); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		}
		if err := h.adminSvc.UpdateSetting(c.Request.Context(), getUserID(c), "robot_webhook_enabled", boolToString(payload.Enabled)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNoUpdates.Error()})
}

func (h *Handler) AdminRobotTest(c *gin.Context) {
	if h.adminSvc == nil && h.settingsSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	if h.broker == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrEventBrokerNotAvailable.Error()})
		return
	}
	var payload struct {
		Event string `json:"event"`
		Data  any    `json:"data"`
	}
	if err := bindJSONOptional(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	eventType := strings.TrimSpace(payload.Event)
	if eventType == "" {
		eventType = "webhook.test"
	}
	ev, err := h.broker.Publish(c.Request.Context(), 0, eventType, map[string]any{
		"event":     eventType,
		"timestamp": time.Now().Unix(),
		"data":      payload.Data,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.robotNotifier != nil {
		_ = h.robotNotifier.NotifyOrderEvent(c.Request.Context(), ev)
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminRealNameConfig(c *gin.Context) {
	if h.realNameSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	enabled, provider, actions := h.realNameSvc.GetConfig(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{
		"enabled":       enabled,
		"provider":      provider,
		"block_actions": actions,
	})
}

func (h *Handler) AdminRealNameConfigUpdate(c *gin.Context) {
	if h.realNameSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	var payload struct {
		Enabled      bool     `json:"enabled"`
		Provider     string   `json:"provider"`
		BlockActions []string `json:"block_actions"`
	}
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	if err := h.realNameSvc.UpdateConfig(c.Request.Context(), payload.Enabled, payload.Provider, payload.BlockActions); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminRealNameProviders(c *gin.Context) {
	if h.realNameSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	type providerInfo struct {
		Key  string `json:"key"`
		Name string `json:"name"`
	}
	out := []providerInfo{}
	for _, provider := range h.realNameSvc.Providers() {
		out = append(out, providerInfo{Key: provider.Key(), Name: provider.Name()})
	}
	c.JSON(http.StatusOK, gin.H{"items": out})
}

func (h *Handler) AdminRealNameRecords(c *gin.Context) {
	if h.realNameSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	limit, offset := paging(c)
	var query struct {
		UserID *int64 `form:"user_id" binding:"omitempty,gt=0"`
		Status string `form:"status"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	// The frontend realname records page filters by status; previously the
	// backend silently ignored it (m059 F6).
	items, total, err := h.realNameSvc.List(c.Request.Context(), query.UserID, strings.TrimSpace(query.Status), limit, offset)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp := make([]RealNameVerificationDTO, 0, len(items))
	for _, item := range items {
		resp = append(resp, toRealNameVerificationDTO(item, h.realNameRedirectURL(c.Request.Context(), item)))
	}
	c.JSON(http.StatusOK, gin.H{"items": resp, "total": total})
}
