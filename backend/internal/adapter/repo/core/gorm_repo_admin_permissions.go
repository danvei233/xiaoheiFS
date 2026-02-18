package repo

import (
	"context"
	"errors"
	"gorm.io/gorm/clause"
	"time"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

func (r *GormRepo) AddAuditLog(ctx context.Context, log domain.AdminAuditLog) error {

	return r.gdb.WithContext(ctx).Create(&adminAuditLogRow{
		AdminID:    log.AdminID,
		Action:     log.Action,
		TargetType: log.TargetType,
		TargetID:   log.TargetID,
		DetailJSON: log.DetailJSON,
	}).Error

}

func (r *GormRepo) ListAuditLogs(ctx context.Context, limit, offset int) ([]domain.AdminAuditLog, int, error) {

	var total int64
	if err := r.gdb.WithContext(ctx).Model(&adminAuditLogRow{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []adminAuditLogRow
	if err := r.gdb.WithContext(ctx).Order("id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.AdminAuditLog, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.AdminAuditLog{
			ID:         row.ID,
			AdminID:    row.AdminID,
			Action:     row.Action,
			TargetType: row.TargetType,
			TargetID:   row.TargetID,
			DetailJSON: row.DetailJSON,
			CreatedAt:  row.CreatedAt,
		})
	}
	return out, int(total), nil

}

func (r *GormRepo) ListPermissionGroups(ctx context.Context) ([]domain.PermissionGroup, error) {

	var rows []permissionGroupRow
	if err := r.gdb.WithContext(ctx).Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.PermissionGroup, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.PermissionGroup{
			ID:              row.ID,
			Name:            row.Name,
			Description:     row.Description,
			PermissionsJSON: row.PermissionsJSON,
			CreatedAt:       row.CreatedAt,
			UpdatedAt:       row.UpdatedAt,
		})
	}
	return out, nil

}

func (r *GormRepo) GetPermissionGroup(ctx context.Context, id int64) (domain.PermissionGroup, error) {

	var row permissionGroupRow
	if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return domain.PermissionGroup{}, r.ensure(err)
	}
	return domain.PermissionGroup{
		ID:              row.ID,
		Name:            row.Name,
		Description:     row.Description,
		PermissionsJSON: row.PermissionsJSON,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}, nil

}

func (r *GormRepo) CreatePermissionGroup(ctx context.Context, group *domain.PermissionGroup) error {

	row := permissionGroupRow{
		Name:            group.Name,
		Description:     group.Description,
		PermissionsJSON: group.PermissionsJSON,
	}
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	group.ID = row.ID
	group.CreatedAt = row.CreatedAt
	group.UpdatedAt = row.UpdatedAt
	return nil

}

func (r *GormRepo) UpdatePermissionGroup(ctx context.Context, group domain.PermissionGroup) error {

	return r.gdb.WithContext(ctx).Model(&permissionGroupRow{}).Where("id = ?", group.ID).Updates(map[string]any{
		"name":             group.Name,
		"description":      group.Description,
		"permissions_json": group.PermissionsJSON,
		"updated_at":       time.Now(),
	}).Error

}

func (r *GormRepo) DeletePermissionGroup(ctx context.Context, id int64) error {

	return r.gdb.WithContext(ctx).Delete(&permissionGroupRow{}, id).Error

}

func (r *GormRepo) ListPermissions(ctx context.Context) ([]domain.Permission, error) {

	var rows []permissionModel
	if err := r.gdb.WithContext(ctx).Order("category, sort_order, code").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.Permission, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Permission{
			ID:           row.ID,
			Code:         row.Code,
			Name:         row.Name,
			FriendlyName: row.FriendlyName,
			Category:     row.Category,
			ParentCode:   row.ParentCode,
			SortOrder:    row.SortOrder,
			CreatedAt:    row.CreatedAt,
			UpdatedAt:    row.UpdatedAt,
		})
	}
	return out, nil

}

func (r *GormRepo) GetPermissionByCode(ctx context.Context, code string) (domain.Permission, error) {

	var row permissionModel
	if err := r.gdb.WithContext(ctx).Where("code = ?", code).First(&row).Error; err != nil {
		return domain.Permission{}, r.ensure(err)
	}
	return domain.Permission{
		ID:           row.ID,
		Code:         row.Code,
		Name:         row.Name,
		FriendlyName: row.FriendlyName,
		Category:     row.Category,
		ParentCode:   row.ParentCode,
		SortOrder:    row.SortOrder,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}, nil

}

func (r *GormRepo) UpsertPermission(ctx context.Context, perm *domain.Permission) error {
	m := permissionModel{
		Code:         perm.Code,
		Name:         perm.Name,
		FriendlyName: perm.FriendlyName,
		Category:     perm.Category,
		ParentCode:   perm.ParentCode,
		SortOrder:    perm.SortOrder,
		UpdatedAt:    time.Now(),
	}
	if err := r.gdb.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "code"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"name", "friendly_name", "category", "parent_code", "sort_order", "updated_at",
			}),
		}).
		Create(&m).Error; err != nil {
		return err
	}
	var got permissionModel
	if err := r.gdb.WithContext(ctx).Where("code = ?", perm.Code).First(&got).Error; err == nil {
		perm.ID = got.ID
	}
	return nil
}

func (r *GormRepo) UpdatePermissionName(ctx context.Context, code string, name string) error {

	return r.gdb.WithContext(ctx).Model(&permissionModel{}).Where("code = ?", code).Updates(map[string]any{
		"name":       name,
		"updated_at": time.Now(),
	}).Error

}

func (r *GormRepo) RegisterPermissions(ctx context.Context, perms []domain.PermissionDefinition) error {
	for _, perm := range perms {
		existing, err := r.GetPermissionByCode(ctx, perm.Code)
		if err != nil && !errors.Is(err, appshared.ErrNotFound) {
			return err
		}
		if err == nil {
			if existing.Name != "" {
				perm.Name = existing.Name
			}
			if existing.FriendlyName != "" {
				perm.FriendlyName = existing.FriendlyName
			}
			if existing.Category != "" {
				perm.Category = existing.Category
			}
			if existing.ParentCode != "" {
				perm.ParentCode = existing.ParentCode
			}
			if existing.SortOrder != 0 {
				perm.SortOrder = existing.SortOrder
			}
		}
		upsert := domain.Permission{
			Code:         perm.Code,
			Name:         perm.Name,
			FriendlyName: perm.FriendlyName,
			Category:     perm.Category,
			ParentCode:   perm.ParentCode,
			SortOrder:    perm.SortOrder,
		}
		if err := r.UpsertPermission(ctx, &upsert); err != nil {
			return err
		}
	}
	return nil
}
