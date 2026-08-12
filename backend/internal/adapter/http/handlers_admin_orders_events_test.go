package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/testutilhttp"
)

// web_api_contract F4：管理端订单详情只返回最近 200 条事件，但必须附带
// events_total 让前端提示"仅显示最近 N 条"，避免误以为事件流被截断是丢数据。
func TestHandlers_AdminOrderDetail_ReturnsEventsTotal(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	groupID := ensureAdminGroup(t, env)
	admin := testutil.CreateAdmin(t, env.Repo, "evtadmin", "evtadmin@example.com", "pass", groupID)
	token := testutil.IssueJWT(t, env.JWTSecret, admin.ID, "admin", time.Hour)

	user := domain.User{Username: "evtuser", Email: "evtuser@example.com", PasswordHash: "hash", Role: domain.UserRoleUser, Status: domain.UserStatusActive}
	if err := env.Repo.CreateUser(context.Background(), &user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	order := domain.Order{UserID: user.ID, OrderNo: "O-EVENTS", Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "USD"}
	if err := env.Repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	items := []domain.OrderItem{{OrderID: order.ID, SpecJSON: "{}", Qty: 1, Amount: 1000, Status: domain.OrderItemStatusPendingPayment, AutomationInstanceID: "", Action: "create", DurationMonths: 1}}
	if err := env.Repo.CreateOrderItems(context.Background(), items); err != nil {
		t.Fatalf("create order items: %v", err)
	}

	// 追加 250 条事件（超过 200 条截断线）
	for i := 0; i < 250; i++ {
		if _, err := env.Repo.AppendEvent(context.Background(), order.ID, "provision.log", `{"step":`+strconv.Itoa(i)+`}`); err != nil {
			t.Fatalf("append event %d: %v", i, err)
		}
	}

	rec := testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/orders/"+strconv.FormatInt(order.ID, 10), nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for order detail, got %d", rec.Code)
	}
	var body struct {
		Events      []json.RawMessage `json:"events"`
		EventsTotal int               `json:"events_total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode order detail: %v", err)
	}
	if len(body.Events) > 200 {
		t.Fatalf("events must be truncated to 200, got %d", len(body.Events))
	}
	if body.EventsTotal != 250 {
		t.Fatalf("expected events_total=250, got %d", body.EventsTotal)
	}
}
