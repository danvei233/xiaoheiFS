package repo

import (
	"context"
	"sync"
	"testing"
	"time"

	"xiaoheiplay/internal/domain"
)

// TestRealNameAttemptBudgetConcurrent verifies the DB-backed attempt budget is
// enforced atomically when many goroutines (simulating multiple service
// instances) race against the same window: exactly max attempts are allowed and
// the rest are rejected, even though all 40 calls happen concurrently.
func TestRealNameAttemptBudgetConcurrent(t *testing.T) {
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
				ok, err := g.AllowRealNameAttempt(ctx, 42, "user", max, 10*time.Minute)
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
		t.Fatalf("expected exactly %d allowed attempts, got %d", max, allowed)
	}
}

// TestRealNameAttemptBudgetPerDimensionIsolation verifies different
// dimensions/users have independent budgets.
func TestRealNameAttemptBudgetPerDimensionIsolation(t *testing.T) {
	g := newTaskLeaseTestRepo(t)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		if ok, err := g.AllowRealNameAttempt(ctx, 1, "user", 5, time.Minute); err != nil || !ok {
			t.Fatalf("attempt %d user: ok=%v err=%v", i, ok, err)
		}
	}
	// A different user still has a full budget.
	if ok, err := g.AllowRealNameAttempt(ctx, 2, "user", 5, time.Minute); err != nil || !ok {
		t.Fatalf("user 2 should be allowed: ok=%v err=%v", ok, err)
	}
	// A different dimension for the same user is independent.
	if ok, err := g.AllowRealNameAttempt(ctx, 1, "user:other_provider", 5, time.Minute); err != nil || !ok {
		t.Fatalf("different dimension should be allowed: ok=%v err=%v", ok, err)
	}
}

// TestRealNameAttemptCleanupBoundsTable verifies expired windows are removed.
func TestRealNameAttemptCleanupBoundsTable(t *testing.T) {
	g := newTaskLeaseTestRepo(t)
	ctx := context.Background()
	// Seed an expired window by inserting directly with an old window_start.
	if err := g.gdb.WithContext(ctx).Create(&realnameAttemptRow{
		UserID:      9,
		Dimension:   "user",
		WindowStart: time.Now().Add(-2 * time.Hour),
		Count:       1,
		UpdatedAt:   time.Now(),
	}).Error; err != nil {
		t.Fatalf("seed expired row: %v", err)
	}
	// A fresh window row.
	if ok, err := g.AllowRealNameAttempt(ctx, 9, "user", 5, 10*time.Minute); err != nil || !ok {
		t.Fatalf("fresh attempt: ok=%v err=%v", ok, err)
	}
	removed, err := g.CleanupRealNameAttempts(ctx, time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if removed != 1 {
		t.Fatalf("expected 1 expired row removed, got %d", removed)
	}
	var count int64
	if err := g.gdb.WithContext(ctx).Model(&realnameAttemptRow{}).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 remaining row (fresh window), got %d", count)
	}
}

// TestListPendingRealNameVerificationsFiltersAtSQL verifies the polling cursor
// only returns status='pending' rows ordered by id ASC past the cursor, so a
// large table with a small pending backlog is not scanned in full every tick.
func TestListPendingRealNameVerificationsFiltersAtSQL(t *testing.T) {
	g := newTaskLeaseTestRepo(t)
	ctx := context.Background()

	// Seed non-pending and pending rows interleaved so ordering matters.
	for i := 0; i < 3; i++ {
		if err := g.CreateRealNameVerification(ctx, &domain.RealNameVerification{
			UserID: int64(i), RealName: "V", IDNumber: "11010519491231002X", Status: "verified", Provider: "fake",
		}); err != nil {
			t.Fatalf("seed verified %d: %v", i, err)
		}
		if err := g.CreateRealNameVerification(ctx, &domain.RealNameVerification{
			UserID: int64(10 + i), RealName: "P", IDNumber: "11010519491231002X", Status: "pending", Provider: "fake",
		}); err != nil {
			t.Fatalf("seed pending %d: %v", i, err)
		}
	}
	// An extra failed row at the end must also be excluded.
	if err := g.CreateRealNameVerification(ctx, &domain.RealNameVerification{
		UserID: 99, RealName: "F", IDNumber: "11010519491231002X", Status: "failed", Provider: "fake",
	}); err != nil {
		t.Fatalf("seed failed: %v", err)
	}

	items, err := g.ListPendingRealNameVerifications(ctx, 0, 50)
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 pending rows, got %d", len(items))
	}
	for i := range items {
		if items[i].Status != "pending" {
			t.Fatalf("item %d status = %s, want pending", i, items[i].Status)
		}
		if i > 0 && items[i].ID <= items[i-1].ID {
			t.Fatalf("items not ordered by id ASC: %d then %d", items[i-1].ID, items[i].ID)
		}
	}

	// Cursor semantics: start after the first pending row returns the rest.
	cursor := items[0].ID
	rest, err := g.ListPendingRealNameVerifications(ctx, cursor, 50)
	if err != nil {
		t.Fatalf("list after cursor: %v", err)
	}
	if len(rest) != 2 || rest[0].ID <= cursor {
		t.Fatalf("cursor paging wrong: len=%d first=%d cursor=%d", len(rest), rest[0].ID, cursor)
	}
}
