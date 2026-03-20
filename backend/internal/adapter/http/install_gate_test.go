package http_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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
		if got := rec.Header().Get("Location"); got != "/install/" {
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

func TestInstallGate_ServesInstallerFromAdminStatic(t *testing.T) {
	lockDir := t.TempDir()
	lockPath := filepath.Join(lockDir, "install.lock")

	adapterhttp.SetInstallLockPathForTest(lockPath)
	t.Cleanup(func() { adapterhttp.SetInstallLockPathForTest("") })

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	if err := os.MkdirAll(filepath.Join("static-admin", "assets"), 0o755); err != nil {
		t.Fatalf("mkdir admin static: %v", err)
	}
	if err := os.WriteFile(filepath.Join("static-admin", "index.html"), []byte("INSTALL_ADMIN_INDEX"), 0o644); err != nil {
		t.Fatalf("write admin index: %v", err)
	}
	if err := os.WriteFile(filepath.Join("static-admin", "assets", "install.js"), []byte("INSTALL_ASSET_OK"), 0o644); err != nil {
		t.Fatalf("write installer asset: %v", err)
	}

	env := testutilhttp.NewTestEnv(t, false)

	{
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/install", nil)
		env.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusTemporaryRedirect {
			t.Fatalf("expected 307, got %d", rec.Code)
		}
		if got := rec.Header().Get("Location"); got != "/install/" {
			t.Fatalf("unexpected location: %q", got)
		}
	}

	{
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/install/", nil)
		env.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "INSTALL_ADMIN_INDEX") {
			t.Fatalf("expected installer index body, got: %q", rec.Body.String())
		}
	}

	{
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/install/assets/install.js", nil)
		env.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if strings.TrimSpace(rec.Body.String()) != "INSTALL_ASSET_OK" {
			t.Fatalf("unexpected installer asset body: %q", rec.Body.String())
		}
	}
}
