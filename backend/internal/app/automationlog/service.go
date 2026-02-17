package automationlog

import (
	"context"

	appports "xiaoheiplay/internal/app/ports"
	"xiaoheiplay/internal/domain"
)

type Service struct {
	repo appports.AutomationLogRepository
}

func NewService(repo appports.AutomationLogRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, orderID int64, limit, offset int) ([]domain.AutomationLog, int, error) {
	if s == nil || s.repo == nil {
		return nil, 0, nil
	}
	return s.repo.ListAutomationLogs(ctx, orderID, limit, offset)
}
