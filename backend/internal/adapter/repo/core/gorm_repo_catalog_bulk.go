package repo

import (
	"context"

	"gorm.io/gorm"

	appshared "xiaoheiplay/internal/app/shared"
)

func (r *GormRepo) BulkDeleteRegions(ctx context.Context, ids []int64) error {
	return r.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, id := range ids {
			var refs int64
			if err := tx.Model(&planGroupRow{}).Where("region_id = ?", id).Count(&refs).Error; err != nil {
				return err
			}
			if refs > 0 {
				return appshared.ErrConflict
			}
			if err := tx.Delete(&regionRow{}, id).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *GormRepo) BulkDeletePlanGroups(ctx context.Context, ids []int64) error {
	return r.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, id := range ids {
			var refs int64
			if err := tx.Model(&packageRow{}).Where("plan_group_id = ?", id).Count(&refs).Error; err != nil {
				return err
			}
			if refs > 0 {
				return appshared.ErrConflict
			}
			if err := tx.Delete(&planGroupRow{}, id).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *GormRepo) BulkDeletePackages(ctx context.Context, ids []int64) error {
	return r.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, id := range ids {
			for _, model := range []any{&cartItemRow{}, &orderItemRow{}, &vpsInstanceRow{}} {
				var refs int64
				if err := tx.Model(model).Where("package_id = ?", id).Count(&refs).Error; err != nil {
					return err
				}
				if refs > 0 {
					return appshared.ErrConflict
				}
			}
			if err := tx.Delete(&packageRow{}, id).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *GormRepo) BulkDeleteBillingCycles(ctx context.Context, ids []int64) error {
	return r.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, id := range ids {
			if err := tx.Delete(&billingCycleRow{}, id).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *GormRepo) BulkDeleteSystemImages(ctx context.Context, ids []int64) error {
	return r.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, id := range ids {
			if err := tx.Where("system_image_id = ?", id).Delete(&lineSystemImageRow{}).Error; err != nil {
				return err
			}
			if err := tx.Delete(&systemImageRow{}, id).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
