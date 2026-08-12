package order_test

import (
	"context"
	"testing"
	"time"

	apporder "xiaoheiplay/internal/app/order"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

// Regression: a job that has outlived provision_watchdog_max_minutes must still
// complete normally when the automation host is already ready. The old code
// declared "timeout" before ever asking the host, so a host that came up during
// a worker crash or slow install was permanently failed with no compensation.
func TestProvisionWorker_TimeoutDoesNotFailReadyHost(t *testing.T) {
	db, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "prov-ready", "prov-ready@example.com", "pass")

	order := domain.Order{UserID: user.ID, OrderNo: "ORD-PROV-READY", Status: domain.OrderStatusProvisioning, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	item := domain.OrderItem{
		OrderID:   order.ID,
		PackageID: seed.Package.ID,
		SystemID:  seed.SystemImage.ID,
		Amount:    1000,
		Status:    domain.OrderItemStatusProvisioning,
		Action:    "create",
		SpecJSON:  "{}",
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{item}); err != nil {
		t.Fatalf("create item: %v", err)
	}
	items, _ := repo.ListOrderItems(context.Background(), order.ID)
	item = items[0]

	inst := domain.VPSInstance{
		UserID:               user.ID,
		OrderItemID:          item.ID,
		AutomationInstanceID: "9001",
		GoodsTypeID:          item.GoodsTypeID,
		Name:                 "vm-ready",
		SystemID:             seed.SystemImage.ID,
		Status:               domain.VPSStatusProvisioning,
		SpecJSON:             "{}",
	}
	if err := repo.CreateInstance(context.Background(), &inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}

	job := domain.ProvisionJob{
		OrderID:     order.ID,
		OrderItemID: item.ID,
		HostID:      9001,
		HostName:    "vm-ready",
		Status:      "pending",
		Attempts:    0,
		NextRunAt:   time.Now().UTC().Add(-time.Minute),
	}
	if err := repo.CreateOrUpdateProvisionJob(context.Background(), &job); err != nil {
		t.Fatalf("create job: %v", err)
	}
	// Backdate the job so it has outlived the default 20-minute watchdog budget.
	if _, err := db.Exec("UPDATE provision_jobs SET created_at = ? WHERE id = ?", time.Now().UTC().Add(-21*time.Minute), job.ID); err != nil {
		t.Fatalf("backdate job: %v", err)
	}

	fakeAuto := &testutil.FakeAutomationClient{
		HostInfo: map[int64]appshared.AutomationHostInfo{
			9001: {HostID: 9001, HostName: "vm-ready", State: 2, RemoteIP: "9.9.9.9"},
		},
	}
	autoResolver := &testutil.FakeAutomationResolver{Client: fakeAuto}
	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, autoResolver, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)

	if err := svc.ProcessProvisionJobs(context.Background(), 10); err != nil {
		t.Fatalf("process jobs: %v", err)
	}
	updatedItem, err := repo.GetOrderItem(context.Background(), item.ID)
	if err != nil {
		t.Fatalf("get item: %v", err)
	}
	if updatedItem.Status != domain.OrderItemStatusActive {
		t.Fatalf("expected item active (host already ready despite age), got %v", updatedItem.Status)
	}
	updatedOrder, err := repo.GetOrder(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if updatedOrder.Status != domain.OrderStatusActive {
		t.Fatalf("expected order active, got %v", updatedOrder.Status)
	}
}

// Guard: a host genuinely stuck in provisioning (State==0) past the watchdog
// budget must still be failed — the timeout must fire only when the host is not
// ready, not merely because the job is old.
func TestProvisionWorker_TimeoutStillFailsHostStuckInProvisioning(t *testing.T) {
	db, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "prov-stuck", "prov-stuck@example.com", "pass")

	order := domain.Order{UserID: user.ID, OrderNo: "ORD-PROV-STUCK-TO", Status: domain.OrderStatusProvisioning, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	item := domain.OrderItem{
		OrderID:   order.ID,
		PackageID: seed.Package.ID,
		SystemID:  seed.SystemImage.ID,
		Amount:    1000,
		Status:    domain.OrderItemStatusProvisioning,
		Action:    "create",
		SpecJSON:  "{}",
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{item}); err != nil {
		t.Fatalf("create item: %v", err)
	}
	items, _ := repo.ListOrderItems(context.Background(), order.ID)
	item = items[0]

	job := domain.ProvisionJob{
		OrderID:     order.ID,
		OrderItemID: item.ID,
		HostID:      9002,
		HostName:    "vm-stuck",
		Status:      "pending",
		Attempts:    0,
		NextRunAt:   time.Now().UTC().Add(-time.Minute),
	}
	if err := repo.CreateOrUpdateProvisionJob(context.Background(), &job); err != nil {
		t.Fatalf("create job: %v", err)
	}
	if _, err := db.Exec("UPDATE provision_jobs SET created_at = ? WHERE id = ?", time.Now().UTC().Add(-21*time.Minute), job.ID); err != nil {
		t.Fatalf("backdate job: %v", err)
	}

	fakeAuto := &testutil.FakeAutomationClient{
		HostInfo: map[int64]appshared.AutomationHostInfo{
			9002: {HostID: 9002, HostName: "vm-stuck", State: 0},
		},
	}
	autoResolver := &testutil.FakeAutomationResolver{Client: fakeAuto}
	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, autoResolver, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)

	if err := svc.ProcessProvisionJobs(context.Background(), 10); err != nil {
		t.Fatalf("process jobs: %v", err)
	}
	updatedItem, err := repo.GetOrderItem(context.Background(), item.ID)
	if err != nil {
		t.Fatalf("get item: %v", err)
	}
	if updatedItem.Status != domain.OrderItemStatusFailed {
		t.Fatalf("expected item failed (host still provisioning past budget), got %v", updatedItem.Status)
	}
	updatedOrder, _ := repo.GetOrder(context.Background(), order.ID)
	if updatedOrder.Status != domain.OrderStatusFailed {
		t.Fatalf("expected order failed, got %v", updatedOrder.Status)
	}
}

// Guard: the watchdog budget must also apply to provisioning states other than
// State==0 (e.g. state 13, which MapAutomationState maps to provisioning). The
// timeout must fire for any not-ready, not-failed host past the budget.
func TestProvisionWorker_TimeoutFailsNonZeroProvisioningState(t *testing.T) {
	db, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "prov-st13", "prov-st13@example.com", "pass")

	order := domain.Order{UserID: user.ID, OrderNo: "ORD-PROV-TO-13", Status: domain.OrderStatusProvisioning, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	item := domain.OrderItem{
		OrderID:   order.ID,
		PackageID: seed.Package.ID,
		SystemID:  seed.SystemImage.ID,
		Amount:    1000,
		Status:    domain.OrderItemStatusProvisioning,
		Action:    "create",
		SpecJSON:  "{}",
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{item}); err != nil {
		t.Fatalf("create item: %v", err)
	}
	items, _ := repo.ListOrderItems(context.Background(), order.ID)
	item = items[0]

	job := domain.ProvisionJob{
		OrderID:     order.ID,
		OrderItemID: item.ID,
		HostID:      9005,
		HostName:    "vm-st13",
		Status:      "pending",
		Attempts:    0,
		NextRunAt:   time.Now().UTC().Add(-time.Minute),
	}
	if err := repo.CreateOrUpdateProvisionJob(context.Background(), &job); err != nil {
		t.Fatalf("create job: %v", err)
	}
	if _, err := db.Exec("UPDATE provision_jobs SET created_at = ? WHERE id = ?", time.Now().UTC().Add(-21*time.Minute), job.ID); err != nil {
		t.Fatalf("backdate job: %v", err)
	}

	fakeAuto := &testutil.FakeAutomationClient{
		HostInfo: map[int64]appshared.AutomationHostInfo{
			9005: {HostID: 9005, HostName: "vm-st13", State: 13},
		},
	}
	autoResolver := &testutil.FakeAutomationResolver{Client: fakeAuto}
	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, autoResolver, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)

	if err := svc.ProcessProvisionJobs(context.Background(), 10); err != nil {
		t.Fatalf("process jobs: %v", err)
	}
	updatedItem, err := repo.GetOrderItem(context.Background(), item.ID)
	if err != nil {
		t.Fatalf("get item: %v", err)
	}
	if updatedItem.Status != domain.OrderItemStatusFailed {
		t.Fatalf("expected item failed (state 13 still provisioning past budget), got %v", updatedItem.Status)
	}
}
