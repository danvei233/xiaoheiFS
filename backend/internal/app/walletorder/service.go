package walletorder

import (
	"context"
	"encoding/json"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
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
}

func NewService(orders appports.WalletOrderRepository, wallets appports.WalletRepository, settings appports.SettingsRepository, vps appports.VPSRepository, orderItems appports.OrderItemRepository, automation appports.AutomationClientResolver, audit appports.AuditRepository) *Service {
	return &Service{orders: orders, wallets: wallets, settings: settings, vps: vps, orderItems: orderItems, automation: automation, audit: audit}
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
	if userID == 0 || input.Amount <= 0 {
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
	order := domain.WalletOrder{UserID: userID, Type: domain.WalletOrderRecharge, Amount: input.Amount, Currency: currency, Status: domain.WalletOrderPendingReview, Note: trimmedNote, MetaJSON: toJSON(input.Meta)}
	if err := s.orders.CreateWalletOrder(ctx, &order); err != nil {
		return domain.WalletOrder{}, err
	}
	return order, nil
}

func (s *Service) CreateWithdraw(ctx context.Context, userID int64, input WalletOrderCreateInput) (domain.WalletOrder, error) {
	if userID == 0 || input.Amount <= 0 {
		return domain.WalletOrder{}, appshared.ErrInvalidInput
	}
	trimmedNote, err := trimAndValidateOptional(input.Note, maxLenPaymentNote)
	if err != nil {
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
	currency := strings.TrimSpace(input.Currency)
	if currency == "" {
		currency = "CNY"
	}
	order := domain.WalletOrder{UserID: userID, Type: domain.WalletOrderWithdraw, Amount: input.Amount, Currency: currency, Status: domain.WalletOrderPendingReview, Note: trimmedNote, MetaJSON: toJSON(input.Meta)}
	if err := s.orders.CreateWalletOrder(ctx, &order); err != nil {
		return domain.WalletOrder{}, err
	}
	return order, nil
}

func (s *Service) ListUserOrders(ctx context.Context, userID int64, limit, offset int) ([]domain.WalletOrder, int, error) {
	return s.orders.ListWalletOrders(ctx, userID, limit, offset)
}

func (s *Service) ListAllOrders(ctx context.Context, status string, limit, offset int) ([]domain.WalletOrder, int, error) {
	return s.orders.ListAllWalletOrders(ctx, status, limit, offset)
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
	status := domain.WalletOrderPendingReview
	if !policy.RequireApproval {
		status = domain.WalletOrderApproved
	}
	order := domain.WalletOrder{UserID: userID, Type: domain.WalletOrderRefund, Amount: amount, Currency: "CNY", Status: status, Note: reason, MetaJSON: toJSON(meta)}
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
	meta := map[string]any{"vps_id": inst.ID, "order_item_id": item.ID, "refund_policy": policy, "reason": reason, "delete_on_approve": false, "trigger": "admin_delete"}
	status := domain.WalletOrderPendingReview
	if !policy.RequireApproval {
		status = domain.WalletOrderApproved
	}
	order := domain.WalletOrder{UserID: inst.UserID, Type: domain.WalletOrderRefund, Amount: amount, Currency: "CNY", Status: status, Note: reason, MetaJSON: toJSON(meta)}
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
	if err := s.orders.UpdateWalletOrderStatus(ctx, order.ID, domain.WalletOrderRejected, &adminID, reason); err != nil {
		return err
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
		return domain.Wallet{}, appshared.ErrInvalidInput
	}
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
	if err := cli.DeleteHost(ctx, hostID); err != nil {
		return err
	}
	_ = s.vps.UpdateInstanceStatus(ctx, inst.ID, domain.VPSStatusUnknown, inst.AutomationState)
	return nil
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
	baseAmount := inst.MonthlyPrice
	if baseAmount <= 0 {
		baseAmount = item.Amount
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
		if point.Percent < 0 {
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
	elapsedRatio := refundElapsedRatio(inst, now)
	if len(policy.Curve) > 0 {
		if ratio, ok := RefundCurveRatio(policy.Curve, elapsedRatio*100); ok {
			return int64(math.Round(float64(amount) * ratio))
		}
	}
	totalHours, ok := refundPeriodHours(inst)
	if !ok {
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
	fullRatio := refundRatioThreshold(totalHours, policy.FullHours, policy.FullDays)
	prorateRatio := refundRatioThreshold(totalHours, policy.ProrateHours, policy.ProrateDays)
	noRefundRatio := refundRatioThreshold(totalHours, policy.NoRefundHours, policy.NoRefundDays)
	if noRefundRatio > 0 && elapsedRatio > noRefundRatio {
		return 0
	}
	if fullRatio > 0 && elapsedRatio <= fullRatio {
		return amount
	}
	if prorateRatio > 0 && elapsedRatio <= prorateRatio {
		ratio := (prorateRatio - elapsedRatio) / prorateRatio
		return int64(math.Round(float64(amount) * ratio))
	}
	return 0
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

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
