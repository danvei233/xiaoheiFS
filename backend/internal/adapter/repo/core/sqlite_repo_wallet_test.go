package repo_test

import (
	"context"
	"testing"
	"time"

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
	monthStart := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.Local)
	summary, err := repo.SummarizeWalletOrders(ctx, user.ID, monthStart, monthStart.AddDate(0, 1, 0))
	if err != nil {
		t.Fatalf("summarize wallet orders: %v", err)
	}
	if summary.Total != 1 || summary.Pending != 1 || summary.MonthRecharge != 0 || summary.MonthWithdraw != 0 {
		t.Fatalf("unexpected pending summary: %+v", summary)
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
	summary, err = repo.SummarizeWalletOrders(ctx, user.ID, monthStart, monthStart.AddDate(0, 1, 0))
	if err != nil {
		t.Fatalf("summarize approved wallet orders: %v", err)
	}
	if summary.Total != 1 || summary.Pending != 0 || summary.MonthRecharge != 2550 {
		t.Fatalf("unexpected approved summary: %+v", summary)
	}
	allOrders, total, err = repo.ListAllWalletOrders(ctx, string(domain.WalletOrderApproved), 10, 0)
	if err != nil {
		t.Fatalf("list approved wallet orders: %v", err)
	}
	if total != 1 || len(allOrders) != 1 || allOrders[0].Status != domain.WalletOrderApproved {
		t.Fatalf("unexpected approved orders: total=%d len=%d status=%s", total, len(allOrders), allOrders[0].Status)
	}
}

func TestSQLiteRepo_WalletOrderSummaryMonthBoundaries(t *testing.T) {
	sqlDB, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	user := domain.User{
		Username:     "wallet_summary_boundary",
		Email:        "wallet-summary-boundary@example.com",
		PasswordHash: "hash",
		Role:         domain.UserRoleUser,
		Status:       domain.UserStatusActive,
	}
	if err := repo.CreateUser(ctx, &user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	other := domain.User{
		Username:     "wallet_summary_boundary_other",
		Email:        "wallet-summary-boundary-other@example.com",
		PasswordHash: "hash",
		Role:         domain.UserRoleUser,
		Status:       domain.UserStatusActive,
	}
	if err := repo.CreateUser(ctx, &other); err != nil {
		t.Fatalf("create other user: %v", err)
	}

	monthStart := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.Local)
	monthEnd := monthStart.AddDate(0, 1, 0)
	seed := func(userID int64, orderType domain.WalletOrderType, status domain.WalletOrderStatus, amount int64, createdAt time.Time) {
		t.Helper()
		order := domain.WalletOrder{
			UserID:   userID,
			Type:     orderType,
			Amount:   amount,
			Currency: "CNY",
			Status:   status,
			MetaJSON: "{}",
		}
		if err := repo.CreateWalletOrder(ctx, &order); err != nil {
			t.Fatalf("create wallet order: %v", err)
		}
		if _, err := sqlDB.ExecContext(ctx, "UPDATE wallet_orders SET created_at = ? WHERE id = ?", createdAt, order.ID); err != nil {
			t.Fatalf("set wallet order created_at: %v", err)
		}
	}

	seed(user.ID, domain.WalletOrderRecharge, domain.WalletOrderApproved, 100, monthStart)
	seed(user.ID, domain.WalletOrderRecharge, domain.WalletOrderApproved, 200, monthEnd.Add(-time.Second))
	seed(user.ID, domain.WalletOrderRecharge, domain.WalletOrderApproved, 400, monthStart.Add(-time.Second))
	seed(user.ID, domain.WalletOrderRecharge, domain.WalletOrderApproved, 800, monthEnd)
	seed(user.ID, domain.WalletOrderWithdraw, domain.WalletOrderApproved, 500, monthStart.Add(time.Hour))
	seed(user.ID, domain.WalletOrderRecharge, domain.WalletOrderPendingReview, 1600, monthStart.Add(time.Hour))
	seed(other.ID, domain.WalletOrderRecharge, domain.WalletOrderApproved, 3200, monthStart.Add(time.Hour))

	summary, err := repo.SummarizeWalletOrders(ctx, user.ID, monthStart, monthEnd)
	if err != nil {
		t.Fatalf("summarize wallet orders: %v", err)
	}
	if summary.Total != 6 || summary.Pending != 1 || summary.MonthRecharge != 300 || summary.MonthWithdraw != 500 {
		t.Fatalf("unexpected boundary summary: %+v", summary)
	}
}
