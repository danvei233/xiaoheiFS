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
		Status  string `form:"status"`
		UserID  *int64 `form:"user_id" binding:"omitempty,gt=0"`
		OrderNo string `form:"order_no"`
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
	// The frontend review page searches by order_no; previously the backend
	// silently ignored it (m059 F6).
	filter.OrderNo = strings.TrimSpace(query.OrderNo)
	orders, total, err := h.adminSvc.ListOrders(c.Request.Context(), filter, limit, offset)
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
	status, err := h.statusSvc.Status(c.Request.Context())
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
	order, items, err := h.orderSvc.GetOrderForAdmin(c.Request.Context(), uri.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrOrderNotFound.Error()})
		return
	}
	var payments []domain.OrderPayment
	if h.orderSvc != nil {
		payments, _ = h.orderSvc.ListPaymentsForOrderAdmin(c.Request.Context(), uri.ID)
	}
	var events []domain.OrderEvent
	eventsTotal := 0
	if h.orderEventSvc != nil {
		events, _ = h.orderEventSvc.ListAfter(c.Request.Context(), uri.ID, 0, 200)
		if total, err := h.orderEventSvc.CountByOrder(c.Request.Context(), uri.ID); err == nil {
			eventsTotal = int(total)
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"order":        toOrderDTO(order),
		"items":        toOrderItemDTOs(items),
		"payments":     toOrderPaymentDTOs(payments),
		"events":       toOrderEventDTOs(events),
		"events_total": eventsTotal,
	})
}

func (h *Handler) AdminOrderApprove(c *gin.Context) {
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	if err := h.orderSvc.ApproveOrder(c.Request.Context(), getUserID(c), uri.ID); err != nil {
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
		writeBodyError(c, err)
		return
	}
	if err := h.orderSvc.RejectOrder(c.Request.Context(), getUserID(c), uri.ID, payload.Reason); err != nil {
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
		has, err := h.permissionSvc.HasPermission(c.Request.Context(), getUserID(c), "order.delete")
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
	if err := h.adminSvc.DeleteOrder(c.Request.Context(), getUserID(c), uri.ID); err != nil {
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
		writeBodyError(c, err)
		return
	}
	payment, err := h.orderSvc.MarkPaid(c.Request.Context(), getUserID(c), uri.ID, payload)
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
	items, total, err := h.ticketSvc.List(c.Request.Context(), appshared.TicketFilter{
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
	ticket, messages, resources, err := h.ticketSvc.GetDetail(c.Request.Context(), uri.ID)
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
	ticket, err := h.ticketSvc.Get(c.Request.Context(), uri.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	var payload struct {
		Subject *string `json:"subject"`
		Status  *string `json:"status"`
	}
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
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
	if err := h.ticketSvc.AdminUpdate(c.Request.Context(), ticket); err != nil {
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
	ticket, err := h.ticketSvc.Get(c.Request.Context(), uri.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	var payload struct {
		Content string `json:"content" binding:"required"`
	}
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	msg, err := h.ticketSvc.AddMessage(c.Request.Context(), ticket, getUserID(c), "admin", payload.Content)
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
	if err := h.ticketSvc.Delete(c.Request.Context(), uri.ID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
