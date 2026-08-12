package cart_test

import (
	"context"
	"testing"
	appcart "xiaoheiplay/internal/app/cart"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func TestCartService_AddUpdate(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "u1", "u1@example.com", "pass")

	cycle := domain.BillingCycle{
		Name:       "monthly",
		Months:     1,
		Multiplier: 1.0,
		MinQty:     1,
		MaxQty:     12,
		Active:     true,
		SortOrder:  1,
	}
	if err := repo.CreateBillingCycle(context.Background(), &cycle); err != nil {
		t.Fatalf("create cycle: %v", err)
	}

	svc := appcart.NewService(repo, repo, repo)
	item, err := svc.Add(context.Background(), user.ID, seed.Package.ID, seed.SystemImage.ID, appshared.CartSpec{BillingCycleID: cycle.ID}, 2)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if item.Qty != 2 {
		t.Fatalf("expected qty 2")
	}
	if item.Amount <= 0 {
		t.Fatalf("expected amount")
	}

	updated, err := svc.Update(context.Background(), user.ID, item.ID, appshared.CartSpec{AddCores: 1, BillingCycleID: cycle.ID}, 1)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Amount <= 0 {
		t.Fatalf("expected updated amount")
	}
}

func TestCartService_InvalidSpec(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "u2", "u2@example.com", "pass")
	svc := appcart.NewService(repo, repo, repo)

	if _, err := svc.Add(context.Background(), user.ID, seed.Package.ID, seed.SystemImage.ID, appshared.CartSpec{AddCores: -1}, 1); err != appshared.ErrInvalidInput {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestCartService_AddRejectsInactivePackage(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "u4", "u4@example.com", "pass")
	svc := appcart.NewService(repo, repo, repo)

	pkg := seed.Package
	pkg.Active = false
	if err := repo.UpdatePackage(context.Background(), pkg); err != nil {
		t.Fatalf("deactivate package: %v", err)
	}

	if _, err := svc.Add(context.Background(), user.ID, pkg.ID, seed.SystemImage.ID, appshared.CartSpec{}, 1); err != appshared.ErrInvalidInput {
		t.Fatalf("expected invalid input for inactive package, got %v", err)
	}
}

func TestCartService_AddRejectsHiddenPackage(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "u5", "u5@example.com", "pass")
	svc := appcart.NewService(repo, repo, repo)

	pkg := seed.Package
	pkg.Visible = false
	if err := repo.UpdatePackage(context.Background(), pkg); err != nil {
		t.Fatalf("hide package: %v", err)
	}

	if _, err := svc.Add(context.Background(), user.ID, pkg.ID, seed.SystemImage.ID, appshared.CartSpec{}, 1); err != appshared.ErrInvalidInput {
		t.Fatalf("expected invalid input for hidden package, got %v", err)
	}
}

func TestCartService_AddRejectsInactivePlanGroup(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "u6", "u6@example.com", "pass")
	svc := appcart.NewService(repo, repo, repo)

	plan := seed.PlanGroup
	plan.Active = false
	if err := repo.UpdatePlanGroup(context.Background(), plan); err != nil {
		t.Fatalf("deactivate plan: %v", err)
	}

	if _, err := svc.Add(context.Background(), user.ID, seed.Package.ID, seed.SystemImage.ID, appshared.CartSpec{}, 1); err != appshared.ErrInvalidInput {
		t.Fatalf("expected invalid input for inactive plan, got %v", err)
	}
}

func TestCartService_UpdateRejectsInactivePackage(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "u7", "u7@example.com", "pass")
	svc := appcart.NewService(repo, repo, repo)

	item, err := svc.Add(context.Background(), user.ID, seed.Package.ID, seed.SystemImage.ID, appshared.CartSpec{}, 1)
	if err != nil {
		t.Fatalf("add: %v", err)
	}

	pkg := seed.Package
	pkg.Active = false
	if err := repo.UpdatePackage(context.Background(), pkg); err != nil {
		t.Fatalf("deactivate package: %v", err)
	}

	if _, err := svc.Update(context.Background(), user.ID, item.ID, appshared.CartSpec{}, 1); err != appshared.ErrInvalidInput {
		t.Fatalf("expected invalid input for inactive package on update, got %v", err)
	}
}

func TestCartService_AmountMatchesOrderPerComponentRounding(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "u8", "u8@example.com", "pass")

	// Non-integer billing-cycle multiplier: the order path rounds each price
	// component separately (round(base*mult) + round(addon*mult)), while the
	// cart used to round the aggregate once. Those disagree on fractional
	// unit prices; the cart snapshot must match the order's authoritative total.
	cycle := domain.BillingCycle{
		Name:       "quarter-discount",
		Months:     1,
		Multiplier: 0.73,
		MinQty:     1,
		MaxQty:     12,
		Active:     true,
		SortOrder:  1,
	}
	if err := repo.CreateBillingCycle(context.Background(), &cycle); err != nil {
		t.Fatalf("create cycle: %v", err)
	}

	plan := seed.PlanGroup
	plan.UnitCore = 5
	plan.UnitMem = 0
	plan.UnitDisk = 0
	plan.UnitBW = 0
	if err := repo.UpdatePlanGroup(context.Background(), plan); err != nil {
		t.Fatalf("update plan: %v", err)
	}

	pkg := seed.Package
	pkg.Monthly = 1234
	if err := repo.UpdatePackage(context.Background(), pkg); err != nil {
		t.Fatalf("update package: %v", err)
	}

	svc := appcart.NewService(repo, repo, repo)
	item, err := svc.Add(context.Background(), user.ID, pkg.ID, seed.SystemImage.ID,
		appshared.CartSpec{AddCores: 7, BillingCycleID: cycle.ID}, 1)
	if err != nil {
		t.Fatalf("add: %v", err)
	}

	// round(1234*0.73)=901 + round(7*5*0.73)=26 => 927.
	// A single-round aggregate would yield round((1234+35)*0.73)=926.
	if item.Amount != 927 {
		t.Fatalf("expected amount 927, got %d", item.Amount)
	}
}

func TestCartService_RejectsExcessiveQty(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "u9", "u9@example.com", "pass")
	svc := appcart.NewService(repo, repo, repo)

	// Order creation expands qty into one order item per unit, so an unbounded
	// qty can balloon memory and the orders table. The service must cap it.
	if _, err := svc.Add(context.Background(), user.ID, seed.Package.ID, seed.SystemImage.ID, appshared.CartSpec{}, 1000); err != appshared.ErrInvalidInput {
		t.Fatalf("expected invalid input for excessive qty, got %v", err)
	}

	// The upper bound itself must remain usable.
	item, err := svc.Add(context.Background(), user.ID, seed.Package.ID, seed.SystemImage.ID, appshared.CartSpec{}, 999)
	if err != nil {
		t.Fatalf("add at bound: %v", err)
	}
	if item.Qty != 999 {
		t.Fatalf("expected qty 999, got %d", item.Qty)
	}
}

func TestCartService_UpdateRejectsExcessiveQty(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "u10", "u10@example.com", "pass")
	svc := appcart.NewService(repo, repo, repo)

	item, err := svc.Add(context.Background(), user.ID, seed.Package.ID, seed.SystemImage.ID, appshared.CartSpec{}, 1)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if _, err := svc.Update(context.Background(), user.ID, item.ID, appshared.CartSpec{}, 5000); err != appshared.ErrInvalidInput {
		t.Fatalf("expected invalid input for excessive qty on update, got %v", err)
	}
}

func TestCartService_AddonDisabledByMinusOne(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "u3", "u3@example.com", "pass")
	svc := appcart.NewService(repo, repo, repo)

	plan, err := repo.GetPlanGroup(context.Background(), seed.PlanGroup.ID)
	if err != nil {
		t.Fatalf("get plan: %v", err)
	}
	plan.AddCoreMin = -1
	plan.AddCoreMax = 0
	if err := repo.UpdatePlanGroup(context.Background(), plan); err != nil {
		t.Fatalf("update plan: %v", err)
	}

	if _, err := svc.Add(context.Background(), user.ID, seed.Package.ID, seed.SystemImage.ID, appshared.CartSpec{AddCores: 1}, 1); err != appshared.ErrInvalidInput {
		t.Fatalf("expected invalid input when addon disabled, got %v", err)
	}
}
