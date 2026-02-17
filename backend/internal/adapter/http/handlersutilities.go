package http

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/pkg/money"
)

func (h *Handler) loadSMSTemplates(ctx *gin.Context) ([]smsTemplateItem, error) {
	raw := strings.TrimSpace(h.getSettingValueByKey(ctx, "sms_templates_json"))
	if raw == "" {
		return []smsTemplateItem{}, nil
	}
	var items []smsTemplateItem
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].CreatedAt.IsZero() {
			items[i].CreatedAt = time.Now()
		}
		if items[i].UpdatedAt.IsZero() {
			items[i].UpdatedAt = items[i].CreatedAt
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].ID > items[j].ID
	})
	return items, nil
}

func defaultSMSTemplates() []smsTemplateItem {
	now := time.Now()
	return []smsTemplateItem{
		{ID: 1, Name: "register_verify_code", Content: "【XXX】您正在注册XXX平台账号，验证码是：{{code}}，3分钟内有效，请及时输入。", Enabled: true, CreatedAt: now, UpdatedAt: now},
		{ID: 2, Name: "login_ip_change_alert", Content: "【XXX】登录提醒：您的账号于 {{time}} 在 {{city}} 发生登录（IP：{{ip}}）。如为本人操作，请忽略本消息；如非本人操作，请立即修改密码并开启二次验证，确保账号安全。", Enabled: true, CreatedAt: now, UpdatedAt: now},
		{ID: 3, Name: "password_reset_verify_code", Content: "【XXX】您好，您在XXX平台（APP）的账号正在进行找回密码操作，切勿将验证码泄露于他人，10分钟内有效。验证码：{{code}}。", Enabled: true, CreatedAt: now, UpdatedAt: now},
		{ID: 4, Name: "phone_bind_verify_code", Content: "【XXX】手机绑定验证码：{{code}}，感谢您的支持！如非本人操作，请忽略本短信。", Enabled: true, CreatedAt: now, UpdatedAt: now},
	}
}

func defaultEmailTemplates() []domain.EmailTemplate {
	return []domain.EmailTemplate{
		{Name: "register_verify_code", Subject: "注册验证码", Body: "您好，您的注册验证码是：{{code}}，请在有效期内完成验证。", Enabled: true},
		{Name: "login_ip_change_alert", Subject: "登录提醒", Body: "您的账号于 {{time}} 在 {{city}} 登录（IP：{{ip}}）。如非本人操作请立即修改密码。", Enabled: true},
		{Name: "password_reset_verify_code", Subject: "找回密码验证码", Body: "您好，您正在进行找回密码操作，验证码：{{code}}，10分钟内有效。", Enabled: true},
		{Name: "email_bind_verify_code", Subject: "邮箱绑定验证码", Body: "您的邮箱绑定验证码：{{code}}，10分钟内有效。", Enabled: true},
	}
}

func (h *Handler) saveSMSTemplates(ctx *gin.Context, adminID int64, items []smsTemplateItem) error {
	for i := range items {
		items[i].Name = strings.TrimSpace(items[i].Name)
		items[i].Content = strings.TrimSpace(items[i].Content)
		if items[i].CreatedAt.IsZero() {
			items[i].CreatedAt = time.Now()
		}
		if items[i].UpdatedAt.IsZero() {
			items[i].UpdatedAt = time.Now()
		}
	}
	raw, err := json.Marshal(items)
	if err != nil {
		return err
	}
	if h.adminSvc == nil {
		return domain.ErrAdminServiceUnavailable
	}
	return h.adminSvc.UpdateSetting(ctx, adminID, "sms_templates_json", string(raw))
}

func (h *Handler) renderSMSTemplateByID(ctx *gin.Context, id int64, vars map[string]any) (string, bool) {
	items, err := h.loadSMSTemplates(ctx)
	if err != nil {
		return "", false
	}
	for _, item := range items {
		if item.ID != id {
			continue
		}
		if !item.Enabled {
			return "", false
		}
		return renderSMSText(item.Content, vars), true
	}
	return "", false
}

func (h *Handler) renderSMSTemplateByName(ctx *gin.Context, name string, vars map[string]any) (string, bool) {
	items, err := h.loadSMSTemplates(ctx)
	if err != nil {
		return "", false
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false
	}
	for _, item := range items {
		if strings.TrimSpace(item.Name) != name {
			continue
		}
		if !item.Enabled {
			return "", false
		}
		return renderSMSText(item.Content, vars), true
	}
	return "", false
}

func nextSMSTemplateID(items []smsTemplateItem) int64 {
	var maxID int64
	for _, item := range items {
		if item.ID > maxID {
			maxID = item.ID
		}
	}
	return maxID + 1
}

func renderSMSText(content string, vars map[string]any) string {
	normalized := normalizeSimpleTemplateVars(content)
	return strings.TrimSpace(appshared.RenderTemplate(normalized, vars, false))
}

func normalizeSimpleTemplateVars(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	return simpleTemplateVarRE.ReplaceAllString(content, "{{.$1}}")
}

func (h *Handler) getSetting(ctx *gin.Context, key string) (domain.Setting, error) {
	if h.adminSvc != nil {
		if item, err := h.adminSvc.GetSetting(ctx, key); err == nil {
			return item, nil
		}
	}
	if h.settingsSvc == nil {
		return domain.Setting{}, appshared.ErrNotFound
	}
	return h.settingsSvc.Get(ctx, key)
}

func (h *Handler) getSettingByContext(ctx context.Context, key string) (domain.Setting, error) {
	if h.adminSvc != nil {
		if item, err := h.adminSvc.GetSetting(ctx, key); err == nil {
			return item, nil
		}
	}
	if h.settingsSvc == nil {
		return domain.Setting{}, appshared.ErrNotFound
	}
	return h.settingsSvc.Get(ctx, key)
}

func (h *Handler) getSettingValueByKey(ctx *gin.Context, key string) string {
	item, err := h.getSetting(ctx, key)
	if err != nil {
		return ""
	}
	return item.ValueJSON
}

func (h *Handler) listSettings(ctx *gin.Context) ([]domain.Setting, error) {
	if h.adminSvc != nil {
		if items, err := h.adminSvc.ListSettings(ctx); err == nil {
			return items, nil
		}
	}
	if h.settingsSvc == nil {
		return nil, appshared.ErrNotFound
	}
	return h.settingsSvc.List(ctx)
}

func (h *Handler) listEmailTemplates(ctx *gin.Context) ([]domain.EmailTemplate, error) {
	if h.adminSvc != nil {
		if items, err := h.adminSvc.ListEmailTemplates(ctx); err == nil {
			return items, nil
		}
	}
	if h.settingsSvc == nil {
		return nil, appshared.ErrNotFound
	}
	return h.settingsSvc.ListEmailTemplates(ctx)
}

func boolToString(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func normalizeChannels(items []string) []string {
	out := make([]string, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		v := strings.ToLower(strings.TrimSpace(item))
		if v != "email" && v != "sms" {
			continue
		}
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

func hasChannel(channels []string, target string) bool {
	target = strings.ToLower(strings.TrimSpace(target))
	for _, ch := range channels {
		if strings.ToLower(strings.TrimSpace(ch)) == target {
			return true
		}
	}
	return false
}

func maskPhone(phone string) string {
	phone = strings.TrimSpace(phone)
	if len(phone) < 7 {
		return phone
	}
	return phone[:2] + "*****" + phone[len(phone)-2:]
}

func maskEmail(email string) string {
	email = strings.TrimSpace(email)
	i := strings.Index(email, "@")
	if i <= 1 {
		return email
	}
	return email[:1] + "*****" + email[i-1:]
}

func randomToken(n int) string {
	if n <= 0 {
		n = 32
	}
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func isDigits(input string) bool {
	for _, r := range strings.TrimSpace(input) {
		if r < '0' || r > '9' {
			return false
		}
	}
	return input != ""
}

func parseAmountCents(value any) (int64, error) {
	switch v := value.(type) {
	case nil:
		return 0, money.ErrInvalidAmount
	case string:
		return money.ParseNumberStringToCents(v)
	case json.Number:
		return money.ParseNumberStringToCents(v.String())
	case float64:
		return floatToCents(v), nil
	case float32:
		return floatToCents(float64(v)), nil
	case int:
		return int64(v) * 100, nil
	case int64:
		return v * 100, nil
	default:
		return 0, money.ErrInvalidAmount
	}
}
