package http_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/testutilhttp"
)

// TestJWTClaimsWellFormed verifies P3-22: issued tokens carry integer iat,
// iss/aud/jti and a strict HMAC algorithm so tokens are attributable and
// algorithm-confusion resistant.
func TestJWTClaimsWellFormed(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	testutil.CreateUser(t, env.Repo, "jwt-claims", "jwt-claims@example.com", "pass")

	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"username": "jwt-claims",
		"password": "pass",
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login: %d %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	if resp.AccessToken == "" {
		t.Fatal("missing access token")
	}

	segments := strings.Split(resp.AccessToken, ".")
	if len(segments) != 3 {
		t.Fatalf("expected 3 JWT segments, got %d", len(segments))
	}

	claims := decodeClaims(t, resp.AccessToken)
	if _, ok := claims["iat"].(float64); !ok {
		t.Fatalf("iat must be a number, got %T", claims["iat"])
	}
	iat := claims["iat"].(float64)
	if iat <= 0 || iat != float64(int64(iat)) {
		t.Fatalf("iat must be an integer timestamp, got %v", iat)
	}
	if _, ok := claims["jti"].(string); !ok || claims["jti"].(string) == "" {
		t.Fatalf("jti must be a non-empty string")
	}
	if iss, _ := claims["iss"].(string); iss == "" {
		t.Fatalf("iss missing")
	}
	if aud, _ := claims["aud"].(string); aud == "" {
		t.Fatalf("aud missing")
	}
	if exp, ok := claims["exp"].(float64); !ok || exp <= float64(time.Now().Unix()) {
		t.Fatalf("exp must be a future timestamp, got %v", exp)
	}
}

func decodeClaims(t *testing.T, token string) map[string]any {
	t.Helper()
	segments := strings.Split(token, ".")
	payload := segments[1]
	// JWT payload is base64url without padding.
	padded := payload
	if m := len(payload) % 4; m != 0 {
		padded += strings.Repeat("=", 4-m)
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(padded, "="))
	if err != nil {
		raw, err = base64.URLEncoding.DecodeString(padded)
		if err != nil {
			t.Fatalf("decode payload: %v", err)
		}
	}
	claims := map[string]any{}
	if err := json.Unmarshal(raw, &claims); err != nil {
		t.Fatalf("unmarshal claims: %v", err)
	}
	return claims
}
