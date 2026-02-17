package order

import (
	"context"
	"strconv"
	"strings"
	"time"

	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

type (
	OrderRepository          = appports.OrderRepository
	OrderItemRepository      = appports.OrderItemRepository
	CartRepository           = appports.CartRepository
	CatalogRepository        = appports.CatalogRepository
	SystemImageRepository    = appports.SystemImageRepository
	BillingCycleRepository   = appports.BillingCycleRepository
	VPSRepository            = appports.VPSRepository
	WalletRepository         = appports.WalletRepository
	PaymentRepository        = appports.PaymentRepository
	EventPublisher           = appports.EventPublisher
	AutomationClientResolver = appports.AutomationClientResolver
	RobotNotifier            = appshared.RobotNotifier
	AuditRepository          = appports.AuditRepository
	UserRepository           = appports.UserRepository
	EmailSender              = appports.EmailSender
	SettingsRepository       = appports.SettingsRepository
	AutomationLogRepository  = appports.AutomationLogRepository
	ProvisionJobRepository   = appports.ProvisionJobRepository
	ResizeTaskRepository     = appports.ResizeTaskRepository
	WalletOrderRepository    = appports.WalletOrderRepository

	AutomationClient               = appshared.AutomationClient
	AutomationHostInfo             = appshared.AutomationHostInfo
	AutomationCreateHostResult     = appshared.AutomationCreateHostResult
	AutomationCreateHostRequest    = appshared.AutomationCreateHostRequest
	AutomationHostSimple           = appshared.AutomationHostSimple
	AutomationElasticUpdateRequest = appshared.AutomationElasticUpdateRequest
	AutomationArea                 = appshared.AutomationArea
	AutomationImage                = appshared.AutomationImage
	AutomationLine                 = appshared.AutomationLine
	AutomationProduct              = appshared.AutomationProduct
	AutomationMonitor              = appshared.AutomationMonitor
	AutomationSnapshot             = appshared.AutomationSnapshot
	AutomationBackup               = appshared.AutomationBackup
	AutomationFirewallRule         = appshared.AutomationFirewallRule
	AutomationPortMapping          = appshared.AutomationPortMapping
	AutomationFirewallRuleCreate   = appshared.AutomationFirewallRuleCreate
	AutomationPortMappingCreate    = appshared.AutomationPortMappingCreate
	CartSpec                       = appshared.CartSpec
	OrderFilter                    = appshared.OrderFilter
	RobotOrderPayload              = appshared.RobotOrderPayload
	RobotOrderItem                 = appshared.RobotOrderItem
)

var (
	ErrConflict            = appshared.ErrConflict
	ErrInvalidInput        = appshared.ErrInvalidInput
	ErrInsufficientBalance = appshared.ErrInsufficientBalance
	ErrNoPaymentRequired   = appshared.ErrNoPaymentRequired
	ErrRealNameRequired    = appshared.ErrRealNameRequired
	ErrNotSupported        = appshared.ErrNotSupported
	ErrResizeDisabled      = appshared.ErrResizeDisabled
	ErrResizeInProgress    = appshared.ErrResizeInProgress
	ErrForbidden           = appshared.ErrForbidden
	ErrNotFound            = appshared.ErrNotFound
	ErrResizeSamePlan      = domain.ErrResizeSamePlan
)

func WithAutomationLogContext(ctx context.Context, orderID, orderItemID int64) context.Context {
	return appshared.WithAutomationLogContext(ctx, orderID, orderItemID)
}

type emergencyRenewPolicy struct {
	Enabled       bool
	WindowDays    int
	RenewDays     int
	IntervalHours int
}

func loadEmergencyRenewPolicy(ctx context.Context, settings SettingsRepository) emergencyRenewPolicy {
	policy := emergencyRenewPolicy{
		Enabled:       true,
		WindowDays:    7,
		RenewDays:     1,
		IntervalHours: 720,
	}
	if v, ok := getSettingBool(ctx, settings, "emergency_renew_enabled"); ok {
		policy.Enabled = v
	}
	if v, ok := getSettingInt(ctx, settings, "emergency_renew_window_days"); ok {
		policy.WindowDays = v
	}
	if v, ok := getSettingInt(ctx, settings, "emergency_renew_days"); ok {
		policy.RenewDays = v
	}
	if v, ok := getSettingInt(ctx, settings, "emergency_renew_interval_hours"); ok {
		policy.IntervalHours = v
	}
	if policy.WindowDays < 0 {
		policy.WindowDays = 0
	}
	if policy.RenewDays <= 0 {
		policy.RenewDays = 1
	}
	if policy.IntervalHours <= 0 {
		policy.IntervalHours = 24
	}
	return policy
}

func emergencyRenewInWindow(now time.Time, expireAt *time.Time, windowDays int) bool {
	if expireAt == nil {
		return false
	}
	if now.After(*expireAt) {
		return false
	}
	if windowDays <= 0 {
		return true
	}
	windowStart := expireAt.Add(-time.Duration(windowDays) * 24 * time.Hour)
	return !now.Before(windowStart)
}

func getSettingInt(ctx context.Context, repo SettingsRepository, key string) (int, bool) {
	if repo == nil {
		return 0, false
	}
	setting, err := repo.GetSetting(ctx, key)
	if err != nil {
		return 0, false
	}
	val, err := strconv.Atoi(strings.TrimSpace(setting.ValueJSON))
	if err != nil {
		return 0, false
	}
	return val, true
}

func getSettingBool(ctx context.Context, repo SettingsRepository, key string) (bool, bool) {
	if repo == nil {
		return false, false
	}
	setting, err := repo.GetSetting(ctx, key)
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

func MapAutomationState(state int) domain.VPSStatus {
	switch state {
	case 0, 1, 13:
		return domain.VPSStatusProvisioning
	case 2:
		return domain.VPSStatusRunning
	case 3:
		return domain.VPSStatusStopped
	case 4:
		return domain.VPSStatusReinstalling
	case 5:
		return domain.VPSStatusReinstallFailed
	case 10:
		return domain.VPSStatusLocked
	default:
		return domain.VPSStatusUnknown
	}
}
