package vps_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"xiaoheiplay/internal/adapter/repo/core"
	appshared "xiaoheiplay/internal/app/shared"
	appvps "xiaoheiplay/internal/app/vps"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func TestVPSService_GetForbidden(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "v1", "v1@example.com", "pass")
	other := testutil.CreateUser(t, repo, "v2", "v2@example.com", "pass")

	order := domain.Order{UserID: user.ID, OrderNo: "ORD-VPS-1", Status: domain.OrderStatusApproved, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	item := domain.OrderItem{OrderID: order.ID, Amount: 1000, Status: domain.OrderItemStatusApproved, Action: "create", SpecJSON: "{}"}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{item}); err != nil {
		t.Fatalf("create item: %v", err)
	}
	items, _ := repo.ListOrderItems(context.Background(), order.ID)

	inst := domain.VPSInstance{
		UserID:               user.ID,
		OrderItemID:          items[0].ID,
		AutomationInstanceID: "1",
		Name:                 "vm",
		SystemID:             1,
		Status:               domain.VPSStatusUnknown,
		SpecJSON:             "{}",
	}
	if err := repo.CreateInstance(context.Background(), &inst); err != nil {
		t.Fatalf("create vps: %v", err)
	}
	autoResolver := &testutil.FakeAutomationResolver{Client: &testutil.FakeAutomationClient{}}
	svc := appvps.NewService(repo, autoResolver, repo)
	if _, err := svc.Get(context.Background(), inst.ID, other.ID); err != appshared.ErrForbidden {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestVPSService_RefreshStatus(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "v3", "v3@example.com", "pass")

	order := domain.Order{UserID: user.ID, OrderNo: "ORD-VPS-2", Status: domain.OrderStatusApproved, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	item := domain.OrderItem{OrderID: order.ID, Amount: 1000, Status: domain.OrderItemStatusApproved, Action: "create", SpecJSON: "{}"}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{item}); err != nil {
		t.Fatalf("create item: %v", err)
	}
	items, _ := repo.ListOrderItems(context.Background(), order.ID)

	inst := domain.VPSInstance{
		UserID:               user.ID,
		OrderItemID:          items[0].ID,
		AutomationInstanceID: "100",
		Name:                 "vm",
		SystemID:             1,
		Status:               domain.VPSStatusUnknown,
		SpecJSON:             "{}",
		ExpireAt:             ptrTime(time.Now().Add(72 * time.Hour)),
	}
	if err := repo.CreateInstance(context.Background(), &inst); err != nil {
		t.Fatalf("create vps: %v", err)
	}
	expire := time.Now().Add(24 * time.Hour)
	fakeAuto := &testutil.FakeAutomationClient{
		HostInfo: map[int64]appshared.AutomationHostInfo{
			100: {HostID: 100, State: 2, ExpireAt: &expire, RemoteIP: "2.2.2.2"},
		},
	}
	autoResolver := &testutil.FakeAutomationResolver{Client: fakeAuto}
	svc := appvps.NewService(repo, autoResolver, repo)
	if _, err := svc.RefreshStatus(context.Background(), inst); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	updated, err := repo.GetInstance(context.Background(), inst.ID)
	if err != nil {
		t.Fatalf("get updated instance: %v", err)
	}
	if updated.ExpireAt == nil {
		t.Fatalf("expected local expire_at preserved")
	}
	// Refresh should not sync expire_at from automation.
	if !updated.ExpireAt.Equal(*inst.ExpireAt) {
		t.Fatalf("expected expire_at unchanged, got %v want %v", updated.ExpireAt, inst.ExpireAt)
	}
}

func TestVPSService_Actions(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "v4", "v4@example.com", "pass")
	inst := createVPSInstance(t, repo, user.ID, "200")

	fakeAuto := &testutil.FakeAutomationClient{}
	autoResolver := &testutil.FakeAutomationResolver{Client: fakeAuto}
	svc := appvps.NewService(repo, autoResolver, repo)

	if err := svc.Start(context.Background(), inst); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := svc.Shutdown(context.Background(), inst); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	if err := svc.Reboot(context.Background(), inst); err != nil {
		t.Fatalf("reboot: %v", err)
	}
	if _, err := svc.Monitor(context.Background(), inst); err != nil {
		t.Fatalf("monitor: %v", err)
	}
	if url, err := svc.GetPanelURL(context.Background(), inst); err != nil || url == "" {
		t.Fatalf("panel url: %v %v", url, err)
	}
	if url, err := svc.VNCURL(context.Background(), inst); err != nil || url == "" {
		t.Fatalf("vnc url: %v %v", url, err)
	}
}

func TestVPSService_RenewAndEmergency(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "v5", "v5@example.com", "pass")
	inst := createVPSInstance(t, repo, user.ID, "300")
	fakeAuto := &testutil.FakeAutomationClient{}
	autoResolver := &testutil.FakeAutomationResolver{Client: fakeAuto}
	svc := appvps.NewService(repo, autoResolver, repo)

	if err := svc.RenewNow(context.Background(), inst, 1); err != nil {
		t.Fatalf("renew now: %v", err)
	}
	if _, err := svc.EmergencyRenew(context.Background(), inst); err != nil {
		t.Fatalf("emergency renew: %v", err)
	}
	updated, err := repo.GetInstance(context.Background(), inst.ID)
	if err != nil {
		t.Fatalf("get instance: %v", err)
	}
	if updated.LastEmergencyRenewAt == nil {
		t.Fatalf("expected emergency renew timestamp")
	}
}

func TestVPSService_AutoLockExpired_RelocksReExpiredInstance(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	fakeAuto := &testutil.FakeAutomationClient{}
	svc := appvps.NewService(repo, &testutil.FakeAutomationResolver{Client: fakeAuto}, repo)

	past := time.Now().Add(-48 * time.Hour)
	inst := domain.VPSInstance{
		UserID:               1,
		AutomationInstanceID: "900",
		Name:                 "vm",
		Status:               domain.VPSStatusRunning,
		ExpireAt:             ptrTime(past),
		SpecJSON:             "{}",
	}
	if err := repo.CreateInstance(ctx, &inst); err != nil {
		t.Fatalf("create vps: %v", err)
	}

	// First expiry: the host is locked and marked locally.
	if err := svc.AutoLockExpired(ctx); err != nil {
		t.Fatalf("auto lock: %v", err)
	}
	if len(fakeAuto.LockCalls) != 1 {
		t.Fatalf("expected 1 lock call after first expiry, got %d", len(fakeAuto.LockCalls))
	}
	updated, _ := repo.GetInstance(ctx, inst.ID)
	if updated.AdminStatus != domain.VPSAdminStatusLocked {
		t.Fatalf("expected admin_status locked, got %q", updated.AdminStatus)
	}

	// Simulate a renewal that only extends the remote expiry and unlocks the
	// host, leaving the local admin_status/status markers stale (as the order
	// renew path does today).
	future := time.Now().Add(24 * time.Hour)
	if err := repo.UpdateInstanceExpireAt(ctx, inst.ID, future); err != nil {
		t.Fatalf("renew expire: %v", err)
	}
	_ = fakeAuto.UnlockHost(ctx, 900)
	// The renewed period elapses and the instance is again expired.
	if err := repo.UpdateInstanceExpireAt(ctx, inst.ID, past); err != nil {
		t.Fatalf("re-expire: %v", err)
	}

	// Second expiry: the host must be re-locked, not permanently skipped.
	if err := svc.AutoLockExpired(ctx); err != nil {
		t.Fatalf("auto lock second: %v", err)
	}
	if len(fakeAuto.LockCalls) != 2 {
		t.Fatalf("expected host to be re-locked on second expiry, lock calls = %d", len(fakeAuto.LockCalls))
	}
}

func TestVPSService_RenewNowResetsAdminStatus(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	past := time.Now().Add(-48 * time.Hour)
	inst := domain.VPSInstance{
		UserID:               1,
		AutomationInstanceID: "901",
		Name:                 "vm",
		Status:               domain.VPSStatusExpiredLocked,
		AdminStatus:          domain.VPSAdminStatusLocked,
		ExpireAt:             ptrTime(past),
		SpecJSON:             "{}",
	}
	if err := repo.CreateInstance(ctx, &inst); err != nil {
		t.Fatalf("create vps: %v", err)
	}
	fakeAuto := &testutil.FakeAutomationClient{}
	svc := appvps.NewService(repo, &testutil.FakeAutomationResolver{Client: fakeAuto}, repo)

	if err := svc.RenewNow(ctx, inst, 30); err != nil {
		t.Fatalf("renew now: %v", err)
	}
	updated, err := repo.GetInstance(ctx, inst.ID)
	if err != nil {
		t.Fatalf("get instance: %v", err)
	}
	if updated.AdminStatus != domain.VPSAdminStatusNormal {
		t.Fatalf("expected admin_status reset to normal after renew, got %q", updated.AdminStatus)
	}
	if updated.Status == domain.VPSStatusExpiredLocked {
		t.Fatalf("expected expired_locked status cleared after renew, got %q", updated.Status)
	}
}

func TestVPSService_EmergencyRenewResetsAdminStatus(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	future := time.Now().Add(24 * time.Hour)
	inst := domain.VPSInstance{
		UserID:               1,
		AutomationInstanceID: "902",
		Name:                 "vm",
		Status:               domain.VPSStatusRunning,
		AdminStatus:          domain.VPSAdminStatusLocked,
		ExpireAt:             ptrTime(future),
		SpecJSON:             "{}",
	}
	if err := repo.CreateInstance(ctx, &inst); err != nil {
		t.Fatalf("create vps: %v", err)
	}
	fakeAuto := &testutil.FakeAutomationClient{}
	svc := appvps.NewService(repo, &testutil.FakeAutomationResolver{Client: fakeAuto}, repo)

	if _, err := svc.EmergencyRenew(ctx, inst); err != nil {
		t.Fatalf("emergency renew: %v", err)
	}
	updated, err := repo.GetInstance(ctx, inst.ID)
	if err != nil {
		t.Fatalf("get instance: %v", err)
	}
	if updated.AdminStatus != domain.VPSAdminStatusNormal {
		t.Fatalf("expected admin_status reset to normal after emergency renew, got %q", updated.AdminStatus)
	}
}

func TestVPSService_AutoDeleteExpired_SkipsAbuseFraud(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	if err := repo.UpsertSetting(ctx, domain.Setting{Key: "auto_delete_enabled", ValueJSON: "true"}); err != nil {
		t.Fatalf("enable auto delete: %v", err)
	}
	if err := repo.UpsertSetting(ctx, domain.Setting{Key: "auto_delete_days", ValueJSON: "0"}); err != nil {
		t.Fatalf("set auto delete days: %v", err)
	}
	past := time.Now().Add(-72 * time.Hour)
	mk := func(automationID string, adminStatus domain.VPSAdminStatus) domain.VPSInstance {
		inst := domain.VPSInstance{
			UserID:               1,
			AutomationInstanceID: automationID,
			Name:                 "vm",
			Status:               domain.VPSStatusExpiredLocked,
			AdminStatus:          adminStatus,
			ExpireAt:             ptrTime(past),
			SpecJSON:             "{}",
		}
		if err := repo.CreateInstance(ctx, &inst); err != nil {
			t.Fatalf("create vps: %v", err)
		}
		return inst
	}
	normal := mk("910", domain.VPSAdminStatusNormal)
	abuse := mk("911", domain.VPSAdminStatusAbuse)
	fraud := mk("912", domain.VPSAdminStatusFraud)

	fakeAuto := &testutil.FakeAutomationClient{}
	svc := appvps.NewService(repo, &testutil.FakeAutomationResolver{Client: fakeAuto}, repo)

	if err := svc.AutoDeleteExpired(ctx); err != nil {
		t.Fatalf("auto delete: %v", err)
	}
	// abuse/fraud instances must be preserved locally.
	for _, id := range []int64{abuse.ID, fraud.ID} {
		if _, err := repo.GetInstance(ctx, id); err != nil {
			t.Fatalf("abuse/fraud instance %d was deleted: %v", id, err)
		}
	}
	// and their remote hosts must not be destroyed.
	for _, hostID := range []int64{911, 912} {
		for _, c := range fakeAuto.DeleteCalls {
			if c == hostID {
				t.Fatalf("host %d of an abuse/fraud instance was destroyed", hostID)
			}
		}
	}
	// the normal expired instance is deleted.
	if _, err := repo.GetInstance(ctx, normal.ID); err == nil {
		t.Fatalf("expected normal expired instance %d to be deleted", normal.ID)
	}
}

func createVPSInstance(t *testing.T, repo *repo.GormRepo, userID int64, automationID string) domain.VPSInstance {
	t.Helper()
	order := domain.Order{UserID: userID, OrderNo: "ORD-VPS-X", Status: domain.OrderStatusApproved, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	item := domain.OrderItem{OrderID: order.ID, Amount: 1000, Status: domain.OrderItemStatusApproved, Action: "create", SpecJSON: "{}"}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{item}); err != nil {
		t.Fatalf("create item: %v", err)
	}
	items, _ := repo.ListOrderItems(context.Background(), order.ID)
	inst := domain.VPSInstance{
		UserID:               userID,
		OrderItemID:          items[0].ID,
		AutomationInstanceID: automationID,
		Name:                 "vm",
		SystemID:             1,
		Status:               domain.VPSStatusUnknown,
		SpecJSON:             "{}",
		ExpireAt:             ptrTime(time.Now().Add(24 * time.Hour)),
	}
	if err := repo.CreateInstance(context.Background(), &inst); err != nil {
		t.Fatalf("create vps: %v", err)
	}
	return inst
}

func ptrTime(v time.Time) *time.Time { return &v }

// TestVPSService_AutoDeleteExpired_SkipsInFlightRenewal locks the P1-8 fix: an
// expired instance with an approved-but-not-yet-executed renewal order must not
// be auto-deleted (the renewal would extend it moments later). Without the
// guard, AutoDeleteExpired racing handleRenew destroyed freshly renewed hosts.
func TestVPSService_AutoDeleteExpired_SkipsInFlightRenewal(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	if err := repo.UpsertSetting(ctx, domain.Setting{Key: "auto_delete_enabled", ValueJSON: "true"}); err != nil {
		t.Fatalf("enable auto delete: %v", err)
	}
	if err := repo.UpsertSetting(ctx, domain.Setting{Key: "auto_delete_days", ValueJSON: "0"}); err != nil {
		t.Fatalf("set auto delete days: %v", err)
	}
	user := testutil.CreateUser(t, repo, "renewguard", "renewguard@example.com", "pass")

	past := time.Now().Add(-72 * time.Hour)
	inst := domain.VPSInstance{
		UserID:               user.ID,
		AutomationInstanceID: "920",
		Name:                 "vm-renew-guard",
		Status:               domain.VPSStatusExpiredLocked,
		AdminStatus:          domain.VPSAdminStatusNormal,
		ExpireAt:             ptrTime(past),
		SpecJSON:             "{}",
	}
	if err := repo.CreateInstance(ctx, &inst); err != nil {
		t.Fatalf("create vps: %v", err)
	}

	// An approved renewal order for this VPS that has not executed yet.
	renewOrder := domain.Order{UserID: user.ID, OrderNo: "REN-GUARD", Status: domain.OrderStatusApproved, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &renewOrder); err != nil {
		t.Fatalf("create renewal order: %v", err)
	}
	renewItem := domain.OrderItem{OrderID: renewOrder.ID, Amount: 1000, Status: domain.OrderItemStatusApproved, Action: "renew", SpecJSON: `{"vps_id":` + testutil.Itoa(inst.ID) + `}`}
	if err := repo.CreateOrderItems(ctx, []domain.OrderItem{renewItem}); err != nil {
		t.Fatalf("create renewal item: %v", err)
	}

	fakeAuto := &testutil.FakeAutomationClient{}
	svc := appvps.NewService(repo, &testutil.FakeAutomationResolver{Client: fakeAuto}, repo)
	svc.SetOrderItemRepository(repo)

	if err := svc.AutoDeleteExpired(ctx); err != nil {
		t.Fatalf("auto delete: %v", err)
	}
	if _, err := repo.GetInstance(ctx, inst.ID); err != nil {
		t.Fatalf("instance with in-flight renewal was auto-deleted: %v", err)
	}
	for _, c := range fakeAuto.DeleteCalls {
		if c == 920 {
			t.Fatalf("host of instance with in-flight renewal was destroyed")
		}
	}
}

// TestVPSService_AutoDeleteExpired_HostNotFoundConverges locks the crash-window
// convergence for the expiry sweep: when the remote host is already destroyed,
// DeleteHost returns "instance does not exist"; that must be treated as an
// idempotent success so the local expired row is cleaned up instead of being
// stranded forever by a provider error on every retry.
func TestVPSService_AutoDeleteExpired_HostNotFoundConverges(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	if err := repo.UpsertSetting(ctx, domain.Setting{Key: "auto_delete_enabled", ValueJSON: "true"}); err != nil {
		t.Fatalf("enable auto delete: %v", err)
	}
	if err := repo.UpsertSetting(ctx, domain.Setting{Key: "auto_delete_days", ValueJSON: "0"}); err != nil {
		t.Fatalf("set auto delete days: %v", err)
	}
	past := time.Now().Add(-72 * time.Hour)
	inst := domain.VPSInstance{
		UserID:               1,
		AutomationInstanceID: "930",
		Name:                 "vm-expired-gone",
		Status:               domain.VPSStatusExpiredLocked,
		AdminStatus:          domain.VPSAdminStatusNormal,
		ExpireAt:             ptrTime(past),
		SpecJSON:             "{}",
	}
	if err := repo.CreateInstance(ctx, &inst); err != nil {
		t.Fatalf("create vps: %v", err)
	}

	fakeAuto := &testutil.FakeAutomationClient{
		DeleteErr: fmt.Errorf("automation: instance %s does not exist", "930"),
	}
	svc := appvps.NewService(repo, &testutil.FakeAutomationResolver{Client: fakeAuto}, repo)

	if err := svc.AutoDeleteExpired(ctx); err != nil {
		t.Fatalf("auto delete: %v", err)
	}
	if _, err := repo.GetInstance(ctx, inst.ID); err == nil {
		t.Fatalf("expired instance must be removed when the remote host is already gone")
	}
}
