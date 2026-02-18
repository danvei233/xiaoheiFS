package http

import (
	"github.com/microcosm-cc/bluemonday"
	"regexp"
	"time"
	appadmin "xiaoheiplay/internal/app/admin"
	appadminvps "xiaoheiplay/internal/app/adminvps"
	appcart "xiaoheiplay/internal/app/cart"
	appcatalog "xiaoheiplay/internal/app/catalog"
	appcms "xiaoheiplay/internal/app/cms"
	appgoodstype "xiaoheiplay/internal/app/goodstype"
	appmessage "xiaoheiplay/internal/app/message"
	apppasswordreset "xiaoheiplay/internal/app/passwordreset"
	apppayment "xiaoheiplay/internal/app/payment"
	apppermission "xiaoheiplay/internal/app/permission"
	appports "xiaoheiplay/internal/app/ports"
	appprobe "xiaoheiplay/internal/app/probe"
	apppush "xiaoheiplay/internal/app/push"
	apprealname "xiaoheiplay/internal/app/realname"
	appscheduledtask "xiaoheiplay/internal/app/scheduledtask"
	appticket "xiaoheiplay/internal/app/ticket"
	appwallet "xiaoheiplay/internal/app/wallet"
	appwalletorder "xiaoheiplay/internal/app/walletorder"
)

var (
	htmlPolicy           = bluemonday.UGCPolicy()
	forgotPwdLimiter     = newRateLimiter()
	loginLimiter         = newRateLimiter()
	adminLoginGuard      = newLoginCooldownGuard()
	admin2FAFailureGuard = newConsecutiveFailureGuard()
	registerCodeLimiter  = newRateLimiter()
	resetCodeLimiter     = newRateLimiter()
	resetVerifyLimiter   = newRateLimiter()
	contactCodeLimiter   = newRateLimiter()
	contactVerifyLimiter = newRateLimiter()
	simpleTemplateVarRE  = regexp.MustCompile(`\{\{\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}`)
)

const (
	adminLoginFailureThreshold = 10
	adminLoginCooldown         = 10 * time.Minute
	admin2FAFailureThreshold   = 10
)

func sanitizeHTML(raw string) string {
	if raw == "" {
		return ""
	}
	return htmlPolicy.Sanitize(raw)
}

type HandlerDeps struct {
	AuthSvc           AuthService
	CatalogSvc        *appcatalog.Service
	GoodsTypes        *appgoodstype.Service
	CartSvc           *appcart.Service
	OrderSvc          OrderService
	VPSSvc            VPSService
	AdminSvc          *appadmin.Service
	AdminVPS          *appadminvps.Service
	Integration       IntegrationService
	ReportSvc         ReportService
	CMSSvc            *appcms.Service
	TicketSvc         *appticket.Service
	WalletSvc         *appwallet.Service
	WalletOrder       *appwalletorder.Service
	PaymentSvc        *apppayment.Service
	MessageSvc        *appmessage.Service
	PushSvc           *apppush.Service
	StatusSvc         StatusService
	RealnameSvc       *apprealname.Service
	OrderEventSvc     OrderEventService
	AutoLogSvc        AutomationLogService
	SettingsSvc       SettingsService
	UploadSvc         UploadService
	Broker            EventBroker
	JWTSecret         string
	PasswordReset     *apppasswordreset.Service
	SecurityTicketSvc SecurityTicketService
	PermissionSvc     *apppermission.Service
	PluginAdmin       PluginAdminService
	TaskSvc           *appscheduledtask.Service
	ProbeSvc          *appprobe.Service
	ProbeHub          *appprobe.Hub
	GeoResolver       GeoResolver
	EmailSender       appports.EmailSender
	SMSSender         appports.SMSSender
	RobotNotifier     RobotEventNotifier
}

type Handler struct {
	authSvc           AuthService
	catalogSvc        *appcatalog.Service
	goodsTypes        *appgoodstype.Service
	cartSvc           *appcart.Service
	orderSvc          OrderService
	vpsSvc            VPSService
	adminSvc          *appadmin.Service
	adminVPS          *appadminvps.Service
	integration       IntegrationService
	reportSvc         ReportService
	cmsSvc            *appcms.Service
	ticketSvc         *appticket.Service
	walletSvc         *appwallet.Service
	walletOrder       *appwalletorder.Service
	paymentSvc        *apppayment.Service
	messageSvc        *appmessage.Service
	pushSvc           *apppush.Service
	statusSvc         StatusService
	realnameSvc       *apprealname.Service
	orderEventSvc     OrderEventService
	autoLogSvc        AutomationLogService
	settingsSvc       SettingsService
	uploadSvc         UploadService
	broker            EventBroker
	jwtSecret         []byte
	passwordReset     *apppasswordreset.Service
	securityTicketSvc SecurityTicketService
	permissionSvc     *apppermission.Service
	pluginAdmin       PluginAdminService
	taskSvc           *appscheduledtask.Service
	probeSvc          *appprobe.Service
	probeHub          *appprobe.Hub
	geoResolver       GeoResolver
	emailSender       appports.EmailSender
	smsSender         appports.SMSSender
	robotNotifier     RobotEventNotifier
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
	return &Handler{
		authSvc:           deps.AuthSvc,
		catalogSvc:        deps.CatalogSvc,
		goodsTypes:        deps.GoodsTypes,
		cartSvc:           deps.CartSvc,
		orderSvc:          deps.OrderSvc,
		vpsSvc:            deps.VPSSvc,
		adminSvc:          deps.AdminSvc,
		adminVPS:          deps.AdminVPS,
		integration:       deps.Integration,
		reportSvc:         deps.ReportSvc,
		cmsSvc:            deps.CMSSvc,
		ticketSvc:         deps.TicketSvc,
		walletSvc:         deps.WalletSvc,
		walletOrder:       deps.WalletOrder,
		paymentSvc:        deps.PaymentSvc,
		messageSvc:        deps.MessageSvc,
		pushSvc:           deps.PushSvc,
		statusSvc:         deps.StatusSvc,
		realnameSvc:       deps.RealnameSvc,
		orderEventSvc:     deps.OrderEventSvc,
		autoLogSvc:        deps.AutoLogSvc,
		settingsSvc:       deps.SettingsSvc,
		uploadSvc:         deps.UploadSvc,
		broker:            deps.Broker,
		jwtSecret:         []byte(deps.JWTSecret),
		passwordReset:     deps.PasswordReset,
		securityTicketSvc: deps.SecurityTicketSvc,
		permissionSvc:     deps.PermissionSvc,
		pluginAdmin:       deps.PluginAdmin,
		taskSvc:           deps.TaskSvc,
		probeSvc:          deps.ProbeSvc,
		probeHub:          deps.ProbeHub,
		geoResolver:       deps.GeoResolver,
		emailSender:       deps.EmailSender,
		smsSender:         deps.SMSSender,
		robotNotifier:     deps.RobotNotifier,
	}
}
