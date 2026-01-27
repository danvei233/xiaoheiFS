package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigDefaults(t *testing.T) {
	_ = os.Unsetenv("APP_ADDR")
	_ = os.Unsetenv("APP_DB_PATH")
	cfg := Load()
	if cfg.Addr == "" || cfg.DBPath == "" {
		t.Fatalf("expected defaults")
	}
}

func TestLoadFromYAMLFile(t *testing.T) {
	td := t.TempDir()
	oldWD, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(oldWD) })
	_ = os.Chdir(td)

	// Ensure env doesn't mask file values.
	_ = os.Unsetenv("APP_ADDR")
	_ = os.Unsetenv("APP_DB_TYPE")
	_ = os.Unsetenv("APP_DB_PATH")
	_ = os.Unsetenv("APP_DB_DSN")
	_ = os.Unsetenv("SITE_URL")

	b := []byte("addr: \":9999\"\ndb:\n  type: sqlite\n  path: ./data/test.db\nsite:\n  url: https://example.com\n")
	if err := os.WriteFile(filepath.Join(td, localConfigYAML), b, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg := Load()
	if cfg.Addr != ":9999" {
		t.Fatalf("expected addr from yaml, got %q", cfg.Addr)
	}
	if cfg.DBPath != "./data/test.db" {
		t.Fatalf("expected db path from yaml, got %q", cfg.DBPath)
	}
	if cfg.SiteURL != "https://example.com" {
		t.Fatalf("expected site url from yaml, got %q", cfg.SiteURL)
	}
}

func TestGetEnvOverride(t *testing.T) {
	const key = "APP_ADDR"
	_ = os.Setenv(key, "127.0.0.1:9000")
	defer os.Unsetenv(key)
	if v := getEnv(key, ":8080"); v != "127.0.0.1:9000" {
		t.Fatalf("expected env override")
	}
}
