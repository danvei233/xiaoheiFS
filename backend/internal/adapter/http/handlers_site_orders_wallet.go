package http

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"

	"github.com/gin-gonic/gin"
)

type walletOrderIDURI struct {
	ID int64 `uri:"id" binding:"required,gt=0"`
}

type walletProviderURI struct {
	Provider string `uri:"provider" binding:"required,max=64"`
}

type notificationStatusQuery struct {
	Status string `form:"status" binding:"omitempty,oneof=read unread all"`
}

type orderListQuery struct {
	Status string `form:"status" binding:"omitempty,oneof=all draft pending_payment pending_review rejected approved provisioning active failed canceled"`
}

func (h *Handler) WalletInfo(c *gin.Context) {
	if h.walletSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrWalletDisabled.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrWalletDisabled.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrWalletOrdersDisabled.Error()})
		return
	}
	var payload struct {
		Amount   any               `json:"amount"`
		Currency string            `json:"currency"`
		Note     string            `json:"note"`
		Meta     map[string]any    `json:"meta"`
		Method   string            `json:"method"`
		Extra    map[string]string `json:"extra"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	amount, err := parseAmountCents(payload.Amount)
	if err != nil || amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidAmount.Error()})
		return
	}
	method := strings.TrimSpace(payload.Method)
	if method == "" {
		method = "approval"
	}
	if h.paymentSvc != nil {
		methods, err := h.paymentSvc.ListUserMethodsByScene(c, getUserID(c), "wallet")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		allowed := method == "approval"
		for _, m := range methods {
			key := strings.TrimSpace(m.Key)
			if key == "" || key == "balance" {
				continue
			}
			if key == method {
				allowed = true
				break
			}
		}
		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{"error": appshared.ErrForbidden.Error()})
			return
		}
	}
	meta := payload.Meta
	if meta == nil {
		meta = map[string]any{}
	}
	meta["payment_method"] = method
	resp := gin.H{}
	// Create the wallet order first so we have a stable ID for deterministic order no.
	order, err := h.walletOrder.CreateRecharge(c, getUserID(c), appshared.WalletOrderCreateInput{
		Amount:   amount,
		Currency: payload.Currency,
		Note:     payload.Note,
		Meta:     meta,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if method != "approval" && h.paymentSvc != nil {
		if payload.Extra == nil {
			payload.Extra = map[string]string{}
		}
		if strings.TrimSpace(payload.Extra["client_ip"]) == "" {
			payload.Extra["client_ip"] = strings.TrimSpace(c.ClientIP())
		}
		returnURL, notifyURL := h.defaultWalletPaymentCallbackURLs(c, method)
		walletOrderNo := walletPaymentOrderNo(order.ID)
		payRes, err := h.paymentSvc.CreateProviderPaymentByScene(c, "wallet", method, appshared.PaymentCreateRequest{
			OrderNo:   walletOrderNo,
			UserID:    getUserID(c),
			Amount:    amount,
			Currency:  payload.Currency,
			Subject:   fmt.Sprintf("Wallet Recharge %s", walletOrderNo),
			ReturnURL: returnURL,
			NotifyURL: notifyURL,
			Extra:     payload.Extra,
		})
		if err != nil {
			status := http.StatusBadRequest
			if err == appshared.ErrForbidden {
				status = http.StatusForbidden
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		meta["payment_order_no"] = walletOrderNo
		meta["payment_trade_no"] = payRes.TradeNo
		meta["payment_pay_url"] = payRes.PayURL
		if len(payRes.Extra) > 0 {
			meta["payment_extra"] = payRes.Extra
		}
		if metaBytes, merr := encodeMapJSON(meta); merr == nil {
			_ = h.walletOrder.UpdateOrderMeta(c, order.ID, string(metaBytes))
			order.MetaJSON = string(metaBytes)
		}
		resp["payment"] = toPaymentSelectDTO(appshared.PaymentSelectResult{
			Method:  method,
			Status:  "pending_payment",
			TradeNo: payRes.TradeNo,
			PayURL:  payRes.PayURL,
			Extra:   payRes.Extra,
		})
	}
	resp["order"] = toWalletOrderDTO(order)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) WalletWithdraw(c *gin.Context) {
	if h.walletOrder == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrWalletOrdersDisabled.Error()})
		return
	}
	var payload struct {
		Amount   any            `json:"amount"`
		Currency string         `json:"currency"`
		Note     string         `json:"note"`
		Meta     map[string]any `json:"meta"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	amount, err := parseAmountCents(payload.Amount)
	if err != nil || amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidAmount.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrWalletOrdersDisabled.Error()})
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

func (h *Handler) WalletOrderPay(c *gin.Context) {
	if h.walletOrder == nil || h.paymentSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrWalletOrdersDisabled.Error()})
		return
	}
	var uri walletOrderIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	var payload struct {
		Extra map[string]string `json:"extra"`
	}
	_ = bindJSONOptional(c, &payload)
	order, err := h.walletOrder.GetUserOrder(c, getUserID(c), uri.ID)
	if err != nil {
		status := http.StatusBadRequest
		if err == appshared.ErrForbidden {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	if order.Type != domain.WalletOrderRecharge || order.Status != domain.WalletOrderPendingReview {
		c.JSON(http.StatusConflict, gin.H{"error": appshared.ErrConflict.Error()})
		return
	}
	meta := parseMapJSON(order.MetaJSON)
	method := strings.TrimSpace(fmt.Sprint(meta["payment_method"]))
	if method == "" || method == "approval" || method == "balance" {
		c.JSON(http.StatusBadRequest, gin.H{"error": appshared.ErrInvalidInput.Error()})
		return
	}
	methods, err := h.paymentSvc.ListUserMethodsByScene(c, getUserID(c), "wallet")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	allowed := false
	for _, m := range methods {
		if strings.TrimSpace(m.Key) == method {
			allowed = true
			break
		}
	}
	if !allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": appshared.ErrForbidden.Error()})
		return
	}
	if payload.Extra == nil {
		payload.Extra = map[string]string{}
	}
	if strings.TrimSpace(payload.Extra["client_ip"]) == "" {
		payload.Extra["client_ip"] = strings.TrimSpace(c.ClientIP())
	}
	walletOrderNo := strings.TrimSpace(fmt.Sprint(meta["payment_order_no"]))
	if walletOrderNo == "" {
		// Deterministic order no derived from wallet order id.
		walletOrderNo = walletPaymentOrderNo(order.ID)
	}
	returnURL, notifyURL := h.defaultWalletPaymentCallbackURLs(c, method)
	payRes, err := h.paymentSvc.CreateProviderPaymentByScene(c, "wallet", method, appshared.PaymentCreateRequest{
		OrderNo:   walletOrderNo,
		UserID:    getUserID(c),
		Amount:    order.Amount,
		Currency:  order.Currency,
		Subject:   fmt.Sprintf("Wallet Recharge %s", walletOrderNo),
		ReturnURL: returnURL,
		NotifyURL: notifyURL,
		Extra:     payload.Extra,
	})
	if err != nil {
		status := http.StatusBadRequest
		if err == appshared.ErrForbidden {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	// Update order meta with payment info so notify callback can match.
	meta["payment_order_no"] = walletOrderNo
	meta["payment_trade_no"] = payRes.TradeNo
	meta["payment_pay_url"] = payRes.PayURL
	if len(payRes.Extra) > 0 {
		meta["payment_extra"] = payRes.Extra
	}
	if metaBytes, merr := encodeMapJSON(meta); merr == nil {
		_ = h.walletOrder.UpdateOrderMeta(c, order.ID, string(metaBytes))
	}
	c.JSON(http.StatusOK, gin.H{
		"payment": toPaymentSelectDTO(appshared.PaymentSelectResult{
			Method:  method,
			Status:  "pending_payment",
			TradeNo: payRes.TradeNo,
			PayURL:  payRes.PayURL,
			Extra:   payRes.Extra,
		}),
	})
}

func (h *Handler) WalletOrderCancel(c *gin.Context) {
	if h.walletOrder == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrWalletOrdersDisabled.Error()})
		return
	}
	var uri walletOrderIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	var payload struct {
		Reason string `json:"reason"`
	}
	_ = bindJSONOptional(c, &payload)
	order, err := h.walletOrder.CancelByUser(c, getUserID(c), uri.ID, payload.Reason)
	if err != nil {
		status := http.StatusBadRequest
		if err == appshared.ErrForbidden {
			status = http.StatusForbidden
		} else if err == appshared.ErrConflict {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"order": toWalletOrderDTO(order), "ok": true})
}

func (h *Handler) defaultWalletPaymentCallbackURLs(c *gin.Context, method string) (string, string) {
	base := h.defaultPaymentCallbackBaseURL(c)
	if base == "" {
		return "", ""
	}
	base = strings.TrimRight(base, "/")
	return base + "/console/billing", base + "/api/v1/wallet/payments/notify/" + strings.TrimSpace(method)
}

func (h *Handler) defaultOrderPaymentCallbackURLs(c *gin.Context, orderID int64, method string) (string, string) {
	base := h.defaultPaymentCallbackBaseURL(c)
	if base == "" {
		return "", ""
	}
	base = strings.TrimRight(base, "/")
	return base + "/console/orders/" + strconv.FormatInt(orderID, 10), base + "/api/v1/payments/notify/" + strings.TrimSpace(method)
}

func (h *Handler) defaultPaymentCallbackBaseURL(c *gin.Context) string {
	if v := buildCallbackBaseURL(strings.TrimSpace(h.getSettingValueByKey(c, "site_url"))); v != "" {
		return v
	}
	return ""
}

func buildCallbackBaseURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	if strings.TrimSpace(u.Host) == "" {
		return ""
	}
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawQuery = ""
	u.Fragment = ""
	return strings.TrimRight(u.String(), "/")
}

func walletPaymentOrderNo(orderID int64) string {
	return fmt.Sprintf("WALLET-ORDER-%d", orderID)
}

func walletPaymentMatched(item domain.WalletOrder, provider, orderNo, tradeNo string) bool {
	meta := parseMapJSON(item.MetaJSON)
	metaMethod := strings.TrimSpace(fmt.Sprint(meta["payment_method"]))
	if metaMethod != provider {
		return false
	}
	metaOrderNo := strings.TrimSpace(fmt.Sprint(meta["payment_order_no"]))
	metaTradeNo := strings.TrimSpace(fmt.Sprint(meta["payment_trade_no"]))
	if orderNo != "" && (metaOrderNo == orderNo || walletPaymentOrderNo(item.ID) == orderNo) {
		return true
	}
	if tradeNo != "" && metaTradeNo == tradeNo {
		return true
	}
	return false
}

func (h *Handler) WalletPaymentNotify(c *gin.Context) {
	if h.paymentSvc == nil || h.walletOrder == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrWalletOrdersDisabled.Error()})
		return
	}
	var uri walletProviderURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	provider := strings.TrimSpace(uri.Provider)
	body, _ := io.ReadAll(c.Request.Body)
	headers := map[string][]string{}
	for k, v := range c.Request.Header {
		copied := make([]string, len(v))
		copy(copied, v)
		headers[k] = copied
	}
	result, err := h.paymentSvc.VerifyNotify(c, provider, appshared.RawHTTPRequest{
		Method:   c.Request.Method,
		Path:     c.Request.URL.Path,
		RawQuery: c.Request.URL.RawQuery,
		Headers:  headers,
		Body:     body,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	orderNo := strings.TrimSpace(result.OrderNo)
	tradeNo := strings.TrimSpace(result.TradeNo)
	if orderNo == "" && tradeNo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": appshared.ErrInvalidInput.Error()})
		return
	}
	limit := 200
	for _, statusFilter := range []string{
		string(domain.WalletOrderPendingReview),
		string(domain.WalletOrderApproved),
	} {
		offset := 0
		for i := 0; i < 20; i++ {
			items, total, listErr := h.walletOrder.ListAllOrders(c, statusFilter, limit, offset)
			if listErr != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": listErr.Error()})
				return
			}
			for _, item := range items {
				if item.Type != domain.WalletOrderRecharge {
					continue
				}
				if !walletPaymentMatched(item, provider, orderNo, tradeNo) {
					continue
				}
				// Idempotent ack: already approved recharge should always return success.
				if item.Status == domain.WalletOrderApproved {
					c.JSON(http.StatusOK, gin.H{"ok": true, "trade_no": result.TradeNo})
					return
				}
				_, _, approveErr := h.walletOrder.Approve(c, 0, item.ID)
				if approveErr != nil {
					if approveErr == appshared.ErrConflict {
						c.JSON(http.StatusOK, gin.H{"ok": true, "trade_no": result.TradeNo})
						return
					}
					c.JSON(http.StatusBadRequest, gin.H{"error": approveErr.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{"ok": true, "trade_no": result.TradeNo})
				return
			}
			offset += len(items)
			if offset >= total || len(items) == 0 {
				break
			}
		}
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": appshared.ErrInvalidInput.Error()})
}

func (h *Handler) Notifications(c *gin.Context) {
	if h.messageSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrMessageCenterDisabled.Error()})
		return
	}
	var query notificationStatusQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	status := strings.TrimSpace(query.Status)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrMessageCenterDisabled.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrMessageCenterDisabled.Error()})
		return
	}
	var uri walletOrderIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	if err := h.messageSvc.MarkRead(c, getUserID(c), uri.ID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) NotificationReadAll(c *gin.Context) {
	if h.messageSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrMessageCenterDisabled.Error()})
		return
	}
	if err := h.messageSvc.MarkAllRead(c, getUserID(c)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) OrderCancel(c *gin.Context) {
	var uri walletOrderIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	if err := h.orderSvc.CancelOrder(c, getUserID(c), uri.ID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) OrderList(c *gin.Context) {
	limit, offset := paging(c)
	var query orderListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidStatus.Error()})
		return
	}
	status := strings.TrimSpace(query.Status)
	if status == "all" {
		status = ""
	}
	filter := appshared.OrderFilter{UserID: getUserID(c), Status: status}
	orders, total, err := h.orderSvc.ListOrders(c, filter, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrListError.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toOrderDTOs(orders), "total": total})
}

func (h *Handler) OrderDetail(c *gin.Context) {
	var uri walletOrderIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	order, items, err := h.orderSvc.GetOrder(c, uri.ID, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrOrderNotFound.Error()})
		return
	}
	var payments []domain.OrderPayment
	if h.orderSvc != nil {
		payments, _ = h.orderSvc.ListPaymentsForOrder(c, getUserID(c), uri.ID)
	}
	c.JSON(http.StatusOK, gin.H{"order": toOrderDTO(order), "items": toOrderItemDTOs(items), "payments": toOrderPaymentDTOs(payments)})
}

func (h *Handler) OrderEvents(c *gin.Context) {
	var uri walletOrderIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	_, _, err := h.orderSvc.GetOrder(c, uri.ID, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrOrderNotFound.Error()})
		return
	}
	last := c.GetHeader("Last-Event-ID")
	var lastSeq int64
	if last != "" {
		lastSeq, _ = strconv.ParseInt(last, 10, 64)
	}
	_ = h.broker.Stream(c, c.Writer, uri.ID, lastSeq)
}

func (h *Handler) OrderRefresh(c *gin.Context) {
	var uri walletOrderIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	instances, err := h.orderSvc.RefreshOrder(c, getUserID(c), uri.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": h.toVPSInstanceDTOsWithLifecycle(c, instances)})
}
