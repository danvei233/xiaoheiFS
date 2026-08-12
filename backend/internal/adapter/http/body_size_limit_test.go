package http_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/testutilhttp"
)

// chunkedBody returns a reader that does not expose its length, so
// httptest.NewRequest reports Content-Length -1 (chunked/unknown-length body),
// exercising the MaxBytesReader path rather than the middleware pre-check.
func chunkedBody(data []byte) io.Reader {
	return io.NopCloser(bytes.NewReader(data))
}

// TestOversizedBodyRejected413WithContentLength verifies the middleware rejects
// a request whose declared Content-Length exceeds the global cap before any
// handler reads the body.
func TestOversizedBodyRejected413WithContentLength(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	body := bytes.Repeat([]byte("a"), (32<<20)+1)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	env.Router.ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413 for oversized declared body, got %d (%s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), domain.ErrRequestBodyTooLarge.Error()) {
		t.Fatalf("expected controlled error, got %s", rec.Body.String())
	}
}

// TestOversizedBodyRejected413Chunked verifies a chunked/unknown-length body
// that exceeds the global cap is rejected with a stable 413 by the bind path
// (MaxBytesReader -> ErrRequestBodyTooLarge -> writeBodyError).
func TestOversizedBodyRejected413Chunked(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	// A valid JSON string value longer than the cap: the decoder reads past the
	// limit and surfaces the MaxBytesError (not a JSON syntax error).
	body := []byte(`{"password":"` + strings.Repeat("a", (32<<20)+1) + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", chunkedBody(body))
	req.Header.Set("Content-Type", "application/json")
	if req.ContentLength != -1 {
		t.Fatalf("expected chunked request (Content-Length -1), got %d", req.ContentLength)
	}
	rec := httptest.NewRecorder()
	env.Router.ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413 for oversized chunked body, got %d (%s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), domain.ErrRequestBodyTooLarge.Error()) {
		t.Fatalf("expected controlled error, got %s", rec.Body.String())
	}
}

// TestNormalSizedBodyNotRejected verifies a reasonable JSON body still reaches
// the handler (the login handler returns its own error for bad credentials, not
// a body-related 413).
func TestNormalSizedBodyNotRejected(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"account":  "nobody",
		"password": "wrong",
	}, "")
	if rec.Code == http.StatusRequestEntityTooLarge {
		t.Fatalf("normal body must not be rejected as too large")
	}
}

// newRobotWebhookRouter registers the RobotWebhook handler on a bare router so
// the handler's own body cap is exercised without the API-key middleware that
// guards the production route.
func newRobotWebhookRouter(t *testing.T) (*gin.Engine, *testutilhttp.Env) {
	t.Helper()
	env := testutilhttp.NewTestEnv(t, false)
	r := gin.New()
	r.POST("/robot/webhook", env.Handler.RobotWebhook)
	return r, env
}

// TestRobotWebhookOversizedBody413 verifies the public robot webhook has its own
// tighter cap and returns 413 (not 400) on overflow.
func TestRobotWebhookOversizedBody413(t *testing.T) {
	r, _ := newRobotWebhookRouter(t)
	body := bytes.Repeat([]byte("x"), 70<<10)
	req := httptest.NewRequest(http.MethodPost, "/robot/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413 for oversized robot webhook, got %d (%s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), domain.ErrRequestBodyTooLarge.Error()) {
		t.Fatalf("expected controlled error, got %s", rec.Body.String())
	}
}

// TestRobotWebhookNormalBodyNot413 verifies a small webhook body is not rejected
// as too large (the webhook may still fail authorisation checks, but never with
// 413).
func TestRobotWebhookNormalBodyNot413(t *testing.T) {
	r, _ := newRobotWebhookRouter(t)
	body := []byte(`{"text":"ok","sender":"me"}`)
	req := httptest.NewRequest(http.MethodPost, "/robot/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code == http.StatusRequestEntityTooLarge {
		t.Fatalf("normal webhook body must not be rejected as too large")
	}
}
