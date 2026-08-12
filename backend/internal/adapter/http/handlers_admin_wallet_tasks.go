package http

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

type adminUserIDURI struct {
	UserID int64 `uri:"user_id" binding:"required,gt=0"`
}

type adminTaskKeyURI struct {
	Key string `uri:"key" binding:"required"`
}

func (h *Handler) AdminWalletInfo(c *gin.Context) {
	if h.walletSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrWalletDisabled.Error()})
		return
	}
	var uri adminUserIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	wallet, err := h.walletSvc.GetWallet(c.Request.Context(), uri.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"wallet": toWalletDTO(wallet)})
}

func (h *Handler) AdminWalletAdjust(c *gin.Context) {
	if h.walletSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrWalletDisabled.Error()})
		return
	}
	var uri adminUserIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	var payload struct {
		Amount any    `json:"amount"`
		Note   string `json:"note"`
	}
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	amount, err := parseAmountCents(payload.Amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidAmount.Error()})
		return
	}
	wallet, err := h.walletSvc.AdjustBalance(c.Request.Context(), getUserID(c), uri.UserID, amount, payload.Note)
	if err != nil {
		status := http.StatusBadRequest
		if err == appshared.ErrInsufficientBalance {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"wallet": toWalletDTO(wallet)})
}

func (h *Handler) AdminWalletTransactions(c *gin.Context) {
	if h.walletSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrWalletDisabled.Error()})
		return
	}
	var uri adminUserIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	limit, offset := paging(c)
	items, total, err := h.walletSvc.ListTransactions(c.Request.Context(), uri.UserID, limit, offset)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toWalletTransactionDTOs(items), "total": total})
}

func (h *Handler) AdminWalletOrders(c *gin.Context) {
	if h.walletOrder == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrWalletOrdersDisabled.Error()})
		return
	}
	var query struct {
		Status string `form:"status"`
		UserID *int64 `form:"user_id" binding:"omitempty,gt=0"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	status := strings.TrimSpace(query.Status)
	limit, offset := paging(c)
	var (
		items []domain.WalletOrder
		total int
		err   error
	)
	if query.UserID != nil {
		items, total, err = h.walletOrder.ListUserOrders(c.Request.Context(), *query.UserID, limit, offset)
	} else {
		items, total, err = h.walletOrder.ListAllOrders(c.Request.Context(), status, limit, offset)
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toWalletOrderDTOs(items), "total": total})
}

// AdminWalletRefundCleanups lists the durable refund-cleanup outbox so pending
// and failed host deletions are visible and an operator can reconcile them
// instead of a "refunded" VPS staying usable forever.
func (h *Handler) AdminWalletRefundCleanups(c *gin.Context) {
	if h.walletOrder == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrWalletOrdersDisabled.Error()})
		return
	}
	limit, offset := paging(c)
	items, total, err := h.walletOrder.ListCleanupJobs(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(items))
	for _, job := range items {
		out = append(out, gin.H{
			"id":              job.ID,
			"wallet_order_id": job.WalletOrderID,
			"user_id":         job.UserID,
			"vps_id":          job.VPSID,
			"host_id":         job.HostID,
			"status":          job.Status,
			"attempts":        job.Attempts,
			"last_error_code": job.LastErrorCode,
			"last_error":      job.LastError,
			"created_at":      job.CreatedAt,
			"updated_at":      job.UpdatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": out, "total": total})
}

func (h *Handler) AdminWalletOrderApprove(c *gin.Context) {
	if h.walletOrder == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrWalletOrdersDisabled.Error()})
		return
	}
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	order, wallet, err := h.walletOrder.Approve(c.Request.Context(), getUserID(c), uri.ID)
	if err != nil {
		status := http.StatusBadRequest
		if err == appshared.ErrConflict {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	resp := gin.H{"order": toWalletOrderDTO(order)}
	if wallet != nil {
		resp["wallet"] = toWalletDTO(*wallet)
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) AdminWalletOrderReject(c *gin.Context) {
	if h.walletOrder == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrWalletOrdersDisabled.Error()})
		return
	}
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
	if err := h.walletOrder.Reject(c.Request.Context(), getUserID(c), uri.ID, payload.Reason); err != nil {
		status := http.StatusBadRequest
		if err == appshared.ErrConflict {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminScheduledTasks(c *gin.Context) {
	if h.taskSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrScheduledTasksDisabled.Error()})
		return
	}
	items, err := h.taskSvc.ListTasks(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) AdminScheduledTaskUpdate(c *gin.Context) {
	if h.taskSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrScheduledTasksDisabled.Error()})
		return
	}
	var uri adminTaskKeyURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	var payload appshared.ScheduledTaskUpdate
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	item, err := h.taskSvc.UpdateTask(c.Request.Context(), uri.Key, payload)
	if err != nil {
		status := http.StatusBadRequest
		if err == appshared.ErrNotFound {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *Handler) AdminScheduledTaskRuns(c *gin.Context) {
	if h.taskSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrScheduledTasksDisabled.Error()})
		return
	}
	var uri adminTaskKeyURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	var query struct {
		Limit int `form:"limit" binding:"omitempty,gte=0,lte=500"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	items, err := h.taskSvc.ListTaskRuns(c.Request.Context(), uri.Key, query.Limit)
	if err != nil {
		status := http.StatusBadRequest
		if err == appshared.ErrInvalidInput {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}
