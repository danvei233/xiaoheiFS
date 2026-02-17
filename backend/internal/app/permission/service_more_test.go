package permission_test

import (
	"context"
	"testing"
	apppermission "xiaoheiplay/internal/app/permission"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func TestPermissionService_MoreChecks(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()

	group := domain.PermissionGroup{Name: "group1", PermissionsJSON: `["order.view","order.edit"]`}
	if err := repo.CreatePermissionGroup(ctx, &group); err != nil {
		t.Fatalf("create group: %v", err)
	}
	user := domain.User{
		Username:          "permuser",
		Email:             "permuser@example.com",
		PasswordHash:      "hash",
		Role:              domain.UserRoleAdmin,
		Status:            domain.UserStatusActive,
		PermissionGroupID: &group.ID,
	}
	if err := repo.CreateUser(ctx, &user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := repo.UpsertPermission(ctx, &domain.Permission{Code: "order.view", Name: "Order View", Category: "order"}); err != nil {
		t.Fatalf("seed permission: %v", err)
	}

	svc := apppermission.NewService(repo, repo, repo)
	if ok, _ := svc.HasAnyPermission(ctx, user.ID, []string{"order.view", "order.delete"}); !ok {
		t.Fatalf("expected any permission")
	}
	if ok, _ := svc.HasAllPermissions(ctx, user.ID, []string{"order.view", "order.edit"}); !ok {
		t.Fatalf("expected all permissions")
	}
	perms, err := svc.GetUserPermissions(ctx, user.ID)
	if err != nil || len(perms) == 0 {
		t.Fatalf("expected permissions list")
	}
}
