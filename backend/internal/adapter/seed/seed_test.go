package seed

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"

	"xiaoheiplay/internal/adapter/repo"
	"xiaoheiplay/internal/pkg/config"
	"xiaoheiplay/internal/pkg/db"
)

func newSeedDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "seed.db")
	conn, err := db.Open(config.Config{DBType: "sqlite", DBPath: dbPath})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		_ = conn.SQL.Close()
	})
	if err := repo.Migrate(conn.Gorm); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return conn.SQL, conn.Dialect
}

func TestSeedIfEmpty(t *testing.T) {
	conn, dialect := newSeedDB(t)
	if err := EnsureSettings(conn, dialect); err != nil {
		t.Fatalf("ensure settings: %v", err)
	}
	if err := SeedIfEmpty(conn); err != nil {
		t.Fatalf("seed if empty: %v", err)
	}
	var count int
	if err := conn.QueryRow(`SELECT COUNT(1) FROM regions`).Scan(&count); err != nil {
		t.Fatalf("count regions: %v", err)
	}
	if count == 0 {
		t.Fatalf("expected seeded regions")
	}
}

func TestEnsureDefaultsAndCMS(t *testing.T) {
	conn, dialect := newSeedDB(t)
	if err := EnsurePermissionDefaults(conn, dialect); err != nil {
		t.Fatalf("ensure permission defaults: %v", err)
	}
	if err := EnsurePermissionGroups(conn, dialect); err != nil {
		t.Fatalf("ensure permission groups: %v", err)
	}
	if err := EnsureCMSDefaults(conn, dialect); err != nil {
		t.Fatalf("ensure cms defaults: %v", err)
	}
	var count int
	if err := conn.QueryRow(`SELECT COUNT(1) FROM cms_categories`).Scan(&count); err != nil {
		t.Fatalf("count cms categories: %v", err)
	}
	if count == 0 {
		t.Fatalf("expected cms categories")
	}
}

func TestSeedIfEmptySkip(t *testing.T) {
	conn, _ := newSeedDB(t)
	if _, err := conn.Exec(`INSERT INTO regions(code,name,active) VALUES (?,?,?)`, "area-x", "Region X", 1); err != nil {
		t.Fatalf("insert region: %v", err)
	}
	if err := SeedIfEmpty(conn); err != nil {
		t.Fatalf("seed if empty: %v", err)
	}
	var count int
	if err := conn.QueryRow(`SELECT COUNT(1) FROM regions`).Scan(&count); err != nil {
		t.Fatalf("count regions: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected unchanged regions")
	}
}

func TestEnsurePermissionGroups_BackfillDashboardRevenue(t *testing.T) {
	conn, dialect := newSeedDB(t)

	// Simulate an existing installation with older permissions for default groups.
	if _, err := conn.Exec(`INSERT INTO permission_groups(name,description,permissions_json) VALUES (?,?,?)`, "运维管理员", "old", `["dashboard.overview","dashboard.vps_status"]`); err != nil {
		t.Fatalf("insert ops group: %v", err)
	}
	if _, err := conn.Exec(`INSERT INTO permission_groups(name,description,permissions_json) VALUES (?,?,?)`, "客服管理员", "old", `["dashboard.overview"]`); err != nil {
		t.Fatalf("insert cs group: %v", err)
	}

	if err := EnsurePermissionGroups(conn, dialect); err != nil {
		t.Fatalf("ensure permission groups: %v", err)
	}

	for _, name := range []string{"运维管理员", "客服管理员"} {
		var raw string
		if err := conn.QueryRow(`SELECT permissions_json FROM permission_groups WHERE name = ?`, name).Scan(&raw); err != nil {
			t.Fatalf("select perms: %v", err)
		}
		var perms []string
		if err := json.Unmarshal([]byte(raw), &perms); err != nil {
			t.Fatalf("unmarshal perms: %v", err)
		}
		found := false
		for _, p := range perms {
			if p == "dashboard.revenue" || p == "dashboard.*" || p == "*" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("%s missing dashboard.revenue in %v", name, perms)
		}
	}
}
