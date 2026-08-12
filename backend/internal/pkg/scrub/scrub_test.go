package scrub

import (
	"strings"
	"testing"
)

func TestRedact_RemovesCredentialsAndTopology(t *testing.T) {
	secret := "sk-live-0123456789abcdef0123456789abcdef"
	token := "AUTH_TOKEN_abcdef123456"
	pwd := "S3cr3t-P@ss!"
	// URL userinfo credentials never contain a literal '@' (it must be encoded),
	// so use a plain password for those cases.
	urlPwd := "S3cr3tPass"
	internalIP := "192.168.5.10"
	internalHost := "db.internal.corp"

	cases := []struct {
		name string
		in   string
	}{
		{"query token", "delete failed: POST http://provider.example.com/delete?id=5&token=" + token},
		{"password pair", "DeleteHost: authentication rejected for password=" + pwd},
		{"url userinfo", "dial tcp admin:" + urlPwd + "@" + internalHost + ":443"},
		{"url userinfo no scheme", "dial tcp admin:" + urlPwd + "@192.168.5.10:443"},
		{"authorization header", "request failed: Authorization: Bearer " + token},
		{"x-api-key header", "response 401, x-api-key: " + secret},
		{"long hex secret", "provider said host " + internalIP + " rejected key " + secret},
		{"url with internal host", "call to http://" + internalHost + "/v1/hosts?sign=" + token + " timed out"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := Redact(tc.in)
			for _, leak := range []string{secret, token, pwd} {
				if strings.Contains(out, leak) {
					t.Fatalf("secret leaked in %q -> %q", tc.in, out)
				}
			}
		})
	}
}

func TestRedact_KeepsBenignTextIntact(t *testing.T) {
	msg := "host 2001 timed out while checking status"
	if got := Redact(msg); got != msg {
		t.Fatalf("benign message altered: %q -> %q", msg, got)
	}
}

func TestSanitizeSummary_IsStableAndSecretFree(t *testing.T) {
	expected := map[string]string{
		"invalid_input":        "cleanup request invalid",
		"not_found":            "instance not found",
		"provider_timeout":     "cleanup provider timed out",
		"provider_unreachable": "cleanup provider unreachable",
		"provider_rejected":    "cleanup provider rejected the request",
		"local_update_failed":  "cleanup completed but local state update failed",
		"provider_error":       "cleanup failed",
		"something_else":       "cleanup failed",
	}
	for code, want := range expected {
		summary := SanitizeSummary(code)
		if summary != want {
			t.Fatalf("SanitizeSummary(%q) = %q, want %q", code, summary, want)
		}
	}
}
