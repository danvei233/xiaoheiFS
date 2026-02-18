package http_test

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	adapterhttp "xiaoheiplay/internal/adapter/http"
	"xiaoheiplay/internal/testutilhttp"
)

func TestInstallGate_RedirectsBeforeInstalled(t *testing.T) {
	lockDir := t.TempDir()
	lockPath := filepath.Join(lockDir, "install.lock")

	adapterhttp.SetInstallLockPathForTest(lockPath)
	t.Cleanup(func() { adapterhttp.SetInstallLockPathForTest("") })

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
