package payment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
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
	PaymentCreateResult  = appshared.PaymentCreateResult
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

const (
	SceneOrder  = "order"
	SceneWallet = "wallet"
)

type sceneAwareRegistry interface {
	GetProviderSceneEnabled(ctx context.Context, key, scene string) (bool, error)
	UpdateProviderSceneEnabled(ctx context.Context, key, scene string, enabled bool) error
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
	return s.ListProvidersByScene(ctx, includeDisabled, SceneOrder)
}

func normalizeScene(scene string) string {
	scene = strings.ToLower(strings.TrimSpace(scene))
	switch scene {
	case SceneWallet:
		return SceneWallet
	default:
		return SceneOrder
	}
}

func (s *Service) sceneEnabled(ctx context.Context, key, scene string) bool {
	scene = normalizeScene(scene)
	if sr, ok := s.registry.(sceneAwareRegistry); ok {
		enabled, err := sr.GetProviderSceneEnabled(ctx, key, scene)
		if err != nil {
			// Fail closed when scene state cannot be loaded.
			return false
		}
		return enabled
	}
	return true
}

func (s *Service) ListProvidersByScene(ctx context.Context, includeDisabled bool, scene string) ([]PaymentProviderInfo, error) {
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
		orderEnabled := s.sceneEnabled(ctx, provider.Key(), SceneOrder)
		walletEnabled := s.sceneEnabled(ctx, provider.Key(), SceneWallet)
		sceneEnabled := s.sceneEnabled(ctx, provider.Key(), scene)
		if !includeDisabled && !sceneEnabled {
			continue
		}
		out = append(out, PaymentProviderInfo{
			Key:           provider.Key(),
			Name:          provider.Name(),
			Enabled:       enabled && sceneEnabled,
			OrderEnabled:  orderEnabled,
			WalletEnabled: walletEnabled,
			SchemaJSON:    provider.SchemaJSON(),
			ConfigJSON:    configJSON,
		})
	}
	return out, nil
}

// ConfigSecretMask is returned in place of a payment-channel secret by the
// admin providers list. The update path treats it as "keep the existing value"
// so a read-modify-write round trip does not overwrite the secret (P2-13).
const ConfigSecretMask = "********"

// MergeConfigMask preserves existing secret values when an admin submits the
// masked config read from the list endpoint: any key whose value is the mask
// literal keeps the previously stored value instead of being overwritten.
func MergeConfigMask(existingJSON, incomingJSON string) (string, error) {
	incoming := strings.TrimSpace(incomingJSON)
	if incoming == "" {
		return incoming, nil
	}
	var in any
	if err := json.Unmarshal([]byte(incoming), &in); err != nil {
		return "", appshared.ErrInvalidInput
	}
	var old any
	_ = json.Unmarshal([]byte(existingJSON), &old)
	merged, changed, err := mergeConfigMasks(old, in)
	if err != nil {
		return "", err
	}
	if !changed {
		return incoming, nil
	}
	out, err := json.Marshal(merged)
	if err != nil {
		return "", fmt.Errorf("marshal payment config: %w", err)
	}
	return string(out), nil
}

func mergeConfigMasks(existing, incoming any) (any, bool, error) {
	switch value := incoming.(type) {
	case string:
		if value == ConfigSecretMask {
			if existing == nil {
				return nil, false, domain.ErrPaymentConfigMaskedSecretUnmatched
			}
			return existing, true, nil
		}
		return incoming, false, nil
	case map[string]any:
		old, _ := existing.(map[string]any)
		changed := false
		for key, child := range value {
			merged, childChanged, err := mergeConfigMasks(old[key], child)
			if err != nil {
				return nil, false, err
			}
			if childChanged {
				value[key] = merged
				changed = true
			}
		}
		return value, changed, nil
	case []any:
		old, _ := existing.([]any)
		changed := false
		for index, child := range value {
			oldChild := matchingConfigArrayItem(old, index, child)
			merged, childChanged, err := mergeConfigMasks(oldChild, child)
			if err != nil {
				return nil, false, err
			}
			if childChanged {
				value[index] = merged
				changed = true
			}
		}
		return value, changed, nil
	default:
		return incoming, false, nil
	}
}

// matchingConfigArrayItem avoids restoring a masked credential from the wrong
// account when an admin reorders a provider's account list before submitting
// it. Object arrays require a stable identity: falling back to the array index
// would silently move one account's credential to another account.
func matchingConfigArrayItem(old []any, index int, incoming any) any {
	incomingMap, ok := incoming.(map[string]any)
	if ok {
		for _, field := range []string{"id", "key", "code", "merchant_id", "merchantId", "account_id", "accountId"} {
			identity, exists := incomingMap[field]
			if !exists {
				continue
			}
			var matched any
			matches := 0
			for _, candidate := range old {
				candidateMap, ok := candidate.(map[string]any)
				if ok && reflect.DeepEqual(candidateMap[field], identity) {
					matched = candidate
					matches++
				}
			}
			if matches == 1 {
				return matched
			}
			return nil
		}
		// Names and positions are editable presentation data, not identities.
		// A masked credential without a stable identity cannot be restored
		// without risking assignment to a different account.
		return nil
	}
	if index < len(old) {
		return old[index]
	}
	return nil
}

func (s *Service) UpdateProvider(ctx context.Context, key string, enabled bool, configJSON string) error {
	if s.registry == nil {
		return appshared.ErrInvalidInput
	}
	// A masked config (read-modify-write from the admin list) must preserve the
	// previously stored secret values.
	merged := configJSON
	existing, _, err := s.registry.GetProviderConfig(ctx, key)
	if err != nil {
		// Updating a masked DTO without its source config would persist mask
		// placeholders as credentials. Storage/read failures must therefore stop
		// the update instead of silently treating it as a new provider.
		return fmt.Errorf("load payment provider config: %w", err)
	}
	merged, err = MergeConfigMask(existing, configJSON)
	if err != nil {
		return err
	}
	return s.registry.UpdateProviderConfig(ctx, key, enabled, merged)
}

func (s *Service) GetProviderSceneEnabled(ctx context.Context, key, scene string) (bool, error) {
	scene = normalizeScene(scene)
	if s.registry == nil {
		return false, appshared.ErrInvalidInput
	}
	if sr, ok := s.registry.(sceneAwareRegistry); ok {
		return sr.GetProviderSceneEnabled(ctx, key, scene)
	}
	return true, nil
}

func (s *Service) UpdateProviderSceneEnabled(ctx context.Context, key, scene string, enabled bool) error {
	scene = normalizeScene(scene)
	if s.registry == nil {
		return appshared.ErrInvalidInput
	}
	sr, ok := s.registry.(sceneAwareRegistry)
	if !ok {
		return appshared.ErrInvalidInput
	}
	return sr.UpdateProviderSceneEnabled(ctx, key, scene, enabled)
}

func (s *Service) ListUserMethods(ctx context.Context, userID int64) ([]PaymentMethodInfo, error) {
	return s.ListUserMethodsByScene(ctx, userID, SceneOrder)
}

func (s *Service) ListUserMethodsByScene(ctx context.Context, userID int64, scene string) ([]PaymentMethodInfo, error) {
	providers, err := s.ListProvidersByScene(ctx, false, scene)
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
		instructions, notice := publicProviderConfig(provider.ConfigJSON)
		info := PaymentMethodInfo{
			Key:          provider.Key,
			Name:         provider.Name,
			Instructions: instructions,
			Notice:       notice,
		}
		if provider.Key == "balance" {
			info.Balance = balance
		}
		out = append(out, info)
	}
	return out, nil
}

func publicProviderConfig(configJSON string) (string, string) {
	var config struct {
		Instructions string `json:"instructions"`
		Notice       string `json:"notice"`
	}
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return "", ""
	}
	return config.Instructions, config.Notice
}

func (s *Service) SelectPayment(ctx context.Context, userID int64, orderID int64, input PaymentSelectInput) (PaymentSelectResult, error) {
	return s.SelectPaymentByScene(ctx, SceneOrder, userID, orderID, input)
}

func (s *Service) SelectPaymentByScene(ctx context.Context, scene string, userID int64, orderID int64, input PaymentSelectInput) (PaymentSelectResult, error) {
	if input.Method == "" {
		return PaymentSelectResult{}, appshared.ErrInvalidInput
	}
	if !s.sceneEnabled(ctx, input.Method, scene) {
		return PaymentSelectResult{}, appshared.ErrForbidden
	}
	order, err := s.orders.GetOrder(ctx, orderID)
	if err != nil {
		return PaymentSelectResult{}, err
	}
	if order.UserID != userID {
		return PaymentSelectResult{}, appshared.ErrForbidden
	}
	if input.Method == "balance" {
		if order.Status == domain.OrderStatusRejected || order.Status == domain.OrderStatusCanceled {
			return PaymentSelectResult{}, appshared.ErrConflict
		}
	} else if order.Status != domain.OrderStatusPendingPayment {
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

func (s *Service) CreateProviderPaymentByScene(ctx context.Context, scene, method string, req PaymentCreateRequest) (PaymentCreateResult, error) {
	scene = normalizeScene(scene)
	method = strings.TrimSpace(method)
	if method == "" {
		return PaymentCreateResult{}, appshared.ErrInvalidInput
	}
	if method == "approval" || method == "balance" {
		return PaymentCreateResult{}, appshared.ErrInvalidInput
	}
	// Basic sanity guards: a provider session must reference a payer, a
	// positive amount and a stable order number. Deeper amount/ownership
	// checks are the caller's (handler) responsibility for now.
	if req.Amount <= 0 || req.UserID <= 0 || strings.TrimSpace(req.OrderNo) == "" {
		return PaymentCreateResult{}, appshared.ErrInvalidInput
	}
	if s.registry == nil {
		return PaymentCreateResult{}, appshared.ErrInvalidInput
	}
	if !s.sceneEnabled(ctx, method, scene) {
		return PaymentCreateResult{}, appshared.ErrForbidden
	}
	provider, err := s.registry.GetProvider(ctx, method)
	if err != nil {
		return PaymentCreateResult{}, err
	}
	return provider.CreatePayment(ctx, req)
}

func (s *Service) VerifyNotify(ctx context.Context, providerKey string, req RawHTTPRequest) (PaymentNotifyResult, error) {
	if s.registry == nil {
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
	return result, nil
}

func (s *Service) HandleNotify(ctx context.Context, providerKey string, req RawHTTPRequest) (PaymentNotifyResult, error) {
	if s.registry == nil || s.payments == nil || s.orders == nil {
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
				var unmatched int
				for i := range items {
					if items[i].Method != providerKey {
						continue
					}
					if strings.TrimSpace(result.TradeNo) == "" || items[i].TradeNo == result.TradeNo || strings.TrimSpace(items[i].TradeNo) == "" {
						payment = items[i]
						break
					}
					if fallback == nil {
						fallback = &items[i]
					}
					unmatched++
				}
				if payment.ID == 0 && fallback != nil {
					// A callback whose trade_no matches no local record can
					// only be safely attributed when there is exactly one
					// candidate session for this order+method; with several
					// candidates approving the latest one may credit the
					// wrong session.
					if unmatched == 1 {
						payment = *fallback
					} else if unmatched > 1 {
						return result, appshared.ErrConflict
					}
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
	order, err := s.orders.GetOrder(ctx, payment.OrderID)
	if err != nil {
		return result, err
	}
	if payment.Amount != order.TotalAmount {
		return result, appshared.ErrPaymentAmountMismatch
	}
	if err := s.ValidateNotifyAmount(order.TotalAmount, result.Amount); err != nil {
		return result, err
	}
	// A payment explicitly rejected by an admin must not be revived by a
	// channel retry callback (channels retry until they receive an ack).
	// Ack the callback without changing any payment or order state.
	if payment.Status == domain.PaymentStatusRejected {
		return result, nil
	}
	if strings.TrimSpace(result.TradeNo) != "" && payment.TradeNo != result.TradeNo {
		if uerr := s.payments.UpdatePaymentTradeNo(ctx, payment.ID, result.TradeNo); uerr == nil {
			payment.TradeNo = result.TradeNo
		}
	}
	if payment.Status != domain.PaymentStatusApproved {
		// A single order must settle exactly once. If the order already has an
		// approved payment (e.g. a balance payment settled after this provider
		// session was created), approving this one too would leave the user
		// double-charged with no automatic refund. Ack the callback so the
		// channel stops retrying, but do not mark the payment paid.
		if s.orderAlreadySettled(ctx, payment.OrderID, payment.ID) {
			// The order already settled via another payment (e.g. a balance
			// payment, or a sibling provider callback that won the race). Reject
			// this losing session so it is not left pending forever, then ack.
			_, _ = s.payments.UpdatePaymentStatusIfCurrent(ctx, payment.ID, domain.PaymentStatusRejected,
				[]domain.PaymentStatus{domain.PaymentStatusPendingPayment, domain.PaymentStatusPendingReview}, nil,
				"order already settled by another payment")
			return result, nil
		}
		// A callback must never settle an order that is no longer payable: an
		// order canceled/rejected by the user or an admin cannot be resurrected
		// by a late (or racing) provider callback, otherwise the user pays for a
		// dead order with no automatic refund. Ack the callback and leave the
		// payment untouched (CancelOrder/RejectOrder have already rejected it).
		switch order.Status {
		case domain.OrderStatusPendingPayment, domain.OrderStatusPendingReview:
		default:
			return result, nil
		}
		// Approve only from a still-pending payment state. The CAS on
		// status IN (pending_payment, pending_review) is what makes a
		// concurrent admin RejectOrderAtomic (which flips the payment to
		// rejected) win over this callback: once the row is rejected, this
		// update touches nothing and the callback acks. Combined with the
		// DB-level unique partial index on (order_id) WHERE status='approved',
		// a second concurrent callback can never turn into a second approved
		// payment for the same order.
		updated, err := s.payments.UpdatePaymentStatusIfCurrent(ctx, payment.ID, domain.PaymentStatusApproved,
			[]domain.PaymentStatus{domain.PaymentStatusPendingPayment, domain.PaymentStatusPendingReview}, nil, "")
		if err != nil {
			// A unique-constraint conflict means a sibling callback already
			// approved a payment for this order (order already settled) — ack.
			// Reject this losing session so it is not left pending forever.
			if errors.Is(err, appshared.ErrConflict) {
				_, _ = s.payments.UpdatePaymentStatusIfCurrent(ctx, payment.ID, domain.PaymentStatusRejected,
					[]domain.PaymentStatus{domain.PaymentStatusPendingPayment, domain.PaymentStatusPendingReview}, nil,
					"order already settled by another payment")
				return result, nil
			}
			return result, err
		}
		if !updated {
			return result, nil
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

// ValidateNotifyAmount checks a verified callback amount against the local
// amount, both represented in cents.
func (s *Service) ValidateNotifyAmount(expectedAmount, actualAmount int64) error {
	if expectedAmount <= 0 || actualAmount <= 0 || actualAmount != expectedAmount {
		return appshared.ErrPaymentAmountMismatch
	}
	return nil
}

func (s *Service) payWithBalance(ctx context.Context, order domain.Order) (PaymentSelectResult, error) {
	if s.payments == nil {
		return PaymentSelectResult{}, appshared.ErrInvalidInput
	}
	tradeNo := fmt.Sprintf("BAL-%d", order.ID)
	payment := domain.OrderPayment{
		OrderID:        order.ID,
		UserID:         order.UserID,
		Method:         "balance",
		Amount:         order.TotalAmount,
		Currency:       order.Currency,
		TradeNo:        tradeNo,
		Status:         domain.PaymentStatusApproved,
		IdempotencyKey: fmt.Sprintf("balance:%d", order.ID),
	}
	wallet, claimed, err := s.payments.PayOrderWithBalanceAtomic(ctx, order, &payment)
	if err != nil {
		return PaymentSelectResult{}, err
	}
	if payment.ID == 0 {
		return PaymentSelectResult{}, appshared.ErrConflict
	}
	latest, err := s.orders.GetOrder(ctx, order.ID)
	if err != nil {
		return PaymentSelectResult{}, err
	}
	if latest.Status == domain.OrderStatusRejected || latest.Status == domain.OrderStatusCanceled {
		// The order was canceled/rejected after the balance payment committed.
		// Refund the deducted amount, otherwise the user's funds stay stranded.
		if err := s.compensateBalancePayment(ctx, order, &payment); err != nil {
			return PaymentSelectResult{}, err
		}
		return PaymentSelectResult{}, appshared.ErrConflict
	}
	if latest.Status == domain.OrderStatusPendingReview && s.approver != nil {
		appErr := s.approver.ApproveOrder(ctx, 0, order.ID)
		if appErr != nil && !errors.Is(appErr, appshared.ErrConflict) {
			return PaymentSelectResult{}, appErr
		}
		if appErr != nil {
			// The approval was contested by a concurrent state change. If the
			// order is now canceled/rejected, refund the deducted balance.
			if current, gerr := s.orders.GetOrder(ctx, order.ID); gerr == nil &&
				(current.Status == domain.OrderStatusRejected || current.Status == domain.OrderStatusCanceled) {
				if cErr := s.compensateBalancePayment(ctx, order, &payment); cErr != nil {
					return PaymentSelectResult{}, cErr
				}
			}
		}
	}
	if claimed {
		// The order is now settled via balance. Any provider session or manual
		// proof created earlier for the same order must never become a second
		// approved payment — reject them so a late callback cannot double-charge
		// the user.
		s.rejectOtherPendingPayments(ctx, order.ID, payment.ID, "order settled via balance")
		if s.events != nil {
			_, _ = s.events.Publish(ctx, order.ID, "payment.approved", map[string]any{
				"method":   "balance",
				"trade_no": tradeNo,
			})
		}
	}
	return PaymentSelectResult{
		Method:  "balance",
		Status:  string(domain.PaymentStatusApproved),
		TradeNo: tradeNo,
		Paid:    true,
		Balance: wallet.Balance,
	}, nil
}

// orderAlreadySettled reports whether the order already has an approved payment
// other than the given payment id. Used by HandleNotify to enforce the
// "single approved payment per order" invariant before confirming a callback.
func (s *Service) orderAlreadySettled(ctx context.Context, orderID, selfID int64) bool {
	if s.payments == nil {
		return false
	}
	payments, err := s.payments.ListPaymentsByOrder(ctx, orderID)
	if err != nil {
		return false
	}
	for _, p := range payments {
		if p.ID == selfID {
			continue
		}
		if p.Status == domain.PaymentStatusApproved {
			return true
		}
	}
	return false
}

// rejectOtherPendingPayments marks every non-approved payment for an order
// (other than the one being kept) as rejected. Called when an order settles via
// balance so a pre-existing provider session / manual proof cannot be confirmed
// afterwards and double-charge the user.
func (s *Service) rejectOtherPendingPayments(ctx context.Context, orderID, keepID int64, reason string) {
	if s.payments == nil {
		return
	}
	payments, err := s.payments.ListPaymentsByOrder(ctx, orderID)
	if err != nil {
		return
	}
	for _, p := range payments {
		if p.ID == keepID {
			continue
		}
		switch p.Status {
		case domain.PaymentStatusPendingPayment, domain.PaymentStatusPendingReview:
			_ = s.payments.UpdatePaymentStatus(ctx, p.ID, domain.PaymentStatusRejected, nil, reason)
		}
	}
}

func (s *Service) payWithProvider(ctx context.Context, order domain.Order, input PaymentSelectInput) (PaymentSelectResult, error) {
	if s.registry == nil || s.payments == nil {
		return PaymentSelectResult{}, appshared.ErrInvalidInput
	}
	provider, err := s.registry.GetProvider(ctx, input.Method)
	if err != nil {
		return PaymentSelectResult{}, err
	}

	// Reuse an existing pending provider session for this order+method so
	// duplicate pay requests do not accumulate orphan payment records.
	idemKey := fmt.Sprintf("provider:%s:%d", input.Method, order.ID)
	existing, lookupErr := s.payments.GetPaymentByIdempotencyKey(ctx, order.ID, idemKey)
	reuse := lookupErr == nil && existing.Status == domain.PaymentStatusPendingPayment

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
		// Nanosecond resolution so repeated submissions within the same second
		// cannot collide on the unique trade_no column.
		tradeNo = fmt.Sprintf("PAY-%d-%d", order.ID, time.Now().UnixNano())
	}

	if reuse {
		// Refresh the existing record's trade_no to the current provider
		// session; the callback will then match this session.
		if existing.TradeNo != tradeNo {
			if uerr := s.payments.UpdatePaymentTradeNo(ctx, existing.ID, tradeNo); uerr == nil {
				existing.TradeNo = tradeNo
			}
		}
	} else {
		payment := domain.OrderPayment{
			OrderID:        order.ID,
			UserID:         order.UserID,
			Method:         input.Method,
			Amount:         order.TotalAmount,
			Currency:       order.Currency,
			TradeNo:        tradeNo,
			Status:         domain.PaymentStatusPendingPayment,
			IdempotencyKey: idemKey,
		}
		if err := s.payments.CreatePayment(ctx, &payment); err != nil {
			// A concurrent request may have created the same idempotency key
			// between the lookup and the insert. Return the existing provider
			// session (stable result) instead of a raw constraint 500.
			if errors.Is(err, appshared.ErrConflict) {
				existing, lerr := s.payments.GetPaymentByIdempotencyKey(ctx, order.ID, idemKey)
				if lerr == nil && existing.Method == input.Method && existing.Status == domain.PaymentStatusPendingPayment {
					return PaymentSelectResult{
						Method:  input.Method,
						Status:  string(domain.PaymentStatusPendingPayment),
						TradeNo: existing.TradeNo,
						PayURL:  result.PayURL,
						Extra:   result.Extra,
					}, nil
				}
			}
			return PaymentSelectResult{}, err
		}
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

// compensateBalancePayment refunds a balance debit that was committed for an
// order which ended up canceled/rejected. It is idempotent via the wallet
// transaction idempotency key so concurrent duplicate pay requests cannot
// double-credit the wallet, and marks the balance payment as rejected.
func (s *Service) compensateBalancePayment(ctx context.Context, order domain.Order, payment *domain.OrderPayment) error {
	if s.wallets == nil {
		return nil
	}
	const refType = "balance_refund"
	idemKey := fmt.Sprintf("balance_refund:%d", order.ID)
	if idem, ok := s.wallets.(interface {
		AdjustWalletBalanceIdempotent(context.Context, int64, int64, string, string, int64, string, string) (domain.Wallet, error)
	}); ok {
		if _, err := idem.AdjustWalletBalanceIdempotent(ctx, order.UserID, order.TotalAmount, "credit", refType, order.ID, "refund: order canceled after balance payment", idemKey); err != nil {
			return err
		}
	} else {
		if has, err := s.wallets.HasWalletTransaction(ctx, order.UserID, refType, order.ID); err == nil && has {
			return nil
		}
		if _, err := s.wallets.AdjustWalletBalance(ctx, order.UserID, order.TotalAmount, "credit", refType, order.ID, "refund: order canceled after balance payment"); err != nil {
			return err
		}
	}
	if payment != nil && payment.ID > 0 && s.payments != nil {
		_ = s.payments.UpdatePaymentStatus(ctx, payment.ID, domain.PaymentStatusRejected, nil, "order canceled after balance payment")
	}
	return nil
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
