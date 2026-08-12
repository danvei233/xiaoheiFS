package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
		var adjustErr error
		wallet, adjustErr = adjustWalletBalanceTx(tx, userID, amount, txType, refType, refID, note, "")
		return adjustErr
	})
	if err != nil {
		return domain.Wallet{}, err
	}
	return wallet, nil
}

func (r *GormRepo) AdjustWalletBalanceIdempotent(ctx context.Context, userID int64, amount int64, txType, refType string, refID int64, note, idempotencyKey string) (wallet domain.Wallet, err error) {
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey == "" {
		return domain.Wallet{}, appshared.ErrInvalidInput
	}
	err = r.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var adjustErr error
		wallet, adjustErr = adjustWalletBalanceTx(tx, userID, amount, txType, refType, refID, note, idempotencyKey)
		return adjustErr
	})
	if err == nil {
		return wallet, nil
	}
	var existing walletTransactionRow
	lookupErr := r.gdb.WithContext(ctx).Where("idempotency_key = ?", idempotencyKey).First(&existing).Error
	if lookupErr != nil {
		return domain.Wallet{}, err
	}
	if existing.UserID != userID || existing.Amount != amount || existing.Type != txType || existing.RefType != refType || existing.RefID != refID {
		return domain.Wallet{}, appshared.ErrConflict
	}
	return r.GetWallet(ctx, userID)
}

func adjustWalletBalanceTx(tx *gorm.DB, userID int64, amount int64, txType, refType string, refID int64, note, idempotencyKey string) (domain.Wallet, error) {
	now := time.Now()
	seed := walletRow{UserID: userID, Balance: 0, UpdatedAt: now}
	if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}}, DoNothing: true}).Create(&seed).Error; err != nil {
		return domain.Wallet{}, err
	}
	result := tx.Model(&walletRow{}).
		Where("user_id = ? AND balance + ? >= 0", userID, amount).
		Updates(map[string]any{
			"balance":    gorm.Expr("balance + ?", amount),
			"updated_at": now,
		})
	if result.Error != nil {
		return domain.Wallet{}, result.Error
	}
	if result.RowsAffected != 1 {
		return domain.Wallet{}, appshared.ErrInsufficientBalance
	}
	txRow := walletTransactionRow{UserID: userID, Amount: amount, Type: txType, RefType: refType, RefID: refID, Note: note}
	if idempotencyKey != "" {
		txRow.IdempotencyKey = &idempotencyKey
	}
	if err := tx.Create(&txRow).Error; err != nil {
		return domain.Wallet{}, err
	}
	var row walletRow
	if err := tx.Where("user_id = ?", userID).First(&row).Error; err != nil {
		return domain.Wallet{}, err
	}
	return domain.Wallet{ID: row.ID, UserID: row.UserID, Balance: row.Balance, UpdatedAt: row.UpdatedAt}, nil
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
	return r.ListWalletOrdersFiltered(ctx, userID, "", "", limit, offset)
}

// ListWalletOrdersFiltered 支持 status / type 服务端过滤（cross-module residual：
// 前端不得靠拉取固定 500 条后本地分页伪装完整结果）。
func (r *GormRepo) ListWalletOrdersFiltered(ctx context.Context, userID int64, status, orderType string, limit, offset int) ([]domain.WalletOrder, int, error) {
	if limit <= 0 {
		limit = 20
	}
	q := r.gdb.WithContext(ctx).Model(&walletOrderRow{}).Where("user_id = ?", userID)
	if st := strings.TrimSpace(status); st != "" && st != "all" {
		q = q.Where("status = ?", st)
	}
	if typ := strings.TrimSpace(orderType); typ != "" && typ != "all" {
		q = q.Where("type = ?", typ)
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

func (r *GormRepo) SummarizeWalletOrders(ctx context.Context, userID int64, monthStart, monthEnd time.Time) (domain.WalletOrderSummary, error) {
	base := func() *gorm.DB {
		return r.gdb.WithContext(ctx).Model(&walletOrderRow{}).Where("user_id = ?", userID)
	}
	var summary domain.WalletOrderSummary
	if err := base().Count(&summary.Total).Error; err != nil {
		return domain.WalletOrderSummary{}, err
	}
	if err := base().Where("status IN ?", []string{"pending", "pending_review"}).Count(&summary.Pending).Error; err != nil {
		return domain.WalletOrderSummary{}, err
	}
	for orderType, target := range map[string]*int64{
		string(domain.WalletOrderRecharge): &summary.MonthRecharge,
		string(domain.WalletOrderWithdraw): &summary.MonthWithdraw,
	} {
		var amount int64
		if err := base().Where("type = ? AND status = ? AND created_at >= ? AND created_at < ?", orderType, domain.WalletOrderApproved, monthStart, monthEnd).Select("COALESCE(SUM(amount), 0)").Scan(&amount).Error; err != nil {
			return domain.WalletOrderSummary{}, err
		}
		*target = amount
	}
	return summary, nil
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

func (r *GormRepo) UpdateWalletOrderStatusIfCurrent(ctx context.Context, id int64, currentStatus, targetStatus domain.WalletOrderStatus, reviewedBy *int64, reason string) (bool, error) {
	res := r.gdb.WithContext(ctx).Model(&walletOrderRow{}).
		Where("id = ? AND status = ?", id, string(currentStatus)).
		Updates(map[string]any{
			"status":        string(targetStatus),
			"reviewed_by":   reviewedBy,
			"review_reason": reason,
			"updated_at":    time.Now(),
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// ApproveWalletOrderAtomic claims a pending wallet order and applies its ledger entry in one transaction.
func (r *GormRepo) ApproveWalletOrderAtomic(ctx context.Context, order domain.WalletOrder, adminID int64, amount int64, txType string) (wallet domain.Wallet, claimed bool, err error) {
	err = r.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		result := tx.Model(&walletOrderRow{}).
			Where("id = ? AND status = ?", order.ID, string(domain.WalletOrderPendingReview)).
			Updates(map[string]any{
				"status":        string(domain.WalletOrderApproved),
				"reviewed_by":   adminID,
				"review_reason": "",
				"updated_at":    now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return nil
		}
		var adjustErr error
		wallet, adjustErr = adjustWalletBalanceTx(tx, order.UserID, amount, txType, "wallet_order", order.ID, string(order.Type), fmt.Sprintf("wallet_order:%d", order.ID))
		if adjustErr != nil {
			return adjustErr
		}
		claimed = true
		return nil
	})
	if err != nil {
		return domain.Wallet{}, false, err
	}
	return wallet, claimed, nil
}

func (r *GormRepo) UpdateWalletOrderMeta(ctx context.Context, id int64, metaJSON string) error {
	return r.gdb.WithContext(ctx).Model(&walletOrderRow{}).Where("id = ?", id).Updates(map[string]any{
		"meta_json":  metaJSON,
		"updated_at": time.Now(),
	}).Error
}

// GetWalletOrderByPaymentTradeNo resolves a recharge wallet order by the
// provider trade_no stored in its meta JSON, using a dialect-aware JSON
// extraction so a deep wallet order is found in O(1) instead of a bounded
// paginated scan (P1-4). Returns gorm.ErrRecordNotFound when unmatched.
func (r *GormRepo) GetWalletOrderByPaymentTradeNo(ctx context.Context, tradeNo string) (domain.WalletOrder, error) {
	tradeNo = strings.TrimSpace(tradeNo)
	if tradeNo == "" {
		return domain.WalletOrder{}, gorm.ErrRecordNotFound
	}
	var expr string
	switch r.gdb.Dialector.Name() {
	case "mysql":
		expr = "JSON_UNQUOTE(JSON_EXTRACT(meta_json, '$.payment_trade_no')) = ?"
	case "postgres", "postgresql":
		expr = "meta_json->>'payment_trade_no' = ?"
	default: // sqlite
		expr = "json_extract(meta_json, '$.payment_trade_no') = ?"
	}
	var row walletOrderRow
	if err := r.gdb.WithContext(ctx).Model(&walletOrderRow{}).
		Where(expr, tradeNo).
		Order("id DESC").
		First(&row).Error; err != nil {
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
