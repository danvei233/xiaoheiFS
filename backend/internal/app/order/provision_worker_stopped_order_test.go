package order_test

import (
	"context"
	"sync"
	"testing"
	"time"

	repo "xiaoheiplay/internal/adapter/repo/core"
	apporder "xiaoheiplay/internal/app/order"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

// rejectDuringHostInfoAutomation simulates an admin rejecting the order while
// the worker is inside the (slow) GetHostInfo call: the order is flipped to
// Rejected before GetHostInfo returns a ready host.
type rejectDuringHostInfoAutomation struct {
	*testutil.FakeAutomationClient
	repo    *repo.GormRepo
	orderID int64
	once    sync.Once
}

func (c *rejectDuringHostInfoAutomation) GetHostInfo(ctx context.Context, hostID int64) (appshared.AutomationHostInfo, error) {
	c.once.Do(func() {
		if o, err := c.repo.GetOrder(context.Background(), c.orderID); err == nil {
			o.Status = domain.OrderStatusRejected
			o.RejectedReason = "admin rejected during provisioning"
			_ = c.repo.UpdateOrderMeta(context.Background(), o)
		}
	})
	return c.FakeAutomationClient.GetHostInfo(ctx, hostID)
}

// Regression: completeProvision must re-check the order status before activating
// the item. If the order was rejected/canceled while GetHostInfo was in flight,
// the worker must not activate the item or VPS for a dead order.
func TestProvisionWorker_DoesNotActivateItemForRejectedOrder(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "prov-rej", "prov-rej@example.com", "pass")

	order := domain.Order{UserID: user.ID, OrderNo: "ORD-PROV-REJ", Status: domain.OrderStatusApproved, TotalAmount: 1000, Currency: "CNY"}
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
		AutomationInstanceID: "9003",
		GoodsTypeID:          item.GoodsTypeID,
		Name:                 "vm-rej",
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
		HostID:      9003,
		HostName:    "vm-rej",
		Status:      "pending",
		Attempts:    0,
		NextRunAt:   time.Now().UTC().Add(-time.Minute),
	}
	if err := repo.CreateOrUpdateProvisionJob(context.Background(), &job); err != nil {
		t.Fatalf("create job: %v", err)
	}

	fakeAuto := &testutil.FakeAutomationClient{
		HostInfo: map[int64]appshared.AutomationHostInfo{
			9003: {HostID: 9003, HostName: "vm-rej", State: 2},
		},
	}
	rejAuto := &rejectDuringHostInfoAutomation{FakeAutomationClient: fakeAuto, repo: repo, orderID: order.ID}
	autoResolver := &testutil.FakeAutomationResolver{Client: rejAuto}
	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, autoResolver, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)

	if err := svc.ProcessProvisionJobs(context.Background(), 10); err != nil {
		t.Fatalf("process jobs: %v", err)
	}
	updatedItem, err := repo.GetOrderItem(context.Background(), item.ID)
	if err != nil {
		t.Fatalf("get item: %v", err)
	}
	if updatedItem.Status == domain.OrderItemStatusActive {
		t.Fatalf("expected item NOT active for rejected order, got %v", updatedItem.Status)
	}
	updatedOrder, err := repo.GetOrder(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if updatedOrder.Status != domain.OrderStatusRejected {
		t.Fatalf("expected order to remain rejected, got %v", updatedOrder.Status)
	}
}
