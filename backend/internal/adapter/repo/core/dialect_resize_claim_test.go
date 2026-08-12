package repo

import (
	"os"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// openMigratedDB opens a gorm DB from a DSN and migrates the schema. Used for
// the MySQL/PostgreSQL concurrency variants of the resize claim tests. The
// tests skip when the DSN env var is not set (CI provides them).
func openMigratedDB(t *testing.T, dialect, dsn string) *GormRepo {
	t.Helper()
	if dsn == "" {
		t.Skip("dialect " + dialect + " not configured (set TEST_" + dialect + "_DSN)")
	}
	var dialector gorm.Dialector
	switch dialect {
	case "MYSQL":
		dialector = mysql.Open(dsn)
	case "PG":
		dialector = postgres.Open(dsn)
	default:
		t.Fatalf("unknown dialect %s", dialect)
	}
	db, err := gorm.Open(dialector, &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open %s: %v", dialect, err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate %s: %v", dialect, err)
	}
	return NewGormRepo(db)
}

func TestResizeClaimConcurrencyMySQL(t *testing.T) {
	g := openMigratedDB(t, "MYSQL", os.Getenv("TEST_MYSQL_DSN"))
	runResizeClaimConcurrency(t, g)
}

func TestResizeClaimConcurrencyPostgreSQL(t *testing.T) {
	g := openMigratedDB(t, "PG", os.Getenv("TEST_PG_DSN"))
	runResizeClaimConcurrency(t, g)
}

func TestResizeCompleteFencedMySQL(t *testing.T) {
	g := openMigratedDB(t, "MYSQL", os.Getenv("TEST_MYSQL_DSN"))
	runResizeCompleteFenced(t, g)
}

func TestResizeCompleteFencedPostgreSQL(t *testing.T) {
	g := openMigratedDB(t, "PG", os.Getenv("TEST_PG_DSN"))
	runResizeCompleteFenced(t, g)
}
