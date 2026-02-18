package repo

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

func (r *GormRepo) GetWallet(ctx context.Context, userID int64) (domain.Wallet, error) {
	var row walletRow
	if err := r.gdb.WithContext(ctx).Where("user_id = ?", userID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			w := domain.Wallet{UserID: userID, Balance: 0}
			if err := r.UpsertWallet(ctx, &w); err != nil {
				return domain.Wallet{}, err
			}
			return r.GetWallet(ctx, userID)
		}
		return domain.Wallet{}, err
	}
	return domain.Wallet{
		ID:        row.ID,
		UserID:    row.UserID,
		Balance:   row.Balance,
		UpdatedAt: row.UpdatedAt,
	}, nil
}

func (r *GormRepo) UpsertWallet(ctx context.Context, wallet *domain.Wallet) error {
	m := walletModel{
		ID:        wallet.ID,
		UserID:    wallet.UserID,
		Balance:   wallet.Balance,
		UpdatedAt: time.Now(),
	}
	if err := r.gdb.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"balance", "updated_at"}),
		}).
		Create(&m).Error; err != nil {
		return err
	}
	var got walletModel
	if err := r.gdb.WithContext(ctx).Select("id").Where("user_id = ?", wallet.UserID).First(&got).Error; err == nil {
		wallet.ID = got.ID
	}
	return nil
}

func (r *GormRepo) AddWalletTransaction(ctx context.Context, txItem *domain.WalletTransaction) error {
	row := walletTransactionRow{
		UserID:  txItem.UserID,
		Amount:  txItem.Amount,
		Type:    txItem.Type,
		RefType: txItem.RefType,
		RefID:   txItem.RefID,
		Note:    txItem.Note,
	}
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	txItem.ID = row.ID
	txItem.CreatedAt = row.CreatedAt
	return nil
}

func (r *GormRepo) ListWalletTransactions(ctx context.Context, userID int64, limit, offset int) ([]domain.WalletTransaction, int, error) {
	if limit <= 0 {
		limit = 20
	}
	q := r.gdb.WithContext(ctx).Model(&walletTransactionRow{}).Where("user_id = ?", userID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []walletTransactionRow
	if err := q.Order("id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.WalletTransaction, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.WalletTransaction{
			ID:        row.ID,
			UserID:    row.UserID,
			Amount:    row.Amount,
			Type:      row.Type,
			RefType:   row.RefType,
			RefID:     row.RefID,
			Note:      row.Note,
			CreatedAt: row.CreatedAt,
		})
	}
	return out, int(total), nil
}

func (r *GormRepo) AdjustWalletBalance(ctx context.Context, userID int64, amount int64, txType, refType string, refID int64, note string) (wallet domain.Wallet, err error) {
	err = r.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var w walletRow
		lock := clause.Locking{Strength: "UPDATE"}
		if e := tx.Clauses(lock).Where("user_id = ?", userID).First(&w).Error; e != nil {
			if errors.Is(e, gorm.ErrRecordNotFound) {
				w = walletRow{UserID: userID, Balance: 0, UpdatedAt: time.Now()}
				if e = tx.Create(&w).Error; e != nil {
					return e
				}
			} else {
				return e
			}
		}
		newBalance := w.Balance + amount
		if newBalance < 0 {
			return appshared.ErrInsufficientBalance
		}
		now := time.Now()
		if e := tx.Model(&walletRow{}).Where("user_id = ?", userID).Updates(map[string]any{
			"balance":    newBalance,
			"updated_at": now,
		}).Error; e != nil {
			return e
		}
		txRow := walletTransactionRow{
			UserID:  userID,
			Amount:  amount,
			Type:    txType,
			RefType: refType,
			RefID:   refID,
			Note:    note,
		}
		if e := tx.Create(&txRow).Error; e != nil {
			return e
		}
		wallet = domain.Wallet{
			ID:        w.ID,
			UserID:    userID,
			Balance:   newBalance,
			UpdatedAt: now,
		}
		return nil
	})
	if err != nil {
		return domain.Wallet{}, err
	}
	return wallet, nil
}

func (r *GormRepo) HasWalletTransaction(ctx context.Context, userID int64, refType string, refID int64) (bool, error) {
	var total int64
	if err := r.gdb.WithContext(ctx).Model(&walletTransactionRow{}).
		Where("user_id = ? AND ref_type = ? AND ref_id = ?", userID, refType, refID).
		Count(&total).Error; err != nil {
		return false, err
	}
	return total > 0, nil
}

func (r *GormRepo) CreateWalletOrder(ctx context.Context, order *domain.WalletOrder) error {
	row := walletOrderRow{
		UserID:   order.UserID,
		Type:     string(order.Type),
		Amount:   order.Amount,
		Currency: order.Currency,
		Status:   string(order.Status),
		Note:     order.Note,
		MetaJSON: order.MetaJSON,
	}
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	order.ID = row.ID
	order.CreatedAt = row.CreatedAt
	order.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *GormRepo) GetWalletOrder(ctx context.Context, id int64) (domain.WalletOrder, error) {
	var row walletOrderRow
	if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return domain.WalletOrder{}, r.ensure(err)
	}
	return domain.WalletOrder{
		ID:           row.ID,
		UserID:       row.UserID,
		Type:         domain.WalletOrderType(row.Type),
		Amount:       row.Amount,
		Currency:     row.Currency,
		Status:       domain.WalletOrderStatus(row.Status),
		Note:         row.Note,
		MetaJSON:     row.MetaJSON,
		ReviewedBy:   row.ReviewedBy,
		ReviewReason: row.ReviewReason,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}, nil
}

func (r *GormRepo) ListWalletOrders(ctx context.Context, userID int64, limit, offset int) ([]domain.WalletOrder, int, error) {
	if limit <= 0 {
		limit = 20
	}
	q := r.gdb.WithContext(ctx).Model(&walletOrderRow{}).Where("user_id = ?", userID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []walletOrderRow
	if err := q.Order("id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.WalletOrder, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.WalletOrder{
			ID:           row.ID,
			UserID:       row.UserID,
			Type:         domain.WalletOrderType(row.Type),
			Amount:       row.Amount,
			Currency:     row.Currency,
			Status:       domain.WalletOrderStatus(row.Status),
			Note:         row.Note,
			MetaJSON:     row.MetaJSON,
			ReviewedBy:   row.ReviewedBy,
			ReviewReason: row.ReviewReason,
			CreatedAt:    row.CreatedAt,
			UpdatedAt:    row.UpdatedAt,
		})
	}
	return out, int(total), nil
}

func (r *GormRepo) ListAllWalletOrders(ctx context.Context, status string, limit, offset int) ([]domain.WalletOrder, int, error) {
	if limit <= 0 {
		limit = 20
	}
	q := r.gdb.WithContext(ctx).Model(&walletOrderRow{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []walletOrderRow
	if err := q.Order("id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.WalletOrder, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.WalletOrder{
			ID:           row.ID,
			UserID:       row.UserID,
			Type:         domain.WalletOrderType(row.Type),
			Amount:       row.Amount,
			Currency:     row.Currency,
			Status:       domain.WalletOrderStatus(row.Status),
			Note:         row.Note,
			MetaJSON:     row.MetaJSON,
			ReviewedBy:   row.ReviewedBy,
			ReviewReason: row.ReviewReason,
			CreatedAt:    row.CreatedAt,
			UpdatedAt:    row.UpdatedAt,
		})
	}
	return out, int(total), nil
}

func (r *GormRepo) UpdateWalletOrderStatus(ctx context.Context, id int64, status domain.WalletOrderStatus, reviewedBy *int64, reason string) error {
	return r.gdb.WithContext(ctx).Model(&walletOrderRow{}).Where("id = ?", id).Updates(map[string]any{
		"status":        string(status),
		"reviewed_by":   reviewedBy,
		"review_reason": reason,
		"updated_at":    time.Now(),
	}).Error
}
