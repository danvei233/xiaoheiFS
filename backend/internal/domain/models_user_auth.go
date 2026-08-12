package domain

import "time"

type UserRole string

const (
	UserRoleUser  UserRole = "user"
	UserRoleAdmin UserRole = "admin"
)

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusDisabled UserStatus = "disabled"
	UserStatusBlocked  UserStatus = "blocked"
)

type OrderStatus string

const (
	OrderStatusDraft          OrderStatus = "draft"
	OrderStatusPendingPayment OrderStatus = "pending_payment"
	OrderStatusPendingReview  OrderStatus = "pending_review"
	OrderStatusRejected       OrderStatus = "rejected"
	OrderStatusApproved       OrderStatus = "approved"
	OrderStatusProvisioning   OrderStatus = "provisioning"
	OrderStatusActive         OrderStatus = "active"
	OrderStatusFailed         OrderStatus = "failed"
	OrderStatusCanceled       OrderStatus = "canceled"
)

type OrderItemStatus string

const (
	OrderItemStatusPendingPayment OrderItemStatus = "pending_payment"
	OrderItemStatusPendingReview  OrderItemStatus = "pending_review"
	OrderItemStatusApproved       OrderItemStatus = "approved"
	OrderItemStatusProvisioning   OrderItemStatus = "provisioning"
	OrderItemStatusActive         OrderItemStatus = "active"
	OrderItemStatusFailed         OrderItemStatus = "failed"
	OrderItemStatusRejected       OrderItemStatus = "rejected"
	OrderItemStatusCanceled       OrderItemStatus = "canceled"
)

type APIKeyStatus string

const (
	APIKeyStatusActive   APIKeyStatus = "active"
	APIKeyStatusDisabled APIKeyStatus = "disabled"
)

type RequestActorMode string

const (
	RequestActorModeUserJWT     RequestActorMode = "user_jwt"
	RequestActorModeUserAPIKey  RequestActorMode = "user_apikey"
	RequestActorModeAdminJWT    RequestActorMode = "admin_jwt"
	RequestActorModeAdminAPIKey RequestActorMode = "admin_apikey"
	RequestActorModeUnknown     RequestActorMode = "unknown"
)

type PaymentStatus string

const (
	PaymentStatusPendingPayment PaymentStatus = "pending_payment"
	PaymentStatusPendingReview  PaymentStatus = "pending_review"
	PaymentStatusApproved       PaymentStatus = "approved"
	PaymentStatusRejected       PaymentStatus = "rejected"
)

type WalletOrderType string

const (
	WalletOrderRecharge WalletOrderType = "recharge"
	WalletOrderWithdraw WalletOrderType = "withdraw"
	WalletOrderRefund   WalletOrderType = "refund"
)

type WalletOrderStatus string

const (
	WalletOrderPendingReview WalletOrderStatus = "pending_review"
	WalletOrderApproved      WalletOrderStatus = "approved"
	WalletOrderRejected      WalletOrderStatus = "rejected"
)

type VPSStatus string

const (
	VPSStatusProvisioning     VPSStatus = "provisioning"
	VPSStatusRunning          VPSStatus = "running"
	VPSStatusStopped          VPSStatus = "stopped"
	VPSStatusReinstalling     VPSStatus = "reinstalling"
	VPSStatusReinstallFailed  VPSStatus = "reinstall_failed"
	VPSStatusExpiredLocked    VPSStatus = "expired_locked"
	VPSStatusRescue           VPSStatus = "rescue"
	VPSStatusCrackingPassword VPSStatus = "cracking_password"
	VPSStatusLocked           VPSStatus = "locked"
	VPSStatusFailed           VPSStatus = "failed"
	VPSStatusDeleting         VPSStatus = "deleting"
	VPSStatusUnknown          VPSStatus = "unknown"
)

type VPSAdminStatus string

const (
	VPSAdminStatusNormal VPSAdminStatus = "normal"
	VPSAdminStatusAbuse  VPSAdminStatus = "abuse"
	VPSAdminStatusFraud  VPSAdminStatus = "fraud"
	VPSAdminStatusLocked VPSAdminStatus = "locked"
)

type ResizeTaskStatus string

const (
	ResizeTaskStatusPending ResizeTaskStatus = "pending"
	ResizeTaskStatusRunning ResizeTaskStatus = "running"
	ResizeTaskStatusDone    ResizeTaskStatus = "done"
	ResizeTaskStatusFailed  ResizeTaskStatus = "failed"
)

type User struct {
	ID                   int64
	Username             string
	Email                string
	QQ                   string
	Avatar               string
	Phone                string
	LastLoginIP          string
	LastLoginAt          *time.Time
	LastLoginCity        string
	LastLoginTZ          string
	TOTPEnabled          bool
	TOTPSecretEnc        string `json:"-"`
	TOTPPendingSecretEnc string `json:"-"`
	Bio                  string
	Intro                string
	PermissionGroupID    *int64
	UserTierGroupID      *int64
	UserTierExpireAt     *time.Time
	PasswordHash         string `json:"-"`
	PasswordChangedAt    *time.Time
	// PasswordVersion is a monotonic counter incremented on every password
	// change. Auth tokens carry the version they were issued at, so a password
	// change revokes every previously issued token without relying on
	// timestamp-precision edge cases (the same-second boundary). Zero means the
	// password has not changed since version tracking was introduced.
	PasswordVersion int64
	Role            UserRole
	Status          UserStatus
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Captcha struct {
	ID        string
	CodeHash  string `json:"-"`
	ExpiresAt time.Time
	CreatedAt time.Time
}

type VerificationCode struct {
	ID        int64
	Channel   string
	Receiver  string
	Purpose   string
	CodeHash  string `json:"-"`
	ExpiresAt time.Time
	CreatedAt time.Time
}

type APIKey struct {
	ID                int64
	Name              string
	KeyHash           string `json:"-"`
	Status            APIKeyStatus
	ScopesJSON        string
	PermissionGroupID *int64
	CreatedAt         time.Time
	UpdatedAt         time.Time
	LastUsedAt        *time.Time
}

type UserAPIKey struct {
	ID      int64
	UserID  int64
	Name    string
	AKID    string
	KeyHash string `json:"-"`
	// SecretEnc holds the AES-GCM-encrypted signing secret. The raw secret is
	// returned to the owner exactly once at creation, so a database leak does
	// not expose usable signing keys (P0-6). Legacy rows may have SecretEnc
	// empty, in which case KeyHash itself is the legacy signing secret.
	SecretEnc  string
	Status     APIKeyStatus
	ScopesJSON string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	LastUsedAt *time.Time
}

type RequestActor struct {
	Mode          RequestActorMode
	UserID        int64
	Role          string
	UserAPIKeyID  int64
	AdminAPIKeyID int64
	// ImpersonatorID is the admin user id that issued an impersonation token
	// (the actor acting as another user). Zero when the request is not an
	// impersonated one, so audit trails can distinguish "user themselves" from
	// "admin acting as user" (round-4 auth review Finding 7).
	ImpersonatorID int64
}

// RobotWebhookOperatorID is the audit actor used for business commands issued
// through the authenticated robot webhook. It is distinct from 0 (unattributed)
// so webhook approvals/rejections are traceable in the audit log.
const RobotWebhookOperatorID int64 = -1

// AdminAPIKeyActorBase is the negative base from which admin API-key actor ids
// are derived. Encoding the actor as a large negative number keeps the three
// actor classes disjoint in audit logs: admin/user ids are > 0,
// RobotWebhookOperatorID is -1, and admin API keys are <= AdminAPIKeyActorBase.
// A raw API key id stored verbatim would collide with admin user ids, so the
// offset keeps the type+id information without collision (Issue #51 P2-22).
const AdminAPIKeyActorBase int64 = -1_000_000_000

// AdminAPIKeyActorID maps a stored admin API-key id to its audit actor id.
func AdminAPIKeyActorID(keyID int64) int64 {
	if keyID <= 0 {
		return 0
	}
	return AdminAPIKeyActorBase - keyID
}

type PasswordResetToken struct {
	ID        int64
	UserID    int64
	Token     string `json:"-"`
	ExpiresAt time.Time
	Used      bool
	CreatedAt time.Time
}

type PasswordResetTicket struct {
	ID        int64
	UserID    int64
	Channel   string
	Receiver  string
	Token     string `json:"-"`
	ExpiresAt time.Time
	Used      bool
	CreatedAt time.Time
}
