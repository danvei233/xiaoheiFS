package repo

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"xiaoheiplay/internal/domain"
)

// isDuplicateKeyError reports whether err is a unique-constraint violation.
// It first checks GORM's translated error, then falls back to driver message
// patterns because the SQLite dialector surfaces the raw "constraint failed"
// text in some code paths.
func (r *GormRepo) isDuplicateKeyError(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "constraint failed")
}

func (r *GormRepo) CreateUserAPIKey(ctx context.Context, key *domain.UserAPIKey) error {
	row := userAPIKeyRow{
		UserID:     key.UserID,
		Name:       key.Name,
		AKID:       key.AKID,
		KeyHash:    key.KeyHash,
		SecretEnc:  key.SecretEnc,
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
		SecretEnc:  row.SecretEnc,
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
			SecretEnc:  row.SecretEnc,
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

// UpdateUserAPIKeySecrets lazily upgrades a legacy key (whose KeyHash was the
// working secret) to the encrypted-at-rest form. After the upgrade KeyHash is
// a one-way digest of the secret and SecretEnc holds the ciphertext.
func (r *GormRepo) UpdateUserAPIKeySecrets(ctx context.Context, id int64, keyHash, secretEnc string) error {
	return r.gdb.WithContext(ctx).Model(&userAPIKeyRow{}).Where("id = ?", id).Updates(map[string]any{
		"key_hash":   keyHash,
		"secret_enc": secretEnc,
		"updated_at": time.Now(),
	}).Error
}

// TouchUserAPIKey throttles last_used_at writes so the hot request path does
// not contend on a write for every signed call (P2-29). The update is skipped
// when the row was touched within the interval.
func (r *GormRepo) TouchUserAPIKey(ctx context.Context, id int64) error {
	const interval = 5 * time.Minute
	return r.gdb.WithContext(ctx).Model(&userAPIKeyRow{}).
		Where("id = ? AND (last_used_at IS NULL OR last_used_at < ?)", id, time.Now().Add(-interval)).
		Update("last_used_at", time.Now()).Error
}

// SaveUserAPIKeyNonce durably records a signed-request nonce. The unique
// (akid, nonce) constraint makes concurrent double-consumption impossible
// across instances; the boolean reports whether this call won the insert.
func (r *GormRepo) SaveUserAPIKeyNonce(ctx context.Context, akid, nonce string, expiresAt time.Time) (bool, error) {
	row := userAPIKeyNonceRow{AKID: akid, Nonce: nonce, ExpiresAt: expiresAt}
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		if r.isDuplicateKeyError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// CountUserAPIKeyNonces returns how many signed requests this key issued since
// the given time — used as the durable per-key request budget.
func (r *GormRepo) CountUserAPIKeyNonces(ctx context.Context, akid string, since time.Time) (int, error) {
	var count int64
	if err := r.gdb.WithContext(ctx).Model(&userAPIKeyNonceRow{}).
		Where("akid = ? AND created_at > ?", akid, since).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

// DeleteExpiredUserAPIKeyNonces sweeps expired replay-protection rows.
func (r *GormRepo) DeleteExpiredUserAPIKeyNonces(ctx context.Context) error {
	return r.gdb.WithContext(ctx).Where("expires_at < ?", time.Now()).Delete(&userAPIKeyNonceRow{}).Error
}
