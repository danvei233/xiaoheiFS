package userapikey_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	appshared "xiaoheiplay/internal/app/shared"
	appuserapikey "xiaoheiplay/internal/app/userapikey"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func TestService_CreateSecretNotStoredInPlaintext(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	svc, _ := appuserapikey.NewService(repo, "test-apikey-encryption-key")

	user := testutil.CreateUser(t, repo, "apikey-plaintext", "apikey-plaintext@example.com", "pass")
	res, err := svc.Create(ctx, user.ID, "ci", []string{appuserapikey.ScopeVPSRead})
	if err != nil {
		t.Fatalf("create key: %v", err)
	}
	secret := res.Secret
	if secret == "" {
		t.Fatal("raw secret must be returned once at creation")
	}

	stored, err := repo.GetUserAPIKeyByAKID(ctx, res.Key.AKID)
	if err != nil {
		t.Fatalf("load key: %v", err)
	}
	if stored.KeyHash == secret {
		t.Fatalf("DB stored the raw secret as the hash")
	}
	if stored.SecretEnc == "" {
		t.Fatalf("expected an encrypted secret field for a new key")
	}
	if stored.SecretEnc == secret {
		t.Fatalf("DB stored the raw secret in plaintext")
	}

	// The raw secret must sign a request successfully (round trip).
	ts := time.Now().UTC().Format(time.RFC3339)
	nonce := "create-roundtrip-nonce"
	canonical := appuserapikey.BuildCanonical("POST", "/api/v1/open/vps/1/reboot", "", ts, nonce, []byte(`{}`))
	key, err := svc.ValidateSignature(ctx, res.Key.AKID, ts, nonce, signHMAC(secret, canonical), canonical)
	if err != nil {
		t.Fatalf("signature with returned secret must verify: %v", err)
	}
	if key.ID != res.Key.ID {
		t.Fatalf("unexpected key returned")
	}
}

func TestService_RequiredScope(t *testing.T) {
	cases := []struct {
		method, path string
		want         string
		ok           bool
	}{
		{"GET", "/api/v1/open/vps", appuserapikey.ScopeVPSRead, true},
		{"GET", "/api/v1/open/vps/1", appuserapikey.ScopeVPSRead, true},
		{"GET", "/api/v1/open/vps/1/snapshots", appuserapikey.ScopeVPSRead, true},
		{"POST", "/api/v1/open/vps/1/reboot", appuserapikey.ScopeVPSWrite, true},
		{"DELETE", "/api/v1/open/vps/1/snapshots/s-1", appuserapikey.ScopeVPSWrite, true},
		{"POST", "/api/v1/open/orders/instant/create", appuserapikey.ScopeOrdersWrite, true},
		{"POST", "/api/v1/open/orders/instant/renew", appuserapikey.ScopeOrdersWrite, true},
		{"POST", "/api/v1/open/orders/instant/refund", appuserapikey.ScopeOrdersRefund, true},
		{"POST", "/api/v1/open/orders/instant/resize", appuserapikey.ScopeOrdersWrite, true},
		{"GET", "/api/v1/other", "", false},
	}
	for _, c := range cases {
		got, ok := appuserapikey.RequiredScope(c.method, c.path)
		if ok != c.ok || (ok && got != c.want) {
			t.Fatalf("RequiredScope(%s %s) = (%q,%v), want (%q,%v)", c.method, c.path, got, ok, c.want, c.ok)
		}
	}
}

func TestService_HasScopeDefaultExcludesRefund(t *testing.T) {
	key := domain.UserAPIKey{ScopesJSON: ""}
	if !appuserapikey.HasScope(key, appuserapikey.ScopeVPSRead) {
		t.Fatal("blank scope set must grant default read scope")
	}
	if appuserapikey.HasScope(key, appuserapikey.ScopeOrdersRefund) {
		t.Fatal("blank scope set must NOT grant refund scope")
	}
	restricted := domain.UserAPIKey{ScopesJSON: appuserapikey.ScopeVPSRead}
	if !appuserapikey.HasScope(restricted, appuserapikey.ScopeVPSRead) {
		t.Fatal("explicit scope must be honored")
	}
	if appuserapikey.HasScope(restricted, appuserapikey.ScopeVPSWrite) {
		t.Fatal("explicit scope must not grant unrelated scopes")
	}
}

// TestService_HasScopeLegacyUnrecognizedFallsBackToDefault verifies F-10: a
// legacy key whose stored scope set is non-empty but contains no recognized
// scope must fall back to the default grant (read/write/orders) instead of being
// locked out of every signed route. Explicit recognized scopes are still honored
// strictly, including the refund exclusion.
func TestService_HasScopeLegacyUnrecognizedFallsBackToDefault(t *testing.T) {
	legacy := domain.UserAPIKey{ScopesJSON: "all"}
	for _, s := range []string{appuserapikey.ScopeVPSRead, appuserapikey.ScopeVPSWrite, appuserapikey.ScopeOrdersWrite} {
		if !appuserapikey.HasScope(legacy, s) {
			t.Fatalf("legacy unrecognized scope %q must fall back to default grant for %s", "all", s)
		}
	}
	if appuserapikey.HasScope(legacy, appuserapikey.ScopeOrdersRefund) {
		t.Fatal("legacy unrecognized scope must NOT grant refund scope")
	}

	// A mix of a recognized scope and an unknown legacy value keeps the
	// recognized entry and ignores the unknown one.
	mixed := domain.UserAPIKey{ScopesJSON: appuserapikey.ScopeVPSRead + ",legacy-junk"}
	if !appuserapikey.HasScope(mixed, appuserapikey.ScopeVPSRead) {
		t.Fatal("recognized scope in mixed legacy set must be honored")
	}
	if appuserapikey.HasScope(mixed, appuserapikey.ScopeVPSWrite) {
		t.Fatal("unknown scope in mixed set must not widen access")
	}
}

func TestService_CreateRejectsUnknownScope(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	svc, _ := appuserapikey.NewService(repo, "test-apikey-encryption-key")
	user := testutil.CreateUser(t, repo, "apikey-badscope", "apikey-badscope@example.com", "pass")
	if _, err := svc.Create(context.Background(), user.ID, "ci", []string{"totally:unknown"}); err != appshared.ErrInvalidInput {
		t.Fatalf("expected ErrInvalidInput for unknown scope, got %v", err)
	}
}

// TestService_RequiredScopeConsoleRoutesRequireWrite verifies F-6: the VPS
// console/credential routes (panel/vnc/monitor) must require the write scope,
// not the read scope, so a vps:read-only key cannot fetch console credentials.
func TestService_RequiredScopeConsoleRoutesRequireWrite(t *testing.T) {
	for _, path := range []string{
		"/api/v1/open/vps/123/panel",
		"/api/v1/open/vps/123/vnc",
		"/api/v1/open/vps/123/monitor",
	} {
		got, ok := appuserapikey.RequiredScope("GET", path)
		if !ok || got != appuserapikey.ScopeVPSWrite {
			t.Fatalf("console route %s must require %s, got %q ok=%v", path, appuserapikey.ScopeVPSWrite, got, ok)
		}
	}
	// Plain read routes (detail, list) still use the read scope.
	read, ok := appuserapikey.RequiredScope("GET", "/api/v1/open/vps/123")
	if !ok || read != appuserapikey.ScopeVPSRead {
		t.Fatalf("vps detail must require read scope, got %q ok=%v", read, ok)
	}
	// Non-GET stays write.
	write, ok := appuserapikey.RequiredScope("POST", "/api/v1/open/vps/123/start")
	if !ok || write != appuserapikey.ScopeVPSWrite {
		t.Fatalf("vps action must require write scope, got %q ok=%v", write, ok)
	}
}

// TestService_EncryptionKeyRequired verifies the service cannot be constructed
// without a non-empty encryption key source (fail-fast, no SHA256("") fallback).
func TestService_EncryptionKeyRequired(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	if _, err := appuserapikey.NewService(repo, "  "); err == nil {
		t.Fatalf("empty encryption key must be rejected")
	}
	if _, err := appuserapikey.NewService(repo, "  \t "); err == nil {
		t.Fatalf("whitespace-only encryption key must be rejected")
	}
}

// TestService_CrossKeyCannotDecrypt verifies the encryption key is
// domain-separated and versioned: a key created under one service instance
// cannot be authenticated by another instance built with a different key.
func TestService_CrossKeyCannotDecrypt(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	svcA, _ := appuserapikey.NewService(repo, "key-A")
	svcB, _ := appuserapikey.NewService(repo, "key-B")

	user := testutil.CreateUser(t, repo, "apikey-cross", "apikey-cross@example.com", "pass")
	res, err := svcA.Create(ctx, user.ID, "ci", nil)
	if err != nil {
		t.Fatalf("create key: %v", err)
	}
	ts := time.Now().UTC().Format(time.RFC3339)
	nonce := "cross-key-nonce"
	canonical := appuserapikey.BuildCanonical("POST", "/api/v1/open/vps/1/reboot", "", ts, nonce, []byte(`{}`))

	if _, err := svcA.ValidateSignature(ctx, res.Key.AKID, ts, nonce, signHMAC(res.Secret, canonical), canonical); err != nil {
		t.Fatalf("same-key signature must verify: %v", err)
	}
	if _, err := svcB.ValidateSignature(ctx, res.Key.AKID, ts, nonce, signHMAC(res.Secret, canonical), canonical); err != appshared.ErrUnauthorized {
		t.Fatalf("cross-key signature must fail with ErrUnauthorized, got %v", err)
	}
}

// TestService_LegacyKeyUpgrade verifies P0-6: a legacy row (SecretEnc empty,
// KeyHash as the working secret) still authenticates and is upgraded to the
// encrypted-at-rest form on first use.
func TestService_LegacyKeyUpgrade(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	svc, _ := appuserapikey.NewService(repo, "test-apikey-encryption-key")

	user := testutil.CreateUser(t, repo, "apikey-legacy", "apikey-legacy@example.com", "pass")
	legacySecret := "legacy-working-secret"
	legacyKey := domain.UserAPIKey{
		UserID:  user.ID,
		Name:    "legacy",
		AKID:    "akid-legacy",
		KeyHash: legacySecret, // legacy: the KeyHash itself is the working secret
		Status:  domain.APIKeyStatusActive,
	}
	if err := repo.CreateUserAPIKey(ctx, &legacyKey); err != nil {
		t.Fatalf("create legacy key: %v", err)
	}
	ts := time.Now().UTC().Format(time.RFC3339)
	nonce := "legacy-nonce"
	canonical := appuserapikey.BuildCanonical("POST", "/api/v1/open/vps/1/reboot", "", ts, nonce, []byte(`{}`))

	if _, err := svc.ValidateSignature(ctx, legacyKey.AKID, ts, nonce, signHMAC(legacySecret, canonical), canonical); err != nil {
		t.Fatalf("legacy key must authenticate: %v", err)
	}
	reloaded, err := repo.GetUserAPIKeyByAKID(ctx, legacyKey.AKID)
	if err != nil {
		t.Fatalf("reload legacy key: %v", err)
	}
	if reloaded.SecretEnc == "" {
		t.Fatalf("legacy key must be upgraded to an encrypted secret after first use")
	}
	if reloaded.KeyHash == legacySecret {
		t.Fatalf("upgraded key must no longer keep the raw secret as the hash")
	}
}

func TestService_NonceIsDurableAcrossInstances(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	svcA, _ := appuserapikey.NewService(repo, "key-A")
	svcB, _ := appuserapikey.NewService(repo, "key-B") // different encryption key, same DB

	user := testutil.CreateUser(t, repo, "apikey-nonce", "apikey-nonce@example.com", "pass")
	res, err := svcA.Create(ctx, user.ID, "ci", nil)
	if err != nil {
		t.Fatalf("create key: %v", err)
	}

	now := time.Now()
	claimed, count, err := svcA.ConsumeNonce(ctx, res.Key.AKID, "shared-nonce", now, time.Minute, 60)
	if err != nil || !claimed {
		t.Fatalf("first consume: claimed=%v err=%v", claimed, err)
	}
	// A different service instance talking to the same DB must see the nonce
	// as already consumed (multi-instance replay protection).
	claimed2, _, err := svcB.ConsumeNonce(ctx, res.Key.AKID, "shared-nonce", now.Add(time.Second), time.Minute, 60)
	if err != nil {
		t.Fatalf("second consume: %v", err)
	}
	if claimed2 {
		t.Fatalf("nonce replayed across instances was accepted")
	}
	if count < 1 {
		t.Fatalf("expected request count >= 1, got %d", count)
	}
}

func signHMAC(secret, canonical string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(canonical))
	return hex.EncodeToString(mac.Sum(nil))
}
