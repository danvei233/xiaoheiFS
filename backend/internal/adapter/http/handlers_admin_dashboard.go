package http

import (
	"encoding/csv"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"strings"
	"time"
	"xiaoheiplay/internal/domain"
)

func (h *Handler) AdminDashboardOverview(c *gin.Context) {
	if h.reportSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	overview, err := h.reportSvc.Overview(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrReportError.Error()})
		return
	}
	c.JSON(http.StatusOK, overview)
}

func (h *Handler) AdminDashboardRevenue(c *gin.Context) {
	if h.reportSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	period := c.Query("period")
	if period == "month" {
		points, err := h.reportSvc.RevenueByMonth(c, 6)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrReportError.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": points})
		return
	}
	points, err := h.reportSvc.RevenueByDay(c, 30)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrReportError.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": points})
}

func (h *Handler) AdminDashboardVPSStatus(c *gin.Context) {
	if h.reportSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	items, err := h.reportSvc.VPSStatus(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrReportError.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) parseRevenueAnalyticsQuery(c *gin.Context) (revenueAnalyticsQueryDTO, bool) {
	var req revenueAnalyticsQueryDTO
	if err := bindJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return req, false
	}
	return req, true
}

func (h *Handler) AdminRevenueAnalyticsOverview(c *gin.Context) {
	if h.reportSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	req, ok := h.parseRevenueAnalyticsQuery(c)
	if !ok {
		return
	}
	query, err := req.toReportQuery()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	overview, err := h.reportSvc.RevenueAnalyticsOverview(c, query)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.auditRevenueQuery(c, "overview", req)
	c.JSON(http.StatusOK, overview)
}

func (h *Handler) AdminRevenueAnalyticsTrend(c *gin.Context) {
	if h.reportSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	req, ok := h.parseRevenueAnalyticsQuery(c)
	if !ok {
		return
	}
	query, err := req.toReportQuery()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	items, err := h.reportSvc.RevenueAnalyticsTrend(c, query)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.auditRevenueQuery(c, "trend", req)
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) AdminRevenueAnalyticsTop(c *gin.Context) {
	if h.reportSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	req, ok := h.parseRevenueAnalyticsQuery(c)
	if !ok {
		return
	}
	query, err := req.toReportQuery()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	items, err := h.reportSvc.RevenueAnalyticsTop(c, query)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.auditRevenueQuery(c, "top", req)
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) AdminRevenueAnalyticsDetails(c *gin.Context) {
	if h.reportSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	req, ok := h.parseRevenueAnalyticsQuery(c)
	if !ok {
		return
	}
	query, err := req.toReportQuery()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	items, total, err := h.reportSvc.RevenueAnalyticsDetails(c, query)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.auditRevenueQuery(c, "details", req)
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	c.JSON(http.StatusOK, gin.H{
		"items":      items,
		"page":       page,
		"page_size":  pageSize,
		"total":      total,
		"queried_at": time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *Handler) AdminRevenueAnalyticsExport(c *gin.Context) {
	if h.reportSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	req, ok := h.parseRevenueAnalyticsQuery(c)
	if !ok {
		return
	}
	query, err := req.toReportQuery()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	query.SortField = "paid_at"
	query.SortOrder = "asc"
	query.Page = 1
	query.PageSize = 200

	overview, err := h.reportSvc.RevenueAnalyticsOverview(c, query)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fileName := fmt.Sprintf("revenue_analytics_audit_%s.csv", time.Now().Format("20060102_150405"))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusOK)
	if _, err := c.Writer.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return
	}

	w := csv.NewWriter(c.Writer)
	write := func(record ...string) bool {
		if err := w.Write(record); err != nil {
			return false
		}
		return true
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if !write("section", "field", "value") {
		return
	}
	metaRows := [][]string{
		{"meta", "exported_at_utc", now},
		{"meta", "from_at", req.FromAt},
		{"meta", "to_at", req.ToAt},
		{"meta", "level", req.Level},
		{"meta", "user_id", strconv.FormatInt(req.UserID, 10)},
		{"meta", "goods_type_id", strconv.FormatInt(req.GoodsTypeID, 10)},
		{"meta", "region_id", strconv.FormatInt(req.RegionID, 10)},
		{"meta", "line_id", strconv.FormatInt(req.LineID, 10)},
		{"meta", "package_id", strconv.FormatInt(req.PackageID, 10)},
		{"summary", "total_revenue_cents", strconv.FormatInt(overview.Summary.TotalRevenueCents, 10)},
		{"summary", "order_count", strconv.Itoa(overview.Summary.OrderCount)},
		{"summary", "yoy_ratio", formatRatioCSV(overview.Summary.YoYRatio, overview.Summary.YoYComparable)},
		{"summary", "mom_ratio", formatRatioCSV(overview.Summary.MoMRatio, overview.Summary.MoMComparable)},
	}
	for _, row := range metaRows {
		if !write(row...) {
			return
		}
	}
	if !write("") {
		return
	}
	if !write(
		"payment_id",
		"order_id",
		"order_no",
		"user_id",
		"amount_cents",
		"amount_yuan",
		"status",
		"paid_at",
		"goods_type_id",
		"region_id",
		"line_id",
		"package_id",
	) {
		return
	}

	page := 1
	for {
		query.Page = page
		items, total, err := h.reportSvc.RevenueAnalyticsDetails(c, query)
		if err != nil {
			_ = w.Write([]string{"error", "query_failed", err.Error()})
			w.Flush()
			return
		}
		for _, item := range items {
			amountYuan := fmt.Sprintf("%.2f", float64(item.AmountCents)/100.0)
			if !write(
				strconv.FormatInt(item.PaymentID, 10),
				strconv.FormatInt(item.OrderID, 10),
				item.OrderNo,
				strconv.FormatInt(item.UserID, 10),
				strconv.FormatInt(item.AmountCents, 10),
				amountYuan,
				item.Status,
				item.PaidAt.Format(time.RFC3339),
				strconv.FormatInt(item.GoodsTypeID, 10),
				strconv.FormatInt(item.RegionID, 10),
				strconv.FormatInt(item.LineID, 10),
				strconv.FormatInt(item.PackageID, 10),
			) {
				return
			}
		}
		w.Flush()
		if len(items) == 0 || page*query.PageSize >= total {
			break
		}
		page++
	}
	h.auditRevenueQuery(c, "export", req)
}

func formatRatioCSV(r *float64, comparable bool) string {
	if !comparable || r == nil {
		return "N/A"
	}
	return fmt.Sprintf("%.6f", *r)
}

func (h *Handler) auditRevenueQuery(c *gin.Context, action string, req revenueAnalyticsQueryDTO) {
	if h.adminSvc == nil {
		return
	}
	operatorID := getUserID(c)
	traceID := strings.TrimSpace(c.GetHeader("X-Trace-ID"))
	if traceID == "" {
		traceID = strings.TrimSpace(c.GetHeader("X-Request-ID"))
	}
	h.adminSvc.Audit(c, operatorID, "dashboard.revenue_analytics."+action, "dashboard_revenue_analytics", action, map[string]any{
		"operator_id":   operatorID,
		"request_path":  c.FullPath(),
		"from_at":       req.FromAt,
		"to_at":         req.ToAt,
		"level":         req.Level,
		"user_id":       req.UserID,
		"goods_type_id": req.GoodsTypeID,
		"region_id":     req.RegionID,
		"line_id":       req.LineID,
		"package_id":    req.PackageID,
		"trace_id":      traceID,
		"filter_summary": map[string]any{
			"level":         req.Level,
			"user_id":       req.UserID,
			"goods_type_id": req.GoodsTypeID,
			"region_id":     req.RegionID,
			"line_id":       req.LineID,
			"package_id":    req.PackageID,
		},
	})
}
