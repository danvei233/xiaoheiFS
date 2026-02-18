package repo

import (
	"context"
	"gorm.io/gorm"
	"time"
	"xiaoheiplay/internal/domain"
)

func (r *GormRepo) CreateInstance(ctx context.Context, inst *domain.VPSInstance) error {

	row := toVPSInstanceRow(*inst)
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
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

	var total int64
	if err := r.gdb.WithContext(ctx).Model(&vpsInstanceRow{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []vpsInstanceRow
	if err := r.gdb.WithContext(ctx).Order("id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
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

	return r.gdb.WithContext(ctx).Delete(&vpsInstanceRow{}, id).Error

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
		orderItemID := inst.OrderItemID
		if orderItemID > 0 {
			switch {
			case isReadyVPSStatus(status):
				_ = tx.Model(&orderItemRow{}).
					Where("id = ? AND action = 'create' AND status IN ?", orderItemID, []string{string(domain.OrderItemStatusApproved), string(domain.OrderItemStatusProvisioning)}).
					Updates(map[string]any{"status": domain.OrderItemStatusActive, "updated_at": time.Now()}).Error
			case isFailedVPSStatus(status):
				_ = tx.Model(&orderItemRow{}).
					Where("id = ? AND action = 'create' AND status IN ?", orderItemID, []string{string(domain.OrderItemStatusApproved), string(domain.OrderItemStatusProvisioning)}).
					Updates(map[string]any{"status": domain.OrderItemStatusFailed, "updated_at": time.Now()}).Error
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
		"name":                   inst.Name,
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
