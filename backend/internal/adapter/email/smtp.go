package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	appports "xiaoheiplay/internal/app/ports"
)

type Sender struct {
	settings appports.SettingsRepository
}

func NewSender(settings appports.SettingsRepository) *Sender {
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

	contentType := "text/plain"
	if isHTMLContent(body) {
		contentType = "text/html"
	}
	msg := []byte(fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: %s; charset=UTF-8\r\n\r\n%s", from, to, subject, contentType, body))

	addr := fmt.Sprintf("%s:%s", host, port)

	// 端口465需要SSL连接，其他端口使用STARTTLS
	if port == "465" {
		return s.sendWithSSL(ctx, addr, host, user, pass, from, to, msg)
	}
	return s.sendWithStartTLS(ctx, addr, host, user, pass, from, to, msg)
}

// sendWithSSL 使用SSL/TLS直接连接（端口465）
func (s *Sender) sendWithSSL(ctx context.Context, addr, host, user, pass, from, to string, msg []byte) error {
	// 创建带超时的context
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// 建立TLS连接
	tlsConfig := &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: false,
	}
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("tls dial failed: %w", err)
	}
	defer conn.Close()

	// 创建SMTP客户端
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("create smtp client failed: %w", err)
	}
	defer client.Quit()

	// 认证
	auth := smtp.PlainAuth("", user, pass, host)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("auth failed: %w", err)
	}

	// 设置发件人和收件人
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("mail from failed: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("rcpt to failed: %w", err)
	}

	// 发送邮件内容
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("data failed: %w", err)
	}
	defer writer.Close()

	_, err = writer.Write(msg)
	return err
}

// sendWithStartTLS 使用STARTTLS（端口587等）
func (s *Sender) sendWithStartTLS(ctx context.Context, addr, host, user, pass, from, to string, msg []byte) error {
	// 创建带超时的context
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// 在goroutine中执行，以便监听超时
	type result struct {
		err error
	}
	resultChan := make(chan result, 1)

	go func() {
		auth := smtp.PlainAuth("", user, pass, host)
		resultChan <- result{err: smtp.SendMail(addr, auth, from, []string{to}, msg)}
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("smtp send timeout")
	case res := <-resultChan:
		return res.err
	}
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
