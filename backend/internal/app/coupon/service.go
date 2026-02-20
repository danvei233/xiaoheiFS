package coupon

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

type Service struct {
	repo  appports.CouponRepository
	audit appports.AuditRepository
}

type QuoteItem struct {
	PackageID       int64
	GoodsTypeID     int64
	RegionID        int64
	PlanGroupID     int64
	AddonCore       int
	AddonMemGB      int
	AddonDiskGB     int
	AddonBWMbps     int
	UnitBaseAmount  int64
	UnitAddonAmount int64
	UnitAddonCore   int64
	UnitAddonMem    int64
	UnitAddonDisk   int64
	UnitAddonBW     int64
	UnitTotalAmount int64
	Qty             int
}

type ApplyResult struct {
	Coupon        domain.Coupon
	Group         domain.CouponProductGroup
	UnitDiscount  []int64
	TotalDiscount int64
}

func NewService(repo appports.CouponRepository, audit appports.AuditRepository) *Service {
	return &Service{repo: repo, audit: audit}
}

func (s *Service) ListProductGroups(ctx context.Context) ([]domain.CouponProductGroup, error) {
	return s.repo.ListCouponProductGroups(ctx)
}

func (s *Service) GetProductGroup(ctx context.Context, id int64) (domain.CouponProductGroup, error) {
	return s.repo.GetCouponProductGroup(ctx, id)
}

func (s *Service) CreateProductGroup(ctx context.Context, adminID int64, group *domain.CouponProductGroup) error {
	if err := validateGroup(*group); err != nil {
		return err
	}
	if err := s.repo.CreateCouponProductGroup(ctx, group); err != nil {
		return err
	}
	s.auditLog(ctx, adminID, "coupon_group.create", "coupon_group", group.ID)
	return nil
}

func (s *Service) UpdateProductGroup(ctx context.Context, adminID int64, group domain.CouponProductGroup) error {
	if err := validateGroup(group); err != nil {
		return err
	}
	if err := s.repo.UpdateCouponProductGroup(ctx, group); err != nil {
		return err
	}
	s.auditLog(ctx, adminID, "coupon_group.update", "coupon_group", group.ID)
	return nil
}

func (s *Service) DeleteProductGroup(ctx context.Context, adminID int64, id int64) error {
	if err := s.repo.DeleteCouponProductGroup(ctx, id); err != nil {
		return err
	}
	s.auditLog(ctx, adminID, "coupon_group.delete", "coupon_group", id)
	return nil
}

func (s *Service) ListCoupons(ctx context.Context, filter appshared.CouponFilter, limit, offset int) ([]domain.Coupon, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListCoupons(ctx, filter, limit, offset)
}

func (s *Service) GetCoupon(ctx context.Context, id int64) (domain.Coupon, error) {
	return s.repo.GetCoupon(ctx, id)
}

func (s *Service) CreateCoupon(ctx context.Context, adminID int64, coupon *domain.Coupon) error {
	if err := s.validateCoupon(ctx, *coupon); err != nil {
		return err
	}
	coupon.Code = normalizeCode(coupon.Code)
	if err := s.repo.CreateCoupon(ctx, coupon); err != nil {
		return err
	}
	s.auditLog(ctx, adminID, "coupon.create", "coupon", coupon.ID)
	return nil
}

func (s *Service) UpdateCoupon(ctx context.Context, adminID int64, coupon domain.Coupon) error {
	if err := s.validateCoupon(ctx, coupon); err != nil {
		return err
	}
	coupon.Code = normalizeCode(coupon.Code)
	if err := s.repo.UpdateCoupon(ctx, coupon); err != nil {
		return err
	}
	s.auditLog(ctx, adminID, "coupon.update", "coupon", coupon.ID)
	return nil
}

func (s *Service) DeleteCoupon(ctx context.Context, adminID int64, id int64) error {
	if err := s.repo.DeleteCoupon(ctx, id); err != nil {
		return err
	}
	s.auditLog(ctx, adminID, "coupon.delete", "coupon", id)
	return nil
}

type BatchGenerateInput struct {
	Prefix string
	Count  int
	Length int
	Coupon domain.Coupon
}

func (s *Service) BatchGenerateCoupons(ctx context.Context, adminID int64, prefix string, count int, length int, coupon domain.Coupon) ([]domain.Coupon, error) {
	return s.BatchGenerate(ctx, adminID, BatchGenerateInput{
		Prefix: prefix,
		Count:  count,
		Length: length,
		Coupon: coupon,
	})
}

func (s *Service) BatchGenerate(ctx context.Context, adminID int64, input BatchGenerateInput) ([]domain.Coupon, error) {
	if input.Count <= 0 || input.Count > 5000 {
		return nil, appshared.ErrInvalidInput
	}
	if input.Length <= 0 {
		input.Length = 8
	}
	if input.Length > 32 {
		return nil, appshared.ErrInvalidInput
	}
	if err := s.validateCoupon(ctx, input.Coupon); err != nil {
		return nil, err
	}
	prefix := strings.ToUpper(strings.TrimSpace(input.Prefix))
	out := make([]domain.Coupon, 0, input.Count)
	exists := map[string]struct{}{}
	maxTry := input.Count * 20
	for i := 0; i < maxTry && len(out) < input.Count; i++ {
		code := prefix + randomToken(input.Length)
		if _, ok := exists[code]; ok {
			continue
		}
		exists[code] = struct{}{}
		c := input.Coupon
		c.Code = code
		if err := s.repo.CreateCoupon(ctx, &c); err != nil {
			continue
		}
		out = append(out, c)
	}
	if len(out) == 0 {
		return nil, appshared.ErrConflict
	}
	s.auditLog(ctx, adminID, "coupon.batch_generate", "coupon", int64(len(out)))
	return out, nil
}

func (s *Service) PreviewDiscount(ctx context.Context, userID int64, code string, items []QuoteItem) (ApplyResult, error) {
	code = normalizeCode(code)
	if code == "" || len(items) == 0 {
		return ApplyResult{}, appshared.ErrInvalidInput
	}
	coupon, err := s.repo.GetCouponByCode(ctx, code)
	if err != nil {
		return ApplyResult{}, err
	}
	if !coupon.Active {
		return ApplyResult{}, appshared.ErrConflict
	}
	now := time.Now()
	if coupon.StartsAt != nil && now.Before(*coupon.StartsAt) {
		return ApplyResult{}, appshared.ErrConflict
	}
	if coupon.EndsAt != nil && now.After(*coupon.EndsAt) {
		return ApplyResult{}, appshared.ErrConflict
	}
	group, err := s.repo.GetCouponProductGroup(ctx, coupon.ProductGroupID)
	if err != nil {
		return ApplyResult{}, err
	}
	rules := parseCouponRules(group)
	if len(rules) == 0 {
		return ApplyResult{}, appshared.ErrConflict
	}
	activeStatuses := []string{domain.CouponRedemptionStatusApplied, domain.CouponRedemptionStatusConfirmed}
	if coupon.TotalLimit >= 0 {
		used, err := s.repo.CountCouponRedemptions(ctx, coupon.ID, nil, activeStatuses)
		if err != nil {
			return ApplyResult{}, err
		}
		if used >= int64(coupon.TotalLimit) {
			return ApplyResult{}, appshared.ErrConflict
		}
	}
	if coupon.PerUserLimit >= 0 {
		uid := userID
		used, err := s.repo.CountCouponRedemptions(ctx, coupon.ID, &uid, activeStatuses)
		if err != nil {
			return ApplyResult{}, err
		}
		if used >= int64(coupon.PerUserLimit) {
			return ApplyResult{}, appshared.ErrConflict
		}
	}
	if coupon.NewUserOnly {
		okCnt, err := s.repo.CountUserSuccessfulOrders(ctx, userID)
		if err != nil {
			return ApplyResult{}, err
		}
		if okCnt > 0 {
			return ApplyResult{}, appshared.ErrConflict
		}
	}
	unitDiscounts := make([]int64, len(items))
	var totalDiscount int64
	for i, item := range items {
		discountable, matched := groupDiscountableAmount(rules, item)
		if !matched || discountable <= 0 {
			continue
		}
		disc := int64(math.Round(float64(discountable) * float64(coupon.DiscountPermille) / 1000.0))
		if disc < 0 {
			disc = 0
		}
		if disc > item.UnitTotalAmount {
			disc = item.UnitTotalAmount
		}
		unitDiscounts[i] = disc
		qty := item.Qty
		if qty <= 0 {
			qty = 1
		}
		totalDiscount += disc * int64(qty)
	}
	if totalDiscount <= 0 {
		return ApplyResult{}, appshared.ErrConflict
	}
	return ApplyResult{
		Coupon:        coupon,
		Group:         group,
		UnitDiscount:  unitDiscounts,
		TotalDiscount: totalDiscount,
	}, nil
}

func (s *Service) MarkOrderCanceled(ctx context.Context, orderID int64) error {
	return s.repo.UpdateCouponRedemptionStatusByOrder(ctx, orderID, []string{
		domain.CouponRedemptionStatusApplied,
	}, domain.CouponRedemptionStatusCanceled)
}

func (s *Service) CreateRedemption(ctx context.Context, redemption *domain.CouponRedemption) error {
	if redemption == nil || redemption.CouponID <= 0 || redemption.OrderID <= 0 || redemption.UserID <= 0 {
		return appshared.ErrInvalidInput
	}
	if strings.TrimSpace(redemption.Status) == "" {
		redemption.Status = domain.CouponRedemptionStatusApplied
	}
	return s.repo.CreateCouponRedemption(ctx, redemption)
}

func (s *Service) MarkOrderConfirmed(ctx context.Context, orderID int64) error {
	return s.repo.UpdateCouponRedemptionStatusByOrder(ctx, orderID, []string{
		domain.CouponRedemptionStatusApplied,
	}, domain.CouponRedemptionStatusConfirmed)
}

func (s *Service) validateCoupon(ctx context.Context, coupon domain.Coupon) error {
	if normalizeCode(coupon.Code) == "" {
		return appshared.ErrInvalidInput
	}
	if coupon.DiscountPermille <= 0 || coupon.DiscountPermille > 1000 {
		return appshared.ErrInvalidInput
	}
	if coupon.ProductGroupID <= 0 {
		return appshared.ErrInvalidInput
	}
	if coupon.TotalLimit < -1 || coupon.PerUserLimit < -1 {
		return appshared.ErrInvalidInput
	}
	if coupon.StartsAt != nil && coupon.EndsAt != nil && coupon.StartsAt.After(*coupon.EndsAt) {
		return appshared.ErrInvalidInput
	}
	if _, err := s.repo.GetCouponProductGroup(ctx, coupon.ProductGroupID); err != nil {
		return err
	}
	return nil
}

func validateGroup(group domain.CouponProductGroup) error {
	if strings.TrimSpace(group.Name) == "" {
		return appshared.ErrInvalidInput
	}
	rules := parseCouponRules(group)
	if len(rules) == 0 {
		return appshared.ErrInvalidInput
	}
	for _, rule := range rules {
		if err := validateCouponRule(rule); err != nil {
			return appshared.ErrInvalidInput
		}
	}
	return nil
}

func validateCouponRule(rule domain.CouponProductRule) error {
	switch rule.Scope {
	case domain.CouponGroupScopeAll, domain.CouponGroupScopeAllAddons:
		return nil
	case domain.CouponGroupScopeGoodsType:
		if rule.GoodsTypeID <= 0 {
			return appshared.ErrInvalidInput
		}
		return nil
	case domain.CouponGroupScopeGoodsTypeRegion:
		if rule.GoodsTypeID <= 0 || rule.RegionID <= 0 {
			return appshared.ErrInvalidInput
		}
		return nil
	case domain.CouponGroupScopePlanGroup:
		if rule.GoodsTypeID <= 0 || rule.RegionID <= 0 || rule.PlanGroupID <= 0 {
			return appshared.ErrInvalidInput
		}
		return nil
	case domain.CouponGroupScopePackage:
		if rule.GoodsTypeID <= 0 || rule.RegionID <= 0 || rule.PlanGroupID <= 0 || rule.PackageID <= 0 {
			return appshared.ErrInvalidInput
		}
		return nil
	case domain.CouponGroupScopeAddonConfig:
		if rule.GoodsTypeID <= 0 || rule.RegionID <= 0 || rule.PlanGroupID <= 0 {
			return appshared.ErrInvalidInput
		}
		return nil
	default:
		return appshared.ErrInvalidInput
	}
}

func groupDiscountableAmount(rules []domain.CouponProductRule, item QuoteItem) (int64, bool) {
	bestScore := -1
	var bestAmount int64
	for _, rule := range rules {
		amount, ok := ruleDiscountableAmount(rule, item)
		if !ok || amount <= 0 {
			continue
		}
		score := couponRuleSpecificity(rule.Scope)
		if score > bestScore {
			bestScore = score
			bestAmount = amount
		}
	}
	if bestScore < 0 {
		return 0, false
	}
	return bestAmount, true
}

func ruleDiscountableAmount(rule domain.CouponProductRule, item QuoteItem) (int64, bool) {
	switch rule.Scope {
	case domain.CouponGroupScopeAll:
		return item.UnitBaseAmount, item.UnitBaseAmount > 0
	case domain.CouponGroupScopeAllAddons:
		return item.UnitAddonAmount, item.UnitAddonAmount > 0
	case domain.CouponGroupScopeGoodsType:
		if rule.GoodsTypeID > 0 && rule.GoodsTypeID == item.GoodsTypeID {
			return item.UnitTotalAmount, true
		}
		return 0, false
	case domain.CouponGroupScopeGoodsTypeRegion:
		if rule.GoodsTypeID == item.GoodsTypeID && rule.RegionID == item.RegionID {
			return item.UnitTotalAmount, true
		}
		return 0, false
	case domain.CouponGroupScopePlanGroup:
		if rule.PlanGroupID <= 0 || rule.PlanGroupID != item.PlanGroupID {
			return 0, false
		}
		if rule.GoodsTypeID > 0 && rule.GoodsTypeID != item.GoodsTypeID {
			return 0, false
		}
		if rule.RegionID > 0 && rule.RegionID != item.RegionID {
			return 0, false
		}
		return item.UnitTotalAmount, true
	case domain.CouponGroupScopePackage:
		if rule.PackageID <= 0 || rule.PackageID != item.PackageID {
			return 0, false
		}
		if rule.PlanGroupID > 0 && rule.PlanGroupID != item.PlanGroupID {
			return 0, false
		}
		if rule.RegionID > 0 && rule.RegionID != item.RegionID {
			return 0, false
		}
		if rule.GoodsTypeID > 0 && rule.GoodsTypeID != item.GoodsTypeID {
			return 0, false
		}
		return item.UnitTotalAmount, true
	case domain.CouponGroupScopeAddonConfig:
		if rule.PlanGroupID <= 0 || rule.PlanGroupID != item.PlanGroupID {
			return 0, false
		}
		if rule.RegionID > 0 && rule.RegionID != item.RegionID {
			return 0, false
		}
		if rule.GoodsTypeID > 0 && rule.GoodsTypeID != item.GoodsTypeID {
			return 0, false
		}
		var part int64
		if rule.AddonCoreEnabled {
			part += item.UnitAddonCore
		}
		if rule.AddonMemEnabled {
			part += item.UnitAddonMem
		}
		if rule.AddonDiskEnabled {
			part += item.UnitAddonDisk
		}
		if rule.AddonBWEnabled {
			part += item.UnitAddonBW
		}
		if !rule.AddonCoreEnabled && !rule.AddonMemEnabled && !rule.AddonDiskEnabled && !rule.AddonBWEnabled {
			part = item.UnitAddonAmount
		}
		return part, part > 0
	default:
		return 0, false
	}
}

func couponRuleSpecificity(scope domain.CouponGroupScope) int {
	switch scope {
	case domain.CouponGroupScopePackage:
		return 60
	case domain.CouponGroupScopePlanGroup, domain.CouponGroupScopeAddonConfig:
		return 50
	case domain.CouponGroupScopeGoodsTypeRegion:
		return 40
	case domain.CouponGroupScopeGoodsType:
		return 30
	case domain.CouponGroupScopeAllAddons:
		return 20
	case domain.CouponGroupScopeAll:
		return 10
	default:
		return 0
	}
}

func normalizeCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func parseCouponRules(group domain.CouponProductGroup) []domain.CouponProductRule {
	raw := strings.TrimSpace(group.RulesJSON)
	if raw != "" {
		var rules []domain.CouponProductRule
		if err := json.Unmarshal([]byte(raw), &rules); err == nil {
			out := make([]domain.CouponProductRule, 0, len(rules))
			for _, rule := range rules {
				if rule.Scope == "" {
					continue
				}
				out = append(out, rule)
			}
			if len(out) > 0 {
				return out
			}
		}
	}
	scope := group.Scope
	if scope == "" {
		scope = domain.CouponGroupScopeAll
	}
	return []domain.CouponProductRule{{
		Scope:            scope,
		GoodsTypeID:      group.GoodsTypeID,
		RegionID:         group.RegionID,
		PlanGroupID:      group.PlanGroupID,
		PackageID:        group.PackageID,
		AddonCoreEnabled: group.AddonCore > 0,
		AddonMemEnabled:  group.AddonMemGB > 0,
		AddonDiskEnabled: group.AddonDiskGB > 0,
		AddonBWEnabled:   group.AddonBWMbps > 0,
	}}
}

func randomToken(n int) string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	if n <= 0 {
		return ""
	}
	out := make([]byte, n)
	max := big.NewInt(int64(len(alphabet)))
	for i := range out {
		v, err := rand.Int(rand.Reader, max)
		if err != nil {
			out[i] = alphabet[0]
			continue
		}
		out[i] = alphabet[v.Int64()]
	}
	return string(out)
}

func (s *Service) auditLog(ctx context.Context, adminID int64, action, targetType string, targetID int64) {
	if s.audit == nil {
		return
	}
	_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{
		AdminID:    adminID,
		Action:     action,
		TargetType: targetType,
		TargetID:   strconv.FormatInt(targetID, 10),
		DetailJSON: "{}",
	})
}
