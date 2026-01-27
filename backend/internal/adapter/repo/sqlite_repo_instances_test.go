package repo_test

import (
	"context"
	"testing"
	"time"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func TestSQLiteRepo_InstanceUpdatesAndEvents(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()

	user := testutil.CreateUser(t, repo, "instuser", "instuser@example.com", "pass")
	order := domain.Order{UserID: user.ID, OrderNo: "ORD-INST", Status: domain.OrderStatusActive, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	item := domain.OrderItem{OrderID: order.ID, Amount: 1000, Status: domain.OrderItemStatusActive, Action: "create", SpecJSON: "{}"}
	if err := repo.CreateOrderItems(ctx, []domain.OrderItem{item}); err != nil {
		t.Fatalf("create item: %v", err)
	}
	items, _ := repo.ListOrderItems(ctx, order.ID)
	if len(items) == 0 {
		t.Fatalf("missing items")
	}

	expireAt := time.Now().Add(24 * time.Hour)
	inst := domain.VPSInstance{
		UserID:               user.ID,
		OrderItemID:          items[0].ID,
		AutomationInstanceID: "123",
		Name:                 "vm1",
		Status:               domain.VPSStatusRunning,
		AdminStatus:          domain.VPSAdminStatusNormal,
		SpecJSON:             "{}",
		ExpireAt:             &expireAt,
	}
	if err := repo.CreateInstance(ctx, &inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}

	list, err := repo.ListInstancesByUser(ctx, user.ID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list by user: %v %d", err, len(list))
	}
	listAll, total, err := repo.ListInstances(ctx, 10, 0)
	if err != nil || total != 1 || len(listAll) != 1 {
		t.Fatalf("list instances: %v %d %d", err, total, len(listAll))
	}
	expiring, err := repo.ListInstancesExpiring(ctx, time.Now().Add(48*time.Hour))
	if err != nil || len(expiring) != 1 {
		t.Fatalf("list expiring: %v %d", err, len(expiring))
	}

	if err := repo.UpdateInstanceStatus(ctx, inst.ID, domain.VPSStatusStopped, 5); err != nil {
		t.Fatalf("update status: %v", err)
	}
	if err := repo.UpdateInstanceAdminStatus(ctx, inst.ID, domain.VPSAdminStatusLocked); err != nil {
		t.Fatalf("update admin status: %v", err)
	}
	if err := repo.UpdateInstancePanelCache(ctx, inst.ID, "https://panel"); err != nil {
		t.Fatalf("update panel: %v", err)
	}
	if err := repo.UpdateInstanceSpec(ctx, inst.ID, `{"x":1}`); err != nil {
		t.Fatalf("update spec: %v", err)
	}
	if err := repo.UpdateInstanceAccessInfo(ctx, inst.ID, `{"remote_ip":"1.1.1.1"}`); err != nil {
		t.Fatalf("update access: %v", err)
	}
	renewAt := time.Now()
	if err := repo.UpdateInstanceEmergencyRenewAt(ctx, inst.ID, renewAt); err != nil {
		t.Fatalf("update emergency renew: %v", err)
	}

	updated, err := repo.GetInstance(ctx, inst.ID)
	if err != nil {
		t.Fatalf("get instance: %v", err)
	}
	if updated.Status != domain.VPSStatusStopped || updated.AdminStatus != domain.VPSAdminStatusLocked {
		t.Fatalf("unexpected status")
	}

	if _, err := repo.AppendEvent(ctx, order.ID, "order.test", `{"ok":true}`); err != nil {
		t.Fatalf("append event: %v", err)
	}
	events, err := repo.ListEventsAfter(ctx, order.ID, 0, 10)
	if err != nil || len(events) == 0 {
		t.Fatalf("list events: %v %d", err, len(events))
	}
}
