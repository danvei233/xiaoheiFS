package usecase_test

import (
	"context"
	"testing"
	"time"

	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/usecase"
)

func TestAuthService_RegisterLogin(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	svc := usecase.NewAuthService(repo, repo)

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
	svc := usecase.NewAuthService(repo, repo)

	if _, err := svc.Login(context.Background(), "missing", "pass"); err != usecase.ErrUnauthorized {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}

func TestAuthService_RegisterCaptchaFail(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	svc := usecase.NewAuthService(repo, repo)

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
