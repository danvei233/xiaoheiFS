package upload

import (
	"context"

	appports "xiaoheiplay/internal/app/ports"
	"xiaoheiplay/internal/domain"
)

type Service struct {
	repo appports.UploadRepository
}

func NewService(repo appports.UploadRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, item *domain.Upload) error {
	return s.repo.CreateUpload(ctx, item)
}

func (s *Service) List(ctx context.Context, limit, offset int) ([]domain.Upload, int, error) {
	return s.repo.ListUploads(ctx, limit, offset)
}
