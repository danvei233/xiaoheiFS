package repo_test

import (
	"context"
	"testing"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

// TestSQLiteRepo_ListImageLineNamesReturnsEveryDistinctLineName verifies that the
// GROUP BY collapses plan_groups that share a line_id only when they share a name too;
// distinct line names for the same image must all be returned. The previous GROUP BY
// (system_image_id, line_id) selected a bare non-grouped column, which on MySQL violates
// ONLY_FULL_GROUP_BY and on any dialect silently drops line names.
func TestSQLiteRepo_ListImageLineNamesReturnsEveryDistinctLineName(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()

	lineA := domain.PlanGroup{GoodsTypeID: 1, Name: "Line Alpha", LineID: 77, RegionID: 1, UnitCore: 1, UnitMem: 1, UnitDisk: 1, UnitBW: 1, Active: true, Visible: true, CapacityRemaining: -1}
	lineB := domain.PlanGroup{GoodsTypeID: 2, Name: "Line Beta", LineID: 77, RegionID: 1, UnitCore: 1, UnitMem: 1, UnitDisk: 1, UnitBW: 1, Active: true, Visible: true, CapacityRemaining: -1}
	if err := repo.CreatePlanGroup(ctx, &lineA); err != nil {
		t.Fatalf("create plan group A: %v", err)
	}
	if err := repo.CreatePlanGroup(ctx, &lineB); err != nil {
		t.Fatalf("create plan group B: %v", err)
	}
	img := domain.SystemImage{ImageID: 1, Name: "Ubuntu", Type: "linux", Enabled: true}
	if err := repo.CreateSystemImage(ctx, &img); err != nil {
		t.Fatalf("create system image: %v", err)
	}
	if err := repo.SetLineSystemImages(ctx, 77, []int64{img.ID}); err != nil {
		t.Fatalf("set line images: %v", err)
	}

	names, err := repo.ListImageLineNames(ctx, []int64{img.ID})
	if err != nil {
		t.Fatalf("list image line names: %v", err)
	}
	got := names[img.ID]
	if len(got) != 2 {
		t.Fatalf("expected both distinct line names for image, got %v", got)
	}
	seen := map[string]bool{}
	for _, name := range got {
		seen[name] = true
	}
	if !seen["Line Alpha"] || !seen["Line Beta"] {
		t.Fatalf("missing a line name, got %v", got)
	}
}
