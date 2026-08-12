package http_test

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/testutilhttp"
)

// web_auth_surface F1：管理员模拟登录签发的 token 必须带 `purpose:"impersonation"`
// claim，普通 auth/login 签发的 token 不带。前端受理 `#impersonate_token=`
// 时只读该 claim 区分，攻击者用自己账号的普通登录 token 构造链接会被拒。
func TestHandlers_AdminUserImpersonate_IssuesPurposeClaim(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	groupID := ensureAdminGroup(t, env)
	admin := testutil.CreateAdmin(t, env.Repo, "imp-admin", "imp-admin@example.com", "pass", groupID)
	adminToken := testutil.IssueJWT(t, env.JWTSecret, admin.ID, "admin", time.Hour)

	user := testutil.CreateUser(t, env.Repo, "imp-target", "imp-target@example.com", "pass")

	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/users/"+strconv.FormatInt(user.ID, 10)+"/impersonate", nil, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for impersonate, got %d: %s", rec.Code, rec.Body.String())
	}
	var payload struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode impersonate response: %v", err)
	}
	if payload.AccessToken == "" {
		t.Fatalf("expected access_token in response")
	}

	claims := jwt.MapClaims{}
	parsed, err := jwt.ParseWithClaims(payload.AccessToken, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(env.JWTSecret), nil
	})
	if err != nil || parsed == nil || !parsed.Valid {
		t.Fatalf("impersonation token must be a valid JWT signed with the server secret")
	}
	if claims["purpose"] != "impersonation" {
		t.Fatalf("impersonation token must carry purpose=impersonation, got %v", claims["purpose"])
	}
	if claims["type"] != "access" {
		t.Fatalf("impersonation token must be type=access, got %v", claims["type"])
	}

	// 普通登录 token 不带 purpose claim（前端据此拒绝攻击者自己的登录 token）。
	loginRec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"username": "imp-target",
		"password": "pass",
	}, "")
	if loginRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for login, got %d: %s", loginRec.Code, loginRec.Body.String())
	}
	var loginPayload struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(loginRec.Body.Bytes(), &loginPayload); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	loginClaims := jwt.MapClaims{}
	if _, err := jwt.ParseWithClaims(loginPayload.AccessToken, loginClaims, func(token *jwt.Token) (interface{}, error) {
		return []byte(env.JWTSecret), nil
	}); err != nil {
		t.Fatalf("login token must parse: %v", err)
	}
	if _, hasPurpose := loginClaims["purpose"]; hasPurpose {
		t.Fatalf("ordinary login token must NOT carry purpose claim")
	}
}
