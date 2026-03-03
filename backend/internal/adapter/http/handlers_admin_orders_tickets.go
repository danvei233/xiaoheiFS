package http

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

func (h *Handler) AdminOrders(c *gin.Context) {
	limit, offset := paging(c)
	var query struct {
		Status string `form:"status"`
		UserID *int64 `form:"user_id" binding:"omitempty,gt=0"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	filter := appshared.OrderFilter{}
	if query.Status != "" {
		filter.Status = strings.TrimSpace(query.Status)
	}
	if query.UserID != nil {
		filter.UserID = *query.UserID
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
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	order, items, err := h.orderSvc.GetOrderForAdmin(c, uri.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrOrderNotFound.Error()})
		return
	}
	var payments []domain.OrderPayment
	if h.orderSvc != nil {
		payments, _ = h.orderSvc.ListPaymentsForOrderAdmin(c, uri.ID)
	}
	var events []domain.OrderEvent
	if h.orderEventSvc != nil {
		events, _ = h.orderEventSvc.ListAfter(c, uri.ID, 0, 200)
	}
	c.JSON(http.StatusOK, gin.H{
		"order":    toOrderDTO(order),
		"items":    toOrderItemDTOs(items),
		"payments": toOrderPaymentDTOs(payments),
		"events":   toOrderEventDTOs(events),
	})
}

func (h *Handler) AdminOrderApprove(c *gin.Context) {
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	if err := h.orderSvc.ApproveOrder(c, getUserID(c), uri.ID); err != nil {
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
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	var payload struct {
		Reason string `json:"reason"`
	}
	if err := bindJSONOptional(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	if err := h.orderSvc.RejectOrder(c, getUserID(c), uri.ID, payload.Reason); err != nil {
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
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	if err := h.adminSvc.DeleteOrder(c, getUserID(c), uri.ID); err != nil {
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
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	var payload appshared.PaymentInput
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	payment, err := h.orderSvc.MarkPaid(c, getUserID(c), uri.ID, payload)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toOrderPaymentDTO(payment))
}

func (h *Handler) AdminOrderRetry(c *gin.Context) {
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	if err := h.orderSvc.RetryProvision(uri.ID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminTickets(c *gin.Context) {
	var query struct {
		Status  string `form:"status"`
		Keyword string `form:"q"`
		UserID  *int64 `form:"user_id" binding:"omitempty,gt=0"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	limit, offset := paging(c)
	items, total, err := h.ticketSvc.List(c, appshared.TicketFilter{
		UserID:  query.UserID,
		Status:  strings.TrimSpace(query.Status),
		Keyword: strings.TrimSpace(query.Keyword),
		Limit:   limit,
		Offset:  offset,
	})
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
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	ticket, messages, resources, err := h.ticketSvc.GetDetail(c, uri.ID)
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
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	ticket, err := h.ticketSvc.Get(c, uri.ID)
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
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	ticket, err := h.ticketSvc.Get(c, uri.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	var payload struct {
		Content string `json:"content" binding:"required"`
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
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	if err := h.ticketSvc.Delete(c, uri.ID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
