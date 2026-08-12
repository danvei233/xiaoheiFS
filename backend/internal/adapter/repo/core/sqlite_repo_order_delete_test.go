package repo_test

import (
	"context"
	"testing"

	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func TestSQLiteRepo_DeleteOrderRejectsMoneyOrInfraOrders(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	user := testutil.CreateUser(t, repo, "del-guard", "del-guard@example.com", "pass")

	// 1. Clean pending order is deletable (rollback/cancel cleanup path).
	clean := domain.Order{UserID: user.ID, OrderNo: "ORD-DEL-CLEAN", Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &clean); err != nil {
		t.Fatalf("create clean order: %v", err)
	}
	if err := repo.DeleteOrder(ctx, clean.ID); err != nil {
		t.Fatalf("delete clean order: %v", err)
	}

	// 2. Approved payment means money moved: deletion must be refused.
	paid := domain.Order{UserID: user.ID, OrderNo: "ORD-DEL-PAID", Status: domain.OrderStatusPendingReview, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &paid); err != nil {
		t.Fatalf("create paid order: %v", err)
	}
	pay := domain.OrderPayment{OrderID: paid.ID, UserID: user.ID, Method: "balance", Amount: 1000, Currency: "CNY", TradeNo: "PAID-1", Status: domain.PaymentStatusApproved}
	if err := repo.CreatePayment(ctx, &pay); err != nil {
		t.Fatalf("create payment: %v", err)
	}
	if err := repo.DeleteOrder(ctx, paid.ID); err != appshared.ErrConflict {
		t.Fatalf("paid order delete should conflict, got %v", err)
	}

	// 3. Wallet debit (even without an approved payment row) means money moved.
	if _, err := repo.AdjustWalletBalance(ctx, user.ID, 1000, "credit", "seed", 1, "init"); err != nil {
		t.Fatalf("seed wallet: %v", err)
	}
	debit := domain.Order{UserID: user.ID, OrderNo: "ORD-DEL-DEBIT", Status: domain.OrderStatusPendingReview, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &debit); err != nil {
		t.Fatalf("create debit order: %v", err)
	}
	if _, err := repo.AdjustWalletBalance(ctx, user.ID, -1000, "debit", "order", debit.ID, "legacy debit"); err != nil {
		t.Fatalf("debit wallet: %v", err)
	}
	if err := repo.DeleteOrder(ctx, debit.ID); err != appshared.ErrConflict {
		t.Fatalf("debit order delete should conflict, got %v", err)
	}

	// 4. A linked VPS instance means real infrastructure: deletion must be refused.
	instOrder := domain.Order{UserID: user.ID, OrderNo: "ORD-DEL-INST", Status: domain.OrderStatusProvisioning, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &instOrder); err != nil {
		t.Fatalf("create inst order: %v", err)
	}
	items := []domain.OrderItem{{OrderID: instOrder.ID, Amount: 1000, Status: domain.OrderItemStatusProvisioning, Action: "create", SpecJSON: "{}"}}
	if err := repo.CreateOrderItems(ctx, items); err != nil {
		t.Fatalf("create inst item: %v", err)
	}
	gotItems, err := repo.ListOrderItems(ctx, instOrder.ID)
	if err != nil || len(gotItems) != 1 {
		t.Fatalf("list inst items: len=%d err=%v", len(gotItems), err)
	}
	inst := domain.VPSInstance{UserID: user.ID, OrderItemID: gotItems[0].ID, AutomationInstanceID: "host-1", Name: "vm-1", Status: domain.VPSStatusRunning, AdminStatus: domain.VPSAdminStatusNormal, SpecJSON: "{}"}
	if err := repo.CreateInstance(ctx, &inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}
	if err := repo.DeleteOrder(ctx, instOrder.ID); err != appshared.ErrConflict {
		t.Fatalf("inst order delete should conflict, got %v", err)
	}

	// 5. Unconfirmed (pending_review) manual payment does NOT block deletion.
	pending := domain.Order{UserID: user.ID, OrderNo: "ORD-DEL-PENDING", Status: domain.OrderStatusPendingReview, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &pending); err != nil {
		t.Fatalf("create pending order: %v", err)
	}
	ppay := domain.OrderPayment{OrderID: pending.ID, UserID: user.ID, Method: "manual", Amount: 1000, Currency: "CNY", TradeNo: "PENDING-1", Status: domain.PaymentStatusPendingReview}
	if err := repo.CreatePayment(ctx, &ppay); err != nil {
		t.Fatalf("create pending payment: %v", err)
	}
	if err := repo.DeleteOrder(ctx, pending.ID); err != nil {
		t.Fatalf("pending payment order should be deletable, got %v", err)
	}
}
