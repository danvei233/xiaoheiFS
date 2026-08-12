package order

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

// TestDeleteVPSForRefund_HostNotFoundIsIdempotentSuccess locks the provision F5
// residual for the order-refund path: after the wallet credit commits, a retry
// of deleteVPSForRefund whose DeleteHost reports the host is already gone must
// be treated as success and still delete the local instance row. Otherwise a
// crash between DeleteHost and DeleteInstance leaves a phantom VPS forever.
func TestDeleteVPSForRefund_HostNotFoundIsIdempotentSuccess(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	user := testutil.CreateUser(t, repo, "refunddel", "refunddel@example.com", "pass")
	seed := testutil.SeedCatalog(t, repo)

	baseOrder := domain.Order{UserID: user.ID, OrderNo: "ORD-REFUND-DEL", Status: domain.OrderStatusActive, TotalAmount: 3000, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &baseOrder); err != nil {
		t.Fatalf("create order: %v", err)
	}
	baseItem := domain.OrderItem{OrderID: baseOrder.ID, PackageID: seed.Package.ID, SystemID: seed.SystemImage.ID, Amount: 3000, Status: domain.OrderItemStatusActive, Action: "create", SpecJSON: "{}"}
	if err := repo.CreateOrderItems(ctx, []domain.OrderItem{baseItem}); err != nil {
		t.Fatalf("create item: %v", err)
	}
	items, err := repo.ListOrderItems(ctx, baseOrder.ID)
	if err != nil || len(items) == 0 {
		t.Fatalf("list items: %v", err)
	}
	expire := time.Now().Add(30 * 24 * time.Hour)
	inst := domain.VPSInstance{
		UserID:               user.ID,
		OrderItemID:          items[0].ID,
		AutomationInstanceID: "9001",
		Name:                 "vm-refund-del",
		PackageID:            seed.Package.ID,
		PackageName:          seed.Package.Name,
		MonthlyPrice:         3000,
		SpecJSON:             "{}",
		Status:               domain.VPSStatusRunning,
		CreatedAt:            time.Now(),
		ExpireAt:             &expire,
	}
	if err := repo.CreateInstance(ctx, &inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}

	automation := &testutil.FakeAutomationClient{DeleteErr: errors.New("host 9001 not found")}
	resolver := &testutil.FakeAutomationResolver{Client: automation}
	svc := NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, resolver, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)

	if err := svc.deleteVPSForRefund(ctx, inst); err != nil {
		t.Fatalf("deleteVPSForRefund with host-not-found must succeed, got %v", err)
	}
	if _, err := repo.GetInstance(ctx, inst.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected local instance deleted, got err=%v", err)
	}
	if len(automation.DeleteCalls) != 1 || automation.DeleteCalls[0] != 9001 {
		t.Fatalf("expected DeleteHost(9001) once, got %v", automation.DeleteCalls)
	}
}

// TestHandleRefund_InlineDeleteFailsEnqueuesCleanup locks the money F4 fix for
// the order path: when the durable cleanup outbox is wired, an inline
// DeleteHost failure after the wallet credit must NOT fail the refund item —
// instead a refund_cleanup_jobs entry is created (or left pending) so the
// cleanup worker retries the deletion.
func TestHandleRefund_InlineDeleteFailsEnqueuesCleanup(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	user := testutil.CreateUser(t, repo, "refundcleanup", "refundcleanup@example.com", "pass")
	if _, err := repo.AdjustWalletBalance(ctx, user.ID, 5000, "credit", "seed", 1, "init"); err != nil {
		t.Fatalf("seed wallet: %v", err)
	}
	seed := testutil.SeedCatalog(t, repo)

	baseOrder := domain.Order{UserID: user.ID, OrderNo: "ORD-REFUND-CLEANUP", Status: domain.OrderStatusActive, TotalAmount: 3000, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &baseOrder); err != nil {
		t.Fatalf("create order: %v", err)
	}
	baseItem := domain.OrderItem{OrderID: baseOrder.ID, PackageID: seed.Package.ID, SystemID: seed.SystemImage.ID, Amount: 3000, Status: domain.OrderItemStatusActive, Action: "create", SpecJSON: "{}"}
	if err := repo.CreateOrderItems(ctx, []domain.OrderItem{baseItem}); err != nil {
		t.Fatalf("create item: %v", err)
	}
	items, err := repo.ListOrderItems(ctx, baseOrder.ID)
	if err != nil || len(items) == 0 {
		t.Fatalf("list items: %v", err)
	}
	expire := time.Now().Add(30 * 24 * time.Hour)
	inst := domain.VPSInstance{
		UserID:               user.ID,
		OrderItemID:          items[0].ID,
		AutomationInstanceID: "9100",
		Name:                 "vm-refund-cleanup",
		PackageID:            seed.Package.ID,
		PackageName:          seed.Package.Name,
		MonthlyPrice:         3000,
		SpecJSON:             "{}",
		Status:               domain.VPSStatusRunning,
		CreatedAt:            time.Now(),
		ExpireAt:             &expire,
	}
	if err := repo.CreateInstance(ctx, &inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}

	// Refund order (approved) with the delete_on_approve refund item.
	refundOrder := domain.Order{UserID: user.ID, OrderNo: "ORD-REF-CLEANUP", Status: domain.OrderStatusApproved, TotalAmount: -3000, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &refundOrder); err != nil {
		t.Fatalf("create refund order: %v", err)
	}
	specJSON, _ := jsonMarshalAny(map[string]any{
		"vps_id": inst.ID, "refund_amount": 3000, "refund_to_wallet": true, "delete_on_approve": true, "reason": "test",
	})
	refundItem := domain.OrderItem{OrderID: refundOrder.ID, Amount: -3000, Status: domain.OrderItemStatusApproved, Action: "refund", SpecJSON: specJSON}
	if err := repo.CreateOrderItems(ctx, []domain.OrderItem{refundItem}); err != nil {
		t.Fatalf("create refund item: %v", err)
	}
	refundItems, err := repo.ListOrderItems(ctx, refundOrder.ID)
	if err != nil || len(refundItems) == 0 {
		t.Fatalf("list refund items: %v", err)
	}

	automation := &testutil.FakeAutomationClient{DeleteErr: errors.New("dial tcp timeout")}
	resolver := &testutil.FakeAutomationResolver{Client: automation}
	svc := NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, resolver, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
	svc.SetRefundCleanupRepo(repo)

	if err := svc.handleRefund(ctx, refundItems[0]); err != nil {
		t.Fatalf("handleRefund must not fail when the cleanup outbox is wired, got %v", err)
	}
	// Wallet credited.
	wallet, err := repo.GetWallet(ctx, user.ID)
	if err != nil {
		t.Fatalf("get wallet: %v", err)
	}
	if wallet.Balance != 8000 {
		t.Fatalf("expected wallet credited to 8000, got %d", wallet.Balance)
	}
	// Cleanup outbox entry exists (pending) so the worker retries the deletion.
	job, err := repo.GetRefundCleanupJobByWalletOrder(ctx, 1)
	if err != nil {
		t.Fatalf("expected refund cleanup job enqueued: %v", err)
	}
	if job.VPSID != inst.ID || job.HostID != 9100 || job.Status != domain.RefundCleanupPending {
		t.Fatalf("unexpected cleanup job: %+v", job)
	}
}

func jsonMarshalAny(v any) (string, error) {
	b, err := json.Marshal(v)
	return string(b), err
}

// TestDeleteVPSForRefund_RealHostErrorFails keeps the opposite direction: a
// genuine provider error (not "not found") must still fail so the refund item
// is marked failed and a manual retry can act.
func TestDeleteVPSForRefund_RealHostErrorFails(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	user := testutil.CreateUser(t, repo, "refunddel2", "refunddel2@example.com", "pass")
	seed := testutil.SeedCatalog(t, repo)

	baseOrder := domain.Order{UserID: user.ID, OrderNo: "ORD-REFUND-DEL-2", Status: domain.OrderStatusActive, TotalAmount: 3000, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &baseOrder); err != nil {
		t.Fatalf("create order: %v", err)
	}
	baseItem := domain.OrderItem{OrderID: baseOrder.ID, PackageID: seed.Package.ID, SystemID: seed.SystemImage.ID, Amount: 3000, Status: domain.OrderItemStatusActive, Action: "create", SpecJSON: "{}"}
	if err := repo.CreateOrderItems(ctx, []domain.OrderItem{baseItem}); err != nil {
		t.Fatalf("create item: %v", err)
	}
	items, err := repo.ListOrderItems(ctx, baseOrder.ID)
	if err != nil || len(items) == 0 {
		t.Fatalf("list items: %v", err)
	}
	expire := time.Now().Add(30 * 24 * time.Hour)
	inst := domain.VPSInstance{
		UserID:               user.ID,
		OrderItemID:          items[0].ID,
		AutomationInstanceID: "9002",
		Name:                 "vm-refund-del-2",
		PackageID:            seed.Package.ID,
		PackageName:          seed.Package.Name,
		MonthlyPrice:         3000,
		SpecJSON:             "{}",
		Status:               domain.VPSStatusRunning,
		CreatedAt:            time.Now(),
		ExpireAt:             &expire,
	}
	if err := repo.CreateInstance(ctx, &inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}

	automation := &testutil.FakeAutomationClient{DeleteErr: errors.New("dial tcp timeout")}
	resolver := &testutil.FakeAutomationResolver{Client: automation}
	svc := NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, resolver, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)

	if err := svc.deleteVPSForRefund(ctx, inst); err == nil {
		t.Fatalf("expected real provider error to fail deleteVPSForRefund")
	}
	if _, err := repo.GetInstance(ctx, inst.ID); err != nil {
		t.Fatalf("expected local instance retained on genuine provider error, got err=%v", err)
	}
}
