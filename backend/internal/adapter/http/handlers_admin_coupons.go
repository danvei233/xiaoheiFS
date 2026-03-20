package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

func (h *Handler) AdminCouponProductGroups(c *gin.Context) {
	if h.couponSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	items, err := h.couponSvc.ListProductGroups(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toCouponProductGroupDTOs(items)})
}

func (h *Handler) AdminCouponProductGroupCreate(c *gin.Context) {
	if h.couponSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	var payload CouponProductGroupDTO
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	group := domain.CouponProductGroup{
		Name:        strings.TrimSpace(payload.Name),
		Scope:       domain.CouponGroupScope(strings.TrimSpace(payload.Scope)),
		GoodsTypeID: payload.GoodsTypeID,
		RegionID:    payload.RegionID,
		PlanGroupID: payload.PlanGroupID,
		PackageID:   payload.PackageID,
		AddonCore:   payload.AddonCore,
		AddonMemGB:  payload.AddonMemGB,
		AddonDiskGB: payload.AddonDiskGB,
		AddonBWMbps: payload.AddonBWMbps,
	}
	applyCouponGroupRulesPayload(&group, payload.Rules)
	if err := h.couponSvc.CreateProductGroup(c, getUserID(c), &group); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toCouponProductGroupDTO(group))
}

func (h *Handler) AdminCouponProductGroupUpdate(c *gin.Context) {
	if h.couponSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	var payload CouponProductGroupDTO
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	group := domain.CouponProductGroup{
		ID:          uri.ID,
		Name:        strings.TrimSpace(payload.Name),
		Scope:       domain.CouponGroupScope(strings.TrimSpace(payload.Scope)),
		GoodsTypeID: payload.GoodsTypeID,
		RegionID:    payload.RegionID,
		PlanGroupID: payload.PlanGroupID,
		PackageID:   payload.PackageID,
		AddonCore:   payload.AddonCore,
		AddonMemGB:  payload.AddonMemGB,
		AddonDiskGB: payload.AddonDiskGB,
		AddonBWMbps: payload.AddonBWMbps,
	}
	applyCouponGroupRulesPayload(&group, payload.Rules)
	if err := h.couponSvc.UpdateProductGroup(c, getUserID(c), group); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toCouponProductGroupDTO(group))
}

func (h *Handler) AdminCouponProductGroupDelete(c *gin.Context) {
	if h.couponSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	if err := h.couponSvc.DeleteProductGroup(c, getUserID(c), uri.ID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminCoupons(c *gin.Context) {
	if h.couponSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	var query struct {
		Limit     int    `form:"limit" binding:"omitempty,gte=1,lte=200"`
		Offset    int    `form:"offset" binding:"omitempty,gte=0"`
		GroupID   int64  `form:"product_group_id" binding:"omitempty,gte=0"`
		ActiveRaw string `form:"active"`
		Keyword   string `form:"keyword"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	limit := 20
	if query.Limit > 0 {
		limit = query.Limit
	}
	offset := query.Offset
	var active *bool
	activeRaw := strings.TrimSpace(query.ActiveRaw)
	if activeRaw != "" {
		v := activeRaw == "1" || strings.EqualFold(activeRaw, "true")
		active = &v
	}
	items, total, err := h.couponSvc.ListCoupons(c, appshared.CouponFilter{
		Keyword:        strings.TrimSpace(query.Keyword),
		ProductGroupID: query.GroupID,
		Active:         active,
	}, limit, offset)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toCouponDTOs(items), "total": total})
}

func (h *Handler) AdminCouponCreate(c *gin.Context) {
	if h.couponSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	var payload CouponDTO
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	item := domain.Coupon{
		Code:             payload.Code,
		DiscountPermille: payload.DiscountPermille,
		ProductGroupID:   payload.ProductGroupID,
		TotalLimit:       payload.TotalLimit,
		PerUserLimit:     payload.PerUserLimit,
		StartsAt:         payload.StartsAt,
		EndsAt:           payload.EndsAt,
		NewUserOnly:      payload.NewUserOnly,
		Active:           payload.Active,
		Note:             payload.Note,
	}
	if err := h.couponSvc.CreateCoupon(c, getUserID(c), &item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toCouponDTO(item))
}

func (h *Handler) AdminCouponUpdate(c *gin.Context) {
	if h.couponSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	var payload CouponDTO
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	item := domain.Coupon{
		ID:               uri.ID,
		Code:             payload.Code,
		DiscountPermille: payload.DiscountPermille,
		ProductGroupID:   payload.ProductGroupID,
		TotalLimit:       payload.TotalLimit,
		PerUserLimit:     payload.PerUserLimit,
		StartsAt:         payload.StartsAt,
		EndsAt:           payload.EndsAt,
		NewUserOnly:      payload.NewUserOnly,
		Active:           payload.Active,
		Note:             payload.Note,
	}
	if err := h.couponSvc.UpdateCoupon(c, getUserID(c), item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toCouponDTO(item))
}

func (h *Handler) AdminCouponDelete(c *gin.Context) {
	if h.couponSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	if err := h.couponSvc.DeleteCoupon(c, getUserID(c), uri.ID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminCouponBatchGenerate(c *gin.Context) {
	if h.couponSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	var payload struct {
		Prefix           string  `json:"prefix"`
		Count            int     `json:"count"`
		Length           int     `json:"length"`
		DiscountPermille int     `json:"discount_permille"`
		ProductGroupID   int64   `json:"product_group_id"`
		TotalLimit       int     `json:"total_limit"`
		PerUserLimit     int     `json:"per_user_limit"`
		StartsAt         *string `json:"starts_at"`
		EndsAt           *string `json:"ends_at"`
		NewUserOnly      bool    `json:"new_user_only"`
		Active           bool    `json:"active"`
		Note             string  `json:"note"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	parseTime := func(raw *string) (*time.Time, error) {
		if raw == nil || strings.TrimSpace(*raw) == "" {
			return nil, nil
		}
		v, err := time.Parse(time.RFC3339, strings.TrimSpace(*raw))
		if err != nil {
			return nil, err
		}
		return &v, nil
	}
	startAt, err := parseTime(payload.StartsAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidExpireAt.Error()})
		return
	}
	endAt, err := parseTime(payload.EndsAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidExpireAt.Error()})
		return
	}
	base := domain.Coupon{
		DiscountPermille: payload.DiscountPermille,
		ProductGroupID:   payload.ProductGroupID,
		TotalLimit:       payload.TotalLimit,
		PerUserLimit:     payload.PerUserLimit,
		StartsAt:         startAt,
		EndsAt:           endAt,
		NewUserOnly:      payload.NewUserOnly,
		Active:           payload.Active,
		Note:             payload.Note,
	}
	items, err := h.couponSvc.BatchGenerateCoupons(c, getUserID(c), payload.Prefix, payload.Count, payload.Length, base)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toCouponDTOs(items), "total": len(items)})
}

func applyCouponGroupRulesPayload(group *domain.CouponProductGroup, rules []CouponProductRuleDTO) {
	if group == nil || len(rules) == 0 {
		return
	}
	normalized := make([]domain.CouponProductRule, 0, len(rules))
	for _, item := range rules {
		scope := domain.CouponGroupScope(strings.TrimSpace(item.Scope))
		if scope == "" {
			continue
		}
		normalized = append(normalized, domain.CouponProductRule{
			Scope:            scope,
			GoodsTypeID:      item.GoodsTypeID,
			RegionID:         item.RegionID,
			PlanGroupID:      item.PlanGroupID,
			PackageID:        item.PackageID,
			AddonCoreEnabled: item.AddonCoreEnabled,
			AddonMemEnabled:  item.AddonMemEnabled,
			AddonDiskEnabled: item.AddonDiskEnabled,
			AddonBWEnabled:   item.AddonBWEnabled,
		})
	}
	if len(normalized) == 0 {
		return
	}
	if raw, err := json.Marshal(normalized); err == nil {
		group.RulesJSON = string(raw)
	}
	first := normalized[0]
	group.Scope = first.Scope
	group.GoodsTypeID = first.GoodsTypeID
	group.RegionID = first.RegionID
	group.PlanGroupID = first.PlanGroupID
	group.PackageID = first.PackageID
	if first.AddonCoreEnabled {
		group.AddonCore = 1
	} else {
		group.AddonCore = 0
	}
	if first.AddonMemEnabled {
		group.AddonMemGB = 1
	} else {
		group.AddonMemGB = 0
	}
	if first.AddonDiskEnabled {
		group.AddonDiskGB = 1
	} else {
		group.AddonDiskGB = 0
	}
	if first.AddonBWEnabled {
		group.AddonBWMbps = 1
	} else {
		group.AddonBWMbps = 0
	}
}
