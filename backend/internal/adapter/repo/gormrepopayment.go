package repo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
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
		return err
	}
	*payment = fromOrderPaymentRow(row)
	return nil
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
		return domain.OrderPayment{}, sql.ErrNoRows
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
