package http

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

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
	wallet, err := h.walletSvc.GetWallet(c.Request.Context(), getUserID(c))
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
	items, total, err := h.walletSvc.ListTransactions(c.Request.Context(), getUserID(c), limit, offset)
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
		writeBodyError(c, err)
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
		methods, err := h.paymentSvc.ListUserMethodsByScene(c.Request.Context(), getUserID(c), "wallet")
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
	order, err := h.walletOrder.CreateRecharge(c.Request.Context(), getUserID(c), appshared.WalletOrderCreateInput{
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
		// The payment gateway receives Extra verbatim; derive client_ip/device
		// from the trusted request context instead of trusting client input.
		if ip := strings.TrimSpace(c.ClientIP()); ip != "" {
			payload.Extra["client_ip"] = ip
		}
		payload.Extra["device"] = detectEZPayDeviceFromUA(c.GetHeader("User-Agent"))
		returnURL, notifyURL := h.defaultWalletPaymentCallbackURLs(c, method)
		walletOrderNo := walletPaymentOrderNo(order.ID)
		payRes, err := h.paymentSvc.CreateProviderPaymentByScene(c.Request.Context(), "wallet", method, appshared.PaymentCreateRequest{
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
			if uerr := h.walletOrder.UpdateOrderMeta(c.Request.Context(), order.ID, string(metaBytes)); uerr != nil {
				// The provider payment already exists; do not fail the request
				// (the client would retry and duplicate it). Log so operators
				// can notice the missing payment_order_no/payment_trade_no.
				log.Printf("wallet recharge meta persist failed order_id=%d err=%v", order.ID, uerr)
			}
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
		writeBodyError(c, err)
		return
	}
	amount, err := parseAmountCents(payload.Amount)
	if err != nil || amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidAmount.Error()})
		return
	}
	order, err := h.walletOrder.CreateWithdraw(c.Request.Context(), getUserID(c), appshared.WalletOrderCreateInput{
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
	var query struct {
		Status string `form:"status"`
		Type   string `form:"type"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	// P3-42: status/type server-side filter; the frontend pages by API instead
	// of pulling 500 rows and slicing locally.
	items, total, err := h.walletOrder.ListUserOrdersFiltered(c.Request.Context(), getUserID(c), strings.TrimSpace(query.Status), strings.TrimSpace(query.Type), limit, offset)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toWalletOrderDTOs(items), "total": total})
}

func (h *Handler) WalletOrderSummary(c *gin.Context) {
	if h.walletOrder == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrWalletOrdersDisabled.Error()})
		return
	}
	summary, err := h.walletOrder.SummarizeUserOrders(c.Request.Context(), getUserID(c), time.Now())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"recharge": centsToFloat(summary.MonthRecharge),
		"withdraw": centsToFloat(summary.MonthWithdraw),
		"pending":  summary.Pending,
		"total":    summary.Total,
	})
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
	order, err := h.walletOrder.GetUserOrder(c.Request.Context(), getUserID(c), uri.ID)
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
	methods, err := h.paymentSvc.ListUserMethodsByScene(c.Request.Context(), getUserID(c), "wallet")
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
	// The payment gateway receives Extra verbatim; derive client_ip/device
	// from the trusted request context instead of trusting client input.
	if ip := strings.TrimSpace(c.ClientIP()); ip != "" {
		payload.Extra["client_ip"] = ip
	}
	payload.Extra["device"] = detectEZPayDeviceFromUA(c.GetHeader("User-Agent"))
	walletOrderNo := strings.TrimSpace(fmt.Sprint(meta["payment_order_no"]))
	if walletOrderNo == "" {
		// Deterministic order no derived from wallet order id.
		walletOrderNo = walletPaymentOrderNo(order.ID)
	}
	returnURL, notifyURL := h.defaultWalletPaymentCallbackURLs(c, method)
	payRes, err := h.paymentSvc.CreateProviderPaymentByScene(c.Request.Context(), "wallet", method, appshared.PaymentCreateRequest{
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
		if uerr := h.walletOrder.UpdateOrderMeta(c.Request.Context(), order.ID, string(metaBytes)); uerr != nil {
			// The provider payment already exists; failing the request would
			// make the client retry and create a duplicate payment.
			log.Printf("wallet order pay meta persist failed order_id=%d err=%v", order.ID, uerr)
		}
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
	order, err := h.walletOrder.CancelByUser(c.Request.Context(), getUserID(c), uri.ID, payload.Reason)
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

// walletOrderIDFromOrderNo extracts the wallet order id from the server-derived
// "WALLET-ORDER-<id>" payment order number, or 0 when the string is not of that
// shape.
func walletOrderIDFromOrderNo(orderNo string) int64 {
	if !strings.HasPrefix(orderNo, "WALLET-ORDER-") {
		return 0
	}
	id, err := strconv.ParseInt(strings.TrimPrefix(orderNo, "WALLET-ORDER-"), 10, 64)
	if err != nil || id <= 0 {
		return 0
	}
	return id
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
	rawRequest, err := readPaymentCallbackRequest(c)
	if err != nil {
		writePaymentCallbackReadError(c, err)
		return
	}
	result, err := h.paymentSvc.VerifyNotify(c.Request.Context(), provider, rawRequest)
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
	// Fast path: the payment order number is server-derived as
	// WALLET-ORDER-<id>, so resolve the wallet order by id directly instead of
	// scanning paginated lists (which was O(N) and truncated at 4000 rows per
	// status, so a deep recharge was never acknowledged and the gateway retried
	// forever).
	if id := walletOrderIDFromOrderNo(orderNo); id > 0 {
		if item, err := h.walletOrder.GetOrderByID(c.Request.Context(), id); err == nil &&
			item.Type == domain.WalletOrderRecharge && walletPaymentMatched(item, provider, orderNo, tradeNo) {
			if amountErr := h.paymentSvc.ValidateNotifyAmount(item.Amount, result.Amount); amountErr != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": amountErr.Error()})
				return
			}
			if item.Status == domain.WalletOrderApproved {
				c.JSON(http.StatusOK, gin.H{"ok": true, "trade_no": result.TradeNo})
				return
			}
			_, _, approveErr := h.walletOrder.Approve(c.Request.Context(), 0, item.ID)
			if approveErr != nil {
				if approveErr == appshared.ErrConflict {
					c.JSON(http.StatusConflict, gin.H{"error": appshared.ErrConflict.Error()})
					return
				}
				c.JSON(http.StatusBadRequest, gin.H{"error": approveErr.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"ok": true, "trade_no": result.TradeNo})
			return
		}
	}
	// Slow-path direct lookup by provider trade_no (P1-4): the gateway may echo
	// a rewritten out_trade_no, so match by the trade_no stored in the wallet
	// order meta instead of scanning bounded pages that truncate deep orders.
	if strings.TrimSpace(tradeNo) != "" {
		if item, err := h.walletOrder.GetOrderByPaymentTradeNo(c.Request.Context(), tradeNo); err == nil &&
			item.Type == domain.WalletOrderRecharge && walletPaymentMatched(item, provider, orderNo, tradeNo) {
			if amountErr := h.paymentSvc.ValidateNotifyAmount(item.Amount, result.Amount); amountErr != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": amountErr.Error()})
				return
			}
			if item.Status == domain.WalletOrderApproved {
				c.JSON(http.StatusOK, gin.H{"ok": true, "trade_no": result.TradeNo})
				return
			}
			_, _, approveErr := h.walletOrder.Approve(c.Request.Context(), 0, item.ID)
			if approveErr != nil {
				if approveErr == appshared.ErrConflict {
					c.JSON(http.StatusConflict, gin.H{"error": appshared.ErrConflict.Error()})
					return
				}
				c.JSON(http.StatusBadRequest, gin.H{"error": approveErr.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"ok": true, "trade_no": result.TradeNo})
			return
		}
	}
	limit := 200
	approvalConflict := false
	for _, statusFilter := range []string{
		string(domain.WalletOrderPendingReview),
		string(domain.WalletOrderApproved),
	} {
		offset := 0
		for i := 0; i < 20; i++ {
			items, total, listErr := h.walletOrder.ListAllOrders(c.Request.Context(), statusFilter, limit, offset)
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
				if amountErr := h.paymentSvc.ValidateNotifyAmount(item.Amount, result.Amount); amountErr != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": amountErr.Error()})
					return
				}
				// Idempotent ack: already approved recharge should always return success.
				if item.Status == domain.WalletOrderApproved {
					c.JSON(http.StatusOK, gin.H{"ok": true, "trade_no": result.TradeNo})
					return
				}
				_, _, approveErr := h.walletOrder.Approve(c.Request.Context(), 0, item.ID)
				if approveErr != nil {
					if approveErr == appshared.ErrConflict {
						approvalConflict = true
						continue
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
	if approvalConflict {
		c.JSON(http.StatusConflict, gin.H{"error": appshared.ErrConflict.Error()})
		return
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
	items, total, err := h.messageSvc.List(c.Request.Context(), getUserID(c), status, limit, offset)
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
	count, err := h.messageSvc.UnreadCount(c.Request.Context(), getUserID(c))
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
	if err := h.messageSvc.MarkRead(c.Request.Context(), getUserID(c), uri.ID); err != nil {
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
	if err := h.messageSvc.MarkAllRead(c.Request.Context(), getUserID(c)); err != nil {
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
	if err := h.orderSvc.CancelOrder(c.Request.Context(), getUserID(c), uri.ID); err != nil {
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
	orders, total, err := h.orderSvc.ListOrders(c.Request.Context(), filter, limit, offset)
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
	order, items, err := h.orderSvc.GetOrder(c.Request.Context(), uri.ID, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrOrderNotFound.Error()})
		return
	}
	var payments []domain.OrderPayment
	if h.orderSvc != nil {
		payments, _ = h.orderSvc.ListPaymentsForOrder(c.Request.Context(), getUserID(c), uri.ID)
	}
	c.JSON(http.StatusOK, gin.H{"order": toOrderDTO(order), "items": toOrderItemDTOs(items), "payments": toOrderPaymentDTOs(payments)})
}

func (h *Handler) OrderEvents(c *gin.Context) {
	var uri walletOrderIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	_, _, err := h.orderSvc.GetOrder(c.Request.Context(), uri.ID, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrOrderNotFound.Error()})
		return
	}
	last := c.GetHeader("Last-Event-ID")
	var lastSeq int64
	if last != "" {
		lastSeq, _ = strconv.ParseInt(last, 10, 64)
	}
	clearWriteDeadline(c.Writer)
	_ = h.broker.Stream(c.Request.Context(), c.Writer, uri.ID, lastSeq)
}

func (h *Handler) OrderRefresh(c *gin.Context) {
	var uri walletOrderIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	instances, err := h.orderSvc.RefreshOrder(c.Request.Context(), getUserID(c), uri.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// User-facing DTO variant (P2-4): this endpoint must not echo panel/VNC/OS
	// passwords. Credentials are only served by the explicit, audited
	// GET /api/v1/vps/:id/credentials endpoint.
	c.JSON(http.StatusOK, gin.H{"items": h.toUserVPSInstanceDTOsWithLifecycle(c, instances)})
}
