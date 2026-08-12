package repo

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestRateLimitBudgetConcurrent verifies the DB atomic rate-limit budget is
// enforced when many goroutines (simulating multiple service instances) race
// against the same bucket: exactly max calls are allowed and the rest rejected.
func TestRateLimitBudgetConcurrent(t *testing.T) {
	g := newTaskLeaseTestRepo(t)
	ctx := context.Background()
	const max = 5
	const workers = 8
	const perWorker = 5
	var wg sync.WaitGroup
	results := make(chan bool, workers*perWorker)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				ok, err := g.AllowRateLimit(ctx, "sms_test:global", max, time.Minute)
				if err != nil {
					t.Errorf("allow: %v", err)
					return
				}
				results <- ok
			}
		}()
	}
	wg.Wait()
	close(results)
	allowed := 0
	for ok := range results {
		if ok {
			allowed++
		}
	}
	if allowed != max {
		t.Fatalf("expected exactly %d allowed, got %d", max, allowed)
	}
}

// TestRateLimitBudgetPerBucketIsolation verifies different buckets have
// independent budgets.
func TestRateLimitBudgetPerBucketIsolation(t *testing.T) {
	g := newTaskLeaseTestRepo(t)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		if ok, err := g.AllowRateLimit(ctx, "a", 5, time.Minute); err != nil || !ok {
			t.Fatalf("bucket a attempt %d: ok=%v err=%v", i, ok, err)
		}
	}
	if ok, err := g.AllowRateLimit(ctx, "b", 5, time.Minute); err != nil || !ok {
		t.Fatalf("bucket b should be independent: ok=%v err=%v", ok, err)
	}
}

// TestRateLimitCleanupBoundsTable verifies expired windows are removed.
func TestRateLimitCleanupBoundsTable(t *testing.T) {
	g := newTaskLeaseTestRepo(t)
	ctx := context.Background()
	if err := g.gdb.WithContext(ctx).Create(&rateLimitBucketRow{
		BucketKey:   "old",
		WindowStart: time.Now().Add(-2 * time.Hour),
		Count:       1,
		UpdatedAt:   time.Now(),
	}).Error; err != nil {
		t.Fatalf("seed expired row: %v", err)
	}
	if ok, err := g.AllowRateLimit(ctx, "fresh", 5, time.Minute); err != nil || !ok {
		t.Fatalf("fresh bucket: ok=%v err=%v", ok, err)
	}
	removed, err := g.CleanupRateLimits(ctx, time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if removed != 1 {
		t.Fatalf("expected 1 expired row removed, got %d", removed)
	}
}
