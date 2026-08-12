package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/mail"
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

	// Validate envelope addresses and obtain header-safe forms. Recipient and
	// sender flow into SMTP commands (MAIL/RCPT) and headers; unvalidated CR/LF
	// would allow header injection and SMTP command injection.
	fromAddr, fromHeader, err := parseMailAddress(from)
	if err != nil {
		return fmt.Errorf("invalid smtp_from: %w", err)
	}
	toAddr, toHeader, err := parseMailAddress(to)
	if err != nil {
		return fmt.Errorf("invalid recipient: %w", err)
	}
	subject = sanitizeHeaderField(subject)

	contentType := "text/plain"
	if isHTMLContent(body) {
		contentType = "text/html"
	}
	msg := []byte(fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: %s; charset=UTF-8\r\n\r\n%s", fromHeader, toHeader, subject, contentType, body))

	addr := fmt.Sprintf("%s:%s", host, port)

	// 端口465需要SSL连接，其他端口使用STARTTLS
	if port == "465" {
		return s.sendWithSSL(ctx, addr, host, user, pass, fromAddr, toAddr, msg)
	}
	return s.sendWithStartTLS(ctx, addr, host, user, pass, fromAddr, toAddr, msg)
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
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

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

// sendWithStartTLS 使用STARTTLS（端口587等）。必须在服务器宣告 STARTTLS
// 扩展后才升级到 TLS；否则拒绝发送，避免凭据/内容以明文传输。
func (s *Sender) sendWithStartTLS(ctx context.Context, addr, host, user, pass, from, to string, msg []byte) error {
	// 创建带超时的context
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	conn, err := (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp dial failed: %w", err)
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("create smtp client failed: %w", err)
	}
	defer client.Quit()

	// 强制 STARTTLS：服务器未宣告该扩展时直接失败，不降级为明文认证/投递。
	ok, _ := client.Extension("STARTTLS")
	if !ok {
		return fmt.Errorf("smtp server does not support STARTTLS")
	}
	tlsConfig := &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: false,
	}
	if err := client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("starttls failed: %w", err)
	}

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

func (s *Sender) getSetting(ctx context.Context, key string) (string, error) {
	setting, err := s.settings.GetSetting(ctx, key)
	if err != nil {
		return "", err
	}
	return setting.ValueJSON, nil
}

// parseMailAddress validates a single RFC 5322 address, rejecting any input
// containing CR/LF, and returns the bare mailbox (for MAIL/RCPT) plus a
// header-safe canonical form (for the From:/To: headers).
func parseMailAddress(raw string) (addr string, header string, err error) {
	if strings.ContainsAny(raw, "\r\n") {
		return "", "", fmt.Errorf("address contains line break")
	}
	parsed, err := mail.ParseAddress(strings.TrimSpace(raw))
	if err != nil {
		return "", "", fmt.Errorf("malformed address %q: %w", raw, err)
	}
	if strings.TrimSpace(parsed.Address) == "" {
		return "", "", fmt.Errorf("empty address")
	}
	return parsed.Address, parsed.String(), nil
}

// sanitizeHeaderField removes CR/LF from a header value so it cannot inject
// additional headers or SMTP commands.
func sanitizeHeaderField(value string) string {
	return strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' {
			return -1
		}
		return r
	}, value)
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
