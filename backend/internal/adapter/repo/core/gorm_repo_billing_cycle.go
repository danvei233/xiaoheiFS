package repo

import (
	"context"
	"time"

	"xiaoheiplay/internal/domain"
)

func (r *GormRepo) ListBillingCycles(ctx context.Context) ([]domain.BillingCycle, error) {

	var rows []billingCycleRow
	if err := r.gdb.WithContext(ctx).Order("sort_order, id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.BillingCycle, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.BillingCycle{
			ID:         row.ID,
			Name:       row.Name,
			Months:     row.Months,
			Multiplier: row.Multiplier,
			MinQty:     row.MinQty,
			MaxQty:     row.MaxQty,
			Active:     row.Active == 1,
			SortOrder:  row.SortOrder,
			CreatedAt:  row.CreatedAt,
			UpdatedAt:  row.UpdatedAt,
		})
	}
	return out, nil

}

func (r *GormRepo) GetBillingCycle(ctx context.Context, id int64) (domain.BillingCycle, error) {

	var row billingCycleRow
	if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return domain.BillingCycle{}, r.ensure(err)
	}
	return domain.BillingCycle{
		ID:         row.ID,
		Name:       row.Name,
		Months:     row.Months,
		Multiplier: row.Multiplier,
		MinQty:     row.MinQty,
		MaxQty:     row.MaxQty,
		Active:     row.Active == 1,
		SortOrder:  row.SortOrder,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}, nil

}

func (r *GormRepo) CreateBillingCycle(ctx context.Context, cycle *domain.BillingCycle) error {

	row := billingCycleRow{
		Name:       cycle.Name,
		Months:     cycle.Months,
		Multiplier: cycle.Multiplier,
		MinQty:     cycle.MinQty,
		MaxQty:     cycle.MaxQty,
		Active:     boolToInt(cycle.Active),
		SortOrder:  cycle.SortOrder,
	}
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	cycle.ID = row.ID
	return nil

}

func (r *GormRepo) UpdateBillingCycle(ctx context.Context, cycle domain.BillingCycle) error {

	return r.gdb.WithContext(ctx).Model(&billingCycleRow{}).Where("id = ?", cycle.ID).Updates(map[string]any{
		"name":       cycle.Name,
		"months":     cycle.Months,
		"multiplier": cycle.Multiplier,
		"min_qty":    cycle.MinQty,
		"max_qty":    cycle.MaxQty,
		"active":     boolToInt(cycle.Active),
		"sort_order": cycle.SortOrder,
		"updated_at": time.Now(),
	}).Error

}

func (r *GormRepo) DeleteBillingCycle(ctx context.Context, id int64) error {

	return r.gdb.WithContext(ctx).Delete(&billingCycleRow{}, id).Error

}
