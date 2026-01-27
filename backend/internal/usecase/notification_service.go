package usecase

import (
	"context"
	"strconv"
	"strings"
	"time"
)

type NotificationService struct {
	settings SettingsRepository
	vps      VPSRepository
	users    UserRepository
	email    EmailSender
	messages *MessageCenterService
}

func NewNotificationService(settings SettingsRepository, vps VPSRepository, users UserRepository, email EmailSender, messages *MessageCenterService) *NotificationService {
	return &NotificationService{settings: settings, vps: vps, users: users, email: email, messages: messages}
}

func (s *NotificationService) SendExpireReminders(ctx context.Context) error {
	if s.email == nil {
		return nil
	}
	enabled, err := s.settings.GetSetting(ctx, "email_expire_enabled")
	if err != nil || strings.ToLower(enabled.ValueJSON) != "true" {
		return nil
	}
	daysSetting, _ := s.settings.GetSetting(ctx, "expire_reminder_days")
	days := 7
	if daysSetting.ValueJSON != "" {
		if v, err := strconv.Atoi(daysSetting.ValueJSON); err == nil {
			days = v
		}
	}
	before := time.Now().Add(time.Duration(days) * 24 * time.Hour)
	instances, err := s.vps.ListInstancesExpiring(ctx, before)
	if err != nil {
		return err
	}
	templates, _ := s.settings.ListEmailTemplates(ctx)
	subject := "VPS Expiration Reminder: {{.vps.name}}"
	body := "Your VPS {{.vps.name}} will expire on {{.vps.expire_at}}."
	for _, tmpl := range templates {
		if tmpl.Name == "expire_reminder" && tmpl.Enabled {
			subject = tmpl.Subject
			body = tmpl.Body
			break
		}
	}
	for _, inst := range instances {
		user, err := s.users.GetUserByID(ctx, inst.UserID)
		if err != nil || user.Email == "" || inst.ExpireAt == nil {
			continue
		}
		data := map[string]any{
			"user": map[string]any{
				"id":       user.ID,
				"username": user.Username,
				"email":    user.Email,
				"qq":       user.QQ,
			},
			"vps": map[string]any{
				"name":      inst.Name,
				"expire_at": inst.ExpireAt.Format("2006-01-02"),
			},
		}
		renderedSubject := RenderTemplate(subject, data, false)
		renderedBody := RenderTemplate(body, data, IsHTMLContent(body))
		_ = s.email.Send(ctx, user.Email, renderedSubject, renderedBody)
		if s.messages != nil {
			_ = s.messages.NotifyUser(ctx, user.ID, "expire", "VPS Expiration Reminder", "Your VPS "+inst.Name+" will expire on "+inst.ExpireAt.Format("2006-01-02"))
		}
	}
	return nil
}
