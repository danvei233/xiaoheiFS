package payment_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	repocore "xiaoheiplay/internal/adapter/repo/core"
	apporder "xiaoheiplay/internal/app/order"
	apppayment "xiaoheiplay/internal/app/payment"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func newOrderServiceForPaymentTest(t *testing.T, repo *repocore.GormRepo) *apporder.OrderService {
	return apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, nil, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
}

// TestPaymentService_ConcurrentTwoChannelCallbacksSettleOnce locks F1: two
// provider sessions on the same order, both callbacks racing to approve. The
// DB-level partial unique index on (order_id) WHERE status='approved' is the
// backstop: exactly one payment may be approved per order regardless of
// interleaving.
func TestPaymentService_ConcurrentTwoChannelCallbacksSettleOnce(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	user := testutil.CreateUser(t, repo, "f1race", "f1race@example.com", "pass")
	order := domain.Order{UserID: user.ID, OrderNo: "ORD-F1-RACE", Status: domain.OrderStatusPendingPayment, TotalAmount: 2000, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &order); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateOrderItems(ctx, []domain.OrderItem{{
		OrderID: order.ID, Amount: 2000, Status: domain.OrderItemStatusPendingPayment, Action: "create", SpecJSON: "{}",
	}}); err != nil {
		t.Fatal(err)
	}
	payA := domain.OrderPayment{OrderID: order.ID, UserID: user.ID, Method: "alipay", Amount: 2000, Currency: "CNY", TradeNo: "TN-ALI", Status: domain.PaymentStatusPendingPayment}
	payB := domain.OrderPayment{OrderID: order.ID, UserID: user.ID, Method: "wechat", Amount: 2000, Currency: "CNY", TradeNo: "TN-WX", Status: domain.PaymentStatusPendingPayment}
	if err := repo.CreatePayment(ctx, &payA); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreatePayment(ctx, &payB); err != nil {
		t.Fatal(err)
	}

	reg := testutil.NewFakePaymentRegistry()
	reg.RegisterProvider(&testutil.FakePaymentProvider{KeyVal: "alipay", NameVal: "Ali", VerifyRes: appshared.PaymentNotifyResult{OrderNo: "ORD-F1-RACE", TradeNo: "TN-ALI", Paid: true, Amount: 2000}}, true, "")
	reg.RegisterProvider(&testutil.FakePaymentProvider{KeyVal: "wechat", NameVal: "Wx", VerifyRes: appshared.PaymentNotifyResult{OrderNo: "ORD-F1-RACE", TradeNo: "TN-WX", Paid: true, Amount: 2000}}, true, "")
	approver := &fakeApprover{}
	svc := apppayment.NewService(repo, repo, repo, reg, repo, approver, nil)

	tradeNo := map[string]string{"alipay": "TN-ALI", "wechat": "TN-WX"}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for method := range tradeNo {
		wg.Add(1)
		go func(method string) {
			defer wg.Done()
			_, err := svc.HandleNotify(ctx, method, appshared.RawHTTPRequest{Method: "POST", Path: "/payments/notify/" + method, RawQuery: "trade_no=" + tradeNo[method]})
			errs <- err
		}(method)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("notify err: %v", err)
		}
	}
	payments, _ := repo.ListPaymentsByOrder(ctx, order.ID)
	approved := 0
	rejected := 0
	for _, p := range payments {
		switch p.Status {
		case domain.PaymentStatusApproved:
			approved++
		case domain.PaymentStatusRejected:
			rejected++
		}
	}
	if approved != 1 {
		t.Fatalf("expected exactly one approved payment, got %d (%+v)", approved, payments)
	}
	if rejected != 1 {
		t.Fatalf("expected the losing session rejected, got rejected=%d (%+v)", rejected, payments)
	}
}

// TestPaymentService_CancelOrderThenProviderCallbackNotApproved locks F2: a
// user cancels an order that still has an in-flight provider session, then a
// valid signed callback arrives. The payment must NOT be approved (the user
// would pay for a dead order) and the order must stay canceled.
func TestPaymentService_CancelOrderThenProviderCallbackNotApproved(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	user := testutil.CreateUser(t, repo, "f2cancel", "f2cancel@example.com", "pass")
	order := domain.Order{UserID: user.ID, OrderNo: "ORD-F2-CANCEL", Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &order); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateOrderItems(ctx, []domain.OrderItem{{
		OrderID: order.ID, Amount: 1000, Status: domain.OrderItemStatusPendingPayment, Action: "create", SpecJSON: "{}",
	}}); err != nil {
		t.Fatal(err)
	}
	pay := domain.OrderPayment{OrderID: order.ID, UserID: user.ID, Method: "alipay", Amount: 1000, Currency: "CNY", TradeNo: "TN-CANCEL", Status: domain.PaymentStatusPendingPayment}
	if err := repo.CreatePayment(ctx, &pay); err != nil {
		t.Fatal(err)
	}

	orderSvc := newOrderServiceForPaymentTest(t, repo)
	if err := orderSvc.CancelOrder(ctx, user.ID, order.ID); err != nil {
		t.Fatalf("cancel order: %v", err)
	}
	latest, _ := repo.GetOrder(ctx, order.ID)
	if latest.Status != domain.OrderStatusCanceled {
		t.Fatalf("expected canceled order, got %s", latest.Status)
	}

	reg := testutil.NewFakePaymentRegistry()
	reg.RegisterProvider(&testutil.FakePaymentProvider{KeyVal: "alipay", NameVal: "Ali", VerifyRes: appshared.PaymentNotifyResult{OrderNo: "ORD-F2-CANCEL", TradeNo: "TN-CANCEL", Paid: true, Amount: 1000}}, true, "")
	svc := apppayment.NewService(repo, repo, repo, reg, repo, &fakeApprover{}, nil)
	if _, err := svc.HandleNotify(ctx, "alipay", appshared.RawHTTPRequest{Method: "POST", Path: "/payments/notify/alipay", RawQuery: "trade_no=TN-CANCEL"}); err != nil {
		t.Fatalf("notify err: %v", err)
	}
	payments, _ := repo.ListPaymentsByOrder(ctx, order.ID)
	for _, p := range payments {
		if p.Status == domain.PaymentStatusApproved {
			t.Fatalf("payment must not be approved on a canceled order")
		}
	}
	latest, _ = repo.GetOrder(ctx, order.ID)
	if latest.Status != domain.OrderStatusCanceled {
		t.Fatalf("order must stay canceled, got %s", latest.Status)
	}
}

// TestPaymentService_ConcurrentCancelAndCallbackDoesNotStrandMoney locks the
// pay-vs-cancel race (finance F1): a user cancels an order at the same moment a
// valid provider callback arrives. The invariant that must hold in every
// interleaving is: an order that ends canceled must have NO approved payment
// (real external money must not be stranded against a dead order without a
// refund), and conversely an approved payment must never coexist with a
// canceled order. The cancel path commits the order CAS, the NOT EXISTS(approved
// external payment) guard and the rejection of still-pending payments in one
// transaction (CancelOrderAtomic); a callback that wins the payment-approval CAS
// first trips the guard and the cancel is refused.
func TestPaymentService_ConcurrentCancelAndCallbackDoesNotStrandMoney(t *testing.T) {
	for i := 0; i < 10; i++ {
		_, repo := testutil.NewTestDB(t, false)
		ctx := context.Background()
		user := testutil.CreateUser(t, repo, fmt.Sprintf("f1cancel%d", i), fmt.Sprintf("f1cancel%d@example.com", i), "pass")
		order := domain.Order{UserID: user.ID, OrderNo: fmt.Sprintf("ORD-F1CAN-%d", i), Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "CNY"}
		if err := repo.CreateOrder(ctx, &order); err != nil {
			t.Fatalf("create order: %v", err)
		}
		if err := repo.CreateOrderItems(ctx, []domain.OrderItem{{
			OrderID: order.ID, Amount: 1000, Status: domain.OrderItemStatusPendingPayment, Action: "create", SpecJSON: "{}",
		}}); err != nil {
			t.Fatalf("create items: %v", err)
		}
		pay := domain.OrderPayment{OrderID: order.ID, UserID: user.ID, Method: "alipay", Amount: 1000, Currency: "CNY", TradeNo: fmt.Sprintf("TN-F1CAN-%d", i), Status: domain.PaymentStatusPendingPayment}
		if err := repo.CreatePayment(ctx, &pay); err != nil {
			t.Fatalf("create payment: %v", err)
		}

		reg := testutil.NewFakePaymentRegistry()
		reg.RegisterProvider(&testutil.FakePaymentProvider{KeyVal: "alipay", NameVal: "Ali", VerifyRes: appshared.PaymentNotifyResult{OrderNo: order.OrderNo, TradeNo: pay.TradeNo, Paid: true, Amount: 1000}}, true, "")
		svc := apppayment.NewService(repo, repo, repo, reg, repo, &fakeApprover{}, nil)
		orderSvc := newOrderServiceForPaymentTest(t, repo)

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = orderSvc.CancelOrder(ctx, user.ID, order.ID)
		}()
		go func() {
			defer wg.Done()
			_, _ = svc.HandleNotify(ctx, "alipay", appshared.RawHTTPRequest{Method: "POST", Path: "/payments/notify/alipay", RawQuery: "trade_no=" + pay.TradeNo})
		}()
		wg.Wait()

		latest, _ := repo.GetOrder(ctx, order.ID)
		payments, _ := repo.ListPaymentsByOrder(ctx, order.ID)
		if latest.Status == domain.OrderStatusCanceled {
			// Cancel won the race: the payment must be rejected, never approved.
			for _, p := range payments {
				if p.Status == domain.PaymentStatusApproved {
					t.Fatalf("payment approved on canceled order (stranded money): %+v (iter %d)", p, i)
				}
			}
		} else if latest.Status != domain.OrderStatusPendingReview && latest.Status != domain.OrderStatusPendingPayment {
			t.Fatalf("unexpected terminal order state: %s (iter %d)", latest.Status, i)
		}
		// A payment approved while the order is not canceled is legitimate
		// settlement (the callback won); the cancel must have been refused.
	}
}

// TestPaymentService_RejectedPaymentNotRevivedByCallback locks F5: an admin
// reject marks the payment rejected; a late (valid) callback must not flip it
// back to approved — the CAS on status IN (pending_payment, pending_review)
// makes the reject win.
func TestPaymentService_RejectedPaymentNotRevivedByCallback(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	user := testutil.CreateUser(t, repo, "f5rej", "f5rej@example.com", "pass")
	order := domain.Order{UserID: user.ID, OrderNo: "ORD-F5-REJ", Status: domain.OrderStatusPendingReview, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &order); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateOrderItems(ctx, []domain.OrderItem{{
		OrderID: order.ID, Amount: 1000, Status: domain.OrderItemStatusPendingReview, Action: "create", SpecJSON: "{}",
	}}); err != nil {
		t.Fatal(err)
	}
	pay := domain.OrderPayment{OrderID: order.ID, UserID: user.ID, Method: "alipay", Amount: 1000, Currency: "CNY", TradeNo: "TN-REJ", Status: domain.PaymentStatusPendingPayment}
	if err := repo.CreatePayment(ctx, &pay); err != nil {
		t.Fatal(err)
	}

	orderSvc := newOrderServiceForPaymentTest(t, repo)
	if err := orderSvc.RejectOrder(ctx, 1, order.ID, "fraud"); err != nil {
		t.Fatalf("reject order: %v", err)
	}
	latest, _ := repo.GetOrder(ctx, order.ID)
	if latest.Status != domain.OrderStatusRejected {
		t.Fatalf("expected rejected order, got %s", latest.Status)
	}

	reg := testutil.NewFakePaymentRegistry()
	reg.RegisterProvider(&testutil.FakePaymentProvider{KeyVal: "alipay", NameVal: "Ali", VerifyRes: appshared.PaymentNotifyResult{OrderNo: "ORD-F5-REJ", TradeNo: "TN-REJ", Paid: true, Amount: 1000}}, true, "")
	svc := apppayment.NewService(repo, repo, repo, reg, repo, &fakeApprover{}, nil)
	if _, err := svc.HandleNotify(ctx, "alipay", appshared.RawHTTPRequest{Method: "POST", Path: "/payments/notify/alipay", RawQuery: "trade_no=TN-REJ"}); err != nil {
		t.Fatalf("notify err: %v", err)
	}
	payments, _ := repo.ListPaymentsByOrder(ctx, order.ID)
	for _, p := range payments {
		if p.Status == domain.PaymentStatusApproved {
			t.Fatalf("payment must not be revived to approved after admin reject")
		}
	}
	latest, _ = repo.GetOrder(ctx, order.ID)
	if latest.Status == domain.OrderStatusApproved {
		t.Fatalf("order must not be revived to approved after admin reject")
	}
}

// TestPaymentService_UpdatePaymentStatusIfCurrent_CAS guards the repo-level CAS
// primitive the F5 fix relies on: only pending states can transition to
// approved, so a rejected payment is un-approvable.
func TestPaymentService_UpdatePaymentStatusIfCurrent_CAS(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	user := testutil.CreateUser(t, repo, "cas", "cas@example.com", "pass")
	order := domain.Order{UserID: user.ID, OrderNo: "ORD-CAS", Status: domain.OrderStatusPendingReview, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &order); err != nil {
		t.Fatal(err)
	}
	pay := domain.OrderPayment{OrderID: order.ID, UserID: user.ID, Method: "alipay", Amount: 1000, Currency: "CNY", TradeNo: "TN-CAS", Status: domain.PaymentStatusPendingPayment}
	if err := repo.CreatePayment(ctx, &pay); err != nil {
		t.Fatal(err)
	}
	// reject the payment first
	if err := repo.UpdatePaymentStatus(ctx, pay.ID, domain.PaymentStatusRejected, nil, "rejected"); err != nil {
		t.Fatal(err)
	}
	updated, err := repo.UpdatePaymentStatusIfCurrent(ctx, pay.ID, domain.PaymentStatusApproved,
		[]domain.PaymentStatus{domain.PaymentStatusPendingPayment, domain.PaymentStatusPendingReview}, nil, "")
	if err != nil {
		t.Fatalf("CAS err: %v", err)
	}
	if updated {
		t.Fatalf("rejected payment must not transition to approved via CAS")
	}
	got, err := repo.GetPaymentByTradeNo(ctx, "TN-CAS")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.PaymentStatusRejected {
		t.Fatalf("payment status must stay rejected, got %s", got.Status)
	}
}

// TestPaymentService_ConcurrentRejectAndCallbackDoesNotResurrect locks the F5
// race: an admin RejectOrder racing a valid provider callback. The callback's
// approve is a CAS on status IN (pending_payment, pending_review), and the
// ApproveOrder auto path refuses a rejected order — so whichever side commits
// first, the rejected order/payment is never resurrected to approved.
func TestPaymentService_ConcurrentRejectAndCallbackDoesNotResurrect(t *testing.T) {
	for i := 0; i < 10; i++ {
		_, repo := testutil.NewTestDB(t, false)
		ctx := context.Background()
		user := testutil.CreateUser(t, repo, fmt.Sprintf("f5conc%d", i), fmt.Sprintf("f5conc%d@example.com", i), "pass")
		order := domain.Order{UserID: user.ID, OrderNo: fmt.Sprintf("ORD-F5C-%d", i), Status: domain.OrderStatusPendingReview, TotalAmount: 1000, Currency: "CNY"}
		if err := repo.CreateOrder(ctx, &order); err != nil {
			t.Fatalf("create order: %v", err)
		}
		if err := repo.CreateOrderItems(ctx, []domain.OrderItem{{
			OrderID: order.ID, Amount: 1000, Status: domain.OrderItemStatusPendingReview, Action: "create", SpecJSON: "{}",
		}}); err != nil {
			t.Fatalf("create items: %v", err)
		}
		pay := domain.OrderPayment{OrderID: order.ID, UserID: user.ID, Method: "alipay", Amount: 1000, Currency: "CNY", TradeNo: fmt.Sprintf("TN-F5C-%d", i), Status: domain.PaymentStatusPendingPayment}
		if err := repo.CreatePayment(ctx, &pay); err != nil {
			t.Fatalf("create payment: %v", err)
		}

		reg := testutil.NewFakePaymentRegistry()
		reg.RegisterProvider(&testutil.FakePaymentProvider{KeyVal: "alipay", NameVal: "Ali", VerifyRes: appshared.PaymentNotifyResult{OrderNo: order.OrderNo, TradeNo: pay.TradeNo, Paid: true, Amount: 1000}}, true, "")
		svc := apppayment.NewService(repo, repo, repo, reg, repo, &fakeApprover{}, nil)
		orderSvc := newOrderServiceForPaymentTest(t, repo)

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = orderSvc.RejectOrder(ctx, 1, order.ID, "fraud")
		}()
		go func() {
			defer wg.Done()
			_, _ = svc.HandleNotify(ctx, "alipay", appshared.RawHTTPRequest{Method: "POST", Path: "/payments/notify/alipay", RawQuery: "trade_no=" + pay.TradeNo})
		}()
		wg.Wait()

		latest, _ := repo.GetOrder(ctx, order.ID)
		payments, _ := repo.ListPaymentsByOrder(ctx, order.ID)
		if latest.Status == domain.OrderStatusRejected {
			// The admin reject committed first. The callback must not have
			// overwritten the payment back to approved (the approve CAS on
			// status IN (pending_payment, pending_review) fails on a rejected
			// row), and must not have revived the order (ApproveOrder refuses
			// to auto-approve a rejected order).
			for _, p := range payments {
				if p.Status == domain.PaymentStatusApproved {
					t.Fatalf("payment must not be approved on a rejected order (resurrected): %s (iter %d)", p.Status, i)
				}
			}
		} else if latest.Status != domain.OrderStatusApproved && latest.Status != domain.OrderStatusPendingReview {
			t.Fatalf("unexpected terminal order state: %s (iter %d)", latest.Status, i)
		}
		// When the reject is blocked (a valid callback already settled the order
		// with real money), the order may be approved/pending_review with an
		// approved payment — that is legitimate settlement, not a resurrection.
	}
}
