package http

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"strings"
	"time"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

func (h *Handler) AdminRobotConfig(c *gin.Context) {
	if h.adminSvc == nil && h.settingsSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	webhooks := appshared.ParseRobotWebhookConfigs(h.getSettingValueByKey(c, "robot_webhooks"))
	c.JSON(http.StatusOK, gin.H{
		"url":      h.getSettingValueByKey(c, "robot_webhook_url"),
		"secret":   h.getSettingValueByKey(c, "robot_webhook_secret"),
		"enabled":  strings.ToLower(h.getSettingValueByKey(c, "robot_webhook_enabled")) == "true",
		"webhooks": webhooks,
	})
}

func (h *Handler) AdminRobotConfigUpdate(c *gin.Context) {
	var payload struct {
		URL      string                         `json:"url"`
		Secret   string                         `json:"secret"`
		Enabled  bool                           `json:"enabled"`
		Webhooks []appshared.RobotWebhookConfig `json:"webhooks"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	if payload.Webhooks != nil {
		raw, _ := json.Marshal(payload.Webhooks)
		_ = h.adminSvc.UpdateSetting(c, getUserID(c), "robot_webhooks", string(raw))
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}
	if payload.URL != "" || payload.Secret != "" || payload.Enabled {
		if err := h.adminSvc.UpdateSetting(c, getUserID(c), "robot_webhook_url", payload.URL); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		_ = h.adminSvc.UpdateSetting(c, getUserID(c), "robot_webhook_secret", payload.Secret)
		_ = h.adminSvc.UpdateSetting(c, getUserID(c), "robot_webhook_enabled", boolToString(payload.Enabled))
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
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	eventType := strings.TrimSpace(payload.Event)
	if eventType == "" {
		eventType = "webhook.test"
	}
	ev, err := h.broker.Publish(c, 0, eventType, map[string]any{
		"event":     eventType,
		"timestamp": time.Now().Unix(),
		"data":      payload.Data,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.robotNotifier != nil {
		_ = h.robotNotifier.NotifyOrderEvent(c, ev)
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminRealNameConfig(c *gin.Context) {
	if h.realnameSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	enabled, provider, actions := h.realnameSvc.GetConfig(c)
	c.JSON(http.StatusOK, gin.H{
		"enabled":       enabled,
		"provider":      provider,
		"block_actions": actions,
	})
}

func (h *Handler) AdminRealNameConfigUpdate(c *gin.Context) {
	if h.realnameSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	var payload struct {
		Enabled      bool     `json:"enabled"`
		Provider     string   `json:"provider"`
		BlockActions []string `json:"block_actions"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	if err := h.realnameSvc.UpdateConfig(c, payload.Enabled, payload.Provider, payload.BlockActions); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminRealNameProviders(c *gin.Context) {
	if h.realnameSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	type providerInfo struct {
		Key  string `json:"key"`
		Name string `json:"name"`
	}
	out := []providerInfo{}
	for _, provider := range h.realnameSvc.Providers() {
		out = append(out, providerInfo{Key: provider.Key(), Name: provider.Name()})
	}
	c.JSON(http.StatusOK, gin.H{"items": out})
}

func (h *Handler) AdminRealNameRecords(c *gin.Context) {
	if h.realnameSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	limit, offset := paging(c)
	var userID *int64
	if val := c.Query("user_id"); val != "" {
		if id, err := strconv.ParseInt(val, 10, 64); err == nil {
			userID = &id
		}
	}
	items, total, err := h.realnameSvc.List(c, userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp := make([]RealNameVerificationDTO, 0, len(items))
	for _, item := range items {
		resp = append(resp, toRealNameVerificationDTO(item))
	}
	c.JSON(http.StatusOK, gin.H{"items": resp, "total": total})
}
