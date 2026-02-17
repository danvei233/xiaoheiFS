package order_test

import (
	"context"
	"testing"
	"time"
	appcart "xiaoheiplay/internal/app/cart"
	apporder "xiaoheiplay/internal/app/order"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func TestOrderService_CreateOrderFromItems(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "buyer", "buyer@example.com", "pass")

	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, nil, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
	order, items, err := svc.CreateOrderFromItems(context.Background(), user.ID, "CNY", []appshared.OrderItemInput{
		{PackageID: seed.Package.ID, SystemID: seed.SystemImage.ID, Qty: 1},
	}, "idem-1")
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	if order.ID == 0 || len(items) != 1 {
		t.Fatalf("expected order and one item")
	}
	if order.Status != domain.OrderStatusPendingPayment {
		t.Fatalf("expected pending payment")
	}
}

func TestOrderService_SubmitPaymentIdempotent(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "pay", "pay@example.com", "pass")
	order := domain.Order{
		UserID:      user.ID,
		OrderNo:     "ORD-100",
		Status:      domain.OrderStatusPendingPayment,
		TotalAmount: 1000,
		Currency:    "CNY",
	}
	if err := repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	item := domain.OrderItem{
		OrderID:  order.ID,
		Amount:   1000,
		Status:   domain.OrderItemStatusPendingPayment,
		Action:   "create",
		SpecJSON: "{}",
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{item}); err != nil {
		t.Fatalf("create items: %v", err)
	}

	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, nil, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
	payment, err := svc.SubmitPayment(context.Background(), user.ID, order.ID, appshared.PaymentInput{
		Method:   "manual",
		Amount:   1000,
		Currency: "CNY",
		TradeNo:  "T-1",
	}, "idem-1")
	if err != nil {
		t.Fatalf("submit payment: %v", err)
	}
	payment2, err := svc.SubmitPayment(context.Background(), user.ID, order.ID, appshared.PaymentInput{
		Method:   "manual",
		Amount:   1000,
		Currency: "CNY",
		TradeNo:  "T-1",
	}, "idem-1")
	if err != nil {
		t.Fatalf("submit payment 2: %v", err)
	}
	if payment.ID != payment2.ID {
		t.Fatalf("expected idempotent payment")
	}
}

func TestOrderService_SubmitPayment_RejectsCrossOrderTradeNoReuse(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "pay2", "pay2@example.com", "pass")
	orderA := domain.Order{UserID: user.ID, OrderNo: "ORD-A", Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "CNY"}
	orderB := domain.Order{UserID: user.ID, OrderNo: "ORD-B", Status: domain.OrderStatusPendingPayment, TotalAmount: 2000, Currency: "CNY"}
	if err := repo.CreateOrder(context.Background(), &orderA); err != nil {
		t.Fatalf("create order a: %v", err)
	}
	if err := repo.CreateOrder(context.Background(), &orderB); err != nil {
		t.Fatalf("create order b: %v", err)
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{{OrderID: orderA.ID, Amount: 1000, Status: domain.OrderItemStatusPendingPayment, Action: "create", SpecJSON: "{}"}}); err != nil {
		t.Fatalf("create items a: %v", err)
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{{OrderID: orderB.ID, Amount: 2000, Status: domain.OrderItemStatusPendingPayment, Action: "create", SpecJSON: "{}"}}); err != nil {
		t.Fatalf("create items b: %v", err)
	}
	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, nil, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
	if _, err := svc.SubmitPayment(context.Background(), user.ID, orderA.ID, appshared.PaymentInput{
		Method:   "approval",
		Amount:   1000,
		Currency: "CNY",
		TradeNo:  "TN-CROSS",
	}, "idem-a"); err != nil {
		t.Fatalf("submit payment a: %v", err)
	}
	if _, err := svc.SubmitPayment(context.Background(), user.ID, orderB.ID, appshared.PaymentInput{
		Method:   "approval",
		Amount:   2000,
		Currency: "CNY",
		TradeNo:  "TN-CROSS",
	}, "idem-b"); err != appshared.ErrConflict {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestOrderService_CreateOrderFromCartAndCancel(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "cartuser", "cartuser@example.com", "pass")

	cartSvc := appcart.NewService(repo, repo, repo)
	if _, err := cartSvc.Add(context.Background(), user.ID, seed.Package.ID, seed.SystemImage.ID, appshared.CartSpec{}, 1); err != nil {
		t.Fatalf("add cart: %v", err)
	}

	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, nil, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
	order, items, err := svc.CreateOrderFromCart(context.Background(), user.ID, "CNY", "idem-cart")
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("expected items")
	}

	if err := svc.CancelOrder(context.Background(), user.ID, order.ID); err != nil {
		t.Fatalf("cancel order: %v", err)
	}
	updated, err := repo.GetOrder(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if updated.Status != domain.OrderStatusCanceled {
		t.Fatalf("expected canceled")
	}
}

func TestOrderService_CreateRefundOrder_UsesInstanceMonthlyPrice(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "refundbase", "refundbase@example.com", "pass")

	baseOrder := domain.Order{
		UserID:      user.ID,
		OrderNo:     "ORD-REFUND-BASE-MP",
		Status:      domain.OrderStatusActive,
		TotalAmount: 200000,
		Currency:    "CNY",
	}
	if err := repo.CreateOrder(context.Background(), &baseOrder); err != nil {
		t.Fatalf("create base order: %v", err)
	}
	baseItem := domain.OrderItem{
		OrderID:   baseOrder.ID,
		PackageID: seed.Package.ID,
		SystemID:  seed.SystemImage.ID,
		Amount:    200000,
		Status:    domain.OrderItemStatusActive,
		Action:    "create",
		SpecJSON:  "{}",
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{baseItem}); err != nil {
		t.Fatalf("create base item: %v", err)
	}
	items, err := repo.ListOrderItems(context.Background(), baseOrder.ID)
	if err != nil || len(items) == 0 {
		t.Fatalf("list base items: %v", err)
	}
	inst := domain.VPSInstance{
		UserID:               user.ID,
		OrderItemID:          items[0].ID,
		AutomationInstanceID: "999",
		Name:                 "vm-refund-base",
		PackageID:            seed.Package.ID,
		PackageName:          seed.Package.Name,
		MonthlyPrice:         3000,
		SpecJSON:             "{}",
		Status:               domain.VPSStatusRunning,
		CreatedAt:            time.Now(),
	}
	expire := time.Now().Add(30 * 24 * time.Hour)
	inst.ExpireAt = &expire
	if err := repo.CreateInstance(context.Background(), &inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}

	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, nil, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
	refundOrder, amount, err := svc.CreateRefundOrder(context.Background(), user.ID, inst.ID, "test")
	if err != nil {
		t.Fatalf("create refund order: %v", err)
	}
	if amount != 3000 {
		t.Fatalf("expected refund amount based on monthly price 3000, got %d", amount)
	}
	if refundOrder.TotalAmount != -3000 {
		t.Fatalf("expected refund order total -3000, got %d", refundOrder.TotalAmount)
	}
}
