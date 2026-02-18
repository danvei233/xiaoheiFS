package repo

import (
	"context"
	"time"
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
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	key.ID = row.ID
	key.CreatedAt = row.CreatedAt
	key.UpdatedAt = row.UpdatedAt
	return nil

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

	return r.gdb.WithContext(ctx).Model(&apiKeyRow{}).Where("id = ?", id).Update("last_used_at", time.Now()).Error

}
