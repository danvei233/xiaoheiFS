package repo

import (
	"context"

	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

const provisionClaimTimeout = 30 * time.Minute

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

// ApproveOrderAtomic claims an approvable order and updates its items and resize tasks in one transaction.
func (r *GormRepo) ApproveOrderAtomic(ctx context.Context, order domain.Order, itemIDs []int64, tasks []*domain.ResizeTask, currentStatuses []domain.OrderStatus) (claimed bool, err error) {
	err = r.withTransientTxRetry(ctx, func(tx *gorm.DB) error {
		var approveErr error
		claimed, approveErr = approveOrderTx(tx, order, itemIDs, tasks, currentStatuses)
		return approveErr
	})
	if err != nil {
		return false, err
	}
	return claimed, nil
}

func approveOrderTx(tx *gorm.DB, order domain.Order, itemIDs []int64, tasks []*domain.ResizeTask, currentStatuses []domain.OrderStatus) (bool, error) {
	statuses := make([]string, 0, len(currentStatuses))
	for _, status := range currentStatuses {
		statuses = append(statuses, string(status))
	}
	if len(statuses) == 0 {
		return false, nil
	}
	now := time.Now()
	result := tx.Model(&orderRow{}).
		Where("id = ? AND status IN ?", order.ID, statuses).
		Updates(map[string]any{
			"status":          order.Status,
			"pending_reason":  order.PendingReason,
			"approved_by":     order.ApprovedBy,
			"approved_at":     order.ApprovedAt,
			"rejected_reason": order.RejectedReason,
			"updated_at":      now,
		})
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected != 1 {
		return false, nil
	}

	for _, task := range tasks {
		if task == nil {
			continue
		}
		var instance vpsInstanceRow
		if err := tx.Select("id").Clauses(clause.Locking{Strength: "UPDATE"}).First(&instance, task.VPSID).Error; err != nil {
			return false, err
		}
		var total int64
		if err := tx.Model(&resizeTaskRow{}).
			Where("vps_id = ? AND status IN ?", task.VPSID, []string{string(domain.ResizeTaskStatusPending), string(domain.ResizeTaskStatusRunning)}).
			Count(&total).Error; err != nil {
			return false, err
		}
		if total > 0 {
			return false, appshared.ErrResizeInProgress
		}
	}

	if len(itemIDs) > 0 {
		if err := tx.Model(&orderItemRow{}).
			Where("order_id = ? AND id IN ?", order.ID, itemIDs).
			Updates(map[string]any{"status": domain.OrderItemStatusApproved, "updated_at": now}).Error; err != nil {
			return false, err
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
			return false, err
		}
		task.ID = row.ID
	}
	paymentUpdates := map[string]any{
		"status":        domain.PaymentStatusApproved,
		"review_reason": "",
		"updated_at":    now,
	}
	if order.ApprovedBy != nil {
		paymentUpdates["reviewed_by"] = *order.ApprovedBy
	}
	// Only approve payments that were awaiting admin review. An abandoned
	// provider payment in pending_payment never received funds and must not be
	// counted as paid when the order is approved (revenue/ledger integrity).
	// Additionally enforce the "single approved payment per order" invariant:
	// when the order is already settled by another approved payment (e.g. a
	// balance payment), a pending_review provider/manual payment must be
	// rejected instead of approved — otherwise the user is double-charged.
	var approvedCount int64
	if err := tx.Model(&orderPaymentRow{}).
		Where("order_id = ? AND status = ?", order.ID, string(domain.PaymentStatusApproved)).
		Count(&approvedCount).Error; err != nil {
		return false, err
	}
	if approvedCount == 0 {
		if err := tx.Model(&orderPaymentRow{}).
			Where("order_id = ? AND status = ?", order.ID, string(domain.PaymentStatusPendingReview)).
			Updates(paymentUpdates).Error; err != nil {
			return false, err
		}
	} else {
		if err := tx.Model(&orderPaymentRow{}).
			Where("order_id = ? AND status = ?", order.ID, string(domain.PaymentStatusPendingReview)).
			Updates(map[string]any{
				"status":        domain.PaymentStatusRejected,
				"review_reason": "order already settled by another payment",
				"updated_at":    now,
			}).Error; err != nil {
			return false, err
		}
	}
	return true, nil
}

func (r *GormRepo) RejectOrderAtomic(ctx context.Context, order domain.Order, adminID int64, currentStatuses []domain.OrderStatus) (claimed bool, err error) {
	statuses := make([]string, 0, len(currentStatuses))
	for _, status := range currentStatuses {
		statuses = append(statuses, string(status))
	}
	if len(statuses) == 0 {
		return false, nil
	}
	err = r.withTransientTxRetry(ctx, func(tx *gorm.DB) error {
		now := time.Now()
		// An approved PROVIDER payment means real external money has arrived;
		// rejecting such an order without an external refund would strand it. A
		// balance payment is a local wallet debit whose refund the service layer
		// issues idempotently (balance_refund:<orderID>), so it does not block
		// rejection.
		approvedExternalPayment := tx.Model(&orderPaymentRow{}).
			Select("1").
			Where("order_payments.order_id = orders.id AND order_payments.status = ? AND order_payments.method <> ?",
				string(domain.PaymentStatusApproved), "balance")
		result := tx.Model(&orderRow{}).
			Where("id = ? AND status IN ?", order.ID, statuses).
			Where("NOT EXISTS (?)", approvedExternalPayment).
			Updates(map[string]any{
				"status":          domain.OrderStatusRejected,
				"pending_reason":  "",
				"rejected_reason": order.RejectedReason,
				"updated_at":      now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return nil
		}
		if err := tx.Model(&orderItemRow{}).
			Where("order_id = ?", order.ID).
			Updates(map[string]any{"status": domain.OrderItemStatusRejected, "updated_at": now}).Error; err != nil {
			return err
		}
		if err := tx.Model(&orderPaymentRow{}).
			Where("order_id = ?", order.ID).
			Updates(map[string]any{
				"status":        domain.PaymentStatusRejected,
				"reviewed_by":   adminID,
				"review_reason": order.RejectedReason,
				"updated_at":    now,
			}).Error; err != nil {
			return err
		}
		claimed = true
		return nil
	})
	return claimed, err
}

// CancelOrderAtomic atomically transitions a cancellable order (pending
// payment/review) to canceled and rejects every still-pending payment in the
// same transaction, guarded by "no approved external payment may exist".
//
// The guard and the payment rejection must share the order CAS transaction, or
// a concurrent provider callback can approve a payment in the window between
// CancelOrder's status read and its payment write: the order ends canceled with
// real external money confirmed and no automatic refund (finance F1). By doing
// the order CAS, the NOT EXISTS(approved external payment) check and the
// payment rejection in one transaction:
//   - if the callback commits first, the guard sees the approved payment and the
//     cancel is refused (claimed=false) — the user goes through the refund flow;
//   - if the cancel commits first, every pending payment is rejected in the same
//     transaction, so the callback's approval CAS (which only matches
//     pending_payment/pending_review) touches nothing and the callback acks.
//
// A balance payment is a local wallet debit whose refund the service layer
// issues idempotently (balance_refund:<orderID>), so it does not block cancel.
func (r *GormRepo) CancelOrderAtomic(ctx context.Context, order domain.Order, currentStatuses []domain.OrderStatus) (claimed bool, err error) {
	statuses := make([]string, 0, len(currentStatuses))
	for _, status := range currentStatuses {
		statuses = append(statuses, string(status))
	}
	if len(statuses) == 0 {
		return false, nil
	}
	err = r.withTransientTxRetry(ctx, func(tx *gorm.DB) error {
		now := time.Now()
		// An approved PROVIDER payment means real external money has arrived;
		// canceling such an order without an external refund would strand it.
		approvedExternalPayment := tx.Model(&orderPaymentRow{}).
			Select("1").
			Where("order_payments.order_id = orders.id AND order_payments.status = ? AND order_payments.method <> ?",
				string(domain.PaymentStatusApproved), "balance")
		result := tx.Model(&orderRow{}).
			Where("id = ? AND status IN ?", order.ID, statuses).
			Where("NOT EXISTS (?)", approvedExternalPayment).
			Updates(map[string]any{
				"status":     domain.OrderStatusCanceled,
				"updated_at": now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return nil
		}
		if err := tx.Model(&orderItemRow{}).
			Where("order_id = ?", order.ID).
			Updates(map[string]any{"status": domain.OrderItemStatusCanceled, "updated_at": now}).Error; err != nil {
			return err
		}
		// Reject only payments that are still pending. A payment already approved
		// by a concurrent callback would have tripped the NOT EXISTS guard
		// above, so touching only pending rows here is both correct and safe.
		if err := tx.Model(&orderPaymentRow{}).
			Where("order_id = ? AND status IN ?", order.ID,
				[]string{string(domain.PaymentStatusPendingPayment), string(domain.PaymentStatusPendingReview)}).
			Updates(map[string]any{
				"status":        domain.PaymentStatusRejected,
				"review_reason": "order canceled",
				"updated_at":    now,
			}).Error; err != nil {
			return err
		}
		claimed = true
		return nil
	})
	return claimed, err
}

func (r *GormRepo) UpdateOrderStatusIfCurrent(ctx context.Context, id int64, status domain.OrderStatus, currentStatuses []domain.OrderStatus) (bool, error) {
	statuses := make([]string, 0, len(currentStatuses))
	for _, current := range currentStatuses {
		statuses = append(statuses, string(current))
	}
	if len(statuses) == 0 {
		return false, nil
	}
	result := r.gdb.WithContext(ctx).Model(&orderRow{}).
		Where("id = ? AND status IN ?", id, statuses).
		Updates(map[string]any{"status": status, "updated_at": time.Now()})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}

func (r *GormRepo) ListOrders(ctx context.Context, filter appshared.OrderFilter, limit, offset int) ([]domain.Order, int, error) {

	q := r.gdb.WithContext(ctx).Model(&orderRow{})
	if filter.Status != "" {
		q = q.Where("status = ?", filter.Status)
	}
	if filter.UserID > 0 {
		q = q.Where("user_id = ?", filter.UserID)
	}
	if orderNo := strings.TrimSpace(filter.OrderNo); orderNo != "" {
		// 审核页按订单号搜索（order_no），子串匹配（m059 F6）。
		q = q.Where("order_no LIKE ?", "%"+orderNo+"%")
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
	// Refuse to hard-delete an order whose money has moved or whose
	// infrastructure still exists: deleting would silently erase a wallet
	// debit (balance payment, no refund) or orphan a running VPS instance.
	var approvedPayments int64
	if err = tx.Model(&orderPaymentRow{}).
		Where("order_id = ? AND status = ?", id, string(domain.PaymentStatusApproved)).
		Count(&approvedPayments).Error; err != nil {
		return err
	}
	if approvedPayments > 0 {
		return appshared.ErrConflict
	}
	// Refuse to hard-delete an order that still carries an in-flight provider
	// payment session. The session means a real external payment can arrive at
	// any moment; deleting the row would strand the callback (money arrives with
	// no record, no refund, no reconciliation). pending_review manual-proof
	// payments have no external session and remain deletable.
	var pendingSessions int64
	if err = tx.Model(&orderPaymentRow{}).
		Where("order_id = ? AND status = ?", id, string(domain.PaymentStatusPendingPayment)).
		Count(&pendingSessions).Error; err != nil {
		return err
	}
	if pendingSessions > 0 {
		return appshared.ErrConflict
	}
	var walletDebits int64
	if err = tx.Model(&walletTransactionRow{}).
		Where("ref_type = ? AND ref_id = ?", "order", id).
		Count(&walletDebits).Error; err != nil {
		return err
	}
	if walletDebits > 0 {
		return appshared.ErrConflict
	}
	sub := tx.Model(&orderItemRow{}).Select("id").Where("order_id = ?", id)
	var linkedInstances int64
	if err = tx.Model(&vpsInstanceRow{}).
		Where("order_item_id IN (?)", sub).
		Count(&linkedInstances).Error; err != nil {
		return err
	}
	if linkedInstances > 0 {
		return appshared.ErrConflict
	}
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
	// The vps_id link is normalized into an indexed column at item-creation
	// time, so this check is a bounded indexed scan — no JSON parsing and no
	// row limit that could miss an older pending operation.
	progressStatuses := []string{
		string(domain.OrderStatusPendingPayment),
		string(domain.OrderStatusPendingReview),
		string(domain.OrderStatusApproved),
		string(domain.OrderStatusProvisioning),
	}
	actions := []string{"renew", "emergency_renew", "resize", "refund"}
	var total int64
	if err := r.gdb.WithContext(ctx).
		Model(&orderItemRow{}).
		Joins("JOIN orders o ON o.id = order_items.order_id").
		Where("o.user_id = ? AND order_items.vps_id = ? AND order_items.action IN ? AND o.status IN ?",
			userID, vpsID, actions, progressStatuses).
		Count(&total).Error; err != nil {
		return false, err
	}
	return total > 0, nil
}

// AcquireVPSLifecycleLock reserves a VPS for exactly one exclusive lifecycle
// operation. The unique index on vps_id is the CAS: a second concurrent order
// for the same VPS fails atomically instead of both passing a check-then-insert.
func (r *GormRepo) AcquireVPSLifecycleLock(ctx context.Context, vpsID, orderID int64, action string) error {
	if vpsID <= 0 {
		return domain.ErrInvalidInput
	}
	err := r.gdb.WithContext(ctx).Create(&vpsLifecycleLockRow{VPSID: vpsID, OrderID: orderID, Action: action}).Error
	if err != nil {
		if isUniqueConstraintError(err) {
			return appshared.ErrConflict
		}
		return err
	}
	return nil
}

// ReleaseVPSLifecycleLocksForOrder releases all reservations held by an order.
func (r *GormRepo) ReleaseVPSLifecycleLocksForOrder(ctx context.Context, orderID int64) error {
	return r.gdb.WithContext(ctx).Where("order_id = ?", orderID).Delete(&vpsLifecycleLockRow{}).Error
}

// ReleaseVPSLifecycleLock releases a single reservation by its owning order.
func (r *GormRepo) ReleaseVPSLifecycleLock(ctx context.Context, orderID, vpsID int64) error {
	return r.gdb.WithContext(ctx).Where("order_id = ? AND vps_id = ?", orderID, vpsID).Delete(&vpsLifecycleLockRow{}).Error
}

// BindVPSLifecycleLock sets the owning order on a reservation created before the
// order row existed.
func (r *GormRepo) BindVPSLifecycleLock(ctx context.Context, vpsID, orderID int64) error {
	return r.gdb.WithContext(ctx).Model(&vpsLifecycleLockRow{}).
		Where("vps_id = ?", vpsID).
		Updates(map[string]any{"order_id": orderID, "updated_at": time.Now()}).Error
}

// ReleaseVPSLifecycleLockByVPS releases a reservation by VPS (used to undo a
// reservation when order creation fails before an order id exists).
func (r *GormRepo) ReleaseVPSLifecycleLockByVPS(ctx context.Context, vpsID int64) error {
	return r.gdb.WithContext(ctx).Where("vps_id = ?", vpsID).Delete(&vpsLifecycleLockRow{}).Error
}

func isUniqueConstraintError(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	// The glebarez SQLite driver surfaces unique violations as a raw SQL error
	// rather than gorm.ErrDuplicatedKey; match the message as a fallback.
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate entry") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "unique index") ||
		strings.Contains(msg, "duplicated key") ||
		strings.Contains(msg, "already exists")
}

func (r *GormRepo) UpdateOrderItemStatus(ctx context.Context, id int64, status domain.OrderItemStatus) error {

	return r.gdb.WithContext(ctx).Model(&orderItemRow{}).Where("id = ?", id).Updates(map[string]any{
		"status":     status,
		"updated_at": time.Now(),
	}).Error

}

// ClaimOrderItemForProvision atomically moves an approved or failed item into
// provisioning, or reclaims a provisioning item whose claim has gone stale
// (updated_at predates provisionClaimTimeout, i.e. a crashed worker). A fresh
// provisioning item is never stolen: the `automation_instance_id LIKE
// 'creating:%'` shortcut that let a second worker immediately re-claim an
// in-flight create is gone. Create items are now exclusively driven by the
// provision job lease (owner+fence), which has proper expiry semantics.
func (r *GormRepo) ClaimOrderItemForProvision(ctx context.Context, id int64) (bool, error) {
	staleBefore := time.Now().Add(-provisionClaimTimeout)
	result := r.gdb.WithContext(ctx).Model(&orderItemRow{}).
		Where("id = ? AND (status IN ? OR (status = ? AND updated_at < ?))", id,
			[]string{string(domain.OrderItemStatusApproved), string(domain.OrderItemStatusFailed)},
			string(domain.OrderItemStatusProvisioning), staleBefore).
		Updates(map[string]any{
			"status":     domain.OrderItemStatusProvisioning,
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}

func (r *GormRepo) UpdateOrderItemAutomation(ctx context.Context, id int64, automationID string) error {

	return r.gdb.WithContext(ctx).Model(&orderItemRow{}).Where("id = ?", id).Updates(map[string]any{
		"automation_instance_id": automationID,
		"updated_at":             time.Now(),
	}).Error

}
