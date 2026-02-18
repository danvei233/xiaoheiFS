package seed

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"gorm.io/gorm"
	"xiaoheiplay/internal/adapter/repo/core"
	"xiaoheiplay/internal/pkg/config"
	"xiaoheiplay/internal/pkg/db"
)

func newSeedDB(t *testing.T) *gorm.DB {
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
	return conn.Gorm
}

func TestSeedIfEmpty(t *testing.T) {
	gdb := newSeedDB(t)
	if err := EnsureSettings(gdb); err != nil {
		t.Fatalf("ensure settings: %v", err)
	}
	if err := SeedIfEmpty(gdb); err != nil {
		t.Fatalf("seed if empty: %v", err)
	}
	var count int64
	if err := gdb.Table("regions").Count(&count).Error; err != nil {
		t.Fatalf("count regions: %v", err)
	}
	if count == 0 {
		t.Fatalf("expected seeded regions")
	}
}

func TestEnsureDefaultsAndCMS(t *testing.T) {
	gdb := newSeedDB(t)
	if err := EnsurePermissionDefaults(gdb); err != nil {
		t.Fatalf("ensure permission defaults: %v", err)
	}
	if err := EnsurePermissionGroups(gdb); err != nil {
		t.Fatalf("ensure permission groups: %v", err)
	}
	if err := EnsureCMSDefaults(gdb); err != nil {
		t.Fatalf("ensure cms defaults: %v", err)
	}
	var count int64
	if err := gdb.Table("cms_categories").Count(&count).Error; err != nil {
		t.Fatalf("count cms categories: %v", err)
	}
	if count == 0 {
		t.Fatalf("expected cms categories")
	}
}

func TestEnsureSettings_BackfillBlankSiteNavItems(t *testing.T) {
	gdb := newSeedDB(t)
	now := time.Now()
	if err := gdb.Table("settings").Create(map[string]any{
		"key":        "site_nav_items",
		"value_json": "   ",
		"updated_at": now,
	}).Error; err != nil {
		t.Fatalf("insert blank site_nav_items: %v", err)
	}

	if err := EnsureSettings(gdb); err != nil {
		t.Fatalf("ensure settings: %v", err)
	}

	var value string
	if err := gdb.Table("settings").
		Select("value_json").
		Where("`key` = ?", "site_nav_items").
		Scan(&value).Error; err != nil {
		t.Fatalf("query site_nav_items: %v", err)
	}
	if value != "[]" {
		t.Fatalf("expected site_nav_items backfilled to [], got %q", value)
	}
}

func TestSeedIfEmptySkip(t *testing.T) {
	gdb := newSeedDB(t)
	now := time.Now()
	if err := gdb.Table("regions").Create(map[string]any{
		"goods_type_id": int64(0),
		"code":          "area-x",
		"name":          "Region X",
		"active":        1,
		"created_at":    now,
		"updated_at":    now,
	}).Error; err != nil {
		t.Fatalf("insert region: %v", err)
	}
	if err := SeedIfEmpty(gdb); err != nil {
		t.Fatalf("seed if empty: %v", err)
	}
	var count int64
	if err := gdb.Table("regions").Count(&count).Error; err != nil {
		t.Fatalf("count regions: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected unchanged regions")
	}
}

func TestEnsurePermissionGroupsBackfillDashboardRevenue(t *testing.T) {
	gdb := newSeedDB(t)

	now := time.Now()
	if err := gdb.Table("permission_groups").Create(map[string]any{
		"name":             "运维管理员",
		"description":      "old",
		"permissions_json": `["dashboard.overview","dashboard.vps_status"]`,
		"created_at":       now,
		"updated_at":       now,
	}).Error; err != nil {
		t.Fatalf("insert ops group: %v", err)
	}
	if err := gdb.Table("permission_groups").Create(map[string]any{
		"name":             "客服管理员",
		"description":      "old",
		"permissions_json": `["dashboard.overview"]`,
		"created_at":       now,
		"updated_at":       now,
	}).Error; err != nil {
		t.Fatalf("insert cs group: %v", err)
	}

	if err := EnsurePermissionGroups(gdb); err != nil {
		t.Fatalf("ensure permission groups: %v", err)
	}

	for _, name := range []string{"运维管理员", "客服管理员"} {
		var rawPerms string
		if err := gdb.Table("permission_groups").
			Select("permissions_json").
			Where("name = ?", name).
			Scan(&rawPerms).Error; err != nil {
			t.Fatalf("select perms: %v", err)
		}
		var perms []string
		if err := json.Unmarshal([]byte(rawPerms), &perms); err != nil {
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
