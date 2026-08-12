package order_test

import (
	"context"
	"strings"
	"testing"
	"time"

	apporder "xiaoheiplay/internal/app/order"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

// panicProvisionClient panics inside GetHostInfo, modelling a buggy automation
// plugin that must not take down the whole watchdog process.
type panicProvisionClient struct {
	*testutil.FakeAutomationClient
}

func (c *panicProvisionClient) GetHostInfo(context.Context, int64) (appshared.AutomationHostInfo, error) {
	panic("boom: provisioning host info")
}

// Regression: a panic inside a single provision job must be contained, recorded
// on the job, and must not prevent the remaining jobs in the same batch from
// being processed.
func TestProvisionWorker_ContainsPanicAndContinues(t *testing.T) {
	db, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "prov-panic", "prov-panic@example.com", "pass")

	order := domain.Order{UserID: user.ID, OrderNo: "ORD-PROV-PANIC", Status: domain.OrderStatusProvisioning, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	item := domain.OrderItem{
		OrderID:   order.ID,
		PackageID: seed.Package.ID,
		SystemID:  seed.SystemImage.ID,
		Amount:    1000,
		Status:    domain.OrderItemStatusProvisioning,
		Action:    "create",
		SpecJSON:  "{}",
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{item}); err != nil {
		t.Fatalf("create item: %v", err)
	}
	items, _ := repo.ListOrderItems(context.Background(), order.ID)

	job1 := domain.ProvisionJob{
		OrderID:     order.ID,
		OrderItemID: items[0].ID,
		HostID:      8001,
		HostName:    "vm-panic",
		Status:      "pending",
		Attempts:    0,
		NextRunAt:   time.Now().UTC().Add(-time.Minute),
	}
	if err := repo.CreateOrUpdateProvisionJob(context.Background(), &job1); err != nil {
		t.Fatalf("create job1: %v", err)
	}
	job2 := domain.ProvisionJob{
		OrderID:     999,
		OrderItemID: 8888,
		HostID:      8002,
		HostName:    "vm-missing",
		Status:      "pending",
		Attempts:    0,
		NextRunAt:   time.Now().UTC().Add(-time.Minute),
	}
	if err := repo.CreateOrUpdateProvisionJob(context.Background(), &job2); err != nil {
		t.Fatalf("create job2: %v", err)
	}

	autoResolver := &testutil.FakeAutomationResolver{Client: &panicProvisionClient{FakeAutomationClient: &testutil.FakeAutomationClient{}}}
	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, autoResolver, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)

	if err := svc.ProcessProvisionJobs(context.Background(), 10); err != nil {
		t.Fatalf("process jobs should return after contained panic: %v", err)
	}

	var job1Status, job1Err string
	if err := db.QueryRow("SELECT status, last_error FROM provision_jobs WHERE id = ?", job1.ID).Scan(&job1Status, &job1Err); err != nil {
		t.Fatalf("query job1: %v", err)
	}
	if job1Status != "retry" {
		t.Fatalf("expected panic job to be left in retry, got %q", job1Status)
	}
	if !strings.Contains(job1Err, "panic") {
		t.Fatalf("expected panic message recorded in job, got %q", job1Err)
	}
	var job2Status string
	if err := db.QueryRow("SELECT status FROM provision_jobs WHERE id = ?", job2.ID).Scan(&job2Status); err != nil {
		t.Fatalf("query job2: %v", err)
	}
	if job2Status != "done" {
		t.Fatalf("expected job2 still processed after panic, got %q", job2Status)
	}
}
