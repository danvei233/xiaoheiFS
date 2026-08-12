package catalog_test

import (
	"context"
	"errors"
	"testing"

	appcatalog "xiaoheiplay/internal/app/catalog"
	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func TestCatalogService_CRUD(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	svc := appcatalog.NewService(repo, repo, repo)
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

func TestCatalogService_RejectsNegativePackagePrice(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	svc := appcatalog.NewService(repo, repo, repo)
	ctx := context.Background()

	region := domain.Region{Code: "r-neg", Name: "Region", Active: true}
	if err := svc.CreateRegion(ctx, &region); err != nil {
		t.Fatalf("create region: %v", err)
	}
	plan := domain.PlanGroup{RegionID: region.ID, Name: "Plan", Active: true, Visible: true}
	if err := svc.CreatePlanGroup(ctx, &plan); err != nil {
		t.Fatalf("create plan: %v", err)
	}

	pkg := domain.Package{PlanGroupID: plan.ID, Name: "Neg", Monthly: -1}
	if err := svc.CreatePackage(ctx, &pkg); err != appshared.ErrInvalidInput {
		t.Fatalf("expected invalid input for negative monthly, got %v", err)
	}
	pkg.Monthly = 100
	if err := svc.CreatePackage(ctx, &pkg); err != nil {
		t.Fatalf("create package: %v", err)
	}
	pkg.Monthly = -5
	if err := svc.UpdatePackage(ctx, pkg); err != appshared.ErrInvalidInput {
		t.Fatalf("expected invalid input for negative monthly on update, got %v", err)
	}
}

func TestCatalogService_RejectsInvalidBillingCycle(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	svc := appcatalog.NewService(repo, repo, repo)
	ctx := context.Background()

	cases := []struct {
		name  string
		cycle domain.BillingCycle
	}{
		{"non-positive months", domain.BillingCycle{Name: "c1", Months: 0, Multiplier: 1}},
		{"non-positive multiplier", domain.BillingCycle{Name: "c2", Months: 1, Multiplier: 0}},
		{"negative multiplier", domain.BillingCycle{Name: "c3", Months: 1, Multiplier: -0.5}},
		{"min qty above max qty", domain.BillingCycle{Name: "c4", Months: 1, Multiplier: 1, MinQty: 5, MaxQty: 2}},
	}
	for _, tc := range cases {
		if err := svc.CreateBillingCycle(ctx, &tc.cycle); err != appshared.ErrInvalidInput {
			t.Fatalf("case %q: expected invalid input, got %v", tc.name, err)
		}
	}

	// A valid cycle must still be creatable and updatable.
	cycle := domain.BillingCycle{Name: "c-ok", Months: 1, Multiplier: 1.5, MinQty: 1, MaxQty: 12, Active: true}
	if err := svc.CreateBillingCycle(ctx, &cycle); err != nil {
		t.Fatalf("create cycle: %v", err)
	}
	cycle.Multiplier = -1
	if err := svc.UpdateBillingCycle(ctx, cycle); err != appshared.ErrInvalidInput {
		t.Fatalf("expected invalid input for negative multiplier on update, got %v", err)
	}
}

func TestCatalogService_ValidateSystemImageType(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	svc := appcatalog.NewService(repo, repo, repo)
	img := domain.SystemImage{ImageID: 1, Name: "Bad", Type: "unknown", Enabled: true}
	if err := svc.CreateSystemImage(context.Background(), &img); err == nil {
		t.Fatalf("expected invalid image type")
	}
}

// failingSystemImagesRepo implements SystemImageRepository via a nil embedded
// interface and overrides only the batch line query so the service must
// propagate the injected failure instead of returning a partial set.
type failingSystemImagesRepo struct {
	appports.SystemImageRepository
}

func (f *failingSystemImagesRepo) ListSystemImagesByLineIDs(ctx context.Context, lineIDs []int64) ([]domain.SystemImage, error) {
	return nil, errors.New("injected line query failure")
}

func TestCatalogService_ListSystemImagesForLines_MultiLineAndDedup(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	svc := appcatalog.NewService(repo, repo, repo)
	ctx := context.Background()

	img1 := domain.SystemImage{ImageID: 1, Name: "Ubuntu", Type: "linux", Enabled: true}
	img2 := domain.SystemImage{ImageID: 2, Name: "Debian", Type: "linux", Enabled: true}
	img3 := domain.SystemImage{ImageID: 3, Name: "Windows", Type: "windows", Enabled: true}
	if err := svc.CreateSystemImage(ctx, &img1); err != nil {
		t.Fatalf("create img1: %v", err)
	}
	if err := svc.CreateSystemImage(ctx, &img2); err != nil {
		t.Fatalf("create img2: %v", err)
	}
	if err := svc.CreateSystemImage(ctx, &img3); err != nil {
		t.Fatalf("create img3: %v", err)
	}
	// img1 is on line 1 only; img2 is on both line 1 and line 2 (duplicate
	// association); img3 is on line 3 (not in the requested set).
	if err := svc.SetLineSystemImages(ctx, 1, []int64{img1.ID, img2.ID}); err != nil {
		t.Fatalf("set line1 images: %v", err)
	}
	if err := svc.SetLineSystemImages(ctx, 2, []int64{img2.ID}); err != nil {
		t.Fatalf("set line2 images: %v", err)
	}
	if err := svc.SetLineSystemImages(ctx, 3, []int64{img3.ID}); err != nil {
		t.Fatalf("set line3 images: %v", err)
	}

	// Disable img2 after linking so the batch query's enabled filter is
	// exercised as well.
	img2.Enabled = false
	if err := svc.UpdateSystemImage(ctx, img2); err != nil {
		t.Fatalf("disable img2: %v", err)
	}

	got, err := svc.ListSystemImagesForLines(ctx, []int64{1, 2})
	if err != nil {
		t.Fatalf("ListSystemImagesForLines: %v", err)
	}
	ids := map[int64]bool{}
	for _, img := range got {
		ids[img.ID] = true
	}
	// img1 (line 1) present; img2 (both lines) deduped and filtered out because
	// it was disabled; img3 not in requested lines.
	if len(got) != 1 || !ids[img1.ID] {
		t.Fatalf("expected only enabled img1 from lines 1/2, got %+v", got)
	}

	// Empty line list returns an empty slice without error.
	empty, err := svc.ListSystemImagesForLines(ctx, nil)
	if err != nil || len(empty) != 0 {
		t.Fatalf("expected empty result for no lines, got %v %v", empty, err)
	}
}

func TestCatalogService_ListSystemImagesForLines_PropagatesRepoError(t *testing.T) {
	_, realRepo := testutil.NewTestDB(t, false)
	svc := appcatalog.NewService(realRepo, &failingSystemImagesRepo{}, realRepo)
	ctx := context.Background()
	if _, err := svc.ListSystemImagesForLines(ctx, []int64{1, 2}); err == nil {
		t.Fatalf("expected the batch query failure to propagate, got nil")
	}
}
