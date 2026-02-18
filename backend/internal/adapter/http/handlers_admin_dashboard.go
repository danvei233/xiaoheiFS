package http

import (
	"github.com/gin-gonic/gin"
	"net/http"
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
