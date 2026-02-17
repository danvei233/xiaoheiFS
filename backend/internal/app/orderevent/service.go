package orderevent

import (
	"context"

	appports "xiaoheiplay/internal/app/ports"
	"xiaoheiplay/internal/domain"
)

type Service struct {
	repo appports.EventRepository
}

func NewService(repo appports.EventRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListAfter(ctx context.Context, orderID int64, afterSeq int64, limit int) ([]domain.OrderEvent, error) {
	if s == nil || s.repo == nil {
		return nil, nil
	}
	return s.repo.ListEventsAfter(ctx, orderID, afterSeq, limit)
}
