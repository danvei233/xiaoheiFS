package repo

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	appports "xiaoheiplay/internal/app/ports"
	"xiaoheiplay/internal/domain"
)

func (r *GormRepo) CreateInstance(ctx context.Context, inst *domain.VPSInstance) error {
	row := toVPSInstanceRow(*inst)
	query := r.gdb.WithContext(ctx)
	if row.OrderItemID == nil {
		if err := query.Create(&row).Error; err != nil {
			return err
		}
		*inst = fromVPSInstanceRow(row)
		return nil
	}
	// Idempotency must be scoped to the order_item_id conflict ONLY. A bare
	// `OnConflict{DoNothing: true}` would silently swallow a conflict on any
	// other unique constraint and then re-read by order_item_id, returning a row
	// that is not the one the caller asked for. Targeting the column makes any
	// other unique violation surface as a real error.
	result := query.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "order_item_id"}},
		DoNothing: true,
	}).Create(&row)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		// A row with this order_item_id already exists. Only treat it as an
		// idempotent replay if it is the SAME logical VPS the caller asked for:
		// a different user or a different automation host under the same order
		// item is a genuine conflict, not a retry.
		var existing vpsInstanceRow
		if err := query.Where("order_item_id = ?", *row.OrderItemID).First(&existing).Error; err != nil {
			return r.ensure(err)
		}
		stored := fromVPSInstanceRow(existing)
		if stored.UserID != inst.UserID {
			return domain.ErrConflict
		}
		incomingAuto := strings.TrimSpace(inst.AutomationInstanceID)
		storedAuto := strings.TrimSpace(stored.AutomationInstanceID)
		if incomingAuto != "" && storedAuto != "" && incomingAuto != storedAuto {
			return domain.ErrConflict
		}
		if existing.ID == 0 {
			return domain.ErrInvalidInput
		}
		*inst = stored
		return nil
	}
	if row.ID == 0 {
		return domain.ErrInvalidInput
	}
	*inst = fromVPSInstanceRow(row)
	return nil
}

func (r *GormRepo) GetInstance(ctx context.Context, id int64) (domain.VPSInstance, error) {

	var row vpsInstanceRow
	if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return domain.VPSInstance{}, r.ensure(err)
	}
	return fromVPSInstanceRow(row), nil

}

func (r *GormRepo) GetInstanceByOrderItem(ctx context.Context, orderItemID int64) (domain.VPSInstance, error) {

	var row vpsInstanceRow
	if err := r.gdb.WithContext(ctx).Where("order_item_id = ?", orderItemID).First(&row).Error; err != nil {
		return domain.VPSInstance{}, r.ensure(err)
	}
	return fromVPSInstanceRow(row), nil

}

func (r *GormRepo) ListInstancesByUser(ctx context.Context, userID int64) ([]domain.VPSInstance, error) {

	var rows []vpsInstanceRow
	if err := r.gdb.WithContext(ctx).Where("user_id = ?", userID).Order("id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.VPSInstance, 0, len(rows))
	for _, row := range rows {
		out = append(out, fromVPSInstanceRow(row))
	}
	return out, nil

}

func (r *GormRepo) ListInstances(ctx context.Context, limit, offset int) ([]domain.VPSInstance, int, error) {
	return r.ListInstancesFiltered(ctx, appports.VPSInstanceFilter{}, limit, offset)
}

func (r *GormRepo) ListInstancesFiltered(ctx context.Context, filter appports.VPSInstanceFilter, limit, offset int) ([]domain.VPSInstance, int, error) {
	if limit <= 0 {
		limit = 20
	}
	q := r.gdb.WithContext(ctx).Model(&vpsInstanceRow{})
	if status := strings.TrimSpace(filter.AdminStatus); status != "" {
		q = q.Where("admin_status = ?", status)
	}
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		idCast, userIDCast := "CAST(id AS TEXT)", "CAST(user_id AS TEXT)"
		if r.gdb.Dialector.Name() == "mysql" {
			idCast, userIDCast = "CAST(id AS CHAR)", "CAST(user_id AS CHAR)"
		}
		q = q.Where(
			idCast+" LIKE ? OR "+userIDCast+" LIKE ? OR name LIKE ? OR region LIKE ? OR package_name LIKE ? OR automation_instance_id LIKE ?",
			like, like, like, like, like, like,
		)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []vpsInstanceRow
	if err := q.Order("id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.VPSInstance, 0, len(rows))
	for _, row := range rows {
		out = append(out, fromVPSInstanceRow(row))
	}
	return out, int(total), nil

}

// ListInstancesByUserFiltered pages a single user's VPS instances with a
// server-side status filter (P3-43). The user console must not depend on a
// full-table return contract.
func (r *GormRepo) ListInstancesByUserFiltered(ctx context.Context, userID int64, filter appports.VPSUserInstanceFilter, limit, offset int) ([]domain.VPSInstance, int, error) {
	if limit <= 0 {
		limit = 20
	}
	q := r.gdb.WithContext(ctx).Model(&vpsInstanceRow{}).Where("user_id = ?", userID)
	if s := strings.TrimSpace(filter.Status); s != "" {
		switch s {
		case string(domain.VPSStatusExpiredLocked):
			// The user-facing status is derived from a locked instance that has
			// passed its expiry time. Keep this predicate identical to the
			// frontend's status normalization while filtering before pagination.
			q = q.Where("status = ? OR (status = ? AND expire_at IS NOT NULL AND expire_at <= ?)", s, domain.VPSStatusLocked, time.Now())
		case string(domain.VPSStatusLocked):
			q = q.Where("status = ? AND (expire_at IS NULL OR expire_at > ?)", s, time.Now())
		default:
			q = q.Where("status = ?", s)
		}
	}
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		idCast := "CAST(id AS TEXT)"
		if r.gdb.Dialector.Name() == "mysql" {
			idCast = "CAST(id AS CHAR)"
		}
		q = q.Where(idCast+" LIKE ? OR name LIKE ? OR region LIKE ? OR package_name LIKE ? OR automation_instance_id LIKE ?", like, like, like, like, like)
	}
	if region := strings.TrimSpace(filter.Region); region != "" {
		q = q.Where("region LIKE ?", "%"+region+"%")
	}
	if filter.ExpireBefore != nil {
		q = q.Where("expire_at IS NOT NULL AND expire_at <= ?", *filter.ExpireBefore)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []vpsInstanceRow
	if err := q.Order("id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.VPSInstance, 0, len(rows))
	for _, row := range rows {
		out = append(out, fromVPSInstanceRow(row))
	}
	return out, int(total), nil
}

func (r *GormRepo) ListInstancesExpiring(ctx context.Context, before time.Time) ([]domain.VPSInstance, error) {

	var rows []vpsInstanceRow
	if err := r.gdb.WithContext(ctx).Where("expire_at IS NOT NULL AND expire_at <= ?", before).Order("expire_at ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.VPSInstance, 0, len(rows))
	for _, row := range rows {
		out = append(out, fromVPSInstanceRow(row))
	}
	return out, nil

}

func (r *GormRepo) DeleteInstance(ctx context.Context, id int64) error {

	return r.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var inst vpsInstanceRow
		if err := tx.Where("id = ?", id).First(&inst).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil
			}
			return err
		}
		// Cascade operational state referencing the deleted instance so a stale
		// resize task cannot keep HasPendingResizeTask true or be picked up by
		// the resize worker against a deleted host. Automation logs are kept as
		// audit trail and are age-purged by log retention.
		if err := tx.Where("vps_id = ?", id).Delete(&resizeTaskRow{}).Error; err != nil {
			return err
		}
		if inst.OrderItemID != nil && *inst.OrderItemID > 0 {
			if err := tx.Where("order_item_id = ?", *inst.OrderItemID).Delete(&provisionJobRow{}).Error; err != nil {
				return err
			}
		}
		return tx.Delete(&vpsInstanceRow{}, id).Error
	})

}

func (r *GormRepo) UpdateInstanceStatus(ctx context.Context, id int64, status domain.VPSStatus, automationState int) error {

	return r.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&vpsInstanceRow{}).Where("id = ?", id).Updates(map[string]any{
			"status":           status,
			"automation_state": automationState,
			"updated_at":       time.Now(),
		}).Error; err != nil {
			return err
		}

		var inst vpsInstanceRow
		if err := tx.Where("id = ?", id).First(&inst).Error; err != nil {
			return err
		}
		if inst.OrderItemID != nil && *inst.OrderItemID > 0 {
			orderItemID := *inst.OrderItemID
			switch {
			case isReadyVPSStatus(status):
				if err := tx.Model(&orderItemRow{}).
					Where("id = ? AND action = 'create' AND status IN ?", orderItemID, []string{string(domain.OrderItemStatusApproved), string(domain.OrderItemStatusProvisioning)}).
					Updates(map[string]any{"status": domain.OrderItemStatusActive, "updated_at": time.Now()}).Error; err != nil {
					return err
				}
			case isFailedVPSStatus(status):
				if err := tx.Model(&orderItemRow{}).
					Where("id = ? AND action = 'create' AND status IN ?", orderItemID, []string{string(domain.OrderItemStatusApproved), string(domain.OrderItemStatusProvisioning)}).
					Updates(map[string]any{"status": domain.OrderItemStatusFailed, "updated_at": time.Now()}).Error; err != nil {
					return err
				}
			}

			var item orderItemRow
			if err := tx.Where("id = ?", orderItemID).First(&item).Error; err == nil && item.OrderID > 0 {
				if err := recomputeOrderStatusByItemsGorm(ctx, tx, item.OrderID); err != nil {
					return err
				}
			}
		}
		return nil
	})

}

func (r *GormRepo) UpdateInstanceAdminStatus(ctx context.Context, id int64, status domain.VPSAdminStatus) error {

	return r.gdb.WithContext(ctx).Model(&vpsInstanceRow{}).Where("id = ?", id).Updates(map[string]any{
		"admin_status": status,
		"updated_at":   time.Now(),
	}).Error

}

func (r *GormRepo) UpdateInstanceExpireAt(ctx context.Context, id int64, expireAt time.Time) error {

	return r.gdb.WithContext(ctx).Model(&vpsInstanceRow{}).Where("id = ?", id).Updates(map[string]any{
		"expire_at":  expireAt,
		"updated_at": time.Now(),
	}).Error

}

func (r *GormRepo) UpdateInstancePanelCache(ctx context.Context, id int64, panelURL string) error {

	return r.gdb.WithContext(ctx).Model(&vpsInstanceRow{}).Where("id = ?", id).Updates(map[string]any{
		"panel_url_cache": panelURL,
		"updated_at":      time.Now(),
	}).Error

}

func (r *GormRepo) UpdateInstanceSpec(ctx context.Context, id int64, specJSON string) error {

	return r.gdb.WithContext(ctx).Model(&vpsInstanceRow{}).Where("id = ?", id).Updates(map[string]any{
		"spec_json":  specJSON,
		"updated_at": time.Now(),
	}).Error

}

func (r *GormRepo) UpdateInstanceAccessInfo(ctx context.Context, id int64, accessJSON string) error {

	return r.gdb.WithContext(ctx).Model(&vpsInstanceRow{}).Where("id = ?", id).Updates(map[string]any{
		"access_info_json": accessJSON,
		"updated_at":       time.Now(),
	}).Error

}

func (r *GormRepo) UpdateInstanceEmergencyRenewAt(ctx context.Context, id int64, at time.Time) error {

	return r.gdb.WithContext(ctx).Model(&vpsInstanceRow{}).Where("id = ?", id).Updates(map[string]any{
		"last_emergency_renew_at": at,
		"updated_at":              time.Now(),
	}).Error

}

func (r *GormRepo) UpdateInstanceLocal(ctx context.Context, inst domain.VPSInstance) error {

	return r.gdb.WithContext(ctx).Model(&vpsInstanceRow{}).Where("id = ?", inst.ID).Updates(map[string]any{
		"automation_instance_id": inst.AutomationInstanceID,
		"goods_type_id":          inst.GoodsTypeID,
		"name":                   inst.Name,
		"region":                 inst.Region,
		"region_id":              inst.RegionID,
		"line_id":                inst.LineID,
		"package_id":             inst.PackageID,
		"package_name":           inst.PackageName,
		"cpu":                    inst.CPU,
		"memory_gb":              inst.MemoryGB,
		"disk_gb":                inst.DiskGB,
		"bandwidth_mbps":         inst.BandwidthMB,
		"port_num":               inst.PortNum,
		"monthly_price":          inst.MonthlyPrice,
		"spec_json":              inst.SpecJSON,
		"system_id":              inst.SystemID,
		"status":                 inst.Status,
		"admin_status":           inst.AdminStatus,
		"panel_url_cache":        inst.PanelURLCache,
		"access_info_json":       inst.AccessInfoJSON,
		"updated_at":             time.Now(),
	}).Error

}
