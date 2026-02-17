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
	PasswordHash         string
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

type Region struct {
	ID          int64
	GoodsTypeID int64
	Code        string
	Name        string
	Active      bool
}

type GoodsType struct {
	ID                   int64
	Code                 string
	Name                 string
	Active               bool
	SortOrder            int
	AutomationCategory   string
	AutomationPluginID   string
	AutomationInstanceID string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type PlanGroup struct {
	ID                int64
	GoodsTypeID       int64
	RegionID          int64
	Name              string
	LineID            int64
	UnitCore          int64
	UnitMem           int64
	UnitDisk          int64
	UnitBW            int64
	AddCoreMin        int
	AddCoreMax        int
	AddCoreStep       int
	AddMemMin         int
	AddMemMax         int
	AddMemStep        int
	AddDiskMin        int
	AddDiskMax        int
	AddDiskStep       int
	AddBWMin          int
	AddBWMax          int
	AddBWStep         int
	Active            bool
	Visible           bool
	CapacityRemaining int
	SortOrder         int
}

type Package struct {
	ID                int64
	GoodsTypeID       int64
	PlanGroupID       int64
	ProductID         int64
	Name              string
	Cores             int
	MemoryGB          int
	DiskGB            int
	BandwidthMB       int
	CPUModel          string
	Monthly           int64
	PortNum           int
	SortOrder         int
	Active            bool
	Visible           bool
	CapacityRemaining int
}

type SystemImage struct {
	ID        int64
	ImageID   int64
	Name      string
	Type      string
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CartItem struct {
	ID        int64
	UserID    int64
	PackageID int64
	SystemID  int64
	SpecJSON  string
	Qty       int
	Amount    int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Order struct {
	ID             int64
	UserID         int64
	OrderNo        string
	Status         OrderStatus
	TotalAmount    int64
	Currency       string
	IdempotencyKey string
	PendingReason  string
	ApprovedBy     *int64
	ApprovedAt     *time.Time
	RejectedReason string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type OrderItem struct {
	ID                   int64
	OrderID              int64
	PackageID            int64
	SystemID             int64
	SpecJSON             string
	Qty                  int
	Amount               int64
	Status               OrderItemStatus
	GoodsTypeID          int64
	AutomationInstanceID string
	Action               string
	DurationMonths       int
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type VPSInstance struct {
	ID                   int64
	UserID               int64
	OrderItemID          int64
	AutomationInstanceID string
	GoodsTypeID          int64
	Name                 string
	Region               string
	RegionID             int64
	LineID               int64
	PackageID            int64
	PackageName          string
	CPU                  int
	MemoryGB             int
	DiskGB               int
	BandwidthMB          int
	PortNum              int
	MonthlyPrice         int64
	SpecJSON             string
	SystemID             int64
	Status               VPSStatus
	AutomationState      int
	AdminStatus          VPSAdminStatus
	ExpireAt             *time.Time
	PanelURLCache        string
	AccessInfoJSON       string
	LastEmergencyRenewAt *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type OrderEvent struct {
	ID        int64
	OrderID   int64
	Seq       int64
	Type      string
	DataJSON  string
	CreatedAt time.Time
}

type OrderPayment struct {
	ID             int64
	OrderID        int64
	UserID         int64
	Method         string
	Amount         int64
	Currency       string
	TradeNo        string
	Note           string
	ScreenshotURL  string
	Status         PaymentStatus
	IdempotencyKey string
	ReviewedBy     *int64
	ReviewReason   string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type AdminAuditLog struct {
	ID         int64
	AdminID    int64
	Action     string
	TargetType string
	TargetID   string
	DetailJSON string
	CreatedAt  time.Time
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

type EmailTemplate struct {
	ID        int64
	Name      string
	Subject   string
	Body      string
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Setting struct {
	Key       string
	ValueJSON string
	UpdatedAt time.Time
}

type BillingCycle struct {
	ID         int64
	Name       string
	Months     int
	Multiplier float64
	MinQty     int
	MaxQty     int
	Active     bool
	SortOrder  int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type AutomationLog struct {
	ID           int64
	OrderID      int64
	OrderItemID  int64
	Action       string
	RequestJSON  string
	ResponseJSON string
	Success      bool
	Message      string
	CreatedAt    time.Time
}

type ProvisionJob struct {
	ID          int64
	OrderID     int64
	OrderItemID int64
	HostID      int64
	HostName    string
	Status      string
	Attempts    int
	NextRunAt   time.Time
	LastError   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ResizeTaskStatus string

const (
	ResizeTaskStatusPending ResizeTaskStatus = "pending"
	ResizeTaskStatusRunning ResizeTaskStatus = "running"
	ResizeTaskStatusDone    ResizeTaskStatus = "done"
	ResizeTaskStatusFailed  ResizeTaskStatus = "failed"
)

type ResizeTask struct {
	ID          int64
	VPSID       int64
	OrderID     int64
	OrderItemID int64
	Status      ResizeTaskStatus
	ScheduledAt *time.Time
	StartedAt   *time.Time
	FinishedAt  *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ScheduledTaskRun struct {
	ID          int64
	TaskKey     string
	Status      string
	StartedAt   time.Time
	FinishedAt  *time.Time
	DurationSec int
	Message     string
	CreatedAt   time.Time
}

type IntegrationSyncLog struct {
	ID        int64
	Target    string
	Mode      string
	Status    string
	Message   string
	CreatedAt time.Time
}

type PermissionGroup struct {
	ID              int64
	Name            string
	Description     string
	PermissionsJSON string
	CreatedAt       time.Time
	UpdatedAt       time.Time
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

type Permission struct {
	ID           int64
	Code         string
	Name         string
	FriendlyName string
	Category     string
	ParentCode   string
	SortOrder    int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type PermissionDefinition struct {
	Code         string
	Name         string
	FriendlyName string
	Category     string
	ParentCode   string
	SortOrder    int
}

type PermissionTree struct {
	Code         string
	Name         string
	FriendlyName string
	Category     string
	Children     []PermissionTree
}
