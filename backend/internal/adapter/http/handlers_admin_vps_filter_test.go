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

// web_refresh_perf F4：管理端 VPS 列表支持服务端 admin_status / keyword 过滤。
// 此前前端带过滤时把全表拉下来客户端过滤，大部署下每次搜索/状态切换 =
// ⌈T/500⌉ 个串行请求。
func TestHandlers_AdminVPSList_ServerSideFilter(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	groupID := ensureAdminGroup(t, env)
	admin := testutil.CreateAdmin(t, env.Repo, "vpsfilter", "vpsfilter@example.com", "pass", groupID)
	token := testutil.IssueJWT(t, env.JWTSecret, admin.ID, "admin", time.Hour)

	user := domain.User{Username: "vpsfuser", Email: "vpsfuser@example.com", PasswordHash: "hash", Role: domain.UserRoleUser, Status: domain.UserStatusActive}
	if err := env.Repo.CreateUser(context.Background(), &user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	// CreateInstance 以 order_item_id 作 DoNothing 冲突键——每个 VPS 必须挂不同的
	// order_item，否则后建的同 order_item VPS 不会真正落库。
	createOrderItem := func(orderNo string) int64 {
		order := domain.Order{UserID: user.ID, OrderNo: orderNo, Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "USD"}
		if err := env.Repo.CreateOrder(context.Background(), &order); err != nil {
			t.Fatalf("create order %s: %v", orderNo, err)
		}
		items := []domain.OrderItem{{OrderID: order.ID, SpecJSON: "{}", Qty: 1, Amount: 1000, Status: domain.OrderItemStatusPendingPayment, AutomationInstanceID: "", Action: "create", DurationMonths: 1}}
		if err := env.Repo.CreateOrderItems(context.Background(), items); err != nil {
			t.Fatalf("create order items %s: %v", orderNo, err)
		}
		return items[0].ID
	}

	createVps := func(name, adminStatus string) {
		orderItemID := createOrderItem("O-" + name)
		rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/vps", map[string]any{
			"user_id":                user.ID,
			"order_item_id":          orderItemID,
			"automation_instance_id": name,
			"name":                   name,
			"status":                 "running",
			"admin_status":           adminStatus,
			"expire_at":              time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339),
		}, token)
		if rec.Code != http.StatusOK {
			t.Fatalf("create vps %s: %d", name, rec.Code)
		}
	}
	createVps("vm-normal-1", "normal")
	createVps("vm-normal-2", "normal")
	createVps("vm-abuse-1", "abuse")
	createVps("vm-fraud-1", "fraud")

	// status=abuse 只返回 abuse 实例
	rec := testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/vps?status=abuse&limit=50", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("list vps: %d", rec.Code)
	}
	var list struct {
		Items []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"items"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode vps list: %v", err)
	}
	if list.Total != 1 || len(list.Items) != 1 {
		t.Fatalf("status=abuse expected 1 item, got total=%d items=%d", list.Total, len(list.Items))
	}
	if list.Items[0].Name != "vm-abuse-1" {
		t.Fatalf("expected vm-abuse-1, got %s", list.Items[0].Name)
	}

	// keyword=vm-normal 返回 2 条 normal 实例
	rec2 := testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/vps?keyword=vm-normal&limit=50", nil, token)
	var list2 struct {
		Items []struct {
			Name string `json:"name"`
		} `json:"items"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(rec2.Body.Bytes(), &list2); err != nil {
		t.Fatalf("decode vps list: %v", err)
	}
	if list2.Total != 2 {
		t.Fatalf("keyword=vm-normal expected 2 items, got total=%d", list2.Total)
	}

	// 无过滤返回全部 4 条
	rec3 := testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/vps?limit=50", nil, token)
	var list3 struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal(rec3.Body.Bytes(), &list3); err != nil {
		t.Fatalf("decode vps list: %v", err)
	}
	if list3.Total != 4 {
		t.Fatalf("no filter expected 4 items, got total=%d", list3.Total)
	}
}
