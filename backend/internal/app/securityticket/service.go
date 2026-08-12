package securityticket

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

// hashToken derives the value persisted for a security ticket. The raw token
// is only ever returned to the authenticated user, so a database leak does not
// yield usable tickets (Issue #51 P1-2 / P3-35).
func hashToken(v string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(v)))
	return hex.EncodeToString(sum[:])
}

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
		Token:     hashToken(token),
		ExpiresAt: expiresAt,
	})
}

func (s *Service) Get(ctx context.Context, token string) (domain.PasswordResetTicket, error) {
	if s == nil || s.repo == nil {
		return domain.PasswordResetTicket{}, appshared.ErrNotFound
	}
	return s.repo.GetPasswordResetTicket(ctx, hashToken(token))
}

func (s *Service) MarkUsed(ctx context.Context, id int64) error {
	if s == nil || s.repo == nil {
		return appshared.ErrNotFound
	}
	return s.repo.MarkPasswordResetTicketUsed(ctx, id)
}

// MarkUsedIfUnused atomically claims a ticket for single-use consumption and
// reports whether this caller won the claim.
func (s *Service) MarkUsedIfUnused(ctx context.Context, id int64) (bool, error) {
	if s == nil || s.repo == nil {
		return false, appshared.ErrNotFound
	}
	return s.repo.MarkPasswordResetTicketUsedIfUnused(ctx, id)
}

// ResetPasswordWithTicket atomically claims a single-use ticket and updates the
// owner's password in one repository transaction. A failed bcrypt/database
// update rolls the claim back, so a usable ticket is never burned. It returns
// ErrTicketUsed when this caller did not win the claim.
func (s *Service) ResetPasswordWithTicket(ctx context.Context, ticketID, userID int64, newPassword string) error {
	if s == nil || s.repo == nil {
		return appshared.ErrNotFound
	}
	if strings.TrimSpace(newPassword) == "" {
		return appshared.ErrInvalidInput
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	claimed, err := s.repo.ConsumePasswordResetTicketAndUpdatePassword(ctx, ticketID, userID, string(hash), time.Now())
	if err != nil {
		return err
	}
	if !claimed {
		return domain.ErrTicketUsed
	}
	return nil
}

// ConsumeContactChange atomically claims a security ticket, re-checks the
// contact uniqueness constraint and applies the new email/phone. It returns
// ErrTicketUsed when this caller did not win the claim.
func (s *Service) ConsumeContactChange(ctx context.Context, ticketID, userID int64, kind, value string) (bool, error) {
	if s == nil || s.repo == nil {
		return false, appshared.ErrNotFound
	}
	return s.repo.ConsumeContactChangeTicketAndUpdateContact(ctx, ticketID, userID, kind, value, time.Now())
}
