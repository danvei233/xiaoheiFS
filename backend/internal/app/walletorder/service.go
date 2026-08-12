package walletorder

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/pkg/money"
	"xiaoheiplay/internal/pkg/scrub"
)

type (
	WalletOrderCreateInput = appshared.WalletOrderCreateInput
)

type RefundCurvePoint struct {
	Percent int     `json:"percent"`
	Ratio   float64 `json:"ratio"`
}

type RefundPolicy struct {
	FullHours          int
	ProrateHours       int
	NoRefundHours      int
	FullDays           int
	ProrateDays        int
	NoRefundDays       int
	Curve              []RefundCurvePoint
	RequireApproval    bool
	AutoRefundOnDelete bool
}

type Service struct {
	orders     appports.WalletOrderRepository
	wallets    appports.WalletRepository
	settings   appports.SettingsRepository
	vps        appports.VPSRepository
	orderItems appports.OrderItemRepository
	automation appports.AutomationClientResolver
	audit      appports.AuditRepository
	userTiers  userTierAutoApprover
	cleanup    appports.RefundCleanupRepository

	// workerID uniquely identifies this service instance so cleanup job leases
	// are owned and fenced per process.
	workerID string
}

func NewService(orders appports.WalletOrderRepository, wallets appports.WalletRepository, settings appports.SettingsRepository, vps appports.VPSRepository, orderItems appports.OrderItemRepository, automation appports.AutomationClientResolver, audit appports.AuditRepository) *Service {
	return &Service{
		orders: orders, wallets: wallets, settings: settings, vps: vps, orderItems: orderItems,
		automation: automation, audit: audit,
		workerID: fmt.Sprintf("walletorder-svc-%d", time.Now().UnixNano()),
	}
}

// SetRefundCleanupRepo wires the durable cleanup outbox repository so refund
// approvals enqueue a delete job that a worker retries with bounded backoff.
func (s *Service) SetRefundCleanupRepo(repo appports.RefundCleanupRepository) {
	s.cleanup = repo
}

type userTierAutoApprover interface {
	TryAutoApproveForUser(ctx context.Context, userID int64, reason string) error
}

func (s *Service) SetUserTierAutoApprover(approver userTierAutoApprover) {
	s.userTiers = approver
}

func (s *Service) CreateRefundOrder(ctx context.Context, userID int64, amount int64, note string, meta map[string]any) (domain.WalletOrder, error) {
	if userID == 0 || amount <= 0 {
		return domain.WalletOrder{}, appshared.ErrInvalidInput
	}
	trimmedNote, err := trimAndValidateOptional(note, maxLenRefundReason)
	if err != nil {
		return domain.WalletOrder{}, appshared.ErrInvalidInput
	}
	order := domain.WalletOrder{UserID: userID, Type: domain.WalletOrderRefund, Amount: amount, Currency: "CNY", Status: domain.WalletOrderPendingReview, Note: trimmedNote, MetaJSON: toJSON(meta)}
	if err := s.orders.CreateWalletOrder(ctx, &order); err != nil {
		return domain.WalletOrder{}, err
	}
	return order, nil
}

func (s *Service) CreateRecharge(ctx context.Context, userID int64, input WalletOrderCreateInput) (domain.WalletOrder, error) {
	if userID == 0 || input.Amount <= 0 || input.Amount > maxWalletOrderAmountCents {
		return domain.WalletOrder{}, appshared.ErrInvalidInput
	}
	trimmedNote, err := trimAndValidateOptional(input.Note, maxLenPaymentNote)
	if err != nil {
		return domain.WalletOrder{}, appshared.ErrInvalidInput
	}
	currency := strings.TrimSpace(input.Currency)
	if currency == "" {
		currency = "CNY"
	}
	if currency != "CNY" {
		return domain.WalletOrder{}, appshared.ErrInvalidInput
	}
	order := domain.WalletOrder{UserID: userID, Type: domain.WalletOrderRecharge, Amount: input.Amount, Currency: currency, Status: domain.WalletOrderPendingReview, Note: trimmedNote, MetaJSON: toJSON(input.Meta)}
	if err := s.orders.CreateWalletOrder(ctx, &order); err != nil {
		return domain.WalletOrder{}, err
	}
	return order, nil
}

func (s *Service) CreateWithdraw(ctx context.Context, userID int64, input WalletOrderCreateInput) (domain.WalletOrder, error) {
	if userID == 0 || input.Amount <= 0 || input.Amount > maxWalletOrderAmountCents {
		return domain.WalletOrder{}, appshared.ErrInvalidInput
	}
	trimmedNote, err := trimAndValidateOptional(input.Note, maxLenPaymentNote)
	if err != nil {
		return domain.WalletOrder{}, appshared.ErrInvalidInput
	}
	currency := strings.TrimSpace(input.Currency)
	if currency == "" {
		currency = "CNY"
	}
	if currency != "CNY" {
		return domain.WalletOrder{}, appshared.ErrInvalidInput
	}
	if s.wallets == nil {
		return domain.WalletOrder{}, appshared.ErrInvalidInput
	}
	wallet, err := s.wallets.GetWallet(ctx, userID)
	if err != nil {
		return domain.WalletOrder{}, err
	}
	if wallet.Balance < input.Amount {
		return domain.WalletOrder{}, appshared.ErrInsufficientBalance
	}
	order := domain.WalletOrder{UserID: userID, Type: domain.WalletOrderWithdraw, Amount: input.Amount, Currency: currency, Status: domain.WalletOrderPendingReview, Note: trimmedNote, MetaJSON: toJSON(input.Meta)}
	if err := s.orders.CreateWalletOrder(ctx, &order); err != nil {
		return domain.WalletOrder{}, err
	}
	return order, nil
}

func (s *Service) ListUserOrders(ctx context.Context, userID int64, limit, offset int) ([]domain.WalletOrder, int, error) {
	return s.orders.ListWalletOrdersFiltered(ctx, userID, "", "", limit, offset)
}

// ListUserOrdersFiltered 支持 status / type 服务端过滤（cross-module residual）。
func (s *Service) ListUserOrdersFiltered(ctx context.Context, userID int64, status, orderType string, limit, offset int) ([]domain.WalletOrder, int, error) {
	return s.orders.ListWalletOrdersFiltered(ctx, userID, status, orderType, limit, offset)
}

// SummarizeUserOrders returns database aggregates for the current calendar
// month and all-time counts without loading every order.
func (s *Service) SummarizeUserOrders(ctx context.Context, userID int64, now time.Time) (domain.WalletOrderSummary, error) {
	if userID <= 0 {
		return domain.WalletOrderSummary{}, appshared.ErrInvalidInput
	}
	localNow := now.In(time.Local)
	monthStart := time.Date(localNow.Year(), localNow.Month(), 1, 0, 0, 0, 0, localNow.Location())
	return s.orders.SummarizeWalletOrders(ctx, userID, monthStart, monthStart.AddDate(0, 1, 0))
}

func (s *Service) ListAllOrders(ctx context.Context, status string, limit, offset int) ([]domain.WalletOrder, int, error) {
	return s.orders.ListAllWalletOrders(ctx, status, limit, offset)
}

func (s *Service) GetUserOrder(ctx context.Context, userID, orderID int64) (domain.WalletOrder, error) {
	if userID == 0 || orderID == 0 {
		return domain.WalletOrder{}, appshared.ErrInvalidInput
	}
	order, err := s.orders.GetWalletOrder(ctx, orderID)
	if err != nil {
		return domain.WalletOrder{}, err
	}
	if order.UserID != userID {
		return domain.WalletOrder{}, appshared.ErrForbidden
	}
	return order, nil
}

// GetOrderByID returns a wallet order without an ownership check. It is used by
// payment-callback handlers that resolve the order from the server-derived
// WALLET-ORDER-<id> order number, avoiding a paginated scan.
func (s *Service) GetOrderByID(ctx context.Context, orderID int64) (domain.WalletOrder, error) {
	if orderID <= 0 {
		return domain.WalletOrder{}, appshared.ErrInvalidInput
	}
	return s.orders.GetWalletOrder(ctx, orderID)
}

// GetOrderByPaymentTradeNo resolves a recharge wallet order by the provider
// trade_no stored in its meta (P1-4): the slow notify path must not scan
// bounded pages that truncate deep orders.
func (s *Service) GetOrderByPaymentTradeNo(ctx context.Context, tradeNo string) (domain.WalletOrder, error) {
	tradeNo = strings.TrimSpace(tradeNo)
	if tradeNo == "" {
		return domain.WalletOrder{}, appshared.ErrInvalidInput
	}
	return s.orders.GetWalletOrderByPaymentTradeNo(ctx, tradeNo)
}

func (s *Service) UpdateOrderMeta(ctx context.Context, orderID int64, metaJSON string) error {
	return s.orders.UpdateWalletOrderMeta(ctx, orderID, metaJSON)
}

func (s *Service) CancelByUser(ctx context.Context, userID, orderID int64, reason string) (domain.WalletOrder, error) {
	reason, err := trimAndValidateOptional(reason, maxLenReviewReason)
	if err != nil {
		return domain.WalletOrder{}, appshared.ErrInvalidInput
	}
	order, err := s.GetUserOrder(ctx, userID, orderID)
	if err != nil {
		return domain.WalletOrder{}, err
	}
	if order.Type != domain.WalletOrderRecharge && order.Type != domain.WalletOrderWithdraw && order.Type != domain.WalletOrderRefund {
		return domain.WalletOrder{}, appshared.ErrInvalidInput
	}
	if order.Status != domain.WalletOrderPendingReview {
		return domain.WalletOrder{}, appshared.ErrConflict
	}
	if reason == "" {
		reason = "user_cancel"
	}
	updated, err := s.orders.UpdateWalletOrderStatusIfCurrent(ctx, order.ID, domain.WalletOrderPendingReview, domain.WalletOrderRejected, nil, reason)
	if err != nil {
		return domain.WalletOrder{}, err
	}
	if !updated {
		return domain.WalletOrder{}, appshared.ErrConflict
	}
	order.Status = domain.WalletOrderRejected
	order.ReviewReason = reason
	order.ReviewedBy = nil
	return order, nil
}

func (s *Service) RequestRefund(ctx context.Context, userID int64, vpsID int64, reason string) (domain.WalletOrder, *domain.Wallet, error) {
	if userID == 0 || vpsID == 0 {
		return domain.WalletOrder{}, nil, appshared.ErrInvalidInput
	}
	reason, err := trimAndValidateRequired(reason, maxLenRefundReason)
	if err != nil {
		return domain.WalletOrder{}, nil, appshared.ErrInvalidInput
	}
	if s.vps == nil || s.orderItems == nil {
		return domain.WalletOrder{}, nil, appshared.ErrInvalidInput
	}
	inst, err := s.vps.GetInstance(ctx, vpsID)
	if err != nil {
		return domain.WalletOrder{}, nil, err
	}
	if inst.UserID != userID {
		return domain.WalletOrder{}, nil, appshared.ErrForbidden
	}
	item, err := s.orderItems.GetOrderItem(ctx, inst.OrderItemID)
	if err != nil {
		return domain.WalletOrder{}, nil, err
	}
	policy := s.refundPolicy(ctx)
	amount := s.calculateRefundAmount(inst, item, policy)
	if amount <= 0 {
		return domain.WalletOrder{}, nil, appshared.ErrForbidden
	}
	meta := map[string]any{"vps_id": inst.ID, "order_item_id": item.ID, "refund_policy": policy, "reason": reason, "delete_on_approve": true}
	autoApprove := !policy.RequireApproval
	order := domain.WalletOrder{UserID: userID, Type: domain.WalletOrderRefund, Amount: amount, Currency: "CNY", Status: domain.WalletOrderPendingReview, Note: reason, MetaJSON: toJSON(meta)}
	if err := s.orders.CreateWalletOrder(ctx, &order); err != nil {
		return domain.WalletOrder{}, nil, err
	}
	if autoApprove {
		wallet, err := s.approveOrder(ctx, 0, order, true)
		if err != nil {
			return domain.WalletOrder{}, nil, err
		}
		order.Status = domain.WalletOrderApproved
		return order, &wallet, nil
	}
	return order, nil, nil
}

func (s *Service) AutoRefundOnAdminDelete(ctx context.Context, adminID int64, vpsID int64, reason string) (domain.WalletOrder, *domain.Wallet, error) {
	if vpsID == 0 {
		return domain.WalletOrder{}, nil, appshared.ErrInvalidInput
	}
	reason, err := trimAndValidateOptional(reason, maxLenRefundReason)
	if err != nil {
		return domain.WalletOrder{}, nil, appshared.ErrInvalidInput
	}
	if s.vps == nil || s.orderItems == nil {
		return domain.WalletOrder{}, nil, appshared.ErrInvalidInput
	}
	policy := s.refundPolicy(ctx)
	if !policy.AutoRefundOnDelete {
		return domain.WalletOrder{}, nil, nil
	}
	inst, err := s.vps.GetInstance(ctx, vpsID)
	if err != nil {
		return domain.WalletOrder{}, nil, err
	}
	item, err := s.orderItems.GetOrderItem(ctx, inst.OrderItemID)
	if err != nil {
		return domain.WalletOrder{}, nil, err
	}
	amount := s.calculateRefundAmount(inst, item, policy)
	if amount <= 0 {
		return domain.WalletOrder{}, nil, nil
	}
	// Idempotency: the admin delete flow creates the refund BEFORE the external
	// DeleteHost. If that delete fails and the admin retries, a second refund
	// order would credit the wallet twice (each wallet order gets its own
	// idempotency key). Reuse an existing pending/approved admin_delete refund
	// for this VPS instead of creating a duplicate.
	if existing, found, findErr := s.findAdminDeleteRefund(ctx, inst.UserID, inst.ID); findErr != nil {
		return domain.WalletOrder{}, nil, findErr
	} else if found {
		if existing.Status == domain.WalletOrderApproved {
			if w, err := s.wallets.GetWallet(ctx, inst.UserID); err == nil {
				return existing, &w, nil
			}
		}
		return existing, nil, nil
	}
	meta := map[string]any{"vps_id": inst.ID, "order_item_id": item.ID, "refund_policy": policy, "reason": reason, "delete_on_approve": false, "trigger": "admin_delete"}
	autoApprove := !policy.RequireApproval
	order := domain.WalletOrder{UserID: inst.UserID, Type: domain.WalletOrderRefund, Amount: amount, Currency: "CNY", Status: domain.WalletOrderPendingReview, Note: reason, MetaJSON: toJSON(meta)}
	if err := s.orders.CreateWalletOrder(ctx, &order); err != nil {
		return domain.WalletOrder{}, nil, err
	}
	if autoApprove {
		wallet, err := s.approveOrder(ctx, adminID, order, false)
		if err != nil {
			return domain.WalletOrder{}, nil, err
		}
		order.Status = domain.WalletOrderApproved
		return order, &wallet, nil
	}
	return order, nil, nil
}

// findAdminDeleteRefund returns an existing pending/approved refund wallet
// order for the given VPS that was triggered by the admin-delete flow, so a
// retried delete after an external DeleteHost failure does not create a second
// refund (double wallet credit).
func (s *Service) findAdminDeleteRefund(ctx context.Context, userID, vpsID int64) (domain.WalletOrder, bool, error) {
	if s.orders == nil {
		return domain.WalletOrder{}, false, nil
	}
	const pageSize = 100
	for offset := 0; ; offset += pageSize {
		orders, total, err := s.orders.ListWalletOrders(ctx, userID, pageSize, offset)
		if err != nil {
			return domain.WalletOrder{}, false, err
		}
		for _, o := range orders {
			if o.Type != domain.WalletOrderRefund {
				continue
			}
			if o.Status != domain.WalletOrderPendingReview && o.Status != domain.WalletOrderApproved {
				continue
			}
			meta := parseJSON(o.MetaJSON)
			if getInt64(meta["vps_id"]) == vpsID && getString(meta["trigger"]) == "admin_delete" {
				return o, true, nil
			}
		}
		if offset+len(orders) >= total || len(orders) == 0 {
			break
		}
	}
	return domain.WalletOrder{}, false, nil
}

func (s *Service) Approve(ctx context.Context, adminID int64, orderID int64) (domain.WalletOrder, *domain.Wallet, error) {
	order, err := s.orders.GetWalletOrder(ctx, orderID)
	if err != nil {
		return domain.WalletOrder{}, nil, err
	}
	if order.Status != domain.WalletOrderPendingReview {
		return domain.WalletOrder{}, nil, appshared.ErrConflict
	}
	wallet, err := s.approveOrder(ctx, adminID, order, order.Type == domain.WalletOrderRefund)
	if err != nil {
		return domain.WalletOrder{}, nil, err
	}
	order.Status = domain.WalletOrderApproved
	order.ReviewedBy = &adminID
	return order, &wallet, nil
}

func (s *Service) Reject(ctx context.Context, adminID int64, orderID int64, reason string) error {
	reason, err := trimAndValidateOptional(reason, maxLenReviewReason)
	if err != nil {
		return appshared.ErrInvalidInput
	}
	order, err := s.orders.GetWalletOrder(ctx, orderID)
	if err != nil {
		return err
	}
	if order.Status != domain.WalletOrderPendingReview {
		return appshared.ErrConflict
	}
	updated, err := s.orders.UpdateWalletOrderStatusIfCurrent(ctx, order.ID, domain.WalletOrderPendingReview, domain.WalletOrderRejected, &adminID, reason)
	if err != nil {
		return err
	}
	if !updated {
		return appshared.ErrConflict
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "wallet_order.reject", TargetType: "wallet_order", TargetID: strconv.FormatInt(order.ID, 10), DetailJSON: mustJSON(map[string]any{"type": order.Type, "amount": order.Amount, "reason": reason})})
	}
	return nil
}

func (s *Service) approveOrder(ctx context.Context, adminID int64, order domain.WalletOrder, allowDelete bool) (domain.Wallet, error) {
	if s.wallets == nil {
		return domain.Wallet{}, appshared.ErrInvalidInput
	}
	var amount int64
	var txType string
	switch order.Type {
	case domain.WalletOrderRecharge:
		amount = order.Amount
		txType = "credit"
	case domain.WalletOrderWithdraw:
		amount = -order.Amount
		txType = "debit"
	case domain.WalletOrderRefund:
		amount = order.Amount
		txType = "credit"
	default:
		return domain.Wallet{}, appshared.ErrInvalidInput
	}
	wallet, claimed, err := s.orders.ApproveWalletOrderAtomic(ctx, order, adminID, amount, txType)
	if err != nil {
		return domain.Wallet{}, err
	}
	if !claimed {
		return domain.Wallet{}, appshared.ErrConflict
	}
	if allowDelete && order.Type == domain.WalletOrderRefund && deleteOnApprove(order.MetaJSON) {
		// Persist the durable cleanup outbox entry BEFORE the inline delete so a
		// crash after the balance credit still deletes the refunded host: the
		// cleanup worker retries any pending entry with bounded backoff.
		s.enqueueRefundCleanup(ctx, order)
		if deleteErr := s.deleteVPS(ctx, order.MetaJSON); deleteErr != nil {
			// External provider errors can embed signed URLs, tokens, passwords
			// and internal topology. Persist only a centralized stable error code
			// and a sanitized one-line summary; the detailed error goes to the
			// log only after passing through the secret scrubber. The cleanup job
			// stays pending and the worker drives the retries.
			code := cleanupErrorCode(deleteErr)
			meta := parseJSON(order.MetaJSON)
			meta["delete_pending"] = true
			meta["delete_error_code"] = code
			meta["delete_error"] = scrub.SanitizeSummary(code)
			_ = s.orders.UpdateWalletOrderMeta(ctx, order.ID, toJSON(meta))
			if s.audit != nil {
				_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "wallet_order.refund_cleanup_failed", TargetType: "wallet_order", TargetID: strconv.FormatInt(order.ID, 10), DetailJSON: mustJSON(map[string]any{"error_code": code, "error": scrub.SanitizeSummary(code)})})
			}
			log.Printf("wallet_order.refund_cleanup_failed order_id=%d code=%s detail=%s", order.ID, code, scrub.Redact(deleteErr.Error()))
		} else if s.cleanup != nil {
			// Inline delete succeeded; retire the outbox entry.
			if job, err := s.cleanup.GetRefundCleanupJobByWalletOrder(ctx, order.ID); err == nil {
				_ = s.cleanup.MarkRefundCleanupDone(ctx, job.ID)
			}
		}
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "wallet_order.approve", TargetType: "wallet_order", TargetID: strconv.FormatInt(order.ID, 10), DetailJSON: mustJSON(map[string]any{"type": order.Type, "amount": order.Amount})})
	}
	if s.userTiers != nil {
		_ = s.userTiers.TryAutoApproveForUser(ctx, order.UserID, "wallet_order_success")
	}
	return wallet, nil
}

func (s *Service) deleteVPS(ctx context.Context, metaJSON string) error {
	if s.vps == nil || s.automation == nil {
		return appshared.ErrInvalidInput
	}
	meta := parseJSON(metaJSON)
	vpsID := getInt64(meta["vps_id"])
	if vpsID == 0 {
		return appshared.ErrInvalidInput
	}
	inst, err := s.vps.GetInstance(ctx, vpsID)
	if err != nil {
		return err
	}
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return appshared.ErrInvalidInput
	}
	cli, err := s.automation.ClientForGoodsType(ctx, inst.GoodsTypeID)
	if err != nil {
		return err
	}
	// A DeleteHost reporting the host is already gone is an idempotent success:
	// the refund already credited the wallet and the cleanup outbox retry must
	// converge on updating the local row rather than being blocked by the
	// provider's "instance does not exist" error.
	if err := cli.DeleteHost(ctx, hostID); err != nil && !appshared.IsHostNotFoundError(err) {
		return err
	}
	return s.vps.UpdateInstanceStatus(ctx, inst.ID, domain.VPSStatusUnknown, inst.AutomationState)
}

func (s *Service) refundPolicy(ctx context.Context) RefundPolicy {
	policy := RefundPolicy{
		FullHours:          0,
		ProrateHours:       0,
		NoRefundHours:      0,
		FullDays:           1,
		ProrateDays:        7,
		NoRefundDays:       30,
		RequireApproval:    true,
		AutoRefundOnDelete: false,
	}
	if s.settings == nil {
		return policy
	}
	if v, ok := getSettingInt(ctx, s.settings, "refund_full_days"); ok {
		policy.FullDays = v
	}
	if v, ok := getSettingInt(ctx, s.settings, "refund_prorate_days"); ok {
		policy.ProrateDays = v
	}
	if v, ok := getSettingInt(ctx, s.settings, "refund_no_refund_days"); ok {
		policy.NoRefundDays = v
	}
	if v, ok := getSettingInt(ctx, s.settings, "refund_full_hours"); ok {
		policy.FullHours = v
	}
	if v, ok := getSettingInt(ctx, s.settings, "refund_prorate_hours"); ok {
		policy.ProrateHours = v
	}
	if v, ok := getSettingInt(ctx, s.settings, "refund_no_refund_hours"); ok {
		policy.NoRefundHours = v
	}
	if v, ok := getSettingBool(ctx, s.settings, "refund_requires_approval"); ok {
		policy.RequireApproval = v
	}
	if v, ok := getSettingBool(ctx, s.settings, "refund_on_admin_delete"); ok {
		policy.AutoRefundOnDelete = v
	}
	if curve, ok := LoadRefundCurve(ctx, s.settings); ok {
		policy.Curve = curve
	}
	return policy
}

func (s *Service) calculateRefundAmount(inst domain.VPSInstance, item domain.OrderItem, policy RefundPolicy) int64 {
	// The refund base must be the amount actually paid for the item, which
	// already accounts for prepaid DurationMonths and any coupon discount.
	// MonthlyPrice (single month) is only a fallback when the paid amount is
	// unavailable; using it for multi-month prepays systematically under-refunds.
	baseAmount := item.Amount
	if baseAmount <= 0 {
		baseAmount = inst.MonthlyPrice
	}
	return calculateRefundAmountForAmount(inst, baseAmount, policy)
}

func (p *RefundCurvePoint) UnmarshalJSON(data []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	p.Percent = 0
	if v, ok := raw["percent"]; ok {
		if n, ok := asInt(v); ok {
			p.Percent = n
		}
	} else if v, ok := raw["hours"]; ok {
		if n, ok := asInt(v); ok {
			p.Percent = n
		}
	}
	p.Ratio = 0
	if v, ok := raw["ratio"]; ok {
		switch val := v.(type) {
		case float64:
			p.Ratio = val
		case int:
			p.Ratio = float64(val)
		case int64:
			p.Ratio = float64(val)
		}
	}
	return nil
}

func asInt(v any) (int, bool) {
	switch val := v.(type) {
	case float64:
		return int(val), true
	case int:
		return val, true
	case int64:
		return int(val), true
	case json.Number:
		i, err := val.Int64()
		if err != nil {
			return 0, false
		}
		return int(i), true
	default:
		return 0, false
	}
}

func LoadRefundCurve(ctx context.Context, repo appports.SettingsRepository) ([]RefundCurvePoint, bool) {
	return LoadRefundCurveByKey(ctx, repo, "refund_curve_json")
}

func LoadRefundCurveByKey(ctx context.Context, repo appports.SettingsRepository, key string) ([]RefundCurvePoint, bool) {
	raw, ok := getSettingString(ctx, repo, key)
	if !ok || strings.TrimSpace(raw) == "" {
		return nil, false
	}
	var points []RefundCurvePoint
	if err := json.Unmarshal([]byte(raw), &points); err != nil {
		return nil, false
	}
	points = NormalizeRefundCurve(points)
	if len(points) == 0 {
		return nil, false
	}
	return points, true
}

func NormalizeRefundCurve(points []RefundCurvePoint) []RefundCurvePoint {
	if len(points) == 0 {
		return nil
	}
	seen := make(map[int]float64)
	order := make([]int, 0, len(points))
	for _, point := range points {
		// The curve x-axis is a 0-100 percentage of the lifecycle. Values
		// outside that range (e.g. hours entered as if they were percent)
		// would silently make the curve degenerate, so drop them.
		if point.Percent < 0 || point.Percent > 100 {
			continue
		}
		ratio := clamp01(point.Ratio)
		if _, ok := seen[point.Percent]; !ok {
			order = append(order, point.Percent)
		}
		seen[point.Percent] = ratio
	}
	if len(order) == 0 {
		return nil
	}
	sort.Ints(order)
	out := make([]RefundCurvePoint, 0, len(order))
	for _, percent := range order {
		out = append(out, RefundCurvePoint{Percent: percent, Ratio: seen[percent]})
	}
	return out
}

func RefundCurveRatio(points []RefundCurvePoint, elapsedPercent float64) (float64, bool) {
	if len(points) == 0 {
		return 0, false
	}
	if elapsedPercent < 0 {
		elapsedPercent = 0
	}
	if elapsedPercent <= float64(points[0].Percent) {
		return clamp01(points[0].Ratio), true
	}
	for i := 1; i < len(points); i++ {
		if elapsedPercent <= float64(points[i].Percent) {
			prev := points[i-1]
			next := points[i]
			span := float64(next.Percent - prev.Percent)
			if span <= 0 {
				return clamp01(next.Ratio), true
			}
			t := (elapsedPercent - float64(prev.Percent)) / span
			ratio := prev.Ratio + t*(next.Ratio-prev.Ratio)
			return clamp01(ratio), true
		}
	}
	return clamp01(points[len(points)-1].Ratio), true
}

func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func calculateRefundAmountForAmount(inst domain.VPSInstance, amount int64, policy RefundPolicy) int64 {
	if amount <= 0 {
		return 0
	}
	now := time.Now()
	if inst.ExpireAt != nil && !inst.ExpireAt.After(now) {
		return 0
	}
	// The elapsed ratio must be measured against the current billing period,
	// not the instance creation time: renewals extend ExpireAt without touching
	// CreatedAt, so a lifetime-based ratio would systematically under-refund
	// every renewed instance. This mirrors the order-package fix (m016 F1).
	periodStart, periodEnd, hasPeriod := inst.CurrentBillingPeriod(now)
	var elapsedRatio float64
	if hasPeriod {
		total := periodEnd.Sub(periodStart)
		if total <= 0 {
			hasPeriod = false
		} else {
			elapsed := now.Sub(periodStart)
			switch {
			case elapsed <= 0:
				elapsedRatio = 0
			case elapsed >= total:
				elapsedRatio = 1
			default:
				elapsedRatio = float64(elapsed) / float64(total)
			}
		}
	}
	if !hasPeriod {
		elapsedRatio = refundElapsedRatio(inst, now)
	}
	if len(policy.Curve) > 0 {
		if ratio, ok := RefundCurveRatio(policy.Curve, elapsedRatio*100); ok {
			return int64(math.Round(float64(amount) * ratio))
		}
	}
	if !hasPeriod {
		ageHoursFloat := now.Sub(inst.CreatedAt).Hours()
		ageHours := int(ageHoursFloat)
		ageDays := int(ageHoursFloat / 24)
		if policy.NoRefundHours > 0 && ageHours > policy.NoRefundHours {
			return 0
		}
		if policy.NoRefundHours <= 0 && policy.NoRefundDays > 0 && ageDays > policy.NoRefundDays {
			return 0
		}
		if policy.FullHours > 0 && ageHours <= policy.FullHours {
			return amount
		}
		if policy.FullHours <= 0 && policy.FullDays > 0 && ageDays <= policy.FullDays {
			return amount
		}
		if policy.ProrateHours > 0 && ageHours <= policy.ProrateHours {
			ratio := float64(policy.ProrateHours-ageHours) / float64(policy.ProrateHours)
			return int64(math.Round(float64(amount) * ratio))
		}
		if policy.ProrateHours <= 0 && policy.ProrateDays > 0 && ageDays <= policy.ProrateDays {
			ratio := float64(policy.ProrateDays-ageDays) / float64(policy.ProrateDays)
			return int64(math.Round(float64(amount) * ratio))
		}
		return 0
	}

	// Integer-exact proration against the current billing period, consistent
	// with the order-package refund path: the refund windows are absolute
	// (e.g. "full refund within 15 days") rather than a ratio of the full
	// lifetime, which would under-refund renewed instances.
	fullNanos := refundThresholdNanos(policy.FullHours, policy.FullDays)
	prorateNanos := refundThresholdNanos(policy.ProrateHours, policy.ProrateDays)
	noRefundNanos := refundThresholdNanos(policy.NoRefundHours, policy.NoRefundDays)
	elapsedNanos := int64(now.Sub(periodStart))
	if elapsedNanos < 0 {
		elapsedNanos = 0
	}
	if noRefundNanos > 0 && elapsedNanos > noRefundNanos {
		return 0
	}
	if fullNanos > 0 && elapsedNanos <= fullNanos {
		return amount
	}
	if prorateNanos > 0 && elapsedNanos <= prorateNanos {
		return money.ProrateCents(amount, prorateNanos-elapsedNanos, prorateNanos)
	}
	return 0
}

// refundThresholdNanos converts a refund policy window expressed in hours
// and/or days into nanoseconds. Hours win when both are set.
func refundThresholdNanos(hours int, days int) int64 {
	h := int64(hours)
	if h <= 0 && days > 0 {
		h = int64(days) * 24
	}
	if h <= 0 {
		return 0
	}
	return h * int64(time.Hour)
}

func refundElapsedRatio(inst domain.VPSInstance, now time.Time) float64 {
	if inst.ExpireAt == nil || inst.CreatedAt.IsZero() {
		return 1
	}
	total := inst.ExpireAt.Sub(inst.CreatedAt)
	if total <= 0 {
		return 1
	}
	if now.Before(inst.CreatedAt) {
		return 0
	}
	if !inst.ExpireAt.After(now) {
		return 1
	}
	ratio := now.Sub(inst.CreatedAt).Seconds() / total.Seconds()
	if ratio < 0 {
		return 0
	}
	if ratio > 1 {
		return 1
	}
	return ratio
}

func refundPeriodHours(inst domain.VPSInstance) (float64, bool) {
	if inst.ExpireAt == nil || inst.CreatedAt.IsZero() {
		return 0, false
	}
	total := inst.ExpireAt.Sub(inst.CreatedAt).Hours()
	if total <= 0 {
		return 0, false
	}
	return total, true
}

func refundRatioThreshold(totalHours float64, hours int, days int) float64 {
	if totalHours <= 0 {
		return 0
	}
	thresholdHours := float64(hours)
	if thresholdHours <= 0 && days > 0 {
		thresholdHours = float64(days) * 24
	}
	if thresholdHours <= 0 {
		return 0
	}
	ratio := thresholdHours / totalHours
	if ratio < 0 {
		return 0
	}
	if ratio > 1 {
		return 1
	}
	return ratio
}

func toJSON(meta map[string]any) string {
	if len(meta) == 0 {
		return ""
	}
	b, _ := json.Marshal(meta)
	return string(b)
}

func parseJSON(raw string) map[string]any {
	if raw == "" {
		return map[string]any{}
	}
	var out map[string]any
	_ = json.Unmarshal([]byte(raw), &out)
	if out == nil {
		out = map[string]any{}
	}
	return out
}

func deleteOnApprove(metaJSON string) bool {
	meta := parseJSON(metaJSON)
	if val, ok := meta["delete_on_approve"]; ok {
		switch v := val.(type) {
		case bool:
			return v
		case string:
			return strings.EqualFold(v, "true") || v == "1"
		case float64:
			return v == 1
		}
	}
	return false
}

func getInt64(v any) int64 {
	switch val := v.(type) {
	case int64:
		return val
	case int:
		return int64(val)
	case float64:
		return int64(val)
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(val), 10, 64)
		return n
	default:
		return 0
	}
}

func getString(v any) string {
	switch val := v.(type) {
	case string:
		return strings.TrimSpace(val)
	case fmt.Stringer:
		return strings.TrimSpace(val.String())
	default:
		return ""
	}
}

func getSettingString(ctx context.Context, repo appports.SettingsRepository, key string) (string, bool) {
	if repo == nil {
		return "", false
	}
	setting, err := repo.GetSetting(ctx, key)
	if err != nil {
		return "", false
	}
	return setting.ValueJSON, true
}

func getSettingInt(ctx context.Context, repo appports.SettingsRepository, key string) (int, bool) {
	if repo == nil {
		return 0, false
	}
	setting, err := repo.GetSetting(ctx, key)
	if err != nil {
		return 0, false
	}
	val, err := strconv.Atoi(strings.TrimSpace(setting.ValueJSON))
	if err != nil {
		return 0, false
	}
	return val, true
}

func getSettingBool(ctx context.Context, repo appports.SettingsRepository, key string) (bool, bool) {
	if repo == nil {
		return false, false
	}
	setting, err := repo.GetSetting(ctx, key)
	if err != nil {
		return false, false
	}
	raw := strings.TrimSpace(setting.ValueJSON)
	if raw == "" {
		return false, false
	}
	switch strings.ToLower(raw) {
	case "true", "1", "yes":
		return true, true
	case "false", "0", "no":
		return false, true
	default:
		return false, false
	}
}

func parseHostID(v string) int64 {
	id, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	return id
}

// cleanupErrorCode classifies a refund-cleanup (DeleteHost) failure into a
// centralized stable code. The provider error itself is never persisted raw;
// callers persist this code plus a scrubbed summary and route the detailed
// error through the secret scrubber before logging.
func cleanupErrorCode(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, appshared.ErrInvalidInput):
		return "invalid_input"
	case errors.Is(err, appshared.ErrNotFound):
		return "not_found"
	case errors.Is(err, context.Canceled):
		return "canceled"
	case errors.Is(err, context.DeadlineExceeded):
		return "provider_timeout"
	default:
		var netErr net.Error
		if errors.As(err, &netErr) {
			if netErr.Timeout() {
				return "provider_timeout"
			}
			return "provider_unreachable"
		}
		return "provider_error"
	}
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
