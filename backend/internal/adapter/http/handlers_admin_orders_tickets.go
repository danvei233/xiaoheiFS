package http

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"strings"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

func (h *Handler) AdminOrders(c *gin.Context) {
	limit, offset := paging(c)
	filter := appshared.OrderFilter{}
	if v := c.Query("status"); v != "" {
		filter.Status = v
	}
	if v := c.Query("user_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			filter.UserID = id
		}
	}
	orders, total, err := h.adminSvc.ListOrders(c, filter, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrListError.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toOrderDTOs(orders), "total": total})
}

func (h *Handler) AdminServerStatus(c *gin.Context) {
	if h.statusSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrStatusDisabled.Error()})
		return
	}
	status, err := h.statusSvc.Status(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toServerStatusDTO(status))
}

func (h *Handler) AdminOrderDetail(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	order, items, err := h.orderSvc.GetOrderForAdmin(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrOrderNotFound.Error()})
		return
	}
	var payments []domain.OrderPayment
	if h.orderSvc != nil {
		payments, _ = h.orderSvc.ListPaymentsForOrderAdmin(c, id)
	}
	var events []domain.OrderEvent
	if h.orderEventSvc != nil {
		events, _ = h.orderEventSvc.ListAfter(c, id, 0, 200)
	}
	c.JSON(http.StatusOK, gin.H{
		"order":    toOrderDTO(order),
		"items":    toOrderItemDTOs(items),
		"payments": toOrderPaymentDTOs(payments),
		"events":   toOrderEventDTOs(events),
	})
}

func (h *Handler) AdminOrderApprove(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.orderSvc.ApproveOrder(c, getUserID(c), id); err != nil {
		status := http.StatusBadRequest
		msg := err.Error()
		if err == appshared.ErrConflict || err == appshared.ErrResizeInProgress {
			status = http.StatusConflict
			if err == appshared.ErrConflict {
				msg = "order status not editable"
			}
		}
		c.JSON(status, gin.H{"error": msg})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminOrderReject(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		Reason string `json:"reason"`
	}
	if err := bindJSONOptional(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	if err := h.orderSvc.RejectOrder(c, getUserID(c), id, payload.Reason); err != nil {
		status := http.StatusBadRequest
		msg := err.Error()
		if err == appshared.ErrConflict {
			status = http.StatusConflict
			msg = "order status not editable"
		}
		c.JSON(status, gin.H{"error": msg})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminOrderDelete(c *gin.Context) {
	if h.permissionSvc != nil {
		has, err := h.permissionSvc.HasPermission(c, getUserID(c), "order.delete")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrPermissionCheckFailed.Error()})
			return
		}
		if !has {
			c.JSON(http.StatusForbidden, gin.H{"error": domain.ErrPermissionDenied.Error()})
			return
		}
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.adminSvc.DeleteOrder(c, getUserID(c), id); err != nil {
		status := http.StatusBadRequest
		msg := err.Error()
		if err == appshared.ErrNotFound {
			status = http.StatusNotFound
		}
		if err == appshared.ErrConflict {
			status = http.StatusConflict
			msg = "approved order cannot be deleted"
		}
		c.JSON(status, gin.H{"error": msg})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminOrderMarkPaid(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload appshared.PaymentInput
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	payment, err := h.orderSvc.MarkPaid(c, getUserID(c), id, payload)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toOrderPaymentDTO(payment))
}

func (h *Handler) AdminOrderRetry(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.orderSvc.RetryProvision(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminTickets(c *gin.Context) {
	status := strings.TrimSpace(c.Query("status"))
	keyword := strings.TrimSpace(c.Query("q"))
	userIDRaw := strings.TrimSpace(c.Query("user_id"))
	limit, offset := paging(c)
	var userID *int64
	if userIDRaw != "" {
		if v, err := strconv.ParseInt(userIDRaw, 10, 64); err == nil {
			userID = &v
		}
	}
	items, total, err := h.ticketSvc.List(c, appshared.TicketFilter{UserID: userID, Status: status, Keyword: keyword, Limit: limit, Offset: offset})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp := make([]TicketDTO, 0, len(items))
	for _, item := range items {
		resp = append(resp, toTicketDTO(item))
	}
	c.JSON(http.StatusOK, gin.H{"items": resp, "total": total})
}

func (h *Handler) AdminTicketDetail(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	ticket, messages, resources, err := h.ticketSvc.GetDetail(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	msgDTOs := make([]TicketMessageDTO, 0, len(messages))
	for _, msg := range messages {
		msgDTOs = append(msgDTOs, toTicketMessageDTO(msg, msg.SenderName, msg.SenderQQ))
	}
	resDTOs := make([]TicketResourceDTO, 0, len(resources))
	for _, res := range resources {
		resDTOs = append(resDTOs, toTicketResourceDTO(res))
	}
	c.JSON(http.StatusOK, gin.H{"ticket": toTicketDTO(ticket), "messages": msgDTOs, "resources": resDTOs})
}

func (h *Handler) AdminTicketUpdate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	ticket, err := h.ticketSvc.Get(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	var payload struct {
		Subject *string `json:"subject"`
		Status  *string `json:"status"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	if payload.Subject != nil {
		ticket.Subject = strings.TrimSpace(*payload.Subject)
	}
	if payload.Status != nil {
		ticket.Status = strings.TrimSpace(*payload.Status)
	}
	if ticket.Subject == "" || ticket.Status == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrSubjectAndStatusRequired.Error()})
		return
	}
	if err := h.ticketSvc.AdminUpdate(c, ticket); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toTicketDTO(ticket))
}

func (h *Handler) AdminTicketMessageCreate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	ticket, err := h.ticketSvc.Get(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	var payload struct {
		Content string `json:"content"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	msg, err := h.ticketSvc.AddMessage(c, ticket, getUserID(c), "admin", payload.Content)
	if err != nil {
		if err == appshared.ErrInvalidInput {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toTicketMessageDTO(msg, msg.SenderName, msg.SenderQQ))
}

func (h *Handler) AdminTicketDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.ticketSvc.Delete(c, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
