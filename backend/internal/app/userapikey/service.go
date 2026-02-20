package userapikey

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

const (
	DefaultSignatureWindowSec = 300
)

type Service struct {
	repo appports.UserAPIKeyRepository
}

type CreateResult struct {
	Key    domain.UserAPIKey
	Secret string
}

func NewService(repo appports.UserAPIKeyRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, userID int64, name string, scopes []string) (CreateResult, error) {
	name = strings.TrimSpace(name)
	if userID <= 0 || name == "" {
		return CreateResult{}, appshared.ErrInvalidInput
	}
	akid, err := randomToken("uak", 10)
	if err != nil {
		return CreateResult{}, err
	}
	secretRaw, err := randomToken("usk", 24)
	if err != nil {
		return CreateResult{}, err
	}
	secret := hashSecret(secretRaw)
	key := domain.UserAPIKey{
		UserID:     userID,
		Name:       name,
		AKID:       akid,
		KeyHash:    secret,
		Status:     domain.APIKeyStatusActive,
		ScopesJSON: strings.Join(scopes, ","),
	}
	if err := s.repo.CreateUserAPIKey(ctx, &key); err != nil {
		return CreateResult{}, err
	}
	return CreateResult{Key: key, Secret: secret}, nil
}

func (s *Service) List(ctx context.Context, userID int64, limit, offset int) ([]domain.UserAPIKey, int, error) {
	if userID <= 0 {
		return nil, 0, appshared.ErrInvalidInput
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListUserAPIKeys(ctx, userID, limit, offset)
}

func (s *Service) UpdateStatus(ctx context.Context, userID, id int64, status domain.APIKeyStatus) error {
	if userID <= 0 || id <= 0 {
		return appshared.ErrInvalidInput
	}
	if status != domain.APIKeyStatusActive && status != domain.APIKeyStatusDisabled {
		return appshared.ErrInvalidInput
	}
	return s.repo.UpdateUserAPIKeyStatus(ctx, userID, id, status)
}

func (s *Service) Delete(ctx context.Context, userID, id int64) error {
	if userID <= 0 || id <= 0 {
		return appshared.ErrInvalidInput
	}
	return s.repo.DeleteUserAPIKey(ctx, userID, id)
}

func (s *Service) ValidateSignature(ctx context.Context, akid, timestamp, nonce, signature, canonical string) (domain.UserAPIKey, error) {
	akid = strings.TrimSpace(akid)
	timestamp = strings.TrimSpace(timestamp)
	nonce = strings.TrimSpace(nonce)
	signature = strings.TrimSpace(signature)
	if akid == "" || timestamp == "" || nonce == "" || signature == "" || canonical == "" {
		return domain.UserAPIKey{}, appshared.ErrUnauthorized
	}
	key, err := s.repo.GetUserAPIKeyByAKID(ctx, akid)
	if err != nil {
		return domain.UserAPIKey{}, appshared.ErrUnauthorized
	}
	if key.Status != domain.APIKeyStatusActive {
		return domain.UserAPIKey{}, appshared.ErrForbidden
	}
	if !verifySignature(key.KeyHash, canonical, signature) {
		return domain.UserAPIKey{}, appshared.ErrUnauthorized
	}
	_ = s.repo.TouchUserAPIKey(ctx, key.ID)
	return key, nil
}

func ParseAndCheckTimestamp(raw string, now time.Time, windowSec int) (time.Time, error) {
	if windowSec <= 0 {
		windowSec = DefaultSignatureWindowSec
	}
	ts, err := time.Parse(time.RFC3339, strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}, appshared.ErrUnauthorized
	}
	delta := now.Sub(ts)
	if delta < 0 {
		delta = -delta
	}
	if delta > time.Duration(windowSec)*time.Second {
		return time.Time{}, appshared.ErrUnauthorized
	}
	return ts, nil
}

func BuildCanonical(method, path, rawQuery, timestamp, nonce string, body []byte) string {
	hash := sha256.Sum256(body)
	return strings.Join([]string{
		strings.ToUpper(strings.TrimSpace(method)),
		path,
		rawQuery,
		strings.TrimSpace(timestamp),
		strings.TrimSpace(nonce),
		hex.EncodeToString(hash[:]),
	}, "\n")
}

func verifySignature(secret, canonical, signature string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(canonical))
	expected := hex.EncodeToString(mac.Sum(nil))
	provided := strings.ToLower(strings.TrimSpace(signature))
	return hmac.Equal([]byte(expected), []byte(provided))
}

func hashSecret(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func randomToken(prefix string, n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	encoded := strings.TrimRight(base64.RawURLEncoding.EncodeToString(buf), "=")
	if prefix == "" {
		return encoded, nil
	}
	return fmt.Sprintf("%s_%s", prefix, encoded), nil
}
