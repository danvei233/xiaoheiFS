package repo

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"xiaoheiplay/internal/domain"
)

func (r *GormRepo) AppendEvent(ctx context.Context, orderID int64, eventType string, dataJSON string) (domain.OrderEvent, error) {

	tx := r.gdb.WithContext(ctx).Begin()
	if tx.Error != nil {
		return domain.OrderEvent{}, tx.Error
	}
	var seq int64
	if err := tx.Model(&orderEventRow{}).Where("order_id = ?", orderID).Select("COALESCE(MAX(seq),0)").Take(&seq).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		_ = tx.Rollback().Error
		return domain.OrderEvent{}, err
	}
	seq++
	row := orderEventRow{OrderID: orderID, Seq: seq, Type: eventType, DataJSON: dataJSON}
	if err := tx.Create(&row).Error; err != nil {
		_ = tx.Rollback().Error
		return domain.OrderEvent{}, err
	}
	if err := tx.Commit().Error; err != nil {
		return domain.OrderEvent{}, err
	}
	return domain.OrderEvent{ID: row.ID, OrderID: row.OrderID, Seq: row.Seq, Type: row.Type, DataJSON: row.DataJSON, CreatedAt: row.CreatedAt}, nil

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
