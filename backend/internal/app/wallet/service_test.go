package wallet_test

import (
	"context"
	"testing"
	appwallet "xiaoheiplay/internal/app/wallet"
	"xiaoheiplay/internal/testutil"
)

func TestWalletService_AdjustBalance(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "ws1", "ws1@example.com", "pass")
	svc := appwallet.NewService(repo, repo)

	wallet, err := svc.AdjustBalance(context.Background(), 1, user.ID, 2000, "seed")
	if err != nil {
		t.Fatalf("adjust: %v", err)
	}
	if wallet.Balance < 2000 {
		t.Fatalf("expected balance")
	}
}
