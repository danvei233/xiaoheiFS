package usecase

import (
	"context"
	"fmt"

	"xiaoheiplay/internal/domain"
)

type WalletService struct {
	wallets WalletRepository
	audit   AuditRepository
}

func NewWalletService(wallets WalletRepository, audit AuditRepository) *WalletService {
	return &WalletService{wallets: wallets, audit: audit}
}

func (s *WalletService) GetWallet(ctx context.Context, userID int64) (domain.Wallet, error) {
	if s.wallets == nil {
		return domain.Wallet{}, ErrInvalidInput
	}
	return s.wallets.GetWallet(ctx, userID)
}

func (s *WalletService) ListTransactions(ctx context.Context, userID int64, limit, offset int) ([]domain.WalletTransaction, int, error) {
	if s.wallets == nil {
		return nil, 0, ErrInvalidInput
	}
	return s.wallets.ListWalletTransactions(ctx, userID, limit, offset)
}

func (s *WalletService) AdjustBalance(ctx context.Context, adminID int64, userID int64, amount int64, note string) (domain.Wallet, error) {
	if s.wallets == nil {
		return domain.Wallet{}, ErrInvalidInput
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
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{
			AdminID:    adminID,
			Action:     "wallet.adjust",
			TargetType: "user",
			TargetID:   fmt.Sprintf("%d", userID),
			DetailJSON: mustJSON(map[string]any{"amount": amount, "note": note}),
		})
	}
	return wallet, nil
}
