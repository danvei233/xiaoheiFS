package http

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"html"
	"net/http"
	"strconv"
	"strings"
	"time"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

const (
	// maxSMSTestPhones caps the recipient count of a single SMS test request.
	maxSMSTestPhones = 10
	// maxMessageTestVars caps the number of template variables in a test send.
	maxMessageTestVars = 20
)

// smsTestPayload is the validator-first DTO for the SMS test endpoint.
type smsTestPayload struct {
	Phone              string         `json:"phone"`
	TemplateID         *int64         `json:"template_id"`
	Content            string         `json:"content"`
	Variables          map[string]any `json:"variables" validate:"omitempty,max=20"`
	PluginID           string         `json:"plugin_id"`
	InstanceID         string         `json:"instance_id"`
	ProviderTemplateID string         `json:"provider_template_id"`
}

// smtpTestPayload is the validator-first DTO for the SMTP test endpoint.
type smtpTestPayload struct {
	To           string         `json:"to" validate:"omitempty,email"`
	TemplateName string         `json:"template_name"`
	Subject      string         `json:"subject"`
	Body         string         `json:"body"`
	Variables    map[string]any `json:"variables" validate:"omitempty,max=20"`
	HTML         bool           `json:"html"`
}

// splitTestPhones splits a comma/semicolon/whitespace separated phone string.
func splitTestPhones(raw string) []string {
	out := make([]string, 0, 4)
	for _, p := range strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\n' || r == '\t'
	}) {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// normalizeTestPhone reduces a phone to a comparable form (optional leading '+'
// plus digits), used to check that a test send targets the admin's own contact.
func normalizeTestPhone(phone string) string {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return ""
	}
	if phone[0] == '+' {
		phone = phone[1:]
	}
	var b strings.Builder
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// stableSMSValidationError maps the shared SMS validation errors to stable
// domain codes so the client never sees a raw validator message.
func stableSMSValidationError(err error) error {
	switch {
	case errors.Is(err, appshared.ErrSMSTooManyRecipients):
		return domain.ErrSMSTooManyRecipients
	case errors.Is(err, appshared.ErrSMSInvalidPhone):
		return domain.ErrSMSInvalidPhone
	case errors.Is(err, appshared.ErrSMSContentRequired):
		return domain.ErrSMSContentRequired
	case errors.Is(err, appshared.ErrSMSContentTooLong):
		return domain.ErrSMSContentTooLong
	default:
		return domain.ErrInvalidInput
	}
}

// allowMessageTestBudget enforces the DB atomic budgets for a test-send cost
// endpoint. Per-admin, per-IP, per-provider and global buckets must all allow
// the call; being DB-backed the budget is consistent across instances and
// restarts. Returns false (and the caller responds 429) when any bucket is
// exhausted or the budget backend is unavailable.
func (h *Handler) allowMessageTestBudget(ctx context.Context, kind, adminID, ip, providerKey string) bool {
	if h.settingsSvc == nil {
		return false
	}
	type bucket struct {
		key    string
		max    int
		window time.Duration
	}
	buckets := []bucket{
		{kind + ":admin:" + adminID, 10, time.Minute},
		{kind + ":ip:" + ip, 20, time.Minute},
		{kind + ":global", 100, time.Hour},
	}
	if providerKey != "" {
		buckets = append(buckets, bucket{kind + ":provider:" + providerKey, 30, time.Minute})
	}
	for _, b := range buckets {
		ok, err := h.settingsSvc.AllowRateLimit(ctx, b.key, b.max, b.window)
		if err != nil || !ok {
			return false
		}
	}
	return true
}

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
		writeBodyError(c, err)
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
	smsUpdates := []struct {
		key   string
		value string
	}{
		{"sms_enabled", boolToString(payload.Enabled)},
		{"sms_plugin_id", pluginID},
		{"sms_instance_id", instanceID},
		{"sms_default_template_id", strings.TrimSpace(payload.DefaultTemplateID)},
		{"sms_provider_template_id", strings.TrimSpace(payload.ProviderTemplateID)},
	}
	// Never swallow a settings write error: a silently ignored failure would
	// leave the SMS config partially updated while the API reports success.
	for _, u := range smsUpdates {
		if err := h.adminSvc.UpdateSetting(c.Request.Context(), adminID, u.key, u.value); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminSMSPreview(c *gin.Context) {
	var payload struct {
		TemplateID *int64         `json:"template_id"`
		Content    string         `json:"content"`
		Variables  map[string]any `json:"variables"`
	}
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
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
	var payload smsTestPayload
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	if h.smsSender == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPluginManagerUnavailable.Error()})
		return
	}
	adminID := getUserID(c)
	// Resolve the current admin's own verified phone; SMS test sends are only
	// permitted to that contact, never to arbitrary recipients.
	adminPhone := ""
	if h.authSvc != nil {
		if adminUser, err := h.authSvc.GetUser(c.Request.Context(), adminID); err == nil {
			adminPhone = strings.TrimSpace(adminUser.Phone)
		}
	}
	phones := splitTestPhones(payload.Phone)
	if len(phones) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrPhoneRequired.Error()})
		return
	}
	if len(phones) > maxSMSTestPhones {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrSMSTooManyRecipients.Error()})
		return
	}
	for _, p := range phones {
		if adminPhone == "" || normalizeTestPhone(p) != normalizeTestPhone(adminPhone) {
			c.JSON(http.StatusForbidden, gin.H{"error": domain.ErrSMSTestTargetForbidden.Error()})
			return
		}
	}
	// DB atomic budget across admin/IP/provider/global dimensions; consistent
	// across instances and restarts (replaces the old process map).
	ip := strings.TrimSpace(c.ClientIP())
	if !h.allowMessageTestBudget(c.Request.Context(), "sms_test", strconv.FormatInt(adminID, 10), ip, strings.TrimSpace(payload.PluginID)) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": domain.ErrTooManyRequests.Error()})
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
	// Structural validation (phone shape, content bounds) via the shared
	// validator; errors map to stable codes.
	msg := appshared.SMSMessage{Content: content, Phones: phones}
	if err := msg.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": stableSMSValidationError(err).Error()})
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

	delivery, err := h.smsSender.Send(c.Request.Context(), pluginID, instanceID, appshared.SMSMessage{
		TemplateID: providerTemplateID,
		Content:    content,
		Phones:     phones,
	})
	if err != nil {
		// Never echo the underlying provider error (it can carry the phone or
		// provider details) back to the client or into the response.
		c.JSON(http.StatusBadGateway, gin.H{"error": domain.ErrSMSDeliveryFailed.Error()})
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
		writeBodyError(c, err)
		return
	}
	var uri struct {
		ID int64 `uri:"id" binding:"omitempty,gt=0"`
	}
	if err := c.ShouldBindUri(&uri); err == nil && uri.ID > 0 {
		payload.ID = uri.ID
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
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	items, err := h.loadSMSTemplates(c)
	if err != nil {
		items = defaultSMSTemplates()
	}
	out := make([]smsTemplateItem, 0, len(items))
	for _, item := range items {
		if item.ID == uri.ID {
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
		"pass":    settingsSecretMask,
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
		writeBodyError(c, err)
		return
	}
	adminID := getUserID(c)
	// Masked/empty pass means "keep existing secret"; never overwrite with the
	// placeholder (R-6 regression from masking).
	if !isRedactedSecretValue(payload.Pass) {
		if err := h.adminSvc.UpdateSetting(c.Request.Context(), adminID, "smtp_pass", payload.Pass); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	updates := []struct {
		key   string
		value string
	}{
		{"smtp_host", payload.Host},
		{"smtp_port", payload.Port},
		{"smtp_user", payload.User},
		{"smtp_from", payload.From},
		{"smtp_enabled", boolToString(payload.Enabled)},
	}
	for _, u := range updates {
		if err := h.adminSvc.UpdateSetting(c.Request.Context(), adminID, u.key, u.value); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminSMTPTest(c *gin.Context) {
	if h.adminSvc == nil && h.settingsSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	var payload smtpTestPayload
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	if payload.To == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrToRequired.Error()})
		return
	}
	if strings.ContainsAny(payload.To, "\r\n") {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	adminID := getUserID(c)
	// SMTP test sends are only permitted to the current admin's own verified
	// email, never to arbitrary recipients.
	adminEmail := ""
	if h.authSvc != nil {
		if adminUser, err := h.authSvc.GetUser(c.Request.Context(), adminID); err == nil {
			adminEmail = strings.ToLower(strings.TrimSpace(adminUser.Email))
		}
	}
	if adminEmail == "" || strings.ToLower(strings.TrimSpace(payload.To)) != adminEmail {
		c.JSON(http.StatusForbidden, gin.H{"error": domain.ErrSMTPTestTargetForbidden.Error()})
		return
	}
	// DB atomic budget across admin/IP/global dimensions (replaces the process map).
	ip := strings.TrimSpace(c.ClientIP())
	if !h.allowMessageTestBudget(c.Request.Context(), "smtp_test", strconv.FormatInt(adminID, 10), ip, "") {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": domain.ErrTooManyRequests.Error()})
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
	if err := h.emailSender.Send(c.Request.Context(), payload.To, subject, body); err != nil {
		// Never echo the underlying SMTP error (it can carry recipient/server
		// details) back to the client.
		c.JSON(http.StatusBadGateway, gin.H{"error": domain.ErrSMTPDeliveryFailed.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
