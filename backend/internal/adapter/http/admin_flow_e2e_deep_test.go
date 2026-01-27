package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/testutilhttp"
)

func TestAdminFlow_E2E_Deep(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	groupID := ensureAdminGroup(t, env)
	admin := testutil.CreateAdmin(t, env.Repo, "admin-deep", "admin-deep@example.com", "pass", groupID)

	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/auth/login", map[string]any{
		"username": admin.Username,
		"password": "pass",
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("admin login: %d", rec.Code)
	}
	var loginResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	adminToken := loginResp.AccessToken

	user := testutil.CreateUser(t, env.Repo, "u-deep", "u-deep@example.com", "pass")
	order := domain.Order{UserID: user.ID, OrderNo: "ORD-DEEP-1", Status: domain.OrderStatusPendingPayment, TotalAmount: 1200, Currency: "CNY"}
	if err := env.Repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	item := domain.OrderItem{OrderID: order.ID, Amount: 1200, Status: domain.OrderItemStatusPendingPayment, Action: "create", SpecJSON: "{}"}
	if err := env.Repo.CreateOrderItems(context.Background(), []domain.OrderItem{item}); err != nil {
		t.Fatalf("create item: %v", err)
	}
	payment := domain.OrderPayment{OrderID: order.ID, UserID: user.ID, Method: "manual", Amount: 1200, Currency: "CNY", TradeNo: "TN-DEEP", Status: domain.PaymentStatusPendingReview}
	if err := env.Repo.CreatePayment(context.Background(), &payment); err != nil {
		t.Fatalf("create payment: %v", err)
	}
	if _, err := env.Repo.AppendEvent(context.Background(), order.ID, "order.test", `{"ok":true}`); err != nil {
		t.Fatalf("append event: %v", err)
	}
	if err := env.Repo.AddAuditLog(context.Background(), domain.AdminAuditLog{AdminID: admin.ID, Action: "order.list", TargetType: "order", TargetID: "1", DetailJSON: "{}"}); err != nil {
		t.Fatalf("audit log: %v", err)
	}
	if err := env.Repo.UpsertSetting(context.Background(), domain.Setting{Key: "site_name", ValueJSON: "Deep"}); err != nil {
		t.Fatalf("setting: %v", err)
	}
	if err := env.Repo.CreateAPIKey(context.Background(), &domain.APIKey{Name: "key1", KeyHash: "hash1", Status: domain.APIKeyStatusActive, ScopesJSON: `[]`}); err != nil {
		t.Fatalf("api key: %v", err)
	}
	env.PaymentReg.RegisterProvider(&testutil.FakePaymentProvider{KeyVal: "fake", NameVal: "Fake"}, true, "")

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/users", nil, adminToken)
	assertListResponse(t, rec, "users")

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/orders", nil, adminToken)
	assertListResponse(t, rec, "orders")

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/orders/"+testutil.Itoa(order.ID), nil, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("order detail: %d", rec.Code)
	}
	var detail struct {
		Order struct {
			ID int64 `json:"id"`
		} `json:"order"`
		Payments []struct {
			ID int64 `json:"id"`
		} `json:"payments"`
		Events []struct {
			ID int64 `json:"id"`
		} `json:"events"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode order detail: %v", err)
	}
	if detail.Order.ID != order.ID || len(detail.Payments) == 0 || len(detail.Events) == 0 {
		t.Fatalf("order detail missing data")
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/audit-logs", nil, adminToken)
	assertListResponse(t, rec, "audit logs")

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/settings", nil, adminToken)
	assertListResponse(t, rec, "settings")

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/api-keys", nil, adminToken)
	assertListResponse(t, rec, "api keys")

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/payments/providers", nil, adminToken)
	assertListResponse(t, rec, "payment providers")
}

func assertListResponse(t *testing.T, rec *httptest.ResponseRecorder, name string) {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("%s list: %d", name, rec.Code)
	}
	var payload struct {
		Items []any `json:"items"`
		Total int   `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode %s: %v", name, err)
	}
	if len(payload.Items) == 0 {
		t.Fatalf("%s empty", name)
	}
}
