package http

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"time"
	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

var validVPSStatuses = map[domain.VPSStatus]bool{
	domain.VPSStatusProvisioning:     true,
	domain.VPSStatusRunning:          true,
	domain.VPSStatusStopped:          true,
	domain.VPSStatusReinstalling:     true,
	domain.VPSStatusReinstallFailed:  true,
	domain.VPSStatusExpiredLocked:    true,
	domain.VPSStatusRescue:           true,
	domain.VPSStatusCrackingPassword: true,
	domain.VPSStatusLocked:           true,
	domain.VPSStatusUnknown:          true,
}

var validVPSAdminStatuses = map[domain.VPSAdminStatus]bool{
	domain.VPSAdminStatusNormal: true,
	domain.VPSAdminStatusAbuse:  true,
	domain.VPSAdminStatusFraud:  true,
	domain.VPSAdminStatusLocked: true,
}

func (h *Handler) AdminVPSList(c *gin.Context) {
	limit, offset := paging(c)
	var query struct {
		Status  string `form:"status"`
		Keyword string `form:"keyword"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	// web_refresh_perf F4: status/keyword filtering is pushed to the backend so
	// the frontend no longer pulls the whole table and filters client-side.
	items, total, err := h.adminSvc.ListInstancesFiltered(c.Request.Context(), appports.VPSInstanceFilter{
		AdminStatus: strings.TrimSpace(query.Status),
		Keyword:     strings.TrimSpace(query.Keyword),
	}, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrListError.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": h.toVPSInstanceDTOsWithLifecycle(c, items), "total": total})
}

func (h *Handler) AdminVPSCreate(c *gin.Context) {
	var payload struct {
		UserID               int64          `json:"user_id"`
		OrderItemID          int64          `json:"order_item_id"`
		AutomationInstanceID string         `json:"automation_instance_id"`
		GoodsTypeID          int64          `json:"goods_type_id"`
		Name                 string         `json:"name"`
		Region               string         `json:"region"`
		RegionID             int64          `json:"region_id"`
		SystemID             int64          `json:"system_id"`
		Status               string         `json:"status"`
		AutomationState      int            `json:"automation_state"`
		AdminStatus          string         `json:"admin_status"`
		ExpireAt             *string        `json:"expire_at"`
		PanelURLCache        string         `json:"panel_url_cache"`
		Spec                 map[string]any `json:"spec"`
		AccessInfo           map[string]any `json:"access_info"`
		Provision            bool           `json:"provision"`
		LineID               int64          `json:"line_id"`
		PackageID            int64          `json:"package_id"`
		PackageName          string         `json:"package_name"`
		OS                   string         `json:"os"`
		CPU                  int            `json:"cpu"`
		MemoryGB             int            `json:"memory_gb"`
		DiskGB               int            `json:"disk_gb"`
		BandwidthMB          int            `json:"bandwidth_mbps"`
		PortNum              int            `json:"port_num"`
		MonthlyPrice         float64        `json:"monthly_price"`
	}
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	if payload.PackageID > 0 && h.catalogSvc != nil {
		if pkg, err := h.catalogSvc.GetPackage(c.Request.Context(), payload.PackageID); err == nil {
			if payload.GoodsTypeID == 0 {
				payload.GoodsTypeID = pkg.GoodsTypeID
			}
			if payload.PackageName == "" {
				payload.PackageName = pkg.Name
			}
			if payload.CPU == 0 {
				payload.CPU = pkg.Cores
			}
			if payload.MemoryGB == 0 {
				payload.MemoryGB = pkg.MemoryGB
			}
			if payload.DiskGB == 0 {
				payload.DiskGB = pkg.DiskGB
			}
			if payload.BandwidthMB == 0 {
				payload.BandwidthMB = pkg.BandwidthMB
			}
			if payload.PortNum == 0 {
				payload.PortNum = pkg.PortNum
			}
			if payload.MonthlyPrice == 0 {
				payload.MonthlyPrice = centsToFloat(pkg.Monthly)
			}
			if plan, err := h.catalogSvc.GetPlanGroup(c.Request.Context(), pkg.PlanGroupID); err == nil {
				if payload.LineID == 0 {
					payload.LineID = plan.LineID
				}
				if payload.RegionID == 0 {
					payload.RegionID = plan.RegionID
				}
			}
		}
	}
	if payload.Region == "" && payload.RegionID > 0 && h.catalogSvc != nil {
		if region, err := h.catalogSvc.GetRegion(c.Request.Context(), payload.RegionID); err == nil {
			payload.Region = region.Name
			if payload.GoodsTypeID == 0 {
				payload.GoodsTypeID = region.GoodsTypeID
			}
		}
	}
	var expireAt *time.Time
	if payload.ExpireAt != nil && strings.TrimSpace(*payload.ExpireAt) != "" {
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(*payload.ExpireAt))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidExpireAt.Error()})
			return
		}
		expireAt = &t
	}
	specJSON := "{}"
	if payload.Spec != nil {
		specJSON = mustJSON(payload.Spec)
	}
	accessJSON := "{}"
	if payload.AccessInfo != nil {
		accessJSON = mustJSON(payload.AccessInfo)
	}
	osName := strings.TrimSpace(payload.OS)
	if payload.Provision && osName == "" && payload.SystemID > 0 {
		if img, err := h.catalogSvc.GetSystemImage(c.Request.Context(), payload.SystemID); err == nil {
			osName = img.Name
		}
	}
	inst, err := h.adminVPS.Create(c.Request.Context(), getUserID(c), appshared.AdminVPSCreateInput{
		UserID:               payload.UserID,
		OrderItemID:          payload.OrderItemID,
		AutomationInstanceID: payload.AutomationInstanceID,
		GoodsTypeID:          payload.GoodsTypeID,
		Name:                 payload.Name,
		Region:               payload.Region,
		RegionID:             payload.RegionID,
		SystemID:             payload.SystemID,
		Status:               domain.VPSStatus(payload.Status),
		AutomationState:      payload.AutomationState,
		AdminStatus:          domain.VPSAdminStatus(payload.AdminStatus),
		ExpireAt:             expireAt,
		PanelURLCache:        payload.PanelURLCache,
		SpecJSON:             specJSON,
		AccessInfoJSON:       accessJSON,
		Provision:            payload.Provision,
		LineID:               payload.LineID,
		PackageID:            payload.PackageID,
		PackageName:          payload.PackageName,
		OS:                   osName,
		CPU:                  payload.CPU,
		MemoryGB:             payload.MemoryGB,
		DiskGB:               payload.DiskGB,
		BandwidthMB:          payload.BandwidthMB,
		PortNum:              payload.PortNum,
		MonthlyPrice:         floatToCents(payload.MonthlyPrice),
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, h.toVPSInstanceDTOWithLifecycle(c, inst))
}

func (h *Handler) AdminVPSDetail(c *gin.Context) {
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	inst, err := h.adminVPS.Get(c.Request.Context(), uri.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	c.JSON(http.StatusOK, h.toVPSInstanceDTOWithLifecycle(c, inst))
}

func (h *Handler) AdminVPSUpdate(c *gin.Context) {
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	var payload struct {
		PackageID     *int64         `json:"package_id"`
		PackageName   *string        `json:"package_name"`
		MonthlyPrice  *float64       `json:"monthly_price"`
		SystemID      *int64         `json:"system_id"`
		Spec          map[string]any `json:"spec"`
		Status        *string        `json:"status"`
		AdminStatus   *string        `json:"admin_status"`
		CPU           *int           `json:"cpu"`
		MemoryGB      *int           `json:"memory_gb"`
		DiskGB        *int           `json:"disk_gb"`
		BandwidthMB   *int           `json:"bandwidth_mbps"`
		PortNum       *int           `json:"port_num"`
		PanelURLCache *string        `json:"panel_url_cache"`
		AccessInfo    map[string]any `json:"access_info"`
		SyncMode      string         `json:"sync_mode"`
	}
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	if payload.PackageID != nil && payload.PackageName == nil && h.catalogSvc != nil {
		if pkg, err := h.catalogSvc.GetPackage(c.Request.Context(), *payload.PackageID); err == nil {
			name := pkg.Name
			payload.PackageName = &name
		}
	}
	specJSON := ""
	if payload.Spec != nil {
		specJSON = mustJSON(payload.Spec)
	}
	accessJSON := ""
	if payload.AccessInfo != nil {
		accessJSON = mustJSON(payload.AccessInfo)
	}
	var statusVal *domain.VPSStatus
	if payload.Status != nil {
		tmp := domain.VPSStatus(*payload.Status)
		if !validVPSStatuses[tmp] {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
			return
		}
		statusVal = &tmp
	}
	var adminStatusVal *domain.VPSAdminStatus
	if payload.AdminStatus != nil {
		tmp := domain.VPSAdminStatus(*payload.AdminStatus)
		if !validVPSAdminStatuses[tmp] {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
			return
		}
		adminStatusVal = &tmp
	}
	var monthlyPrice *int64
	if payload.MonthlyPrice != nil {
		val := floatToCents(*payload.MonthlyPrice)
		monthlyPrice = &val
	}
	input := appshared.AdminVPSUpdateInput{
		PackageID:     payload.PackageID,
		PackageName:   payload.PackageName,
		MonthlyPrice:  monthlyPrice,
		SystemID:      payload.SystemID,
		Status:        statusVal,
		AdminStatus:   adminStatusVal,
		CPU:           payload.CPU,
		MemoryGB:      payload.MemoryGB,
		DiskGB:        payload.DiskGB,
		BandwidthMB:   payload.BandwidthMB,
		PortNum:       payload.PortNum,
		PanelURLCache: payload.PanelURLCache,
		SyncMode:      strings.TrimSpace(payload.SyncMode),
	}
	if specJSON != "" {
		input.SpecJSON = &specJSON
	}
	if accessJSON != "" {
		input.AccessInfoJSON = &accessJSON
	}
	inst, err := h.adminVPS.Update(c.Request.Context(), getUserID(c), uri.ID, input)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": domain.ErrForbidden.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, h.toVPSInstanceDTOWithLifecycle(c, inst))
}

func (h *Handler) AdminVPSUpdateExpire(c *gin.Context) {
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	var payload struct {
		ExpireAt string `json:"expire_at"`
	}
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	if payload.ExpireAt == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrExpireAtRequired.Error()})
		return
	}
	t, err := time.Parse("2006-01-02 15:04:05", payload.ExpireAt)
	if err != nil {
		t, err = time.Parse("2006-01-02", payload.ExpireAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidExpireAt.Error()})
			return
		}
	}
	inst, err := h.adminVPS.UpdateExpireAt(c.Request.Context(), getUserID(c), uri.ID, t)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, h.toVPSInstanceDTOWithLifecycle(c, inst))
}

func (h *Handler) AdminVPSLock(c *gin.Context) {
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	if err := h.adminVPS.SetAdminStatus(c.Request.Context(), getUserID(c), uri.ID, domain.VPSAdminStatusLocked, "lock"); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminVPSUnlock(c *gin.Context) {
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	if err := h.adminVPS.SetAdminStatus(c.Request.Context(), getUserID(c), uri.ID, domain.VPSAdminStatusNormal, "unlock"); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminVPSDelete(c *gin.Context) {
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	var payload struct {
		Reason string `json:"reason"`
	}
	if err := bindJSONOptional(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	// Create the automatic refund BEFORE the instance is hard-deleted. The
	// refund service re-fetches the VPS instance (GetInstance) to compute the
	// amount, so deleting first makes the refund silently fail — the row is gone
	// by the time it is read. Abort the delete if the refund cannot be created,
	// otherwise the user would lose both the VPS and the money.
	if h.walletOrder != nil {
		if _, _, refundErr := h.walletOrder.AutoRefundOnAdminDelete(c.Request.Context(), getUserID(c), uri.ID, payload.Reason); refundErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": refundErr.Error()})
			return
		}
	}
	if err := h.adminVPS.Delete(c.Request.Context(), getUserID(c), uri.ID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminVPSResize(c *gin.Context) {
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	var payload struct {
		CPU       int `json:"cpu"`
		MemoryGB  int `json:"memory_gb"`
		DiskGB    int `json:"disk_gb"`
		Bandwidth int `json:"bandwidth_mbps"`
	}
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	req := appshared.AutomationElasticUpdateRequest{}
	if payload.CPU > 0 {
		req.CPU = &payload.CPU
	}
	if payload.MemoryGB > 0 {
		req.MemoryGB = &payload.MemoryGB
	}
	if payload.DiskGB > 0 {
		req.DiskGB = &payload.DiskGB
	}
	if payload.Bandwidth > 0 {
		req.Bandwidth = &payload.Bandwidth
	}
	if err := h.adminVPS.Resize(c.Request.Context(), getUserID(c), uri.ID, req, mustJSON(payload)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminVPSStatus(c *gin.Context) {
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	var payload struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	}
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	status := domain.VPSAdminStatus(payload.Status)
	if err := h.adminVPS.SetAdminStatus(c.Request.Context(), getUserID(c), uri.ID, status, payload.Reason); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminVPSEmergencyRenew(c *gin.Context) {
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	inst, err := h.adminVPS.Get(c.Request.Context(), uri.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	_, err = h.orderSvc.CreateEmergencyRenewOrder(c.Request.Context(), inst.UserID, inst.ID)
	if err != nil {
		status := http.StatusBadRequest
		if err == appshared.ErrConflict {
			status = http.StatusConflict
		} else if err == appshared.ErrForbidden {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	updated, _ := h.adminVPS.Get(c.Request.Context(), uri.ID)
	c.JSON(http.StatusOK, h.toVPSInstanceDTOWithLifecycle(c, updated))
}

func (h *Handler) AdminVPSRefresh(c *gin.Context) {
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	inst, err := h.adminVPS.Refresh(c.Request.Context(), getUserID(c), uri.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, h.toVPSInstanceDTOWithLifecycle(c, inst))
}
