package order

import "context"

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
		if err := s.executeResizeTask(ctx, task); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
