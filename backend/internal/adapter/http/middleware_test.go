package http_test

import (
	"encoding/base64"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	httpadapter "xiaoheiplay/internal/adapter/http"
)

func TestMiddleware_RequireUser_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mw := httpadapter.NewMiddleware("secret", nil, nil, nil, nil)
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
	mw := httpadapter.NewMiddleware("secret", nil, nil, nil, nil)
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
	mw := httpadapter.NewMiddleware("secret", nil, nil, nil, nil)
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
