package http

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"xiaoheiplay/internal/domain"
)

func TestIsSecretSettingKey(t *testing.T) {
	for _, key := range []string{"smtp_pass", "robot_webhook_secret", "robot_webhooks", "payment_plugin_upload_password", "fcm_server_key", "fcm_service_account_json", "automation_api_key", "auth_geetest_captcha_key"} {
		if !isSecretSettingKey(key) {
			t.Fatalf("expected %q to be a secret key", key)
		}
	}
	if isSecretSettingKey("site_name") {
		t.Fatalf("site_name must not be secret")
	}
	if isSecretSettingKey("smtp_host") {
		t.Fatalf("smtp_host must not be secret")
	}
}

// TestRedactRobotWebhooksValueMasksSecrets verifies the settings-list echo of
// robot_webhooks keeps the per-webhook HMAC secret out of the response while
// preserving the URLs/names/events the admin UI needs.
func TestRedactRobotWebhooksValueMasksSecrets(t *testing.T) {
	raw := `[{"name":"ops","url":"https://hooks.example.com/ops","secret":"whsec-opssupersecret","enabled":true,"events":["order.active"]}]`
	got := redactRobotWebhooksValue(raw)
	if strings.Contains(got, "whsec-opssupersecret") {
		t.Fatalf("robot webhook secret leaked into settings echo: %s", got)
	}
	if !strings.Contains(got, "https://hooks.example.com/ops") {
		t.Fatalf("robot webhook URL dropped from settings echo: %s", got)
	}
	if !strings.Contains(got, redactedSecretValue) {
		t.Fatalf("expected redaction marker, got %s", got)
	}
}

// TestRestoreRobotWebhookSecretsInJSONPreservesRealSecret verifies a save from
// the admin UI (masked secrets) never overwrites a real per-webhook secret with
// the placeholder, and that a genuinely new secret is kept.
func TestRestoreRobotWebhookSecretsInJSONPreservesRealSecret(t *testing.T) {
	existing := `[{"name":"ops","url":"https://hooks.example.com/ops","secret":"whsec-real","enabled":true,"events":["order.active"]}]`
	submitted := `[{"name":"ops","url":"https://hooks.example.com/ops","secret":"***","enabled":true,"events":["order.active"]},{"name":"new","url":"https://hooks.example.com/new","secret":"whsec-fresh","enabled":true,"events":[]}]`
	got, err := restoreRobotWebhookSecretsInJSON(existing, submitted)
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	var webhooks []map[string]any
	if err := json.Unmarshal([]byte(got), &webhooks); err != nil {
		t.Fatalf("restored value is not JSON: %v", err)
	}
	if len(webhooks) != 2 {
		t.Fatalf("expected 2 webhooks, got %d", len(webhooks))
	}
	if strings.Contains(got, "\"***\"") {
		t.Fatalf("masked secret was persisted: %s", got)
	}
	if !strings.Contains(got, "whsec-real") {
		t.Fatalf("real secret not restored: %s", got)
	}
	if !strings.Contains(got, "whsec-fresh") {
		t.Fatalf("new secret not kept: %s", got)
	}
}

func TestIsRedactedSecretValue(t *testing.T) {
	for _, v := range []string{"", "***", "********"} {
		if !isRedactedSecretValue(v) {
			t.Fatalf("expected %q to be treated as a keep-marker", v)
		}
	}
	if isRedactedSecretValue("real-secret") {
		t.Fatalf("real secret must not be treated as a keep-marker")
	}
}

func TestAdminAuditLogDTORedactsSecretSettings(t *testing.T) {
	log := domain.AdminAuditLog{
		Action:     "settings.update",
		TargetType: "setting",
		TargetID:   "smtp_pass",
		DetailJSON: `"super-secret-password"`,
		CreatedAt:  time.Now(),
	}
	dto := toAdminAuditLogDTO(log)
	if strings.Contains(string(dto.Detail), "super-secret-password") {
		t.Fatalf("audit log DTO leaked secret: %s", string(dto.Detail))
	}
	var parsed map[string]any
	if err := json.Unmarshal(dto.Detail, &parsed); err != nil {
		t.Fatalf("detail is not JSON object: %v", err)
	}
	if parsed["value"] != "***" {
		t.Fatalf("expected masked value in detail, got %v", parsed)
	}
}

func TestAdminAuditLogDTONonSecretUntouched(t *testing.T) {
	log := domain.AdminAuditLog{
		Action:     "settings.update",
		TargetType: "setting",
		TargetID:   "site_name",
		DetailJSON: `"My Site"`,
		CreatedAt:  time.Now(),
	}
	dto := toAdminAuditLogDTO(log)
	if !strings.Contains(string(dto.Detail), "My Site") {
		t.Fatalf("non-secret detail was altered: %s", string(dto.Detail))
	}
}
