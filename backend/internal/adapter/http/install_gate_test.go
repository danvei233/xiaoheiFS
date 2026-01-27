package http_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"xiaoheiplay/internal/testutilhttp"
)

func TestInstallGate_RedirectsBeforeInstalled(t *testing.T) {
	lockDir := t.TempDir()
	lockPath := filepath.Join(lockDir, "install.lock")

	prev := os.Getenv("APP_INSTALL_LOCK_PATH")
	_ = os.Setenv("APP_INSTALL_LOCK_PATH", lockPath)
	t.Cleanup(func() {
		if prev == "" {
			_ = os.Unsetenv("APP_INSTALL_LOCK_PATH")
		} else {
			_ = os.Setenv("APP_INSTALL_LOCK_PATH", prev)
		}
	})

	env := testutilhttp.NewTestEnv(t, false)

	// Site traffic should redirect to /install.
	{
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		env.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusFound {
			t.Fatalf("expected 302, got %d", rec.Code)
		}
		if got := rec.Header().Get("Location"); got != "/install" {
			t.Fatalf("unexpected location: %q", got)
		}
	}

	// Non-install API calls should be blocked with 503.
	{
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/catalog", nil)
		env.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected 503, got %d", rec.Code)
		}
	}
}
