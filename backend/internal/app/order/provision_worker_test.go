package order_test

import (
	"context"
	"fmt"
	"testing"
	"time"
	apporder "xiaoheiplay/internal/app/order"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func TestProvisionWorker_MarksFailedOnCreateFailedState(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "provfail", "provfail@example.com", "pass")

	order := domain.Order{UserID: user.ID, OrderNo: "ORD-PROV-FAIL-STATE", Status: domain.OrderStatusProvisioning, TotalAmount: 1000, Currency: "CNY"}
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
	if len(items) == 0 {
		t.Fatalf("expected order item")
	}

	job := domain.ProvisionJob{
		OrderID:     order.ID,
		OrderItemID: items[0].ID,
		HostID:      2001,
		HostName:    "vm-fail",
		Status:      "pending",
		Attempts:    0,
		NextRunAt:   time.Now().UTC().Add(-time.Minute),
	}
	if err := repo.CreateOrUpdateProvisionJob(context.Background(), &job); err != nil {
		t.Fatalf("create job: %v", err)
	}

	fakeAuto := &testutil.FakeAutomationClient{
		HostInfo: map[int64]appshared.AutomationHostInfo{
			2001: {HostID: 2001, HostName: "vm-fail", State: 11},
		},
	}
	autoResolver := &testutil.FakeAutomationResolver{Client: fakeAuto}
	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, autoResolver, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)

	if err := svc.ProcessProvisionJobs(context.Background(), 10); err != nil {
		t.Fatalf("process jobs: %v", err)
	}
	updatedItems, _ := repo.ListOrderItems(context.Background(), order.ID)
	if len(updatedItems) == 0 || updatedItems[0].Status != domain.OrderItemStatusFailed {
		t.Fatalf("expected item failed, got %+v", updatedItems)
	}
	updatedOrder, _ := repo.GetOrder(context.Background(), order.ID)
	if updatedOrder.Status != domain.OrderStatusFailed {
		t.Fatalf("expected order failed, got %v", updatedOrder.Status)
	}
	if jobs, err := repo.ListDueProvisionJobs(context.Background(), 10); err != nil || len(jobs) != 0 {
		t.Fatalf("expected no due jobs, got %v %v", len(jobs), err)
	}
}

func TestProvisionWorker_DoneWhenOrderMissing(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	job := domain.ProvisionJob{
		OrderID:     999,
		OrderItemID: 1000,
		HostID:      3001,
		HostName:    "vm-missing",
		Status:      "pending",
		Attempts:    0,
		NextRunAt:   time.Now().UTC().Add(-time.Minute),
	}
	if err := repo.CreateOrUpdateProvisionJob(context.Background(), &job); err != nil {
		t.Fatalf("create job: %v", err)
	}
	autoResolver := &testutil.FakeAutomationResolver{Client: &testutil.FakeAutomationClient{}}
	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, autoResolver, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)

	if err := svc.ProcessProvisionJobs(context.Background(), 10); err != nil {
		t.Fatalf("process jobs: %v", err)
	}
	if jobs, err := repo.ListDueProvisionJobs(context.Background(), 10); err != nil || len(jobs) != 0 {
		t.Fatalf("expected no due jobs, got %v %v", len(jobs), err)
	}
}

func TestProvisionWorker_Reconcile_RepairsPrematureProvisioning(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "provfix", "provfix@example.com", "pass")

	order := domain.Order{UserID: user.ID, OrderNo: "ORD-PROV-STUCK", Status: domain.OrderStatusProvisioning, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	item := domain.OrderItem{
		OrderID:   order.ID,
		PackageID: seed.Package.ID,
		SystemID:  seed.SystemImage.ID,
		Amount:    1000,
		Status:    domain.OrderItemStatusPendingReview,
		Action:    "create",
		SpecJSON:  "{}",
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{item}); err != nil {
		t.Fatalf("create item: %v", err)
	}

	autoResolver := &testutil.FakeAutomationResolver{Client: &testutil.FakeAutomationClient{}}
	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, autoResolver, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)

	if _, err := svc.ReconcileProvisioningOrders(context.Background(), 20); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	updatedOrder, _ := repo.GetOrder(context.Background(), order.ID)
	if updatedOrder.Status != domain.OrderStatusPendingReview {
		t.Fatalf("expected pending_review, got %v", updatedOrder.Status)
	}
}

func TestProvisionWorker_UpdatesInstanceAutomationIDOnRetryHostSwitch(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "provsync", "provsync@example.com", "pass")

	order := domain.Order{
		UserID:      user.ID,
		OrderNo:     "ORD-PROV-SYNC-HOST",
		Status:      domain.OrderStatusProvisioning,
		TotalAmount: 1000,
		Currency:    "CNY",
	}
	if err := repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	item := domain.OrderItem{
		OrderID:              order.ID,
		PackageID:            seed.Package.ID,
		SystemID:             seed.SystemImage.ID,
		Amount:               1000,
		Status:               domain.OrderItemStatusProvisioning,
		Action:               "create",
		SpecJSON:             "{}",
		AutomationInstanceID: "5001",
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{item}); err != nil {
		t.Fatalf("create item: %v", err)
	}
	items, _ := repo.ListOrderItems(context.Background(), order.ID)
	if len(items) == 0 {
		t.Fatalf("expected order item")
	}
	item = items[0]

	inst := domain.VPSInstance{
		UserID:               user.ID,
		OrderItemID:          item.ID,
		AutomationInstanceID: "5001",
		GoodsTypeID:          item.GoodsTypeID,
		Name:                 "vm-old",
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
		HostID:      6001,
		HostName:    "vm-new",
		Status:      "pending",
		Attempts:    0,
		NextRunAt:   time.Now().UTC().Add(-time.Minute),
	}
	if err := repo.CreateOrUpdateProvisionJob(context.Background(), &job); err != nil {
		t.Fatalf("create job: %v", err)
	}

	fakeAuto := &testutil.FakeAutomationClient{
		HostInfo: map[int64]appshared.AutomationHostInfo{
			6001: {HostID: 6001, HostName: "vm-new", State: 2, RemoteIP: "2.2.2.2"},
		},
	}
	autoResolver := &testutil.FakeAutomationResolver{Client: fakeAuto}
	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, autoResolver, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)

	if err := svc.ProcessProvisionJobs(context.Background(), 10); err != nil {
		t.Fatalf("process jobs: %v", err)
	}

	updatedItem, err := repo.GetOrderItem(context.Background(), item.ID)
	if err != nil {
		t.Fatalf("get order item: %v", err)
	}
	if updatedItem.AutomationInstanceID != "6001" {
		t.Fatalf("expected order item automation id 6001, got %q", updatedItem.AutomationInstanceID)
	}

	updatedInst, err := repo.GetInstanceByOrderItem(context.Background(), item.ID)
	if err != nil {
		t.Fatalf("get instance: %v", err)
	}
	if updatedInst.AutomationInstanceID != "6001" {
		t.Fatalf("expected instance automation id 6001, got %q", updatedInst.AutomationInstanceID)
	}
}

// TestProvisionWorker_TimeoutFailsHostStuckOnAPIFailure guards the regression
// where a persistently failing GetHostInfo (e.g. the automation plugin is
// down) retried forever because the timeout check moved behind the API call.
// A host that never reports ready and whose watchdog budget has elapsed must
// reach a terminal failed state, not spin forever.
func TestProvisionWorker_TimeoutFailsHostStuckOnAPIFailure(t *testing.T) {
	rawDB, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "provapierr", "provapierr@example.com", "pass")

	order := domain.Order{UserID: user.ID, OrderNo: "ORD-PROV-API-ERR", Status: domain.OrderStatusProvisioning, TotalAmount: 1000, Currency: "CNY"}
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
	if len(items) == 0 {
		t.Fatalf("expected order item")
	}

	job := domain.ProvisionJob{
		OrderID:     order.ID,
		OrderItemID: items[0].ID,
		HostID:      7001,
		HostName:    "vm-api-err",
		Status:      "pending",
		Attempts:    0,
		NextRunAt:   time.Now().UTC().Add(-time.Minute),
	}
	if err := repo.CreateOrUpdateProvisionJob(context.Background(), &job); err != nil {
		t.Fatalf("create job: %v", err)
	}
	// The watchdog budget is 20 minutes by default; age the job past it so the
	// timeout path is exercised without waiting.
	if _, err := rawDB.Exec("UPDATE provision_jobs SET created_at = datetime('now', '-21 minutes') WHERE id = ?", job.ID); err != nil {
		t.Fatalf("age job: %v", err)
	}

	fakeAuto := &testutil.FakeAutomationClient{HostInfoErr: fmt.Errorf("automation api down")}
	autoResolver := &testutil.FakeAutomationResolver{Client: fakeAuto}
	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, autoResolver, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)

	if err := svc.ProcessProvisionJobs(context.Background(), 10); err != nil {
		t.Fatalf("process jobs: %v", err)
	}
	updatedItems, _ := repo.ListOrderItems(context.Background(), order.ID)
	if len(updatedItems) == 0 || updatedItems[0].Status != domain.OrderItemStatusFailed {
		t.Fatalf("expected item failed after watchdog timeout on API error, got %+v", updatedItems)
	}
	updatedOrder, _ := repo.GetOrder(context.Background(), order.ID)
	if updatedOrder.Status != domain.OrderStatusFailed {
		t.Fatalf("expected order failed, got %v", updatedOrder.Status)
	}
	if jobs, err := repo.ListDueProvisionJobs(context.Background(), 10); err != nil || len(jobs) != 0 {
		t.Fatalf("expected no due jobs after terminal failure, got %v %v", len(jobs), err)
	}
}

// TestProvisionWorker_Reconcile_RecoversStrandedCreatingItem locks the
// recovery path for a `creating:<host>` order item stranded with no durable
// provision job and no local instance (a crash in a version before the durable
// intent was persisted). ReconcileProvisioningOrders must enqueue a
// create_host_sent-stage job so the watchdog reconciles by hostname instead of
// leaving the order stuck in provisioning forever.
func TestProvisionWorker_Reconcile_RecoversStrandedCreatingItem(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "provrecover", "provrecover@example.com", "pass")

	order := domain.Order{UserID: user.ID, OrderNo: "ORD-PROV-STRANDED", Status: domain.OrderStatusProvisioning, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	item := domain.OrderItem{
		OrderID:              order.ID,
		PackageID:            seed.Package.ID,
		SystemID:             seed.SystemImage.ID,
		Amount:               1000,
		Status:               domain.OrderItemStatusProvisioning,
		Action:               "create",
		SpecJSON:             "{}",
		AutomationInstanceID: "creating:ecs-42-43-44",
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{item}); err != nil {
		t.Fatalf("create item: %v", err)
	}
	items, _ := repo.ListOrderItems(context.Background(), order.ID)
	if len(items) == 0 {
		t.Fatalf("expected order item")
	}
	// No provision job and no local instance exist: the stranded state.
	if _, err := repo.GetProvisionJobByOrderItem(context.Background(), items[0].ID); err == nil {
		t.Fatalf("expected no provision job for the stranded item")
	}

	autoResolver := &testutil.FakeAutomationResolver{Client: &testutil.FakeAutomationClient{}}
	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, autoResolver, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)

	if _, err := svc.ReconcileProvisioningOrders(context.Background(), 20); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	job, err := repo.GetProvisionJobByOrderItem(context.Background(), items[0].ID)
	if err != nil {
		t.Fatalf("expected a recovery job to be enqueued for the stranded item, got %v", err)
	}
	if job.HostName != "ecs-42-43-44" {
		t.Fatalf("recovery job must reconcile by the deterministic hostname from the marker, got %q", job.HostName)
	}
	if job.Stage != domain.ProvisionStageCreateHostSent {
		t.Fatalf("recovery job must be create_host_sent (reconcile-only, never re-issues CreateHost), got %q", job.Stage)
	}
}
