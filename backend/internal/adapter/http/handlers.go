package http

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"xiaoheiplay/internal/adapter/email"
	"xiaoheiplay/internal/adapter/robot"
	"xiaoheiplay/internal/adapter/sse"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/pkg/money"
	"xiaoheiplay/internal/pkg/permissions"
	"xiaoheiplay/internal/usecase"
)

type Handler struct {
	authSvc       *usecase.AuthService
	catalogSvc    *usecase.CatalogService
	cartSvc       *usecase.CartService
	orderSvc      *usecase.OrderService
	vpsSvc        *usecase.VPSService
	adminSvc      *usecase.AdminService
	adminVPS      *usecase.AdminVPSService
	integration   *usecase.IntegrationService
	reportSvc     *usecase.ReportService
	cmsSvc        *usecase.CMSService
	ticketSvc     *usecase.TicketService
	walletSvc     *usecase.WalletService
	walletOrder   *usecase.WalletOrderService
	paymentSvc    *usecase.PaymentService
	messageSvc    *usecase.MessageCenterService
	statusSvc     *usecase.ServerStatusService
	realnameSvc   *usecase.RealNameService
	orderItems    usecase.OrderItemRepository
	users         usecase.UserRepository
	orderRepo     usecase.OrderRepository
	vpsRepo       usecase.VPSRepository
	payments      usecase.PaymentRepository
	eventsRepo    usecase.EventRepository
	automationLog usecase.AutomationLogRepository
	settings      usecase.SettingsRepository
	permissions   usecase.PermissionRepository
	uploads       usecase.UploadRepository
	broker        *sse.Broker
	jwtSecret     []byte
	passwordReset *usecase.PasswordResetService
	permissionSvc *usecase.PermissionService
	automation    usecase.AutomationClient
	pluginDir     string
	pluginPass    string
	taskSvc       *usecase.ScheduledTaskService
}

func NewHandler(authSvc *usecase.AuthService, catalogSvc *usecase.CatalogService, cartSvc *usecase.CartService, orderSvc *usecase.OrderService, vpsSvc *usecase.VPSService, adminSvc *usecase.AdminService, adminVPS *usecase.AdminVPSService, integration *usecase.IntegrationService, reportSvc *usecase.ReportService, cmsSvc *usecase.CMSService, ticketSvc *usecase.TicketService, walletSvc *usecase.WalletService, walletOrder *usecase.WalletOrderService, paymentSvc *usecase.PaymentService, messageSvc *usecase.MessageCenterService, statusSvc *usecase.ServerStatusService, realnameSvc *usecase.RealNameService, orderItems usecase.OrderItemRepository, users usecase.UserRepository, orderRepo usecase.OrderRepository, vpsRepo usecase.VPSRepository, payments usecase.PaymentRepository, eventsRepo usecase.EventRepository, automationLogs usecase.AutomationLogRepository, settings usecase.SettingsRepository, permissions usecase.PermissionRepository, uploads usecase.UploadRepository, broker *sse.Broker, jwtSecret string, automation usecase.AutomationClient, passwordReset *usecase.PasswordResetService, permissionSvc *usecase.PermissionService, taskSvc *usecase.ScheduledTaskService) *Handler {
	return &Handler{
		authSvc:       authSvc,
		catalogSvc:    catalogSvc,
		cartSvc:       cartSvc,
		orderSvc:      orderSvc,
		vpsSvc:        vpsSvc,
		adminSvc:      adminSvc,
		adminVPS:      adminVPS,
		integration:   integration,
		reportSvc:     reportSvc,
		cmsSvc:        cmsSvc,
		ticketSvc:     ticketSvc,
		walletSvc:     walletSvc,
		walletOrder:   walletOrder,
		paymentSvc:    paymentSvc,
		messageSvc:    messageSvc,
		statusSvc:     statusSvc,
		realnameSvc:   realnameSvc,
		orderItems:    orderItems,
		users:         users,
		orderRepo:     orderRepo,
		vpsRepo:       vpsRepo,
		payments:      payments,
		eventsRepo:    eventsRepo,
		automationLog: automationLogs,
		settings:      settings,
		permissions:   permissions,
		uploads:       uploads,
		broker:        broker,
		jwtSecret:     []byte(jwtSecret),
		automation:    automation,
		passwordReset: passwordReset,
		permissionSvc: permissionSvc,
		taskSvc:       taskSvc,
	}
}

func (h *Handler) SetPaymentPluginConfig(dir, password string) {
	h.pluginDir = strings.TrimSpace(dir)
	h.pluginPass = strings.TrimSpace(password)
}

func (h *Handler) Captcha(c *gin.Context) {
	captcha, code, err := h.authSvc.CreateCaptcha(c, 5*time.Minute)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "captcha error"})
		return
	}
	img := renderCaptcha(code)
	var buf strings.Builder
	enc := base64.NewEncoder(base64.StdEncoding, &buf)
	if err := png.Encode(enc, img); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "captcha encode error"})
		return
	}
	_ = enc.Close()
	c.JSON(http.StatusOK, gin.H{
		"captcha_id":   captcha.ID,
		"image_base64": buf.String(),
	})
}

func (h *Handler) Register(c *gin.Context) {
	var payload struct {
		Username    string `json:"username"`
		Email       string `json:"email"`
		QQ          string `json:"qq"`
		Phone       string `json:"phone"`
		Password    string `json:"password"`
		CaptchaID   string `json:"captcha_id"`
		CaptchaCode string `json:"captcha_code"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	user, err := h.authSvc.Register(c, usecase.RegisterInput{
		Username:    payload.Username,
		Email:       payload.Email,
		QQ:          payload.QQ,
		Phone:       payload.Phone,
		Password:    payload.Password,
		CaptchaID:   payload.CaptchaID,
		CaptchaCode: payload.CaptchaCode,
	})
	if err != nil {
		status := http.StatusBadRequest
		if err == usecase.ErrRealNameRequired || err == usecase.ErrForbidden {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": user.ID, "username": user.Username, "email": user.Email})
}

func (h *Handler) Login(c *gin.Context) {
	var payload struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	user, err := h.authSvc.Login(c, payload.Username, payload.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"role":    user.Role,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})
	signed, _ := token.SignedString(h.jwtSecret)
	c.JSON(http.StatusOK, gin.H{"access_token": signed, "expires_in": 86400, "user": gin.H{"id": user.ID, "username": user.Username, "role": user.Role}})
}

func (h *Handler) Logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) Refresh(c *gin.Context) {
	userID := getUserID(c)
	roleVal, _ := c.Get("role")
	role, _ := roleVal.(string)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})
	signed, _ := token.SignedString(h.jwtSecret)
	c.JSON(http.StatusOK, gin.H{"access_token": signed, "expires_in": 86400})
}

func (h *Handler) Me(c *gin.Context) {
	userID := getUserID(c)
	user, err := h.users.GetUserByID(c, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, toUserDTO(user))
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	var payload struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		QQ       string `json:"qq"`
		Phone    string `json:"phone"`
		Bio      string `json:"bio"`
		Intro    string `json:"intro"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	user, err := h.authSvc.UpdateProfile(c, getUserID(c), usecase.UpdateProfileInput{
		Username: payload.Username,
		Email:    payload.Email,
		QQ:       payload.QQ,
		Phone:    payload.Phone,
		Bio:      payload.Bio,
		Intro:    payload.Intro,
		Password: payload.Password,
	})
	if err != nil {
		status := http.StatusBadRequest
		if err == usecase.ErrRealNameRequired || err == usecase.ErrForbidden {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toUserDTO(user))
}

func (h *Handler) RealNameStatus(c *gin.Context) {
	if h.realnameSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	enabled, provider, actions := h.realnameSvc.GetConfig(c)
	var record *domain.RealNameVerification
	if latest, err := h.realnameSvc.Latest(c, getUserID(c)); err == nil {
		record = &latest
	}
	verified := false
	if record != nil && record.Status == "verified" {
		verified = true
	}
	resp := gin.H{
		"enabled":       enabled,
		"provider":      provider,
		"block_actions": actions,
		"verified":      verified,
		"verification":  nil,
	}
	if record != nil {
		resp["verification"] = toRealNameVerificationDTO(*record)
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) RealNameVerify(c *gin.Context) {
	if h.realnameSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	var payload struct {
		RealName string `json:"real_name"`
		IDNumber string `json:"id_number"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	record, err := h.realnameSvc.Verify(c, getUserID(c), payload.RealName, payload.IDNumber)
	if err != nil {
		status := http.StatusBadRequest
		if err == usecase.ErrForbidden {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toRealNameVerificationDTO(record))
}

func (h *Handler) Catalog(c *gin.Context) {
	regions, plans, packages, images, cycles, err := h.catalogSvc.Catalog(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "catalog error"})
		return
	}
	plans = filterVisiblePlanGroups(plans)
	packages = filterVisiblePackages(packages, plans)
	if len(plans) == 0 {
		images = []domain.SystemImage{}
	} else {
		images = filterEnabledSystemImages(images, plans)
	}
	c.JSON(http.StatusOK, gin.H{
		"regions":        toRegionDTOs(regions),
		"plan_groups":    toPlanGroupDTOs(plans),
		"packages":       toPackageDTOs(packages),
		"system_images":  toSystemImageDTOs(images),
		"billing_cycles": toBillingCycleDTOs(cycles),
	})
}

func (h *Handler) SystemImages(c *gin.Context) {
	lineID, _ := strconv.ParseInt(c.Query("line_id"), 10, 64)
	planGroupID, _ := strconv.ParseInt(c.Query("plan_group_id"), 10, 64)
	if planGroupID > 0 {
		if plan, err := h.catalogSvc.GetPlanGroup(c, planGroupID); err == nil {
			if !plan.Active || !plan.Visible {
				c.JSON(http.StatusOK, gin.H{"items": []SystemImageDTO{}})
				return
			}
			lineID = plan.LineID
		}
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
	items, err := h.catalogSvc.ListPlanGroups(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list error"})
		return
	}
	items = filterVisiblePlanGroups(items)
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
	items, err := h.catalogSvc.ListPackages(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list error"})
		return
	}
	visiblePlans := listVisiblePlanGroups(h.catalogSvc, c)
	items = filterVisiblePackages(items, visiblePlans)
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
	orders, _, _ := h.orderSvc.ListOrders(c, usecase.OrderFilter{UserID: userID}, 1000, 0)
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
		PackageID int64            `json:"package_id"`
		SystemID  int64            `json:"system_id"`
		Spec      usecase.CartSpec `json:"spec"`
		Qty       int              `json:"qty"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
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
		Spec usecase.CartSpec `json:"spec"`
		Qty  int              `json:"qty"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
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
		Items []usecase.OrderItemInput `json:"items"`
	}
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&payload); err != nil {
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
		Items []usecase.OrderItemInput `json:"items"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
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
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	amount, err := parseAmountCents(payload.Amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount"})
		return
	}
	input := usecase.PaymentInput{
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
		if err == usecase.ErrNoPaymentRequired {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err == usecase.ErrForbidden {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if err == usecase.ErrConflict {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		if err == usecase.ErrInvalidInput {
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
		Method    string `json:"method"`
		ReturnURL string `json:"return_url"`
		NotifyURL string `json:"notify_url"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	result, err := h.paymentSvc.SelectPayment(c, getUserID(c), id, usecase.PaymentSelectInput{
		Method:    payload.Method,
		ReturnURL: payload.ReturnURL,
		NotifyURL: payload.NotifyURL,
	})
	if err != nil {
		status := http.StatusBadRequest
		if err == usecase.ErrForbidden {
			status = http.StatusForbidden
		} else if err == usecase.ErrNoPaymentRequired {
			status = http.StatusBadRequest
		} else if err == usecase.ErrConflict {
			status = http.StatusConflict
		} else if err == usecase.ErrInsufficientBalance {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toPaymentSelectDTO(result))
}

func (h *Handler) PaymentNotify(c *gin.Context) {
	if h.paymentSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payment disabled"})
		return
	}
	provider := c.Param("provider")
	if err := c.Request.ParseForm(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	params := map[string]string{}
	for key, values := range c.Request.Form {
		if len(values) == 0 {
			continue
		}
		params[key] = values[0]
	}
	result, err := h.paymentSvc.HandleNotify(c, provider, params)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "trade_no": result.TradeNo})
}

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
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	amount, err := parseAmountCents(payload.Amount)
	if err != nil || amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount"})
		return
	}
	order, err := h.walletOrder.CreateRecharge(c, getUserID(c), usecase.WalletOrderCreateInput{
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
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	amount, err := parseAmountCents(payload.Amount)
	if err != nil || amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount"})
		return
	}
	order, err := h.walletOrder.CreateWithdraw(c, getUserID(c), usecase.WalletOrderCreateInput{
		Amount:   amount,
		Currency: payload.Currency,
		Note:     payload.Note,
		Meta:     payload.Meta,
	})
	if err != nil {
		status := http.StatusBadRequest
		if err == usecase.ErrInsufficientBalance {
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
	filter := usecase.OrderFilter{UserID: getUserID(c), Status: status}
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
	if h.payments != nil {
		payments, _ = h.payments.ListPaymentsByOrder(c, id)
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

func (h *Handler) VPSList(c *gin.Context) {
	items, err := h.vpsSvc.ListByUser(c, getUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "vps list error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": h.toVPSInstanceDTOsWithLifecycle(c, items)})
}

func (h *Handler) VPSDetail(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, h.toVPSInstanceDTOWithLifecycle(c, inst))
}

func (h *Handler) VPSRefresh(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	updated, err := h.vpsSvc.RefreshStatus(c, inst)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, h.toVPSInstanceDTOWithLifecycle(c, updated))
}

func (h *Handler) VPSPanel(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	url, err := h.vpsSvc.GetPanelURL(c, inst)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Redirect(http.StatusFound, url)
}

func (h *Handler) VPSMonitor(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if refreshed, err := h.vpsSvc.RefreshStatus(c, inst); err == nil {
		inst = refreshed
	}
	payload := gin.H{
		"status":           string(inst.Status),
		"automation_state": inst.AutomationState,
		"access_info":      parseMapJSON(inst.AccessInfoJSON),
		"spec":             parseRawJSON(inst.SpecJSON),
	}
	monitor, err := h.vpsSvc.Monitor(c, inst)
	if err != nil {
		if strings.Contains(err.Error(), "创建中") {
			_ = h.vpsSvc.SetStatus(c, inst, domain.VPSStatusProvisioning, 0)
			payload["status"] = string(domain.VPSStatusProvisioning)
			payload["automation_state"] = 0
		}
		payload["monitor_error"] = err.Error()
		c.JSON(http.StatusOK, payload)
		return
	}
	payload["cpu"] = monitor.CPUPercent
	payload["memory"] = monitor.MemoryPercent
	payload["bytes_in"] = monitor.BytesIn
	payload["bytes_out"] = monitor.BytesOut
	payload["storage"] = monitor.StoragePercent
	c.JSON(http.StatusOK, payload)
}

func (h *Handler) VPSVNC(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	url, err := h.vpsSvc.VNCURL(c, inst)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Redirect(http.StatusFound, url)
}

func (h *Handler) VPSStart(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err := h.vpsSvc.Start(c, inst); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) VPSShutdown(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err := h.vpsSvc.Shutdown(c, inst); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) VPSReboot(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err := h.vpsSvc.Reboot(c, inst); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) VPSResetOS(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var payload map[string]any
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	parseInt := func(val any) int64 {
		switch v := val.(type) {
		case float64:
			return int64(v)
		case string:
			parsed, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
			return parsed
		default:
			return 0
		}
	}
	hostID := parseInt(payload["host_id"])
	templateID := parseInt(payload["template_id"])
	password, _ := payload["password"].(string)
	if hostID != 0 && hostID != id {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}
	if err := h.vpsSvc.ResetOS(c, inst, templateID, strings.TrimSpace(password)); err != nil {
		if err == usecase.ErrInvalidInput {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) VPSResetOSPassword(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var payload struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if err := h.vpsSvc.ResetOSPassword(c, inst, strings.TrimSpace(payload.Password)); err != nil {
		if err == usecase.ErrInvalidInput {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) VPSSnapshots(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	switch c.Request.Method {
	case http.MethodGet:
		items, err := h.vpsSvc.ListSnapshots(c, inst)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": items})
	case http.MethodPost:
		if err := h.vpsSvc.CreateSnapshot(c, inst); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	default:
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "method not allowed"})
	}
}

func (h *Handler) VPSSnapshotDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	snapshotID, _ := strconv.ParseInt(c.Param("snapshotId"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err := h.vpsSvc.DeleteSnapshot(c, inst, snapshotID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) VPSSnapshotRestore(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	snapshotID, _ := strconv.ParseInt(c.Param("snapshotId"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err := h.vpsSvc.RestoreSnapshot(c, inst, snapshotID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) VPSBackups(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	switch c.Request.Method {
	case http.MethodGet:
		items, err := h.vpsSvc.ListBackups(c, inst)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": items})
	case http.MethodPost:
		if err := h.vpsSvc.CreateBackup(c, inst); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	default:
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "method not allowed"})
	}
}

func (h *Handler) VPSBackupDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	backupID, _ := strconv.ParseInt(c.Param("backupId"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err := h.vpsSvc.DeleteBackup(c, inst, backupID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) VPSBackupRestore(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	backupID, _ := strconv.ParseInt(c.Param("backupId"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err := h.vpsSvc.RestoreBackup(c, inst, backupID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) VPSFirewallRules(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	switch c.Request.Method {
	case http.MethodGet:
		items, err := h.vpsSvc.ListFirewallRules(c, inst)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": items})
	case http.MethodPost:
		var payload struct {
			Direction string `json:"direction"`
			Protocol  string `json:"protocol"`
			Method    string `json:"method"`
			Port      string `json:"port"`
			IP        string `json:"ip"`
		}
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
			return
		}
		req := usecase.AutomationFirewallRuleCreate{
			Direction: strings.TrimSpace(payload.Direction),
			Protocol:  strings.TrimSpace(payload.Protocol),
			Method:    strings.TrimSpace(payload.Method),
			Port:      strings.TrimSpace(payload.Port),
			IP:        strings.TrimSpace(payload.IP),
		}
		if req.Direction == "" || req.Protocol == "" || req.Method == "" || req.Port == "" || req.IP == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
			return
		}
		if err := h.vpsSvc.AddFirewallRule(c, inst, req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	default:
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "method not allowed"})
	}
}

func (h *Handler) VPSFirewallDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	ruleID, _ := strconv.ParseInt(c.Param("ruleId"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err := h.vpsSvc.DeleteFirewallRule(c, inst, ruleID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) VPSPortMappings(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	switch c.Request.Method {
	case http.MethodGet:
		items, err := h.vpsSvc.ListPortMappings(c, inst)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": items})
	case http.MethodPost:
		var payload map[string]any
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
			return
		}
		name := strings.TrimSpace(fmt.Sprint(payload["name"]))
		sport := strings.TrimSpace(fmt.Sprint(payload["sport"]))
		if sport == "<nil>" {
			sport = ""
		}
		dport, ok := parsePortValue(payload["dport"])
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
			return
		}
		req := usecase.AutomationPortMappingCreate{
			Name:  name,
			Sport: sport,
			Dport: dport,
		}
		if err := h.vpsSvc.AddPortMapping(c, inst, req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	default:
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "method not allowed"})
	}
}

func parsePortValue(value any) (int64, bool) {
	switch v := value.(type) {
	case float64:
		if v <= 0 {
			return 0, false
		}
		return int64(v), true
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0, false
		}
		parsed, err := strconv.ParseInt(trimmed, 10, 64)
		if err != nil || parsed <= 0 {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func (h *Handler) VPSPortCandidates(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	keywords := strings.TrimSpace(c.Query("keywords"))
	items, err := h.vpsSvc.FindPortCandidates(c, inst, keywords)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *Handler) VPSPortMappingDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	mappingID, _ := strconv.ParseInt(c.Param("mappingId"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err := h.vpsSvc.DeletePortMapping(c, inst, mappingID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) TicketCreate(c *gin.Context) {
	var payload struct {
		Subject   string `json:"subject"`
		Content   string `json:"content"`
		Resources []struct {
			ResourceType string `json:"resource_type"`
			ResourceID   int64  `json:"resource_id"`
			ResourceName string `json:"resource_name"`
		} `json:"resources"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	resources := make([]domain.TicketResource, 0, len(payload.Resources))
	for _, res := range payload.Resources {
		resources = append(resources, domain.TicketResource{ResourceType: res.ResourceType, ResourceID: res.ResourceID, ResourceName: res.ResourceName})
	}
	ticket, messages, resItems, err := h.ticketSvc.Create(c, getUserID(c), payload.Subject, payload.Content, resources)
	if err != nil {
		if err == usecase.ErrInvalidInput {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	msgDTOs := make([]TicketMessageDTO, 0, len(messages))
	for _, msg := range messages {
		msgDTOs = append(msgDTOs, toTicketMessageDTO(msg, msg.SenderName, msg.SenderQQ))
	}
	resDTOs := make([]TicketResourceDTO, 0, len(resItems))
	for _, res := range resItems {
		resDTOs = append(resDTOs, toTicketResourceDTO(res))
	}
	c.JSON(http.StatusOK, gin.H{"ticket": toTicketDTO(ticket), "messages": msgDTOs, "resources": resDTOs})
}

func (h *Handler) TicketList(c *gin.Context) {
	status := strings.TrimSpace(c.Query("status"))
	limit, offset := paging(c)
	userID := getUserID(c)
	filter := usecase.TicketFilter{UserID: &userID, Status: status, Limit: limit, Offset: offset}
	items, total, err := h.ticketSvc.List(c, filter)
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

func (h *Handler) TicketDetail(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	ticket, messages, resources, err := h.ticketSvc.GetDetail(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if ticket.UserID != getUserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
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

func (h *Handler) TicketMessageCreate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	ticket, err := h.ticketSvc.Get(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if ticket.UserID != getUserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	var payload struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	msg, err := h.ticketSvc.AddMessage(c, ticket, getUserID(c), "user", payload.Content)
	if err != nil {
		if err == usecase.ErrForbidden {
			c.JSON(http.StatusForbidden, gin.H{"error": "ticket closed"})
			return
		}
		if err == usecase.ErrInvalidInput {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toTicketMessageDTO(msg, msg.SenderName, msg.SenderQQ))
}

func (h *Handler) TicketClose(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	ticket, err := h.ticketSvc.Get(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err := h.ticketSvc.Close(c, ticket, getUserID(c)); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) VPSEmergencyRenew(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	inst, err := h.vpsSvc.Get(c, id, getUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	_, err = h.orderSvc.CreateEmergencyRenewOrder(c, getUserID(c), inst.ID)
	if err != nil {
		status := http.StatusBadRequest
		if err == usecase.ErrConflict {
			status = http.StatusConflict
		} else if err == usecase.ErrForbidden {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	updated, _ := h.vpsSvc.Get(c, id, getUserID(c))
	c.JSON(http.StatusOK, h.toVPSInstanceDTOWithLifecycle(c, updated))
}

func (h *Handler) VPSRenewOrder(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		RenewDays      int `json:"renew_days"`
		DurationMonths int `json:"duration_months"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	order, err := h.orderSvc.CreateRenewOrder(c, getUserID(c), id, payload.RenewDays, payload.DurationMonths)
	if err != nil {
		status := http.StatusBadRequest
		if err == usecase.ErrRealNameRequired || err == usecase.ErrForbidden {
			status = http.StatusForbidden
		} else if errors.Is(err, usecase.ErrConflict) {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toOrderDTO(order))
}

func (h *Handler) VPSResizeOrder(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		Spec            *usecase.CartSpec `json:"spec"`
		TargetPackageID int64             `json:"target_package_id"`
		ResetAddons     bool              `json:"reset_addons"`
		ScheduledAt     string            `json:"scheduled_at"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	var scheduledAt *time.Time
	if strings.TrimSpace(payload.ScheduledAt) != "" {
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(payload.ScheduledAt))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scheduled_at"})
			return
		}
		scheduledAt = &t
	}
	order, _, err := h.orderSvc.CreateResizeOrder(c, getUserID(c), id, payload.Spec, payload.TargetPackageID, payload.ResetAddons, scheduledAt)
	if err != nil {
		status := http.StatusBadRequest
		if err == usecase.ErrRealNameRequired || err == usecase.ErrForbidden || err == usecase.ErrResizeDisabled {
			status = http.StatusForbidden
		} else if err == usecase.ErrResizeInProgress || err == usecase.ErrConflict {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"order": toOrderDTO(order)})
}

func (h *Handler) VPSResizeQuote(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		Spec            *usecase.CartSpec `json:"spec"`
		TargetPackageID int64             `json:"target_package_id"`
		ResetAddons     bool              `json:"reset_addons"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	quote, targetSpec, err := h.orderSvc.QuoteResizeOrder(c, getUserID(c), id, payload.Spec, payload.TargetPackageID, payload.ResetAddons)
	if err != nil {
		status := http.StatusBadRequest
		if err == usecase.ErrRealNameRequired || err == usecase.ErrForbidden || err == usecase.ErrResizeDisabled {
			status = http.StatusForbidden
		} else if err == usecase.ErrResizeInProgress || err == usecase.ErrConflict {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	resp := quote.ToPayload(id, targetSpec)
	resp["charge_amount"] = centsToFloat(quote.ChargeAmount)
	resp["refund_amount"] = centsToFloat(quote.RefundAmount)
	c.JSON(http.StatusOK, gin.H{"quote": resp})
}

func (h *Handler) VPSRefund(c *gin.Context) {
	if h.orderSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "orders disabled"})
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&payload)
	order, amount, err := h.orderSvc.CreateRefundOrder(c, getUserID(c), id, payload.Reason)
	if err != nil {
		status := http.StatusBadRequest
		if err == usecase.ErrForbidden {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"order": toOrderDTO(order), "refund_amount": centsToFloat(amount)})
}

func (h *Handler) RobotApprove(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.orderSvc.ApproveOrder(c, 0, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) RobotReject(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&payload)
	if err := h.orderSvc.RejectOrder(c, 0, id, payload.Reason); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) RobotWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	var payload struct {
		Text      string `json:"text"`
		Sender    string `json:"sender"`
		Timestamp any    `json:"timestamp"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if h.settings != nil {
		if enabled := strings.ToLower(getSettingValue(c, h.settings, "robot_webhook_enabled")); enabled == "false" {
			c.JSON(http.StatusForbidden, gin.H{"error": "robot webhook disabled"})
			return
		}
		secret := getSettingValue(c, h.settings, "robot_webhook_secret")
		if secret != "" {
			signature := c.GetHeader("X-Signature")
			if signature == "" {
				signature = c.GetHeader("X-Robot-Signature")
			}
			if signature == "" || !verifyHMAC(body, secret, signature) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
				return
			}
		}
	}
	text := strings.TrimSpace(payload.Text)
	if strings.HasPrefix(text, "通过订单") {
		rest := strings.TrimSpace(strings.TrimPrefix(text, "通过订单"))
		idStr := strings.Fields(rest)
		if len(idStr) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing order id"})
			return
		}
		orderID, err := strconv.ParseInt(idStr[0], 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
			return
		}
		if err := h.orderSvc.ApproveOrder(c, 0, orderID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}
	if strings.HasPrefix(text, "驳回订单") {
		rest := strings.TrimSpace(strings.TrimPrefix(text, "驳回订单"))
		parts := strings.Fields(rest)
		if len(parts) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing order id"})
			return
		}
		orderID, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
			return
		}
		reason := ""
		if len(parts) > 1 {
			reason = strings.TrimSpace(strings.TrimPrefix(strings.Join(parts[1:], " "), "原因"))
		}
		if err := h.orderSvc.RejectOrder(c, 0, orderID, reason); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": "unknown command"})
}

func (h *Handler) AdminLogin(c *gin.Context) {
	var payload struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	user, err := h.authSvc.Login(c, payload.Username, payload.Password)
	if err != nil || user.Role != domain.UserRoleAdmin {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"role":    user.Role,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})
	signed, _ := token.SignedString(h.jwtSecret)
	c.JSON(http.StatusOK, gin.H{"access_token": signed, "expires_in": 86400})
}

func (h *Handler) AdminUsers(c *gin.Context) {
	limit, offset := paging(c)
	users, total, err := h.adminSvc.ListUsers(c, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toUserDTOs(users), "total": total})
}

func (h *Handler) AdminUserDetail(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	user, err := h.adminSvc.GetUser(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if user.Role == domain.UserRoleAdmin {
		c.JSON(http.StatusBadRequest, gin.H{"error": "admin user not editable"})
		return
	}
	if user.Role == domain.UserRoleAdmin {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, toUserDTO(user))
}

func (h *Handler) AdminUserCreate(c *gin.Context) {
	var payload struct {
		Username          string `json:"username"`
		Email             string `json:"email"`
		QQ                string `json:"qq"`
		Phone             string `json:"phone"`
		Bio               string `json:"bio"`
		Intro             string `json:"intro"`
		Password          string `json:"password"`
		Role              string `json:"role"`
		Status            string `json:"status"`
		PermissionGroupID *int64 `json:"permission_group_id"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if payload.Role != "" && strings.TrimSpace(payload.Role) != string(domain.UserRoleUser) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "admin role not allowed"})
		return
	}
	user, err := h.adminSvc.CreateUser(c, getUserID(c), domain.User{
		Username:          payload.Username,
		Email:             payload.Email,
		QQ:                payload.QQ,
		Phone:             payload.Phone,
		Bio:               payload.Bio,
		Intro:             payload.Intro,
		PermissionGroupID: payload.PermissionGroupID,
		Role:              domain.UserRoleUser,
		Status:            domain.UserStatus(payload.Status),
	}, payload.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toUserDTO(user))
}

func (h *Handler) AdminUserUpdate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		Username          *string `json:"username"`
		Email             *string `json:"email"`
		QQ                *string `json:"qq"`
		Phone             *string `json:"phone"`
		Bio               *string `json:"bio"`
		Intro             *string `json:"intro"`
		Avatar            *string `json:"avatar"`
		Role              *string `json:"role"`
		Status            *string `json:"status"`
		PermissionGroupID *int64  `json:"permission_group_id"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	user, err := h.adminSvc.GetUser(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if payload.Username != nil {
		user.Username = strings.TrimSpace(*payload.Username)
	}
	if payload.Email != nil {
		user.Email = strings.TrimSpace(*payload.Email)
	}
	if payload.QQ != nil {
		user.QQ = strings.TrimSpace(*payload.QQ)
	}
	if payload.Phone != nil {
		user.Phone = strings.TrimSpace(*payload.Phone)
	}
	if payload.Bio != nil {
		user.Bio = *payload.Bio
	}
	if payload.Intro != nil {
		user.Intro = *payload.Intro
	}
	if payload.Avatar != nil {
		user.Avatar = strings.TrimSpace(*payload.Avatar)
	}
	if payload.Role != nil {
		role := strings.TrimSpace(*payload.Role)
		if role != "" && role != string(domain.UserRoleUser) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "admin role not allowed"})
			return
		}
		user.Role = domain.UserRoleUser
	}
	if payload.Status != nil {
		user.Status = domain.UserStatus(strings.TrimSpace(*payload.Status))
	}
	if payload.PermissionGroupID != nil {
		user.PermissionGroupID = payload.PermissionGroupID
	}
	if err := h.adminSvc.UpdateUser(c, getUserID(c), user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminUserResetPassword(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	user, err := h.adminSvc.GetUser(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if user.Role == domain.UserRoleAdmin {
		c.JSON(http.StatusBadRequest, gin.H{"error": "admin user not editable"})
		return
	}
	if err := h.adminSvc.ResetUserPassword(c, getUserID(c), id, payload.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminUserStatus(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	user, err := h.adminSvc.GetUser(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if user.Role == domain.UserRoleAdmin {
		c.JSON(http.StatusBadRequest, gin.H{"error": "admin user not editable"})
		return
	}
	status := domain.UserStatus(payload.Status)
	if err := h.adminSvc.UpdateUserStatus(c, getUserID(c), id, status); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminUserRealNameStatus(c *gin.Context) {
	if h.realnameSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "realname disabled"})
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	user, err := h.adminSvc.GetUser(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if user.Role == domain.UserRoleAdmin {
		c.JSON(http.StatusBadRequest, gin.H{"error": "admin user not editable"})
		return
	}
	record, err := h.realnameSvc.Latest(c, id)
	if err != nil {
		if err == usecase.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "realname record not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.realnameSvc.UpdateStatus(c, record.ID, payload.Status, payload.Reason); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updated, err := h.realnameSvc.Latest(c, id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toRealNameVerificationDTO(updated))
}

func (h *Handler) AdminUserImpersonate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	user, err := h.adminSvc.GetUser(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if user.Role != domain.UserRoleUser {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not a user account"})
		return
	}
	if user.Status != domain.UserStatusActive {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user disabled"})
		return
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"role":    user.Role,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})
	signed, _ := token.SignedString(h.jwtSecret)
	c.JSON(http.StatusOK, gin.H{"access_token": signed, "expires_in": 86400, "user": gin.H{"id": user.ID, "username": user.Username, "role": user.Role}})
}

func (h *Handler) AdminOrders(c *gin.Context) {
	limit, offset := paging(c)
	filter := usecase.OrderFilter{}
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toOrderDTOs(orders), "total": total})
}

func (h *Handler) AdminPaymentProviders(c *gin.Context) {
	if h.paymentSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payment disabled"})
		return
	}
	items, err := h.paymentSvc.ListProviders(c, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toPaymentProviderDTOs(items)})
}

func (h *Handler) AdminPaymentProviderUpdate(c *gin.Context) {
	if h.paymentSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payment disabled"})
		return
	}
	key := c.Param("key")
	var payload struct {
		Enabled    *bool  `json:"enabled"`
		ConfigJSON string `json:"config_json"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	enabled := true
	if payload.Enabled != nil {
		enabled = *payload.Enabled
	}
	if err := h.paymentSvc.UpdateProvider(c, key, enabled, payload.ConfigJSON); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPaymentPluginUpload(c *gin.Context) {
	password := c.PostForm("password")
	if password == "" {
		password = c.GetHeader("X-Plugin-Password")
	}
	expected := h.pluginPass
	if expected == "" && h.settings != nil {
		expected = getSettingValue(c, h.settings, "payment_plugin_upload_password")
	}
	if expected == "" {
		expected = "qweasd123456"
	}
	if password == "" || password != expected {
		c.JSON(http.StatusForbidden, gin.H{"error": "invalid password"})
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing file"})
		return
	}
	dir := strings.TrimSpace(h.pluginDir)
	if dir == "" && h.settings != nil {
		dir = strings.TrimSpace(getSettingValue(c, h.settings, "payment_plugin_dir"))
	}
	if dir == "" {
		dir = "plugins/payment"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "mkdir failed"})
		return
	}
	filename := filepath.Base(file.Filename)
	if filename == "." || filename == "" || strings.Contains(filename, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid filename"})
		return
	}
	dst := filepath.Join(dir, filename)
	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "upload failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "path": dst})
}

func (h *Handler) AdminServerStatus(c *gin.Context) {
	if h.statusSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status disabled"})
		return
	}
	status, err := h.statusSvc.Status(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toServerStatusDTO(status))
}

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
	if err := c.ShouldBindJSON(&payload); err != nil {
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
		if err == usecase.ErrInsufficientBalance {
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
		if err == usecase.ErrConflict {
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
	_ = c.ShouldBindJSON(&payload)
	if err := h.walletOrder.Reject(c, getUserID(c), id, payload.Reason); err != nil {
		status := http.StatusBadRequest
		if err == usecase.ErrConflict {
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
	var payload usecase.ScheduledTaskUpdate
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	item, err := h.taskSvc.UpdateTask(c, key, payload)
	if err != nil {
		status := http.StatusBadRequest
		if err == usecase.ErrNotFound {
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
		if err == usecase.ErrInvalidInput {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) AdminOrderDetail(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	order, err := h.orderRepo.GetOrder(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	items, err := h.orderItems.ListOrderItems(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order items not found"})
		return
	}
	var payments []domain.OrderPayment
	if h.payments != nil {
		payments, _ = h.payments.ListPaymentsByOrder(c, id)
	}
	var events []domain.OrderEvent
	if h.eventsRepo != nil {
		events, _ = h.eventsRepo.ListEventsAfter(c, id, 0, 200)
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
		if err == usecase.ErrConflict || err == usecase.ErrResizeInProgress {
			status = http.StatusConflict
			if err == usecase.ErrConflict {
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
	_ = c.ShouldBindJSON(&payload)
	if err := h.orderSvc.RejectOrder(c, getUserID(c), id, payload.Reason); err != nil {
		status := http.StatusBadRequest
		msg := err.Error()
		if err == usecase.ErrConflict {
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "permission check failed"})
			return
		}
		if !has {
			c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
			return
		}
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.adminSvc.DeleteOrder(c, getUserID(c), id); err != nil {
		status := http.StatusBadRequest
		if err == usecase.ErrNotFound {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminOrderMarkPaid(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload usecase.PaymentInput
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
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
	items, total, err := h.ticketSvc.List(c, usecase.TicketFilter{UserID: userID, Status: status, Keyword: keyword, Limit: limit, Offset: offset})
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
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
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
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var payload struct {
		Subject *string `json:"subject"`
		Status  *string `json:"status"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if payload.Subject != nil {
		ticket.Subject = strings.TrimSpace(*payload.Subject)
	}
	if payload.Status != nil {
		ticket.Status = strings.TrimSpace(*payload.Status)
	}
	if ticket.Subject == "" || ticket.Status == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subject and status required"})
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
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var payload struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	msg, err := h.ticketSvc.AddMessage(c, ticket, getUserID(c), "admin", payload.Content)
	if err != nil {
		if err == usecase.ErrInvalidInput {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
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

func (h *Handler) AdminVPSList(c *gin.Context) {
	limit, offset := paging(c)
	items, total, err := h.adminSvc.ListInstances(c, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": h.toVPSInstanceDTOsWithLifecycle(c, items), "total": total})
}

func (h *Handler) AdminVPSCreate(c *gin.Context) {
	var payload struct {
		UserID               int64          `json:"user_id"`
		OrderItemID          int64          `json:"order_item_id"`
		AutomationInstanceID string         `json:"automation_instance_id"`
		Name                 string         `json:"name"`
		Region               string         `json:"region"`
		RegionID             int64          `json:"region_id"`
		SystemID             int64          `json:"system_id"`
		Status               string         `json:"status"`
		AutomationState      int            `json:"automation_state"`
		AdminStatus          string         `json:"admin_status"`
		ExpireAt             string         `json:"expire_at"`
		PanelURLCache        string         `json:"panel_url_cache"`
		Spec                 map[string]any `json:"spec"`
		AccessInfo           map[string]any `json:"access_info"`
		Provision            bool           `json:"provision"`
		LineID               int64          `json:"line_id"`
		PackageID            int64          `json:"package_id"`
		PackageName          string         `json:"package_name"`
		OS                   string         `json:"os"`
		CPU                  int            `json:"cpu"`
		MemoryGB             int            `json:"memory_gb"`
		DiskGB               int            `json:"disk_gb"`
		BandwidthMB          int            `json:"bandwidth_mbps"`
		PortNum              int            `json:"port_num"`
		MonthlyPrice         float64        `json:"monthly_price"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if payload.PackageID > 0 && h.catalogSvc != nil {
		if pkg, err := h.catalogSvc.GetPackage(c, payload.PackageID); err == nil {
			if payload.PackageName == "" {
				payload.PackageName = pkg.Name
			}
			if payload.CPU == 0 {
				payload.CPU = pkg.Cores
			}
			if payload.MemoryGB == 0 {
				payload.MemoryGB = pkg.MemoryGB
			}
			if payload.DiskGB == 0 {
				payload.DiskGB = pkg.DiskGB
			}
			if payload.BandwidthMB == 0 {
				payload.BandwidthMB = pkg.BandwidthMB
			}
			if payload.PortNum == 0 {
				payload.PortNum = pkg.PortNum
			}
			if payload.MonthlyPrice == 0 {
				payload.MonthlyPrice = centsToFloat(pkg.Monthly)
			}
			if plan, err := h.catalogSvc.GetPlanGroup(c, pkg.PlanGroupID); err == nil {
				if payload.LineID == 0 {
					payload.LineID = plan.LineID
				}
				if payload.RegionID == 0 {
					payload.RegionID = plan.RegionID
				}
			}
		}
	}
	if payload.Region == "" && payload.RegionID > 0 && h.catalogSvc != nil {
		if region, err := h.catalogSvc.GetRegion(c, payload.RegionID); err == nil {
			payload.Region = region.Name
		}
	}
	var expireAt *time.Time
	if payload.ExpireAt != "" {
		t, err := time.Parse(time.RFC3339, payload.ExpireAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid expire_at"})
			return
		}
		expireAt = &t
	}
	specJSON := "{}"
	if payload.Spec != nil {
		specJSON = mustJSON(payload.Spec)
	}
	accessJSON := "{}"
	if payload.AccessInfo != nil {
		accessJSON = mustJSON(payload.AccessInfo)
	}
	osName := strings.TrimSpace(payload.OS)
	if payload.Provision && osName == "" && payload.SystemID > 0 {
		if img, err := h.catalogSvc.GetSystemImage(c, payload.SystemID); err == nil {
			osName = img.Name
		}
	}
	inst, err := h.adminVPS.Create(c, getUserID(c), usecase.AdminVPSCreateInput{
		UserID:               payload.UserID,
		OrderItemID:          payload.OrderItemID,
		AutomationInstanceID: payload.AutomationInstanceID,
		Name:                 payload.Name,
		Region:               payload.Region,
		RegionID:             payload.RegionID,
		SystemID:             payload.SystemID,
		Status:               domain.VPSStatus(payload.Status),
		AutomationState:      payload.AutomationState,
		AdminStatus:          domain.VPSAdminStatus(payload.AdminStatus),
		ExpireAt:             expireAt,
		PanelURLCache:        payload.PanelURLCache,
		SpecJSON:             specJSON,
		AccessInfoJSON:       accessJSON,
		Provision:            payload.Provision,
		LineID:               payload.LineID,
		PackageID:            payload.PackageID,
		PackageName:          payload.PackageName,
		OS:                   osName,
		CPU:                  payload.CPU,
		MemoryGB:             payload.MemoryGB,
		DiskGB:               payload.DiskGB,
		BandwidthMB:          payload.BandwidthMB,
		PortNum:              payload.PortNum,
		MonthlyPrice:         floatToCents(payload.MonthlyPrice),
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, h.toVPSInstanceDTOWithLifecycle(c, inst))
}

func (h *Handler) AdminVPSDetail(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	inst, err := h.vpsRepo.GetInstance(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, h.toVPSInstanceDTOWithLifecycle(c, inst))
}

func (h *Handler) AdminVPSUpdate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		PackageID     *int64         `json:"package_id"`
		PackageName   *string        `json:"package_name"`
		MonthlyPrice  *float64       `json:"monthly_price"`
		SystemID      *int64         `json:"system_id"`
		Spec          map[string]any `json:"spec"`
		Status        *string        `json:"status"`
		AdminStatus   *string        `json:"admin_status"`
		CPU           *int           `json:"cpu"`
		MemoryGB      *int           `json:"memory_gb"`
		DiskGB        *int           `json:"disk_gb"`
		BandwidthMB   *int           `json:"bandwidth_mbps"`
		PortNum       *int           `json:"port_num"`
		PanelURLCache *string        `json:"panel_url_cache"`
		AccessInfo    map[string]any `json:"access_info"`
		SyncMode      string         `json:"sync_mode"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if payload.PackageID != nil && payload.PackageName == nil && h.catalogSvc != nil {
		if pkg, err := h.catalogSvc.GetPackage(c, *payload.PackageID); err == nil {
			name := pkg.Name
			payload.PackageName = &name
		}
	}
	specJSON := ""
	if payload.Spec != nil {
		specJSON = mustJSON(payload.Spec)
	}
	accessJSON := ""
	if payload.AccessInfo != nil {
		accessJSON = mustJSON(payload.AccessInfo)
	}
	var statusVal *domain.VPSStatus
	if payload.Status != nil {
		tmp := domain.VPSStatus(*payload.Status)
		statusVal = &tmp
	}
	var adminStatusVal *domain.VPSAdminStatus
	if payload.AdminStatus != nil {
		tmp := domain.VPSAdminStatus(*payload.AdminStatus)
		adminStatusVal = &tmp
	}
	var monthlyPrice *int64
	if payload.MonthlyPrice != nil {
		val := floatToCents(*payload.MonthlyPrice)
		monthlyPrice = &val
	}
	input := usecase.AdminVPSUpdateInput{
		PackageID:     payload.PackageID,
		PackageName:   payload.PackageName,
		MonthlyPrice:  monthlyPrice,
		SystemID:      payload.SystemID,
		Status:        statusVal,
		AdminStatus:   adminStatusVal,
		CPU:           payload.CPU,
		MemoryGB:      payload.MemoryGB,
		DiskGB:        payload.DiskGB,
		BandwidthMB:   payload.BandwidthMB,
		PortNum:       payload.PortNum,
		PanelURLCache: payload.PanelURLCache,
		SyncMode:      strings.TrimSpace(payload.SyncMode),
	}
	if specJSON != "" {
		input.SpecJSON = &specJSON
	}
	if accessJSON != "" {
		input.AccessInfoJSON = &accessJSON
	}
	inst, err := h.adminVPS.Update(c, getUserID(c), id, input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, h.toVPSInstanceDTOWithLifecycle(c, inst))
}

func (h *Handler) AdminVPSUpdateExpire(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		ExpireAt string `json:"expire_at"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if payload.ExpireAt == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "expire_at required"})
		return
	}
	t, err := time.Parse("2006-01-02 15:04:05", payload.ExpireAt)
	if err != nil {
		t, err = time.Parse("2006-01-02", payload.ExpireAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid expire_at"})
			return
		}
	}
	inst, err := h.adminVPS.UpdateExpireAt(c, getUserID(c), id, t)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, h.toVPSInstanceDTOWithLifecycle(c, inst))
}

func (h *Handler) AdminVPSLock(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.adminVPS.SetAdminStatus(c, getUserID(c), id, domain.VPSAdminStatusLocked, "lock"); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminVPSUnlock(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.adminVPS.SetAdminStatus(c, getUserID(c), id, domain.VPSAdminStatusNormal, "unlock"); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminVPSDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&payload)
	if err := h.adminVPS.Delete(c, getUserID(c), id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.walletOrder != nil {
		_, _, _ = h.walletOrder.AutoRefundOnAdminDelete(c, getUserID(c), id, payload.Reason)
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminVPSResize(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		CPU       int `json:"cpu"`
		MemoryGB  int `json:"memory_gb"`
		DiskGB    int `json:"disk_gb"`
		Bandwidth int `json:"bandwidth_mbps"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	req := usecase.AutomationElasticUpdateRequest{}
	if payload.CPU > 0 {
		req.CPU = &payload.CPU
	}
	if payload.MemoryGB > 0 {
		req.MemoryGB = &payload.MemoryGB
	}
	if payload.DiskGB > 0 {
		req.DiskGB = &payload.DiskGB
	}
	if payload.Bandwidth > 0 {
		req.Bandwidth = &payload.Bandwidth
	}
	if err := h.adminVPS.Resize(c, getUserID(c), id, req, mustJSON(payload)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminVPSStatus(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	status := domain.VPSAdminStatus(payload.Status)
	if err := h.adminVPS.SetAdminStatus(c, getUserID(c), id, status, payload.Reason); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminVPSEmergencyRenew(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	inst, err := h.adminVPS.Get(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	_, err = h.orderSvc.CreateEmergencyRenewOrder(c, inst.UserID, inst.ID)
	if err != nil {
		status := http.StatusBadRequest
		if err == usecase.ErrConflict {
			status = http.StatusConflict
		} else if err == usecase.ErrForbidden {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	updated, _ := h.adminVPS.Get(c, id)
	c.JSON(http.StatusOK, h.toVPSInstanceDTOWithLifecycle(c, updated))
}

func (h *Handler) AdminVPSRefresh(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	inst, err := h.adminVPS.Refresh(c, getUserID(c), id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, h.toVPSInstanceDTOWithLifecycle(c, inst))
}

func (h *Handler) AdminAuditLogs(c *gin.Context) {
	limit, offset := paging(c)
	items, total, err := h.adminSvc.ListAuditLogs(c, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toAdminAuditLogDTOs(items), "total": total})
}

func (h *Handler) AdminSystemImages(c *gin.Context) {
	lineID, _ := strconv.ParseInt(c.Query("line_id"), 10, 64)
	planGroupID, _ := strconv.ParseInt(c.Query("plan_group_id"), 10, 64)
	if planGroupID > 0 {
		if plan, err := h.catalogSvc.GetPlanGroup(c, planGroupID); err == nil {
			lineID = plan.LineID
		}
	}
	items, err := h.catalogSvc.ListSystemImages(c, lineID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toSystemImageDTOs(items)})
}

func (h *Handler) AdminRegions(c *gin.Context) {
	items, err := h.catalogSvc.ListRegions(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toRegionDTOs(items)})
}

func (h *Handler) AdminRegionCreate(c *gin.Context) {
	var payload RegionDTO
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	region := regionDTOToDomain(payload)
	if err := h.catalogSvc.CreateRegion(c, &region); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toRegionDTO(region))
}

func (h *Handler) AdminRegionUpdate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload RegionDTO
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	payload.ID = id
	region := regionDTOToDomain(payload)
	if err := h.catalogSvc.UpdateRegion(c, region); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toRegionDTO(region))
}

func (h *Handler) AdminRegionDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.catalogSvc.DeleteRegion(c, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminRegionBulkDelete(c *gin.Context) {
	var payload struct {
		IDs []int64 `json:"ids"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if len(payload.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids required"})
		return
	}
	for _, id := range payload.IDs {
		if err := h.catalogSvc.DeleteRegion(c, id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPlanGroups(c *gin.Context) {
	items, err := h.catalogSvc.ListPlanGroups(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toPlanGroupDTOs(items)})
}

func (h *Handler) AdminLines(c *gin.Context) {
	h.AdminPlanGroups(c)
}

func (h *Handler) AdminPlanGroupCreate(c *gin.Context) {
	var payload PlanGroupDTO
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	plan := planGroupDTOToDomain(payload)
	if err := h.catalogSvc.CreatePlanGroup(c, &plan); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toPlanGroupDTO(plan))
}

func (h *Handler) AdminLineCreate(c *gin.Context) {
	h.AdminPlanGroupCreate(c)
}

func (h *Handler) AdminPlanGroupUpdate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		RegionID          *int64   `json:"region_id"`
		Name              *string  `json:"name"`
		LineID            *int64   `json:"line_id"`
		UnitCore          *float64 `json:"unit_core"`
		UnitMem           *float64 `json:"unit_mem"`
		UnitDisk          *float64 `json:"unit_disk"`
		UnitBW            *float64 `json:"unit_bw"`
		AddCoreMin        *int     `json:"add_core_min"`
		AddCoreMax        *int     `json:"add_core_max"`
		AddCoreStep       *int     `json:"add_core_step"`
		AddMemMin         *int     `json:"add_mem_min"`
		AddMemMax         *int     `json:"add_mem_max"`
		AddMemStep        *int     `json:"add_mem_step"`
		AddDiskMin        *int     `json:"add_disk_min"`
		AddDiskMax        *int     `json:"add_disk_max"`
		AddDiskStep       *int     `json:"add_disk_step"`
		AddBWMin          *int     `json:"add_bw_min"`
		AddBWMax          *int     `json:"add_bw_max"`
		AddBWStep         *int     `json:"add_bw_step"`
		Active            *bool    `json:"active"`
		Visible           *bool    `json:"visible"`
		CapacityRemaining *int     `json:"capacity_remaining"`
		SortOrder         *int     `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	plan, err := h.catalogSvc.GetPlanGroup(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if payload.RegionID != nil {
		plan.RegionID = *payload.RegionID
	}
	if payload.Name != nil {
		plan.Name = *payload.Name
	}
	if payload.LineID != nil {
		plan.LineID = *payload.LineID
	}
	if payload.UnitCore != nil {
		plan.UnitCore = floatToCents(*payload.UnitCore)
	}
	if payload.UnitMem != nil {
		plan.UnitMem = floatToCents(*payload.UnitMem)
	}
	if payload.UnitDisk != nil {
		plan.UnitDisk = floatToCents(*payload.UnitDisk)
	}
	if payload.UnitBW != nil {
		plan.UnitBW = floatToCents(*payload.UnitBW)
	}
	if payload.AddCoreMin != nil {
		plan.AddCoreMin = *payload.AddCoreMin
	}
	if payload.AddCoreMax != nil {
		plan.AddCoreMax = *payload.AddCoreMax
	}
	if payload.AddCoreStep != nil {
		plan.AddCoreStep = *payload.AddCoreStep
	}
	if payload.AddMemMin != nil {
		plan.AddMemMin = *payload.AddMemMin
	}
	if payload.AddMemMax != nil {
		plan.AddMemMax = *payload.AddMemMax
	}
	if payload.AddMemStep != nil {
		plan.AddMemStep = *payload.AddMemStep
	}
	if payload.AddDiskMin != nil {
		plan.AddDiskMin = *payload.AddDiskMin
	}
	if payload.AddDiskMax != nil {
		plan.AddDiskMax = *payload.AddDiskMax
	}
	if payload.AddDiskStep != nil {
		plan.AddDiskStep = *payload.AddDiskStep
	}
	if payload.AddBWMin != nil {
		plan.AddBWMin = *payload.AddBWMin
	}
	if payload.AddBWMax != nil {
		plan.AddBWMax = *payload.AddBWMax
	}
	if payload.AddBWStep != nil {
		plan.AddBWStep = *payload.AddBWStep
	}
	if payload.Active != nil {
		plan.Active = *payload.Active
	}
	if payload.Visible != nil {
		plan.Visible = *payload.Visible
	}
	if payload.CapacityRemaining != nil {
		plan.CapacityRemaining = *payload.CapacityRemaining
	}
	if payload.SortOrder != nil {
		plan.SortOrder = *payload.SortOrder
	}
	if err := h.catalogSvc.UpdatePlanGroup(c, plan); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toPlanGroupDTO(plan))
}

func (h *Handler) AdminLineUpdate(c *gin.Context) {
	h.AdminPlanGroupUpdate(c)
}

func (h *Handler) AdminLineSystemImages(c *gin.Context) {
	planGroupID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		ImageIDs []int64 `json:"image_ids"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if planGroupID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid line id"})
		return
	}
	plan, err := h.catalogSvc.GetPlanGroup(c, planGroupID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if plan.LineID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "line_id required"})
		return
	}
	if err := h.catalogSvc.SetLineSystemImages(c, plan.LineID, payload.ImageIDs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPlanGroupDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.catalogSvc.DeletePlanGroup(c, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPlanGroupBulkDelete(c *gin.Context) {
	var payload struct {
		IDs []int64 `json:"ids"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if len(payload.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids required"})
		return
	}
	for _, id := range payload.IDs {
		if err := h.catalogSvc.DeletePlanGroup(c, id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminLineDelete(c *gin.Context) {
	h.AdminPlanGroupDelete(c)
}

func (h *Handler) AdminPackages(c *gin.Context) {
	planGroupID, _ := strconv.ParseInt(c.Query("plan_group_id"), 10, 64)
	items, err := h.catalogSvc.ListPackages(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list error"})
		return
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

func (h *Handler) AdminProducts(c *gin.Context) {
	h.AdminPackages(c)
}

func (h *Handler) AdminPackageCreate(c *gin.Context) {
	var payload PackageDTO
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	pkg := packageDTOToDomain(payload)
	if err := h.catalogSvc.CreatePackage(c, &pkg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toPackageDTO(pkg))
}

func (h *Handler) AdminProductCreate(c *gin.Context) {
	h.AdminPackageCreate(c)
}

func (h *Handler) AdminPackageUpdate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		PlanGroupID       *int64   `json:"plan_group_id"`
		ProductID         *int64   `json:"product_id"`
		Name              *string  `json:"name"`
		Cores             *int     `json:"cores"`
		MemoryGB          *int     `json:"memory_gb"`
		DiskGB            *int     `json:"disk_gb"`
		BandwidthMB       *int     `json:"bandwidth_mbps"`
		CPUModel          *string  `json:"cpu_model"`
		MonthlyPrice      *float64 `json:"monthly_price"`
		PortNum           *int     `json:"port_num"`
		SortOrder         *int     `json:"sort_order"`
		Active            *bool    `json:"active"`
		Visible           *bool    `json:"visible"`
		CapacityRemaining *int     `json:"capacity_remaining"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	pkg, err := h.catalogSvc.GetPackage(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if payload.PlanGroupID != nil {
		if *payload.PlanGroupID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan_group_id"})
			return
		}
		pkg.PlanGroupID = *payload.PlanGroupID
	}
	if payload.ProductID != nil {
		pkg.ProductID = *payload.ProductID
	}
	if payload.Name != nil {
		pkg.Name = *payload.Name
	}
	if payload.Cores != nil {
		pkg.Cores = *payload.Cores
	}
	if payload.MemoryGB != nil {
		pkg.MemoryGB = *payload.MemoryGB
	}
	if payload.DiskGB != nil {
		pkg.DiskGB = *payload.DiskGB
	}
	if payload.BandwidthMB != nil {
		pkg.BandwidthMB = *payload.BandwidthMB
	}
	if payload.CPUModel != nil {
		pkg.CPUModel = *payload.CPUModel
	}
	if payload.MonthlyPrice != nil {
		pkg.Monthly = floatToCents(*payload.MonthlyPrice)
	}
	if payload.PortNum != nil {
		pkg.PortNum = *payload.PortNum
	}
	if payload.SortOrder != nil {
		pkg.SortOrder = *payload.SortOrder
	}
	if payload.Active != nil {
		pkg.Active = *payload.Active
	}
	if payload.Visible != nil {
		pkg.Visible = *payload.Visible
	}
	if payload.CapacityRemaining != nil {
		pkg.CapacityRemaining = *payload.CapacityRemaining
	}
	if err := h.catalogSvc.UpdatePackage(c, pkg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toPackageDTO(pkg))
}

func (h *Handler) AdminProductUpdate(c *gin.Context) {
	h.AdminPackageUpdate(c)
}

func (h *Handler) AdminPackageDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.catalogSvc.DeletePackage(c, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPackageBulkDelete(c *gin.Context) {
	var payload struct {
		IDs []int64 `json:"ids"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if len(payload.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids required"})
		return
	}
	for _, id := range payload.IDs {
		if err := h.catalogSvc.DeletePackage(c, id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminProductDelete(c *gin.Context) {
	h.AdminPackageDelete(c)
}
func (h *Handler) AdminBillingCycles(c *gin.Context) {
	items, err := h.catalogSvc.ListBillingCycles(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toBillingCycleDTOs(items)})
}

func (h *Handler) AdminBillingCycleCreate(c *gin.Context) {
	var payload BillingCycleDTO
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	cycle := billingCycleDTOToDomain(payload)
	if err := h.catalogSvc.CreateBillingCycle(c, &cycle); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toBillingCycleDTO(cycle))
}

func (h *Handler) AdminBillingCycleUpdate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload BillingCycleDTO
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	payload.ID = id
	cycle := billingCycleDTOToDomain(payload)
	if err := h.catalogSvc.UpdateBillingCycle(c, cycle); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toBillingCycleDTO(cycle))
}

func (h *Handler) AdminBillingCycleDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.catalogSvc.DeleteBillingCycle(c, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminBillingCycleBulkDelete(c *gin.Context) {
	var payload struct {
		IDs []int64 `json:"ids"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if len(payload.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids required"})
		return
	}
	for _, id := range payload.IDs {
		if err := h.catalogSvc.DeleteBillingCycle(c, id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminSystemImageCreate(c *gin.Context) {
	var payload SystemImageDTO
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	img := systemImageDTOToDomain(payload)
	if err := h.catalogSvc.CreateSystemImage(c, &img); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toSystemImageDTO(img))
}

func (h *Handler) AdminSystemImageUpdate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload SystemImageDTO
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	payload.ID = id
	img := systemImageDTOToDomain(payload)
	if err := h.catalogSvc.UpdateSystemImage(c, img); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toSystemImageDTO(img))
}

func (h *Handler) AdminSystemImageDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.catalogSvc.DeleteSystemImage(c, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminSystemImageBulkDelete(c *gin.Context) {
	var payload struct {
		IDs []int64 `json:"ids"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if len(payload.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids required"})
		return
	}
	for _, id := range payload.IDs {
		if err := h.catalogSvc.DeleteSystemImage(c, id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminSystemImageSync(c *gin.Context) {
	lineID, _ := strconv.ParseInt(c.Query("line_id"), 10, 64)
	if lineID <= 0 {
		planGroupID, _ := strconv.ParseInt(c.Query("plan_group_id"), 10, 64)
		if planGroupID > 0 {
			if plan, err := h.catalogSvc.GetPlanGroup(c, planGroupID); err == nil {
				lineID = plan.LineID
			}
		}
	}
	if lineID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "line_id required"})
		return
	}
	images, err := h.automation.ListImages(c, lineID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	existingImages, _ := h.catalogSvc.ListSystemImages(c, 0)
	byImageID := map[int64]domain.SystemImage{}
	for _, img := range existingImages {
		if img.ImageID > 0 {
			byImageID[img.ImageID] = img
		}
	}
	mappedIDs := make([]int64, 0, len(images))
	for _, img := range images {
		imgType := "linux"
		if strings.Contains(strings.ToLower(img.Type), "win") {
			imgType = "windows"
		}
		if existing, ok := byImageID[img.ImageID]; ok {
			existing.Name = img.Name
			existing.Type = imgType
			_ = h.catalogSvc.UpdateSystemImage(c, existing)
			if existing.ID > 0 {
				mappedIDs = append(mappedIDs, existing.ID)
			}
			continue
		}
		newImg := domain.SystemImage{ImageID: img.ImageID, Name: img.Name, Type: imgType, Enabled: true}
		_ = h.catalogSvc.CreateSystemImage(c, &newImg)
		if newImg.ID > 0 {
			mappedIDs = append(mappedIDs, newImg.ID)
		}
	}
	if err := h.catalogSvc.SetLineSystemImages(c, lineID, mappedIDs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": len(images)})
}

func (h *Handler) AdminAPIKeys(c *gin.Context) {
	limit, offset := paging(c)
	items, total, err := h.adminSvc.ListAPIKeys(c, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toAPIKeyDTOs(items), "total": total})
}

func (h *Handler) AdminAPIKeyCreate(c *gin.Context) {
	var payload struct {
		Name              string   `json:"name"`
		PermissionGroupID *int64   `json:"permission_group_id"`
		Scopes            []string `json:"scopes"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	raw, key, err := h.adminSvc.CreateAPIKey(c, getUserID(c), payload.Name, payload.PermissionGroupID, payload.Scopes)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"api_key": raw, "record": toAPIKeyDTO(key)})
}

func (h *Handler) AdminAPIKeyUpdate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	status := domain.APIKeyStatus(payload.Status)
	if err := h.adminSvc.UpdateAPIKeyStatus(c, getUserID(c), id, status); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminSettingsList(c *gin.Context) {
	items, err := h.adminSvc.ListSettings(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toSettingDTOs(items)})
}

func (h *Handler) AdminSettingsUpdate(c *gin.Context) {
	var payload struct {
		Key   string `json:"key"`
		Value string `json:"value"`
		Items []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"items"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if len(payload.Items) > 0 {
		for _, item := range payload.Items {
			if strings.TrimSpace(item.Key) == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid key"})
				return
			}
			if err := h.adminSvc.UpdateSetting(c, getUserID(c), item.Key, item.Value); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		}
	} else {
		if err := h.adminSvc.UpdateSetting(c, getUserID(c), payload.Key, payload.Value); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminDebugStatus(c *gin.Context) {
	if h.settings == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	enabled := strings.ToLower(getSettingValue(c, h.settings, "debug_enabled")) == "true"
	c.JSON(http.StatusOK, gin.H{"enabled": enabled})
}

func (h *Handler) AdminDebugStatusUpdate(c *gin.Context) {
	if h.settings == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	var payload struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if err := h.adminSvc.UpdateSetting(c, getUserID(c), "debug_enabled", boolToString(payload.Enabled)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminDebugLogs(c *gin.Context) {
	if h.settings == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	if strings.ToLower(getSettingValue(c, h.settings, "debug_enabled")) != "true" {
		c.JSON(http.StatusForbidden, gin.H{"error": "debug disabled"})
		return
	}
	limit, offset := paging(c)
	types := strings.ToLower(strings.TrimSpace(c.Query("types")))
	includeAll := types == ""
	includeType := func(name string) bool {
		if includeAll {
			return true
		}
		for _, item := range strings.Split(types, ",") {
			if strings.TrimSpace(item) == name {
				return true
			}
		}
		return false
	}

	resp := gin.H{}
	if includeType("audit") && h.adminSvc != nil {
		items, total, err := h.adminSvc.ListAuditLogs(c, limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "list audit logs error"})
			return
		}
		resp["audit_logs"] = gin.H{"items": toAdminAuditLogDTOs(items), "total": total}
	}
	if includeType("automation") && h.automationLog != nil {
		orderID, _ := strconv.ParseInt(c.Query("order_id"), 10, 64)
		items, total, err := h.automationLog.ListAutomationLogs(c, orderID, limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "list automation logs error"})
			return
		}
		resp["automation_logs"] = gin.H{"items": toAutomationLogDTOs(items), "total": total}
	}
	if includeType("sync") && h.integration != nil {
		target := c.Query("target")
		items, total, err := h.integration.ListSyncLogs(c, target, limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "list sync logs error"})
			return
		}
		resp["sync_logs"] = gin.H{"items": toIntegrationSyncLogDTOs(items), "total": total}
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) AdminAutomationConfig(c *gin.Context) {
	if h.integration == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	cfg, err := h.integration.GetAutomationConfig(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "config error"})
		return
	}
	c.JSON(http.StatusOK, cfg)
}

func (h *Handler) AdminAutomationConfigUpdate(c *gin.Context) {
	if h.integration == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	var payload usecase.AutomationConfig
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if err := h.integration.UpdateAutomationConfig(c, getUserID(c), payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminAutomationSync(c *gin.Context) {
	if h.integration == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	mode := c.Query("mode")
	result, err := h.integration.SyncAutomation(c, mode)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) AdminAutomationSyncLogs(c *gin.Context) {
	if h.integration == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	limit, offset := paging(c)
	target := c.Query("target")
	items, total, err := h.integration.ListSyncLogs(c, target, limit, offset)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toIntegrationSyncLogDTOs(items), "total": total})
}

func (h *Handler) AdminRobotConfig(c *gin.Context) {
	if h.settings == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	webhooks := usecase.ParseRobotWebhookConfigs(getSettingValue(c, h.settings, "robot_webhooks"))
	c.JSON(http.StatusOK, gin.H{
		"url":      getSettingValue(c, h.settings, "robot_webhook_url"),
		"secret":   getSettingValue(c, h.settings, "robot_webhook_secret"),
		"enabled":  strings.ToLower(getSettingValue(c, h.settings, "robot_webhook_enabled")) == "true",
		"webhooks": webhooks,
	})
}

func (h *Handler) AdminRobotConfigUpdate(c *gin.Context) {
	var payload struct {
		URL      string                       `json:"url"`
		Secret   string                       `json:"secret"`
		Enabled  bool                         `json:"enabled"`
		Webhooks []usecase.RobotWebhookConfig `json:"webhooks"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if payload.Webhooks != nil {
		raw, _ := json.Marshal(payload.Webhooks)
		_ = h.adminSvc.UpdateSetting(c, getUserID(c), "robot_webhooks", string(raw))
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}
	if payload.URL != "" || payload.Secret != "" || payload.Enabled {
		if err := h.adminSvc.UpdateSetting(c, getUserID(c), "robot_webhook_url", payload.URL); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		_ = h.adminSvc.UpdateSetting(c, getUserID(c), "robot_webhook_secret", payload.Secret)
		_ = h.adminSvc.UpdateSetting(c, getUserID(c), "robot_webhook_enabled", boolToString(payload.Enabled))
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": "no updates"})
}

func (h *Handler) AdminRobotTest(c *gin.Context) {
	if h.settings == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	if h.broker == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "event broker not available"})
		return
	}
	var payload struct {
		Event string `json:"event"`
		Data  any    `json:"data"`
	}
	_ = c.ShouldBindJSON(&payload)
	eventType := strings.TrimSpace(payload.Event)
	if eventType == "" {
		eventType = "webhook.test"
	}
	ev, err := h.broker.Publish(c, 0, eventType, map[string]any{
		"event":     eventType,
		"timestamp": time.Now().Unix(),
		"data":      payload.Data,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	notifier := robot.NewWebhookNotifier(h.settings)
	_ = notifier.NotifyOrderEvent(c, ev)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminRealNameConfig(c *gin.Context) {
	if h.realnameSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	enabled, provider, actions := h.realnameSvc.GetConfig(c)
	c.JSON(http.StatusOK, gin.H{
		"enabled":       enabled,
		"provider":      provider,
		"block_actions": actions,
	})
}

func (h *Handler) AdminRealNameConfigUpdate(c *gin.Context) {
	if h.realnameSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	var payload struct {
		Enabled      bool     `json:"enabled"`
		Provider     string   `json:"provider"`
		BlockActions []string `json:"block_actions"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if err := h.realnameSvc.UpdateConfig(c, payload.Enabled, payload.Provider, payload.BlockActions); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminRealNameProviders(c *gin.Context) {
	if h.realnameSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	type providerInfo struct {
		Key  string `json:"key"`
		Name string `json:"name"`
	}
	out := []providerInfo{}
	for _, provider := range h.realnameSvc.Providers() {
		out = append(out, providerInfo{Key: provider.Key(), Name: provider.Name()})
	}
	c.JSON(http.StatusOK, gin.H{"items": out})
}

func (h *Handler) AdminRealNameRecords(c *gin.Context) {
	if h.realnameSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	limit, offset := paging(c)
	var userID *int64
	if val := c.Query("user_id"); val != "" {
		if id, err := strconv.ParseInt(val, 10, 64); err == nil {
			userID = &id
		}
	}
	items, total, err := h.realnameSvc.List(c, userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp := make([]RealNameVerificationDTO, 0, len(items))
	for _, item := range items {
		resp = append(resp, toRealNameVerificationDTO(item))
	}
	c.JSON(http.StatusOK, gin.H{"items": resp, "total": total})
}

func (h *Handler) AdminSMTPConfig(c *gin.Context) {
	if h.settings == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"host":    getSettingValue(c, h.settings, "smtp_host"),
		"port":    getSettingValue(c, h.settings, "smtp_port"),
		"user":    getSettingValue(c, h.settings, "smtp_user"),
		"pass":    getSettingValue(c, h.settings, "smtp_pass"),
		"from":    getSettingValue(c, h.settings, "smtp_from"),
		"enabled": strings.ToLower(getSettingValue(c, h.settings, "smtp_enabled")) == "true",
	})
}

func (h *Handler) AdminSMTPConfigUpdate(c *gin.Context) {
	var payload struct {
		Host    string `json:"host"`
		Port    string `json:"port"`
		User    string `json:"user"`
		Pass    string `json:"pass"`
		From    string `json:"from"`
		Enabled bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	_ = h.adminSvc.UpdateSetting(c, getUserID(c), "smtp_host", payload.Host)
	_ = h.adminSvc.UpdateSetting(c, getUserID(c), "smtp_port", payload.Port)
	_ = h.adminSvc.UpdateSetting(c, getUserID(c), "smtp_user", payload.User)
	_ = h.adminSvc.UpdateSetting(c, getUserID(c), "smtp_pass", payload.Pass)
	_ = h.adminSvc.UpdateSetting(c, getUserID(c), "smtp_from", payload.From)
	_ = h.adminSvc.UpdateSetting(c, getUserID(c), "smtp_enabled", boolToString(payload.Enabled))
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminSMTPTest(c *gin.Context) {
	if h.settings == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	var payload struct {
		To           string         `json:"to"`
		TemplateName string         `json:"template_name"`
		Subject      string         `json:"subject"`
		Body         string         `json:"body"`
		Variables    map[string]any `json:"variables"`
		HTML         bool           `json:"html"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if strings.TrimSpace(payload.To) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "to required"})
		return
	}
	subject := strings.TrimSpace(payload.Subject)
	body := payload.Body
	if payload.TemplateName != "" {
		templates, _ := h.settings.ListEmailTemplates(c)
		found := false
		for _, tmpl := range templates {
			if tmpl.Name == payload.TemplateName {
				subject = tmpl.Subject
				body = tmpl.Body
				found = true
				break
			}
		}
		if !found {
			c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
			return
		}
	}
	if subject == "" {
		subject = "SMTP Test"
	}
	if strings.TrimSpace(body) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "body required"})
		return
	}
	data := map[string]any{
		"now": time.Now().Format(time.RFC3339),
	}
	for k, v := range payload.Variables {
		data[k] = v
	}
	subject = usecase.RenderTemplate(subject, data, false)
	body = usecase.RenderTemplate(body, data, usecase.IsHTMLContent(body))
	if payload.HTML && !usecase.IsHTMLContent(body) {
		body = "<html><body><pre>" + html.EscapeString(body) + "</pre></body></html>"
	}
	sender := email.NewSender(h.settings)
	if err := sender.Send(c, payload.To, subject, body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminEmailTemplates(c *gin.Context) {
	items, err := h.adminSvc.ListEmailTemplates(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toEmailTemplateDTOs(items)})
}

func (h *Handler) AdminEmailTemplateUpsert(c *gin.Context) {
	var payload EmailTemplateDTO
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	payload.ID = id
	tmpl := emailTemplateDTOToDomain(payload)
	if err := h.adminSvc.UpsertEmailTemplate(c, getUserID(c), &tmpl); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toEmailTemplateDTO(tmpl))
}

func (h *Handler) AdminEmailTemplateDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if h.settings == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	if err := h.settings.DeleteEmailTemplate(c, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminDashboardOverview(c *gin.Context) {
	if h.reportSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	overview, err := h.reportSvc.Overview(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "report error"})
		return
	}
	c.JSON(http.StatusOK, overview)
}

func (h *Handler) AdminDashboardRevenue(c *gin.Context) {
	if h.reportSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	period := c.Query("period")
	if period == "month" {
		points, err := h.reportSvc.RevenueByMonth(c, 6)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "report error"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": points})
		return
	}
	points, err := h.reportSvc.RevenueByDay(c, 30)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "report error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": points})
}

func (h *Handler) AdminDashboardVPSStatus(c *gin.Context) {
	if h.reportSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	items, err := h.reportSvc.VPSStatus(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "report error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) AdminAdmins(c *gin.Context) {
	limit, offset := paging(c)
	status := strings.TrimSpace(c.Query("status"))
	if status == "" {
		status = "active"
	}
	admins, total, err := h.adminSvc.ListAdmins(c, status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": admins, "total": total})
}

func (h *Handler) AdminAdminCreate(c *gin.Context) {
	var payload struct {
		Username          string `json:"username" binding:"required"`
		Email             string `json:"email" binding:"required,email"`
		QQ                string `json:"qq"`
		Password          string `json:"password" binding:"required"`
		PermissionGroupID *int64 `json:"permission_group_id"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if payload.QQ != "" && !isDigits(payload.QQ) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "qq must be numeric"})
		return
	}
	admin, err := h.adminSvc.CreateAdmin(c, getUserID(c), payload.Username, payload.Email, payload.QQ, payload.Password, payload.PermissionGroupID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toUserDTO(admin))
}

func (h *Handler) AdminAdminUpdate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		Username          string `json:"username" binding:"required"`
		Email             string `json:"email" binding:"required,email"`
		QQ                string `json:"qq"`
		PermissionGroupID *int64 `json:"permission_group_id"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if payload.QQ != "" && !isDigits(payload.QQ) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "qq must be numeric"})
		return
	}
	if id == getUserID(c) {
		if payload.PermissionGroupID != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot update permission group"})
			return
		}
		existing, err := h.users.GetUserByID(c, id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		payload.PermissionGroupID = existing.PermissionGroupID
	}
	if err := h.adminSvc.UpdateAdmin(c, getUserID(c), id, payload.Username, payload.Email, payload.QQ, payload.PermissionGroupID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminAdminStatus(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if id == getUserID(c) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot update self status"})
		return
	}
	status := strings.TrimSpace(payload.Status)
	if status != string(domain.UserStatusActive) && status != string(domain.UserStatusDisabled) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
		return
	}
	if err := h.adminSvc.UpdateAdminStatus(c, getUserID(c), id, domain.UserStatus(status)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminAdminDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.adminSvc.DeleteAdmin(c, getUserID(c), id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPermissionGroups(c *gin.Context) {
	groups, err := h.adminSvc.ListPermissionGroups(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": groups})
}

func (h *Handler) AdminPermissionGroupCreate(c *gin.Context) {
	var payload struct {
		Name        string   `json:"name" binding:"required"`
		Description string   `json:"description"`
		Permissions []string `json:"permissions" binding:"required"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	permJSON := mustJSON(payload.Permissions)
	group := &domain.PermissionGroup{
		Name:            payload.Name,
		Description:     payload.Description,
		PermissionsJSON: permJSON,
	}
	if err := h.adminSvc.CreatePermissionGroup(c, getUserID(c), group); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, group)
}

func (h *Handler) AdminPermissionGroupUpdate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		Name        string   `json:"name" binding:"required"`
		Description string   `json:"description"`
		Permissions []string `json:"permissions" binding:"required"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	permJSON := mustJSON(payload.Permissions)
	group := domain.PermissionGroup{
		ID:              id,
		Name:            payload.Name,
		Description:     payload.Description,
		PermissionsJSON: permJSON,
	}
	if err := h.adminSvc.UpdatePermissionGroup(c, getUserID(c), group); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPermissionGroupDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.adminSvc.DeletePermissionGroup(c, getUserID(c), id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminProfile(c *gin.Context) {
	userID := getUserID(c)
	user, err := h.users.GetUserByID(c, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	dto := toUserDTO(user)
	// Fetch user permissions
	if h.permissionSvc != nil {
		perms, err := h.permissionSvc.GetUserPermissions(c, userID)
		if err == nil {
			dto.Permissions = perms
		}
	}
	c.JSON(http.StatusOK, dto)
}

func (h *Handler) AdminProfileUpdate(c *gin.Context) {
	var payload struct {
		Email string `json:"email" binding:"omitempty,email"`
		QQ    string `json:"qq"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if err := h.adminSvc.UpdateProfile(c, getUserID(c), payload.Email, payload.QQ); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminProfileChangePassword(c *gin.Context) {
	var payload struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if err := h.adminSvc.ChangePassword(c, getUserID(c), payload.OldPassword, payload.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminForgotPassword(c *gin.Context) {
	var payload struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if h.passwordReset == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	if err := h.passwordReset.RequestReset(c, payload.Email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminResetPassword(c *gin.Context) {
	var payload struct {
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if h.passwordReset == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	if err := h.passwordReset.ResetPassword(c, payload.Token, payload.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) SiteSettings(c *gin.Context) {
	if h.settings == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	allowed := map[string]bool{
		"site_name":                true,
		"site_url":                 true,
		"logo_url":                 true,
		"favicon_url":              true,
		"site_description":         true,
		"site_keywords":            true,
		"company_name":             true,
		"contact_phone":            true,
		"contact_email":            true,
		"contact_qq":               true,
		"wechat_qrcode":            true,
		"icp_number":               true,
		"psbe_number":              true,
		"maintenance_mode":         true,
		"maintenance_message":      true,
		"analytics_code":           true,
		"site_nav_items":           true,
		"site_logo":                true,
		"site_icp":                 true,
		"site_maintenance_mode":    true,
		"site_maintenance_message": true,
	}
	aliases := map[string]string{
		"site_logo":                "logo_url",
		"site_icp":                 "icp_number",
		"site_maintenance_mode":    "maintenance_mode",
		"site_maintenance_message": "maintenance_message",
	}
	items, err := h.settings.ListSettings(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list error"})
		return
	}
	filtered := make([]domain.Setting, 0)
	indexed := make(map[string]domain.Setting)
	for _, item := range items {
		if allowed[item.Key] {
			filtered = append(filtered, item)
			indexed[item.Key] = item
		}
	}
	for legacy, current := range aliases {
		if _, ok := indexed[current]; ok {
			continue
		}
		if legacyItem, ok := indexed[legacy]; ok {
			filtered = append(filtered, domain.Setting{Key: current, ValueJSON: legacyItem.ValueJSON})
		}
	}
	c.JSON(http.StatusOK, gin.H{"items": toSettingDTOs(filtered)})
}

func (h *Handler) toVPSInstanceDTOWithLifecycle(c *gin.Context, inst domain.VPSInstance) VPSInstanceDTO {
	dto := toVPSInstanceDTO(inst)
	destroyAt, destroyInDays := h.lifecycleDestroyInfo(c, inst.ExpireAt)
	dto.DestroyAt = destroyAt
	dto.DestroyInDays = destroyInDays
	return dto
}

func (h *Handler) toVPSInstanceDTOsWithLifecycle(c *gin.Context, items []domain.VPSInstance) []VPSInstanceDTO {
	out := make([]VPSInstanceDTO, 0, len(items))
	for _, item := range items {
		out = append(out, h.toVPSInstanceDTOWithLifecycle(c, item))
	}
	return out
}

func (h *Handler) lifecycleDestroyInfo(c *gin.Context, expireAt *time.Time) (*time.Time, *int) {
	if expireAt == nil || h.settings == nil {
		return nil, nil
	}
	enabled, ok := h.getSettingBool(c, "auto_delete_enabled")
	if !ok || !enabled {
		return nil, nil
	}
	days, ok := h.getSettingInt(c, "auto_delete_days")
	if !ok {
		days = 0
	}
	if days < 0 {
		days = 0
	}
	destroyAt := expireAt.Add(time.Duration(days) * 24 * time.Hour)
	inDays := int(math.Ceil(destroyAt.Sub(time.Now()).Hours() / 24))
	return &destroyAt, &inDays
}

func (h *Handler) getSettingInt(c *gin.Context, key string) (int, bool) {
	if h.settings == nil {
		return 0, false
	}
	setting, err := h.settings.GetSetting(c, key)
	if err != nil {
		return 0, false
	}
	val, err := strconv.Atoi(strings.TrimSpace(setting.ValueJSON))
	if err != nil {
		return 0, false
	}
	return val, true
}

func (h *Handler) getSettingBool(c *gin.Context, key string) (bool, bool) {
	if h.settings == nil {
		return false, false
	}
	setting, err := h.settings.GetSetting(c, key)
	if err != nil {
		return false, false
	}
	raw := strings.TrimSpace(setting.ValueJSON)
	if raw == "" {
		return false, false
	}
	switch strings.ToLower(raw) {
	case "true", "1", "yes":
		return true, true
	case "false", "0", "no":
		return false, true
	default:
		return false, false
	}
}

func (h *Handler) CMSBlocksPublic(c *gin.Context) {
	page := strings.TrimSpace(c.Query("page"))
	lang := strings.TrimSpace(c.Query("lang"))
	if lang == "" {
		lang = "zh-CN"
	}
	items, err := h.cmsSvc.ListBlocks(c, page, lang, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp := make([]CMSBlockDTO, 0, len(items))
	for _, item := range items {
		resp = append(resp, toCMSBlockDTO(item))
	}
	c.JSON(http.StatusOK, gin.H{"items": resp})
}

func (h *Handler) CMSPostsPublic(c *gin.Context) {
	lang := strings.TrimSpace(c.Query("lang"))
	if lang == "" {
		lang = "zh-CN"
	}
	categoryKey := strings.TrimSpace(c.Query("category_key"))
	limit, offset := paging(c)
	items, total, err := h.cmsSvc.ListPosts(c, usecase.CMSPostFilter{CategoryKey: categoryKey, Lang: lang, PublishedOnly: true, Limit: limit, Offset: offset})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp := make([]CMSPostDTO, 0, len(items))
	for _, item := range items {
		resp = append(resp, toCMSPostDTO(item))
	}
	c.JSON(http.StatusOK, gin.H{"items": resp, "total": total})
}

func (h *Handler) CMSPostDetailPublic(c *gin.Context) {
	slug := strings.TrimSpace(c.Param("slug"))
	post, err := h.cmsSvc.GetPostBySlug(c, slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if post.Status != "published" {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, toCMSPostDTO(post))
}

func (h *Handler) AdminCMSCategories(c *gin.Context) {
	lang := strings.TrimSpace(c.Query("lang"))
	items, err := h.cmsSvc.ListCategories(c, lang, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp := make([]CMSCategoryDTO, 0, len(items))
	for _, item := range items {
		resp = append(resp, toCMSCategoryDTO(item))
	}
	c.JSON(http.StatusOK, gin.H{"items": resp})
}

func (h *Handler) AdminCMSCategoryCreate(c *gin.Context) {
	var payload struct {
		Key       string `json:"key"`
		Name      string `json:"name"`
		Lang      string `json:"lang"`
		SortOrder int    `json:"sort_order"`
		Visible   *bool  `json:"visible"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	key := strings.TrimSpace(payload.Key)
	name := strings.TrimSpace(payload.Name)
	lang := strings.TrimSpace(payload.Lang)
	if lang == "" {
		lang = "zh-CN"
	}
	if key == "" || name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key and name required"})
		return
	}
	visible := true
	if payload.Visible != nil {
		visible = *payload.Visible
	}
	item := domain.CMSCategory{Key: key, Name: name, Lang: lang, SortOrder: payload.SortOrder, Visible: visible}
	if err := h.cmsSvc.CreateCategory(c, &item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toCMSCategoryDTO(item))
}

func (h *Handler) AdminCMSCategoryUpdate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	item, err := h.cmsSvc.GetCategory(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var payload struct {
		Key       *string `json:"key"`
		Name      *string `json:"name"`
		Lang      *string `json:"lang"`
		SortOrder *int    `json:"sort_order"`
		Visible   *bool   `json:"visible"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if payload.Key != nil {
		item.Key = strings.TrimSpace(*payload.Key)
	}
	if payload.Name != nil {
		item.Name = strings.TrimSpace(*payload.Name)
	}
	if payload.Lang != nil {
		item.Lang = strings.TrimSpace(*payload.Lang)
	}
	if payload.SortOrder != nil {
		item.SortOrder = *payload.SortOrder
	}
	if payload.Visible != nil {
		item.Visible = *payload.Visible
	}
	if item.Key == "" || item.Name == "" || item.Lang == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key, name and lang required"})
		return
	}
	if err := h.cmsSvc.UpdateCategory(c, item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toCMSCategoryDTO(item))
}

func (h *Handler) AdminCMSCategoryDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.cmsSvc.DeleteCategory(c, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminCMSPosts(c *gin.Context) {
	lang := strings.TrimSpace(c.Query("lang"))
	status := strings.TrimSpace(c.Query("status"))
	categoryIDRaw := strings.TrimSpace(c.Query("category_id"))
	limit, offset := paging(c)
	var categoryID *int64
	if categoryIDRaw != "" {
		if v, err := strconv.ParseInt(categoryIDRaw, 10, 64); err == nil {
			categoryID = &v
		}
	}
	items, total, err := h.cmsSvc.ListPosts(c, usecase.CMSPostFilter{CategoryID: categoryID, Status: status, Lang: lang, Limit: limit, Offset: offset})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp := make([]CMSPostDTO, 0, len(items))
	for _, item := range items {
		resp = append(resp, toCMSPostDTO(item))
	}
	c.JSON(http.StatusOK, gin.H{"items": resp, "total": total})
}

func (h *Handler) AdminCMSPostCreate(c *gin.Context) {
	var payload struct {
		CategoryID  int64  `json:"category_id"`
		Title       string `json:"title"`
		Slug        string `json:"slug"`
		Summary     string `json:"summary"`
		ContentHTML string `json:"content_html"`
		CoverURL    string `json:"cover_url"`
		Lang        string `json:"lang"`
		Status      string `json:"status"`
		Pinned      bool   `json:"pinned"`
		SortOrder   int    `json:"sort_order"`
		PublishedAt string `json:"published_at"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	lang := strings.TrimSpace(payload.Lang)
	if lang == "" {
		lang = "zh-CN"
	}
	status := strings.TrimSpace(payload.Status)
	if status == "" {
		status = "draft"
	}
	if payload.CategoryID == 0 || strings.TrimSpace(payload.Title) == "" || strings.TrimSpace(payload.Slug) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "category_id, title, slug required"})
		return
	}
	if containsDisallowedHTML(payload.ContentHTML) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content_html contains disallowed tags"})
		return
	}
	var publishedAt *time.Time
	if payload.PublishedAt != "" {
		if t, err := time.Parse(time.RFC3339, payload.PublishedAt); err == nil {
			publishedAt = &t
		}
	}
	if status == "published" && publishedAt == nil {
		now := time.Now()
		publishedAt = &now
	}
	post := domain.CMSPost{CategoryID: payload.CategoryID, Title: strings.TrimSpace(payload.Title), Slug: strings.TrimSpace(payload.Slug), Summary: payload.Summary, ContentHTML: payload.ContentHTML, CoverURL: payload.CoverURL, Lang: lang, Status: status, Pinned: payload.Pinned, SortOrder: payload.SortOrder, PublishedAt: publishedAt}
	if err := h.cmsSvc.CreatePost(c, &post); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toCMSPostDTO(post))
}

func (h *Handler) AdminCMSPostUpdate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	post, err := h.cmsSvc.GetPost(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var payload struct {
		CategoryID  *int64  `json:"category_id"`
		Title       *string `json:"title"`
		Slug        *string `json:"slug"`
		Summary     *string `json:"summary"`
		ContentHTML *string `json:"content_html"`
		CoverURL    *string `json:"cover_url"`
		Lang        *string `json:"lang"`
		Status      *string `json:"status"`
		Pinned      *bool   `json:"pinned"`
		SortOrder   *int    `json:"sort_order"`
		PublishedAt *string `json:"published_at"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if payload.CategoryID != nil {
		post.CategoryID = *payload.CategoryID
	}
	if payload.Title != nil {
		post.Title = strings.TrimSpace(*payload.Title)
	}
	if payload.Slug != nil {
		post.Slug = strings.TrimSpace(*payload.Slug)
	}
	if payload.Summary != nil {
		post.Summary = *payload.Summary
	}
	if payload.ContentHTML != nil {
		if containsDisallowedHTML(*payload.ContentHTML) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "content_html contains disallowed tags"})
			return
		}
		post.ContentHTML = *payload.ContentHTML
	}
	if payload.CoverURL != nil {
		post.CoverURL = *payload.CoverURL
	}
	if payload.Lang != nil {
		post.Lang = strings.TrimSpace(*payload.Lang)
	}
	if payload.Status != nil {
		post.Status = strings.TrimSpace(*payload.Status)
	}
	if payload.Pinned != nil {
		post.Pinned = *payload.Pinned
	}
	if payload.SortOrder != nil {
		post.SortOrder = *payload.SortOrder
	}
	if payload.PublishedAt != nil {
		if *payload.PublishedAt == "" {
			post.PublishedAt = nil
		} else if t, err := time.Parse(time.RFC3339, *payload.PublishedAt); err == nil {
			post.PublishedAt = &t
		}
	}
	if post.CategoryID == 0 || post.Title == "" || post.Slug == "" || post.Lang == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "category_id, title, slug, lang required"})
		return
	}
	if post.Status == "published" && post.PublishedAt == nil {
		now := time.Now()
		post.PublishedAt = &now
	}
	if err := h.cmsSvc.UpdatePost(c, post); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toCMSPostDTO(post))
}

func (h *Handler) AdminCMSPostDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.cmsSvc.DeletePost(c, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminCMSBlocks(c *gin.Context) {
	page := strings.TrimSpace(c.Query("page"))
	lang := strings.TrimSpace(c.Query("lang"))
	items, err := h.cmsSvc.ListBlocks(c, page, lang, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp := make([]CMSBlockDTO, 0, len(items))
	for _, item := range items {
		resp = append(resp, toCMSBlockDTO(item))
	}
	c.JSON(http.StatusOK, gin.H{"items": resp})
}

func (h *Handler) AdminCMSBlockCreate(c *gin.Context) {
	var payload struct {
		Page        string `json:"page"`
		Type        string `json:"type"`
		Title       string `json:"title"`
		Subtitle    string `json:"subtitle"`
		ContentJSON string `json:"content_json"`
		CustomHTML  string `json:"custom_html"`
		Lang        string `json:"lang"`
		Visible     *bool  `json:"visible"`
		SortOrder   int    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	page := strings.TrimSpace(payload.Page)
	typeName := strings.TrimSpace(payload.Type)
	lang := strings.TrimSpace(payload.Lang)
	if lang == "" {
		lang = "zh-CN"
	}
	if page == "" || typeName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "page and type required"})
		return
	}
	if err := validateCMSPageKey(page); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if payload.ContentJSON != "" && !json.Valid([]byte(payload.ContentJSON)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content_json invalid"})
		return
	}
	if typeName == "custom_html" && containsDisallowedHTML(payload.CustomHTML) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "custom_html contains disallowed tags"})
		return
	}
	visible := true
	if payload.Visible != nil {
		visible = *payload.Visible
	}
	block := domain.CMSBlock{Page: page, Type: typeName, Title: payload.Title, Subtitle: payload.Subtitle, ContentJSON: payload.ContentJSON, CustomHTML: payload.CustomHTML, Lang: lang, Visible: visible, SortOrder: payload.SortOrder}
	if err := h.cmsSvc.CreateBlock(c, &block); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toCMSBlockDTO(block))
}

func (h *Handler) AdminCMSBlockUpdate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	block, err := h.cmsSvc.GetBlock(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var payload struct {
		Page        *string `json:"page"`
		Type        *string `json:"type"`
		Title       *string `json:"title"`
		Subtitle    *string `json:"subtitle"`
		ContentJSON *string `json:"content_json"`
		CustomHTML  *string `json:"custom_html"`
		Lang        *string `json:"lang"`
		Visible     *bool   `json:"visible"`
		SortOrder   *int    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if payload.Page != nil {
		block.Page = strings.TrimSpace(*payload.Page)
	}
	if payload.Type != nil {
		block.Type = strings.TrimSpace(*payload.Type)
	}
	if payload.Title != nil {
		block.Title = *payload.Title
	}
	if payload.Subtitle != nil {
		block.Subtitle = *payload.Subtitle
	}
	if payload.ContentJSON != nil {
		if *payload.ContentJSON != "" && !json.Valid([]byte(*payload.ContentJSON)) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "content_json invalid"})
			return
		}
		block.ContentJSON = *payload.ContentJSON
	}
	if payload.CustomHTML != nil {
		if block.Type == "custom_html" && containsDisallowedHTML(*payload.CustomHTML) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "custom_html contains disallowed tags"})
			return
		}
		block.CustomHTML = *payload.CustomHTML
	}
	if payload.Lang != nil {
		block.Lang = strings.TrimSpace(*payload.Lang)
	}
	if payload.Visible != nil {
		block.Visible = *payload.Visible
	}
	if payload.SortOrder != nil {
		block.SortOrder = *payload.SortOrder
	}
	if block.Page == "" || block.Type == "" || block.Lang == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "page, type, lang required"})
		return
	}
	if err := validateCMSPageKey(block.Page); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.cmsSvc.UpdateBlock(c, block); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toCMSBlockDTO(block))
}

func (h *Handler) AdminCMSBlockDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.cmsSvc.DeleteBlock(c, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminUploadCreate(c *gin.Context) {
	if h.uploads == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file required"})
		return
	}
	const maxUploadSize = 20 << 20
	if file.Size > maxUploadSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file too large"})
		return
	}
	dateDir := time.Now().Format("20060102")
	if err := os.MkdirAll(filepath.Join("uploads", dateDir), 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "upload dir error"})
		return
	}
	name := buildUploadName(file.Filename)
	localPath := filepath.Join("uploads", dateDir, name)
	if err := c.SaveUploadedFile(file, localPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "save failed"})
		return
	}
	url := "/uploads/" + dateDir + "/" + name
	item := domain.Upload{Name: file.Filename, Path: localPath, URL: url, Mime: file.Header.Get("Content-Type"), Size: file.Size, UploaderID: getUserID(c)}
	if err := h.uploads.CreateUpload(c, &item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toUploadDTO(item))
}

func (h *Handler) AdminUploads(c *gin.Context) {
	limit, offset := paging(c)
	items, total, err := h.uploads.ListUploads(c, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp := make([]UploadDTO, 0, len(items))
	for _, item := range items {
		resp = append(resp, toUploadDTO(item))
	}
	c.JSON(http.StatusOK, gin.H{"items": resp, "total": total})
}

func containsDisallowedHTML(raw string) bool {
	lower := strings.ToLower(raw)
	return strings.Contains(lower, "<script") || strings.Contains(lower, "<iframe")
}

func validateCMSPageKey(page string) error {
	page = strings.TrimSpace(page)
	if page == "" {
		return errors.New("page required")
	}
	if strings.Contains(page, "..") || strings.ContainsAny(page, "/\\") {
		return errors.New("page invalid")
	}
	switch strings.ToLower(page) {
	case "api", "admin", "uploads", "assets", "static", "install":
		return errors.New("page reserved")
	default:
		return nil
	}
}

func buildUploadName(original string) string {
	ext := filepath.Ext(original)
	buf := make([]byte, 6)
	_, _ = rand.Read(buf)
	random := fmt.Sprintf("%x", buf)
	return time.Now().Format("150405") + "_" + random + ext
}

func (h *Handler) AdminPermissions(c *gin.Context) {
	perms, err := h.permissions.ListPermissions(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	tree := buildPermissionTree(perms)
	c.JSON(http.StatusOK, tree)
}

func (h *Handler) AdminPermissionsList(c *gin.Context) {
	perms, err := h.permissions.ListPermissions(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	items := make([]permissionItemDTO, 0, len(perms))
	for _, perm := range perms {
		items = append(items, toPermissionDTO(perm))
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) AdminPermissionDetail(c *gin.Context) {
	code := c.Param("code")
	perm, err := h.permissions.GetPermissionByCode(c, code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "permission not found"})
		return
	}
	c.JSON(http.StatusOK, toPermissionDTO(perm))
}

func (h *Handler) AdminPermissionsUpdate(c *gin.Context) {
	code := c.Param("code")
	var payload struct {
		Name         *string `json:"name"`
		FriendlyName *string `json:"friendly_name"`
		Category     *string `json:"category"`
		ParentCode   *string `json:"parent_code"`
		SortOrder    *int    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	perm, err := h.permissions.GetPermissionByCode(c, code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "permission not found"})
		return
	}
	if payload.Name != nil {
		perm.Name = strings.TrimSpace(*payload.Name)
	}
	if payload.FriendlyName != nil {
		perm.FriendlyName = strings.TrimSpace(*payload.FriendlyName)
	}
	if payload.Category != nil {
		perm.Category = strings.TrimSpace(*payload.Category)
	}
	if payload.ParentCode != nil {
		perm.ParentCode = strings.TrimSpace(*payload.ParentCode)
	}
	if payload.SortOrder != nil {
		perm.SortOrder = *payload.SortOrder
	}
	if perm.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}
	if perm.Category == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "category required"})
		return
	}
	if err := h.permissions.UpsertPermission(c, &perm); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toPermissionDTO(perm))
}

func (h *Handler) AdminPermissionsSync(c *gin.Context) {
	perms := permissions.GetDefinitions()
	if err := h.permissions.RegisterPermissions(c, perms); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "total": len(perms)})
}

type permissionItemDTO struct {
	Code         string               `json:"code"`
	Name         string               `json:"name"`
	FriendlyName string               `json:"friendly_name"`
	Category     string               `json:"category"`
	ParentCode   string               `json:"parent_code,omitempty"`
	SortOrder    int                  `json:"sort_order"`
	Children     []*permissionItemDTO `json:"children,omitempty"`
}

func toPermissionDTO(perm domain.Permission) permissionItemDTO {
	return permissionItemDTO{
		Code:         perm.Code,
		Name:         perm.Name,
		FriendlyName: perm.FriendlyName,
		Category:     perm.Category,
		ParentCode:   perm.ParentCode,
		SortOrder:    perm.SortOrder,
	}
}

func buildPermissionTree(perms []domain.Permission) []*permissionItemDTO {
	nodes := make(map[string]*permissionItemDTO, len(perms))
	for _, perm := range perms {
		item := toPermissionDTO(perm)
		nodes[perm.Code] = &item
	}

	roots := make([]*permissionItemDTO, 0)
	for _, perm := range perms {
		node := nodes[perm.Code]
		if perm.ParentCode != "" {
			parent, ok := nodes[perm.ParentCode]
			if ok {
				parent.Children = append(parent.Children, node)
				continue
			}
		}
		roots = append(roots, node)
	}

	sortPermissionNodes(roots)

	return roots
}

func sortPermissionNodes(nodes []*permissionItemDTO) {
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].SortOrder != nodes[j].SortOrder {
			return nodes[i].SortOrder < nodes[j].SortOrder
		}
		return nodes[i].Code < nodes[j].Code
	})
	for i := range nodes {
		if len(nodes[i].Children) == 0 {
			continue
		}
		sortPermissionNodes(nodes[i].Children)
	}
}

func renderCaptcha(code string) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 120, 40))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{240, 240, 240, 255}}, image.Point{}, draw.Src)
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.RGBA{30, 30, 30, 255}),
		Face: basicfont.Face7x13,
		Dot:  fixed.P(10, 25),
	}
	d.DrawString(code)
	return img
}

func parseHostIDLocal(v string) int64 {
	var id int64
	_, _ = fmt.Sscan(v, &id)
	return id
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func paging(c *gin.Context) (int, int) {
	limit := 20
	offset := 0
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil {
			limit = v
		}
	}
	if o := c.Query("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil {
			offset = v
		}
	}
	page := 0
	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			page = v
		}
	}
	if p := c.Query("pages"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			limit = v
		}
	}
	if p := c.Query("page_size"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			limit = v
		}
	}
	if page > 0 && limit > 0 {
		offset = (page - 1) * limit
	}
	return limit, offset
}

func listVisiblePlanGroups(catalog *usecase.CatalogService, ctx *gin.Context) []domain.PlanGroup {
	items, err := catalog.ListPlanGroups(ctx)
	if err != nil {
		return nil
	}
	return filterVisiblePlanGroups(items)
}

func filterVisiblePlanGroups(items []domain.PlanGroup) []domain.PlanGroup {
	if len(items) == 0 {
		return items
	}
	out := make([]domain.PlanGroup, 0, len(items))
	for _, item := range items {
		if item.Active && item.Visible {
			out = append(out, item)
		}
	}
	return out
}

func filterVisiblePackages(items []domain.Package, plans []domain.PlanGroup) []domain.Package {
	if len(items) == 0 {
		return items
	}
	planIndex := make(map[int64]struct{}, len(plans))
	for _, plan := range plans {
		planIndex[plan.ID] = struct{}{}
	}
	out := make([]domain.Package, 0, len(items))
	for _, item := range items {
		if !item.Active || !item.Visible {
			continue
		}
		if _, ok := planIndex[item.PlanGroupID]; !ok {
			continue
		}
		out = append(out, item)
	}
	return out
}

func filterEnabledSystemImages(items []domain.SystemImage, plans []domain.PlanGroup) []domain.SystemImage {
	if len(items) == 0 {
		return items
	}
	out := make([]domain.SystemImage, 0, len(items))
	for _, item := range items {
		if !item.Enabled {
			continue
		}
		out = append(out, item)
	}
	return out
}

func verifyHMAC(body []byte, secret string, signature string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	expected := fmt.Sprintf("%x", mac.Sum(nil))
	return hmac.Equal([]byte(strings.ToLower(signature)), []byte(strings.ToLower(expected)))
}

func getSettingValue(ctx *gin.Context, settings usecase.SettingsRepository, key string) string {
	if settings == nil {
		return ""
	}
	val, err := settings.GetSetting(ctx, key)
	if err != nil {
		return ""
	}
	return val.ValueJSON
}

func boolToString(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func isDigits(input string) bool {
	for _, r := range strings.TrimSpace(input) {
		if r < '0' || r > '9' {
			return false
		}
	}
	return input != ""
}

func parseAmountCents(value any) (int64, error) {
	switch v := value.(type) {
	case nil:
		return 0, money.ErrInvalidAmount
	case string:
		return money.ParseNumberStringToCents(v)
	case json.Number:
		return money.ParseNumberStringToCents(v.String())
	case float64:
		return floatToCents(v), nil
	case float32:
		return floatToCents(float64(v)), nil
	case int:
		return int64(v) * 100, nil
	case int64:
		return v * 100, nil
	default:
		return 0, money.ErrInvalidAmount
	}
}
