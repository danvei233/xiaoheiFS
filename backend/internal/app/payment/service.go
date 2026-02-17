package payment

import (
	"context"
	"fmt"
	"strings"
	"time"

	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

type (
	PaymentProviderInfo  = appshared.PaymentProviderInfo
	PaymentMethodInfo    = appshared.PaymentMethodInfo
	PaymentSelectInput   = appshared.PaymentSelectInput
	PaymentSelectResult  = appshared.PaymentSelectResult
	PaymentCreateRequest = appshared.PaymentCreateRequest
	PaymentNotifyResult  = appshared.PaymentNotifyResult
	RawHTTPRequest       = appshared.RawHTTPRequest
)

type Service struct {
	orders   appports.OrderRepository
	items    appports.OrderItemRepository
	payments appports.PaymentRepository
	registry appports.PaymentProviderRegistry
	wallets  appports.WalletRepository
	approver appports.OrderApprover
	events   appports.EventPublisher
}

func NewService(orders appports.OrderRepository, items appports.OrderItemRepository, payments appports.PaymentRepository, registry appports.PaymentProviderRegistry, wallets appports.WalletRepository, approver appports.OrderApprover, events appports.EventPublisher) *Service {
	return &Service{
		orders:   orders,
		items:    items,
		payments: payments,
		registry: registry,
		wallets:  wallets,
		approver: approver,
		events:   events,
	}
}

func (s *Service) ListProviders(ctx context.Context, includeDisabled bool) ([]PaymentProviderInfo, error) {
	if s.registry == nil {
		return nil, appshared.ErrInvalidInput
	}
	providers, err := s.registry.ListProviders(ctx, includeDisabled)
	if err != nil {
		return nil, err
	}
	out := make([]PaymentProviderInfo, 0, len(providers))
	for _, provider := range providers {
		configJSON, enabled, _ := s.registry.GetProviderConfig(ctx, provider.Key())
		out = append(out, PaymentProviderInfo{
			Key:        provider.Key(),
			Name:       provider.Name(),
			Enabled:    enabled,
			SchemaJSON: provider.SchemaJSON(),
			ConfigJSON: configJSON,
		})
	}
	return out, nil
}

func (s *Service) UpdateProvider(ctx context.Context, key string, enabled bool, configJSON string) error {
	if s.registry == nil {
		return appshared.ErrInvalidInput
	}
	return s.registry.UpdateProviderConfig(ctx, key, enabled, configJSON)
}

func (s *Service) ListUserMethods(ctx context.Context, userID int64) ([]PaymentMethodInfo, error) {
	providers, err := s.ListProviders(ctx, false)
	if err != nil {
		return nil, err
	}
	var balance int64
	if s.wallets != nil {
		if wallet, err := s.wallets.GetWallet(ctx, userID); err == nil {
			balance = wallet.Balance
		}
	}
	out := make([]PaymentMethodInfo, 0, len(providers))
	for _, provider := range providers {
		info := PaymentMethodInfo{
			Key:        provider.Key,
			Name:       provider.Name,
			SchemaJSON: provider.SchemaJSON,
			ConfigJSON: provider.ConfigJSON,
		}
		if provider.Key == "balance" {
			info.Balance = balance
		}
		out = append(out, info)
	}
	return out, nil
}

func (s *Service) SelectPayment(ctx context.Context, userID int64, orderID int64, input PaymentSelectInput) (PaymentSelectResult, error) {
	if input.Method == "" {
		return PaymentSelectResult{}, appshared.ErrInvalidInput
	}
	order, err := s.orders.GetOrder(ctx, orderID)
	if err != nil {
		return PaymentSelectResult{}, err
	}
	if order.UserID != userID {
		return PaymentSelectResult{}, appshared.ErrForbidden
	}
	if order.Status != domain.OrderStatusPendingPayment {
		return PaymentSelectResult{}, appshared.ErrConflict
	}
	if order.TotalAmount <= 0 {
		return PaymentSelectResult{
			Method:  "none",
			Status:  "no_payment_required",
			Paid:    true,
			Message: "no payment required",
		}, nil
	}
	switch input.Method {
	case "approval":
		return PaymentSelectResult{
			Method:  input.Method,
			Status:  "manual",
			Message: "submit payment proof to /api/v1/orders/{id}/payments",
		}, nil
	case "balance":
		return s.payWithBalance(ctx, order)
	default:
		return s.payWithProvider(ctx, order, input)
	}
}

func (s *Service) HandleNotify(ctx context.Context, providerKey string, req RawHTTPRequest) (PaymentNotifyResult, error) {
	if s.registry == nil || s.payments == nil {
		return PaymentNotifyResult{}, appshared.ErrInvalidInput
	}
	provider, err := s.registry.GetProvider(ctx, providerKey)
	if err != nil {
		return PaymentNotifyResult{}, err
	}
	result, err := provider.VerifyNotify(ctx, req)
	if err != nil {
		return result, err
	}
	if !result.Paid {
		return result, appshared.ErrInvalidInput
	}
	var payment domain.OrderPayment
	var lookupErr error
	if s.orders != nil && strings.TrimSpace(result.OrderNo) != "" {
		order, oerr := s.orders.GetOrderByNo(ctx, result.OrderNo)
		if oerr == nil {
			items, perr := s.payments.ListPaymentsByOrder(ctx, order.ID)
			if perr == nil {
				var fallback *domain.OrderPayment
				for i := range items {
					if items[i].Method != providerKey {
						continue
					}
					if fallback == nil {
						fallback = &items[i]
					}
					if strings.TrimSpace(result.TradeNo) == "" || items[i].TradeNo == result.TradeNo || strings.TrimSpace(items[i].TradeNo) == "" {
						payment = items[i]
						break
					}
				}
				if payment.ID == 0 && fallback != nil {
					payment = *fallback
				}
			}
		}
	}
	if payment.ID == 0 && strings.TrimSpace(result.TradeNo) != "" {
		p, gerr := s.payments.GetPaymentByTradeNo(ctx, result.TradeNo)
		if gerr == nil {
			if p.Method != providerKey {
				return result, appshared.ErrConflict
			}
			payment = p
		} else {
			lookupErr = gerr
		}
	}
	if payment.ID == 0 {
		if lookupErr != nil {
			return result, lookupErr
		}
		return result, appshared.ErrInvalidInput
	}
	if strings.TrimSpace(result.TradeNo) != "" && payment.TradeNo != result.TradeNo {
		if uerr := s.payments.UpdatePaymentTradeNo(ctx, payment.ID, result.TradeNo); uerr == nil {
			payment.TradeNo = result.TradeNo
		}
	}
	if payment.Status != domain.PaymentStatusApproved {
		if err := s.payments.UpdatePaymentStatus(ctx, payment.ID, domain.PaymentStatusApproved, nil, ""); err != nil {
			return result, err
		}
		if err := s.ensurePendingReview(ctx, payment.OrderID); err != nil && err != appshared.ErrConflict {
			return result, err
		}
		if s.approver != nil {
			_ = s.approver.ApproveOrder(ctx, 0, payment.OrderID)
		}
		if s.events != nil {
			_, _ = s.events.Publish(ctx, payment.OrderID, "payment.confirmed", map[string]any{
				"method":   payment.Method,
				"trade_no": payment.TradeNo,
			})
		}
	}
	return result, nil
}

func (s *Service) payWithBalance(ctx context.Context, order domain.Order) (PaymentSelectResult, error) {
	if s.wallets == nil || s.payments == nil {
		return PaymentSelectResult{}, appshared.ErrInvalidInput
	}
	wallet, err := s.wallets.AdjustWalletBalance(ctx, order.UserID, -order.TotalAmount, "debit", "order", order.ID, "balance payment")
	if err != nil {
		return PaymentSelectResult{}, err
	}
	tradeNo := fmt.Sprintf("BAL-%d-%d", order.ID, time.Now().Unix())
	payment := domain.OrderPayment{
		OrderID:  order.ID,
		UserID:   order.UserID,
		Method:   "balance",
		Amount:   order.TotalAmount,
		Currency: order.Currency,
		TradeNo:  tradeNo,
		Status:   domain.PaymentStatusApproved,
	}
	if err := s.payments.CreatePayment(ctx, &payment); err != nil {
		return PaymentSelectResult{}, err
	}
	if err := s.ensurePendingReview(ctx, order.ID); err != nil && err != appshared.ErrConflict {
		return PaymentSelectResult{}, err
	}
	if s.approver != nil {
		_ = s.approver.ApproveOrder(ctx, 0, order.ID)
	}
	if s.events != nil {
		_, _ = s.events.Publish(ctx, order.ID, "payment.approved", map[string]any{
			"method":   "balance",
			"trade_no": tradeNo,
		})
	}
	return PaymentSelectResult{
		Method:  "balance",
		Status:  string(domain.PaymentStatusApproved),
		TradeNo: tradeNo,
		Paid:    true,
		Balance: wallet.Balance,
	}, nil
}

func (s *Service) payWithProvider(ctx context.Context, order domain.Order, input PaymentSelectInput) (PaymentSelectResult, error) {
	if s.registry == nil || s.payments == nil {
		return PaymentSelectResult{}, appshared.ErrInvalidInput
	}
	provider, err := s.registry.GetProvider(ctx, input.Method)
	if err != nil {
		return PaymentSelectResult{}, err
	}
	result, err := provider.CreatePayment(ctx, PaymentCreateRequest{
		OrderID:   order.ID,
		OrderNo:   order.OrderNo,
		UserID:    order.UserID,
		Amount:    order.TotalAmount,
		Currency:  order.Currency,
		Subject:   fmt.Sprintf("Order %s", order.OrderNo),
		ReturnURL: input.ReturnURL,
		NotifyURL: input.NotifyURL,
		Extra:     input.Extra,
	})
	if err != nil {
		return PaymentSelectResult{}, err
	}
	tradeNo := result.TradeNo
	if tradeNo == "" {
		tradeNo = fmt.Sprintf("PAY-%d-%d", order.ID, time.Now().Unix())
	}
	payment := domain.OrderPayment{
		OrderID:  order.ID,
		UserID:   order.UserID,
		Method:   input.Method,
		Amount:   order.TotalAmount,
		Currency: order.Currency,
		TradeNo:  tradeNo,
		Status:   domain.PaymentStatusPendingPayment,
	}
	if err := s.payments.CreatePayment(ctx, &payment); err != nil {
		return PaymentSelectResult{}, err
	}
	if s.events != nil {
		_, _ = s.events.Publish(ctx, order.ID, "payment.created", map[string]any{
			"method":   input.Method,
			"trade_no": tradeNo,
		})
	}
	return PaymentSelectResult{
		Method:  input.Method,
		Status:  string(domain.PaymentStatusPendingPayment),
		TradeNo: tradeNo,
		PayURL:  result.PayURL,
		Extra:   result.Extra,
	}, nil
}

func (s *Service) ensurePendingReview(ctx context.Context, orderID int64) error {
	order, err := s.orders.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}
	if order.Status == domain.OrderStatusPendingReview {
		return appshared.ErrConflict
	}
	if order.Status != domain.OrderStatusPendingPayment {
		return appshared.ErrConflict
	}
	order.Status = domain.OrderStatusPendingReview
	order.PendingReason = ""
	if err := s.orders.UpdateOrderMeta(ctx, order); err != nil {
		return err
	}
	items, _ := s.items.ListOrderItems(ctx, order.ID)
	for _, item := range items {
		_ = s.items.UpdateOrderItemStatus(ctx, item.ID, domain.OrderItemStatusPendingReview)
	}
	if s.events != nil {
		_, _ = s.events.Publish(ctx, order.ID, "order.pending_review", map[string]any{"status": order.Status})
	}
	return nil
}
