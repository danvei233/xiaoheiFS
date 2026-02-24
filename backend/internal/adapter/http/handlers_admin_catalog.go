package http

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"sort"
	"strconv"
	"xiaoheiplay/internal/domain"
)

func (h *Handler) AdminAuditLogs(c *gin.Context) {
	limit, offset := paging(c)
	items, total, err := h.adminSvc.ListAuditLogs(c, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrListError.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toAdminAuditLogDTOs(items), "total": total})
}

func (h *Handler) AdminSystemImages(c *gin.Context) {
	lineID, _ := strconv.ParseInt(c.Query("line_id"), 10, 64)
	planGroupID, _ := strconv.ParseInt(c.Query("plan_group_id"), 10, 64)
	if planGroupID > 0 {
		plan, err := h.catalogSvc.GetPlanGroup(c, planGroupID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrPlanGroupNotFound.Error()})
			return
		}
		if plan.LineID <= 0 {
			c.JSON(http.StatusOK, gin.H{"items": []SystemImageDTO{}})
			return
		}
		lineID = plan.LineID
	}
	items, err := h.catalogSvc.ListSystemImages(c, lineID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrListError.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toSystemImageDTOs(items)})
}

func (h *Handler) AdminRegions(c *gin.Context) {
	goodsTypeID, _ := strconv.ParseInt(c.Query("goods_type_id"), 10, 64)
	items, err := h.catalogSvc.ListRegions(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrListError.Error()})
		return
	}
	if goodsTypeID > 0 {
		filtered := make([]domain.Region, 0, len(items))
		for _, item := range items {
			if item.GoodsTypeID == goodsTypeID {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	c.JSON(http.StatusOK, gin.H{"items": toRegionDTOs(items)})
}

func (h *Handler) AdminRegionCreate(c *gin.Context) {
	var payload RegionDTO
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	region := regionDTOToDomain(payload)
	if region.GoodsTypeID <= 0 {
		region.GoodsTypeID = h.defaultGoodsTypeID(c)
	}
	if err := h.catalogSvc.CreateRegion(c, &region); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toRegionDTO(region))
}

func (h *Handler) AdminRegionUpdate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload RegionDTO
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	payload.ID = id
	region := regionDTOToDomain(payload)
	if region.GoodsTypeID <= 0 {
		if current, err := h.catalogSvc.GetRegion(c, id); err == nil && current.GoodsTypeID > 0 {
			region.GoodsTypeID = current.GoodsTypeID
		}
	}
	if err := h.catalogSvc.UpdateRegion(c, region); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toRegionDTO(region))
}

func (h *Handler) AdminRegionDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.catalogSvc.DeleteRegion(c, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminRegionBulkDelete(c *gin.Context) {
	var payload struct {
		IDs []int64 `json:"ids"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	if len(payload.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrIdsRequired.Error()})
		return
	}
	for _, id := range payload.IDs {
		if err := h.catalogSvc.DeleteRegion(c, id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPlanGroups(c *gin.Context) {
	goodsTypeID, _ := strconv.ParseInt(c.Query("goods_type_id"), 10, 64)
	items, err := h.catalogSvc.ListPlanGroups(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrListError.Error()})
		return
	}
	if goodsTypeID > 0 {
		filtered := make([]domain.PlanGroup, 0, len(items))
		for _, item := range items {
			if item.GoodsTypeID == goodsTypeID {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	c.JSON(http.StatusOK, gin.H{"items": toPlanGroupDTOs(items)})
}

func (h *Handler) AdminLines(c *gin.Context) {
	h.AdminPlanGroups(c)
}

func (h *Handler) AdminPlanGroupCreate(c *gin.Context) {
	var payload PlanGroupDTO
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	plan := planGroupDTOToDomain(payload)
	if plan.RegionID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidRegionId.Error()})
		return
	}
	if region, err := h.catalogSvc.GetRegion(c, plan.RegionID); err == nil {
		plan.GoodsTypeID = region.GoodsTypeID
	}
	if plan.GoodsTypeID <= 0 {
		plan.GoodsTypeID = h.defaultGoodsTypeID(c)
	}
	if err := h.catalogSvc.CreatePlanGroup(c, &plan); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toPlanGroupDTO(plan))
}

func (h *Handler) AdminLineCreate(c *gin.Context) {
	h.AdminPlanGroupCreate(c)
}

func (h *Handler) AdminPlanGroupUpdate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		RegionID          *int64   `json:"region_id"`
		Name              *string  `json:"name"`
		LineID            *int64   `json:"line_id"`
		UnitCore          *float64 `json:"unit_core"`
		UnitMem           *float64 `json:"unit_mem"`
		UnitDisk          *float64 `json:"unit_disk"`
		UnitBW            *float64 `json:"unit_bw"`
		AddCoreMin        *int     `json:"add_core_min"`
		AddCoreMax        *int     `json:"add_core_max"`
		AddCoreStep       *int     `json:"add_core_step"`
		AddMemMin         *int     `json:"add_mem_min"`
		AddMemMax         *int     `json:"add_mem_max"`
		AddMemStep        *int     `json:"add_mem_step"`
		AddDiskMin        *int     `json:"add_disk_min"`
		AddDiskMax        *int     `json:"add_disk_max"`
		AddDiskStep       *int     `json:"add_disk_step"`
		AddBWMin          *int     `json:"add_bw_min"`
		AddBWMax          *int     `json:"add_bw_max"`
		AddBWStep         *int     `json:"add_bw_step"`
		Active            *bool    `json:"active"`
		Visible           *bool    `json:"visible"`
		CapacityRemaining *int     `json:"capacity_remaining"`
		SortOrder         *int     `json:"sort_order"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	plan, err := h.catalogSvc.GetPlanGroup(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if payload.RegionID != nil {
		plan.RegionID = *payload.RegionID
		if region, err := h.catalogSvc.GetRegion(c, plan.RegionID); err == nil && region.GoodsTypeID > 0 {
			plan.GoodsTypeID = region.GoodsTypeID
		}
	}
	if payload.Name != nil {
		plan.Name = *payload.Name
	}
	if payload.LineID != nil {
		plan.LineID = *payload.LineID
	}
	if payload.UnitCore != nil {
		plan.UnitCore = floatToCents(*payload.UnitCore)
	}
	if payload.UnitMem != nil {
		plan.UnitMem = floatToCents(*payload.UnitMem)
	}
	if payload.UnitDisk != nil {
		plan.UnitDisk = floatToCents(*payload.UnitDisk)
	}
	if payload.UnitBW != nil {
		plan.UnitBW = floatToCents(*payload.UnitBW)
	}
	if payload.AddCoreMin != nil {
		plan.AddCoreMin = *payload.AddCoreMin
	}
	if payload.AddCoreMax != nil {
		plan.AddCoreMax = *payload.AddCoreMax
	}
	if payload.AddCoreStep != nil {
		plan.AddCoreStep = *payload.AddCoreStep
	}
	if payload.AddMemMin != nil {
		plan.AddMemMin = *payload.AddMemMin
	}
	if payload.AddMemMax != nil {
		plan.AddMemMax = *payload.AddMemMax
	}
	if payload.AddMemStep != nil {
		plan.AddMemStep = *payload.AddMemStep
	}
	if payload.AddDiskMin != nil {
		plan.AddDiskMin = *payload.AddDiskMin
	}
	if payload.AddDiskMax != nil {
		plan.AddDiskMax = *payload.AddDiskMax
	}
	if payload.AddDiskStep != nil {
		plan.AddDiskStep = *payload.AddDiskStep
	}
	if payload.AddBWMin != nil {
		plan.AddBWMin = *payload.AddBWMin
	}
	if payload.AddBWMax != nil {
		plan.AddBWMax = *payload.AddBWMax
	}
	if payload.AddBWStep != nil {
		plan.AddBWStep = *payload.AddBWStep
	}
	if payload.Active != nil {
		plan.Active = *payload.Active
	}
	if payload.Visible != nil {
		plan.Visible = *payload.Visible
	}
	if payload.CapacityRemaining != nil {
		plan.CapacityRemaining = *payload.CapacityRemaining
	}
	if payload.SortOrder != nil {
		plan.SortOrder = *payload.SortOrder
	}
	if err := h.catalogSvc.UpdatePlanGroup(c, plan); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toPlanGroupDTO(plan))
}

func (h *Handler) AdminLineUpdate(c *gin.Context) {
	h.AdminPlanGroupUpdate(c)
}

func (h *Handler) AdminLineSystemImages(c *gin.Context) {
	planGroupID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		ImageIDs []int64 `json:"image_ids"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	if planGroupID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidLineId.Error()})
		return
	}
	plan, err := h.catalogSvc.GetPlanGroup(c, planGroupID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if plan.LineID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrLineIdRequired.Error()})
		return
	}
	if err := h.catalogSvc.SetLineSystemImages(c, plan.LineID, payload.ImageIDs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPlanGroupDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.catalogSvc.DeletePlanGroup(c, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPlanGroupBulkDelete(c *gin.Context) {
	var payload struct {
		IDs []int64 `json:"ids"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	if len(payload.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrIdsRequired.Error()})
		return
	}
	for _, id := range payload.IDs {
		if err := h.catalogSvc.DeletePlanGroup(c, id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminLineDelete(c *gin.Context) {
	h.AdminPlanGroupDelete(c)
}

func (h *Handler) AdminPackages(c *gin.Context) {
	planGroupID, _ := strconv.ParseInt(c.Query("plan_group_id"), 10, 64)
	goodsTypeID, _ := strconv.ParseInt(c.Query("goods_type_id"), 10, 64)
	items, err := h.catalogSvc.ListPackages(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrListError.Error()})
		return
	}
	if goodsTypeID > 0 {
		filtered := make([]domain.Package, 0, len(items))
		for _, item := range items {
			if item.GoodsTypeID == goodsTypeID {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	if planGroupID > 0 {
		filtered := make([]domain.Package, 0, len(items))
		for _, item := range items {
			if item.PlanGroupID == planGroupID {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	c.JSON(http.StatusOK, gin.H{"items": toPackageDTOs(items)})
}

func (h *Handler) AdminProducts(c *gin.Context) {
	h.AdminPackages(c)
}

func (h *Handler) AdminPackageCreate(c *gin.Context) {
	var payload PackageDTO
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	pkg := packageDTOToDomain(payload)
	if pkg.PlanGroupID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidPlanGroupId.Error()})
		return
	}
	if plan, err := h.catalogSvc.GetPlanGroup(c, pkg.PlanGroupID); err == nil {
		pkg.GoodsTypeID = plan.GoodsTypeID
	}
	if pkg.GoodsTypeID <= 0 {
		pkg.GoodsTypeID = h.defaultGoodsTypeID(c)
	}
	if err := h.catalogSvc.CreatePackage(c, &pkg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toPackageDTO(pkg))
}

func (h *Handler) AdminProductCreate(c *gin.Context) {
	h.AdminPackageCreate(c)
}

func (h *Handler) AdminPackageUpdate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		PlanGroupID          *int64   `json:"plan_group_id"`
		ProductID            *int64   `json:"product_id"`
		IntegrationPackageID *int64   `json:"integration_package_id"`
		Name                 *string  `json:"name"`
		Cores                *int     `json:"cores"`
		MemoryGB             *int     `json:"memory_gb"`
		DiskGB               *int     `json:"disk_gb"`
		BandwidthMB          *int     `json:"bandwidth_mbps"`
		CPUModel             *string  `json:"cpu_model"`
		MonthlyPrice         *float64 `json:"monthly_price"`
		PortNum              *int     `json:"port_num"`
		SortOrder            *int     `json:"sort_order"`
		Active               *bool    `json:"active"`
		Visible              *bool    `json:"visible"`
		CapacityRemaining    *int     `json:"capacity_remaining"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	pkg, err := h.catalogSvc.GetPackage(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	if payload.PlanGroupID != nil {
		if *payload.PlanGroupID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidPlanGroupId.Error()})
			return
		}
		pkg.PlanGroupID = *payload.PlanGroupID
		if plan, err := h.catalogSvc.GetPlanGroup(c, pkg.PlanGroupID); err == nil && plan.GoodsTypeID > 0 {
			pkg.GoodsTypeID = plan.GoodsTypeID
		}
	}
	if payload.ProductID != nil {
		pkg.ProductID = *payload.ProductID
	}
	if payload.IntegrationPackageID != nil {
		pkg.IntegrationPackageID = *payload.IntegrationPackageID
	}
	if payload.Name != nil {
		pkg.Name = *payload.Name
	}
	if payload.Cores != nil {
		pkg.Cores = *payload.Cores
	}
	if payload.MemoryGB != nil {
		pkg.MemoryGB = *payload.MemoryGB
	}
	if payload.DiskGB != nil {
		pkg.DiskGB = *payload.DiskGB
	}
	if payload.BandwidthMB != nil {
		pkg.BandwidthMB = *payload.BandwidthMB
	}
	if payload.CPUModel != nil {
		pkg.CPUModel = *payload.CPUModel
	}
	if payload.MonthlyPrice != nil {
		pkg.Monthly = floatToCents(*payload.MonthlyPrice)
	}
	if payload.PortNum != nil {
		pkg.PortNum = *payload.PortNum
	}
	if payload.SortOrder != nil {
		pkg.SortOrder = *payload.SortOrder
	}
	if payload.Active != nil {
		pkg.Active = *payload.Active
	}
	if payload.Visible != nil {
		pkg.Visible = *payload.Visible
	}
	if payload.CapacityRemaining != nil {
		pkg.CapacityRemaining = *payload.CapacityRemaining
	}
	if err := h.catalogSvc.UpdatePackage(c, pkg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toPackageDTO(pkg))
}

func (h *Handler) AdminProductUpdate(c *gin.Context) {
	h.AdminPackageUpdate(c)
}

func (h *Handler) AdminPackageDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.catalogSvc.DeletePackage(c, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPackageCapabilitiesGet(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	if _, err := h.catalogSvc.GetPackage(c, id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	resizeEnabled, resizeSource := h.packageCapabilityResolvedValue(c, id, "resize", "resize_enabled", true)
	refundEnabled, refundSource := h.packageCapabilityResolvedValue(c, id, "refund", "refund_enabled", true)
	raw := h.getPackageCapabilityPolicy(c, id)
	c.JSON(http.StatusOK, gin.H{
		"package_id":             id,
		"resize_enabled":         resizeEnabled,
		"refund_enabled":         refundEnabled,
		"resize_source":          resizeSource,
		"refund_source":          refundSource,
		"package_resize_enabled": raw.ResizeEnabled,
		"package_refund_enabled": raw.RefundEnabled,
	})
}

func (h *Handler) AdminPackageCapabilitiesUpdate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	if _, err := h.catalogSvc.GetPackage(c, id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	var payload struct {
		ResizeEnabled *bool `json:"resize_enabled"`
		RefundEnabled *bool `json:"refund_enabled"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	if err := h.savePackageCapabilityPolicy(c, id, packageCapabilityPolicy{
		ResizeEnabled: payload.ResizeEnabled,
		RefundEnabled: payload.RefundEnabled,
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPackageBulkDelete(c *gin.Context) {
	var payload struct {
		IDs []int64 `json:"ids"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	if len(payload.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrIdsRequired.Error()})
		return
	}
	for _, id := range payload.IDs {
		if err := h.catalogSvc.DeletePackage(c, id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminProductDelete(c *gin.Context) {
	h.AdminPackageDelete(c)
}
func (h *Handler) AdminBillingCycles(c *gin.Context) {
	items, err := h.catalogSvc.ListBillingCycles(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrListError.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toBillingCycleDTOs(items)})
}

func (h *Handler) AdminBillingCycleCreate(c *gin.Context) {
	var payload BillingCycleDTO
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	cycle := billingCycleDTOToDomain(payload)
	if err := h.catalogSvc.CreateBillingCycle(c, &cycle); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toBillingCycleDTO(cycle))
}

func (h *Handler) AdminBillingCycleUpdate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload BillingCycleDTO
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	payload.ID = id
	cycle := billingCycleDTOToDomain(payload)
	if err := h.catalogSvc.UpdateBillingCycle(c, cycle); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toBillingCycleDTO(cycle))
}

func (h *Handler) AdminBillingCycleDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.catalogSvc.DeleteBillingCycle(c, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminBillingCycleBulkDelete(c *gin.Context) {
	var payload struct {
		IDs []int64 `json:"ids"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	if len(payload.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrIdsRequired.Error()})
		return
	}
	for _, id := range payload.IDs {
		if err := h.catalogSvc.DeleteBillingCycle(c, id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminSystemImageCreate(c *gin.Context) {
	var payload SystemImageDTO
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	img := systemImageDTOToDomain(payload)
	if err := h.catalogSvc.CreateSystemImage(c, &img); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toSystemImageDTO(img))
}

func (h *Handler) AdminSystemImageUpdate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload SystemImageDTO
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	payload.ID = id
	img := systemImageDTOToDomain(payload)
	if err := h.catalogSvc.UpdateSystemImage(c, img); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toSystemImageDTO(img))
}

func (h *Handler) AdminSystemImageDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.catalogSvc.DeleteSystemImage(c, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminSystemImageBulkDelete(c *gin.Context) {
	var payload struct {
		IDs []int64 `json:"ids"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	if len(payload.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrIdsRequired.Error()})
		return
	}
	for _, id := range payload.IDs {
		if err := h.catalogSvc.DeleteSystemImage(c, id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminSystemImageSync(c *gin.Context) {
	if h.integration == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}

	lineID, _ := strconv.ParseInt(c.Query("line_id"), 10, 64)
	planGroupID, _ := strconv.ParseInt(c.Query("plan_group_id"), 10, 64)
	if planGroupID > 0 {
		plan, err := h.catalogSvc.GetPlanGroup(c, planGroupID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrPlanGroupNotFound.Error()})
			return
		}
		if plan.LineID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrLineIdRequiredForPlanGroup.Error()})
			return
		}
		lineID = plan.LineID
	}

	if lineID > 0 {
		count, err := h.integration.SyncAutomationImagesForLine(c, lineID, "merge")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		resp := gin.H{
			"count":   count,
			"line_id": lineID,
		}
		if images, lerr := h.catalogSvc.ListSystemImages(c, lineID); lerr == nil {
			resp["line_image_count"] = len(images)
		}
		c.JSON(http.StatusOK, resp)
		return
	}

	goodsTypeID, _ := strconv.ParseInt(c.Query("goods_type_id"), 10, 64)
	if goodsTypeID <= 0 {
		if h.goodsTypes == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrGoodsTypeIdRequired.Error()})
			return
		}
		items, err := h.goodsTypes.List(c)
		if err != nil || len(items) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrGoodsTypeIdRequired.Error()})
			return
		}
		sort.SliceStable(items, func(i, j int) bool {
			if items[i].SortOrder == items[j].SortOrder {
				return items[i].ID < items[j].ID
			}
			return items[i].SortOrder < items[j].SortOrder
		})
		goodsTypeID = items[0].ID
	}
	result, err := h.integration.SyncAutomationForGoodsType(c, goodsTypeID, "merge")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"count":         result.Images,
		"goods_type_id": goodsTypeID,
		"sync_result": gin.H{
			"lines":    result.Lines,
			"products": result.Products,
			"images":   result.Images,
		},
	})
}
