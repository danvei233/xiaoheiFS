package passwordreset

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"

	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

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
		Token:     token,
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
	resetToken, err := s.tokens.GetPasswordResetToken(ctx, token)
	if err != nil {
		return appshared.ErrNotFound
	}

	if resetToken.Used {
		return domain.ErrTokenUsed
	}

	if time.Now().After(resetToken.ExpiresAt) {
		return domain.ErrTokenExpired
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	if err := s.users.UpdateUserPassword(ctx, resetToken.UserID, string(hash)); err != nil {
		return err
	}

	_ = s.tokens.MarkPasswordResetTokenUsed(ctx, resetToken.ID)

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
