package http_test

import (
	"context"
	"net/http"
	"testing"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/testutilhttp"
)

func TestHandlers_AdminSettingsUpdate_RejectsAnalyticsCodePayloads(t *testing.T) {
	cases := []string{
		`<script>alert(1)</script>`,
		`<svg onload=alert(1)>`,
		`<img onerror = alert(1)>`,
		`javascript:alert(1)`,
		`console.log("analytics")`,
	}
	for _, value := range cases {
		t.Run(value, func(t *testing.T) {
			env := testutilhttp.NewTestEnv(t, false)
			token := issueSettingsAdminToken(t, env)
			rec := testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/settings", map[string]any{
				"key": "analytics_code", "value": value,
			}, token)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestHandlers_AdminSettingsUpdate_AllowsClearingAnalyticsCode(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	token := issueSettingsAdminToken(t, env)
	if err := env.Repo.UpsertSetting(context.Background(), domain.Setting{
		Key: "analytics_code", ValueJSON: `<script>legacy()</script>`,
	}); err != nil {
		t.Fatalf("seed analytics code: %v", err)
	}

	rec := testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/settings", map[string]any{
		"key": "analytics_code", "value": "",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	setting, err := env.Repo.GetSetting(context.Background(), "analytics_code")
	if err != nil {
		t.Fatalf("read analytics code: %v", err)
	}
	if setting.ValueJSON != "" {
		t.Fatalf("expected analytics code to be cleared, got %q", setting.ValueJSON)
	}
}
