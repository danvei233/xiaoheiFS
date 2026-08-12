package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"

	"xiaoheiplay/internal/domain"
)

// migrateGorm creates or repairs the schema for all supported dialects. It is
// the single migration entry point used by main and the install bootstrap, and
// includes a legacy SQLite (MySQL-dump) repair path plus MySQL-specific index
// and text-column normalizations.
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
		&userAPIKeyNonceRow{},
		&robotWebhookNonceRow{},
		&authFailureBudgetRow{},
		&refreshTokenFamilyRow{},
		&refreshTokenUseRow{},
		&settingRow{},
		&settingListValueRow{},
		&scheduledTaskConfigRow{},
		&robotWebhookRow{},
		&smsTemplateRow{},
		&packageCapabilityRow{},
		&goodsTypeCapabilityRow{},
		&emailTemplateRow{},
		&orderPaymentRow{},
		&billingCycleRow{},
		&automationLogRow{},
		&provisionJobRow{},
		&resizeTaskRow{},
		&integrationSyncLogRow{},
		&permissionGroupRow{},
		&permissionGroupPermissionRow{},
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
		&scheduledTaskLeaseRow{},
		&notificationRow{},
		&pushTokenRow{},
		&realnameVerificationRow{},
		&realnameAttemptRow{},
		&rateLimitBucketRow{},
		&pluginInstallationRow{},
		&pluginPaymentMethodRow{},
		&probeNodeRow{},
		&probeEnrollTokenRow{},
		&probeStatusEventRow{},
		&probeLogSessionRow{},
		&migrationConflictRow{},
		&refundCleanupJobRow{},
		&vpsLifecycleLockRow{},
	}
	legacySQLite := db.Dialector != nil && db.Dialector.Name() == "sqlite" && isLegacySQLiteFromMySQLDump(db)
	if db.Dialector != nil && db.Dialector.Name() == "mysql" {
		if err := ensureMySQLPartialUniqueIndexesBeforeMigrate(db); err != nil {
			return err
		}
		if err := prepareMySQLTextColumnsBeforeMigrate(db); err != nil {
			return err
		}
	}
	if legacySQLite {
		if err := normalizeSQLiteBigIntPrimaryKeys(db); err != nil {
			return err
		}
		if err := createMissingTablesOnly(db, models); err != nil {
			return err
		}
		// Existing legacy tables predate the current models and miss columns and
		// constraints added since the dump (e.g. users.permission_group_id, the
		// order/payment/provision idempotency indexes). AutoMigrate completes
		// them on the first run instead of leaving a gap until a second restart.
		//
		// Legacy data can contradict the current models in two ways that make
		// the first AutoMigrate abort: a NULL value in a column the model now
		// declares NOT NULL, and a missing NOT NULL column on a non-empty table
		// (SQLite cannot ADD COLUMN ... NOT NULL without a constant default).
		// Backfill and pre-create those columns before AutoMigrate runs so the
		// table-recreation data copy never hits a NOT NULL violation.
		if err := prepareLegacySQLiteNotNullNulls(db, models); err != nil {
			return err
		}
		// Legacy tables may contain rows that violate unique indexes the current
		// models add via AutoMigrate (provision_jobs.order_item_id was not unique
		// historically, so a crashed retry could leave two jobs for one item).
		// Dedupe before AutoMigrate or the first CREATE UNIQUE INDEX aborts startup.
		if err := normalizeLegacySQLiteUniqueIndexData(db); err != nil {
			return err
		}
		if err := db.AutoMigrate(models...); err != nil {
			return err
		}
	} else {
		if err := db.AutoMigrate(models...); err != nil {
			return err
		}
	}
	if err := normalizeVPSOrderItemLinks(db); err != nil {
		return err
	}
	if err := normalizeLegacyCapabilitySettings(db); err != nil {
		return err
	}
	if err := ensureWalletTransactionIdempotencyIndex(db); err != nil {
		return err
	}
	if err := ensureApprovedPaymentUniqueIndex(db, legacySQLite); err != nil {
		return err
	}
	if err := ensureVPSOrderItemUniqueIndex(db, legacySQLite); err != nil {
		return err
	}
	if err := ensureUserPhoneUniqueIndex(db); err != nil {
		return err
	}
	if err := ensureCouponRedemptionPerUserUniqueIndex(db); err != nil {
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
	if err := ensureRealNameProviderState(db); err != nil {
		return err
	}
	if err := repairTimestampNulls(db, models); err != nil {
		return err
	}
	return nil
}

// ensureRealNameProviderState adds the realname_verifications.provider_state
// column used to hold the encrypted pending KYC continuation state.
func ensureRealNameProviderState(db *gorm.DB) error {
	if !db.Migrator().HasTable(&realnameVerificationRow{}) {
		return nil
	}
	if db.Migrator().HasColumn(&realnameVerificationRow{}, "provider_state") {
		return nil
	}
	return db.Migrator().AddColumn(&realnameVerificationRow{}, "ProviderState")
}

func ensureVPSOrderItemUniqueIndex(db *gorm.DB, legacySQLite bool) error {
	const indexName = "uniq_vps_instances_order_item_id"
	if db.Migrator().HasIndex(&vpsInstanceRow{}, indexName) {
		// HasIndex matches by name only: a legacy same-named NON-unique index
		// would otherwise make this a permanent no-op and the uniqueness that the
		// code relies on (CreateInstance idempotency, one VPS per order item)
		// would silently never exist. Verify the actual index definition.
		isUnique, err := indexIsUnique(db, "vps_instances", indexName)
		if err != nil {
			return err
		}
		if isUnique {
			// Older versions created a PARTIAL unique index on legacy SQLite
			// (`WHERE order_item_id > 0`). SQLite only accepts an `ON CONFLICT
			// (order_item_id)` clause when it matches a unique constraint
			// exactly — a partial predicate is not matched — so every
			// CreateInstance insert failed on those databases. Rebuild as a full
			// unique index; safe because normalizeVPSOrderItemLinks has already
			// nulled zero-links and detached duplicates (SQLite never conflicts
			// on NULLs). MySQL/PostgreSQL/SQLite fresh installs never carried the
			// partial form, so the rebuild is legacy-SQLite only.
			if legacySQLite {
				partial, err := sqliteIndexIsPartial(db, "vps_instances", indexName)
				if err != nil {
					return err
				}
				if partial {
					if err := db.Migrator().DropIndex(&vpsInstanceRow{}, indexName); err != nil {
						return fmt.Errorf("drop partial %s before rebuild: %w", indexName, err)
					}
					return db.Exec("CREATE UNIQUE INDEX " + indexName + " ON vps_instances(order_item_id)").Error
				}
			}
			return nil
		}
		if err := db.Migrator().DropIndex(&vpsInstanceRow{}, indexName); err != nil {
			return fmt.Errorf("drop non-unique %s before recreate: %w", indexName, err)
		}
	}
	return db.Exec("CREATE UNIQUE INDEX " + indexName + " ON vps_instances(order_item_id)").Error
}

// sqliteIndexIsPartial reports whether the named index on the given table is a
// partial index (has a WHERE predicate). Only SQLite expresses partial indexes
// in its DDL, so the check is SQLite-only.
func sqliteIndexIsPartial(db *gorm.DB, table, indexName string) (bool, error) {
	var sqlText string
	if err := db.Raw("SELECT sql FROM sqlite_master WHERE type='index' AND tbl_name = ? AND name = ?", table, indexName).Scan(&sqlText).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return strings.Contains(strings.ToUpper(strings.TrimSpace(sqlText)), " WHERE "), nil
}

// indexIsUnique reports whether the named index actually enforces uniqueness,
// reading the index definition rather than trusting the name. Returns (false,
// nil) when the index does not exist, so callers treat a missing index as "not
// unique" and create it.
func indexIsUnique(db *gorm.DB, table, indexName string) (bool, error) {
	switch db.Dialector.Name() {
	case "mysql":
		return mysqlIndexIsUnique(db, table, indexName)
	case "postgres", "postgresql":
		var isUnique bool
		if err := db.Raw(
			`SELECT i.indisunique FROM pg_index i JOIN pg_class c ON c.oid = i.indexrelid WHERE c.relname = ? LIMIT 1`,
			indexName,
		).Scan(&isUnique).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return false, nil
			}
			return false, err
		}
		return isUnique, nil
	default: // sqlite
		var sqlText string
		if err := db.Raw("SELECT sql FROM sqlite_master WHERE type='index' AND name = ?", indexName).Scan(&sqlText).Error; err != nil {
			return false, err
		}
		return strings.Contains(strings.ToUpper(sqlText), "UNIQUE"), nil
	}
}

func normalizeVPSOrderItemLinks(db *gorm.DB) error {
	if !db.Migrator().HasTable(&vpsInstanceRow{}) || !db.Migrator().HasColumn(&vpsInstanceRow{}, "order_item_id") {
		return nil
	}
	zeroed := db.Model(&vpsInstanceRow{}).Where("order_item_id = 0").UpdateColumn("order_item_id", nil)
	if zeroed.Error != nil {
		return zeroed.Error
	}
	type duplicateLink struct {
		OrderItemID int64 `gorm:"column:order_item_id"`
		KeepID      int64 `gorm:"column:keep_id"`
	}
	var duplicates []duplicateLink
	if err := db.Model(&vpsInstanceRow{}).
		Select("order_item_id, MIN(id) AS keep_id").
		Where("order_item_id IS NOT NULL AND order_item_id > 0").
		Group("order_item_id").
		Having("COUNT(*) > 1").
		Scan(&duplicates).Error; err != nil {
		return err
	}
	detached := 0
	conflicts := 0
	var liveConflictDetails []string
	// Legacy tables may predate the vps_instances.status column; conflicts can
	// only be classified against a status we can read.
	hasStatus := db.Migrator().HasColumn(&vpsInstanceRow{}, "status")
	type dupRow struct {
		ID     int64 `gorm:"column:id"`
		Status string
	}
	for _, duplicate := range duplicates {
		// Detaching a VPS that still represents a live host silently orphans it:
		// the user loses the order-item link and every lifecycle operation
		// (renew/refund/resize) keyed on it. Record an actionable conflict so an
		// operator reconciles before the unique index hides the duplicate.
		var dupRows []dupRow
		q := db.Model(&vpsInstanceRow{}).Where("order_item_id = ? AND id <> ?", duplicate.OrderItemID, duplicate.KeepID)
		if hasStatus {
			q = q.Select("id, status")
		} else {
			q = q.Select("id, '' AS status")
		}
		if err := q.Find(&dupRows).Error; err != nil {
			return err
		}
		hasLiveConflict := false
		for _, dup := range dupRows {
			if isLiveVPSStatus(dup.Status) {
				hasLiveConflict = true
				detail := fmt.Sprintf(
					"vps_instances id=%d and id=%d share order_item_id=%d; both links preserved for operator reconciliation (status=%s)",
					duplicate.KeepID, dup.ID, duplicate.OrderItemID, dup.Status)
				if err := db.Create(&migrationConflictRow{
					SourceTable: "vps_instances",
					RefID:       dup.ID,
					Type:        "duplicate_order_item_link",
					Detail:      detail,
					Status:      "pending",
				}).Error; err != nil {
					return fmt.Errorf("record vps duplicate conflict: %w", err)
				}
				liveConflictDetails = append(liveConflictDetails, detail)
				conflicts++
			}
		}
		if hasLiveConflict {
			continue
		}
		res := db.Model(&vpsInstanceRow{}).
			Where("order_item_id = ? AND id <> ?", duplicate.OrderItemID, duplicate.KeepID).
			UpdateColumn("order_item_id", nil)
		if res.Error != nil {
			return res.Error
		}
		detached += int(res.RowsAffected)
	}
	if zeroed.RowsAffected > 0 || len(duplicates) > 0 {
		log.Printf("migration: vps_instances order_item_id normalized: %d zero-link(s) nulled, %d duplicate link(s) detached (%d row(s)), %d live conflict(s) recorded",
			zeroed.RowsAffected, len(duplicates), detached, conflicts)
	}
	if len(liveConflictDetails) > 0 {
		return fmt.Errorf("vps order-item migration requires operator reconciliation: %s", strings.Join(liveConflictDetails, "; "))
	}
	return nil
}

// isLiveVPSStatus reports whether a VPS row still represents a live host that
// must not be silently orphaned by a migration. Unknown/empty statuses mean the
// host is gone or the row never materialized, so detaching those is safe.
func isLiveVPSStatus(status string) bool {
	switch status {
	case "", string(domain.VPSStatusUnknown):
		return false
	default:
		return true
	}
}

func ensureWalletTransactionIdempotencyIndex(db *gorm.DB) error {
	const indexName = "idx_wallet_transactions_idempotency_key"
	if !db.Migrator().HasColumn(&walletTransactionRow{}, "idempotency_key") {
		if err := db.Migrator().AddColumn(&walletTransactionRow{}, "IdempotencyKey"); err != nil {
			return err
		}
	}
	if db.Migrator().HasIndex(&walletTransactionRow{}, indexName) {
		return nil
	}
	return db.Migrator().CreateIndex(&walletTransactionRow{}, indexName)
}

// ensureApprovedPaymentUniqueIndex enforces the "single approved payment per
// order" invariant at the database level. A concurrent pair of provider
// callbacks (or a callback racing an admin approval) can both pass an
// in-process "already settled?" read before either commits; only a unique
// constraint closes that window unconditionally. SQLite and PostgreSQL support
// a partial unique index (WHERE status='approved'); MySQL does not, so there the
// invariant is enforced by the transactional count check in ApproveOrderAtomic
// plus the CAS transition in HandleNotify (UpdatePaymentStatusIfCurrent).
func ensureApprovedPaymentUniqueIndex(db *gorm.DB, legacySQLite bool) error {
	const indexName = "uniq_order_payments_approved_order"
	if !db.Migrator().HasTable(&orderPaymentRow{}) || !db.Migrator().HasColumn(&orderPaymentRow{}, "status") {
		return nil
	}
	if db.Migrator().HasIndex(&orderPaymentRow{}, indexName) {
		// HasIndex matches by name only; verify the definition actually enforces
		// uniqueness on a partial WHERE, and rebuild a non-unique/missing index.
		isUnique, err := indexIsUnique(db, "order_payments", indexName)
		if err != nil {
			return err
		}
		if isUnique {
			return nil
		}
		if err := db.Migrator().DropIndex(&orderPaymentRow{}, indexName); err != nil {
			return fmt.Errorf("drop non-unique %s before recreate: %w", indexName, err)
		}
	}
	if db.Dialector != nil && db.Dialector.Name() == "mysql" {
		// MySQL cannot express a partial unique index directly, but it can enforce
		// the same invariant with a generated column that is NULL for every
		// non-approved row (NULLs are exempt from unique enforcement) and a unique
		// index on that column. Without this, two concurrent provider callbacks
		// for two different payments of one order can both pass the in-process
		// "already settled?" read and both commit approved, double-charging the
		// user. Normalize any legacy double-approval rows first so the unique
		// index does not abort startup on data the old code produced.
		if err := normalizeDuplicateApprovedPayments(db); err != nil {
			return err
		}
		if err := ensureMySQLApprovedPaymentColumn(db); err != nil {
			return err
		}
		return db.Exec("CREATE UNIQUE INDEX " + indexName + " ON order_payments(approved_order_id)").Error
	}
	// SQLite and PostgreSQL support a partial unique index. Normalize any legacy
	// double-approval rows first so the index does not abort startup on data the
	// old code produced.
	if err := normalizeDuplicateApprovedPayments(db); err != nil {
		return err
	}
	return db.Exec("CREATE UNIQUE INDEX " + indexName + " ON order_payments(order_id) WHERE status = 'approved'").Error
}

// ensureMySQLApprovedPaymentColumn adds the generated column the approved-payment
// unique index is built on. The column mirrors order_id for approved rows and is
// NULL otherwise, so at most one approved payment can exist per order on MySQL.
// Idempotent: once the column exists nothing runs.
func ensureMySQLApprovedPaymentColumn(db *gorm.DB) error {
	var count int64
	if err := db.Raw(
		"SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'order_payments' AND column_name = 'approved_order_id'",
	).Scan(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return db.Exec(
		"ALTER TABLE order_payments ADD COLUMN approved_order_id BIGINT GENERATED ALWAYS AS (CASE WHEN status = 'approved' THEN order_id ELSE NULL END) STORED",
	).Error
}

// normalizeDuplicateApprovedPayments demotes legacy duplicate approved payments
// (two approved rows for one order — produced by the pre-invariant double-pay
// bug) to rejected so the partial unique index can be created. The canonical
// payment kept is the earliest id; every newer approved row is recorded in
// migration_conflicts for operator visibility before being rejected.
func normalizeDuplicateApprovedPayments(db *gorm.DB) error {
	type duplicateApproved struct {
		OrderID int64 `gorm:"column:order_id"`
		KeepID  int64 `gorm:"column:keep_id"`
	}
	var duplicates []duplicateApproved
	if err := db.Model(&orderPaymentRow{}).
		Select("order_id, MIN(id) AS keep_id").
		Where("status = ?", string(domain.PaymentStatusApproved)).
		Group("order_id").
		Having("COUNT(*) > 1").
		Scan(&duplicates).Error; err != nil {
		return err
	}
	for _, d := range duplicates {
		var dupIDs []int64
		if err := db.Model(&orderPaymentRow{}).
			Where("order_id = ? AND status = ? AND id <> ?", d.OrderID, string(domain.PaymentStatusApproved), d.KeepID).
			Pluck("id", &dupIDs).Error; err != nil {
			return err
		}
		for _, id := range dupIDs {
			if err := db.Create(&migrationConflictRow{
				SourceTable: "order_payments",
				RefID:       id,
				Type:        "duplicate_approved_payment",
				Detail: fmt.Sprintf(
					"order_payments id=%d demoted (kept id=%d) so one approved payment per order holds", id, d.KeepID),
				Status: "pending",
			}).Error; err != nil {
				return fmt.Errorf("record duplicate payment conflict: %w", err)
			}
		}
		res := db.Model(&orderPaymentRow{}).
			Where("order_id = ? AND status = ? AND id <> ?", d.OrderID, string(domain.PaymentStatusApproved), d.KeepID).
			Updates(map[string]any{
				"status":        domain.PaymentStatusRejected,
				"review_reason": "duplicate approved payment normalized by migration",
				"updated_at":    time.Now(),
			})
		if res.Error != nil {
			return res.Error
		}
		log.Printf("migration: order_payments order_id=%d duplicate approved payment(s) demoted to rejected, kept id=%d", d.OrderID, d.KeepID)
	}
	return nil
}

// ensureCouponRedemptionPerUserUniqueIndex adds a durable per-user-per-coupon
// gate on SQLite/PostgreSQL: a partial unique index over active redemptions
// (status IN ('applied','confirmed')). The redemption count-then-insert uses
// FOR UPDATE on the coupon row, which serializes concurrent redemptions on
// MySQL/PostgreSQL but is a no-op on SQLite; the index is what makes concurrent
// per-user redemptions converge to a stable conflict there instead of relying
// on retry-only semantics. MySQL cannot express a partial unique index and is
// covered by the FOR UPDATE row lock. Legacy duplicate active redemptions are
// normalized first (keep MIN(id), demote the rest to canceled) so the index
// does not abort startup.
func ensureCouponRedemptionPerUserUniqueIndex(db *gorm.DB) error {
	const indexName = "uniq_coupon_redemptions_active_user"
	if db.Dialector != nil && db.Dialector.Name() == "mysql" {
		return nil
	}
	if !db.Migrator().HasTable(&couponRedemptionRow{}) || !db.Migrator().HasColumn(&couponRedemptionRow{}, "status") {
		return nil
	}
	if db.Migrator().HasIndex(&couponRedemptionRow{}, indexName) {
		ok, err := indexIsUnique(db, "coupon_redemptions", indexName)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		if err := db.Migrator().DropIndex(&couponRedemptionRow{}, indexName); err != nil {
			return fmt.Errorf("drop non-unique %s before recreate: %w", indexName, err)
		}
	}
	type dup struct {
		CouponID int64 `gorm:"column:coupon_id"`
		UserID   int64 `gorm:"column:user_id"`
		KeepID   int64 `gorm:"column:keep_id"`
	}
	var duplicates []dup
	if err := db.Model(&couponRedemptionRow{}).
		Select("coupon_id, user_id, MIN(id) AS keep_id").
		Where("status IN ?", []string{string(domain.CouponRedemptionStatusApplied), string(domain.CouponRedemptionStatusConfirmed)}).
		Group("coupon_id, user_id").
		Having("COUNT(*) > 1").
		Scan(&duplicates).Error; err != nil {
		return err
	}
	for _, d := range duplicates {
		var dupIDs []int64
		if err := db.Model(&couponRedemptionRow{}).
			Where("coupon_id = ? AND user_id = ? AND status IN ? AND id <> ?",
				d.CouponID, d.UserID,
				[]string{string(domain.CouponRedemptionStatusApplied), string(domain.CouponRedemptionStatusConfirmed)},
				d.KeepID).
			Pluck("id", &dupIDs).Error; err != nil {
			return err
		}
		for _, id := range dupIDs {
			if err := db.Create(&migrationConflictRow{
				SourceTable: "coupon_redemptions",
				RefID:       id,
				Type:        "duplicate_active_coupon_redemption",
				Detail:      fmt.Sprintf("coupon_redemptions id=%d demoted (kept id=%d) so per-user active redemption uniqueness holds", id, d.KeepID),
				Status:      "pending",
			}).Error; err != nil {
				return fmt.Errorf("record coupon redemption conflict: %w", err)
			}
		}
		if err := db.Model(&couponRedemptionRow{}).
			Where("coupon_id = ? AND user_id = ? AND status IN ? AND id <> ?",
				d.CouponID, d.UserID,
				[]string{string(domain.CouponRedemptionStatusApplied), string(domain.CouponRedemptionStatusConfirmed)},
				d.KeepID).
			Updates(map[string]any{"status": domain.CouponRedemptionStatusCanceled, "updated_at": time.Now()}).Error; err != nil {
			return err
		}
	}
	return db.Exec("CREATE UNIQUE INDEX " + indexName + " ON coupon_redemptions(coupon_id, user_id) WHERE status IN ('applied', 'confirmed')").Error
}

func isLegacySQLiteFromMySQLDump(db *gorm.DB) bool {
	var ddl string
	if err := db.Raw("SELECT sql FROM sqlite_master WHERE type='table' AND name='users'").Scan(&ddl).Error; err != nil {
		return false
	}
	ddl = strings.ToLower(strings.TrimSpace(ddl))
	return strings.Contains(ddl, "bigint(20)")
}

// prepareLegacySQLiteNotNullNulls repairs two classes of legacy-data/schema
// mismatches before the first AutoMigrate on a legacy SQLite (MySQL-dump)
// database:
//
//  1. A NULL value in a column the current model declares NOT NULL. When
//     AutoMigrate recreates the table to tighten the column, the data copy hits
//     a NOT NULL constraint and startup aborts. Backfill the NULLs to the model
//     default (or a type-appropriate zero) so the copy succeeds.
//  2. A missing column the model declares NOT NULL without a default, on a
//     table that already has rows. SQLite rejects `ADD COLUMN ... NOT NULL`
//     without a constant default; pre-create the column as nullable, backfill a
//     zero value, and let AutoMigrate tighten it.
func prepareLegacySQLiteNotNullNulls(db *gorm.DB, models []any) error {
	for _, model := range models {
		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(model); err != nil {
			return err
		}
		table := stmt.Schema.Table
		if table == "" || !db.Migrator().HasTable(table) {
			continue
		}
		colTypes, err := db.Migrator().ColumnTypes(model)
		if err != nil {
			return err
		}
		existing := map[string]bool{}
		nullable := map[string]bool{}
		for _, ct := range colTypes {
			name := ct.Name()
			existing[name] = true
			if n, ok := ct.Nullable(); ok {
				nullable[name] = n
			}
		}
		hasRows := sqliteTableHasRows(db, table)
		for _, field := range stmt.Schema.Fields {
			if !field.NotNull || field.DBName == "" {
				continue
			}
			col := field.DBName
			if !existing[col] {
				if !hasRows {
					continue
				}
				colType := sqliteLegacyColumnType(field)
				if err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, col, colType)).Error; err != nil {
					return err
				}
				if err := db.Exec(fmt.Sprintf("UPDATE %s SET %s = ? WHERE %s IS NULL", table, col, col), sqliteLegacyZeroValue(field)).Error; err != nil {
					return err
				}
				existing[col] = true
				nullable[col] = true
				continue
			}
			if n, ok := nullable[col]; ok && !n {
				continue // already NOT NULL; nothing to backfill
			}
			if err := db.Exec(fmt.Sprintf("UPDATE %s SET %s = ? WHERE %s IS NULL", table, col, col), sqliteLegacyDefaultValue(field)).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func sqliteTableHasRows(db *gorm.DB, table string) bool {
	var count int64
	if err := db.Table(table).Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}

// sqliteLegacyColumnType maps a GORM field's DataType to a SQLite column type
// suitable for ALTER TABLE ADD COLUMN.
func sqliteLegacyColumnType(field *schema.Field) string {
	switch field.DataType {
	case "int", "bigint", "integer", "bool", "boolean":
		return "integer"
	case "string":
		if field.Size > 0 {
			return fmt.Sprintf("varchar(%d)", field.Size)
		}
		return "text"
	case "time", "datetime", "date":
		return "datetime"
	case "decimal", "numeric":
		return "numeric"
	default:
		return "text"
	}
}

// sqliteLegacyZeroValue returns the value used to backfill a newly added or
// legacy-NULL column when the model carries no explicit default.
func sqliteLegacyZeroValue(field *schema.Field) any {
	switch field.DataType {
	case "int", "bigint", "integer", "bool", "boolean":
		return 0
	case "time", "datetime", "date":
		return time.Unix(0, 0).UTC().Format("2006-01-02 15:04:05")
	default:
		return ""
	}
}

// sqliteLegacyDefaultValue returns the model default for a column, falling back
// to a type-appropriate zero when the model has none.
func sqliteLegacyDefaultValue(field *schema.Field) any {
	if field.HasDefaultValue && field.DefaultValueInterface != nil {
		return field.DefaultValueInterface
	}
	return sqliteLegacyZeroValue(field)
}

// normalizeSQLiteBigIntPrimaryKeys rewrites legacy MySQL-dump tables whose id
// column was captured as bigint(20) NOT NULL into a native INTEGER PRIMARY KEY
// AUTOINCREMENT form, preserving data, indexes, triggers and foreign keys.
//
// SQLite forbids switching `PRAGMA foreign_keys` inside a transaction, and the
// table rebuild (DROP + RENAME) would violate immediate-mode foreign keys on a
// referenced parent table. The rebuild therefore runs on a single dedicated
// connection: foreign keys are disabled outside the transaction on that same
// connection, the whole rebuild runs inside one transaction on it, and the
// previous pragma state is always restored afterwards — including on failure.
func normalizeSQLiteBigIntPrimaryKeys(db *gorm.DB) error {
	ctx := context.Background()
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Read the connection's current foreign-key enforcement so we can restore
	// exactly what we found, not a hardcoded ON.
	fkRestore := "ON"
	var fkBefore sql.NullString
	if err := conn.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&fkBefore); err == nil && fkBefore.Valid && strings.EqualFold(strings.TrimSpace(fkBefore.String), "0") {
		fkRestore = "OFF"
	}
	if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
		return err
	}
	restored := false
	restoreFK := func() {
		if restored {
			return
		}
		restored = true
		// Restore even if the connection is being torn down: SQLite keeps the
		// setting on the pooled connection otherwise.
		_, _ = conn.ExecContext(ctx, "PRAGMA foreign_keys = "+fkRestore)
	}
	defer restoreFK()

	type tableDDL struct {
		Name string
		SQL  string
	}
	var tables []tableDDL
	tableRows, err := conn.QueryContext(ctx, "SELECT name, sql FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'")
	if err != nil {
		return err
	}
	for tableRows.Next() {
		var t tableDDL
		if err := tableRows.Scan(&t.Name, &t.SQL); err != nil {
			tableRows.Close()
			return err
		}
		tables = append(tables, t)
	}
	tableRows.Close()
	if err := tableRows.Err(); err != nil {
		return err
	}

	pkLineRE := regexp.MustCompile(`(?mi)^\s*PRIMARY KEY\s*\("id"\)\s*,?\s*$`)
	// Only strip a column-list comma right before the closing paren of the
	// statement. Anchoring at the absolute end avoids touching ", ...)" inside a
	// parenthesised type (e.g. decimal(10, 2)) or default expression.
	trailingCommaRE := regexp.MustCompile(`,\s*\)\s*\z`)
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
		if t.Name == "vps_instances" {
			orderItemNotNullRE := regexp.MustCompile(`(?i)("order_item_id"\s+[^,\n]+)\s+NOT\s+NULL`)
			newDDL = orderItemNotNullRE.ReplaceAllString(newDDL, "$1")
		}

		cols, err := sqliteTableColumns(conn, ctx, t.Name)
		if err != nil {
			return err
		}
		if len(cols) == 0 {
			continue
		}
		colNames := make([]string, 0, len(cols))
		for _, c := range cols {
			colNames = append(colNames, fmt.Sprintf(`"%s"`, c))
		}
		colList := strings.Join(colNames, ", ")

		schemaObjects, err := sqliteSchemaObjects(conn, ctx, t.Name)
		if err != nil {
			return err
		}

		// Run the whole rebuild in one transaction on the FK-disabled dedicated
		// connection so a mid-rebuild failure rolls back the schema changes.
		tx, err := conn.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		rebuildFails := func(stepErr error) error {
			_ = tx.Rollback()
			return stepErr
		}
		if _, err := tx.ExecContext(ctx, newDDL); err != nil {
			return rebuildFails(err)
		}
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO "%s" (%s) SELECT %s FROM "%s"`, tmpName, colList, colList, t.Name)); err != nil {
			return rebuildFails(err)
		}
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`DROP TABLE "%s"`, t.Name)); err != nil {
			return rebuildFails(err)
		}
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`ALTER TABLE "%s" RENAME TO "%s"`, tmpName, t.Name)); err != nil {
			return rebuildFails(err)
		}
		for _, object := range schemaObjects {
			if _, err := tx.ExecContext(ctx, object); err != nil {
				return rebuildFails(err)
			}
		}
		if err := tx.Commit(); err != nil {
			// The transaction may have been rolled back by the driver; make sure
			// the pragma restore in defer still runs.
			return err
		}
	}
	return nil
}

func sqliteTableColumns(conn *sql.Conn, ctx context.Context, table string) ([]string, error) {
	rows, err := conn.QueryContext(ctx, fmt.Sprintf(`PRAGMA table_info("%s")`, table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var cid int
		var name, ctype string
		var notNull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notNull, &dflt, &pk); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

func sqliteSchemaObjects(conn *sql.Conn, ctx context.Context, table string) ([]string, error) {
	rows, err := conn.QueryContext(ctx, `SELECT sql FROM sqlite_master WHERE tbl_name = ? AND type IN ('index', 'trigger') AND sql IS NOT NULL ORDER BY type, name`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var sqlText sql.NullString
		if err := rows.Scan(&sqlText); err != nil {
			return nil, err
		}
		if sqlText.Valid && strings.TrimSpace(sqlText.String) != "" {
			out = append(out, sqlText.String)
		}
	}
	return out, rows.Err()
}

// normalizeLegacySQLiteUniqueIndexData resolves rows that would violate a
// unique index the current models add on the first legacy AutoMigrate. Keeping
// the earliest row (MIN(id)) and removing later duplicates mirrors the
// vps_instances order_item_id reconciliation: the migration must not abort on
// `constraint failed` when the historical schema allowed duplicates.
func normalizeLegacySQLiteUniqueIndexData(db *gorm.DB) error {
	if db.Migrator().HasTable("provision_jobs") {
		type dupGroup struct {
			OrderItemID int64
			Cnt         int64
		}
		var groups []dupGroup
		if err := db.Model(&provisionJobRow{}).
			Select("order_item_id, COUNT(*) AS cnt").
			Group("order_item_id").
			Having("COUNT(*) > 1").
			Scan(&groups).Error; err != nil {
			return err
		}
		removed := 0
		for _, g := range groups {
			var keepID int64
			if err := db.Model(&provisionJobRow{}).
				Where("order_item_id = ?", g.OrderItemID).
				Order("id ASC").
				Limit(1).
				Pluck("id", &keepID).Error; err != nil {
				return err
			}
			res := db.Where("order_item_id = ? AND id <> ?", g.OrderItemID, keepID).Delete(&provisionJobRow{})
			if res.Error != nil {
				return res.Error
			}
			removed += int(res.RowsAffected)
		}
		if len(groups) > 0 {
			log.Printf("migration: provision_jobs duplicates normalized: %d group(s), %d row(s) removed", len(groups), removed)
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

// repairTimestampNulls backfills NULL created_at/updated_at columns introduced
// by legacy MySQL dumps. It probes for NULLs first so the UPDATE (which has no
// usable index on the IS NULL predicate) only runs when there is something to
// fix — on fresh/healthy databases every boot stays a cheap LIMIT 1 probe
// instead of a full-table scan per model.
func repairTimestampNulls(db *gorm.DB, models []any) error {
	for _, model := range models {
		if db.Migrator().HasColumn(model, "created_at") {
			if err := repairOneTimestampNull(db, model, "created_at"); err != nil {
				return err
			}
		}
		if db.Migrator().HasColumn(model, "updated_at") {
			if err := repairOneTimestampNull(db, model, "updated_at"); err != nil {
				return err
			}
		}
	}
	return nil
}

// ensureUserPhoneUniqueIndex enforces phone uniqueness at the database level
// without breaking the (very common) case of accounts that have no phone bound.
// A generated column normalizes the empty string to NULL — NULL values do not
// participate in unique indexes on SQLite, MySQL and PostgreSQL — and a unique
// index over that column makes two accounts with the same non-empty phone
// impossible, closing the register/contact-change TOCTOU and the cross-instance
// same-phone race (Issue #51 phone binding).
func ensureUserPhoneUniqueIndex(db *gorm.DB) error {
	if db.Dialector == nil {
		return nil
	}
	type duplicatePhone struct {
		Phone string `gorm:"column:phone"`
		Count int64  `gorm:"column:count"`
	}
	var duplicates []duplicatePhone
	if err := db.Model(&userRow{}).
		Select("phone, COUNT(*) AS count").
		Where("TRIM(phone) <> ''").
		Group("phone").
		Having("COUNT(*) > 1").
		Limit(10).
		Scan(&duplicates).Error; err != nil {
		return err
	}
	if len(duplicates) > 0 {
		return fmt.Errorf("users phone migration requires operator reconciliation for %d duplicate non-empty phone group(s)", len(duplicates))
	}
	columnType := "TEXT"
	if db.Dialector.Name() == "mysql" {
		columnType = "VARCHAR(32)"
	}
	if !db.Migrator().HasColumn(&userRow{}, "phone_uniq") {
		expr := "CASE WHEN TRIM(phone) = '' THEN NULL ELSE phone END"
		kind := "VIRTUAL"
		if db.Dialector.Name() == "mysql" || db.Dialector.Name() == "postgres" {
			kind = "STORED"
		}
		sql := fmt.Sprintf("ALTER TABLE users ADD COLUMN phone_uniq %s GENERATED ALWAYS AS (%s) %s", columnType, expr, kind)
		if err := db.Exec(sql).Error; err != nil {
			return err
		}
	}
	const indexName = "uniq_users_phone"
	if db.Migrator().HasIndex(&userRow{}, indexName) {
		return nil
	}
	return db.Exec("CREATE UNIQUE INDEX " + indexName + " ON users(phone_uniq)").Error
}

func repairOneTimestampNull(db *gorm.DB, model any, column string) error {
	// Existence probe with LIMIT 1 short-circuits on the first NULL row instead
	// of a COUNT(*) that still scans the whole table (P3-60).
	var probe struct {
		One int
	}
	err := db.Model(model).Select("1 AS one").Where(column + " IS NULL").Limit(1).Take(&probe).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return db.Model(model).
		Where(column+" IS NULL").
		UpdateColumn(column, clause.Expr{SQL: "CURRENT_TIMESTAMP"}).Error
}

// mysqlPartialUniqueIndexes lists the model indexes whose GORM partial
// (WHERE ...) uniqueness cannot be expressed on MySQL. GORM's base migrator
// drops the WHERE clause, so AutoMigrate would otherwise emit a full-column
// UNIQUE index, which either aborts startup on legacy rows sharing empty/zero
// sentinel keys or is destroyed by this rewrite on the next restart. The
// entries are kept in non-unique form.
var mysqlPartialUniqueIndexes = []struct {
	table     string
	name      string
	cols      []string
	createSQL string
}{
	{"goods_types", "idx_goods_types_code_unique", []string{"code"}, "CREATE INDEX idx_goods_types_code_unique ON goods_types(code)"},
	{"plan_groups", "idx_plan_groups_gt_line_unique", []string{"goods_type_id", "line_id"}, "CREATE INDEX idx_plan_groups_gt_line_unique ON plan_groups(goods_type_id, line_id)"},
	{"packages", "idx_packages_gt_product_unique", []string{"goods_type_id", "plan_group_id", "product_id"}, "CREATE INDEX idx_packages_gt_product_unique ON packages(goods_type_id, plan_group_id, product_id)"},
	{"packages", "idx_packages_integration", []string{"integration_package_id"}, "CREATE INDEX idx_packages_integration ON packages(integration_package_id)"},
}

// mysqlIndexRewritePlan returns whether a partial-unique index name needs a
// destructive rewrite given its current state. Once the non-unique form is in
// place the rewrite is a no-op, so repeated startups do not emit DDL.
func mysqlIndexRewritePlan(indexExists, indexUnique bool) (needDrop, needCreate bool) {
	if indexExists && !indexUnique {
		return false, false
	}
	if indexExists {
		return true, true
	}
	return false, true
}

// mysqlIndexIsUnique reports whether the named index on table is currently a
// UNIQUE index (Non_unique = 0). An absent index reports false so callers skip
// the destructive rewrite. SQLite has no information_schema; its index
// uniqueness is read from the index DDL in sqlite_master, which keeps
// ensureMySQLNonUniqueIndex testable on SQLite while remaining MySQL-only in
// production (fixMySQLPartialUniqueIndexes is only invoked for the mysql
// dialect).
func mysqlIndexIsUnique(db *gorm.DB, table, indexName string) (bool, error) {
	if db.Dialector != nil && db.Dialector.Name() == "sqlite" {
		var sqlText string
		if err := db.Raw("SELECT sql FROM sqlite_master WHERE type = 'index' AND tbl_name = ? AND name = ?", table, indexName).Scan(&sqlText).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return false, nil
			}
			return false, err
		}
		// Match the DDL prefix, not the bare word "UNIQUE": our index names
		// themselves contain "unique" (e.g. idx_goods_types_code_unique).
		return strings.Contains(strings.ToUpper(strings.TrimSpace(sqlText)), "CREATE UNIQUE INDEX"), nil
	}
	type statRow struct {
		NonUnique int `gorm:"column:Non_unique"`
	}
	var rows []statRow
	if err := db.Raw(
		"SELECT Non_unique FROM information_schema.STATISTICS WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ? LIMIT 1",
		table, indexName,
	).Scan(&rows).Error; err != nil {
		return false, err
	}
	if len(rows) == 0 {
		return false, nil
	}
	return rows[0].NonUnique == 0, nil
}

// ensureMySQLNonUniqueIndex makes sure an index with the given name exists in
// non-unique form, dropping a UNIQUE index with the same name first. It is
// idempotent: when the non-unique form is already in place nothing runs.
func ensureMySQLNonUniqueIndex(db *gorm.DB, table, indexName, createSQL string) error {
	indexExists := db.Migrator().HasIndex(table, indexName)
	indexUnique := false
	if indexExists {
		var err error
		indexUnique, err = mysqlIndexIsUnique(db, table, indexName)
		if err != nil {
			return err
		}
	}
	needDrop, needCreate := mysqlIndexRewritePlan(indexExists, indexUnique)
	if needDrop {
		// GORM's DropIndex emits the dialect-correct DDL (MySQL `DROP INDEX x ON
		// t`, SQLite `DROP INDEX x`), so the idempotency path stays portable and
		// testable on SQLite.
		if err := db.Migrator().DropIndex(table, indexName); err != nil {
			return err
		}
	}
	if needCreate {
		return db.Exec(createSQL).Error
	}
	return nil
}

// ensureMySQLPartialUniqueIndexesBeforeMigrate runs before AutoMigrate so the
// partial-unique index names already exist in non-unique form on pre-existing
// tables. AutoMigrate matches indexes by name, so a pre-existing non-unique
// index prevents it from emitting a full-column UNIQUE index that would fail on
// legacy rows sharing empty/zero sentinel keys.
func ensureMySQLPartialUniqueIndexesBeforeMigrate(db *gorm.DB) error {
	for _, idx := range mysqlPartialUniqueIndexes {
		if !db.Migrator().HasTable(idx.table) {
			continue
		}
		colsReady := true
		for _, col := range idx.cols {
			if !db.Migrator().HasColumn(idx.table, col) {
				colsReady = false
				break
			}
		}
		if !colsReady {
			continue
		}
		if err := ensureMySQLNonUniqueIndex(db, idx.table, idx.name, idx.createSQL); err != nil {
			return err
		}
	}
	return nil
}

// MySQL does not support partial unique indexes. We keep the same index names but make them non-unique.
func fixMySQLPartialUniqueIndexes(db *gorm.DB) error {
	for _, idx := range mysqlPartialUniqueIndexes {
		if err := ensureMySQLNonUniqueIndex(db, idx.table, idx.name, idx.createSQL); err != nil {
			return err
		}
	}
	// Obsolete unique index from an earlier model version. The model no longer
	// declares the name, so AutoMigrate never recreates it; dropping it once is
	// enough.
	if db.Migrator().HasIndex("packages", "idx_packages_gt_integration_unique") {
		if err := db.Exec("DROP INDEX idx_packages_gt_integration_unique ON packages").Error; err != nil {
			return err
		}
	}
	return nil
}

// mysqlTextColumns lists the model columns that must be LONGTEXT NOT NULL on
// MySQL. The model fields either declare type:longtext or (for conditions_json)
// rely on MySQL's default longtext mapping for an unsized string.
var mysqlTextColumns = []struct{ table, column string }{
	{"cms_blocks", "content_json"},
	{"cms_blocks", "custom_html"},
	{"cms_posts", "content_html"},
	{"user_tier_auto_rules", "conditions_json"},
}

func mysqlColumnIsNullable(db *gorm.DB, table, column string) (bool, error) {
	type colRow struct {
		IsNullable string `gorm:"column:IS_NULLABLE"`
	}
	var rows []colRow
	if err := db.Raw(
		"SELECT IS_NULLABLE FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ? LIMIT 1",
		table, column,
	).Scan(&rows).Error; err != nil {
		return false, err
	}
	if len(rows) == 0 {
		return false, nil
	}
	return rows[0].IsNullable == "YES", nil
}

func mysqlColumnIsLongText(db *gorm.DB, table, column string) (bool, error) {
	type colRow struct {
		DataType string `gorm:"column:DATA_TYPE"`
	}
	var rows []colRow
	if err := db.Raw(
		"SELECT DATA_TYPE FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ? LIMIT 1",
		table, column,
	).Scan(&rows).Error; err != nil {
		return false, err
	}
	if len(rows) == 0 {
		return false, nil
	}
	return strings.EqualFold(rows[0].DataType, "longtext"), nil
}

// prepareMySQLTextColumnsBeforeMigrate backfills NULLs in the text columns that
// AutoMigrate will upgrade to LONGTEXT NOT NULL. MySQL rejects MODIFY ... NOT
// NULL when legacy rows contain NULL, which would abort startup before the
// post-migrate fix runs.
func prepareMySQLTextColumnsBeforeMigrate(db *gorm.DB) error {
	for _, c := range mysqlTextColumns {
		if !db.Migrator().HasColumn(c.table, c.column) {
			continue
		}
		nullable, err := mysqlColumnIsNullable(db, c.table, c.column)
		if err != nil {
			return err
		}
		if !nullable {
			continue
		}
		if err := db.Exec(fmt.Sprintf("UPDATE %s SET %s = '' WHERE %s IS NULL", c.table, c.column, c.column)).Error; err != nil {
			return err
		}
	}
	return nil
}

func fixMySQLTextColumns(db *gorm.DB) error {
	for _, c := range mysqlTextColumns {
		if !db.Migrator().HasColumn(c.table, c.column) {
			continue
		}
		isLongText, err := mysqlColumnIsLongText(db, c.table, c.column)
		if err != nil {
			return err
		}
		if isLongText {
			continue
		}
		// Safety net for a column AutoMigrate left untouched (e.g. a table that
		// predates the type:longtext model tags); NULLs were already backfilled
		// by the pre-migrate pass.
		if err := db.Exec(fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s LONGTEXT NOT NULL", c.table, c.column)).Error; err != nil {
			return err
		}
	}
	return nil
}
