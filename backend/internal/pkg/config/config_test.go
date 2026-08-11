package config

import (
	"bytes"
	"log"
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

// TestLoadPersistsSecretsToLoadedConfigPath guards against the CWD/shadow-config
// bug: generated secrets must be written back to the config file that was
// actually loaded (which may live outside the process CWD), otherwise every
// restart rotates JWTSecret/PluginMasterKey and invalidates JWTs / plugin data.
func TestLoadPersistsSecretsToLoadedConfigPath(t *testing.T) {
	outer := t.TempDir()
	inner := filepath.Join(outer, "cwd")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatalf("mkdir cwd: %v", err)
	}
	oldWD, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(oldWD) })
	_ = os.Chdir(inner)

	t.Setenv("APP_JWT_SECRET", "")
	t.Setenv("APP_PLUGIN_MASTER_KEY", "")

	cfgPath := filepath.Join(outer, localConfigYAML)
	b := []byte("db:\n  type: sqlite\n  path: ./data/test.db\n")
	if err := os.WriteFile(cfgPath, b, 0o600); err != nil {
		t.Fatalf("write parent config: %v", err)
	}

	cfg1 := Load()
	if cfg1.JWTSecret == "" || cfg1.PluginMasterKey == "" {
		t.Fatalf("expected generated secrets")
	}

	// Secrets must be persisted to the loaded (parent) config file.
	got, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("expected secrets written to loaded config: %v", err)
	}
	if !strings.Contains(string(got), "jwt_secret") || !strings.Contains(string(got), "plugin_master_key") {
		t.Fatalf("expected both secrets persisted to loaded config, got:\n%s", got)
	}
	// CWD must not contain a shadow config that would diverge on next restart.
	if _, err := os.Stat(filepath.Join(inner, localConfigYAML)); err == nil {
		t.Fatalf("unexpected shadow config written to CWD")
	}

	cfg2 := Load()
	if cfg2.JWTSecret != cfg1.JWTSecret || cfg2.PluginMasterKey != cfg1.PluginMasterKey {
		t.Fatalf("expected stable secrets across loads (got %q vs %q)", cfg1.JWTSecret, cfg2.JWTSecret)
	}
}

// TestLoadWarnsOnUnparseableConfig ensures a type error in the config file is not
// silently swallowed (which previously made the whole file ignored with no log).
func TestLoadWarnsOnUnparseableConfig(t *testing.T) {
	td := t.TempDir()
	oldWD, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(oldWD) })
	_ = os.Chdir(td)

	t.Setenv("APP_DB_TYPE", "")
	t.Setenv("APP_DB_PATH", "")
	t.Setenv("APP_DB_DSN", "")

	// `db` as a scalar is a YAML type error for the struct field and makes the
	// whole file unparseable.
	b := []byte("addr: \":8080\"\ndb: 8080\n")
	if err := os.WriteFile(filepath.Join(td, localConfigYAML), b, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	stderr := captureStderr(t, func() { Load() })
	if !strings.Contains(stderr, "unparseable") {
		t.Fatalf("expected a config parse warning on stderr, got: %q", stderr)
	}
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := log.Writer()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	log.SetOutput(w)
	defer log.SetOutput(old)
	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		_, _ = buf.ReadFrom(r)
		close(done)
	}()
	fn()
	_ = w.Close()
	<-done
	return buf.String()
}
