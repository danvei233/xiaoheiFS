package walletorder_test

import (
	"context"
	"testing"
	appshared "xiaoheiplay/internal/app/shared"
	appwalletorder "xiaoheiplay/internal/app/walletorder"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func TestWalletOrderService_CreateWithdrawInsufficient(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "w1", "w1@example.com", "pass")
	svc := appwalletorder.NewService(repo, repo, repo, repo, repo, nil, repo)

	_, err := svc.CreateWithdraw(context.Background(), user.ID, appshared.WalletOrderCreateInput{Amount: 100000, Currency: "CNY"})
	if err != appshared.ErrInsufficientBalance {
		t.Fatalf("expected insufficient balance, got %v", err)
	}
}

func TestWalletOrderService_ApproveRecharge(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "w2", "w2@example.com", "pass")
	svc := appwalletorder.NewService(repo, repo, repo, repo, repo, nil, repo)

	order, err := svc.CreateRecharge(context.Background(), user.ID, appshared.WalletOrderCreateInput{Amount: 250000, Currency: "CNY"})
	if err != nil {
		t.Fatalf("create recharge: %v", err)
	}
	if order.Status != domain.WalletOrderPendingReview {
		t.Fatalf("expected pending review")
	}
	_, wallet, err := svc.Approve(context.Background(), 1, order.ID)
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if wallet == nil || wallet.Balance < 2500 {
		t.Fatalf("expected wallet credited")
	}
}
