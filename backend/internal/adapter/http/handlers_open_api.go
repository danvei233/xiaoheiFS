package http

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	apporder "xiaoheiplay/internal/app/order"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

type userAPIKeyDTO struct {
	ID         int64      `json:"id"`
	Name       string     `json:"name"`
	AKID       string     `json:"akid"`
	Status     string     `json:"status"`
	ScopesJSON string     `json:"scopes_json"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func toUserAPIKeyDTO(k domain.UserAPIKey) userAPIKeyDTO {
	return userAPIKeyDTO{
		ID:         k.ID,
		Name:       k.Name,
		AKID:       k.AKID,
		Status:     string(k.Status),
		ScopesJSON: k.ScopesJSON,
		LastUsedAt: k.LastUsedAt,
		CreatedAt:  k.CreatedAt,
		UpdatedAt:  k.UpdatedAt,
	}
}

func (h *Handler) OpenUserAPIKeys(c *gin.Context) {
	if h.userAPIKeySvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrApiKeyDisabled.Error()})
		return
	}
	limit, offset := paging(c)
	items, total, err := h.userAPIKeySvc.List(c, getUserID(c), limit, offset)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp := make([]userAPIKeyDTO, 0, len(items))
	for _, item := range items {
		resp = append(resp, toUserAPIKeyDTO(item))
	}
	c.JSON(http.StatusOK, gin.H{"items": resp, "total": total})
}

func (h *Handler) OpenUserAPIKeyCreate(c *gin.Context) {
	if h.userAPIKeySvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrApiKeyDisabled.Error()})
		return
	}
	var payload struct {
		Name   string   `json:"name"`
		Scopes []string `json:"scopes"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	result, err := h.userAPIKeySvc.Create(c, getUserID(c), payload.Name, payload.Scopes)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"item":   toUserAPIKeyDTO(result.Key),
		"key":    result.Secret,
		"secret": result.Secret,
	})
}

func (h *Handler) OpenUserAPIKeyPatch(c *gin.Context) {
	if h.userAPIKeySvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrApiKeyDisabled.Error()})
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		Status string `json:"status"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	if err := h.userAPIKeySvc.UpdateStatus(c, getUserID(c), id, domain.APIKeyStatus(strings.TrimSpace(payload.Status))); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) OpenUserAPIKeyDelete(c *gin.Context) {
	if h.userAPIKeySvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrApiKeyDisabled.Error()})
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.userAPIKeySvc.Delete(c, getUserID(c), id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) OpenInstantOrderCreate(c *gin.Context) {
	if h.openAPISvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrOrdersDisabled.Error()})
		return
	}
	var payload struct {
		Items      []appshared.OrderItemInput `json:"items"`
		CouponCode string                     `json:"coupon_code"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	if len(payload.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrItemsRequired.Error()})
		return
	}
	ctx := apporder.WithOrderSource(c, apporder.OrderSourceUserAPIKey)
	order, items, payRes, err := h.openAPISvc.InstantCreate(ctx, getUserID(c), payload.Items, c.GetHeader("Idempotency-Key"), payload.CouponCode)
	if err != nil {
		h.writeOpenOrderError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"order": toOrderDTO(order), "items": toOrderItemDTOs(items), "payment": toPaymentSelectDTO(payRes)})
}

func (h *Handler) OpenInstantOrderRenew(c *gin.Context) {
	if h.openAPISvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrOrdersDisabled.Error()})
		return
	}
	var payload struct {
		VPSID          int64 `json:"vps_id"`
		RenewDays      int   `json:"renew_days"`
		DurationMonths int   `json:"duration_months"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	ctx := apporder.WithOrderSource(c, apporder.OrderSourceUserAPIKey)
	order, payRes, err := h.openAPISvc.InstantRenew(ctx, getUserID(c), payload.VPSID, payload.RenewDays, payload.DurationMonths)
	if err != nil {
		h.writeOpenOrderError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"order": toOrderDTO(order), "payment": toPaymentSelectDTO(payRes)})
}

func (h *Handler) OpenInstantOrderResize(c *gin.Context) {
	if h.openAPISvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrOrdersDisabled.Error()})
		return
	}
	var payload struct {
		VPSID           int64               `json:"vps_id"`
		Spec            *appshared.CartSpec `json:"spec"`
		TargetPackageID int64               `json:"target_package_id"`
		ResetAddons     bool                `json:"reset_addons"`
		ScheduledAt     string              `json:"scheduled_at"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	var scheduledAt *time.Time
	if strings.TrimSpace(payload.ScheduledAt) != "" {
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(payload.ScheduledAt))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidScheduledAt.Error()})
			return
		}
		scheduledAt = &t
	}
	ctx := apporder.WithOrderSource(c, apporder.OrderSourceUserAPIKey)
	order, quote, payRes, err := h.openAPISvc.InstantResize(ctx, getUserID(c), payload.VPSID, payload.Spec, payload.TargetPackageID, payload.ResetAddons, scheduledAt)
	if err != nil {
		h.writeOpenOrderError(c, err)
		return
	}
	resp := map[string]any{
		"vps_id":             payload.VPSID,
		"charge_amount":      centsToFloat(quote.ChargeAmount),
		"refund_amount":      centsToFloat(quote.RefundAmount),
		"target_package_id":  quote.TargetPackageID,
		"current_package_id": quote.CurrentPackageID,
	}
	c.JSON(http.StatusOK, gin.H{"order": toOrderDTO(order), "quote": resp, "payment": toPaymentSelectDTO(payRes)})
}

func (h *Handler) OpenInstantOrderRefund(c *gin.Context) {
	if h.openAPISvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrOrdersDisabled.Error()})
		return
	}
	var payload struct {
		VPSID  int64  `json:"vps_id"`
		Reason string `json:"reason"`
	}
	if err := bindJSONOptional(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	ctx := apporder.WithOrderSource(c, apporder.OrderSourceUserAPIKey)
	order, amount, err := h.openAPISvc.InstantRefund(ctx, getUserID(c), payload.VPSID, payload.Reason)
	if err != nil {
		h.writeOpenOrderError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"order": toOrderDTO(order), "refund_amount": centsToFloat(amount)})
}

func (h *Handler) writeOpenOrderError(c *gin.Context, err error) {
	status := http.StatusBadRequest
	if errors.Is(err, appshared.ErrForbidden) || errors.Is(err, appshared.ErrRealNameRequired) {
		status = http.StatusForbidden
	}
	if errors.Is(err, appshared.ErrConflict) || errors.Is(err, appshared.ErrInsufficientBalance) {
		status = http.StatusConflict
	}
	c.JSON(status, gin.H{"error": err.Error()})
}
