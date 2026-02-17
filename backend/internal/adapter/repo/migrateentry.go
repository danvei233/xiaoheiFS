package repo

import (
	"fmt"

	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("nil gorm db")
	}
	return migrateGorm(db)
}
