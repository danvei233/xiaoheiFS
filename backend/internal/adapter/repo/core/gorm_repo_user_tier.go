package repo

import (
	"context"
	"time"

	"gorm.io/gorm/clause"

	"xiaoheiplay/internal/domain"
)

func (r *GormRepo) ListUserTierGroups(ctx context.Context) ([]domain.UserTierGroup, error) {
	var rows []userTierGroupRow
	if err := r.gdb.WithContext(ctx).Order("priority DESC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.UserTierGroup, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.UserTierGroup{
			ID:                 row.ID,
			Name:               row.Name,
			Color:              row.Color,
			Icon:               row.Icon,
			Priority:           row.Priority,
			AutoApproveEnabled: row.AutoApproveEnabled == 1,
			IsDefault:          row.IsDefault == 1,
			CreatedAt:          row.CreatedAt,
			UpdatedAt:          row.UpdatedAt,
		})
	}
	return out, nil
}

func (r *GormRepo) GetUserTierGroup(ctx context.Context, id int64) (domain.UserTierGroup, error) {
	var row userTierGroupRow
	if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return domain.UserTierGroup{}, r.ensure(err)
	}
	return domain.UserTierGroup{
		ID:                 row.ID,
		Name:               row.Name,
		Color:              row.Color,
		Icon:               row.Icon,
		Priority:           row.Priority,
		AutoApproveEnabled: row.AutoApproveEnabled == 1,
		IsDefault:          row.IsDefault == 1,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}, nil
}

func (r *GormRepo) CreateUserTierGroup(ctx context.Context, group *domain.UserTierGroup) error {
	row := userTierGroupRow{
		Name:               group.Name,
		Color:              group.Color,
		Icon:               group.Icon,
		Priority:           group.Priority,
		AutoApproveEnabled: boolToInt(group.AutoApproveEnabled),
		IsDefault:          boolToInt(group.IsDefault),
	}
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	group.ID = row.ID
	group.CreatedAt = row.CreatedAt
	group.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *GormRepo) UpdateUserTierGroup(ctx context.Context, group domain.UserTierGroup) error {
	return r.gdb.WithContext(ctx).Model(&userTierGroupRow{}).Where("id = ?", group.ID).Updates(map[string]any{
		"name":                 group.Name,
		"color":                group.Color,
		"icon":                 group.Icon,
		"priority":             group.Priority,
		"auto_approve_enabled": boolToInt(group.AutoApproveEnabled),
		"is_default":           boolToInt(group.IsDefault),
		"updated_at":           time.Now(),
	}).Error
}

func (r *GormRepo) DeleteUserTierGroup(ctx context.Context, id int64) error {
	return r.gdb.WithContext(ctx).Delete(&userTierGroupRow{}, "id = ?", id).Error
}

func (r *GormRepo) ListUserTierDiscountRules(ctx context.Context, groupID int64) ([]domain.UserTierDiscountRule, error) {
	var rows []userTierDiscountRuleRow
	if err := r.gdb.WithContext(ctx).Where("group_id = ?", groupID).Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.UserTierDiscountRule, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.UserTierDiscountRule{
			ID:               row.ID,
			GroupID:          row.GroupID,
			Scope:            domain.UserTierScope(row.Scope),
			GoodsTypeID:      row.GoodsTypeID,
			RegionID:         row.RegionID,
			PlanGroupID:      row.PlanGroupID,
			PackageID:        row.PackageID,
			DiscountPermille: row.DiscountPermille,
			FixedPrice:       row.FixedPrice,
			AddCorePermille:  row.AddCorePermille,
			AddMemPermille:   row.AddMemPermille,
			AddDiskPermille:  row.AddDiskPermille,
			AddBWPermille:    row.AddBWPermille,
			CreatedAt:        row.CreatedAt,
			UpdatedAt:        row.UpdatedAt,
		})
	}
	return out, nil
}

func (r *GormRepo) CreateUserTierDiscountRule(ctx context.Context, rule *domain.UserTierDiscountRule) error {
	row := userTierDiscountRuleRow{
		GroupID:          rule.GroupID,
		Scope:            string(rule.Scope),
		GoodsTypeID:      rule.GoodsTypeID,
		RegionID:         rule.RegionID,
		PlanGroupID:      rule.PlanGroupID,
		PackageID:        rule.PackageID,
		DiscountPermille: rule.DiscountPermille,
		FixedPrice:       rule.FixedPrice,
		AddCorePermille:  rule.AddCorePermille,
		AddMemPermille:   rule.AddMemPermille,
		AddDiskPermille:  rule.AddDiskPermille,
		AddBWPermille:    rule.AddBWPermille,
	}
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	rule.ID = row.ID
	rule.CreatedAt = row.CreatedAt
	rule.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *GormRepo) UpdateUserTierDiscountRule(ctx context.Context, rule domain.UserTierDiscountRule) error {
	return r.gdb.WithContext(ctx).Model(&userTierDiscountRuleRow{}).Where("id = ?", rule.ID).Updates(map[string]any{
		"group_id":          rule.GroupID,
		"scope":             string(rule.Scope),
		"goods_type_id":     rule.GoodsTypeID,
		"region_id":         rule.RegionID,
		"plan_group_id":     rule.PlanGroupID,
		"package_id":        rule.PackageID,
		"discount_permille": rule.DiscountPermille,
		"fixed_price":       rule.FixedPrice,
		"add_core_permille": rule.AddCorePermille,
		"add_mem_permille":  rule.AddMemPermille,
		"add_disk_permille": rule.AddDiskPermille,
		"add_bw_permille":   rule.AddBWPermille,
		"updated_at":        time.Now(),
	}).Error
}

func (r *GormRepo) DeleteUserTierDiscountRule(ctx context.Context, id int64) error {
	return r.gdb.WithContext(ctx).Delete(&userTierDiscountRuleRow{}, "id = ?", id).Error
}

func (r *GormRepo) ListUserTierAutoRules(ctx context.Context, groupID int64) ([]domain.UserTierAutoRule, error) {
	var rows []userTierAutoRuleRow
	if err := r.gdb.WithContext(ctx).Where("group_id = ?", groupID).Order("sort_order ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.UserTierAutoRule, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.UserTierAutoRule{
			ID:             row.ID,
			GroupID:        row.GroupID,
			DurationDays:   row.DurationDays,
			ConditionsJSON: row.ConditionsJSON,
			SortOrder:      row.SortOrder,
			CreatedAt:      row.CreatedAt,
			UpdatedAt:      row.UpdatedAt,
		})
	}
	return out, nil
}

func (r *GormRepo) CreateUserTierAutoRule(ctx context.Context, rule *domain.UserTierAutoRule) error {
	row := userTierAutoRuleRow{
		GroupID:        rule.GroupID,
		DurationDays:   rule.DurationDays,
		ConditionsJSON: rule.ConditionsJSON,
		SortOrder:      rule.SortOrder,
	}
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	rule.ID = row.ID
	rule.CreatedAt = row.CreatedAt
	rule.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *GormRepo) UpdateUserTierAutoRule(ctx context.Context, rule domain.UserTierAutoRule) error {
	return r.gdb.WithContext(ctx).Model(&userTierAutoRuleRow{}).Where("id = ?", rule.ID).Updates(map[string]any{
		"group_id":        rule.GroupID,
		"duration_days":   rule.DurationDays,
		"conditions_json": rule.ConditionsJSON,
		"sort_order":      rule.SortOrder,
		"updated_at":      time.Now(),
	}).Error
}

func (r *GormRepo) DeleteUserTierAutoRule(ctx context.Context, id int64) error {
	return r.gdb.WithContext(ctx).Delete(&userTierAutoRuleRow{}, "id = ?", id).Error
}

func (r *GormRepo) GetUserTierMembership(ctx context.Context, userID int64) (domain.UserTierMembership, error) {
	var row userTierMembershipRow
	if err := r.gdb.WithContext(ctx).Where("user_id = ?", userID).First(&row).Error; err != nil {
		return domain.UserTierMembership{}, r.ensure(err)
	}
	return domain.UserTierMembership{
		UserID:    row.UserID,
		GroupID:   row.GroupID,
		Source:    row.Source,
		ExpiresAt: row.ExpiresAt,
		UpdatedAt: row.UpdatedAt,
	}, nil
}

func (r *GormRepo) UpsertUserTierMembership(ctx context.Context, item *domain.UserTierMembership) error {
	row := userTierMembershipRow{
		UserID:    item.UserID,
		GroupID:   item.GroupID,
		Source:    item.Source,
		ExpiresAt: item.ExpiresAt,
		UpdatedAt: time.Now(),
	}
	if err := r.gdb.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"group_id", "source", "expires_at", "updated_at"}),
	}).Create(&row).Error; err != nil {
		return err
	}
	item.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *GormRepo) ClearUserTierMembership(ctx context.Context, userID int64) error {
	return r.gdb.WithContext(ctx).Delete(&userTierMembershipRow{}, "user_id = ?", userID).Error
}

func (r *GormRepo) ListExpiredUserTierMemberships(ctx context.Context, now time.Time, limit int) ([]domain.UserTierMembership, error) {
	if limit <= 0 {
		limit = 200
	}
	var rows []userTierMembershipRow
	if err := r.gdb.WithContext(ctx).
		Where("expires_at IS NOT NULL AND expires_at <= ?", now).
		Order("updated_at ASC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.UserTierMembership, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.UserTierMembership{
			UserID:    row.UserID,
			GroupID:   row.GroupID,
			Source:    row.Source,
			ExpiresAt: row.ExpiresAt,
			UpdatedAt: row.UpdatedAt,
		})
	}
	return out, nil
}

func (r *GormRepo) GetUserTierPriceCache(ctx context.Context, groupID int64, packageID int64) (domain.UserTierPriceCache, error) {
	var row userTierPriceCacheRow
	if err := r.gdb.WithContext(ctx).Where("group_id = ? AND package_id = ?", groupID, packageID).First(&row).Error; err != nil {
		return domain.UserTierPriceCache{}, r.ensure(err)
	}
	return domain.UserTierPriceCache{
		ID:           row.ID,
		GroupID:      row.GroupID,
		PackageID:    row.PackageID,
		MonthlyPrice: row.MonthlyPrice,
		UnitCore:     row.UnitCore,
		UnitMem:      row.UnitMem,
		UnitDisk:     row.UnitDisk,
		UnitBW:       row.UnitBW,
		UpdatedAt:    row.UpdatedAt,
	}, nil
}

func (r *GormRepo) DeleteUserTierPriceCachesByGroup(ctx context.Context, groupID int64) error {
	return r.gdb.WithContext(ctx).Where("group_id = ?", groupID).Delete(&userTierPriceCacheRow{}).Error
}

func (r *GormRepo) UpsertUserTierPriceCaches(ctx context.Context, items []domain.UserTierPriceCache) error {
	if len(items) == 0 {
		return nil
	}
	rows := make([]userTierPriceCacheRow, 0, len(items))
	now := time.Now()
	for _, item := range items {
		rows = append(rows, userTierPriceCacheRow{
			GroupID:      item.GroupID,
			PackageID:    item.PackageID,
			MonthlyPrice: item.MonthlyPrice,
			UnitCore:     item.UnitCore,
			UnitMem:      item.UnitMem,
			UnitDisk:     item.UnitDisk,
			UnitBW:       item.UnitBW,
			UpdatedAt:    now,
		})
	}
	return r.gdb.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "group_id"}, {Name: "package_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"monthly_price", "unit_core", "unit_mem", "unit_disk", "unit_bw", "updated_at"}),
	}).Create(&rows).Error
}
