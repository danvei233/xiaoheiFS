package repo

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"xiaoheiplay/internal/domain"
)

// AllowRateLimit atomically records one call for the bucket in the current
// window and reports whether the count is still within max. The upsert-increment
// is a single atomic statement, so two instances (or a restart) cannot bypass
// the budget. The counter may grow past max; the row is bounded by
// CleanupRateLimits.
func (r *GormRepo) AllowRateLimit(ctx context.Context, bucket string, max int, window time.Duration) (bool, error) {
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return false, domain.ErrInvalidInput
	}
	if max <= 0 {
		max = 1
	}
	if window <= 0 {
		window = time.Minute
	}
	now := time.Now()
	windowStart := now.Truncate(window)
	row := rateLimitBucketRow{
		BucketKey:   bucket,
		WindowStart: windowStart,
		Count:       1,
		UpdatedAt:   now,
	}
	if err := r.gdb.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "bucket_key"},
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
	if err := r.gdb.WithContext(ctx).Model(&rateLimitBucketRow{}).
		Where("bucket_key = ? AND window_start = ?", bucket, windowStart).
		Pluck("count", &count).Error; err != nil {
		return false, err
	}
	return count <= int64(max), nil
}

// CleanupRateLimits deletes rate-limit windows older than before, bounding the
// table size. Returns the number of rows removed.
func (r *GormRepo) CleanupRateLimits(ctx context.Context, before time.Time) (int64, error) {
	res := r.gdb.WithContext(ctx).Where("window_start < ?", before).Delete(&rateLimitBucketRow{})
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}
