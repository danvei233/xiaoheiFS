package openapi

import (
	"context"
	"errors"
	"testing"
	"time"

	apporder "xiaoheiplay/internal/app/order"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

type fakeOrderService struct {
	createFn func(ctx context.Context, userID int64, currency string, inputs []appshared.OrderItemInput, idemKey string, couponCode string) (domain.Order, []domain.OrderItem, error)
}

func (f *fakeOrderService) CreateOrderFromItems(ctx context.Context, userID int64, currency string, inputs []appshared.OrderItemInput, idemKey string, couponCode string) (domain.Order, []domain.OrderItem, error) {
	return f.createFn(ctx, userID, currency, inputs, idemKey, couponCode)
}
func (f *fakeOrderService) CreateRenewOrder(ctx context.Context, userID int64, vpsID int64, renewDays int, durationMonths int) (domain.Order, error) {
	return domain.Order{}, errors.New("not implemented")
}
func (f *fakeOrderService) CreateResizeOrder(ctx context.Context, userID int64, vpsID int64, spec *appshared.CartSpec, targetPackageID int64, resetAddons bool, scheduledAt *time.Time) (domain.Order, apporder.ResizeQuote, error) {
	return domain.Order{}, apporder.ResizeQuote{}, errors.New("not implemented")
}
func (f *fakeOrderService) CreateRefundOrder(ctx context.Context, userID int64, vpsID int64, reason string) (domain.Order, int64, error) {
	return domain.Order{}, 0, errors.New("not implemented")
}

type fakePaymentService struct {
	calls int
	fn    func(ctx context.Context, userID int64, orderID int64, input appshared.PaymentSelectInput) (appshared.PaymentSelectResult, error)
}

func (f *fakePaymentService) SelectPayment(ctx context.Context, userID int64, orderID int64, input appshared.PaymentSelectInput) (appshared.PaymentSelectResult, error) {
	f.calls++
	return f.fn(ctx, userID, orderID, input)
}

func TestService_InstantCreate_InsufficientBalanceRollback(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "oa_u1", "oa_u1@example.com", "pass")
	var createdOrderID int64

	orders := &fakeOrderService{createFn: func(ctx context.Context, userID int64, currency string, inputs []appshared.OrderItemInput, idemKey string, couponCode string) (domain.Order, []domain.OrderItem, error) {
		order := domain.Order{
			UserID:      userID,
			OrderNo:     "OA-ROLLBACK-1",
			Status:      domain.OrderStatusPendingPayment,
			TotalAmount: 500,
			Currency:    "CNY",
		}
		if err := repo.CreateOrder(ctx, &order); err != nil {
			return domain.Order{}, nil, err
		}
		createdOrderID = order.ID
		return order, nil, nil
	}}
	pay := &fakePaymentService{fn: func(ctx context.Context, userID int64, orderID int64, input appshared.PaymentSelectInput) (appshared.PaymentSelectResult, error) {
		return appshared.PaymentSelectResult{}, appshared.ErrInsufficientBalance
	}}
	svc := NewService(orders, pay, repo)

	_, _, _, err := svc.InstantCreate(context.Background(), user.ID, []appshared.OrderItemInput{{Qty: 1}}, "idem-open-1", "")
	if !errors.Is(err, appshared.ErrInsufficientBalance) {
		t.Fatalf("expected insufficient balance, got %v", err)
	}

	if _, err := repo.GetOrder(context.Background(), createdOrderID); err == nil {
		t.Fatalf("expected order rolled back (deleted)")
	}
}

func TestService_InstantCreate_ZeroAmountSkipsPayment(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "oa_u2", "oa_u2@example.com", "pass")

	orders := &fakeOrderService{createFn: func(ctx context.Context, userID int64, currency string, inputs []appshared.OrderItemInput, idemKey string, couponCode string) (domain.Order, []domain.OrderItem, error) {
		order := domain.Order{
			UserID:      userID,
			OrderNo:     "OA-ZERO-1",
			Status:      domain.OrderStatusApproved,
			TotalAmount: 0,
			Currency:    "CNY",
		}
		if err := repo.CreateOrder(ctx, &order); err != nil {
			return domain.Order{}, nil, err
		}
		return order, nil, nil
	}}
	pay := &fakePaymentService{fn: func(ctx context.Context, userID int64, orderID int64, input appshared.PaymentSelectInput) (appshared.PaymentSelectResult, error) {
		return appshared.PaymentSelectResult{Method: "balance", Paid: true, Status: "approved"}, nil
	}}
	svc := NewService(orders, pay, repo)

	_, _, payRes, err := svc.InstantCreate(context.Background(), user.ID, []appshared.OrderItemInput{{Qty: 1}}, "idem-open-2", "")
	if err != nil {
		t.Fatalf("instant create: %v", err)
	}
	if !payRes.Paid || payRes.Method != "balance" {
		t.Fatalf("expected paid by balance shortcut, got %+v", payRes)
	}
	if pay.calls != 0 {
		t.Fatalf("expected payment service not called for zero amount")
	}
}
