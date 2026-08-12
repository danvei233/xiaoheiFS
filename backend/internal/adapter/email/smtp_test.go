package email

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"testing"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func TestSender_Disabled(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	_ = repo.UpsertSetting(context.Background(), domain.Setting{Key: "smtp_enabled", ValueJSON: "false"})
	sender := NewSender(repo)
	if err := sender.Send(context.Background(), "a@example.com", "subj", "body"); err == nil {
		t.Fatalf("expected smtp disabled error")
	}
}

func TestIsHTMLContent(t *testing.T) {
	if !isHTMLContent("<html><body>hi</body></html>") {
		t.Fatalf("expected html content")
	}
	if isHTMLContent("plain text") {
		t.Fatalf("expected non-html")
	}
}

// TestSend_RejectsCRLFRecipient guards against SMTP header/envelope injection:
// a recipient containing CR/LF must be rejected before any dialing happens.
func TestSend_RejectsCRLFRecipient(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	for _, kv := range []domain.Setting{
		{Key: "smtp_enabled", ValueJSON: "true"},
		{Key: "smtp_host", ValueJSON: "127.0.0.1"},
		{Key: "smtp_port", ValueJSON: "2525"},
		{Key: "smtp_from", ValueJSON: "no-reply@example.com"},
	} {
		if err := repo.UpsertSetting(ctx, kv); err != nil {
			t.Fatalf("upsert setting: %v", err)
		}
	}
	sender := NewSender(repo)
	malicious := []string{
		"victim@example.com\r\nBcc: spam@evil.com",
		"victim@example.com\nRCPT TO:<spam@evil.com>",
		"victim@example.com\r\nTo: other@evil.com",
	}
	for _, to := range malicious {
		if err := sender.Send(ctx, to, "subject", "body"); err == nil {
			t.Fatalf("expected CRLF recipient %q to be rejected", to)
		}
	}
}

// TestSend_RejectsCRLFFrom guards against MAIL FROM injection via smtp_from.
func TestSend_RejectsCRLFFrom(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	for _, kv := range []domain.Setting{
		{Key: "smtp_enabled", ValueJSON: "true"},
		{Key: "smtp_host", ValueJSON: "127.0.0.1"},
		{Key: "smtp_port", ValueJSON: "2525"},
		{Key: "smtp_from", ValueJSON: "no-reply@example.com\r\nBcc: spam@evil.com"},
	} {
		if err := repo.UpsertSetting(ctx, kv); err != nil {
			t.Fatalf("upsert setting: %v", err)
		}
	}
	sender := NewSender(repo)
	if err := sender.Send(ctx, "a@example.com", "subject", "body"); err == nil {
		t.Fatalf("expected CRLF smtp_from to be rejected")
	}
}

// TestSend_RequiresSTARTTLS guards F2: the 587 path must refuse to send when
// the server does not advertise STARTTLS, instead of falling back to plaintext.
func TestSend_RequiresSTARTTLS(t *testing.T) {
	addr := startFakeSMTPServer(t, false)
	host, port, ok := strings.Cut(addr, ":")
	if !ok {
		t.Fatalf("bad addr %q", addr)
	}
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	for _, kv := range []domain.Setting{
		{Key: "smtp_enabled", ValueJSON: "true"},
		{Key: "smtp_host", ValueJSON: host},
		{Key: "smtp_port", ValueJSON: port},
		{Key: "smtp_from", ValueJSON: "no-reply@example.com"},
	} {
		if err := repo.UpsertSetting(ctx, kv); err != nil {
			t.Fatalf("upsert setting: %v", err)
		}
	}
	sender := NewSender(repo)
	err := sender.Send(ctx, "a@example.com", "subject", "body")
	if err == nil {
		t.Fatalf("expected error when server does not advertise STARTTLS")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "starttls") {
		t.Fatalf("expected STARTTLS-related error, got %v", err)
	}
}

func TestParseMailAddress(t *testing.T) {
	addr, header, err := parseMailAddress("a@example.com")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if addr != "a@example.com" || header != "<a@example.com>" {
		t.Fatalf("unexpected parse result: %q %q", addr, header)
	}
	addr, header, err = parseMailAddress("Xiaohei <no-reply@example.com>")
	if err != nil {
		t.Fatalf("parse display name: %v", err)
	}
	if addr != "no-reply@example.com" {
		t.Fatalf("expected bare addr, got %q", addr)
	}
	if strings.Contains(header, "\r") || strings.Contains(header, "\n") {
		t.Fatalf("header form contains CR/LF: %q", header)
	}
	if _, _, err := parseMailAddress("a@example.com\r\nBcc: spam@evil.com"); err == nil {
		t.Fatalf("expected CRLF address to be rejected")
	}
	if _, _, err := parseMailAddress("not-an-email"); err == nil {
		t.Fatalf("expected invalid address to be rejected")
	}
}

func TestSanitizeHeaderField(t *testing.T) {
	if got := sanitizeHeaderField("subject\r\nBcc: spam@evil.com"); strings.Contains(got, "\r") || strings.Contains(got, "\n") {
		t.Fatalf("CR/LF not stripped: %q", got)
	}
}

// startFakeSMTPServer runs a minimal SMTP server on 127.0.0.1 and returns its
// address. When advertiseStartTLS is false the EHLO greeting omits STARTTLS.
func startFakeSMTPServer(t *testing.T, advertiseStartTLS bool) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		br := bufio.NewReader(conn)
		write := func(s string) { _, _ = fmt.Fprintf(conn, "%s\r\n", s) }
		write("220 fake.example ESMTP")
		for {
			line, err := br.ReadString('\n')
			if err != nil {
				return
			}
			cmd := strings.ToUpper(strings.TrimSpace(line))
			switch {
			case strings.HasPrefix(cmd, "EHLO") || strings.HasPrefix(cmd, "HELO"):
				write("250-fake.example")
				if advertiseStartTLS {
					write("250-STARTTLS")
				}
				write("250 HELP")
			case strings.HasPrefix(cmd, "QUIT"):
				write("221 2.0.0 Bye")
				return
			default:
				write("250 OK")
			}
		}
	}()
	return ln.Addr().String()
}
