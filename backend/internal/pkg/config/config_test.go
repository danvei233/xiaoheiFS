package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigDefaults(t *testing.T) {
	cfg := Load()
	if cfg.Addr == "" {
		t.Fatalf("expected defaults")
	}
	if strings.TrimSpace(cfg.DBType) != "" || strings.TrimSpace(cfg.DBPath) != "" {
		t.Fatalf("expected db defaults to be empty, got type=%q path=%q", cfg.DBType, cfg.DBPath)
	}
}

func TestLoadFromYAMLFile(t *testing.T) {
	td := t.TempDir()
	oldWD, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(oldWD) })
	_ = os.Chdir(td)

	b := []byte("addr: \":9999\"\ndb:\n  type: sqlite\n  path: ./data/test.db\njwt_secret: \"fixed\"\n")
	if err := os.WriteFile(filepath.Join(td, localConfigYAML), b, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg := Load()
	if cfg.Addr != ":9999" {
		t.Fatalf("expected addr from yaml, got %q", cfg.Addr)
	}
	wantDBPath := filepath.Join(td, "data", "test.db")
	if cfg.DBPath != wantDBPath {
		t.Fatalf("expected db path from yaml, got %q", cfg.DBPath)
	}
	if cfg.JWTSecret != "fixed" {
		t.Fatalf("expected jwt secret from yaml, got %q", cfg.JWTSecret)
	}
}

func TestLoadGeneratesJWTSecretAndPersists(t *testing.T) {
	td := t.TempDir()
	oldWD, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(oldWD) })
	_ = os.Chdir(td)

	cfg1 := Load()
	if strings.TrimSpace(cfg1.JWTSecret) == "" {
		t.Fatalf("expected jwt secret to be generated")
	}
	b, err := os.ReadFile(filepath.Join(td, localConfigYAML))
	if err != nil {
		t.Fatalf("expected config file written: %v", err)
	}
	if !strings.Contains(string(b), "jwt_secret") {
		t.Fatalf("expected jwt_secret persisted")
	}

	cfg2 := Load()
	if cfg2.JWTSecret != cfg1.JWTSecret {
		t.Fatalf("expected jwt secret to be stable across loads")
	}
}

func TestLoadEnvOverridesDBConfig(t *testing.T) {
	td := t.TempDir()
	oldWD, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(oldWD) })
	_ = os.Chdir(td)

	b := []byte("db:\n  type: sqlite\n  path: ./data/test.db\n")
	if err := os.WriteFile(filepath.Join(td, localConfigYAML), b, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	t.Setenv("APP_DB_TYPE", "mysql")
	t.Setenv("APP_DB_DSN", "root:pass@tcp(mysql:3306)/xiaohei")

	cfg := Load()
	if cfg.DBType != "mysql" {
		t.Fatalf("expected env db type override, got %q", cfg.DBType)
	}
	if cfg.DBDSN != "root:pass@tcp(mysql:3306)/xiaohei" {
		t.Fatalf("expected env db dsn override, got %q", cfg.DBDSN)
	}
}
