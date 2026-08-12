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
	ListUsersFiltered(ctx context.Context, keyword, status string, limit, offset int) ([]domain.User, int, error)
	ListUsersByRoleStatus(ctx context.Context, role string, status string, limit, offset int) ([]domain.User, int, error)
	UpdateUserStatus(ctx context.Context, id int64, status domain.UserStatus) error
	// UpdateAdminStatusSafe atomically updates an admin's status while
	// guaranteeing the system keeps at least one active admin; concurrent
	// disables serialize on the active-admin rows.
	UpdateAdminStatusSafe(ctx context.Context, id int64, status domain.UserStatus) error
	UpdateUser(ctx context.Context, user domain.User) error
	UpdateUserPassword(ctx context.Context, id int64, passwordHash string) error
	GetMinUserIDByRole(ctx context.Context, role string) (int64, error)
	CountUsersByPermissionGroup(ctx context.Context, groupID int64) (int64, error)
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
	SetRegionActive(ctx context.Context, id int64, active bool) error
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
	// ListSystemImagesByLineIDs returns the enabled system images linked to any
	// of the given lines in a single batch query, each image once. It is the
	// N+1-free replacement for calling ListSystemImages per line.
	ListSystemImagesByLineIDs(ctx context.Context, lineIDs []int64) ([]domain.SystemImage, error)
	ListAllSystemImages(ctx context.Context) ([]domain.SystemImage, error)
	GetSystemImage(ctx context.Context, id int64) (domain.SystemImage, error)
	CreateSystemImage(ctx context.Context, img *domain.SystemImage) error
	UpdateSystemImage(ctx context.Context, img domain.SystemImage) error
	DeleteSystemImage(ctx context.Context, id int64) error
	SetLineSystemImages(ctx context.Context, lineID int64, systemImageIDs []int64) error
	ListImageLineNames(ctx context.Context, imageIDs []int64) (map[int64][]string, error)
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
	ApproveOrderAtomic(ctx context.Context, order domain.Order, itemIDs []int64, tasks []*domain.ResizeTask, currentStatuses []domain.OrderStatus) (bool, error)
	RejectOrderAtomic(ctx context.Context, order domain.Order, adminID int64, currentStatuses []domain.OrderStatus) (bool, error)
	CancelOrderAtomic(ctx context.Context, order domain.Order, currentStatuses []domain.OrderStatus) (bool, error)
	UpdateOrderStatusIfCurrent(ctx context.Context, id int64, status domain.OrderStatus, currentStatuses []domain.OrderStatus) (bool, error)
	ListOrders(ctx context.Context, filter appshared.OrderFilter, limit, offset int) ([]domain.Order, int, error)
	DeleteOrder(ctx context.Context, id int64) error
}

type OrderItemRepository interface {
	CreateOrderItems(ctx context.Context, items []domain.OrderItem) error
	ListOrderItems(ctx context.Context, orderID int64) ([]domain.OrderItem, error)
	GetOrderItem(ctx context.Context, id int64) (domain.OrderItem, error)
	UpdateOrderItemStatus(ctx context.Context, id int64, status domain.OrderItemStatus) error
	ClaimOrderItemForProvision(ctx context.Context, id int64) (bool, error)
	UpdateOrderItemAutomation(ctx context.Context, id int64, automationID string) error
	HasPendingRenewOrder(ctx context.Context, userID, vpsID int64) (bool, error)
	HasPendingResizeOrder(ctx context.Context, userID, vpsID int64) (bool, error)
	HasPendingRefundOrder(ctx context.Context, userID, vpsID int64) (bool, error)
}

type PaymentRepository interface {
	CreatePayment(ctx context.Context, payment *domain.OrderPayment) error
	PayOrderWithBalanceAtomic(ctx context.Context, order domain.Order, payment *domain.OrderPayment) (domain.Wallet, bool, error)
	ListPaymentsByOrder(ctx context.Context, orderID int64) ([]domain.OrderPayment, error)
	GetPaymentByTradeNo(ctx context.Context, tradeNo string) (domain.OrderPayment, error)
	GetPaymentByIdempotencyKey(ctx context.Context, orderID int64, key string) (domain.OrderPayment, error)
	UpdatePaymentTradeNo(ctx context.Context, id int64, tradeNo string) error
	UpdatePaymentStatus(ctx context.Context, id int64, status domain.PaymentStatus, reviewedBy *int64, reason string) error
	UpdatePaymentStatusIfCurrent(ctx context.Context, id int64, status domain.PaymentStatus, currentStatuses []domain.PaymentStatus, reviewedBy *int64, reason string) (bool, error)
	ListPayments(ctx context.Context, filter appshared.PaymentFilter, limit, offset int) ([]domain.OrderPayment, int, error)
}

type RevenueAnalyticsRepository interface {
	ListRevenueAnalyticsRows(ctx context.Context, fromAt, toAt time.Time) ([]RevenueAnalyticsRow, error)
}

type RevenueAnalyticsRow struct {
	PaymentID    int64
	OrderID      int64
	OrderNo      string
	UserID       int64
	Amount       int64
	PaidAt       time.Time
	GoodsTypeID  int64
	RegionID     int64
	LineID       int64
	PackageID    int64
	DimensionID  int64
	DimensionStr string
}

// VPSInstanceFilter is the server-side filter for the admin VPS list
// (web_refresh_perf F4): previously the frontend pulled the whole table and
// filtered client-side; the filter is pushed to the backend so a search/status
// switch returns only the requested page.
type VPSInstanceFilter struct {
	// AdminStatus corresponds to admin_status (normal/abuse/fraud).
	AdminStatus string
	// Keyword fuzzy-matches id / user_id / name / region / package_name / automation_instance_id.
	Keyword string
}

// VPSUserInstanceFilter is the bounded filter set exposed by the user VPS
// list. Keeping it separate from the admin filter prevents cross-tenant query
// reuse.
type VPSUserInstanceFilter struct {
	Status       string
	Keyword      string
	Region       string
	ExpireBefore *time.Time
}

type VPSRepository interface {
	CreateInstance(ctx context.Context, inst *domain.VPSInstance) error
	GetInstance(ctx context.Context, id int64) (domain.VPSInstance, error)
	GetInstanceByOrderItem(ctx context.Context, orderItemID int64) (domain.VPSInstance, error)
	ListInstancesByUser(ctx context.Context, userID int64) ([]domain.VPSInstance, error)
	ListInstancesByUserFiltered(ctx context.Context, userID int64, filter VPSUserInstanceFilter, limit, offset int) ([]domain.VPSInstance, int, error)
	ListInstances(ctx context.Context, limit, offset int) ([]domain.VPSInstance, int, error)
	ListInstancesFiltered(ctx context.Context, filter VPSInstanceFilter, limit, offset int) ([]domain.VPSInstance, int, error)
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
	CountEventsByOrder(ctx context.Context, orderID int64) (int64, error)
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
	CountAPIKeysByPermissionGroup(ctx context.Context, groupID int64) (int64, error)
}

type UserAPIKeyRepository interface {
	CreateUserAPIKey(ctx context.Context, key *domain.UserAPIKey) error
	GetUserAPIKeyByAKID(ctx context.Context, akid string) (domain.UserAPIKey, error)
	GetUserByID(ctx context.Context, id int64) (domain.User, error)
	ListUserAPIKeys(ctx context.Context, userID int64, limit, offset int) ([]domain.UserAPIKey, int, error)
	UpdateUserAPIKeyStatus(ctx context.Context, userID, id int64, status domain.APIKeyStatus) error
	DeleteUserAPIKey(ctx context.Context, userID, id int64) error
	TouchUserAPIKey(ctx context.Context, id int64) error
	// UpdateUserAPIKeySecrets upgrades a legacy key to encrypted-at-rest form.
	UpdateUserAPIKeySecrets(ctx context.Context, id int64, keyHash, secretEnc string) error
	// SaveUserAPIKeyNonce durably records a signed-request nonce; the boolean
	// reports whether this call won the unique insert (multi-instance replay
	// protection).
	SaveUserAPIKeyNonce(ctx context.Context, akid, nonce string, expiresAt time.Time) (bool, error)
	// CountUserAPIKeyNonces returns how many signed requests a key issued since
	// the given time (durable per-key request budget).
	CountUserAPIKeyNonces(ctx context.Context, akid string, since time.Time) (int, error)
	DeleteExpiredUserAPIKeyNonces(ctx context.Context) error
}

type SettingsRepository interface {
	GetSetting(ctx context.Context, key string) (domain.Setting, error)
	UpsertSetting(ctx context.Context, setting domain.Setting) error
	// UpdateSettingAtomic applies mutate to the stored value of a settings key
	// inside a transaction with a row lock, so concurrent read-modify-write of a
	// shared document (e.g. capability policy JSON) across instances does not
	// last-write-wins (P2-8).
	UpdateSettingAtomic(ctx context.Context, key string, mutate func(current string) (string, error)) error
	// UpsertSettings persists a batch of settings atomically (all or nothing).
	// The admin settings batch endpoint uses it so a failure on one key does not
	// leave earlier keys half-written.
	UpsertSettings(ctx context.Context, settings []domain.Setting) error
	ListSettings(ctx context.Context) ([]domain.Setting, error)
	ListEmailTemplates(ctx context.Context) ([]domain.EmailTemplate, error)
	GetEmailTemplate(ctx context.Context, id int64) (domain.EmailTemplate, error)
	UpsertEmailTemplate(ctx context.Context, tmpl *domain.EmailTemplate) error
	DeleteEmailTemplate(ctx context.Context, id int64) error
}

// CapabilityPolicyRepository persists each policy as an independently
// addressable row. Updating one package or goods type must not rewrite a shared
// JSON document and lose a concurrent update from another application instance.
type CapabilityPolicyRepository interface {
	ListCapabilityPolicies(ctx context.Context, scope domain.CapabilityPolicyScope) (map[int64]domain.CapabilityPolicy, error)
	GetCapabilityPolicy(ctx context.Context, scope domain.CapabilityPolicyScope, itemID int64) (domain.CapabilityPolicy, error)
	UpsertCapabilityPolicy(ctx context.Context, scope domain.CapabilityPolicyScope, itemID int64, policy domain.CapabilityPolicy) error
}

// RateLimitRepository is the DB-backed atomic window budget used by
// cost-sensitive endpoints (SMS/SMTP test sends). Being DB-backed it is
// consistent across instances and restarts, unlike a process map.
type RateLimitRepository interface {
	AllowRateLimit(ctx context.Context, bucket string, max int, window time.Duration) (bool, error)
	CleanupRateLimits(ctx context.Context, before time.Time) (int64, error)
}

// RefreshTokenUseRepository is the durable single-use ledger for rotating
// refresh-token families. Only token digests cross this repository boundary.
type RefreshTokenUseRepository interface {
	RotateRefreshToken(ctx context.Context, currentHash, nextHash, familyID string, currentExpiresAt, nextExpiresAt time.Time) (rotated bool, familyRevoked bool, err error)
	RevokeRefreshTokenFamily(ctx context.Context, familyID, tombstoneHash string, expiresAt time.Time) error
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
	ListAuditLogs(ctx context.Context, filter appshared.AuditLogFilter, limit, offset int) ([]domain.AdminAuditLog, int, error)
}

type BillingCycleRepository interface {
	ListBillingCycles(ctx context.Context) ([]domain.BillingCycle, error)
	GetBillingCycle(ctx context.Context, id int64) (domain.BillingCycle, error)
	CreateBillingCycle(ctx context.Context, cycle *domain.BillingCycle) error
	UpdateBillingCycle(ctx context.Context, cycle domain.BillingCycle) error
	DeleteBillingCycle(ctx context.Context, id int64) error
}

// CatalogBulkDeleteRepository owns atomic catalog batch deletion. All IDs in
// a request are processed in one database transaction; any validation or
// reference conflict rolls the entire batch back.
type CatalogBulkDeleteRepository interface {
	BulkDeleteRegions(ctx context.Context, ids []int64) error
	BulkDeletePlanGroups(ctx context.Context, ids []int64) error
	BulkDeletePackages(ctx context.Context, ids []int64) error
	BulkDeleteBillingCycles(ctx context.Context, ids []int64) error
	BulkDeleteSystemImages(ctx context.Context, ids []int64) error
}

type AutomationLogRepository interface {
	CreateAutomationLog(ctx context.Context, log *domain.AutomationLog) error
	ListAutomationLogs(ctx context.Context, orderID int64, limit, offset int) ([]domain.AutomationLog, int, error)
	PurgeAutomationLogs(ctx context.Context, before time.Time) error
}

type ProvisionJobRepository interface {
	CreateOrUpdateProvisionJob(ctx context.Context, job *domain.ProvisionJob) error
	GetProvisionJobByOrderItem(ctx context.Context, orderItemID int64) (domain.ProvisionJob, error)
	ListDueProvisionJobs(ctx context.Context, limit int) ([]domain.ProvisionJob, error)
	GetProvisionJob(ctx context.Context, id int64) (domain.ProvisionJob, error)
	// RequeueProvisionJob resets a terminal job (done/failed) to retry with the
	// lease released so a manual retry (or the watchdog) can resume it from its
	// persisted stage. The stage is intentionally preserved; the caller decides
	// whether to clear it.
	RequeueProvisionJob(ctx context.Context, jobID int64) (domain.ProvisionJob, error)
	// UpdateProvisionJob persists a job write CAS'd on the current lease owner
	// and fence. If the worker no longer holds the lease (a second instance took
	// over after the lease expired), the update is a no-op and the caller must
	// abandon further side effects.
	UpdateProvisionJob(ctx context.Context, job domain.ProvisionJob) (bool, error)
	// ClaimProvisionJob atomically claims a due job for this worker: it
	// transitions a pending/retry job, or reclaims a running job whose lease has
	// elapsed (a crashed worker). The returned job carries the fresh owner,
	// fence and attempts so the worker acts on post-claim state, not a stale
	// snapshot. It returns claimed=false for the losing worker.
	ClaimProvisionJob(ctx context.Context, jobID int64, owner string, leaseUntil time.Time) (domain.ProvisionJob, bool, error)
	// RenewProvisionJobLease extends a held lease. It is CAS'd on owner+fence so
	// a worker that lost the lease cannot renew it. Returns true only when the
	// lease was renewed.
	RenewProvisionJobLease(ctx context.Context, jobID int64, owner string, fence int, leaseUntil time.Time) (bool, error)
}

type ResizeTaskRepository interface {
	CreateResizeTask(ctx context.Context, task *domain.ResizeTask) error
	GetResizeTask(ctx context.Context, id int64) (domain.ResizeTask, error)
	// UpdateResizeTask persists a task write CAS'd on the lease owner and fence.
	// Returns false when the writer no longer holds the lease, in which case the
	// caller must abandon further side effects.
	UpdateResizeTask(ctx context.Context, task domain.ResizeTask) (bool, error)
	ListDueResizeTasks(ctx context.Context, limit int) ([]domain.ResizeTask, error)
	HasPendingResizeTask(ctx context.Context, vpsID int64) (bool, error)
	// ClaimResizeTask atomically transitions a pending task, or reclaims a
	// running task whose lease has elapsed (a crashed worker), assigning the
	// lease to this worker and bumping the fence. Returns the fresh task and
	// claimed=false for the losing worker.
	ClaimResizeTask(ctx context.Context, taskID int64, owner string, leaseUntil time.Time) (domain.ResizeTask, bool, error)
	// RenewResizeTaskLease extends a held lease, CAS'd on owner+fence.
	RenewResizeTaskLease(ctx context.Context, taskID int64, owner string, fence int, leaseUntil time.Time) (bool, error)
}

type ScheduledTaskRunRepository interface {
	CreateTaskRun(ctx context.Context, run *domain.ScheduledTaskRun) error
	UpdateTaskRun(ctx context.Context, run domain.ScheduledTaskRun) error
	ListTaskRuns(ctx context.Context, key string, limit int) ([]domain.ScheduledTaskRun, error)
	// AcquireTaskLease atomically takes the exclusive execution lease for a task.
	// It returns true when this caller is the lease winner. A lease whose
	// lease_until has passed can be stolen (CAS), so a crashed worker is
	// recoverable by the next contender.
	AcquireTaskLease(ctx context.Context, taskKey, owner string, leaseFor time.Duration) (domain.ScheduledTaskLease, bool, error)
	// RenewTaskLease extends lease_until when the caller still holds the lease
	// (owner + fencing token unchanged). It returns false if the lease was lost.
	RenewTaskLease(ctx context.Context, taskKey, owner string, token int64, leaseFor time.Duration) (bool, error)
	// ReleaseTaskLease clears the lease row (used on clean completion).
	ReleaseTaskLease(ctx context.Context, taskKey, owner string, token int64) error
	// CompleteTaskRun writes the final run status only when the caller still
	// holds the lease (owner + fencing token + not expired). It returns false
	// when the lease was lost, so an old worker cannot commit stale results.
	CompleteTaskRun(ctx context.Context, run domain.ScheduledTaskRun, taskKey, owner string, token int64) (bool, error)
}

// RefundCleanupRepository persists the wallet-refund VPS cleanup outbox. It is
// lease+fence claimed so exactly one worker deletes a refunded host, with
// bounded backoff on failure and full visibility for admin reconciliation.
type RefundCleanupRepository interface {
	CreateRefundCleanupJob(ctx context.Context, job *domain.RefundCleanupJob) error
	GetRefundCleanupJobByWalletOrder(ctx context.Context, walletOrderID int64) (domain.RefundCleanupJob, error)
	ListDueRefundCleanupJobs(ctx context.Context, limit int) ([]domain.RefundCleanupJob, error)
	ListRefundCleanupJobs(ctx context.Context, limit, offset int) ([]domain.RefundCleanupJob, int, error)
	ClaimRefundCleanupJob(ctx context.Context, jobID int64, owner string, leaseUntil time.Time) (domain.RefundCleanupJob, bool, error)
	UpdateRefundCleanupJob(ctx context.Context, job domain.RefundCleanupJob) (bool, error)
	MarkRefundCleanupDone(ctx context.Context, jobID int64) error
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
	// DeletePermissionGroupSafe atomically re-checks user/API-key references and
	// deletes the group in one transaction, closing the count-then-delete TOCTOU.
	DeletePermissionGroupSafe(ctx context.Context, id int64) error
}

type UserTierRepository interface {
	ListUserTierGroups(ctx context.Context) ([]domain.UserTierGroup, error)
	GetUserTierGroup(ctx context.Context, id int64) (domain.UserTierGroup, error)
	CreateUserTierGroup(ctx context.Context, group *domain.UserTierGroup) error
	UpdateUserTierGroup(ctx context.Context, group domain.UserTierGroup) error
	DeleteUserTierGroup(ctx context.Context, id int64) error

	ListUserTierDiscountRules(ctx context.Context, groupID int64) ([]domain.UserTierDiscountRule, error)
	CreateUserTierDiscountRule(ctx context.Context, rule *domain.UserTierDiscountRule) error
	UpdateUserTierDiscountRule(ctx context.Context, rule domain.UserTierDiscountRule) error
	DeleteUserTierDiscountRule(ctx context.Context, id int64) error

	ListUserTierAutoRules(ctx context.Context, groupID int64) ([]domain.UserTierAutoRule, error)
	CreateUserTierAutoRule(ctx context.Context, rule *domain.UserTierAutoRule) error
	UpdateUserTierAutoRule(ctx context.Context, rule domain.UserTierAutoRule) error
	DeleteUserTierAutoRule(ctx context.Context, id int64) error

	GetUserTierMembership(ctx context.Context, userID int64) (domain.UserTierMembership, error)
	UpsertUserTierMembership(ctx context.Context, item *domain.UserTierMembership) error
	ClearUserTierMembership(ctx context.Context, userID int64) error
	ListExpiredUserTierMemberships(ctx context.Context, now time.Time, limit int) ([]domain.UserTierMembership, error)

	GetUserTierPriceCache(ctx context.Context, groupID int64, packageID int64) (domain.UserTierPriceCache, error)
	DeleteUserTierPriceCachesByGroup(ctx context.Context, groupID int64) error
	UpsertUserTierPriceCaches(ctx context.Context, items []domain.UserTierPriceCache) error
}

type CouponRepository interface {
	ListCouponProductGroups(ctx context.Context) ([]domain.CouponProductGroup, error)
	GetCouponProductGroup(ctx context.Context, id int64) (domain.CouponProductGroup, error)
	CreateCouponProductGroup(ctx context.Context, group *domain.CouponProductGroup) error
	UpdateCouponProductGroup(ctx context.Context, group domain.CouponProductGroup) error
	DeleteCouponProductGroup(ctx context.Context, id int64) error

	ListCoupons(ctx context.Context, filter appshared.CouponFilter, limit, offset int) ([]domain.Coupon, int, error)
	GetCoupon(ctx context.Context, id int64) (domain.Coupon, error)
	GetCouponByCode(ctx context.Context, code string) (domain.Coupon, error)
	CreateCoupon(ctx context.Context, coupon *domain.Coupon) error
	UpdateCoupon(ctx context.Context, coupon domain.Coupon) error
	DeleteCoupon(ctx context.Context, id int64) error

	CountCouponRedemptions(ctx context.Context, couponID int64, userID *int64, statuses []string) (int64, error)
	CreateCouponRedemption(ctx context.Context, redemption *domain.CouponRedemption) error
	UpdateCouponRedemptionStatusByOrder(ctx context.Context, orderID int64, fromStatuses []string, toStatus string) error
	CountUserSuccessfulOrders(ctx context.Context, userID int64) (int64, error)
}

type PasswordResetTokenRepository interface {
	CreatePasswordResetToken(ctx context.Context, token *domain.PasswordResetToken) error
	GetPasswordResetToken(ctx context.Context, token string) (domain.PasswordResetToken, error)
	MarkPasswordResetTokenUsed(ctx context.Context, tokenID int64) error
	// MarkPasswordResetTokenUsedIfUnused atomically claims a token for
	// single-use consumption. It reports whether this caller won the claim,
	// so concurrent consumers cannot both succeed (TOCTOU).
	MarkPasswordResetTokenUsedIfUnused(ctx context.Context, tokenID int64) (bool, error)
	// ConsumePasswordResetTokenAndUpdatePassword atomically verifies the token
	// is unused/unexpired, marks it used and updates the owner's password in
	// one transaction, so a failed update cannot burn a usable token.
	ConsumePasswordResetTokenAndUpdatePassword(ctx context.Context, tokenID, userID int64, passwordHash string, now time.Time) (bool, error)
	DeleteExpiredTokens(ctx context.Context) error
}

type PasswordResetTicketRepository interface {
	CreatePasswordResetTicket(ctx context.Context, ticket *domain.PasswordResetTicket) error
	GetPasswordResetTicket(ctx context.Context, token string) (domain.PasswordResetTicket, error)
	MarkPasswordResetTicketUsed(ctx context.Context, ticketID int64) error
	// MarkPasswordResetTicketUsedIfUnused atomically claims a ticket for
	// single-use consumption (see MarkPasswordResetTokenUsedIfUnused).
	MarkPasswordResetTicketUsedIfUnused(ctx context.Context, ticketID int64) (bool, error)
	// ConsumePasswordResetTicketAndUpdatePassword atomically claims the ticket
	// and updates the owner's password in one transaction.
	ConsumePasswordResetTicketAndUpdatePassword(ctx context.Context, ticketID, userID int64, passwordHash string, now time.Time) (bool, error)
	// ConsumeContactChangeTicketAndUpdateContact atomically claims the ticket,
	// re-checks contact uniqueness and applies the new email/phone.
	ConsumeContactChangeTicketAndUpdateContact(ctx context.Context, ticketID, userID int64, kind, value string, now time.Time) (bool, error)
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
	GetWalletOrderByPaymentTradeNo(ctx context.Context, tradeNo string) (domain.WalletOrder, error)
	ListWalletOrders(ctx context.Context, userID int64, limit, offset int) ([]domain.WalletOrder, int, error)
	ListWalletOrdersFiltered(ctx context.Context, userID int64, status, orderType string, limit, offset int) ([]domain.WalletOrder, int, error)
	SummarizeWalletOrders(ctx context.Context, userID int64, monthStart, monthEnd time.Time) (domain.WalletOrderSummary, error)
	ListAllWalletOrders(ctx context.Context, status string, limit, offset int) ([]domain.WalletOrder, int, error)
	UpdateWalletOrderStatus(ctx context.Context, id int64, status domain.WalletOrderStatus, reviewedBy *int64, reason string) error
	UpdateWalletOrderStatusIfCurrent(ctx context.Context, id int64, currentStatus, targetStatus domain.WalletOrderStatus, reviewedBy *int64, reason string) (bool, error)
	ApproveWalletOrderAtomic(ctx context.Context, order domain.WalletOrder, adminID int64, amount int64, txType string) (domain.Wallet, bool, error)
	UpdateWalletOrderMeta(ctx context.Context, id int64, metaJSON string) error
}

type ProbeNodeRepository interface {
	CreateProbeNode(ctx context.Context, node *domain.ProbeNode) error
	GetProbeNode(ctx context.Context, id int64) (domain.ProbeNode, error)
	GetProbeNodeByAgentID(ctx context.Context, agentID string) (domain.ProbeNode, error)
	ListProbeNodes(ctx context.Context, filter appshared.ProbeNodeFilter, limit, offset int) ([]domain.ProbeNode, int, error)
	// ListProbeNodesByCursor pages the probe table with an id > afterID cursor so
	// offline scans cover every node without a LIMIT truncation (P2-34).
	ListProbeNodesByCursor(ctx context.Context, afterID int64, limit int) ([]domain.ProbeNode, error)
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
	MarkProbeEnrollTokenUsedIfUnused(ctx context.Context, id int64, usedAt time.Time) (bool, error)
	// ConsumeProbeEnrollTokenAndUpdateNode atomically claims a one-time enroll
	// token and applies the node binding/secret update in one transaction, so a
	// failed node update rolls the claim back.
	ConsumeProbeEnrollTokenAndUpdateNode(ctx context.Context, tokenID, nodeID int64, updates map[string]any, usedAt time.Time) (bool, error)
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

// PaymentProviderRegistry is the payment-provider registry contract. It is
// declared as an alias of appshared.PaymentProviderRegistry (the canonical
// definition) so the two names can never drift apart: methods added to one are
// automatically part of the other.
type PaymentProviderRegistry = appshared.PaymentProviderRegistry

type OrderApprover interface {
	ApproveOrder(ctx context.Context, adminID int64, orderID int64) error
}

type AutomationClientResolver interface {
	ClientForGoodsType(ctx context.Context, goodsTypeID int64) (appshared.AutomationClient, error)
}

// AutomationCapabilityResolver exposes the same manifest capability baseline
// used by user-facing displays, without starting or invoking a plugin.
type AutomationCapabilityResolver interface {
	CapabilitiesForGoodsType(ctx context.Context, goodsTypeID int64) (*appshared.PluginAutomationCapability, error)
}

type EmailSender interface {
	Send(ctx context.Context, to string, subject string, body string) error
}

type RealNameRepository interface {
	CreateRealNameVerification(ctx context.Context, record *domain.RealNameVerification) error
	GetLatestRealNameVerification(ctx context.Context, userID int64) (domain.RealNameVerification, error)
	ListRealNameVerifications(ctx context.Context, userID *int64, status string, limit, offset int) ([]domain.RealNameVerification, int, error)
	// ListPendingRealNameVerifications returns up to limit records with status
	// 'pending' and id > afterID ordered by id ASC. It is the polling cursor
	// used by realname.PollPending: filtering at the SQL layer avoids scanning
	// the whole table each tick, and the stable id cursor avoids the duplicate
	// reads of offset pagination while records are updated during a poll.
	ListPendingRealNameVerifications(ctx context.Context, afterID int64, limit int) ([]domain.RealNameVerification, error)
	UpdateRealNameStatus(ctx context.Context, id int64, status string, reason string, verifiedAt *time.Time) error
	// UpdateRealNameProviderState persists the (encrypted) provider state and
	// sanitizes the stored Reason, used to migrate legacy pending records whose
	// token was embedded in the Reason column.
	UpdateRealNameProviderState(ctx context.Context, id int64, providerState string, stableReason string) error
	// AllowRealNameAttempt atomically records one verification attempt for the
	// user+dimension in the current window and reports whether the budget is
	// still within max. It is DB-backed so the budget holds across instances
	// and restarts.
	AllowRealNameAttempt(ctx context.Context, userID int64, dimension string, max int, window time.Duration) (bool, error)
	// MarkRealNamePolled records that a pending verification was polled against
	// the provider, persisting the poll time and bumping the poll counter so the
	// poll cooldown and give-up cap hold across instances and restarts.
	MarkRealNamePolled(ctx context.Context, id int64, at time.Time) error
	// CleanupRealNameAttempts deletes attempt windows older than before.
	CleanupRealNameAttempts(ctx context.Context, before time.Time) (int64, error)
}

type SMSSender interface {
	Send(ctx context.Context, pluginID, instanceID string, msg appshared.SMSMessage) (appshared.SMSDelivery, error)
}
