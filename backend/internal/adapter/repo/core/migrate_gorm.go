package repo

import (
	"fmt"
	"regexp"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// migrateGorm creates the schema for non-sqlite databases.
// The runtime repository uses database/sql with portable SQL, so column names must match the queries.
func migrateGorm(db *gorm.DB) error {
	models := []any{
		&userRow{},
		&captchaRow{},
		&verificationCodeRow{},
		&goodsTypeRow{},
		&regionRow{},
		&planGroupRow{},
		&packageRow{},
		&systemImageRow{},
		&lineSystemImageRow{},
		&cartItemRow{},
		&orderRow{},
		&orderItemRow{},
		&vpsInstanceRow{},
		&orderEventRow{},
		&adminAuditLogRow{},
		&apiKeyRow{},
		&userAPIKeyRow{},
		&settingRow{},
		&emailTemplateRow{},
		&orderPaymentRow{},
		&billingCycleRow{},
		&automationLogRow{},
		&provisionJobRow{},
		&resizeTaskRow{},
		&integrationSyncLogRow{},
		&permissionGroupRow{},
		&userTierGroupRow{},
		&userTierDiscountRuleRow{},
		&userTierAutoRuleRow{},
		&userTierMembershipRow{},
		&userTierPriceCacheRow{},
		&couponProductGroupRow{},
		&couponRow{},
		&couponRedemptionRow{},
		&passwordResetTokenRow{},
		&passwordResetTicketRow{},
		&permissionRow{},
		&cmsCategoryRow{},
		&cmsPostRow{},
		&cmsBlockRow{},
		&uploadRow{},
		&ticketRow{},
		&ticketMessageRow{},
		&ticketResourceRow{},
		&walletRow{},
		&walletTransactionRow{},
		&walletOrderRow{},
		&scheduledTaskRunRow{},
		&notificationRow{},
		&pushTokenRow{},
		&realnameVerificationRow{},
		&pluginInstallationRow{},
		&pluginPaymentMethodRow{},
		&probeNodeRow{},
		&probeEnrollTokenRow{},
		&probeStatusEventRow{},
		&probeLogSessionRow{},
	}
	if db.Dialector != nil && db.Dialector.Name() == "sqlite" && isLegacySQLiteFromMySQLDump(db) {
		if err := normalizeSQLiteBigIntPrimaryKeys(db); err != nil {
			return err
		}
		if err := createMissingTablesOnly(db, models); err != nil {
			return err
		}
	} else {
		if err := db.AutoMigrate(models...); err != nil {
			return err
		}
	}
	if db.Dialector != nil && db.Dialector.Name() == "mysql" {
		if err := fixMySQLPartialUniqueIndexes(db); err != nil {
			return err
		}
		if err := fixMySQLTextColumns(db); err != nil {
			return err
		}
	}
	if err := repairTimestampNulls(db, models); err != nil {
		return err
	}
	return nil
}

func isLegacySQLiteFromMySQLDump(db *gorm.DB) bool {
	var ddl string
	if err := db.Raw("SELECT sql FROM sqlite_master WHERE type='table' AND name='users'").Scan(&ddl).Error; err != nil {
		return false
	}
	ddl = strings.ToLower(strings.TrimSpace(ddl))
	return strings.Contains(ddl, "bigint(20)")
}

func normalizeSQLiteBigIntPrimaryKeys(db *gorm.DB) error {
	type tableDDL struct {
		Name string
		SQL  string
	}
	var tables []tableDDL
	if err := db.Raw("SELECT name, sql FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'").Scan(&tables).Error; err != nil {
		return err
	}
	pkLineRE := regexp.MustCompile(`(?mi)^\s*PRIMARY KEY\s*\("id"\)\s*,?\s*$`)
	trailingCommaRE := regexp.MustCompile(`,\s*\)`)
	for _, t := range tables {
		ddl := strings.TrimSpace(t.SQL)
		lower := strings.ToLower(ddl)
		if !strings.Contains(lower, `"id" bigint(20)`) || !strings.Contains(lower, `primary key ("id")`) {
			continue
		}
		tmpName := t.Name + "__norm"
		newDDL := strings.Replace(ddl, fmt.Sprintf(`CREATE TABLE "%s"`, t.Name), fmt.Sprintf(`CREATE TABLE "%s"`, tmpName), 1)
		newDDL = strings.Replace(newDDL, `"id" bigint(20) NOT NULL ,`, `"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,`, 1)
		newDDL = strings.Replace(newDDL, `"id" bigint(20) NOT NULL,`, `"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,`, 1)
		newDDL = pkLineRE.ReplaceAllString(newDDL, "")
		newDDL = trailingCommaRE.ReplaceAllString(newDDL, "\n)")

		type colRow struct {
			Name string `gorm:"column:name"`
		}
		var cols []colRow
		if err := db.Raw(fmt.Sprintf(`PRAGMA table_info("%s")`, t.Name)).Scan(&cols).Error; err != nil {
			return err
		}
		if len(cols) == 0 {
			continue
		}
		colNames := make([]string, 0, len(cols))
		for _, c := range cols {
			colNames = append(colNames, fmt.Sprintf(`"%s"`, c.Name))
		}
		colList := strings.Join(colNames, ", ")

		tx := db.Begin()
		if tx.Error != nil {
			return tx.Error
		}
		if err := tx.Exec("PRAGMA foreign_keys = OFF").Error; err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Exec(newDDL).Error; err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Exec(fmt.Sprintf(`INSERT INTO "%s" (%s) SELECT %s FROM "%s"`, tmpName, colList, colList, t.Name)).Error; err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Exec(fmt.Sprintf(`DROP TABLE "%s"`, t.Name)).Error; err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Exec(fmt.Sprintf(`ALTER TABLE "%s" RENAME TO "%s"`, tmpName, t.Name)).Error; err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit().Error; err != nil {
			return err
		}
	}
	return nil
}

func createMissingTablesOnly(db *gorm.DB, models []any) error {
	for _, model := range models {
		if db.Migrator().HasTable(model) {
			continue
		}
		if err := db.AutoMigrate(model); err != nil {
			return err
		}
	}
	return nil
}

func repairTimestampNulls(db *gorm.DB, models []any) error {
	for _, model := range models {
		if db.Migrator().HasColumn(model, "created_at") {
			if err := db.Model(model).
				Where("created_at IS NULL").
				Update("created_at", clause.Expr{SQL: "CURRENT_TIMESTAMP"}).Error; err != nil {
				return err
			}
		}
		if db.Migrator().HasColumn(model, "updated_at") {
			if err := db.Model(model).
				Where("updated_at IS NULL").
				Update("updated_at", clause.Expr{SQL: "CURRENT_TIMESTAMP"}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// MySQL does not support partial unique indexes. We keep the same index names but make them non-unique.
func fixMySQLPartialUniqueIndexes(db *gorm.DB) error {
	if db.Migrator().HasIndex(&goodsTypeRow{}, "idx_goods_types_code_unique") {
		if err := db.Exec("DROP INDEX idx_goods_types_code_unique ON goods_types").Error; err != nil {
			return err
		}
	}
	if err := db.Exec("CREATE INDEX idx_goods_types_code_unique ON goods_types(code)").Error; err != nil {
		return err
	}

	if db.Migrator().HasIndex(&planGroupRow{}, "idx_plan_groups_gt_line_unique") {
		if err := db.Exec("DROP INDEX idx_plan_groups_gt_line_unique ON plan_groups").Error; err != nil {
			return err
		}
	}
	if err := db.Exec("CREATE INDEX idx_plan_groups_gt_line_unique ON plan_groups(goods_type_id, line_id)").Error; err != nil {
		return err
	}

	if db.Migrator().HasIndex(&packageRow{}, "idx_packages_gt_product_unique") {
		if err := db.Exec("DROP INDEX idx_packages_gt_product_unique ON packages").Error; err != nil {
			return err
		}
	}
	if err := db.Exec("CREATE INDEX idx_packages_gt_product_unique ON packages(goods_type_id, plan_group_id, product_id)").Error; err != nil {
		return err
	}

	return nil
}

func fixMySQLTextColumns(db *gorm.DB) error {
	stmts := []string{
		"ALTER TABLE cms_blocks MODIFY COLUMN content_json LONGTEXT NOT NULL",
		"ALTER TABLE cms_blocks MODIFY COLUMN custom_html LONGTEXT NOT NULL",
		"ALTER TABLE cms_posts MODIFY COLUMN content_html LONGTEXT NOT NULL",
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}
