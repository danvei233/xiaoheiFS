package http_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/testutilhttp"
)

func adminToken(t *testing.T, env *testutilhttp.Env) string {
	t.Helper()
	admin := testutil.CreateAdmin(t, env.Repo, "settingsadmin", "settingsadmin@example.com", "pass", 0)
	return testutil.IssueJWT(t, env.JWTSecret, admin.ID, "admin", time.Hour)
}

func readSetting(t *testing.T, env *testutilhttp.Env, key string) string {
	t.Helper()
	st, err := env.Repo.GetSetting(context.Background(), key)
	if err != nil {
		t.Fatalf("get setting %s: %v", key, err)
	}
	return st.ValueJSON
}

// TestAdminSMSConfigUpdatePersistsAllSettings verifies the multi-key SMS config
// save writes every key and reports success (no swallowed partial write).
func TestAdminSMSConfigUpdatePersistsAllSettings(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	token := adminToken(t, env)

	rec := testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/integrations/sms", map[string]any{
		"enabled":               true,
		"plugin_id":             "duanxinbao",
		"instance_id":           "main",
		"default_template_id":   "t1",
		"provider_template_id":  "p1",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("sms config update: code=%d body=%s", rec.Code, rec.Body.String())
	}

	cases := map[string]string{
		"sms_enabled":               "true",
		"sms_plugin_id":             "duanxinbao",
		"sms_instance_id":           "main",
		"sms_default_template_id":   "t1",
		"sms_provider_template_id":  "p1",
	}
	for key, want := range cases {
		if got := readSetting(t, env, key); got != want {
			t.Fatalf("setting %s = %q, want %q", key, got, want)
		}
	}
}

// TestAdminSMTPConfigUpdateMaskedPassPreservesSecret verifies the masked
// placeholder never overwrites the stored SMTP password.
func TestAdminSMTPConfigUpdateMaskedPassPreservesSecret(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	token := adminToken(t, env)

	if err := env.Repo.UpsertSetting(context.Background(), domain.Setting{Key: "smtp_pass", ValueJSON: "real-smtp-secret"}); err != nil {
		t.Fatalf("seed smtp_pass: %v", err)
	}

	// Frontend submits the keep-marker; the stored secret must survive.
	rec := testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/integrations/smtp", map[string]any{
		"host":    "smtp.example.com",
		"port":    "587",
		"user":    "admin",
		"pass":    "********",
		"from":    "noreply@example.com",
		"enabled": true,
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("smtp config update: code=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := readSetting(t, env, "smtp_pass"); got != "real-smtp-secret" {
		t.Fatalf("smtp_pass was overwritten by masked placeholder: %q", got)
	}
	if got := readSetting(t, env, "smtp_host"); got != "smtp.example.com" {
		t.Fatalf("smtp_host = %q, want smtp.example.com", got)
	}
}

// TestAdminRobotConfigUpdateMaskedSecretPreservesSecret verifies the robot
// webhook secret placeholder never overwrites the stored secret.
func TestAdminRobotConfigUpdateMaskedSecretPreservesSecret(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	token := adminToken(t, env)

	if err := env.Repo.UpsertSetting(context.Background(), domain.Setting{Key: "robot_webhook_secret", ValueJSON: "real-robot-secret"}); err != nil {
		t.Fatalf("seed robot_webhook_secret: %v", err)
	}

	rec := testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/integrations/robot", map[string]any{
		"url":     "https://example.com/hook",
		"secret":  "********",
		"enabled": true,
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("robot config update: code=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := readSetting(t, env, "robot_webhook_secret"); got != "real-robot-secret" {
		t.Fatalf("robot_webhook_secret was overwritten by masked placeholder: %q", got)
	}
	if got := readSetting(t, env, "robot_webhook_url"); got != "https://example.com/hook" {
		t.Fatalf("robot_webhook_url = %q, want https://example.com/hook", got)
	}
}
