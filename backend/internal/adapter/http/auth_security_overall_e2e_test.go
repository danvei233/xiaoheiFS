package http_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/testutilhttp"
)

func TestAuthSecurity_Overall_RegisterSMSAndPhoneLogin(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	ctx := context.Background()
	applyAuthSecurityDefaults(t, env)

	verifyCode, err := env.AuthSvc.CreateVerificationCode(ctx, "sms", "13900010001", "register", time.Minute)
	if err != nil {
		t.Fatalf("create sms verify code: %v", err)
	}
	captcha, captchaCode, err := env.AuthSvc.CreateCaptcha(ctx, time.Minute)
	if err != nil {
		t.Fatalf("create captcha: %v", err)
	}
	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"username":       "overall-sms-reg",
		"phone":          "13900010001",
		"password":       "pass123",
		"verify_channel": "sms",
		"verify_code":    verifyCode,
		"captcha_id":     captcha.ID,
		"captcha_code":   captchaCode,
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("register by sms expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	user, err := env.Repo.GetUserByUsernameOrEmail(ctx, "overall-sms-reg")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if strings.TrimSpace(user.Email) != "" {
		t.Fatalf("email must stay empty for sms-register user, got %q", user.Email)
	}
	if strings.TrimSpace(user.Phone) != "13900010001" {
		t.Fatalf("unexpected phone: %q", user.Phone)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"username": "13900010001",
		"password": "pass123",
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("phone login expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAuthSecurity_Overall_LoginNotifyMatrix(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	ctx := context.Background()
	applyAuthSecurityDefaults(t, env)

	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_login_notify_enabled", "true"); err != nil {
		t.Fatalf("enable login notify: %v", err)
	}
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_login_notify_on_first_login", "true"); err != nil {
		t.Fatalf("set first login notify: %v", err)
	}
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_login_notify_on_ip_change", "true"); err != nil {
		t.Fatalf("set ip change notify: %v", err)
	}
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_login_notify_channels", `["email"]`); err != nil {
		t.Fatalf("set notify channels: %v", err)
	}

	_ = testutil.CreateUser(t, env.Repo, "overall-notify", "overall-notify@example.com", "pass123")

	rec := doJSONWithIP(t, env.Router, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"username": "overall-notify",
		"password": "pass123",
	}, "", "1.1.1.1")
	if rec.Code != http.StatusOK {
		t.Fatalf("first login expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	user, err := env.Repo.GetUserByUsernameOrEmail(ctx, "overall-notify")
	if err != nil {
		t.Fatalf("get user after first login: %v", err)
	}
	if user.LastLoginIP != "1.1.1.1" {
		t.Fatalf("expected first login ip recorded, got %q", user.LastLoginIP)
	}

	rec = doJSONWithIP(t, env.Router, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"username": "overall-notify",
		"password": "pass123",
	}, "", "1.1.1.1")
	if rec.Code != http.StatusOK {
		t.Fatalf("second login same ip expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = doJSONWithIP(t, env.Router, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"username": "overall-notify",
		"password": "pass123",
	}, "", "8.8.8.8")
	if rec.Code != http.StatusOK {
		t.Fatalf("third login changed ip expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	user, err = env.Repo.GetUserByUsernameOrEmail(ctx, "overall-notify")
	if err != nil {
		t.Fatalf("get user after ip-changed login: %v", err)
	}
	if user.LastLoginIP != "8.8.8.8" {
		t.Fatalf("expected changed login ip recorded, got %q", user.LastLoginIP)
	}
}

func TestAuthSecurity_Overall_PasswordResetStateMachine(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	ctx := context.Background()
	applyAuthSecurityDefaults(t, env)

	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_password_reset_enabled", "true"); err != nil {
		t.Fatalf("enable password reset: %v", err)
	}
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_password_reset_channels", `["email","sms"]`); err != nil {
		t.Fatalf("set reset channels: %v", err)
	}

	user := testutil.CreateUser(t, env.Repo, "overall-reset", "overall-reset@example.com", "pass123")
	user.Phone = "13900010002"
	if err := env.Repo.UpdateUser(ctx, user); err != nil {
		t.Fatalf("bind phone: %v", err)
	}

	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/password-reset/options", map[string]any{
		"account": "overall-reset",
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("options expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	body := parseJSONBody(t, rec)
	channels, ok := body["channels"].([]any)
	if !ok || len(channels) != 2 {
		t.Fatalf("options channels expected [email,sms], got: %#v", body["channels"])
	}
	if v, _ := body["sms_requires_phone_full"].(bool); !v {
		t.Fatalf("sms_requires_phone_full should be true when account != full phone")
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/password-reset/send-code", map[string]any{
		"account": "overall-reset",
		"channel": "sms",
	}, "")
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "phone_full required") {
		t.Fatalf("sms reset without phone_full should fail, got %d: %s", rec.Code, rec.Body.String())
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/password-reset/send-code", map[string]any{
		"account":    "overall-reset",
		"channel":    "sms",
		"phone_full": "13900019999",
	}, "")
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "phone mismatch") {
		t.Fatalf("sms reset with wrong phone_full should fail, got %d: %s", rec.Code, rec.Body.String())
	}

	code, err := env.AuthSvc.CreateVerificationCode(ctx, "email", "overall-reset@example.com", "password_reset", time.Minute)
	if err != nil {
		t.Fatalf("create reset verify code: %v", err)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/password-reset/verify-code", map[string]any{
		"account": "overall-reset",
		"channel": "email",
		"code":    code,
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("verify reset code expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var verifyResp struct {
		ResetTicket string `json:"reset_ticket"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &verifyResp); err != nil {
		t.Fatalf("decode verify response: %v", err)
	}
	if strings.TrimSpace(verifyResp.ResetTicket) == "" {
		t.Fatalf("reset ticket missing")
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/password-reset/confirm", map[string]any{
		"reset_ticket": verifyResp.ResetTicket,
		"new_password": "pass456",
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("confirm reset expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"username": "overall-reset",
		"password": "pass456",
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login with reset password expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAuthSecurity_Overall_BindRebindAnd2FA(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	ctx := context.Background()
	applyAuthSecurityDefaults(t, env)

	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_email_bind_enabled", "true"); err != nil {
		t.Fatalf("enable email bind: %v", err)
	}
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_phone_bind_enabled", "true"); err != nil {
		t.Fatalf("enable phone bind: %v", err)
	}
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_bind_require_password_when_no_2fa", "false"); err != nil {
		t.Fatalf("set bind password rule: %v", err)
	}
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_rebind_require_password_when_no_2fa", "true"); err != nil {
		t.Fatalf("set rebind password rule: %v", err)
	}
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_2fa_enabled", "true"); err != nil {
		t.Fatalf("enable 2fa: %v", err)
	}
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_2fa_bind_enabled", "true"); err != nil {
		t.Fatalf("enable 2fa bind: %v", err)
	}
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "auth_2fa_rebind_enabled", "true"); err != nil {
		t.Fatalf("enable 2fa rebind: %v", err)
	}

	user := testutil.CreateUser(t, env.Repo, "overall-bind", "", "pass123")
	token := testutil.IssueJWT(t, env.JWTSecret, user.ID, "user", time.Hour)

	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/me/security/email/send-code", map[string]any{
		"value": "first-bind-overall@example.com",
	}, token)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "smtp disabled") {
		t.Fatalf("first bind without 2fa should pass auth gate then fail smtp in test env, got %d: %s", rec.Code, rec.Body.String())
	}

	code, err := env.AuthSvc.CreateVerificationCode(ctx, "email", "first-bind-overall@example.com", "bind_email", time.Minute)
	if err != nil {
		t.Fatalf("create bind email code: %v", err)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/me/security/email/confirm", map[string]any{
		"value": "first-bind-overall@example.com",
		"code":  code,
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("confirm first bind expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/me/security/email/send-code", map[string]any{
		"value": "second-bind-overall@example.com",
	}, token)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "invalid password") {
		t.Fatalf("rebind without 2fa should require password, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/me/security/2fa/setup", map[string]any{
		"password": "pass123",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("2fa setup expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var setupResp struct {
		Secret string `json:"secret"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &setupResp); err != nil {
		t.Fatalf("decode 2fa setup: %v", err)
	}
	if strings.TrimSpace(setupResp.Secret) == "" {
		t.Fatalf("2fa setup secret missing")
	}
	totpCode := generateCurrentTOTPCode(t, setupResp.Secret)

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/me/security/2fa/confirm", map[string]any{
		"code": totpCode,
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("2fa confirm expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/me/security/email/send-code", map[string]any{
		"value":            "third-bind-overall@example.com",
		"current_password": "pass123",
	}, token)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "security ticket required") {
		t.Fatalf("when 2fa enabled, password-only rebind should be rejected, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/me/security/email/verify-2fa", map[string]any{
		"totp_code": generateCurrentTOTPCode(t, setupResp.Secret),
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("verify 2fa for contact bind expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var verifyResp struct {
		SecurityTicket string `json:"security_ticket"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &verifyResp); err != nil {
		t.Fatalf("decode verify 2fa response: %v", err)
	}
	if strings.TrimSpace(verifyResp.SecurityTicket) == "" {
		t.Fatalf("security ticket missing")
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/me/security/email/send-code", map[string]any{
		"value":           "third-bind-overall@example.com",
		"security_ticket": verifyResp.SecurityTicket,
	}, token)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "smtp disabled") {
		t.Fatalf("when 2fa enabled, security ticket rebind should pass auth gate then fail smtp in test env, got %d: %s", rec.Code, rec.Body.String())
	}

	code, err = env.AuthSvc.CreateVerificationCode(ctx, "email", "third-bind-overall@example.com", "bind_email", time.Minute)
	if err != nil {
		t.Fatalf("create rebind email code: %v", err)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/me/security/email/confirm", map[string]any{
		"value":           "third-bind-overall@example.com",
		"code":            code,
		"security_ticket": verifyResp.SecurityTicket,
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("confirm rebind expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func applyAuthSecurityDefaults(t *testing.T, env *testutilhttp.Env) {
	t.Helper()
	ctx := context.Background()
	settings := map[string]string{
		"auth_register_verify_channels":            `["email","sms"]`,
		"auth_register_email_required":             "false",
		"auth_register_captcha_enabled":            "true",
		"auth_login_captcha_enabled":               "false",
		"auth_login_notify_enabled":                "true",
		"auth_login_notify_on_first_login":         "true",
		"auth_login_notify_on_ip_change":           "true",
		"auth_login_notify_channels":               `["email"]`,
		"auth_password_reset_enabled":              "true",
		"auth_password_reset_channels":             `["email","sms"]`,
		"auth_password_reset_verify_ttl_sec":       "600",
		"auth_email_bind_enabled":                  "true",
		"auth_phone_bind_enabled":                  "true",
		"auth_contact_bind_verify_ttl_sec":         "600",
		"auth_bind_require_password_when_no_2fa":   "false",
		"auth_rebind_require_password_when_no_2fa": "true",
		"auth_2fa_enabled":                         "true",
		"auth_2fa_bind_enabled":                    "true",
		"auth_2fa_rebind_enabled":                  "true",
	}
	for k, v := range settings {
		if err := env.AdminSvc.UpdateSetting(ctx, 1, k, v); err != nil {
			t.Fatalf("set %s failed: %v", k, err)
		}
	}
}

func generateCurrentTOTPCode(t *testing.T, secret string) string {
	t.Helper()
	now := time.Now()
	code := generateTOTPCode(secret, uint64(now.Unix()/30))
	if strings.TrimSpace(code) != "" {
		return code
	}
	t.Fatalf("generate totp code failed")
	return ""
}

func generateTOTPCode(secret string, counter uint64) string {
	secret = strings.ToUpper(strings.TrimSpace(secret))
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		return ""
	}
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], counter)
	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(msg[:])
	sum := mac.Sum(nil)
	if len(sum) < 20 {
		return ""
	}
	offset := int(sum[len(sum)-1] & 0x0f)
	bin := int32(sum[offset]&0x7f)<<24 | int32(sum[offset+1])<<16 | int32(sum[offset+2])<<8 | int32(sum[offset+3])
	otp := int(bin % 1000000)
	return fmt.Sprintf("%06d", otp)
}
