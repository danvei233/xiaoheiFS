package usecase_test

import (
	"context"
	"testing"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/usecase"
)

func TestAdminService_CreateUpdateDeleteAdmin(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	group := domain.PermissionGroup{Name: "admin-group", PermissionsJSON: `["*"]`}
	if err := repo.CreatePermissionGroup(context.Background(), &group); err != nil {
		t.Fatalf("create group: %v", err)
	}
	adminSvc := usecase.NewAdminService(repo, repo, repo, repo, repo, repo, repo)

	admin, err := adminSvc.CreateAdmin(context.Background(), 0, "admin1", "admin1@example.com", "", "pass", &group.ID)
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}
	if admin.Role != domain.UserRoleAdmin {
		t.Fatalf("expected admin role")
	}

	dupUser := testutil.CreateUser(t, repo, "dup", "dup@example.com", "pass")
	if err := adminSvc.UpdateAdmin(context.Background(), 0, admin.ID, dupUser.Username, admin.Email, "", admin.PermissionGroupID); err != usecase.ErrConflict {
		t.Fatalf("expected conflict, got %v", err)
	}

	if err := adminSvc.DeleteAdmin(context.Background(), admin.ID, admin.ID); err == nil {
		t.Fatalf("expected cannot delete self error")
	}
}
