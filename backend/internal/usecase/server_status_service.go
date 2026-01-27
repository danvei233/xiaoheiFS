package usecase

import "context"

type ServerStatusService struct {
	provider SystemInfoProvider
}

func NewServerStatusService(provider SystemInfoProvider) *ServerStatusService {
	return &ServerStatusService{provider: provider}
}

func (s *ServerStatusService) Status(ctx context.Context) (ServerStatus, error) {
	if s.provider == nil {
		return ServerStatus{}, ErrInvalidInput
	}
	return s.provider.Status(ctx)
}
