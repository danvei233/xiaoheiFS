package repo

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

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
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("prepare legacy schema: %v", err)
		}
	}

	if err := normalizeSQLiteBigIntPrimaryKeys(db); err != nil {
		t.Fatalf("normalize legacy schema: %v", err)
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
}
