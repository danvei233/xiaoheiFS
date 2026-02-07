package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/testutilhttp"
)

func TestHandlers_AdminAPIKeysAndDebug(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	groupID := ensureAdminGroup(t, env)
	admin := testutil.CreateAdmin(t, env.Repo, "admindebug", "admindebug@example.com", "pass", groupID)
	token := testutil.IssueJWT(t, env.JWTSecret, admin.ID, "admin", time.Hour)

	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/api-keys", map[string]any{
		"name":   "cli",
		"scopes": []string{"*"},
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("api key create: %d", rec.Code)
	}
	var apiKeyResp struct {
		Record struct {
			ID int64 `json:"id"`
		} `json:"record"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &apiKeyResp)
	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/api-keys/"+testutil.Itoa(apiKeyResp.Record.ID), map[string]any{
		"status": "disabled",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("api key update: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/api-keys", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("api keys list: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/integrations/automation", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("automation config: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/integrations/automation", map[string]any{
		"base_url":    "https://auto.local",
		"api_key":     "k",
		"enabled":     true,
		"timeout_sec": 10,
		"retry":       1,
		"dry_run":     true,
	}, token)
	if rec.Code != http.StatusGone {
		t.Fatalf("automation config update: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/integrations/automation/sync?mode=merge", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("automation sync: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/integrations/automation/sync-logs", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("automation sync logs: %d", rec.Code)
	}

	if err := env.Repo.AddAuditLog(context.Background(), domain.AdminAuditLog{AdminID: admin.ID, Action: "test", TargetType: "order", TargetID: "1", DetailJSON: "{}"}); err != nil {
		t.Fatalf("add audit log: %v", err)
	}
	if err := env.Repo.CreateAutomationLog(context.Background(), &domain.AutomationLog{OrderID: 1, OrderItemID: 1, Action: "create", RequestJSON: "{}", ResponseJSON: "{}", Success: true, Message: "ok"}); err != nil {
		t.Fatalf("add automation log: %v", err)
	}
	if err := env.Repo.CreateSyncLog(context.Background(), &domain.IntegrationSyncLog{Target: "area", Mode: "manual", Status: "ok", Message: "done"}); err != nil {
		t.Fatalf("add sync log: %v", err)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/debug/status", map[string]any{
		"enabled": true,
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("debug status update: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/debug/status", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("debug status: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/debug/logs?types=audit,automation,sync", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("debug logs: %d", rec.Code)
	}

	order := domain.Order{UserID: admin.ID, OrderNo: "O-DBG", Status: domain.OrderStatusPendingPayment, TotalAmount: 100, Currency: "USD"}
	if err := env.Repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	items := []domain.OrderItem{{OrderID: order.ID, SpecJSON: "{}", Qty: 1, Amount: 100, Status: domain.OrderItemStatusPendingPayment, Action: "create", DurationMonths: 1}}
	if err := env.Repo.CreateOrderItems(context.Background(), items); err != nil {
		t.Fatalf("create order items: %v", err)
	}
	orderItemID := items[0].ID

	inst := &domain.VPSInstance{
		UserID:               admin.ID,
		OrderItemID:          orderItemID,
		AutomationInstanceID: "1",
		Name:                 "vm",
		Status:               domain.VPSStatusRunning,
	}
	if err := env.Repo.CreateInstance(context.Background(), inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/vps/"+testutil.Itoa(inst.ID)+"/refresh", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin vps refresh: %d", rec.Code)
	}
}
