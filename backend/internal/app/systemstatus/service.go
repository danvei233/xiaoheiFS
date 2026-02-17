package systemstatus

import (
	"context"

	appshared "xiaoheiplay/internal/app/shared"
)

type ServerStatus = appshared.ServerStatus
type SystemInfoProvider = appshared.SystemInfoProvider

type Service struct {
	provider SystemInfoProvider
}

func NewService(provider SystemInfoProvider) *Service {
	return &Service{provider: provider}
}

func (s *Service) Status(ctx context.Context) (ServerStatus, error) {
	if s.provider == nil {
		return ServerStatus{}, appshared.ErrInvalidInput
	}
	return s.provider.Status(ctx)
}
