package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	apphttp "xiaoheiplay/internal/adapter/http"
)

// TestTrustedProxiesNilByDefault verifies P0-1: when the operator has not
// configured a trusted proxy list, the gin engine must trust no proxies.
// Otherwise a client can spoof X-Forwarded-For and rotate every IP-keyed rate
// limiter / cooldown bucket (login, admin login, password reset, captcha, ...).
func TestTrustedProxiesNilByDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// This mirrors how NewServer configures the engine; keep in sync.
	r := gin.New()
	if err := apphttp.ConfigureTrustedProxies(r); err != nil {
		t.Fatalf("ConfigureTrustedProxies: %v", err)
	}
	r.GET("/echo", func(c *gin.Context) {
		c.String(http.StatusOK, c.ClientIP())
	})

	req := httptest.NewRequest(http.MethodGet, "/echo", nil)
	// Direct peer comes from RemoteAddr; XFF is a spoofed, attacker-controlled
	// header.
	req.RemoteAddr = "203.0.113.9:45678"
	req.Header.Set("X-Forwarded-For", "198.51.100.7, 10.0.0.1")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	got := rec.Body.String()
	if got != "203.0.113.9" {
		t.Fatalf("ClientIP trusted spoofed X-Forwarded-For: got %q, want direct peer 203.0.113.9", got)
	}
}

// TestConfigureTrustedProxiesRejectsInvalidInput verifies the configuration
// helper fails closed: invalid CIDRs must not leave the engine trusting all
// proxies.
func TestConfigureTrustedProxiesRejectsInvalidInput(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	if err := apphttp.ConfigureTrustedProxiesWith(r, []string{"not-a-cidr", "10.0.0.0/8"}); err == nil {
		t.Fatal("expected error for invalid CIDR input")
	}
	// After a failed configure the engine must still be safe (trust none). We
	// verify by routing a request with a spoofed header and checking ClientIP.
	r.GET("/echo", func(c *gin.Context) {
		c.String(http.StatusOK, c.ClientIP())
	})
	req := httptest.NewRequest(http.MethodGet, "/echo", nil)
	req.RemoteAddr = "203.0.113.11:5000"
	req.Header.Set("X-Forwarded-For", "192.0.2.66")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if got := rec.Body.String(); got != "203.0.113.11" {
		t.Fatalf("failed configure left engine trusting proxies: got %q", got)
	}
}

// TestConfigureTrustedProxiesHonorsExplicitList verifies operators can opt in
// to a specific proxy CIDR (e.g. their reverse proxy) and only that range is
// trusted.
func TestConfigureTrustedProxiesHonorsExplicitList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	if err := apphttp.ConfigureTrustedProxiesWith(r, []string{"10.0.0.0/8"}); err != nil {
		t.Fatalf("ConfigureTrustedProxiesWith: %v", err)
	}
	r.GET("/echo", func(c *gin.Context) {
		c.String(http.StatusOK, c.ClientIP())
	})

	// A request coming from the trusted proxy range may carry XFF.
	req := httptest.NewRequest(http.MethodGet, "/echo", nil)
	req.RemoteAddr = "10.1.2.3:5000"
	req.Header.Set("X-Forwarded-For", "198.51.100.7")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if got := rec.Body.String(); got != "198.51.100.7" {
		t.Fatalf("trusted proxy XFF not honored: got %q", got)
	}

	// A request from a non-trusted peer must ignore XFF.
	req2 := httptest.NewRequest(http.MethodGet, "/echo", nil)
	req2.RemoteAddr = "203.0.113.20:5000"
	req2.Header.Set("X-Forwarded-For", "192.0.2.66")
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)
	if got := rec2.Body.String(); got != "203.0.113.20" {
		t.Fatalf("non-trusted peer accepted XFF: got %q", got)
	}
}
