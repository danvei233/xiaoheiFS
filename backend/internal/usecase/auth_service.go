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
}

type RegisterInput struct {
	Username    string
	Email       string
	QQ          string
	Phone       string
	Password    string
	CaptchaID   string
	CaptchaCode string
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

func NewAuthService(users UserRepository, captchas CaptchaRepository) *AuthService {
	return &AuthService{users: users, captchas: captchas}
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
	if in.Username == "" || in.Password == "" || in.Email == "" {
		return domain.User{}, ErrInvalidInput
	}
	if err := s.verifyCaptcha(ctx, in.CaptchaID, in.CaptchaCode); err != nil {
		return domain.User{}, err
	}
	if _, err := s.users.GetUserByUsernameOrEmail(ctx, in.Username); err == nil {
		return domain.User{}, ErrConflict
	}
	if _, err := s.users.GetUserByUsernameOrEmail(ctx, in.Email); err == nil {
		return domain.User{}, ErrConflict
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return domain.User{}, err
	}
	user := domain.User{
		Username:     in.Username,
		Email:        in.Email,
		QQ:           in.QQ,
		Phone:        in.Phone,
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
	if usernameOrEmail == "" || password == "" {
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

func (s *AuthService) UpdateProfile(ctx context.Context, userID int64, in UpdateProfileInput) (domain.User, error) {
	if userID == 0 {
		return domain.User{}, ErrInvalidInput
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
