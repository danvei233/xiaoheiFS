package repo

import (
	"context"
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"sort"
	"strings"
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

func (r *GormRepo) PurgeAuditLogs(ctx context.Context, before time.Time) error {
	return r.gdb.WithContext(ctx).
		Where("created_at < ?", before).
		Delete(&adminAuditLogRow{}).Error
}

func (r *GormRepo) ListPermissionGroups(ctx context.Context) ([]domain.PermissionGroup, error) {

	var rows []permissionGroupRow
	if err := r.gdb.WithContext(ctx).Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.PermissionGroup, 0, len(rows))
	for _, row := range rows {
		permsJSON, err := r.loadPermissionGroupPermissionsJSON(ctx, row.ID, row.PermissionsJSON)
		if err != nil {
			return nil, err
		}
		out = append(out, domain.PermissionGroup{
			ID:              row.ID,
			Name:            row.Name,
			Description:     row.Description,
			PermissionsJSON: permsJSON,
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
	permsJSON, err := r.loadPermissionGroupPermissionsJSON(ctx, row.ID, row.PermissionsJSON)
	if err != nil {
		return domain.PermissionGroup{}, err
	}
	return domain.PermissionGroup{
		ID:              row.ID,
		Name:            row.Name,
		Description:     row.Description,
		PermissionsJSON: permsJSON,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}, nil

}

func (r *GormRepo) CreatePermissionGroup(ctx context.Context, group *domain.PermissionGroup) error {
	perms, err := normalizePermissionCodes(group.PermissionsJSON)
	if err != nil {
		return err
	}
	permJSON, _ := json.Marshal(perms)
	row := permissionGroupRow{
		Name:            group.Name,
		Description:     group.Description,
		PermissionsJSON: string(permJSON),
	}
	return r.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		if err := replacePermissionGroupPermissionsTx(tx, row.ID, perms); err != nil {
			return err
		}
		group.ID = row.ID
		group.CreatedAt = row.CreatedAt
		group.UpdatedAt = row.UpdatedAt
		group.PermissionsJSON = string(permJSON)
		return nil
	})

}

func (r *GormRepo) UpdatePermissionGroup(ctx context.Context, group domain.PermissionGroup) error {
	perms, err := normalizePermissionCodes(group.PermissionsJSON)
	if err != nil {
		return err
	}
	permJSON, _ := json.Marshal(perms)
	return r.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&permissionGroupRow{}).Where("id = ?", group.ID).Updates(map[string]any{
			"name":             group.Name,
			"description":      group.Description,
			"permissions_json": string(permJSON),
			"updated_at":       time.Now(),
		}).Error; err != nil {
			return err
		}
		return replacePermissionGroupPermissionsTx(tx, group.ID, perms)
	})

}

func (r *GormRepo) DeletePermissionGroup(ctx context.Context, id int64) error {
	return r.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("permission_group_id = ?", id).Delete(&permissionGroupPermissionRow{}).Error; err != nil {
			return err
		}
		return tx.Delete(&permissionGroupRow{}, id).Error
	})

}

func (r *GormRepo) loadPermissionGroupPermissionsJSON(ctx context.Context, groupID int64, fallback string) (string, error) {
	var rows []permissionGroupPermissionRow
	if err := r.gdb.WithContext(ctx).
		Where("permission_group_id = ?", groupID).
		Order("permission_code ASC").
		Find(&rows).Error; err != nil {
		return "", err
	}
	if len(rows) == 0 {
		normalized, err := normalizePermissionCodes(fallback)
		if err != nil {
			return "", err
		}
		raw, _ := json.Marshal(normalized)
		return string(raw), nil
	}
	perms := make([]string, 0, len(rows))
	for _, row := range rows {
		v := strings.TrimSpace(row.PermissionCode)
		if v != "" {
			perms = append(perms, v)
		}
	}
	sort.Strings(perms)
	raw, _ := json.Marshal(perms)
	return string(raw), nil
}

func normalizePermissionCodes(raw string) ([]string, error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return []string{}, nil
	}
	var perms []string
	if err := json.Unmarshal([]byte(text), &perms); err != nil {
		return nil, err
	}
	uniq := make(map[string]struct{}, len(perms))
	out := make([]string, 0, len(perms))
	for _, item := range perms {
		v := strings.TrimSpace(item)
		if v == "" {
			continue
		}
		if _, ok := uniq[v]; ok {
			continue
		}
		uniq[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out, nil
}

func replacePermissionGroupPermissionsTx(tx *gorm.DB, groupID int64, perms []string) error {
	if err := tx.Where("permission_group_id = ?", groupID).Delete(&permissionGroupPermissionRow{}).Error; err != nil {
		return err
	}
	if len(perms) == 0 {
		return nil
	}
	rows := make([]permissionGroupPermissionRow, 0, len(perms))
	for _, perm := range perms {
		rows = append(rows, permissionGroupPermissionRow{
			PermissionGroupID: groupID,
			PermissionCode:    perm,
		})
	}
	return tx.Create(&rows).Error
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
