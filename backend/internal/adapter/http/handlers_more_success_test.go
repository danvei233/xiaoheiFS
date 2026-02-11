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

func TestHandlers_UserExtraSuccess(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	seed := testutil.SeedCatalog(t, env.Repo)
	user := testutil.CreateUser(t, env.Repo, "extra", "extra@example.com", "pass")
	token := testutil.IssueJWT(t, env.JWTSecret, user.ID, "user", time.Hour)

	rec := testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/captcha", nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("captcha: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/me", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("me: %d", rec.Code)
	}

	if err := env.Repo.UpsertSetting(context.Background(), domain.Setting{Key: "realname_enabled", ValueJSON: "true"}); err != nil {
		t.Fatalf("enable realname: %v", err)
	}
	if err := env.Repo.UpsertSetting(context.Background(), domain.Setting{Key: "realname_provider", ValueJSON: "fake"}); err != nil {
		t.Fatalf("set provider: %v", err)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/realname/verify", map[string]any{
		"real_name": "Test User",
		"id_number": "11010519491231002X",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("realname verify: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/plan-groups?region_id="+testutil.Itoa(seed.Region.ID), nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("plan groups: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/packages?plan_group_id="+testutil.Itoa(seed.PlanGroup.ID), nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("packages: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/system-images?plan_group_id="+testutil.Itoa(seed.PlanGroup.ID), nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("system images: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/billing-cycles", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("billing cycles: %d", rec.Code)
	}

	order := domain.Order{UserID: user.ID, OrderNo: "ORD-PAY-1", Status: domain.OrderStatusPendingPayment, TotalAmount: 1200, Currency: "CNY"}
	if err := env.Repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	item := domain.OrderItem{OrderID: order.ID, Amount: 1200, Status: domain.OrderItemStatusPendingPayment, Action: "create", SpecJSON: "{}"}
	if err := env.Repo.CreateOrderItems(context.Background(), []domain.OrderItem{item}); err != nil {
		t.Fatalf("create item: %v", err)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/orders/"+testutil.Itoa(order.ID)+"/payments", map[string]any{
		"method":   "manual",
		"amount":   12,
		"currency": "CNY",
		"trade_no": "TN-M1",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("order payment: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/orders", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("orders list: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/orders/"+testutil.Itoa(order.ID), nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("order detail: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/wallet", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("wallet info: %d", rec.Code)
	}
	if _, err := env.Repo.AdjustWalletBalance(context.Background(), user.ID, 20, "credit", "seed", 1, "init"); err != nil {
		t.Fatalf("seed wallet: %v", err)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/wallet/transactions", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("wallet transactions: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/wallet/recharge", map[string]any{
		"amount":   50,
		"currency": "CNY",
		"note":     "topup",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("wallet recharge: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/wallet/orders", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("wallet orders: %d", rec.Code)
	}

	notice := domain.Notification{UserID: user.ID, Type: "info", Title: "hello", Content: "world"}
	if err := env.Repo.CreateNotification(context.Background(), &notice); err != nil {
		t.Fatalf("create notification: %v", err)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/notifications/"+testutil.Itoa(notice.ID)+"/read", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("notification read: %d", rec.Code)
	}

	order2 := domain.Order{UserID: user.ID, OrderNo: "ORD-DASH", Status: domain.OrderStatusPendingReview, TotalAmount: 800, Currency: "CNY"}
	if err := env.Repo.CreateOrder(context.Background(), &order2); err != nil {
		t.Fatalf("create order2: %v", err)
	}
	item2 := domain.OrderItem{OrderID: order2.ID, Amount: 800, Status: domain.OrderItemStatusPendingReview, Action: "create", SpecJSON: "{}"}
	if err := env.Repo.CreateOrderItems(context.Background(), []domain.OrderItem{item2}); err != nil {
		t.Fatalf("create item2: %v", err)
	}
	items2, _ := env.Repo.ListOrderItems(context.Background(), order2.ID)
	expireAt := time.Now().Add(3 * 24 * time.Hour)
	inst := domain.VPSInstance{UserID: user.ID, OrderItemID: items2[0].ID, Name: "vm", Status: domain.VPSStatusRunning, SpecJSON: "{}", ExpireAt: &expireAt}
	if err := env.Repo.CreateInstance(context.Background(), &inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/dashboard", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("dashboard: %d", rec.Code)
	}
	var dash struct {
		Orders int `json:"orders"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &dash)
	if dash.Orders == 0 {
		t.Fatalf("dashboard missing orders")
	}
}

func TestHandlers_SystemImages_ByPlanGroupWithoutLineID_ReturnsEmpty(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	seed := testutil.SeedCatalog(t, env.Repo)
	user := testutil.CreateUser(t, env.Repo, "imgscope", "imgscope@example.com", "pass")
	token := testutil.IssueJWT(t, env.JWTSecret, user.ID, "user", time.Hour)

	plan := domain.PlanGroup{
		RegionID:          seed.Region.ID,
		Name:              "NoLine",
		LineID:            0,
		UnitCore:          1,
		UnitMem:           1,
		UnitDisk:          1,
		UnitBW:            1,
		Active:            true,
		Visible:           true,
		CapacityRemaining: -1,
		SortOrder:         0,
	}
	if err := env.Repo.CreatePlanGroup(context.Background(), &plan); err != nil {
		t.Fatalf("create plan group: %v", err)
	}

	rec := testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/system-images?plan_group_id="+testutil.Itoa(plan.ID), nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("system images: %d", rec.Code)
	}
	var resp struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Items) != 0 {
		t.Fatalf("expected empty items, got %d", len(resp.Items))
	}
}
