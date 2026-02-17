package http

import (
	"context"
	"io"
	"net/http"
	"time"

	appintegration "xiaoheiplay/internal/app/integration"
	appreport "xiaoheiplay/internal/app/report"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

type RobotEventNotifier interface {
	NotifyOrderEvent(ctx context.Context, ev domain.OrderEvent) error
}

type EventBroker interface {
	Publish(ctx context.Context, orderID int64, eventType string, payload any) (domain.OrderEvent, error)
	Stream(ctx context.Context, w http.ResponseWriter, orderID int64, lastSeq int64) error
}

type SettingsService interface {
	Get(ctx context.Context, key string) (domain.Setting, error)
	List(ctx context.Context) ([]domain.Setting, error)
	ListEmailTemplates(ctx context.Context) ([]domain.EmailTemplate, error)
}

type SecurityTicketService interface {
	DeleteExpired(ctx context.Context) error
	Create(ctx context.Context, userID int64, channel, receiver, token string, expiresAt time.Time) error
	Get(ctx context.Context, token string) (domain.PasswordResetTicket, error)
	MarkUsed(ctx context.Context, id int64) error
}

type OrderEventService interface {
	ListAfter(ctx context.Context, orderID int64, afterSeq int64, limit int) ([]domain.OrderEvent, error)
}

type AutomationLogService interface {
	List(ctx context.Context, orderID int64, limit, offset int) ([]domain.AutomationLog, int, error)
}

type StatusService interface {
	Status(ctx context.Context) (appshared.ServerStatus, error)
}

type UploadService interface {
	Create(ctx context.Context, item *domain.Upload) error
	List(ctx context.Context, limit, offset int) ([]domain.Upload, int, error)
}

type AuthService interface {
	CreateCaptchaWithPolicy(ctx context.Context, ttl time.Duration, length int, complexity string) (domain.Captcha, string, error)
	VerifyCaptcha(ctx context.Context, id, code string) error
	Register(ctx context.Context, in appshared.RegisterInput) (domain.User, error)
	Login(ctx context.Context, usernameOrEmail, password string) (domain.User, error)
	VerifyPassword(ctx context.Context, userID int64, password string) error
	GetUser(ctx context.Context, userID int64) (domain.User, error)
	GetUserByPhone(ctx context.Context, phone string) (domain.User, error)
	GetUserByUsernameOrEmail(ctx context.Context, account string) (domain.User, error)
	UpdateUser(ctx context.Context, user domain.User) error
	UpdateLoginSecurity(ctx context.Context, userID int64, ip, city, tz string, at time.Time) error
	SetupTOTP(ctx context.Context, userID int64, password, currentCode string) (string, string, error)
	ConfirmTOTP(ctx context.Context, userID int64, code string) error
	VerifyTOTP(ctx context.Context, userID int64, code string) error
	UpdateProfile(ctx context.Context, userID int64, in appshared.UpdateProfileInput) (domain.User, error)
	CreateVerificationCodeWithPolicy(ctx context.Context, channel, receiver, purpose string, ttl time.Duration, length int, complexity string) (string, error)
	VerifyVerificationCode(ctx context.Context, channel, receiver, purpose, code string) error
}

type OrderService interface {
	GetOrderForAdmin(ctx context.Context, orderID int64) (domain.Order, []domain.OrderItem, error)
	ListPaymentsForOrderAdmin(ctx context.Context, orderID int64) ([]domain.OrderPayment, error)
	ApproveOrder(ctx context.Context, adminID int64, orderID int64) error
	RejectOrder(ctx context.Context, adminID int64, orderID int64, reason string) error
	MarkPaid(ctx context.Context, adminID int64, orderID int64, input appshared.PaymentInput) (domain.OrderPayment, error)
	RetryProvision(orderID int64) error
	CreateEmergencyRenewOrder(ctx context.Context, userID int64, vpsID int64) (domain.Order, error)
	CancelOrder(ctx context.Context, userID int64, orderID int64) error
	ListOrders(ctx context.Context, filter appshared.OrderFilter, limit, offset int) ([]domain.Order, int, error)
	GetOrder(ctx context.Context, orderID int64, userID int64) (domain.Order, []domain.OrderItem, error)
	ListPaymentsForOrder(ctx context.Context, userID int64, orderID int64) ([]domain.OrderPayment, error)
	RefreshOrder(ctx context.Context, userID int64, orderID int64) ([]domain.VPSInstance, error)
	CreateOrderFromItems(ctx context.Context, userID int64, currency string, inputs []appshared.OrderItemInput, idemKey string) (domain.Order, []domain.OrderItem, error)
	CreateOrderFromCart(ctx context.Context, userID int64, currency string, idemKey string) (domain.Order, []domain.OrderItem, error)
	SubmitPayment(ctx context.Context, userID int64, orderID int64, input appshared.PaymentInput, idemKey string) (domain.OrderPayment, error)
	CreateRenewOrder(ctx context.Context, userID int64, vpsID int64, renewDays int, durationMonths int) (domain.Order, error)
	CreateResizeOrder(ctx context.Context, userID int64, vpsID int64, spec *appshared.CartSpec, targetPackageID int64, resetAddons bool, scheduledAt *time.Time) (domain.Order, appshared.ResizeQuote, error)
	QuoteResizeOrder(ctx context.Context, userID int64, vpsID int64, spec *appshared.CartSpec, targetPackageID int64, resetAddons bool) (appshared.ResizeQuote, appshared.CartSpec, error)
	CreateRefundOrder(ctx context.Context, userID int64, vpsID int64, reason string) (domain.Order, int64, error)
}

type VPSService interface {
	ListByUser(ctx context.Context, userID int64) ([]domain.VPSInstance, error)
	Get(ctx context.Context, id int64, userID int64) (domain.VPSInstance, error)
	RefreshStatus(ctx context.Context, inst domain.VPSInstance) (domain.VPSInstance, error)
	GetPanelURL(ctx context.Context, inst domain.VPSInstance) (string, error)
	Monitor(ctx context.Context, inst domain.VPSInstance) (appshared.AutomationMonitor, error)
	SetStatus(ctx context.Context, inst domain.VPSInstance, status domain.VPSStatus, automationState int) error
	VNCURL(ctx context.Context, inst domain.VPSInstance) (string, error)
	Start(ctx context.Context, inst domain.VPSInstance) error
	Shutdown(ctx context.Context, inst domain.VPSInstance) error
	Reboot(ctx context.Context, inst domain.VPSInstance) error
	ResetOS(ctx context.Context, inst domain.VPSInstance, templateID int64, password string) error
	UpdateLocalSystemID(ctx context.Context, inst domain.VPSInstance, systemID int64) error
	ResetOSPassword(ctx context.Context, inst domain.VPSInstance, password string) error
	ListSnapshots(ctx context.Context, inst domain.VPSInstance) ([]appshared.AutomationSnapshot, error)
	CreateSnapshot(ctx context.Context, inst domain.VPSInstance) error
	DeleteSnapshot(ctx context.Context, inst domain.VPSInstance, snapshotID int64) error
	RestoreSnapshot(ctx context.Context, inst domain.VPSInstance, snapshotID int64) error
	ListBackups(ctx context.Context, inst domain.VPSInstance) ([]appshared.AutomationBackup, error)
	CreateBackup(ctx context.Context, inst domain.VPSInstance) error
	DeleteBackup(ctx context.Context, inst domain.VPSInstance, backupID int64) error
	RestoreBackup(ctx context.Context, inst domain.VPSInstance, backupID int64) error
	ListFirewallRules(ctx context.Context, inst domain.VPSInstance) ([]appshared.AutomationFirewallRule, error)
	AddFirewallRule(ctx context.Context, inst domain.VPSInstance, req appshared.AutomationFirewallRuleCreate) error
	DeleteFirewallRule(ctx context.Context, inst domain.VPSInstance, ruleID int64) error
	ListPortMappings(ctx context.Context, inst domain.VPSInstance) ([]appshared.AutomationPortMapping, error)
	AddPortMapping(ctx context.Context, inst domain.VPSInstance, req appshared.AutomationPortMappingCreate) error
	FindPortCandidates(ctx context.Context, inst domain.VPSInstance, keywords string) ([]int64, error)
	DeletePortMapping(ctx context.Context, inst domain.VPSInstance, mappingID int64) error
}

type ReportService interface {
	Overview(ctx context.Context) (appreport.OverviewReport, error)
	RevenueByDay(ctx context.Context, days int) ([]appreport.RevenuePoint, error)
	RevenueByMonth(ctx context.Context, months int) ([]appreport.RevenuePoint, error)
	VPSStatus(ctx context.Context) ([]appreport.StatusPoint, error)
}

type IntegrationService interface {
	SyncAutomation(ctx context.Context, mode string) (appintegration.SyncResult, error)
	SyncAutomationForGoodsType(ctx context.Context, goodsTypeID int64, mode string) (appintegration.SyncResult, error)
	SyncAutomationImagesForLine(ctx context.Context, lineID int64, mode string) (int, error)
	ListSyncLogs(ctx context.Context, target string, limit, offset int) ([]domain.IntegrationSyncLog, int, error)
}

type PluginAdminService interface {
	UpsertPaymentMethod(ctx context.Context, category, pluginID, instanceID, method string, enabled bool) error
	ResolveUploadPassword(ctx context.Context, configuredPassword string) string
	ResolveUploadDir(ctx context.Context, configuredDir string) string
	List(ctx context.Context) ([]appshared.PluginListItem, error)
	DiscoverOnDisk(ctx context.Context) ([]appshared.PluginDiscoverItem, error)
	ListPaymentMethods(ctx context.Context, category, pluginID, instanceID string) ([]appshared.PluginPaymentMethodState, error)
	UpdatePaymentMethod(ctx context.Context, category, pluginID, instanceID, method string, enabled bool) error
	Install(ctx context.Context, filename string, r io.Reader) (domain.PluginInstallation, error)
	Uninstall(ctx context.Context, category, pluginID string) error
	SignatureStatusOnDisk(category, pluginID string) (domain.PluginSignatureStatus, error)
	ImportFromDisk(ctx context.Context, category, pluginID string) (domain.PluginInstallation, error)
	EnableInstance(ctx context.Context, category, pluginID, instanceID string) error
	DisableInstance(ctx context.Context, category, pluginID, instanceID string) error
	DeleteInstance(ctx context.Context, category, pluginID, instanceID string) error
	GetConfigSchemaInstance(ctx context.Context, category, pluginID, instanceID string) (jsonSchema, uiSchema string, err error)
	GetConfigInstance(ctx context.Context, category, pluginID, instanceID string) (string, error)
	UpdateConfigInstance(ctx context.Context, category, pluginID, instanceID, configJSON string) error
	CreateInstance(ctx context.Context, category, pluginID, instanceID string) (domain.PluginInstallation, error)
	DeletePluginFiles(ctx context.Context, category, pluginID string) error
}
