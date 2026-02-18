package repo_test

import (
	"context"
	"testing"
	"time"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func TestSQLiteRepo_CatalogSystemImagePackagePaths(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()

	region := domain.Region{Code: "r1", Name: "Region1", Active: true}
	if err := repo.CreateRegion(ctx, &region); err != nil {
		t.Fatalf("create region: %v", err)
	}
	gotRegion, err := repo.GetRegion(ctx, region.ID)
	if err != nil {
		t.Fatalf("get region: %v", err)
	}
	if gotRegion.ID != region.ID {
		t.Fatalf("region mismatch")
	}
	if regions, err := repo.ListRegions(ctx); err != nil || len(regions) == 0 {
		t.Fatalf("list regions: %v", err)
	}
	region.Name = "Region1b"
	if err := repo.UpdateRegion(ctx, region); err != nil {
		t.Fatalf("update region: %v", err)
	}

	plan := domain.PlanGroup{
		RegionID:          region.ID,
		Name:              "Plan",
		LineID:            10,
		UnitCore:          1,
		UnitMem:           1,
		UnitDisk:          1,
		UnitBW:            1,
		Active:            true,
		Visible:           true,
		CapacityRemaining: -1,
		SortOrder:         0,
	}
	if err := repo.CreatePlanGroup(ctx, &plan); err != nil {
		t.Fatalf("create plan group: %v", err)
	}
	gotPlan, err := repo.GetPlanGroup(ctx, plan.ID)
	if err != nil {
		t.Fatalf("get plan group: %v", err)
	}
	if gotPlan.ID != plan.ID || gotPlan.LineID != plan.LineID {
		t.Fatalf("plan group mismatch")
	}
	if plans, err := repo.ListPlanGroups(ctx); err != nil || len(plans) == 0 {
		t.Fatalf("list plan groups: %v", err)
	}
	plan.Name = "Plan2"
	if err := repo.UpdatePlanGroup(ctx, plan); err != nil {
		t.Fatalf("update plan group: %v", err)
	}

	pkg := domain.Package{
		PlanGroupID:       plan.ID,
		Name:              "Basic",
		Cores:             2,
		MemoryGB:          4,
		DiskGB:            40,
		BandwidthMB:       10,
		CPUModel:          "x",
		Monthly:           1000,
		PortNum:           30,
		SortOrder:         0,
		Active:            true,
		Visible:           true,
		CapacityRemaining: -1,
	}
	if err := repo.CreatePackage(ctx, &pkg); err != nil {
		t.Fatalf("create package: %v", err)
	}
	gotPkg, err := repo.GetPackage(ctx, pkg.ID)
	if err != nil {
		t.Fatalf("get package: %v", err)
	}
	if gotPkg.ID != pkg.ID || gotPkg.Name != pkg.Name {
		t.Fatalf("package mismatch")
	}
	if packages, err := repo.ListPackages(ctx); err != nil || len(packages) == 0 {
		t.Fatalf("list packages: %v", err)
	}
	pkg.Name = "Basic2"
	if err := repo.UpdatePackage(ctx, pkg); err != nil {
		t.Fatalf("update package: %v", err)
	}
	if err := repo.DeletePackage(ctx, pkg.ID); err != nil {
		t.Fatalf("delete package: %v", err)
	}

	img := domain.SystemImage{ImageID: 1, Name: "Ubuntu", Type: "linux", Enabled: true}
	if err := repo.CreateSystemImage(ctx, &img); err != nil {
		t.Fatalf("create system image: %v", err)
	}
	gotImg, err := repo.GetSystemImage(ctx, img.ID)
	if err != nil {
		t.Fatalf("get system image: %v", err)
	}
	if gotImg.ID != img.ID || gotImg.Name != img.Name {
		t.Fatalf("system image mismatch")
	}
	if err := repo.SetLineSystemImages(ctx, plan.LineID, []int64{img.ID}); err != nil {
		t.Fatalf("set line images: %v", err)
	}
	if items, err := repo.ListSystemImages(ctx, plan.LineID); err != nil || len(items) != 1 {
		t.Fatalf("list system images: %v len=%d", err, len(items))
	}
	if items, err := repo.ListAllSystemImages(ctx); err != nil || len(items) != 1 {
		t.Fatalf("list all system images: %v len=%d", err, len(items))
	}
	img.Name = "Ubuntu2"
	img.Enabled = false
	if err := repo.UpdateSystemImage(ctx, img); err != nil {
		t.Fatalf("update system image: %v", err)
	}
	if err := repo.DeleteSystemImage(ctx, img.ID); err != nil {
		t.Fatalf("delete system image: %v", err)
	}
	if err := repo.DeletePlanGroup(ctx, plan.ID); err != nil {
		t.Fatalf("delete plan group: %v", err)
	}
	if err := repo.DeleteRegion(ctx, region.ID); err != nil {
		t.Fatalf("delete region: %v", err)
	}

	captcha := domain.Captcha{ID: "captcha-1", CodeHash: "hash", ExpiresAt: time.Now().Add(time.Minute)}
	if err := repo.CreateCaptcha(ctx, captcha); err != nil {
		t.Fatalf("create captcha: %v", err)
	}
	if _, err := repo.GetCaptcha(ctx, captcha.ID); err != nil {
		t.Fatalf("get captcha: %v", err)
	}
	if err := repo.DeleteCaptcha(ctx, captcha.ID); err != nil {
		t.Fatalf("delete captcha: %v", err)
	}
}

func TestSQLiteRepo_CartVPSAndRealNamePaths(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()

	user := domain.User{
		Username:     "cartuser",
		Email:        "cartuser@example.com",
		PasswordHash: "hash",
		Role:         domain.UserRoleUser,
		Status:       domain.UserStatusActive,
	}
	if err := repo.CreateUser(ctx, &user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	cart := domain.CartItem{
		UserID:    user.ID,
		PackageID: 1,
		SystemID:  1,
		SpecJSON:  `{"cpu":1}`,
		Qty:       1,
		Amount:    990,
	}
	if err := repo.AddCartItem(ctx, &cart); err != nil {
		t.Fatalf("add cart item: %v", err)
	}
	items, err := repo.ListCartItems(ctx, user.ID)
	if err != nil || len(items) != 1 {
		t.Fatalf("list cart items: %v len=%d", err, len(items))
	}

	order := domain.Order{
		UserID:      user.ID,
		OrderNo:     "ORD-VC-1",
		Status:      domain.OrderStatusPendingPayment,
		TotalAmount: 990,
		Currency:    "CNY",
	}
	if err := repo.CreateOrder(ctx, &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	orderItems := []domain.OrderItem{{
		OrderID:  order.ID,
		Amount:   990,
		Status:   domain.OrderItemStatusPendingPayment,
		Action:   "create",
		SpecJSON: "{}",
		Qty:      1,
	}}
	if err := repo.CreateOrderItems(ctx, orderItems); err != nil {
		t.Fatalf("create order items: %v", err)
	}
	expireAt := time.Now().Add(24 * time.Hour)
	inst := domain.VPSInstance{
		UserID:               user.ID,
		OrderItemID:          orderItems[0].ID,
		AutomationInstanceID: "1001",
		Name:                 "vps-1",
		Region:               "R1",
		RegionID:             1,
		LineID:               10,
		PackageID:            1,
		PackageName:          "Basic",
		CPU:                  1,
		MemoryGB:             1,
		DiskGB:               10,
		BandwidthMB:          10,
		PortNum:              30,
		MonthlyPrice:         990,
		SpecJSON:             "{}",
		SystemID:             1,
		Status:               domain.VPSStatusRunning,
		AutomationState:      2,
		AdminStatus:          domain.VPSAdminStatusNormal,
		ExpireAt:             &expireAt,
	}
	if err := repo.CreateInstance(ctx, &inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}
	gotInst, err := repo.GetInstanceByOrderItem(ctx, orderItems[0].ID)
	if err != nil || gotInst.ID != inst.ID {
		t.Fatalf("get instance by order item: %v", err)
	}
	if err := repo.DeleteInstance(ctx, inst.ID); err != nil {
		t.Fatalf("delete instance: %v", err)
	}

	record := domain.RealNameVerification{
		UserID:     user.ID,
		RealName:   "Tester",
		IDNumber:   "1234567890123456",
		Status:     "verified",
		Provider:   "fake",
		Reason:     "",
		VerifiedAt: func() *time.Time { t := time.Now().UTC(); return &t }(),
	}
	if err := repo.CreateRealNameVerification(ctx, &record); err != nil {
		t.Fatalf("create realname: %v", err)
	}
	userID := user.ID
	records, total, err := repo.ListRealNameVerifications(ctx, &userID, 10, 0)
	if err != nil || total == 0 || len(records) == 0 {
		t.Fatalf("list realname verifications: %v total=%d", err, total)
	}
}
