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
	if err := r.UpdateProvisionJob(ctx, jobUpdate); err != nil {
		t.Fatalf("update provision job: %v", err)
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
