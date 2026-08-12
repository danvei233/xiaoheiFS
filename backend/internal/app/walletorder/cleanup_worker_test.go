package walletorder_test

import (
	"context"
	"strings"
	"testing"
	"time"

	appwalletorder "xiaoheiplay/internal/app/walletorder"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

// TestWalletOrderService_CleanupWorkerDeletesHostAndMarksDone covers the happy
// path: a pending cleanup job is claimed and the host is deleted idempotently,
// the local instance is marked unknown, and the job is retired.
func TestWalletOrderService_CleanupWorkerDeletesHostAndMarksDone(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	user := testutil.CreateUser(t, repo, "cleanup-ok", "cleanup-ok@example.com", "pass")
	inst := domain.VPSInstance{UserID: user.ID, AutomationInstanceID: "900", GoodsTypeID: 1, Name: "v", Status: domain.VPSStatusRunning, AdminStatus: domain.VPSAdminStatusNormal, SpecJSON: "{}"}
	if err := repo.CreateInstance(ctx, &inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}

	fakeAuto := &testutil.FakeAutomationClient{}
	svc := appwalletorder.NewService(repo, repo, nil, repo, repo, &testutil.FakeAutomationResolver{Client: fakeAuto}, repo)
	svc.SetRefundCleanupRepo(repo)

	job := domain.RefundCleanupJob{WalletOrderID: 1, UserID: user.ID, VPSID: inst.ID, HostID: 900, GoodsTypeID: 1, Status: domain.RefundCleanupPending, NextRunAt: time.Now().Add(-time.Minute)}
	if err := repo.CreateRefundCleanupJob(ctx, &job); err != nil {
		t.Fatalf("create cleanup job: %v", err)
	}

	if err := svc.ProcessCleanupJobs(ctx, 10); err != nil {
		t.Fatalf("process cleanup jobs: %v", err)
	}
	stored, err := repo.GetRefundCleanupJobByWalletOrder(ctx, 1)
	if err != nil {
		t.Fatalf("get cleanup job: %v", err)
	}
	if stored.Status != domain.RefundCleanupDone {
		t.Fatalf("expected cleanup job done, got %s", stored.Status)
	}
	if len(fakeAuto.DeleteCalls) != 1 || fakeAuto.DeleteCalls[0] != 900 {
		t.Fatalf("expected DeleteHost(900) once, got %v", fakeAuto.DeleteCalls)
	}
	updated, err := repo.GetInstance(ctx, inst.ID)
	if err != nil {
		t.Fatalf("get instance: %v", err)
	}
	if updated.Status != domain.VPSStatusUnknown {
		t.Fatalf("expected instance marked unknown after cleanup, got %s", updated.Status)
	}
}

// TestWalletOrderService_CleanupWorkerRetriesWithBackoffAndStableError covers
// the failure path: a provider delete failure retries with bounded backoff and
// persists only the stable error code / sanitized summary.
func TestWalletOrderService_CleanupWorkerRetriesWithBackoffAndStableError(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	user := testutil.CreateUser(t, repo, "cleanup-retry", "cleanup-retry@example.com", "pass")
	inst := domain.VPSInstance{UserID: user.ID, AutomationInstanceID: "901", GoodsTypeID: 1, Name: "v", Status: domain.VPSStatusRunning, AdminStatus: domain.VPSAdminStatusNormal, SpecJSON: "{}"}
	if err := repo.CreateInstance(ctx, &inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}
	secret := "sk-live-deadbeef0123456789abcdef"
	internalIP := "10.0.0.99"
	fakeAuto := &testutil.FakeAutomationClient{DeleteErr: &providerDeleteError{msg: "delete failed: dial tcp " + internalIP + ":443 token=" + secret}}
	svc := appwalletorder.NewService(repo, repo, nil, repo, repo, &testutil.FakeAutomationResolver{Client: fakeAuto}, repo)
	svc.SetRefundCleanupRepo(repo)

	job := domain.RefundCleanupJob{WalletOrderID: 2, UserID: user.ID, VPSID: inst.ID, HostID: 901, GoodsTypeID: 1, Status: domain.RefundCleanupPending, NextRunAt: time.Now().Add(-time.Minute)}
	if err := repo.CreateRefundCleanupJob(ctx, &job); err != nil {
		t.Fatalf("create cleanup job: %v", err)
	}

	if err := svc.ProcessCleanupJobs(ctx, 10); err == nil {
		t.Fatalf("expected the worker to surface the delete failure")
	}
	stored, err := repo.GetRefundCleanupJobByWalletOrder(ctx, 2)
	if err != nil {
		t.Fatalf("get cleanup job: %v", err)
	}
	if stored.Status != domain.RefundCleanupRetry {
		t.Fatalf("expected cleanup job retry, got %s", stored.Status)
	}
	if stored.Attempts != 1 || stored.LastErrorCode == "" {
		t.Fatalf("expected attempts/code persisted: %+v", stored)
	}
	if containsAny(stored.LastError, secret, internalIP) {
		t.Fatalf("secret leaked into cleanup job meta: %s", stored.LastError)
	}
	if !stored.NextRunAt.After(time.Now()) {
		t.Fatalf("expected next_run_at scheduled in the future (backoff)")
	}
}

// TestWalletOrderService_CleanupWorkerHostNotFoundIsIdempotentSuccess covers the
// crash-window reconcile: a host that is already gone is a success, so a job
// retried after an external-delete/local-update split converges.
func TestWalletOrderService_CleanupWorkerHostNotFoundIsIdempotentSuccess(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	user := testutil.CreateUser(t, repo, "cleanup-gone", "cleanup-gone@example.com", "pass")
	inst := domain.VPSInstance{UserID: user.ID, AutomationInstanceID: "902", GoodsTypeID: 1, Name: "v", Status: domain.VPSStatusRunning, AdminStatus: domain.VPSAdminStatusNormal, SpecJSON: "{}"}
	if err := repo.CreateInstance(ctx, &inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}
	fakeAuto := &testutil.FakeAutomationClient{DeleteErr: &providerDeleteError{msg: "host 902 not found"}}
	svc := appwalletorder.NewService(repo, repo, nil, repo, repo, &testutil.FakeAutomationResolver{Client: fakeAuto}, repo)
	svc.SetRefundCleanupRepo(repo)

	job := domain.RefundCleanupJob{WalletOrderID: 3, UserID: user.ID, VPSID: inst.ID, HostID: 902, GoodsTypeID: 1, Status: domain.RefundCleanupRetry, Attempts: 1, NextRunAt: time.Now().Add(-time.Minute)}
	if err := repo.CreateRefundCleanupJob(ctx, &job); err != nil {
		t.Fatalf("create cleanup job: %v", err)
	}

	if err := svc.ProcessCleanupJobs(ctx, 10); err != nil {
		t.Fatalf("process cleanup jobs: %v", err)
	}
	stored, err := repo.GetRefundCleanupJobByWalletOrder(ctx, 3)
	if err != nil {
		t.Fatalf("get cleanup job: %v", err)
	}
	if stored.Status != domain.RefundCleanupDone {
		t.Fatalf("expected host-not-found to be idempotent success, got %s", stored.Status)
	}
	updated, _ := repo.GetInstance(ctx, inst.ID)
	if updated.Status != domain.VPSStatusUnknown {
		t.Fatalf("expected local instance reconciled to unknown, got %s", updated.Status)
	}
}

// TestWalletOrderService_CleanupWorkerFreshLeaseNotStolen covers the
// multi-instance CAS: a job claimed with a fresh lease by worker A is not
// stealable by worker B, and B cannot fence-write over A's result.
func TestWalletOrderService_CleanupWorkerFreshLeaseNotStolen(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	user := testutil.CreateUser(t, repo, "cleanup-lease", "cleanup-lease@example.com", "pass")
	inst := domain.VPSInstance{UserID: user.ID, AutomationInstanceID: "903", GoodsTypeID: 1, Name: "v", Status: domain.VPSStatusRunning, AdminStatus: domain.VPSAdminStatusNormal, SpecJSON: "{}"}
	if err := repo.CreateInstance(ctx, &inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}
	job := domain.RefundCleanupJob{WalletOrderID: 4, UserID: user.ID, VPSID: inst.ID, HostID: 903, GoodsTypeID: 1, Status: domain.RefundCleanupPending, NextRunAt: time.Now().Add(-time.Minute)}
	if err := repo.CreateRefundCleanupJob(ctx, &job); err != nil {
		t.Fatalf("create cleanup job: %v", err)
	}
	// Worker A claims and holds a fresh lease.
	a, claimed, err := repo.ClaimRefundCleanupJob(ctx, job.ID, "worker-A", time.Now().Add(time.Hour))
	if err != nil || !claimed {
		t.Fatalf("worker A claim: claimed=%v err=%v", claimed, err)
	}
	// Worker B cannot take over a fresh lease.
	if _, claimed, err := repo.ClaimRefundCleanupJob(ctx, job.ID, "worker-B", time.Now().Add(time.Hour)); err != nil || claimed {
		t.Fatalf("fresh cleanup lease stolen: claimed=%v err=%v", claimed, err)
	}
	// Expire A's lease (A is the current holder, so its fenced write passes).
	expired := a
	past := time.Now().Add(-time.Second)
	expired.LeaseUntil = &past
	if ok, err := repo.UpdateRefundCleanupJob(ctx, expired); err != nil || !ok {
		t.Fatalf("expire A lease: ok=%v err=%v", ok, err)
	}
	// After A's lease expires, B takes over and bumps the fence.
	if _, claimed, err := repo.ClaimRefundCleanupJob(ctx, job.ID, "worker-B", time.Now().Add(time.Hour)); err != nil || !claimed {
		t.Fatalf("worker B takeover after expiry: claimed=%v err=%v", claimed, err)
	}
	// A's original fenced write (owner=A, fence from A's claim) must now fail
	// because the fence was bumped by B's takeover.
	a.Status = domain.RefundCleanupDone
	ok, err := repo.UpdateRefundCleanupJob(ctx, a)
	if err != nil || ok {
		t.Fatalf("stale worker A write should be fenced off after takeover: ok=%v err=%v", ok, err)
	}
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
