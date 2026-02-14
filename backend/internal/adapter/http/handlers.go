package http

import (
	"context"
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
	"github.com/microcosm-cc/bluemonday"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"xiaoheiplay/internal/adapter/email"
	plugins "xiaoheiplay/internal/adapter/plugins"
	"xiaoheiplay/internal/adapter/robot"
	"xiaoheiplay/internal/adapter/sse"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/pkg/money"
	"xiaoheiplay/internal/pkg/permissions"
	"xiaoheiplay/internal/usecase"
	pluginv1 "xiaoheiplay/plugin/v1"
)

var (
	htmlPolicy          = bluemonday.UGCPolicy()
	forgotPwdLimiter    = newRateLimiter()
	loginLimiter        = newRateLimiter()
	registerCodeLimiter = newRateLimiter()
)

func sanitizeHTML(raw string) string {
	if raw == "" {
		return ""
	}
	return htmlPolicy.Sanitize(raw)
}

type Handler struct {
	authSvc       *usecase.AuthService
	catalogSvc    *usecase.CatalogService
	goodsTypes    *usecase.GoodsTypeService
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
	pushSvc       *usecase.PushService
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
	pluginDir     string
	pluginPass    string
	pluginMgr     *plugins.Manager
	pluginPayMeth usecase.PluginPaymentMethodRepository
	taskSvc       *usecase.ScheduledTaskService
	probeSvc      *usecase.ProbeService
	probeHub      *usecase.ProbeHub
}

type authSettings struct {
	RegisterEnabled        bool
	RegisterRequiredFields []string
	PasswordMinLen         int
	PasswordRequireUpper   bool
	PasswordRequireLower   bool
	PasswordRequireNumber  bool
	PasswordRequireSymbol  bool
	RegisterVerifyType     string // none|email|sms
	RegisterVerifyTTL      time.Duration
	RegisterCaptchaEnabled bool
	RegisterEmailSubject   string
	RegisterEmailBody      string
	RegisterSMSPluginID    string
	RegisterSMSInstanceID  string
	RegisterSMSTemplateID  string
	LoginCaptchaEnabled    bool
	LoginRateLimitEnabled  bool
	LoginRateLimitWindow   time.Duration
	LoginRateLimitMax      int
}

func NewHandler(authSvc *usecase.AuthService, catalogSvc *usecase.CatalogService, goodsTypes *usecase.GoodsTypeService, cartSvc *usecase.CartService, orderSvc *usecase.OrderService, vpsSvc *usecase.VPSService, adminSvc *usecase.AdminService, adminVPS *usecase.AdminVPSService, integration *usecase.IntegrationService, reportSvc *usecase.ReportService, cmsSvc *usecase.CMSService, ticketSvc *usecase.TicketService, walletSvc *usecase.WalletService, walletOrder *usecase.WalletOrderService, paymentSvc *usecase.PaymentService, messageSvc *usecase.MessageCenterService, statusSvc *usecase.ServerStatusService, realnameSvc *usecase.RealNameService, orderItems usecase.OrderItemRepository, users usecase.UserRepository, orderRepo usecase.OrderRepository, vpsRepo usecase.VPSRepository, payments usecase.PaymentRepository, eventsRepo usecase.EventRepository, automationLogs usecase.AutomationLogRepository, settings usecase.SettingsRepository, permissions usecase.PermissionRepository, uploads usecase.UploadRepository, broker *sse.Broker, jwtSecret string, passwordReset *usecase.PasswordResetService, permissionSvc *usecase.PermissionService, taskSvc *usecase.ScheduledTaskService) *Handler {
	return &Handler{
		authSvc:       authSvc,
		catalogSvc:    catalogSvc,
		goodsTypes:    goodsTypes,
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
		passwordReset: passwordReset,
		permissionSvc: permissionSvc,
		taskSvc:       taskSvc,
	}
}

func (h *Handler) loadAuthSettings(ctx context.Context) authSettings {
	get := func(key string) string {
		if h.settings == nil {
			return ""
		}
		s, err := h.settings.GetSetting(ctx, key)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(s.ValueJSON)
	}
	getBool := func(key string, def bool) bool {
		val := strings.ToLower(strings.TrimSpace(get(key)))
		if val == "" {
			return def
		}
		return val == "true" || val == "1" || val == "yes"
	}
	getInt := func(key string, def int) int {
		val := strings.TrimSpace(get(key))
		if val == "" {
			return def
		}
		if n, err := strconv.Atoi(val); err == nil {
			return n
		}
		return def
	}
	getString := func(key, def string) string {
		val := strings.TrimSpace(get(key))
		if val == "" {
			return def
		}
		return val
	}
	getStringSlice := func(key string, def []string) []string {
		raw := get(key)
		if raw == "" {
			return def
		}
		var out []string
		if err := json.Unmarshal([]byte(raw), &out); err == nil {
			return out
		}
		return def
	}

	verifyType := strings.ToLower(getString("auth_register_verify_type", "none"))
	if verifyType != "email" && verifyType != "sms" {
		verifyType = "none"
	}

	return authSettings{
		RegisterEnabled:        getBool("auth_register_enabled", true),
		RegisterRequiredFields: getStringSlice("auth_register_required_fields", []string{"username", "email", "password"}),
		PasswordMinLen:         getInt("auth_password_min_len", 6),
		PasswordRequireUpper:   getBool("auth_password_require_upper", false),
		PasswordRequireLower:   getBool("auth_password_require_lower", false),
		PasswordRequireNumber:  getBool("auth_password_require_number", false),
		PasswordRequireSymbol:  getBool("auth_password_require_symbol", false),
		RegisterVerifyType:     verifyType,
		RegisterVerifyTTL:      time.Duration(getInt("auth_register_verify_ttl_sec", 600)) * time.Second,
		RegisterCaptchaEnabled: getBool("auth_register_captcha_enabled", true),
		RegisterEmailSubject:   getString("auth_register_email_subject", "Your verification code"),
		RegisterEmailBody:      getString("auth_register_email_body", "Your verification code is: {{code}}"),
		RegisterSMSPluginID:    getString("auth_register_sms_plugin_id", getString("sms_plugin_id", "")),
		RegisterSMSInstanceID:  getString("auth_register_sms_instance_id", getString("sms_instance_id", "default")),
		RegisterSMSTemplateID:  getString("auth_register_sms_template_id", getString("sms_provider_template_id", "")),
		LoginCaptchaEnabled:    getBool("auth_login_captcha_enabled", false),
		LoginRateLimitEnabled:  getBool("auth_login_rate_limit_enabled", true),
		LoginRateLimitWindow:   time.Duration(getInt("auth_login_rate_limit_window_sec", 300)) * time.Second,
		LoginRateLimitMax:      getInt("auth_login_rate_limit_max_attempts", 5),
	}
}

func validatePasswordBySettings(password string, s authSettings) error {
	if strings.TrimSpace(password) == "" {
		return usecase.ErrInvalidInput
	}
	if s.PasswordMinLen > 0 && len(password) < s.PasswordMinLen {
		return errors.New("password too short")
	}
	var hasUpper, hasLower, hasNumber, hasSymbol bool
	for _, r := range password {
		switch {
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= 'a' && r <= 'z':
			hasLower = true
		case r >= '0' && r <= '9':
			hasNumber = true
		default:
			hasSymbol = true
		}
	}
	if s.PasswordRequireUpper && !hasUpper {
		return errors.New("password requires uppercase")
	}
	if s.PasswordRequireLower && !hasLower {
		return errors.New("password requires lowercase")
	}
	if s.PasswordRequireNumber && !hasNumber {
		return errors.New("password requires number")
	}
	if s.PasswordRequireSymbol && !hasSymbol {
		return errors.New("password requires symbol")
	}
	return nil
}

func (h *Handler) SetPaymentPluginConfig(dir, password string) {
	h.pluginDir = strings.TrimSpace(dir)
	h.pluginPass = strings.TrimSpace(password)
}

func (h *Handler) SetPushService(pushSvc *usecase.PushService) {
	h.pushSvc = pushSvc
}

func (h *Handler) SetPluginManager(mgr *plugins.Manager) {
	h.pluginMgr = mgr
}

func (h *Handler) SetPluginPaymentMethodRepo(repo usecase.PluginPaymentMethodRepository) {
	h.pluginPayMeth = repo
}

func (h *Handler) SetProbeService(svc *usecase.ProbeService, hub *usecase.ProbeHub) {
	h.probeSvc = svc
	h.probeHub = hub
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
		VerifyCode  string `json:"verify_code"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	settings := h.loadAuthSettings(c)
	if !settings.RegisterEnabled {
		c.JSON(http.StatusForbidden, gin.H{"error": "registration disabled"})
		return
	}
	required := map[string]bool{
		"username": true,
		"email":    true,
		"password": true,
	}
	for _, f := range settings.RegisterRequiredFields {
		required[strings.ToLower(strings.TrimSpace(f))] = true
	}
	if required["username"] && strings.TrimSpace(payload.Username) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username required"})
		return
	}
	if required["email"] && strings.TrimSpace(payload.Email) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email required"})
		return
	}
	if required["phone"] && strings.TrimSpace(payload.Phone) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone required"})
		return
	}
	if required["qq"] && strings.TrimSpace(payload.QQ) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "qq required"})
		return
	}
	if err := validatePasswordBySettings(payload.Password, settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if settings.RegisterVerifyType != "none" {
		code := strings.TrimSpace(payload.VerifyCode)
		if code == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "verification code required"})
			return
		}
		switch settings.RegisterVerifyType {
		case "email":
			if strings.TrimSpace(payload.Email) == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "email required"})
				return
			}
			if err := h.authSvc.VerifyVerificationCode(c, "email", strings.TrimSpace(payload.Email), "register", code); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid verification code"})
				return
			}
		case "sms":
			if strings.TrimSpace(payload.Phone) == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "phone required"})
				return
			}
			if err := h.authSvc.VerifyVerificationCode(c, "sms", strings.TrimSpace(payload.Phone), "register", code); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid verification code"})
				return
			}
		}
	}
	user, err := h.authSvc.Register(c, usecase.RegisterInput{
		Username:        payload.Username,
		Email:           payload.Email,
		QQ:              payload.QQ,
		Phone:           payload.Phone,
		Password:        payload.Password,
		CaptchaID:       payload.CaptchaID,
		CaptchaCode:     payload.CaptchaCode,
		CaptchaRequired: settings.RegisterCaptchaEnabled,
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
		Username    string `json:"username"`
		Password    string `json:"password"`
		CaptchaID   string `json:"captcha_id"`
		CaptchaCode string `json:"captcha_code"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	settings := h.loadAuthSettings(c)
	if settings.LoginRateLimitEnabled {
		key := "login:" + strings.ToLower(strings.TrimSpace(payload.Username)) + ":" + strings.TrimSpace(c.ClientIP())
		if !loginLimiter.Allow(key, settings.LoginRateLimitMax, settings.LoginRateLimitWindow) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many attempts"})
			return
		}
	}
	if settings.LoginCaptchaEnabled {
		if err := h.authSvc.VerifyCaptcha(c, payload.CaptchaID, payload.CaptchaCode); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "captcha failed"})
			return
		}
	}
	user, err := h.authSvc.Login(c, payload.Username, payload.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	accessToken, err := h.signAuthToken(user.ID, string(user.Role), 24*time.Hour, "access")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sign token failed"})
		return
	}
	refreshToken, err := h.signAuthToken(user.ID, string(user.Role), 7*24*time.Hour, "refresh")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sign token failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    86400,
		"user":          gin.H{"id": user.ID, "username": user.Username, "role": user.Role},
	})
}

func (h *Handler) Logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AuthSettings(c *gin.Context) {
	settings := h.loadAuthSettings(c)
	c.JSON(http.StatusOK, gin.H{
		"register_enabled":         settings.RegisterEnabled,
		"register_required_fields": settings.RegisterRequiredFields,
		"password_min_len":         settings.PasswordMinLen,
		"password_require_upper":   settings.PasswordRequireUpper,
		"password_require_lower":   settings.PasswordRequireLower,
		"password_require_number":  settings.PasswordRequireNumber,
		"password_require_symbol":  settings.PasswordRequireSymbol,
		"register_verify_type":     settings.RegisterVerifyType,
		"register_captcha_enabled": settings.RegisterCaptchaEnabled,
		"login_captcha_enabled":    settings.LoginCaptchaEnabled,
	})
}

func (h *Handler) RegisterCode(c *gin.Context) {
	var payload struct {
		Email       string `json:"email"`
		Phone       string `json:"phone"`
		CaptchaID   string `json:"captcha_id"`
		CaptchaCode string `json:"captcha_code"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	settings := h.loadAuthSettings(c)
	if !settings.RegisterEnabled {
		c.JSON(http.StatusForbidden, gin.H{"error": "registration disabled"})
		return
	}
	if settings.RegisterCaptchaEnabled {
		if err := h.authSvc.VerifyCaptcha(c, payload.CaptchaID, payload.CaptchaCode); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "captcha failed"})
			return
		}
	}

	switch settings.RegisterVerifyType {
	case "email":
		emailVal := strings.TrimSpace(payload.Email)
		if emailVal == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email required"})
			return
		}
		if !registerCodeLimiter.Allow("register_code:email:"+strings.ToLower(emailVal), 3, 10*time.Minute) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
			return
		}
		code, err := h.authSvc.CreateVerificationCode(c, "email", emailVal, "register", settings.RegisterVerifyTTL)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		subject := strings.TrimSpace(settings.RegisterEmailSubject)
		if subject == "" {
			subject = "Your verification code"
		}
		body := strings.ReplaceAll(settings.RegisterEmailBody, "{{code}}", code)
		sender := email.NewSender(h.settings)
		if err := sender.Send(c, emailVal, subject, body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email send failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	case "sms":
		phoneVal := strings.TrimSpace(payload.Phone)
		if phoneVal == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "phone required"})
			return
		}
		if settings.RegisterSMSPluginID == "" || h.pluginMgr == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "sms plugin not configured"})
			return
		}
		if !registerCodeLimiter.Allow("register_code:sms:"+phoneVal, 3, 10*time.Minute) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
			return
		}
		code, err := h.authSvc.CreateVerificationCode(c, "sms", phoneVal, "register", settings.RegisterVerifyTTL)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if _, err := h.pluginMgr.EnsureRunning(c, "sms", settings.RegisterSMSPluginID, settings.RegisterSMSInstanceID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		client, ok := h.pluginMgr.GetSMSClient("sms", settings.RegisterSMSPluginID, settings.RegisterSMSInstanceID)
		if !ok || client == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "sms plugin not running"})
			return
		}
		req := &pluginv1.SendSmsRequest{
			TemplateId: settings.RegisterSMSTemplateID,
			Vars: map[string]string{
				"code": code,
			},
			Phones: []string{phoneVal},
		}
		if settings.RegisterSMSTemplateID == "" {
			content := "Your verification code is: " + code
			if v := strings.TrimSpace(getSettingValue(c, h.settings, "sms_default_template_id")); v != "" {
				if tid, err := strconv.ParseInt(v, 10, 64); err == nil && tid > 0 {
					if rendered, ok := h.renderSMSTemplateByID(c, tid, map[string]any{
						"code":  code,
						"phone": phoneVal,
						"now":   time.Now().Format(time.RFC3339),
					}); ok {
						content = rendered
					}
				}
			}
			req.Content = content
		}
		cctx, cancel := context.WithTimeout(c, 10*time.Second)
		defer cancel()
		resp, err := client.Send(cctx, req)
		if err != nil || resp == nil || !resp.Ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "sms send failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "verification not enabled"})
		return
	}
}

func (h *Handler) Refresh(c *gin.Context) {
	var payload struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	claims, err := h.parseRefreshToken(payload.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}
	userID, ok := parseMapInt64(claims["user_id"])
	if !ok || userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}
	role, _ := claims["role"].(string)
	if role == "" {
		role = "user"
	}
	if h.users != nil {
		if _, err := h.users.GetUserByID(c, userID); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
			return
		}
	}
	accessToken, err := h.signAuthToken(userID, role, 24*time.Hour, "access")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sign token failed"})
		return
	}
	newRefreshToken, err := h.signAuthToken(userID, role, 7*24*time.Hour, "refresh")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sign token failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": newRefreshToken,
		"expires_in":    86400,
	})
}

func (h *Handler) signAuthToken(userID int64, role string, ttl time.Duration, tokenType string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"type":    tokenType,
		"iat":     time.Now().Unix(),
		"exp":     time.Now().Add(ttl).Unix(),
	})
	return token.SignedString(h.jwtSecret)
}

func (h *Handler) parseRefreshToken(raw string) (jwt.MapClaims, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New("empty refresh token")
	}
	claims := jwt.MapClaims{}
	parsed, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return h.jwtSecret, nil
	})
	if err != nil || parsed == nil || !parsed.Valid {
		return nil, errors.New("invalid refresh token")
	}
	tokenType, _ := claims["type"].(string)
	// Backward compatible: allow legacy tokens without type.
	if tokenType != "" && tokenType != "refresh" {
		return nil, errors.New("invalid token type")
	}
	return claims, nil
}

func parseMapInt64(v any) (int64, bool) {
	switch t := v.(type) {
	case int64:
		return t, true
	case int:
		return int64(t), true
	case float64:
		return int64(t), true
	case json.Number:
		n, err := t.Int64()
		return n, err == nil
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		return n, err == nil
	default:
		return 0, false
	}
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
		Phone    string `json:"phone"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	phone := strings.TrimSpace(payload.Phone)
	if h.users != nil {
		if user, err := h.users.GetUserByID(c, getUserID(c)); err == nil {
			if strings.TrimSpace(user.Phone) != "" {
				phone = strings.TrimSpace(user.Phone)
			}
		}
	}
	record, err := h.realnameSvc.VerifyWithInput(c, getUserID(c), usecase.RealNameVerifyInput{
		RealName: payload.RealName,
		IDNumber: payload.IDNumber,
		Phone:    phone,
	})
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
		Method    string            `json:"method"`
		ReturnURL string            `json:"return_url"`
		NotifyURL string            `json:"notify_url"`
		Extra     map[string]string `json:"extra"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
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
	result, err := h.paymentSvc.SelectPayment(c, getUserID(c), id, usecase.PaymentSelectInput{
		Method:    payload.Method,
		ReturnURL: payload.ReturnURL,
		NotifyURL: payload.NotifyURL,
		Extra:     payload.Extra,
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
	result, err := h.paymentSvc.HandleNotify(c, provider, usecase.RawHTTPRequest{
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
	if templateID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}
	var matchedSystemID int64
	// Validate template against instance line to prevent cross-line image reinstall.
	lineID := inst.LineID
	if lineID <= 0 && inst.PackageID > 0 {
		if pkg, pkgErr := h.catalogSvc.GetPackage(c, inst.PackageID); pkgErr == nil && pkg.PlanGroupID > 0 {
			if plan, planErr := h.catalogSvc.GetPlanGroup(c, pkg.PlanGroupID); planErr == nil {
				lineID = plan.LineID
			}
		}
	}
	if lineID > 0 {
		allowedImages, listErr := h.catalogSvc.ListSystemImages(c, lineID)
		if listErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "list error"})
			return
		}
		allowed := false
		for _, img := range allowedImages {
			if !img.Enabled {
				continue
			}
			if img.ImageID == templateID || img.ID == templateID {
				allowed = true
				matchedSystemID = img.ID
				break
			}
		}
		if !allowed {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
			return
		}
	}
	if matchedSystemID == 0 {
		if img, imgErr := h.catalogSvc.GetSystemImage(c, templateID); imgErr == nil && img.ID > 0 {
			matchedSystemID = img.ID
		}
	}
	if err := h.vpsSvc.ResetOS(c, inst, templateID, strings.TrimSpace(password)); err != nil {
		if err == usecase.ErrInvalidInput {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if matchedSystemID > 0 && h.vpsRepo != nil {
		if latest, getErr := h.vpsRepo.GetInstance(c, inst.ID); getErr == nil {
			latest.SystemID = matchedSystemID
			_ = h.vpsRepo.UpdateInstanceLocal(c, latest)
		}
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
			status := http.StatusBadRequest
			if errors.Is(err, usecase.ErrNotSupported) {
				status = http.StatusNotImplemented
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": items})
	case http.MethodPost:
		if err := h.vpsSvc.CreateSnapshot(c, inst); err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, usecase.ErrNotSupported) {
				status = http.StatusNotImplemented
			}
			c.JSON(status, gin.H{"error": err.Error()})
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
		status := http.StatusBadRequest
		if errors.Is(err, usecase.ErrNotSupported) {
			status = http.StatusNotImplemented
		}
		c.JSON(status, gin.H{"error": err.Error()})
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
		status := http.StatusBadRequest
		if errors.Is(err, usecase.ErrNotSupported) {
			status = http.StatusNotImplemented
		}
		c.JSON(status, gin.H{"error": err.Error()})
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
			status := http.StatusBadRequest
			if errors.Is(err, usecase.ErrNotSupported) {
				status = http.StatusNotImplemented
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": items})
	case http.MethodPost:
		if err := h.vpsSvc.CreateBackup(c, inst); err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, usecase.ErrNotSupported) {
				status = http.StatusNotImplemented
			}
			c.JSON(status, gin.H{"error": err.Error()})
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
		status := http.StatusBadRequest
		if errors.Is(err, usecase.ErrNotSupported) {
			status = http.StatusNotImplemented
		}
		c.JSON(status, gin.H{"error": err.Error()})
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
		status := http.StatusBadRequest
		if errors.Is(err, usecase.ErrNotSupported) {
			status = http.StatusNotImplemented
		}
		c.JSON(status, gin.H{"error": err.Error()})
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
			status := http.StatusBadRequest
			if errors.Is(err, usecase.ErrNotSupported) {
				status = http.StatusNotImplemented
			}
			c.JSON(status, gin.H{"error": err.Error()})
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
			Priority  *int   `json:"priority"`
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
		if payload.Priority != nil {
			req.Priority = *payload.Priority
		}
		if req.Direction == "" || req.Protocol == "" || req.Method == "" || req.Port == "" || req.IP == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
			return
		}
		if err := h.vpsSvc.AddFirewallRule(c, inst, req); err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, usecase.ErrNotSupported) {
				status = http.StatusNotImplemented
			}
			c.JSON(status, gin.H{"error": err.Error()})
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
		status := http.StatusBadRequest
		if errors.Is(err, usecase.ErrNotSupported) {
			status = http.StatusNotImplemented
		}
		c.JSON(status, gin.H{"error": err.Error()})
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
			status := http.StatusBadRequest
			if errors.Is(err, usecase.ErrNotSupported) {
				status = http.StatusNotImplemented
			}
			c.JSON(status, gin.H{"error": err.Error()})
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
			status := http.StatusBadRequest
			if errors.Is(err, usecase.ErrNotSupported) {
				status = http.StatusNotImplemented
			}
			c.JSON(status, gin.H{"error": err.Error()})
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
		status := http.StatusBadRequest
		if errors.Is(err, usecase.ErrNotSupported) {
			status = http.StatusNotImplemented
		}
		c.JSON(status, gin.H{"error": err.Error()})
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
		status := http.StatusBadRequest
		if errors.Is(err, usecase.ErrNotSupported) {
			status = http.StatusNotImplemented
		}
		c.JSON(status, gin.H{"error": err.Error()})
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
		} else if err == usecase.ErrConflict {
			status = http.StatusConflict
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
	accessToken, err := h.signAuthToken(user.ID, string(user.Role), 24*time.Hour, "access")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sign token failed"})
		return
	}
	refreshToken, err := h.signAuthToken(user.ID, string(user.Role), 7*24*time.Hour, "refresh")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sign token failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    86400,
	})
}

func (h *Handler) AdminRefresh(c *gin.Context) {
	var payload struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	claims, err := h.parseRefreshToken(payload.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}
	userID, ok := parseMapInt64(claims["user_id"])
	if !ok || userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}
	role, _ := claims["role"].(string)
	if role != string(domain.UserRoleAdmin) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}
	if h.users != nil {
		user, err := h.users.GetUserByID(c, userID)
		if err != nil || user.Role != domain.UserRoleAdmin {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
			return
		}
	}
	accessToken, err := h.signAuthToken(userID, role, 24*time.Hour, "access")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sign token failed"})
		return
	}
	newRefreshToken, err := h.signAuthToken(userID, role, 7*24*time.Hour, "refresh")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sign token failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": newRefreshToken,
		"expires_in":    86400,
	})
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
	signed, err := token.SignedString(h.jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sign token failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"access_token": signed, "expires_in": 86400, "user": gin.H{"id": user.ID, "username": user.Username, "role": user.Role}})
}

func (h *Handler) AdminQQAvatar(c *gin.Context) {
	qq := strings.TrimSpace(c.Param("qq"))
	if qq == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid qq"})
		return
	}
	for _, r := range qq {
		if r < '0' || r > '9' {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid qq"})
			return
		}
	}
	url := "https://q1.qlogo.cn/g?b=qq&nk=" + qq + "&s=100"
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "request failed"})
		return
	}
	client := &http.Client{Timeout: 6 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fetch failed"})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "avatar not found"})
		return
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}
	c.Header("Cache-Control", "public, max-age=86400")
	c.DataFromReader(http.StatusOK, resp.ContentLength, contentType, resp.Body, nil)
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
	includeDisabled := strings.EqualFold(strings.TrimSpace(c.Query("include_disabled")), "true")
	includeLegacy := strings.EqualFold(strings.TrimSpace(c.Query("include_legacy")), "true")
	items, err := h.paymentSvc.ListProviders(c, includeDisabled)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !includeLegacy {
		filtered := make([]usecase.PaymentProviderInfo, 0, len(items))
		for _, item := range items {
			k := strings.ToLower(strings.TrimSpace(item.Key))
			if k == "yipay" || k == "custom" {
				continue
			}
			filtered = append(filtered, item)
		}
		items = filtered
	}
	c.JSON(http.StatusOK, gin.H{"items": toPaymentProviderDTOs(items)})
}

func (h *Handler) AdminPaymentProviderUpdate(c *gin.Context) {
	if h.paymentSvc == nil && h.pluginPayMeth == nil {
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
	trimmedKey := strings.TrimSpace(key)
	if strings.Contains(trimmedKey, ".") {
		if h.pluginPayMeth == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "payment method repo missing"})
			return
		}
		parts := strings.Split(trimmedKey, ".")
		pluginID := ""
		instanceID := plugins.DefaultInstanceID
		method := ""
		switch len(parts) {
		case 2:
			pluginID = strings.TrimSpace(parts[0])
			method = strings.TrimSpace(parts[1])
		default:
			pluginID = strings.TrimSpace(parts[0])
			instanceID = strings.TrimSpace(parts[1])
			method = strings.TrimSpace(strings.Join(parts[2:], "."))
		}
		if pluginID == "" || instanceID == "" || method == "" || payload.Enabled == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plugin payment key or payload"})
			return
		}
		if err := h.pluginPayMeth.UpsertPluginPaymentMethod(c, &domain.PluginPaymentMethod{
			Category:   "payment",
			PluginID:   pluginID,
			InstanceID: instanceID,
			Method:     method,
			Enabled:    enabled,
		}); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if h.adminSvc != nil {
			h.adminSvc.Audit(c, getUserID(c), "plugin.payment_method.update", "plugin", "payment/"+pluginID+"/"+instanceID, map[string]any{
				"method":  method,
				"enabled": enabled,
				"via":     "payments.providers.update",
			})
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}
	if h.paymentSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payment disabled"})
		return
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

func (h *Handler) AdminPluginsList(c *gin.Context) {
	if h.pluginMgr == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plugins disabled"})
		return
	}
	items, err := h.pluginMgr.List(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) AdminPluginsDiscover(c *gin.Context) {
	if h.pluginMgr == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plugins disabled"})
		return
	}
	items, err := h.pluginMgr.DiscoverOnDisk(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) AdminPluginPaymentMethodsList(c *gin.Context) {
	if h.pluginMgr == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plugins disabled"})
		return
	}
	if h.pluginPayMeth == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payment method repo missing"})
		return
	}
	category := strings.TrimSpace(c.Query("category"))
	pluginID := strings.TrimSpace(c.Query("plugin_id"))
	instanceID := strings.TrimSpace(c.Query("instance_id"))
	if category == "" {
		category = "payment"
	}
	if instanceID == "" {
		instanceID = plugins.DefaultInstanceID
	}
	if category == "" || pluginID == "" || instanceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "category/plugin_id/instance_id required"})
		return
	}

	items, err := h.pluginMgr.List(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var supported []string
	for _, it := range items {
		if it.Category != category || it.PluginID != pluginID || it.InstanceID != instanceID {
			continue
		}
		if it.Capabilities.Capabilities.Payment == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "not a payment plugin instance"})
			return
		}
		supported = it.Capabilities.Capabilities.Payment.Methods
		break
	}
	if len(supported) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "plugin instance not found"})
		return
	}

	overrides, _ := h.pluginPayMeth.ListPluginPaymentMethods(c, category, pluginID, instanceID)
	enabledMap := map[string]bool{}
	for _, ov := range overrides {
		enabledMap[ov.Method] = ov.Enabled
	}

	type itemDTO struct {
		Method  string `json:"method"`
		Enabled bool   `json:"enabled"`
	}
	out := make([]itemDTO, 0, len(supported))
	for _, m := range supported {
		m = strings.TrimSpace(m)
		if m == "" {
			continue
		}
		enabled, ok := enabledMap[m]
		if !ok {
			enabled = true
		}
		out = append(out, itemDTO{Method: m, Enabled: enabled})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Method < out[j].Method })
	c.JSON(http.StatusOK, gin.H{"items": out})
}

func (h *Handler) AdminPluginPaymentMethodsUpdate(c *gin.Context) {
	if h.pluginMgr == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plugins disabled"})
		return
	}
	if h.pluginPayMeth == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payment method repo missing"})
		return
	}
	var payload struct {
		Category   string `json:"category"`
		PluginID   string `json:"plugin_id"`
		InstanceID string `json:"instance_id"`
		Method     string `json:"method"`
		Enabled    *bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	category := strings.TrimSpace(payload.Category)
	pluginID := strings.TrimSpace(payload.PluginID)
	instanceID := strings.TrimSpace(payload.InstanceID)
	method := strings.TrimSpace(payload.Method)
	if category == "" {
		category = "payment"
	}
	if instanceID == "" {
		instanceID = plugins.DefaultInstanceID
	}
	if category == "" || pluginID == "" || instanceID == "" || method == "" || payload.Enabled == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "category/plugin_id/instance_id/method/enabled required"})
		return
	}

	enabled := *payload.Enabled
	if err := h.pluginPayMeth.UpsertPluginPaymentMethod(c, &domain.PluginPaymentMethod{
		Category:   category,
		PluginID:   pluginID,
		InstanceID: instanceID,
		Method:     method,
		Enabled:    enabled,
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c, getUserID(c), "plugin.payment_method.update", "plugin", category+"/"+pluginID+"/"+instanceID, map[string]any{
			"method":  method,
			"enabled": enabled,
		})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPluginInstall(c *gin.Context) {
	if h.pluginMgr == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plugins disabled"})
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing file"})
		return
	}
	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "open file failed"})
		return
	}
	defer f.Close()

	inst, err := h.pluginMgr.Install(c, file.Filename, f)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Security gate: official signature => allow. Otherwise require admin password confirmation.
	if inst.SignatureStatus != domain.PluginSignatureOfficial {
		adminPassword := strings.TrimSpace(c.PostForm("admin_password"))
		if adminPassword == "" {
			_ = h.pluginMgr.Uninstall(c, inst.Category, inst.PluginID)
			c.JSON(http.StatusForbidden, gin.H{"error": "admin_password required for untrusted plugin"})
			return
		}
		if h.authSvc == nil {
			_ = h.pluginMgr.Uninstall(c, inst.Category, inst.PluginID)
			c.JSON(http.StatusBadRequest, gin.H{"error": "auth disabled"})
			return
		}
		if err := h.authSvc.VerifyPassword(c, getUserID(c), adminPassword); err != nil {
			_ = h.pluginMgr.Uninstall(c, inst.Category, inst.PluginID)
			c.JSON(http.StatusForbidden, gin.H{"error": "invalid admin password"})
			return
		}
	}

	if h.adminSvc != nil {
		h.adminSvc.Audit(c, getUserID(c), "plugin.install", "plugin", inst.Category+"/"+inst.PluginID, map[string]any{
			"signature_status": inst.SignatureStatus,
		})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "plugin": inst})
}

func (h *Handler) AdminPluginImportFromDisk(c *gin.Context) {
	if h.pluginMgr == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plugins disabled"})
		return
	}
	category := c.Param("category")
	pluginID := c.Param("plugin_id")

	var payload struct {
		AdminPassword string `json:"admin_password"`
	}
	_ = c.ShouldBindJSON(&payload)

	// Peek signature status to decide security gate BEFORE writing DB.
	targetSig, err := h.pluginMgr.SignatureStatusOnDisk(category, pluginID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if targetSig != domain.PluginSignatureOfficial {
		adminPassword := strings.TrimSpace(payload.AdminPassword)
		if adminPassword == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "admin_password required for untrusted plugin"})
			return
		}
		if h.authSvc == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "auth disabled"})
			return
		}
		if err := h.authSvc.VerifyPassword(c, getUserID(c), adminPassword); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "invalid admin password"})
			return
		}
	}

	inst, err := h.pluginMgr.ImportFromDisk(c, category, pluginID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c, getUserID(c), "plugin.import", "plugin", inst.Category+"/"+inst.PluginID, map[string]any{
			"signature_status": inst.SignatureStatus,
		})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "plugin": inst})
}

func (h *Handler) AdminPluginEnable(c *gin.Context) {
	if h.pluginMgr == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plugins disabled"})
		return
	}
	category := c.Param("category")
	pluginID := c.Param("plugin_id")
	if err := h.pluginMgr.EnableInstance(c, category, pluginID, plugins.DefaultInstanceID); err != nil {
		writePluginHandlerError(c, err)
		return
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c, getUserID(c), "plugin.enable", "plugin", category+"/"+pluginID+"/"+plugins.DefaultInstanceID, map[string]any{})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPluginDisable(c *gin.Context) {
	if h.pluginMgr == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plugins disabled"})
		return
	}
	category := c.Param("category")
	pluginID := c.Param("plugin_id")
	if err := h.pluginMgr.DisableInstance(c, category, pluginID, plugins.DefaultInstanceID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c, getUserID(c), "plugin.disable", "plugin", category+"/"+pluginID+"/"+plugins.DefaultInstanceID, map[string]any{})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPluginUninstall(c *gin.Context) {
	if h.pluginMgr == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plugins disabled"})
		return
	}
	category := c.Param("category")
	pluginID := c.Param("plugin_id")
	if err := h.pluginMgr.DeleteInstance(c, category, pluginID, plugins.DefaultInstanceID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c, getUserID(c), "plugin.uninstall", "plugin", category+"/"+pluginID+"/"+plugins.DefaultInstanceID, map[string]any{})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPluginConfigSchema(c *gin.Context) {
	if h.pluginMgr == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plugins disabled"})
		return
	}
	category := c.Param("category")
	pluginID := c.Param("plugin_id")
	jsonSchema, uiSchema, err := h.pluginMgr.GetConfigSchemaInstance(c, category, pluginID, plugins.DefaultInstanceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"json_schema": jsonSchema, "ui_schema": uiSchema})
}

func (h *Handler) AdminPluginConfigGet(c *gin.Context) {
	if h.pluginMgr == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plugins disabled"})
		return
	}
	category := c.Param("category")
	pluginID := c.Param("plugin_id")
	cfg, err := h.pluginMgr.GetConfigInstance(c, category, pluginID, plugins.DefaultInstanceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"config_json": cfg})
}

func (h *Handler) AdminPluginConfigUpdate(c *gin.Context) {
	if h.pluginMgr == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plugins disabled"})
		return
	}
	category := c.Param("category")
	pluginID := c.Param("plugin_id")
	var payload struct {
		ConfigJSON string `json:"config_json"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if err := h.pluginMgr.UpdateConfigInstance(c, category, pluginID, plugins.DefaultInstanceID, payload.ConfigJSON); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c, getUserID(c), "plugin.config_update", "plugin", category+"/"+pluginID+"/"+plugins.DefaultInstanceID, map[string]any{})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPluginInstanceCreate(c *gin.Context) {
	if h.pluginMgr == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plugins disabled"})
		return
	}
	category := c.Param("category")
	pluginID := c.Param("plugin_id")
	var payload struct {
		InstanceID string `json:"instance_id"`
		ConfigJSON string `json:"config_json"`
	}
	_ = c.ShouldBindJSON(&payload)

	inst, err := h.pluginMgr.CreateInstance(c, category, pluginID, payload.InstanceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(payload.ConfigJSON) != "" {
		if err := h.pluginMgr.UpdateConfigInstance(c, category, pluginID, inst.InstanceID, payload.ConfigJSON); err != nil {
			_ = h.pluginMgr.DeleteInstance(c, category, pluginID, inst.InstanceID)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c, getUserID(c), "plugin.instance_create", "plugin", category+"/"+pluginID+"/"+inst.InstanceID, map[string]any{})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "plugin": inst})
}

func (h *Handler) AdminPluginInstanceEnable(c *gin.Context) {
	if h.pluginMgr == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plugins disabled"})
		return
	}
	category := c.Param("category")
	pluginID := c.Param("plugin_id")
	instanceID := c.Param("instance_id")
	if err := h.pluginMgr.EnableInstance(c, category, pluginID, instanceID); err != nil {
		writePluginHandlerError(c, err)
		return
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c, getUserID(c), "plugin.enable", "plugin", category+"/"+pluginID+"/"+instanceID, map[string]any{})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPluginInstanceDisable(c *gin.Context) {
	if h.pluginMgr == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plugins disabled"})
		return
	}
	category := c.Param("category")
	pluginID := c.Param("plugin_id")
	instanceID := c.Param("instance_id")
	if err := h.pluginMgr.DisableInstance(c, category, pluginID, instanceID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c, getUserID(c), "plugin.disable", "plugin", category+"/"+pluginID+"/"+instanceID, map[string]any{})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPluginInstanceDelete(c *gin.Context) {
	if h.pluginMgr == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plugins disabled"})
		return
	}
	category := c.Param("category")
	pluginID := c.Param("plugin_id")
	instanceID := c.Param("instance_id")
	if err := h.pluginMgr.DeleteInstance(c, category, pluginID, instanceID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c, getUserID(c), "plugin.instance_delete", "plugin", category+"/"+pluginID+"/"+instanceID, map[string]any{})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPluginInstanceConfigSchema(c *gin.Context) {
	if h.pluginMgr == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plugins disabled"})
		return
	}
	category := c.Param("category")
	pluginID := c.Param("plugin_id")
	instanceID := c.Param("instance_id")
	jsonSchema, uiSchema, err := h.pluginMgr.GetConfigSchemaInstance(c, category, pluginID, instanceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"json_schema": jsonSchema, "ui_schema": uiSchema})
}

func (h *Handler) AdminPluginInstanceConfigGet(c *gin.Context) {
	if h.pluginMgr == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plugins disabled"})
		return
	}
	category := c.Param("category")
	pluginID := c.Param("plugin_id")
	instanceID := c.Param("instance_id")
	cfg, err := h.pluginMgr.GetConfigInstance(c, category, pluginID, instanceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"config_json": cfg})
}

func (h *Handler) AdminPluginInstanceConfigUpdate(c *gin.Context) {
	if h.pluginMgr == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plugins disabled"})
		return
	}
	category := c.Param("category")
	pluginID := c.Param("plugin_id")
	instanceID := c.Param("instance_id")
	var payload struct {
		ConfigJSON string `json:"config_json"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if err := h.pluginMgr.UpdateConfigInstance(c, category, pluginID, instanceID, payload.ConfigJSON); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c, getUserID(c), "plugin.config_update", "plugin", category+"/"+pluginID+"/"+instanceID, map[string]any{})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func writePluginHandlerError(c *gin.Context, err error) {
	if cfgErr, ok := plugins.AsConfigValidationError(err); ok && cfgErr != nil {
		resp := gin.H{
			"error": cfgErr.Error(),
			"code":  strings.TrimSpace(cfgErr.Code),
		}
		if len(cfgErr.MissingFields) > 0 {
			resp["missing_fields"] = cfgErr.MissingFields
		}
		if p := strings.TrimSpace(cfgErr.RedirectPath); p != "" {
			resp["redirect_path"] = p
		}
		c.JSON(http.StatusConflict, resp)
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
}

func (h *Handler) AdminPluginDeleteFiles(c *gin.Context) {
	if h.pluginMgr == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plugins disabled"})
		return
	}
	category := c.Param("category")
	pluginID := c.Param("plugin_id")
	if err := h.pluginMgr.DeletePluginFiles(c, category, pluginID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.adminSvc != nil {
		h.adminSvc.Audit(c, getUserID(c), "plugin.delete_files", "plugin", category+"/"+pluginID, map[string]any{})
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
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
		msg := err.Error()
		if err == usecase.ErrNotFound {
			status = http.StatusNotFound
		}
		if err == usecase.ErrConflict {
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
		GoodsTypeID          int64          `json:"goods_type_id"`
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
			if payload.GoodsTypeID == 0 {
				payload.GoodsTypeID = pkg.GoodsTypeID
			}
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
			if payload.GoodsTypeID == 0 {
				payload.GoodsTypeID = region.GoodsTypeID
			}
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
		GoodsTypeID:          payload.GoodsTypeID,
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
		plan, err := h.catalogSvc.GetPlanGroup(c, planGroupID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "plan_group not found"})
			return
		}
		if plan.LineID <= 0 {
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
	c.JSON(http.StatusOK, gin.H{"items": toSystemImageDTOs(items)})
}

func (h *Handler) AdminRegions(c *gin.Context) {
	goodsTypeID, _ := strconv.ParseInt(c.Query("goods_type_id"), 10, 64)
	items, err := h.catalogSvc.ListRegions(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list error"})
		return
	}
	if goodsTypeID > 0 {
		filtered := make([]domain.Region, 0, len(items))
		for _, item := range items {
			if item.GoodsTypeID == goodsTypeID {
				filtered = append(filtered, item)
			}
		}
		items = filtered
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
	if region.GoodsTypeID <= 0 {
		region.GoodsTypeID = h.defaultGoodsTypeID(c)
	}
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
	if region.GoodsTypeID <= 0 {
		if current, err := h.catalogSvc.GetRegion(c, id); err == nil && current.GoodsTypeID > 0 {
			region.GoodsTypeID = current.GoodsTypeID
		}
	}
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
	goodsTypeID, _ := strconv.ParseInt(c.Query("goods_type_id"), 10, 64)
	items, err := h.catalogSvc.ListPlanGroups(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list error"})
		return
	}
	if goodsTypeID > 0 {
		filtered := make([]domain.PlanGroup, 0, len(items))
		for _, item := range items {
			if item.GoodsTypeID == goodsTypeID {
				filtered = append(filtered, item)
			}
		}
		items = filtered
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
	if plan.RegionID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid region_id"})
		return
	}
	if region, err := h.catalogSvc.GetRegion(c, plan.RegionID); err == nil {
		plan.GoodsTypeID = region.GoodsTypeID
	}
	if plan.GoodsTypeID <= 0 {
		plan.GoodsTypeID = h.defaultGoodsTypeID(c)
	}
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
		if region, err := h.catalogSvc.GetRegion(c, plan.RegionID); err == nil && region.GoodsTypeID > 0 {
			plan.GoodsTypeID = region.GoodsTypeID
		}
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
	goodsTypeID, _ := strconv.ParseInt(c.Query("goods_type_id"), 10, 64)
	items, err := h.catalogSvc.ListPackages(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list error"})
		return
	}
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
	if pkg.PlanGroupID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan_group_id"})
		return
	}
	if plan, err := h.catalogSvc.GetPlanGroup(c, pkg.PlanGroupID); err == nil {
		pkg.GoodsTypeID = plan.GoodsTypeID
	}
	if pkg.GoodsTypeID <= 0 {
		pkg.GoodsTypeID = h.defaultGoodsTypeID(c)
	}
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
		if plan, err := h.catalogSvc.GetPlanGroup(c, pkg.PlanGroupID); err == nil && plan.GoodsTypeID > 0 {
			pkg.GoodsTypeID = plan.GoodsTypeID
		}
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
	if h.integration == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}

	lineID, _ := strconv.ParseInt(c.Query("line_id"), 10, 64)
	planGroupID, _ := strconv.ParseInt(c.Query("plan_group_id"), 10, 64)
	if planGroupID > 0 {
		plan, err := h.catalogSvc.GetPlanGroup(c, planGroupID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "plan_group not found"})
			return
		}
		if plan.LineID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "line_id required for plan_group"})
			return
		}
		lineID = plan.LineID
	}

	if lineID > 0 {
		count, err := h.integration.SyncAutomationImagesForLine(c, lineID, "merge")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		resp := gin.H{
			"count":   count,
			"line_id": lineID,
		}
		if images, lerr := h.catalogSvc.ListSystemImages(c, lineID); lerr == nil {
			resp["line_image_count"] = len(images)
		}
		c.JSON(http.StatusOK, resp)
		return
	}

	goodsTypeID, _ := strconv.ParseInt(c.Query("goods_type_id"), 10, 64)
	if goodsTypeID <= 0 {
		if h.goodsTypes == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "goods_type_id required"})
			return
		}
		items, err := h.goodsTypes.List(c)
		if err != nil || len(items) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "goods_type_id required"})
			return
		}
		sort.SliceStable(items, func(i, j int) bool {
			if items[i].SortOrder == items[j].SortOrder {
				return items[i].ID < items[j].ID
			}
			return items[i].SortOrder < items[j].SortOrder
		})
		goodsTypeID = items[0].ID
	}
	result, err := h.integration.SyncAutomationForGoodsType(c, goodsTypeID, "merge")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"count":         result.Images,
		"goods_type_id": goodsTypeID,
		"sync_result": gin.H{
			"lines":    result.Lines,
			"products": result.Products,
			"images":   result.Images,
		},
	})
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

func (h *Handler) AdminPushTokenRegister(c *gin.Context) {
	if h.pushSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	var payload struct {
		Token    string `json:"token"`
		Platform string `json:"platform"`
		DeviceID string `json:"device_id"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if err := h.pushSvc.RegisterToken(c, getUserID(c), payload.Platform, payload.Token, payload.DeviceID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPushTokenDelete(c *gin.Context) {
	if h.pushSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	var payload struct {
		Token string `json:"token"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if err := h.pushSvc.RemoveToken(c, getUserID(c), payload.Token); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
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
	if h.pluginMgr == nil {
		c.JSON(http.StatusOK, gin.H{
			"base_url":      "",
			"api_key":       "",
			"enabled":       false,
			"timeout_sec":   12,
			"retry":         0,
			"dry_run":       false,
			"configured":    false,
			"compat_mode":   false,
			"plugins_ready": false,
			"config_source": "goods_type_plugin_instance",
		})
		return
	}
	cfg, present, binding, enabled, err := h.readAutomationPluginConfig(c)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error":         "no writable automation plugin instance found; configure automation plugin instance first",
			"code":          "no_writable_automation_instance",
			"redirect_path": "/admin/catalog",
		})
		return
	}
	if cfg.TimeoutSec <= 0 {
		cfg.TimeoutSec = 12
	}
	if cfg.Retry < 0 {
		cfg.Retry = 0
	}
	configured := present["base_url"] && strings.TrimSpace(cfg.BaseURL) != "" &&
		present["api_key"] && strings.TrimSpace(cfg.APIKey) != ""
	resp := gin.H{
		"base_url":      cfg.BaseURL,
		"api_key":       cfg.APIKey,
		"enabled":       enabled,
		"timeout_sec":   cfg.TimeoutSec,
		"retry":         cfg.Retry,
		"dry_run":       cfg.DryRun,
		"plugin_id":     binding.PluginID,
		"instance_id":   binding.InstanceID,
		"configured":    configured,
		"compat_mode":   false,
		"config_source": "goods_type_plugin_instance",
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) AdminAutomationConfigUpdate(c *gin.Context) {
	if h.pluginMgr == nil {
		c.JSON(http.StatusOK, gin.H{
			"ok":            true,
			"compat_mode":   false,
			"plugins_ready": false,
		})
		return
	}
	var payload struct {
		BaseURL    *string `json:"base_url"`
		APIKey     *string `json:"api_key"`
		Enabled    *bool   `json:"enabled"`
		TimeoutSec *int    `json:"timeout_sec"`
		Retry      *int    `json:"retry"`
		DryRun     *bool   `json:"dry_run"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	binding, err := h.resolveWritableAutomationBinding(c)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error":         "no writable automation plugin instance found; configure automation plugin instance first",
			"code":          "no_writable_automation_instance",
			"redirect_path": "/admin/catalog",
		})
		return
	}

	current := usecase.AutomationConfig{}
	cfgJSON, err := h.pluginMgr.GetConfigInstance(c, "automation", binding.PluginID, binding.InstanceID)
	if err == nil {
		if cfg, _, perr := parseAutomationConfigJSON(cfgJSON); perr == nil {
			current = cfg
		}
	}
	if payload.BaseURL != nil {
		current.BaseURL = strings.TrimSpace(*payload.BaseURL)
	}
	if payload.APIKey != nil {
		current.APIKey = strings.TrimSpace(*payload.APIKey)
	}
	if payload.TimeoutSec != nil {
		current.TimeoutSec = *payload.TimeoutSec
	}
	if payload.Retry != nil {
		current.Retry = *payload.Retry
	}
	if payload.DryRun != nil {
		current.DryRun = *payload.DryRun
	}
	if current.TimeoutSec <= 0 {
		current.TimeoutSec = 12
	}
	if current.Retry < 0 {
		current.Retry = 0
	}

	rawCfg, _ := json.Marshal(map[string]any{
		"base_url":    current.BaseURL,
		"api_key":     current.APIKey,
		"timeout_sec": current.TimeoutSec,
		"retry":       current.Retry,
		"dry_run":     current.DryRun,
	})
	if err := h.pluginMgr.UpdateConfigInstance(c, "automation", binding.PluginID, binding.InstanceID, string(rawCfg)); err != nil {
		writePluginHandlerError(c, err)
		return
	}

	if payload.Enabled != nil {
		if *payload.Enabled {
			if err := h.pluginMgr.EnableInstance(c, "automation", binding.PluginID, binding.InstanceID); err != nil {
				writePluginHandlerError(c, err)
				return
			}
		} else {
			if err := h.pluginMgr.DisableInstance(c, "automation", binding.PluginID, binding.InstanceID); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":          true,
		"compat_mode": false,
		"plugin_id":   binding.PluginID,
		"instance_id": binding.InstanceID,
	})
}

type automationBinding struct {
	PluginID   string
	InstanceID string
}

func (h *Handler) readAutomationPluginConfig(c *gin.Context) (usecase.AutomationConfig, map[string]bool, automationBinding, bool, error) {
	if h.pluginMgr == nil {
		return usecase.AutomationConfig{}, nil, automationBinding{}, false, errors.New("plugins disabled")
	}
	items, err := h.pluginMgr.List(c)
	if err != nil {
		return usecase.AutomationConfig{}, nil, automationBinding{}, false, err
	}
	enabledByBinding := map[string]bool{}
	for _, item := range items {
		if !strings.EqualFold(strings.TrimSpace(item.Category), "automation") {
			continue
		}
		key := strings.TrimSpace(item.PluginID) + ":" + strings.TrimSpace(item.InstanceID)
		enabledByBinding[key] = item.Enabled
	}
	for _, binding := range h.collectAutomationBindingCandidates(c) {
		cfgJSON, err := h.pluginMgr.GetConfigInstance(c, "automation", binding.PluginID, binding.InstanceID)
		if err != nil {
			continue
		}
		cfg, present, err := parseAutomationConfigJSON(cfgJSON)
		if err != nil {
			continue
		}
		key := binding.PluginID + ":" + binding.InstanceID
		return cfg, present, binding, enabledByBinding[key], nil
	}
	return usecase.AutomationConfig{}, nil, automationBinding{}, false, errors.New("automation plugin instance not found")
}

func (h *Handler) resolveWritableAutomationBinding(c *gin.Context) (automationBinding, error) {
	if h.pluginMgr == nil {
		return automationBinding{}, errors.New("plugins disabled")
	}
	for _, binding := range h.collectAutomationBindingCandidates(c) {
		if _, err := h.pluginMgr.GetConfigInstance(c, "automation", binding.PluginID, binding.InstanceID); err == nil {
			return binding, nil
		}
	}
	return automationBinding{}, errors.New("automation plugin instance not found")
}

func (h *Handler) collectAutomationBindingCandidates(c *gin.Context) []automationBinding {
	candidates := make([]automationBinding, 0, 4)
	if h.goodsTypes != nil {
		items, err := h.goodsTypes.List(c)
		if err == nil {
			sort.SliceStable(items, func(i, j int) bool {
				if items[i].SortOrder == items[j].SortOrder {
					return items[i].ID < items[j].ID
				}
				return items[i].SortOrder < items[j].SortOrder
			})
			for _, item := range items {
				if !strings.EqualFold(strings.TrimSpace(item.AutomationCategory), "automation") {
					continue
				}
				pluginID := strings.TrimSpace(item.AutomationPluginID)
				instanceID := strings.TrimSpace(item.AutomationInstanceID)
				if pluginID == "" || instanceID == "" {
					continue
				}
				candidates = append(candidates, automationBinding{PluginID: pluginID, InstanceID: instanceID})
			}
		}
	}

	uniq := make(map[string]struct{}, len(candidates))
	out := make([]automationBinding, 0, len(candidates))
	for _, candidate := range candidates {
		key := candidate.PluginID + ":" + candidate.InstanceID
		if _, exists := uniq[key]; exists {
			continue
		}
		uniq[key] = struct{}{}
		out = append(out, candidate)
	}
	return out
}

func parseAutomationConfigJSON(raw string) (usecase.AutomationConfig, map[string]bool, error) {
	cfg := usecase.AutomationConfig{}
	present := map[string]bool{}
	payload := strings.TrimSpace(raw)
	if payload == "" {
		return cfg, present, nil
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(payload), &obj); err != nil {
		return cfg, present, err
	}
	if v, ok := obj["base_url"]; ok {
		cfg.BaseURL = strings.TrimSpace(toString(v))
		present["base_url"] = true
	}
	if v, ok := obj["api_key"]; ok {
		cfg.APIKey = strings.TrimSpace(toString(v))
		present["api_key"] = true
	}
	if v, ok := obj["timeout_sec"]; ok {
		if n, ok := toInt(v); ok {
			cfg.TimeoutSec = n
		}
		present["timeout_sec"] = true
	}
	if v, ok := obj["retry"]; ok {
		if n, ok := toInt(v); ok {
			cfg.Retry = n
		}
		present["retry"] = true
	}
	if v, ok := obj["dry_run"]; ok {
		if b, ok := v.(bool); ok {
			cfg.DryRun = b
		}
		present["dry_run"] = true
	}
	return cfg, present, nil
}

func mergeAutomationConfig(base, override usecase.AutomationConfig, present map[string]bool) usecase.AutomationConfig {
	out := base
	if present["base_url"] && strings.TrimSpace(override.BaseURL) != "" {
		out.BaseURL = override.BaseURL
	}
	if present["api_key"] && strings.TrimSpace(override.APIKey) != "" {
		out.APIKey = override.APIKey
	}
	if present["timeout_sec"] && override.TimeoutSec > 0 {
		out.TimeoutSec = override.TimeoutSec
	}
	if present["retry"] && override.Retry >= 0 {
		out.Retry = override.Retry
	}
	if present["dry_run"] {
		out.DryRun = override.DryRun
	}
	return out
}

func toString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	default:
		return fmt.Sprintf("%v", t)
	}
}

func toInt(v any) (int, bool) {
	switch t := v.(type) {
	case int:
		return t, true
	case int8:
		return int(t), true
	case int16:
		return int(t), true
	case int32:
		return int(t), true
	case int64:
		return int(t), true
	case uint:
		return int(t), true
	case uint8:
		return int(t), true
	case uint16:
		return int(t), true
	case uint32:
		return int(t), true
	case uint64:
		return int(t), true
	case float32:
		return int(t), true
	case float64:
		return int(t), true
	case json.Number:
		n, err := t.Int64()
		if err != nil {
			return 0, false
		}
		return int(n), true
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(t))
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
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

func (h *Handler) AdminGoodsTypes(c *gin.Context) {
	if h.goodsTypes == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	items, err := h.goodsTypes.List(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) AdminGoodsTypeCreate(c *gin.Context) {
	if h.goodsTypes == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	var payload struct {
		Code               string `json:"code"`
		Name               string `json:"name"`
		Active             bool   `json:"active"`
		SortOrder          int    `json:"sort_order"`
		AutomationPluginID string `json:"automation_plugin_id"`
		AutomationInstance string `json:"automation_instance_id"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	gt := &domain.GoodsType{
		Code:                 strings.TrimSpace(payload.Code),
		Name:                 strings.TrimSpace(payload.Name),
		Active:               payload.Active,
		SortOrder:            payload.SortOrder,
		AutomationCategory:   "automation",
		AutomationPluginID:   strings.TrimSpace(payload.AutomationPluginID),
		AutomationInstanceID: strings.TrimSpace(payload.AutomationInstance),
	}
	if err := h.goodsTypes.Create(c, gt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gt)
}

func (h *Handler) AdminGoodsTypeUpdate(c *gin.Context) {
	if h.goodsTypes == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var payload struct {
		Code               string `json:"code"`
		Name               string `json:"name"`
		Active             bool   `json:"active"`
		SortOrder          int    `json:"sort_order"`
		AutomationPluginID string `json:"automation_plugin_id"`
		AutomationInstance string `json:"automation_instance_id"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	gt := domain.GoodsType{
		ID:                   id,
		Code:                 strings.TrimSpace(payload.Code),
		Name:                 strings.TrimSpace(payload.Name),
		Active:               payload.Active,
		SortOrder:            payload.SortOrder,
		AutomationCategory:   "automation",
		AutomationPluginID:   strings.TrimSpace(payload.AutomationPluginID),
		AutomationInstanceID: strings.TrimSpace(payload.AutomationInstance),
	}
	if err := h.goodsTypes.Update(c, gt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminGoodsTypeDelete(c *gin.Context) {
	if h.goodsTypes == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.goodsTypes.Delete(c, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminGoodsTypeSyncAutomation(c *gin.Context) {
	if h.integration == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	mode := c.Query("mode")
	result, err := h.integration.SyncAutomationForGoodsType(c, id, mode)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
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
	mangzhu := h.realnameSvc.GetMangzhuConfig(c)
	c.JSON(http.StatusOK, gin.H{
		"enabled":       enabled,
		"provider":      provider,
		"block_actions": actions,
		"mangzhu": gin.H{
			"base_url":      mangzhu.BaseURL,
			"auth_mode":     mangzhu.AuthMode,
			"face_provider": mangzhu.FaceProvider,
			"timeout_sec":   mangzhu.TimeoutSec,
			"key_set":       mangzhu.KeySet,
		},
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
		Mangzhu      struct {
			BaseURL      string `json:"base_url"`
			Key          string `json:"key"`
			AuthMode     string `json:"auth_mode"`
			FaceProvider string `json:"face_provider"`
			TimeoutSec   int    `json:"timeout_sec"`
		} `json:"mangzhu"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if err := h.realnameSvc.UpdateConfig(c, payload.Enabled, payload.Provider, payload.BlockActions); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.realnameSvc.UpdateMangzhuConfig(c, usecase.RealNameMangzhuConfig{
		BaseURL:      payload.Mangzhu.BaseURL,
		Key:          payload.Mangzhu.Key,
		AuthMode:     payload.Mangzhu.AuthMode,
		FaceProvider: payload.Mangzhu.FaceProvider,
		TimeoutSec:   payload.Mangzhu.TimeoutSec,
	}); err != nil {
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

type smsTemplateItem struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (h *Handler) AdminSMSConfig(c *gin.Context) {
	if h.settings == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	enabledRaw := strings.TrimSpace(getSettingValue(c, h.settings, "sms_enabled"))
	enabled := true
	if enabledRaw != "" {
		enabled = strings.EqualFold(enabledRaw, "true") || enabledRaw == "1"
	}
	c.JSON(http.StatusOK, gin.H{
		"enabled":              enabled,
		"plugin_id":            strings.TrimSpace(getSettingValue(c, h.settings, "sms_plugin_id")),
		"instance_id":          strings.TrimSpace(getSettingValue(c, h.settings, "sms_instance_id")),
		"default_template_id":  strings.TrimSpace(getSettingValue(c, h.settings, "sms_default_template_id")),
		"provider_template_id": strings.TrimSpace(getSettingValue(c, h.settings, "sms_provider_template_id")),
	})
}

func (h *Handler) AdminSMSConfigUpdate(c *gin.Context) {
	var payload struct {
		Enabled            bool   `json:"enabled"`
		PluginID           string `json:"plugin_id"`
		InstanceID         string `json:"instance_id"`
		DefaultTemplateID  string `json:"default_template_id"`
		ProviderTemplateID string `json:"provider_template_id"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	pluginID := strings.TrimSpace(payload.PluginID)
	instanceID := strings.TrimSpace(payload.InstanceID)
	if pluginID != "" && instanceID == "" {
		instanceID = "default"
	}
	if pluginID == "" {
		instanceID = ""
	}
	adminID := getUserID(c)
	_ = h.adminSvc.UpdateSetting(c, adminID, "sms_enabled", boolToString(payload.Enabled))
	_ = h.adminSvc.UpdateSetting(c, adminID, "sms_plugin_id", pluginID)
	_ = h.adminSvc.UpdateSetting(c, adminID, "sms_instance_id", instanceID)
	_ = h.adminSvc.UpdateSetting(c, adminID, "sms_default_template_id", strings.TrimSpace(payload.DefaultTemplateID))
	_ = h.adminSvc.UpdateSetting(c, adminID, "sms_provider_template_id", strings.TrimSpace(payload.ProviderTemplateID))
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminSMSPreview(c *gin.Context) {
	var payload struct {
		TemplateID *int64         `json:"template_id"`
		Content    string         `json:"content"`
		Variables  map[string]any `json:"variables"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	vars := map[string]any{"now": time.Now().Format(time.RFC3339)}
	for k, v := range payload.Variables {
		vars[k] = v
	}
	content := strings.TrimSpace(payload.Content)
	if payload.TemplateID != nil && *payload.TemplateID > 0 {
		rendered, ok := h.renderSMSTemplateByID(c, *payload.TemplateID, vars)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
			return
		}
		content = rendered
	} else if content != "" {
		content = renderSMSText(content, vars)
	}
	if strings.TrimSpace(content) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content required"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"content": content})
}

func (h *Handler) AdminSMSTest(c *gin.Context) {
	if h.settings == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not supported"})
		return
	}
	var payload struct {
		Phone              string         `json:"phone"`
		TemplateID         *int64         `json:"template_id"`
		Content            string         `json:"content"`
		Variables          map[string]any `json:"variables"`
		PluginID           string         `json:"plugin_id"`
		InstanceID         string         `json:"instance_id"`
		ProviderTemplateID string         `json:"provider_template_id"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if h.pluginMgr == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "plugin manager unavailable"})
		return
	}
	phoneRaw := strings.TrimSpace(payload.Phone)
	if phoneRaw == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone required"})
		return
	}
	phones := make([]string, 0, 4)
	for _, p := range strings.FieldsFunc(phoneRaw, func(r rune) bool { return r == ',' || r == ';' || r == ' ' || r == '\n' || r == '\t' }) {
		p = strings.TrimSpace(p)
		if p != "" {
			phones = append(phones, p)
		}
	}
	if len(phones) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone required"})
		return
	}

	vars := map[string]any{"now": time.Now().Format(time.RFC3339)}
	for k, v := range payload.Variables {
		vars[k] = v
	}
	content := strings.TrimSpace(payload.Content)
	if payload.TemplateID != nil && *payload.TemplateID > 0 {
		rendered, ok := h.renderSMSTemplateByID(c, *payload.TemplateID, vars)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
			return
		}
		content = rendered
	} else if content != "" {
		content = renderSMSText(content, vars)
	} else {
		defaultTemplateID := strings.TrimSpace(getSettingValue(c, h.settings, "sms_default_template_id"))
		if defaultTemplateID != "" {
			if tid, err := strconv.ParseInt(defaultTemplateID, 10, 64); err == nil && tid > 0 {
				if rendered, ok := h.renderSMSTemplateByID(c, tid, vars); ok {
					content = rendered
				}
			}
		}
	}
	if strings.TrimSpace(content) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content required"})
		return
	}

	pluginID := strings.TrimSpace(payload.PluginID)
	instanceID := strings.TrimSpace(payload.InstanceID)
	if pluginID == "" {
		pluginID = strings.TrimSpace(getSettingValue(c, h.settings, "sms_plugin_id"))
	}
	if instanceID == "" {
		instanceID = strings.TrimSpace(getSettingValue(c, h.settings, "sms_instance_id"))
	}
	if instanceID == "" {
		instanceID = "default"
	}
	if pluginID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sms plugin not configured"})
		return
	}
	providerTemplateID := strings.TrimSpace(payload.ProviderTemplateID)
	if providerTemplateID == "" {
		providerTemplateID = strings.TrimSpace(getSettingValue(c, h.settings, "sms_provider_template_id"))
	}

	if _, err := h.pluginMgr.EnsureRunning(c, "sms", pluginID, instanceID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	client, ok := h.pluginMgr.GetSMSClient("sms", pluginID, instanceID)
	if !ok || client == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sms plugin not running"})
		return
	}
	resp, err := client.Send(c, &pluginv1.SendSmsRequest{
		TemplateId: providerTemplateID,
		Content:    content,
		Phones:     phones,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if resp == nil || !resp.Ok {
		errMsg := "sms send failed"
		if resp != nil && strings.TrimSpace(resp.Error) != "" {
			errMsg = strings.TrimSpace(resp.Error)
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"ok":          true,
		"message_id":  strings.TrimSpace(resp.MessageId),
		"plugin_id":   pluginID,
		"instance_id": instanceID,
	})
}

func (h *Handler) AdminSMSTemplates(c *gin.Context) {
	items, err := h.loadSMSTemplates(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sms_templates_json"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) AdminSMSTemplateUpsert(c *gin.Context) {
	var payload smsTemplateItem
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if idParam := strings.TrimSpace(c.Param("id")); idParam != "" {
		if id, err := strconv.ParseInt(idParam, 10, 64); err == nil {
			payload.ID = id
		}
	}
	payload.Name = strings.TrimSpace(payload.Name)
	payload.Content = strings.TrimSpace(payload.Content)
	if payload.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}
	if payload.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content required"})
		return
	}
	items, err := h.loadSMSTemplates(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sms_templates_json"})
		return
	}
	now := time.Now()
	if payload.ID <= 0 {
		payload.ID = nextSMSTemplateID(items)
		payload.CreatedAt = now
		payload.UpdatedAt = now
		items = append(items, payload)
	} else {
		updated := false
		for i := range items {
			if items[i].ID != payload.ID {
				continue
			}
			payload.CreatedAt = items[i].CreatedAt
			if payload.CreatedAt.IsZero() {
				payload.CreatedAt = now
			}
			payload.UpdatedAt = now
			items[i] = payload
			updated = true
			break
		}
		if !updated {
			payload.CreatedAt = now
			payload.UpdatedAt = now
			items = append(items, payload)
		}
	}
	if err := h.saveSMSTemplates(c, getUserID(c), items); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, payload)
}

func (h *Handler) AdminSMSTemplateDelete(c *gin.Context) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	items, err := h.loadSMSTemplates(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sms_templates_json"})
		return
	}
	out := make([]smsTemplateItem, 0, len(items))
	for _, item := range items {
		if item.ID == id {
			continue
		}
		out = append(out, item)
	}
	if err := h.saveSMSTemplates(c, getUserID(c), out); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
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
		existing, err := h.users.GetUserByID(c, id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// Allow self-update requests that include the current permission_group_id.
		// Only block an actual attempt to switch permission group.
		if payload.PermissionGroupID != nil {
			existingGroupID := int64(0)
			if existing.PermissionGroupID != nil {
				existingGroupID = *existing.PermissionGroupID
			}
			if *payload.PermissionGroupID != existingGroupID {
				c.JSON(http.StatusBadRequest, gin.H{"error": "cannot update permission group"})
				return
			}
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
		if isPrimary, err := h.permissionSvc.IsPrimaryAdmin(c, userID); err == nil && isPrimary {
			dto.Permissions = []string{"*"}
			c.JSON(http.StatusOK, dto)
			return
		}
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
	ip := strings.TrimSpace(c.ClientIP())
	if ip == "" {
		ip = "unknown"
	}
	if !forgotPwdLimiter.Allow("admin_forgot_password:ip:"+ip, 5, 15*time.Minute) ||
		!forgotPwdLimiter.Allow("admin_forgot_password:email:"+strings.ToLower(strings.TrimSpace(payload.Email)), 3, 15*time.Minute) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
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
	payload.ContentHTML = sanitizeHTML(payload.ContentHTML)
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
		post.ContentHTML = sanitizeHTML(*payload.ContentHTML)
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
	if typeName == "custom_html" {
		payload.CustomHTML = sanitizeHTML(payload.CustomHTML)
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
		if block.Type == "custom_html" {
			block.CustomHTML = sanitizeHTML(*payload.CustomHTML)
		} else {
			block.CustomHTML = *payload.CustomHTML
		}
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
	opened, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file open failed"})
		return
	}
	head := make([]byte, 512)
	n, _ := io.ReadFull(opened, head)
	_ = opened.Close()
	detected := http.DetectContentType(head[:n])
	allowed := map[string]bool{
		"image/png":  true,
		"image/jpeg": true,
		"image/gif":  true,
		"image/webp": true,
	}
	if !allowed[detected] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported file type"})
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
	item := domain.Upload{Name: file.Filename, Path: localPath, URL: url, Mime: detected, Size: file.Size, UploaderID: getUserID(c)}
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

func (h *Handler) loadSMSTemplates(ctx *gin.Context) ([]smsTemplateItem, error) {
	raw := strings.TrimSpace(getSettingValue(ctx, h.settings, "sms_templates_json"))
	if raw == "" {
		return []smsTemplateItem{}, nil
	}
	var items []smsTemplateItem
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].CreatedAt.IsZero() {
			items[i].CreatedAt = time.Now()
		}
		if items[i].UpdatedAt.IsZero() {
			items[i].UpdatedAt = items[i].CreatedAt
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].ID > items[j].ID
	})
	return items, nil
}

func (h *Handler) saveSMSTemplates(ctx *gin.Context, adminID int64, items []smsTemplateItem) error {
	for i := range items {
		items[i].Name = strings.TrimSpace(items[i].Name)
		items[i].Content = strings.TrimSpace(items[i].Content)
		if items[i].CreatedAt.IsZero() {
			items[i].CreatedAt = time.Now()
		}
		if items[i].UpdatedAt.IsZero() {
			items[i].UpdatedAt = time.Now()
		}
	}
	raw, err := json.Marshal(items)
	if err != nil {
		return err
	}
	if h.adminSvc == nil {
		return errors.New("admin service unavailable")
	}
	return h.adminSvc.UpdateSetting(ctx, adminID, "sms_templates_json", string(raw))
}

func (h *Handler) renderSMSTemplateByID(ctx *gin.Context, id int64, vars map[string]any) (string, bool) {
	items, err := h.loadSMSTemplates(ctx)
	if err != nil {
		return "", false
	}
	for _, item := range items {
		if item.ID != id {
			continue
		}
		if !item.Enabled {
			return "", false
		}
		return renderSMSText(item.Content, vars), true
	}
	return "", false
}

func nextSMSTemplateID(items []smsTemplateItem) int64 {
	var maxID int64
	for _, item := range items {
		if item.ID > maxID {
			maxID = item.ID
		}
	}
	return maxID + 1
}

func renderSMSText(content string, vars map[string]any) string {
	replacer := strings.NewReplacer(
		"{{code}}", "{{.code}}",
		"{{ code }}", "{{.code}}",
		"{{phone}}", "{{.phone}}",
		"{{ phone }}", "{{.phone}}",
	)
	normalized := replacer.Replace(strings.TrimSpace(content))
	return strings.TrimSpace(usecase.RenderTemplate(normalized, vars, false))
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
