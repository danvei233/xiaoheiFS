package usecase_test

import (
	"context"
	"testing"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/usecase"
)

func TestPermissionService_HasPermission(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	group := domain.PermissionGroup{Name: "test", PermissionsJSON: `["order.*"]`}
	if err := repo.CreatePermissionGroup(context.Background(), &group); err != nil {
		t.Fatalf("create group: %v", err)
	}
	admin := testutil.CreateAdmin(t, repo, "admin1", "a1@example.com", "pass", group.ID)
	user := testutil.CreateUser(t, repo, "user1", "u1@example.com", "pass")

	svc := usecase.NewPermissionService(repo, repo, repo)
	ok, err := svc.HasPermission(context.Background(), admin.ID, "order.view")
	if err != nil || !ok {
		t.Fatalf("expected permission ok, err=%v", err)
	}
	ok, err = svc.HasPermission(context.Background(), user.ID, "order.view")
	if err != nil || ok {
		t.Fatalf("expected non-admin denied")
	}
}

func TestPermissionService_PrimaryAdmin(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	group := domain.PermissionGroup{Name: "test2", PermissionsJSON: `["*"]`}
	if err := repo.CreatePermissionGroup(context.Background(), &group); err != nil {
		t.Fatalf("create group: %v", err)
	}
	admin1 := testutil.CreateAdmin(t, repo, "admin2", "a2@example.com", "pass", group.ID)
	admin2 := testutil.CreateAdmin(t, repo, "admin3", "a3@example.com", "pass", group.ID)

	svc := usecase.NewPermissionService(repo, repo, repo)
	isPrimary, err := svc.IsPrimaryAdmin(context.Background(), admin1.ID)
	if err != nil || !isPrimary {
		t.Fatalf("expected primary admin")
	}
	isPrimary, err = svc.IsPrimaryAdmin(context.Background(), admin2.ID)
	if err != nil || isPrimary {
		t.Fatalf("expected non-primary admin")
	}
}
