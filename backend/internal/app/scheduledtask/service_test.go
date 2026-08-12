package scheduledtask

import (
	"errors"
	"gorm.io/gorm"
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

type fakeVPS struct {
	panicOnRefresh bool
	// blockOnRefresh makes RefreshAll block until the passed context is done,
	// used to observe lease-lost cancellation of a running task.
	blockOnRefresh bool
	// refreshCtxDone is closed when a blocking RefreshAll returns via ctx done.
	refreshCtxDone chan struct{}
	mu             sync.Mutex
	refreshCalls   int
}

func (f *fakeVPS) RefreshAll(ctx context.Context, limit int) (int, error) {
	f.mu.Lock()
	f.refreshCalls++
	block := f.blockOnRefresh
	done := f.refreshCtxDone
	f.mu.Unlock()
	if f.panicOnRefresh {
		panic("boom")
	}
	if block {
		<-ctx.Done()
		if done != nil {
			close(done)
		}
		return 0, ctx.Err()
	}
	return 0, nil
}
func (f *fakeVPS) AutoDeleteExpired(ctx context.Context) error { return nil }
func (f *fakeVPS) AutoLockExpired(ctx context.Context) error   { return nil }

type fakeSettingsRepo struct{}

func (f *fakeSettingsRepo) UpdateSettingAtomic(ctx context.Context, key string, mutate func(current string) (string, error)) error {
	current, err := f.GetSetting(ctx, key)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			current = domain.Setting{}
		} else {
			return err
		}
	}
	next, err := mutate(current.ValueJSON)
	if err != nil {
		return err
	}
	return f.UpsertSetting(ctx, domain.Setting{Key: key, ValueJSON: next})
}
func (f *fakeSettingsRepo) GetSetting(ctx context.Context, key string) (domain.Setting, error) {
	return domain.Setting{}, appshared.ErrNotFound
}
func (f *fakeSettingsRepo) UpsertSetting(ctx context.Context, setting domain.Setting) error {
	return nil
}
func (f *fakeSettingsRepo) UpsertSettings(ctx context.Context, settings []domain.Setting) error {
	return nil
}
func (f *fakeSettingsRepo) ListSettings(ctx context.Context) ([]domain.Setting, error) {
	return nil, nil
}
func (f *fakeSettingsRepo) ListEmailTemplates(ctx context.Context) ([]domain.EmailTemplate, error) {
	return nil, nil
}
func (f *fakeSettingsRepo) GetEmailTemplate(ctx context.Context, id int64) (domain.EmailTemplate, error) {
	return domain.EmailTemplate{}, appshared.ErrNotFound
}
func (f *fakeSettingsRepo) UpsertEmailTemplate(ctx context.Context, tmpl *domain.EmailTemplate) error {
	return nil
}
func (f *fakeSettingsRepo) DeleteEmailTemplate(ctx context.Context, id int64) error { return nil }

type fakeRunsRepo struct {
	mu     sync.Mutex
	runs   []domain.ScheduledTaskRun
	leases map[string]domain.ScheduledTaskLease
	order  []string
}

func newFakeRunsRepo() *fakeRunsRepo {
	return &fakeRunsRepo{leases: map[string]domain.ScheduledTaskLease{}}
}

func (f *fakeRunsRepo) CreateTaskRun(ctx context.Context, run *domain.ScheduledTaskRun) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.runs = append(f.runs, *run)
	return nil
}
func (f *fakeRunsRepo) UpdateTaskRun(ctx context.Context, run domain.ScheduledTaskRun) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := range f.runs {
		if f.runs[i].TaskKey == run.TaskKey && f.runs[i].Status == "running" {
			f.runs[i] = run
			break
		}
	}
	return nil
}
func (f *fakeRunsRepo) ListTaskRuns(ctx context.Context, key string, limit int) ([]domain.ScheduledTaskRun, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]domain.ScheduledTaskRun, 0, limit)
	for i := len(f.runs) - 1; i >= 0 && len(out) < limit; i-- {
		if f.runs[i].TaskKey == key {
			out = append(out, f.runs[i])
		}
	}
	return out, nil
}

func (f *fakeRunsRepo) AcquireTaskLease(ctx context.Context, taskKey, owner string, leaseFor time.Duration) (domain.ScheduledTaskLease, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	now := time.Now()
	if cur, ok := f.leases[taskKey]; ok && cur.LeaseUntil.After(now) {
		return domain.ScheduledTaskLease{}, false, nil
	}
	lease := domain.ScheduledTaskLease{TaskKey: taskKey, Owner: owner, LeaseUntil: now.Add(leaseFor), FencingToken: int64(time.Now().UnixNano())}
	if f.leases == nil {
		f.leases = map[string]domain.ScheduledTaskLease{}
	}
	f.leases[taskKey] = lease
	return lease, true, nil
}

func (f *fakeRunsRepo) RenewTaskLease(ctx context.Context, taskKey, owner string, token int64, leaseFor time.Duration) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	cur, ok := f.leases[taskKey]
	if !ok || cur.Owner != owner || cur.FencingToken != token {
		return false, nil
	}
	cur.LeaseUntil = time.Now().Add(leaseFor)
	f.leases[taskKey] = cur
	return true, nil
}

func (f *fakeRunsRepo) ReleaseTaskLease(ctx context.Context, taskKey, owner string, token int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cur, ok := f.leases[taskKey]
	if ok && cur.Owner == owner && cur.FencingToken == token {
		delete(f.leases, taskKey)
	}
	return nil
}

func (f *fakeRunsRepo) CompleteTaskRun(ctx context.Context, run domain.ScheduledTaskRun, taskKey, owner string, token int64) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	cur, ok := f.leases[taskKey]
	if !ok || cur.Owner != owner || cur.FencingToken != token || cur.LeaseUntil.Before(time.Now()) {
		return false, nil
	}
	for i := range f.runs {
		if f.runs[i].ID == run.ID {
			f.runs[i] = run
			break
		}
	}
	return true, nil
}

var _ appports.ScheduledTaskRunRepository = (*fakeRunsRepo)(nil)
var _ appports.SettingsRepository = (*fakeSettingsRepo)(nil)

func newTaskService(vps *fakeVPS, runs appports.ScheduledTaskRunRepository) *Service {
	return NewService(&fakeSettingsRepo{}, vps, nil, nil, runs)
}

func TestExecuteTaskRecoversPanic(t *testing.T) {
	vps := &fakeVPS{panicOnRefresh: true}
	runs := newFakeRunsRepo()
	svc := newTaskService(vps, runs)
	cfg := ScheduledTaskConfig{Key: "vps_refresh", Enabled: true, Strategy: TaskStrategyInterval, IntervalSec: 300}

	svc.executeTask(context.Background(), cfg)

	deadline := time.Now().Add(2 * time.Second)
	for {
		svc.mu.Lock()
		rt := svc.runtime["vps_refresh"]
		running := rt.running
		status := rt.lastStatus
		errMsg := rt.lastError
		svc.mu.Unlock()
		if !running {
			if status != "failed" {
				t.Fatalf("expected task marked failed, got status=%q err=%q", status, errMsg)
			}
			if errMsg == "" {
				t.Fatalf("expected panic error recorded")
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("task did not finish after panic; still running")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestValidateTaskConfigRejectsHugeInterval(t *testing.T) {
	cfg := ScheduledTaskConfig{Strategy: TaskStrategyInterval, IntervalSec: int(maxIntervalSec) + 1}
	if err := validateTaskConfig(cfg); err == nil {
		t.Fatalf("expected huge interval to be rejected")
	}
	ok := ScheduledTaskConfig{Strategy: TaskStrategyInterval, IntervalSec: 300}
	if err := validateTaskConfig(ok); err != nil {
		t.Fatalf("expected 300s interval accepted, got %v", err)
	}
}

func TestHydrateLastRunsRestoresDailyLastRun(t *testing.T) {
	vps := &fakeVPS{}
	runs := newFakeRunsRepo()
	svc := newTaskService(vps, runs)
	now := time.Now()

	// Simulate a previous daily run persisted before restart.
	run := &domain.ScheduledTaskRun{TaskKey: "expire_reminder", Status: "success", StartedAt: now.Add(-time.Hour)}
	if err := runs.CreateTaskRun(context.Background(), run); err != nil {
		t.Fatalf("seed run: %v", err)
	}

	svc.hydrateLastRuns(context.Background())
	svc.mu.Lock()
	rt := svc.runtime["expire_reminder"]
	svc.mu.Unlock()
	if rt == nil || rt.lastRun.IsZero() {
		t.Fatalf("expected daily last run to be hydrated from persisted runs")
	}
	if !rt.lastRun.Equal(now.Add(-time.Hour).Truncate(time.Nanosecond)) {
		t.Fatalf("expected hydrated lastRun to match seeded run, got %s", rt.lastRun.Format(time.RFC3339))
	}
}

func TestHydrateLastRunsSkipsCrashedRunningRun(t *testing.T) {
	vps := &fakeVPS{}
	runs := newFakeRunsRepo()
	svc := newTaskService(vps, runs)
	now := time.Now()

	// A run left "running" by a crashed worker must not suppress the next run.
	run := &domain.ScheduledTaskRun{TaskKey: "expire_reminder", Status: "running", StartedAt: now.Add(-time.Hour)}
	if err := runs.CreateTaskRun(context.Background(), run); err != nil {
		t.Fatalf("seed run: %v", err)
	}

	svc.hydrateLastRuns(context.Background())
	svc.mu.Lock()
	rt := svc.runtime["expire_reminder"]
	svc.mu.Unlock()
	if rt != nil && !rt.lastRun.IsZero() {
		t.Fatalf("expected crashed running run to be ignored for hydration")
	}
}

func TestExecuteTaskSkipsWhenLeaseHeldByAnotherInstance(t *testing.T) {
	vps := &fakeVPS{}
	runs := newFakeRunsRepo()
	svcA := newTaskService(vps, runs)
	svcB := newTaskService(vps, runs)

	// Acquire the lease directly as if instance A had.
	owner := svcA.instanceOwner()
	lease, ok, err := runs.AcquireTaskLease(context.Background(), "vps_refresh", owner, taskLeaseDuration)
	if err != nil || !ok {
		t.Fatalf("seed lease: ok=%v err=%v", ok, err)
	}
	_ = lease

	cfg := ScheduledTaskConfig{Key: "vps_refresh", Enabled: true, Strategy: TaskStrategyInterval, IntervalSec: 300}
	svcB.executeTask(context.Background(), cfg)

	// Instance B must not mark the task as running locally.
	svcB.mu.Lock()
	rt := svcB.runtime["vps_refresh"]
	svcB.mu.Unlock()
	if rt != nil && rt.running {
		t.Fatalf("instance B must not run a task whose lease is held by another instance")
	}
	if rt != nil && rt.lastStatus == "success" {
		t.Fatalf("instance B must not report a successful run for a leased task")
	}
}

func TestCompleteTaskRunFencedAfterLeaseLost(t *testing.T) {
	runs := newFakeRunsRepo()
	owner := "worker-A"
	lease, ok, err := runs.AcquireTaskLease(context.Background(), "expire_reminder", owner, taskLeaseDuration)
	if err != nil || !ok {
		t.Fatalf("seed lease: ok=%v err=%v", ok, err)
	}

	// A second worker steals the lease after A's lease expires.
	runs.mu.Lock()
	l := runs.leases["expire_reminder"]
	l.LeaseUntil = time.Now().Add(-time.Minute)
	runs.leases["expire_reminder"] = l
	runs.mu.Unlock()
	if _, ok, err := runs.AcquireTaskLease(context.Background(), "expire_reminder", "worker-B", taskLeaseDuration); err != nil || !ok {
		t.Fatalf("worker B steal: ok=%v err=%v", ok, err)
	}

	// Worker A tries to commit its run — must be rejected (fenced).
	run := domain.ScheduledTaskRun{ID: 1, TaskKey: "expire_reminder", Status: "success"}
	committed, err := runs.CompleteTaskRun(context.Background(), run, "expire_reminder", owner, lease.FencingToken)
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if committed {
		t.Fatalf("old worker committed after losing its lease")
	}
}

func TestShouldRunDailyClockBoundary(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*3600)
	// Daily task at 09:00 local.
	nowBefore := time.Date(2026, 8, 11, 8, 59, 59, 0, loc)
	if shouldRunDaily(nowBefore, time.Time{}, "09:00") {
		t.Fatalf("must not run before the daily target")
	}
	nowAt := time.Date(2026, 8, 11, 9, 0, 0, 0, loc)
	if !shouldRunDaily(nowAt, time.Time{}, "09:00") {
		t.Fatalf("must run at/after the daily target when never run")
	}
	// Ran earlier today at 08:00 (before today's 09:00 target) -> the 09:00
	// trigger has not fired yet, so it must run.
	ranTodayEarly := time.Date(2026, 8, 11, 8, 0, 0, 0, loc)
	if !shouldRunDaily(nowAt, ranTodayEarly, "09:00") {
		t.Fatalf("must run when last run predates today's target")
	}
	// Ran yesterday -> must run today.
	ranYesterday := time.Date(2026, 8, 10, 9, 5, 0, 0, loc)
	if !shouldRunDaily(nowAt, ranYesterday, "09:00") {
		t.Fatalf("must run when last run was before today's target")
	}
	// Same-day run at target time -> no second run.
	ranAtTarget := time.Date(2026, 8, 11, 9, 0, 0, 0, loc)
	if shouldRunDaily(nowAt, ranAtTarget, "09:00") {
		t.Fatalf("must not re-run at the exact target time")
	}
}

func TestIntervalRestartSkipsWhenWithinWindow(t *testing.T) {
	svc := &Service{runtime: map[string]*taskRuntime{}}
	cfg := ScheduledTaskConfig{Strategy: TaskStrategyInterval, IntervalSec: 300}
	now := time.Now()
	svc.mu.Lock()
	rt := svc.ensureRuntime(cfg.Key)
	rt.lastRun = now.Add(-60 * time.Second)
	svc.mu.Unlock()
	if svc.shouldRun(cfg) {
		t.Fatalf("interval task must not re-run within its window after restart")
	}

	svc.mu.Lock()
	rt.lastRun = now.Add(-301 * time.Second)
	svc.mu.Unlock()
	if !svc.shouldRun(cfg) {
		t.Fatalf("interval task must run once the window has elapsed")
	}
}

// waitTaskDone blocks until the given task key is no longer running on the
// service, mirroring how executeTask's completion is observed in existing tests.
func waitTaskDone(t *testing.T, svc *Service, key string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		svc.mu.Lock()
		rt := svc.runtime[key]
		running := rt != nil && rt.running
		svc.mu.Unlock()
		if !running {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("task %q did not finish in time", key)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// countCompletedRuns returns the number of non-running persisted runs for a task.
func countCompletedRuns(t *testing.T, runs *fakeRunsRepo, key string) int {
	t.Helper()
	runs.mu.Lock()
	defer runs.mu.Unlock()
	n := 0
	for _, r := range runs.runs {
		if r.TaskKey == key && !strings.EqualFold(strings.TrimSpace(r.Status), "running") {
			n++
		}
	}
	return n
}

// TestMultiInstanceDailyTaskRunsOnceAcrossInstances reproduces the cross-instance
// duplicate-execution window: two replicas share one runs repo and both hold a
// stale in-memory lastRun (hydrated at startup). Instance A completes the daily
// task; instance B, seeing the same stale lastRun, must NOT re-run it in the
// same period. Before the under-lease DB re-check B would run the task a second
// time.
func TestMultiInstanceDailyTaskRunsOnceAcrossInstances(t *testing.T) {
	vps := &fakeVPS{}
	runs := newFakeRunsRepo()
	svcA := newTaskService(vps, runs)
	svcB := newTaskService(vps, runs)

	cfg := ScheduledTaskConfig{Key: "expire_reminder", Enabled: true, Strategy: TaskStrategyDaily, DailyAt: "00:00"}
	// Both replicas hydrated "yesterday" as their last run (startup snapshot).
	yesterday := time.Now().Add(-24 * time.Hour)
	for _, svc := range []*Service{svcA, svcB} {
		svc.mu.Lock()
		svc.runtime[cfg.Key] = &taskRuntime{lastRun: yesterday}
		svc.mu.Unlock()
	}

	svcA.executeTask(context.Background(), cfg)
	waitTaskDone(t, svcA, cfg.Key)
	if got := countCompletedRuns(t, runs, cfg.Key); got != 1 {
		t.Fatalf("instance A: expected 1 completed run, got %d", got)
	}

	// Instance B has the same stale lastRun but must skip because A already
	// completed this period's run. No run row may be added.
	svcB.executeTask(context.Background(), cfg)
	if got := countCompletedRuns(t, runs, cfg.Key); got != 1 {
		t.Fatalf("instance B re-ran the daily task: expected 1 completed run total, got %d", got)
	}
}

// TestMultiInstanceIntervalTaskRunsOnceWithinWindow is the interval counterpart:
// A completes the interval task and B (stale lastRun) must not re-run inside the
// same window.
func TestMultiInstanceIntervalTaskRunsOnceWithinWindow(t *testing.T) {
	vps := &fakeVPS{}
	runs := newFakeRunsRepo()
	svcA := newTaskService(vps, runs)
	svcB := newTaskService(vps, runs)

	cfg := ScheduledTaskConfig{Key: "vps_refresh", Enabled: true, Strategy: TaskStrategyInterval, IntervalSec: 300}
	stale := time.Now().Add(-10 * time.Minute)
	for _, svc := range []*Service{svcA, svcB} {
		svc.mu.Lock()
		svc.runtime[cfg.Key] = &taskRuntime{lastRun: stale}
		svc.mu.Unlock()
	}

	svcA.executeTask(context.Background(), cfg)
	waitTaskDone(t, svcA, cfg.Key)
	if got := countCompletedRuns(t, runs, cfg.Key); got != 1 {
		t.Fatalf("instance A: expected 1 completed run, got %d", got)
	}

	svcB.executeTask(context.Background(), cfg)
	if got := countCompletedRuns(t, runs, cfg.Key); got != 1 {
		t.Fatalf("instance B re-ran the interval task: expected 1 completed run total, got %d", got)
	}
}

// failingRenewRunsRepo fails RenewTaskLease after a fixed number of calls so a
// task's heartbeat stops and the lease is lost while the task body is still
// running.
type failingRenewRunsRepo struct {
	*fakeRunsRepo
	failAfter  int
	renewCalls int
}

func (f *failingRenewRunsRepo) RenewTaskLease(ctx context.Context, taskKey, owner string, token int64, leaseFor time.Duration) (bool, error) {
	f.mu.Lock()
	f.renewCalls++
	fail := f.renewCalls >= f.failAfter
	f.mu.Unlock()
	if fail {
		return false, nil
	}
	return f.fakeRunsRepo.RenewTaskLease(ctx, taskKey, owner, token, leaseFor)
}

// TestExecuteTaskCancelsTaskOnLeaseLost verifies that when the heartbeat stops
// renewing the lease (lease lost to another instance), the running task's
// context is cancelled so external side effects stop early instead of running to
// completion and only being blocked by the fenced status write.
func TestExecuteTaskCancelsTaskOnLeaseLost(t *testing.T) {
	runs := &failingRenewRunsRepo{fakeRunsRepo: newFakeRunsRepo(), failAfter: 2}
	vps := &fakeVPS{blockOnRefresh: true, refreshCtxDone: make(chan struct{})}
	svc := newTaskService(vps, runs)
	svc.leaseFor = 300 * time.Millisecond

	cfg := ScheduledTaskConfig{Key: "vps_refresh", Enabled: true, Strategy: TaskStrategyInterval, IntervalSec: 300}
	svc.executeTask(context.Background(), cfg)

	select {
	case <-vps.refreshCtxDone:
		// The blocking task observed the context cancellation after the lease was
		// lost. Without the fix the task would still be blocked on ctx.Done().
	case <-time.After(5 * time.Second):
		t.Fatalf("task context was not cancelled after the lease was lost")
	}
	waitTaskDone(t, svc, cfg.Key)
}
