package repo_test

import (
	"context"
	"testing"
	"time"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func TestSQLiteRepo_ProvisionLease_FreshLeaseNeverStolen(t *testing.T) {
	_, r := newTestRepo(t)
	ctx := context.Background()
	user := testutil.CreateUser(t, r, "lease-fresh", "lease-fresh@example.com", "pass")
	order := domain.Order{UserID: user.ID, OrderNo: "O-LEASE-FRESH", Status: domain.OrderStatusProvisioning, TotalAmount: 100, Currency: "CNY"}
	if err := r.CreateOrder(ctx, &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	job := domain.ProvisionJob{OrderID: order.ID, OrderItemID: 1, HostID: 1, HostName: "h", Status: "pending", NextRunAt: time.Now().Add(-time.Minute)}
	if err := r.CreateOrUpdateProvisionJob(ctx, &job); err != nil {
		t.Fatalf("create job: %v", err)
	}

	// Worker A claims the job and holds a fresh lease.
	claimedJob, claimed, err := r.ClaimProvisionJob(ctx, job.ID, "worker-A", time.Now().Add(domain.ProvisionLeaseDuration))
	if err != nil || !claimed {
		t.Fatalf("worker A claim: claimed=%v err=%v", claimed, err)
	}
	if claimedJob.Fence != 1 || claimedJob.LeaseOwner != "worker-A" {
		t.Fatalf("claim did not assign owner/fence: %+v", claimedJob)
	}

	// Worker B must NOT be able to claim it while the lease is fresh, even
	// though the job status is running and next_run_at is due.
	_, claimedB, err := r.ClaimProvisionJob(ctx, job.ID, "worker-B", time.Now().Add(domain.ProvisionLeaseDuration))
	if err != nil {
		t.Fatalf("worker B claim: %v", err)
	}
	if claimedB {
		t.Fatalf("a fresh lease was stolen by another worker")
	}
	// The due list must not surface a fresh-lease running job.
	due, err := r.ListDueProvisionJobs(ctx, 10)
	if err != nil {
		t.Fatalf("list due: %v", err)
	}
	for _, d := range due {
		if d.ID == job.ID {
			t.Fatalf("fresh-lease running job surfaced as due")
		}
	}
}

func TestSQLiteRepo_ProvisionLease_ExpiredLeaseTakenOverByCAS(t *testing.T) {
	_, r := newTestRepo(t)
	ctx := context.Background()
	user := testutil.CreateUser(t, r, "lease-expired", "lease-expired@example.com", "pass")
	order := domain.Order{UserID: user.ID, OrderNo: "O-LEASE-EXP", Status: domain.OrderStatusProvisioning, TotalAmount: 100, Currency: "CNY"}
	if err := r.CreateOrder(ctx, &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	job := domain.ProvisionJob{OrderID: order.ID, OrderItemID: 2, HostID: 1, HostName: "h", Status: "pending", NextRunAt: time.Now().Add(-time.Minute)}
	if err := r.CreateOrUpdateProvisionJob(ctx, &job); err != nil {
		t.Fatalf("create job: %v", err)
	}
	if _, claimed, err := r.ClaimProvisionJob(ctx, job.ID, "worker-A", time.Now().Add(-time.Second)); err != nil || !claimed {
		t.Fatalf("worker A claim: claimed=%v err=%v", claimed, err)
	}

	// Lease expired: worker B can take over via CAS, bumping the fence.
	taken, claimed, err := r.ClaimProvisionJob(ctx, job.ID, "worker-B", time.Now().Add(domain.ProvisionLeaseDuration))
	if err != nil || !claimed {
		t.Fatalf("worker B takeover: claimed=%v err=%v", claimed, err)
	}
	if taken.LeaseOwner != "worker-B" || taken.Fence != 2 {
		t.Fatalf("takeover did not reassign owner and bump fence: %+v", taken)
	}
}

func TestSQLiteRepo_ProvisionLease_FencedUpdateRejectsLostLease(t *testing.T) {
	_, r := newTestRepo(t)
	ctx := context.Background()
	user := testutil.CreateUser(t, r, "lease-fence", "lease-fence@example.com", "pass")
	order := domain.Order{UserID: user.ID, OrderNo: "O-LEASE-FENCE", Status: domain.OrderStatusProvisioning, TotalAmount: 100, Currency: "CNY"}
	if err := r.CreateOrder(ctx, &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	job := domain.ProvisionJob{OrderID: order.ID, OrderItemID: 3, HostID: 1, HostName: "h", Status: "pending", NextRunAt: time.Now().Add(-time.Minute)}
	if err := r.CreateOrUpdateProvisionJob(ctx, &job); err != nil {
		t.Fatalf("create job: %v", err)
	}
	a, claimed, err := r.ClaimProvisionJob(ctx, job.ID, "worker-A", time.Now().Add(-time.Second))
	if err != nil || !claimed {
		t.Fatalf("worker A claim: claimed=%v err=%v", claimed, err)
	}
	// B takes over (lease expired).
	if _, claimed, err := r.ClaimProvisionJob(ctx, job.ID, "worker-B", time.Now().Add(domain.ProvisionLeaseDuration)); err != nil || !claimed {
		t.Fatalf("worker B takeover: claimed=%v err=%v", claimed, err)
	}

	// A's fenced write with its (now stale) owner+fence must be rejected.
	a.Status = "done"
	ok, err := r.UpdateProvisionJob(ctx, a)
	if err != nil {
		t.Fatalf("worker A fenced update: %v", err)
	}
	if ok {
		t.Fatalf("stale worker A overwrote the takeover worker's job")
	}

	// B's fenced write with the new fence still succeeds.
	b, _ := r.GetProvisionJob(ctx, job.ID)
	b.Status = "done"
	ok, err = r.UpdateProvisionJob(ctx, b)
	if err != nil || !ok {
		t.Fatalf("worker B fenced update: ok=%v err=%v", ok, err)
	}
}

func TestSQLiteRepo_ProvisionLease_RenewalOnlyForHolder(t *testing.T) {
	_, r := newTestRepo(t)
	ctx := context.Background()
	user := testutil.CreateUser(t, r, "lease-renew", "lease-renew@example.com", "pass")
	order := domain.Order{UserID: user.ID, OrderNo: "O-LEASE-RENEW", Status: domain.OrderStatusProvisioning, TotalAmount: 100, Currency: "CNY"}
	if err := r.CreateOrder(ctx, &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	job := domain.ProvisionJob{OrderID: order.ID, OrderItemID: 4, HostID: 1, HostName: "h", Status: "pending", NextRunAt: time.Now().Add(-time.Minute)}
	if err := r.CreateOrUpdateProvisionJob(ctx, &job); err != nil {
		t.Fatalf("create job: %v", err)
	}
	a, claimed, err := r.ClaimProvisionJob(ctx, job.ID, "worker-A", time.Now().Add(time.Minute))
	if err != nil || !claimed {
		t.Fatalf("worker A claim: claimed=%v err=%v", claimed, err)
	}

	// The holder can renew.
	ok, err := r.RenewProvisionJobLease(ctx, job.ID, a.LeaseOwner, a.Fence, time.Now().Add(2*time.Minute))
	if err != nil || !ok {
		t.Fatalf("holder renewal: ok=%v err=%v", ok, err)
	}
	// A stale non-holder cannot renew.
	ok, err = r.RenewProvisionJobLease(ctx, job.ID, "worker-A", a.Fence+1, time.Now().Add(3*time.Minute))
	if err != nil || ok {
		t.Fatalf("stale renewal should be rejected: ok=%v err=%v", ok, err)
	}
}

func TestSQLiteRepo_ResizeLease_FreshNotStolenExpiredTakenOver(t *testing.T) {
	_, r := newTestRepo(t)
	ctx := context.Background()

	pending := domain.ResizeTask{VPSID: 1, OrderID: 1, OrderItemID: 1, Status: domain.ResizeTaskStatusPending}
	if err := r.CreateResizeTask(ctx, &pending); err != nil {
		t.Fatalf("create pending task: %v", err)
	}
	a, claimed, err := r.ClaimResizeTask(ctx, pending.ID, "worker-A", time.Now().Add(domain.ResizeTaskStaleDuration))
	if err != nil || !claimed {
		t.Fatalf("worker A claim: claimed=%v err=%v", claimed, err)
	}
	if a.LeaseOwner != "worker-A" || a.Fence != 1 {
		t.Fatalf("claim did not assign resize lease/fence: %+v", a)
	}
	// Fresh lease: worker B cannot take over.
	if _, claimed, err := r.ClaimResizeTask(ctx, pending.ID, "worker-B", time.Now().Add(domain.ResizeTaskStaleDuration)); err != nil || claimed {
		t.Fatalf("fresh resize lease stolen: claimed=%v err=%v", claimed, err)
	}
	// Expired lease: worker B takes over, fence bumps.
	expired := domain.ResizeTask{VPSID: 2, OrderID: 2, OrderItemID: 2, Status: domain.ResizeTaskStatusRunning}
	lu := time.Now().Add(-time.Second)
	expired.LeaseUntil = &lu
	expired.LeaseOwner = "worker-A"
	expired.Fence = 3
	if err := r.CreateResizeTask(ctx, &expired); err != nil {
		t.Fatalf("create running task: %v", err)
	}
	b, claimed, err := r.ClaimResizeTask(ctx, expired.ID, "worker-B", time.Now().Add(domain.ResizeTaskStaleDuration))
	if err != nil || !claimed {
		t.Fatalf("worker B takeover: claimed=%v err=%v", claimed, err)
	}
	if b.Fence != 4 || b.LeaseOwner != "worker-B" {
		t.Fatalf("takeover did not reassign resize lease/fence: %+v", b)
	}
}
