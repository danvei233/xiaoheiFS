package testutil

import (
	"context"
	"testing"

	"xiaoheiplay/internal/domain"
)

type CatalogSeed struct {
	Region      domain.Region
	PlanGroup   domain.PlanGroup
	Package     domain.Package
	SystemImage domain.SystemImage
}

func SeedCatalog(t *testing.T, repo interface {
	CreateRegion(ctx context.Context, region *domain.Region) error
	CreatePlanGroup(ctx context.Context, plan *domain.PlanGroup) error
	CreatePackage(ctx context.Context, pkg *domain.Package) error
	CreateSystemImage(ctx context.Context, img *domain.SystemImage) error
	SetLineSystemImages(ctx context.Context, lineID int64, systemImageIDs []int64) error
}) CatalogSeed {
	t.Helper()
	ctx := context.Background()
	region := domain.Region{Code: "area-1", Name: "Region", Active: true}
	if err := repo.CreateRegion(ctx, &region); err != nil {
		t.Fatalf("create region: %v", err)
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
		SortOrder:         0,
	}
	if err := repo.CreatePlanGroup(ctx, &plan); err != nil {
		t.Fatalf("create plan group: %v", err)
	}
	pkg := domain.Package{
		PlanGroupID:       plan.ID,
		Name:              "Basic",
		Cores:             2,
		MemoryGB:          4,
		DiskGB:            40,
		BandwidthMB:       10,
		CPUModel:          "x",
		Monthly:           10,
		PortNum:           30,
		SortOrder:         0,
		Active:            true,
		Visible:           true,
		CapacityRemaining: -1,
	}
	if err := repo.CreatePackage(ctx, &pkg); err != nil {
		t.Fatalf("create package: %v", err)
	}
	img := domain.SystemImage{ImageID: 1, Name: "Ubuntu", Type: "linux", Enabled: true}
	if err := repo.CreateSystemImage(ctx, &img); err != nil {
		t.Fatalf("create image: %v", err)
	}
	if err := repo.SetLineSystemImages(ctx, plan.LineID, []int64{img.ID}); err != nil {
		t.Fatalf("set line images: %v", err)
	}
	return CatalogSeed{
		Region:      region,
		PlanGroup:   plan,
		Package:     pkg,
		SystemImage: img,
	}
}
