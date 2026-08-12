package http_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/testutilhttp"
)

// buildJWT mints an HS256 token with the given claims. Callers override the
// fields they want to test; the defaults below are a well-formed access token.
func buildJWT(t *testing.T, secret string, overrides jwt.MapClaims) string {
	t.Helper()
	claims := jwt.MapClaims{
		"user_id": int64(1),
		"role":    "user",
		"type":    "access",
		"iss":     "xiaoheiplay",
		"aud":     "xiaoheiplay",
		"iat":     time.Now().Unix(),
		"exp":     time.Now().Add(time.Hour).Unix(),
	}
	for k, v := range overrides {
		claims[k] = v
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign test jwt: %v", err)
	}
	return signed
}

// callMe hits /api/v1/me with a bearer token and returns the status.
func callMe(t *testing.T, env *testutilhttp.Env, token string) int {
	t.Helper()
	rec := testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/me", nil, token)
	return rec.Code
}

// TestJWTValidation_RejectsWrongIssuer verifies a token minted by a different
// issuer is rejected on parse, even with a valid signature.
func TestJWTValidation_RejectsWrongIssuer(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	user := testutil.CreateUser(t, env.Repo, "jwt-iss", "jwt-iss@example.com", "pass")
	token := buildJWT(t, env.JWTSecret, jwt.MapClaims{"user_id": user.ID, "iss": "evil-issuer"})
	if code := callMe(t, env, token); code != http.StatusUnauthorized {
		t.Fatalf("wrong issuer must be rejected, got %d", code)
	}
}

// TestJWTValidation_RejectsWrongAudience verifies a token addressed to a
// different audience is rejected.
func TestJWTValidation_RejectsWrongAudience(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	user := testutil.CreateUser(t, env.Repo, "jwt-aud", "jwt-aud@example.com", "pass")
	token := buildJWT(t, env.JWTSecret, jwt.MapClaims{"user_id": user.ID, "aud": "evil-audience"})
	if code := callMe(t, env, token); code != http.StatusUnauthorized {
		t.Fatalf("wrong audience must be rejected, got %d", code)
	}
}

// TestJWTValidation_RejectsRefreshAsAccess verifies a refresh token cannot be
// used on an access-token route (type is enforced on every parse).
func TestJWTValidation_RejectsRefreshAsAccess(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	user := testutil.CreateUser(t, env.Repo, "jwt-refr", "jwt-refr@example.com", "pass")
	token := buildJWT(t, env.JWTSecret, jwt.MapClaims{"user_id": user.ID, "type": "refresh"})
	if code := callMe(t, env, token); code != http.StatusUnauthorized {
		t.Fatalf("refresh token must not authorize an access route, got %d", code)
	}
}

// TestJWTValidation_RejectsNonHS256 verifies an HMAC token signed with a
// different algorithm than HS256 is rejected (algorithm-confusion guard).
func TestJWTValidation_RejectsNonHS256(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	user := testutil.CreateUser(t, env.Repo, "jwt-alg", "jwt-alg@example.com", "pass")
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"role":    "user",
		"type":    "access",
		"iss":     "xiaoheiplay",
		"aud":     "xiaoheiplay",
		"iat":     time.Now().Unix(),
		"exp":     time.Now().Add(time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	signed, err := tok.SignedString([]byte(env.JWTSecret))
	if err != nil {
		t.Fatalf("sign hs512: %v", err)
	}
	if code := callMe(t, env, signed); code != http.StatusUnauthorized {
		t.Fatalf("HS512 token must be rejected, got %d", code)
	}
}

// TestJWTValidation_RejectsExpired verifies an expired token is rejected.
func TestJWTValidation_RejectsExpired(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	user := testutil.CreateUser(t, env.Repo, "jwt-exp", "jwt-exp@example.com", "pass")
	token := buildJWT(t, env.JWTSecret, jwt.MapClaims{"user_id": user.ID, "exp": time.Now().Add(-time.Hour).Unix()})
	if code := callMe(t, env, token); code != http.StatusUnauthorized {
		t.Fatalf("expired token must be rejected, got %d", code)
	}
}

// TestJWTValidation_RejectsDisabledUser verifies a token for a disabled account
// is rejected even when the signature and claims are valid.
func TestJWTValidation_RejectsDisabledUser(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	user := testutil.CreateUser(t, env.Repo, "jwt-dis", "jwt-dis@example.com", "pass")
	token := buildJWT(t, env.JWTSecret, jwt.MapClaims{"user_id": user.ID})
	if err := env.Repo.UpdateUserStatus(t.Context(), user.ID, "disabled"); err != nil {
		t.Fatalf("disable user: %v", err)
	}
	if code := callMe(t, env, token); code != http.StatusUnauthorized {
		t.Fatalf("disabled user token must be rejected, got %d", code)
	}
}

// TestJWTValidation_RejectsRoleMismatch verifies a token whose role no longer
// matches the account's current role is rejected (current-role re-check).
func TestJWTValidation_RejectsRoleMismatch(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	user := testutil.CreateUser(t, env.Repo, "jwt-role", "jwt-role@example.com", "pass")
	token := buildJWT(t, env.JWTSecret, jwt.MapClaims{"user_id": user.ID, "role": "admin"})
	if code := callMe(t, env, token); code != http.StatusUnauthorized {
		t.Fatalf("role-mismatched token must be rejected, got %d", code)
	}
	_ = user
}

// TestJWTValidation_PasswordChangeRevokesOldTokens verifies the monotonic
// password-version revocation: tokens issued before a password change are
// rejected afterwards, and fresh tokens work — including in the same second the
// password changed (no timestamp-precision dependence).
func TestJWTValidation_PasswordChangeRevokesOldTokens(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	testutil.CreateUser(t, env.Repo, "jwt-pwv", "jwt-pwv@example.com", "pass")

	// Login and capture the token issued at password version 0.
	loginRec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"username": "jwt-pwv",
		"password": "pass",
	}, "")
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login: %d %s", loginRec.Code, loginRec.Body.String())
	}
	var resp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(loginRec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	oldToken := resp.AccessToken

	// Change the password (bumps password_version 0 -> 1).
	changeRec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/me/password/change", map[string]any{
		"current_password": "pass",
		"new_password":     "newpass",
	}, oldToken)
	if changeRec.Code != http.StatusOK {
		t.Fatalf("password change: %d %s", changeRec.Code, changeRec.Body.String())
	}

	// The pre-change token must be rejected.
	if code := callMe(t, env, oldToken); code != http.StatusUnauthorized {
		t.Fatalf("pre-change token must be revoked, got %d", code)
	}

	// A fresh login (same second is fine) must issue a working token.
	login2 := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"username": "jwt-pwv",
		"password": "newpass",
	}, "")
	if login2.Code != http.StatusOK {
		t.Fatalf("re-login: %d %s", login2.Code, login2.Body.String())
	}
	var resp2 struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(login2.Body.Bytes(), &resp2); err != nil {
		t.Fatalf("decode re-login: %v", err)
	}
	if code := callMe(t, env, resp2.AccessToken); code != http.StatusOK {
		t.Fatalf("fresh token after password change must be accepted, got %d", code)
	}
}
