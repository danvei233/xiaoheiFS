package usecase_test

import (
	"context"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/usecase"
)

func TestAdminService_UserOrderAndSettingsOps(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()
	svc := usecase.NewAdminService(repo, repo, repo, repo, repo, repo, repo)

	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	admin := domain.User{
		Username:     "adminops",
		Email:        "adminops@example.com",
		PasswordHash: string(hash),
		Role:         domain.UserRoleAdmin,
		Status:       domain.UserStatusActive,
	}
	if err := repo.CreateUser(ctx, &admin); err != nil {
		t.Fatalf("create admin: %v", err)
	}

	user, err := svc.CreateUser(ctx, admin.ID, domain.User{Username: "u1", Email: "u1@example.com"}, "u1pass")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, err := svc.GetUser(ctx, user.ID); err != nil {
		t.Fatalf("get user: %v", err)
	}
	if _, total, err := svc.ListUsers(ctx, 10, 0); err != nil || total == 0 {
		t.Fatalf("list users: %v", err)
	}
	user.Email = "u1+2@example.com"
	if err := svc.UpdateUser(ctx, admin.ID, user); err != nil {
		t.Fatalf("update user: %v", err)
	}
	if err := svc.UpdateUserStatus(ctx, admin.ID, user.ID, domain.UserStatusDisabled); err != nil {
		t.Fatalf("update user status: %v", err)
	}
	if err := svc.ResetUserPassword(ctx, admin.ID, user.ID, "newpass"); err != nil {
		t.Fatalf("reset user password: %v", err)
	}

	order := domain.Order{UserID: user.ID, OrderNo: "ORD-OPS-1", Status: domain.OrderStatusPendingPayment, TotalAmount: 100, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	if err := repo.CreateOrderItems(ctx, []domain.OrderItem{{OrderID: order.ID, Amount: 100, Status: domain.OrderItemStatusPendingPayment, Action: "create", SpecJSON: "{}"}}); err != nil {
		t.Fatalf("create order items: %v", err)
	}
	if _, total, err := svc.ListOrders(ctx, usecase.OrderFilter{}, 10, 0); err != nil || total == 0 {
		t.Fatalf("list orders: %v", err)
	}
	if err := svc.DeleteOrder(ctx, admin.ID, order.ID); err != nil {
		t.Fatalf("delete order: %v", err)
	}
	approvedOrder := domain.Order{UserID: user.ID, OrderNo: "ORD-OPS-2", Status: domain.OrderStatusApproved, TotalAmount: 100, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &approvedOrder); err != nil {
		t.Fatalf("create approved order: %v", err)
	}
	if err := svc.DeleteOrder(ctx, admin.ID, approvedOrder.ID); err != usecase.ErrConflict {
		t.Fatalf("delete approved order should conflict: %v", err)
	}

	raw, apiKey, err := svc.CreateAPIKey(ctx, admin.ID, "api1", nil, []string{"order.view"})
	if err != nil || apiKey.ID == 0 || raw == "" {
		t.Fatalf("create api key: %v", err)
	}
	if _, total, err := svc.ListAPIKeys(ctx, 10, 0); err != nil || total == 0 {
		t.Fatalf("list api keys: %v", err)
	}
	if err := svc.UpdateAPIKeyStatus(ctx, admin.ID, apiKey.ID, domain.APIKeyStatusDisabled); err != nil {
		t.Fatalf("update api key status: %v", err)
	}

	if err := svc.UpdateSetting(ctx, admin.ID, "site_name", "Demo"); err != nil {
		t.Fatalf("update setting: %v", err)
	}
	if items, err := svc.ListSettings(ctx); err != nil || len(items) == 0 {
		t.Fatalf("list settings: %v", err)
	}

	tmpl := domain.EmailTemplate{Name: "welcome", Subject: "Hello", Body: "Body", Enabled: true}
	if err := svc.UpsertEmailTemplate(ctx, admin.ID, &tmpl); err != nil {
		t.Fatalf("upsert email template: %v", err)
	}
	if items, err := svc.ListEmailTemplates(ctx); err != nil || len(items) == 0 {
		t.Fatalf("list email templates: %v", err)
	}

	group := domain.PermissionGroup{Name: "ops", PermissionsJSON: `["*"]`}
	if err := repo.CreatePermissionGroup(ctx, &group); err != nil {
		t.Fatalf("create permission group: %v", err)
	}
	if _, err := svc.GetPermissionGroup(ctx, group.ID); err != nil {
		t.Fatalf("get permission group: %v", err)
	}
	if items, err := svc.ListPermissionGroups(ctx); err != nil || len(items) == 0 {
		t.Fatalf("list permission groups: %v", err)
	}

	if err := repo.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: admin.ID, Action: "user.create", TargetType: "user", TargetID: "1", DetailJSON: "{}"}); err != nil {
		t.Fatalf("add audit log: %v", err)
	}
	if _, total, err := svc.ListAuditLogs(ctx, 10, 0); err != nil || total == 0 {
		t.Fatalf("list audit logs: %v", err)
	}
}

func TestCatalogService_UpdateDeleteOps(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	cycle := domain.BillingCycle{Name: "monthly", Months: 1, Multiplier: 1, MinQty: 1, MaxQty: 12, Active: true, SortOrder: 1}
	if err := repo.CreateBillingCycle(ctx, &cycle); err != nil {
		t.Fatalf("create billing cycle: %v", err)
	}
	img := domain.SystemImage{ImageID: 1, Name: "Ubuntu", Type: "linux", Enabled: true}
	if err := repo.CreateSystemImage(ctx, &img); err != nil {
		t.Fatalf("create image: %v", err)
	}

	svc := usecase.NewCatalogService(repo, repo, repo)
	cycle.Name = "monthly2"
	if err := svc.UpdateBillingCycle(ctx, cycle); err != nil {
		t.Fatalf("update billing cycle: %v", err)
	}
	if err := svc.DeleteBillingCycle(ctx, cycle.ID); err != nil {
		t.Fatalf("delete billing cycle: %v", err)
	}
	img.Name = "Ubuntu2"
	if err := svc.UpdateSystemImage(ctx, img); err != nil {
		t.Fatalf("update system image: %v", err)
	}
	if err := svc.DeleteSystemImage(ctx, img.ID); err != nil {
		t.Fatalf("delete system image: %v", err)
	}
}

func TestMessageCenterService_MarkAndNotify(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()
	svc := usecase.NewMessageCenterService(repo, repo)

	user := domain.User{
		Username:     "msg",
		Email:        "msg@example.com",
		PasswordHash: "hash",
		Role:         domain.UserRoleUser,
		Status:       domain.UserStatusActive,
	}
	if err := repo.CreateUser(ctx, &user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := repo.CreateNotification(ctx, &domain.Notification{
		UserID:  user.ID,
		Type:    "info",
		Title:   "t",
		Content: "c",
	}); err != nil {
		t.Fatalf("create notification: %v", err)
	}
	if err := svc.MarkAllRead(ctx, user.ID); err != nil {
		t.Fatalf("mark all read: %v", err)
	}
	if err := svc.NotifyUsers(ctx, []int64{user.ID}, "notice", "Title", "Body"); err != nil {
		t.Fatalf("notify users: %v", err)
	}
}
