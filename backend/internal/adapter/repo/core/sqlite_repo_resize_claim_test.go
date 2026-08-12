package repo

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"xiaoheiplay/internal/domain"
)

// runResizeClaimConcurrency exercises the atomic per-task lease claim on the
// given repo: a task is claimed at most once (the loser's lease claim is
// rejected, never overwriting the winner), and every due task is claimable.
// The one-active-resize-per-VPS invariant is enforced at order creation via
// HasPendingResizeTask + vps_lifecycle_locks, not at claim time.
func runResizeClaimConcurrency(t *testing.T, g *GormRepo) {
	t.Helper()
	ctx := context.Background()
	now := time.Now()
	tasks := []*domain.ResizeTask{
		{VPSID: 1, OrderID: 101, OrderItemID: 1001, Status: domain.ResizeTaskStatusPending, ScheduledAt: &now},
		{VPSID: 1, OrderID: 102, OrderItemID: 1002, Status: domain.ResizeTaskStatusPending, ScheduledAt: &now},
		{VPSID: 2, OrderID: 103, OrderItemID: 1003, Status: domain.ResizeTaskStatusPending, ScheduledAt: &now},
		{VPSID: 3, OrderID: 104, OrderItemID: 1004, Status: domain.ResizeTaskStatusPending, ScheduledAt: &now},
	}
	for _, task := range tasks {
		if err := g.CreateResizeTask(ctx, task); err != nil {
			t.Fatalf("create task: %v", err)
		}
	}

	// Two workers race to claim everything by per-task lease.
	var wg sync.WaitGroup
	claimed := make(chan domain.ResizeTask, 8)
	for w := 0; w < 2; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			due, err := g.ListDueResizeTasks(ctx, 10)
			if err != nil {
				t.Errorf("list due: %v", err)
				return
			}
			for _, task := range due {
				got, ok, err := g.ClaimResizeTask(ctx, task.ID, "worker-A", time.Now().Add(domain.ResizeTaskStaleDuration))
				if err != nil {
					t.Errorf("claim: %v", err)
					continue
				}
				if ok {
					claimed <- got
				}
			}
		}()
	}
	wg.Wait()
	close(claimed)

	byID := map[int64]domain.ResizeTask{}
	for task := range claimed {
		if _, dup := byID[task.ID]; dup {
			t.Fatalf("task %d claimed more than once", task.ID)
		}
		byID[task.ID] = task
	}
	// A fresh lease is never stolen, but distinct tasks are all claimable.
	if len(byID) != 4 {
		t.Fatalf("expected 4 distinct tasks claimed, got %d", len(byID))
	}
	for id, task := range byID {
		if task.Status != domain.ResizeTaskStatusRunning {
			t.Fatalf("task %d status = %s, want running", id, task.Status)
		}
		if task.LeaseOwner == "" {
			t.Fatalf("task %d has empty lease owner", id)
		}
	}
}

func TestResizeClaimConcurrencySQLite(t *testing.T) {
	g := newTaskLeaseTestRepo(t)
	runResizeClaimConcurrency(t, g)
}

func TestResizeCompleteFencedSQLite(t *testing.T) {
	g := newTaskLeaseTestRepo(t)
	runResizeCompleteFenced(t, g)
}

func runResizeCompleteFenced(t *testing.T, g *GormRepo) {
	t.Helper()
	ctx := context.Background()
	now := time.Now()
	task := &domain.ResizeTask{VPSID: 9, OrderID: 1, OrderItemID: 2, Status: domain.ResizeTaskStatusPending, ScheduledAt: &now}
	if err := g.CreateResizeTask(ctx, task); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, claimed, err := g.ClaimResizeTask(ctx, task.ID, "worker-A", time.Now().Add(domain.ResizeTaskStaleDuration))
	if err != nil || !claimed {
		t.Fatalf("claim: claimed=%v err=%v", claimed, err)
	}
	finish := time.Now()
	got.Status = domain.ResizeTaskStatusDone
	got.FinishedAt = &finish

	// A stale worker (wrong fence) must be fenced out.
	stale := got
	stale.Fence = got.Fence + 1
	if ok, err := g.UpdateResizeTask(ctx, stale); err != nil || ok {
		t.Fatalf("stale-fence update must be rejected: ok=%v err=%v", ok, err)
	}
	// The lease holder commits.
	if ok, err := g.UpdateResizeTask(ctx, got); err != nil || !ok {
		t.Fatalf("holder update: ok=%v err=%v", ok, err)
	}
	persisted, err := g.GetResizeTask(ctx, got.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if persisted.Status != domain.ResizeTaskStatusDone {
		t.Fatalf("expected done status, got %s", persisted.Status)
	}
	// A completed task is no longer due and does not block a future resize.
	if due, err := g.ListDueResizeTasks(ctx, 10); err != nil || len(due) != 0 {
		t.Fatalf("expected no due resize after completion: n=%d err=%v", len(due), err)
	}
}

// runProvisionClaimConcurrency exercises the atomic per-job lease claim: two
// concurrent workers must never claim the same job twice.
func runProvisionClaimConcurrency(t *testing.T, g *GormRepo) {
	t.Helper()
	ctx := context.Background()
	jobs := []*domain.ProvisionJob{
		{OrderID: 1, OrderItemID: 1001, HostID: 1, HostName: "vm1", Status: "pending", NextRunAt: time.Now().Add(-time.Minute)},
		{OrderID: 2, OrderItemID: 1002, HostID: 2, HostName: "vm2", Status: "pending", NextRunAt: time.Now().Add(-time.Minute)},
		{OrderID: 3, OrderItemID: 1003, HostID: 3, HostName: "vm3", Status: "pending", NextRunAt: time.Now().Add(-time.Minute)},
		{OrderID: 4, OrderItemID: 1004, HostID: 4, HostName: "vm4", Status: "pending", NextRunAt: time.Now().Add(-time.Minute)},
	}
	for _, job := range jobs {
		if err := g.CreateOrUpdateProvisionJob(ctx, job); err != nil {
			t.Fatalf("create job: %v", err)
		}
	}
	var wg sync.WaitGroup
	claimed := make(chan domain.ProvisionJob, 8)
	for w := 0; w < 2; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			due, err := g.ListDueProvisionJobs(ctx, 10)
			if err != nil {
				t.Errorf("list due: %v", err)
				return
			}
			for _, job := range due {
				got, ok, err := g.ClaimProvisionJob(ctx, job.ID, "worker-A", time.Now().Add(domain.ProvisionLeaseDuration))
				if err != nil {
					t.Errorf("claim: %v", err)
					continue
				}
				if ok {
					claimed <- got
				}
			}
		}()
	}
	wg.Wait()
	close(claimed)
	byID := map[int64]domain.ProvisionJob{}
	for job := range claimed {
		if _, dup := byID[job.ID]; dup {
			t.Fatalf("job %d claimed more than once", job.ID)
		}
		byID[job.ID] = job
	}
	if len(byID) != 4 {
		t.Fatalf("expected 4 jobs claimed, got %d", len(byID))
	}
}

func TestProvisionClaimConcurrencySQLite(t *testing.T) {
	g := newTaskLeaseTestRepo(t)
	runProvisionClaimConcurrency(t, g)
}

func TestProvisionClaimConcurrencyMySQL(t *testing.T) {
	g := openMigratedDB(t, "MYSQL", os.Getenv("TEST_MYSQL_DSN"))
	runProvisionClaimConcurrency(t, g)
}

func TestProvisionClaimConcurrencyPostgreSQL(t *testing.T) {
	g := openMigratedDB(t, "PG", os.Getenv("TEST_PG_DSN"))
	runProvisionClaimConcurrency(t, g)
}

// runResizeLeaseRecovery verifies a running task is not reclaimed while its
// lease is fresh, is reclaimed after the lease expires (crash recovery), and
// the stale worker's final write is fenced out.
func runResizeLeaseRecovery(t *testing.T, g *GormRepo) {
	t.Helper()
	ctx := context.Background()
	now := time.Now()
	task := &domain.ResizeTask{VPSID: 11, OrderID: 1, OrderItemID: 2, Status: domain.ResizeTaskStatusPending, ScheduledAt: &now}
	if err := g.CreateResizeTask(ctx, task); err != nil {
		t.Fatalf("create: %v", err)
	}
	first, claimed, err := g.ClaimResizeTask(ctx, task.ID, "worker-A", time.Now().Add(domain.ResizeTaskStaleDuration))
	if err != nil || !claimed {
		t.Fatalf("claim: claimed=%v err=%v", claimed, err)
	}
	// A healthy running task is not reclaimed while its lease is fresh.
	if _, ok, err := g.ClaimResizeTask(ctx, task.ID, "worker-B", time.Now().Add(domain.ResizeTaskStaleDuration)); err != nil || ok {
		t.Fatalf("healthy running task must not be reclaimed: ok=%v err=%v", ok, err)
	}
	// Simulate a crashed worker: expire the lease directly in the DB.
	past := time.Now().Add(-time.Minute)
	if err := g.gdb.WithContext(ctx).Model(&resizeTaskRow{}).Where("id = ?", first.ID).Update("lease_until", past).Error; err != nil {
		t.Fatalf("expire lease: %v", err)
	}
	second, reclaimed, err := g.ClaimResizeTask(ctx, task.ID, "worker-B", time.Now().Add(domain.ResizeTaskStaleDuration))
	if err != nil || !reclaimed {
		t.Fatalf("reclaim: claimed=%v err=%v", reclaimed, err)
	}
	if second.Fence == first.Fence {
		t.Fatalf("expected fresh fence on reclaim")
	}
	// Old worker's final write must be fenced out.
	finish := time.Now()
	stale := first
	stale.Status = domain.ResizeTaskStatusDone
	stale.FinishedAt = &finish
	if ok, err := g.UpdateResizeTask(ctx, stale); err != nil || ok {
		t.Fatalf("stale worker update must be fenced: ok=%v err=%v", ok, err)
	}
	// New worker's final write commits.
	done2 := second
	done2.Status = domain.ResizeTaskStatusDone
	done2.FinishedAt = &finish
	if ok, err := g.UpdateResizeTask(ctx, done2); err != nil || !ok {
		t.Fatalf("new worker update: ok=%v err=%v", ok, err)
	}
	// The VPS is now free for a new resize.
	if pending, err := g.HasPendingResizeTask(ctx, 11); err != nil || pending {
		t.Fatalf("expected no pending resize after completion: pending=%v err=%v", pending, err)
	}
}

func TestResizeLeaseRecoverySQLite(t *testing.T) {
	g := newTaskLeaseTestRepo(t)
	runResizeLeaseRecovery(t, g)
}

func TestResizeLeaseRecoveryMySQL(t *testing.T) {
	g := openMigratedDB(t, "MYSQL", os.Getenv("TEST_MYSQL_DSN"))
	runResizeLeaseRecovery(t, g)
}

func TestResizeLeaseRecoveryPostgreSQL(t *testing.T) {
	g := openMigratedDB(t, "PG", os.Getenv("TEST_PG_DSN"))
	runResizeLeaseRecovery(t, g)
}

// runProvisionLeaseRecovery verifies a provision job is not reclaimed while its
// lease is fresh, is reclaimed after the lease expires, and the stale worker's
// final write is fenced out.
func runProvisionLeaseRecovery(t *testing.T, g *GormRepo) {
	t.Helper()
	ctx := context.Background()
	job := &domain.ProvisionJob{
		OrderID:     7,
		OrderItemID: 77,
		HostID:      1,
		HostName:    "vm1",
		Status:      "pending",
		Attempts:    0,
		NextRunAt:   time.Now().Add(-time.Minute),
	}
	if err := g.CreateOrUpdateProvisionJob(ctx, job); err != nil {
		t.Fatalf("create job: %v", err)
	}
	first, claimed, err := g.ClaimProvisionJob(ctx, job.ID, "worker-A", time.Now().Add(domain.ProvisionLeaseDuration))
	if err != nil || !claimed {
		t.Fatalf("claim: claimed=%v err=%v", claimed, err)
	}
	// A healthy running job is not reclaimed while its lease is fresh.
	if _, ok, err := g.ClaimProvisionJob(ctx, job.ID, "worker-B", time.Now().Add(domain.ProvisionLeaseDuration)); err != nil || ok {
		t.Fatalf("healthy running job must not be reclaimed: ok=%v err=%v", ok, err)
	}
	// Simulate a crashed worker: expire the lease directly in the DB.
	past := time.Now().Add(-time.Minute)
	if err := g.gdb.WithContext(ctx).Model(&provisionJobRow{}).Where("id = ?", first.ID).Update("lease_until", past).Error; err != nil {
		t.Fatalf("expire lease: %v", err)
	}
	second, reclaimed, err := g.ClaimProvisionJob(ctx, job.ID, "worker-B", time.Now().Add(domain.ProvisionLeaseDuration))
	if err != nil || !reclaimed {
		t.Fatalf("reclaim: claimed=%v err=%v", reclaimed, err)
	}
	if second.Fence == first.Fence {
		t.Fatalf("expected fresh fence on reclaim")
	}
	// Old worker's final write must be fenced out.
	done := first
	done.Status = "done"
	done.NextRunAt = time.Now().Add(365 * 24 * time.Hour)
	if ok, err := g.UpdateProvisionJob(ctx, done); err != nil || ok {
		t.Fatalf("stale worker update must be fenced: ok=%v err=%v", ok, err)
	}
	// New worker's final write commits.
	done2 := second
	done2.Status = "done"
	done2.NextRunAt = time.Now().Add(365 * 24 * time.Hour)
	if ok, err := g.UpdateProvisionJob(ctx, done2); err != nil || !ok {
		t.Fatalf("new worker update: ok=%v err=%v", ok, err)
	}
	// No due jobs remain.
	if jobs, err := g.ListDueProvisionJobs(ctx, 10); err != nil || len(jobs) != 0 {
		t.Fatalf("expected no due jobs after done: n=%d err=%v", len(jobs), err)
	}
}

func TestProvisionLeaseRecoverySQLite(t *testing.T) {
	g := newTaskLeaseTestRepo(t)
	runProvisionLeaseRecovery(t, g)
}

func TestProvisionLeaseRecoveryMySQL(t *testing.T) {
	g := openMigratedDB(t, "MYSQL", os.Getenv("TEST_MYSQL_DSN"))
	runProvisionLeaseRecovery(t, g)
}

func TestProvisionLeaseRecoveryPostgreSQL(t *testing.T) {
	g := openMigratedDB(t, "PG", os.Getenv("TEST_PG_DSN"))
	runProvisionLeaseRecovery(t, g)
}
