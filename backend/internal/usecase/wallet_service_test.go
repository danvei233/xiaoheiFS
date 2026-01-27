package usecase_test

import (
	"context"
	"testing"

	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/usecase"
)

func TestWalletService_AdjustBalance(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "ws1", "ws1@example.com", "pass")
	svc := usecase.NewWalletService(repo, repo)

	wallet, err := svc.AdjustBalance(context.Background(), 1, user.ID, 2000, "seed")
	if err != nil {
		t.Fatalf("adjust: %v", err)
	}
	if wallet.Balance < 2000 {
		t.Fatalf("expected balance")
	}
}
