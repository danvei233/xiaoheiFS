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
	"xiaoheiplay/internal/pkg/cryptox"
)

const (
	DefaultSignatureWindowSec = 300
)

// API key scopes. Keys are granted a subset of these; signed requests are
// rejected when the key lacks the scope required by the route (P0-7).
const (
	ScopeVPSRead      = "vps:read"
	ScopeVPSWrite     = "vps:write"
	ScopeOrdersWrite  = "orders:write"
	ScopeOrdersRefund = "orders:refund"
)

var validScopes = map[string]bool{
	ScopeVPSRead:      true,
	ScopeVPSWrite:     true,
	ScopeOrdersWrite:  true,
	ScopeOrdersRefund: true,
}

type Service struct {
	repo appports.UserAPIKeyRepository
	// ring is the versioned AES-GCM key ring used to encrypt API key signing
	// secrets at rest. The active key is derived from the operator-provided
	// source with a purpose prefix so it is independent of other keys derived
	// from the same source (e.g. the TOTP key).
	ring *cryptox.KeyRing
}

type CreateResult struct {
	Key    domain.UserAPIKey
	Secret string
}

// NewService builds a user-API-key service with a versioned encryption key
// ring. An empty or malformed key source is an error (fail-fast): the service
// must never silently fall back to a fixed key or store plaintext.
func NewService(repo appports.UserAPIKeyRepository, encryptionKey string) (*Service, error) {
	trimmed := strings.TrimSpace(encryptionKey)
	if trimmed == "" {
		return nil, domain.ErrEncryptionKeyRequired
	}
	// Domain-separated derivation: the API-key encryption key is independent of
	// the TOTP key even when both are configured from the same source secret.
	active := sha256.Sum256([]byte("userapikey:" + trimmed))
	ring, err := cryptox.NewKeyRing("v1", map[string][]byte{"v1": active[:]})
	if err != nil {
		return nil, fmt.Errorf("userapikey: build key ring: %w", err)
	}
	return &Service{repo: repo, ring: ring}, nil
}

func (s *Service) encryptSecret(v string) (string, error) {
	if s == nil || s.ring == nil || strings.TrimSpace(v) == "" {
		return "", nil
	}
	return s.ring.EncryptString(v)
}

func (s *Service) decryptSecret(v string) (string, error) {
	if s == nil || s.ring == nil || strings.TrimSpace(v) == "" {
		return "", nil
	}
	return s.ring.DecryptString(v)
}

// normalizeScopes validates the requested scope set and applies the default
// grant for empty input. The default deliberately excludes orders:refund so a
// key created without an explicit grant cannot refund/delete instances.
func normalizeScopes(scopes []string) ([]string, error) {
	seen := map[string]bool{}
	var out []string
	for _, s := range scopes {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if !validScopes[s] {
			return nil, appshared.ErrInvalidInput
		}
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		out = []string{ScopeVPSRead, ScopeVPSWrite, ScopeOrdersWrite}
	}
	return out, nil
}

// RequiredScope maps a signed OpenAPI route to the scope it requires.
func RequiredScope(method, path string) (string, bool) {
	p := strings.TrimPrefix(strings.TrimSpace(path), "/api/v1/open")
	p = strings.TrimPrefix(p, "/")
	segs := strings.Split(p, "/")
	if len(segs) == 0 || segs[0] == "" {
		return "", false
	}
	switch segs[0] {
	case "orders":
		if len(segs) >= 3 && segs[2] == "refund" {
			return ScopeOrdersRefund, true
		}
		return ScopeOrdersWrite, true
	case "vps":
		// Console/credential endpoints (panel, VNC, monitor) carry access
		// credentials (panel/VNC passwords, access_info), so they require the
		// write scope rather than the read scope. The default grant includes both
		// vps:read and vps:write, so default-created keys keep working; only a
		// key explicitly restricted to vps:read loses console access (F-6).
		if strings.ToUpper(strings.TrimSpace(method)) == "GET" && len(segs) >= 3 {
			switch segs[2] {
			case "panel", "vnc", "monitor":
				return ScopeVPSWrite, true
			}
		}
		if strings.ToUpper(strings.TrimSpace(method)) == "GET" {
			return ScopeVPSRead, true
		}
		return ScopeVPSWrite, true
	}
	return "", false
}

func (s *Service) Create(ctx context.Context, userID int64, name string, scopes []string) (CreateResult, error) {
	name = strings.TrimSpace(name)
	if userID <= 0 || name == "" {
		return CreateResult{}, appshared.ErrInvalidInput
	}
	normalized, err := normalizeScopes(scopes)
	if err != nil {
		return CreateResult{}, err
	}
	akid, err := randomToken("uak", 10)
	if err != nil {
		return CreateResult{}, err
	}
	secretRaw, err := randomToken("usk", 24)
	if err != nil {
		return CreateResult{}, err
	}
	enc, err := s.encryptSecret(secretRaw)
	if err != nil {
		return CreateResult{}, err
	}
	key := domain.UserAPIKey{
		UserID:     userID,
		Name:       name,
		AKID:       akid,
		KeyHash:    hashSecret(secretRaw),
		SecretEnc:  enc,
		Status:     domain.APIKeyStatusActive,
		ScopesJSON: strings.Join(normalized, ","),
	}
	if err := s.repo.CreateUserAPIKey(ctx, &key); err != nil {
		return CreateResult{}, err
	}
	// The raw secret is returned exactly once; only the encrypted form is kept.
	return CreateResult{Key: key, Secret: secretRaw}, nil
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

	hmacSecret, ok := s.resolveSigningSecret(ctx, &key)
	if !ok {
		return domain.UserAPIKey{}, appshared.ErrUnauthorized
	}
	if !verifySignature(hmacSecret, canonical, signature) {
		return domain.UserAPIKey{}, appshared.ErrUnauthorized
	}
	owner, err := s.repo.GetUserByID(ctx, key.UserID)
	if err != nil || owner.Status != domain.UserStatusActive {
		return domain.UserAPIKey{}, appshared.ErrForbidden
	}
	return key, nil
}

// resolveSigningSecret returns the working HMAC secret for a key. New keys
// store the encrypted secret; legacy keys (empty SecretEnc) fall back to KeyHash
// and are upgraded to encrypted-at-rest form in a one-time read-then-reencrypt
// (P0-6). A decrypt failure never falls back to the stored value.
func (s *Service) resolveSigningSecret(ctx context.Context, key *domain.UserAPIKey) (string, bool) {
	if strings.TrimSpace(key.SecretEnc) != "" {
		plain, err := s.decryptSecret(key.SecretEnc)
		if err != nil {
			return "", false
		}
		return plain, true
	}
	legacy := key.KeyHash
	if legacy == "" {
		return "", false
	}
	if s.repo != nil {
		enc, err := s.encryptSecret(legacy)
		if err == nil {
			_ = s.repo.UpdateUserAPIKeySecrets(ctx, key.ID, hashSecret(legacy), enc)
		}
	}
	return legacy, true
}

// HasScope returns whether the key's stored scope set contains scope. A blank
// scope set (legacy key or a UI-created key before scope selection existed) is
// treated as the default grant, which intentionally excludes orders:refund.
//
// A non-empty set that contains no recognized scope (e.g. a legacy key created
// before scope validation stored `all` or `read`) is also treated as the
// default grant rather than locking the key out of every signed route: the old
// server stored arbitrary caller-supplied scope strings verbatim, and failing
// closed on them would silently break every existing integration (F-10). A set
// with at least one recognized scope is filtered to those recognized entries,
// preserving the operator's explicit intent including orders:refund.
func HasScope(key domain.UserAPIKey, scope string) bool {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return true
	}
	raw := strings.Split(key.ScopesJSON, ",")
	recognized := make([]string, 0, len(raw))
	for _, s := range raw {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if validScopes[s] {
			recognized = append(recognized, s)
		}
	}
	if len(recognized) == 0 {
		// Blank scope set or all-unrecognized legacy scopes: default grant.
		switch scope {
		case ScopeVPSRead, ScopeVPSWrite, ScopeOrdersWrite:
			return true
		default:
			return false
		}
	}
	for _, s := range recognized {
		if s == scope {
			return true
		}
	}
	return false
}

// ConsumeNonce durably records a signed-request nonce and returns whether this
// request won the unique insert, plus how many requests the key issued within
// the rate window. Both operations are database-backed so replay protection
// and the request budget hold across instances and restarts (P1-10).
func (s *Service) ConsumeNonce(ctx context.Context, akid, nonce string, now time.Time, rateWindow time.Duration, rateLimit int) (claimed bool, count int, err error) {
	if s.repo == nil {
		return false, 0, appshared.ErrNotSupported
	}
	_ = s.repo.DeleteExpiredUserAPIKeyNonces(ctx)
	expiresAt := now.Add(time.Duration(DefaultSignatureWindowSec) * time.Second)
	claimed, err = s.repo.SaveUserAPIKeyNonce(ctx, akid, nonce, expiresAt)
	if err != nil || !claimed {
		return claimed, 0, err
	}
	if rateWindow > 0 {
		count, err = s.repo.CountUserAPIKeyNonces(ctx, akid, now.Add(-rateWindow))
		if err != nil {
			return claimed, count, err
		}
	}
	return true, count, nil
}

func (s *Service) RecordUse(ctx context.Context, id int64) error {
	if id <= 0 {
		return appshared.ErrInvalidInput
	}
	return s.repo.TouchUserAPIKey(ctx, id)
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
