package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/testutilhttp"
)

func TestHandlers_AdminSettingsList_MasksSecretValues(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	ctx := context.Background()
	if err := env.Repo.UpsertSetting(ctx, domain.Setting{Key: "smtp_pass", ValueJSON: "real-pass", UpdatedAt: time.Now()}); err != nil {
		t.Fatalf("seed smtp_pass: %v", err)
	}
	group := domain.PermissionGroup{Name: "all", PermissionsJSON: `["*"]`}
	if err := env.Repo.CreatePermissionGroup(ctx, &group); err != nil {
		t.Fatalf("create group: %v", err)
	}
	admin := testutil.CreateAdmin(t, env.Repo, "admin", "admin@example.com", "pass", group.ID)
	adminToken := testutil.IssueJWT(t, env.JWTSecret, admin.ID, "admin", time.Hour)

	rec := testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/settings", nil, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("settings list code: %d", rec.Code)
	}
	var body struct {
		Items []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, item := range body.Items {
		if item.Key == "smtp_pass" {
			if item.Value == "real-pass" || strings.Contains(item.Value, "real-pass") {
				t.Fatalf("smtp_pass leaked in settings list: %q", item.Value)
			}
		}
	}
}

func TestHandlers_AdminSettingsUpdate_MaskedSecretNotRejected(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	ctx := context.Background()
	// Pre-seed an FCM service-account JSON so a masked keep-marker must not
	// overwrite it.
	if err := env.Repo.UpsertSetting(ctx, domain.Setting{Key: "fcm_service_account_json", ValueJSON: `{"type":"service_account"}`, UpdatedAt: time.Now()}); err != nil {
		t.Fatalf("seed fcm json: %v", err)
	}
	group := domain.PermissionGroup{Name: "all", PermissionsJSON: `["*"]`}
	if err := env.Repo.CreatePermissionGroup(ctx, &group); err != nil {
		t.Fatalf("create group: %v", err)
	}
	admin := testutil.CreateAdmin(t, env.Repo, "admin", "admin@example.com", "pass", group.ID)
	adminToken := testutil.IssueJWT(t, env.JWTSecret, admin.ID, "admin", time.Hour)

	// R-7 regression: submitting the masked placeholder for a JSON secret key
	// must be treated as "keep", not rejected by JSON validation.
	rec := testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/settings", map[string]any{
		"items": []map[string]string{
			{"key": "fcm_service_account_json", "value": "***"},
		},
	}, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("masked secret update rejected: %d body=%s", rec.Code, rec.Body.String())
	}
	stored, err := env.Repo.GetSetting(ctx, "fcm_service_account_json")
	if err != nil {
		t.Fatalf("get setting: %v", err)
	}
	if !strings.Contains(stored.ValueJSON, "service_account") {
		t.Fatalf("masked update overwrote stored secret: %q", stored.ValueJSON)
	}
}
