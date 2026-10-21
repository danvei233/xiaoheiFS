package testutilhttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"xiaoheiplay/internal/domain"
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
	if err := os.WriteFile(filepath.Join("static", "assets", "index-abc123.js"), []byte("HASHED_OK"), 0o644); err != nil {
		t.Fatalf("write hashed asset: %v", err)
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
		if cacheControl := rec.Header().Get("Cache-Control"); cacheControl != "no-store" {
			t.Fatalf("expected index cache header no-store, got %q", cacheControl)
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
		if cacheControl := rec.Header().Get("Cache-Control"); cacheControl != "no-cache" {
			t.Fatalf("expected plain asset cache header no-cache, got %q", cacheControl)
		}
	}

	// Hashed asset should be long-cached.
	{
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/assets/index-abc123.js", nil)
		env.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if strings.TrimSpace(rec.Body.String()) != "HASHED_OK" {
			t.Fatalf("unexpected hashed asset body: %q", rec.Body.String())
		}
		if cacheControl := rec.Header().Get("Cache-Control"); cacheControl != "public, max-age=31536000, immutable" {
			t.Fatalf("expected hashed asset cache header immutable, got %q", cacheControl)
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

func TestSPAServesAdminIndexForConfiguredAdminPath(t *testing.T) {
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
		t.Fatalf("mkdir public static: %v", err)
	}
	if err := os.MkdirAll(filepath.Join("static-admin", "assets"), 0o755); err != nil {
		t.Fatalf("mkdir admin static: %v", err)
	}
	if err := os.WriteFile(filepath.Join("static", "index.html"), []byte("PUBLIC_INDEX_OK"), 0o644); err != nil {
		t.Fatalf("write public index: %v", err)
	}
	if err := os.WriteFile(filepath.Join("static-admin", "index.html"), []byte("ADMIN_INDEX_OK"), 0o644); err != nil {
		t.Fatalf("write admin index: %v", err)
	}
	if err := os.WriteFile(filepath.Join("static", "assets", "public.txt"), []byte("PUBLIC_ASSET_OK"), 0o644); err != nil {
		t.Fatalf("write public asset: %v", err)
	}
	if err := os.WriteFile(filepath.Join("static-admin", "assets", "admin.txt"), []byte("ADMIN_ASSET_OK"), 0o644); err != nil {
		t.Fatalf("write admin asset: %v", err)
	}
	if err := os.WriteFile(filepath.Join("static-admin", "assets", "index-abc123.js"), []byte("ADMIN_HASHED_OK"), 0o644); err != nil {
		t.Fatalf("write admin hashed asset: %v", err)
	}

	env := NewTestEnv(t, false)
	if err := env.Repo.UpsertSetting(context.Background(), domain.Setting{
		Key:       "admin_path",
		ValueJSON: "Secret123",
	}); err != nil {
		t.Fatalf("seed admin_path: %v", err)
	}

	{
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/products", nil)
		env.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("public route expected 200, got %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "PUBLIC_INDEX_OK") {
			t.Fatalf("expected public index body, got: %q", rec.Body.String())
		}
	}

	{
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/Secret123", nil)
		env.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusTemporaryRedirect {
			t.Fatalf("admin root redirect expected 307, got %d", rec.Code)
		}
		if location := rec.Header().Get("Location"); location != "/Secret123/" {
			t.Fatalf("unexpected admin root redirect: %q", location)
		}
	}

	{
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/Secret123/", nil)
		env.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("admin canonical root expected 200, got %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "ADMIN_INDEX_OK") {
			t.Fatalf("expected admin root body, got: %q", rec.Body.String())
		}
		if cacheControl := rec.Header().Get("Cache-Control"); cacheControl != "no-store" {
			t.Fatalf("expected admin index cache header no-store, got %q", cacheControl)
		}
	}

	{
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/Secret123/dashboard/console?id=7", nil)
		env.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusTemporaryRedirect {
			t.Fatalf("admin history redirect expected 307, got %d", rec.Code)
		}
		if location := rec.Header().Get("Location"); location != "/Secret123/#/dashboard/console?id=7" {
			t.Fatalf("unexpected admin history redirect: %q", location)
		}
	}

	{
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/Secret123/assets/admin.txt", nil)
		env.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("admin asset expected 200, got %d", rec.Code)
		}
		if strings.TrimSpace(rec.Body.String()) != "ADMIN_ASSET_OK" {
			t.Fatalf("unexpected admin asset body: %q", rec.Body.String())
		}
		if cacheControl := rec.Header().Get("Cache-Control"); cacheControl != "no-cache" {
			t.Fatalf("expected admin plain asset cache header no-cache, got %q", cacheControl)
		}
	}

	{
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/Secret123/assets/index-abc123.js", nil)
		env.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("admin hashed asset expected 200, got %d", rec.Code)
		}
		if strings.TrimSpace(rec.Body.String()) != "ADMIN_HASHED_OK" {
			t.Fatalf("unexpected admin hashed asset body: %q", rec.Body.String())
		}
		if cacheControl := rec.Header().Get("Cache-Control"); cacheControl != "public, max-age=31536000, immutable" {
			t.Fatalf("expected admin hashed asset cache header immutable, got %q", cacheControl)
		}
	}

	{
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/assets/public.txt", nil)
		env.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("public asset expected 200, got %d", rec.Code)
		}
		if strings.TrimSpace(rec.Body.String()) != "PUBLIC_ASSET_OK" {
			t.Fatalf("unexpected public asset body: %q", rec.Body.String())
		}
	}
}
