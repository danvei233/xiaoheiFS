package http_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/testutilhttp"
)

func issueSettingsAdminToken(t *testing.T, env *testutilhttp.Env) string {
	t.Helper()
	groupID := ensureAdminGroup(t, env)
	admin := testutil.CreateAdmin(t, env.Repo, "admin-settings", "admin-settings@example.com", "pass", groupID)
	return testutil.IssueJWT(t, env.JWTSecret, admin.ID, "admin", time.Hour)
}

func TestHandlers_AdminSettingsUpdate_RejectsInvalidAdminPathSingle(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	token := issueSettingsAdminToken(t, env)

	rec := testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/settings", map[string]any{
		"key":   "admin_path",
		"value": "admin/path",
	}, token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid admin_path, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), domain.ErrAdminPathInvalid.Error()) {
		t.Fatalf("expected error containing %q, got %s", domain.ErrAdminPathInvalid.Error(), rec.Body.String())
	}

	setting, err := env.Repo.GetSetting(context.Background(), "admin_path")
	if err == nil && setting.ValueJSON == "admin/path" {
		t.Fatalf("invalid admin_path should not be persisted")
	}
}

func TestHandlers_AdminSettingsUpdate_RejectsReservedAdminPathInBatch(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	token := issueSettingsAdminToken(t, env)

	if err := env.Repo.UpsertSetting(context.Background(), domain.Setting{Key: "site_name", ValueJSON: "Original Site"}); err != nil {
		t.Fatalf("seed site_name: %v", err)
	}

	rec := testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/settings", map[string]any{
		"items": []map[string]any{
			{"key": "site_name", "value": "Changed Site"},
			{"key": "admin_path", "value": "admin"},
		},
	}, token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for reserved admin_path, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), domain.ErrAdminPathReserved.Error()) {
		t.Fatalf("expected error containing %q, got %s", domain.ErrAdminPathReserved.Error(), rec.Body.String())
	}

	siteName, err := env.Repo.GetSetting(context.Background(), "site_name")
	if err != nil {
		t.Fatalf("read site_name after failed batch: %v", err)
	}
	if siteName.ValueJSON != "Original Site" {
		t.Fatalf("batch should be rejected before updates, site_name=%q", siteName.ValueJSON)
	}
}

func TestHandlers_AdminSettingsUpdate_NormalizesAdminPathBeforeSave(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	token := issueSettingsAdminToken(t, env)

	rec := testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/settings", map[string]any{
		"key":   "admin_path",
		"value": "  AbC123  ",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid admin_path, got %d", rec.Code)
	}

	setting, err := env.Repo.GetSetting(context.Background(), "admin_path")
	if err != nil {
		t.Fatalf("read saved admin_path: %v", err)
	}
	if setting.ValueJSON != "AbC123" {
		t.Fatalf("expected trimmed admin_path %q, got %q", "AbC123", setting.ValueJSON)
	}
}
