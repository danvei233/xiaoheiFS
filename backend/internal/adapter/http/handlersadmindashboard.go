package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) AdminDashboardOverview(c *gin.Context) {
	if h.reportSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	overview, err := h.reportSvc.Overview(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "report error"})
		return
	}
	c.JSON(http.StatusOK, overview)
}

func (h *Handler) AdminDashboardRevenue(c *gin.Context) {
	if h.reportSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	period := c.Query("period")
	if period == "month" {
		points, err := h.reportSvc.RevenueByMonth(c, 6)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "report error"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": points})
		return
	}
	points, err := h.reportSvc.RevenueByDay(c, 30)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "report error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": points})
}

func (h *Handler) AdminDashboardVPSStatus(c *gin.Context) {
	if h.reportSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	items, err := h.reportSvc.VPSStatus(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "report error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}
