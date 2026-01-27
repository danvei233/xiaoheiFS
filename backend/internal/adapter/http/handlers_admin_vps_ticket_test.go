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
