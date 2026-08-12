package order_test

import (
	"context"
	"testing"

	apporder "xiaoheiplay/internal/app/order"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

// TestOrderService_ApproveOrderOnlyApprovesReviewablePayments locks the
// end-to-end semantics of the repo-level fix (m030 M1): approving an order
// must upgrade only the admin-reviewable (pending_review) payment. A provider
// payment that never entered manual review (pending_payment) must NOT be
// force-approved, otherwise revenue reports count money that never arrived.
func TestOrderService_ApproveOrderOnlyApprovesReviewablePayments(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()

	user := testutil.CreateUser(t, repo, "approval-narrow", "approval-narrow@example.com", "pass")

	order := domain.Order{UserID: user.ID, OrderNo: "APPRV-NARROW", Status: domain.OrderStatusPendingReview, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	item := domain.OrderItem{OrderID: order.ID, SpecJSON: "{}", Qty: 1, Amount: 1000, Status: domain.OrderItemStatusPendingReview, AutomationInstanceID: "", Action: "create", DurationMonths: 1}
	if err := repo.CreateOrderItems(ctx, []domain.OrderItem{item}); err != nil {
		t.Fatalf("create order item: %v", err)
	}

	// A provider payment (e.g. created by payWithProvider) that never entered
	// manual review.
	providerPay := &domain.OrderPayment{
		OrderID: order.ID, UserID: user.ID, Method: "ezpay", Amount: 1000, Currency: "CNY",
		TradeNo: "PROVIDER-TXN-1", Status: domain.PaymentStatusPendingPayment, IdempotencyKey: "provider:<ezpay>:<orderID>",
	}
	if err := repo.CreatePayment(ctx, providerPay); err != nil {
		t.Fatalf("create provider payment: %v", err)
	}
	// A manual review payment (e.g. created by MarkPaid for offline payment).
	reviewPay := &domain.OrderPayment{
		OrderID: order.ID, UserID: user.ID, Method: "custom", Amount: 1000, Currency: "CNY",
		TradeNo: "MANUAL-TXN-1", Status: domain.PaymentStatusPendingReview, IdempotencyKey: "manual:<orderID>",
	}
	if err := repo.CreatePayment(ctx, reviewPay); err != nil {
		t.Fatalf("create review payment: %v", err)
	}

	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, &testutil.FakeAutomationResolver{Client: &testutil.FakeAutomationClient{}}, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
	if err := svc.ApproveOrder(ctx, 1, order.ID); err != nil {
		t.Fatalf("approve order: %v", err)
	}

	pays, err := repo.ListPaymentsByOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("list payments: %v", err)
	}
	var providerStatus, reviewStatus domain.PaymentStatus
	for _, p := range pays {
		switch p.TradeNo {
		case "PROVIDER-TXN-1":
			providerStatus = p.Status
		case "MANUAL-TXN-1":
			reviewStatus = p.Status
		}
	}
	if reviewStatus != domain.PaymentStatusApproved {
		t.Fatalf("expected manual review payment approved, got %q", reviewStatus)
	}
	if providerStatus != domain.PaymentStatusPendingPayment {
		t.Fatalf("provider payment must stay pending_payment after order approval, got %q", providerStatus)
	}
}
