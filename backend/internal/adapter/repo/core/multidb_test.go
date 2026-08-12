package repo_test

import (
	"os"
	"strings"
	"testing"

	"xiaoheiplay/internal/adapter/repo/core"
	"xiaoheiplay/internal/pkg/config"
	"xiaoheiplay/internal/pkg/db"
	"xiaoheiplay/internal/testutil"
)

// availableDialects returns the dialects that should run in three-database
// tests: SQLite always, plus MySQL and PostgreSQL when their test DSNs are
// configured.
func availableDialects(t *testing.T) []string {
	t.Helper()
	dialects := []string{"sqlite"}
	if strings.TrimSpace(os.Getenv("XIAOHEI_TEST_MYSQL_DSN")) != "" {
		dialects = append(dialects, "mysql")
	}
	if strings.TrimSpace(os.Getenv("XIAOHEI_TEST_PG_DSN")) != "" {
		dialects = append(dialects, "postgres")
	}
	return dialects
}

// openDialectRepo opens a fresh repository on the given dialect, running the
// full migration. MySQL and PostgreSQL tests are skipped when their DSN env var
// is unset. Each test must use unique usernames/order numbers because a shared
// external database may already hold rows from earlier test runs.
func openDialectRepo(t *testing.T, dialect string) *repo.GormRepo {
	t.Helper()
	switch dialect {
	case "sqlite":
		_, r := testutil.NewTestDB(t, false)
		return r
	case "mysql":
		dsn := strings.TrimSpace(os.Getenv("XIAOHEI_TEST_MYSQL_DSN"))
		if dsn == "" {
			t.Skip("set XIAOHEI_TEST_MYSQL_DSN to run MySQL repo tests")
		}
		conn, err := db.Open(config.Config{DBType: "mysql", DBDSN: dsn})
		if err != nil {
			t.Fatalf("open mysql: %v", err)
		}
		t.Cleanup(func() { _ = conn.SQL.Close() })
		if err := repo.Migrate(conn.Gorm); err != nil {
			t.Fatalf("migrate mysql: %v", err)
		}
		return repo.NewGormRepo(conn.Gorm)
	case "postgres", "postgresql":
		dsn := strings.TrimSpace(os.Getenv("XIAOHEI_TEST_PG_DSN"))
		if dsn == "" {
			t.Skip("set XIAOHEI_TEST_PG_DSN to run PostgreSQL repo tests")
		}
		conn, err := db.Open(config.Config{DBType: "postgres", DBDSN: dsn})
		if err != nil {
			t.Fatalf("open postgres: %v", err)
		}
		t.Cleanup(func() { _ = conn.SQL.Close() })
		if err := repo.Migrate(conn.Gorm); err != nil {
			t.Fatalf("migrate postgres: %v", err)
		}
		return repo.NewGormRepo(conn.Gorm)
	default:
		t.Fatalf("unsupported dialect %q", dialect)
		return nil
	}
}
