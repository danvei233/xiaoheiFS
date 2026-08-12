package wallet

import (
	"context"
	"encoding/json"
	"fmt"

	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

type Service struct {
	wallets appports.WalletRepository
	audit   appports.AuditRepository
}

func NewService(wallets appports.WalletRepository, audit appports.AuditRepository) *Service {
	return &Service{wallets: wallets, audit: audit}
}

func (s *Service) GetWallet(ctx context.Context, userID int64) (domain.Wallet, error) {
	if s.wallets == nil {
		return domain.Wallet{}, appshared.ErrInvalidInput
	}
	return s.wallets.GetWallet(ctx, userID)
}

func (s *Service) ListTransactions(ctx context.Context, userID int64, limit, offset int) ([]domain.WalletTransaction, int, error) {
	if s.wallets == nil {
		return nil, 0, appshared.ErrInvalidInput
	}
	return s.wallets.ListWalletTransactions(ctx, userID, limit, offset)
}

// maxAdminAdjustAmountCents bounds a single admin wallet adjustment so a
// malformed payload cannot mint or burn an unbounded balance (and overflow
// int64 during amount parsing).
const maxAdminAdjustAmountCents int64 = 100_000_000_000 // 10亿 元 in cents

func (s *Service) AdjustBalance(ctx context.Context, adminID int64, userID int64, amount int64, note string) (domain.Wallet, error) {
	if s.wallets == nil {
		return domain.Wallet{}, appshared.ErrInvalidInput
	}
	if amount > maxAdminAdjustAmountCents || amount < -maxAdminAdjustAmountCents {
		return domain.Wallet{}, appshared.ErrInvalidInput
	}
	if amount == 0 {
		// A zero adjustment is a no-op: do not emit a 0-value ledger row.
		return s.wallets.GetWallet(ctx, userID)
	}
	txType := "credit"
	if amount < 0 {
		txType = "debit"
	}
	wallet, err := s.wallets.AdjustWalletBalance(ctx, userID, amount, txType, "admin_adjust", adminID, note)
	if err != nil {
		return domain.Wallet{}, err
	}
	if s.audit != nil {
		// Surface audit failures instead of silently recording a balance
		// change with no trace (the balance is already committed).
		if err := s.audit.AddAuditLog(ctx, domain.AdminAuditLog{
			AdminID:    adminID,
			Action:     "wallet.adjust",
			TargetType: "user",
			TargetID:   fmt.Sprintf("%d", userID),
			DetailJSON: mustJSON(map[string]any{"amount": amount, "note": note}),
		}); err != nil {
			return wallet, err
		}
	}
	return wallet, nil
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
