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
	TOTPSecretEnc        string
	TOTPPendingSecretEnc string
	Bio                  string
	Intro                string
	PermissionGroupID    *int64
	UserTierGroupID      *int64
	UserTierExpireAt     *time.Time
	PasswordHash         string
	PasswordChangedAt    *time.Time
	Role                 UserRole
	Status               UserStatus
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type Captcha struct {
	ID        string
	CodeHash  string
	ExpiresAt time.Time
	CreatedAt time.Time
}

type VerificationCode struct {
	ID        int64
	Channel   string
	Receiver  string
	Purpose   string
	CodeHash  string
	ExpiresAt time.Time
	CreatedAt time.Time
}

type APIKey struct {
	ID                int64
	Name              string
	KeyHash           string
	Status            APIKeyStatus
	ScopesJSON        string
	PermissionGroupID *int64
	CreatedAt         time.Time
	UpdatedAt         time.Time
	LastUsedAt        *time.Time
}

type UserAPIKey struct {
	ID         int64
	UserID     int64
	Name       string
	AKID       string
	KeyHash    string
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
}

type PasswordResetToken struct {
	ID        int64
	UserID    int64
	Token     string
	ExpiresAt time.Time
	Used      bool
	CreatedAt time.Time
}

type PasswordResetTicket struct {
	ID        int64
	UserID    int64
	Channel   string
	Receiver  string
	Token     string
	ExpiresAt time.Time
	Used      bool
	CreatedAt time.Time
}
