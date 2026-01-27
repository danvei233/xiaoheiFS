package usecase_test

import (
	"context"
	"testing"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/usecase"
)

func TestOrderService_CreateOrderFromItems(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "buyer", "buyer@example.com", "pass")

	svc := usecase.NewOrderService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, nil, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
	order, items, err := svc.CreateOrderFromItems(context.Background(), user.ID, "CNY", []usecase.OrderItemInput{
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

	svc := usecase.NewOrderService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, nil, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
	payment, err := svc.SubmitPayment(context.Background(), user.ID, order.ID, usecase.PaymentInput{
		Method:   "manual",
		Amount:   1000,
		Currency: "CNY",
		TradeNo:  "T-1",
	}, "idem-1")
	if err != nil {
		t.Fatalf("submit payment: %v", err)
	}
	payment2, err := svc.SubmitPayment(context.Background(), user.ID, order.ID, usecase.PaymentInput{
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

func TestOrderService_CreateOrderFromCartAndCancel(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "cartuser", "cartuser@example.com", "pass")

	cartSvc := usecase.NewCartService(repo, repo, repo)
	if _, err := cartSvc.Add(context.Background(), user.ID, seed.Package.ID, seed.SystemImage.ID, usecase.CartSpec{}, 1); err != nil {
		t.Fatalf("add cart: %v", err)
	}

	svc := usecase.NewOrderService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, nil, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
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
