package repo

import (
	"context"
	"encoding/json"
	"gorm.io/gorm"
	"time"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

func (r *GormRepo) CreateOrder(ctx context.Context, order *domain.Order) error {

	row := toOrderRow(*order)
	if err := r.gdb.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	*order = fromOrderRow(row)
	return nil

}

func (r *GormRepo) CreateOrderFromCartAtomic(ctx context.Context, order domain.Order, items []domain.OrderItem) (created domain.Order, createdItems []domain.OrderItem, err error) {

	tx := r.gdb.WithContext(ctx).Begin()
	if tx.Error != nil {
		return domain.Order{}, nil, tx.Error
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback().Error
		}
	}()

	or := toOrderRow(order)
	if err = tx.Create(&or).Error; err != nil {
		return domain.Order{}, nil, err
	}
	order = fromOrderRow(or)

	itemRows := make([]orderItemRow, 0, len(items))
	for i := range items {
		items[i].OrderID = order.ID
		itemRows = append(itemRows, toOrderItemRow(items[i]))
	}
	if len(itemRows) > 0 {
		if err = tx.Create(&itemRows).Error; err != nil {
			return domain.Order{}, nil, err
		}
		createdItems = make([]domain.OrderItem, 0, len(itemRows))
		for _, row := range itemRows {
			createdItems = append(createdItems, fromOrderItemRow(row))
		}
	}

	if err = tx.Where("user_id = ?", order.UserID).Delete(&cartItemRow{}).Error; err != nil {
		return domain.Order{}, nil, err
	}
	if err = tx.Commit().Error; err != nil {
		return domain.Order{}, nil, err
	}
	return order, createdItems, nil

}

func (r *GormRepo) GetOrder(ctx context.Context, id int64) (domain.Order, error) {

	var row orderRow
	if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return domain.Order{}, r.ensure(err)
	}
	return fromOrderRow(row), nil

}

func (r *GormRepo) GetOrderByNo(ctx context.Context, orderNo string) (domain.Order, error) {

	var row orderRow
	if err := r.gdb.WithContext(ctx).Where("order_no = ?", orderNo).First(&row).Error; err != nil {
		return domain.Order{}, r.ensure(err)
	}
	return fromOrderRow(row), nil

}

func (r *GormRepo) GetOrderByIdempotencyKey(ctx context.Context, userID int64, key string) (domain.Order, error) {

	var row orderRow
	if err := r.gdb.WithContext(ctx).Where("user_id = ? AND idempotency_key = ?", userID, key).First(&row).Error; err != nil {
		return domain.Order{}, r.ensure(err)
	}
	return fromOrderRow(row), nil

}

func (r *GormRepo) UpdateOrderStatus(ctx context.Context, id int64, status domain.OrderStatus) error {

	return r.gdb.WithContext(ctx).Model(&orderRow{}).Where("id = ?", id).Updates(map[string]any{
		"status":     status,
		"updated_at": time.Now(),
	}).Error

}

func (r *GormRepo) UpdateOrderMeta(ctx context.Context, order domain.Order) error {

	return r.gdb.WithContext(ctx).Model(&orderRow{}).Where("id = ?", order.ID).Updates(map[string]any{
		"status":          order.Status,
		"pending_reason":  order.PendingReason,
		"approved_by":     order.ApprovedBy,
		"approved_at":     order.ApprovedAt,
		"rejected_reason": order.RejectedReason,
		"updated_at":      time.Now(),
	}).Error

}

func (r *GormRepo) ApproveResizeOrderWithTasks(ctx context.Context, order domain.Order, items []domain.OrderItem, tasks []*domain.ResizeTask) error {

	return r.gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, task := range tasks {
			if task == nil {
				continue
			}
			var total int64
			if err := tx.Model(&resizeTaskRow{}).
				Where("vps_id = ? AND status IN ?", task.VPSID, []string{string(domain.ResizeTaskStatusPending), string(domain.ResizeTaskStatusRunning)}).
				Count(&total).Error; err != nil {
				return err
			}
			if total > 0 {
				return appshared.ErrResizeInProgress
			}
		}

		if err := tx.Model(&orderRow{}).Where("id = ?", order.ID).Updates(map[string]any{
			"status":          order.Status,
			"pending_reason":  order.PendingReason,
			"approved_by":     order.ApprovedBy,
			"approved_at":     order.ApprovedAt,
			"rejected_reason": order.RejectedReason,
			"updated_at":      time.Now(),
		}).Error; err != nil {
			return err
		}

		itemIDs := make([]int64, 0, len(items))
		for _, item := range items {
			itemIDs = append(itemIDs, item.ID)
		}
		if len(itemIDs) > 0 {
			if err := tx.Model(&orderItemRow{}).Where("id IN ?", itemIDs).Updates(map[string]any{
				"status":     domain.OrderItemStatusApproved,
				"updated_at": time.Now(),
			}).Error; err != nil {
				return err
			}
		}

		for _, task := range tasks {
			if task == nil {
				continue
			}
			row := resizeTaskRow{
				VPSID:       task.VPSID,
				OrderID:     task.OrderID,
				OrderItemID: task.OrderItemID,
				Status:      string(task.Status),
				ScheduledAt: task.ScheduledAt,
				StartedAt:   task.StartedAt,
				FinishedAt:  task.FinishedAt,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
			task.ID = row.ID
		}
		return nil
	})

}

func (r *GormRepo) ListOrders(ctx context.Context, filter appshared.OrderFilter, limit, offset int) ([]domain.Order, int, error) {

	q := r.gdb.WithContext(ctx).Model(&orderRow{})
	if filter.Status != "" {
		q = q.Where("status = ?", filter.Status)
	}
	if filter.UserID > 0 {
		q = q.Where("user_id = ?", filter.UserID)
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
	var rows []orderRow
	if err := q.Order("id DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.Order, 0, len(rows))
	for _, row := range rows {
		out = append(out, fromOrderRow(row))
	}
	return out, int(total), nil

}

func (r *GormRepo) DeleteOrder(ctx context.Context, id int64) (err error) {

	tx := r.gdb.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback().Error
		}
	}()
	var exists int64
	if err = tx.Model(&orderRow{}).Where("id = ?", id).Count(&exists).Error; err != nil {
		return err
	}
	if exists == 0 {
		return appshared.ErrNotFound
	}
	sub := tx.Model(&orderItemRow{}).Select("id").Where("order_id = ?", id)
	if err = tx.Where("order_item_id IN (?)", sub).Delete(&vpsInstanceRow{}).Error; err != nil {
		return err
	}
	if err = tx.Where("order_id = ?", id).Delete(&provisionJobRow{}).Error; err != nil {
		return err
	}
	if err = tx.Where("order_id = ?", id).Delete(&resizeTaskRow{}).Error; err != nil {
		return err
	}
	if err = tx.Where("order_id = ?", id).Delete(&automationLogRow{}).Error; err != nil {
		return err
	}
	if err = tx.Where("order_id = ?", id).Delete(&orderEventRow{}).Error; err != nil {
		return err
	}
	if err = tx.Where("order_id = ?", id).Delete(&orderPaymentRow{}).Error; err != nil {
		return err
	}
	if err = tx.Where("order_id = ?", id).Delete(&orderItemRow{}).Error; err != nil {
		return err
	}
	if err = tx.Delete(&orderRow{}, id).Error; err != nil {
		return err
	}
	return tx.Commit().Error

}

func (r *GormRepo) CreateOrderItems(ctx context.Context, items []domain.OrderItem) error {

	if len(items) == 0 {
		return nil
	}
	rows := make([]orderItemRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, toOrderItemRow(item))
	}
	if err := r.gdb.WithContext(ctx).Create(&rows).Error; err != nil {
		return err
	}
	for i := range rows {
		items[i].ID = rows[i].ID
		items[i].CreatedAt = rows[i].CreatedAt
		items[i].UpdatedAt = rows[i].UpdatedAt
	}
	return nil

}

func (r *GormRepo) ListOrderItems(ctx context.Context, orderID int64) ([]domain.OrderItem, error) {

	var rows []orderItemRow
	if err := r.gdb.WithContext(ctx).Where("order_id = ?", orderID).Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.OrderItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, fromOrderItemRow(row))
	}
	return out, nil

}

func (r *GormRepo) GetOrderItem(ctx context.Context, id int64) (domain.OrderItem, error) {

	var row orderItemRow
	if err := r.gdb.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return domain.OrderItem{}, r.ensure(err)
	}
	return fromOrderItemRow(row), nil

}

func (r *GormRepo) HasPendingRenewOrder(ctx context.Context, userID, vpsID int64) (bool, error) {
	return r.hasExclusiveVPSOrderInProgress(ctx, userID, vpsID)
}

func (r *GormRepo) HasPendingResizeOrder(ctx context.Context, userID, vpsID int64) (bool, error) {
	return r.hasExclusiveVPSOrderInProgress(ctx, userID, vpsID)
}

func (r *GormRepo) HasPendingRefundOrder(ctx context.Context, userID, vpsID int64) (bool, error) {
	return r.hasExclusiveVPSOrderInProgress(ctx, userID, vpsID)
}

func (r *GormRepo) hasExclusiveVPSOrderInProgress(ctx context.Context, userID, vpsID int64) (bool, error) {
	if vpsID <= 0 {
		return false, nil
	}
	progressStatuses := []string{
		string(domain.OrderStatusPendingPayment),
		string(domain.OrderStatusPendingReview),
		string(domain.OrderStatusApproved),
		string(domain.OrderStatusProvisioning),
	}
	actions := []string{"renew", "emergency_renew", "resize", "refund"}
	var rows []orderItemRow
	if err := r.gdb.WithContext(ctx).
		Joins("JOIN orders o ON o.id = order_items.order_id").
		Where("o.user_id = ? AND order_items.action IN ? AND o.status IN ?",
			userID, actions, progressStatuses).
		Order("order_items.id DESC").
		Limit(50).
		Select("order_items.spec_json").
		Find(&rows).Error; err != nil {
		return false, err
	}
	for _, row := range rows {
		var payload struct {
			VPSID int64 `json:"vps_id"`
		}
		if err := json.Unmarshal([]byte(row.SpecJSON), &payload); err == nil && payload.VPSID == vpsID {
			return true, nil
		}
	}
	return false, nil
}

func (r *GormRepo) UpdateOrderItemStatus(ctx context.Context, id int64, status domain.OrderItemStatus) error {

	return r.gdb.WithContext(ctx).Model(&orderItemRow{}).Where("id = ?", id).Updates(map[string]any{
		"status":     status,
		"updated_at": time.Now(),
	}).Error

}

func (r *GormRepo) UpdateOrderItemAutomation(ctx context.Context, id int64, automationID string) error {

	return r.gdb.WithContext(ctx).Model(&orderItemRow{}).Where("id = ?", id).Updates(map[string]any{
		"automation_instance_id": automationID,
		"updated_at":             time.Now(),
	}).Error

}
