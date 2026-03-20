package http

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"xiaoheiplay/internal/domain"
)

type adminGroupRuleURI struct {
	ID     int64 `uri:"id" binding:"required,gt=0"`
	RuleID int64 `uri:"rule_id" binding:"required,gt=0"`
}

type adminGroupURI struct {
	ID int64 `uri:"id" binding:"required,gt=0"`
}

type adminRebuildGroupURI struct {
	ID int64 `uri:"id" binding:"gte=0"`
}

func (h *Handler) AdminUserTierGroups(c *gin.Context) {
	if h.userTierSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	items, err := h.userTierSvc.ListGroups(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toUserTierGroupDTOs(items)})
}

func (h *Handler) AdminUserTierGroupCreate(c *gin.Context) {
	if h.userTierSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	var payload UserTierGroupDTO
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	group := domain.UserTierGroup{
		Name:               strings.TrimSpace(payload.Name),
		Color:              strings.TrimSpace(payload.Color),
		Icon:               strings.TrimSpace(payload.Icon),
		Priority:           payload.Priority,
		AutoApproveEnabled: payload.AutoApproveEnabled,
	}
	if err := h.userTierSvc.CreateGroup(c, getUserID(c), &group); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toUserTierGroupDTO(group))
}

func (h *Handler) AdminUserTierGroupUpdate(c *gin.Context) {
	if h.userTierSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	var uri adminGroupURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	old, err := h.userTierSvc.GetGroup(c, uri.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	var payload UserTierGroupDTO
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	old.Name = strings.TrimSpace(payload.Name)
	old.Color = strings.TrimSpace(payload.Color)
	old.Icon = strings.TrimSpace(payload.Icon)
	old.Priority = payload.Priority
	old.AutoApproveEnabled = payload.AutoApproveEnabled
	if err := h.userTierSvc.UpdateGroup(c, getUserID(c), old); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toUserTierGroupDTO(old))
}

func (h *Handler) AdminUserTierGroupDelete(c *gin.Context) {
	if h.userTierSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	var uri adminGroupURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	if err := h.userTierSvc.DeleteGroup(c, getUserID(c), uri.ID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminUserTierDiscountRules(c *gin.Context) {
	if h.userTierSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	var uri adminGroupURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	items, err := h.userTierSvc.ListDiscountRules(c, uri.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toUserTierDiscountRuleDTOs(items)})
}

func (h *Handler) AdminUserTierDiscountRuleCreate(c *gin.Context) {
	if h.userTierSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	var uri adminGroupURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	var payload UserTierDiscountRuleDTO
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	rule := domain.UserTierDiscountRule{
		GroupID:          uri.ID,
		Scope:            domain.UserTierScope(strings.TrimSpace(payload.Scope)),
		GoodsTypeID:      payload.GoodsTypeID,
		RegionID:         payload.RegionID,
		PlanGroupID:      payload.PlanGroupID,
		PackageID:        payload.PackageID,
		DiscountPermille: payload.DiscountPermille,
		FixedPrice:       payload.FixedPrice,
		AddCorePermille:  payload.AddCorePermille,
		AddMemPermille:   payload.AddMemPermille,
		AddDiskPermille:  payload.AddDiskPermille,
		AddBWPermille:    payload.AddBWPermille,
	}
	if err := h.userTierSvc.CreateDiscountRule(c, getUserID(c), &rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toUserTierDiscountRuleDTO(rule))
}

func (h *Handler) AdminUserTierDiscountRuleUpdate(c *gin.Context) {
	if h.userTierSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	var uri adminGroupRuleURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	var payload UserTierDiscountRuleDTO
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	rule := domain.UserTierDiscountRule{
		ID:               uri.RuleID,
		GroupID:          uri.ID,
		Scope:            domain.UserTierScope(strings.TrimSpace(payload.Scope)),
		GoodsTypeID:      payload.GoodsTypeID,
		RegionID:         payload.RegionID,
		PlanGroupID:      payload.PlanGroupID,
		PackageID:        payload.PackageID,
		DiscountPermille: payload.DiscountPermille,
		FixedPrice:       payload.FixedPrice,
		AddCorePermille:  payload.AddCorePermille,
		AddMemPermille:   payload.AddMemPermille,
		AddDiskPermille:  payload.AddDiskPermille,
		AddBWPermille:    payload.AddBWPermille,
	}
	if err := h.userTierSvc.UpdateDiscountRule(c, getUserID(c), rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toUserTierDiscountRuleDTO(rule))
}

func (h *Handler) AdminUserTierDiscountRuleDelete(c *gin.Context) {
	if h.userTierSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	var uri adminGroupRuleURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	if err := h.userTierSvc.DeleteDiscountRule(c, getUserID(c), uri.ID, uri.RuleID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminUserTierAutoRules(c *gin.Context) {
	if h.userTierSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	var uri adminGroupURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	items, err := h.userTierSvc.ListAutoRules(c, uri.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toUserTierAutoRuleDTOs(items)})
}

func (h *Handler) AdminUserTierAutoRuleCreate(c *gin.Context) {
	if h.userTierSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	var uri adminGroupURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	var payload UserTierAutoRuleDTO
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	rule := domain.UserTierAutoRule{
		GroupID:        uri.ID,
		DurationDays:   payload.DurationDays,
		ConditionsJSON: payload.ConditionsJSON,
		SortOrder:      payload.SortOrder,
	}
	if err := h.userTierSvc.CreateAutoRule(c, getUserID(c), &rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toUserTierAutoRuleDTO(rule))
}

func (h *Handler) AdminUserTierAutoRuleUpdate(c *gin.Context) {
	if h.userTierSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	var uri adminGroupRuleURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	var payload UserTierAutoRuleDTO
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	rule := domain.UserTierAutoRule{
		ID:             uri.RuleID,
		GroupID:        uri.ID,
		DurationDays:   payload.DurationDays,
		ConditionsJSON: payload.ConditionsJSON,
		SortOrder:      payload.SortOrder,
	}
	if err := h.userTierSvc.UpdateAutoRule(c, getUserID(c), rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toUserTierAutoRuleDTO(rule))
}

func (h *Handler) AdminUserTierAutoRuleDelete(c *gin.Context) {
	if h.userTierSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	var uri adminGroupRuleURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	if err := h.userTierSvc.DeleteAutoRule(c, getUserID(c), uri.ID, uri.RuleID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminUserTierRebuild(c *gin.Context) {
	if h.userTierSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	var uri adminRebuildGroupURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	if uri.ID > 0 {
		h.userTierSvc.RebuildGroupPriceCacheAsync(uri.ID)
	} else {
		h.userTierSvc.RebuildAllPriceCachesAsync()
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminUserSetTier(c *gin.Context) {
	if h.userTierSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	var uri adminGroupURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	var payload struct {
		GroupID  int64  `json:"group_id"`
		ExpireAt string `json:"expire_at"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	var exp *time.Time
	if strings.TrimSpace(payload.ExpireAt) != "" {
		v, err := time.Parse(time.RFC3339, strings.TrimSpace(payload.ExpireAt))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidExpireAt.Error()})
			return
		}
		exp = &v
	}
	if err := h.userTierSvc.SetUserGroup(c, getUserID(c), uri.ID, payload.GroupID, exp); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
