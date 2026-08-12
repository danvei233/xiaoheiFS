package repo_test

import (
	"context"
	"testing"

	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

// TestSQLiteRepo_ListCouponsDefaultsInvalidLimit verifies the repository defends against
// non-positive pagination inputs instead of silently returning an empty set or an
// unbounded query.
func TestSQLiteRepo_ListCouponsDefaultsInvalidLimit(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()

	group := domain.CouponProductGroup{Name: "g", Scope: domain.CouponGroupScopeAll, RulesJSON: `[{"scope":"all"}]`}
	if err := repo.CreateCouponProductGroup(ctx, &group); err != nil {
		t.Fatalf("create product group: %v", err)
	}
	for _, code := range []string{"AAA", "BBB"} {
		coupon := domain.Coupon{Code: code, DiscountPermille: 900, ProductGroupID: group.ID, TotalLimit: -1, PerUserLimit: -1, Active: true}
		if err := repo.CreateCoupon(ctx, &coupon); err != nil {
			t.Fatalf("create coupon %s: %v", code, err)
		}
	}

	if items, _, err := repo.ListCoupons(ctx, appshared.CouponFilter{}, 0, 0); err != nil {
		t.Fatalf("list coupons limit=0: %v", err)
	} else if len(items) == 0 {
		t.Fatalf("limit=0 must fall back to a sane page, got 0 items")
	}
	if items, _, err := repo.ListCoupons(ctx, appshared.CouponFilter{}, -1, 0); err != nil {
		t.Fatalf("list coupons limit=-1: %v", err)
	} else if len(items) == 0 {
		t.Fatalf("limit=-1 must fall back to a sane page, got 0 items")
	}
}

// TestSQLiteRepo_ListCouponsEscapesLikeWildcards verifies that a keyword containing a
// LIKE wildcard (% or _) matches only the literal substring instead of broadening the
// match set.
func TestSQLiteRepo_ListCouponsEscapesLikeWildcards(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()

	group := domain.CouponProductGroup{Name: "g", Scope: domain.CouponGroupScopeAll, RulesJSON: `[{"scope":"all"}]`}
	if err := repo.CreateCouponProductGroup(ctx, &group); err != nil {
		t.Fatalf("create product group: %v", err)
	}
	codes := []string{"PAY50NOW", "PAY50%OFF"}
	for _, code := range codes {
		coupon := domain.Coupon{Code: code, DiscountPermille: 900, ProductGroupID: group.ID, TotalLimit: -1, PerUserLimit: -1, Active: true}
		if err := repo.CreateCoupon(ctx, &coupon); err != nil {
			t.Fatalf("create coupon %s: %v", code, err)
		}
	}

	items, total, err := repo.ListCoupons(ctx, appshared.CouponFilter{Keyword: "50%"}, 20, 0)
	if err != nil {
		t.Fatalf("search coupons: %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].Code != "PAY50%OFF" {
		t.Fatalf("keyword with %% must match only the literal code, got total=%d items=%+v", total, items)
	}
}
