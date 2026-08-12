package repo_test

import (
	"context"
	"fmt"
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

func TestSQLiteRepo_CreateInstanceIsIdempotentByOrderItem(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	user := testutil.CreateUser(t, repo, "inst-idempotent", "inst-idempotent@example.com", "pass")
	order := domain.Order{UserID: user.ID, OrderNo: "ORD-INST-IDEMPOTENT", Status: domain.OrderStatusApproved, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	if err := repo.CreateOrderItems(ctx, []domain.OrderItem{{OrderID: order.ID, Amount: 1000, Status: domain.OrderItemStatusApproved, Action: "create", SpecJSON: "{}"}}); err != nil {
		t.Fatalf("create item: %v", err)
	}
	items, err := repo.ListOrderItems(ctx, order.ID)
	if err != nil || len(items) != 1 {
		t.Fatalf("list items: len=%d err=%v", len(items), err)
	}

	first := domain.VPSInstance{UserID: user.ID, OrderItemID: items[0].ID, AutomationInstanceID: "host-1", Name: "vm-1", Status: domain.VPSStatusRunning, AdminStatus: domain.VPSAdminStatusNormal, SpecJSON: "{}"}
	if err := repo.CreateInstance(ctx, &first); err != nil {
		t.Fatalf("create first instance: %v", err)
	}

	// Replaying the SAME logical host (same automation id) is idempotent.
	replay := domain.VPSInstance{UserID: user.ID, OrderItemID: items[0].ID, AutomationInstanceID: "host-1", Name: "vm-1", Status: domain.VPSStatusRunning, AdminStatus: domain.VPSAdminStatusNormal, SpecJSON: "{}"}
	if err := repo.CreateInstance(ctx, &replay); err != nil {
		t.Fatalf("replay same host: %v", err)
	}
	if replay.ID != first.ID || replay.AutomationInstanceID != first.AutomationInstanceID {
		t.Fatalf("idempotent replay did not return existing instance: first=%+v replay=%+v", first, replay)
	}

	// A DIFFERENT automation host for the same order item is a genuine conflict,
	// not an idempotent success: a fresh host under an already-linked order item
	// would silently orphan the real VPS.
	different := domain.VPSInstance{UserID: user.ID, OrderItemID: items[0].ID, AutomationInstanceID: "host-2", Name: "vm-2", Status: domain.VPSStatusRunning, AdminStatus: domain.VPSAdminStatusNormal, SpecJSON: "{}"}
	if err := repo.CreateInstance(ctx, &different); err != domain.ErrConflict {
		t.Fatalf("expected ErrConflict for a different automation host under the same order item, got %v", err)
	}

	instances, total, err := repo.ListInstances(ctx, 10, 0)
	if err != nil || total != 1 || len(instances) != 1 {
		t.Fatalf("expected one instance: total=%d len=%d err=%v", total, len(instances), err)
	}

	adminFirst := domain.VPSInstance{UserID: user.ID, AutomationInstanceID: "manual-1", Name: "manual-1", Status: domain.VPSStatusRunning, AdminStatus: domain.VPSAdminStatusNormal, SpecJSON: "{}"}
	adminSecond := domain.VPSInstance{UserID: user.ID, AutomationInstanceID: "manual-2", Name: "manual-2", Status: domain.VPSStatusRunning, AdminStatus: domain.VPSAdminStatusNormal, SpecJSON: "{}"}
	if err := repo.CreateInstance(ctx, &adminFirst); err != nil {
		t.Fatalf("create first manual instance: %v", err)
	}
	if err := repo.CreateInstance(ctx, &adminSecond); err != nil {
		t.Fatalf("create second manual instance: %v", err)
	}
}

func TestSQLiteRepo_DeleteInstanceCascadesResizeAndProvisionJobs(t *testing.T) {
	_, r := newTestRepo(t)
	ctx := context.Background()

	user := domain.User{Username: "delinst", Email: "delinst@example.com", PasswordHash: "hash", Role: domain.UserRoleUser, Status: domain.UserStatusActive}
	if err := r.CreateUser(ctx, &user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	order := domain.Order{UserID: user.ID, OrderNo: "ORD-DEL-INST", Status: domain.OrderStatusApproved, TotalAmount: 1000, Currency: "CNY"}
	if err := r.CreateOrder(ctx, &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	item := domain.OrderItem{OrderID: order.ID, Amount: 1000, Status: domain.OrderItemStatusApproved, Action: "create", SpecJSON: "{}"}
	if err := r.CreateOrderItems(ctx, []domain.OrderItem{item}); err != nil {
		t.Fatalf("create item: %v", err)
	}
	items, err := r.ListOrderItems(ctx, order.ID)
	if err != nil || len(items) != 1 {
		t.Fatalf("list items: len=%d err=%v", len(items), err)
	}

	inst := domain.VPSInstance{UserID: user.ID, OrderItemID: items[0].ID, AutomationInstanceID: "d-1", Name: "vm-del", Status: domain.VPSStatusRunning, AdminStatus: domain.VPSAdminStatusNormal, SpecJSON: "{}"}
	if err := r.CreateInstance(ctx, &inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}

	resize := &domain.ResizeTask{VPSID: inst.ID, OrderID: order.ID, OrderItemID: items[0].ID, Status: domain.ResizeTaskStatusPending}
	if err := r.CreateResizeTask(ctx, resize); err != nil {
		t.Fatalf("create resize task: %v", err)
	}
	job := &domain.ProvisionJob{OrderID: order.ID, OrderItemID: items[0].ID, HostID: 1, HostName: "vm-del", Status: "pending", Attempts: 0, NextRunAt: time.Now().Add(-time.Minute), LastError: ""}
	if err := r.CreateOrUpdateProvisionJob(ctx, job); err != nil {
		t.Fatalf("create provision job: %v", err)
	}

	if pending, err := r.HasPendingResizeTask(ctx, inst.ID); err != nil || !pending {
		t.Fatalf("expected pending resize task before delete: pending=%v err=%v", pending, err)
	}

	if err := r.DeleteInstance(ctx, inst.ID); err != nil {
		t.Fatalf("delete instance: %v", err)
	}

	if pending, err := r.HasPendingResizeTask(ctx, inst.ID); err != nil || pending {
		t.Fatalf("expected no pending resize task after delete: pending=%v err=%v", pending, err)
	}
	if _, err := r.GetInstance(ctx, inst.ID); err == nil {
		t.Fatalf("expected instance to be gone")
	}
	due, err := r.ListDueProvisionJobs(ctx, 10)
	if err != nil {
		t.Fatalf("list due provision jobs: %v", err)
	}
	for _, d := range due {
		if d.OrderItemID == items[0].ID {
			t.Fatalf("expected provision job for deleted instance to be removed, still listed")
		}
	}
}

func TestSQLiteRepo_ListInstancesDefaultsLimit(t *testing.T) {
	_, r := newTestRepo(t)
	ctx := context.Background()

	user := domain.User{Username: "limuser", Email: "limuser@example.com", PasswordHash: "hash", Role: domain.UserRoleUser, Status: domain.UserStatusActive}
	if err := r.CreateUser(ctx, &user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	for i := 0; i < 25; i++ {
		inst := domain.VPSInstance{UserID: user.ID, AutomationInstanceID: fmt.Sprintf("lim-%d", i), Name: fmt.Sprintf("vm-%d", i), Status: domain.VPSStatusRunning, AdminStatus: domain.VPSAdminStatusNormal, SpecJSON: "{}"}
		if err := r.CreateInstance(ctx, &inst); err != nil {
			t.Fatalf("create instance %d: %v", i, err)
		}
	}

	rows, total, err := r.ListInstances(ctx, 0, 0)
	if err != nil {
		t.Fatalf("list instances with limit=0: %v", err)
	}
	if total != 25 {
		t.Fatalf("expected total=25, got %d", total)
	}
	if len(rows) != 20 {
		t.Fatalf("expected limit=0 to default to 20 rows, got %d", len(rows))
	}
}

func TestSQLiteRepo_UpdateInstanceStatus_AdvancesOrderFromProvisioning(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()

	user := testutil.CreateUser(t, repo, "provuser", "provuser@example.com", "pass")
	order := domain.Order{UserID: user.ID, OrderNo: "ORD-PROV-SYNC", Status: domain.OrderStatusProvisioning, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	item := domain.OrderItem{
		OrderID: order.ID, Amount: 1000, Status: domain.OrderItemStatusProvisioning,
		Action: "create", SpecJSON: "{}", GoodsTypeID: 1,
	}
	if err := repo.CreateOrderItems(ctx, []domain.OrderItem{item}); err != nil {
		t.Fatalf("create item: %v", err)
	}
	items, err := repo.ListOrderItems(ctx, order.ID)
	if err != nil || len(items) != 1 {
		t.Fatalf("list items: %v %d", err, len(items))
	}

	expireAt := time.Now().Add(24 * time.Hour)
	inst := domain.VPSInstance{
		UserID:               user.ID,
		OrderItemID:          items[0].ID,
		AutomationInstanceID: "3001",
		GoodsTypeID:          1,
		Name:                 "vm-prov",
		Status:               domain.VPSStatusProvisioning,
		AdminStatus:          domain.VPSAdminStatusNormal,
		SpecJSON:             "{}",
		ExpireAt:             &expireAt,
	}
	if err := repo.CreateInstance(ctx, &inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}

	if err := repo.UpdateInstanceStatus(ctx, inst.ID, domain.VPSStatusRunning, 2); err != nil {
		t.Fatalf("update status: %v", err)
	}

	itemAfter, err := repo.GetOrderItem(ctx, items[0].ID)
	if err != nil {
		t.Fatalf("get item: %v", err)
	}
	if itemAfter.Status != domain.OrderItemStatusActive {
		t.Fatalf("expected item active, got %s", itemAfter.Status)
	}
	orderAfter, err := repo.GetOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if orderAfter.Status != domain.OrderStatusActive {
		t.Fatalf("expected order active, got %s", orderAfter.Status)
	}
}
