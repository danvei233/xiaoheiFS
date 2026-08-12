package order_test

import (
	"context"
	"testing"
	"time"

	repo "xiaoheiplay/internal/adapter/repo/core"
	apporder "xiaoheiplay/internal/app/order"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

// staleProvisioningOrderRepo models the concurrent-consumer race: the worker
// reads a stale "provisioning" order status while the DB row has actually been
// moved to a terminal state by another actor. refreshOrderStatus must not
// overwrite the terminal state.
type staleProvisioningOrderRepo struct {
	*repo.GormRepo
	orderID     int64
	staleStatus domain.OrderStatus
}

func (r *staleProvisioningOrderRepo) GetOrder(ctx context.Context, id int64) (domain.Order, error) {
	o, err := r.GormRepo.GetOrder(ctx, id)
	if err == nil && id == r.orderID {
		o.Status = r.staleStatus
	}
	return o, err
}

// Regression: the order-status transition in refreshOrderStatus must be atomic.
// When a stale worker computes "Active" from a provisioning snapshot while the
// order has actually been canceled, the transition must not overwrite the
// canceled state (which would resurrect a dead order and send duplicate
// completion notifications in a concurrent-consumer deployment).
func TestProvisionWorker_DoesNotOverwriteConcurrentOrderStatus(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "prov-stale", "prov-stale@example.com", "pass")

	order := domain.Order{UserID: user.ID, OrderNo: "ORD-PROV-STALE", Status: domain.OrderStatusCanceled, TotalAmount: 1000, Currency: "CNY"}
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
		AutomationInstanceID: "9004",
		GoodsTypeID:          item.GoodsTypeID,
		Name:                 "vm-stale",
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
		HostID:      9004,
		HostName:    "vm-stale",
		Status:      "pending",
		Attempts:    0,
		NextRunAt:   time.Now().UTC().Add(-time.Minute),
	}
	if err := repo.CreateOrUpdateProvisionJob(context.Background(), &job); err != nil {
		t.Fatalf("create job: %v", err)
	}

	fakeAuto := &testutil.FakeAutomationClient{
		HostInfo: map[int64]appshared.AutomationHostInfo{
			9004: {HostID: 9004, HostName: "vm-stale", State: 2},
		},
	}
	autoResolver := &testutil.FakeAutomationResolver{Client: fakeAuto}
	staleRepo := &staleProvisioningOrderRepo{GormRepo: repo, orderID: order.ID, staleStatus: domain.OrderStatusProvisioning}
	svc := apporder.NewService(staleRepo, repo, repo, repo, repo, repo, repo, repo, repo, nil, autoResolver, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)

	if err := svc.ProcessProvisionJobs(context.Background(), 10); err != nil {
		t.Fatalf("process jobs: %v", err)
	}
	got, err := repo.GetOrder(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if got.Status != domain.OrderStatusCanceled {
		t.Fatalf("expected order to remain canceled, got %v", got.Status)
	}
}
