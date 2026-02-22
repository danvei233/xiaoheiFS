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

func TestAdminSecurityGuard_RevenueAnalyticsDeniedWithoutPermission(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	// Ensure the tested admin is not treated as primary admin.
	rootGroup := domain.PermissionGroup{Name: "root", PermissionsJSON: `["dashboard.revenue","dashboard.overview"]`}
	if err := env.Repo.CreatePermissionGroup(context.Background(), &rootGroup); err != nil {
		t.Fatalf("create root group: %v", err)
	}
	_ = testutil.CreateAdmin(t, env.Repo, "root_admin", "root_admin@example.com", "pass", rootGroup.ID)

	group := domain.PermissionGroup{Name: "limited", PermissionsJSON: `["dashboard.overview"]`}
	if err := env.Repo.CreatePermissionGroup(context.Background(), &group); err != nil {
		t.Fatalf("create group: %v", err)
	}
	admin := testutil.CreateAdmin(t, env.Repo, "limited_admin", "limited_admin@example.com", "pass", group.ID)
	token := testutil.IssueJWT(t, env.JWTSecret, admin.ID, "admin", time.Hour)

	payload := map[string]any{
		"from_at":       time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
		"to_at":         time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		"level":         "goods_type",
		"goods_type_id": 1,
	}
	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/dashboard/revenue-analytics/overview", payload, token)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for missing dashboard.revenue permission, got %d", rec.Code)
	}
}
