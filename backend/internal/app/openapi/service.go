package openapi

import (
	"context"
	"errors"
	"time"

	apporder "xiaoheiplay/internal/app/order"
	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

type orderService interface {
	CreateOrderFromItems(ctx context.Context, userID int64, currency string, inputs []appshared.OrderItemInput, idemKey string, couponCode string) (domain.Order, []domain.OrderItem, error)
	CreateRenewOrder(ctx context.Context, userID int64, vpsID int64, renewDays int, durationMonths int) (domain.Order, error)
	CreateResizeOrder(ctx context.Context, userID int64, vpsID int64, spec *appshared.CartSpec, targetPackageID int64, resetAddons bool, scheduledAt *time.Time) (domain.Order, apporder.ResizeQuote, error)
	CreateRefundOrder(ctx context.Context, userID int64, vpsID int64, reason string) (domain.Order, int64, error)
}

type paymentService interface {
	SelectPayment(ctx context.Context, userID int64, orderID int64, input appshared.PaymentSelectInput) (appshared.PaymentSelectResult, error)
}

type Service struct {
	orders   orderService
	payments paymentService
	repo     appports.OrderRepository
}

func NewService(orders orderService, payments paymentService, repo appports.OrderRepository) *Service {
	return &Service{orders: orders, payments: payments, repo: repo}
}

func (s *Service) InstantCreate(ctx context.Context, userID int64, items []appshared.OrderItemInput, idemKey, couponCode string) (domain.Order, []domain.OrderItem, appshared.PaymentSelectResult, error) {
	order, orderItems, err := s.orders.CreateOrderFromItems(ctx, userID, "CNY", items, idemKey, couponCode)
	if err != nil {
		return domain.Order{}, nil, appshared.PaymentSelectResult{}, err
	}
	payRes, err := s.payBalanceOrRollback(ctx, userID, order)
	if err != nil {
		return domain.Order{}, nil, appshared.PaymentSelectResult{}, err
	}
	return order, orderItems, payRes, nil
}

func (s *Service) InstantRenew(ctx context.Context, userID, vpsID int64, renewDays, durationMonths int) (domain.Order, appshared.PaymentSelectResult, error) {
	order, err := s.orders.CreateRenewOrder(ctx, userID, vpsID, renewDays, durationMonths)
	if err != nil {
		return domain.Order{}, appshared.PaymentSelectResult{}, err
	}
	payRes, err := s.payBalanceOrRollback(ctx, userID, order)
	if err != nil {
		return domain.Order{}, appshared.PaymentSelectResult{}, err
	}
	return order, payRes, nil
}

func (s *Service) InstantResize(ctx context.Context, userID, vpsID int64, spec *appshared.CartSpec, targetPackageID int64, resetAddons bool, scheduledAt *time.Time) (domain.Order, apporder.ResizeQuote, appshared.PaymentSelectResult, error) {
	order, quote, err := s.orders.CreateResizeOrder(ctx, userID, vpsID, spec, targetPackageID, resetAddons, scheduledAt)
	if err != nil {
		return domain.Order{}, apporder.ResizeQuote{}, appshared.PaymentSelectResult{}, err
	}
	payRes, err := s.payBalanceOrRollback(ctx, userID, order)
	if err != nil {
		return domain.Order{}, apporder.ResizeQuote{}, appshared.PaymentSelectResult{}, err
	}
	return order, quote, payRes, nil
}

func (s *Service) InstantRefund(ctx context.Context, userID, vpsID int64, reason string) (domain.Order, int64, error) {
	order, amount, err := s.orders.CreateRefundOrder(ctx, userID, vpsID, reason)
	if err != nil {
		return domain.Order{}, 0, err
	}
	return order, amount, nil
}

func (s *Service) payBalanceOrRollback(ctx context.Context, userID int64, order domain.Order) (appshared.PaymentSelectResult, error) {
	if order.TotalAmount <= 0 {
		return appshared.PaymentSelectResult{Method: "balance", Paid: true, Status: "approved"}, nil
	}
	if s.payments == nil {
		return appshared.PaymentSelectResult{}, appshared.ErrInvalidInput
	}
	payRes, err := s.payments.SelectPayment(ctx, userID, order.ID, appshared.PaymentSelectInput{Method: "balance"})
	if err != nil {
		if errors.Is(err, appshared.ErrInsufficientBalance) && s.repo != nil {
			_ = s.repo.DeleteOrder(ctx, order.ID)
		}
		return appshared.PaymentSelectResult{}, err
	}
	return payRes, nil
}
