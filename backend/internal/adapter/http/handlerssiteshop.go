package http

import (
	"context"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

func (h *Handler) Catalog(c *gin.Context) {
	goodsTypeID, _ := strconv.ParseInt(c.Query("goods_type_id"), 10, 64)
	regions, plans, packages, images, cycles, err := h.catalogSvc.Catalog(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "catalog error"})
		return
	}
	if goodsTypeID > 0 {
		filteredRegions := make([]domain.Region, 0, len(regions))
		for _, r := range regions {
			if r.GoodsTypeID == goodsTypeID {
				filteredRegions = append(filteredRegions, r)
			}
		}
		regions = filteredRegions
		filteredPlans := make([]domain.PlanGroup, 0, len(plans))
		for _, p := range plans {
			if p.GoodsTypeID == goodsTypeID {
				filteredPlans = append(filteredPlans, p)
			}
		}
		plans = filteredPlans
		filteredPackages := make([]domain.Package, 0, len(packages))
		for _, pkg := range packages {
			if pkg.GoodsTypeID == goodsTypeID {
				filteredPackages = append(filteredPackages, pkg)
			}
		}
		packages = filteredPackages
	}
	plans = filterVisiblePlanGroups(plans)
	packages = filterVisiblePackages(packages, plans)
	if len(plans) == 0 {
		images = []domain.SystemImage{}
	} else {
		images = filterEnabledSystemImages(images, plans)
	}
	var goodsTypes []domain.GoodsType
	if h.goodsTypes != nil {
		items, _ := h.goodsTypes.List(c)
		for _, it := range items {
			if it.Active {
				goodsTypes = append(goodsTypes, it)
			}
		}
		sort.SliceStable(goodsTypes, func(i, j int) bool {
			if goodsTypes[i].SortOrder != goodsTypes[j].SortOrder {
				return goodsTypes[i].SortOrder < goodsTypes[j].SortOrder
			}
			return goodsTypes[i].ID < goodsTypes[j].ID
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"goods_types":    goodsTypes,
		"regions":        toRegionDTOs(regions),
		"plan_groups":    toPlanGroupDTOs(plans),
		"packages":       toPackageDTOs(packages),
		"system_images":  toSystemImageDTOs(images),
		"billing_cycles": toBillingCycleDTOs(cycles),
	})
}

func (h *Handler) GoodsTypes(c *gin.Context) {
	if h.goodsTypes == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	items, err := h.goodsTypes.List(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list error"})
		return
	}
	active := make([]domain.GoodsType, 0, len(items))
	for _, it := range items {
		if it.Active {
			active = append(active, it)
		}
	}
	sort.SliceStable(active, func(i, j int) bool {
		if active[i].SortOrder != active[j].SortOrder {
			return active[i].SortOrder < active[j].SortOrder
		}
		return active[i].ID < active[j].ID
	})
	c.JSON(http.StatusOK, gin.H{"items": active})
}

func (h *Handler) defaultGoodsTypeID(ctx context.Context) int64 {
	if h.goodsTypes == nil {
		return 0
	}
	items, err := h.goodsTypes.List(ctx)
	if err != nil || len(items) == 0 {
		return 0
	}
	var best domain.GoodsType
	for _, it := range items {
		if !it.Active {
			continue
		}
		if best.ID == 0 || it.SortOrder < best.SortOrder || (it.SortOrder == best.SortOrder && it.ID < best.ID) {
			best = it
		}
	}
	if best.ID > 0 {
		return best.ID
	}
	for _, it := range items {
		if best.ID == 0 || it.SortOrder < best.SortOrder || (it.SortOrder == best.SortOrder && it.ID < best.ID) {
			best = it
		}
	}
	return best.ID
}

func (h *Handler) SystemImages(c *gin.Context) {
	lineID, _ := strconv.ParseInt(c.Query("line_id"), 10, 64)
	planGroupID, _ := strconv.ParseInt(c.Query("plan_group_id"), 10, 64)
	if planGroupID > 0 {
		plan, err := h.catalogSvc.GetPlanGroup(c, planGroupID)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"items": []SystemImageDTO{}})
			return
		}
		if !plan.Active || !plan.Visible || plan.LineID <= 0 {
			c.JSON(http.StatusOK, gin.H{"items": []SystemImageDTO{}})
			return
		}
		lineID = plan.LineID
	}
	items, err := h.catalogSvc.ListSystemImages(c, lineID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list error"})
		return
	}
	items = filterEnabledSystemImages(items, nil)
	c.JSON(http.StatusOK, gin.H{"items": toSystemImageDTOs(items)})
}

func (h *Handler) PlanGroups(c *gin.Context) {
	regionID, _ := strconv.ParseInt(c.Query("region_id"), 10, 64)
	goodsTypeID, _ := strconv.ParseInt(c.Query("goods_type_id"), 10, 64)
	items, err := h.catalogSvc.ListPlanGroups(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list error"})
		return
	}
	items = filterVisiblePlanGroups(items)
	if goodsTypeID > 0 {
		filtered := make([]domain.PlanGroup, 0, len(items))
		for _, item := range items {
			if item.GoodsTypeID == goodsTypeID {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	if regionID > 0 {
		filtered := make([]domain.PlanGroup, 0, len(items))
		for _, item := range items {
			if item.RegionID == regionID {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	c.JSON(http.StatusOK, gin.H{"items": toPlanGroupDTOs(items)})
}

func (h *Handler) Packages(c *gin.Context) {
	planGroupID, _ := strconv.ParseInt(c.Query("plan_group_id"), 10, 64)
	goodsTypeID, _ := strconv.ParseInt(c.Query("goods_type_id"), 10, 64)
	items, err := h.catalogSvc.ListPackages(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list error"})
		return
	}
	visiblePlans := listVisiblePlanGroups(h.catalogSvc, c)
	items = filterVisiblePackages(items, visiblePlans)
	if goodsTypeID > 0 {
		filtered := make([]domain.Package, 0, len(items))
		for _, item := range items {
			if item.GoodsTypeID == goodsTypeID {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	if planGroupID > 0 {
		filtered := make([]domain.Package, 0, len(items))
		for _, item := range items {
			if item.PlanGroupID == planGroupID {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	c.JSON(http.StatusOK, gin.H{"items": toPackageDTOs(items)})
}

func (h *Handler) BillingCycles(c *gin.Context) {
	items, err := h.catalogSvc.ListBillingCycles(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toBillingCycleDTOs(items)})
}

func (h *Handler) Dashboard(c *gin.Context) {
	userID := getUserID(c)
	orders, _, _ := h.orderSvc.ListOrders(c, appshared.OrderFilter{UserID: userID}, 1000, 0)
	vpsList, _ := h.vpsSvc.ListByUser(c, userID)
	pending := 0
	var spend30 int64
	from := time.Now().AddDate(0, 0, -30)
	for _, order := range orders {
		if order.Status == domain.OrderStatusPendingReview {
			pending++
		}
		if order.CreatedAt.After(from) && (order.Status == domain.OrderStatusApproved || order.Status == domain.OrderStatusProvisioning || order.Status == domain.OrderStatusActive) {
			spend30 += order.TotalAmount
		}
	}
	expiring := 0
	now := time.Now()
	for _, inst := range vpsList {
		if inst.ExpireAt != nil && inst.ExpireAt.Before(now.Add(7*24*time.Hour)) {
			expiring++
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"orders":         len(orders),
		"vps":            len(vpsList),
		"pending_review": pending,
		"expiring":       expiring,
		"spend_30d":      centsToFloat(spend30),
	})
}

func (h *Handler) CartList(c *gin.Context) {
	items, err := h.cartSvc.List(c, getUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cart error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toCartItemDTOs(items)})
}

func (h *Handler) CartAdd(c *gin.Context) {
	var payload struct {
		PackageID int64              `json:"package_id"`
		SystemID  int64              `json:"system_id"`
		Spec      appshared.CartSpec `json:"spec"`
		Qty       int                `json:"qty"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	item, err := h.cartSvc.Add(c, getUserID(c), payload.PackageID, payload.SystemID, payload.Spec, payload.Qty)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toCartItemDTO(item))
}

func (h *Handler) CartUpdate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		Spec appshared.CartSpec `json:"spec"`
		Qty  int                `json:"qty"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	item, err := h.cartSvc.Update(c, getUserID(c), id, payload.Spec, payload.Qty)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toCartItemDTO(item))
}

func (h *Handler) CartDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.cartSvc.Remove(c, getUserID(c), id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) CartClear(c *gin.Context) {
	if err := h.cartSvc.Clear(c, getUserID(c)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) OrderCreate(c *gin.Context) {
	var payload struct {
		Items []appshared.OrderItemInput `json:"items"`
	}
	if c.Request.ContentLength > 0 {
		if err := bindJSON(c, &payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
			return
		}
	}
	idem := c.GetHeader("Idempotency-Key")
	var order domain.Order
	var items []domain.OrderItem
	var err error
	if len(payload.Items) > 0 {
		order, items, err = h.orderSvc.CreateOrderFromItems(c, getUserID(c), "CNY", payload.Items, idem)
	} else {
		order, items, err = h.orderSvc.CreateOrderFromCart(c, getUserID(c), "CNY", idem)
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"order": toOrderDTO(order), "items": toOrderItemDTOs(items)})
}

func (h *Handler) OrderCreateItems(c *gin.Context) {
	var payload struct {
		Items []appshared.OrderItemInput `json:"items"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if len(payload.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "items required"})
		return
	}
	idem := c.GetHeader("Idempotency-Key")
	order, items, err := h.orderSvc.CreateOrderFromItems(c, getUserID(c), "CNY", payload.Items, idem)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"order": toOrderDTO(order), "items": toOrderItemDTOs(items)})
}

func (h *Handler) OrderPayment(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		Method        string `json:"method"`
		Amount        any    `json:"amount"`
		Currency      string `json:"currency"`
		TradeNo       string `json:"trade_no"`
		Note          string `json:"note"`
		ScreenshotURL string `json:"screenshot_url"`
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
	input := appshared.PaymentInput{
		Method:        payload.Method,
		Amount:        amount,
		Currency:      payload.Currency,
		TradeNo:       payload.TradeNo,
		Note:          payload.Note,
		ScreenshotURL: payload.ScreenshotURL,
	}
	idem := c.GetHeader("Idempotency-Key")
	payment, err := h.orderSvc.SubmitPayment(c, getUserID(c), id, input, idem)
	if err != nil {
		if err == appshared.ErrNoPaymentRequired {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err == appshared.ErrForbidden {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if err == appshared.ErrConflict {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		if err == appshared.ErrInvalidInput {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toOrderPaymentDTO(payment))
}

func (h *Handler) PaymentMethods(c *gin.Context) {
	if h.paymentSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payment disabled"})
		return
	}
	methods, err := h.paymentSvc.ListUserMethods(c, getUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toPaymentMethodDTOs(methods)})
}

func (h *Handler) OrderPay(c *gin.Context) {
	if h.paymentSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payment disabled"})
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		Method    string            `json:"method"`
		ReturnURL string            `json:"return_url"`
		NotifyURL string            `json:"notify_url"`
		Extra     map[string]string `json:"extra"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if payload.Extra == nil {
		payload.Extra = map[string]string{}
	}
	if strings.TrimSpace(payload.Extra["client_ip"]) == "" {
		ip := strings.TrimSpace(c.ClientIP())
		if ip != "" {
			payload.Extra["client_ip"] = ip
		}
	}
	if strings.TrimSpace(payload.Extra["device"]) == "" {
		payload.Extra["device"] = detectEZPayDeviceFromUA(c.GetHeader("User-Agent"))
	}
	result, err := h.paymentSvc.SelectPayment(c, getUserID(c), id, appshared.PaymentSelectInput{
		Method:    payload.Method,
		ReturnURL: payload.ReturnURL,
		NotifyURL: payload.NotifyURL,
		Extra:     payload.Extra,
	})
	if err != nil {
		status := http.StatusBadRequest
		if err == appshared.ErrForbidden {
			status = http.StatusForbidden
		} else if err == appshared.ErrNoPaymentRequired {
			status = http.StatusBadRequest
		} else if err == appshared.ErrConflict {
			status = http.StatusConflict
		} else if err == appshared.ErrInsufficientBalance {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toPaymentSelectDTO(result))
}

func detectEZPayDeviceFromUA(ua string) string {
	ua = strings.ToLower(strings.TrimSpace(ua))
	if ua == "" {
		return "mobile"
	}
	switch {
	case strings.Contains(ua, "micromessenger"):
		return "wechat"
	case strings.Contains(ua, "alipayclient"):
		return "alipay"
	case strings.Contains(ua, "mqqbrowser"), strings.Contains(ua, " qq/"):
		return "qq"
	case strings.Contains(ua, "mobile"), strings.Contains(ua, "android"), strings.Contains(ua, "iphone"), strings.Contains(ua, "ipad"):
		return "mobile"
	default:
		return "pc"
	}
}

func (h *Handler) PaymentNotify(c *gin.Context) {
	if h.paymentSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payment disabled"})
		return
	}
	provider := c.Param("provider")
	body, _ := io.ReadAll(c.Request.Body)
	headers := map[string][]string{}
	for k, v := range c.Request.Header {
		copied := make([]string, len(v))
		copy(copied, v)
		headers[k] = copied
	}
	result, err := h.paymentSvc.HandleNotify(c, provider, appshared.RawHTTPRequest{
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
	if result.AckBody != "" {
		ct := "text/plain; charset=utf-8"
		if s := strings.TrimSpace(result.AckBody); strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[") {
			ct = "application/json; charset=utf-8"
		}
		c.Data(http.StatusOK, ct, []byte(result.AckBody))
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "trade_no": result.TradeNo})
}
