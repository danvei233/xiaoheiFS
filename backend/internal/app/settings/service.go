package settings

import (
	"context"
	"time"

	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

type Service struct {
	repo   appports.SettingsRepository
	limits appports.RateLimitRepository
}

func NewService(repo appports.SettingsRepository) *Service {
	s := &Service{repo: repo}
	// The same GormRepo implements the rate-limit budget; expose it only when
	// the injected repo is capable, so callers stay decoupled from GORM.
	if rl, ok := repo.(appports.RateLimitRepository); ok {
		s.limits = rl
	}
	return s
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

// AllowRateLimit delegates to the DB atomic rate-limit budget.
func (s *Service) AllowRateLimit(ctx context.Context, bucket string, max int, window time.Duration) (bool, error) {
	if s == nil || s.limits == nil {
		return false, appshared.ErrNotFound
	}
	return s.limits.AllowRateLimit(ctx, bucket, max, window)
}
