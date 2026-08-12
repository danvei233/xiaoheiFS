package order_test

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	repo "xiaoheiplay/internal/adapter/repo/core"
	apporder "xiaoheiplay/internal/app/order"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/pkg/cryptox"
	"xiaoheiplay/internal/testutil"
)

func testCipher(t *testing.T) *cryptox.AESGCM {
	t.Helper()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	c, err := cryptox.NewAESGCM(base64.RawURLEncoding.EncodeToString(key))
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	return c
}

func newCreateOrder(t *testing.T, repo *repo.GormRepo, username, orderNo string) (domain.Order, domain.OrderItem) {
	t.Helper()
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, username, username+"@example.com", "pass")
	order := domain.Order{UserID: user.ID, OrderNo: orderNo, Status: domain.OrderStatusApproved, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{{
		OrderID: order.ID, PackageID: seed.Package.ID, SystemID: seed.SystemImage.ID, Amount: 1000,
		Status: domain.OrderItemStatusApproved, Action: "create", SpecJSON: "{}",
	}}); err != nil {
		t.Fatalf("create item: %v", err)
	}
	items, _ := repo.ListOrderItems(context.Background(), order.ID)
	return order, items[0]
}

// TestProvisionDurableIntent_PersistsCreds verifies the provision intent writes
// the deterministic hostname and the encrypted system/VNC credentials BEFORE the
// external call, and that a completed provision uses those persisted creds when
// the provider never returns passwords again (the crash-window guarantee).
func TestProvisionDurableIntent_PersistsCreds(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	order, item := newCreateOrder(t, repo, "prov-intent", "ORD-INTENT")
	fakeAuto := &testutil.FakeAutomationClient{
		CreateHostResult: appshared.AutomationCreateHostResult{HostID: 3001},
		// No passwords returned by GetHostInfo: the persisted creds must survive.
		HostInfo: map[int64]appshared.AutomationHostInfo{3001: {HostID: 3001, HostName: "h", State: 2}},
	}
	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, &testutil.FakeAutomationResolver{Client: fakeAuto}, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
	svc.SetProvisionCipher(testCipher(t))
	if err := svc.ApproveOrder(context.Background(), 1, order.ID); err != nil {
		t.Fatalf("approve: %v", err)
	}
	waitForOrder(t, repo, order.ID, domain.OrderStatusActive)

	job, err := repo.GetProvisionJobByOrderItem(context.Background(), item.ID)
	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	if job.CredsEncrypted == "" {
		t.Fatalf("provision intent did not persist encrypted credentials")
	}
	if !strings.HasPrefix(job.HostName, "ecs-") {
		t.Fatalf("deterministic hostname not persisted: %q", job.HostName)
	}

	inst, err := repo.GetInstanceByOrderItem(context.Background(), item.ID)
	if err != nil {
		t.Fatalf("get instance: %v", err)
	}
	if !strings.Contains(inst.AccessInfoJSON, "os_password") || !strings.Contains(inst.AccessInfoJSON, "vnc_password") {
		t.Fatalf("completed provision lost the persisted credentials: %s", inst.AccessInfoJSON)
	}
	if strings.Contains(inst.AccessInfoJSON, "[") {
		t.Fatalf("access info contains raw placeholder: %s", inst.AccessInfoJSON)
	}
}

// TestProvisionDurableIntent_CrashBeforeCreateReconciles models a crash after
// the `creating:` marker / stage write but before CreateHost returned: the next
// worker must reconcile by the deterministic hostname and must NOT issue a
// second CreateHost.
func TestProvisionDurableIntent_CrashBeforeCreateReconciles(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	order, item := newCreateOrder(t, repo, "prov-reconcile", "ORD-RECONCILE")
	// Persist an intent stuck at create_host_sent (the crash window).
	job := domain.ProvisionJob{
		OrderID: order.ID, OrderItemID: item.ID, HostID: 0, HostName: "ecs-reconcile",
		Status: "pending", Attempts: 1, NextRunAt: time.Now().Add(-time.Minute),
		Stage: domain.ProvisionStageCreateHostSent,
	}
	if err := repo.CreateOrUpdateProvisionJob(context.Background(), &job); err != nil {
		t.Fatalf("create job: %v", err)
	}

	fakeAuto := &testutil.FakeAutomationClient{
		ListHostSimpleItems: []appshared.AutomationHostSimple{{ID: 4001, HostName: "ecs-reconcile"}},
		HostInfo:            map[int64]appshared.AutomationHostInfo{4001: {HostID: 4001, HostName: "ecs-reconcile", State: 2}},
	}
	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, &testutil.FakeAutomationResolver{Client: fakeAuto}, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
	if err := svc.ProcessProvisionJobs(context.Background(), 10); err != nil {
		t.Fatalf("process jobs: %v", err)
	}
	waitForOrder(t, repo, order.ID, domain.OrderStatusActive)
	if got := fakeAuto.CreateHostRequestCount(); got != 0 {
		t.Fatalf("reconcile must not issue a second CreateHost, calls=%d", got)
	}
}

// TestProvisionDurableIntent_CrashAfterCreateReusesHost models a crash after
// CreateHost returned but before the local host-id write: the job's host_id is
// persisted, so the next worker reuses it without a second CreateHost.
func TestProvisionDurableIntent_CrashAfterCreateReusesHost(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	order, item := newCreateOrder(t, repo, "prov-reuse", "ORD-REUSE")
	job := domain.ProvisionJob{
		OrderID: order.ID, OrderItemID: item.ID, HostID: 5001, HostName: "ecs-reuse",
		Status: "pending", Attempts: 1, NextRunAt: time.Now().Add(-time.Minute),
		Stage: domain.ProvisionStageHostCreated,
	}
	if err := repo.CreateOrUpdateProvisionJob(context.Background(), &job); err != nil {
		t.Fatalf("create job: %v", err)
	}
	fakeAuto := &testutil.FakeAutomationClient{
		HostInfo: map[int64]appshared.AutomationHostInfo{5001: {HostID: 5001, HostName: "ecs-reuse", State: 2}},
	}
	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, &testutil.FakeAutomationResolver{Client: fakeAuto}, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
	if err := svc.ProcessProvisionJobs(context.Background(), 10); err != nil {
		t.Fatalf("process jobs: %v", err)
	}
	waitForOrder(t, repo, order.ID, domain.OrderStatusActive)
	if got := fakeAuto.CreateHostRequestCount(); got != 0 {
		t.Fatalf("host reuse must not call CreateHost, calls=%d", got)
	}
}
