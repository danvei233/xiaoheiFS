package usecase_test

import (
	"context"
	"testing"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/usecase"
)

func TestOrderService_RefreshOrderUpdates(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "refresh", "refresh@example.com", "pass")

	order := domain.Order{UserID: user.ID, OrderNo: "ORD-REFRESH", Status: domain.OrderStatusApproved, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	item := domain.OrderItem{
		OrderID:   order.ID,
		PackageID: seed.Package.ID,
		SystemID:  seed.SystemImage.ID,
		Amount:    1000,
		Status:    domain.OrderItemStatusApproved,
		Action:    "create",
		SpecJSON:  "{}",
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{item}); err != nil {
		t.Fatalf("create item: %v", err)
	}
	items, _ := repo.ListOrderItems(context.Background(), order.ID)
	inst := domain.VPSInstance{
		UserID:               user.ID,
		OrderItemID:          items[0].ID,
		AutomationInstanceID: "123",
		Name:                 "vm-refresh",
		Status:               domain.VPSStatusUnknown,
		SpecJSON:             "{}",
	}
	if err := repo.CreateInstance(context.Background(), &inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}

	fakeAuto := &testutil.FakeAutomationClient{
		HostInfo: map[int64]usecase.AutomationHostInfo{
			123: {HostID: 123, HostName: "vm-refresh", State: 2, RemoteIP: "1.1.1.1"},
		},
	}
	svc := usecase.NewOrderService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, fakeAuto, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
	updated, err := svc.RefreshOrder(context.Background(), user.ID, order.ID)
	if err != nil || len(updated) != 1 {
		t.Fatalf("refresh order: %v %d", err, len(updated))
	}
	if updated[0].Status != domain.VPSStatusRunning {
		t.Fatalf("expected running status")
	}
}

func TestOrderService_RetryProvisionConflict(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "retry", "retry@example.com", "pass")
	order := domain.Order{UserID: user.ID, OrderNo: "ORD-RETRY", Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	svc := usecase.NewOrderService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, nil, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
	if err := svc.RetryProvision(order.ID); err != usecase.ErrConflict {
		t.Fatalf("expected conflict, got %v", err)
	}
	if err := repo.UpdateOrderStatus(context.Background(), order.ID, domain.OrderStatusApproved); err != nil {
		t.Fatalf("update order: %v", err)
	}
	if err := svc.RetryProvision(order.ID); err != nil {
		t.Fatalf("retry provision: %v", err)
	}
}

func TestOrderService_RejectOrderWithPayment(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "reject", "reject@example.com", "pass")
	order := domain.Order{UserID: user.ID, OrderNo: "ORD-REJ", Status: domain.OrderStatusPendingReview, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	item := domain.OrderItem{OrderID: order.ID, Amount: 1000, Status: domain.OrderItemStatusPendingReview, Action: "create", SpecJSON: "{}"}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{item}); err != nil {
		t.Fatalf("create item: %v", err)
	}
	payment := domain.OrderPayment{OrderID: order.ID, UserID: user.ID, Method: "manual", Amount: 1000, Currency: "CNY", TradeNo: "TN-REJ", Status: domain.PaymentStatusPendingReview}
	if err := repo.CreatePayment(context.Background(), &payment); err != nil {
		t.Fatalf("create payment: %v", err)
	}

	svc := usecase.NewOrderService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, nil, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
	if err := svc.RejectOrder(context.Background(), 1, order.ID, "bad"); err != nil {
		t.Fatalf("reject order: %v", err)
	}
	updated, _ := repo.GetOrder(context.Background(), order.ID)
	if updated.Status != domain.OrderStatusRejected {
		t.Fatalf("expected rejected status")
	}
	pay, _ := repo.GetPaymentByTradeNo(context.Background(), "TN-REJ")
	if pay.Status != domain.PaymentStatusRejected {
		t.Fatalf("expected rejected payment")
	}
}

func TestPaymentService_UpdateProvider(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	reg := testutil.NewFakePaymentRegistry()
	svc := usecase.NewPaymentService(repo, repo, repo, reg, repo, nil, nil)
	if err := svc.UpdateProvider(context.Background(), "fake", true, `{"k":"v"}`); err != nil {
		t.Fatalf("update provider: %v", err)
	}
	if cfg, enabled, _ := reg.GetProviderConfig(context.Background(), "fake"); !enabled || cfg == "" {
		t.Fatalf("expected config updated")
	}
}
