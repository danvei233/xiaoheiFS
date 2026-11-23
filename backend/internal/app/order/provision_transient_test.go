package order_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	repo "xiaoheiplay/internal/adapter/repo/core"
	apporder "xiaoheiplay/internal/app/order"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func TestApproveProvisionTimeoutFallsBackToAsyncWorker(t *testing.T) {
	_, r := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, r)
	user := testutil.CreateUser(t, r, "prov-timeout", "prov-timeout@example.com", "pass")

	order := domain.Order{UserID: user.ID, OrderNo: "ORD-PROV-TIMEOUT", Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "CNY"}
	if err := r.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	item := domain.OrderItem{
		OrderID:        order.ID,
		PackageID:      seed.Package.ID,
		SystemID:       seed.SystemImage.ID,
		GoodsTypeID:    seed.Package.GoodsTypeID,
		Amount:         1000,
		Status:         domain.OrderItemStatusPendingPayment,
		Action:         "create",
		SpecJSON:       "{}",
		DurationMonths: 1,
	}
	if err := r.CreateOrderItems(context.Background(), []domain.OrderItem{item}); err != nil {
		t.Fatalf("create item: %v", err)
	}

	fakeAuto := &testutil.FakeAutomationClient{
		CreateHostResult: appshared.AutomationCreateHostResult{HostID: 3003},
		HostInfoErr:      fmt.Errorf("%w: rpc error: code = DeadlineExceeded", appshared.ErrAutomationTransient),
	}
	autoResolver := &testutil.FakeAutomationResolver{Client: fakeAuto}
	svc := apporder.NewService(r, r, r, r, r, r, r, r, r, nil, autoResolver, nil, r, r, nil, r, r, r, nil, nil, nil)

	if err := svc.ApproveOrder(context.Background(), 1, order.ID); err != nil {
		t.Fatalf("approve order: %v", err)
	}

	items := waitItemProvisioning(t, r, order.ID)
	if items[0].Status != domain.OrderItemStatusProvisioning {
		t.Fatalf("item status = %s, want provisioning", items[0].Status)
	}
	if items[0].AutomationInstanceID != "3003" {
		t.Fatalf("automation instance = %q, want 3003", items[0].AutomationInstanceID)
	}
	job := waitProvisionJob(t, r, items[0].ID)
	if job.Status == "done" {
		t.Fatalf("provision job finished, want re-armed for async polling")
	}
	o, err := r.GetOrder(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if o.Status != domain.OrderStatusProvisioning {
		t.Fatalf("order status = %s, want provisioning", o.Status)
	}
}

func waitItemProvisioning(t *testing.T, r *repo.GormRepo, orderID int64) []domain.OrderItem {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		items, err := r.ListOrderItems(context.Background(), orderID)
		if err == nil && len(items) > 0 && items[0].Status == domain.OrderItemStatusProvisioning && items[0].AutomationInstanceID != "" {
			return items
		}
		time.Sleep(20 * time.Millisecond)
	}
	items, _ := r.ListOrderItems(context.Background(), orderID)
	if len(items) > 0 {
		t.Fatalf("item = %s/%s, want provisioning with host id", items[0].Status, items[0].AutomationInstanceID)
	}
	t.Fatalf("item not found")
	return nil
}

func waitProvisionJob(t *testing.T, r *repo.GormRepo, orderItemID int64) domain.ProvisionJob {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		jobs, err := r.ListDueProvisionJobs(context.Background(), 20)
		if err == nil {
			for _, job := range jobs {
				if job.OrderItemID == orderItemID {
					return job
				}
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("provision job for item %d not found", orderItemID)
	return domain.ProvisionJob{}
}
