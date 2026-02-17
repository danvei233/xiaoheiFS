package auth_test

import (
	"context"
	"regexp"
	"testing"
	"time"

	appauth "xiaoheiplay/internal/app/auth"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/testutil"
)

func TestAuthService_RegisterLogin(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	svc := appauth.NewService(repo, repo, repo)

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
	svc := appauth.NewService(repo, repo, repo)

	if _, err := svc.Login(context.Background(), "missing", "pass"); err != appshared.ErrUnauthorized {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}

func TestAuthService_RegisterCaptchaFail(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	svc := appauth.NewService(repo, repo, repo)

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
	svc := appauth.NewService(repo, repo, repo)

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
