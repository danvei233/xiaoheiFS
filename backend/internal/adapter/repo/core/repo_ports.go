package repo

import (
	"gorm.io/gorm"

	appports "xiaoheiplay/internal/app/ports"
)

// Domain-scoped repositories wrap shared GormRepo implementation.
type UserRepo struct{ *GormRepo }
type CaptchaRepo struct{ *GormRepo }
type CatalogRepo struct{ *GormRepo }
type SystemImageRepo struct{ *GormRepo }
type CartRepo struct{ *GormRepo }
type OrderRepo struct{ *GormRepo }
type OrderItemRepo struct{ *GormRepo }
type PaymentRepo struct{ *GormRepo }
type VPSRepo struct{ *GormRepo }
type EventRepo struct{ *GormRepo }
type APIKeyRepo struct{ *GormRepo }
type SettingsRepo struct{ *GormRepo }
type AuditRepo struct{ *GormRepo }
type BillingCycleRepo struct{ *GormRepo }
type AutomationLogRepo struct{ *GormRepo }
type ProvisionJobRepo struct{ *GormRepo }
type ResizeTaskRepo struct{ *GormRepo }
type IntegrationLogRepo struct{ *GormRepo }
type PermissionGroupRepo struct{ *GormRepo }
type UserTierRepo struct{ *GormRepo }
type CouponRepo struct{ *GormRepo }
type PasswordResetTokenRepo struct{ *GormRepo }
type PasswordResetTicketRepo struct{ *GormRepo }
type PermissionRepo struct{ *GormRepo }
type CMSCategoryRepo struct{ *GormRepo }
type CMSPostRepo struct{ *GormRepo }
type CMSBlockRepo struct{ *GormRepo }
type UploadRepo struct{ *GormRepo }
type TicketRepo struct{ *GormRepo }
type NotificationRepo struct{ *GormRepo }
type PushTokenRepo struct{ *GormRepo }
type WalletRepo struct{ *GormRepo }
type WalletOrderRepo struct{ *GormRepo }
type ProbeNodeRepo struct{ *GormRepo }
type ProbeEnrollTokenRepo struct{ *GormRepo }
type ProbeStatusEventRepo struct{ *GormRepo }
type ProbeLogSessionRepo struct{ *GormRepo }

func NewUserRepo(gdb *gorm.DB) *UserRepo                 { return &UserRepo{NewGormRepo(gdb)} }
func NewCaptchaRepo(gdb *gorm.DB) *CaptchaRepo           { return &CaptchaRepo{NewGormRepo(gdb)} }
func NewCatalogRepo(gdb *gorm.DB) *CatalogRepo           { return &CatalogRepo{NewGormRepo(gdb)} }
func NewSystemImageRepo(gdb *gorm.DB) *SystemImageRepo   { return &SystemImageRepo{NewGormRepo(gdb)} }
func NewCartRepo(gdb *gorm.DB) *CartRepo                 { return &CartRepo{NewGormRepo(gdb)} }
func NewOrderRepo(gdb *gorm.DB) *OrderRepo               { return &OrderRepo{NewGormRepo(gdb)} }
func NewOrderItemRepo(gdb *gorm.DB) *OrderItemRepo       { return &OrderItemRepo{NewGormRepo(gdb)} }
func NewPaymentRepo(gdb *gorm.DB) *PaymentRepo           { return &PaymentRepo{NewGormRepo(gdb)} }
func NewVPSRepo(gdb *gorm.DB) *VPSRepo                   { return &VPSRepo{NewGormRepo(gdb)} }
func NewEventRepo(gdb *gorm.DB) *EventRepo               { return &EventRepo{NewGormRepo(gdb)} }
func NewAPIKeyRepo(gdb *gorm.DB) *APIKeyRepo             { return &APIKeyRepo{NewGormRepo(gdb)} }
func NewSettingsRepo(gdb *gorm.DB) *SettingsRepo         { return &SettingsRepo{NewGormRepo(gdb)} }
func NewAuditRepo(gdb *gorm.DB) *AuditRepo               { return &AuditRepo{NewGormRepo(gdb)} }
func NewBillingCycleRepo(gdb *gorm.DB) *BillingCycleRepo { return &BillingCycleRepo{NewGormRepo(gdb)} }
func NewAutomationLogRepo(gdb *gorm.DB) *AutomationLogRepo {
	return &AutomationLogRepo{NewGormRepo(gdb)}
}
func NewProvisionJobRepo(gdb *gorm.DB) *ProvisionJobRepo { return &ProvisionJobRepo{NewGormRepo(gdb)} }
func NewResizeTaskRepo(gdb *gorm.DB) *ResizeTaskRepo     { return &ResizeTaskRepo{NewGormRepo(gdb)} }
func NewIntegrationLogRepo(gdb *gorm.DB) *IntegrationLogRepo {
	return &IntegrationLogRepo{NewGormRepo(gdb)}
}
func NewPermissionGroupRepo(gdb *gorm.DB) *PermissionGroupRepo {
	return &PermissionGroupRepo{NewGormRepo(gdb)}
}
func NewUserTierRepo(gdb *gorm.DB) *UserTierRepo { return &UserTierRepo{NewGormRepo(gdb)} }
func NewCouponRepo(gdb *gorm.DB) *CouponRepo     { return &CouponRepo{NewGormRepo(gdb)} }
func NewPasswordResetTokenRepo(gdb *gorm.DB) *PasswordResetTokenRepo {
	return &PasswordResetTokenRepo{NewGormRepo(gdb)}
}
func NewPasswordResetTicketRepo(gdb *gorm.DB) *PasswordResetTicketRepo {
	return &PasswordResetTicketRepo{NewGormRepo(gdb)}
}
func NewPermissionRepo(gdb *gorm.DB) *PermissionRepo     { return &PermissionRepo{NewGormRepo(gdb)} }
func NewCMSCategoryRepo(gdb *gorm.DB) *CMSCategoryRepo   { return &CMSCategoryRepo{NewGormRepo(gdb)} }
func NewCMSPostRepo(gdb *gorm.DB) *CMSPostRepo           { return &CMSPostRepo{NewGormRepo(gdb)} }
func NewCMSBlockRepo(gdb *gorm.DB) *CMSBlockRepo         { return &CMSBlockRepo{NewGormRepo(gdb)} }
func NewUploadRepo(gdb *gorm.DB) *UploadRepo             { return &UploadRepo{NewGormRepo(gdb)} }
func NewTicketRepo(gdb *gorm.DB) *TicketRepo             { return &TicketRepo{NewGormRepo(gdb)} }
func NewNotificationRepo(gdb *gorm.DB) *NotificationRepo { return &NotificationRepo{NewGormRepo(gdb)} }
func NewPushTokenRepo(gdb *gorm.DB) *PushTokenRepo       { return &PushTokenRepo{NewGormRepo(gdb)} }
func NewWalletRepo(gdb *gorm.DB) *WalletRepo             { return &WalletRepo{NewGormRepo(gdb)} }
func NewWalletOrderRepo(gdb *gorm.DB) *WalletOrderRepo   { return &WalletOrderRepo{NewGormRepo(gdb)} }
func NewProbeNodeRepo(gdb *gorm.DB) *ProbeNodeRepo       { return &ProbeNodeRepo{NewGormRepo(gdb)} }
func NewProbeEnrollTokenRepo(gdb *gorm.DB) *ProbeEnrollTokenRepo {
	return &ProbeEnrollTokenRepo{NewGormRepo(gdb)}
}
func NewProbeStatusEventRepo(gdb *gorm.DB) *ProbeStatusEventRepo {
	return &ProbeStatusEventRepo{NewGormRepo(gdb)}
}
func NewProbeLogSessionRepo(gdb *gorm.DB) *ProbeLogSessionRepo {
	return &ProbeLogSessionRepo{NewGormRepo(gdb)}
}

var (
	_ appports.UserRepository                = (*UserRepo)(nil)
	_ appports.CaptchaRepository             = (*CaptchaRepo)(nil)
	_ appports.CatalogRepository             = (*CatalogRepo)(nil)
	_ appports.SystemImageRepository         = (*SystemImageRepo)(nil)
	_ appports.CartRepository                = (*CartRepo)(nil)
	_ appports.OrderRepository               = (*OrderRepo)(nil)
	_ appports.OrderItemRepository           = (*OrderItemRepo)(nil)
	_ appports.PaymentRepository             = (*PaymentRepo)(nil)
	_ appports.VPSRepository                 = (*VPSRepo)(nil)
	_ appports.EventRepository               = (*EventRepo)(nil)
	_ appports.APIKeyRepository              = (*APIKeyRepo)(nil)
	_ appports.SettingsRepository            = (*SettingsRepo)(nil)
	_ appports.AuditRepository               = (*AuditRepo)(nil)
	_ appports.BillingCycleRepository        = (*BillingCycleRepo)(nil)
	_ appports.AutomationLogRepository       = (*AutomationLogRepo)(nil)
	_ appports.ProvisionJobRepository        = (*ProvisionJobRepo)(nil)
	_ appports.ResizeTaskRepository          = (*ResizeTaskRepo)(nil)
	_ appports.IntegrationLogRepository      = (*IntegrationLogRepo)(nil)
	_ appports.PermissionGroupRepository     = (*PermissionGroupRepo)(nil)
	_ appports.UserTierRepository            = (*UserTierRepo)(nil)
	_ appports.CouponRepository              = (*CouponRepo)(nil)
	_ appports.PasswordResetTokenRepository  = (*PasswordResetTokenRepo)(nil)
	_ appports.PasswordResetTicketRepository = (*PasswordResetTicketRepo)(nil)
	_ appports.PermissionRepository          = (*PermissionRepo)(nil)
	_ appports.CMSCategoryRepository         = (*CMSCategoryRepo)(nil)
	_ appports.CMSPostRepository             = (*CMSPostRepo)(nil)
	_ appports.CMSBlockRepository            = (*CMSBlockRepo)(nil)
	_ appports.UploadRepository              = (*UploadRepo)(nil)
	_ appports.TicketRepository              = (*TicketRepo)(nil)
	_ appports.NotificationRepository        = (*NotificationRepo)(nil)
	_ appports.PushTokenRepository           = (*PushTokenRepo)(nil)
	_ appports.WalletRepository              = (*WalletRepo)(nil)
	_ appports.WalletOrderRepository         = (*WalletOrderRepo)(nil)
	_ appports.ProbeNodeRepository           = (*ProbeNodeRepo)(nil)
	_ appports.ProbeEnrollTokenRepository    = (*ProbeEnrollTokenRepo)(nil)
	_ appports.ProbeStatusEventRepository    = (*ProbeStatusEventRepo)(nil)
	_ appports.ProbeLogSessionRepository     = (*ProbeLogSessionRepo)(nil)
)
