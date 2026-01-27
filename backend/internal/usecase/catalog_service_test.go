package usecase_test

import (
	"context"
	"testing"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/usecase"
)

func TestCatalogService_CRUD(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	svc := usecase.NewCatalogService(repo, repo, repo)
	ctx := context.Background()

	region := domain.Region{Code: "r1", Name: "Region", Active: true}
	if err := svc.CreateRegion(ctx, &region); err != nil {
		t.Fatalf("create region: %v", err)
	}
	region.Name = "Region-2"
	if err := svc.UpdateRegion(ctx, region); err != nil {
		t.Fatalf("update region: %v", err)
	}
	if list, err := svc.ListRegions(ctx); err != nil || len(list) == 0 {
		t.Fatalf("list regions: %v %v", list, err)
	}

	plan := domain.PlanGroup{
		RegionID:          region.ID,
		Name:              "Plan",
		LineID:            1,
		UnitCore:          1,
		UnitMem:           1,
		UnitDisk:          1,
		UnitBW:            1,
		Active:            true,
		Visible:           true,
		CapacityRemaining: -1,
	}
	if err := svc.CreatePlanGroup(ctx, &plan); err != nil {
		t.Fatalf("create plan: %v", err)
	}
	plan.Name = "Plan-2"
	if err := svc.UpdatePlanGroup(ctx, plan); err != nil {
		t.Fatalf("update plan: %v", err)
	}
	if list, err := svc.ListPlanGroups(ctx); err != nil || len(list) == 0 {
		t.Fatalf("list plans: %v %v", list, err)
	}

	pkg := domain.Package{
		PlanGroupID:       plan.ID,
		Name:              "Basic",
		Cores:             1,
		MemoryGB:          1,
		DiskGB:            10,
		BandwidthMB:       10,
		Monthly:           1000,
		PortNum:           5,
		Active:            true,
		Visible:           true,
		CapacityRemaining: -1,
	}
	if err := svc.CreatePackage(ctx, &pkg); err != nil {
		t.Fatalf("create package: %v", err)
	}
	pkg.Name = "Basic-2"
	if err := svc.UpdatePackage(ctx, pkg); err != nil {
		t.Fatalf("update package: %v", err)
	}
	if list, err := svc.ListPackages(ctx); err != nil || len(list) == 0 {
		t.Fatalf("list packages: %v %v", list, err)
	}

	cycle := domain.BillingCycle{Name: "monthly", Months: 1, Multiplier: 1}
	if err := svc.CreateBillingCycle(ctx, &cycle); err != nil {
		t.Fatalf("create cycle: %v", err)
	}
	if list, err := svc.ListBillingCycles(ctx); err != nil || len(list) == 0 {
		t.Fatalf("list cycles: %v %v", list, err)
	}

	img := domain.SystemImage{ImageID: 1, Name: "Ubuntu", Type: "linux", Enabled: true}
	if err := svc.CreateSystemImage(ctx, &img); err != nil {
		t.Fatalf("create image: %v", err)
	}
	if images, err := svc.ListSystemImages(ctx, 0); err != nil || len(images) == 0 {
		t.Fatalf("list images: %v %v", images, err)
	}
	if err := svc.SetLineSystemImages(ctx, plan.LineID, []int64{img.ID}); err != nil {
		t.Fatalf("set line images: %v", err)
	}
	if _, err := svc.GetSystemImage(ctx, img.ID); err != nil {
		t.Fatalf("get image: %v", err)
	}

	if err := svc.DeletePackage(ctx, pkg.ID); err != nil {
		t.Fatalf("delete package: %v", err)
	}
	if err := svc.DeletePlanGroup(ctx, plan.ID); err != nil {
		t.Fatalf("delete plan: %v", err)
	}
	if err := svc.DeleteRegion(ctx, region.ID); err != nil {
		t.Fatalf("delete region: %v", err)
	}
}

func TestCatalogService_ValidateSystemImageType(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	svc := usecase.NewCatalogService(repo, repo, repo)
	img := domain.SystemImage{ImageID: 1, Name: "Bad", Type: "unknown", Enabled: true}
	if err := svc.CreateSystemImage(context.Background(), &img); err == nil {
		t.Fatalf("expected invalid image type")
	}
}
