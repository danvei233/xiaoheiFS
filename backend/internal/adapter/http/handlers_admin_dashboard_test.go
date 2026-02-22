package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"testing"
	"time"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/testutilhttp"
)

func TestHandlers_AdminRevenueAnalyticsAuthAndValidation(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)

	group := domain.PermissionGroup{Name: "ops", PermissionsJSON: `["dashboard.revenue","dashboard.overview"]`}
	if err := env.Repo.CreatePermissionGroup(context.Background(), &group); err != nil {
		t.Fatalf("create group: %v", err)
	}
	admin := testutil.CreateAdmin(t, env.Repo, "analytics_admin", "analytics_admin@example.com", "pass", group.ID)
	adminToken := testutil.IssueJWT(t, env.JWTSecret, admin.ID, "admin", time.Hour)
	user := testutil.CreateUser(t, env.Repo, "analytics_user", "analytics_user@example.com", "pass")
	userToken := testutil.IssueJWT(t, env.JWTSecret, user.ID, "user", time.Hour)

	payload := map[string]any{
		"from_at":       time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
		"to_at":         time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		"level":         "goods_type",
		"goods_type_id": 1,
	}

	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/dashboard/revenue-analytics/overview", payload, userToken)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-admin, got %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/dashboard/revenue-analytics/overview", payload, adminToken)
	if rec.Code != http.StatusOK && rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 200/400 for admin request, got %d", rec.Code)
	}

	logs, total, err := env.Repo.ListAuditLogs(context.Background(), 50, 0)
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}
	if total == 0 || len(logs) == 0 {
		t.Fatalf("expected audit log for analytics query")
	}
	found := false
	for _, log := range logs {
		if log.Action == "dashboard.revenue_analytics.overview" {
			var detail map[string]any
			if err := json.Unmarshal([]byte(log.DetailJSON), &detail); err != nil {
				t.Fatalf("unmarshal audit detail: %v", err)
			}
			if detail["request_path"] == "" {
				t.Fatalf("expected request_path in audit detail")
			}
			filterSummary, ok := detail["filter_summary"].(map[string]any)
			if !ok || filterSummary["level"] == nil {
				t.Fatalf("expected filter_summary in audit detail, got: %v", detail)
			}
			found = true
			break
		}
	}
	if !found {
		out, _ := json.Marshal(logs)
		t.Fatalf("analytics audit action not found: %s", string(out))
	}

	invalid := map[string]any{"from_at": "bad", "to_at": "bad", "level": "goods_type"}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/dashboard/revenue-analytics/details", invalid, adminToken)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid payload, got %d", rec.Code)
	}
}

func TestHandlers_AdminRevenueAnalyticsLatencyBudget(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	group := domain.PermissionGroup{Name: "perf", PermissionsJSON: `["dashboard.revenue","dashboard.overview"]`}
	if err := env.Repo.CreatePermissionGroup(context.Background(), &group); err != nil {
		t.Fatalf("create group: %v", err)
	}
	admin := testutil.CreateAdmin(t, env.Repo, "analytics_perf", "analytics_perf@example.com", "pass", group.ID)
	adminToken := testutil.IssueJWT(t, env.JWTSecret, admin.ID, "admin", time.Hour)

	payload := map[string]any{
		"from_at":       time.Now().Add(-7 * 24 * time.Hour).Format(time.RFC3339),
		"to_at":         time.Now().Format(time.RFC3339),
		"level":         "goods_type",
		"goods_type_id": 1,
		"page":          1,
		"page_size":     20,
	}

	type target struct {
		path       string
		threshold  time.Duration
		sampleSize int
	}
	targets := []target{
		{path: "/admin/api/v1/dashboard/revenue-analytics/overview", threshold: 3 * time.Second, sampleSize: 20},
		{path: "/admin/api/v1/dashboard/revenue-analytics/trend", threshold: 3 * time.Second, sampleSize: 20},
		{path: "/admin/api/v1/dashboard/revenue-analytics/top", threshold: 3 * time.Second, sampleSize: 20},
		{path: "/admin/api/v1/dashboard/revenue-analytics/details", threshold: 1 * time.Second, sampleSize: 20},
	}

	for _, tc := range targets {
		latencies := make([]time.Duration, 0, tc.sampleSize)
		for i := 0; i < tc.sampleSize; i++ {
			begin := time.Now()
			rec := testutil.DoJSON(t, env.Router, http.MethodPost, tc.path, payload, adminToken)
			latencies = append(latencies, time.Since(begin))
			if rec.Code != http.StatusOK && rec.Code != http.StatusBadRequest {
				t.Fatalf("unexpected status %d for %s", rec.Code, tc.path)
			}
		}
		p95 := percentile95(latencies)
		t.Logf("latency p95 %s = %s", tc.path, p95)
		if p95 > tc.threshold {
			t.Fatalf("p95 latency exceeded for %s: %s > %s", tc.path, p95, tc.threshold)
		}
	}
}

func percentile95(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}
	values := append([]time.Duration(nil), latencies...)
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	idx := int(float64(len(values)-1) * 0.95)
	return values[idx]
}
