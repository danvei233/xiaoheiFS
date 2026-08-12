package passwordreset_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	apppasswordreset "xiaoheiplay/internal/app/passwordreset"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func TestPasswordResetService_RequestAndReset(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	email := &testutil.FakeEmailSender{}
	svc := apppasswordreset.NewService(repo, repo, email, repo)

	admin := testutil.CreateAdmin(t, repo, "admin-reset", "admin-reset@example.com", "pass", 0)
	if err := svc.RequestReset(context.Background(), admin.Email); err != nil {
		t.Fatalf("request reset: %v", err)
	}
	if len(email.Sends) != 1 {
		t.Fatalf("expected email sent")
	}
	body := email.Sends[0].Body
	re := regexp.MustCompile(`[a-f0-9]{64}`)
	token := re.FindString(body)
	if token == "" {
		t.Fatalf("token not found in email body")
	}
	if err := svc.ResetPassword(context.Background(), token, "new-pass"); err != nil {
		t.Fatalf("reset password: %v", err)
	}
	updated, err := repo.GetUserByID(context.Background(), admin.ID)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(updated.PasswordHash), []byte("new-pass")); err != nil {
		t.Fatalf("expected password updated")
	}
}

func TestPasswordResetService_RequestNonAdminIgnored(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	email := &testutil.FakeEmailSender{}
	svc := apppasswordreset.NewService(repo, repo, email, repo)

	user := testutil.CreateUser(t, repo, "user-reset", "user-reset@example.com", "pass")
	if err := svc.RequestReset(context.Background(), user.Email); err != nil {
		t.Fatalf("request reset: %v", err)
	}
	if len(email.Sends) != 0 {
		t.Fatalf("expected no email for non-admin")
	}
}

func TestPasswordResetService_ResetErrors(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	svc := apppasswordreset.NewService(repo, repo, nil, repo)

	admin := testutil.CreateAdmin(t, repo, "admin-exp", "admin-exp@example.com", "pass", 0)
	expired := domain.PasswordResetToken{
		UserID:    admin.ID,
		Token:     "expired",
		ExpiresAt: time.Now().Add(-time.Hour),
		Used:      false,
	}
	if err := repo.CreatePasswordResetToken(context.Background(), &expired); err != nil {
		t.Fatalf("create token: %v", err)
	}
	if err := svc.ResetPassword(context.Background(), "expired", "new"); err == nil {
		t.Fatalf("expected expired error")
	}
	used := domain.PasswordResetToken{
		UserID:    admin.ID,
		Token:     "used",
		ExpiresAt: time.Now().Add(time.Hour),
		Used:      true,
	}
	if err := repo.CreatePasswordResetToken(context.Background(), &used); err != nil {
		t.Fatalf("create used token: %v", err)
	}
	if err := svc.ResetPassword(context.Background(), "used", "new"); err == nil {
		t.Fatalf("expected used error")
	}
}

// TestPasswordResetService_StoresTokenHash verifies P1-2: the raw reset token
// must never be persisted; only a SHA-256 digest is stored.
func TestPasswordResetService_StoresTokenHash(t *testing.T) {
	db, repo := testutil.NewTestDB(t, false)
	email := &testutil.FakeEmailSender{}
	svc := apppasswordreset.NewService(repo, repo, email, repo)

	admin := testutil.CreateAdmin(t, repo, "admin-hash", "admin-hash@example.com", "pass", 0)
	if err := svc.RequestReset(context.Background(), admin.Email); err != nil {
		t.Fatalf("request reset: %v", err)
	}
	if len(email.Sends) != 1 {
		t.Fatalf("expected email sent")
	}
	raw := regexp.MustCompile(`[a-f0-9]{64}`).FindString(email.Sends[0].Body)
	if raw == "" {
		t.Fatalf("raw token not found in email")
	}

	var storedToken string
	if err := db.QueryRow(`SELECT token FROM password_reset_tokens LIMIT 1`).Scan(&storedToken); err != nil {
		t.Fatalf("query stored token: %v", err)
	}
	if storedToken == raw {
		t.Fatalf("raw reset token persisted to DB")
	}
	sum := sha256.Sum256([]byte(raw))
	if storedToken != hex.EncodeToString(sum[:]) {
		t.Fatalf("stored value is not the SHA-256 hash of the raw token")
	}
}

// TestPasswordResetService_TokenSingleUse verifies the token is consumed
// atomically: after a successful reset, reusing the same token fails.
func TestPasswordResetService_TokenSingleUse(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	email := &testutil.FakeEmailSender{}
	svc := apppasswordreset.NewService(repo, repo, email, repo)

	admin := testutil.CreateAdmin(t, repo, "admin-single", "admin-single@example.com", "pass", 0)
	if err := svc.RequestReset(context.Background(), admin.Email); err != nil {
		t.Fatalf("request reset: %v", err)
	}
	raw := regexp.MustCompile(`[a-f0-9]{64}`).FindString(email.Sends[0].Body)

	if err := svc.ResetPassword(context.Background(), raw, "new-pass-1"); err != nil {
		t.Fatalf("first reset: %v", err)
	}
	if err := svc.ResetPassword(context.Background(), raw, "new-pass-2"); !errors.Is(err, domain.ErrTokenUsed) {
		t.Fatalf("second reset must fail with ErrTokenUsed, got %v", err)
	}
}

// TestPasswordResetService_ResetPasswordFailureDoesNotBurnToken verifies P2-8:
// when the password update fails (owner row missing), the token is not marked
// used, so a transient failure can never consume a still-usable token.
func TestPasswordResetService_ResetPasswordFailureDoesNotBurnToken(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	svc := apppasswordreset.NewService(repo, repo, nil, repo)

	// A token whose owner row no longer exists. The atomic repo operation must
	// roll the claim back when the password update cannot be applied.
	orphan := domain.PasswordResetToken{
		UserID:    999999,
		Token:     "orphan-hash",
		ExpiresAt: time.Now().Add(time.Hour),
		Used:      false,
	}
	if err := repo.CreatePasswordResetToken(context.Background(), &orphan); err != nil {
		t.Fatalf("create token: %v", err)
	}
	if err := svc.ResetPassword(context.Background(), "orphan-hash", "new-pass"); err == nil {
		t.Fatalf("expected reset to fail for missing owner")
	}
	reloaded, err := repo.GetPasswordResetToken(context.Background(), "orphan-hash")
	if err != nil {
		t.Fatalf("reload token: %v", err)
	}
	if reloaded.Used {
		t.Fatalf("token must remain unused after a failed password update")
	}
}
