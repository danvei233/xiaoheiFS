package apikey

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

type Service struct {
	keys appports.APIKeyRepository
}

func NewService(keys appports.APIKeyRepository) *Service {
	return &Service{keys: keys}
}

func (s *Service) Validate(ctx context.Context, raw string) (domain.APIKey, error) {
	hash := hashAPIKey(raw)
	key, err := s.keys.GetAPIKeyByHash(ctx, hash)
	if err != nil {
		return domain.APIKey{}, appshared.ErrUnauthorized
	}
	if key.Status != domain.APIKeyStatusActive {
		return domain.APIKey{}, appshared.ErrForbidden
	}
	_ = s.keys.TouchAPIKey(ctx, key.ID)
	return key, nil
}

func hashAPIKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
