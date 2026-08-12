package repo

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"xiaoheiplay/internal/domain"
)

func (r *GormRepo) CreateAutomationLog(ctx context.Context, log *domain.AutomationLog) error {
	row := automationLogRow{
		OrderID:      log.OrderID,
		OrderItemID:  log.OrderItemID,
		Action:       log.Action,
		RequestJSON:  log.RequestJSON,
		ResponseJSON: log.ResponseJSON,
		Success:      boolToInt(log.Success),
		Message:      log.Message,
	}
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	log.ID = row.ID
	log.CreatedAt = row.CreatedAt
	return nil
}

func (r *GormRepo) ListAutomationLogs(ctx context.Context, orderID int64, limit, offset int) ([]domain.AutomationLog, int, error) {
	q := r.gdb.WithContext(ctx).Model(&automationLogRow{})
	if orderID > 0 {
		q = q.Where("order_id = ?", orderID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 20
	}
	var rows []automationLogRow
	if err := q.Order("id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.AutomationLog, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.AutomationLog{
			ID:           row.ID,
			OrderID:      row.OrderID,
			OrderItemID:  row.OrderItemID,
			Action:       row.Action,
			RequestJSON:  row.RequestJSON,
			ResponseJSON: row.ResponseJSON,
			Success:      row.Success == 1,
			Message:      row.Message,
			CreatedAt:    row.CreatedAt,
		})
	}
	return out, int(total), nil
}

func (r *GormRepo) PurgeAutomationLogs(ctx context.Context, before time.Time) error {
	return r.gdb.WithContext(ctx).
		Where("created_at < ?", before).
		Delete(&automationLogRow{}).Error
}

func (r *GormRepo) CreateOrUpdateProvisionJob(ctx context.Context, job *domain.ProvisionJob) error {
	now := time.Now()
	m := provisionJobRow{
		ID:             job.ID,
		OrderID:        job.OrderID,
		OrderItemID:    job.OrderItemID,
		HostID:         job.HostID,
		HostName:       job.HostName,
		Status:         job.Status,
		Attempts:       job.Attempts,
		NextRunAt:      job.NextRunAt,
		LastError:      job.LastError,
		LeaseOwner:     job.LeaseOwner,
		LeaseUntil:     job.LeaseUntil,
		Fence:          job.Fence,
		Stage:          job.Stage,
		AttemptID:      job.AttemptID,
		CredsEncrypted: job.CredsEncrypted,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := r.gdb.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "order_item_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"host_id", "host_name", "status", "attempts", "next_run_at", "last_error",
				"lease_owner", "lease_until", "fence", "stage", "attempt_id", "creds_encrypted", "updated_at",
			}),
		}).
		Create(&m).Error; err != nil {
		return err
	}
	var got provisionJobRow
	if err := r.gdb.WithContext(ctx).Select("id").Where("order_item_id = ?", job.OrderItemID).First(&got).Error; err == nil {
		job.ID = got.ID
	}
	return nil
}

func (r *GormRepo) GetProvisionJob(ctx context.Context, id int64) (domain.ProvisionJob, error) {
	var row provisionJobRow
	if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return domain.ProvisionJob{}, r.ensure(err)
	}
	return fromProvisionJobRow(row), nil
}

func (r *GormRepo) GetProvisionJobByOrderItem(ctx context.Context, orderItemID int64) (domain.ProvisionJob, error) {
	var row provisionJobRow
	if err := r.gdb.WithContext(ctx).Where("order_item_id = ?", orderItemID).First(&row).Error; err != nil {
		return domain.ProvisionJob{}, r.ensure(err)
	}
	return fromProvisionJobRow(row), nil
}

func (r *GormRepo) RequeueProvisionJob(ctx context.Context, jobID int64) (domain.ProvisionJob, error) {
	now := time.Now()
	var current provisionJobRow
	if err := r.gdb.WithContext(ctx).Where("id = ?", jobID).First(&current).Error; err != nil {
		return domain.ProvisionJob{}, r.ensure(err)
	}
	// A create_host_sent stage means an unknown CreateHost result: the stage is
	// preserved so the retry reconciles by hostname and never issues a second
	// CreateHost. Any other terminal stage is cleared so the next run resolves
	// the stage from the persisted host id (reuse the created host) or re-creates
	// when the host never existed.
	nextStage := current.Stage
	if nextStage != domain.ProvisionStageCreateHostSent {
		nextStage = ""
	}
	if err := r.gdb.WithContext(ctx).Model(&provisionJobRow{}).
		Where("id = ?", jobID).
		Updates(map[string]any{
			"status":      provisionJobRetry,
			"next_run_at": now,
			"lease_owner": "",
			"lease_until": nil,
			"stage":       nextStage,
			"updated_at":  now,
		}).Error; err != nil {
		return domain.ProvisionJob{}, err
	}
	return r.GetProvisionJob(ctx, jobID)
}

const provisionJobRetry = "retry"

func (r *GormRepo) ListDueProvisionJobs(ctx context.Context, limit int) ([]domain.ProvisionJob, error) {
	if limit <= 0 {
		limit = 20
	}
	now := time.Now()
	fetchLimit := limit * 5
	if fetchLimit < 50 {
		fetchLimit = 50
	}

	var rows []provisionJobRow
	if err := r.gdb.WithContext(ctx).
		Where("(status IN ? AND next_run_at <= ?) OR (status = ? AND (lease_until IS NOT NULL AND lease_until <= ?))",
			[]string{"pending", "retry"}, now,
			"running", now).
		Order("id ASC").
		Limit(fetchLimit).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.ProvisionJob, 0, limit)
	for _, row := range rows {
		out = append(out, fromProvisionJobRow(row))
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

// UpdateProvisionJob persists a job write CAS'd on the lease owner and fence.
// It returns false when the writer no longer holds the lease (a second worker
// took over after the lease elapsed), in which case the caller must abandon all
// further side effects.
func (r *GormRepo) UpdateProvisionJob(ctx context.Context, job domain.ProvisionJob) (bool, error) {
	res := r.gdb.WithContext(ctx).Model(&provisionJobRow{}).
		Where("id = ? AND lease_owner = ? AND fence = ?", job.ID, job.LeaseOwner, job.Fence).
		Updates(map[string]any{
			"status":      job.Status,
			"attempts":    job.Attempts,
			"next_run_at": job.NextRunAt,
			"last_error":  job.LastError,
			"host_id":     job.HostID,
			"host_name":   job.HostName,
			"stage":       job.Stage,
			"attempt_id":  job.AttemptID,
			"creds_encrypted": job.CredsEncrypted,
			"lease_owner": job.LeaseOwner,
			"lease_until": job.LeaseUntil,
			"updated_at":  time.Now(),
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// RenewProvisionJobLease extends a held lease. It is CAS'd on owner+fence so a
// worker that lost the lease cannot renew it.
func (r *GormRepo) RenewProvisionJobLease(ctx context.Context, jobID int64, owner string, fence int, leaseUntil time.Time) (bool, error) {
	res := r.gdb.WithContext(ctx).Model(&provisionJobRow{}).
		Where("id = ? AND lease_owner = ? AND fence = ?", jobID, owner, fence).
		Updates(map[string]any{
			"lease_until": leaseUntil,
			"updated_at":  time.Now(),
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// ClaimProvisionJob atomically transitions a due provision job to running for
// this worker and returns the fresh claimed state. It claims jobs in
// pending/retry, or a running job whose lease has elapsed (a crashed worker).
// An actively-processing running job keeps lease_until in the future and is not
// reclaimed, so a fresh lease is never stolen and only an expired one — via the
// CAS on the row state — can be taken over. The fence is bumped on every claim
// so the previous worker's final writes cannot overwrite the new owner.
func (r *GormRepo) ClaimProvisionJob(ctx context.Context, jobID int64, owner string, leaseUntil time.Time) (domain.ProvisionJob, bool, error) {
	now := time.Now()
	updates := map[string]any{
		"status":      "running",
		"attempts":    gorm.Expr("attempts + 1"),
		"next_run_at": now.Add(domain.ProvisionLeaseDuration),
		"lease_owner": owner,
		"lease_until": leaseUntil,
		"fence":       gorm.Expr("fence + 1"),
		"updated_at":  now,
	}
	res := r.gdb.WithContext(ctx).Model(&provisionJobRow{}).
		Where("id = ? AND status IN ?", jobID, []string{"pending", "retry"}).
		Updates(updates)
	if res.Error != nil {
		return domain.ProvisionJob{}, false, res.Error
	}
	if res.RowsAffected == 1 {
		return r.claimedProvisionJob(ctx, jobID)
	}
	// Reclaim a stale running job whose lease has elapsed.
	res = r.gdb.WithContext(ctx).Model(&provisionJobRow{}).
		Where("id = ? AND status = ? AND (lease_until IS NULL OR lease_until <= ?)", jobID, "running", now).
		Updates(updates)
	if res.Error != nil {
		return domain.ProvisionJob{}, false, res.Error
	}
	if res.RowsAffected == 1 {
		return r.claimedProvisionJob(ctx, jobID)
	}
	return domain.ProvisionJob{}, false, nil
}

func (r *GormRepo) claimedProvisionJob(ctx context.Context, jobID int64) (domain.ProvisionJob, bool, error) {
	job, err := r.GetProvisionJob(ctx, jobID)
	if err != nil {
		return domain.ProvisionJob{}, false, err
	}
	return job, true, nil
}

func fromProvisionJobRow(row provisionJobRow) domain.ProvisionJob {
	return domain.ProvisionJob{
		ID:             row.ID,
		OrderID:        row.OrderID,
		OrderItemID:    row.OrderItemID,
		HostID:         row.HostID,
		HostName:       row.HostName,
		Status:         row.Status,
		Attempts:       row.Attempts,
		NextRunAt:      row.NextRunAt,
		LastError:      row.LastError,
		LeaseOwner:     row.LeaseOwner,
		LeaseUntil:     row.LeaseUntil,
		Fence:          row.Fence,
		Stage:          row.Stage,
		AttemptID:      row.AttemptID,
		CredsEncrypted: row.CredsEncrypted,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

func (r *GormRepo) CreateTaskRun(ctx context.Context, run *domain.ScheduledTaskRun) error {

	row := scheduledTaskRunRow{
		TaskKey:     run.TaskKey,
		Status:      run.Status,
		StartedAt:   run.StartedAt,
		FinishedAt:  run.FinishedAt,
		DurationSec: run.DurationSec,
		Message:     run.Message,
	}
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	run.ID = row.ID
	run.CreatedAt = row.CreatedAt
	return nil

}

func (r *GormRepo) UpdateTaskRun(ctx context.Context, run domain.ScheduledTaskRun) error {

	return r.gdb.WithContext(ctx).Model(&scheduledTaskRunRow{}).Where("id = ?", run.ID).Updates(map[string]any{
		"status":       run.Status,
		"finished_at":  run.FinishedAt,
		"duration_sec": run.DurationSec,
		"message":      run.Message,
	}).Error

}

func (r *GormRepo) ListTaskRuns(ctx context.Context, key string, limit int) ([]domain.ScheduledTaskRun, error) {

	if limit <= 0 {
		limit = 20
	}
	var rows []scheduledTaskRunRow
	if err := r.gdb.WithContext(ctx).Where("task_key = ?", key).Order("id DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.ScheduledTaskRun, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.ScheduledTaskRun{
			ID:          row.ID,
			TaskKey:     row.TaskKey,
			Status:      row.Status,
			StartedAt:   row.StartedAt,
			FinishedAt:  row.FinishedAt,
			DurationSec: row.DurationSec,
			Message:     row.Message,
			CreatedAt:   row.CreatedAt,
		})
	}
	return out, nil

}

func (r *GormRepo) PurgeTaskRuns(ctx context.Context, before time.Time) error {
	return r.gdb.WithContext(ctx).
		Where("created_at < ?", before).
		Delete(&scheduledTaskRunRow{}).Error
}

// newFencingToken returns a random non-negative int64 used to distinguish lease
// generations. Predictability is not a concern; the token only needs to differ
// across acquisitions.
func newFencingToken() int64 {
	var b [8]byte
	_, _ = rand.Read(b[:])
	u := binary.LittleEndian.Uint64(b[:])
	if u > (1 << 63) {
		u = u & (1<<63 - 1)
	}
	return int64(u)
}

// AcquireTaskLease implements the scheduled-task DB CAS lease. Insertion wins
// immediately; an existing row is stolen only when its lease has expired, so a
// crashed worker (whose heartbeat stopped) is recoverable by the next
// contender.
func (r *GormRepo) AcquireTaskLease(ctx context.Context, taskKey, owner string, leaseFor time.Duration) (domain.ScheduledTaskLease, bool, error) {
	taskKey = strings.TrimSpace(taskKey)
	if taskKey == "" {
		return domain.ScheduledTaskLease{}, false, domain.ErrInvalidInput
	}
	if leaseFor <= 0 {
		leaseFor = 60 * time.Second
	}
	now := time.Now()
	lease := domain.ScheduledTaskLease{
		TaskKey:      taskKey,
		Owner:        strings.TrimSpace(owner),
		LeaseUntil:   now.Add(leaseFor),
		FencingToken: newFencingToken(),
	}
	res := r.gdb.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&scheduledTaskLeaseRow{
		TaskKey:      lease.TaskKey,
		Owner:        lease.Owner,
		LeaseUntil:   lease.LeaseUntil,
		FencingToken: lease.FencingToken,
	})
	if res.Error != nil {
		return domain.ScheduledTaskLease{}, false, res.Error
	}
	if res.RowsAffected > 0 {
		return lease, true, nil
	}
	// A row exists. Steal it only when the previous lease has expired (CAS).
	steal := r.gdb.WithContext(ctx).Model(&scheduledTaskLeaseRow{}).
		Where("task_key = ? AND (lease_until IS NULL OR lease_until < ?)", taskKey, now).
		Updates(map[string]any{
			"owner":         lease.Owner,
			"lease_until":   lease.LeaseUntil,
			"fencing_token": lease.FencingToken,
		})
	if steal.Error != nil {
		return domain.ScheduledTaskLease{}, false, steal.Error
	}
	if steal.RowsAffected > 0 {
		return lease, true, nil
	}
	return domain.ScheduledTaskLease{}, false, nil
}

// RenewTaskLease extends the lease only while this caller still holds it
// (owner + fencing token unchanged and not expired).
func (r *GormRepo) RenewTaskLease(ctx context.Context, taskKey, owner string, token int64, leaseFor time.Duration) (bool, error) {
	if leaseFor <= 0 {
		leaseFor = 60 * time.Second
	}
	res := r.gdb.WithContext(ctx).Model(&scheduledTaskLeaseRow{}).
		Where("task_key = ? AND owner = ? AND fencing_token = ? AND lease_until > ?", taskKey, owner, token, time.Now()).
		Update("lease_until", time.Now().Add(leaseFor))
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// ReleaseTaskLease removes the lease row on clean completion.
func (r *GormRepo) ReleaseTaskLease(ctx context.Context, taskKey, owner string, token int64) error {
	return r.gdb.WithContext(ctx).
		Where("task_key = ? AND owner = ? AND fencing_token = ?", taskKey, owner, token).
		Delete(&scheduledTaskLeaseRow{}).Error
}

// CompleteTaskRun writes the final run status only when the caller still holds
// the lease. When the lease was lost (stolen after expiry, or released), the
// update is a no-op and false is returned, so a stale worker cannot overwrite a
// newer run's state.
func (r *GormRepo) CompleteTaskRun(ctx context.Context, run domain.ScheduledTaskRun, taskKey, owner string, token int64) (bool, error) {
	err := r.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var cnt int64
		if err := tx.Model(&scheduledTaskLeaseRow{}).
			Where("task_key = ? AND owner = ? AND fencing_token = ? AND lease_until >= ?", taskKey, owner, token, time.Now()).
			Count(&cnt).Error; err != nil {
			return err
		}
		if cnt == 0 {
			return domain.ErrTaskLeaseLost
		}
		return tx.Model(&scheduledTaskRunRow{}).Where("id = ?", run.ID).Updates(map[string]any{
			"status":       run.Status,
			"finished_at":  run.FinishedAt,
			"duration_sec": run.DurationSec,
			"message":      run.Message,
		}).Error
	})
	if err == nil {
		return true, nil
	}
	if errors.Is(err, domain.ErrTaskLeaseLost) {
		return false, nil
	}
	return false, err
}

func (r *GormRepo) CreateResizeTask(ctx context.Context, task *domain.ResizeTask) error {

	row := resizeTaskRow{
		VPSID:       task.VPSID,
		OrderID:     task.OrderID,
		OrderItemID: task.OrderItemID,
		Status:      string(task.Status),
		ScheduledAt: task.ScheduledAt,
		StartedAt:   task.StartedAt,
		FinishedAt:  task.FinishedAt,
		LeaseOwner:  task.LeaseOwner,
		LeaseUntil:  task.LeaseUntil,
		Fence:       task.Fence,
	}
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	task.ID = row.ID
	task.CreatedAt = row.CreatedAt
	task.UpdatedAt = row.UpdatedAt
	return nil

}

func (r *GormRepo) GetResizeTask(ctx context.Context, id int64) (domain.ResizeTask, error) {

	var row resizeTaskRow
	if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return domain.ResizeTask{}, r.ensure(err)
	}
	return fromResizeTaskRow(row), nil

}

func (r *GormRepo) UpdateResizeTask(ctx context.Context, task domain.ResizeTask) (bool, error) {
	res := r.gdb.WithContext(ctx).Model(&resizeTaskRow{}).
		Where("id = ? AND lease_owner = ? AND fence = ?", task.ID, task.LeaseOwner, task.Fence).
		Updates(map[string]any{
			"status":       task.Status,
			"scheduled_at": task.ScheduledAt,
			"started_at":   task.StartedAt,
			"finished_at":  task.FinishedAt,
			"lease_owner":  task.LeaseOwner,
			"lease_until":  task.LeaseUntil,
			"updated_at":   time.Now(),
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// ClaimResizeTask atomically transitions a pending resize task to running, or
// reclaims a running task whose lease has elapsed (a crashed worker), assigning
// the lease to this worker and bumping the fence. A fresh lease is never stolen.
func (r *GormRepo) ClaimResizeTask(ctx context.Context, taskID int64, owner string, leaseUntil time.Time) (domain.ResizeTask, bool, error) {
	now := time.Now()
	updates := map[string]any{
		"status":      string(domain.ResizeTaskStatusRunning),
		"started_at":  now,
		"lease_owner": owner,
		"lease_until": leaseUntil,
		"fence":       gorm.Expr("fence + 1"),
		"updated_at":  now,
	}
	res := r.gdb.WithContext(ctx).Model(&resizeTaskRow{}).
		Where("id = ? AND status = ?", taskID, string(domain.ResizeTaskStatusPending)).
		Updates(updates)
	if res.Error != nil {
		return domain.ResizeTask{}, false, res.Error
	}
	if res.RowsAffected == 1 {
		return r.claimedResizeTask(ctx, taskID)
	}
	res = r.gdb.WithContext(ctx).Model(&resizeTaskRow{}).
		Where("id = ? AND status = ? AND (lease_until IS NULL OR lease_until <= ?)", taskID, string(domain.ResizeTaskStatusRunning), now).
		Updates(updates)
	if res.Error != nil {
		return domain.ResizeTask{}, false, res.Error
	}
	if res.RowsAffected == 1 {
		return r.claimedResizeTask(ctx, taskID)
	}
	return domain.ResizeTask{}, false, nil
}

func (r *GormRepo) claimedResizeTask(ctx context.Context, taskID int64) (domain.ResizeTask, bool, error) {
	task, err := r.GetResizeTask(ctx, taskID)
	if err != nil {
		return domain.ResizeTask{}, false, err
	}
	return task, true, nil
}

func (r *GormRepo) RenewResizeTaskLease(ctx context.Context, taskID int64, owner string, fence int, leaseUntil time.Time) (bool, error) {
	res := r.gdb.WithContext(ctx).Model(&resizeTaskRow{}).
		Where("id = ? AND lease_owner = ? AND fence = ?", taskID, owner, fence).
		Updates(map[string]any{
			"lease_until": leaseUntil,
			"updated_at":  time.Now(),
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

func fromResizeTaskRow(row resizeTaskRow) domain.ResizeTask {
	return domain.ResizeTask{
		ID:          row.ID,
		VPSID:       row.VPSID,
		OrderID:     row.OrderID,
		OrderItemID: row.OrderItemID,
		Status:      domain.ResizeTaskStatus(row.Status),
		ScheduledAt: row.ScheduledAt,
		StartedAt:   row.StartedAt,
		FinishedAt:  row.FinishedAt,
		LeaseOwner:  row.LeaseOwner,
		LeaseUntil:  row.LeaseUntil,
		Fence:       row.Fence,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

func (r *GormRepo) ListDueResizeTasks(ctx context.Context, limit int) ([]domain.ResizeTask, error) {

	if limit <= 0 {
		limit = 20
	}
	now := time.Now()
	// Legacy running tasks (created before the lease columns) carry a NULL
	// lease_until; they are reclaimed on the historical started_at staleness
	// threshold so pre-migration stuck tasks do not hang forever.
	staleCutoff := now.Add(-domain.ResizeTaskStaleDuration)
	var rows []resizeTaskRow
	if err := r.gdb.WithContext(ctx).
		Where("(status = ? AND (scheduled_at IS NULL OR scheduled_at <= ?)) OR (status = ? AND ((lease_until IS NOT NULL AND lease_until <= ?) OR (lease_until IS NULL AND (started_at IS NULL OR started_at <= ?))))",
			domain.ResizeTaskStatusPending, now,
			domain.ResizeTaskStatusRunning, now, staleCutoff).
		Order("scheduled_at ASC, id ASC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.ResizeTask, 0, len(rows))
	for _, row := range rows {
		out = append(out, fromResizeTaskRow(row))
	}
	return out, nil

}

func (r *GormRepo) HasPendingResizeTask(ctx context.Context, vpsID int64) (bool, error) {
	if vpsID <= 0 {
		return false, nil
	}
	var total int64
	if err := r.gdb.WithContext(ctx).Model(&resizeTaskRow{}).Where("vps_id = ? AND status IN ?", vpsID, []string{string(domain.ResizeTaskStatusPending), string(domain.ResizeTaskStatusRunning)}).Count(&total).Error; err != nil {
		return false, err
	}
	return total > 0, nil
}
