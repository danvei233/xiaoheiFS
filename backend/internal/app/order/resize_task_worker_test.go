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

// panicResizeClient panics inside ElasticUpdate, modelling a buggy automation
// plugin that must not take down the resize task runner.
type panicResizeClient struct {
	*testutil.FakeAutomationClient
}

func (c *panicResizeClient) ElasticUpdate(context.Context, appshared.AutomationElasticUpdateRequest) error {
	panic("boom: resize elastic update")
}

// Regression: a panic inside a resize task must be contained and the task
// marked failed, not crash the worker or leave the task stuck running.
func TestResizeWorker_ContainsPanicAndFailsTask(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "rsz-panic", "rsz-panic@example.com", "pass")

	order := domain.Order{UserID: user.ID, OrderNo: "ORD-RSZ-PANIC", Status: domain.OrderStatusProvisioning, TotalAmount: 0, Currency: "CNY"}
	if err := repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	inst := domain.VPSInstance{
		UserID:               user.ID,
		AutomationInstanceID: "7001",
		GoodsTypeID:          seed.Package.GoodsTypeID,
		PackageID:            seed.Package.ID,
		Name:                 "vm-rsz-panic",
		Status:               domain.VPSStatusRunning,
		SpecJSON:             "{}",
	}
	if err := repo.CreateInstance(context.Background(), &inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}
	item := domain.OrderItem{
		OrderID:   order.ID,
		PackageID: seed.Package.ID,
		SystemID:  seed.SystemImage.ID,
		Amount:    0,
		Status:    domain.OrderItemStatusProvisioning,
		Action:    "resize",
		SpecJSON:  fmt.Sprintf(`{"vps_id": %d}`, inst.ID),
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{item}); err != nil {
		t.Fatalf("create item: %v", err)
	}
	items, _ := repo.ListOrderItems(context.Background(), order.ID)
	item = items[0]

	task := domain.ResizeTask{
		VPSID:       inst.ID,
		OrderID:     order.ID,
		OrderItemID: item.ID,
		Status:      domain.ResizeTaskStatusPending,
		ScheduledAt: &[]time.Time{time.Now().UTC().Add(-time.Minute)}[0],
	}
	if err := repo.CreateResizeTask(context.Background(), &task); err != nil {
		t.Fatalf("create resize task: %v", err)
	}

	autoResolver := &testutil.FakeAutomationResolver{Client: &panicResizeClient{FakeAutomationClient: &testutil.FakeAutomationClient{}}}
	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, autoResolver, nil, repo, repo, nil, repo, repo, repo, repo, nil, nil)

	if err := svc.ProcessResizeTasks(context.Background(), 10); err != nil {
		t.Fatalf("process resize tasks should return after contained panic: %v", err)
	}
	got, err := repo.GetResizeTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("get resize task: %v", err)
	}
	if got.Status != domain.ResizeTaskStatusFailed {
		t.Fatalf("expected resize task failed after panic, got %v", got.Status)
	}
}
