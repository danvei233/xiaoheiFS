package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"xiaoheiplay/internal/domain"
)

type PaymentProviderInfo struct {
	Key        string
	Name       string
	Enabled    bool
	SchemaJSON string
	ConfigJSON string
}

type PaymentMethodInfo struct {
	Key        string
	Name       string
	SchemaJSON string
	ConfigJSON string
	Balance    int64
}

type PaymentSelectInput struct {
	Method    string
	ReturnURL string
	NotifyURL string
	Extra     map[string]string
}

type PaymentSelectResult struct {
	Method  string
	Status  string
	TradeNo string
	PayURL  string
	Extra   map[string]string
	Paid    bool
	Message string
	Balance int64
}

type PaymentService struct {
	orders   OrderRepository
	items    OrderItemRepository
	payments PaymentRepository
	registry PaymentProviderRegistry
	wallets  WalletRepository
	approver OrderApprover
	events   EventPublisher
}

func NewPaymentService(orders OrderRepository, items OrderItemRepository, payments PaymentRepository, registry PaymentProviderRegistry, wallets WalletRepository, approver OrderApprover, events EventPublisher) *PaymentService {
	return &PaymentService{
		orders:   orders,
		items:    items,
		payments: payments,
		registry: registry,
		wallets:  wallets,
		approver: approver,
		events:   events,
	}
}

func (s *PaymentService) ListProviders(ctx context.Context, includeDisabled bool) ([]PaymentProviderInfo, error) {
	if s.registry == nil {
		return nil, ErrInvalidInput
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

func (s *PaymentService) UpdateProvider(ctx context.Context, key string, enabled bool, configJSON string) error {
	if s.registry == nil {
		return ErrInvalidInput
	}
	return s.registry.UpdateProviderConfig(ctx, key, enabled, configJSON)
}

func (s *PaymentService) ListUserMethods(ctx context.Context, userID int64) ([]PaymentMethodInfo, error) {
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

func (s *PaymentService) SelectPayment(ctx context.Context, userID int64, orderID int64, input PaymentSelectInput) (PaymentSelectResult, error) {
	if input.Method == "" {
		return PaymentSelectResult{}, ErrInvalidInput
	}
	order, err := s.orders.GetOrder(ctx, orderID)
	if err != nil {
		return PaymentSelectResult{}, err
	}
	if order.UserID != userID {
		return PaymentSelectResult{}, ErrForbidden
	}
	if order.Status != domain.OrderStatusPendingPayment {
		return PaymentSelectResult{}, ErrConflict
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
	case "custom":
		return s.selectCustom(ctx, input.Method)
	case "balance":
		return s.payWithBalance(ctx, order)
	default:
		return s.payWithProvider(ctx, order, input)
	}
}

func (s *PaymentService) HandleNotify(ctx context.Context, providerKey string, req RawHTTPRequest) (PaymentNotifyResult, error) {
	if s.registry == nil || s.payments == nil {
		return PaymentNotifyResult{}, ErrInvalidInput
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
		return result, ErrInvalidInput
	}
	var payment domain.OrderPayment
	var lookupErr error
	// Prefer order_no+method correlation to avoid cross-order collisions when trade_no is reused/empty.
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
				return result, ErrConflict
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
		return result, ErrInvalidInput
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
		if err := s.ensurePendingReview(ctx, payment.OrderID); err != nil && err != ErrConflict {
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

func (s *PaymentService) selectCustom(ctx context.Context, method string) (PaymentSelectResult, error) {
	if s.registry == nil {
		return PaymentSelectResult{}, ErrInvalidInput
	}
	configJSON, enabled, err := s.registry.GetProviderConfig(ctx, method)
	if err != nil {
		return PaymentSelectResult{}, err
	}
	if !enabled {
		return PaymentSelectResult{}, ErrForbidden
	}
	return PaymentSelectResult{
		Method: method,
		Status: "manual",
		Extra:  map[string]string{"config_json": configJSON},
	}, nil
}

func (s *PaymentService) payWithBalance(ctx context.Context, order domain.Order) (PaymentSelectResult, error) {
	if s.wallets == nil || s.payments == nil {
		return PaymentSelectResult{}, ErrInvalidInput
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
	if err := s.ensurePendingReview(ctx, order.ID); err != nil && err != ErrConflict {
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

func (s *PaymentService) payWithProvider(ctx context.Context, order domain.Order, input PaymentSelectInput) (PaymentSelectResult, error) {
	if s.registry == nil || s.payments == nil {
		return PaymentSelectResult{}, ErrInvalidInput
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

func (s *PaymentService) ensurePendingReview(ctx context.Context, orderID int64) error {
	order, err := s.orders.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}
	if order.Status == domain.OrderStatusPendingReview {
		return ErrConflict
	}
	if order.Status != domain.OrderStatusPendingPayment {
		return ErrConflict
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
