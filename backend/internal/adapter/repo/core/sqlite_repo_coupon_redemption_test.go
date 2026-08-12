package repo_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func newTestCoupon(t *testing.T, repo interface {
	CreateCouponProductGroup(ctx context.Context, group *domain.CouponProductGroup) error
	CreateCoupon(ctx context.Context, coupon *domain.Coupon) error
}, code string, totalLimit, perUserLimit int) domain.Coupon {
	t.Helper()
	group := domain.CouponProductGroup{Name: "g", Scope: domain.CouponGroupScopeAll, RulesJSON: `[{"scope":"all"}]`}
	if err := repo.CreateCouponProductGroup(context.Background(), &group); err != nil {
		t.Fatalf("create product group: %v", err)
	}
	coupon := domain.Coupon{Code: code, DiscountPermille: 900, ProductGroupID: group.ID, TotalLimit: totalLimit, PerUserLimit: perUserLimit, Active: true}
	if err := repo.CreateCoupon(context.Background(), &coupon); err != nil {
		t.Fatalf("create coupon: %v", err)
	}
	return coupon
}

// TestSQLiteRepo_CreateCouponRedemptionEnforcesTotalLimitAtInsert verifies that the
// redemption insert itself re-checks the coupon TotalLimit inside the repository.
// Previously the limit was only checked by the coupon service (count-then-insert in
// two separate calls), leaving a TOCTOU window where a second concurrent order could
// exceed the limit. The insert must be the authoritative gate.
func TestSQLiteRepo_CreateCouponRedemptionEnforcesTotalLimitAtInsert(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	coupon := newTestCoupon(t, repo, "LIMIT1", 1, -1)

	first := domain.CouponRedemption{CouponID: coupon.ID, OrderID: 1001, UserID: 1, Status: domain.CouponRedemptionStatusApplied, DiscountAmount: 100}
	if err := repo.CreateCouponRedemption(ctx, &first); err != nil {
		t.Fatalf("first redemption: %v", err)
	}

	second := domain.CouponRedemption{CouponID: coupon.ID, OrderID: 1002, UserID: 2, Status: domain.CouponRedemptionStatusApplied, DiscountAmount: 100}
	if err := repo.CreateCouponRedemption(ctx, &second); !errors.Is(err, appshared.ErrConflict) {
		t.Fatalf("expected conflict on redemption exceeding TotalLimit=1, got %v", err)
	}

	total, err := repo.CountCouponRedemptions(ctx, coupon.ID, nil, []string{domain.CouponRedemptionStatusApplied, domain.CouponRedemptionStatusConfirmed})
	if err != nil {
		t.Fatalf("count redemptions: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected exactly one redemption to persist, got %d", total)
	}
}

// TestSQLiteRepo_CreateCouponRedemptionEnforcesPerUserLimitAtInsert verifies the same
// atomic gate for the per-user limit: a second redemption for the same user is rejected,
// while a different user is still allowed.
func TestSQLiteRepo_CreateCouponRedemptionEnforcesPerUserLimitAtInsert(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	coupon := newTestCoupon(t, repo, "PER1", -1, 1)

	u1o1 := domain.CouponRedemption{CouponID: coupon.ID, OrderID: 2001, UserID: 11, Status: domain.CouponRedemptionStatusApplied, DiscountAmount: 100}
	if err := repo.CreateCouponRedemption(ctx, &u1o1); err != nil {
		t.Fatalf("first redemption: %v", err)
	}
	u1o2 := domain.CouponRedemption{CouponID: coupon.ID, OrderID: 2002, UserID: 11, Status: domain.CouponRedemptionStatusApplied, DiscountAmount: 100}
	if err := repo.CreateCouponRedemption(ctx, &u1o2); !errors.Is(err, appshared.ErrConflict) {
		t.Fatalf("expected conflict for same user exceeding PerUserLimit=1, got %v", err)
	}
	u2o3 := domain.CouponRedemption{CouponID: coupon.ID, OrderID: 2003, UserID: 12, Status: domain.CouponRedemptionStatusApplied, DiscountAmount: 100}
	if err := repo.CreateCouponRedemption(ctx, &u2o3); err != nil {
		t.Fatalf("different user should be allowed: %v", err)
	}
}

// TestSQLiteRepo_CanceledRedemptionFreesQuota verifies that a redemption whose status
// moved away from applied/confirmed no longer counts against the limit, so a later
// redemption for the same user is accepted again.
func TestSQLiteRepo_CanceledRedemptionFreesQuota(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	coupon := newTestCoupon(t, repo, "CANCEL", -1, 1)

	first := domain.CouponRedemption{CouponID: coupon.ID, OrderID: 3001, UserID: 21, Status: domain.CouponRedemptionStatusApplied, DiscountAmount: 100}
	if err := repo.CreateCouponRedemption(ctx, &first); err != nil {
		t.Fatalf("first redemption: %v", err)
	}
	if err := repo.UpdateCouponRedemptionStatusByOrder(ctx, 3001, []string{domain.CouponRedemptionStatusApplied}, domain.CouponRedemptionStatusCanceled); err != nil {
		t.Fatalf("cancel redemption: %v", err)
	}
	second := domain.CouponRedemption{CouponID: coupon.ID, OrderID: 3002, UserID: 21, Status: domain.CouponRedemptionStatusApplied, DiscountAmount: 100}
	if err := repo.CreateCouponRedemption(ctx, &second); err != nil {
		t.Fatalf("redemption after cancel should be allowed: %v", err)
	}
}

// TestSQLiteRepo_CreateCouponRedemptionConcurrentRespectsTotalLimit drives concurrent
// redemption inserts against a TotalLimit=1 coupon and asserts the durable invariant:
// no more than one redemption row may persist.
func TestSQLiteRepo_CreateCouponRedemptionConcurrentRespectsTotalLimit(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	coupon := newTestCoupon(t, repo, "CONC1", 1, 1)

	const n = 8
	results := make(chan error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r := domain.CouponRedemption{
				CouponID:       coupon.ID,
				OrderID:        int64(4000 + i),
				UserID:         31,
				Status:         domain.CouponRedemptionStatusApplied,
				DiscountAmount: 100,
			}
			results <- repo.CreateCouponRedemption(ctx, &r)
		}(i)
	}
	wg.Wait()
	close(results)

	successes := 0
	for err := range results {
		if err == nil {
			successes++
		}
	}
	total, err := repo.CountCouponRedemptions(ctx, coupon.ID, nil, []string{domain.CouponRedemptionStatusApplied, domain.CouponRedemptionStatusConfirmed})
	if err != nil {
		t.Fatalf("count redemptions: %v", err)
	}
	if successes != int(total) {
		t.Fatalf("successful inserts (%d) do not match persisted rows (%d)", successes, total)
	}
	if total > 1 {
		t.Fatalf("coupon TotalLimit=1 over-inserted under concurrency: %d rows", total)
	}
	if total != 1 {
		t.Fatalf("expected exactly one redemption to win, got %d", total)
	}
}
