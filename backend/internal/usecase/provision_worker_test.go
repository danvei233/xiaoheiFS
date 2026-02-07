package usecase_test

import (
	"context"
	"testing"
	"time"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/usecase"
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
		HostInfo: map[int64]usecase.AutomationHostInfo{
			2001: {HostID: 2001, HostName: "vm-fail", State: 11},
		},
	}
	autoResolver := &testutil.FakeAutomationResolver{Client: fakeAuto}
	svc := usecase.NewOrderService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, autoResolver, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)

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
	svc := usecase.NewOrderService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, autoResolver, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)

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
	svc := usecase.NewOrderService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, autoResolver, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)

	if _, err := svc.ReconcileProvisioningOrders(context.Background(), 20); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	updatedOrder, _ := repo.GetOrder(context.Background(), order.ID)
	if updatedOrder.Status != domain.OrderStatusPendingReview {
		t.Fatalf("expected pending_review, got %v", updatedOrder.Status)
	}
}
