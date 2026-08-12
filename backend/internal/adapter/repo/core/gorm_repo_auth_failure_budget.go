package repo

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	appshared "xiaoheiplay/internal/app/shared"
)

// PeekAuthFailure returns the current failure count and whether the key is
// locked (count >= limit within the current window) without incrementing.
func (r *GormRepo) PeekAuthFailure(ctx context.Context, key string, limit int, window time.Duration, now time.Time) (int, bool, error) {
	if key == "" || limit <= 0 || window <= 0 {
		return 0, false, appshared.ErrInvalidInput
	}
	var row authFailureBudgetRow
	err := r.gdb.WithContext(ctx).Where("key = ?", key).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	if !row.WindowEnd.After(now) {
		// Window expired; treat as reset.
		return 0, false, nil
	}
	return row.Count, row.Count >= limit, nil
}

// RecordAuthFailure atomically increments the failure counter for a key within
// a fixed window and returns the new count plus the locked-until time when the
// limit is reached. The increment is a single INSERT ... ON CONFLICT upsert so
// concurrent callers across instances serialize without a read-then-write lock
// escalation (which SQLite cannot resolve). The window resets when it expires.
func (r *GormRepo) RecordAuthFailure(ctx context.Context, key string, limit int, window time.Duration, now time.Time) (int, time.Time, error) {
	if key == "" || limit <= 0 || window <= 0 {
		return 0, time.Time{}, appshared.ErrInvalidInput
	}
	windowEnd := now.Add(window)
	row := authFailureBudgetRow{Key: key, Count: 1, WindowEnd: windowEnd, UpdatedAt: now}
	err := r.gdb.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "key"}},
		DoUpdates: clause.Assignments(map[string]any{
			"count":      gorm.Expr("CASE WHEN window_end <= ? THEN 1 ELSE count + 1 END", now),
			"window_end": gorm.Expr("CASE WHEN window_end <= ? THEN ? ELSE window_end END", now, windowEnd),
			"updated_at": now,
		}),
	}).Create(&row).Error
	if err != nil {
		return 0, time.Time{}, err
	}
	var stored authFailureBudgetRow
	if err := r.gdb.WithContext(ctx).Where("key = ?", key).First(&stored).Error; err != nil {
		return 0, time.Time{}, err
	}
	count := stored.Count
	lockedUntil := time.Time{}
	if count >= limit {
		lockedUntil = stored.WindowEnd
	}
	return count, lockedUntil, nil
}

// ResetAuthFailure clears the failure budget for a key (used on success).
func (r *GormRepo) ResetAuthFailure(ctx context.Context, key string) error {
	if key == "" {
		return appshared.ErrInvalidInput
	}
	return r.gdb.WithContext(ctx).Where("key = ?", key).Delete(&authFailureBudgetRow{}).Error
}

// DeleteExpiredAuthFailureBudgets sweeps expired budget rows so the table stays
// bounded.
func (r *GormRepo) DeleteExpiredAuthFailureBudgets(ctx context.Context) error {
	return r.gdb.WithContext(ctx).Where("window_end < ?", time.Now()).Delete(&authFailureBudgetRow{}).Error
}
