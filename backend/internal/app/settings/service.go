package settings

import (
	"context"

	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

type Service struct {
	repo appports.SettingsRepository
}

func NewService(repo appports.SettingsRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Get(ctx context.Context, key string) (domain.Setting, error) {
	if s == nil || s.repo == nil {
		return domain.Setting{}, appshared.ErrNotFound
	}
	return s.repo.GetSetting(ctx, key)
}

func (s *Service) List(ctx context.Context) ([]domain.Setting, error) {
	if s == nil || s.repo == nil {
		return nil, appshared.ErrNotFound
	}
	return s.repo.ListSettings(ctx)
}

func (s *Service) ListEmailTemplates(ctx context.Context) ([]domain.EmailTemplate, error) {
	if s == nil || s.repo == nil {
		return nil, appshared.ErrNotFound
	}
	return s.repo.ListEmailTemplates(ctx)
}
