package repo_test

import (
	"context"
	"testing"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func TestSQLiteRepo_WalletQueries(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()

	user := domain.User{
		Username:     "wallet_user",
		Email:        "wallet@example.com",
		PasswordHash: "hash",
		Role:         domain.UserRoleUser,
		Status:       domain.UserStatusActive,
	}
	if err := repo.CreateUser(ctx, &user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	wallet, err := repo.GetWallet(ctx, user.ID)
	if err != nil {
		t.Fatalf("get wallet: %v", err)
	}
	if wallet.UserID != user.ID || wallet.Balance != 0 {
		t.Fatalf("unexpected wallet: %+v", wallet)
	}

	wallet.Balance = 9950
	if err := repo.UpsertWallet(ctx, &wallet); err != nil {
		t.Fatalf("upsert wallet: %v", err)
	}
	wallet, err = repo.GetWallet(ctx, user.ID)
	if err != nil {
		t.Fatalf("get wallet after upsert: %v", err)
	}
	if wallet.Balance != 9950 {
		t.Fatalf("expected balance 9950, got %d", wallet.Balance)
	}

	tx := domain.WalletTransaction{
		UserID:  user.ID,
		Amount:  2550,
		Type:    "credit",
		RefType: "seed",
		RefID:   101,
		Note:    "initial",
	}
	if err := repo.AddWalletTransaction(ctx, &tx); err != nil {
		t.Fatalf("add wallet transaction: %v", err)
	}
	if tx.ID == 0 {
		t.Fatalf("expected transaction id")
	}
	items, total, err := repo.ListWalletTransactions(ctx, user.ID, 10, 0)
	if err != nil {
		t.Fatalf("list wallet transactions: %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("unexpected tx list: total=%d len=%d", total, len(items))
	}
	hasTx, err := repo.HasWalletTransaction(ctx, user.ID, "seed", 101)
	if err != nil {
		t.Fatalf("has wallet transaction: %v", err)
	}
	if !hasTx {
		t.Fatalf("expected wallet transaction to exist")
	}

	order := domain.WalletOrder{
		UserID:   user.ID,
		Type:     domain.WalletOrderRecharge,
		Amount:   2550,
		Currency: "CNY",
		Status:   domain.WalletOrderPendingReview,
		Note:     "recharge",
		MetaJSON: "{}",
	}
	if err := repo.CreateWalletOrder(ctx, &order); err != nil {
		t.Fatalf("create wallet order: %v", err)
	}
	if order.ID == 0 {
		t.Fatalf("expected wallet order id")
	}
	gotOrder, err := repo.GetWalletOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("get wallet order: %v", err)
	}
	if gotOrder.ID != order.ID || gotOrder.Status != order.Status {
		t.Fatalf("unexpected wallet order: %+v", gotOrder)
	}
	userOrders, total, err := repo.ListWalletOrders(ctx, user.ID, 10, 0)
	if err != nil {
		t.Fatalf("list wallet orders: %v", err)
	}
	if total != 1 || len(userOrders) != 1 {
		t.Fatalf("unexpected user orders: total=%d len=%d", total, len(userOrders))
	}
	allOrders, total, err := repo.ListAllWalletOrders(ctx, "", 10, 0)
	if err != nil {
		t.Fatalf("list all wallet orders: %v", err)
	}
	if total != 1 || len(allOrders) != 1 {
		t.Fatalf("unexpected all orders: total=%d len=%d", total, len(allOrders))
	}

	reviewer := user.ID
	if err := repo.UpdateWalletOrderStatus(ctx, order.ID, domain.WalletOrderApproved, &reviewer, "ok"); err != nil {
		t.Fatalf("update wallet order status: %v", err)
	}
	allOrders, total, err = repo.ListAllWalletOrders(ctx, string(domain.WalletOrderApproved), 10, 0)
	if err != nil {
		t.Fatalf("list approved wallet orders: %v", err)
	}
	if total != 1 || len(allOrders) != 1 || allOrders[0].Status != domain.WalletOrderApproved {
		t.Fatalf("unexpected approved orders: total=%d len=%d status=%s", total, len(allOrders), allOrders[0].Status)
	}
}
