package order_test

import (
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	repo "xiaoheiplay/internal/adapter/repo/core"
	apporder "xiaoheiplay/internal/app/order"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

type failOnceVPSRepo struct {
	*repo.GormRepo
	failed atomic.Bool
}

type delayedCreateVisibilityAutomation struct {
	*testutil.FakeAutomationClient
	createCalls atomic.Int32
	listCalls   atomic.Int32
	hostID      int64
	mu          sync.Mutex
	hostName    string
}

type definitiveFirstFailureAutomation struct {
	*testutil.FakeAutomationClient
	createCalls atomic.Int32
	hostID      int64
}

func (f *definitiveFirstFailureAutomation) CreateHost(context.Context, appshared.AutomationCreateHostRequest) (appshared.AutomationCreateHostResult, error) {
	if f.createCalls.Add(1) == 1 {
		return appshared.AutomationCreateHostResult{}, &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}
	}
	return appshared.AutomationCreateHostResult{HostID: f.hostID}, nil
}

func (f *delayedCreateVisibilityAutomation) CreateHost(_ context.Context, req appshared.AutomationCreateHostRequest) (appshared.AutomationCreateHostResult, error) {
	f.createCalls.Add(1)
	f.mu.Lock()
	f.hostName = req.HostName
	f.mu.Unlock()
	return appshared.AutomationCreateHostResult{}, context.DeadlineExceeded
}

func (f *delayedCreateVisibilityAutomation) ListHostSimple(context.Context, string) ([]appshared.AutomationHostSimple, error) {
	if f.listCalls.Add(1) == 1 {
		return nil, nil
	}
	f.mu.Lock()
	hostName := f.hostName
	f.mu.Unlock()
	return []appshared.AutomationHostSimple{{ID: f.hostID, HostName: hostName}}, nil
}

func (r *failOnceVPSRepo) CreateInstance(ctx context.Context, instance *domain.VPSInstance) error {
	if r.failed.CompareAndSwap(false, true) {
		return errors.New("injected local persistence failure")
	}
	return r.GormRepo.CreateInstance(ctx, instance)
}

func TestOrderService_ConcurrentApproveProvisionsEachItemOnce(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "prov-cas", "prov-cas@example.com", "pass")
	order := domain.Order{UserID: user.ID, OrderNo: "ORD-PROV-CAS", Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{{
		OrderID: order.ID, PackageID: seed.Package.ID, SystemID: seed.SystemImage.ID, Amount: 1000,
		Status: domain.OrderItemStatusPendingPayment, Action: "create", SpecJSON: "{}",
	}}); err != nil {
		t.Fatalf("create item: %v", err)
	}

	fakeAuto := &testutil.FakeAutomationClient{
		CreateHostResult: appshared.AutomationCreateHostResult{HostID: 1001},
		HostInfo: map[int64]appshared.AutomationHostInfo{
			1001: {HostID: 1001, HostName: "host", State: 2, RemoteIP: "1.1.1.1"},
		},
	}
	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, &testutil.FakeAutomationResolver{Client: fakeAuto}, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)

	start := make(chan struct{})
	var wg sync.WaitGroup
	for adminID := int64(1); adminID <= 2; adminID++ {
		wg.Add(1)
		go func(adminID int64) {
			defer wg.Done()
			<-start
			_ = svc.ApproveOrder(context.Background(), adminID, order.ID)
		}(adminID)
	}
	close(start)
	wg.Wait()

	deadline := time.Now().Add(2 * time.Second)
	completed := false
	for time.Now().Before(deadline) {
		items, err := repo.ListOrderItems(context.Background(), order.ID)
		storedOrder, orderErr := repo.GetOrder(context.Background(), order.ID)
		if err == nil && orderErr == nil && len(items) == 1 &&
			items[0].Status == domain.OrderItemStatusActive && storedOrder.Status == domain.OrderStatusActive {
			completed = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !completed {
		storedOrder, _ := repo.GetOrder(context.Background(), order.ID)
		items, _ := repo.ListOrderItems(context.Background(), order.ID)
		t.Fatalf("order did not settle active: order=%s items=%+v", storedOrder.Status, items)
	}
	if got := fakeAuto.CreateHostRequestCount(); got != 1 {
		t.Fatalf("expected one CreateHost call, got %d", got)
	}
}

func TestOrderService_RetryAfterLocalFailureReusesCreatedHost(t *testing.T) {
	_, baseRepo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, baseRepo)
	user := testutil.CreateUser(t, baseRepo, "prov-recover", "prov-recover@example.com", "pass")
	order := domain.Order{UserID: user.ID, OrderNo: "ORD-PROV-RECOVER", Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "CNY"}
	if err := baseRepo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	if err := baseRepo.CreateOrderItems(context.Background(), []domain.OrderItem{{
		OrderID: order.ID, PackageID: seed.Package.ID, SystemID: seed.SystemImage.ID, Amount: 1000,
		Status: domain.OrderItemStatusPendingPayment, Action: "create", SpecJSON: "{}",
	}}); err != nil {
		t.Fatalf("create item: %v", err)
	}

	fakeAuto := &testutil.FakeAutomationClient{
		CreateHostResult: appshared.AutomationCreateHostResult{HostID: 2001},
		HostInfo: map[int64]appshared.AutomationHostInfo{
			2001: {HostID: 2001, HostName: "recovered-host", State: 2, RemoteIP: "1.1.1.2"},
		},
	}
	flakyVPS := &failOnceVPSRepo{GormRepo: baseRepo}
	svc := apporder.NewService(baseRepo, baseRepo, baseRepo, baseRepo, baseRepo, baseRepo, flakyVPS, baseRepo, baseRepo, nil, &testutil.FakeAutomationResolver{Client: fakeAuto}, nil, baseRepo, baseRepo, nil, baseRepo, baseRepo, baseRepo, nil, nil, nil)
	if err := svc.ApproveOrder(context.Background(), 1, order.ID); err != nil {
		t.Fatalf("approve order: %v", err)
	}
	waitForProvisionOrderStatus(t, baseRepo, order.ID, domain.OrderStatusFailed)

	if err := svc.RetryProvision(order.ID); err != nil {
		t.Fatalf("retry provision: %v", err)
	}
	waitForProvisionOrderStatus(t, baseRepo, order.ID, domain.OrderStatusActive)
	if got := fakeAuto.CreateHostRequestCount(); got != 1 {
		t.Fatalf("retry created a duplicate host: calls=%d", got)
	}
}

func TestOrderService_UncertainRemoteCreateReconcilesWithoutSecondCreate(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "prov-delayed", "prov-delayed@example.com", "pass")
	order := domain.Order{UserID: user.ID, OrderNo: "ORD-PROV-DELAYED", Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{{
		OrderID: order.ID, PackageID: seed.Package.ID, SystemID: seed.SystemImage.ID, Amount: 1000,
		Status: domain.OrderItemStatusPendingPayment, Action: "create", SpecJSON: "{}",
	}}); err != nil {
		t.Fatalf("create item: %v", err)
	}

	const hostID = int64(3001)
	fakeAuto := &delayedCreateVisibilityAutomation{
		FakeAutomationClient: &testutil.FakeAutomationClient{HostInfo: map[int64]appshared.AutomationHostInfo{
			hostID: {HostID: hostID, HostName: "ecs-delayed", State: 2, RemoteIP: "1.1.1.3"},
		}},
		hostID: hostID,
	}
	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, &testutil.FakeAutomationResolver{Client: fakeAuto}, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
	if err := svc.ApproveOrder(context.Background(), 1, order.ID); err != nil {
		t.Fatalf("approve order: %v", err)
	}
	waitForProvisionOrderStatus(t, repo, order.ID, domain.OrderStatusFailed)

	if err := svc.RetryProvision(order.ID); err != nil {
		t.Fatalf("first retry provision: %v", err)
	}
	waitForProvisionOrderStatus(t, repo, order.ID, domain.OrderStatusProvisioning)
	if err := svc.RetryProvision(order.ID); err != nil {
		t.Fatalf("second retry provision: %v", err)
	}
	waitForProvisionOrderStatus(t, repo, order.ID, domain.OrderStatusActive)
	if got := fakeAuto.createCalls.Load(); got != 1 {
		t.Fatalf("uncertain create was repeated: calls=%d", got)
	}
}

func TestOrderService_DefinitiveRemoteFailureClearsMarkerForRetry(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "prov-definitive", "prov-definitive@example.com", "pass")
	order := domain.Order{UserID: user.ID, OrderNo: "ORD-PROV-DEFINITIVE", Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{{
		OrderID: order.ID, PackageID: seed.Package.ID, SystemID: seed.SystemImage.ID, Amount: 1000,
		Status: domain.OrderItemStatusPendingPayment, Action: "create", SpecJSON: "{}",
	}}); err != nil {
		t.Fatalf("create item: %v", err)
	}

	const hostID = int64(4001)
	fakeAuto := &definitiveFirstFailureAutomation{
		FakeAutomationClient: &testutil.FakeAutomationClient{HostInfo: map[int64]appshared.AutomationHostInfo{
			hostID: {HostID: hostID, HostName: "retried-host", State: 2, RemoteIP: "1.1.1.4"},
		}},
		hostID: hostID,
	}
	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, &testutil.FakeAutomationResolver{Client: fakeAuto}, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
	if err := svc.ApproveOrder(context.Background(), 1, order.ID); err != nil {
		t.Fatalf("approve order: %v", err)
	}
	waitForProvisionOrderStatus(t, repo, order.ID, domain.OrderStatusFailed)
	items, err := repo.ListOrderItems(context.Background(), order.ID)
	if err != nil || len(items) != 1 {
		t.Fatalf("list failed item: len=%d err=%v", len(items), err)
	}
	if items[0].AutomationInstanceID != "" {
		t.Fatalf("definitive failure retained uncertain marker: %q", items[0].AutomationInstanceID)
	}

	if err := svc.RetryProvision(order.ID); err != nil {
		t.Fatalf("retry provision: %v", err)
	}
	waitForProvisionOrderStatus(t, repo, order.ID, domain.OrderStatusActive)
	if got := fakeAuto.createCalls.Load(); got != 2 {
		t.Fatalf("definitive failure was not retried exactly once: calls=%d", got)
	}
}

func waitForProvisionOrderStatus(t *testing.T, repo *repo.GormRepo, orderID int64, want domain.OrderStatus) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		order, err := repo.GetOrder(context.Background(), orderID)
		if err == nil && order.Status == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	order, err := repo.GetOrder(context.Background(), orderID)
	t.Fatalf("order status did not become %s: order=%+v err=%v", want, order, err)
}

func TestOrderService_ApproveProvisionCreatesVPS(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "prov", "prov@example.com", "pass")

	order := domain.Order{UserID: user.ID, OrderNo: "ORD-PROV", Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	item := domain.OrderItem{
		OrderID:   order.ID,
		PackageID: seed.Package.ID,
		SystemID:  seed.SystemImage.ID,
		Amount:    1000,
		Status:    domain.OrderItemStatusPendingPayment,
		Action:    "create",
		SpecJSON:  "{}",
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{item}); err != nil {
		t.Fatalf("create item: %v", err)
	}

	fakeAuto := &testutil.FakeAutomationClient{
		CreateHostResult: appshared.AutomationCreateHostResult{HostID: 1001},
		HostInfo: map[int64]appshared.AutomationHostInfo{
			1001: {HostID: 1001, HostName: "host", State: 2, RemoteIP: "1.1.1.1"},
		},
	}
	autoResolver := &testutil.FakeAutomationResolver{Client: fakeAuto}
	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, autoResolver, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)

	if err := svc.ApproveOrder(context.Background(), 1, order.ID); err != nil {
		t.Fatalf("approve order: %v", err)
	}

	var inst domain.VPSInstance
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		itemRow, err := repo.ListOrderItems(context.Background(), order.ID)
		if err == nil && len(itemRow) > 0 {
			inst, err = repo.GetInstanceByOrderItem(context.Background(), itemRow[0].ID)
			if err == nil && inst.ID > 0 {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if inst.ID == 0 {
		t.Fatalf("expected vps instance created")
	}
}

func TestOrderService_ApproveProvisionFailure(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "prov2", "prov2@example.com", "pass")

	order := domain.Order{UserID: user.ID, OrderNo: "ORD-PROV-FAIL", Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	item := domain.OrderItem{
		OrderID:   order.ID,
		PackageID: seed.Package.ID,
		SystemID:  seed.SystemImage.ID,
		Amount:    1000,
		Status:    domain.OrderItemStatusPendingPayment,
		Action:    "create",
		SpecJSON:  "{}",
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{item}); err != nil {
		t.Fatalf("create item: %v", err)
	}

	fakeAuto := &testutil.FakeAutomationClient{
		CreateHostErr: context.DeadlineExceeded,
	}
	autoResolver := &testutil.FakeAutomationResolver{Client: fakeAuto}
	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, autoResolver, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)

	if err := svc.ApproveOrder(context.Background(), 1, order.ID); err != nil {
		t.Fatalf("approve order: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		items, err := repo.ListOrderItems(context.Background(), order.ID)
		if err == nil && len(items) > 0 {
			if items[0].Status == domain.OrderItemStatusFailed {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("expected item failed")
}
