package repo

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"xiaoheiplay/internal/domain"
)

func (r *GormRepo) CreateRealNameVerification(ctx context.Context, record *domain.RealNameVerification) error {
	row := realnameVerificationRow{
		UserID:        record.UserID,
		RealName:      record.RealName,
		IDNumber:      record.IDNumber,
		Status:        record.Status,
		Provider:      record.Provider,
		Reason:        record.Reason,
		ProviderState: record.ProviderState,
		VerifiedAt:    record.VerifiedAt,
	}
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	record.ID = row.ID
	record.CreatedAt = row.CreatedAt
	return nil
}

// ListPendingRealNameVerifications pages status='pending' records with an id
// cursor (id > afterID, ASC). Filtering at the SQL layer keeps realname polling
// from scanning the whole verification table every scheduler tick, and the
// stable cursor avoids offset-pagination duplicates while records transition
// out of pending during a poll.
func (r *GormRepo) ListPendingRealNameVerifications(ctx context.Context, afterID int64, limit int) ([]domain.RealNameVerification, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows []realnameVerificationRow
	if err := r.gdb.WithContext(ctx).
		Where("status = ? AND id > ?", "pending", afterID).
		Order("id ASC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.RealNameVerification, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.RealNameVerification{
			ID:            row.ID,
			UserID:        row.UserID,
			RealName:      row.RealName,
			IDNumber:      row.IDNumber,
			Status:        row.Status,
			Provider:      row.Provider,
			Reason:        row.Reason,
			ProviderState: row.ProviderState,
			LastPolledAt:  row.LastPolledAt,
			PollCount:     row.PollCount,
			CreatedAt:     row.CreatedAt,
			VerifiedAt:    row.VerifiedAt,
		})
	}
	return out, nil
}

func (r *GormRepo) GetLatestRealNameVerification(ctx context.Context, userID int64) (domain.RealNameVerification, error) {
	var row realnameVerificationRow
	if err := r.gdb.WithContext(ctx).Where("user_id = ?", userID).Order("id DESC").Limit(1).First(&row).Error; err != nil {
		return domain.RealNameVerification{}, r.ensure(err)
	}
	return domain.RealNameVerification{
		ID:            row.ID,
		UserID:        row.UserID,
		RealName:      row.RealName,
		IDNumber:      row.IDNumber,
		Status:        row.Status,
		Provider:      row.Provider,
		Reason:        row.Reason,
		ProviderState: row.ProviderState,
		LastPolledAt:  row.LastPolledAt,
		PollCount:     row.PollCount,
		CreatedAt:     row.CreatedAt,
		VerifiedAt:    row.VerifiedAt,
	}, nil
}

func (r *GormRepo) ListRealNameVerifications(ctx context.Context, userID *int64, status string, limit, offset int) ([]domain.RealNameVerification, int, error) {
	q := r.gdb.WithContext(ctx).Model(&realnameVerificationRow{})
	if userID != nil {
		q = q.Where("user_id = ?", *userID)
	}
	if status = strings.ToLower(strings.TrimSpace(status)); status != "" {
		// 实名记录页按状态筛选（m059 F6）。
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 20
	}
	var rows []realnameVerificationRow
	if err := q.Order("id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.RealNameVerification, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.RealNameVerification{
			ID:            row.ID,
			UserID:        row.UserID,
			RealName:      row.RealName,
			IDNumber:      row.IDNumber,
			Status:        row.Status,
			Provider:      row.Provider,
			Reason:        row.Reason,
			ProviderState: row.ProviderState,
			LastPolledAt:  row.LastPolledAt,
			PollCount:     row.PollCount,
			CreatedAt:     row.CreatedAt,
			VerifiedAt:    row.VerifiedAt,
		})
	}
	return out, int(total), nil
}

// MarkRealNamePolled records that a pending verification was polled against the
// provider at the given time and bumps the poll counter. The persisted timestamp
// keeps the poll cooldown consistent across instances and restarts.
func (r *GormRepo) MarkRealNamePolled(ctx context.Context, id int64, at time.Time) error {
	return r.gdb.WithContext(ctx).Model(&realnameVerificationRow{}).Where("id = ?", id).Updates(map[string]any{
		"last_polled_at": at,
		"poll_count":     gorm.Expr("poll_count + 1"),
	}).Error
}

func (r *GormRepo) UpdateRealNameStatus(ctx context.Context, id int64, status string, reason string, verifiedAt *time.Time) error {
	return r.gdb.WithContext(ctx).Model(&realnameVerificationRow{}).Where("id = ?", id).Updates(map[string]any{
		"status":      status,
		"reason":      reason,
		"verified_at": verifiedAt,
	}).Error
}

func (r *GormRepo) UpdateRealNameProviderState(ctx context.Context, id int64, providerState string, stableReason string) error {
	return r.gdb.WithContext(ctx).Model(&realnameVerificationRow{}).Where("id = ?", id).Updates(map[string]any{
		"provider_state": providerState,
		"reason":         stableReason,
	}).Error
}

// AllowRealNameAttempt atomically records one verification attempt for the
// user+dimension in the current window and reports whether the budget is still
// within max. The upsert-increment is a single atomic statement, so two service
// instances (or a restart) cannot bypass the budget. The counter may grow past
// max; the window row is bounded by CleanupRealNameAttempts.
func (r *GormRepo) AllowRealNameAttempt(ctx context.Context, userID int64, dimension string, max int, window time.Duration) (bool, error) {
	if userID <= 0 || strings.TrimSpace(dimension) == "" {
		return false, domain.ErrInvalidInput
	}
	if max <= 0 {
		max = 1
	}
	if window <= 0 {
		window = 10 * time.Minute
	}
	now := time.Now()
	windowStart := now.Truncate(window)
	row := realnameAttemptRow{
		UserID:      userID,
		Dimension:   strings.TrimSpace(dimension),
		WindowStart: windowStart,
		Count:       1,
		UpdatedAt:   now,
	}
	// Atomically increment or seed the counter for this window.
	if err := r.gdb.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "user_id"},
				{Name: "dimension"},
				{Name: "window_start"},
			},
			DoUpdates: clause.Assignments(map[string]any{
				"count":      gorm.Expr("count + 1"),
				"updated_at": now,
			}),
		}).
		Create(&row).Error; err != nil {
		return false, err
	}
	var count int64
	if err := r.gdb.WithContext(ctx).Model(&realnameAttemptRow{}).
		Where("user_id = ? AND dimension = ? AND window_start = ?", userID, dimension, windowStart).
		Pluck("count", &count).Error; err != nil {
		return false, err
	}
	return count <= int64(max), nil
}

// CleanupRealNameAttempts deletes attempt windows older than before, bounding
// the table size. Returns the number of rows removed.
func (r *GormRepo) CleanupRealNameAttempts(ctx context.Context, before time.Time) (int64, error) {
	res := r.gdb.WithContext(ctx).Where("window_start < ?", before).Delete(&realnameAttemptRow{})
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}
