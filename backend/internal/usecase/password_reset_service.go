package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"

	"xiaoheiplay/internal/domain"
)

type PasswordResetService struct {
	users     UserRepository
	tokens    PasswordResetTokenRepository
	email     EmailSender
	templates SettingsRepository
}

func NewPasswordResetService(users UserRepository, tokens PasswordResetTokenRepository, email EmailSender, templates SettingsRepository) *PasswordResetService {
	return &PasswordResetService{
		users:     users,
		tokens:    tokens,
		email:     email,
		templates: templates,
	}
}

func (s *PasswordResetService) RequestReset(ctx context.Context, email string) error {
	user, err := s.users.GetUserByUsernameOrEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
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

func (s *PasswordResetService) ResetPassword(ctx context.Context, token string, newPassword string) error {
	resetToken, err := s.tokens.GetPasswordResetToken(ctx, token)
	if err != nil {
		return ErrNotFound
	}

	if resetToken.Used {
		return errors.New("token already used")
	}

	if time.Now().After(resetToken.ExpiresAt) {
		return errors.New("token expired")
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

func (s *PasswordResetService) generateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (s *PasswordResetService) sendResetEmail(ctx context.Context, user domain.User, token string) error {
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

	renderedSubject := RenderTemplate(subject, data, false)
	renderedBody := RenderTemplate(body, data, IsHTMLContent(body))

	return s.email.Send(ctx, user.Email, renderedSubject, renderedBody)
}
