package passwordreset

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

// hashToken derives the value persisted for a reset token. The raw token is
// only ever emailed to the account owner, so a database leak does not yield
// usable tokens (Issue #51 P1-2 / P3-35).
func hashToken(v string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(v)))
	return hex.EncodeToString(sum[:])
}

type Service struct {
	users     appports.UserRepository
	tokens    appports.PasswordResetTokenRepository
	email     appports.EmailSender
	templates appports.SettingsRepository
}

func NewService(users appports.UserRepository, tokens appports.PasswordResetTokenRepository, email appports.EmailSender, templates appports.SettingsRepository) *Service {
	return &Service{
		users:     users,
		tokens:    tokens,
		email:     email,
		templates: templates,
	}
}

func (s *Service) RequestReset(ctx context.Context, email string) error {
	user, err := s.users.GetUserByUsernameOrEmail(ctx, email)
	if err != nil {
		if errors.Is(err, appshared.ErrNotFound) {
			return nil
		}
		return err
	}

	if user.Role != domain.UserRoleAdmin {
		return nil
	}

	_ = s.tokens.DeleteExpiredTokens(ctx)

	token, err := s.generateToken()
	if err != nil {
		return err
	}

	resetToken := &domain.PasswordResetToken{
		UserID:    user.ID,
		Token:     hashToken(token),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Used:      false,
	}

	if err := s.tokens.CreatePasswordResetToken(ctx, resetToken); err != nil {
		return err
	}

	if s.email != nil {
		_ = s.sendResetEmail(ctx, user, token)
	}

	return nil
}

func (s *Service) ResetPassword(ctx context.Context, token string, newPassword string) error {
	newPassword, err := trimAndValidateRequired(newPassword, maxLenPassword)
	if err != nil {
		return appshared.ErrInvalidInput
	}
	resetToken, err := s.tokens.GetPasswordResetToken(ctx, hashToken(token))
	if err != nil {
		return appshared.ErrNotFound
	}

	if resetToken.Used {
		return domain.ErrTokenUsed
	}

	if time.Now().After(resetToken.ExpiresAt) {
		return domain.ErrTokenExpired
	}

	// Hash before entering the transaction so a bcrypt failure is not mistaken
	// for a consumed token, then claim + update the password atomically. A
	// failed password update rolls back the claim, so a usable token is never
	// burned (Issue #51 P2-8).
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	claimed, err := s.tokens.ConsumePasswordResetTokenAndUpdatePassword(ctx, resetToken.ID, resetToken.UserID, string(hash), time.Now())
	if err != nil {
		return err
	}
	if !claimed {
		return domain.ErrTokenUsed
	}

	return nil
}

func (s *Service) generateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (s *Service) sendResetEmail(ctx context.Context, user domain.User, token string) error {
	templates, _ := s.templates.ListEmailTemplates(ctx)
	subject := "Password Reset Request"
	body := `Use the following token to reset your password: {{ .token }}`

	for _, tmpl := range templates {
		if tmpl.Name == "password_reset" && tmpl.Enabled {
			subject = tmpl.Subject
			body = tmpl.Body
			break
		}
	}

	data := map[string]any{
		"user": map[string]any{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"qq":       user.QQ,
		},
		"token": token,
	}

	renderedSubject := appshared.RenderTemplate(subject, data, false)
	renderedBody := appshared.RenderTemplate(body, data, appshared.IsHTMLContent(body))

	return s.email.Send(ctx, user.Email, renderedSubject, renderedBody)
}
