package repo

import (
	"context"
	"time"

	"gorm.io/gorm"

	"xiaoheiplay/internal/domain"
)

func (r *GormRepo) CreateAPIKey(ctx context.Context, key *domain.APIKey) error {

	row := apiKeyRow{
		Name:              key.Name,
		KeyHash:           key.KeyHash,
		Status:            string(key.Status),
		ScopesJSON:        key.ScopesJSON,
		PermissionGroupID: key.PermissionGroupID,
	}
	create := func(db *gorm.DB) error {
		if err := db.Create(&row).Error; err != nil {
			return err
		}
		key.ID = row.ID
		key.CreatedAt = row.CreatedAt
		key.UpdatedAt = row.UpdatedAt
		return nil
	}
	if key.PermissionGroupID != nil {
		// Serialize with group deletion so a concurrent delete cannot strand a
		// new API key with a dangling permission-group reference.
		return r.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := lockPermissionGroupForWrite(tx, key.PermissionGroupID); err != nil {
				return err
			}
			return create(tx)
		})
	}
	return create(r.gdb.WithContext(ctx))

}

func (r *GormRepo) GetAPIKeyByHash(ctx context.Context, keyHash string) (domain.APIKey, error) {

	var row apiKeyRow
	if err := r.gdb.WithContext(ctx).Where("key_hash = ?", keyHash).First(&row).Error; err != nil {
		return domain.APIKey{}, r.ensure(err)
	}
	var out domain.APIKey
	out.ID = row.ID
	out.Name = row.Name
	out.KeyHash = row.KeyHash
	out.Status = domain.APIKeyStatus(row.Status)
	out.ScopesJSON = row.ScopesJSON
	out.PermissionGroupID = row.PermissionGroupID
	out.CreatedAt = row.CreatedAt
	out.UpdatedAt = row.UpdatedAt
	return out, nil

}

func (r *GormRepo) ListAPIKeys(ctx context.Context, limit, offset int) ([]domain.APIKey, int, error) {

	var total int64
	if err := r.gdb.WithContext(ctx).Model(&apiKeyRow{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []apiKeyRow
	if err := r.gdb.WithContext(ctx).Order("id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.APIKey, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.APIKey{
			ID:                row.ID,
			Name:              row.Name,
			KeyHash:           row.KeyHash,
			Status:            domain.APIKeyStatus(row.Status),
			ScopesJSON:        row.ScopesJSON,
			PermissionGroupID: row.PermissionGroupID,
			CreatedAt:         row.CreatedAt,
			UpdatedAt:         row.UpdatedAt,
		})
	}
	return out, int(total), nil

}

func (r *GormRepo) UpdateAPIKeyStatus(ctx context.Context, id int64, status domain.APIKeyStatus) error {

	return r.gdb.WithContext(ctx).Model(&apiKeyRow{}).Where("id = ?", id).Updates(map[string]any{
		"status":     status,
		"updated_at": time.Now(),
	}).Error

}

func (r *GormRepo) TouchAPIKey(ctx context.Context, id int64) error {
	// Throttle last_used_at writes so the hot request path does not contend on
	// a write for every call (P2-29).
	const interval = 5 * time.Minute
	return r.gdb.WithContext(ctx).Model(&apiKeyRow{}).
		Where("id = ? AND (last_used_at IS NULL OR last_used_at < ?)", id, time.Now().Add(-interval)).
		Update("last_used_at", time.Now()).Error

}

func (r *GormRepo) CountAPIKeysByPermissionGroup(ctx context.Context, groupID int64) (int64, error) {
	var count int64
	if err := r.gdb.WithContext(ctx).Model(&apiKeyRow{}).Where("permission_group_id = ?", groupID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
