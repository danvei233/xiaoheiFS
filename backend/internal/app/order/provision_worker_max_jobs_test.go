package order_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	apporder "xiaoheiplay/internal/app/order"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

// Regression: provision_watchdog_max_jobs must be able to raise the per-cycle
// cap. The old limit logic clamped any configured value >= the caller's limit
// back down to the default of 8, so operators could never raise the throughput.
func TestProvisionWorker_RespectsConfiguredMaxJobsAboveDefault(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	if err := repo.UpsertSetting(context.Background(), domain.Setting{Key: "provision_watchdog_max_jobs", ValueJSON: "50"}); err != nil {
		t.Fatalf("upsert setting: %v", err)
	}

	for i := 0; i < 12; i++ {
		job := domain.ProvisionJob{
			OrderID:     999,
			OrderItemID: int64(5000 + i),
			HostID:      int64(7000 + i),
			HostName:    fmt.Sprintf("vm-missing-%d", i),
			Status:      "pending",
			Attempts:    0,
			NextRunAt:   time.Now().UTC().Add(-time.Minute),
		}
		if err := repo.CreateOrUpdateProvisionJob(context.Background(), &job); err != nil {
			t.Fatalf("create job %d: %v", i, err)
		}
	}

	autoResolver := &testutil.FakeAutomationResolver{Client: &testutil.FakeAutomationClient{}}
	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, autoResolver, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)

	if err := svc.ProcessProvisionJobs(context.Background(), 50); err != nil {
		t.Fatalf("process jobs: %v", err)
	}
	due, err := repo.ListDueProvisionJobs(context.Background(), 100)
	if err != nil {
		t.Fatalf("list due jobs: %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("expected all 12 jobs processed when max_jobs=50, %d still due", len(due))
	}
}
