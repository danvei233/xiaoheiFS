package repo_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

// TestMultiDialect_HasExclusiveVPSOrderInProgressBeyondLimit locks the regression
// where the exclusivity check only scanned the most recent 50 items and parsed
// spec_json.vps_id. With the normalized, indexed vps_id column an older pending
// operation (buried past 50 rows) must still be found.
func TestMultiDialect_HasExclusiveVPSOrderInProgressBeyondLimit(t *testing.T) {
	for _, dialect := range availableDialects(t) {
		t.Run(dialect, func(t *testing.T) {
			repo := openDialectRepo(t, dialect)
			ctx := context.Background()
			run := fmt.Sprintf("%s-%d", dialect, time.Now().UnixNano())
			user := testutil.CreateUser(t, repo, "excl-hist-"+run, "excl-hist-"+run+"@example.com", "pass")

			// Create the target VPS/order/item first (it will be the OLDEST).
			targetOrder := domain.Order{UserID: user.ID, OrderNo: "ORD-EXCL-" + run, Status: domain.OrderStatusPendingReview, TotalAmount: 100, Currency: "CNY"}
			if err := repo.CreateOrder(ctx, &targetOrder); err != nil {
				t.Fatalf("create target order: %v", err)
			}
			if err := repo.CreateOrderItems(ctx, []domain.OrderItem{{
				OrderID: targetOrder.ID, Qty: 1, Amount: 100, Status: domain.OrderItemStatusPendingReview,
				Action: "renew", VPSID: 424242, SpecJSON: `{"vps_id": 424242}`,
			}}); err != nil {
				t.Fatalf("create target item: %v", err)
			}

			// Then bury it under 80 older-priority (higher id) unrelated items.
			for i := 0; i < 80; i++ {
				o := domain.Order{UserID: user.ID, OrderNo: fmt.Sprintf("ORD-EXCL-H-%s-%d", run, i), Status: domain.OrderStatusPendingReview, TotalAmount: 1, Currency: "CNY"}
				if err := repo.CreateOrder(ctx, &o); err != nil {
					t.Fatalf("create history order: %v", err)
				}
				if err := repo.CreateOrderItems(ctx, []domain.OrderItem{{
					OrderID: o.ID, Qty: 1, Amount: 1, Status: domain.OrderItemStatusPendingReview,
					Action: "renew", VPSID: int64(100000 + i), SpecJSON: fmt.Sprintf(`{"vps_id": %d}`, 100000+i),
				}}); err != nil {
					t.Fatalf("create history item: %v", err)
				}
			}

			pending, err := repo.HasPendingRenewOrder(ctx, user.ID, 424242)
			if err != nil {
				t.Fatalf("has pending renew: %v", err)
			}
			if !pending {
				t.Fatalf("target exclusive operation buried past 50 rows was not found")
			}
			// A VPS with no pending operation must be free.
			free, err := repo.HasPendingRenewOrder(ctx, user.ID, 999999)
			if err != nil || free {
				t.Fatalf("unrelated vps reported pending: free=%v err=%v", free, err)
			}
		})
	}
}

// TestMultiDialect_VPSLifecycleLockConcurrentExactlyOneWinner runs N concurrent
// reservations for the same VPS. The unique index admits exactly one winner; the
// rest fail atomically instead of both passing a check-then-insert.
func TestMultiDialect_VPSLifecycleLockConcurrentExactlyOneWinner(t *testing.T) {
	for _, dialect := range availableDialects(t) {
		t.Run(dialect, func(t *testing.T) {
			repo := openDialectRepo(t, dialect)
			ctx := context.Background()

			const n = 8
			// A shared external database persists rows across runs, so the lock
			// target must be unique per invocation.
			vpsID := int64(555000) + time.Now().UnixNano()%10000
			start := make(chan struct{})
			var wg sync.WaitGroup
			results := make(chan int64, n)
			errs := make(chan error, n)
			for i := 0; i < n; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					<-start
					err := repo.AcquireVPSLifecycleLock(ctx, vpsID, int64(i+1), "renew")
					if err != nil {
						errs <- err
						return
					}
					results <- int64(i + 1)
				}(i)
			}
			close(start)
			wg.Wait()
			close(results)
			close(errs)

			winnerOrder := int64(0)
			conflicts := 0
			for orderID := range results {
				if winnerOrder == 0 {
					winnerOrder = orderID
				}
			}
			for err := range errs {
				if err == domain.ErrConflict {
					conflicts++
					continue
				}
				t.Fatalf("acquire lock: %v", err)
			}
			if winnerOrder == 0 || (conflicts+1) != n {
				t.Fatalf("expected exactly one lock winner, winner=%d conflicts=%d", winnerOrder, conflicts)
			}

			// The lock must be releasable by its owner order.
			if err := repo.ReleaseVPSLifecycleLocksForOrder(ctx, winnerOrder); err != nil {
				t.Fatalf("release lock: %v", err)
			}
			// After release, a new lock can be acquired.
			if err := repo.AcquireVPSLifecycleLock(ctx, vpsID, 100, "refund"); err != nil {
				t.Fatalf("re-acquire after release: %v", err)
			}
		})
	}
}
