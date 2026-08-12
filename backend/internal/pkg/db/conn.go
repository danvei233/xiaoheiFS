package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	mysqlDriver "github.com/go-sql-driver/mysql"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"fmt"
	"xiaoheiplay/internal/pkg/config"
)

type Conn struct {
	Gorm    *gorm.DB
	SQL     *sql.DB
	Dialect string
}

func Open(cfg config.Config) (*Conn, error) {
	dbType := strings.ToLower(strings.TrimSpace(cfg.DBType))
	if dbType == "" {
		return nil, fmt.Errorf("missing APP_DB_TYPE")
	}

	var gdb *gorm.DB
	var err error

	switch dbType {
	case "sqlite":
		if strings.TrimSpace(cfg.DBPath) == "" {
			return nil, fmt.Errorf("missing APP_DB_PATH for sqlite")
		}
		dir := filepath.Dir(cfg.DBPath)
		if dir != "" && dir != "." {
			if mkErr := os.MkdirAll(dir, 0o755); mkErr != nil {
				return nil, mkErr
			}
		}
		gdb, err = gorm.Open(sqlite.Open(sqliteDSN(cfg.DBPath)), &gorm.Config{
			DisableForeignKeyConstraintWhenMigrating: true,
			Logger:                                   logger.Default.LogMode(logger.Silent),
		})
	case "mysql":
		dsn := strings.TrimSpace(cfg.DBDSN)
		if dsn == "" {
			return nil, fmt.Errorf("missing APP_DB_DSN for mysql")
		}
		dsn = normalizeMySQLDSN(dsn)
		gdb, err = gorm.Open(gormmysql.Open(dsn), &gorm.Config{
			DisableForeignKeyConstraintWhenMigrating: true,
			Logger:                                   logger.Default.LogMode(logger.Silent),
		})
	case "postgres", "postgresql":
		dsn := strings.TrimSpace(cfg.DBDSN)
		if dsn == "" {
			return nil, fmt.Errorf("missing APP_DB_DSN for postgres")
		}
		gdb, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			DisableForeignKeyConstraintWhenMigrating: true,
			Logger:                                   logger.Default.LogMode(logger.Silent),
		})
	default:
		return nil, fmt.Errorf("%s", "unsupported APP_DB_TYPE: "+dbType)
	}
	if err != nil {
		return nil, err
	}

	if dbType == "sqlite" {
		// The driver creates the DB file with the process umask (usually 0644);
		// tighten it to protect password hashes, wallet balances and plugin
		// ciphertext on multi-user hosts.
		if strings.TrimSpace(cfg.DBPath) != "" && cfg.DBPath != ":memory:" {
			if fi, statErr := os.Stat(cfg.DBPath); statErr == nil {
				if fi.Mode().Perm() != 0o600 {
					_ = os.Chmod(cfg.DBPath, 0o600)
				}
			}
		}
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, err
	}

	// Conservative pool defaults to avoid overloading small deployments.
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)

	return &Conn{
		Gorm:    gdb,
		SQL:     sqlDB,
		Dialect: gdb.Dialector.Name(),
	}, nil
}

// sqliteDSN appends journal_mode(WAL) so concurrent readers/writers do not hit
// "database is locked". The glebarez driver already applies a default
// busy_timeout(5000) on every connection.
func sqliteDSN(path string) string {
	if strings.TrimSpace(path) == "" || path == ":memory:" {
		return path
	}
	if strings.Contains(path, "?") {
		return path + "&_pragma=journal_mode(WAL)"
	}
	return path + "?_pragma=journal_mode(WAL)"
}

func normalizeMySQLDSN(raw string) string {
	raw = strings.TrimSpace(raw)
	cfg, err := mysqlDriver.ParseDSN(raw)
	if err != nil {
		// Keep backward compatibility for unusual DSN formats ParseDSN may reject.
		return raw
	}
	cfg.ParseTime = true
	if !strings.Contains(strings.ToLower(raw), "loc=") {
		cfg.Loc = time.Local
	}
	if cfg.Params == nil {
		cfg.Params = map[string]string{}
	}
	if _, ok := cfg.Params["charset"]; !ok {
		cfg.Params["charset"] = "utf8mb4"
	}
	return cfg.FormatDSN()
}
