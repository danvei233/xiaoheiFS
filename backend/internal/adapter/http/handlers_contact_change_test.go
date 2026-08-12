package http_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"testing"
	"time"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/testutilhttp"
)

// contactTicketHash mirrors the securityticket service's persistence: the raw
// token is hashed before storage, so a test that seeds a ticket via the repo
// must store the digest.
func contactTicketHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// seedContactTicket writes a security ticket directly and returns its raw
// token for use in handler payloads.
func seedContactTicket(t *testing.T, env *testutilhttp.Env, userID int64, used bool) string {
	t.Helper()
	raw := "contact-ticket-" + testutil.Itoa(userID) + time.Now().Format("150405.000000000")
	ticket := domain.PasswordResetTicket{
		UserID:    userID,
		Channel:   "contact_bind_email",
		Receiver:  "-",
		Token:     contactTicketHash(raw),
		ExpiresAt: time.Now().Add(20 * time.Minute),
		Used:      used,
	}
	if err := env.Repo.CreatePasswordResetTicket(context.Background(), &ticket); err != nil {
		t.Fatalf("create contact ticket: %v", err)
	}
	return raw
}

// seedContactVerificationCode stores a random code that meSecurityContactConfirm
// will verify for a new email value and returns it for the request payload.
func seedContactVerificationCode(t *testing.T, env *testutilhttp.Env, value string) string {
	t.Helper()
	code, err := env.AuthSvc.CreateVerificationCodeWithPolicy(context.Background(), "email", value, "bind_email", 10*time.Minute, 6, "digits")
	if err != nil {
		t.Fatalf("create verification code: %v", err)
	}
	return code
}

// TestHandlers_ContactChange_UsedTicketCannotChangeContact verifies P2-8: a
// security ticket that is already consumed (or otherwise invalid) can never
// complete a contact change — the email stays untouched and no partial update
// is committed.
func TestHandlers_ContactChange_UsedTicketCannotChangeContact(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	ctx := context.Background()
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_2fa_enabled", "true"); err != nil {
		t.Fatalf("enable 2fa: %v", err)
	}
	user := testutil.CreateUser(t, env.Repo, "contact-used", "contact-used@example.com", "pass123")
	user.TOTPEnabled = true
	user.TOTPSecretEnc = encryptTOTPSecretForTest(t, "JBSWY3DPEHPK3PXP")
	if err := env.Repo.UpdateUser(ctx, user); err != nil {
		t.Fatalf("enable user 2fa: %v", err)
	}
	token := testutil.IssueJWT(t, env.JWTSecret, user.ID, "user", time.Hour)

	rawTicket := seedContactTicket(t, env, user.ID, true) // already used
	code := seedContactVerificationCode(t, env, "hijack@example.com")

	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/me/security/email/confirm", map[string]any{
		"value":           "hijack@example.com",
		"code":            code,
		"security_ticket": rawTicket,
	}, token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("used ticket must be rejected, got %d: %s", rec.Code, rec.Body.String())
	}
	reloaded, err := env.Repo.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if reloaded.Email != "contact-used@example.com" {
		t.Fatalf("email must be unchanged after a rejected change, got %q", reloaded.Email)
	}
}

// TestHandlers_ContactChange_SuccessConsumesTicket verifies the atomic happy
// path: a valid ticket + code changes the email, marks the ticket used, and a
// second use of the same ticket is rejected.
func TestHandlers_ContactChange_SuccessConsumesTicket(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	ctx := context.Background()
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_2fa_enabled", "true"); err != nil {
		t.Fatalf("enable 2fa: %v", err)
	}
	user := testutil.CreateUser(t, env.Repo, "contact-ok", "contact-ok@example.com", "pass123")
	user.TOTPEnabled = true
	user.TOTPSecretEnc = encryptTOTPSecretForTest(t, "JBSWY3DPEHPK3PXP")
	if err := env.Repo.UpdateUser(ctx, user); err != nil {
		t.Fatalf("enable user 2fa: %v", err)
	}
	token := testutil.IssueJWT(t, env.JWTSecret, user.ID, "user", time.Hour)

	rawTicket := seedContactTicket(t, env, user.ID, false)
	code := seedContactVerificationCode(t, env, "new-contact@example.com")

	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/me/security/email/confirm", map[string]any{
		"value":           "new-contact@example.com",
		"code":            code,
		"security_ticket": rawTicket,
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("valid contact change should succeed, got %d: %s", rec.Code, rec.Body.String())
	}
	reloaded, err := env.Repo.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if reloaded.Email != "new-contact@example.com" {
		t.Fatalf("email must be updated, got %q", reloaded.Email)
	}

	// The same ticket must not be usable a second time.
	code2 := seedContactVerificationCode(t, env, "another@example.com")
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/me/security/email/confirm", map[string]any{
		"value":           "another@example.com",
		"code":            code2,
		"security_ticket": rawTicket,
	}, token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("consumed ticket must be rejected on reuse, got %d: %s", rec.Code, rec.Body.String())
	}
}
