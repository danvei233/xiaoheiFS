package email

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"

	"xiaoheiplay/internal/usecase"
)

type Sender struct {
	settings usecase.SettingsRepository
}

func NewSender(settings usecase.SettingsRepository) *Sender {
	return &Sender{settings: settings}
}

func (s *Sender) Send(ctx context.Context, to string, subject string, body string) error {
	enabled, _ := s.getSetting(ctx, "smtp_enabled")
	if enabled != "" && enabled != "true" {
		return fmt.Errorf("smtp disabled")
	}
	host, _ := s.getSetting(ctx, "smtp_host")
	port, _ := s.getSetting(ctx, "smtp_port")
	user, _ := s.getSetting(ctx, "smtp_user")
	pass, _ := s.getSetting(ctx, "smtp_pass")
	from, _ := s.getSetting(ctx, "smtp_from")
	if host == "" || port == "" || from == "" {
		return fmt.Errorf("smtp not configured")
	}
	addr := fmt.Sprintf("%s:%s", host, port)
	auth := smtp.PlainAuth("", user, pass, host)
	contentType := "text/plain"
	if isHTMLContent(body) {
		contentType = "text/html"
	}
	msg := []byte(fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: %s; charset=UTF-8\r\n\r\n%s", from, to, subject, contentType, body))
	return smtp.SendMail(addr, auth, from, []string{to}, msg)
}

func (s *Sender) getSetting(ctx context.Context, key string) (string, error) {
	setting, err := s.settings.GetSetting(ctx, key)
	if err != nil {
		return "", err
	}
	return setting.ValueJSON, nil
}

func isHTMLContent(body string) bool {
	lower := strings.ToLower(body)
	return strings.Contains(lower, "<html") ||
		strings.Contains(lower, "<body") ||
		strings.Contains(lower, "<div") ||
		strings.Contains(lower, "<table") ||
		strings.Contains(lower, "<p") ||
		strings.Contains(lower, "<br")
}
