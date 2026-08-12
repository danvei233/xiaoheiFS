package repo

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"xiaoheiplay/internal/domain"
)

func newTaskLeaseTestRepo(t *testing.T) *GormRepo {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "lease.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return NewGormRepo(db)
}

// TestTaskLeaseConcurrentAcquireSingleWinner verifies the CAS lease grants the
// task to exactly one contender when several workers race for it.
func TestTaskLeaseConcurrentAcquireSingleWinner(t *testing.T) {
	g := newTaskLeaseTestRepo(t)
	const workers = 8
	var wg sync.WaitGroup
	winners := make(chan bool, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_, ok, err := g.AcquireTaskLease(context.Background(), "vps_refresh", "worker-"+string(rune('A'+n)), 30*time.Second)
			if err != nil {
				t.Errorf("acquire: %v", err)
				return
			}
			winners <- ok
		}(i)
	}
	wg.Wait()
	close(winners)
	got := 0
	for ok := range winners {
		if ok {
			got++
		}
	}
	if got != 1 {
		t.Fatalf("expected exactly one lease winner, got %d", got)
	}
}

// TestTaskLeaseExpiryStealAndFencing verifies a crashed worker's lease can be
// reclaimed after expiry and that the old worker cannot commit its run after
// losing the lease.
func TestTaskLeaseExpiryStealAndFencing(t *testing.T) {
	g := newTaskLeaseTestRepo(t)
	ctx := context.Background()

	leaseA, ok, err := g.AcquireTaskLease(ctx, "expire_reminder", "worker-A", 30*time.Second)
	if err != nil || !ok {
		t.Fatalf("worker A acquire: ok=%v err=%v", ok, err)
	}
	// While the lease is valid, worker B must not steal it.
	if _, ok, err := g.AcquireTaskLease(ctx, "expire_reminder", "worker-B", 30*time.Second); err != nil || ok {
		t.Fatalf("worker B must not steal a valid lease: ok=%v err=%v", ok, err)
	}

	// Backdate A's lease to simulate a crashed worker whose heartbeat stopped.
	if err := g.gdb.Model(&scheduledTaskLeaseRow{}).
		Where("task_key = ? AND owner = ? AND fencing_token = ?", "expire_reminder", "worker-A", leaseA.FencingToken).
		Update("lease_until", time.Now().Add(-time.Minute)).Error; err != nil {
		t.Fatalf("backdate lease: %v", err)
	}

	// Worker B can now reclaim the expired lease.
	leaseB, ok, err := g.AcquireTaskLease(ctx, "expire_reminder", "worker-B", 30*time.Second)
	if err != nil || !ok {
		t.Fatalf("worker B reclaim after expiry: ok=%v err=%v", ok, err)
	}

	// Worker A attempts to complete its run with its old fencing token -> fenced.
	run := domain.ScheduledTaskRun{ID: 1, TaskKey: "expire_reminder", Status: "success", StartedAt: time.Now()}
	committed, err := g.CompleteTaskRun(ctx, run, "expire_reminder", "worker-A", leaseA.FencingToken)
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if committed {
		t.Fatalf("old worker committed after losing the lease")
	}

	// Worker B (current holder) can complete its run.
	runB := domain.ScheduledTaskRun{ID: 2, TaskKey: "expire_reminder", Status: "success", StartedAt: time.Now()}
	committed, err = g.CompleteTaskRun(ctx, runB, "expire_reminder", "worker-B", leaseB.FencingToken)
	if err != nil {
		t.Fatalf("worker B complete: %v", err)
	}
	if !committed {
		t.Fatalf("worker B should commit while holding the lease")
	}
}

// TestTaskLeaseRenewKeepsOwnership verifies the heartbeat renewal path.
func TestTaskLeaseRenewKeepsOwnership(t *testing.T) {
	g := newTaskLeaseTestRepo(t)
	ctx := context.Background()
	lease, ok, err := g.AcquireTaskLease(ctx, "plugin_schedule", "worker-A", 2*time.Second)
	if err != nil || !ok {
		t.Fatalf("acquire: ok=%v err=%v", ok, err)
	}
	time.Sleep(50 * time.Millisecond)
	renewed, err := g.RenewTaskLease(ctx, "plugin_schedule", "worker-A", lease.FencingToken, 2*time.Second)
	if err != nil || !renewed {
		t.Fatalf("renew: ok=%v err=%v", renewed, err)
	}
	// Wrong owner/token must not renew.
	if ok, err := g.RenewTaskLease(ctx, "plugin_schedule", "worker-B", lease.FencingToken, 2*time.Second); err != nil || ok {
		t.Fatalf("foreign renew must fail: ok=%v err=%v", ok, err)
	}
	if err := g.ReleaseTaskLease(ctx, "plugin_schedule", "worker-A", lease.FencingToken); err != nil {
		t.Fatalf("release: %v", err)
	}
}
