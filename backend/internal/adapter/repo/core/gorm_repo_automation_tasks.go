package repo

import (
	"context"
	"gorm.io/gorm/clause"
	"time"
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
		ID:          job.ID,
		OrderID:     job.OrderID,
		OrderItemID: job.OrderItemID,
		HostID:      job.HostID,
		HostName:    job.HostName,
		Status:      job.Status,
		Attempts:    job.Attempts,
		NextRunAt:   job.NextRunAt,
		LastError:   job.LastError,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := r.gdb.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "order_item_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"host_id", "host_name", "status", "attempts", "next_run_at", "last_error", "updated_at",
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

func (r *GormRepo) ListDueProvisionJobs(ctx context.Context, limit int) ([]domain.ProvisionJob, error) {
	if limit <= 0 {
		limit = 20
	}
	var rows []provisionJobRow
	if err := r.gdb.WithContext(ctx).
		Where("status IN ? AND next_run_at <= ?", []string{"pending", "retry", "running"}, time.Now()).
		Order("id ASC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.ProvisionJob, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.ProvisionJob{
			ID:          row.ID,
			OrderID:     row.OrderID,
			OrderItemID: row.OrderItemID,
			HostID:      row.HostID,
			HostName:    row.HostName,
			Status:      row.Status,
			Attempts:    row.Attempts,
			NextRunAt:   row.NextRunAt,
			LastError:   row.LastError,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		})
	}
	return out, nil
}

func (r *GormRepo) UpdateProvisionJob(ctx context.Context, job domain.ProvisionJob) error {
	return r.gdb.WithContext(ctx).Model(&provisionJobRow{}).Where("id = ?", job.ID).Updates(map[string]any{
		"status":      job.Status,
		"attempts":    job.Attempts,
		"next_run_at": job.NextRunAt,
		"last_error":  job.LastError,
		"updated_at":  time.Now(),
	}).Error
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

func (r *GormRepo) CreateResizeTask(ctx context.Context, task *domain.ResizeTask) error {

	row := resizeTaskRow{
		VPSID:       task.VPSID,
		OrderID:     task.OrderID,
		OrderItemID: task.OrderItemID,
		Status:      string(task.Status),
		ScheduledAt: task.ScheduledAt,
		StartedAt:   task.StartedAt,
		FinishedAt:  task.FinishedAt,
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
	return domain.ResizeTask{
		ID:          row.ID,
		VPSID:       row.VPSID,
		OrderID:     row.OrderID,
		OrderItemID: row.OrderItemID,
		Status:      domain.ResizeTaskStatus(row.Status),
		ScheduledAt: row.ScheduledAt,
		StartedAt:   row.StartedAt,
		FinishedAt:  row.FinishedAt,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}, nil

}

func (r *GormRepo) UpdateResizeTask(ctx context.Context, task domain.ResizeTask) error {

	return r.gdb.WithContext(ctx).Model(&resizeTaskRow{}).Where("id = ?", task.ID).Updates(map[string]any{
		"status":       task.Status,
		"scheduled_at": task.ScheduledAt,
		"started_at":   task.StartedAt,
		"finished_at":  task.FinishedAt,
		"updated_at":   time.Now(),
	}).Error

}

func (r *GormRepo) ListDueResizeTasks(ctx context.Context, limit int) ([]domain.ResizeTask, error) {

	if limit <= 0 {
		limit = 20
	}
	var rows []resizeTaskRow
	if err := r.gdb.WithContext(ctx).
		Where("status = ? AND (scheduled_at IS NULL OR scheduled_at <= CURRENT_TIMESTAMP)", domain.ResizeTaskStatusPending).
		Order("scheduled_at ASC, id ASC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.ResizeTask, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.ResizeTask{
			ID:          row.ID,
			VPSID:       row.VPSID,
			OrderID:     row.OrderID,
			OrderItemID: row.OrderItemID,
			Status:      domain.ResizeTaskStatus(row.Status),
			ScheduledAt: row.ScheduledAt,
			StartedAt:   row.StartedAt,
			FinishedAt:  row.FinishedAt,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		})
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
