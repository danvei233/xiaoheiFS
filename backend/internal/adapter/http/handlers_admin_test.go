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

func TestHandlers_AdminRoutesSmoke(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)

	group := domain.PermissionGroup{Name: "all", PermissionsJSON: `["*"]`}
	if err := env.Repo.CreatePermissionGroup(context.Background(), &group); err != nil {
		t.Fatalf("create group: %v", err)
	}
	admin := testutil.CreateAdmin(t, env.Repo, "admin", "admin@example.com", "pass", group.ID)
	adminToken := testutil.IssueJWT(t, env.JWTSecret, admin.ID, "admin", time.Hour)

	user := testutil.CreateUser(t, env.Repo, "u1", "u1@example.com", "pass")
	userToken := testutil.IssueJWT(t, env.JWTSecret, user.ID, "user", time.Hour)

	rec := testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/users", nil, userToken)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/auth/login", map[string]any{
		"username": "admin",
		"password": "pass",
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("admin login code: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/users/"+testutil.Itoa(user.ID), map[string]any{
		"username": user.Username,
		"email":    "u1-new@example.com",
		"qq":       "",
	}, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin user update code: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/users/"+testutil.Itoa(user.ID)+"/status", map[string]any{
		"status": "disabled",
	}, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin user status code: %d", rec.Code)
	}

	order := domain.Order{UserID: user.ID, OrderNo: "ORD-DEL", Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "CNY"}
	if err := env.Repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodDelete, "/admin/api/v1/orders/"+testutil.Itoa(order.ID), nil, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin order delete code: %d", rec.Code)
	}

	perm := domain.Permission{Code: "order.view", Name: "Order View", Category: "order"}
	if err := env.Repo.UpsertPermission(context.Background(), &perm); err != nil {
		t.Fatalf("create permission: %v", err)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/permissions/order.view", map[string]any{
		"name":     "Order View",
		"category": "order",
	}, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("permissions update code: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/profile", map[string]any{
		"email": "admin2@example.com",
	}, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("profile update code: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/settings", map[string]any{
		"key":        "site_name",
		"value_json": "Test Site",
	}, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("settings update code: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/payments/providers/fake", map[string]any{
		"enabled":     true,
		"config_json": `{"pay_url":"x"}`,
	}, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("payment provider update code: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/scheduled-tasks/task.vps_refresh", map[string]any{
		"enabled": false,
	}, adminToken)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("scheduled task update code: %d", rec.Code)
	}
}

func TestHandlers_AdminVPSUpdate(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	group := domain.PermissionGroup{Name: "vps", PermissionsJSON: `["vps.*"]`}
	if err := env.Repo.CreatePermissionGroup(context.Background(), &group); err != nil {
		t.Fatalf("create group: %v", err)
	}
	admin := testutil.CreateAdmin(t, env.Repo, "adminvps", "adminvps@example.com", "pass", group.ID)
	adminToken := testutil.IssueJWT(t, env.JWTSecret, admin.ID, "admin", time.Hour)
	user := testutil.CreateUser(t, env.Repo, "u2", "u2@example.com", "pass")

	order := domain.Order{UserID: user.ID, OrderNo: "ORD-VPS", Status: domain.OrderStatusApproved, TotalAmount: 1000, Currency: "CNY"}
	if err := env.Repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	item := domain.OrderItem{OrderID: order.ID, Amount: 1000, Status: domain.OrderItemStatusApproved, Action: "create", SpecJSON: "{}"}
	if err := env.Repo.CreateOrderItems(context.Background(), []domain.OrderItem{item}); err != nil {
		t.Fatalf("create items: %v", err)
	}
	items, _ := env.Repo.ListOrderItems(context.Background(), order.ID)
	vps := domain.VPSInstance{
		UserID:               user.ID,
		OrderItemID:          items[0].ID,
		AutomationInstanceID: "123",
		Name:                 "vm",
		SystemID:             1,
		Status:               domain.VPSStatusUnknown,
		SpecJSON:             "{}",
	}
	if err := env.Repo.CreateInstance(context.Background(), &vps); err != nil {
		t.Fatalf("create vps: %v", err)
	}

	rec := testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/vps/"+testutil.Itoa(vps.ID), map[string]any{
		"status": "running",
	}, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("vps update code: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/vps/"+testutil.Itoa(vps.ID)+"/expire-at", map[string]any{
		"expire_at": time.Now().Add(24 * time.Hour).Format("2006-01-02"),
	}, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("vps expire update code: %d", rec.Code)
	}
}
