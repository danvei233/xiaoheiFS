package admin_test

import (
	"context"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	appadmin "xiaoheiplay/internal/app/admin"
	appadminvps "xiaoheiplay/internal/app/adminvps"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

func TestAdminService_PermissionGroupsAndProfile(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	hash, _ := bcrypt.GenerateFromPassword([]byte("oldpass"), bcrypt.DefaultCost)
	admin := domain.User{
		Username:     "admin",
		Email:        "admin@example.com",
		PasswordHash: string(hash),
		Role:         domain.UserRoleAdmin,
		Status:       domain.UserStatusActive,
	}
	if err := repo.CreateUser(ctx, &admin); err != nil {
		t.Fatalf("create admin: %v", err)
	}

	svc := appadmin.NewService(repo, repo, repo, repo, repo, repo, repo)

	group := domain.PermissionGroup{Name: "ops", PermissionsJSON: `["order.view"]`}
	if err := svc.CreatePermissionGroup(ctx, admin.ID, &group); err != nil {
		t.Fatalf("create permission group: %v", err)
	}
	group.Name = "ops2"
	if err := svc.UpdatePermissionGroup(ctx, admin.ID, group); err != nil {
		t.Fatalf("update permission group: %v", err)
	}
	if err := svc.DeletePermissionGroup(ctx, admin.ID, group.ID); err != nil {
		t.Fatalf("delete permission group: %v", err)
	}

	if _, total, err := svc.ListAdmins(ctx, "", 10, 0); err != nil || total == 0 {
		t.Fatalf("list admins: %v", err)
	}

	if err := svc.UpdateProfile(ctx, admin.ID, "admin2@example.com", "qq1"); err != nil {
		t.Fatalf("update profile: %v", err)
	}
	if err := svc.ChangePassword(ctx, admin.ID, "oldpass", "newpass"); err != nil {
		t.Fatalf("change password: %v", err)
	}
}

func TestAdminService_UpdateProfile_SelfEmailNoConflict(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	admin := domain.User{
		Username:     "admin_self",
		Email:        "admin_self@example.com",
		PasswordHash: string(hash),
		Role:         domain.UserRoleAdmin,
		Status:       domain.UserStatusActive,
	}
	if err := repo.CreateUser(ctx, &admin); err != nil {
		t.Fatalf("create admin: %v", err)
	}
	other := domain.User{
		Username:     "other_user",
		Email:        "other@example.com",
		PasswordHash: string(hash),
		Role:         domain.UserRoleAdmin,
		Status:       domain.UserStatusActive,
	}
	if err := repo.CreateUser(ctx, &other); err != nil {
		t.Fatalf("create other: %v", err)
	}

	svc := appadmin.NewService(repo, repo, repo, repo, repo, repo, repo)

	if err := svc.UpdateProfile(ctx, admin.ID, "admin_self@example.com", "123456"); err != nil {
		t.Fatalf("self email should not conflict: %v", err)
	}
	if err := svc.UpdateProfile(ctx, admin.ID, "other@example.com", "123456"); err != appshared.ErrConflict {
		t.Fatalf("expected conflict for duplicate email, got: %v", err)
	}
}

func TestAdminVPSService_CoreOps(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	user := domain.User{
		Username:     "u1",
		Email:        "u1@example.com",
		PasswordHash: "hash",
		Role:         domain.UserRoleUser,
		Status:       domain.UserStatusActive,
	}
	if err := repo.CreateUser(ctx, &user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	automation := &usecaseTestAutomation{}
	svc := appadminvps.NewService(repo, automation, repo, repo, repo, nil)

	region := domain.Region{Code: "r1", Name: "Region1", Active: true}
	if err := repo.CreateRegion(ctx, &region); err != nil {
		t.Fatalf("create region: %v", err)
	}
	plan := domain.PlanGroup{RegionID: region.ID, Name: "Plan1", LineID: 1, UnitCore: 1, UnitMem: 1, UnitDisk: 1, UnitBW: 1, Active: true, Visible: true}
	if err := repo.CreatePlanGroup(ctx, &plan); err != nil {
		t.Fatalf("create plan group: %v", err)
	}
	pkg := domain.Package{PlanGroupID: plan.ID, Name: "Pkg1", Cores: 1, MemoryGB: 1, DiskGB: 10, BandwidthMB: 10, Monthly: 100, PortNum: 1, Active: true, Visible: true}
	if err := repo.CreatePackage(ctx, &pkg); err != nil {
		t.Fatalf("create package: %v", err)
	}
	img := domain.SystemImage{ImageID: 1, Name: "Ubuntu", Type: "linux", Enabled: true}
	if err := repo.CreateSystemImage(ctx, &img); err != nil {
		t.Fatalf("create image: %v", err)
	}
	order := domain.Order{
		UserID:      user.ID,
		OrderNo:     "ORD-1",
		Status:      domain.OrderStatusPendingPayment,
		TotalAmount: 100,
		Currency:    "CNY",
	}
	if err := repo.CreateOrder(ctx, &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	orderItems := []domain.OrderItem{{
		OrderID:   order.ID,
		PackageID: pkg.ID,
		SystemID:  img.ID,
		SpecJSON:  "{}",
		Qty:       1,
		Amount:    100,
		Status:    domain.OrderItemStatusPendingPayment,
		Action:    "create",
	}}
	if err := repo.CreateOrderItems(ctx, orderItems); err != nil {
		t.Fatalf("create order items: %v", err)
	}

	inst, err := svc.Create(ctx, 1, appshared.AdminVPSCreateInput{
		UserID:               user.ID,
		OrderItemID:          orderItems[0].ID,
		Name:                 "vps-1",
		AutomationInstanceID: "1001",
		LineID:               10,
		PackageID:            pkg.ID,
		PackageName:          pkg.Name,
		SystemID:             img.ID,
		CPU:                  1,
		MemoryGB:             1,
		DiskGB:               10,
		Provision:            false,
	})
	if err != nil {
		t.Fatalf("create vps: %v", err)
	}
	automation.hostID = 1001

	if _, err := svc.Refresh(ctx, 1, inst.ID); err != nil {
		t.Fatalf("refresh vps: %v", err)
	}
	if err := svc.SetAdminStatus(ctx, 1, inst.ID, domain.VPSAdminStatusLocked, "reason"); err != nil {
		t.Fatalf("set admin status: %v", err)
	}
	if _, err := svc.UpdateExpireAt(ctx, 1, inst.ID, time.Now().Add(24*time.Hour)); err != nil {
		t.Fatalf("update expire: %v", err)
	}
	if _, err := svc.EmergencyRenew(ctx, 1, inst.ID); err != nil {
		t.Fatalf("emergency renew: %v", err)
	}
	cpu := 2
	if err := svc.Resize(ctx, 1, inst.ID, appshared.AutomationElasticUpdateRequest{CPU: &cpu}, `{"cpu":2}`); err != nil {
		t.Fatalf("resize: %v", err)
	}
	pkgName := "pkg2"
	if _, err := svc.Update(ctx, 1, inst.ID, appshared.AdminVPSUpdateInput{PackageName: &pkgName}); err != nil {
		t.Fatalf("update vps: %v", err)
	}
	if err := svc.Delete(ctx, 1, inst.ID); err != nil {
		t.Fatalf("delete vps: %v", err)
	}
}

type usecaseTestAutomation struct {
	hostID int64
}

func (f *usecaseTestAutomation) ClientForGoodsType(ctx context.Context, goodsTypeID int64) (appshared.AutomationClient, error) {
	_ = ctx
	_ = goodsTypeID
	return f, nil
}

func (f *usecaseTestAutomation) CreateHost(ctx context.Context, req appshared.AutomationCreateHostRequest) (appshared.AutomationCreateHostResult, error) {
	if f.hostID == 0 {
		f.hostID = 1001
	}
	return appshared.AutomationCreateHostResult{HostID: f.hostID}, nil
}
func (f *usecaseTestAutomation) GetHostInfo(ctx context.Context, hostID int64) (appshared.AutomationHostInfo, error) {
	expire := time.Now().Add(24 * time.Hour)
	return appshared.AutomationHostInfo{HostID: hostID, State: 2, HostName: "host", ExpireAt: &expire}, nil
}
func (f *usecaseTestAutomation) ListHostSimple(ctx context.Context, searchTag string) ([]appshared.AutomationHostSimple, error) {
	return []appshared.AutomationHostSimple{{ID: f.hostID, HostName: searchTag}}, nil
}
func (f *usecaseTestAutomation) ElasticUpdate(ctx context.Context, req appshared.AutomationElasticUpdateRequest) error {
	return nil
}
func (f *usecaseTestAutomation) RenewHost(ctx context.Context, hostID int64, nextDueDate time.Time) error {
	return nil
}
func (f *usecaseTestAutomation) LockHost(ctx context.Context, hostID int64) error {
	return nil
}
func (f *usecaseTestAutomation) UnlockHost(ctx context.Context, hostID int64) error {
	return nil
}
func (f *usecaseTestAutomation) DeleteHost(ctx context.Context, hostID int64) error {
	return nil
}
func (f *usecaseTestAutomation) StartHost(ctx context.Context, hostID int64) error {
	return nil
}
func (f *usecaseTestAutomation) ShutdownHost(ctx context.Context, hostID int64) error {
	return nil
}
func (f *usecaseTestAutomation) RebootHost(ctx context.Context, hostID int64) error {
	return nil
}
func (f *usecaseTestAutomation) ResetOS(ctx context.Context, hostID int64, templateID int64, password string) error {
	return nil
}
func (f *usecaseTestAutomation) ResetOSPassword(ctx context.Context, hostID int64, password string) error {
	return nil
}
func (f *usecaseTestAutomation) ListSnapshots(ctx context.Context, hostID int64) ([]appshared.AutomationSnapshot, error) {
	return nil, nil
}
func (f *usecaseTestAutomation) CreateSnapshot(ctx context.Context, hostID int64) error { return nil }
func (f *usecaseTestAutomation) DeleteSnapshot(ctx context.Context, hostID int64, snapshotID int64) error {
	return nil
}
func (f *usecaseTestAutomation) RestoreSnapshot(ctx context.Context, hostID int64, snapshotID int64) error {
	return nil
}
func (f *usecaseTestAutomation) ListBackups(ctx context.Context, hostID int64) ([]appshared.AutomationBackup, error) {
	return nil, nil
}
func (f *usecaseTestAutomation) CreateBackup(ctx context.Context, hostID int64) error { return nil }
func (f *usecaseTestAutomation) DeleteBackup(ctx context.Context, hostID int64, backupID int64) error {
	return nil
}
func (f *usecaseTestAutomation) RestoreBackup(ctx context.Context, hostID int64, backupID int64) error {
	return nil
}
func (f *usecaseTestAutomation) ListFirewallRules(ctx context.Context, hostID int64) ([]appshared.AutomationFirewallRule, error) {
	return nil, nil
}
func (f *usecaseTestAutomation) AddFirewallRule(ctx context.Context, req appshared.AutomationFirewallRuleCreate) error {
	return nil
}
func (f *usecaseTestAutomation) DeleteFirewallRule(ctx context.Context, hostID int64, ruleID int64) error {
	return nil
}
func (f *usecaseTestAutomation) ListPortMappings(ctx context.Context, hostID int64) ([]appshared.AutomationPortMapping, error) {
	return nil, nil
}
func (f *usecaseTestAutomation) AddPortMapping(ctx context.Context, req appshared.AutomationPortMappingCreate) error {
	return nil
}
func (f *usecaseTestAutomation) DeletePortMapping(ctx context.Context, hostID int64, mappingID int64) error {
	return nil
}
func (f *usecaseTestAutomation) FindPortCandidates(ctx context.Context, hostID int64, keywords string) ([]int64, error) {
	return []int64{}, nil
}
func (f *usecaseTestAutomation) GetPanelURL(ctx context.Context, hostName string, panelPassword string) (string, error) {
	return "https://panel.local/" + hostName, nil
}
func (f *usecaseTestAutomation) ListAreas(ctx context.Context) ([]appshared.AutomationArea, error) {
	return []appshared.AutomationArea{}, nil
}
func (f *usecaseTestAutomation) ListImages(ctx context.Context, lineID int64) ([]appshared.AutomationImage, error) {
	return []appshared.AutomationImage{}, nil
}
func (f *usecaseTestAutomation) ListLines(ctx context.Context) ([]appshared.AutomationLine, error) {
	return []appshared.AutomationLine{}, nil
}
func (f *usecaseTestAutomation) ListProducts(ctx context.Context, lineID int64) ([]appshared.AutomationProduct, error) {
	return []appshared.AutomationProduct{}, nil
}
func (f *usecaseTestAutomation) GetMonitor(ctx context.Context, hostID int64) (appshared.AutomationMonitor, error) {
	return appshared.AutomationMonitor{CPUPercent: 10, MemoryPercent: 20}, nil
}
func (f *usecaseTestAutomation) GetVNCURL(ctx context.Context, hostID int64) (string, error) {
	return "https://vnc.local/host", nil
}
