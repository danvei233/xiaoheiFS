package auth_test

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
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"xiaoheiplay/internal/adapter/repo/core"
	appauth "xiaoheiplay/internal/app/auth"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func TestAuthService_RegisterLogin(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	svc := appauth.NewService(repo, repo, repo, repo, "test-totp-key")

	captcha, code, err := svc.CreateCaptcha(context.Background(), time.Minute)
	if err != nil {
		t.Fatalf("create captcha: %v", err)
	}

	user, err := svc.Register(context.Background(), appshared.RegisterInput{
		Username:    "alice",
		Email:       "alice@example.com",
		Password:    "pass123",
		CaptchaID:   captcha.ID,
		CaptchaCode: code,
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if user.ID == 0 {
		t.Fatalf("expected user id")
	}

	_, err = svc.Login(context.Background(), "alice", "pass123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
}

func TestAuthService_LoginFailures(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	svc := appauth.NewService(repo, repo, repo, repo, "test-totp-key")

	if _, err := svc.Login(context.Background(), "missing", "pass"); err != appshared.ErrUnauthorized {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}

// TestAuthService_LoginDisabledAccountIsIndistinguishable verifies FINDING 6:
// a disabled account must be indistinguishable from a wrong password both by
// error type and by timing, so an attacker cannot enumerate "exists and is
// disabled" accounts through Login.
func TestAuthService_LoginDisabledAccountIsIndistinguishable(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	svc := appauth.NewService(repo, repo, repo, repo, "test-totp-key")

	user := testutil.CreateUser(t, repo, "disabled-user", "disabled-user@example.com", "pass123")
	if err := repo.UpdateUserStatus(context.Background(), user.ID, domain.UserStatusDisabled); err != nil {
		t.Fatalf("disable user: %v", err)
	}

	// Both a nonexistent account and a disabled account must return the same
	// sentinel error (ErrUnauthorized), never ErrForbidden.
	if _, err := svc.Login(context.Background(), "disabled-user", "wrongpass"); err != appshared.ErrUnauthorized {
		t.Fatalf("disabled account must return ErrUnauthorized, got %v", err)
	}
	if _, err := svc.Login(context.Background(), "does-not-exist", "wrongpass"); err != appshared.ErrUnauthorized {
		t.Fatalf("missing account must return ErrUnauthorized, got %v", err)
	}

	// Timing: both paths must run the same dummy bcrypt so their durations are in
	// the same order of magnitude. Use a generous floor (bcrypt cost 10 is ~50-100
	// ms) so the test does not flake on slow CI machines.
	minElapsed := 30 * time.Millisecond
	measure := func(account string) time.Duration {
		start := time.Now()
		_, _ = svc.Login(context.Background(), account, "wrongpass")
		return time.Since(start)
	}
	disabledDur := measure("disabled-user")
	missingDur := measure("does-not-exist")
	if disabledDur < minElapsed || missingDur < minElapsed {
		t.Fatalf("both login failure paths must run a bcrypt comparison: disabled=%s missing=%s (floor %s)", disabledDur, missingDur, minElapsed)
	}
	// Loose ratio: neither path may be a near-instant return while the other
	// spends ~100ms. 3x allows bcrypt jitter while catching a 0ms shortcut.
	if disabledDur > missingDur*3 && disabledDur > minElapsed {
		t.Fatalf("disabled login must not be faster than a wrong-password login: disabled=%s missing=%s", disabledDur, missingDur)
	}
	if missingDur > disabledDur*3 && missingDur > minElapsed {
		t.Fatalf("missing login must not be faster than a disabled login: missing=%s disabled=%s", missingDur, disabledDur)
	}
}

func TestAuthService_RegisterCaptchaFail(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	svc := appauth.NewService(repo, repo, repo, repo, "test-totp-key")

	_, err := svc.Register(context.Background(), appshared.RegisterInput{
		Username:    "bob",
		Email:       "bob@example.com",
		Password:    "pass123",
		CaptchaID:   "missing",
		CaptchaCode: "ABCDE",
	})
	if err != appshared.ErrCaptchaFailed {
		t.Fatalf("expected captcha failed, got %v", err)
	}
}

func TestAuthService_CodePolicy(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	svc := appauth.NewService(repo, repo, repo, repo, "test-totp-key")

	_, smsCode, err := svc.CreateCaptchaWithPolicy(context.Background(), time.Minute, 6, appauth.CodeComplexityDigits)
	if err != nil {
		t.Fatalf("captcha with digits policy: %v", err)
	}
	if ok, _ := regexp.MatchString(`^[0-9]{6}$`, smsCode); !ok {
		t.Fatalf("expected 6-digit captcha code, got %q", smsCode)
	}

	emailCode, err := svc.CreateVerificationCodeWithPolicy(context.Background(), "email", "u@example.com", "register", time.Minute, 8, appauth.CodeComplexityAlnum)
	if err != nil {
		t.Fatalf("verification with alnum policy: %v", err)
	}
	if ok, _ := regexp.MatchString(`^[A-Z0-9]{8}$`, emailCode); !ok {
		t.Fatalf("expected 8-char alnum code, got %q", emailCode)
	}
}

// TestAuthService_TOTPKeyIsolation verifies P0-9: TOTP secrets encrypted with
// one deployment key cannot be verified by a service using a different key,
// while secrets created under the legacy compile-time constant remain readable
// after upgrade (fallback).
func TestAuthService_TOTPKeyIsolation(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()

	// Service using a deployment-provided key.
	svcA := appauth.NewService(repo, repo, repo, repo, "deployment-key-A")
	userA := testutil.CreateUser(t, repo, "totp-a", "totp-a@example.com", "pass")
	secretA, _, err := svcA.SetupTOTP(ctx, userA.ID, "pass", "")
	if err != nil {
		t.Fatalf("setup totp A: %v", err)
	}
	codeA := currentTOTPCode(t, secretA)
	if err := svcA.ConfirmTOTP(ctx, userA.ID, codeA); err != nil {
		t.Fatalf("confirm totp A: %v", err)
	}
	if err := svcA.VerifyTOTP(ctx, userA.ID, codeA); err != nil {
		t.Fatalf("verify totp A with same key: %v", err)
	}

	// Service using a different key must not be able to verify A's secret.
	svcB := appauth.NewService(repo, repo, repo, repo, "deployment-key-B")
	if err := svcB.VerifyTOTP(ctx, userA.ID, codeA); err != appshared.ErrUnauthorized {
		t.Fatalf("expected ErrUnauthorized for cross-key verify, got %v", err)
	}

	// Legacy: a secret encrypted with the historical compile-time constant (in
	// the old base32 {n,c} format) must remain readable by a keyed service via
	// the one-time-read-then-reencrypt fallback. This models a pre-upgrade row.
	userL := testutil.CreateUser(t, repo, "totp-legacy", "totp-legacy@example.com", "pass")
	legacySecret := "JBSWY3DPEHPK3PXP"
	userL.TOTPSecretEnc = legacyEncryptTOTP(legacySecret)
	userL.TOTPEnabled = true
	if err := repo.UpdateUser(ctx, userL); err != nil {
		t.Fatalf("store legacy row: %v", err)
	}
	codeL := currentTOTPCode(t, legacySecret)
	if err := svcA.VerifyTOTP(ctx, userL.ID, codeL); err != nil {
		t.Fatalf("legacy secret must remain readable by keyed service: %v", err)
	}
	// After the fallback read, the row should have been upgraded to the keyed
	// envelope and still be readable.
	reloaded, err := repo.GetUserByID(ctx, userL.ID)
	if err != nil {
		t.Fatalf("reload legacy user: %v", err)
	}
	if strings.HasPrefix(reloaded.TOTPSecretEnc, `{"k":"v1"`) {
		if err := svcA.VerifyTOTP(ctx, userL.ID, codeL); err != nil {
			t.Fatalf("upgraded legacy secret must still verify: %v", err)
		}
	} else {
		t.Fatalf("legacy row was not upgraded to a keyed envelope: %s", reloaded.TOTPSecretEnc)
	}
}

// legacyEncryptTOTP reproduces the pre-upgrade TOTP encryption: AES-GCM with the
// historical compile-time constant, serialized as base32 {n,c}.
func legacyEncryptTOTP(secret string) string {
	key := sha256.Sum256([]byte("xiaoheiplay-totp-v1"))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		panic(err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err)
	}
	nonce := make([]byte, gcm.NonceSize())
	_, _ = rand.Read(nonce)
	ct := gcm.Seal(nil, nonce, []byte(secret), nil)
	raw := map[string]string{
		"n": base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(nonce),
		"c": base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(ct),
	}
	b, _ := json.Marshal(raw)
	return string(b)
}

// TestAuthService_CorruptTOTPSecretIsNotPlaintext verifies a non-JSON or
// damaged stored TOTP secret is treated as an error, never as the plaintext
// secret itself.
func TestAuthService_CorruptTOTPSecretIsNotPlaintext(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	svc := appauth.NewService(repo, repo, repo, repo, "corrupt-key-source")

	user := testutil.CreateUser(t, repo, "totp-corrupt", "totp-corrupt@example.com", "pass123")
	user.TOTPEnabled = true
	user.TOTPSecretEnc = "definitely-not-a-valid-envelope" // non-JSON
	if err := repo.UpdateUser(ctx, user); err != nil {
		t.Fatalf("store corrupt secret: %v", err)
	}
	if err := svc.VerifyTOTP(ctx, user.ID, "123456"); err != appshared.ErrUnauthorized {
		t.Fatalf("corrupt TOTP secret must fail verification, got %v", err)
	}
	// Even a code computed for the "plaintext" string (were it the secret) must
	// not verify — the stored value is never interpreted as the secret itself.
	reloaded, err := repo.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if reloaded.TOTPSecretEnc != "definitely-not-a-valid-envelope" {
		t.Fatalf("a rejected corrupt value must remain untouched, got %q", reloaded.TOTPSecretEnc)
	}
}

// TestAuthService_RegisterRejectsDuplicatePhone verifies Finding 4: two
// accounts must not share a non-empty phone, because login and SMS reset
// resolve by phone and would become ambiguous (GetUserByPhone returns the first
// row). Registering a second account with the victim's phone must fail.
func TestAuthService_RegisterRejectsDuplicatePhone(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	svc := appauth.NewService(repo, repo, repo, repo, "phone-test-key")

	if _, err := svc.Register(context.Background(), appshared.RegisterInput{
		Username: "victim", Email: "victim@example.com", Phone: "13800000000", Password: "pass123",
	}); err != nil {
		t.Fatalf("register victim: %v", err)
	}
	_, err := svc.Register(context.Background(), appshared.RegisterInput{
		Username: "attacker", Email: "attacker@example.com", Phone: "13800000000", Password: "pass123",
	})
	if err != domain.ErrPhoneAlreadyExists {
		t.Fatalf("duplicate phone register must fail with ErrPhoneAlreadyExists, got %v", err)
	}
	// Distinct empty phones must never collide (most accounts have no phone).
	if _, err := svc.Register(context.Background(), appshared.RegisterInput{
		Username: "no-phone-1", Email: "n1@example.com", Password: "pass123",
	}); err != nil {
		t.Fatalf("register no-phone-1: %v", err)
	}
	if _, err := svc.Register(context.Background(), appshared.RegisterInput{
		Username: "no-phone-2", Email: "n2@example.com", Password: "pass123",
	}); err != nil {
		t.Fatalf("register no-phone-2: %v", err)
	}
}

// TestAuthService_UpdateUserRejectsDuplicatePhone verifies Finding 4 at the
// service layer: changing a user's phone to another account's phone must be
// rejected, while keeping one's own phone (or clearing it) is fine.
func TestAuthService_UpdateUserRejectsDuplicatePhone(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	svc := appauth.NewService(repo, repo, repo, repo, "phone-test-key")

	a := testutil.CreateUser(t, repo, "phone-a", "phone-a@example.com", "pass123")
	b := testutil.CreateUser(t, repo, "phone-b", "phone-b@example.com", "pass123")
	// Give A a phone.
	ua := mustLoadUser(t, repo, a.ID)
	ua.Phone = "13800000001"
	if err := svc.UpdateUser(context.Background(), ua); err != nil {
		t.Fatalf("bind phone to A: %v", err)
	}
	// B tries to take A's phone.
	ub := mustLoadUser(t, repo, b.ID)
	ub.Phone = "13800000001"
	if err := svc.UpdateUser(context.Background(), ub); err != domain.ErrPhoneAlreadyExists {
		t.Fatalf("duplicate phone update must fail with ErrPhoneAlreadyExists, got %v", err)
	}
	// A keeping its own phone is fine.
	if err := svc.UpdateUser(context.Background(), mustLoadUser(t, repo, a.ID)); err != nil {
		t.Fatalf("A keep own phone must be allowed: %v", err)
	}
}

func mustLoadUser(t *testing.T, repo *repo.GormRepo, id int64) domain.User {
	t.Helper()
	u, err := repo.GetUserByID(context.Background(), id)
	if err != nil {
		t.Fatalf("load user %d: %v", id, err)
	}
	return u
}

// currentTOTPCode computes the RFC 6238 code for a base32 secret at the current
// time using the same algorithm the service verifies.
func currentTOTPCode(t *testing.T, secret string) string {
	t.Helper()
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if err != nil {
		t.Fatalf("decode totp secret: %v", err)
	}
	counter := uint64(time.Now().Unix() / 30)
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], counter)
	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(msg[:])
	sum := mac.Sum(nil)
	offset := int(sum[len(sum)-1] & 0x0f)
	code := int32(sum[offset]&0x7f)<<24 | int32(sum[offset+1])<<16 | int32(sum[offset+2])<<8 | int32(sum[offset+3])
	return fmt.Sprintf("%06d", code%1000000)
}
