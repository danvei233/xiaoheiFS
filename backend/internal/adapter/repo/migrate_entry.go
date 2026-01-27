package repo

import (
	"fmt"

	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("nil gorm db")
	}
	switch db.Dialector.Name() {
	case "sqlite":
		sqlDB, err := db.DB()
		if err != nil {
			return err
		}
		return migrateSQLite(sqlDB)
	default:
		return migrateGorm(db)
	}
}
