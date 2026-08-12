package repo_test

import (
	"context"
	"testing"

	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

// 跨层搜索契约（m059 F6）：前端发送的 keyword/action/user（审计）、order_no
// （订单）、status（实名）此前被后端静默忽略，搜索是纯 no-op。这里验证
// repo 层过滤真实生效。
func TestSQLiteRepo_AuditLogFilters(t *testing.T) {
	_, r := newTestRepo(t)
	ctx := context.Background()

	add := func(adminID int64, action, targetType, targetID, detail string) {
		t.Helper()
		if err := r.AddAuditLog(ctx, domain.AdminAuditLog{
			AdminID:    adminID,
			Action:     action,
			TargetType: targetType,
			TargetID:   targetID,
			DetailJSON: detail,
		}); err != nil {
			t.Fatalf("add audit log: %v", err)
		}
	}
	add(1, "user.create", "user", "101", `{"name":"a"}`)
	add(2, "order.approve", "order", "202", `{"amount":100}`)
	add(3, "user.delete", "user", "303", `{"name":"c"}`)

	cases := []struct {
		name   string
		filter appshared.AuditLogFilter
		want   int
	}{
		{"action 子串", appshared.AuditLogFilter{Action: "user"}, 2},
		{"action 精确", appshared.AuditLogFilter{Action: "order.approve"}, 1},
		{"keyword 命中 target_id", appshared.AuditLogFilter{Keyword: "202"}, 1},
		{"keyword 命中 target_type", appshared.AuditLogFilter{Keyword: "order"}, 1},
		{"keyword 命中 detail", appshared.AuditLogFilter{Keyword: "amount"}, 1},
		{"user 数字 admin_id", appshared.AuditLogFilter{User: "3"}, 1},
		{"user 非数字无匹配", appshared.AuditLogFilter{User: "alice"}, 0},
		{"空过滤返回全部", appshared.AuditLogFilter{}, 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			items, total, err := r.ListAuditLogs(ctx, tc.filter, 20, 0)
			if err != nil {
				t.Fatalf("list audit logs: %v", err)
			}
			if total != tc.want {
				t.Fatalf("filter=%+v total=%d want=%d (items=%d)", tc.filter, total, tc.want, len(items))
			}
		})
	}
}

func TestSQLiteRepo_OrderNoFilter(t *testing.T) {
	_, r := newTestRepo(t)
	ctx := context.Background()

	for _, no := range []string{"20260801001", "20260801002", "20260802001"} {
		order := &domain.Order{UserID: 1, OrderNo: no, Status: domain.OrderStatusPendingPayment, TotalAmount: 100, Currency: "CNY"}
		if err := r.CreateOrder(ctx, order); err != nil {
			t.Fatalf("create order %s: %v", no, err)
		}
	}

	// order_no 子串匹配
	items, total, err := r.ListOrders(ctx, appshared.OrderFilter{OrderNo: "20260801"}, 20, 0)
	if err != nil {
		t.Fatalf("list orders: %v", err)
	}
	if total != 2 {
		t.Fatalf("order_no 子串过滤 total=%d want=2", total)
	}
	for _, it := range items {
		if len(it.OrderNo) < 8 || it.OrderNo[:8] != "20260801" {
			t.Fatalf("unexpected order returned: %s", it.OrderNo)
		}
	}

	// 精确 order_no
	items, total, err = r.ListOrders(ctx, appshared.OrderFilter{OrderNo: "20260801002"}, 20, 0)
	if err != nil {
		t.Fatalf("list orders: %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].OrderNo != "20260801002" {
		t.Fatalf("order_no 精确过滤 total=%d want=1", total)
	}
}

func TestSQLiteRepo_RealNameStatusFilter(t *testing.T) {
	_, r := newTestRepo(t)
	ctx := context.Background()

	for i, status := range []string{"pending", "verified", "failed"} {
		record := &domain.RealNameVerification{
			UserID:   int64(i + 1),
			RealName: "user" + string(rune('a'+i)),
			Status:   status,
			Provider: "manual",
		}
		if err := r.CreateRealNameVerification(ctx, record); err != nil {
			t.Fatalf("create realname %s: %v", status, err)
		}
	}

	_, total, err := r.ListRealNameVerifications(ctx, nil, "pending", 20, 0)
	if err != nil {
		t.Fatalf("list realname: %v", err)
	}
	if total != 1 {
		t.Fatalf("status=pending total=%d want=1", total)
	}

	_, total, err = r.ListRealNameVerifications(ctx, nil, "verified", 20, 0)
	if err != nil {
		t.Fatalf("list realname: %v", err)
	}
	if total != 1 {
		t.Fatalf("status=verified total=%d want=1", total)
	}

	// user_id + status 组合
	uid := int64(2)
	items, total, err := r.ListRealNameVerifications(ctx, &uid, "verified", 20, 0)
	if err != nil {
		t.Fatalf("list realname: %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].UserID != 2 {
		t.Fatalf("组合过滤 total=%d want=1", total)
	}
}
