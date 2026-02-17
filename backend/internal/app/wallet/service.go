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

func (s *Service) AdjustBalance(ctx context.Context, adminID int64, userID int64, amount int64, note string) (domain.Wallet, error) {
	if s.wallets == nil {
		return domain.Wallet{}, appshared.ErrInvalidInput
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

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
