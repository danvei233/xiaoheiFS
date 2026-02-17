package http

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

func (h *Handler) WalletInfo(c *gin.Context) {
	if h.walletSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wallet disabled"})
		return
	}
	wallet, err := h.walletSvc.GetWallet(c, getUserID(c))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"wallet": toWalletDTO(wallet)})
}

func (h *Handler) WalletTransactions(c *gin.Context) {
	if h.walletSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wallet disabled"})
		return
	}
	limit, offset := paging(c)
	items, total, err := h.walletSvc.ListTransactions(c, getUserID(c), limit, offset)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toWalletTransactionDTOs(items), "total": total})
}

func (h *Handler) WalletRecharge(c *gin.Context) {
	if h.walletOrder == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wallet orders disabled"})
		return
	}
	var payload struct {
		Amount   any            `json:"amount"`
		Currency string         `json:"currency"`
		Note     string         `json:"note"`
		Meta     map[string]any `json:"meta"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	amount, err := parseAmountCents(payload.Amount)
	if err != nil || amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount"})
		return
	}
	order, err := h.walletOrder.CreateRecharge(c, getUserID(c), appshared.WalletOrderCreateInput{
		Amount:   amount,
		Currency: payload.Currency,
		Note:     payload.Note,
		Meta:     payload.Meta,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"order": toWalletOrderDTO(order)})
}

func (h *Handler) WalletWithdraw(c *gin.Context) {
	if h.walletOrder == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wallet orders disabled"})
		return
	}
	var payload struct {
		Amount   any            `json:"amount"`
		Currency string         `json:"currency"`
		Note     string         `json:"note"`
		Meta     map[string]any `json:"meta"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	amount, err := parseAmountCents(payload.Amount)
	if err != nil || amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount"})
		return
	}
	order, err := h.walletOrder.CreateWithdraw(c, getUserID(c), appshared.WalletOrderCreateInput{
		Amount:   amount,
		Currency: payload.Currency,
		Note:     payload.Note,
		Meta:     payload.Meta,
	})
	if err != nil {
		status := http.StatusBadRequest
		if err == appshared.ErrInsufficientBalance {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"order": toWalletOrderDTO(order)})
}

func (h *Handler) WalletOrders(c *gin.Context) {
	if h.walletOrder == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wallet orders disabled"})
		return
	}
	limit, offset := paging(c)
	items, total, err := h.walletOrder.ListUserOrders(c, getUserID(c), limit, offset)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toWalletOrderDTOs(items), "total": total})
}

func (h *Handler) Notifications(c *gin.Context) {
	if h.messageSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message center disabled"})
		return
	}
	status := strings.TrimSpace(c.Query("status"))
	limit, offset := paging(c)
	items, total, err := h.messageSvc.List(c, getUserID(c), status, limit, offset)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp := make([]NotificationDTO, 0, len(items))
	for _, item := range items {
		resp = append(resp, toNotificationDTO(item))
	}
	c.JSON(http.StatusOK, gin.H{"items": resp, "total": total})
}

func (h *Handler) NotificationsUnreadCount(c *gin.Context) {
	if h.messageSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message center disabled"})
		return
	}
	count, err := h.messageSvc.UnreadCount(c, getUserID(c))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"unread": count})
}

func (h *Handler) NotificationRead(c *gin.Context) {
	if h.messageSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message center disabled"})
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.messageSvc.MarkRead(c, getUserID(c), id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) NotificationReadAll(c *gin.Context) {
	if h.messageSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message center disabled"})
		return
	}
	if err := h.messageSvc.MarkAllRead(c, getUserID(c)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) OrderCancel(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.orderSvc.CancelOrder(c, getUserID(c), id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) OrderList(c *gin.Context) {
	limit, offset := paging(c)
	status := strings.TrimSpace(c.Query("status"))
	if status == "all" {
		status = ""
	}
	if status != "" &&
		status != string(domain.OrderStatusDraft) &&
		status != string(domain.OrderStatusPendingPayment) &&
		status != string(domain.OrderStatusPendingReview) &&
		status != string(domain.OrderStatusRejected) &&
		status != string(domain.OrderStatusApproved) &&
		status != string(domain.OrderStatusProvisioning) &&
		status != string(domain.OrderStatusActive) &&
		status != string(domain.OrderStatusFailed) &&
		status != string(domain.OrderStatusCanceled) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
		return
	}
	filter := appshared.OrderFilter{UserID: getUserID(c), Status: status}
	orders, total, err := h.orderSvc.ListOrders(c, filter, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toOrderDTOs(orders), "total": total})
}

func (h *Handler) OrderDetail(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	order, items, err := h.orderSvc.GetOrder(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	var payments []domain.OrderPayment
	if h.orderSvc != nil {
		payments, _ = h.orderSvc.ListPaymentsForOrder(c, getUserID(c), id)
	}
	c.JSON(http.StatusOK, gin.H{"order": toOrderDTO(order), "items": toOrderItemDTOs(items), "payments": toOrderPaymentDTOs(payments)})
}

func (h *Handler) OrderEvents(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	_, _, err := h.orderSvc.GetOrder(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	last := c.GetHeader("Last-Event-ID")
	var lastSeq int64
	if last != "" {
		lastSeq, _ = strconv.ParseInt(last, 10, 64)
	}
	_ = h.broker.Stream(c, c.Writer, id, lastSeq)
}

func (h *Handler) OrderRefresh(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	instances, err := h.orderSvc.RefreshOrder(c, getUserID(c), id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": h.toVPSInstanceDTOsWithLifecycle(c, instances)})
}
