package usertier

import (
	"context"
	"testing"

	"xiaoheiplay/internal/domain"
	repocore "xiaoheiplay/internal/adapter/repo/core"
	"xiaoheiplay/internal/testutil"
)

func TestDeleteGroup_NoDanglingCacheRebuild(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	svc := NewService(repo, repo, repo, repo, nil)

	def, err := svc.EnsureDefaultGroup(ctx)
	if err != nil {
		t.Fatalf("ensure default group: %v", err)
	}

	// A non-default group with a discount rule.
	group := domain.UserTierGroup{Name: "vip", AutoApproveEnabled: true, Priority: 10}
	if err := svc.CreateGroup(ctx, 1, &group); err != nil {
		t.Fatalf("create group: %v", err)
	}
	rule := domain.UserTierDiscountRule{
		GroupID:          group.ID,
		Scope:            domain.UserTierScopeAll,
		DiscountPermille: 500,
	}
	if err := svc.CreateDiscountRule(ctx, 1, &rule); err != nil {
		t.Fatalf("create discount rule: %v", err)
	}

	// A plan group + package so price caches can be computed.
	plan := domain.PlanGroup{Name: "p1", RegionID: 1, UnitCore: 1}
	if err := repo.CreatePlanGroup(ctx, &plan); err != nil {
		t.Fatalf("create plan group: %v", err)
	}
	pkg := domain.Package{Name: "pk", GoodsTypeID: 1, PlanGroupID: plan.ID, Monthly: 10000}
	if err := repo.CreatePackage(ctx, &pkg); err != nil {
		t.Fatalf("create package: %v", err)
	}

	// Assign a user to the vip group.
	user := testutil.CreateUser(t, repo, "u1", "u1@example.com", "pass")
	if err := svc.SetUserGroup(ctx, 1, user.ID, group.ID, nil); err != nil {
		t.Fatalf("set user group: %v", err)
	}

	// Delete the group.
	if err := svc.DeleteGroup(ctx, 1, group.ID); err != nil {
		t.Fatalf("delete group: %v", err)
	}

	// A rebuild triggered for the deleted group (e.g. from a stale async call or
	// an admin rebuild of a dead id) must not resurrect its price caches.
	svc.rebuildGroupPriceCache(group.ID)
	if _, err := repo.GetUserTierPriceCache(ctx, group.ID, pkg.ID); err == nil {
		t.Fatalf("price cache for deleted group %d was re-created", group.ID)
	}

	// A user still referencing the deleted group resolves pricing under the
	// default group, not the dangling one.
	cache, gotGroupID, err := svc.ResolvePackagePricing(ctx, user.ID, pkg.ID)
	if err != nil {
		t.Fatalf("resolve pricing: %v", err)
	}
	if gotGroupID != def.ID {
		t.Fatalf("expected default group id %d, got %d", def.ID, gotGroupID)
	}
	if cache.GroupID != def.ID {
		t.Fatalf("expected cache group id %d, got %d", def.ID, cache.GroupID)
	}
}

// TestResolvePackagePricing_CacheMissAppliesDiscount locks the P1-9 fix: a
// price-cache miss must resolve the tier discount synchronously instead of
// returning the full catalog price while an async rebuild races to populate
// the cache. Before the fix, the first (and any miss) pricing call overcharged
// the user until a rebuild landed, and a permanently-failing rebuild left the
// user at full price forever.
func TestResolvePackagePricing_CacheMissAppliesDiscount(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	svc := NewService(repo, repo, repo, repo, nil)

	def, err := svc.EnsureDefaultGroup(ctx)
	if err != nil {
		t.Fatalf("ensure default group: %v", err)
	}
	_ = def

	vip := domain.UserTierGroup{Name: "vip-miss", AutoApproveEnabled: true, Priority: 10}
	if err := svc.CreateGroup(ctx, 1, &vip); err != nil {
		t.Fatalf("create group: %v", err)
	}
	rule := domain.UserTierDiscountRule{GroupID: vip.ID, Scope: domain.UserTierScopeAll, DiscountPermille: 500}
	if err := svc.CreateDiscountRule(ctx, 1, &rule); err != nil {
		t.Fatalf("create discount rule: %v", err)
	}

	plan := domain.PlanGroup{Name: "p1", RegionID: 1, UnitCore: 1}
	if err := repo.CreatePlanGroup(ctx, &plan); err != nil {
		t.Fatalf("create plan group: %v", err)
	}
	pkg := domain.Package{Name: "pk", GoodsTypeID: 1, PlanGroupID: plan.ID, Monthly: 10000}
	if err := repo.CreatePackage(ctx, &pkg); err != nil {
		t.Fatalf("create package: %v", err)
	}

	user := testutil.CreateUser(t, repo, "u_miss", "u_miss@example.com", "pass")
	if err := svc.SetUserGroup(ctx, 1, user.ID, vip.ID, nil); err != nil {
		t.Fatalf("set user group: %v", err)
	}

	// Ensure the price cache is empty for this group/package so the miss path
	// runs (CreateDiscountRule/SetUserGroup may have already rebuilt it).
	if err := repo.DeleteUserTierPriceCachesByGroup(ctx, vip.ID); err != nil {
		t.Fatalf("clear price cache: %v", err)
	}

	cache, gotGroupID, err := svc.ResolvePackagePricing(ctx, user.ID, pkg.ID)
	if err != nil {
		t.Fatalf("resolve pricing: %v", err)
	}
	if gotGroupID != vip.ID {
		t.Fatalf("expected vip group id %d, got %d", vip.ID, gotGroupID)
	}
	if cache.MonthlyPrice != 9500 {
		t.Fatalf("expected discounted monthly price 9500 on cache miss, got %d", cache.MonthlyPrice)
	}

	// The resolved price must have been persisted so the next call hits the cache.
	got, err := repo.GetUserTierPriceCache(ctx, vip.ID, pkg.ID)
	if err != nil {
		t.Fatalf("expected cache row after miss-path resolve: %v", err)
	}
	if got.MonthlyPrice != 9500 {
		t.Fatalf("expected cached price 9500, got %d", got.MonthlyPrice)
	}
}


// panicTierRepo wraps a real GormRepo and panics on GetUserTierGroup so the
// async price-cache rebuild hits an injected panic.
type panicTierRepo struct {
	*repocore.GormRepo
}

func (p *panicTierRepo) GetUserTierGroup(ctx context.Context, id int64) (domain.UserTierGroup, error) {
	panic("injected panic")
}

// TestRebuildGroupPriceCacheAsync_PanicDoesNotCrash locks P1-13: an async
// price-cache rebuild must not take down the process when the repo panics. The
// goroutine guard recovers and logs instead of crashing.
func TestRebuildGroupPriceCacheAsync_PanicDoesNotCrash(t *testing.T) {
	_, db := testutil.NewTestDB(t, false)
	pr := &panicTierRepo{GormRepo: db}
	svc := NewService(pr, pr, pr, pr, nil)
	done := make(chan struct{})
	go func() {
		defer close(done)
		svc.RebuildGroupPriceCacheAsync(1)
	}()
	<-done
	// If we reach here the goroutine guard contained the panic and the process
	// did not crash.
}
