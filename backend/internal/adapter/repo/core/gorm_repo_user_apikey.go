package repo

import (
	"context"
	"time"

	"xiaoheiplay/internal/domain"
)

func (r *GormRepo) CreateUserAPIKey(ctx context.Context, key *domain.UserAPIKey) error {
	row := userAPIKeyRow{
		UserID:     key.UserID,
		Name:       key.Name,
		AKID:       key.AKID,
		KeyHash:    key.KeyHash,
		Status:     string(key.Status),
		ScopesJSON: key.ScopesJSON,
	}
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	key.ID = row.ID
	key.CreatedAt = row.CreatedAt
	key.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *GormRepo) GetUserAPIKeyByAKID(ctx context.Context, akid string) (domain.UserAPIKey, error) {
	var row userAPIKeyRow
	if err := r.gdb.WithContext(ctx).Where("akid = ?", akid).First(&row).Error; err != nil {
		return domain.UserAPIKey{}, r.ensure(err)
	}
	return domain.UserAPIKey{
		ID:         row.ID,
		UserID:     row.UserID,
		Name:       row.Name,
		AKID:       row.AKID,
		KeyHash:    row.KeyHash,
		Status:     domain.APIKeyStatus(row.Status),
		ScopesJSON: row.ScopesJSON,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
		LastUsedAt: row.LastUsedAt,
	}, nil
}

func (r *GormRepo) ListUserAPIKeys(ctx context.Context, userID int64, limit, offset int) ([]domain.UserAPIKey, int, error) {
	var total int64
	if err := r.gdb.WithContext(ctx).Model(&userAPIKeyRow{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []userAPIKeyRow
	if err := r.gdb.WithContext(ctx).Where("user_id = ?", userID).Order("id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.UserAPIKey, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.UserAPIKey{
			ID:         row.ID,
			UserID:     row.UserID,
			Name:       row.Name,
			AKID:       row.AKID,
			KeyHash:    row.KeyHash,
			Status:     domain.APIKeyStatus(row.Status),
			ScopesJSON: row.ScopesJSON,
			CreatedAt:  row.CreatedAt,
			UpdatedAt:  row.UpdatedAt,
			LastUsedAt: row.LastUsedAt,
		})
	}
	return out, int(total), nil
}

func (r *GormRepo) UpdateUserAPIKeyStatus(ctx context.Context, userID, id int64, status domain.APIKeyStatus) error {
	return r.gdb.WithContext(ctx).Model(&userAPIKeyRow{}).Where("id = ? AND user_id = ?", id, userID).Updates(map[string]any{
		"status":     string(status),
		"updated_at": time.Now(),
	}).Error
}

func (r *GormRepo) DeleteUserAPIKey(ctx context.Context, userID, id int64) error {
	return r.gdb.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&userAPIKeyRow{}).Error
}

func (r *GormRepo) TouchUserAPIKey(ctx context.Context, id int64) error {
	return r.gdb.WithContext(ctx).Model(&userAPIKeyRow{}).Where("id = ?", id).Update("last_used_at", time.Now()).Error
}
