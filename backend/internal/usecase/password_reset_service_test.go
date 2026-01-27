package usecase_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/usecase"
)

func TestPasswordResetService_RequestAndReset(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	email := &testutil.FakeEmailSender{}
	svc := usecase.NewPasswordResetService(repo, repo, email, repo)

	admin := testutil.CreateAdmin(t, repo, "admin-reset", "admin-reset@example.com", "pass", 0)
	if err := svc.RequestReset(context.Background(), admin.Email); err != nil {
		t.Fatalf("request reset: %v", err)
	}
	if len(email.Sends) != 1 {
		t.Fatalf("expected email sent")
	}
	body := email.Sends[0].Body
	colon := strings.LastIndex(body, ":")
	if colon == -1 {
		t.Fatalf("unexpected email body: %q", body)
	}
	token := strings.TrimSpace(body[colon+1:])
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
	svc := usecase.NewPasswordResetService(repo, repo, email, repo)

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
	svc := usecase.NewPasswordResetService(repo, repo, nil, repo)

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
