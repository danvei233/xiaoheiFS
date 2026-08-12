package payment_test

import (
	"context"
	"errors"
	"testing"
	repocore "xiaoheiplay/internal/adapter/repo/core"
	apppayment "xiaoheiplay/internal/app/payment"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

// cancelAwareOrderRepo wraps the real order repository and simulates a user
// cancel landing between the atomic balance payment and the post-payment
// status re-read inside payWithBalance.
type cancelAwareOrderRepo struct {
	*repocore.GormRepo
	calls int
}

func (c *cancelAwareOrderRepo) GetOrder(ctx context.Context, id int64) (domain.Order, error) {
	order, err := c.GormRepo.GetOrder(ctx, id)
	if err != nil {
		return order, err
	}
	c.calls++
	if c.calls == 2 {
		order.Status = domain.OrderStatusCanceled
	}
	return order, nil
}

type fakeApprover struct {
	count int
}

type sceneLookupErrorRegistry struct {
	*testutil.FakePaymentRegistry
}

type configLookupErrorRegistry struct {
	*testutil.FakePaymentRegistry
	updates int
}

func (r *configLookupErrorRegistry) GetProviderConfig(context.Context, string) (string, bool, error) {
	return "", false, errors.New("config storage unavailable")
}

func (r *configLookupErrorRegistry) UpdateProviderConfig(ctx context.Context, key string, enabled bool, configJSON string) error {
	r.updates++
	return r.FakePaymentRegistry.UpdateProviderConfig(ctx, key, enabled, configJSON)
}

func (r *sceneLookupErrorRegistry) GetProviderSceneEnabled(ctx context.Context, key, scene string) (bool, error) {
	return false, errors.New("scene lookup failed")
}

func (r *sceneLookupErrorRegistry) UpdateProviderSceneEnabled(ctx context.Context, key, scene string, enabled bool) error {
	return nil
}

func (f *fakeApprover) ApproveOrder(ctx context.Context, adminID int64, orderID int64) error {
	f.count++
	return nil
}

func TestPaymentService_ListProvidersByScene_FailCloseOnSceneLookupError(t *testing.T) {
	base := testutil.NewFakePaymentRegistry()
	base.RegisterProvider(&testutil.FakePaymentProvider{KeyVal: "fake", NameVal: "Fake"}, true, `{}`)
	reg := &sceneLookupErrorRegistry{FakePaymentRegistry: base}

	svc := apppayment.NewService(nil, nil, nil, reg, nil, nil, nil)
	providers, err := svc.ListProvidersByScene(context.Background(), false, apppayment.SceneWallet)
	if err != nil {
		t.Fatalf("list providers by scene: %v", err)
	}
	if len(providers) != 0 {
		t.Fatalf("expected provider filtered out when scene lookup fails")
	}
}

func TestPaymentService_UpdateProviderFailsClosedWhenExistingConfigCannotLoad(t *testing.T) {
	reg := &configLookupErrorRegistry{FakePaymentRegistry: testutil.NewFakePaymentRegistry()}
	svc := apppayment.NewService(nil, nil, nil, reg, nil, nil, nil)
	if err := svc.UpdateProvider(context.Background(), "fake", true, `{"secret":"********"}`); err == nil {
		t.Fatal("expected config lookup failure")
	}
	if reg.updates != 0 {
		t.Fatalf("masked config was persisted after lookup failure: updates=%d", reg.updates)
	}
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
	svc := apppayment.NewService(repo, repo, repo, reg, repo, approver, nil)

	res, err := svc.SelectPayment(context.Background(), user.ID, order.ID, appshared.PaymentSelectInput{Method: "balance"})
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
		VerifyRes: appshared.PaymentNotifyResult{TradeNo: "TN1", Paid: true, Amount: 1000},
	}, true, "")
	approver := &fakeApprover{}
	svc := apppayment.NewService(repo, repo, repo, reg, repo, approver, nil)

	raw := appshared.RawHTTPRequest{Method: "POST", Path: "/payments/notify/fake", RawQuery: "trade_no=TN1"}
	if _, err := svc.HandleNotify(context.Background(), "fake", raw); err != nil {
		t.Fatalf("notify: %v", err)
	}
	if _, err := svc.HandleNotify(context.Background(), "fake", raw); err != nil {
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

func TestPaymentService_BalancePay_CanceledOrderRefunded(t *testing.T) {
	_, base := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, base, "bal-cancel", "bal-cancel@example.com", "pass")
	order := domain.Order{
		UserID:      user.ID,
		OrderNo:     "ORD-BAL-CANCEL",
		Status:      domain.OrderStatusPendingPayment,
		TotalAmount: 2000,
		Currency:    "CNY",
	}
	if err := base.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	if err := base.CreateOrderItems(context.Background(), []domain.OrderItem{{
		OrderID:  order.ID,
		Amount:   2000,
		Status:   domain.OrderItemStatusPendingPayment,
		Action:   "create",
		SpecJSON: "{}",
	}}); err != nil {
		t.Fatalf("create items: %v", err)
	}
	if _, err := base.AdjustWalletBalance(context.Background(), user.ID, 5000, "credit", "seed", 1, "init"); err != nil {
		t.Fatalf("seed wallet: %v", err)
	}

	orders := &cancelAwareOrderRepo{GormRepo: base}
	reg := testutil.NewFakePaymentRegistry()
	svc := apppayment.NewService(orders, base, base, reg, base, nil, nil)

	if _, err := svc.SelectPayment(context.Background(), user.ID, order.ID, appshared.PaymentSelectInput{Method: "balance"}); err != appshared.ErrConflict {
		t.Fatalf("expected conflict for canceled order, got %v", err)
	}

	wallet, err := base.GetWallet(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("get wallet: %v", err)
	}
	if wallet.Balance != 5000 {
		t.Fatalf("expected balance refunded to 5000, got %d", wallet.Balance)
	}
	hasRefund, err := base.HasWalletTransaction(context.Background(), user.ID, "balance_refund", order.ID)
	if err != nil || !hasRefund {
		t.Fatalf("expected balance_refund transaction, has=%v err=%v", hasRefund, err)
	}
	payments, err := base.ListPaymentsByOrder(context.Background(), order.ID)
	if err != nil || len(payments) != 1 {
		t.Fatalf("list payments: len=%d err=%v", len(payments), err)
	}
	if payments[0].Status != domain.PaymentStatusRejected {
		t.Fatalf("expected payment rejected after refund, got %s", payments[0].Status)
	}
}

// TestPaymentService_ProviderSessionThenBalancePayDoesNotDoubleSettle locks
// the F2 invariant: a provider session created before a balance payment must
// never become a second approved payment. After the balance settles the order,
// a late provider callback is acked but the provider payment stays rejected, so
// the order has exactly one approved payment and the wallet is debited once.
func TestPaymentService_ProviderSessionThenBalancePayDoesNotDoubleSettle(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	user := testutil.CreateUser(t, repo, "doublepay", "doublepay@example.com", "pass")
	order := domain.Order{
		UserID:      user.ID,
		OrderNo:     "ORD-DOUBLE-PAY",
		Status:      domain.OrderStatusPendingPayment,
		TotalAmount: 2000,
		Currency:    "CNY",
	}
	if err := repo.CreateOrder(ctx, &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	if err := repo.CreateOrderItems(ctx, []domain.OrderItem{{
		OrderID:  order.ID,
		Amount:   2000,
		Status:   domain.OrderItemStatusPendingPayment,
		Action:   "create",
		SpecJSON: "{}",
	}}); err != nil {
		t.Fatalf("create items: %v", err)
	}
	if _, err := repo.AdjustWalletBalance(ctx, user.ID, 5000, "credit", "seed", 1, "init"); err != nil {
		t.Fatalf("seed wallet: %v", err)
	}

	// Step 1: user opens a provider payment session (order stays pending_payment).
	providerPayment := domain.OrderPayment{
		OrderID:  order.ID,
		UserID:   user.ID,
		Method:   "fake",
		Amount:   2000,
		Currency: "CNY",
		TradeNo:  "TN-PROVIDER",
		Status:   domain.PaymentStatusPendingPayment,
	}
	if err := repo.CreatePayment(ctx, &providerPayment); err != nil {
		t.Fatalf("create provider payment: %v", err)
	}

	// Step 2: the user pays with balance — order settles, balance payment approved.
	reg := testutil.NewFakePaymentRegistry()
	reg.RegisterProvider(&testutil.FakePaymentProvider{
		KeyVal:    "fake",
		NameVal:   "Fake",
		VerifyRes: appshared.PaymentNotifyResult{OrderNo: "ORD-DOUBLE-PAY", TradeNo: "TN-PROVIDER", Paid: true, Amount: 2000},
	}, true, "")
	approver := &fakeApprover{}
	svc := apppayment.NewService(repo, repo, repo, reg, repo, approver, nil)

	if res, err := svc.SelectPayment(ctx, user.ID, order.ID, appshared.PaymentSelectInput{Method: "balance"}); err != nil || !res.Paid {
		t.Fatalf("balance pay: res=%+v err=%v", res, err)
	}
	latest, err := repo.GetOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	// The stub approver does not advance the order state, so the durable handoff
	// is pending_review with an approved balance payment — the state the real
	// approver consumes. The critical invariant is that the order is out of
	// pending_payment and the balance payment is committed.
	if latest.Status == domain.OrderStatusPendingPayment {
		t.Fatalf("expected order moved out of pending_payment after balance pay, got %s", latest.Status)
	}

	// Step 3: the provider callback arrives late.
	if _, err := svc.HandleNotify(ctx, "fake", appshared.RawHTTPRequest{Method: "POST", Path: "/payments/notify/fake", RawQuery: "trade_no=TN-PROVIDER"}); err != nil {
		t.Fatalf("late provider notify: %v", err)
	}

	// Assertions: exactly one approved payment, wallet debited once.
	payments, err := repo.ListPaymentsByOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("list payments: %v", err)
	}
	approved := 0
	for _, p := range payments {
		if p.Status == domain.PaymentStatusApproved {
			approved++
		}
		if p.Method == "fake" && p.Status == domain.PaymentStatusApproved {
			t.Fatalf("provider payment must not be approved after balance settlement")
		}
	}
	if approved != 1 {
		t.Fatalf("expected exactly one approved payment, got %d", approved)
	}
	wallet, err := repo.GetWallet(ctx, user.ID)
	if err != nil {
		t.Fatalf("get wallet: %v", err)
	}
	if wallet.Balance != 3000 {
		t.Fatalf("expected wallet debited exactly once (balance 3000), got %d", wallet.Balance)
	}
}

// TestPaymentService_ProviderCallbackThenBalancePayRejected covers the reverse
// ordering: once the provider callback settles the order into pending_review /
// approved, a subsequent balance payment must be rejected (the CAS on
// pending_payment blocks it), not double-settle.
func TestPaymentService_ProviderCallbackThenBalancePayRejected(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	user := testutil.CreateUser(t, repo, "doublepay2", "doublepay2@example.com", "pass")
	order := domain.Order{
		UserID:      user.ID,
		OrderNo:     "ORD-DOUBLE-PAY-2",
		Status:      domain.OrderStatusPendingPayment,
		TotalAmount: 2000,
		Currency:    "CNY",
	}
	if err := repo.CreateOrder(ctx, &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	if err := repo.CreateOrderItems(ctx, []domain.OrderItem{{
		OrderID:  order.ID,
		Amount:   2000,
		Status:   domain.OrderItemStatusPendingPayment,
		Action:   "create",
		SpecJSON: "{}",
	}}); err != nil {
		t.Fatalf("create items: %v", err)
	}
	if _, err := repo.AdjustWalletBalance(ctx, user.ID, 5000, "credit", "seed", 1, "init"); err != nil {
		t.Fatalf("seed wallet: %v", err)
	}
	// The provider session exists before the callback (a real provider callback
	// only arrives after a payment session was created).
	providerPayment := domain.OrderPayment{
		OrderID:  order.ID,
		UserID:   user.ID,
		Method:   "fake",
		Amount:   2000,
		Currency: "CNY",
		TradeNo:  "TN-PROVIDER-2",
		Status:   domain.PaymentStatusPendingPayment,
	}
	if err := repo.CreatePayment(ctx, &providerPayment); err != nil {
		t.Fatalf("create provider payment: %v", err)
	}

	reg := testutil.NewFakePaymentRegistry()
	reg.RegisterProvider(&testutil.FakePaymentProvider{
		KeyVal:    "fake",
		NameVal:   "Fake",
		VerifyRes: appshared.PaymentNotifyResult{OrderNo: "ORD-DOUBLE-PAY-2", TradeNo: "TN-PROVIDER-2", Paid: true, Amount: 2000},
	}, true, "")
	approver := &fakeApprover{}
	svc := apppayment.NewService(repo, repo, repo, reg, repo, approver, nil)

	if _, err := svc.HandleNotify(ctx, "fake", appshared.RawHTTPRequest{Method: "POST", Path: "/payments/notify/fake", RawQuery: "trade_no=TN-PROVIDER-2"}); err != nil {
		t.Fatalf("provider notify: %v", err)
	}
	latest, err := repo.GetOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if latest.Status == domain.OrderStatusPendingPayment {
		t.Fatalf("expected order moved out of pending_payment after provider callback")
	}

	if _, err := svc.SelectPayment(ctx, user.ID, order.ID, appshared.PaymentSelectInput{Method: "balance"}); err != appshared.ErrConflict {
		t.Fatalf("expected balance pay rejected after provider settlement, got %v", err)
	}
	wallet, err := repo.GetWallet(ctx, user.ID)
	if err != nil {
		t.Fatalf("get wallet: %v", err)
	}
	if wallet.Balance != 5000 {
		t.Fatalf("expected wallet untouched after rejected balance pay, got %d", wallet.Balance)
	}
}

func TestPaymentService_HandleNotifyRejectsAmountMismatchWithoutStateChanges(t *testing.T) {
	tests := []struct {
		name   string
		amount int64
	}{
		{name: "underpayment", amount: 999},
		{name: "zero amount", amount: 0},
		{name: "other mismatch", amount: 1001},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, repo := testutil.NewTestDB(t, false)
			user := testutil.CreateUser(t, repo, "amount-check", "amount-check@example.com", "pass")
			order := domain.Order{
				UserID:      user.ID,
				OrderNo:     "ORD-AMOUNT-CHECK",
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
				TradeNo:  "TN-LOCAL",
				Status:   domain.PaymentStatusPendingPayment,
			}
			if err := repo.CreatePayment(context.Background(), &payment); err != nil {
				t.Fatalf("create payment: %v", err)
			}

			reg := testutil.NewFakePaymentRegistry()
			reg.RegisterProvider(&testutil.FakePaymentProvider{
				KeyVal:  "fake",
				NameVal: "Fake",
				VerifyRes: appshared.PaymentNotifyResult{
					OrderNo: "ORD-AMOUNT-CHECK",
					TradeNo: "TN-PROVIDER",
					Paid:    true,
					Amount:  tt.amount,
				},
			}, true, "")
			approver := &fakeApprover{}
			svc := apppayment.NewService(repo, repo, repo, reg, repo, approver, nil)

			if _, err := svc.HandleNotify(context.Background(), "fake", appshared.RawHTTPRequest{}); err != appshared.ErrPaymentAmountMismatch {
				t.Fatalf("expected amount mismatch error, got %v", err)
			}

			payments, err := repo.ListPaymentsByOrder(context.Background(), order.ID)
			if err != nil || len(payments) != 1 {
				t.Fatalf("list payments: len=%d err=%v", len(payments), err)
			}
			if payments[0].Status != domain.PaymentStatusPendingPayment {
				t.Fatalf("payment status changed: %s", payments[0].Status)
			}
			if payments[0].TradeNo != "TN-LOCAL" {
				t.Fatalf("payment trade number changed: %q", payments[0].TradeNo)
			}
			updatedOrder, err := repo.GetOrder(context.Background(), order.ID)
			if err != nil {
				t.Fatalf("get order: %v", err)
			}
			if updatedOrder.Status != domain.OrderStatusPendingPayment {
				t.Fatalf("order status changed: %s", updatedOrder.Status)
			}
			if approver.count != 0 {
				t.Fatalf("approver called %d times", approver.count)
			}
		})
	}
}

func TestPaymentService_HandleNotify_FallbackByOrderNo(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "notify2", "notify2@example.com", "pass")
	order := domain.Order{
		UserID:      user.ID,
		OrderNo:     "ORD-NOTIFY-2",
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
		Method:   "ezpay.wxpay",
		Amount:   1000,
		Currency: "CNY",
		TradeNo:  "TN-LOCAL",
		Status:   domain.PaymentStatusPendingPayment,
	}
	if err := repo.CreatePayment(context.Background(), &payment); err != nil {
		t.Fatalf("create payment: %v", err)
	}

	reg := testutil.NewFakePaymentRegistry()
	reg.RegisterProvider(&testutil.FakePaymentProvider{
		KeyVal:  "ezpay.wxpay",
		NameVal: "EZPay / wxpay",
		VerifyRes: appshared.PaymentNotifyResult{
			OrderNo: "ORD-NOTIFY-2",
			TradeNo: "TN-PROV",
			Paid:    true,
			Amount:  1000,
		},
	}, true, "")
	approver := &fakeApprover{}
	svc := apppayment.NewService(repo, repo, repo, reg, repo, approver, nil)

	raw := appshared.RawHTTPRequest{Method: "GET", Path: "/payments/notify/ezpay.wxpay", RawQuery: "out_trade_no=ORD-NOTIFY-2&trade_no=TN-PROV"}
	if _, err := svc.HandleNotify(context.Background(), "ezpay.wxpay", raw); err != nil {
		t.Fatalf("notify: %v", err)
	}
	updated, err := repo.GetPaymentByTradeNo(context.Background(), "TN-PROV")
	if err != nil {
		t.Fatalf("get payment by new trade_no: %v", err)
	}
	if updated.Status != domain.PaymentStatusApproved {
		t.Fatalf("expected approved")
	}
	if updated.Method != "ezpay.wxpay" {
		t.Fatalf("expected method preserved")
	}
}

func TestPaymentService_HandleNotify_DoesNotReviveRejectedPayment(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "notify-rejected", "notify-rejected@example.com", "pass")
	order := domain.Order{
		UserID:         user.ID,
		OrderNo:        "ORD-REJECTED",
		Status:         domain.OrderStatusRejected,
		TotalAmount:    1000,
		Currency:       "CNY",
		RejectedReason: "admin rejected",
	}
	if err := repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{{
		OrderID:  order.ID,
		Amount:   1000,
		Status:   domain.OrderItemStatusRejected,
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
		TradeNo:  "TN-REJ",
		Status:   domain.PaymentStatusRejected,
	}
	if err := repo.CreatePayment(context.Background(), &payment); err != nil {
		t.Fatalf("create payment: %v", err)
	}

	reg := testutil.NewFakePaymentRegistry()
	reg.RegisterProvider(&testutil.FakePaymentProvider{
		KeyVal:  "fake",
		NameVal: "Fake",
		VerifyRes: appshared.PaymentNotifyResult{
			OrderNo: "ORD-REJECTED",
			TradeNo: "TN-REJ",
			Paid:    true,
			Amount:  1000,
		},
	}, true, "")
	approver := &fakeApprover{}
	svc := apppayment.NewService(repo, repo, repo, reg, repo, approver, nil)

	raw := appshared.RawHTTPRequest{Method: "POST", Path: "/payments/notify/fake", RawQuery: "trade_no=TN-REJ"}
	if _, err := svc.HandleNotify(context.Background(), "fake", raw); err != nil {
		t.Fatalf("notify: %v", err)
	}

	updatedPayment, err := repo.GetPaymentByTradeNo(context.Background(), "TN-REJ")
	if err != nil {
		t.Fatalf("get payment: %v", err)
	}
	if updatedPayment.Status != domain.PaymentStatusRejected {
		t.Fatalf("rejected payment was revived to %s", updatedPayment.Status)
	}
	updatedOrder, err := repo.GetOrder(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if updatedOrder.Status != domain.OrderStatusRejected {
		t.Fatalf("rejected order changed to %s", updatedOrder.Status)
	}
	if approver.count != 0 {
		t.Fatalf("approver called %d times for rejected payment", approver.count)
	}
}

func TestPaymentService_HandleNotify_AmbiguousFallbackRejected(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "notify-ambig", "notify-ambig@example.com", "pass")
	order := domain.Order{
		UserID:      user.ID,
		OrderNo:     "ORD-AMBIG",
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
	p1 := domain.OrderPayment{OrderID: order.ID, UserID: user.ID, Method: "fake", Amount: 1000, Currency: "CNY", TradeNo: "TN-A", Status: domain.PaymentStatusPendingPayment}
	p2 := domain.OrderPayment{OrderID: order.ID, UserID: user.ID, Method: "fake", Amount: 1000, Currency: "CNY", TradeNo: "TN-B", Status: domain.PaymentStatusPendingPayment}
	if err := repo.CreatePayment(context.Background(), &p1); err != nil {
		t.Fatalf("create payment a: %v", err)
	}
	if err := repo.CreatePayment(context.Background(), &p2); err != nil {
		t.Fatalf("create payment b: %v", err)
	}

	reg := testutil.NewFakePaymentRegistry()
	reg.RegisterProvider(&testutil.FakePaymentProvider{
		KeyVal:  "fake",
		NameVal: "Fake",
		VerifyRes: appshared.PaymentNotifyResult{
			OrderNo: "ORD-AMBIG",
			TradeNo: "TN-C",
			Paid:    true,
			Amount:  1000,
		},
	}, true, "")
	approver := &fakeApprover{}
	svc := apppayment.NewService(repo, repo, repo, reg, repo, approver, nil)

	if _, err := svc.HandleNotify(context.Background(), "fake", appshared.RawHTTPRequest{}); err != appshared.ErrConflict {
		t.Fatalf("expected conflict for ambiguous callback, got %v", err)
	}
	payments, err := repo.ListPaymentsByOrder(context.Background(), order.ID)
	if err != nil || len(payments) != 2 {
		t.Fatalf("list payments: len=%d err=%v", len(payments), err)
	}
	for _, p := range payments {
		if p.Status != domain.PaymentStatusPendingPayment {
			t.Fatalf("payment %s was approved/changed to %s", p.TradeNo, p.Status)
		}
		if p.TradeNo == "TN-C" {
			t.Fatalf("payment trade_no was rewritten to callback value")
		}
	}
	if approver.count != 0 {
		t.Fatalf("approver called %d times", approver.count)
	}
}

func TestPaymentService_HandleNotify_OrderNoPreferredWhenTradeNoEmpty(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "notify3", "notify3@example.com", "pass")
	orderA := domain.Order{UserID: user.ID, OrderNo: "ORD-N-A", Status: domain.OrderStatusPendingReview, TotalAmount: 1000, Currency: "CNY"}
	orderB := domain.Order{UserID: user.ID, OrderNo: "ORD-N-B", Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(context.Background(), &orderA); err != nil {
		t.Fatalf("create order a: %v", err)
	}
	if err := repo.CreateOrder(context.Background(), &orderB); err != nil {
		t.Fatalf("create order b: %v", err)
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{{OrderID: orderA.ID, Amount: 1000, Status: domain.OrderItemStatusPendingReview, Action: "create", SpecJSON: "{}"}}); err != nil {
		t.Fatalf("create items a: %v", err)
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{{OrderID: orderB.ID, Amount: 1000, Status: domain.OrderItemStatusPendingPayment, Action: "create", SpecJSON: "{}"}}); err != nil {
		t.Fatalf("create items b: %v", err)
	}
	paymentA := domain.OrderPayment{OrderID: orderA.ID, UserID: user.ID, Method: "approval", Amount: 1000, Currency: "CNY", TradeNo: "", Status: domain.PaymentStatusPendingReview}
	paymentB := domain.OrderPayment{OrderID: orderB.ID, UserID: user.ID, Method: "ezpay.wxpay", Amount: 1000, Currency: "CNY", TradeNo: "", Status: domain.PaymentStatusPendingPayment}
	if err := repo.CreatePayment(context.Background(), &paymentA); err != nil {
		t.Fatalf("create payment a: %v", err)
	}
	if err := repo.CreatePayment(context.Background(), &paymentB); err != nil {
		t.Fatalf("create payment b: %v", err)
	}

	reg := testutil.NewFakePaymentRegistry()
	reg.RegisterProvider(&testutil.FakePaymentProvider{
		KeyVal:  "ezpay.wxpay",
		NameVal: "EZPay / wxpay",
		VerifyRes: appshared.PaymentNotifyResult{
			OrderNo: "ORD-N-B",
			TradeNo: "",
			Paid:    true,
			Amount:  1000,
		},
	}, true, "")
	approver := &fakeApprover{}
	svc := apppayment.NewService(repo, repo, repo, reg, repo, approver, nil)

	if _, err := svc.HandleNotify(context.Background(), "ezpay.wxpay", appshared.RawHTTPRequest{Method: "GET", Path: "/payments/notify/ezpay.wxpay"}); err != nil {
		t.Fatalf("notify: %v", err)
	}
	paysA, err := repo.ListPaymentsByOrder(context.Background(), orderA.ID)
	if err != nil || len(paysA) == 0 {
		t.Fatalf("list payments a: %v", err)
	}
	if paysA[0].Status == domain.PaymentStatusApproved {
		t.Fatalf("approval payment should not be touched by external notify")
	}
	paysB, err := repo.ListPaymentsByOrder(context.Background(), orderB.ID)
	if err != nil || len(paysB) == 0 {
		t.Fatalf("list payments b: %v", err)
	}
	if paysB[0].Status != domain.PaymentStatusApproved {
		t.Fatalf("expected order b payment approved")
	}
}

func TestPaymentService_DisabledProvider(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "disabledpay", "disabledpay@example.com", "pass")
	order := domain.Order{
		UserID:      user.ID,
		OrderNo:     "ORD-DISABLED-PAY",
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
	reg.RegisterProvider(&testutil.FakePaymentProvider{KeyVal: "disabled.pay", NameVal: "Disabled Pay"}, false, `{}`)
	svc := apppayment.NewService(repo, repo, repo, reg, repo, nil, nil)

	if _, err := svc.SelectPayment(context.Background(), user.ID, order.ID, appshared.PaymentSelectInput{Method: "disabled.pay"}); err != appshared.ErrForbidden {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestPaymentService_PayWithProvider_IsIdempotentPerOrder(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "prov-idem", "prov-idem@example.com", "pass")
	order := domain.Order{
		UserID:      user.ID,
		OrderNo:     "ORD-PROV-IDEM",
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
		CreateRes: appshared.PaymentCreateResult{PayURL: "https://pay.local", TradeNo: "TN-PROV"},
	}, true, "")
	svc := apppayment.NewService(repo, repo, repo, reg, repo, nil, nil)

	for i := 0; i < 2; i++ {
		res, err := svc.SelectPayment(context.Background(), user.ID, order.ID, appshared.PaymentSelectInput{Method: "fake"})
		if err != nil {
			t.Fatalf("select payment call %d: %v", i+1, err)
		}
		if res.TradeNo != "TN-PROV" {
			t.Fatalf("call %d: expected trade_no TN-PROV, got %q", i+1, res.TradeNo)
		}
	}
	payments, err := repo.ListPaymentsByOrder(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("list payments: %v", err)
	}
	if len(payments) != 1 {
		t.Fatalf("expected exactly one payment record after duplicate pay calls, got %d", len(payments))
	}
	if payments[0].Status != domain.PaymentStatusPendingPayment {
		t.Fatalf("expected pending payment, got %s", payments[0].Status)
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
		CreateRes: appshared.PaymentCreateResult{PayURL: "https://pay.local", TradeNo: "TN-PROV"},
	}, true, "")
	svc := apppayment.NewService(repo, repo, repo, reg, repo, nil, nil)

	res, err := svc.SelectPayment(context.Background(), user.ID, order.ID, appshared.PaymentSelectInput{Method: "fake"})
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

func TestPaymentService_CreateProviderPaymentByScene_RejectsInvalidRequests(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	reg := testutil.NewFakePaymentRegistry()
	reg.RegisterProvider(&testutil.FakePaymentProvider{
		KeyVal:    "fake",
		NameVal:   "Fake",
		CreateRes: appshared.PaymentCreateResult{PayURL: "https://pay.local", TradeNo: "TN-1"},
	}, true, "")
	svc := apppayment.NewService(repo, repo, repo, reg, repo, nil, nil)

	tests := []struct {
		name string
		req  appshared.PaymentCreateRequest
	}{
		{name: "zero amount", req: appshared.PaymentCreateRequest{OrderNo: "WALLET-ORDER-1", UserID: 1, Amount: 0, Currency: "CNY"}},
		{name: "negative amount", req: appshared.PaymentCreateRequest{OrderNo: "WALLET-ORDER-1", UserID: 1, Amount: -100, Currency: "CNY"}},
		{name: "zero user", req: appshared.PaymentCreateRequest{OrderNo: "WALLET-ORDER-1", UserID: 0, Amount: 100, Currency: "CNY"}},
		{name: "empty order no", req: appshared.PaymentCreateRequest{OrderNo: "  ", UserID: 1, Amount: 100, Currency: "CNY"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := svc.CreateProviderPaymentByScene(context.Background(), "wallet", "fake", tt.req); err != appshared.ErrInvalidInput {
				t.Fatalf("expected invalid input, got %v", err)
			}
		})
	}

	if _, err := svc.CreateProviderPaymentByScene(context.Background(), "wallet", "fake", appshared.PaymentCreateRequest{
		OrderNo: "WALLET-ORDER-1", UserID: 1, Amount: 100, Currency: "CNY",
	}); err != nil {
		t.Fatalf("valid request rejected: %v", err)
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
	svc := apppayment.NewService(repo, repo, repo, reg, repo, nil, nil)

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
		VerifyRes: appshared.PaymentNotifyResult{TradeNo: "TN-404", Paid: true, Amount: 1000},
	}, true, "")
	svc := apppayment.NewService(repo, repo, repo, reg, repo, nil, nil)
	if _, err := svc.HandleNotify(context.Background(), "bad", appshared.RawHTTPRequest{RawQuery: "trade_no=TN-404"}); err == nil {
		t.Fatalf("expected missing payment error")
	}

	reg.RegisterProvider(&testutil.FakePaymentProvider{
		KeyVal:    "unpaid",
		NameVal:   "Unpaid",
		VerifyRes: appshared.PaymentNotifyResult{TradeNo: "TN-1", Paid: false, Amount: 1000},
	}, true, "")
	if _, err := svc.HandleNotify(context.Background(), "unpaid", appshared.RawHTTPRequest{RawQuery: "trade_no=TN-1"}); err != appshared.ErrInvalidInput {
		t.Fatalf("expected invalid input, got %v", err)
	}
}
