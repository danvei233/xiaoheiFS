package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

func (r *GormRepo) CreatePayment(ctx context.Context, payment *domain.OrderPayment) error {
	tradeNo := strings.TrimSpace(payment.TradeNo)
	if tradeNo == "" {
		// Keep external semantics for empty trade_no while avoiding unique-key collisions.
		tradeNo = fmt.Sprintf("pending-%d-%d", payment.OrderID, time.Now().UnixNano())
	}
	row := toOrderPaymentRow(*payment)
	row.TradeNo = tradeNo
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		// Two concurrent pay requests with the same idempotency key can both miss
		// the lookup and race on the unique index. Surface a stable conflict
		// instead of a raw constraint error / 500; the caller returns the
		// existing session.
		if isUniqueConstraintError(err) {
			return appshared.ErrConflict
		}
		return err
	}
	*payment = fromOrderPaymentRow(row)
	return nil
}

// PayOrderWithBalanceAtomic persists the debit, ledger entry, payment, and review handoff together.
func (r *GormRepo) PayOrderWithBalanceAtomic(ctx context.Context, order domain.Order, payment *domain.OrderPayment) (wallet domain.Wallet, claimed bool, err error) {
	if payment == nil || strings.TrimSpace(payment.IdempotencyKey) == "" {
		return domain.Wallet{}, false, appshared.ErrInvalidInput
	}
	err = r.withTransientTxRetry(ctx, func(tx *gorm.DB) error {
		var existing orderPaymentRow
		lookupErr := tx.Where("order_id = ? AND idempotency_key = ?", order.ID, payment.IdempotencyKey).First(&existing).Error
		if lookupErr == nil {
			if existing.Method != "balance" || existing.Status != string(domain.PaymentStatusApproved) {
				return appshared.ErrConflict
			}
			settled, err := hasBalanceSettlementTx(tx, order)
			if err != nil {
				return err
			}
			if !settled {
				return appshared.ErrConflict
			}
			*payment = fromOrderPaymentRow(existing)
			var storedWallet walletRow
			if err := tx.Where("user_id = ?", order.UserID).First(&storedWallet).Error; err != nil {
				return err
			}
			wallet = domain.Wallet{ID: storedWallet.ID, UserID: storedWallet.UserID, Balance: storedWallet.Balance, UpdatedAt: storedWallet.UpdatedAt}
			return nil
		}
		if !errors.Is(lookupErr, gorm.ErrRecordNotFound) {
			return lookupErr
		}

		now := time.Now()
		result := tx.Model(&orderRow{}).
			Where("id = ? AND status = ?", order.ID, string(domain.OrderStatusPendingPayment)).
			Updates(map[string]any{
				"status":         domain.OrderStatusPendingReview,
				"pending_reason": "",
				"updated_at":     now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return nil
		}
		if err := tx.Model(&orderItemRow{}).
			Where("order_id = ? AND status = ?", order.ID, string(domain.OrderItemStatusPendingPayment)).
			Updates(map[string]any{"status": domain.OrderItemStatusPendingReview, "updated_at": now}).Error; err != nil {
			return err
		}
		wallet, err = adjustWalletBalanceTx(tx, order.UserID, -order.TotalAmount, "debit", "order", order.ID, "balance payment", balanceSettlementKey(order.ID))
		if err != nil {
			return err
		}
		row := toOrderPaymentRow(*payment)
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		*payment = fromOrderPaymentRow(row)
		claimed = true
		return nil
	})
	if err != nil {
		return domain.Wallet{}, false, err
	}
	if claimed || payment.ID > 0 {
		return wallet, claimed, nil
	}

	existing, lookupErr := r.GetPaymentByIdempotencyKey(ctx, order.ID, payment.IdempotencyKey)
	if lookupErr != nil {
		if errors.Is(lookupErr, appshared.ErrNotFound) {
			return domain.Wallet{}, false, nil
		}
		return domain.Wallet{}, false, lookupErr
	}
	if existing.Method != "balance" || existing.Status != domain.PaymentStatusApproved {
		return domain.Wallet{}, false, appshared.ErrConflict
	}
	hasTx, lookupErr := hasBalanceSettlementTx(r.gdb.WithContext(ctx), order)
	if lookupErr != nil {
		return domain.Wallet{}, false, lookupErr
	}
	if !hasTx {
		return domain.Wallet{}, false, nil
	}
	wallet, lookupErr = r.GetWallet(ctx, order.UserID)
	if lookupErr != nil {
		return domain.Wallet{}, false, lookupErr
	}
	*payment = existing
	return wallet, false, nil
}

func balanceSettlementKey(orderID int64) string {
	return fmt.Sprintf("balance_order:%d", orderID)
}

func hasBalanceSettlementTx(db *gorm.DB, order domain.Order) (bool, error) {
	var total int64
	err := db.Model(&walletTransactionRow{}).
		Where("idempotency_key = ? AND user_id = ? AND amount = ? AND type = ? AND ref_type = ? AND ref_id = ?",
			balanceSettlementKey(order.ID), order.UserID, -order.TotalAmount, "debit", "order", order.ID).
		Count(&total).Error
	return total == 1, err
}

func (r *GormRepo) ListPaymentsByOrder(ctx context.Context, orderID int64) ([]domain.OrderPayment, error) {

	var rows []orderPaymentRow
	if err := r.gdb.WithContext(ctx).Where("order_id = ?", orderID).Order("id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.OrderPayment, 0, len(rows))
	for _, row := range rows {
		out = append(out, fromOrderPaymentRow(row))
	}
	return out, nil

}

func (r *GormRepo) GetPaymentByTradeNo(ctx context.Context, tradeNo string) (domain.OrderPayment, error) {
	if strings.TrimSpace(tradeNo) == "" {
		return domain.OrderPayment{}, r.ensure(sql.ErrNoRows)
	}
	var row orderPaymentRow
	if err := r.gdb.WithContext(ctx).Where("trade_no = ?", tradeNo).First(&row).Error; err != nil {
		return domain.OrderPayment{}, r.ensure(err)
	}
	return fromOrderPaymentRow(row), nil
}

func (r *GormRepo) GetPaymentByIdempotencyKey(ctx context.Context, orderID int64, key string) (domain.OrderPayment, error) {

	var row orderPaymentRow
	if err := r.gdb.WithContext(ctx).Where("order_id = ? AND idempotency_key = ?", orderID, key).First(&row).Error; err != nil {
		return domain.OrderPayment{}, r.ensure(err)
	}
	return fromOrderPaymentRow(row), nil

}

func (r *GormRepo) UpdatePaymentStatus(ctx context.Context, id int64, status domain.PaymentStatus, reviewedBy *int64, reason string) error {

	return r.gdb.WithContext(ctx).Model(&orderPaymentRow{}).Where("id = ?", id).Updates(map[string]any{
		"status":        status,
		"reviewed_by":   reviewedBy,
		"review_reason": reason,
		"updated_at":    time.Now(),
	}).Error

}

// UpdatePaymentStatusIfCurrent transitions a payment's status only when it is
// currently in one of the allowed statuses (a compare-and-swap). It returns
// whether the row was updated. When the transition would violate the
// "single approved payment per order" invariant (a concurrent callback already
// approved a sibling payment), the unique partial index raises a constraint
// error which is mapped to appshared.ErrConflict so the caller can treat the
// race as "already settled" instead of surfacing a raw 500.
func (r *GormRepo) UpdatePaymentStatusIfCurrent(ctx context.Context, id int64, status domain.PaymentStatus, currentStatuses []domain.PaymentStatus, reviewedBy *int64, reason string) (bool, error) {
	statuses := make([]string, 0, len(currentStatuses))
	for _, s := range currentStatuses {
		statuses = append(statuses, string(s))
	}
	if len(statuses) == 0 {
		return false, nil
	}
	res := r.gdb.WithContext(ctx).Model(&orderPaymentRow{}).
		Where("id = ? AND status IN ?", id, statuses).
		Updates(map[string]any{
			"status":        status,
			"reviewed_by":   reviewedBy,
			"review_reason": reason,
			"updated_at":    time.Now(),
		})
	if res.Error != nil {
		if isUniqueConstraintError(res.Error) {
			return false, appshared.ErrConflict
		}
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

func (r *GormRepo) UpdatePaymentTradeNo(ctx context.Context, id int64, tradeNo string) error {

	return r.gdb.WithContext(ctx).Model(&orderPaymentRow{}).Where("id = ?", id).Updates(map[string]any{
		"trade_no":   tradeNo,
		"updated_at": time.Now(),
	}).Error

}

func (r *GormRepo) ListPayments(ctx context.Context, filter appshared.PaymentFilter, limit, offset int) ([]domain.OrderPayment, int, error) {

	q := r.gdb.WithContext(ctx).Model(&orderPaymentRow{})
	if filter.Status != "" {
		q = q.Where("status = ?", filter.Status)
	}
	if filter.From != nil {
		q = q.Where("created_at >= ?", filter.From)
	}
	if filter.To != nil {
		q = q.Where("created_at <= ?", filter.To)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 20
	}
	var rows []orderPaymentRow
	if err := q.Order("id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.OrderPayment, 0, len(rows))
	for _, row := range rows {
		out = append(out, fromOrderPaymentRow(row))
	}
	return out, int(total), nil

}
