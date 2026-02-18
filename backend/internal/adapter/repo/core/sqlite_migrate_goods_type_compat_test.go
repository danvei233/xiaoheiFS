package repo

import (
	"testing"

	"xiaoheiplay/internal/pkg/config"
	"xiaoheiplay/internal/pkg/db"
)

func TestMigrateSQLite_OldCatalogSchemaWithoutGoodsTypeID(t *testing.T) {
	dir := t.TempDir()
	conn, err := db.Open(config.Config{DBType: "sqlite", DBPath: dir + "/test.db"})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer conn.SQL.Close()

	// Simulate an old schema where regions/plan_groups/packages existed without goods_type_id columns.
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS regions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			active INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS plan_groups (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			region_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			line_id INTEGER NOT NULL DEFAULT 0,
			unit_core INTEGER NOT NULL DEFAULT 1,
			unit_mem INTEGER NOT NULL DEFAULT 1,
			unit_disk INTEGER NOT NULL DEFAULT 1,
			unit_bw INTEGER NOT NULL DEFAULT 1,
			active INTEGER NOT NULL DEFAULT 1,
			visible INTEGER NOT NULL DEFAULT 1,
			capacity_remaining INTEGER NOT NULL DEFAULT -1,
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS packages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			plan_group_id INTEGER NOT NULL,
			product_id INTEGER NOT NULL DEFAULT 0,
			name TEXT NOT NULL,
			cores INTEGER NOT NULL DEFAULT 1,
			memory_gb INTEGER NOT NULL DEFAULT 1,
			disk_gb INTEGER NOT NULL DEFAULT 10,
			bandwidth_mbps INTEGER NOT NULL DEFAULT 10,
			cpu_model TEXT NOT NULL DEFAULT '',
			monthly_price INTEGER NOT NULL DEFAULT 100,
			port_num INTEGER NOT NULL DEFAULT 30,
			sort_order INTEGER NOT NULL DEFAULT 0,
			active INTEGER NOT NULL DEFAULT 1,
			visible INTEGER NOT NULL DEFAULT 1,
			capacity_remaining INTEGER NOT NULL DEFAULT -1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
	}
	for _, s := range stmts {
		if _, err := conn.SQL.Exec(s); err != nil {
			t.Fatalf("seed old schema: %v", err)
		}
	}

	if err := Migrate(conn.Gorm); err != nil {
		t.Fatalf("migrate: %v", err)
	}
}
