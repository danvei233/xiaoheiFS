package http

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

func (h *Handler) AdminWalletInfo(c *gin.Context) {
	if h.walletSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wallet disabled"})
		return
	}
	userID, _ := strconv.ParseInt(c.Param("user_id"), 10, 64)
	wallet, err := h.walletSvc.GetWallet(c, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"wallet": toWalletDTO(wallet)})
}

func (h *Handler) AdminWalletAdjust(c *gin.Context) {
	if h.walletSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wallet disabled"})
		return
	}
	userID, _ := strconv.ParseInt(c.Param("user_id"), 10, 64)
	var payload struct {
		Amount any    `json:"amount"`
		Note   string `json:"note"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	amount, err := parseAmountCents(payload.Amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount"})
		return
	}
	wallet, err := h.walletSvc.AdjustBalance(c, getUserID(c), userID, amount, payload.Note)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "wallet disabled"})
		return
	}
	userID, _ := strconv.ParseInt(c.Param("user_id"), 10, 64)
	limit, offset := paging(c)
	items, total, err := h.walletSvc.ListTransactions(c, userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toWalletTransactionDTOs(items), "total": total})
}

func (h *Handler) AdminWalletOrders(c *gin.Context) {
	if h.walletOrder == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wallet orders disabled"})
		return
	}
	status := strings.TrimSpace(c.Query("status"))
	userID, _ := strconv.ParseInt(c.Query("user_id"), 10, 64)
	limit, offset := paging(c)
	var (
		items []domain.WalletOrder
		total int
		err   error
	)
	if userID > 0 {
		items, total, err = h.walletOrder.ListUserOrders(c, userID, limit, offset)
	} else {
		items, total, err = h.walletOrder.ListAllOrders(c, status, limit, offset)
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toWalletOrderDTOs(items), "total": total})
}

func (h *Handler) AdminWalletOrderApprove(c *gin.Context) {
	if h.walletOrder == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wallet orders disabled"})
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	order, wallet, err := h.walletOrder.Approve(c, getUserID(c), id)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "wallet orders disabled"})
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		Reason string `json:"reason"`
	}
	if err := bindJSONOptional(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if err := h.walletOrder.Reject(c, getUserID(c), id, payload.Reason); err != nil {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "scheduled tasks disabled"})
		return
	}
	items, err := h.taskSvc.ListTasks(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) AdminScheduledTaskUpdate(c *gin.Context) {
	if h.taskSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "scheduled tasks disabled"})
		return
	}
	key := c.Param("key")
	var payload appshared.ScheduledTaskUpdate
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	item, err := h.taskSvc.UpdateTask(c, key, payload)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "scheduled tasks disabled"})
		return
	}
	key := c.Param("key")
	limit, _ := strconv.Atoi(c.Query("limit"))
	items, err := h.taskSvc.ListTaskRuns(c, key, limit)
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
