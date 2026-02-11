package db

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

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
		return nil, errors.New("missing APP_DB_TYPE")
	}

	var gdb *gorm.DB
	var err error

	switch dbType {
	case "sqlite":
		if strings.TrimSpace(cfg.DBPath) == "" {
			return nil, errors.New("missing APP_DB_PATH for sqlite")
		}
		dir := filepath.Dir(cfg.DBPath)
		if dir != "" && dir != "." {
			if mkErr := os.MkdirAll(dir, 0o755); mkErr != nil {
				return nil, mkErr
			}
		}
		gdb, err = gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{
			DisableForeignKeyConstraintWhenMigrating: true,
			Logger:                                   logger.Default.LogMode(logger.Silent),
		})
	case "mysql":
		dsn := strings.TrimSpace(cfg.DBDSN)
		if dsn == "" {
			return nil, errors.New("missing APP_DB_DSN for mysql")
		}
		gdb, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
			DisableForeignKeyConstraintWhenMigrating: true,
			Logger:                                   logger.Default.LogMode(logger.Silent),
		})
	case "postgres", "postgresql":
		dsn := strings.TrimSpace(cfg.DBDSN)
		if dsn == "" {
			return nil, errors.New("missing APP_DB_DSN for postgres")
		}
		gdb, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			DisableForeignKeyConstraintWhenMigrating: true,
			Logger:                                   logger.Default.LogMode(logger.Silent),
		})
	default:
		return nil, errors.New("unsupported APP_DB_TYPE: " + dbType)
	}
	if err != nil {
		return nil, err
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
