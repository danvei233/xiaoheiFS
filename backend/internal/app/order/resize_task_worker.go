package order

import (
	"context"
	"fmt"
	"time"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/pkg/safego"
)

func (s *OrderService) ProcessResizeTasks(ctx context.Context, limit int) error {
	if s.resizeTasks == nil {
		return ErrInvalidInput
	}
	if limit <= 0 {
		limit = 20
	}
	tasks, err := s.resizeTasks.ListDueResizeTasks(ctx, limit)
	if err != nil {
		return err
	}
	var firstErr error
	for _, task := range tasks {
		// A panic in a buggy automation client must not take down the resize
		// runner. If this worker still holds the lease, mark the task failed so
		// it is not stuck running forever; a lost lease leaves the outcome to the
		// takeover worker.
		func() {
			defer func() {
				if r := recover(); r != nil {
					safego.LogPanic("order resize task", r)
					if current, err := s.resizeTasks.GetResizeTask(ctx, task.ID); err == nil && current.LeaseOwner == s.workerID {
						current.Status = domain.ResizeTaskStatusFailed
						finished := time.Now()
						current.FinishedAt = &finished
						current.LeaseUntil = nil
						_, _ = s.resizeTasks.UpdateResizeTask(ctx, current)
					}
				}
			}()
			if err := s.executeResizeTask(ctx, task); err != nil && firstErr == nil {
				firstErr = err
			}
		}()
	}
	return firstErr
}

func stringifyPanic(r any) string {
	return fmt.Sprintf("%v", r)
}
