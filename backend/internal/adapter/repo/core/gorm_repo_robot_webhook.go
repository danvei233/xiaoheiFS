package repo

import (
	"context"
	"time"
)

// SaveRobotWebhookNonce durably records a robot-webhook replay key. The unique
// (replay_key) constraint makes concurrent or restarted deliveries of the same
// signed command lose all but one consumer across instances (P1-3 follow-up).
// The boolean reports whether this caller won the insert.
func (r *GormRepo) SaveRobotWebhookNonce(ctx context.Context, replayKey string, expiresAt time.Time) (bool, error) {
	row := robotWebhookNonceRow{ReplayKey: replayKey, ExpiresAt: expiresAt}
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		if r.isDuplicateKeyError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// DeleteExpiredRobotWebhookNonces sweeps expired replay-protection rows so the
// ledger stays bounded.
func (r *GormRepo) DeleteExpiredRobotWebhookNonces(ctx context.Context) error {
	return r.gdb.WithContext(ctx).Where("expires_at < ?", time.Now()).Delete(&robotWebhookNonceRow{}).Error
}
