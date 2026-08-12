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

func TestWalletOrderService_CreateRechargeWithdraw_RejectsNonCNYCurrency(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "w-cur", "w-cur@example.com", "pass")
	svc := appwalletorder.NewService(repo, repo, repo, repo, repo, nil, repo)

	if _, err := svc.CreateRecharge(context.Background(), user.ID, appshared.WalletOrderCreateInput{Amount: 10000, Currency: "USD"}); err != appshared.ErrInvalidInput {
		t.Fatalf("expected invalid input for USD recharge, got %v", err)
	}
	if _, err := svc.CreateWithdraw(context.Background(), user.ID, appshared.WalletOrderCreateInput{Amount: 10000, Currency: "EUR"}); err != appshared.ErrInvalidInput {
		t.Fatalf("expected invalid input for EUR withdraw, got %v", err)
	}
	order, err := svc.CreateRecharge(context.Background(), user.ID, appshared.WalletOrderCreateInput{Amount: 10000, Currency: ""})
	if err != nil {
		t.Fatalf("create recharge with default currency: %v", err)
	}
	if order.Currency != "CNY" {
		t.Fatalf("expected default currency CNY, got %s", order.Currency)
	}
}

func TestWalletOrderService_CreateRechargeWithdraw_RejectsAbsurdAmount(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "w-amt", "w-amt@example.com", "pass")
	svc := appwalletorder.NewService(repo, repo, repo, repo, repo, nil, repo)

	if _, err := svc.CreateRecharge(context.Background(), user.ID, appshared.WalletOrderCreateInput{Amount: 1_000_000_000_000_000, Currency: "CNY"}); err != appshared.ErrInvalidInput {
		t.Fatalf("expected invalid input for absurd recharge amount, got %v", err)
	}
	// A withdraw larger than the balance must keep returning the balance error,
	// not the ceiling error (insufficient-balance check comes after ceiling).
	if _, err := svc.CreateWithdraw(context.Background(), user.ID, appshared.WalletOrderCreateInput{Amount: 1_000_000_000_000_000, Currency: "CNY"}); err != appshared.ErrInvalidInput {
		t.Fatalf("expected invalid input for absurd withdraw amount, got %v", err)
	}
}

func TestWalletOrderService_CancelWithdrawByUser(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "w-cancel-wd", "w-cancel-wd@example.com", "pass")
	if _, err := repo.AdjustWalletBalance(context.Background(), user.ID, 50000, "credit", "seed", 1, "init"); err != nil {
		t.Fatalf("seed wallet: %v", err)
	}
	svc := appwalletorder.NewService(repo, repo, repo, repo, repo, nil, repo)
	order, err := svc.CreateWithdraw(context.Background(), user.ID, appshared.WalletOrderCreateInput{Amount: 10000, Currency: "CNY"})
	if err != nil {
		t.Fatalf("create withdraw: %v", err)
	}
	cancelled, err := svc.CancelByUser(context.Background(), user.ID, order.ID, "user_cancel")
	if err != nil {
		t.Fatalf("cancel withdraw: %v", err)
	}
	if cancelled.Status != domain.WalletOrderRejected {
		t.Fatalf("expected rejected withdraw, got %s", cancelled.Status)
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
