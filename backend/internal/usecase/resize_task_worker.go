package usecase

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
	for _, task := range tasks {
		_ = s.executeResizeTask(ctx, task)
	}
	return nil
}
