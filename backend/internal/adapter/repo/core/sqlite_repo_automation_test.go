package repo_test

import (
	"context"
	"time"

	"testing"

	"xiaoheiplay/internal/domain"
)

func TestSQLiteRepo_AutomationLogsAndProvisionJobs(t *testing.T) {
	_, r := newTestRepo(t)
	ctx := context.Background()

	cycle := &domain.BillingCycle{Name: "monthly", Months: 1, Multiplier: 1.0, MinQty: 1, MaxQty: 12, Active: true, SortOrder: 1}
	if err := r.CreateBillingCycle(ctx, cycle); err != nil {
		t.Fatalf("create billing cycle: %v", err)
	}
	if err := r.DeleteBillingCycle(ctx, cycle.ID); err != nil {
		t.Fatalf("delete billing cycle: %v", err)
	}

	logEntry := &domain.AutomationLog{
		OrderID:      10,
		OrderItemID:  11,
		Action:       "create",
		RequestJSON:  "{}",
		ResponseJSON: "{}",
		Success:      true,
		Message:      "ok",
	}
	if err := r.CreateAutomationLog(ctx, logEntry); err != nil {
		t.Fatalf("create automation log: %v", err)
	}
	logs, total, err := r.ListAutomationLogs(ctx, logEntry.OrderID, 10, 0)
	if err != nil {
		t.Fatalf("list automation logs: %v", err)
	}
	if total == 0 || len(logs) == 0 {
		t.Fatalf("expected automation logs")
	}
	if err := r.PurgeAutomationLogs(ctx, time.Now().Add(1*time.Hour)); err != nil {
		t.Fatalf("purge automation logs: %v", err)
	}

	job := &domain.ProvisionJob{
		OrderID:     20,
		OrderItemID: 21,
		HostID:      99,
		HostName:    "vm-1",
		Status:      "pending",
		Attempts:    0,
		NextRunAt:   time.Unix(0, 0),
		LastError:   "",
	}
	if err := r.CreateOrUpdateProvisionJob(ctx, job); err != nil {
		t.Fatalf("create provision job: %v", err)
	}
	due, err := r.ListDueProvisionJobs(ctx, 10)
	if err != nil {
		t.Fatalf("list due provision jobs: %v", err)
	}
	if len(due) == 0 {
		t.Fatalf("expected due provision job")
	}
	jobUpdate := due[0]
	jobUpdate.Status = "retry"
	jobUpdate.Attempts = 1
	jobUpdate.LastError = "retrying"
	jobUpdate.NextRunAt = time.Now().Add(5 * time.Minute)
	if ok, err := r.UpdateProvisionJob(ctx, jobUpdate); err != nil || !ok {
		t.Fatalf("update provision job: ok=%v err=%v", ok, err)
	}
}

func TestSQLiteRepo_ListDueProvisionJobs_DoesNotStarveDueBehindFutureRunning(t *testing.T) {
	_, r := newTestRepo(t)
	ctx := context.Background()

	// Many running jobs whose next_run_at is in the future, all created before
	// (lower id than) a pending job that is actually due. With the in-memory-only
	// next_run_at filter, the capped fetch (max(limit*5, 50)) returns only the
	// future running rows and the due job is never examined -> starvation.
	future := time.Now().Add(24 * time.Hour)
	for i := 1; i <= 60; i++ {
		job := &domain.ProvisionJob{
			OrderID:     1000,
			OrderItemID: int64(2000 + i),
			HostID:      int64(i),
			HostName:    "vm-run",
			Status:      "running",
			Attempts:    2,
			NextRunAt:   future,
			LastError:   "",
		}
		if err := r.CreateOrUpdateProvisionJob(ctx, job); err != nil {
			t.Fatalf("create running job %d: %v", i, err)
		}
	}
	due := &domain.ProvisionJob{
		OrderID:     1000,
		OrderItemID: 9001,
		HostID:      99,
		HostName:    "vm-due",
		Status:      "pending",
		Attempts:    0,
		NextRunAt:   time.Now().Add(-1 * time.Minute),
		LastError:   "",
	}
	if err := r.CreateOrUpdateProvisionJob(ctx, due); err != nil {
		t.Fatalf("create due job: %v", err)
	}

	got, err := r.ListDueProvisionJobs(ctx, 1)
	if err != nil {
		t.Fatalf("list due provision jobs: %v", err)
	}
	if len(got) == 0 {
		t.Fatalf("expected the due pending job to be returned despite future running jobs")
	}
	if got[0].OrderItemID != due.OrderItemID {
		t.Fatalf("expected due job order_item_id=%d, got %d", due.OrderItemID, got[0].OrderItemID)
	}
}

func TestSQLiteRepo_ListDueProvisionJobs_WithRFC3339Timezone(t *testing.T) {
	_, r := newTestRepo(t)
	ctx := context.Background()

	job := &domain.ProvisionJob{
		OrderID:     101,
		OrderItemID: 202,
		HostID:      303,
		HostName:    "vm-timezone",
		Status:      "pending",
		Attempts:    0,
		NextRunAt:   time.Now().In(time.FixedZone("CST+8", 8*3600)).Add(-2 * time.Minute),
		LastError:   "",
	}
	if err := r.CreateOrUpdateProvisionJob(ctx, job); err != nil {
		t.Fatalf("create provision job: %v", err)
	}
	due, err := r.ListDueProvisionJobs(ctx, 10)
	if err != nil {
		t.Fatalf("list due provision jobs: %v", err)
	}
	found := false
	for _, item := range due {
		if item.OrderItemID == job.OrderItemID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected timezone due job to be listed")
	}
}

// TestSQLiteRepo_ClaimProvisionJob_ExactlyOneWinner verifies the P0-17 atomic
// claim: two workers racing for the same pending job must see exactly one
// winner, so the job cannot be double-consumed.
func TestSQLiteRepo_ClaimProvisionJob_ExactlyOneWinner(t *testing.T) {
	_, r := newTestRepo(t)
	ctx := context.Background()

	order := domain.Order{UserID: 1, OrderNo: "O-CLAIM", Status: domain.OrderStatusProvisioning, TotalAmount: 100, Currency: "CNY"}
	if err := r.CreateOrder(ctx, &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	item := domain.OrderItem{OrderID: order.ID, SpecJSON: "{}", Qty: 1, Amount: 100, Status: domain.OrderItemStatusProvisioning, AutomationInstanceID: "", Action: "create", DurationMonths: 1}
	if err := r.CreateOrderItems(ctx, []domain.OrderItem{item}); err != nil {
		t.Fatalf("create item: %v", err)
	}
	job := domain.ProvisionJob{OrderID: order.ID, OrderItemID: item.ID, HostID: 1, HostName: "h", Status: "pending", NextRunAt: time.Now().Add(-time.Minute)}
	if err := r.CreateOrUpdateProvisionJob(ctx, &job); err != nil {
		t.Fatalf("create job: %v", err)
	}

	const workers = 8
	start := make(chan struct{})
	results := make(chan bool, workers)
	for i := 0; i < workers; i++ {
		go func() {
			<-start
			_, claimed, _ := r.ClaimProvisionJob(ctx, job.ID, "worker", time.Now().Add(5*time.Second))
			results <- claimed
		}()
	}
	close(start)
	wins := 0
	for i := 0; i < workers; i++ {
		if <-results {
			wins++
		}
	}
	if wins != 1 {
		t.Fatalf("expected exactly one worker to claim the job, got %d", wins)
	}
}

// TestSQLiteRepo_ListDueResizeTasksReclaimsStaleRunning locks the P0-17 resize
// crash window: a task left running by a crashed worker (started_at older than
// the stale threshold) must be picked up again so the resize is not stuck
// forever.
func TestSQLiteRepo_ListDueResizeTasksReclaimsStaleRunning(t *testing.T) {
	_, r := newTestRepo(t)
	ctx := context.Background()

	now := time.Now()
	stale := now.Add(-2 * time.Hour)
	fresh := now.Add(-time.Minute)
	tasks := []domain.ResizeTask{
		{ID: 1, VPSID: 1, OrderID: 1, OrderItemID: 1, Status: domain.ResizeTaskStatusRunning, StartedAt: &stale},
		{ID: 2, VPSID: 2, OrderID: 2, OrderItemID: 2, Status: domain.ResizeTaskStatusRunning, StartedAt: &fresh},
		{ID: 3, VPSID: 3, OrderID: 3, OrderItemID: 3, Status: domain.ResizeTaskStatusPending},
	}
	for _, task := range tasks {
		if err := r.CreateResizeTask(ctx, &task); err != nil {
			t.Fatalf("create task %d: %v", task.ID, err)
		}
	}

	got, err := r.ListDueResizeTasks(ctx, 10)
	if err != nil {
		t.Fatalf("list due: %v", err)
	}
	seen := map[int64]bool{}
	for _, task := range got {
		seen[task.ID] = true
	}
	if !seen[1] {
		t.Fatalf("stale running task (id=1) was not reclaimed, got %+v", got)
	}
	if seen[2] {
		t.Fatalf("fresh running task (id=2) must not be reclaimed, got %+v", got)
	}
	if !seen[3] {
		t.Fatalf("pending task (id=3) must be due, got %+v", got)
	}
}
