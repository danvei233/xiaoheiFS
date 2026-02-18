package http

import (
	"github.com/gin-gonic/gin"
	"html"
	"net/http"
	"strconv"
	"strings"
	"time"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

type smsTemplateItem struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (h *Handler) AdminSMSConfig(c *gin.Context) {
	if h.adminSvc == nil && h.settingsSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	enabledRaw := strings.TrimSpace(h.getSettingValueByKey(c, "sms_enabled"))
	enabled := true
	if enabledRaw != "" {
		enabled = strings.EqualFold(enabledRaw, "true") || enabledRaw == "1"
	}
	c.JSON(http.StatusOK, gin.H{
		"enabled":              enabled,
		"plugin_id":            strings.TrimSpace(h.getSettingValueByKey(c, "sms_plugin_id")),
		"instance_id":          strings.TrimSpace(h.getSettingValueByKey(c, "sms_instance_id")),
		"default_template_id":  strings.TrimSpace(h.getSettingValueByKey(c, "sms_default_template_id")),
		"provider_template_id": strings.TrimSpace(h.getSettingValueByKey(c, "sms_provider_template_id")),
	})
}

func (h *Handler) AdminSMSConfigUpdate(c *gin.Context) {
	var payload struct {
		Enabled            bool   `json:"enabled"`
		PluginID           string `json:"plugin_id"`
		InstanceID         string `json:"instance_id"`
		DefaultTemplateID  string `json:"default_template_id"`
		ProviderTemplateID string `json:"provider_template_id"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	pluginID := strings.TrimSpace(payload.PluginID)
	instanceID := strings.TrimSpace(payload.InstanceID)
	if pluginID != "" && instanceID == "" {
		instanceID = "default"
	}
	if pluginID == "" {
		instanceID = ""
	}
	adminID := getUserID(c)
	_ = h.adminSvc.UpdateSetting(c, adminID, "sms_enabled", boolToString(payload.Enabled))
	_ = h.adminSvc.UpdateSetting(c, adminID, "sms_plugin_id", pluginID)
	_ = h.adminSvc.UpdateSetting(c, adminID, "sms_instance_id", instanceID)
	_ = h.adminSvc.UpdateSetting(c, adminID, "sms_default_template_id", strings.TrimSpace(payload.DefaultTemplateID))
	_ = h.adminSvc.UpdateSetting(c, adminID, "sms_provider_template_id", strings.TrimSpace(payload.ProviderTemplateID))
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminSMSPreview(c *gin.Context) {
	var payload struct {
		TemplateID *int64         `json:"template_id"`
		Content    string         `json:"content"`
		Variables  map[string]any `json:"variables"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	vars := map[string]any{"now": time.Now().Format(time.RFC3339)}
	for k, v := range payload.Variables {
		vars[k] = v
	}
	content := strings.TrimSpace(payload.Content)
	if payload.TemplateID != nil && *payload.TemplateID > 0 {
		rendered, ok := h.renderSMSTemplateByID(c, *payload.TemplateID, vars)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrTemplateNotFound.Error()})
			return
		}
		content = rendered
	} else if content != "" {
		content = renderSMSText(content, vars)
	}
	if strings.TrimSpace(content) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrContentRequired.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"content": content})
}

func (h *Handler) AdminSMSTest(c *gin.Context) {
	if h.adminSvc == nil && h.settingsSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	var payload struct {
		Phone              string         `json:"phone"`
		TemplateID         *int64         `json:"template_id"`
		Content            string         `json:"content"`
		Variables          map[string]any `json:"variables"`
		PluginID           string         `json:"plugin_id"`
		InstanceID         string         `json:"instance_id"`
		ProviderTemplateID string         `json:"provider_template_id"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	if h.smsSender == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPluginManagerUnavailable.Error()})
		return
	}
	phoneRaw := strings.TrimSpace(payload.Phone)
	if phoneRaw == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPhoneRequired.Error()})
		return
	}
	phones := make([]string, 0, 4)
	for _, p := range strings.FieldsFunc(phoneRaw, func(r rune) bool { return r == ',' || r == ';' || r == ' ' || r == '\n' || r == '\t' }) {
		p = strings.TrimSpace(p)
		if p != "" {
			phones = append(phones, p)
		}
	}
	if len(phones) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPhoneRequired.Error()})
		return
	}

	vars := map[string]any{"now": time.Now().Format(time.RFC3339)}
	for k, v := range payload.Variables {
		vars[k] = v
	}
	content := strings.TrimSpace(payload.Content)
	if payload.TemplateID != nil && *payload.TemplateID > 0 {
		rendered, ok := h.renderSMSTemplateByID(c, *payload.TemplateID, vars)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrTemplateNotFound.Error()})
			return
		}
		content = rendered
	} else if content != "" {
		content = renderSMSText(content, vars)
	} else {
		defaultTemplateID := strings.TrimSpace(h.getSettingValueByKey(c, "sms_default_template_id"))
		if defaultTemplateID != "" {
			if tid, err := strconv.ParseInt(defaultTemplateID, 10, 64); err == nil && tid > 0 {
				if rendered, ok := h.renderSMSTemplateByID(c, tid, vars); ok {
					content = rendered
				}
			}
		}
		if strings.TrimSpace(content) == "" {
			if items, err := h.loadSMSTemplates(c); err == nil {
				for _, item := range items {
					if !item.Enabled {
						continue
					}
					content = renderSMSText(item.Content, vars)
					break
				}
			}
		}
	}
	if strings.TrimSpace(content) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrContentRequired.Error()})
		return
	}

	pluginID := strings.TrimSpace(payload.PluginID)
	instanceID := strings.TrimSpace(payload.InstanceID)
	if pluginID == "" {
		pluginID = strings.TrimSpace(h.getSettingValueByKey(c, "sms_plugin_id"))
	}
	if instanceID == "" {
		instanceID = strings.TrimSpace(h.getSettingValueByKey(c, "sms_instance_id"))
	}
	if instanceID == "" {
		instanceID = "default"
	}
	if pluginID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrSMSPluginNotConfigured.Error()})
		return
	}
	providerTemplateID := strings.TrimSpace(payload.ProviderTemplateID)
	if providerTemplateID == "" {
		providerTemplateID = strings.TrimSpace(h.getSettingValueByKey(c, "sms_provider_template_id"))
	}

	delivery, err := h.smsSender.Send(c, pluginID, instanceID, appshared.SMSMessage{
		TemplateID: providerTemplateID,
		Content:    content,
		Phones:     phones,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"ok":          true,
		"message_id":  strings.TrimSpace(delivery.MessageID),
		"plugin_id":   pluginID,
		"instance_id": instanceID,
	})
}

func (h *Handler) AdminSMSTemplates(c *gin.Context) {
	items, err := h.loadSMSTemplates(c)
	if err != nil {
		items = defaultSMSTemplates()
		if h.adminSvc != nil {
			_ = h.saveSMSTemplates(c, getUserID(c), items)
		}
	}
	if len(items) == 0 {
		items = defaultSMSTemplates()
		if h.adminSvc != nil {
			_ = h.saveSMSTemplates(c, getUserID(c), items)
		}
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) AdminSMSTemplateUpsert(c *gin.Context) {
	var payload smsTemplateItem
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	if idParam := strings.TrimSpace(c.Param("id")); idParam != "" {
		if id, err := strconv.ParseInt(idParam, 10, 64); err == nil {
			payload.ID = id
		}
	}
	payload.Name = strings.TrimSpace(payload.Name)
	payload.Content = strings.TrimSpace(payload.Content)
	if payload.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNameRequired.Error()})
		return
	}
	if payload.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrContentRequired.Error()})
		return
	}
	items, err := h.loadSMSTemplates(c)
	if err != nil {
		items = defaultSMSTemplates()
	}
	now := time.Now()
	if payload.ID <= 0 {
		payload.ID = nextSMSTemplateID(items)
		payload.CreatedAt = now
		payload.UpdatedAt = now
		items = append(items, payload)
	} else {
		updated := false
		for i := range items {
			if items[i].ID != payload.ID {
				continue
			}
			payload.CreatedAt = items[i].CreatedAt
			if payload.CreatedAt.IsZero() {
				payload.CreatedAt = now
			}
			payload.UpdatedAt = now
			items[i] = payload
			updated = true
			break
		}
		if !updated {
			payload.CreatedAt = now
			payload.UpdatedAt = now
			items = append(items, payload)
		}
	}
	if err := h.saveSMSTemplates(c, getUserID(c), items); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, payload)
}

func (h *Handler) AdminSMSTemplateDelete(c *gin.Context) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	items, err := h.loadSMSTemplates(c)
	if err != nil {
		items = defaultSMSTemplates()
	}
	out := make([]smsTemplateItem, 0, len(items))
	for _, item := range items {
		if item.ID == id {
			continue
		}
		out = append(out, item)
	}
	if err := h.saveSMSTemplates(c, getUserID(c), out); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminSMTPConfig(c *gin.Context) {
	if h.adminSvc == nil && h.settingsSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"host":    h.getSettingValueByKey(c, "smtp_host"),
		"port":    h.getSettingValueByKey(c, "smtp_port"),
		"user":    h.getSettingValueByKey(c, "smtp_user"),
		"pass":    h.getSettingValueByKey(c, "smtp_pass"),
		"from":    h.getSettingValueByKey(c, "smtp_from"),
		"enabled": strings.ToLower(h.getSettingValueByKey(c, "smtp_enabled")) == "true",
	})
}

func (h *Handler) AdminSMTPConfigUpdate(c *gin.Context) {
	var payload struct {
		Host    string `json:"host"`
		Port    string `json:"port"`
		User    string `json:"user"`
		Pass    string `json:"pass"`
		From    string `json:"from"`
		Enabled bool   `json:"enabled"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	_ = h.adminSvc.UpdateSetting(c, getUserID(c), "smtp_host", payload.Host)
	_ = h.adminSvc.UpdateSetting(c, getUserID(c), "smtp_port", payload.Port)
	_ = h.adminSvc.UpdateSetting(c, getUserID(c), "smtp_user", payload.User)
	_ = h.adminSvc.UpdateSetting(c, getUserID(c), "smtp_pass", payload.Pass)
	_ = h.adminSvc.UpdateSetting(c, getUserID(c), "smtp_from", payload.From)
	_ = h.adminSvc.UpdateSetting(c, getUserID(c), "smtp_enabled", boolToString(payload.Enabled))
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminSMTPTest(c *gin.Context) {
	if h.adminSvc == nil && h.settingsSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	var payload struct {
		To           string         `json:"to"`
		TemplateName string         `json:"template_name"`
		Subject      string         `json:"subject"`
		Body         string         `json:"body"`
		Variables    map[string]any `json:"variables"`
		HTML         bool           `json:"html"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	if strings.TrimSpace(payload.To) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrToRequired.Error()})
		return
	}
	subject := strings.TrimSpace(payload.Subject)
	body := payload.Body
	if payload.TemplateName != "" {
		templates, _ := h.listEmailTemplates(c)
		found := false
		for _, tmpl := range templates {
			if tmpl.Name == payload.TemplateName {
				subject = tmpl.Subject
				body = tmpl.Body
				found = true
				break
			}
		}
		if !found {
			c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrTemplateNotFound.Error()})
			return
		}
	}
	if subject == "" {
		subject = "SMTP Test"
	}
	if strings.TrimSpace(body) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrBodyRequired.Error()})
		return
	}
	data := map[string]any{
		"now": time.Now().Format(time.RFC3339),
	}
	for k, v := range payload.Variables {
		data[k] = v
	}
	subject = appshared.RenderTemplate(subject, data, false)
	body = appshared.RenderTemplate(body, data, appshared.IsHTMLContent(body))
	if payload.HTML && !appshared.IsHTMLContent(body) {
		body = "<html><body><pre>" + html.EscapeString(body) + "</pre></body></html>"
	}
	if h.emailSender == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrEmailSenderNotConfigured.Error()})
		return
	}
	if err := h.emailSender.Send(c, payload.To, subject, body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
