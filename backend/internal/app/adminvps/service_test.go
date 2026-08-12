package adminvps_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	appadminvps "xiaoheiplay/internal/app/adminvps"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func TestAdminVPSCreate_NoExactHostMatch_RejectsFallback(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "u1", "u1@example.com", "pass")

	// The automation returns hosts, but none matches the requested name.
	fakeAuto := &testutil.FakeAutomationClient{
		ListHostSimpleItems: []appshared.AutomationHostSimple{
			{ID: 777, HostName: "unrelated-host"},
		},
	}
	resolver := &testutil.FakeAutomationResolver{Client: fakeAuto}
	svc := appadminvps.NewService(repo, resolver, repo, nil, repo, nil)

	input := appshared.AdminVPSCreateInput{
		UserID:      user.ID,
		Name:        "target-host",
		GoodsTypeID: 1,
	}
	if _, err := svc.Create(context.Background(), 1, input); err != domain.ErrHostIDNotFound {
		t.Fatalf("expected ErrHostIDNotFound when no host matches the requested name, got %v", err)
	}
}

// TestAdminVPSUpdate_AdminStatusRoutesThroughSetAdminStatus locks the P1-20
// fix: an admin_status change through the generic update endpoint must invoke
// the automation lock/unlock side effect (via SetAdminStatus) regardless of
// sync mode, not just write the local column.
func TestAdminVPSUpdate_AdminStatusRoutesThroughSetAdminStatus(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "updlock", "updlock@example.com", "pass")

	inst := domain.VPSInstance{
		UserID:               user.ID,
		AutomationInstanceID: "9001",
		Name:                 "vm-upd-lock",
		Status:               domain.VPSStatusRunning,
		AdminStatus:          domain.VPSAdminStatusNormal,
		SpecJSON:             "{}",
	}
	if err := repo.CreateInstance(context.Background(), &inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}

	fakeAuto := &testutil.FakeAutomationClient{}
	resolver := &testutil.FakeAutomationResolver{Client: fakeAuto}
	svc := appadminvps.NewService(repo, resolver, nil, repo, nil, nil)

	locked := domain.VPSAdminStatusLocked
	input := appshared.AdminVPSUpdateInput{AdminStatus: &locked, SyncMode: ""}
	if _, err := svc.Update(context.Background(), 1, inst.ID, input); err != nil {
		t.Fatalf("update admin_status: %v", err)
	}
	if len(fakeAuto.LockCalls) == 0 {
		t.Fatalf("expected LockHost to be called for admin_status=locked, got no lock calls")
	}
	updated, err := repo.GetInstance(context.Background(), inst.ID)
	if err != nil {
		t.Fatalf("get instance: %v", err)
	}
	if updated.AdminStatus != domain.VPSAdminStatusLocked {
		t.Fatalf("expected admin_status locked, got %q", updated.AdminStatus)
	}
}

// TestAdminVPSUpdate_RejectsUnknownStatus guards the service against direct
// status writes with an enum the handler validation would have caught.
func TestAdminVPSUpdate_RejectsUnknownStatus(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "updbad", "updbad@example.com", "pass")

	inst := domain.VPSInstance{
		UserID:               user.ID,
		AutomationInstanceID: "9002",
		Name:                 "vm-upd-bad",
		Status:               domain.VPSStatusRunning,
		AdminStatus:          domain.VPSAdminStatusNormal,
		SpecJSON:             "{}",
	}
	if err := repo.CreateInstance(context.Background(), &inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}
	svc := appadminvps.NewService(repo, &testutil.FakeAutomationResolver{Client: &testutil.FakeAutomationClient{}}, nil, repo, nil, nil)

	bad := domain.VPSStatus("hacked")
	input := appshared.AdminVPSUpdateInput{Status: &bad}
	if _, err := svc.Update(context.Background(), 1, inst.ID, input); err != appshared.ErrInvalidInput {
		t.Fatalf("expected ErrInvalidInput for unknown status, got %v", err)
	}
}

// TestAdminVPSDelete_HostNotFoundIsIdempotentSuccess locks the crash-window
// convergence fix: after a crash between DeleteHost and the local DeleteInstance,
// a retry re-runs DeleteHost against a destroyed host. The provider reports
// "instance does not exist" (host-not-found), which must be treated as an
// idempotent success so the local row is removed instead of being stranded.
func TestAdminVPSDelete_HostNotFoundIsIdempotentSuccess(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "delhostnf", "delhostnf@example.com", "pass")
	inst := domain.VPSInstance{
		UserID:               user.ID,
		AutomationInstanceID: "4242",
		Name:                 "vm-host-not-found",
		Status:               domain.VPSStatusRunning,
		AdminStatus:          domain.VPSAdminStatusNormal,
		SpecJSON:             "{}",
	}
	if err := repo.CreateInstance(context.Background(), &inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}

	fakeAuto := &testutil.FakeAutomationClient{
		DeleteErr: fmt.Errorf("automation: instance %d does not exist", 4242),
	}
	resolver := &testutil.FakeAutomationResolver{Client: fakeAuto}
	svc := appadminvps.NewService(repo, resolver, nil, repo, nil, nil)

	if err := svc.Delete(context.Background(), 1, inst.ID); err != nil {
		t.Fatalf("delete with host-not-found must be idempotent success, got %v", err)
	}
	if _, err := repo.GetInstance(context.Background(), inst.ID); err == nil {
		t.Fatalf("local instance row must be removed after host-not-found delete")
	}
	if len(fakeAuto.DeleteCalls) != 1 {
		t.Fatalf("expected exactly one DeleteHost call, got %d", len(fakeAuto.DeleteCalls))
	}
}

// TestAdminVPSUpdateExpireAt_RejectsPastTime locks P3-28: UpdateExpireAt must
// refuse a lease expiry set into the past instead of silently applying it.
func TestAdminVPSUpdateExpireAt_RejectsPastTime(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "expirepast", "expirepast@example.com", "pass")
	inst := domain.VPSInstance{
		UserID:               user.ID,
		AutomationInstanceID: "777",
		Name:                 "vm-expire-past",
		Status:               domain.VPSStatusRunning,
		AdminStatus:          domain.VPSAdminStatusNormal,
		SpecJSON:             "{}",
	}
	if err := repo.CreateInstance(context.Background(), &inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}
	svc := appadminvps.NewService(repo, &testutil.FakeAutomationResolver{Client: &testutil.FakeAutomationClient{}}, nil, repo, nil, nil)

	past := time.Now().Add(-time.Hour)
	if _, err := svc.UpdateExpireAt(context.Background(), 1, inst.ID, past); err != domain.ErrInvalidExpireAt {
		t.Fatalf("expected ErrInvalidExpireAt for a past expiry, got %v", err)
	}
	reloaded, err := repo.GetInstance(context.Background(), inst.ID)
	if err != nil {
		t.Fatalf("reload instance: %v", err)
	}
	if reloaded.ExpireAt != nil {
		t.Fatalf("past expiry must not be persisted, got %v", reloaded.ExpireAt)
	}

	// A future expiry is still accepted.
	future := time.Now().Add(48 * time.Hour)
	if _, err := svc.UpdateExpireAt(context.Background(), 1, inst.ID, future); err != nil {
		t.Fatalf("future expiry should be accepted, got %v", err)
	}
}

// TestAdminVPSSetAdminStatus_LockedPropagatesToUserStatus locks P3-28: locking
// (or abuse/fraud) an instance must surface the locked user-visible status, not
// only the admin_status column.
func TestAdminVPSSetAdminStatus_LockedPropagatesToUserStatus(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "lockprop", "lockprop@example.com", "pass")
	inst := domain.VPSInstance{
		UserID:               user.ID,
		AutomationInstanceID: "888",
		Name:                 "vm-lock-prop",
		Status:               domain.VPSStatusRunning,
		AdminStatus:          domain.VPSAdminStatusNormal,
		SpecJSON:             "{}",
	}
	if err := repo.CreateInstance(context.Background(), &inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}
	svc := appadminvps.NewService(repo, &testutil.FakeAutomationResolver{Client: &testutil.FakeAutomationClient{}}, nil, repo, nil, nil)

	if err := svc.SetAdminStatus(context.Background(), 1, inst.ID, domain.VPSAdminStatusLocked, "test"); err != nil {
		t.Fatalf("set admin status: %v", err)
	}
	reloaded, err := repo.GetInstance(context.Background(), inst.ID)
	if err != nil {
		t.Fatalf("reload instance: %v", err)
	}
	if reloaded.AdminStatus != domain.VPSAdminStatusLocked {
		t.Fatalf("expected admin_status locked, got %q", reloaded.AdminStatus)
	}
	if reloaded.Status != domain.VPSStatusLocked {
		t.Fatalf("expected user-visible status locked after admin lock, got %q", reloaded.Status)
	}
}
