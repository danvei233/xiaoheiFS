package usecase_test

import (
	"context"
	"testing"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/usecase"
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

	svc := usecase.NewCartService(repo, repo, repo)
	item, err := svc.Add(context.Background(), user.ID, seed.Package.ID, seed.SystemImage.ID, usecase.CartSpec{BillingCycleID: cycle.ID}, 2)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if item.Qty != 2 {
		t.Fatalf("expected qty 2")
	}
	if item.Amount <= 0 {
		t.Fatalf("expected amount")
	}

	updated, err := svc.Update(context.Background(), user.ID, item.ID, usecase.CartSpec{AddCores: 1, BillingCycleID: cycle.ID}, 1)
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
	svc := usecase.NewCartService(repo, repo, repo)

	if _, err := svc.Add(context.Background(), user.ID, seed.Package.ID, seed.SystemImage.ID, usecase.CartSpec{AddCores: -1}, 1); err != usecase.ErrInvalidInput {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestCartService_AddonDisabledByMinusOne(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "u3", "u3@example.com", "pass")
	svc := usecase.NewCartService(repo, repo, repo)

	plan, err := repo.GetPlanGroup(context.Background(), seed.PlanGroup.ID)
	if err != nil {
		t.Fatalf("get plan: %v", err)
	}
	plan.AddCoreMin = -1
	plan.AddCoreMax = 0
	if err := repo.UpdatePlanGroup(context.Background(), plan); err != nil {
		t.Fatalf("update plan: %v", err)
	}

	if _, err := svc.Add(context.Background(), user.ID, seed.Package.ID, seed.SystemImage.ID, usecase.CartSpec{AddCores: 1}, 1); err != usecase.ErrInvalidInput {
		t.Fatalf("expected invalid input when addon disabled, got %v", err)
	}
}
