package usecase

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"xiaoheiplay/internal/domain"
)

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

type WalletOrderService struct {
	orders     WalletOrderRepository
	wallets    WalletRepository
	settings   SettingsRepository
	vps        VPSRepository
	orderItems OrderItemRepository
	automation AutomationClientResolver
	audit      AuditRepository
}

func NewWalletOrderService(orders WalletOrderRepository, wallets WalletRepository, settings SettingsRepository, vps VPSRepository, orderItems OrderItemRepository, automation AutomationClientResolver, audit AuditRepository) *WalletOrderService {
	return &WalletOrderService{
		orders:     orders,
		wallets:    wallets,
		settings:   settings,
		vps:        vps,
		orderItems: orderItems,
		automation: automation,
		audit:      audit,
	}
}

type WalletOrderCreateInput struct {
	Amount   int64
	Currency string
	Note     string
	Meta     map[string]any
}

func (s *WalletOrderService) CreateRefundOrder(ctx context.Context, userID int64, amount int64, note string, meta map[string]any) (domain.WalletOrder, error) {
	if userID == 0 || amount <= 0 {
		return domain.WalletOrder{}, ErrInvalidInput
	}
	order := domain.WalletOrder{
		UserID:   userID,
		Type:     domain.WalletOrderRefund,
		Amount:   amount,
		Currency: "CNY",
		Status:   domain.WalletOrderPendingReview,
		Note:     strings.TrimSpace(note),
		MetaJSON: toJSON(meta),
	}
	if err := s.orders.CreateWalletOrder(ctx, &order); err != nil {
		return domain.WalletOrder{}, err
	}
	return order, nil
}

func (s *WalletOrderService) CreateRecharge(ctx context.Context, userID int64, input WalletOrderCreateInput) (domain.WalletOrder, error) {
	if userID == 0 || input.Amount <= 0 {
		return domain.WalletOrder{}, ErrInvalidInput
	}
	currency := strings.TrimSpace(input.Currency)
	if currency == "" {
		currency = "CNY"
	}
	order := domain.WalletOrder{
		UserID:   userID,
		Type:     domain.WalletOrderRecharge,
		Amount:   input.Amount,
		Currency: currency,
		Status:   domain.WalletOrderPendingReview,
		Note:     strings.TrimSpace(input.Note),
		MetaJSON: toJSON(input.Meta),
	}
	if err := s.orders.CreateWalletOrder(ctx, &order); err != nil {
		return domain.WalletOrder{}, err
	}
	return order, nil
}

func (s *WalletOrderService) CreateWithdraw(ctx context.Context, userID int64, input WalletOrderCreateInput) (domain.WalletOrder, error) {
	if userID == 0 || input.Amount <= 0 {
		return domain.WalletOrder{}, ErrInvalidInput
	}
	if s.wallets == nil {
		return domain.WalletOrder{}, ErrInvalidInput
	}
	wallet, err := s.wallets.GetWallet(ctx, userID)
	if err != nil {
		return domain.WalletOrder{}, err
	}
	if wallet.Balance < input.Amount {
		return domain.WalletOrder{}, ErrInsufficientBalance
	}
	currency := strings.TrimSpace(input.Currency)
	if currency == "" {
		currency = "CNY"
	}
	order := domain.WalletOrder{
		UserID:   userID,
		Type:     domain.WalletOrderWithdraw,
		Amount:   input.Amount,
		Currency: currency,
		Status:   domain.WalletOrderPendingReview,
		Note:     strings.TrimSpace(input.Note),
		MetaJSON: toJSON(input.Meta),
	}
	if err := s.orders.CreateWalletOrder(ctx, &order); err != nil {
		return domain.WalletOrder{}, err
	}
	return order, nil
}

func (s *WalletOrderService) ListUserOrders(ctx context.Context, userID int64, limit, offset int) ([]domain.WalletOrder, int, error) {
	return s.orders.ListWalletOrders(ctx, userID, limit, offset)
}

func (s *WalletOrderService) ListAllOrders(ctx context.Context, status string, limit, offset int) ([]domain.WalletOrder, int, error) {
	return s.orders.ListAllWalletOrders(ctx, status, limit, offset)
}

func (s *WalletOrderService) RequestRefund(ctx context.Context, userID int64, vpsID int64, reason string) (domain.WalletOrder, *domain.Wallet, error) {
	if userID == 0 || vpsID == 0 {
		return domain.WalletOrder{}, nil, ErrInvalidInput
	}
	if s.vps == nil || s.orderItems == nil {
		return domain.WalletOrder{}, nil, ErrInvalidInput
	}
	inst, err := s.vps.GetInstance(ctx, vpsID)
	if err != nil {
		return domain.WalletOrder{}, nil, err
	}
	if inst.UserID != userID {
		return domain.WalletOrder{}, nil, ErrForbidden
	}
	item, err := s.orderItems.GetOrderItem(ctx, inst.OrderItemID)
	if err != nil {
		return domain.WalletOrder{}, nil, err
	}
	policy := s.refundPolicy(ctx)
	amount := s.calculateRefundAmount(inst, item, policy)
	if amount <= 0 {
		return domain.WalletOrder{}, nil, ErrForbidden
	}
	meta := map[string]any{
		"vps_id":            inst.ID,
		"order_item_id":     item.ID,
		"refund_policy":     policy,
		"reason":            strings.TrimSpace(reason),
		"delete_on_approve": true,
	}
	status := domain.WalletOrderPendingReview
	if !policy.RequireApproval {
		status = domain.WalletOrderApproved
	}
	order := domain.WalletOrder{
		UserID:   userID,
		Type:     domain.WalletOrderRefund,
		Amount:   amount,
		Currency: "CNY",
		Status:   status,
		Note:     strings.TrimSpace(reason),
		MetaJSON: toJSON(meta),
	}
	if err := s.orders.CreateWalletOrder(ctx, &order); err != nil {
		return domain.WalletOrder{}, nil, err
	}
	if status == domain.WalletOrderApproved {
		wallet, err := s.approveOrder(ctx, 0, order, true)
		if err != nil {
			return domain.WalletOrder{}, nil, err
		}
		return order, &wallet, nil
	}
	return order, nil, nil
}

func (s *WalletOrderService) AutoRefundOnAdminDelete(ctx context.Context, adminID int64, vpsID int64, reason string) (domain.WalletOrder, *domain.Wallet, error) {
	if vpsID == 0 {
		return domain.WalletOrder{}, nil, ErrInvalidInput
	}
	if s.vps == nil || s.orderItems == nil {
		return domain.WalletOrder{}, nil, ErrInvalidInput
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
	meta := map[string]any{
		"vps_id":            inst.ID,
		"order_item_id":     item.ID,
		"refund_policy":     policy,
		"reason":            strings.TrimSpace(reason),
		"delete_on_approve": false,
		"trigger":           "admin_delete",
	}
	status := domain.WalletOrderPendingReview
	if !policy.RequireApproval {
		status = domain.WalletOrderApproved
	}
	order := domain.WalletOrder{
		UserID:   inst.UserID,
		Type:     domain.WalletOrderRefund,
		Amount:   amount,
		Currency: "CNY",
		Status:   status,
		Note:     strings.TrimSpace(reason),
		MetaJSON: toJSON(meta),
	}
	if err := s.orders.CreateWalletOrder(ctx, &order); err != nil {
		return domain.WalletOrder{}, nil, err
	}
	if status == domain.WalletOrderApproved {
		wallet, err := s.approveOrder(ctx, adminID, order, false)
		if err != nil {
			return domain.WalletOrder{}, nil, err
		}
		return order, &wallet, nil
	}
	return order, nil, nil
}

func (s *WalletOrderService) Approve(ctx context.Context, adminID int64, orderID int64) (domain.WalletOrder, *domain.Wallet, error) {
	order, err := s.orders.GetWalletOrder(ctx, orderID)
	if err != nil {
		return domain.WalletOrder{}, nil, err
	}
	if order.Status != domain.WalletOrderPendingReview {
		return domain.WalletOrder{}, nil, ErrConflict
	}
	wallet, err := s.approveOrder(ctx, adminID, order, order.Type == domain.WalletOrderRefund)
	if err != nil {
		return domain.WalletOrder{}, nil, err
	}
	order.Status = domain.WalletOrderApproved
	order.ReviewedBy = &adminID
	return order, &wallet, nil
}

func (s *WalletOrderService) Reject(ctx context.Context, adminID int64, orderID int64, reason string) error {
	order, err := s.orders.GetWalletOrder(ctx, orderID)
	if err != nil {
		return err
	}
	if order.Status != domain.WalletOrderPendingReview {
		return ErrConflict
	}
	if err := s.orders.UpdateWalletOrderStatus(ctx, order.ID, domain.WalletOrderRejected, &adminID, strings.TrimSpace(reason)); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{
			AdminID:    adminID,
			Action:     "wallet_order.reject",
			TargetType: "wallet_order",
			TargetID:   strconv.FormatInt(order.ID, 10),
			DetailJSON: mustJSON(map[string]any{"type": order.Type, "amount": order.Amount, "reason": reason}),
		})
	}
	return nil
}

func (s *WalletOrderService) approveOrder(ctx context.Context, adminID int64, order domain.WalletOrder, allowDelete bool) (domain.Wallet, error) {
	if s.wallets == nil {
		return domain.Wallet{}, ErrInvalidInput
	}
	if allowDelete && order.Type == domain.WalletOrderRefund {
		if deleteOnApprove(order.MetaJSON) {
			if err := s.deleteVPS(ctx, order.MetaJSON); err != nil {
				return domain.Wallet{}, err
			}
		}
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
		return domain.Wallet{}, ErrInvalidInput
	}

	// Idempotency: wallet_transactions uses (user_id, ref_type, ref_id) to dedupe.
	// This makes Approve safe to retry when balance adjustment succeeded but order status update failed.
	refType := "wallet_order"
	exists, err := s.wallets.HasWalletTransaction(ctx, order.UserID, refType, order.ID)
	if err != nil {
		return domain.Wallet{}, err
	}
	var wallet domain.Wallet
	if exists {
		wallet, err = s.wallets.GetWallet(ctx, order.UserID)
		if err != nil {
			return domain.Wallet{}, err
		}
	} else {
		wallet, err = s.wallets.AdjustWalletBalance(ctx, order.UserID, amount, txType, refType, order.ID, string(order.Type))
		if err != nil {
			// In case of a concurrent approve, the transaction insert may have raced and won elsewhere.
			ok, checkErr := s.wallets.HasWalletTransaction(ctx, order.UserID, refType, order.ID)
			if checkErr == nil && ok {
				wallet, checkErr = s.wallets.GetWallet(ctx, order.UserID)
				if checkErr != nil {
					return domain.Wallet{}, checkErr
				}
			} else {
				return domain.Wallet{}, err
			}
		}
	}
	if err := s.orders.UpdateWalletOrderStatus(ctx, order.ID, domain.WalletOrderApproved, &adminID, ""); err != nil {
		return domain.Wallet{}, err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{
			AdminID:    adminID,
			Action:     "wallet_order.approve",
			TargetType: "wallet_order",
			TargetID:   strconv.FormatInt(order.ID, 10),
			DetailJSON: mustJSON(map[string]any{"type": order.Type, "amount": order.Amount}),
		})
	}
	return wallet, nil
}

func (s *WalletOrderService) deleteVPS(ctx context.Context, metaJSON string) error {
	if s.vps == nil || s.automation == nil {
		return ErrInvalidInput
	}
	meta := parseJSON(metaJSON)
	vpsID := getInt64(meta["vps_id"])
	if vpsID == 0 {
		return ErrInvalidInput
	}
	inst, err := s.vps.GetInstance(ctx, vpsID)
	if err != nil {
		return err
	}
	hostID := parseHostID(inst.AutomationInstanceID)
	if hostID == 0 {
		return ErrInvalidInput
	}
	cli, err := s.automation.ClientForGoodsType(ctx, inst.GoodsTypeID)
	if err != nil {
		return err
	}
	if err := cli.DeleteHost(ctx, hostID); err != nil {
		return err
	}
	_ = s.vps.UpdateInstanceStatus(ctx, inst.ID, domain.VPSStatusUnknown, inst.AutomationState)
	return nil
}

func (s *WalletOrderService) refundPolicy(ctx context.Context) RefundPolicy {
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

func (s *WalletOrderService) calculateRefundAmount(inst domain.VPSInstance, item domain.OrderItem, policy RefundPolicy) int64 {
	baseAmount := inst.MonthlyPrice
	if baseAmount <= 0 {
		baseAmount = item.Amount
	}
	return calculateRefundAmountForAmount(inst, baseAmount, policy)
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

func getSettingInt(ctx context.Context, repo SettingsRepository, key string) (int, bool) {
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

func getSettingBool(ctx context.Context, repo SettingsRepository, key string) (bool, bool) {
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
