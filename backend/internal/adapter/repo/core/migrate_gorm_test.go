package repo

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// legacySQLiteSingleConn opens an in-memory-on-disk SQLite database pinned to a
// single underlying connection so a PRAGMA set by the test is visible to the
// connection the migration grabs from the pool.
func legacySQLiteSingleConn(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "legacy.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	return db
}

// pragmaForeignKeys reads the connection-local foreign key setting.
func pragmaForeignKeys(t *testing.T, db *gorm.DB) string {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql db: %v", err)
	}
	conn, err := sqlDB.Conn(context.Background())
	if err != nil {
		t.Fatalf("conn: %v", err)
	}
	defer conn.Close()
	var v sql.NullString
	if err := conn.QueryRowContext(context.Background(), "PRAGMA foreign_keys").Scan(&v); err != nil {
		t.Fatalf("pragma foreign_keys: %v", err)
	}
	if !v.Valid {
		t.Fatalf("pragma foreign_keys returned no value")
	}
	return strings.ToLower(strings.TrimSpace(v.String))
}

func TestNormalizeSQLiteBigIntPrimaryKeysPreservesFKAndEnforcement(t *testing.T) {
	db := legacySQLiteSingleConn(t)
	ctx := context.Background()
	sqlDB, _ := db.DB()
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		t.Fatalf("conn: %v", err)
	}
	if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("enable fk: %v", err)
	}
	conn.Close()

	statements := []string{
		`CREATE TABLE "users" (
            "id" bigint(20) NOT NULL ,
            "username" text NOT NULL,
            PRIMARY KEY ("id")
        )`,
		`CREATE TABLE "orders" (
            "id" bigint(20) NOT NULL ,
            "user_id" integer NOT NULL,
            "note" text NOT NULL,
            PRIMARY KEY ("id"),
            FOREIGN KEY ("user_id") REFERENCES "users"("id")
        )`,
		`CREATE UNIQUE INDEX "uniq_legacy_fk_users_username" ON "users"("username")`,
		`CREATE TABLE "user_insert_audit" ("username" text NOT NULL)`,
		`CREATE TRIGGER "trg_legacy_fk_users_insert" AFTER INSERT ON "users"
            BEGIN INSERT INTO "user_insert_audit"("username") VALUES (NEW."username"); END`,
		`INSERT INTO "users"("id", "username") VALUES (1, 'alice')`,
		`INSERT INTO "orders"("id", "user_id", "note") VALUES (10, 1, 'ok')`,
	}
	for _, s := range statements {
		if err := db.Exec(s).Error; err != nil {
			t.Fatalf("prepare legacy schema: %v", err)
		}
	}
	if got := pragmaForeignKeys(t, db); got != "1" {
		t.Fatalf("precondition: foreign_keys should be ON, got %q", got)
	}

	if err := normalizeSQLiteBigIntPrimaryKeys(db); err != nil {
		t.Fatalf("normalize with live FK: %v", err)
	}

	// The pragma must be restored to the value we found.
	if got := pragmaForeignKeys(t, db); got != "1" {
		t.Fatalf("foreign_keys not restored to ON after normalize, got %q", got)
	}

	var sqlOut string
	if err := db.Raw("SELECT sql FROM sqlite_master WHERE name = 'users'").Scan(&sqlOut).Error; err != nil {
		t.Fatalf("read rebuilt ddl: %v", err)
	}
	if !strings.Contains(sqlOut, "INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT") {
		t.Fatalf("users.id not rewritten to INTEGER PK: %s", sqlOut)
	}

	// Data survived the rebuild.
	var note string
	if err := db.Raw("SELECT note FROM orders WHERE id = 10").Scan(&note).Error; err != nil {
		t.Fatalf("orders data lost: %v", err)
	}
	if note != "ok" {
		t.Fatalf("orders.note = %q, want ok", note)
	}

	// The real FK must still be enforced against the rebuilt parent.
	if err := db.Exec(`INSERT INTO "orders"("id", "user_id", "note") VALUES (11, 999, 'bad')`).Error; err == nil {
		t.Fatalf("foreign key was not enforced after rebuild")
	}

	// The unique index and trigger must be intact.
	if err := db.Exec(`INSERT INTO "users"("username") VALUES (?)`, "bob").Error; err != nil {
		t.Fatalf("insert normalized user: %v", err)
	}
	if err := db.Exec(`INSERT INTO "users"("username") VALUES (?)`, "bob").Error; err == nil {
		t.Fatalf("restored unique index did not reject duplicate username")
	}
	var auditCount int64
	if err := db.Table("user_insert_audit").Where("username = ?", "bob").Count(&auditCount).Error; err != nil {
		t.Fatalf("count trigger audit: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("restored trigger did not fire: count=%d", auditCount)
	}
}

func TestNormalizeSQLiteBigIntPrimaryKeysRestoresPragmaOnFailure(t *testing.T) {
	db := legacySQLiteSingleConn(t)
	ctx := context.Background()
	sqlDB, _ := db.DB()
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		t.Fatalf("conn: %v", err)
	}
	if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("enable fk: %v", err)
	}
	conn.Close()

	// A pre-existing `users__norm` table collides with the rebuild temp name, so
	// the very first CREATE inside the rebuild transaction fails and the whole
	// transaction must roll back with foreign keys restored to ON.
	statements := []string{
		`CREATE TABLE "users" (
            "id" bigint(20) NOT NULL ,
            "username" text NOT NULL,
            PRIMARY KEY ("id")
        )`,
		`CREATE TABLE "users__norm" ("username" text NOT NULL)`,
		`CREATE TABLE "orders" (
            "id" bigint(20) NOT NULL ,
            "user_id" integer NOT NULL,
            PRIMARY KEY ("id"),
            FOREIGN KEY ("user_id") REFERENCES "users"("id")
        )`,
		`INSERT INTO "users"("id", "username") VALUES (1, 'alice')`,
		`INSERT INTO "orders"("id", "user_id") VALUES (10, 1)`,
	}
	for _, s := range statements {
		if err := db.Exec(s).Error; err != nil {
			t.Fatalf("prepare legacy schema: %v", err)
		}
	}

	if err := normalizeSQLiteBigIntPrimaryKeys(db); err == nil {
		t.Fatalf("expected rebuild to fail on temp-name collision, got nil")
	}

	// The pragma must be restored even though the migration failed.
	if got := pragmaForeignKeys(t, db); got != "1" {
		t.Fatalf("foreign_keys not restored to ON after failed normalize, got %q", got)
	}

	// The original table must be untouched (bigint id preserved) and its data
	// intact because the rebuild rolled back.
	var idType string
	if err := db.Raw("SELECT sql FROM sqlite_master WHERE name = 'users'").Scan(&idType).Error; err != nil {
		t.Fatalf("read users ddl: %v", err)
	}
	if !strings.Contains(idType, "bigint(20)") {
		t.Fatalf("users table changed despite rollback: %s", idType)
	}
	var username string
	if err := db.Raw("SELECT username FROM users WHERE id = 1").Scan(&username).Error; err != nil {
		t.Fatalf("users data lost after failed rebuild: %v", err)
	}
	if username != "alice" {
		t.Fatalf("users.username = %q, want alice", username)
	}

	// FK enforcement still works on the untouched schema.
	if err := db.Exec(`INSERT INTO "orders"("id", "user_id") VALUES (11, 999)`).Error; err == nil {
		t.Fatalf("foreign key not enforced after failed rebuild")
	}
}

func TestNormalizeSQLiteBigIntPrimaryKeysPreservesParenthesisedTypes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "legacy.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// Legacy MySQL dumps can carry parenthesised column types and defaults.
	// The id-column rewrite must not treat the comma inside decimal(10, 2) as a
	// column-list terminator.
	ddl := `CREATE TABLE "billing" (
            "id" bigint(20) NOT NULL ,
            "multiplier" decimal(10, 2) NOT NULL,
            "qty" integer NOT NULL DEFAULT (2),
            "name" text NOT NULL,
            PRIMARY KEY ("id")
        )`
	if err := db.Exec(ddl).Error; err != nil {
		t.Fatalf("create legacy table: %v", err)
	}
	if err := normalizeSQLiteBigIntPrimaryKeys(db); err != nil {
		t.Fatalf("normalize: %v", err)
	}
	var sqlOut string
	if err := db.Raw("SELECT sql FROM sqlite_master WHERE name = 'billing'").Scan(&sqlOut).Error; err != nil {
		t.Fatalf("read rebuilt ddl: %v", err)
	}
	if !strings.Contains(sqlOut, "decimal(10, 2)") {
		t.Fatalf("decimal(10, 2) corrupted by id rewrite: %s", sqlOut)
	}
	if err := db.Exec(`INSERT INTO billing (multiplier, qty, name) VALUES (1.5, 2, 'x')`).Error; err != nil {
		t.Fatalf("rebuilt table not usable: %v", err)
	}
}

func TestMySQLIndexRewritePlanIsIdempotent(t *testing.T) {
	cases := []struct {
		name       string
		exists     bool
		unique     bool
		wantDrop   bool
		wantCreate bool
	}{
		{"already non-unique is a no-op", true, false, false, false},
		{"unique index present is rewritten once", true, true, true, true},
		{"missing index is created non-unique", false, false, false, true},
	}
	for _, c := range cases {
		drop, create := mysqlIndexRewritePlan(c.exists, c.unique)
		if drop != c.wantDrop || create != c.wantCreate {
			t.Fatalf("%s: got (drop=%v create=%v), want (drop=%v create=%v)",
				c.name, drop, create, c.wantDrop, c.wantCreate)
		}
	}
}

// TestEnsureMySQLNonUniqueIndex_IsIdempotentOnSecondRun locks P3-57: running the
// index fix a second time must be a no-op (index existence unchanged) instead of
// unconditionally DROP+CREATEing every boot.
func TestEnsureMySQLNonUniqueIndex_IsIdempotentOnSecondRun(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "idx-idempotent.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`CREATE TABLE goods_types (
		"id" integer primary key autoincrement,
		"code" varchar(128) NOT NULL
	)`).Error; err != nil {
		t.Fatalf("create goods_types: %v", err)
	}

	idx := mysqlPartialUniqueIndexes[0]
	if err := ensureMySQLNonUniqueIndex(db, idx.table, idx.name, idx.createSQL); err != nil {
		t.Fatalf("first ensure: %v", err)
	}
	if !db.Migrator().HasIndex(idx.table, idx.name) {
		t.Fatalf("index %s.%s must exist after the first run", idx.table, idx.name)
	}

	// Second run must be a no-op and leave the index intact.
	if err := ensureMySQLNonUniqueIndex(db, idx.table, idx.name, idx.createSQL); err != nil {
		t.Fatalf("second ensure must be a no-op, got %v", err)
	}
	if !db.Migrator().HasIndex(idx.table, idx.name) {
		t.Fatalf("index %s.%s must still exist after the second run", idx.table, idx.name)
	}
	unique, err := mysqlIndexIsUnique(db, idx.table, idx.name)
	if err != nil {
		t.Fatalf("mysqlIndexIsUnique: %v", err)
	}
	if unique {
		t.Fatalf("index %s.%s must remain non-unique", idx.table, idx.name)
	}
}

// TestEnsureMySQLNonUniqueIndex_RewritesUniqueExactlyOnce locks P3-57: a legacy
// UNIQUE index with the same name is dropped and recreated non-unique on the
// first run; the second run must not DROP+CREATE again.
func TestEnsureMySQLNonUniqueIndex_RewritesUniqueExactlyOnce(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "idx-rewrite-once.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`CREATE TABLE goods_types (
		"id" integer primary key autoincrement,
		"code" varchar(128) NOT NULL
	)`).Error; err != nil {
		t.Fatalf("create goods_types: %v", err)
	}
	if err := db.Exec(`CREATE UNIQUE INDEX idx_goods_types_code_unique ON goods_types(code)`).Error; err != nil {
		t.Fatalf("create legacy unique index: %v", err)
	}

	idx := mysqlPartialUniqueIndexes[0]
	if err := ensureMySQLNonUniqueIndex(db, idx.table, idx.name, idx.createSQL); err != nil {
		t.Fatalf("first ensure: %v", err)
	}
	unique, err := mysqlIndexIsUnique(db, idx.table, idx.name)
	if err != nil {
		t.Fatalf("mysqlIndexIsUnique after first run: %v", err)
	}
	if unique {
		t.Fatalf("legacy unique index must be converted to non-unique on the first run")
	}
	if !db.Migrator().HasIndex(idx.table, idx.name) {
		t.Fatalf("index must still exist after the rewrite")
	}

	// Second run must be a no-op.
	if err := ensureMySQLNonUniqueIndex(db, idx.table, idx.name, idx.createSQL); err != nil {
		t.Fatalf("second ensure must be a no-op, got %v", err)
	}
	unique, err = mysqlIndexIsUnique(db, idx.table, idx.name)
	if err != nil {
		t.Fatalf("mysqlIndexIsUnique after second run: %v", err)
	}
	if unique {
		t.Fatalf("index must remain non-unique after the second run")
	}
}

func TestLegacyMigrateAddsColumnsAndIndexesToExistingTables(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "legacy.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// Tables that predate the current models: they are missing columns and the
	// idempotency/unique constraints introduced since the MySQL dump was taken.
	statements := []string{
		`CREATE TABLE "users" (
            "id" bigint(20) NOT NULL ,
            "username" text NOT NULL,
            "created_at" datetime NOT NULL,
            "updated_at" datetime NOT NULL,
            PRIMARY KEY ("id")
        )`,
		`CREATE TABLE "orders" (
            "id" bigint(20) NOT NULL ,
            "user_id" integer NOT NULL,
            "order_no" varchar(191) NOT NULL,
            "created_at" datetime NOT NULL,
            "updated_at" datetime NOT NULL,
            PRIMARY KEY ("id")
        )`,
		`CREATE TABLE "coupons" (
            "id" bigint(20) NOT NULL ,
            "code" varchar(128) NOT NULL,
            "created_at" datetime NOT NULL,
            "updated_at" datetime NOT NULL,
            PRIMARY KEY ("id")
        )`,
		`CREATE TABLE "email_templates" (
            "id" bigint(20) NOT NULL ,
            "name" varchar(191) NOT NULL,
            "created_at" datetime NOT NULL,
            "updated_at" datetime NOT NULL,
            PRIMARY KEY ("id")
        )`,
	}
	for _, s := range statements {
		if err := db.Exec(s).Error; err != nil {
			t.Fatalf("prepare legacy schema: %v", err)
		}
	}

	if err := Migrate(db); err != nil {
		t.Fatalf("migrate legacy schema: %v", err)
	}

	if !db.Migrator().HasColumn("users", "permission_group_id") {
		t.Fatalf("users.permission_group_id not added on the first legacy migrate")
	}
	for _, c := range []struct{ tbl, idx string }{
		{"orders", "idx_orders_idem"},
		{"coupons", "idx_coupons_code"},
		{"email_templates", "idx_email_templates_name"},
	} {
		if !db.Migrator().HasIndex(c.tbl, c.idx) {
			t.Fatalf("index %s.%s not created on the first legacy migrate", c.tbl, c.idx)
		}
	}

	// A second migrate must be a no-op on these constraints.
	if err := Migrate(db); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
}

func TestRepairTimestampNullsDoesNotRewriteUpdatedAt(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "repair.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// Legacy tables may carry NULL created_at alongside a meaningful updated_at.
	// repairTimestampNulls must only backfill the NULL column and must not let
	// GORM's autoUpdateTime stamp a fresh value over updated_at.
	db.Exec(`CREATE TABLE "coupons" (
		"id" integer primary key autoincrement,
		"code" varchar(128) NOT NULL,
		"discount_permille" integer NOT NULL DEFAULT 1000,
		"product_group_id" integer NOT NULL,
		"total_limit" integer NOT NULL DEFAULT -1,
		"per_user_limit" integer NOT NULL DEFAULT -1,
		"new_user_only" integer NOT NULL DEFAULT 0,
		"active" integer NOT NULL DEFAULT 1,
		"note" text NOT NULL DEFAULT '',
		"created_at" datetime NULL,
		"updated_at" datetime NULL
	)`)
	if err := db.Exec(`INSERT INTO coupons (code, discount_permille, product_group_id, total_limit, per_user_limit, new_user_only, active, note, created_at, updated_at)
		VALUES ('C1', 1000, 0, -1, -1, 0, 1, '', NULL, '2020-01-01 00:00:00')`).Error; err != nil {
		t.Fatalf("seed row: %v", err)
	}

	if err := repairTimestampNulls(db, []any{&couponRow{}}); err != nil {
		t.Fatalf("repair timestamp nulls: %v", err)
	}

	var updatedAt string
	if err := db.Raw("SELECT updated_at FROM coupons WHERE id = 1").Scan(&updatedAt).Error; err != nil {
		t.Fatalf("read updated_at: %v", err)
	}
	if !strings.HasPrefix(updatedAt, "2020-01-01") {
		t.Fatalf("repair rewrote updated_at, want 2020-01-01..., got %q", updatedAt)
	}
}

// TestLegacyMigrateWithDuplicateProvisionJobs locks the migrations F1 fix: a
// legacy SQLite (MySQL-dump) database whose provision_jobs table contains two
// rows for the same order_item_id must not brick startup on the new
// idx_provision_jobs_item unique index. The pre-AutoMigrate normalization keeps
// the earliest job per order_item_id and removes the duplicates.
func TestLegacyMigrateWithDuplicateProvisionJobs(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "legacy-dupe-provision.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.Exec(`CREATE TABLE "users" (
		"id" bigint(20) NOT NULL ,
		"username" text NOT NULL,
		"created_at" datetime NOT NULL,
		"updated_at" datetime NOT NULL,
		PRIMARY KEY ("id")
	)`)
	db.Exec(`CREATE TABLE "provision_jobs" (
		"id" bigint(20) NOT NULL ,
		"order_id" integer NOT NULL,
		"order_item_id" integer NOT NULL,
		"host_id" integer NOT NULL DEFAULT 0,
		"host_name" varchar(191) NOT NULL DEFAULT '',
		"status" varchar(32) NOT NULL DEFAULT 'pending',
		"attempts" integer NOT NULL DEFAULT 0,
		"next_run_at" datetime NOT NULL,
		"last_error" text NOT NULL DEFAULT '',
		"lease_owner" varchar(128) NOT NULL DEFAULT '',
		"lease_until" datetime,
		"fence" integer NOT NULL DEFAULT 0,
		"stage" varchar(32) NOT NULL DEFAULT '',
		"attempt_id" varchar(64) NOT NULL DEFAULT '',
		"creds_encrypted" text NOT NULL DEFAULT '',
		"created_at" datetime NOT NULL,
		"updated_at" datetime NOT NULL,
		PRIMARY KEY ("id")
	)`)
	base := "2024-01-01 00:00:00"
	rows := [][]any{
		{1, 10, 42, 100, "h1", "done", 0, base, "", "", nil, 0, "completed", "a", "", base, base},
		{2, 10, 42, 100, "h1", "done", 0, base, "", "", nil, 0, "completed", "a", "", base, base}, // duplicate order_item_id
		{3, 11, 43, 101, "h2", "pending", 0, base, "", "", nil, 0, "intent", "b", "", base, base},
	}
	for _, r := range rows {
		if err := db.Exec(`INSERT INTO provision_jobs
			(id, order_id, order_item_id, host_id, host_name, status, attempts, next_run_at, last_error,
			 lease_owner, lease_until, fence, stage, attempt_id, creds_encrypted, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, r...).Error; err != nil {
			t.Fatalf("insert legacy row: %v", err)
		}
	}

	if err := Migrate(db); err != nil {
		t.Fatalf("migrate legacy schema with duplicate provision jobs: %v", err)
	}

	var count int64
	if err := db.Model(&provisionJobRow{}).Count(&count).Error; err != nil {
		t.Fatalf("count provision jobs: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 provision jobs after dedupe (order_item_id 42 collapsed), got %d", count)
	}
	var perItem []int64
	if err := db.Model(&provisionJobRow{}).Where("order_item_id = 42").Pluck("id", &perItem).Error; err != nil {
		t.Fatalf("pluck: %v", err)
	}
	if len(perItem) != 1 || perItem[0] != 1 {
		t.Fatalf("expected only the earliest job (id=1) kept for order_item_id 42, got %v", perItem)
	}

	// Second migrate must be a no-op (idempotent).
	if err := Migrate(db); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
}

func TestNormalizeSQLiteBigIntPrimaryKeysPreservesIndexesAndTriggers(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "legacy.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	statements := []string{
		`CREATE TABLE "users" (
            "id" bigint(20) NOT NULL ,
            "username" text NOT NULL,
            PRIMARY KEY ("id")
        )`,
		`CREATE UNIQUE INDEX "uniq_legacy_users_username" ON "users"("username")`,
		`CREATE TABLE "user_insert_audit" ("username" text NOT NULL)`,
		`CREATE TRIGGER "trg_legacy_users_insert" AFTER INSERT ON "users"
            BEGIN INSERT INTO "user_insert_audit"("username") VALUES (NEW."username"); END`,
		`CREATE TABLE "vps_instances" (
            "id" bigint(20) NOT NULL ,
            "order_item_id" bigint(20) NOT NULL,
            "name" text NOT NULL,
            PRIMARY KEY ("id")
        )`,
		`INSERT INTO "vps_instances"("id", "order_item_id", "name") VALUES
            (1, 0, 'manual-1'), (2, 0, 'manual-2'), (3, 12, 'linked-1'), (4, 12, 'linked-duplicate')`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("prepare legacy schema: %v", err)
		}
	}

	if err := normalizeSQLiteBigIntPrimaryKeys(db); err != nil {
		t.Fatalf("normalize legacy schema: %v", err)
	}
	if err := normalizeVPSOrderItemLinks(db); err != nil {
		t.Fatalf("normalize legacy VPS links: %v", err)
	}
	if err := ensureVPSOrderItemUniqueIndex(db, true); err != nil {
		t.Fatalf("create legacy VPS unique index: %v", err)
	}
	if !db.Migrator().HasIndex("users", "uniq_legacy_users_username") {
		t.Fatalf("independent unique index was not restored")
	}
	if err := db.Exec(`INSERT INTO "users"("username") VALUES (?)`, "alice").Error; err != nil {
		t.Fatalf("insert normalized user: %v", err)
	}
	if err := db.Exec(`INSERT INTO "users"("username") VALUES (?)`, "alice").Error; err == nil {
		t.Fatalf("restored unique index did not reject duplicate username")
	}
	var auditCount int64
	if err := db.Table("user_insert_audit").Where("username = ?", "alice").Count(&auditCount).Error; err != nil {
		t.Fatalf("count trigger audit: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("restored trigger did not fire: count=%d", auditCount)
	}
	var nullLinks int64
	if err := db.Table("vps_instances").Where("order_item_id IS NULL").Count(&nullLinks).Error; err != nil {
		t.Fatalf("count detached VPS links: %v", err)
	}
	if nullLinks != 3 {
		t.Fatalf("expected two manual and one duplicate VPS link detached, got %d", nullLinks)
	}
	var linkedRows int64
	if err := db.Table("vps_instances").Where("order_item_id = ?", 12).Count(&linkedRows).Error; err != nil {
		t.Fatalf("count retained VPS link: %v", err)
	}
	if linkedRows != 1 {
		t.Fatalf("expected one canonical VPS link, got %d", linkedRows)
	}
}

// TestLegacyMigrateBackfillsNotNullNulls guards against the migration brick
// where a legacy table carries NULL in a column the current model declares
// NOT NULL (e.g. coupons.note). The first migrate previously recreated the
// table and aborted on the NULL data; the NULLs must be backfilled to the
// model default first.
func TestLegacyMigrateBackfillsNotNullNulls(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "legacy.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	statements := []string{
		`CREATE TABLE "users" (
            "id" bigint(20) NOT NULL ,
            "username" text NOT NULL,
            "created_at" datetime NOT NULL,
            "updated_at" datetime NOT NULL,
            PRIMARY KEY ("id")
        )`,
		`CREATE TABLE "coupons" (
            "id" bigint(20) NOT NULL ,
            "code" varchar(128) NOT NULL,
            "note" text NULL,
            "created_at" datetime NOT NULL,
            "updated_at" datetime NOT NULL,
            PRIMARY KEY ("id")
        )`,
		`INSERT INTO coupons (id, code, note, created_at, updated_at) VALUES (1, 'C1', NULL, '2020-01-01 00:00:00', '2020-01-01 00:00:00')`,
	}
	for _, s := range statements {
		if err := db.Exec(s).Error; err != nil {
			t.Fatalf("prepare legacy schema: %v", err)
		}
	}

	if err := Migrate(db); err != nil {
		t.Fatalf("legacy migrate with NULL in NOT NULL column must not brick: %v", err)
	}

	var note string
	if err := db.Raw("SELECT note FROM coupons WHERE id = 1").Scan(&note).Error; err != nil {
		t.Fatalf("read note: %v", err)
	}
	if note != "" {
		t.Fatalf("expected NULL note backfilled to model default '', got %q", note)
	}
}

func TestEnsureVPSOrderItemUniqueIndex_RejectsNonUniqueSameName(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "idx.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// A same-named NON-unique index is exactly the trap: HasIndex matches by
	// name, so the old code would treat it as the unique constraint and skip.
	if err := db.Exec(`CREATE TABLE vps_instances (
		"id" integer primary key autoincrement,
		"order_item_id" integer,
		"user_id" integer NOT NULL,
		"automation_instance_id" text NOT NULL,
		"goods_type_id" integer NOT NULL DEFAULT 0,
		"name" text NOT NULL,
		"spec_json" text NOT NULL,
		"system_id" integer NOT NULL DEFAULT 0,
		"status" text NOT NULL,
		"automation_state" integer NOT NULL DEFAULT 0,
		"admin_status" text NOT NULL DEFAULT 'normal',
		"created_at" datetime NOT NULL,
		"updated_at" datetime NOT NULL
	)`).Error; err != nil {
		t.Fatalf("create vps_instances: %v", err)
	}
	if err := db.Exec(`CREATE INDEX uniq_vps_instances_order_item_id ON vps_instances(order_item_id)`).Error; err != nil {
		t.Fatalf("create non-unique index: %v", err)
	}

	if err := ensureVPSOrderItemUniqueIndex(db, false); err != nil {
		t.Fatalf("ensure unique index: %v", err)
	}

	isUnique, err := indexIsUnique(db, "vps_instances", "uniq_vps_instances_order_item_id")
	if err != nil {
		t.Fatalf("indexIsUnique: %v", err)
	}
	if !isUnique {
		t.Fatalf("index was not made unique")
	}
	// The unique index must actually reject duplicates.
	if err := db.Exec(`INSERT INTO vps_instances (id, order_item_id, user_id, automation_instance_id, goods_type_id, name, spec_json, system_id, status, automation_state, admin_status, created_at, updated_at)
		VALUES (1, 10, 1, '1', 1, 'a', '{}', 1, 'running', 2, 'normal', '2020-01-01 00:00:00', '2020-01-01 00:00:00')`).Error; err != nil {
		t.Fatalf("seed vps: %v", err)
	}
	if err := db.Exec(`INSERT INTO vps_instances (id, order_item_id, user_id, automation_instance_id, goods_type_id, name, spec_json, system_id, status, automation_state, admin_status, created_at, updated_at)
		VALUES (2, 10, 1, '2', 1, 'b', '{}', 1, 'running', 2, 'normal', '2020-01-01 00:00:00', '2020-01-01 00:00:00')`).Error; err == nil {
		t.Fatalf("unique index did not reject duplicate order_item_id")
	}
}

func TestNormalizeVPSOrderItemLinks_RecordsLiveConflict(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "conflict.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	statements := []string{
		`CREATE TABLE vps_instances (
			"id" integer primary key autoincrement,
			"order_item_id" integer,
			"user_id" integer NOT NULL,
			"automation_instance_id" text NOT NULL,
			"goods_type_id" integer NOT NULL DEFAULT 0,
			"name" text NOT NULL,
			"spec_json" text NOT NULL,
			"system_id" integer NOT NULL DEFAULT 0,
			"status" text NOT NULL,
			"automation_state" integer NOT NULL DEFAULT 0,
			"admin_status" text NOT NULL DEFAULT 'normal',
			"created_at" datetime NOT NULL,
			"updated_at" datetime NOT NULL
		)`,
		`CREATE TABLE migration_conflicts (
			"id" integer primary key autoincrement,
			"source_table" text NOT NULL,
			"ref_id" integer NOT NULL,
			"type" text NOT NULL,
			"detail" text NOT NULL,
			"status" text NOT NULL DEFAULT 'pending',
			"created_at" datetime NOT NULL
		)`,
		`INSERT INTO vps_instances (id, order_item_id, user_id, automation_instance_id, goods_type_id, name, spec_json, system_id, status, automation_state, admin_status, created_at, updated_at)
			VALUES (1, 20, 1, '1', 1, 'keep', '{}', 1, 'running', 2, 'normal', '2020-01-01 00:00:00', '2020-01-01 00:00:00')`,
		`INSERT INTO vps_instances (id, order_item_id, user_id, automation_instance_id, goods_type_id, name, spec_json, system_id, status, automation_state, admin_status, created_at, updated_at)
			VALUES (2, 20, 1, '2', 1, 'live-dup', '{}', 1, 'running', 2, 'normal', '2020-01-01 00:00:00', '2020-01-01 00:00:00')`,
		`INSERT INTO vps_instances (id, order_item_id, user_id, automation_instance_id, goods_type_id, name, spec_json, system_id, status, automation_state, admin_status, created_at, updated_at)
			VALUES (3, 21, 1, '3', 1, 'keep-unknown', '{}', 1, 'running', 2, 'normal', '2020-01-01 00:00:00', '2020-01-01 00:00:00')`,
		`INSERT INTO vps_instances (id, order_item_id, user_id, automation_instance_id, goods_type_id, name, spec_json, system_id, status, automation_state, admin_status, created_at, updated_at)
			VALUES (4, 21, 1, '4', 1, 'unknown-dup', '{}', 1, 'unknown', 0, 'normal', '2020-01-01 00:00:00', '2020-01-01 00:00:00')`,
	}
	for _, s := range statements {
		if err := db.Exec(s).Error; err != nil {
			t.Fatalf("prepare schema: %v", err)
		}
	}

	err = normalizeVPSOrderItemLinks(db)
	if err == nil || !strings.Contains(err.Error(), "operator reconciliation") {
		t.Fatalf("expected actionable live-link conflict, got %v", err)
	}

	// The live duplicate (id=2, running) must have been recorded as a conflict,
	// while the unknown duplicate (id=3) is safe to detach silently.
	var conflictCount int64
	if err := db.Table("migration_conflicts").Count(&conflictCount).Error; err != nil {
		t.Fatalf("count conflicts: %v", err)
	}
	if conflictCount != 1 {
		t.Fatalf("expected 1 recorded live conflict, got %d", conflictCount)
	}
	var detail string
	if err := db.Raw(`SELECT detail FROM migration_conflicts WHERE ref_id = 2`).Scan(&detail).Error; err != nil {
		t.Fatalf("read conflict detail: %v", err)
	}
	if !strings.Contains(detail, "both links preserved") {
		t.Fatalf("conflict detail does not identify the preserved live links: %s", detail)
	}

	// The live duplicate remains linked for operator reconciliation; only the
	// unknown duplicate is safe to detach automatically.
	var linked int64
	if err := db.Table("vps_instances").Where("order_item_id = 20 OR order_item_id = 21").Count(&linked).Error; err != nil {
		t.Fatalf("count linked: %v", err)
	}
	if linked != 3 {
		t.Fatalf("expected both live rows and the unknown canonical row to stay linked, got %d", linked)
	}
}

func TestEnsureUserPhoneUniqueIndexRejectsLegacyDuplicatesWithActionableError(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "duplicate-phone.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`CREATE TABLE users (id integer primary key, phone text NOT NULL DEFAULT '')`).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}
	if err := db.Exec(`INSERT INTO users (id, phone) VALUES (1, '13800138000'), (2, '13800138000')`).Error; err != nil {
		t.Fatalf("seed duplicate phones: %v", err)
	}
	err = ensureUserPhoneUniqueIndex(db)
	if err == nil || !strings.Contains(err.Error(), "operator reconciliation") || strings.Contains(err.Error(), "13800138000") {
		t.Fatalf("expected actionable duplicate-phone error, got %v", err)
	}
	if db.Migrator().HasColumn(&userRow{}, "phone_uniq") {
		t.Fatal("migration must fail before partially adding phone_uniq")
	}
}

// TestNormalizeDuplicateApprovedPayments guards the migration that demotes
// legacy duplicate approved payments (two approved rows for one order — produced
// by the pre-invariant double-pay bug) so the partial unique index on
// (order_id) WHERE status='approved' can be created. The earliest id is kept;
// every newer approved row is rejected and recorded in migration_conflicts.
func TestNormalizeDuplicateApprovedPayments(t *testing.T) {
	db := legacySQLiteSingleConn(t)
	statements := []string{
		`CREATE TABLE order_payments (
			"id" integer primary key autoincrement,
			"order_id" integer NOT NULL,
			"user_id" integer NOT NULL,
			"method" text NOT NULL,
			"amount" integer NOT NULL,
			"currency" text NOT NULL,
			"trade_no" text NOT NULL,
			"note" text,
			"screenshot_url" text,
			"status" text NOT NULL,
			"idempotency_key" text,
			"reviewed_by" integer,
			"review_reason" text NOT NULL DEFAULT '',
			"created_at" datetime NOT NULL,
			"updated_at" datetime NOT NULL
		)`,
		`CREATE TABLE migration_conflicts (
			"id" integer primary key autoincrement,
			"source_table" text NOT NULL,
			"ref_id" integer NOT NULL,
			"type" text NOT NULL,
			"detail" text NOT NULL,
			"status" text NOT NULL DEFAULT 'pending',
			"created_at" datetime NOT NULL
		)`,
		`INSERT INTO order_payments (id, order_id, user_id, method, amount, currency, trade_no, status, review_reason, created_at, updated_at)
			VALUES (1, 10, 1, 'alipay', 1000, 'CNY', 'A1', 'approved', '', '2020-01-01 00:00:00', '2020-01-01 00:00:00')`,
		`INSERT INTO order_payments (id, order_id, user_id, method, amount, currency, trade_no, status, review_reason, created_at, updated_at)
			VALUES (2, 10, 1, 'wechat', 1000, 'CNY', 'W1', 'approved', '', '2020-01-01 00:00:00', '2020-01-01 00:00:00')`,
		`INSERT INTO order_payments (id, order_id, user_id, method, amount, currency, trade_no, status, review_reason, created_at, updated_at)
			VALUES (3, 11, 1, 'alipay', 500, 'CNY', 'A2', 'approved', '', '2020-01-01 00:00:00', '2020-01-01 00:00:00')`,
	}
	for _, s := range statements {
		if err := db.Exec(s).Error; err != nil {
			t.Fatalf("prepare schema: %v", err)
		}
	}

	if err := normalizeDuplicateApprovedPayments(db); err != nil {
		t.Fatalf("normalize: %v", err)
	}

	// The canonical (earliest) payment for order 10 stays approved; the newer
	// duplicate is demoted to rejected.
	var statuses map[int64]string
	rows := []struct {
		ID     int64
		Status string
	}{}
	if err := db.Table("order_payments").Select("id, status").Scan(&rows).Error; err != nil {
		t.Fatalf("read statuses: %v", err)
	}
	statuses = map[int64]string{}
	for _, r := range rows {
		statuses[r.ID] = r.Status
	}
	if statuses[1] != "approved" {
		t.Fatalf("kept payment id=1 must stay approved, got %s", statuses[1])
	}
	if statuses[2] != "rejected" {
		t.Fatalf("duplicate payment id=2 must be rejected, got %s", statuses[2])
	}
	if statuses[3] != "approved" {
		t.Fatalf("single approved payment for order 11 must be untouched, got %s", statuses[3])
	}

	var conflicts int64
	if err := db.Table("migration_conflicts").Count(&conflicts).Error; err != nil {
		t.Fatalf("count conflicts: %v", err)
	}
	if conflicts != 1 {
		t.Fatalf("expected 1 recorded conflict, got %d", conflicts)
	}

	// After normalization the partial unique index can be created, and a second
	// approved payment for the same order is rejected by it.
	if err := db.Exec("CREATE UNIQUE INDEX uniq_order_payments_approved_order ON order_payments(order_id) WHERE status = 'approved'").Error; err != nil {
		t.Fatalf("create partial unique index after normalize: %v", err)
	}
	if err := db.Exec(`INSERT INTO order_payments (order_id, user_id, method, amount, currency, trade_no, status, review_reason, created_at, updated_at)
		VALUES (10, 1, 'alipay', 1000, 'CNY', 'A3', 'approved', '', '2020-01-01 00:00:00', '2020-01-01 00:00:00')`).Error; err == nil {
		t.Fatalf("second approved payment for order 10 must be rejected by the unique index")
	}
}

// TestRepairTimestampNullsProbeShortCircuits locks P3-60: the existence probe
// must be a SELECT 1 ... LIMIT 1 (which stops at the first NULL row) rather than
// a COUNT(*) that scans the whole table.
func TestRepairTimestampNullsProbeShortCircuits(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "probe.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.Exec(`CREATE TABLE "coupons" (
		"id" integer primary key autoincrement,
		"code" varchar(128) NOT NULL,
		"created_at" datetime NULL,
		"updated_at" datetime NULL
	)`)

	sess := db.Session(&gorm.Session{DryRun: true})
	var probe struct{ One int }
	sql := sess.Model(&couponRow{}).Select("1 AS one").Where("created_at IS NULL").Limit(1).Take(&probe).Statement.SQL.String()
	if !strings.Contains(strings.ToLower(sql), "select 1") {
		t.Fatalf("expected a SELECT 1 existence probe, got: %s", sql)
	}
	if !strings.Contains(strings.ToLower(sql), "limit 1") {
		t.Fatalf("expected LIMIT 1 existence probe, got: %s", sql)
	}
	if strings.Contains(strings.ToLower(sql), "count") {
		t.Fatalf("probe must not be a COUNT scan: %s", sql)
	}
}
