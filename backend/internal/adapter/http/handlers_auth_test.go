package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/testutilhttp"
)

func doJSONWithIP(t *testing.T, router http.Handler, method, path string, body any, token, ip string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if ip != "" {
		req.RemoteAddr = ip + ":12345"
		req.Header.Set("X-Forwarded-For", ip)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func parseJSONBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	out := map[string]any{}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("parse response body: %v body=%s", err, rec.Body.String())
	}
	return out
}

func TestHandlers_RegisterLoginAndProfile(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	ctx := context.Background()
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_register_verify_type", "none"); err != nil {
		t.Fatalf("set register verify type: %v", err)
	}
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_register_verify_channels", `[]`); err != nil {
		t.Fatalf("set register verify channels: %v", err)
	}

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
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "/api/v1/me/security/*") {
		t.Fatalf("update profile email should be rejected, got %d: %s", rec.Code, rec.Body.String())
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

func TestHandlers_Me_HidesEmailAndPhone(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	user := testutil.CreateUser(t, env.Repo, "me-hide-contact", "me-hide-contact@example.com", "pass123")
	user.Phone = "13900005555"
	if err := env.Repo.UpdateUser(context.Background(), user); err != nil {
		t.Fatalf("set phone: %v", err)
	}
	token := testutil.IssueJWT(t, env.JWTSecret, user.ID, "user", time.Hour)

	rec := testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/me", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("me expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	body := parseJSONBody(t, rec)
	if v, _ := body["email"].(string); strings.TrimSpace(v) != "" {
		t.Fatalf("me should hide email, got %q", v)
	}
	if v, _ := body["phone"].(string); strings.TrimSpace(v) != "" {
		t.Fatalf("me should hide phone, got %q", v)
	}
}

func TestHandlers_Register_SMSChannel_DoesNotRequireEmail(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	ctx := context.Background()

	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_register_verify_channels", `["sms"]`); err != nil {
		t.Fatalf("set verify channels: %v", err)
	}
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_register_email_required", "true"); err != nil {
		t.Fatalf("set email required: %v", err)
	}

	captcha, captchaCode, err := env.AuthSvc.CreateCaptcha(ctx, time.Minute)
	if err != nil {
		t.Fatalf("captcha: %v", err)
	}
	verifyCode, err := env.AuthSvc.CreateVerificationCode(ctx, "sms", "13900000001", "register", time.Minute)
	if err != nil {
		t.Fatalf("verification code: %v", err)
	}

	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"username":       "smsuser1",
		"phone":          "13900000001",
		"password":       "pass123",
		"verify_channel": "sms",
		"verify_code":    verifyCode,
		"captcha_id":     captcha.ID,
		"captcha_code":   captchaCode,
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("register by sms expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlers_Login_ByPhone(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	ctx := context.Background()

	user := testutil.CreateUser(t, env.Repo, "phone-login-u", "phone-login-u@example.com", "pass123")
	user.Phone = "13900000002"
	if err := env.Repo.UpdateUser(ctx, user); err != nil {
		t.Fatalf("update user phone: %v", err)
	}

	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"username": "13900000002",
		"password": "pass123",
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("phone login expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlers_Register_EmailChannel_DoesNotBindPhone(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	ctx := context.Background()

	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_register_verify_channels", `["email","sms"]`); err != nil {
		t.Fatalf("set verify channels: %v", err)
	}
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_register_email_required", "false"); err != nil {
		t.Fatalf("set email required: %v", err)
	}

	captcha, captchaCode, err := env.AuthSvc.CreateCaptcha(ctx, time.Minute)
	if err != nil {
		t.Fatalf("captcha: %v", err)
	}
	verifyCode, err := env.AuthSvc.CreateVerificationCode(ctx, "email", "email-tab-user@example.com", "register", time.Minute)
	if err != nil {
		t.Fatalf("verification code: %v", err)
	}
	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"username":       "emailtabuser",
		"email":          "email-tab-user@example.com",
		"phone":          "13900000009",
		"password":       "pass123",
		"verify_channel": "email",
		"verify_code":    verifyCode,
		"captcha_id":     captcha.ID,
		"captcha_code":   captchaCode,
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("register by email expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	user, err := env.Repo.GetUserByUsernameOrEmail(ctx, "emailtabuser")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if user.Phone != "" {
		t.Fatalf("expected phone empty for email-channel register, got %q", user.Phone)
	}
	if user.Email != "email-tab-user@example.com" {
		t.Fatalf("unexpected email: %q", user.Email)
	}
}

func TestHandlers_LoginNotify_FirstDisabled_DoesNotTriggerOnFirstLogin(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	ctx := context.Background()

	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_login_notify_enabled", "true"); err != nil {
		t.Fatalf("set login notify enabled: %v", err)
	}
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_login_notify_on_first_login", "false"); err != nil {
		t.Fatalf("set login notify first: %v", err)
	}
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_login_notify_on_ip_change", "true"); err != nil {
		t.Fatalf("set login notify ip change: %v", err)
	}
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_login_notify_channels", `["email"]`); err != nil {
		t.Fatalf("set login notify channels: %v", err)
	}

	_ = testutil.CreateUser(t, env.Repo, "notify-first-off", "notify-first-off@example.com", "pass123")
	rec := doJSONWithIP(t, env.Router, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"username": "notify-first-off",
		"password": "pass123",
	}, "", "8.8.8.8")
	if rec.Code != http.StatusOK {
		t.Fatalf("login expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	user, err := env.Repo.GetUserByUsernameOrEmail(ctx, "notify-first-off")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if user.LastLoginIP != "8.8.8.8" {
		t.Fatalf("unexpected last_login_ip: %q", user.LastLoginIP)
	}
	if user.LastLoginCity != "" {
		t.Fatalf("expected no notify city update on first login when first trigger disabled, got %q", user.LastLoginCity)
	}
}

func TestHandlers_LoginNotify_IPChange_TriggersWhenIPChanged(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	ctx := context.Background()

	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_login_notify_enabled", "true"); err != nil {
		t.Fatalf("set login notify enabled: %v", err)
	}
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_login_notify_on_first_login", "false"); err != nil {
		t.Fatalf("set login notify first: %v", err)
	}
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_login_notify_on_ip_change", "true"); err != nil {
		t.Fatalf("set login notify ip change: %v", err)
	}
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_login_notify_channels", `["email"]`); err != nil {
		t.Fatalf("set login notify channels: %v", err)
	}

	user := testutil.CreateUser(t, env.Repo, "notify-ip-change", "notify-ip-change@example.com", "pass123")
	lastAt := time.Now().Add(-time.Hour)
	user.LastLoginIP = "1.1.1.1"
	user.LastLoginAt = &lastAt
	user.LastLoginCity = "old-city"
	user.LastLoginTZ = "GMT+08:00"
	if err := env.Repo.UpdateUser(ctx, user); err != nil {
		t.Fatalf("update user: %v", err)
	}

	rec := doJSONWithIP(t, env.Router, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"username": "notify-ip-change",
		"password": "pass123",
	}, "", "8.8.4.4")
	if rec.Code != http.StatusOK {
		t.Fatalf("login expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	updated, err := env.Repo.GetUserByUsernameOrEmail(ctx, "notify-ip-change")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if updated.LastLoginIP != "8.8.4.4" {
		t.Fatalf("unexpected last_login_ip: %q", updated.LastLoginIP)
	}
	if updated.LastLoginCity != "未知地区" {
		t.Fatalf("expected notify path to update city fallback to 未知地区, got %q", updated.LastLoginCity)
	}
}

func TestHandlers_Register_VerificationCodeTTLHonored(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	ctx := context.Background()
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_register_verify_channels", `["email"]`); err != nil {
		t.Fatalf("set verify channels: %v", err)
	}
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_register_captcha_enabled", "false"); err != nil {
		t.Fatalf("disable captcha: %v", err)
	}

	code, err := env.AuthSvc.CreateVerificationCode(ctx, "email", "ttl-register@example.com", "register", time.Second)
	if err != nil {
		t.Fatalf("create verification code: %v", err)
	}
	time.Sleep(1200 * time.Millisecond)

	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"username":       "ttlregisteruser",
		"email":          "ttl-register@example.com",
		"password":       "pass123",
		"verify_channel": "email",
		"verify_code":    code,
	}, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for expired register code, got %d: %s", rec.Code, rec.Body.String())
	}
	if msg := strings.ToLower(rec.Body.String()); !strings.Contains(msg, "invalid verification code") {
		t.Fatalf("expected invalid verification code error, got %s", rec.Body.String())
	}
}

func TestHandlers_PasswordReset_PhoneFullValidationAndTicketLifecycle(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	ctx := context.Background()
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_password_reset_enabled", "true"); err != nil {
		t.Fatalf("enable password reset: %v", err)
	}
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_password_reset_channels", `["email","sms"]`); err != nil {
		t.Fatalf("set password reset channels: %v", err)
	}
	user := testutil.CreateUser(t, env.Repo, "pwreset-user", "pwreset-user@example.com", "pass123")
	user.Phone = "13900000033"
	if err := env.Repo.UpdateUser(ctx, user); err != nil {
		t.Fatalf("set user phone: %v", err)
	}

	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/password-reset/options", map[string]any{
		"account": "pwreset-user",
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("options expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	options := parseJSONBody(t, rec)
	channels, _ := options["channels"].([]any)
	if len(channels) != 2 {
		t.Fatalf("expected 2 channels, got %v", options["channels"])
	}
	if requires, ok := options["sms_requires_phone_full"].(bool); !ok || !requires {
		t.Fatalf("expected sms_requires_phone_full=true for username lookup, got %v", options["sms_requires_phone_full"])
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/password-reset/send-code", map[string]any{
		"account": "pwreset-user",
		"channel": "sms",
	}, "")
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "phone_full required") {
		t.Fatalf("expected phone_full required, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/password-reset/send-code", map[string]any{
		"account":    "pwreset-user",
		"channel":    "sms",
		"phone_full": "13900009999",
	}, "")
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "phone mismatch") {
		t.Fatalf("expected phone mismatch, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/password-reset/send-code", map[string]any{
		"account": "13900000033",
		"channel": "sms",
	}, "")
	if rec.Code != http.StatusBadRequest || strings.Contains(rec.Body.String(), "phone_full required") {
		t.Fatalf("full phone as account should not require phone_full, got %d: %s", rec.Code, rec.Body.String())
	}

	expiredCode, err := env.AuthSvc.CreateVerificationCode(ctx, "email", "pwreset-user@example.com", "password_reset", time.Second)
	if err != nil {
		t.Fatalf("create expired code: %v", err)
	}
	time.Sleep(1200 * time.Millisecond)
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/password-reset/verify-code", map[string]any{
		"account": "pwreset-user",
		"channel": "email",
		"code":    expiredCode,
	}, "")
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "invalid verification code") {
		t.Fatalf("expected expired verify-code failure, got %d: %s", rec.Code, rec.Body.String())
	}

	validCode, err := env.AuthSvc.CreateVerificationCode(ctx, "email", "pwreset-user@example.com", "password_reset", time.Minute)
	if err != nil {
		t.Fatalf("create valid code: %v", err)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/password-reset/verify-code", map[string]any{
		"account": "pwreset-user",
		"channel": "email",
		"code":    validCode,
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("verify-code expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	verifyBody := parseJSONBody(t, rec)
	resetTicket, _ := verifyBody["reset_ticket"].(string)
	if strings.TrimSpace(resetTicket) == "" {
		t.Fatalf("reset ticket missing")
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/password-reset/confirm", map[string]any{
		"reset_ticket": resetTicket,
		"new_password": "newpass123",
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("confirm expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/password-reset/confirm", map[string]any{
		"reset_ticket": resetTicket,
		"new_password": "newpass456",
	}, "")
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "invalid reset ticket") {
		t.Fatalf("expected used ticket rejection, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"username": "pwreset-user",
		"password": "newpass123",
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login with new password expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlers_PasswordReset_InvalidatesOldTokensAndReturnsNewTokens(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	ctx := context.Background()
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_password_reset_enabled", "true"); err != nil {
		t.Fatalf("enable password reset: %v", err)
	}
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_password_reset_channels", `["email"]`); err != nil {
		t.Fatalf("set password reset channels: %v", err)
	}
	_ = testutil.CreateUser(t, env.Repo, "reset-token-user", "reset-token-user@example.com", "pass123")

	loginRec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"username": "reset-token-user",
		"password": "pass123",
	}, "")
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login expected 200, got %d: %s", loginRec.Code, loginRec.Body.String())
	}
	loginBody := parseJSONBody(t, loginRec)
	oldAccess, _ := loginBody["access_token"].(string)
	oldRefresh, _ := loginBody["refresh_token"].(string)
	if strings.TrimSpace(oldAccess) == "" || strings.TrimSpace(oldRefresh) == "" {
		t.Fatalf("login tokens missing")
	}

	code, err := env.AuthSvc.CreateVerificationCode(ctx, "email", "reset-token-user@example.com", "password_reset", time.Minute)
	if err != nil {
		t.Fatalf("create verify code: %v", err)
	}
	verifyRec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/password-reset/verify-code", map[string]any{
		"account": "reset-token-user",
		"channel": "email",
		"code":    code,
	}, "")
	if verifyRec.Code != http.StatusOK {
		t.Fatalf("verify-code expected 200, got %d: %s", verifyRec.Code, verifyRec.Body.String())
	}
	verifyBody := parseJSONBody(t, verifyRec)
	resetTicket, _ := verifyBody["reset_ticket"].(string)
	if strings.TrimSpace(resetTicket) == "" {
		t.Fatalf("reset ticket missing")
	}

	confirmRec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/password-reset/confirm", map[string]any{
		"reset_ticket": resetTicket,
		"new_password": "newpass123",
	}, "")
	if confirmRec.Code != http.StatusOK {
		t.Fatalf("confirm expected 200, got %d: %s", confirmRec.Code, confirmRec.Body.String())
	}
	confirmBody := parseJSONBody(t, confirmRec)
	newAccess, _ := confirmBody["access_token"].(string)
	newRefresh, _ := confirmBody["refresh_token"].(string)
	if strings.TrimSpace(newAccess) == "" || strings.TrimSpace(newRefresh) == "" {
		t.Fatalf("new tokens missing in reset confirm response")
	}

	oldMe := testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/me", nil, oldAccess)
	if oldMe.Code != http.StatusUnauthorized {
		t.Fatalf("old access token should be invalid, got %d: %s", oldMe.Code, oldMe.Body.String())
	}

	newMe := testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/me", nil, newAccess)
	if newMe.Code != http.StatusOK {
		t.Fatalf("new access token should work, got %d: %s", newMe.Code, newMe.Body.String())
	}

	oldRefreshRec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/refresh", map[string]any{
		"refresh_token": oldRefresh,
	}, "")
	if oldRefreshRec.Code != http.StatusUnauthorized {
		t.Fatalf("old refresh token should be invalid, got %d: %s", oldRefreshRec.Code, oldRefreshRec.Body.String())
	}

	newRefreshRec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/refresh", map[string]any{
		"refresh_token": newRefresh,
	}, "")
	if newRefreshRec.Code != http.StatusOK {
		t.Fatalf("new refresh token should work, got %d: %s", newRefreshRec.Code, newRefreshRec.Body.String())
	}
}

func TestHandlers_MePasswordChange_RequiresCurrentPassword(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)

	user := testutil.CreateUser(t, env.Repo, "pwd-change-user", "pwd-change-user@example.com", "pass123")
	token := testutil.IssueJWT(t, env.JWTSecret, user.ID, "user", time.Hour)

	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/me/password/change", map[string]any{
		"current_password": "wrong-pass",
		"new_password":     "newpass123",
	}, token)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "current password invalid") {
		t.Fatalf("expected current password invalid, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/me/password/change", map[string]any{
		"current_password": "pass123",
		"new_password":     "newpass123",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("password change expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"username": "pwd-change-user",
		"password": "newpass123",
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login with updated password expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlers_UpdateProfile_RejectsPasswordChange(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	user := testutil.CreateUser(t, env.Repo, "profile-pwd-reject", "profile-pwd-reject@example.com", "pass123")
	token := testutil.IssueJWT(t, env.JWTSecret, user.ID, "user", time.Hour)

	rec := testutil.DoJSON(t, env.Router, http.MethodPatch, "/api/v1/me", map[string]any{
		"password": "newpass123",
	}, token)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "/api/v1/me/password/change") {
		t.Fatalf("profile password update should be rejected, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"username": "profile-pwd-reject",
		"password": "pass123",
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("old password should still work, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlers_UpdateProfile_Requires2FAForSensitiveFields(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	ctx := context.Background()
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_2fa_enabled", "true"); err != nil {
		t.Fatalf("enable 2fa switch: %v", err)
	}
	user := testutil.CreateUser(t, env.Repo, "profile-2fa-user", "profile-2fa-user@example.com", "pass123")
	user.TOTPEnabled = true
	user.TOTPSecretEnc = "JBSWY3DPEHPK3PXP"
	if err := env.Repo.UpdateUser(ctx, user); err != nil {
		t.Fatalf("enable user 2fa: %v", err)
	}
	token := testutil.IssueJWT(t, env.JWTSecret, user.ID, "user", time.Hour)

	rec := testutil.DoJSON(t, env.Router, http.MethodPatch, "/api/v1/me", map[string]any{
		"username": "profile-2fa-user-new",
	}, token)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "invalid 2fa code") {
		t.Fatalf("sensitive profile update without 2fa should fail, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/api/v1/me", map[string]any{
		"username":  "profile-2fa-user-new",
		"totp_code": generateCurrentTOTPCode(t, "JBSWY3DPEHPK3PXP"),
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("sensitive profile update with valid 2fa should pass, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlers_MePasswordChange_Requires2FAWhenEnabled(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	ctx := context.Background()
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_2fa_enabled", "true"); err != nil {
		t.Fatalf("enable 2fa switch: %v", err)
	}
	user := testutil.CreateUser(t, env.Repo, "pwd-change-2fa-user", "pwd-change-2fa@example.com", "pass123")
	user.TOTPEnabled = true
	user.TOTPSecretEnc = "JBSWY3DPEHPK3PXP"
	if err := env.Repo.UpdateUser(ctx, user); err != nil {
		t.Fatalf("enable user 2fa: %v", err)
	}
	token := testutil.IssueJWT(t, env.JWTSecret, user.ID, "user", time.Hour)

	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/me/password/change", map[string]any{
		"current_password": "pass123",
		"new_password":     "newpass123",
	}, token)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "invalid 2fa code") {
		t.Fatalf("password change without 2fa should fail, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/me/password/change", map[string]any{
		"current_password": "pass123",
		"new_password":     "newpass123",
		"totp_code":        generateCurrentTOTPCode(t, "JBSWY3DPEHPK3PXP"),
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("password change with 2fa expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlers_ContactBind_PasswordAnd2FARules(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	ctx := context.Background()

	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_email_bind_enabled", "true"); err != nil {
		t.Fatalf("enable email bind: %v", err)
	}
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_bind_require_password_when_no_2fa", "false"); err != nil {
		t.Fatalf("set bind password rule: %v", err)
	}
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_rebind_require_password_when_no_2fa", "true"); err != nil {
		t.Fatalf("set rebind password rule: %v", err)
	}
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_2fa_enabled", "true"); err != nil {
		t.Fatalf("enable 2fa switch: %v", err)
	}

	bindUser := testutil.CreateUser(t, env.Repo, "bind-no-pass", "", "pass123")
	bindToken := testutil.IssueJWT(t, env.JWTSecret, bindUser.ID, "user", time.Hour)
	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/me/security/email/send-code", map[string]any{
		"value": "first-bind@example.com",
	}, bindToken)
	if rec.Code != http.StatusBadRequest || strings.Contains(rec.Body.String(), "invalid password") {
		t.Fatalf("first bind without 2fa should not require password, got %d: %s", rec.Code, rec.Body.String())
	}

	rebindUser := testutil.CreateUser(t, env.Repo, "rebind-pass", "old-bind@example.com", "pass123")
	rebindToken := testutil.IssueJWT(t, env.JWTSecret, rebindUser.ID, "user", time.Hour)
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/me/security/email/send-code", map[string]any{
		"value": "new-bind@example.com",
	}, rebindToken)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "invalid password") {
		t.Fatalf("rebind without 2fa should require password, got %d: %s", rec.Code, rec.Body.String())
	}

	rebindUser.TOTPEnabled = true
	rebindUser.TOTPSecretEnc = "JBSWY3DPEHPK3PXP"
	if err := env.Repo.UpdateUser(ctx, rebindUser); err != nil {
		t.Fatalf("enable user 2fa: %v", err)
	}
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_2fa_rebind_enabled", "false"); err != nil {
		t.Fatalf("set 2fa rebind switch: %v", err)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/me/security/email/send-code", map[string]any{
		"value":            "new-bind-2fa@example.com",
		"current_password": "pass123",
	}, rebindToken)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "security ticket required") {
		t.Fatalf("when user has 2fa enabled, contact rebind should require security ticket, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlers_PasswordReset_DisabledBlocksVerifyAndConfirm(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	ctx := context.Background()
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_password_reset_enabled", "false"); err != nil {
		t.Fatalf("disable password reset: %v", err)
	}

	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/password-reset/verify-code", map[string]any{
		"account": "whatever",
		"channel": "email",
		"code":    "123456",
	}, "")
	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "password reset disabled") {
		t.Fatalf("verify-code should be blocked when disabled, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/password-reset/confirm", map[string]any{
		"reset_ticket": "t",
		"new_password": "newpass123",
	}, "")
	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "password reset disabled") {
		t.Fatalf("confirm should be blocked when disabled, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandlers_PasswordResetSendCode_RateLimit(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	ctx := context.Background()
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_password_reset_enabled", "true"); err != nil {
		t.Fatalf("enable password reset: %v", err)
	}
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_password_reset_channels", `["email"]`); err != nil {
		t.Fatalf("set password reset channels: %v", err)
	}
	_ = testutil.CreateUser(t, env.Repo, "reset-rate-user", "reset-rate-user@example.com", "pass123")

	var last *httptest.ResponseRecorder
	for i := 0; i < 4; i++ {
		last = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/password-reset/send-code", map[string]any{
			"account": "reset-rate-user",
			"channel": "email",
		}, "")
	}
	if last == nil {
		t.Fatalf("missing response")
	}
	if last.Code != http.StatusTooManyRequests || !strings.Contains(last.Body.String(), "too many requests") {
		t.Fatalf("expected 429 rate limit on 4th request, got %d: %s", last.Code, last.Body.String())
	}
}

func TestHandlers_Register_CaptchaFailDoesNotConsumeVerificationCode(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	ctx := context.Background()
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_register_verify_channels", `["sms"]`); err != nil {
		t.Fatalf("set verify channels: %v", err)
	}
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_register_email_required", "false"); err != nil {
		t.Fatalf("set email required: %v", err)
	}
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_register_captcha_enabled", "true"); err != nil {
		t.Fatalf("enable captcha: %v", err)
	}

	code, err := env.AuthSvc.CreateVerificationCode(ctx, "sms", "13900000111", "register", time.Minute)
	if err != nil {
		t.Fatalf("create verification code: %v", err)
	}

	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"username":       "captcha-order-user",
		"phone":          "13900000111",
		"password":       "pass123",
		"verify_channel": "sms",
		"verify_code":    code,
		"captcha_id":     "bad",
		"captcha_code":   "bad",
	}, "")
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "captcha failed") {
		t.Fatalf("expected captcha failed, got %d: %s", rec.Code, rec.Body.String())
	}

	captcha, captchaCode, err := env.AuthSvc.CreateCaptcha(ctx, time.Minute)
	if err != nil {
		t.Fatalf("create captcha: %v", err)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"username":       "captcha-order-user",
		"phone":          "13900000111",
		"password":       "pass123",
		"verify_channel": "sms",
		"verify_code":    code,
		"captcha_id":     captcha.ID,
		"captcha_code":   captchaCode,
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected second submit succeed with same verify code, got %d: %s", rec.Code, rec.Body.String())
	}
}
