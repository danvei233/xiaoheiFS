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

func TestWalletOrderService_CancelPendingRechargeAndRefundByUser(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "w3", "w3@example.com", "pass")
	svc := appwalletorder.NewService(repo, repo, repo, repo, repo, nil, repo)

	recharge, err := svc.CreateRecharge(context.Background(), user.ID, appshared.WalletOrderCreateInput{Amount: 10000, Currency: "CNY"})
	if err != nil {
		t.Fatalf("create recharge: %v", err)
	}
	cancelledRecharge, err := svc.CancelByUser(context.Background(), user.ID, recharge.ID, "user_cancel")
	if err != nil {
		t.Fatalf("cancel recharge: %v", err)
	}
	if cancelledRecharge.Status != domain.WalletOrderRejected {
		t.Fatalf("expected rejected recharge, got %s", cancelledRecharge.Status)
	}

	refund, err := svc.CreateRefundOrder(context.Background(), user.ID, 5000, "refund test", map[string]any{"source": "test"})
	if err != nil {
		t.Fatalf("create refund: %v", err)
	}
	cancelledRefund, err := svc.CancelByUser(context.Background(), user.ID, refund.ID, "user_cancel")
	if err != nil {
		t.Fatalf("cancel refund: %v", err)
	}
	if cancelledRefund.Status != domain.WalletOrderRejected {
		t.Fatalf("expected rejected refund, got %s", cancelledRefund.Status)
	}
}

func TestWalletOrderService_CancelByUserConflictOnNonPending(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "w4", "w4@example.com", "pass")
	svc := appwalletorder.NewService(repo, repo, repo, repo, repo, nil, repo)

	order, err := svc.CreateRecharge(context.Background(), user.ID, appshared.WalletOrderCreateInput{Amount: 10000, Currency: "CNY"})
	if err != nil {
		t.Fatalf("create recharge: %v", err)
	}
	if _, _, err = svc.Approve(context.Background(), 1, order.ID); err != nil {
		t.Fatalf("approve: %v", err)
	}
	if _, err = svc.CancelByUser(context.Background(), user.ID, order.ID, "late_cancel"); err != appshared.ErrConflict {
		t.Fatalf("expected conflict on approved order, got %v", err)
	}
}
