package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"xiaoheiplay/internal/domain"
)

type AuthService struct {
	users    UserRepository
	captchas CaptchaRepository
	verify   VerificationCodeRepository
}

type RegisterInput struct {
	Username        string
	Email           string
	QQ              string
	Phone           string
	Password        string
	CaptchaID       string
	CaptchaCode     string
	CaptchaRequired bool
}

type UpdateProfileInput struct {
	Username string
	Email    string
	QQ       string
	Phone    string
	Bio      string
	Intro    string
	Password string
}

func NewAuthService(users UserRepository, captchas CaptchaRepository, verify VerificationCodeRepository) *AuthService {
	return &AuthService{users: users, captchas: captchas, verify: verify}
}

func (s *AuthService) CreateCaptcha(ctx context.Context, ttl time.Duration) (domain.Captcha, string, error) {
	code := randomCode(5)
	id := randomID(12)
	captcha := domain.Captcha{
		ID:        id,
		CodeHash:  hashText(code),
		ExpiresAt: time.Now().Add(ttl),
		CreatedAt: time.Now(),
	}
	if err := s.captchas.CreateCaptcha(ctx, captcha); err != nil {
		return domain.Captcha{}, "", err
	}
	return captcha, code, nil
}

func (s *AuthService) Register(ctx context.Context, in RegisterInput) (domain.User, error) {
	username, err := trimAndValidateRequired(in.Username, maxLenUsername)
	if err != nil {
		return domain.User{}, ErrInvalidInput
	}
	email, err := trimAndValidateRequired(in.Email, maxLenEmail)
	if err != nil {
		return domain.User{}, ErrInvalidInput
	}
	password, err := trimAndValidateRequired(in.Password, maxLenPassword)
	if err != nil {
		return domain.User{}, ErrInvalidInput
	}
	qq, err := trimAndValidateOptional(in.QQ, maxLenQQ)
	if err != nil {
		return domain.User{}, ErrInvalidInput
	}
	phone, err := trimAndValidateOptional(in.Phone, maxLenPhone)
	if err != nil {
		return domain.User{}, ErrInvalidInput
	}
	if in.CaptchaID != "" || in.CaptchaCode != "" || in.CaptchaRequired {
		if err := s.verifyCaptcha(ctx, in.CaptchaID, in.CaptchaCode); err != nil {
			return domain.User{}, err
		}
	}
	if _, err := s.users.GetUserByUsernameOrEmail(ctx, username); err == nil {
		return domain.User{}, ErrConflict
	}
	if _, err := s.users.GetUserByUsernameOrEmail(ctx, email); err == nil {
		return domain.User{}, ErrConflict
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return domain.User{}, err
	}
	user := domain.User{
		Username:     username,
		Email:        email,
		QQ:           qq,
		Phone:        phone,
		PasswordHash: string(hash),
		Role:         domain.UserRoleUser,
		Status:       domain.UserStatusActive,
	}
	if err := s.users.CreateUser(ctx, &user); err != nil {
		return domain.User{}, err
	}
	return user, nil
}

func (s *AuthService) Login(ctx context.Context, usernameOrEmail string, password string) (domain.User, error) {
	usernameOrEmail, err := trimAndValidateRequired(usernameOrEmail, maxLenEmail)
	if err != nil {
		return domain.User{}, ErrInvalidInput
	}
	password, err = trimAndValidateRequired(password, maxLenPassword)
	if err != nil {
		return domain.User{}, ErrInvalidInput
	}
	user, err := s.users.GetUserByUsernameOrEmail(ctx, usernameOrEmail)
	if err != nil {
		return domain.User{}, ErrUnauthorized
	}
	if user.Status != domain.UserStatusActive {
		return domain.User{}, ErrForbidden
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return domain.User{}, ErrUnauthorized
	}
	return user, nil
}

func (s *AuthService) VerifyPassword(ctx context.Context, userID int64, password string) error {
	password, err := trimAndValidateRequired(password, maxLenPassword)
	if userID == 0 || err != nil {
		return ErrInvalidInput
	}
	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return ErrUnauthorized
	}
	if user.Status != domain.UserStatusActive {
		return ErrForbidden
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return ErrUnauthorized
	}
	return nil
}

func (s *AuthService) UpdateProfile(ctx context.Context, userID int64, in UpdateProfileInput) (domain.User, error) {
	if userID == 0 {
		return domain.User{}, ErrInvalidInput
	}
	if in.Username != "" {
		normalized, err := trimAndValidateRequired(in.Username, maxLenUsername)
		if err != nil {
			return domain.User{}, ErrInvalidInput
		}
		in.Username = normalized
	}
	if in.Email != "" {
		normalized, err := trimAndValidateRequired(in.Email, maxLenEmail)
		if err != nil {
			return domain.User{}, ErrInvalidInput
		}
		in.Email = normalized
	}
	if in.QQ != "" {
		normalized, err := trimAndValidateOptional(in.QQ, maxLenQQ)
		if err != nil {
			return domain.User{}, ErrInvalidInput
		}
		in.QQ = normalized
	}
	if in.Phone != "" {
		normalized, err := trimAndValidateOptional(in.Phone, maxLenPhone)
		if err != nil {
			return domain.User{}, ErrInvalidInput
		}
		in.Phone = normalized
	}
	if in.Bio != "" {
		normalized, err := trimAndValidateOptional(in.Bio, maxLenBio)
		if err != nil {
			return domain.User{}, ErrInvalidInput
		}
		in.Bio = normalized
	}
	if in.Intro != "" {
		normalized, err := trimAndValidateOptional(in.Intro, maxLenIntro)
		if err != nil {
			return domain.User{}, ErrInvalidInput
		}
		in.Intro = normalized
	}
	if in.Password != "" {
		normalized, err := trimAndValidateRequired(in.Password, maxLenPassword)
		if err != nil {
			return domain.User{}, ErrInvalidInput
		}
		in.Password = normalized
	}
	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return domain.User{}, err
	}
	if in.Username != "" && in.Username != user.Username {
		if existing, err := s.users.GetUserByUsernameOrEmail(ctx, in.Username); err == nil && existing.ID != user.ID {
			return domain.User{}, ErrConflict
		}
		user.Username = in.Username
	}
	if in.Email != "" && in.Email != user.Email {
		if existing, err := s.users.GetUserByUsernameOrEmail(ctx, in.Email); err == nil && existing.ID != user.ID {
			return domain.User{}, ErrConflict
		}
		user.Email = in.Email
	}
	if in.QQ != "" {
		user.QQ = in.QQ
	}
	if in.Phone != "" {
		user.Phone = in.Phone
	}
	if in.Bio != "" {
		user.Bio = in.Bio
	}
	if in.Intro != "" {
		user.Intro = in.Intro
	}
	if err := s.users.UpdateUser(ctx, user); err != nil {
		return domain.User{}, err
	}
	if in.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
		if err != nil {
			return domain.User{}, err
		}
		if err := s.users.UpdateUserPassword(ctx, user.ID, string(hash)); err != nil {
			return domain.User{}, err
		}
	}
	return s.users.GetUserByID(ctx, user.ID)
}

func (s *AuthService) verifyCaptcha(ctx context.Context, id, code string) error {
	if id == "" || code == "" {
		return ErrCaptchaFailed
	}
	captcha, err := s.captchas.GetCaptcha(ctx, id)
	if err != nil {
		return ErrCaptchaFailed
	}
	if time.Now().After(captcha.ExpiresAt) {
		_ = s.captchas.DeleteCaptcha(ctx, id)
		return ErrCaptchaFailed
	}
	if captcha.CodeHash != hashText(strings.ToUpper(code)) {
		return ErrCaptchaFailed
	}
	_ = s.captchas.DeleteCaptcha(ctx, id)
	return nil
}

func (s *AuthService) VerifyCaptcha(ctx context.Context, id, code string) error {
	return s.verifyCaptcha(ctx, id, code)
}

func (s *AuthService) CreateVerificationCode(ctx context.Context, channel, receiver, purpose string, ttl time.Duration) (string, error) {
	if strings.TrimSpace(channel) == "" || strings.TrimSpace(receiver) == "" || strings.TrimSpace(purpose) == "" {
		return "", ErrInvalidInput
	}
	if s.verify == nil {
		return "", ErrNotSupported
	}
	code := randomCode(6)
	item := domain.VerificationCode{
		Channel:   strings.TrimSpace(channel),
		Receiver:  strings.TrimSpace(receiver),
		Purpose:   strings.TrimSpace(purpose),
		CodeHash:  hashText(strings.ToUpper(code)),
		ExpiresAt: time.Now().Add(ttl),
		CreatedAt: time.Now(),
	}
	if err := s.verify.CreateVerificationCode(ctx, item); err != nil {
		return "", err
	}
	return code, nil
}

func (s *AuthService) VerifyVerificationCode(ctx context.Context, channel, receiver, purpose, code string) error {
	if strings.TrimSpace(channel) == "" || strings.TrimSpace(receiver) == "" || strings.TrimSpace(purpose) == "" || strings.TrimSpace(code) == "" {
		return ErrInvalidInput
	}
	if s.verify == nil {
		return ErrNotSupported
	}
	item, err := s.verify.GetLatestVerificationCode(ctx, strings.TrimSpace(channel), strings.TrimSpace(receiver), strings.TrimSpace(purpose))
	if err != nil {
		return ErrCaptchaFailed
	}
	if time.Now().After(item.ExpiresAt) {
		_ = s.verify.DeleteVerificationCodes(ctx, item.Channel, item.Receiver, item.Purpose)
		return ErrCaptchaFailed
	}
	if item.CodeHash != hashText(strings.ToUpper(code)) {
		return ErrCaptchaFailed
	}
	_ = s.verify.DeleteVerificationCodes(ctx, item.Channel, item.Receiver, item.Purpose)
	return nil
}

func randomCode(n int) string {
	letters := []rune("ABCDEFGHJKLMNPQRSTUVWXYZ23456789")
	b := make([]byte, n)
	_, _ = rand.Read(b)
	out := make([]rune, n)
	for i := range out {
		out[i] = letters[int(b[i])%len(letters)]
	}
	return string(out)
}

func randomID(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return strings.TrimRight(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b), "=")
}

func hashText(v string) string {
	sum := sha256.Sum256([]byte(v))
	return hex.EncodeToString(sum[:])
}
