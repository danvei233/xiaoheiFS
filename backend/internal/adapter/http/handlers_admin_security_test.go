package http_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/testutilhttp"
)

func TestHandlers_AdminPasswordChangeInvalidatesOldAccessAndRefreshTokens(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	admin := testutil.CreateAdmin(t, env.Repo, "admin-token-revoke", "admin-token-revoke@example.com", "pass123", ensureAdminGroup(t, env))

	loginRec := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/auth/login", map[string]any{
		"username": admin.Username,
		"password": "pass123",
	}, "")
	if loginRec.Code != http.StatusOK {
		t.Fatalf("admin login expected 200, got %d: %s", loginRec.Code, loginRec.Body.String())
	}
	var loginResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(loginRec.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if loginResp.AccessToken == "" || loginResp.RefreshToken == "" {
		t.Fatal("admin login tokens missing")
	}

	changeRec := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/profile/change-password", map[string]any{
		"old_password": "pass123",
		"new_password": "newpass123",
	}, loginResp.AccessToken)
	if changeRec.Code != http.StatusOK {
		t.Fatalf("admin password change expected 200, got %d: %s", changeRec.Code, changeRec.Body.String())
	}

	accessRec := testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/profile", nil, loginResp.AccessToken)
	if accessRec.Code != http.StatusUnauthorized {
		t.Fatalf("old admin access token must return 401, got %d: %s", accessRec.Code, accessRec.Body.String())
	}
	refreshRec := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/auth/refresh", map[string]any{
		"refresh_token": loginResp.RefreshToken,
	}, "")
	if refreshRec.Code != http.StatusUnauthorized {
		t.Fatalf("old admin refresh token must return 401, got %d: %s", refreshRec.Code, refreshRec.Body.String())
	}
}

func TestHandlers_AdminRefreshTokenRotatesExactlyOnce(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	admin := testutil.CreateAdmin(t, env.Repo, "admin-refresh-once", "admin-refresh-once@example.com", "pass123", ensureAdminGroup(t, env))
	login := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/auth/login", map[string]any{
		"username": admin.Username,
		"password": "pass123",
	}, "")
	var loginBody struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(login.Body.Bytes(), &loginBody); err != nil || loginBody.RefreshToken == "" {
		t.Fatalf("decode admin login: err=%v body=%s", err, login.Body.String())
	}
	first := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/auth/refresh", map[string]any{"refresh_token": loginBody.RefreshToken}, "")
	if first.Code != http.StatusOK {
		t.Fatalf("first admin refresh must succeed: %d %s", first.Code, first.Body.String())
	}
	second := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/auth/refresh", map[string]any{"refresh_token": loginBody.RefreshToken}, "")
	if second.Code != http.StatusUnauthorized {
		t.Fatalf("replayed admin refresh token must be rejected: %d %s", second.Code, second.Body.String())
	}
	var firstBody struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &firstBody); err != nil || firstBody.RefreshToken == "" {
		t.Fatalf("decode rotated admin token: err=%v body=%s", err, first.Body.String())
	}
	rotated := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/auth/refresh", map[string]any{"refresh_token": firstBody.RefreshToken}, "")
	if rotated.Code != http.StatusUnauthorized {
		t.Fatalf("admin replay detection must revoke the rotated family: %d %s", rotated.Code, rotated.Body.String())
	}
}

func TestHandlers_AdminLogoutRevokesRefreshTokenFamily(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	admin := testutil.CreateAdmin(t, env.Repo, "admin-logout-family", "admin-logout-family@example.com", "pass123", ensureAdminGroup(t, env))
	login := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/auth/login", map[string]any{
		"username": admin.Username,
		"password": "pass123",
	}, "")
	var loginBody struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(login.Body.Bytes(), &loginBody); err != nil || loginBody.AccessToken == "" || loginBody.RefreshToken == "" {
		t.Fatalf("decode admin login: err=%v body=%s", err, login.Body.String())
	}
	logout := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/auth/logout", nil, loginBody.AccessToken)
	if logout.Code != http.StatusOK {
		t.Fatalf("admin logout must succeed: %d %s", logout.Code, logout.Body.String())
	}
	refresh := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/auth/refresh", map[string]any{
		"refresh_token": loginBody.RefreshToken,
	}, "")
	if refresh.Code != http.StatusUnauthorized {
		t.Fatalf("admin logout must revoke refresh family: %d %s", refresh.Code, refresh.Body.String())
	}
}

func TestHandlers_AdminLogin_LockAfterConsecutiveFailures(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	groupID := ensureAdminGroup(t, env)
	admin := testutil.CreateAdmin(t, env.Repo, "admin-lockout-case", "admin-lockout-case@example.com", "pass", groupID)

	for i := 0; i < 9; i++ {
		rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/auth/login", map[string]any{
			"username": admin.Username,
			"password": "wrong-pass",
		}, "")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d expected 401, got %d: %s", i+1, rec.Code, rec.Body.String())
		}
	}

	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/auth/login", map[string]any{
		"username": admin.Username,
		"password": "wrong-pass",
	}, "")
	if rec.Code != http.StatusTooManyRequests || !strings.Contains(rec.Body.String(), "too many attempts") {
		t.Fatalf("10th failure should trigger cooldown, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/auth/login", map[string]any{
		"username": admin.Username,
		"password": "pass",
	}, "")
	if rec.Code != http.StatusTooManyRequests || !strings.Contains(rec.Body.String(), "too many attempts") {
		t.Fatalf("cooldown period should block login, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestHandlers_Admin2FAUnlock_TemporaryLockoutNotDisable verifies P0-3:
// repeated wrong 2FA codes apply a temporary cooldown and must never
// permanently disable the admin account (which would be an operator DoS).
func TestHandlers_Admin2FAUnlock_TemporaryLockoutNotDisable(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	groupID := ensureAdminGroup(t, env)
	admin := testutil.CreateAdmin(t, env.Repo, "admin-2fa-lock-case", "admin-2fa-lock-case@example.com", "pass", groupID)
	admin.TOTPEnabled = true
	admin.TOTPSecretEnc = encryptTOTPSecretForTest(t, "JBSWY3DPEHPK3PXP")
	if err := env.Repo.UpdateUser(context.Background(), admin); err != nil {
		t.Fatalf("enable admin totp: %v", err)
	}

	loginRec := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/auth/login", map[string]any{
		"username": admin.Username,
		"password": "pass",
	}, "")
	if loginRec.Code != http.StatusOK {
		t.Fatalf("admin login code: %d: %s", loginRec.Code, loginRec.Body.String())
	}
	var loginResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(loginRec.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if strings.TrimSpace(loginResp.AccessToken) == "" {
		t.Fatalf("access token missing")
	}

	// Wrong codes up to the threshold are rejected as invalid codes.
	for i := 0; i < 4; i++ {
		rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/auth/2fa/unlock", map[string]any{
			"totp_code": "000000",
		}, loginResp.AccessToken)
		if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "invalid 2fa code") {
			t.Fatalf("2fa attempt %d expected invalid code, got %d: %s", i+1, rec.Code, rec.Body.String())
		}
	}

	// The threshold attempt triggers a temporary cooldown (429), not a disable.
	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/auth/2fa/unlock", map[string]any{
		"totp_code": "000000",
	}, loginResp.AccessToken)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("threshold invalid 2fa should trigger cooldown, got %d: %s", rec.Code, rec.Body.String())
	}

	// The account must remain active and able to log in — nothing was disabled.
	updatedAdmin, err := env.Repo.GetUserByID(context.Background(), admin.ID)
	if err != nil {
		t.Fatalf("query admin after lockout: %v", err)
	}
	if updatedAdmin.Status != domain.UserStatusActive {
		t.Fatalf("admin status must stay active after failed 2FA, got %s", updatedAdmin.Status)
	}

	loginRec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/auth/login", map[string]any{
		"username": admin.Username,
		"password": "pass",
	}, "")
	if loginRec.Code != http.StatusOK {
		t.Fatalf("admin must still be able to log in after 2FA cooldown, got %d: %s", loginRec.Code, loginRec.Body.String())
	}
}

// TestHandlers_AdminLogin_IPTThrottle verifies P0-4: a single source IP that
// rotates many usernames (credential spray) hits an IP-level throttle rather
// than only per-username cooldowns.
func TestHandlers_AdminLogin_IPTThrottle(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	groupID := ensureAdminGroup(t, env)
	_ = testutil.CreateAdmin(t, env.Repo, "admin-ip-throttle", "admin-ip-throttle@example.com", "pass", groupID)

	const ipBudget = 50 // keep in sync with handlers.go adminLoginIPMax
	for i := 0; i < ipBudget; i++ {
		// Each attempt uses a fresh username so the per-username cooldown never
		// triggers; only the IP budget can stop the spray.
		rec := doJSONWithIP(t, env.Router, http.MethodPost, "/admin/api/v1/auth/login", map[string]any{
			"username": fmt.Sprintf("spray-user-%d", i),
			"password": "wrong-pass",
		}, "", "198.51.100.5")
		if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusTooManyRequests {
			t.Fatalf("attempt %d expected 401 or 429, got %d: %s", i+1, rec.Code, rec.Body.String())
		}
	}
	// The very next attempt from the same IP must be throttled even with a new
	// username.
	rec := doJSONWithIP(t, env.Router, http.MethodPost, "/admin/api/v1/auth/login", map[string]any{
		"username": "spray-victim",
		"password": "wrong-pass",
	}, "", "198.51.100.5")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("IP throttle should block further spray, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestHandlers_Admin2FAUnlock_IPRotationCannotBypassAccountBudget verifies the
// 2FA failure budget is split across account and IP dimensions: locking the
// account from one IP still blocks it from another IP (a rotating attacker
// cannot reset the account budget by changing source IPs).
func TestHandlers_Admin2FAUnlock_IPRotationCannotBypassAccountBudget(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	groupID := ensureAdminGroup(t, env)
	admin := testutil.CreateAdmin(t, env.Repo, "admin-2fa-iprot", "admin-2fa-iprot@example.com", "pass123", groupID)
	admin.TOTPEnabled = true
	admin.TOTPSecretEnc = encryptTOTPSecretForTest(t, "JBSWY3DPEHPK3PXP")
	if err := env.Repo.UpdateUser(context.Background(), admin); err != nil {
		t.Fatalf("enable admin totp: %v", err)
	}

	loginRec := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/auth/login", map[string]any{
		"username": admin.Username,
		"password": "pass123",
	}, "")
	if loginRec.Code != http.StatusOK {
		t.Fatalf("admin login: %d", loginRec.Code)
	}
	var loginResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(loginRec.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("decode login: %v", err)
	}

	// Four wrong codes from IP A: not yet locked.
	for i := 0; i < 4; i++ {
		rec := doJSONWithIP(t, env.Router, http.MethodPost, "/admin/api/v1/auth/2fa/unlock", map[string]any{"totp_code": "000000"}, loginResp.AccessToken, "198.51.100.10")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("attempt %d from IP A expected invalid code, got %d", i+1, rec.Code)
		}
	}
	// Fifth wrong code from IP A locks the account budget.
	rec := doJSONWithIP(t, env.Router, http.MethodPost, "/admin/api/v1/auth/2fa/unlock", map[string]any{"totp_code": "000000"}, loginResp.AccessToken, "198.51.100.10")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("fifth wrong code from IP A should lock the account, got %d", rec.Code)
	}
	// Rotating to IP B must not bypass the account budget.
	rec = doJSONWithIP(t, env.Router, http.MethodPost, "/admin/api/v1/auth/2fa/unlock", map[string]any{"totp_code": "000000"}, loginResp.AccessToken, "198.51.100.11")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("account budget must survive an IP rotation, got %d: %s", rec.Code, rec.Body.String())
	}
}
