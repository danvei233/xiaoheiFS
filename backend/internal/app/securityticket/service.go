package securityticket

import (
	"context"
	"strings"
	"time"

	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

type Service struct {
	repo appports.PasswordResetTicketRepository
}

func NewService(repo appports.PasswordResetTicketRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) DeleteExpired(ctx context.Context) error {
	if s == nil || s.repo == nil {
		return nil
	}
	return s.repo.DeleteExpiredPasswordResetTickets(ctx)
}

func (s *Service) Create(ctx context.Context, userID int64, channel, receiver, token string, expiresAt time.Time) error {
	if s == nil || s.repo == nil {
		return appshared.ErrNotFound
	}
	return s.repo.CreatePasswordResetTicket(ctx, &domain.PasswordResetTicket{
		UserID:    userID,
		Channel:   strings.TrimSpace(channel),
		Receiver:  strings.TrimSpace(receiver),
		Token:     strings.TrimSpace(token),
		ExpiresAt: expiresAt,
	})
}

func (s *Service) Get(ctx context.Context, token string) (domain.PasswordResetTicket, error) {
	if s == nil || s.repo == nil {
		return domain.PasswordResetTicket{}, appshared.ErrNotFound
	}
	return s.repo.GetPasswordResetTicket(ctx, strings.TrimSpace(token))
}

func (s *Service) MarkUsed(ctx context.Context, id int64) error {
	if s == nil || s.repo == nil {
		return appshared.ErrNotFound
	}
	return s.repo.MarkPasswordResetTicketUsed(ctx, id)
}
