package admin_test

import (
	"context"
	"testing"

	appadmin "xiaoheiplay/internal/app/admin"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func TestAdminService_CreateUpdateDeleteAdmin(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	group := domain.PermissionGroup{Name: "admin-group", PermissionsJSON: `["*"]`}
	if err := repo.CreatePermissionGroup(context.Background(), &group); err != nil {
		t.Fatalf("create group: %v", err)
	}
	adminSvc := appadmin.NewService(repo, repo, repo, repo, repo, repo, repo)

	admin, err := adminSvc.CreateAdmin(context.Background(), 0, "admin1", "admin1@example.com", "", "pass", &group.ID)
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}
	if admin.Role != domain.UserRoleAdmin {
		t.Fatalf("expected admin role")
	}

	dupUser := testutil.CreateUser(t, repo, "dup", "dup@example.com", "pass")
	if err := adminSvc.UpdateAdmin(context.Background(), 0, admin.ID, dupUser.Username, admin.Email, "", admin.PermissionGroupID); err != appshared.ErrConflict {
		t.Fatalf("expected conflict, got %v", err)
	}

	if err := adminSvc.DeleteAdmin(context.Background(), admin.ID, admin.ID); err == nil {
		t.Fatalf("expected cannot delete self error")
	}
}
