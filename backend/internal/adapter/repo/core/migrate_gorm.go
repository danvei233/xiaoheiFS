package repo

import (
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
	if err := db.AutoMigrate(models...); err != nil {
		return err
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

	if db.Migrator().HasIndex(&packageRow{}, "idx_packages_gt_integration_unique") {
		if err := db.Exec("DROP INDEX idx_packages_gt_integration_unique ON packages").Error; err != nil {
			return err
		}
	}
	if db.Migrator().HasIndex(&packageRow{}, "idx_packages_integration") {
		if err := db.Exec("DROP INDEX idx_packages_integration ON packages").Error; err != nil {
			return err
		}
	}
	if err := db.Exec("CREATE INDEX idx_packages_integration ON packages(integration_package_id)").Error; err != nil {
		return err
	}

	return nil
}

func fixMySQLTextColumns(db *gorm.DB) error {
	stmts := []string{
		"ALTER TABLE cms_blocks MODIFY COLUMN content_json LONGTEXT NOT NULL",
		"ALTER TABLE cms_blocks MODIFY COLUMN custom_html LONGTEXT NOT NULL",
		"ALTER TABLE cms_posts MODIFY COLUMN content_html LONGTEXT NOT NULL",
		"ALTER TABLE user_tier_auto_rules MODIFY COLUMN conditions_json LONGTEXT NOT NULL",
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}
