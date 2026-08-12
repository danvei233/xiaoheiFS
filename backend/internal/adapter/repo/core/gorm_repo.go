package repo

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

type GormRepo struct {
	db  *sql.DB
	gdb *gorm.DB
}

func NewGormRepo(gdb *gorm.DB) *GormRepo {
	if gdb == nil {
		return &GormRepo{}
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		// A broken *gorm.DB (e.g. already-closed pool) has no usable *sql.DB.
		// Keep the handle so raw SQL paths fail with a typed error instead of
		// dereferencing a nil r.db.
		return &GormRepo{gdb: gdb}
	}
	return &GormRepo{db: sqlDB, gdb: gdb}
}

func nullIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

// escapeLikePattern escapes LIKE wildcards so user-supplied search keywords are
// matched literally. Callers wrap the result with '%' on both sides and must
// pair the query with likeEscapeClause.
func escapeLikePattern(keyword string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(keyword)
}

// likeEscapeClause returns a dialect-correct ESCAPE clause for LIKE queries.
// MySQL string literals treat a backslash as an escape character, so the clause
// must double it (`ESCAPE '\\'` in SQL source) to denote a single backslash;
// SQLite and PostgreSQL accept a single backslash. Using the wrong form is a
// syntax error on MySQL.
func likeEscapeClause(db *gorm.DB) string {
	if db != nil && db.Dialector != nil && db.Dialector.Name() == "mysql" {
		return " ESCAPE '\\\\'"
	}
	return " ESCAPE '\\'"
}

func isReadyVPSStatus(status domain.VPSStatus) bool {
	switch status {
	case domain.VPSStatusRunning, domain.VPSStatusStopped, domain.VPSStatusRescue, domain.VPSStatusLocked, domain.VPSStatusExpiredLocked:
		return true
	default:
		return false
	}
}

func isFailedVPSStatus(status domain.VPSStatus) bool {
	return status == domain.VPSStatusReinstallFailed
}

func recomputeOrderStatusByItemsGorm(ctx context.Context, tx *gorm.DB, orderID int64) error {
	var order orderRow
	if err := tx.WithContext(ctx).Where("id = ?", orderID).First(&order).Error; err != nil {
		return err
	}
	switch order.Status {
	case string(domain.OrderStatusApproved), string(domain.OrderStatusProvisioning), string(domain.OrderStatusActive), string(domain.OrderStatusFailed):
	default:
		return nil
	}

	var activeCount, failedCount, pendingCount int64
	if err := tx.WithContext(ctx).Model(&orderItemRow{}).Where("order_id = ? AND status = ?", orderID, domain.OrderItemStatusActive).Count(&activeCount).Error; err != nil {
		return err
	}
	if err := tx.WithContext(ctx).Model(&orderItemRow{}).Where("order_id = ? AND status = ?", orderID, domain.OrderItemStatusFailed).Count(&failedCount).Error; err != nil {
		return err
	}
	if err := tx.WithContext(ctx).Model(&orderItemRow{}).
		Where("order_id = ? AND status NOT IN ?", orderID, []string{
			string(domain.OrderItemStatusActive),
			string(domain.OrderItemStatusFailed),
			string(domain.OrderItemStatusCanceled),
			string(domain.OrderItemStatusRejected),
		}).Count(&pendingCount).Error; err != nil {
		return err
	}

	next := order.Status
	switch {
	case failedCount > 0:
		next = string(domain.OrderStatusFailed)
	case pendingCount > 0:
		next = string(domain.OrderStatusProvisioning)
	case activeCount > 0:
		next = string(domain.OrderStatusActive)
	}
	if next == order.Status {
		return nil
	}
	return tx.WithContext(ctx).Model(&orderRow{}).Where("id = ?", orderID).Updates(map[string]any{
		"status":     next,
		"updated_at": time.Now(),
	}).Error
}

func (r *GormRepo) ensure(err error) error {
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, gorm.ErrRecordNotFound) {
		return appshared.ErrNotFound
	}
	return err
}

func rEnsure(err error) error {
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, gorm.ErrRecordNotFound) {
		return appshared.ErrNotFound
	}
	return err
}
