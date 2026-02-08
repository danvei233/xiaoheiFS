package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/testutilhttp"
)

func TestHandlers_RegisterLoginAndProfile(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)

	captcha, code, err := env.AuthSvc.CreateCaptcha(context.Background(), time.Minute)
	if err != nil {
		t.Fatalf("captcha: %v", err)
	}
	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"username":     "alice",
		"email":        "alice@example.com",
		"password":     "pass123",
		"captcha_id":   captcha.ID,
		"captcha_code": code,
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("register code: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"username": "alice",
		"password": "pass123",
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login code: %d", rec.Code)
	}
	var loginResp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("parse login response: %v", err)
	}
	refreshToken, _ := loginResp["refresh_token"].(string)
	if refreshToken == "" {
		t.Fatalf("refresh token missing")
	}

	user := testutil.CreateUser(t, env.Repo, "bob", "bob@example.com", "pass")
	token := testutil.IssueJWT(t, env.JWTSecret, user.ID, "user", time.Hour)

	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/api/v1/me", map[string]any{
		"email": "bob2@example.com",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("update profile code: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/refresh", map[string]any{
		"refresh_token": refreshToken,
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("refresh code: %d", rec.Code)
	}
}

func TestHandlers_AuthFailures(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)

	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"username": "missing",
		"password": "pass",
	}, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/me", nil, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/forgot-password", map[string]any{
		"email": "a@example.com",
	}, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/reset-password", map[string]any{
		"token":        "t",
		"new_password": "abcdef",
	}, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
