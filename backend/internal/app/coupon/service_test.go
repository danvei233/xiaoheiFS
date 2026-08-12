package coupon_test

import (
	"context"
	"testing"

	appcoupon "xiaoheiplay/internal/app/coupon"
	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func newTestCouponService(t *testing.T) (*appcoupon.Service, appports.CouponRepository) {
	t.Helper()
	_, repo := testutil.NewTestDB(t, false)
	return appcoupon.NewService(repo, nil), repo
}

func createCouponGroup(t *testing.T, repo appports.CouponRepository) domain.CouponProductGroup {
	t.Helper()
	group := domain.CouponProductGroup{Name: "all", Scope: domain.CouponGroupScopeAll}
	if err := repo.CreateCouponProductGroup(context.Background(), &group); err != nil {
		t.Fatalf("create group: %v", err)
	}
	return group
}

func TestCouponService_BatchGenerate_EmptyTemplateCode(t *testing.T) {
	svc, repo := newTestCouponService(t)
	ctx := context.Background()
	group := createCouponGroup(t, repo)

	// Batch generation uses a template coupon that intentionally has no Code:
	// codes are generated per item. The template must still be validated for all
	// other fields, but the empty-code rule must not make the whole batch fail.
	items, err := svc.BatchGenerate(ctx, 1, appcoupon.BatchGenerateInput{
		Count:  5,
		Length: 8,
		Coupon: domain.Coupon{
			DiscountPermille: 500,
			ProductGroupID:   group.ID,
			TotalLimit:       -1,
			PerUserLimit:     -1,
			Active:           true,
		},
	})
	if err != nil {
		t.Fatalf("batch generate: %v", err)
	}
	if len(items) != 5 {
		t.Fatalf("expected 5 coupons, got %d", len(items))
	}
	seen := map[string]struct{}{}
	for _, it := range items {
		if it.Code == "" {
			t.Fatalf("expected non-empty generated code")
		}
		if _, dup := seen[it.Code]; dup {
			t.Fatalf("duplicate code %q", it.Code)
		}
		seen[it.Code] = struct{}{}
	}
}

func TestCouponService_BatchGenerate_TemplateValidationStillApplies(t *testing.T) {
	svc, repo := newTestCouponService(t)
	ctx := context.Background()
	group := createCouponGroup(t, repo)

	// Invalid template fields must still be rejected (only the empty-code rule
	// is relaxed for batch generation).
	_, err := svc.BatchGenerate(ctx, 1, appcoupon.BatchGenerateInput{
		Count:  3,
		Length: 8,
		Coupon: domain.Coupon{
			DiscountPermille: 1500, // > 1000, invalid
			ProductGroupID:   group.ID,
			TotalLimit:       -1,
			PerUserLimit:     -1,
		},
	})
	if err != appshared.ErrInvalidInput {
		t.Fatalf("expected invalid input for bad template, got %v", err)
	}
}

func TestCouponService_CreateCoupon_RequiresCode(t *testing.T) {
	svc, repo := newTestCouponService(t)
	ctx := context.Background()
	group := createCouponGroup(t, repo)

	coupon := domain.Coupon{
		DiscountPermille: 500,
		ProductGroupID:   group.ID,
		TotalLimit:       -1,
		PerUserLimit:     -1,
		Active:           true,
	}
	if err := svc.CreateCoupon(ctx, 1, &coupon); err != appshared.ErrInvalidInput {
		t.Fatalf("expected invalid input for empty code, got %v", err)
	}
}

func TestCouponService_RejectsZeroLimits(t *testing.T) {
	svc, repo := newTestCouponService(t)
	ctx := context.Background()
	group := createCouponGroup(t, repo)

	// PreviewDiscount treats limit 0 as "already fully used" (used >= 0 always
	// true), so a coupon with TotalLimit/PerUserLimit 0 can never be redeemed.
	// Reject the footgun at create/update time.
	base := domain.Coupon{
		Code:             "ZERO",
		DiscountPermille: 500,
		ProductGroupID:   group.ID,
		TotalLimit:       -1,
		PerUserLimit:     -1,
		Active:           true,
	}
	for _, mutate := range []func(*domain.Coupon){
		func(c *domain.Coupon) { c.TotalLimit = 0 },
		func(c *domain.Coupon) { c.PerUserLimit = 0 },
	} {
		c := base
		mutate(&c)
		if err := svc.CreateCoupon(ctx, 1, &c); err != appshared.ErrInvalidInput {
			t.Fatalf("expected invalid input for zero limit, got %v", err)
		}
	}

	// Unlimited (-1) and positive limits must still be accepted.
	ok := base
	ok.TotalLimit = 10
	ok.PerUserLimit = 1
	if err := svc.CreateCoupon(ctx, 1, &ok); err != nil {
		t.Fatalf("expected valid coupon to be created, got %v", err)
	}
}
