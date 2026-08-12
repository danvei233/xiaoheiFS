package http_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	httpadapter "xiaoheiplay/internal/adapter/http"
	appuserapikey "xiaoheiplay/internal/app/userapikey"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func TestMiddleware_RequireUser_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mw := httpadapter.NewMiddleware("secret", nil, nil, nil, nil, nil)
	r := gin.New()
	r.GET("/me", mw.RequireUser(), func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestMiddleware_RequireAdmin_Forbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mw := httpadapter.NewMiddleware("secret", nil, nil, nil, nil, nil)
	r := gin.New()
	r.GET("/admin", mw.RequireAdmin(), func(c *gin.Context) { c.Status(http.StatusOK) })

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": int64(1),
		"role":    "user",
		"type":    "access",
		"iss":     "xiaoheiplay",
		"aud":     "xiaoheiplay",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	signed, _ := token.SignedString([]byte("secret"))

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestMiddleware_RequireUser_RejectsAdminRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mw := httpadapter.NewMiddleware("secret", nil, nil, nil, nil, nil)
	r := gin.New()
	r.GET("/me", mw.RequireUser(), func(c *gin.Context) { c.Status(http.StatusOK) })

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": int64(1),
		"role":    "admin",
		"type":    "access",
		"iss":     "xiaoheiplay",
		"aud":     "xiaoheiplay",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	signed, _ := token.SignedString([]byte("secret"))

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestMiddleware_RequireUser_DoesNotAcceptQueryToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mw := httpadapter.NewMiddleware("secret", nil, nil, nil, nil, nil)
	r := gin.New()
	r.GET("/vps/:id/panel", mw.RequireUser(), func(c *gin.Context) { c.Status(http.StatusOK) })

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": int64(1),
		"role":    "user",
		"type":    "access",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	signed, _ := token.SignedString([]byte("secret"))

	req := httptest.NewRequest(http.MethodGet, "/vps/1/panel?token="+signed, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestMiddleware_RejectsNoneAlg(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mw := httpadapter.NewMiddleware("secret", nil, nil, nil, nil, nil)
	r := gin.New()
	r.GET("/me", mw.RequireUser(), func(c *gin.Context) { c.Status(http.StatusOK) })

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{"user_id":%d,"role":"user","exp":%d}`, 1, time.Now().Add(time.Hour).Unix())))
	tokenStr := header + "." + payload + "."

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestMiddleware_RequireUserAPIKeySigned_SuccessAndFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_, repoSQLite := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repoSQLite, "uak_user", "uak_user@example.com", "pass")
	userAPIKeySvc, _ := appuserapikey.NewService(repoSQLite, "test-apikey-key")
	created, err := userAPIKeySvc.Create(t.Context(), user.ID, "ci-key", nil)
	if err != nil {
		t.Fatalf("create user api key: %v", err)
	}

	mw := httpadapter.NewMiddleware("secret", nil, userAPIKeySvc, nil, nil, nil)
	r := gin.New()
	r.POST("/api/v1/open/orders/instant/renew", mw.RequireUserAPIKeySigned(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	body := []byte(`{"x":1}`)
	ts := time.Now().UTC().Format(time.RFC3339)
	unknownAKID := "uak-created-after-invalid-request"
	unknownNonce := "unknown-key-nonce"
	unknownReq := httptest.NewRequest(http.MethodPost, "/api/v1/open/orders/instant/renew", bytes.NewReader(body))
	unknownReq.Header.Set("X-AKID", unknownAKID)
	unknownReq.Header.Set("X-Timestamp", ts)
	unknownReq.Header.Set("X-Nonce", unknownNonce)
	unknownReq.Header.Set("X-Signature", "invalid")
	unknownRec := httptest.NewRecorder()
	r.ServeHTTP(unknownRec, unknownReq)
	if unknownRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unknown api key, got %d", unknownRec.Code)
	}
	laterKey := domain.UserAPIKey{
		UserID:  user.ID,
		Name:    "created-later",
		AKID:    unknownAKID,
		KeyHash: "created-later-secret",
		Status:  domain.APIKeyStatusActive,
	}
	if err := repoSQLite.CreateUserAPIKey(t.Context(), &laterKey); err != nil {
		t.Fatalf("create formerly unknown api key: %v", err)
	}
	laterCanonical := appuserapikey.BuildCanonical(http.MethodPost, "/api/v1/open/orders/instant/renew", "", ts, unknownNonce, body)
	laterReq := httptest.NewRequest(http.MethodPost, "/api/v1/open/orders/instant/renew", bytes.NewReader(body))
	laterReq.Header.Set("X-AKID", unknownAKID)
	laterReq.Header.Set("X-Timestamp", ts)
	laterReq.Header.Set("X-Nonce", unknownNonce)
	laterReq.Header.Set("X-Signature", signLikeService(laterKey.KeyHash, laterCanonical))
	laterRec := httptest.NewRecorder()
	r.ServeHTTP(laterRec, laterReq)
	if laterRec.Code != http.StatusOK {
		t.Fatalf("unknown api key request must not consume nonce, got %d", laterRec.Code)
	}

	nonce := "n1"
	canonical := appuserapikey.BuildCanonical(http.MethodPost, "/api/v1/open/orders/instant/renew", "", ts, nonce, body)
	sig := signLikeService(created.Secret, canonical)

	okReq := httptest.NewRequest(http.MethodPost, "/api/v1/open/orders/instant/renew", bytes.NewReader(body))
	okReq.Header.Set("X-AKID", created.Key.AKID)
	okReq.Header.Set("X-Timestamp", ts)
	okReq.Header.Set("X-Nonce", nonce)
	okReq.Header.Set("X-Signature", sig)
	okRec := httptest.NewRecorder()
	r.ServeHTTP(okRec, okReq)
	if okRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", okRec.Code)
	}
	usedKey, err := repoSQLite.GetUserAPIKeyByAKID(t.Context(), created.Key.AKID)
	if err != nil || usedKey.LastUsedAt == nil {
		t.Fatalf("successful request must update last_used_at: key=%+v err=%v", usedKey, err)
	}
	lastUsedAt := *usedKey.LastUsedAt

	replayReq := httptest.NewRequest(http.MethodPost, "/api/v1/open/orders/instant/renew", bytes.NewReader(body))
	replayReq.Header.Set("X-AKID", created.Key.AKID)
	replayReq.Header.Set("X-Timestamp", ts)
	replayReq.Header.Set("X-Nonce", nonce)
	replayReq.Header.Set("X-Signature", sig)
	replayRec := httptest.NewRecorder()
	r.ServeHTTP(replayRec, replayReq)
	if replayRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for replayed nonce, got %d", replayRec.Code)
	}
	replayedKey, err := repoSQLite.GetUserAPIKeyByAKID(t.Context(), created.Key.AKID)
	if err != nil || replayedKey.LastUsedAt == nil || !replayedKey.LastUsedAt.Equal(lastUsedAt) {
		t.Fatalf("replayed request must not update last_used_at: before=%v after=%v err=%v", lastUsedAt, replayedKey.LastUsedAt, err)
	}

	badReq := httptest.NewRequest(http.MethodPost, "/api/v1/open/orders/instant/renew", bytes.NewReader(body))
	badReq.Header.Set("X-AKID", created.Key.AKID)
	badReq.Header.Set("X-Timestamp", ts)
	badReq.Header.Set("X-Nonce", "n2")
	badReq.Header.Set("X-Signature", "bad")
	badRec := httptest.NewRecorder()
	r.ServeHTTP(badRec, badReq)
	if badRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for bad signature, got %d", badRec.Code)
	}

	retryCanonical := appuserapikey.BuildCanonical(http.MethodPost, "/api/v1/open/orders/instant/renew", "", ts, "n2", body)
	retryReq := httptest.NewRequest(http.MethodPost, "/api/v1/open/orders/instant/renew", bytes.NewReader(body))
	retryReq.Header.Set("X-AKID", created.Key.AKID)
	retryReq.Header.Set("X-Timestamp", ts)
	retryReq.Header.Set("X-Nonce", "n2")
	retryReq.Header.Set("X-Signature", signLikeService(created.Secret, retryCanonical))
	retryRec := httptest.NewRecorder()
	r.ServeHTTP(retryRec, retryReq)
	if retryRec.Code != http.StatusOK {
		t.Fatalf("bad signature must not consume nonce, got %d", retryRec.Code)
	}

	staleReq := httptest.NewRequest(http.MethodPost, "/api/v1/open/orders/instant/renew", bytes.NewReader(body))
	staleReq.Header.Set("X-AKID", created.Key.AKID)
	staleReq.Header.Set("X-Timestamp", time.Now().UTC().Add(-10*time.Minute).Format(time.RFC3339))
	staleReq.Header.Set("X-Nonce", "n3")
	staleReq.Header.Set("X-Signature", sig)
	staleRec := httptest.NewRecorder()
	r.ServeHTTP(staleRec, staleReq)
	if staleRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for stale signature, got %d", staleRec.Code)
	}

	if err := repoSQLite.UpdateUserStatus(t.Context(), user.ID, domain.UserStatusDisabled); err != nil {
		t.Fatalf("disable api key owner: %v", err)
	}
	ownerDisabledTS := time.Now().UTC().Format(time.RFC3339)
	ownerDisabledReq := httptest.NewRequest(http.MethodPost, "/api/v1/open/orders/instant/renew", bytes.NewReader(body))
	ownerDisabledReq.Header.Set("X-AKID", created.Key.AKID)
	ownerDisabledReq.Header.Set("X-Timestamp", ownerDisabledTS)
	ownerDisabledReq.Header.Set("X-Nonce", "n-owner-disabled")
	ownerDisabledCanonical := appuserapikey.BuildCanonical(http.MethodPost, "/api/v1/open/orders/instant/renew", "", ownerDisabledTS, "n-owner-disabled", body)
	ownerDisabledReq.Header.Set("X-Signature", signLikeService(created.Secret, ownerDisabledCanonical))
	ownerDisabledRec := httptest.NewRecorder()
	r.ServeHTTP(ownerDisabledRec, ownerDisabledReq)
	if ownerDisabledRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for disabled api key owner, got %d", ownerDisabledRec.Code)
	}
	if err := repoSQLite.UpdateUserStatus(t.Context(), user.ID, domain.UserStatusActive); err != nil {
		t.Fatalf("reactivate api key owner: %v", err)
	}

	if err := repoSQLite.UpdateUserAPIKeyStatus(t.Context(), user.ID, created.Key.ID, domain.APIKeyStatusDisabled); err != nil {
		t.Fatalf("disable key: %v", err)
	}
	disabledReq := httptest.NewRequest(http.MethodPost, "/api/v1/open/orders/instant/renew", bytes.NewReader(body))
	disabledReq.Header.Set("X-AKID", created.Key.AKID)
	disabledReq.Header.Set("X-Timestamp", time.Now().UTC().Format(time.RFC3339))
	disabledReq.Header.Set("X-Nonce", "n4")
	disabledCanonical := appuserapikey.BuildCanonical(http.MethodPost, "/api/v1/open/orders/instant/renew", "", disabledReq.Header.Get("X-Timestamp"), "n4", body)
	disabledReq.Header.Set("X-Signature", signLikeService(created.Secret, disabledCanonical))
	disabledRec := httptest.NewRecorder()
	r.ServeHTTP(disabledRec, disabledReq)
	if disabledRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for disabled key, got %d", disabledRec.Code)
	}
}

func TestMiddleware_RequireUserAPIKeySigned_RejectsOversizedBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_, repoSQLite := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repoSQLite, "uak_body_user", "uak_body_user@example.com", "pass")
	userAPIKeySvc, _ := appuserapikey.NewService(repoSQLite, "test-apikey-key")
	created, err := userAPIKeySvc.Create(t.Context(), user.ID, "body-key", nil)
	if err != nil {
		t.Fatalf("create user api key: %v", err)
	}

	mw := httpadapter.NewMiddleware("secret", nil, userAPIKeySvc, nil, nil, nil)
	r := gin.New()
	r.POST("/api/v1/open/orders/instant/renew", mw.RequireUserAPIKeySigned(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	body := bytes.Repeat([]byte("a"), (1<<20)+1)
	ts := time.Now().UTC().Format(time.RFC3339)
	nonce := "oversized-body"
	canonical := appuserapikey.BuildCanonical(http.MethodPost, "/api/v1/open/orders/instant/renew", "", ts, nonce, body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/open/orders/instant/renew", bytes.NewReader(body))
	req.Header.Set("X-AKID", created.Key.AKID)
	req.Header.Set("X-Timestamp", ts)
	req.Header.Set("X-Nonce", nonce)
	req.Header.Set("X-Signature", signLikeService(created.Secret, canonical))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413 for oversized signed body, got %d", rec.Code)
	}
}

func signLikeService(secret, canonical string) string {
	return fmt.Sprintf("%x", hmacSHA256([]byte(secret), []byte(canonical)))
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	_, _ = h.Write(data)
	return h.Sum(nil)
}

// TestMiddleware_RequireUserAPIKeySigned_EnforcesScope verifies P0-7: a signed
// request is rejected when the key lacks the route's required scope, even with
// a valid signature.
func TestMiddleware_RequireUserAPIKeySigned_EnforcesScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_, repoSQLite := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repoSQLite, "uak_scope_user", "uak_scope_user@example.com", "pass")
	userAPIKeySvc, _ := appuserapikey.NewService(repoSQLite, "test-apikey-key")

	// Read-only key: only vps:read is granted.
	created, err := userAPIKeySvc.Create(t.Context(), user.ID, "readonly", []string{appuserapikey.ScopeVPSRead})
	if err != nil {
		t.Fatalf("create read-only key: %v", err)
	}

	mw := httpadapter.NewMiddleware("secret", nil, userAPIKeySvc, nil, nil, nil)
	r := gin.New()
	r.GET("/api/v1/open/vps/:id", mw.RequireUserAPIKeySigned(), func(c *gin.Context) { c.Status(http.StatusOK) })
	r.POST("/api/v1/open/vps/:id/reboot", mw.RequireUserAPIKeySigned(), func(c *gin.Context) { c.Status(http.StatusOK) })

	body := []byte(`{}`)
	ts := time.Now().UTC().Format(time.RFC3339)

	// GET /vps/:id needs vps:read -> allowed.
	getCanonical := appuserapikey.BuildCanonical(http.MethodGet, "/api/v1/open/vps/1", "", ts, "scope-get", body)
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/open/vps/1", bytes.NewReader(body))
	getReq.Header.Set("X-AKID", created.Key.AKID)
	getReq.Header.Set("X-Timestamp", ts)
	getReq.Header.Set("X-Nonce", "scope-get")
	getReq.Header.Set("X-Signature", signLikeService(created.Secret, getCanonical))
	getRec := httptest.NewRecorder()
	r.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET with vps:read scope expected 200, got %d", getRec.Code)
	}

	// POST /vps/:id/reboot needs vps:write -> denied.
	rebootCanonical := appuserapikey.BuildCanonical(http.MethodPost, "/api/v1/open/vps/1/reboot", "", ts, "scope-reboot", body)
	rebootReq := httptest.NewRequest(http.MethodPost, "/api/v1/open/vps/1/reboot", bytes.NewReader(body))
	rebootReq.Header.Set("X-AKID", created.Key.AKID)
	rebootReq.Header.Set("X-Timestamp", ts)
	rebootReq.Header.Set("X-Nonce", "scope-reboot")
	rebootReq.Header.Set("X-Signature", signLikeService(created.Secret, rebootCanonical))
	rebootRec := httptest.NewRecorder()
	r.ServeHTTP(rebootRec, rebootReq)
	if rebootRec.Code != http.StatusForbidden {
		t.Fatalf("POST with only vps:read scope expected 403, got %d", rebootRec.Code)
	}
}
