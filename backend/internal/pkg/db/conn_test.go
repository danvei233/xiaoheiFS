package db

import (
	"path/filepath"
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
