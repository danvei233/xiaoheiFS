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

func TestHandlers_AdminVPSAndTickets(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	groupID := ensureAdminGroup(t, env)
	admin := testutil.CreateAdmin(t, env.Repo, "adminvps", "adminvps@example.com", "pass", groupID)
	token := testutil.IssueJWT(t, env.JWTSecret, admin.ID, "admin", time.Hour)

	user := domain.User{Username: "vpsuser", Email: "vpsuser@example.com", PasswordHash: "hash", Role: domain.UserRoleUser, Status: domain.UserStatusActive}
	if err := env.Repo.CreateUser(context.Background(), &user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	order := domain.Order{UserID: user.ID, OrderNo: "O-VPS", Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "USD"}
	if err := env.Repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	items := []domain.OrderItem{
		{OrderID: order.ID, SpecJSON: "{}", Qty: 1, Amount: 1000, Status: domain.OrderItemStatusPendingPayment, AutomationInstanceID: "", Action: "create", DurationMonths: 1},
	}
	if err := env.Repo.CreateOrderItems(context.Background(), items); err != nil {
		t.Fatalf("create order items: %v", err)
	}
	orderItemID := items[0].ID
	// A second order item so the second VPS is independent: the corrected
	// CreateInstance semantics treat a different automation host under the SAME
	// order item as a conflict (one VPS per order item), not an idempotent replay.
	order2 := domain.Order{UserID: user.ID, OrderNo: "O-VPS-2", Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "USD"}
	if err := env.Repo.CreateOrder(context.Background(), &order2); err != nil {
		t.Fatalf("create order 2: %v", err)
	}
	items2 := []domain.OrderItem{
		{OrderID: order2.ID, SpecJSON: "{}", Qty: 1, Amount: 1000, Status: domain.OrderItemStatusPendingPayment, AutomationInstanceID: "", Action: "create", DurationMonths: 1},
	}
	if err := env.Repo.CreateOrderItems(context.Background(), items2); err != nil {
		t.Fatalf("create order items 2: %v", err)
	}
	orderItemID2 := items2[0].ID

	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/vps", map[string]any{
		"user_id":                user.ID,
		"order_item_id":          orderItemID,
		"automation_instance_id": "1",
		"name":                   "vm-1",
		"status":                 "running",
		"admin_status":           "normal",
		"expire_at":              time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339),
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin vps create: %d", rec.Code)
	}
	var vpsResp struct {
		ID int64 `json:"id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &vpsResp)
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/vps", map[string]any{
		"user_id":                user.ID,
		"order_item_id":          orderItemID2,
		"automation_instance_id": "2",
		"name":                   "vm-2",
		"status":                 "running",
		"admin_status":           "normal",
		"expire_at":              nil,
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin vps create with null expire_at: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/vps", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin vps list: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/vps/"+testutil.Itoa(vpsResp.ID), nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin vps detail: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/vps/"+testutil.Itoa(vpsResp.ID)+"/lock", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin vps lock: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/vps/"+testutil.Itoa(vpsResp.ID)+"/unlock", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin vps unlock: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/vps/"+testutil.Itoa(vpsResp.ID)+"/status", map[string]any{
		"status": "locked",
		"reason": "ok",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin vps status: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/vps/"+testutil.Itoa(vpsResp.ID)+"/resize", map[string]any{
		"cpu":            2,
		"memory_gb":      4,
		"disk_gb":        40,
		"bandwidth_mbps": 10,
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin vps resize: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/vps/"+testutil.Itoa(vpsResp.ID)+"/emergency-renew", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin vps emergency renew: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/server/status", nil, token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("admin server status: %d", rec.Code)
	}

	ticket := &domain.Ticket{UserID: user.ID, Subject: "help", Status: "open"}
	msg := &domain.TicketMessage{SenderID: user.ID, SenderRole: string(domain.UserRoleUser), SenderName: "vpsuser", Content: "need help"}
	if err := env.Repo.CreateTicketWithDetails(context.Background(), ticket, msg, nil); err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/tickets/"+testutil.Itoa(ticket.ID), map[string]any{
		"subject": "help updated",
		"status":  "closed",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin ticket update: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/tickets/"+testutil.Itoa(ticket.ID)+"/messages", map[string]any{
		"content": "reply",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin ticket message: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodDelete, "/admin/api/v1/tickets/"+testutil.Itoa(ticket.ID), nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin ticket delete: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/vps/"+testutil.Itoa(vpsResp.ID)+"/delete", map[string]any{
		"reason": "cleanup",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin vps delete: %d", rec.Code)
	}
}

func TestAdminVPSDelete_AutoRefundCreatedBeforeDelete(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	groupID := ensureAdminGroup(t, env)
	admin := testutil.CreateAdmin(t, env.Repo, "adminrefund", "adminrefund@example.com", "pass", groupID)
	token := testutil.IssueJWT(t, env.JWTSecret, admin.ID, "admin", time.Hour)

	user := domain.User{Username: "refunduser", Email: "refunduser@example.com", PasswordHash: "hash", Role: domain.UserRoleUser, Status: domain.UserStatusActive}
	if err := env.Repo.CreateUser(context.Background(), &user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	order := domain.Order{UserID: user.ID, OrderNo: "O-REFUND", Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "CNY"}
	if err := env.Repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	items := []domain.OrderItem{{OrderID: order.ID, SpecJSON: "{}", Qty: 1, Amount: 1000, Status: domain.OrderItemStatusPendingPayment, AutomationInstanceID: "1", Action: "create", DurationMonths: 1}}
	if err := env.Repo.CreateOrderItems(context.Background(), items); err != nil {
		t.Fatalf("create order items: %v", err)
	}
	// Enable automatic refund on admin delete.
	if err := env.Repo.UpsertSetting(context.Background(), domain.Setting{Key: "refund_on_admin_delete", ValueJSON: "true"}); err != nil {
		t.Fatalf("enable refund on admin delete: %v", err)
	}

	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/vps", map[string]any{
		"user_id":                user.ID,
		"order_item_id":          items[0].ID,
		"automation_instance_id": "1",
		"name":                   "vm-refund",
		"status":                 "running",
		"admin_status":           "normal",
		"expire_at":              time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339),
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("create vps: %d body=%s", rec.Code, rec.Body.String())
	}
	var vpsResp struct {
		ID int64 `json:"id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &vpsResp)

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/vps/"+testutil.Itoa(vpsResp.ID)+"/delete", map[string]any{
		"reason": "cleanup",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin vps delete: %d body=%s", rec.Code, rec.Body.String())
	}

	walletOrders, _, err := env.Repo.ListWalletOrders(context.Background(), user.ID, 10, 0)
	if err != nil {
		t.Fatalf("list wallet orders: %v", err)
	}
	if len(walletOrders) == 0 {
		t.Fatalf("expected an automatic refund wallet order for user %d after admin delete, got none", user.ID)
	}
}

// TestAdminVPSUpdate_SubAdminCannotChangePriceCredentialsStatus locks P1-20: a
// sub-admin holding only vps.update must not be able to change the monthly
// price, write credentials, lock the instance, or resize it through the generic
// update endpoint — each of those fields requires its own permission and the
// service must answer 403, while a benign field update still succeeds.
func TestAdminVPSUpdate_SubAdminCannotChangePriceCredentialsStatus(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	fullGroupID := ensureAdminGroup(t, env)

	// The first admin created is the primary admin (lowest admin id), so the
	// sub-admin created afterwards is a genuine sub-admin for IsPrimaryAdmin.
	primary := testutil.CreateAdmin(t, env.Repo, "vpsprimary", "vpsprimary@example.com", "pass", fullGroupID)
	primaryToken := testutil.IssueJWT(t, env.JWTSecret, primary.ID, "admin", time.Hour)

	limitedGroup := domain.PermissionGroup{Name: "vps-update-only", PermissionsJSON: `["vps.update"]`}
	if err := env.Repo.CreatePermissionGroup(context.Background(), &limitedGroup); err != nil {
		t.Fatalf("create limited group: %v", err)
	}
	subAdmin := testutil.CreateAdmin(t, env.Repo, "vpslimited", "vpslimited@example.com", "pass", limitedGroup.ID)
	subToken := testutil.IssueJWT(t, env.JWTSecret, subAdmin.ID, "admin", time.Hour)

	user := testutil.CreateUser(t, env.Repo, "vpslimituser", "vpslimituser@example.com", "pass")
	order := domain.Order{UserID: user.ID, OrderNo: "O-LIMIT", Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "CNY"}
	if err := env.Repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	items := []domain.OrderItem{{OrderID: order.ID, SpecJSON: "{}", Qty: 1, Amount: 1000, Status: domain.OrderItemStatusPendingPayment, AutomationInstanceID: "1", Action: "create", DurationMonths: 1}}
	if err := env.Repo.CreateOrderItems(context.Background(), items); err != nil {
		t.Fatalf("create order items: %v", err)
	}

	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/vps", map[string]any{
		"user_id":                user.ID,
		"order_item_id":          items[0].ID,
		"automation_instance_id": "1",
		"name":                   "vm-limit",
		"status":                 "running",
		"admin_status":           "normal",
		"expire_at":              time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339),
	}, primaryToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("create vps: %d body=%s", rec.Code, rec.Body.String())
	}
	var vpsResp struct {
		ID int64 `json:"id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &vpsResp)
	path := "/admin/api/v1/vps/" + testutil.Itoa(vpsResp.ID)

	for name, body := range map[string]map[string]any{
		"monthly_price": {"monthly_price": 9.9},
		"access_info":   {"access_info": map[string]any{"os_password": "leaked"}},
		"admin_status":  {"admin_status": "locked"},
		"cpu":           {"cpu": 8},
		"spec":          {"spec": map[string]any{"cores": 8}},
	} {
		rec = testutil.DoJSON(t, env.Router, http.MethodPatch, path, body, subToken)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("sub-admin %s change must be 403, got %d body=%s", name, rec.Code, rec.Body.String())
		}
	}

	// A benign update (package_name) is still allowed with vps.update.
	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, path, map[string]any{"package_name": "renamed"}, subToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("sub-admin benign update must be 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	// The primary admin can still change the price.
	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, path, map[string]any{"monthly_price": 12.5}, primaryToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("primary admin price change must be 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminVPSRead_SubAdminNeedsCredentialsPermission(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	ctx := context.Background()
	fullGroupID := ensureAdminGroup(t, env)
	primary := testutil.CreateAdmin(t, env.Repo, "vps-read-primary", "vps-read-primary@example.com", "pass", fullGroupID)
	primaryToken := testutil.IssueJWT(t, env.JWTSecret, primary.ID, "admin", time.Hour)

	limitedGroup := domain.PermissionGroup{Name: "vps-read-limited", PermissionsJSON: `["vps.list","vps.view"]`}
	if err := env.Repo.CreatePermissionGroup(ctx, &limitedGroup); err != nil {
		t.Fatalf("create limited group: %v", err)
	}
	limited := testutil.CreateAdmin(t, env.Repo, "vps-read-limited", "vps-read-limited@example.com", "pass", limitedGroup.ID)
	limitedToken := testutil.IssueJWT(t, env.JWTSecret, limited.ID, "admin", time.Hour)

	user := testutil.CreateUser(t, env.Repo, "vps-read-user", "vps-read-user@example.com", "pass")
	inst := domain.VPSInstance{UserID: user.ID, Name: "secret-vps", Status: domain.VPSStatusRunning, AdminStatus: domain.VPSAdminStatusNormal, SpecJSON: `{}`, AccessInfoJSON: `{"os_password":"do-not-list"}`}
	if err := env.Repo.CreateInstance(ctx, &inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}

	for _, request := range []struct {
		name string
		path string
	}{
		{name: "list", path: "/admin/api/v1/vps?limit=20"},
		{name: "detail", path: "/admin/api/v1/vps/" + testutil.Itoa(inst.ID)},
	} {
		t.Run(request.name+" redacted", func(t *testing.T) {
			rec := testutil.DoJSON(t, env.Router, http.MethodGet, request.path, nil, limitedToken)
			if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "do-not-list") || strings.Contains(rec.Body.String(), "access_info") {
				t.Fatalf("limited response leaked credentials: %d %s", rec.Code, rec.Body.String())
			}
		})
	}
	primaryDetail := testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/vps/"+testutil.Itoa(inst.ID), nil, primaryToken)
	if primaryDetail.Code != http.StatusOK || !strings.Contains(primaryDetail.Body.String(), "do-not-list") {
		t.Fatalf("primary detail should reveal credentials: %d %s", primaryDetail.Code, primaryDetail.Body.String())
	}
	primaryList := testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/vps?limit=20", nil, primaryToken)
	if primaryList.Code != http.StatusOK || strings.Contains(primaryList.Body.String(), "do-not-list") {
		t.Fatalf("inventory list must never reveal credentials: %d %s", primaryList.Code, primaryList.Body.String())
	}
}

func TestAdminVPSUpdate_RejectsInvalidStatusEnum(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	groupID := ensureAdminGroup(t, env)
	admin := testutil.CreateAdmin(t, env.Repo, "adminenum", "adminenum@example.com", "pass", groupID)
	token := testutil.IssueJWT(t, env.JWTSecret, admin.ID, "admin", time.Hour)

	user := domain.User{Username: "enumuser", Email: "enumuser@example.com", PasswordHash: "hash", Role: domain.UserRoleUser, Status: domain.UserStatusActive}
	if err := env.Repo.CreateUser(context.Background(), &user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	order := domain.Order{UserID: user.ID, OrderNo: "O-ENUM", Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "CNY"}
	if err := env.Repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	items := []domain.OrderItem{{OrderID: order.ID, SpecJSON: "{}", Qty: 1, Amount: 1000, Status: domain.OrderItemStatusPendingPayment, AutomationInstanceID: "1", Action: "create", DurationMonths: 1}}
	if err := env.Repo.CreateOrderItems(context.Background(), items); err != nil {
		t.Fatalf("create order items: %v", err)
	}

	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/vps", map[string]any{
		"user_id":                user.ID,
		"order_item_id":          items[0].ID,
		"automation_instance_id": "1",
		"name":                   "vm-enum",
		"status":                 "running",
		"admin_status":           "normal",
		"expire_at":              time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339),
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("create vps: %d", rec.Code)
	}
	var vpsResp struct {
		ID int64 `json:"id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &vpsResp)

	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/vps/"+testutil.Itoa(vpsResp.ID), map[string]any{
		"status": "hacked",
	}, token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid status enum, got %d body=%s", rec.Code, rec.Body.String())
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/vps/"+testutil.Itoa(vpsResp.ID), map[string]any{
		"admin_status": "hacked",
	}, token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid admin_status enum, got %d body=%s", rec.Code, rec.Body.String())
	}
}
