package usecase_test

import (
	"context"
	"testing"
	"time"

	"xiaoheiplay/internal/adapter/repo"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/usecase"
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
	svc := usecase.NewVPSService(repo, autoResolver, repo)
	if _, err := svc.Get(context.Background(), inst.ID, other.ID); err != usecase.ErrForbidden {
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
		HostInfo: map[int64]usecase.AutomationHostInfo{
			100: {HostID: 100, State: 2, ExpireAt: &expire, RemoteIP: "2.2.2.2"},
		},
	}
	autoResolver := &testutil.FakeAutomationResolver{Client: fakeAuto}
	svc := usecase.NewVPSService(repo, autoResolver, repo)
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
	svc := usecase.NewVPSService(repo, autoResolver, repo)

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
	svc := usecase.NewVPSService(repo, autoResolver, repo)

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
