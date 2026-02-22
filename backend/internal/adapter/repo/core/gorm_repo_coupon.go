package repo

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

func (r *GormRepo) ListCouponProductGroups(ctx context.Context) ([]domain.CouponProductGroup, error) {
	var rows []couponProductGroupRow
	if err := r.gdb.WithContext(ctx).Order("id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.CouponProductGroup, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.CouponProductGroup{
			ID:          row.ID,
			Name:        row.Name,
			RulesJSON:   normalizeCouponGroupRulesJSON(row.RulesJSON, row.Scope, row.GoodsTypeID, row.RegionID, row.PlanGroupID, row.PackageID, row.AddonCore, row.AddonMemGB, row.AddonDiskGB, row.AddonBWMbps),
			Scope:       domain.CouponGroupScope(row.Scope),
			GoodsTypeID: row.GoodsTypeID,
			RegionID:    row.RegionID,
			PlanGroupID: row.PlanGroupID,
			PackageID:   row.PackageID,
			AddonCore:   row.AddonCore,
			AddonMemGB:  row.AddonMemGB,
			AddonDiskGB: row.AddonDiskGB,
			AddonBWMbps: row.AddonBWMbps,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		})
	}
	return out, nil
}

func (r *GormRepo) GetCouponProductGroup(ctx context.Context, id int64) (domain.CouponProductGroup, error) {
	var row couponProductGroupRow
	if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return domain.CouponProductGroup{}, r.ensure(err)
	}
	return domain.CouponProductGroup{
		ID:          row.ID,
		Name:        row.Name,
		RulesJSON:   normalizeCouponGroupRulesJSON(row.RulesJSON, row.Scope, row.GoodsTypeID, row.RegionID, row.PlanGroupID, row.PackageID, row.AddonCore, row.AddonMemGB, row.AddonDiskGB, row.AddonBWMbps),
		Scope:       domain.CouponGroupScope(row.Scope),
		GoodsTypeID: row.GoodsTypeID,
		RegionID:    row.RegionID,
		PlanGroupID: row.PlanGroupID,
		PackageID:   row.PackageID,
		AddonCore:   row.AddonCore,
		AddonMemGB:  row.AddonMemGB,
		AddonDiskGB: row.AddonDiskGB,
		AddonBWMbps: row.AddonBWMbps,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}

func (r *GormRepo) CreateCouponProductGroup(ctx context.Context, group *domain.CouponProductGroup) error {
	row := couponProductGroupRow{
		Name:        group.Name,
		RulesJSON:   normalizeCouponGroupRulesJSON(group.RulesJSON, string(group.Scope), group.GoodsTypeID, group.RegionID, group.PlanGroupID, group.PackageID, group.AddonCore, group.AddonMemGB, group.AddonDiskGB, group.AddonBWMbps),
		Scope:       string(group.Scope),
		GoodsTypeID: group.GoodsTypeID,
		RegionID:    group.RegionID,
		PlanGroupID: group.PlanGroupID,
		PackageID:   group.PackageID,
		AddonCore:   group.AddonCore,
		AddonMemGB:  group.AddonMemGB,
		AddonDiskGB: group.AddonDiskGB,
		AddonBWMbps: group.AddonBWMbps,
	}
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	group.ID = row.ID
	group.CreatedAt = row.CreatedAt
	group.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *GormRepo) UpdateCouponProductGroup(ctx context.Context, group domain.CouponProductGroup) error {
	return r.gdb.WithContext(ctx).Model(&couponProductGroupRow{}).Where("id = ?", group.ID).Updates(map[string]any{
		"name":          group.Name,
		"rules_json":    normalizeCouponGroupRulesJSON(group.RulesJSON, string(group.Scope), group.GoodsTypeID, group.RegionID, group.PlanGroupID, group.PackageID, group.AddonCore, group.AddonMemGB, group.AddonDiskGB, group.AddonBWMbps),
		"scope":         string(group.Scope),
		"goods_type_id": group.GoodsTypeID,
		"region_id":     group.RegionID,
		"plan_group_id": group.PlanGroupID,
		"package_id":    group.PackageID,
		"addon_core":    group.AddonCore,
		"addon_mem_gb":  group.AddonMemGB,
		"addon_disk_gb": group.AddonDiskGB,
		"addon_bw_mbps": group.AddonBWMbps,
		"updated_at":    time.Now(),
	}).Error
}

func (r *GormRepo) DeleteCouponProductGroup(ctx context.Context, id int64) error {
	return r.gdb.WithContext(ctx).Delete(&couponProductGroupRow{}, "id = ?", id).Error
}

func (r *GormRepo) ListCoupons(ctx context.Context, filter appshared.CouponFilter, limit, offset int) ([]domain.Coupon, int, error) {
	q := r.gdb.WithContext(ctx).Model(&couponRow{})
	if v := strings.TrimSpace(filter.Keyword); v != "" {
		like := "%" + strings.ToUpper(v) + "%"
		q = q.Where("UPPER(code) LIKE ? OR note LIKE ?", like, "%"+v+"%")
	}
	if filter.ProductGroupID > 0 {
		q = q.Where("product_group_id = ?", filter.ProductGroupID)
	}
	if filter.Active != nil {
		q = q.Where("active = ?", boolToInt(*filter.Active))
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []couponRow
	if err := q.Order("id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.Coupon, 0, len(rows))
	for _, row := range rows {
		out = append(out, couponFromRow(row))
	}
	return out, int(total), nil
}

func (r *GormRepo) GetCoupon(ctx context.Context, id int64) (domain.Coupon, error) {
	var row couponRow
	if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return domain.Coupon{}, r.ensure(err)
	}
	return couponFromRow(row), nil
}

func (r *GormRepo) GetCouponByCode(ctx context.Context, code string) (domain.Coupon, error) {
	var row couponRow
	if err := r.gdb.WithContext(ctx).Where("code = ?", strings.ToUpper(strings.TrimSpace(code))).First(&row).Error; err != nil {
		return domain.Coupon{}, r.ensure(err)
	}
	return couponFromRow(row), nil
}

func (r *GormRepo) CreateCoupon(ctx context.Context, coupon *domain.Coupon) error {
	row := couponRow{
		Code:             strings.ToUpper(strings.TrimSpace(coupon.Code)),
		DiscountPermille: coupon.DiscountPermille,
		ProductGroupID:   coupon.ProductGroupID,
		TotalLimit:       coupon.TotalLimit,
		PerUserLimit:     coupon.PerUserLimit,
		StartsAt:         coupon.StartsAt,
		EndsAt:           coupon.EndsAt,
		NewUserOnly:      boolToInt(coupon.NewUserOnly),
		Active:           boolToInt(coupon.Active),
		Note:             coupon.Note,
	}
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	*coupon = couponFromRow(row)
	return nil
}

func (r *GormRepo) UpdateCoupon(ctx context.Context, coupon domain.Coupon) error {
	return r.gdb.WithContext(ctx).Model(&couponRow{}).Where("id = ?", coupon.ID).Updates(map[string]any{
		"code":              strings.ToUpper(strings.TrimSpace(coupon.Code)),
		"discount_permille": coupon.DiscountPermille,
		"product_group_id":  coupon.ProductGroupID,
		"total_limit":       coupon.TotalLimit,
		"per_user_limit":    coupon.PerUserLimit,
		"starts_at":         coupon.StartsAt,
		"ends_at":           coupon.EndsAt,
		"new_user_only":     boolToInt(coupon.NewUserOnly),
		"active":            boolToInt(coupon.Active),
		"note":              coupon.Note,
		"updated_at":        time.Now(),
	}).Error
}

func (r *GormRepo) DeleteCoupon(ctx context.Context, id int64) error {
	return r.gdb.WithContext(ctx).Delete(&couponRow{}, "id = ?", id).Error
}

func (r *GormRepo) CountCouponRedemptions(ctx context.Context, couponID int64, userID *int64, statuses []string) (int64, error) {
	q := r.gdb.WithContext(ctx).Model(&couponRedemptionRow{}).Where("coupon_id = ?", couponID)
	if userID != nil && *userID > 0 {
		q = q.Where("user_id = ?", *userID)
	}
	if len(statuses) > 0 {
		q = q.Where("status IN ?", statuses)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

func (r *GormRepo) CreateCouponRedemption(ctx context.Context, redemption *domain.CouponRedemption) error {
	row := couponRedemptionRow{
		CouponID:       redemption.CouponID,
		OrderID:        redemption.OrderID,
		UserID:         redemption.UserID,
		Status:         redemption.Status,
		DiscountAmount: redemption.DiscountAmount,
	}
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	redemption.ID = row.ID
	redemption.CreatedAt = row.CreatedAt
	redemption.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *GormRepo) UpdateCouponRedemptionStatusByOrder(ctx context.Context, orderID int64, fromStatuses []string, toStatus string) error {
	q := r.gdb.WithContext(ctx).Model(&couponRedemptionRow{}).Where("order_id = ?", orderID)
	if len(fromStatuses) > 0 {
		q = q.Where("status IN ?", fromStatuses)
	}
	return q.Updates(map[string]any{
		"status":     toStatus,
		"updated_at": time.Now(),
	}).Error
}

func (r *GormRepo) CountUserSuccessfulOrders(ctx context.Context, userID int64) (int64, error) {
	var total int64
	if err := r.gdb.WithContext(ctx).Model(&orderRow{}).
		Where("user_id = ? AND status IN ?", userID, []string{
			string(domain.OrderStatusApproved),
			string(domain.OrderStatusProvisioning),
			string(domain.OrderStatusActive),
		}).
		Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

func couponFromRow(row couponRow) domain.Coupon {
	return domain.Coupon{
		ID:               row.ID,
		Code:             row.Code,
		DiscountPermille: row.DiscountPermille,
		ProductGroupID:   row.ProductGroupID,
		TotalLimit:       row.TotalLimit,
		PerUserLimit:     row.PerUserLimit,
		StartsAt:         row.StartsAt,
		EndsAt:           row.EndsAt,
		NewUserOnly:      row.NewUserOnly == 1,
		Active:           row.Active == 1,
		Note:             row.Note,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
}

func normalizeCouponGroupRulesJSON(rulesJSON, fallbackScope string, goodsTypeID, regionID, planGroupID, packageID int64, addonCore, addonMem, addonDisk, addonBW int) string {
	raw := strings.TrimSpace(rulesJSON)
	if raw != "" && raw != "null" {
		var rows []domain.CouponProductRule
		if err := json.Unmarshal([]byte(raw), &rows); err == nil && len(rows) > 0 {
			out, _ := json.Marshal(rows)
			return string(out)
		}
	}
	rule := domain.CouponProductRule{
		Scope:       domain.CouponGroupScope(fallbackScope),
		GoodsTypeID: goodsTypeID,
		RegionID:    regionID,
		PlanGroupID: planGroupID,
		PackageID:   packageID,
	}
	if rule.Scope == domain.CouponGroupScopeAddonConfig {
		rule.AddonCoreEnabled = addonCore > 0
		rule.AddonMemEnabled = addonMem > 0
		rule.AddonDiskEnabled = addonDisk > 0
		rule.AddonBWEnabled = addonBW > 0
	}
	if rule.Scope == "" {
		rule.Scope = domain.CouponGroupScopeAll
	}
	out, _ := json.Marshal([]domain.CouponProductRule{rule})
	return string(out)
}
