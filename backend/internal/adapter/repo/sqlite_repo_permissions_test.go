package repo_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"xiaoheiplay/internal/adapter/repo"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/pkg/config"
	"xiaoheiplay/internal/pkg/db"
)

func newTestRepo(t *testing.T) (*sql.DB, *repo.SQLiteRepo) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "repo.db")
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
	return conn.SQL, repo.NewSQLiteRepo(conn.Gorm)
}

func TestSQLiteRepo_PermissionsAndGroups(t *testing.T) {
	_, r := newTestRepo(t)
	ctx := context.Background()

	perm := &domain.Permission{Code: "order.view", Name: "Order View", Category: "order"}
	if err := r.UpsertPermission(ctx, perm); err != nil {
		t.Fatalf("upsert permission: %v", err)
	}
	if perm.ID == 0 {
		t.Fatalf("expected permission id")
	}
	got, err := r.GetPermissionByCode(ctx, "order.view")
	if err != nil {
		t.Fatalf("get permission: %v", err)
	}
	if got.Code != perm.Code {
		t.Fatalf("unexpected permission code: %s", got.Code)
	}
	if err := r.UpdatePermissionName(ctx, "order.view", "Order View Updated"); err != nil {
		t.Fatalf("update permission name: %v", err)
	}
	perms, err := r.ListPermissions(ctx)
	if err != nil {
		t.Fatalf("list permissions: %v", err)
	}
	if len(perms) == 0 {
		t.Fatalf("expected permissions")
	}
	if err := r.RegisterPermissions(ctx, []domain.PermissionDefinition{
		{Code: "order.view", Name: "Order View", Category: "order"},
		{Code: "order.list", Name: "Order List", Category: "order"},
	}); err != nil {
		t.Fatalf("register permissions: %v", err)
	}

	group := &domain.PermissionGroup{
		Name:            "ops",
		Description:     "ops team",
		PermissionsJSON: `["order.view"]`,
	}
	if err := r.CreatePermissionGroup(ctx, group); err != nil {
		t.Fatalf("create permission group: %v", err)
	}
	if group.ID == 0 {
		t.Fatalf("expected group id")
	}
	groups, err := r.ListPermissionGroups(ctx)
	if err != nil {
		t.Fatalf("list permission groups: %v", err)
	}
	if len(groups) == 0 {
		t.Fatalf("expected permission groups")
	}
	gotGroup, err := r.GetPermissionGroup(ctx, group.ID)
	if err != nil {
		t.Fatalf("get permission group: %v", err)
	}
	gotGroup.Description = "ops team updated"
	if err := r.UpdatePermissionGroup(ctx, gotGroup); err != nil {
		t.Fatalf("update permission group: %v", err)
	}
	if err := r.DeletePermissionGroup(ctx, group.ID); err != nil {
		t.Fatalf("delete permission group: %v", err)
	}
}
