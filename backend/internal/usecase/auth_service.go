package usecase

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
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

const (
	CodeComplexityDigits  = "digits"
	CodeComplexityLetters = "letters"
	CodeComplexityAlnum   = "alnum"
)

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
	return s.CreateCaptchaWithPolicy(ctx, ttl, 5, CodeComplexityAlnum)
}

func (s *AuthService) CreateCaptchaWithPolicy(ctx context.Context, ttl time.Duration, length int, complexity string) (domain.Captcha, string, error) {
	code, err := randomCodeByPolicy(length, complexity)
	if err != nil {
		return domain.Captcha{}, "", err
	}
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
	email, err := trimAndValidateOptional(in.Email, maxLenEmail)
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
	if email != "" {
		if _, err := s.users.GetUserByUsernameOrEmail(ctx, email); err == nil {
			return domain.User{}, ErrConflict
		}
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
	if err != nil && looksLikePhoneAccount(usernameOrEmail) {
		if byPhone, phoneErr := s.users.GetUserByPhone(ctx, usernameOrEmail); phoneErr == nil {
			user = byPhone
			err = nil
		}
	}
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

func looksLikePhoneAccount(v string) bool {
	v = strings.TrimSpace(v)
	if len(v) < 6 {
		return false
	}
	for _, r := range v {
		if (r < '0' || r > '9') && r != '+' && r != '-' && r != ' ' {
			return false
		}
	}
	return true
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

func (s *AuthService) UpdateLoginSecurity(ctx context.Context, userID int64, ip, city, tz string, at time.Time) error {
	if userID <= 0 {
		return ErrInvalidInput
	}
	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	user.LastLoginIP = strings.TrimSpace(ip)
	user.LastLoginCity = strings.TrimSpace(city)
	user.LastLoginTZ = strings.TrimSpace(tz)
	if at.IsZero() {
		at = time.Now()
	}
	user.LastLoginAt = &at
	return s.users.UpdateUser(ctx, user)
}

func (s *AuthService) SetupTOTP(ctx context.Context, userID int64, password, currentCode string) (string, string, error) {
	if userID <= 0 {
		return "", "", ErrInvalidInput
	}
	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return "", "", ErrNotFound
	}
	if user.TOTPEnabled {
		if ok := verifyTOTPCode(decryptText(user.TOTPSecretEnc), strings.TrimSpace(currentCode), time.Now(), 1); !ok {
			return "", "", ErrUnauthorized
		}
	} else {
		if err := s.VerifyPassword(ctx, userID, password); err != nil {
			return "", "", err
		}
	}
	secret := generateTOTPSecret()
	user.TOTPPendingSecretEnc = encryptText(secret)
	if err := s.users.UpdateUser(ctx, user); err != nil {
		return "", "", err
	}
	issuer := "XiaoHei"
	otpURL := fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s", url.QueryEscape(issuer), url.QueryEscape(user.Username), secret, url.QueryEscape(issuer))
	return secret, otpURL, nil
}

func (s *AuthService) ConfirmTOTP(ctx context.Context, userID int64, code string) error {
	if userID <= 0 {
		return ErrInvalidInput
	}
	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return ErrNotFound
	}
	secret := decryptText(user.TOTPPendingSecretEnc)
	if strings.TrimSpace(secret) == "" {
		return ErrNotFound
	}
	if !verifyTOTPCode(secret, strings.TrimSpace(code), time.Now(), 1) {
		return ErrUnauthorized
	}
	user.TOTPSecretEnc = user.TOTPPendingSecretEnc
	user.TOTPPendingSecretEnc = ""
	user.TOTPEnabled = true
	return s.users.UpdateUser(ctx, user)
}

func (s *AuthService) VerifyTOTP(ctx context.Context, userID int64, code string) error {
	if userID <= 0 {
		return ErrInvalidInput
	}
	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return ErrNotFound
	}
	if !user.TOTPEnabled {
		return ErrForbidden
	}
	if !verifyTOTPCode(decryptText(user.TOTPSecretEnc), strings.TrimSpace(code), time.Now(), 1) {
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
	return s.CreateVerificationCodeWithPolicy(ctx, channel, receiver, purpose, ttl, 6, CodeComplexityAlnum)
}

func (s *AuthService) CreateVerificationCodeWithPolicy(ctx context.Context, channel, receiver, purpose string, ttl time.Duration, length int, complexity string) (string, error) {
	if strings.TrimSpace(channel) == "" || strings.TrimSpace(receiver) == "" || strings.TrimSpace(purpose) == "" {
		return "", ErrInvalidInput
	}
	if s.verify == nil {
		return "", ErrNotSupported
	}
	code, err := randomCodeByPolicy(length, complexity)
	if err != nil {
		return "", err
	}
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
	out, err := randomCodeByPolicy(n, CodeComplexityAlnum)
	if err != nil {
		return ""
	}
	return out
}

func randomCodeByPolicy(n int, complexity string) (string, error) {
	if n < 4 || n > 12 {
		return "", ErrInvalidInput
	}
	letters := []rune("ABCDEFGHJKLMNPQRSTUVWXYZ")
	digits := []rune("0123456789")
	complexity = strings.ToLower(strings.TrimSpace(complexity))
	charset := []rune{}
	switch complexity {
	case CodeComplexityDigits:
		charset = digits
	case CodeComplexityLetters:
		charset = letters
	case CodeComplexityAlnum:
		charset = append(append([]rune{}, letters...), digits...)
	default:
		return "", ErrInvalidInput
	}
	if len(charset) == 0 {
		return "", ErrInvalidInput
	}
	b := make([]byte, n)
	_, _ = rand.Read(b)
	out := make([]rune, n)
	for i := range out {
		out[i] = charset[int(b[i])%len(charset)]
	}
	return string(out), nil
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

var totpEncryptKey = sha256.Sum256([]byte("xiaoheiplay-totp-v1"))

func encryptText(v string) string {
	if strings.TrimSpace(v) == "" {
		return ""
	}
	block, err := aes.NewCipher(totpEncryptKey[:])
	if err != nil {
		return v
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return v
	}
	nonce := make([]byte, gcm.NonceSize())
	_, _ = rand.Read(nonce)
	ciphertext := gcm.Seal(nil, nonce, []byte(v), nil)
	raw := map[string]string{
		"n": base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(nonce),
		"c": base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(ciphertext),
	}
	b, _ := json.Marshal(raw)
	return string(b)
}

func decryptText(v string) string {
	if strings.TrimSpace(v) == "" {
		return ""
	}
	var raw map[string]string
	if err := json.Unmarshal([]byte(v), &raw); err != nil {
		return v
	}
	nonce, errN := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(raw["n"])
	ciphertext, errC := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(raw["c"])
	if errN != nil || errC != nil {
		return ""
	}
	block, err := aes.NewCipher(totpEncryptKey[:])
	if err != nil {
		return ""
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return ""
	}
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return ""
	}
	return string(plain)
}

func generateTOTPSecret() string {
	b := make([]byte, 20)
	_, _ = rand.Read(b)
	return strings.TrimRight(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b), "=")
}

func verifyTOTPCode(secret, code string, now time.Time, window int) bool {
	if strings.TrimSpace(secret) == "" || strings.TrimSpace(code) == "" {
		return false
	}
	code = strings.TrimSpace(code)
	for i := -window; i <= window; i++ {
		counter := uint64(now.Add(time.Duration(i)*30*time.Second).Unix() / 30)
		if generateTOTPCode(secret, counter) == code {
			return true
		}
	}
	return false
}

func generateTOTPCode(secret string, counter uint64) string {
	secret = strings.ToUpper(strings.TrimSpace(secret))
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		return ""
	}
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], counter)
	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(msg[:])
	sum := mac.Sum(nil)
	if len(sum) < 20 {
		return ""
	}
	offset := int(sum[len(sum)-1] & 0x0f)
	bin := int32(sum[offset]&0x7f)<<24 | int32(sum[offset+1])<<16 | int32(sum[offset+2])<<8 | int32(sum[offset+3])
	otp := int(bin % 1000000)
	return fmt.Sprintf("%06d", otp)
}
