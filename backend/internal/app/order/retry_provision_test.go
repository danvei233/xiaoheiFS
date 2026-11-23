package order_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	repo "xiaoheiplay/internal/adapter/repo/core"
	apporder "xiaoheiplay/internal/app/order"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func setupFailedCreateOrder(t *testing.T, orderNo, itemAutomationID string) (*sql.DB, *repo.GormRepo, domain.Order, domain.OrderItem) {
	t.Helper()
	db, r := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, r)
	user := testutil.CreateUser(t, r, orderNo+"-user", orderNo+"@example.com", "pass")

	order := domain.Order{
		UserID:      user.ID,
		OrderNo:     orderNo,
		Status:      domain.OrderStatusFailed,
		TotalAmount: 1000,
		Currency:    "CNY",
	}
	if err := r.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	item := domain.OrderItem{
		OrderID:              order.ID,
		PackageID:            seed.Package.ID,
		SystemID:             seed.SystemImage.ID,
		GoodsTypeID:          seed.Package.GoodsTypeID,
		Amount:               1000,
		Status:               domain.OrderItemStatusFailed,
		Action:               "create",
		SpecJSON:             "{}",
		DurationMonths:       1,
		AutomationInstanceID: itemAutomationID,
	}
	if err := r.CreateOrderItems(context.Background(), []domain.OrderItem{item}); err != nil {
		t.Fatalf("create item: %v", err)
	}
	items, err := r.ListOrderItems(context.Background(), order.ID)
	if err != nil || len(items) != 1 {
		t.Fatalf("list items: %v %d", err, len(items))
	}
	return db, r, order, items[0]
}

func createFailedInstance(t *testing.T, r *repo.GormRepo, userID, orderItemID int64, automationID string) {
	t.Helper()
	inst := domain.VPSInstance{
		UserID:               userID,
		OrderItemID:          orderItemID,
		AutomationInstanceID: automationID,
		Status:               domain.VPSStatusUnknown,
		AdminStatus:          domain.VPSAdminStatusNormal,
	}
	if err := r.CreateInstance(context.Background(), &inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}
}

func waitItemActive(t *testing.T, r *repo.GormRepo, orderID int64, wantAutomationID string) []domain.OrderItem {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		items, err := r.ListOrderItems(context.Background(), orderID)
		if err == nil && len(items) > 0 && items[0].Status == domain.OrderItemStatusActive && items[0].AutomationInstanceID == wantAutomationID {
			return items
		}
		time.Sleep(20 * time.Millisecond)
	}
	items, _ := r.ListOrderItems(context.Background(), orderID)
	if len(items) > 0 {
		t.Fatalf("item = %s/%s, want %s/%s", items[0].Status, items[0].AutomationInstanceID, domain.OrderItemStatusActive, wantAutomationID)
	}
	t.Fatalf("item not found")
	return nil
}

func countItemInstances(t *testing.T, db *sql.DB, orderItemID int64) int {
	t.Helper()
	var count int
	if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM vps_instances WHERE order_item_id = ?", orderItemID).Scan(&count); err != nil {
		t.Fatalf("count instances: %v", err)
	}
	return count
}

func TestRetryProvisionResumesReadyHost(t *testing.T) {
	db, r, order, item := setupFailedCreateOrder(t, "ORD-RETRY-READY", "1001")
	createFailedInstance(t, r, order.UserID, item.ID, "1001")

	fakeAuto := &testutil.FakeAutomationClient{
		CreateHostResult: appshared.AutomationCreateHostResult{HostID: 2002},
		HostInfo: map[int64]appshared.AutomationHostInfo{
			1001: {HostID: 1001, HostName: "host-1001", State: 2, RemoteIP: "1.1.1.1"},
		},
	}
	autoResolver := &testutil.FakeAutomationResolver{Client: fakeAuto}
	svc := apporder.NewService(r, r, r, r, r, r, r, r, r, nil, autoResolver, nil, r, r, nil, r, r, r, nil, nil, nil)

	if err := svc.RetryProvision(order.ID); err != nil {
		t.Fatalf("retry provision: %v", err)
	}
	items := waitItemActive(t, r, order.ID, "1001")
	if len(fakeAuto.CreateHostRequests) != 0 {
		t.Fatalf("expected no new host, got %d create calls", len(fakeAuto.CreateHostRequests))
	}
	if items[0].AutomationInstanceID != "1001" {
		t.Fatalf("automation instance = %q, want 1001", items[0].AutomationInstanceID)
	}
	inst, err := r.GetInstanceByOrderItem(context.Background(), item.ID)
	if err != nil {
		t.Fatalf("get instance: %v", err)
	}
	if inst.Status != domain.VPSStatusRunning {
		t.Fatalf("instance status = %s, want running", inst.Status)
	}
	if got := countItemInstances(t, db, item.ID); got != 1 {
		t.Fatalf("instance rows = %d, want 1", got)
	}
}

func TestRetryProvisionRecreatesFailedHostWithoutDuplicatingInstance(t *testing.T) {
	db, r, order, item := setupFailedCreateOrder(t, "ORD-RETRY-RECREATE", "1001")
	createFailedInstance(t, r, order.UserID, item.ID, "1001")

	fakeAuto := &testutil.FakeAutomationClient{
		CreateHostResult: appshared.AutomationCreateHostResult{HostID: 2002},
		HostInfo: map[int64]appshared.AutomationHostInfo{
			1001: {HostID: 1001, HostName: "host-1001", State: 11},
			2002: {HostID: 2002, HostName: "host-2002", State: 2, RemoteIP: "2.2.2.2"},
		},
	}
	autoResolver := &testutil.FakeAutomationResolver{Client: fakeAuto}
	svc := apporder.NewService(r, r, r, r, r, r, r, r, r, nil, autoResolver, nil, r, r, nil, r, r, r, nil, nil, nil)

	if err := svc.RetryProvision(order.ID); err != nil {
		t.Fatalf("retry provision: %v", err)
	}
	items := waitItemActive(t, r, order.ID, "2002")
	if len(fakeAuto.CreateHostRequests) != 1 {
		t.Fatalf("create calls = %d, want 1", len(fakeAuto.CreateHostRequests))
	}
	if items[0].AutomationInstanceID != "2002" {
		t.Fatalf("automation instance = %q, want 2002", items[0].AutomationInstanceID)
	}
	inst, err := r.GetInstanceByOrderItem(context.Background(), item.ID)
	if err != nil {
		t.Fatalf("get instance: %v", err)
	}
	if inst.AutomationInstanceID != "2002" {
		t.Fatalf("local instance automation = %q, want 2002", inst.AutomationInstanceID)
	}
	if inst.Status != domain.VPSStatusRunning {
		t.Fatalf("local instance status = %s, want running", inst.Status)
	}
	if got := countItemInstances(t, db, item.ID); got != 1 {
		t.Fatalf("instance rows = %d, want 1", got)
	}
}

func TestRetryProvisionResumesProvisioningHostWithoutRecreate(t *testing.T) {
	_, r, order, item := setupFailedCreateOrder(t, "ORD-RETRY-PENDING", "1001")
	createFailedInstance(t, r, order.UserID, item.ID, "1001")

	fakeAuto := &testutil.FakeAutomationClient{
		CreateHostResult: appshared.AutomationCreateHostResult{HostID: 2002},
		HostInfo: map[int64]appshared.AutomationHostInfo{
			1001: {HostID: 1001, HostName: "host-1001", State: 1},
		},
	}
	autoResolver := &testutil.FakeAutomationResolver{Client: fakeAuto}
	svc := apporder.NewService(r, r, r, r, r, r, r, r, r, nil, autoResolver, nil, r, r, nil, r, r, r, nil, nil, nil)

	if err := svc.RetryProvision(order.ID); err != nil {
		t.Fatalf("retry provision: %v", err)
	}

	var job domain.ProvisionJob
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		due, err := r.ListDueProvisionJobs(context.Background(), 10)
		if err == nil {
			for _, itemJob := range due {
				if itemJob.OrderItemID == item.ID {
					job = itemJob
					break
				}
			}
		}
		if job.ID > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if job.ID == 0 {
		t.Fatalf("expected the retry to re-arm a provision job")
	}
	if len(fakeAuto.CreateHostRequests) != 0 {
		t.Fatalf("expected no new host, got %d create calls", len(fakeAuto.CreateHostRequests))
	}
	items, err := r.ListOrderItems(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("list items: %v", err)
	}
	if items[0].Status != domain.OrderItemStatusProvisioning {
		t.Fatalf("item status = %s, want provisioning", items[0].Status)
	}
	if items[0].AutomationInstanceID != "1001" {
		t.Fatalf("automation instance = %q, want 1001", items[0].AutomationInstanceID)
	}
}
