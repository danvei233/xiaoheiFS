package db

import (
	"path/filepath"
	"strings"
	"testing"

	"xiaoheiplay/internal/pkg/config"
)

func TestOpenSQLite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data", "app.db")
	conn, err := Open(config.Config{DBType: "sqlite", DBPath: path})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := conn.SQL.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func TestOpenMissingDBType(t *testing.T) {
	_, err := Open(config.Config{})
	if err == nil || !strings.Contains(err.Error(), "missing APP_DB_TYPE") {
		t.Fatalf("expected missing APP_DB_TYPE error, got %v", err)
	}
}

func TestOpenSQLiteMissingDBPath(t *testing.T) {
	_, err := Open(config.Config{DBType: "sqlite"})
	if err == nil || !strings.Contains(err.Error(), "missing APP_DB_PATH for sqlite") {
		t.Fatalf("expected missing APP_DB_PATH for sqlite, got %v", err)
	}
}

func TestNormalizeMySQLDSN_AddsCompatibilityDefaults(t *testing.T) {
	dsn := normalizeMySQLDSN("root:pass@tcp(127.0.0.1:3306)/xiaohei")
	if !strings.Contains(dsn, "parseTime=true") {
		t.Fatalf("expected parseTime=true, got %q", dsn)
	}
	if !strings.Contains(dsn, "loc=Local") {
		t.Fatalf("expected loc=Local, got %q", dsn)
	}
	if !strings.Contains(dsn, "charset=utf8mb4") {
		t.Fatalf("expected charset=utf8mb4, got %q", dsn)
	}
}
