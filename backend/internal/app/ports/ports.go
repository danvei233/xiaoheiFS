package ports

import (
	"context"
	"time"

	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user *domain.User) error
	GetUserByID(ctx context.Context, id int64) (domain.User, error)
	GetUserByUsernameOrEmail(ctx context.Context, usernameOrEmail string) (domain.User, error)
	GetUserByPhone(ctx context.Context, phone string) (domain.User, error)
	ListUsers(ctx context.Context, limit, offset int) ([]domain.User, int, error)
	ListUsersByRoleStatus(ctx context.Context, role string, status string, limit, offset int) ([]domain.User, int, error)
	UpdateUserStatus(ctx context.Context, id int64, status domain.UserStatus) error
	UpdateUser(ctx context.Context, user domain.User) error
	UpdateUserPassword(ctx context.Context, id int64, passwordHash string) error
	GetMinUserIDByRole(ctx context.Context, role string) (int64, error)
}

type CaptchaRepository interface {
	CreateCaptcha(ctx context.Context, captcha domain.Captcha) error
	GetCaptcha(ctx context.Context, id string) (domain.Captcha, error)
	DeleteCaptcha(ctx context.Context, id string) error
}

type VerificationCodeRepository interface {
	CreateVerificationCode(ctx context.Context, code domain.VerificationCode) error
	GetLatestVerificationCode(ctx context.Context, channel, receiver, purpose string) (domain.VerificationCode, error)
	DeleteVerificationCodes(ctx context.Context, channel, receiver, purpose string) error
}

type CatalogRepository interface {
	ListRegions(ctx context.Context) ([]domain.Region, error)
	ListPlanGroups(ctx context.Context) ([]domain.PlanGroup, error)
	ListPackages(ctx context.Context) ([]domain.Package, error)
	GetPackage(ctx context.Context, id int64) (domain.Package, error)
	GetPlanGroup(ctx context.Context, id int64) (domain.PlanGroup, error)
	GetRegion(ctx context.Context, id int64) (domain.Region, error)
	CreateRegion(ctx context.Context, region *domain.Region) error
	UpdateRegion(ctx context.Context, region domain.Region) error
	DeleteRegion(ctx context.Context, id int64) error
	CreatePlanGroup(ctx context.Context, plan *domain.PlanGroup) error
	UpdatePlanGroup(ctx context.Context, plan domain.PlanGroup) error
	DeletePlanGroup(ctx context.Context, id int64) error
	CreatePackage(ctx context.Context, pkg *domain.Package) error
	UpdatePackage(ctx context.Context, pkg domain.Package) error
	DeletePackage(ctx context.Context, id int64) error
}

type GoodsTypeRepository interface {
	ListGoodsTypes(ctx context.Context) ([]domain.GoodsType, error)
	GetGoodsType(ctx context.Context, id int64) (domain.GoodsType, error)
	CreateGoodsType(ctx context.Context, gt *domain.GoodsType) error
	UpdateGoodsType(ctx context.Context, gt domain.GoodsType) error
	DeleteGoodsType(ctx context.Context, id int64) error
}

type SystemImageRepository interface {
	ListSystemImages(ctx context.Context, lineID int64) ([]domain.SystemImage, error)
	ListAllSystemImages(ctx context.Context) ([]domain.SystemImage, error)
	GetSystemImage(ctx context.Context, id int64) (domain.SystemImage, error)
	CreateSystemImage(ctx context.Context, img *domain.SystemImage) error
	UpdateSystemImage(ctx context.Context, img domain.SystemImage) error
	DeleteSystemImage(ctx context.Context, id int64) error
	SetLineSystemImages(ctx context.Context, lineID int64, systemImageIDs []int64) error
}

type CartRepository interface {
	ListCartItems(ctx context.Context, userID int64) ([]domain.CartItem, error)
	AddCartItem(ctx context.Context, item *domain.CartItem) error
	UpdateCartItem(ctx context.Context, item domain.CartItem) error
	DeleteCartItem(ctx context.Context, id int64, userID int64) error
	ClearCart(ctx context.Context, userID int64) error
}

type OrderRepository interface {
	CreateOrder(ctx context.Context, order *domain.Order) error
	GetOrder(ctx context.Context, id int64) (domain.Order, error)
	GetOrderByNo(ctx context.Context, orderNo string) (domain.Order, error)
	GetOrderByIdempotencyKey(ctx context.Context, userID int64, key string) (domain.Order, error)
	UpdateOrderStatus(ctx context.Context, id int64, status domain.OrderStatus) error
	UpdateOrderMeta(ctx context.Context, order domain.Order) error
	ListOrders(ctx context.Context, filter appshared.OrderFilter, limit, offset int) ([]domain.Order, int, error)
	DeleteOrder(ctx context.Context, id int64) error
}

type OrderItemRepository interface {
	CreateOrderItems(ctx context.Context, items []domain.OrderItem) error
	ListOrderItems(ctx context.Context, orderID int64) ([]domain.OrderItem, error)
	GetOrderItem(ctx context.Context, id int64) (domain.OrderItem, error)
	UpdateOrderItemStatus(ctx context.Context, id int64, status domain.OrderItemStatus) error
	UpdateOrderItemAutomation(ctx context.Context, id int64, automationID string) error
	HasPendingRenewOrder(ctx context.Context, userID, vpsID int64) (bool, error)
	HasPendingResizeOrder(ctx context.Context, userID, vpsID int64) (bool, error)
	HasPendingRefundOrder(ctx context.Context, userID, vpsID int64) (bool, error)
}

type PaymentRepository interface {
	CreatePayment(ctx context.Context, payment *domain.OrderPayment) error
	ListPaymentsByOrder(ctx context.Context, orderID int64) ([]domain.OrderPayment, error)
	GetPaymentByTradeNo(ctx context.Context, tradeNo string) (domain.OrderPayment, error)
	GetPaymentByIdempotencyKey(ctx context.Context, orderID int64, key string) (domain.OrderPayment, error)
	UpdatePaymentTradeNo(ctx context.Context, id int64, tradeNo string) error
	UpdatePaymentStatus(ctx context.Context, id int64, status domain.PaymentStatus, reviewedBy *int64, reason string) error
	ListPayments(ctx context.Context, filter appshared.PaymentFilter, limit, offset int) ([]domain.OrderPayment, int, error)
}

type VPSRepository interface {
	CreateInstance(ctx context.Context, inst *domain.VPSInstance) error
	GetInstance(ctx context.Context, id int64) (domain.VPSInstance, error)
	GetInstanceByOrderItem(ctx context.Context, orderItemID int64) (domain.VPSInstance, error)
	ListInstancesByUser(ctx context.Context, userID int64) ([]domain.VPSInstance, error)
	ListInstances(ctx context.Context, limit, offset int) ([]domain.VPSInstance, int, error)
	ListInstancesExpiring(ctx context.Context, before time.Time) ([]domain.VPSInstance, error)
	DeleteInstance(ctx context.Context, id int64) error
	UpdateInstanceStatus(ctx context.Context, id int64, status domain.VPSStatus, automationState int) error
	UpdateInstanceAdminStatus(ctx context.Context, id int64, status domain.VPSAdminStatus) error
	UpdateInstanceExpireAt(ctx context.Context, id int64, expireAt time.Time) error
	UpdateInstancePanelCache(ctx context.Context, id int64, panelURL string) error
	UpdateInstanceSpec(ctx context.Context, id int64, specJSON string) error
	UpdateInstanceAccessInfo(ctx context.Context, id int64, accessJSON string) error
	UpdateInstanceEmergencyRenewAt(ctx context.Context, id int64, at time.Time) error
	UpdateInstanceLocal(ctx context.Context, inst domain.VPSInstance) error
}

type EventRepository interface {
	AppendEvent(ctx context.Context, orderID int64, eventType string, dataJSON string) (domain.OrderEvent, error)
	ListEventsAfter(ctx context.Context, orderID int64, afterSeq int64, limit int) ([]domain.OrderEvent, error)
}

type EventPublisher interface {
	Publish(ctx context.Context, orderID int64, eventType string, payload any) (domain.OrderEvent, error)
}

type APIKeyRepository interface {
	CreateAPIKey(ctx context.Context, key *domain.APIKey) error
	GetAPIKeyByHash(ctx context.Context, keyHash string) (domain.APIKey, error)
	ListAPIKeys(ctx context.Context, limit, offset int) ([]domain.APIKey, int, error)
	UpdateAPIKeyStatus(ctx context.Context, id int64, status domain.APIKeyStatus) error
	TouchAPIKey(ctx context.Context, id int64) error
}

type SettingsRepository interface {
	GetSetting(ctx context.Context, key string) (domain.Setting, error)
	UpsertSetting(ctx context.Context, setting domain.Setting) error
	ListSettings(ctx context.Context) ([]domain.Setting, error)
	ListEmailTemplates(ctx context.Context) ([]domain.EmailTemplate, error)
	GetEmailTemplate(ctx context.Context, id int64) (domain.EmailTemplate, error)
	UpsertEmailTemplate(ctx context.Context, tmpl *domain.EmailTemplate) error
	DeleteEmailTemplate(ctx context.Context, id int64) error
}

type PluginInstallationRepository interface {
	UpsertPluginInstallation(ctx context.Context, inst *domain.PluginInstallation) error
	GetPluginInstallation(ctx context.Context, category, pluginID, instanceID string) (domain.PluginInstallation, error)
	ListPluginInstallations(ctx context.Context) ([]domain.PluginInstallation, error)
	DeletePluginInstallation(ctx context.Context, category, pluginID, instanceID string) error
}

type PluginPaymentMethodRepository interface {
	ListPluginPaymentMethods(ctx context.Context, category, pluginID, instanceID string) ([]domain.PluginPaymentMethod, error)
	UpsertPluginPaymentMethod(ctx context.Context, m *domain.PluginPaymentMethod) error
	DeletePluginPaymentMethod(ctx context.Context, category, pluginID, instanceID, method string) error
}

type AuditRepository interface {
	AddAuditLog(ctx context.Context, log domain.AdminAuditLog) error
	ListAuditLogs(ctx context.Context, limit, offset int) ([]domain.AdminAuditLog, int, error)
}

type BillingCycleRepository interface {
	ListBillingCycles(ctx context.Context) ([]domain.BillingCycle, error)
	GetBillingCycle(ctx context.Context, id int64) (domain.BillingCycle, error)
	CreateBillingCycle(ctx context.Context, cycle *domain.BillingCycle) error
	UpdateBillingCycle(ctx context.Context, cycle domain.BillingCycle) error
	DeleteBillingCycle(ctx context.Context, id int64) error
}

type AutomationLogRepository interface {
	CreateAutomationLog(ctx context.Context, log *domain.AutomationLog) error
	ListAutomationLogs(ctx context.Context, orderID int64, limit, offset int) ([]domain.AutomationLog, int, error)
	PurgeAutomationLogs(ctx context.Context, before time.Time) error
}

type ProvisionJobRepository interface {
	CreateOrUpdateProvisionJob(ctx context.Context, job *domain.ProvisionJob) error
	ListDueProvisionJobs(ctx context.Context, limit int) ([]domain.ProvisionJob, error)
	UpdateProvisionJob(ctx context.Context, job domain.ProvisionJob) error
}

type ResizeTaskRepository interface {
	CreateResizeTask(ctx context.Context, task *domain.ResizeTask) error
	GetResizeTask(ctx context.Context, id int64) (domain.ResizeTask, error)
	UpdateResizeTask(ctx context.Context, task domain.ResizeTask) error
	ListDueResizeTasks(ctx context.Context, limit int) ([]domain.ResizeTask, error)
	HasPendingResizeTask(ctx context.Context, vpsID int64) (bool, error)
}

type ScheduledTaskRunRepository interface {
	CreateTaskRun(ctx context.Context, run *domain.ScheduledTaskRun) error
	UpdateTaskRun(ctx context.Context, run domain.ScheduledTaskRun) error
	ListTaskRuns(ctx context.Context, key string, limit int) ([]domain.ScheduledTaskRun, error)
}

type IntegrationLogRepository interface {
	CreateSyncLog(ctx context.Context, log *domain.IntegrationSyncLog) error
	ListSyncLogs(ctx context.Context, target string, limit, offset int) ([]domain.IntegrationSyncLog, int, error)
}

type PermissionGroupRepository interface {
	ListPermissionGroups(ctx context.Context) ([]domain.PermissionGroup, error)
	GetPermissionGroup(ctx context.Context, id int64) (domain.PermissionGroup, error)
	CreatePermissionGroup(ctx context.Context, group *domain.PermissionGroup) error
	UpdatePermissionGroup(ctx context.Context, group domain.PermissionGroup) error
	DeletePermissionGroup(ctx context.Context, id int64) error
}

type PasswordResetTokenRepository interface {
	CreatePasswordResetToken(ctx context.Context, token *domain.PasswordResetToken) error
	GetPasswordResetToken(ctx context.Context, token string) (domain.PasswordResetToken, error)
	MarkPasswordResetTokenUsed(ctx context.Context, tokenID int64) error
	DeleteExpiredTokens(ctx context.Context) error
}

type PasswordResetTicketRepository interface {
	CreatePasswordResetTicket(ctx context.Context, ticket *domain.PasswordResetTicket) error
	GetPasswordResetTicket(ctx context.Context, token string) (domain.PasswordResetTicket, error)
	MarkPasswordResetTicketUsed(ctx context.Context, ticketID int64) error
	DeleteExpiredPasswordResetTickets(ctx context.Context) error
}

type PermissionRepository interface {
	ListPermissions(ctx context.Context) ([]domain.Permission, error)
	GetPermissionByCode(ctx context.Context, code string) (domain.Permission, error)
	UpsertPermission(ctx context.Context, perm *domain.Permission) error
	UpdatePermissionName(ctx context.Context, code string, name string) error
	RegisterPermissions(ctx context.Context, perms []domain.PermissionDefinition) error
}

type CMSCategoryRepository interface {
	ListCMSCategories(ctx context.Context, lang string, includeHidden bool) ([]domain.CMSCategory, error)
	GetCMSCategory(ctx context.Context, id int64) (domain.CMSCategory, error)
	GetCMSCategoryByKey(ctx context.Context, key, lang string) (domain.CMSCategory, error)
	CreateCMSCategory(ctx context.Context, category *domain.CMSCategory) error
	UpdateCMSCategory(ctx context.Context, category domain.CMSCategory) error
	DeleteCMSCategory(ctx context.Context, id int64) error
}

type CMSPostRepository interface {
	ListCMSPosts(ctx context.Context, filter appshared.CMSPostFilter) ([]domain.CMSPost, int, error)
	GetCMSPost(ctx context.Context, id int64) (domain.CMSPost, error)
	GetCMSPostBySlug(ctx context.Context, slug string) (domain.CMSPost, error)
	CreateCMSPost(ctx context.Context, post *domain.CMSPost) error
	UpdateCMSPost(ctx context.Context, post domain.CMSPost) error
	DeleteCMSPost(ctx context.Context, id int64) error
}

type CMSBlockRepository interface {
	ListCMSBlocks(ctx context.Context, page, lang string, includeHidden bool) ([]domain.CMSBlock, error)
	GetCMSBlock(ctx context.Context, id int64) (domain.CMSBlock, error)
	CreateCMSBlock(ctx context.Context, block *domain.CMSBlock) error
	UpdateCMSBlock(ctx context.Context, block domain.CMSBlock) error
	DeleteCMSBlock(ctx context.Context, id int64) error
}

type UploadRepository interface {
	CreateUpload(ctx context.Context, upload *domain.Upload) error
	ListUploads(ctx context.Context, limit, offset int) ([]domain.Upload, int, error)
}

type TicketRepository interface {
	ListTickets(ctx context.Context, filter appshared.TicketFilter) ([]domain.Ticket, int, error)
	GetTicket(ctx context.Context, id int64) (domain.Ticket, error)
	CreateTicketWithDetails(ctx context.Context, ticket *domain.Ticket, message *domain.TicketMessage, resources []domain.TicketResource) error
	AddTicketMessage(ctx context.Context, message *domain.TicketMessage) error
	ListTicketMessages(ctx context.Context, ticketID int64) ([]domain.TicketMessage, error)
	ListTicketResources(ctx context.Context, ticketID int64) ([]domain.TicketResource, error)
	UpdateTicket(ctx context.Context, ticket domain.Ticket) error
	DeleteTicket(ctx context.Context, id int64) error
}

type NotificationRepository interface {
	CreateNotification(ctx context.Context, notification *domain.Notification) error
	ListNotifications(ctx context.Context, filter appshared.NotificationFilter) ([]domain.Notification, int, error)
	CountUnread(ctx context.Context, userID int64) (int, error)
	MarkNotificationRead(ctx context.Context, userID, notificationID int64) error
	MarkAllRead(ctx context.Context, userID int64) error
}

type PushTokenRepository interface {
	UpsertPushToken(ctx context.Context, token *domain.PushToken) error
	DeletePushToken(ctx context.Context, userID int64, token string) error
	ListPushTokensByUserIDs(ctx context.Context, userIDs []int64) ([]domain.PushToken, error)
}

type PushSender interface {
	Send(ctx context.Context, config appshared.PushConfig, tokens []string, payload appshared.PushPayload) error
}

type SystemInfoProvider interface {
	Status(ctx context.Context) (appshared.ServerStatus, error)
}

type WalletRepository interface {
	GetWallet(ctx context.Context, userID int64) (domain.Wallet, error)
	UpsertWallet(ctx context.Context, wallet *domain.Wallet) error
	AddWalletTransaction(ctx context.Context, tx *domain.WalletTransaction) error
	ListWalletTransactions(ctx context.Context, userID int64, limit, offset int) ([]domain.WalletTransaction, int, error)
	AdjustWalletBalance(ctx context.Context, userID int64, amount int64, txType, refType string, refID int64, note string) (domain.Wallet, error)
	HasWalletTransaction(ctx context.Context, userID int64, refType string, refID int64) (bool, error)
}

type WalletOrderRepository interface {
	CreateWalletOrder(ctx context.Context, order *domain.WalletOrder) error
	GetWalletOrder(ctx context.Context, id int64) (domain.WalletOrder, error)
	ListWalletOrders(ctx context.Context, userID int64, limit, offset int) ([]domain.WalletOrder, int, error)
	ListAllWalletOrders(ctx context.Context, status string, limit, offset int) ([]domain.WalletOrder, int, error)
	UpdateWalletOrderStatus(ctx context.Context, id int64, status domain.WalletOrderStatus, reviewedBy *int64, reason string) error
}

type ProbeNodeRepository interface {
	CreateProbeNode(ctx context.Context, node *domain.ProbeNode) error
	GetProbeNode(ctx context.Context, id int64) (domain.ProbeNode, error)
	GetProbeNodeByAgentID(ctx context.Context, agentID string) (domain.ProbeNode, error)
	ListProbeNodes(ctx context.Context, filter appshared.ProbeNodeFilter, limit, offset int) ([]domain.ProbeNode, int, error)
	UpdateProbeNode(ctx context.Context, node domain.ProbeNode) error
	DeleteProbeNode(ctx context.Context, id int64) error
	UpdateProbeNodeStatus(ctx context.Context, id int64, status domain.ProbeStatus, reason string, at time.Time) error
	UpdateProbeNodeHeartbeat(ctx context.Context, id int64, at time.Time) error
	UpdateProbeNodeSnapshot(ctx context.Context, id int64, at time.Time, snapshotJSON string, osType string) error
}

type ProbeEnrollTokenRepository interface {
	CreateProbeEnrollToken(ctx context.Context, token *domain.ProbeEnrollToken) error
	GetValidProbeEnrollTokenByHash(ctx context.Context, tokenHash string, now time.Time) (domain.ProbeEnrollToken, error)
	MarkProbeEnrollTokenUsed(ctx context.Context, id int64, usedAt time.Time) error
	DeleteProbeEnrollTokensByProbe(ctx context.Context, probeID int64) error
}

type ProbeStatusEventRepository interface {
	CreateProbeStatusEvent(ctx context.Context, ev *domain.ProbeStatusEvent) error
	ListProbeStatusEvents(ctx context.Context, probeID int64, from, to time.Time) ([]domain.ProbeStatusEvent, error)
	GetLatestProbeStatusEventBefore(ctx context.Context, probeID int64, before time.Time) (domain.ProbeStatusEvent, error)
	DeleteProbeStatusEventsBefore(ctx context.Context, before time.Time) error
}

type ProbeLogSessionRepository interface {
	CreateProbeLogSession(ctx context.Context, session *domain.ProbeLogSession) error
	GetProbeLogSession(ctx context.Context, id int64) (domain.ProbeLogSession, error)
	UpdateProbeLogSession(ctx context.Context, session domain.ProbeLogSession) error
}

type PaymentProviderRegistry interface {
	ListProviders(ctx context.Context, includeDisabled bool) ([]appshared.PaymentProvider, error)
	GetProvider(ctx context.Context, key string) (appshared.PaymentProvider, error)
	GetProviderConfig(ctx context.Context, key string) (string, bool, error)
	UpdateProviderConfig(ctx context.Context, key string, enabled bool, configJSON string) error
}

type OrderApprover interface {
	ApproveOrder(ctx context.Context, adminID int64, orderID int64) error
}

type AutomationClientResolver interface {
	ClientForGoodsType(ctx context.Context, goodsTypeID int64) (appshared.AutomationClient, error)
}

type EmailSender interface {
	Send(ctx context.Context, to string, subject string, body string) error
}

type RealNameRepository interface {
	CreateRealNameVerification(ctx context.Context, record *domain.RealNameVerification) error
	GetLatestRealNameVerification(ctx context.Context, userID int64) (domain.RealNameVerification, error)
	ListRealNameVerifications(ctx context.Context, userID *int64, limit, offset int) ([]domain.RealNameVerification, int, error)
	UpdateRealNameStatus(ctx context.Context, id int64, status string, reason string, verifiedAt *time.Time) error
}

type SMSSender interface {
	Send(ctx context.Context, pluginID, instanceID string, msg appshared.SMSMessage) (appshared.SMSDelivery, error)
}
