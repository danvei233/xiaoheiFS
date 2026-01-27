package usecase_test

import (
	"context"
	"testing"
	"time"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/usecase"
)

func TestOrderService_ApproveProvisionCreatesVPS(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "prov", "prov@example.com", "pass")

	order := domain.Order{UserID: user.ID, OrderNo: "ORD-PROV", Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	item := domain.OrderItem{
		OrderID:   order.ID,
		PackageID: seed.Package.ID,
		SystemID:  seed.SystemImage.ID,
		Amount:    1000,
		Status:    domain.OrderItemStatusPendingPayment,
		Action:    "create",
		SpecJSON:  "{}",
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{item}); err != nil {
		t.Fatalf("create item: %v", err)
	}

	fakeAuto := &testutil.FakeAutomationClient{
		CreateHostResult: usecase.AutomationCreateHostResult{HostID: 1001},
		HostInfo: map[int64]usecase.AutomationHostInfo{
			1001: {HostID: 1001, HostName: "host", State: 2, RemoteIP: "1.1.1.1"},
		},
	}
	svc := usecase.NewOrderService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, fakeAuto, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)

	if err := svc.ApproveOrder(context.Background(), 1, order.ID); err != nil {
		t.Fatalf("approve order: %v", err)
	}

	var inst domain.VPSInstance
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		itemRow, err := repo.ListOrderItems(context.Background(), order.ID)
		if err == nil && len(itemRow) > 0 {
			inst, err = repo.GetInstanceByOrderItem(context.Background(), itemRow[0].ID)
			if err == nil && inst.ID > 0 {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if inst.ID == 0 {
		t.Fatalf("expected vps instance created")
	}
}

func TestOrderService_ApproveProvisionFailure(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "prov2", "prov2@example.com", "pass")

	order := domain.Order{UserID: user.ID, OrderNo: "ORD-PROV-FAIL", Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	item := domain.OrderItem{
		OrderID:   order.ID,
		PackageID: seed.Package.ID,
		SystemID:  seed.SystemImage.ID,
		Amount:    1000,
		Status:    domain.OrderItemStatusPendingPayment,
		Action:    "create",
		SpecJSON:  "{}",
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{item}); err != nil {
		t.Fatalf("create item: %v", err)
	}

	fakeAuto := &testutil.FakeAutomationClient{
		CreateHostErr: context.DeadlineExceeded,
	}
	svc := usecase.NewOrderService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, fakeAuto, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)

	if err := svc.ApproveOrder(context.Background(), 1, order.ID); err != nil {
		t.Fatalf("approve order: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		items, err := repo.ListOrderItems(context.Background(), order.ID)
		if err == nil && len(items) > 0 {
			if items[0].Status == domain.OrderItemStatusFailed {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("expected item failed")
}
