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
	userAPIKeySvc := appuserapikey.NewService(repoSQLite)
	created, err := userAPIKeySvc.Create(t.Context(), user.ID, "ci-key", nil)
	if err != nil {
		t.Fatalf("create user api key: %v", err)
	}

	mw := httpadapter.NewMiddleware("secret", nil, userAPIKeySvc, nil, nil, nil)
	r := gin.New()
	r.POST("/open/test", mw.RequireUserAPIKeySigned(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	body := []byte(`{"x":1}`)
	ts := time.Now().UTC().Format(time.RFC3339)
	nonce := "n1"
	canonical := appuserapikey.BuildCanonical(http.MethodPost, "/open/test", "", ts, nonce, body)
	sig := signLikeService(created.Secret, canonical)

	okReq := httptest.NewRequest(http.MethodPost, "/open/test", bytes.NewReader(body))
	okReq.Header.Set("X-AKID", created.Key.AKID)
	okReq.Header.Set("X-Timestamp", ts)
	okReq.Header.Set("X-Nonce", nonce)
	okReq.Header.Set("X-Signature", sig)
	okRec := httptest.NewRecorder()
	r.ServeHTTP(okRec, okReq)
	if okRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", okRec.Code)
	}

	badReq := httptest.NewRequest(http.MethodPost, "/open/test", bytes.NewReader(body))
	badReq.Header.Set("X-AKID", created.Key.AKID)
	badReq.Header.Set("X-Timestamp", ts)
	badReq.Header.Set("X-Nonce", "n2")
	badReq.Header.Set("X-Signature", "bad")
	badRec := httptest.NewRecorder()
	r.ServeHTTP(badRec, badReq)
	if badRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for bad signature, got %d", badRec.Code)
	}

	staleReq := httptest.NewRequest(http.MethodPost, "/open/test", bytes.NewReader(body))
	staleReq.Header.Set("X-AKID", created.Key.AKID)
	staleReq.Header.Set("X-Timestamp", time.Now().UTC().Add(-10*time.Minute).Format(time.RFC3339))
	staleReq.Header.Set("X-Nonce", "n3")
	staleReq.Header.Set("X-Signature", sig)
	staleRec := httptest.NewRecorder()
	r.ServeHTTP(staleRec, staleReq)
	if staleRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for stale signature, got %d", staleRec.Code)
	}

	if err := repoSQLite.UpdateUserAPIKeyStatus(t.Context(), user.ID, created.Key.ID, domain.APIKeyStatusDisabled); err != nil {
		t.Fatalf("disable key: %v", err)
	}
	disabledReq := httptest.NewRequest(http.MethodPost, "/open/test", bytes.NewReader(body))
	disabledReq.Header.Set("X-AKID", created.Key.AKID)
	disabledReq.Header.Set("X-Timestamp", time.Now().UTC().Format(time.RFC3339))
	disabledReq.Header.Set("X-Nonce", "n4")
	disabledCanonical := appuserapikey.BuildCanonical(http.MethodPost, "/open/test", "", disabledReq.Header.Get("X-Timestamp"), "n4", body)
	disabledReq.Header.Set("X-Signature", signLikeService(created.Secret, disabledCanonical))
	disabledRec := httptest.NewRecorder()
	r.ServeHTTP(disabledRec, disabledReq)
	if disabledRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for disabled key, got %d", disabledRec.Code)
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
