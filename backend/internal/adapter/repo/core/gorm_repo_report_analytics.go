package repo

import (
	"context"
	"fmt"
	"time"
)

type revenueAnalyticsJoinRow struct {
	PaymentID    int64     `gorm:"column:payment_id"`
	OrderID      int64     `gorm:"column:order_id"`
	OrderNo      string    `gorm:"column:order_no"`
	UserID       int64     `gorm:"column:user_id"`
	Amount       int64     `gorm:"column:amount"`
	PaidAt       time.Time `gorm:"column:paid_at"`
	GoodsTypeID  int64     `gorm:"column:goods_type_id"`
	RegionID     int64     `gorm:"column:region_id"`
	LineID       int64     `gorm:"column:line_id"`
	PackageID    int64     `gorm:"column:package_id"`
	DimensionID  int64     `gorm:"column:dimension_id"`
	DimensionStr string    `gorm:"column:dimension_name"`
}

// ListRevenueAnalyticsRows provides a reusable join baseline for analytics queries.
func (r *GormRepo) ListRevenueAnalyticsRows(ctx context.Context, fromAt, toAt time.Time) ([]revenueAnalyticsJoinRow, error) {
	var rows []revenueAnalyticsJoinRow
	err := r.gdb.WithContext(ctx).
		Table("order_payments op").
		Select(`
			op.id as payment_id,
			op.order_id as order_id,
			o.order_no as order_no,
			o.user_id as user_id,
			op.amount as amount,
			op.created_at as paid_at,
			oi.goods_type_id as goods_type_id,
			pg.region_id as region_id,
			pg.line_id as line_id,
			oi.package_id as package_id,
			oi.package_id as dimension_id,
			pkg.name as dimension_name`).
		Joins("join orders o on o.id = op.order_id").
		Joins("join order_items oi on oi.order_id = o.id").
		Joins("join packages pkg on pkg.id = oi.package_id").
		Joins("join plan_groups pg on pg.id = pkg.plan_group_id").
		Where("op.status = ?", "approved").
		Where("op.created_at >= ? and op.created_at <= ?", fromAt, toAt).
		Find(&rows).Error
	return rows, err
}

func (r *GormRepo) ListRevenueAnalyticsDetails(
	ctx context.Context,
	fromAt, toAt time.Time,
	sortField string,
	sortOrder string,
	limit int,
	offset int,
) ([]revenueAnalyticsJoinRow, int, error) {
	q := r.gdb.WithContext(ctx).
		Table("order_payments op").
		Select(`
			op.id as payment_id,
			op.order_id as order_id,
			o.order_no as order_no,
			o.user_id as user_id,
			op.amount as amount,
			op.created_at as paid_at,
			oi.goods_type_id as goods_type_id,
			pg.region_id as region_id,
			pg.line_id as line_id,
			oi.package_id as package_id,
			oi.package_id as dimension_id,
			pkg.name as dimension_name`).
		Joins("join orders o on o.id = op.order_id").
		Joins("join order_items oi on oi.order_id = o.id").
		Joins("join packages pkg on pkg.id = oi.package_id").
		Joins("join plan_groups pg on pg.id = pkg.plan_group_id").
		Where("op.status = ?", "approved").
		Where("op.created_at >= ? and op.created_at <= ?", fromAt, toAt)

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	orderClause := "op.created_at desc"
	if sortField == "amount" {
		orderClause = "op.amount"
	} else {
		orderClause = "op.created_at"
	}
	if sortOrder == "asc" {
		orderClause += " asc"
	} else {
		orderClause += " desc"
	}
	if limit <= 0 {
		limit = 20
	}
	var rows []revenueAnalyticsJoinRow
	if err := q.Order(orderClause).Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list revenue analytics details: %w", err)
	}
	return rows, int(total), nil
}
