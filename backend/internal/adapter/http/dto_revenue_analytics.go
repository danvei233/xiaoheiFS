package http

import (
	"fmt"
	"strings"
	"time"

	appreport "xiaoheiplay/internal/app/report"
)

type revenueAnalyticsQueryDTO struct {
	FromAt      string `json:"from_at" binding:"required"`
	ToAt        string `json:"to_at" binding:"required"`
	Level       string `json:"level" binding:"required,oneof=overall goods_type region line package"`
	GoodsTypeID int64  `json:"goods_type_id"`
	RegionID    int64  `json:"region_id"`
	LineID      int64  `json:"line_id"`
	PackageID   int64  `json:"package_id"`
	Page        int    `json:"page"`
	PageSize    int    `json:"page_size"`
	SortField   string `json:"sort_field"`
	SortOrder   string `json:"sort_order"`
}

func (q revenueAnalyticsQueryDTO) toReportQuery() (appreport.RevenueAnalyticsQuery, error) {
	fromAt, err := parseQueryTime(q.FromAt)
	if err != nil {
		return appreport.RevenueAnalyticsQuery{}, fmt.Errorf("invalid from_at")
	}
	toAt, err := parseQueryTime(q.ToAt)
	if err != nil {
		return appreport.RevenueAnalyticsQuery{}, fmt.Errorf("invalid to_at")
	}
	level := appreport.RevenueAnalyticsLevel(strings.TrimSpace(q.Level))
	sortField := strings.TrimSpace(q.SortField)
	if sortField == "" {
		sortField = "paid_at"
	}
	sortOrder := strings.TrimSpace(strings.ToLower(q.SortOrder))
	if sortOrder == "" {
		sortOrder = "desc"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		return appreport.RevenueAnalyticsQuery{}, fmt.Errorf("invalid sort_order")
	}
	return appreport.RevenueAnalyticsQuery{
		FromAt:      fromAt,
		ToAt:        toAt,
		Level:       level,
		GoodsTypeID: q.GoodsTypeID,
		RegionID:    q.RegionID,
		LineID:      q.LineID,
		PackageID:   q.PackageID,
		Page:        q.Page,
		PageSize:    q.PageSize,
		SortField:   sortField,
		SortOrder:   sortOrder,
	}, nil
}

func parseQueryTime(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, fmt.Errorf("empty")
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02", raw); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("unsupported time format")
}
