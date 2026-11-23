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

// 回归：配置文件不在进程当前目录时，自动生成的 jwt_secret 必须写回「实际加载的那个文件」，
// 否则每次重启都会重新生成密钥，导致所有已登录用户被踢下线。
func TestLoadPersistsJWTSecretIntoLoadedConfigWhenCWDDiffers(t *testing.T) {
	td := t.TempDir()
	runDir := filepath.Join(td, "run")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run dir: %v", err)
	}

	// 配置文件里有 DB 配置但没有 jwt_secret。
	cfgPath := filepath.Join(td, localConfigYAML)
	if err := os.WriteFile(cfgPath, []byte("db:\n  type: sqlite\n  path: ./data/test.db\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	oldWD, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(oldWD) })
	// 工作目录是配置目录的子目录，readLocalConfig 会通过 "../app.config.yaml" 命中。
	if err := os.Chdir(runDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	cfg1 := Load()
	if strings.TrimSpace(cfg1.JWTSecret) == "" {
		t.Fatalf("expected jwt secret generated")
	}

	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(raw), "jwt_secret") {
		t.Fatalf("expected jwt_secret persisted into loaded config file, got:\n%s", string(raw))
	}
	// 回写密钥时必须保留原有配置（DB 段等），否则重启后会连不上数据库。
	if !strings.Contains(string(raw), "sqlite") {
		t.Fatalf("expected existing db config preserved, got:\n%s", string(raw))
	}
	if _, err := os.Stat(filepath.Join(runDir, localConfigYAML)); !os.IsNotExist(err) {
		t.Fatalf("must not create a second config file in cwd")
	}

	cfg2 := Load()
	if cfg2.JWTSecret != cfg1.JWTSecret {
		t.Fatalf("expected jwt secret stable across restarts, got %q then %q", cfg1.JWTSecret, cfg2.JWTSecret)
	}
	if cfg2.PluginMasterKey != cfg1.PluginMasterKey {
		t.Fatalf("expected plugin master key stable across restarts")
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
