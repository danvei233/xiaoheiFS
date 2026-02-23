package http

import (
	"github.com/gin-gonic/gin"
	"net/http"
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
