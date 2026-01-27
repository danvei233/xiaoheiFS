package usecase_test

import (
	"context"
	"testing"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/usecase"
)

type fakeApprover struct {
	count int
}

func (f *fakeApprover) ApproveOrder(ctx context.Context, adminID int64, orderID int64) error {
	f.count++
	return nil
}

func TestPaymentService_Balance(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "wallet", "wallet@example.com", "pass")
	order := domain.Order{
		UserID:      user.ID,
		OrderNo:     "ORD-BAL",
		Status:      domain.OrderStatusPendingPayment,
		TotalAmount: 2000,
		Currency:    "CNY",
	}
	if err := repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{{
		OrderID:  order.ID,
		Amount:   2000,
		Status:   domain.OrderItemStatusPendingPayment,
		Action:   "create",
		SpecJSON: "{}",
	}}); err != nil {
		t.Fatalf("create items: %v", err)
	}
	if _, err := repo.AdjustWalletBalance(context.Background(), user.ID, 5000, "credit", "seed", 1, "init"); err != nil {
		t.Fatalf("seed wallet: %v", err)
	}

	reg := testutil.NewFakePaymentRegistry()
	approver := &fakeApprover{}
	svc := usecase.NewPaymentService(repo, repo, repo, reg, repo, approver, nil)

	res, err := svc.SelectPayment(context.Background(), user.ID, order.ID, usecase.PaymentSelectInput{Method: "balance"})
	if err != nil {
		t.Fatalf("select payment: %v", err)
	}
	if !res.Paid {
		t.Fatalf("expected paid")
	}
	wallet, err := repo.GetWallet(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("get wallet: %v", err)
	}
	if wallet.Balance >= 5000 {
		t.Fatalf("expected balance reduced")
	}
}

func TestPaymentService_HandleNotifyIdempotent(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "notify", "notify@example.com", "pass")
	order := domain.Order{
		UserID:      user.ID,
		OrderNo:     "ORD-NOTIFY",
		Status:      domain.OrderStatusPendingPayment,
		TotalAmount: 1000,
		Currency:    "CNY",
	}
	if err := repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{{
		OrderID:  order.ID,
		Amount:   1000,
		Status:   domain.OrderItemStatusPendingPayment,
		Action:   "create",
		SpecJSON: "{}",
	}}); err != nil {
		t.Fatalf("create items: %v", err)
	}
	payment := domain.OrderPayment{
		OrderID:  order.ID,
		UserID:   user.ID,
		Method:   "fake",
		Amount:   1000,
		Currency: "CNY",
		TradeNo:  "TN1",
		Status:   domain.PaymentStatusPendingPayment,
	}
	if err := repo.CreatePayment(context.Background(), &payment); err != nil {
		t.Fatalf("create payment: %v", err)
	}

	reg := testutil.NewFakePaymentRegistry()
	reg.RegisterProvider(&testutil.FakePaymentProvider{
		KeyVal:    "fake",
		NameVal:   "Fake",
		VerifyRes: usecase.PaymentNotifyResult{TradeNo: "TN1", Paid: true, Amount: 1000},
	}, true, "")
	approver := &fakeApprover{}
	svc := usecase.NewPaymentService(repo, repo, repo, reg, repo, approver, nil)

	if _, err := svc.HandleNotify(context.Background(), "fake", map[string]string{"trade_no": "TN1"}); err != nil {
		t.Fatalf("notify: %v", err)
	}
	if _, err := svc.HandleNotify(context.Background(), "fake", map[string]string{"trade_no": "TN1"}); err != nil {
		t.Fatalf("notify 2: %v", err)
	}
	updated, err := repo.GetPaymentByTradeNo(context.Background(), "TN1")
	if err != nil {
		t.Fatalf("get payment: %v", err)
	}
	if updated.Status != domain.PaymentStatusApproved {
		t.Fatalf("expected approved")
	}
	if approver.count != 1 {
		t.Fatalf("expected approver once, got %d", approver.count)
	}
}

func TestPaymentService_CustomProviderDisabled(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "custom", "custom@example.com", "pass")
	order := domain.Order{
		UserID:      user.ID,
		OrderNo:     "ORD-CUS",
		Status:      domain.OrderStatusPendingPayment,
		TotalAmount: 1000,
		Currency:    "CNY",
	}
	if err := repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{{
		OrderID:  order.ID,
		Amount:   1000,
		Status:   domain.OrderItemStatusPendingPayment,
		Action:   "create",
		SpecJSON: "{}",
	}}); err != nil {
		t.Fatalf("create items: %v", err)
	}

	reg := testutil.NewFakePaymentRegistry()
	reg.RegisterProvider(&testutil.FakePaymentProvider{KeyVal: "custom", NameVal: "Custom"}, false, `{"pay_url":"x"}`)
	svc := usecase.NewPaymentService(repo, repo, repo, reg, repo, nil, nil)

	if _, err := svc.SelectPayment(context.Background(), user.ID, order.ID, usecase.PaymentSelectInput{Method: "custom"}); err != usecase.ErrForbidden {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestPaymentService_PayWithProvider(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "prov", "prov@example.com", "pass")
	order := domain.Order{
		UserID:      user.ID,
		OrderNo:     "ORD-PROV",
		Status:      domain.OrderStatusPendingPayment,
		TotalAmount: 1500,
		Currency:    "CNY",
	}
	if err := repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{{
		OrderID:  order.ID,
		Amount:   1500,
		Status:   domain.OrderItemStatusPendingPayment,
		Action:   "create",
		SpecJSON: "{}",
	}}); err != nil {
		t.Fatalf("create items: %v", err)
	}
	reg := testutil.NewFakePaymentRegistry()
	reg.RegisterProvider(&testutil.FakePaymentProvider{
		KeyVal:    "fake",
		NameVal:   "Fake",
		CreateRes: usecase.PaymentCreateResult{PayURL: "https://pay.local", TradeNo: "TN-PROV"},
	}, true, "")
	svc := usecase.NewPaymentService(repo, repo, repo, reg, repo, nil, nil)

	res, err := svc.SelectPayment(context.Background(), user.ID, order.ID, usecase.PaymentSelectInput{Method: "fake"})
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if res.PayURL == "" || res.TradeNo == "" {
		t.Fatalf("expected pay url and trade no")
	}
	payment, err := repo.GetPaymentByTradeNo(context.Background(), "TN-PROV")
	if err != nil {
		t.Fatalf("get payment: %v", err)
	}
	if payment.Status != domain.PaymentStatusPendingPayment {
		t.Fatalf("expected pending payment")
	}
}

func TestPaymentService_ListProvidersAndMethods(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "prov2", "prov2@example.com", "pass")
	if _, err := repo.AdjustWalletBalance(context.Background(), user.ID, 30, "credit", "seed", 1, "init"); err != nil {
		t.Fatalf("seed wallet: %v", err)
	}
	reg := testutil.NewFakePaymentRegistry()
	reg.RegisterProvider(&testutil.FakePaymentProvider{KeyVal: "balance", NameVal: "Balance"}, true, "")
	reg.RegisterProvider(&testutil.FakePaymentProvider{KeyVal: "other", NameVal: "Other"}, false, "")
	svc := usecase.NewPaymentService(repo, repo, repo, reg, repo, nil, nil)

	if providers, err := svc.ListProviders(context.Background(), true); err != nil || len(providers) != 2 {
		t.Fatalf("list providers: %v %v", providers, err)
	}
	if methods, err := svc.ListUserMethods(context.Background(), user.ID); err != nil || len(methods) != 1 {
		t.Fatalf("list methods: %v %v", methods, err)
	}
	if methods, _ := svc.ListUserMethods(context.Background(), user.ID); methods[0].Balance == 0 {
		t.Fatalf("expected balance in method")
	}
}

func TestPaymentService_HandleNotifyErrors(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	reg := testutil.NewFakePaymentRegistry()
	reg.RegisterProvider(&testutil.FakePaymentProvider{
		KeyVal:    "bad",
		NameVal:   "Bad",
		VerifyRes: usecase.PaymentNotifyResult{TradeNo: "TN-404", Paid: true, Amount: 1000},
	}, true, "")
	svc := usecase.NewPaymentService(repo, repo, repo, reg, repo, nil, nil)
	if _, err := svc.HandleNotify(context.Background(), "bad", map[string]string{"trade_no": "TN-404"}); err == nil {
		t.Fatalf("expected missing payment error")
	}

	reg.RegisterProvider(&testutil.FakePaymentProvider{
		KeyVal:    "unpaid",
		NameVal:   "Unpaid",
		VerifyRes: usecase.PaymentNotifyResult{TradeNo: "TN-1", Paid: false, Amount: 1000},
	}, true, "")
	if _, err := svc.HandleNotify(context.Background(), "unpaid", map[string]string{"trade_no": "TN-1"}); err != usecase.ErrInvalidInput {
		t.Fatalf("expected invalid input, got %v", err)
	}
}
