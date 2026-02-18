package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/testutilhttp"
)

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

func TestHandlers_Admin2FAUnlock_DisableAdminAfterConsecutiveFailures(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	groupID := ensureAdminGroup(t, env)
	admin := testutil.CreateAdmin(t, env.Repo, "admin-2fa-lock-case", "admin-2fa-lock-case@example.com", "pass", groupID)
	admin.TOTPEnabled = true
	admin.TOTPSecretEnc = "JBSWY3DPEHPK3PXP"
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

	for i := 0; i < 9; i++ {
		rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/auth/2fa/unlock", map[string]any{
			"totp_code": "000000",
		}, loginResp.AccessToken)
		if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "invalid 2fa code") {
			t.Fatalf("2fa attempt %d expected invalid code, got %d: %s", i+1, rec.Code, rec.Body.String())
		}
	}

	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/auth/2fa/unlock", map[string]any{
		"totp_code": "000000",
	}, loginResp.AccessToken)
	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "user disabled") {
		t.Fatalf("10th invalid 2fa should disable admin, got %d: %s", rec.Code, rec.Body.String())
	}

	updatedAdmin, err := env.Repo.GetUserByID(context.Background(), admin.ID)
	if err != nil {
		t.Fatalf("query admin after disable: %v", err)
	}
	if updatedAdmin.Status != domain.UserStatusDisabled {
		t.Fatalf("admin status should be disabled, got %s", updatedAdmin.Status)
	}

	loginRec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/auth/login", map[string]any{
		"username": admin.Username,
		"password": "pass",
	}, "")
	if loginRec.Code != http.StatusUnauthorized {
		t.Fatalf("disabled admin login should fail, got %d: %s", loginRec.Code, loginRec.Body.String())
	}
}
