package repo

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"xiaoheiplay/internal/domain"
)

// maxEventAppendAttempts bounds retries when two concurrent appends for the same
// order race on the MAX(seq)+1 read and one loses the unique (order_id, seq)
// index (or hits a busy lock on SQLite). A lost race must not drop the event.
const maxEventAppendAttempts = 6

// eventAppendRetryDelay is the pause between retries so a competing transaction
// has time to commit before we re-read MAX(seq).
const eventAppendRetryDelay = 10 * time.Millisecond

// maxTransientTxRetries bounds retries for atomic write transactions that lose
// a write-lock race on SQLite (SQLITE_BUSY) or MySQL (deadlock / lock-wait
// timeout). Unlike a check-then-insert, these transactions roll back whole on
// error, so retrying the same closure is safe: a retry either wins the CAS or
// observes the other transaction's committed outcome.
const maxTransientTxRetries = 6

const transientTxRetryDelay = 10 * time.Millisecond

// withTransientTxRetry runs a read-write transaction, retrying on transient
// busy/lock errors so a concurrent winner resolves to a clean claimed=false (or
// a successful CAS) instead of surfacing SQLITE_BUSY / deadlock to the caller.
func (r *GormRepo) withTransientTxRetry(ctx context.Context, fn func(tx *gorm.DB) error) error {
	var lastErr error
	for attempt := 0; attempt < maxTransientTxRetries; attempt++ {
		err := r.gdb.WithContext(ctx).Transaction(fn)
		if err == nil || !isEventAppendTransient(err) {
			return err
		}
		lastErr = err
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(transientTxRetryDelay):
		}
	}
	return lastErr
}

func (r *GormRepo) AppendEvent(ctx context.Context, orderID int64, eventType string, dataJSON string) (domain.OrderEvent, error) {
	var lastErr error
	for attempt := 0; attempt < maxEventAppendAttempts; attempt++ {
		ev, err := r.appendEventOnce(ctx, orderID, eventType, dataJSON)
		if err == nil {
			return ev, nil
		}
		lastErr = err
		if !isEventAppendTransient(err) {
			return domain.OrderEvent{}, err
		}
		select {
		case <-ctx.Done():
			return domain.OrderEvent{}, ctx.Err()
		case <-time.After(eventAppendRetryDelay):
		}
	}
	return domain.OrderEvent{}, lastErr
}

func (r *GormRepo) appendEventOnce(ctx context.Context, orderID int64, eventType string, dataJSON string) (ev domain.OrderEvent, err error) {
	tx := r.gdb.WithContext(ctx).Begin()
	if tx.Error != nil {
		return domain.OrderEvent{}, tx.Error
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback().Error
		}
	}()
	var seq int64
	if err := tx.Model(&orderEventRow{}).Where("order_id = ?", orderID).Select("COALESCE(MAX(seq),0)").Take(&seq).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.OrderEvent{}, err
	}
	seq++
	row := orderEventRow{OrderID: orderID, Seq: seq, Type: eventType, DataJSON: dataJSON}
	if err := tx.Create(&row).Error; err != nil {
		return domain.OrderEvent{}, err
	}
	if err := tx.Commit().Error; err != nil {
		return domain.OrderEvent{}, err
	}
	return domain.OrderEvent{ID: row.ID, OrderID: row.OrderID, Seq: row.Seq, Type: row.Type, DataJSON: row.DataJSON, CreatedAt: row.CreatedAt}, nil
}

// isEventAppendTransient reports whether a failed event append is safe to retry:
// a concurrent append either raced on MAX(seq) (unique-constraint violation on
// MySQL) or collided with another transaction's write lock (SQLITE_BUSY,
// MySQL deadlock / lock-wait timeout).
func isEventAppendTransient(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate entry") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "constraint failed") ||
		strings.Contains(msg, "database is locked") ||
		strings.Contains(msg, "database table is locked") ||
		strings.Contains(msg, "deadlock") ||
		strings.Contains(msg, "lock wait timeout")
}

func (r *GormRepo) ListEventsAfter(ctx context.Context, orderID int64, afterSeq int64, limit int) ([]domain.OrderEvent, error) {

	var rows []orderEventRow
	if err := r.gdb.WithContext(ctx).
		Where("order_id = ? AND seq > ?", orderID, afterSeq).
		Order("seq ASC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.OrderEvent, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.OrderEvent{ID: row.ID, OrderID: row.OrderID, Seq: row.Seq, Type: row.Type, DataJSON: row.DataJSON, CreatedAt: row.CreatedAt})
	}
	return out, nil

}

// CountEventsByOrder 返回订单事件总数（web_api_contract F4：详情只返回最近
// 200 条，前端用总数提示"仅显示最近 N 条"）。
func (r *GormRepo) CountEventsByOrder(ctx context.Context, orderID int64) (int64, error) {
	var total int64
	if err := r.gdb.WithContext(ctx).Model(&orderEventRow{}).Where("order_id = ?", orderID).Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil

}
