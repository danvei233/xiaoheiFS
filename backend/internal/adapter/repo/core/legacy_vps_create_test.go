package repo

import (
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"xiaoheiplay/internal/domain"
)

// TestLegacySQLiteCreateInstanceIsIdempotentByOrderItem guards the legacy
// SQLite (MySQL-dump) upgrade path: the unique index on vps_instances.order_item_id
// was created as a PARTIAL index (`WHERE order_item_id > 0`) for those databases,
// but CreateInstance issues `ON CONFLICT (order_item_id) DO NOTHING`. SQLite only
// accepts a conflict target that matches a unique constraint exactly, so the
// partial predicate made every provision insert fail with "ON CONFLICT clause
// does not match any PRIMARY KEY or UNIQUE constraint". The migration must
// rebuild the legacy index as a full unique index (safe: zero-links are nulled
// and duplicates detached by normalizeVPSOrderItemLinks before the index is
// created, and SQLite never conflicts on NULLs).
func TestLegacySQLiteCreateInstanceIsIdempotentByOrderItem(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	// Legacy MySQL-dump marker that flips isLegacySQLiteFromMySQLDump.
	if err := db.Exec(`CREATE TABLE users (id bigint(20) NOT NULL PRIMARY KEY, username varchar(191))`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&vpsInstanceRow{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	if err := ensureVPSOrderItemUniqueIndex(db, true); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// The index must be a FULL unique index (no WHERE), so it can be a conflict
	// target for ON CONFLICT (order_item_id).
	var idxSQL string
	if err := db.Raw("SELECT sql FROM sqlite_master WHERE type='index' AND name='uniq_vps_instances_order_item_id'").Scan(&idxSQL).Error; err != nil {
		t.Fatalf("read index: %v", err)
	}
	t.Logf("index SQL: %s", idxSQL)
	if strings.Contains(idxSQL, " WHERE ") {
		t.Fatalf("legacy SQLite unique index must be full, got partial: %s", idxSQL)
	}
	r := NewGormRepo(db)
	now := time.Now()
	inst := domain.VPSInstance{UserID: 1, OrderItemID: 42, AutomationInstanceID: "1001", Status: domain.VPSStatusRunning, AdminStatus: domain.VPSAdminStatusNormal, ExpireAt: &now}
	if err := r.CreateInstance(t.Context(), &inst); err != nil {
		t.Fatalf("legacy SQLite CreateInstance failed: %v", err)
	}
	if inst.ID == 0 {
		t.Fatal("CreateInstance did not assign an id")
	}
	// Second insert for the same order_item_id must be an idempotent replay,
	// not an error.
	replay := domain.VPSInstance{UserID: 1, OrderItemID: 42, AutomationInstanceID: "1001", Status: domain.VPSStatusRunning, AdminStatus: domain.VPSAdminStatusNormal, ExpireAt: &now}
	if err := r.CreateInstance(t.Context(), &replay); err != nil {
		t.Fatalf("idempotent replay failed: %v", err)
	}
	// A different user under the same order item is a genuine conflict.
	conflict := domain.VPSInstance{UserID: 2, OrderItemID: 42, AutomationInstanceID: "9999", Status: domain.VPSStatusRunning, AdminStatus: domain.VPSAdminStatusNormal, ExpireAt: &now}
	if err := r.CreateInstance(t.Context(), &conflict); err == nil {
		t.Fatal("expected ErrConflict for different user under same order_item_id")
	}
}

// TestEnsureVPSOrderItemUniqueIndex_RebuildsLegacyPartialIndex verifies that a
// database already carrying the old PARTIAL index (as created by a previous
// version on legacy SQLite) has it rebuilt as a full unique index on the next
// migration.
func TestEnsureVPSOrderItemUniqueIndex_RebuildsLegacyPartialIndex(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&vpsInstanceRow{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	// Simulate the old partial index.
	if err := db.Exec("CREATE UNIQUE INDEX uniq_vps_instances_order_item_id ON vps_instances(order_item_id) WHERE order_item_id > 0").Error; err != nil {
		t.Fatal(err)
	}
	if err := ensureVPSOrderItemUniqueIndex(db, true); err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	var idxSQL string
	if err := db.Raw("SELECT sql FROM sqlite_master WHERE type='index' AND name='uniq_vps_instances_order_item_id'").Scan(&idxSQL).Error; err != nil {
		t.Fatalf("read index: %v", err)
	}
	if strings.Contains(idxSQL, " WHERE ") {
		t.Fatalf("expected partial index rebuilt as full, got: %s", idxSQL)
	}
	// The rebuilt index must reject a duplicate order_item_id.
	if err := db.Exec(`INSERT INTO vps_instances (id, order_item_id, user_id, automation_instance_id, goods_type_id, name, spec_json, system_id, status, automation_state, admin_status, created_at, updated_at)
		VALUES (1, 10, 1, 'a', 0, 'n', '{}', 1, 'running', 2, 'normal', '2020-01-01 00:00:00', '2020-01-01 00:00:00')`).Error; err != nil {
		t.Fatalf("seed vps: %v", err)
	}
	if err := db.Exec(`INSERT INTO vps_instances (id, order_item_id, user_id, automation_instance_id, goods_type_id, name, spec_json, system_id, status, automation_state, admin_status, created_at, updated_at)
		VALUES (2, 10, 2, 'b', 0, 'n', '{}', 1, 'running', 2, 'normal', '2020-01-01 00:00:00', '2020-01-01 00:00:00')`).Error; err == nil {
		t.Fatal("full unique index did not reject duplicate order_item_id")
	}
}

// TestEnsureCouponRedemptionPerUserUniqueIndex_CreatesGateAndDedupes guards the
// (coupon_id, user_id) partial unique index on active redemptions: on SQLite it
// is the durable per-user redemption gate (FOR UPDATE is a no-op there), and the
// migration normalizes legacy duplicate active redemptions first so the index
// does not abort startup.
func TestEnsureCouponRedemptionPerUserUniqueIndex_CreatesGateAndDedupes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&couponRedemptionRow{}, &migrationConflictRow{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	// Seed a legacy duplicate: user 1 redeemed coupon 1 twice (both active).
	rows := []couponRedemptionRow{
		{CouponID: 1, UserID: 1, OrderID: 10, Status: string(domain.CouponRedemptionStatusApplied)},
		{CouponID: 1, UserID: 1, OrderID: 11, Status: string(domain.CouponRedemptionStatusConfirmed)},
		{CouponID: 1, UserID: 2, OrderID: 12, Status: string(domain.CouponRedemptionStatusApplied)},
	}
	for _, row := range rows {
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("seed redemption: %v", err)
		}
	}
	if err := ensureCouponRedemptionPerUserUniqueIndex(db); err != nil {
		t.Fatalf("ensure index: %v", err)
	}
	// The index must exist and be unique.
	isUnique, err := indexIsUnique(db, "coupon_redemptions", "uniq_coupon_redemptions_active_user")
	if err != nil {
		t.Fatalf("indexIsUnique: %v", err)
	}
	if !isUnique {
		t.Fatal("expected coupon redemption per-user index to be unique")
	}
	// Duplicate active redemption for user 1 must have been demoted to canceled.
	var canceled int64
	if err := db.Model(&couponRedemptionRow{}).Where("coupon_id = ? AND user_id = ? AND status = ?", 1, 1, domain.CouponRedemptionStatusCanceled).Count(&canceled).Error; err != nil {
		t.Fatalf("count canceled: %v", err)
	}
	if canceled != 1 {
		t.Fatalf("expected one duplicate demoted to canceled, got %d", canceled)
	}
	// A new active redemption for a fresh (coupon, user) pair works; a second
	// one for the same pair conflicts.
	if err := db.Create(&couponRedemptionRow{CouponID: 2, UserID: 1, OrderID: 13, Status: string(domain.CouponRedemptionStatusConfirmed)}).Error; err != nil {
		t.Fatalf("insert non-duplicate redemption: %v", err)
	}
	if err := db.Create(&couponRedemptionRow{CouponID: 2, UserID: 1, OrderID: 14, Status: string(domain.CouponRedemptionStatusApplied)}).Error; err == nil {
		t.Fatal("expected the partial unique index to reject a second active redemption for the same user/coupon")
	}
}
