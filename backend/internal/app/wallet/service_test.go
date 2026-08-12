package wallet_test

import (
	"context"
	"errors"
	"testing"
	appwallet "xiaoheiplay/internal/app/wallet"
	"xiaoheiplay/internal/domain"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/testutil"
)

type failingAuditRepo struct{}

func (f failingAuditRepo) ListAuditLogs(ctx context.Context, filter appshared.AuditLogFilter, limit, offset int) ([]domain.AdminAuditLog, int, error) {
	return nil, 0, domain.ErrListAuditLogsError
}

func (failingAuditRepo) AddAuditLog(ctx context.Context, log domain.AdminAuditLog) error {
	return errors.New("audit backend down")
}


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

func TestWalletService_AdjustBalance_ZeroAmountIsNoop(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "ws-zero", "ws-zero@example.com", "pass")
	svc := appwallet.NewService(repo, repo)

	wallet, err := svc.AdjustBalance(context.Background(), 1, user.ID, 0, "noop")
	if err != nil {
		t.Fatalf("adjust zero: %v", err)
	}
	if wallet.Balance != 0 {
		t.Fatalf("expected unchanged balance, got %d", wallet.Balance)
	}
	_, total, err := repo.ListWalletTransactions(context.Background(), user.ID, 10, 0)
	if err != nil {
		t.Fatalf("list transactions: %v", err)
	}
	if total != 0 {
		t.Fatalf("expected no transaction for zero adjustment, got %d", total)
	}
}

func TestWalletService_AdjustBalance_AuditErrorSurfaced(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "ws-audit", "ws-audit@example.com", "pass")
	svc := appwallet.NewService(repo, failingAuditRepo{})

	if _, err := svc.AdjustBalance(context.Background(), 1, user.ID, 2000, "seed"); err == nil {
		t.Fatalf("expected audit error to be surfaced")
	}
}

// TestAdjustBalance_RejectsAbsurdAmount locks P2-25: an admin wallet
// adjustment must be bounded so a malformed huge payload cannot mint or burn an
// unbounded balance (and overflow int64 during parsing).
func TestAdjustBalance_RejectsAbsurdAmount(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	user := testutil.CreateUser(t, repo, "wallet_bad", "wallet_bad@example.com", "pass")
	svc := appwallet.NewService(repo, repo)

	if _, err := svc.AdjustBalance(ctx, 1, user.ID, 200_000_000_000, "huge"); err != domain.ErrInvalidInput {
		t.Fatalf("expected ErrInvalidInput for absurd credit, got %v", err)
	}
	if _, err := svc.AdjustBalance(ctx, 1, user.ID, -200_000_000_000, "huge"); err != domain.ErrInvalidInput {
		t.Fatalf("expected ErrInvalidInput for absurd debit, got %v", err)
	}
}
