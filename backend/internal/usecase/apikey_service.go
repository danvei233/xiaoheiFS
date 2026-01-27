package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"xiaoheiplay/internal/domain"
)

type APIKeyService struct {
	keys APIKeyRepository
}

func NewAPIKeyService(keys APIKeyRepository) *APIKeyService {
	return &APIKeyService{keys: keys}
}

func (s *APIKeyService) Validate(ctx context.Context, raw string) (domain.APIKey, error) {
	hash := hashAPIKey(raw)
	key, err := s.keys.GetAPIKeyByHash(ctx, hash)
	if err != nil {
		return domain.APIKey{}, ErrUnauthorized
	}
	if key.Status != domain.APIKeyStatusActive {
		return domain.APIKey{}, ErrForbidden
	}
	_ = s.keys.TouchAPIKey(ctx, key.ID)
	return key, nil
}

func hashAPIKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
