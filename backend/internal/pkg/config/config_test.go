package config

import (
	"os"
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

func TestGetEnvOverride(t *testing.T) {
	const key = "APP_ADDR"
	_ = os.Setenv(key, "127.0.0.1:9000")
	defer os.Unsetenv(key)
	if v := getEnv(key, ":8080"); v != "127.0.0.1:9000" {
		t.Fatalf("expected env override")
	}
}
