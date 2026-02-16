package usecase_test

import (
	"context"
	"regexp"
	"testing"
	"time"

	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/usecase"
)

func TestAuthService_RegisterLogin(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	svc := usecase.NewAuthService(repo, repo, repo)

	captcha, code, err := svc.CreateCaptcha(context.Background(), time.Minute)
	if err != nil {
		t.Fatalf("create captcha: %v", err)
	}

	user, err := svc.Register(context.Background(), usecase.RegisterInput{
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
	svc := usecase.NewAuthService(repo, repo, repo)

	if _, err := svc.Login(context.Background(), "missing", "pass"); err != usecase.ErrUnauthorized {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}

func TestAuthService_RegisterCaptchaFail(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	svc := usecase.NewAuthService(repo, repo, repo)

	_, err := svc.Register(context.Background(), usecase.RegisterInput{
		Username:    "bob",
		Email:       "bob@example.com",
		Password:    "pass123",
		CaptchaID:   "missing",
		CaptchaCode: "ABCDE",
	})
	if err != usecase.ErrCaptchaFailed {
		t.Fatalf("expected captcha failed, got %v", err)
	}
}

func TestAuthService_CodePolicy(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	svc := usecase.NewAuthService(repo, repo, repo)

	_, smsCode, err := svc.CreateCaptchaWithPolicy(context.Background(), time.Minute, 6, usecase.CodeComplexityDigits)
	if err != nil {
		t.Fatalf("captcha with digits policy: %v", err)
	}
	if ok, _ := regexp.MatchString(`^[0-9]{6}$`, smsCode); !ok {
		t.Fatalf("expected 6-digit captcha code, got %q", smsCode)
	}

	emailCode, err := svc.CreateVerificationCodeWithPolicy(context.Background(), "email", "u@example.com", "register", time.Minute, 8, usecase.CodeComplexityAlnum)
	if err != nil {
		t.Fatalf("verification with alnum policy: %v", err)
	}
	if ok, _ := regexp.MatchString(`^[A-Z0-9]{8}$`, emailCode); !ok {
		t.Fatalf("expected 8-char alnum code, got %q", emailCode)
	}
}
