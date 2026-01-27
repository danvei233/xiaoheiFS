package testutilhttp

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSPAServesIndexForHistoryRoutes(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	if err := os.MkdirAll(filepath.Join("static", "assets"), 0o755); err != nil {
		t.Fatalf("mkdir static: %v", err)
	}
	if err := os.WriteFile(filepath.Join("static", "index.html"), []byte("INDEX_OK"), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	if err := os.WriteFile(filepath.Join("static", "assets", "hello.txt"), []byte("HELLO_OK"), 0o644); err != nil {
		t.Fatalf("write asset: %v", err)
	}

	env := NewTestEnv(t, false)

	// History-mode SPA route should fall back to index.html.
	{
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/console/orders/123", nil)
		env.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "INDEX_OK") {
			t.Fatalf("expected index.html body, got: %q", rec.Body.String())
		}
	}

	// Static asset should be served directly.
	{
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/assets/hello.txt", nil)
		env.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if strings.TrimSpace(rec.Body.String()) != "HELLO_OK" {
			t.Fatalf("unexpected asset body: %q", rec.Body.String())
		}
	}

	// API unknown routes should not fall back to index.html.
	{
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/does-not-exist", nil)
		env.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
		if strings.Contains(rec.Body.String(), "INDEX_OK") {
			t.Fatalf("api 404 should not return index.html")
		}
	}
}
