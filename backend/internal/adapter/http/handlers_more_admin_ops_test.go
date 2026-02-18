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

func TestHandlers_AdminOpsMore(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	seed := testutil.SeedCatalog(t, env.Repo)
	groupID := ensureAdminGroup(t, env)
	admin := testutil.CreateAdmin(t, env.Repo, "adminops", "adminops@example.com", "pass", groupID)
	adminToken := testutil.IssueJWT(t, env.JWTSecret, admin.ID, "admin", time.Hour)

	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/users", map[string]any{
		"username": "newuser",
		"email":    "newuser@example.com",
		"password": "pass",
		"role":     "user",
		"status":   "active",
	}, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin user create: %d", rec.Code)
	}
	var created struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil || created.ID == 0 {
		t.Fatalf("decode created user")
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/users/"+testutil.Itoa(created.ID), nil, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin user detail: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/users/"+testutil.Itoa(created.ID)+"/reset-password", map[string]any{
		"password": "newpass",
	}, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin reset password: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/wallets/"+testutil.Itoa(created.ID)+"/adjust", map[string]any{
		"amount": 25,
		"note":   "grant",
	}, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin wallet adjust: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/wallets/"+testutil.Itoa(created.ID)+"/transactions", nil, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin wallet tx: %d", rec.Code)
	}

	userToken := testutil.IssueJWT(t, env.JWTSecret, created.ID, "user", time.Hour)
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/wallet/recharge", map[string]any{
		"amount":   10,
		"currency": "CNY",
		"note":     "topup",
	}, userToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("user recharge: %d", rec.Code)
	}
	var walletResp struct {
		Order struct {
			ID int64 `json:"id"`
		} `json:"order"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &walletResp)
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/wallet/orders?user_id="+testutil.Itoa(created.ID), nil, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin wallet orders: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/wallet/orders/"+testutil.Itoa(walletResp.Order.ID)+"/reject", map[string]any{
		"reason": "invalid",
	}, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin wallet order reject: %d", rec.Code)
	}

	order := domain.Order{UserID: created.ID, OrderNo: "ORD-ADM", Status: domain.OrderStatusPendingPayment, TotalAmount: 900, Currency: "CNY"}
	if err := env.Repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	item := domain.OrderItem{OrderID: order.ID, Amount: 900, Status: domain.OrderItemStatusPendingPayment, Action: "create", SpecJSON: "{}"}
	if err := env.Repo.CreateOrderItems(context.Background(), []domain.OrderItem{item}); err != nil {
		t.Fatalf("create item: %v", err)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/orders/"+testutil.Itoa(order.ID)+"/mark-paid", map[string]any{
		"method":   "manual",
		"amount":   900,
		"currency": "CNY",
		"trade_no": "TN-ADM",
	}, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin mark paid: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/orders/"+testutil.Itoa(order.ID)+"/reject", map[string]any{
		"reason": "invalid",
	}, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin reject: %d", rec.Code)
	}

	approveOrder := domain.Order{UserID: created.ID, OrderNo: "ORD-APP", Status: domain.OrderStatusPendingReview, TotalAmount: 1100, Currency: "CNY"}
	if err := env.Repo.CreateOrder(context.Background(), &approveOrder); err != nil {
		t.Fatalf("create approve order: %v", err)
	}
	appItem := domain.OrderItem{OrderID: approveOrder.ID, PackageID: seed.Package.ID, SystemID: seed.SystemImage.ID, Amount: 1100, Status: domain.OrderItemStatusPendingReview, Action: "create", SpecJSON: "{}"}
	if err := env.Repo.CreateOrderItems(context.Background(), []domain.OrderItem{appItem}); err != nil {
		t.Fatalf("create approve item: %v", err)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/orders/"+testutil.Itoa(approveOrder.ID)+"/approve", nil, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin approve: %d", rec.Code)
	}

	retryOrder := domain.Order{UserID: created.ID, OrderNo: "ORD-RET", Status: domain.OrderStatusFailed, TotalAmount: 500, Currency: "CNY"}
	if err := env.Repo.CreateOrder(context.Background(), &retryOrder); err != nil {
		t.Fatalf("create retry order: %v", err)
	}
	retItem := domain.OrderItem{OrderID: retryOrder.ID, PackageID: seed.Package.ID, SystemID: seed.SystemImage.ID, Amount: 500, Status: domain.OrderItemStatusFailed, Action: "create", SpecJSON: "{}"}
	if err := env.Repo.CreateOrderItems(context.Background(), []domain.OrderItem{retItem}); err != nil {
		t.Fatalf("create retry item: %v", err)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/orders/"+testutil.Itoa(retryOrder.ID)+"/retry", nil, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin retry: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/scheduled-tasks", nil, adminToken)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("admin scheduled tasks: %d", rec.Code)
	}

	ticket := domain.Ticket{UserID: created.ID, Subject: "Help", Status: "open"}
	msg := domain.TicketMessage{SenderID: created.ID, SenderRole: "user", Content: "help"}
	if err := env.Repo.CreateTicketWithDetails(context.Background(), &ticket, &msg, nil); err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/tickets", nil, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin tickets: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/tickets/"+testutil.Itoa(ticket.ID), nil, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin ticket detail: %d", rec.Code)
	}
}
