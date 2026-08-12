package repo

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"xiaoheiplay/internal/domain"
)

func (r *GormRepo) CreateRefundCleanupJob(ctx context.Context, job *domain.RefundCleanupJob) error {
	row := toRefundCleanupJobRow(*job)
	// One cleanup outbox entry per wallet order; a re-approve/retry must not
	// stack duplicate jobs.
	if err := r.gdb.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "wallet_order_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"vps_id", "host_id", "goods_type_id", "user_id", "status", "next_run_at", "updated_at",
			}),
		}).
		Create(&row).Error; err != nil {
		return err
	}
	*job = fromRefundCleanupJobRow(row)
	return nil
}

func (r *GormRepo) GetRefundCleanupJobByWalletOrder(ctx context.Context, walletOrderID int64) (domain.RefundCleanupJob, error) {
	var row refundCleanupJobRow
	if err := r.gdb.WithContext(ctx).Where("wallet_order_id = ?", walletOrderID).First(&row).Error; err != nil {
		return domain.RefundCleanupJob{}, r.ensure(err)
	}
	return fromRefundCleanupJobRow(row), nil
}

func (r *GormRepo) ListDueRefundCleanupJobs(ctx context.Context, limit int) ([]domain.RefundCleanupJob, error) {
	if limit <= 0 {
		limit = 20
	}
	now := time.Now()
	var rows []refundCleanupJobRow
	if err := r.gdb.WithContext(ctx).
		Where("(status IN ? AND next_run_at <= ?) OR (status = ? AND (lease_until IS NOT NULL AND lease_until <= ?))",
			[]string{domain.RefundCleanupPending, domain.RefundCleanupRetry}, now,
			domain.RefundCleanupRunning, now).
		Order("id ASC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.RefundCleanupJob, 0, len(rows))
	for _, row := range rows {
		out = append(out, fromRefundCleanupJobRow(row))
	}
	return out, nil
}

func (r *GormRepo) ListRefundCleanupJobs(ctx context.Context, limit, offset int) ([]domain.RefundCleanupJob, int, error) {
	if limit <= 0 {
		limit = 20
	}
	var total int64
	if err := r.gdb.WithContext(ctx).Model(&refundCleanupJobRow{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []refundCleanupJobRow
	if err := r.gdb.WithContext(ctx).Order("id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.RefundCleanupJob, 0, len(rows))
	for _, row := range rows {
		out = append(out, fromRefundCleanupJobRow(row))
	}
	return out, int(total), nil
}

func (r *GormRepo) ClaimRefundCleanupJob(ctx context.Context, jobID int64, owner string, leaseUntil time.Time) (domain.RefundCleanupJob, bool, error) {
	now := time.Now()
	updates := map[string]any{
		"status":      domain.RefundCleanupRunning,
		"attempts":    gorm.Expr("attempts + 1"),
		"lease_owner": owner,
		"lease_until": leaseUntil,
		"fence":       gorm.Expr("fence + 1"),
		"updated_at":  now,
	}
	res := r.gdb.WithContext(ctx).Model(&refundCleanupJobRow{}).
		Where("id = ? AND status IN ?", jobID, []string{domain.RefundCleanupPending, domain.RefundCleanupRetry}).
		Updates(updates)
	if res.Error != nil {
		return domain.RefundCleanupJob{}, false, res.Error
	}
	if res.RowsAffected == 1 {
		return r.claimedRefundCleanupJob(ctx, jobID)
	}
	res = r.gdb.WithContext(ctx).Model(&refundCleanupJobRow{}).
		Where("id = ? AND status = ? AND (lease_until IS NULL OR lease_until <= ?)", jobID, domain.RefundCleanupRunning, now).
		Updates(updates)
	if res.Error != nil {
		return domain.RefundCleanupJob{}, false, res.Error
	}
	if res.RowsAffected == 1 {
		return r.claimedRefundCleanupJob(ctx, jobID)
	}
	return domain.RefundCleanupJob{}, false, nil
}

func (r *GormRepo) claimedRefundCleanupJob(ctx context.Context, jobID int64) (domain.RefundCleanupJob, bool, error) {
	var row refundCleanupJobRow
	if err := r.gdb.WithContext(ctx).Where("id = ?", jobID).First(&row).Error; err != nil {
		return domain.RefundCleanupJob{}, false, r.ensure(err)
	}
	return fromRefundCleanupJobRow(row), true, nil
}

// UpdateRefundCleanupJob persists a job write CAS'd on the lease owner and
// fence. A lost lease abandons the write.
func (r *GormRepo) UpdateRefundCleanupJob(ctx context.Context, job domain.RefundCleanupJob) (bool, error) {
	res := r.gdb.WithContext(ctx).Model(&refundCleanupJobRow{}).
		Where("id = ? AND lease_owner = ? AND fence = ?", job.ID, job.LeaseOwner, job.Fence).
		Updates(map[string]any{
			"status":          job.Status,
			"attempts":        job.Attempts,
			"next_run_at":     job.NextRunAt,
			"last_error_code": job.LastErrorCode,
			"last_error":      job.LastError,
			"lease_owner":     job.LeaseOwner,
			"lease_until":     job.LeaseUntil,
			"updated_at":      time.Now(),
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

func (r *GormRepo) MarkRefundCleanupDone(ctx context.Context, jobID int64) error {
	return r.gdb.WithContext(ctx).Model(&refundCleanupJobRow{}).
		Where("id = ?", jobID).
		Updates(map[string]any{
			"status":      domain.RefundCleanupDone,
			"lease_owner": "",
			"lease_until": nil,
			"updated_at":  time.Now(),
		}).Error
}

func toRefundCleanupJobRow(job domain.RefundCleanupJob) refundCleanupJobRow {
	return refundCleanupJobRow{
		ID:            job.ID,
		WalletOrderID: job.WalletOrderID,
		UserID:        job.UserID,
		VPSID:         job.VPSID,
		HostID:        job.HostID,
		GoodsTypeID:   job.GoodsTypeID,
		Status:        job.Status,
		Attempts:      job.Attempts,
		NextRunAt:     job.NextRunAt,
		LastErrorCode: job.LastErrorCode,
		LastError:     job.LastError,
		LeaseOwner:    job.LeaseOwner,
		LeaseUntil:    job.LeaseUntil,
		Fence:         job.Fence,
		CreatedAt:     job.CreatedAt,
		UpdatedAt:     job.UpdatedAt,
	}
}

func fromRefundCleanupJobRow(row refundCleanupJobRow) domain.RefundCleanupJob {
	return domain.RefundCleanupJob{
		ID:            row.ID,
		WalletOrderID: row.WalletOrderID,
		UserID:        row.UserID,
		VPSID:         row.VPSID,
		HostID:        row.HostID,
		GoodsTypeID:   row.GoodsTypeID,
		Status:        row.Status,
		Attempts:      row.Attempts,
		NextRunAt:     row.NextRunAt,
		LastErrorCode: row.LastErrorCode,
		LastError:     row.LastError,
		LeaseOwner:    row.LeaseOwner,
		LeaseUntil:    row.LeaseUntil,
		Fence:         row.Fence,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
}
