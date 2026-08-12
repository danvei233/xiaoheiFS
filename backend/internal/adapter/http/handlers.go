package http

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	appadmin "xiaoheiplay/internal/app/admin"
	appadminvps "xiaoheiplay/internal/app/adminvps"
	appcart "xiaoheiplay/internal/app/cart"
	appcatalog "xiaoheiplay/internal/app/catalog"
	appcms "xiaoheiplay/internal/app/cms"
	appgoodstype "xiaoheiplay/internal/app/goodstype"
	appmessage "xiaoheiplay/internal/app/message"
	appopenapi "xiaoheiplay/internal/app/openapi"
	apppasswordreset "xiaoheiplay/internal/app/passwordreset"
	apppayment "xiaoheiplay/internal/app/payment"
	apppermission "xiaoheiplay/internal/app/permission"
	appports "xiaoheiplay/internal/app/ports"
	appprobe "xiaoheiplay/internal/app/probe"
	apppush "xiaoheiplay/internal/app/push"
	apprealname "xiaoheiplay/internal/app/realname"
	appscheduledtask "xiaoheiplay/internal/app/scheduledtask"
	appticket "xiaoheiplay/internal/app/ticket"
	appuserapikey "xiaoheiplay/internal/app/userapikey"
	appwallet "xiaoheiplay/internal/app/wallet"
	appwalletorder "xiaoheiplay/internal/app/walletorder"
	"xiaoheiplay/internal/domain"

	"github.com/microcosm-cc/bluemonday"
)

// clearWriteDeadline disables the per-connection write deadline for streaming
// responses (SSE). A server-level WriteTimeout sets an absolute deadline before
// the handler runs; long-lived event streams must opt out so they are not cut
// at the timeout boundary.
func clearWriteDeadline(w http.ResponseWriter) {
	if c, ok := w.(interface{ Unwrap() http.ResponseWriter }); ok {
		w = c.Unwrap()
	}
	if sc, ok := w.(interface{ SetWriteDeadline(time.Time) error }); ok {
		_ = sc.SetWriteDeadline(time.Time{})
	}
}

var (
	htmlPolicy          = bluemonday.UGCPolicy()
	captchaLimiter      = newRateLimiter()
	simpleTemplateVarRE = regexp.MustCompile(`\{\{\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*}}`)
)

const (
	adminLoginFailureThreshold = 10
	adminLoginCooldown         = 10 * time.Minute
	// adminLoginIPMax / adminLoginGlobalMax bound admin credential attempts per
	// source. The username cooldown alone can be hit by a single attacker using
	// many accounts (spray) or can be used to lock a victim's account out; an
	// IP and a global budget blunt both.
	adminLoginIPWindow     = 10 * time.Minute
	adminLoginIPMax        = 50
	adminLoginGlobalWindow = 10 * time.Minute
	adminLoginGlobalMax    = 500
	// admin2FAAttemptThreshold / Cooldown apply a temporary lockout on repeated
	// wrong 2FA codes. The account is deliberately never disabled by failed 2FA
	// attempts — permanent disable is a denial-of-service against the operator
	// (P0-3).
	admin2FAAttemptThreshold = 5
	admin2FACooldown         = 10 * time.Minute
	// totpVerifyAccountMax / totpVerifyIPMax bound user TOTP verification per
	// account and per source IP independently (account and IP dimensions are
	// separate, so rotating IPs cannot reset the account budget).
	totpVerifyAccountMax = 5
	totpVerifyIPMax      = 20
	tokenExpirySeconds   = 86400 // 24h
)

func sanitizeHTML(raw string) string {
	if raw == "" {
		return ""
	}
	return htmlPolicy.Sanitize(raw)
}

// isPrimaryAdminActor reports whether the given actor is the primary admin. It
// returns (false, nil) when permission resolution is unavailable so the caller
// fails closed rather than granting access on a nil service.
func (h *Handler) isPrimaryAdminActor(ctx context.Context, actorID int64) (bool, error) {
	if actorID <= 0 || h.permissionSvc == nil {
		return false, nil
	}
	return h.permissionSvc.IsPrimaryAdmin(ctx, actorID)
}

func requestActor(c *gin.Context) domain.RequestActor {
	value, ok := c.Get("request_actor")
	if !ok {
		return domain.RequestActor{}
	}
	actor, _ := value.(domain.RequestActor)
	return actor
}

// totpBudgetDims returns the account and IP dimensions for a user's TOTP
// verification budget.
func (h *Handler) totpBudgetDims(c *gin.Context, userID int64) []budgetDim {
	ip := strings.TrimSpace(c.ClientIP())
	if ip == "" {
		ip = "unknown"
	}
	return []budgetDim{
		{key: fmt.Sprintf("totp_verify:user:%d", userID), limit: totpVerifyAccountMax, window: 10 * time.Minute},
		{key: "totp_verify:ip:" + ip, limit: totpVerifyIPMax, window: 10 * time.Minute},
	}
}

// allowTOTPVerify bounds TOTP verification attempts per user so a stolen or
// captured session cannot be used to brute-force 2FA codes offline-style
// (Issue #51 P0-9). The budget is durable and cross-instance, split across the
// account and IP dimensions so rotating IPs cannot reset the account budget.
// It returns false (deny) when either dimension is locked; failures are
// recorded by the caller via recordAuthFailures so the same counter is not
// double-counted here.
func (h *Handler) allowTOTPVerify(c *gin.Context, userID int64) bool {
	if userID <= 0 || h.authFailureBudget == nil {
		return false
	}
	_, locked := h.peekAuthBudgets(c.Request.Context(), h.totpBudgetDims(c, userID), time.Now())
	return !locked
}

// budgetDim is a single failure-budget dimension (key, limit, window).
type budgetDim struct {
	key    string
	limit  int
	window time.Duration
}

// peekAuthBudgets returns whether any dimension is currently locked, sweeping
// expired rows first.
func (h *Handler) peekAuthBudgets(ctx context.Context, dims []budgetDim, now time.Time) (time.Time, bool) {
	if h.authFailureBudget == nil {
		return time.Time{}, false
	}
	_ = h.authFailureBudget.DeleteExpiredAuthFailureBudgets(ctx)
	for _, d := range dims {
		if _, locked, err := h.authFailureBudget.PeekAuthFailure(ctx, d.key, d.limit, d.window, now); err == nil && locked {
			return time.Time{}, true
		}
	}
	return time.Time{}, false
}

// recordAuthFailures increments every dimension and reports whether any is now
// locked, with the farthest locked-until time.
func (h *Handler) recordAuthFailures(ctx context.Context, dims []budgetDim, now time.Time) (time.Time, bool) {
	if h.authFailureBudget == nil {
		return time.Time{}, false
	}
	var lockedUntil time.Time
	locked := false
	for _, d := range dims {
		if _, lu, err := h.authFailureBudget.RecordAuthFailure(ctx, d.key, d.limit, d.window, now); err == nil {
			if lu.After(lockedUntil) {
				lockedUntil = lu
			}
			if lu.After(now) {
				locked = true
			}
		}
	}
	return lockedUntil, locked
}

// resetAuthFailures clears the given dimension keys on success. Only the scope
// that legitimately succeeded is reset — the global budget is left intact.
func (h *Handler) resetAuthFailures(ctx context.Context, keys ...string) {
	if h.authFailureBudget == nil {
		return
	}
	for _, k := range keys {
		_ = h.authFailureBudget.ResetAuthFailure(ctx, k)
	}
}

// durableAllow is a cross-instance fixed-window rate limit backed by the
// auth_failure_budgets table. It records every request and denies once the
// window limit is reached, so multi-instance deployments share one counter
// instead of N independent in-memory maps (P1-1). It matches the in-memory
// `Allow` semantics: the first `limit` requests in a window are allowed, the
// (limit+1)th is denied. When no durable backend is wired (tests without a
// budget repo) it falls back to a per-process limiter so behavior stays bounded.
func (h *Handler) durableAllow(ctx context.Context, key string, limit int, window time.Duration) bool {
	if limit <= 0 || window <= 0 {
		return true
	}
	if h.authFailureBudget == nil {
		return captchaLimiter.Allow(key, limit, window)
	}
	now := time.Now()
	_ = h.authFailureBudget.DeleteExpiredAuthFailureBudgets(ctx)
	count, _, err := h.authFailureBudget.RecordAuthFailure(ctx, key, limit, window, now)
	if err != nil {
		// On a backend error, fail open for the rate limit (availability over
		// strictness) rather than blocking legitimate traffic.
		return true
	}
	return count <= limit
}

type HandlerDeps struct {
	AuthSvc            AuthService
	CatalogSvc         *appcatalog.Service
	GoodsTypes         *appgoodstype.Service
	CartSvc            *appcart.Service
	OrderSvc           OrderService
	VPSSvc             VPSService
	AdminSvc           *appadmin.Service
	AdminVPS           *appadminvps.Service
	Integration        IntegrationService
	ReportSvc          ReportService
	CMSSvc             *appcms.Service
	TicketSvc          *appticket.Service
	WalletSvc          *appwallet.Service
	WalletOrder        *appwalletorder.Service
	PaymentSvc         *apppayment.Service
	MessageSvc         *appmessage.Service
	PushSvc            *apppush.Service
	StatusSvc          StatusService
	RealNameSvc        *apprealname.Service
	OrderEventSvc      OrderEventService
	AutoLogSvc         AutomationLogService
	SettingsSvc        SettingsService
	UploadSvc          UploadService
	Broker             EventBroker
	JWTSecret          string
	PasswordReset      *apppasswordreset.Service
	SecurityTicketSvc  SecurityTicketService
	PermissionSvc      *apppermission.Service
	PluginAdmin        PluginAdminService
	UserTierSvc        UserTierService
	CouponSvc          CouponService
	TaskSvc            *appscheduledtask.Service
	UserAPIKeySvc      *appuserapikey.Service
	OpenAPISvc         *appopenapi.Service
	ProbeSvc           *appprobe.Service
	ProbeHub           *appprobe.Hub
	GeoResolver        GeoResolver
	EmailSender        appports.EmailSender
	SMSSender          appports.SMSSender
	RobotNotifier      RobotEventNotifier
	RobotWebhookReplay RobotWebhookReplayRepository
	AuthFailureBudget  AuthFailureBudgetRepository
}

type Handler struct {
	authSvc            AuthService
	catalogSvc         *appcatalog.Service
	goodsTypes         *appgoodstype.Service
	cartSvc            *appcart.Service
	orderSvc           OrderService
	vpsSvc             VPSService
	adminSvc           *appadmin.Service
	adminVPS           *appadminvps.Service
	integration        IntegrationService
	reportSvc          ReportService
	cmsSvc             *appcms.Service
	ticketSvc          *appticket.Service
	walletSvc          *appwallet.Service
	walletOrder        *appwalletorder.Service
	paymentSvc         *apppayment.Service
	messageSvc         *appmessage.Service
	pushSvc            *apppush.Service
	statusSvc          StatusService
	realNameSvc        *apprealname.Service
	orderEventSvc      OrderEventService
	autoLogSvc         AutomationLogService
	settingsSvc        SettingsService
	uploadSvc          UploadService
	broker             EventBroker
	jwtSecret          []byte
	passwordReset      *apppasswordreset.Service
	securityTicketSvc  SecurityTicketService
	permissionSvc      *apppermission.Service
	pluginAdmin        PluginAdminService
	userTierSvc        UserTierService
	couponSvc          CouponService
	taskSvc            *appscheduledtask.Service
	userAPIKeySvc      *appuserapikey.Service
	openAPISvc         *appopenapi.Service
	probeSvc           *appprobe.Service
	probeHub           *appprobe.Hub
	geoResolver        GeoResolver
	emailSender        appports.EmailSender
	smsSender          appports.SMSSender
	robotNotifier      RobotEventNotifier
	robotWebhookReplay RobotWebhookReplayRepository
	authFailureBudget  AuthFailureBudgetRepository
}

type authSettings struct {
	RegisterEnabled        bool
	RegisterRequiredFields []string
	RegisterEmailRequired  bool
	PasswordMinLen         int
	PasswordRequireUpper   bool
	PasswordRequireLower   bool
	PasswordRequireNumber  bool
	PasswordRequireSymbol  bool
	RegisterVerifyType     string // legacy none|email|sms
	RegisterVerifyChannels []string
	RegisterVerifyTTL      time.Duration
	RegisterCaptchaEnabled bool
	CaptchaProvider        string
	GeeTestCaptchaID       string
	GeeTestCaptchaKey      string
	GeeTestAPIServer       string
	RegisterEmailSubject   string
	RegisterEmailBody      string
	RegisterSMSPluginID    string
	RegisterSMSInstanceID  string
	RegisterSMSTemplateID  string
	LoginCaptchaEnabled    bool
	LoginRateLimitEnabled  bool
	LoginRateLimitWindow   time.Duration
	LoginRateLimitMax      int
	LoginNotifyEnabled     bool
	LoginNotifyOnFirst     bool
	LoginNotifyOnIPChange  bool
	LoginNotifyChannels    []string

	PasswordResetEnabled   bool
	PasswordResetChannels  []string
	PasswordResetVerifyTTL time.Duration

	SMSCodeLength       int
	SMSCodeComplexity   string
	EmailCodeLength     int
	EmailCodeComplexity string
	CaptchaLength       int
	CaptchaComplexity   string

	EmailBindEnabled               bool
	PhoneBindEnabled               bool
	ContactChangeNotifyOldEnabled  bool
	ContactBindVerifyTTL           time.Duration
	BindRequirePasswordWhenNo2FA   bool
	RebindRequirePasswordWhenNo2FA bool
	TwoFAEnabled                   bool
	TwoFABindEnabled               bool
	TwoFARebindEnabled             bool
	GeoIPMMDBPath                  string
}

func NewHandler(deps HandlerDeps) *Handler {
	if deps.GeoResolver == nil {
		deps.GeoResolver = NewMMDBGeoResolver()
	}
	// A missing JWT secret would leave every issued token unverifiable or
	// signed with an empty key; fail fast at startup rather than deploying an
	// auth-broken build (P2-57).
	if strings.TrimSpace(deps.JWTSecret) == "" {
		panic("NewHandler: JWTSecret must not be empty")
	}
	return &Handler{
		authSvc:            deps.AuthSvc,
		catalogSvc:         deps.CatalogSvc,
		goodsTypes:         deps.GoodsTypes,
		cartSvc:            deps.CartSvc,
		orderSvc:           deps.OrderSvc,
		vpsSvc:             deps.VPSSvc,
		adminSvc:           deps.AdminSvc,
		adminVPS:           deps.AdminVPS,
		integration:        deps.Integration,
		reportSvc:          deps.ReportSvc,
		cmsSvc:             deps.CMSSvc,
		ticketSvc:          deps.TicketSvc,
		walletSvc:          deps.WalletSvc,
		walletOrder:        deps.WalletOrder,
		paymentSvc:         deps.PaymentSvc,
		messageSvc:         deps.MessageSvc,
		pushSvc:            deps.PushSvc,
		statusSvc:          deps.StatusSvc,
		realNameSvc:        deps.RealNameSvc,
		orderEventSvc:      deps.OrderEventSvc,
		autoLogSvc:         deps.AutoLogSvc,
		settingsSvc:        deps.SettingsSvc,
		uploadSvc:          deps.UploadSvc,
		broker:             deps.Broker,
		jwtSecret:          []byte(deps.JWTSecret),
		passwordReset:      deps.PasswordReset,
		securityTicketSvc:  deps.SecurityTicketSvc,
		permissionSvc:      deps.PermissionSvc,
		pluginAdmin:        deps.PluginAdmin,
		userTierSvc:        deps.UserTierSvc,
		couponSvc:          deps.CouponSvc,
		taskSvc:            deps.TaskSvc,
		userAPIKeySvc:      deps.UserAPIKeySvc,
		openAPISvc:         deps.OpenAPISvc,
		probeSvc:           deps.ProbeSvc,
		probeHub:           deps.ProbeHub,
		geoResolver:        deps.GeoResolver,
		emailSender:        deps.EmailSender,
		smsSender:          deps.SMSSender,
		robotNotifier:      deps.RobotNotifier,
		robotWebhookReplay: deps.RobotWebhookReplay,
		authFailureBudget:  deps.AuthFailureBudget,
	}
}
