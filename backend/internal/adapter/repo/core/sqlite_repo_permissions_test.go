package repo_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"xiaoheiplay/internal/adapter/repo/core"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/pkg/config"
	"xiaoheiplay/internal/pkg/db"
)

func newTestRepo(t *testing.T) (*sql.DB, *repo.GormRepo) {
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
	return conn.SQL, repo.NewGormRepo(conn.Gorm)
}

func TestSQLiteRepo_UpdatePermissionGroupRejectsUnknownGroup(t *testing.T) {
	rawDB, r := newTestRepo(t)
	ctx := context.Background()

	missing := domain.PermissionGroup{ID: 999999, Name: "ghost", PermissionsJSON: `["order.view"]`}
	err := r.UpdatePermissionGroup(ctx, missing)
	if err == nil {
		t.Fatalf("expected update of non-existent group to fail")
	}
	if !errors.Is(err, appshared.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	var count int64
	if err := rawDB.QueryRow("SELECT COUNT(1) FROM permission_group_permissions WHERE permission_group_id = ?", missing.ID).Scan(&count); err != nil {
		t.Fatalf("count dangling join rows: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no orphan permission_group_permissions rows, got %d", count)
	}
}

func TestSQLiteRepo_DeletePermissionGroupDetachesUserAndAPIKeyRefs(t *testing.T) {
	_, r := newTestRepo(t)
	ctx := context.Background()

	group := &domain.PermissionGroup{Name: "ops", Description: "ops team", PermissionsJSON: `["order.view"]`}
	if err := r.CreatePermissionGroup(ctx, group); err != nil {
		t.Fatalf("create permission group: %v", err)
	}
	groupID := group.ID

	user := domain.User{
		Username:          "grpuser",
		Email:             "grpuser@example.com",
		PasswordHash:      "hash",
		Role:              domain.UserRoleAdmin,
		Status:            domain.UserStatusActive,
		PermissionGroupID: &groupID,
	}
	if err := r.CreateUser(ctx, &user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	key := domain.APIKey{Name: "grpkey", KeyHash: "hash-grpkey", Status: domain.APIKeyStatusActive, ScopesJSON: `["*"]`, PermissionGroupID: &groupID}
	if err := r.CreateAPIKey(ctx, &key); err != nil {
		t.Fatalf("create api key: %v", err)
	}

	if err := r.DeletePermissionGroup(ctx, groupID); err != nil {
		t.Fatalf("delete permission group: %v", err)
	}

	gotUser, err := r.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if gotUser.PermissionGroupID != nil {
		t.Fatalf("expected user.permission_group_id detached after group delete, got %d", *gotUser.PermissionGroupID)
	}
	gotKey, err := r.GetAPIKeyByHash(ctx, "hash-grpkey")
	if err != nil {
		t.Fatalf("get api key: %v", err)
	}
	if gotKey.PermissionGroupID != nil {
		t.Fatalf("expected api_key.permission_group_id detached after group delete, got %d", *gotKey.PermissionGroupID)
	}
	if _, err := r.GetPermissionGroup(ctx, groupID); err == nil {
		t.Fatalf("expected permission group to be gone")
	}
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
