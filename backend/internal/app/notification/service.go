package notification

import (
	"context"
	"strconv"
	"strings"
	"time"

	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
)

type messageCenter interface {
	NotifyUser(ctx context.Context, userID int64, typ, title, content string) error
}

type Service struct {
	settings appports.SettingsRepository
	vps      appports.VPSRepository
	users    appports.UserRepository
	email    appports.EmailSender
	messages messageCenter
}

func NewService(
	settings appports.SettingsRepository,
	vps appports.VPSRepository,
	users appports.UserRepository,
	email appports.EmailSender,
	messages messageCenter,
) *Service {
	return &Service{
		settings: settings,
		vps:      vps,
		users:    users,
		email:    email,
		messages: messages,
	}
}

func (s *Service) SendExpireReminders(ctx context.Context) error {
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
		if v, convErr := strconv.Atoi(daysSetting.ValueJSON); convErr == nil {
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
		user, userErr := s.users.GetUserByID(ctx, inst.UserID)
		if userErr != nil || user.Email == "" || inst.ExpireAt == nil {
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
		renderedSubject := appshared.RenderTemplate(subject, data, false)
		renderedBody := appshared.RenderTemplate(body, data, appshared.IsHTMLContent(body))
		_ = s.email.Send(ctx, user.Email, renderedSubject, renderedBody)
		if s.messages != nil {
			_ = s.messages.NotifyUser(ctx, user.ID, "expire", "VPS Expiration Reminder", "Your VPS "+inst.Name+" will expire on "+inst.ExpireAt.Format("2006-01-02"))
		}
	}
	return nil
}
