package db

import (
	"os"
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

// TestOpenSQLiteConcurrency guards against SQLITE_BUSY under concurrent writes:
// WAL journal mode plus a non-zero busy_timeout are required for the multi-
// connection pool used by this package.
func TestOpenSQLiteConcurrency(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.db")
	conn, err := Open(config.Config{DBType: "sqlite", DBPath: path})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer conn.SQL.Close()

	var mode string
	if err := conn.Gorm.Raw("PRAGMA journal_mode").Scan(&mode).Error; err != nil {
		t.Fatalf("query journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Fatalf("expected journal_mode=wal, got %q", mode)
	}

	var bt int64
	if err := conn.Gorm.Raw("PRAGMA busy_timeout").Scan(&bt).Error; err != nil {
		t.Fatalf("query busy_timeout: %v", err)
	}
	if bt <= 0 {
		t.Fatalf("expected positive busy_timeout, got %d", bt)
	}
}

// TestOpenSQLiteFilePerms ensures the SQLite file is not left world-readable.
func TestOpenSQLiteFilePerms(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.db")
	conn, err := Open(config.Config{DBType: "sqlite", DBPath: path})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer conn.SQL.Close()

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := fi.Mode().Perm(); perm != 0o600 {
		t.Fatalf("expected DB file mode 0600, got %o", perm)
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
