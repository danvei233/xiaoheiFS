package domain

import "time"

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

// Order source identifiers. The repo persists these strings on orders for
// auditability (user UI vs signed API key).
const (
	OrderSourceUserUI     = "user_ui"
	OrderSourceUserAPIKey = "user_apikey"
)

type Order struct {
	ID             int64
	UserID         int64
	OrderNo        string
	Source         string
	Status         OrderStatus
	TotalAmount    int64
	Currency       string
	CouponID       *int64
	CouponCode     string
	CouponDiscount int64
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
	// VPSID is the normalized link to the target VPS of an exclusive lifecycle
	// operation (renew/emergency_renew/resize/refund), mirrored from
	// spec_json.vps_id so exclusivity checks are indexed, not JSON-scanned.
	VPSID          int64
	Action         string
	DurationMonths int
	CreatedAt      time.Time
	UpdatedAt      time.Time
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
	// BandwidthMB is the VPS bandwidth cap in Mbps (DB column bandwidth_mbps),
	// not megabytes. The name is historical; treat the value as Mbps.
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

// Job lease/duration constants shared by the app workers and the repository so
// neither layer can drift on how long a lease may be held before a crashed
// worker's job is reclaimed.
const (
	// ProvisionLeaseDuration is how long a worker holds a provision job lease
	// before another worker may reclaim it after a crash. The worker renews this
	// lease with heartbeats while a slow external call is in flight.
	ProvisionLeaseDuration = 60 * time.Second
	// ResizeTaskStaleDuration is how long a running resize task may outlive its
	// lease before another worker reclaims it (the previous process crashed).
	ResizeTaskStaleDuration = 30 * time.Minute
)

// ProvisionJobStage values. The stage is the durable provision intent: it tells
// a recovering worker exactly what side effects have already happened so it
// reconciles instead of blindly re-creating a host.
const (
	ProvisionStageIntent            = "intent"              // intent persisted; CreateHost not yet called
	ProvisionStageCreateHostSent    = "create_host_sent"    // CreateHost called, result unknown (reconcile only)
	ProvisionStageHostCreated       = "host_created"        // host id known; waiting for ready state
	ProvisionStageCompleted         = "completed"           // terminal success
	ProvisionStageFailed            = "failed"              // terminal failure
	ProvisionStageManualRecovery    = "manual_recovery"     // needs an operator
)

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
	// LeaseOwner identifies the worker instance currently holding the job. A
	// running job whose LeaseUntil has elapsed is reclaimable by another worker.
	LeaseOwner string
	LeaseUntil *time.Time
	// Fence is a monotonic token bumped on every claim. Every final write is
	// CAS'd on (LeaseOwner, Fence); a worker that lost the lease cannot overwrite
	// the new owner's results.
	Fence int
	// Stage is the durable provision intent (see ProvisionStage*).
	Stage string
	// AttemptID uniquely identifies one logical provision attempt (survives
	// restarts so external idempotency keys can be derived deterministically).
	AttemptID string
	// CredsEncrypted holds the encrypted system/VNC passwords generated for this
	// attempt, so a crash between CreateHost and the local host-id write does not
	// lose the credentials the automation provider may never return again.
	CredsEncrypted string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ResizeTask struct {
	ID          int64
	VPSID       int64
	OrderID     int64
	OrderItemID int64
	Status      ResizeTaskStatus
	ScheduledAt *time.Time
	StartedAt   *time.Time
	FinishedAt  *time.Time
	// LeaseOwner/Fence give resize tasks the same fencing guarantees as
	// provision jobs: a worker that lost the lease cannot overwrite a takeover.
	LeaseOwner string
	LeaseUntil *time.Time
	Fence      int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// RefundCleanupJob is a durable outbox entry for deleting a refunded VPS host.
// It survives restarts and is claimed with a lease+fence by exactly one worker.
type RefundCleanupJob struct {
	ID            int64
	WalletOrderID int64
	UserID        int64
	VPSID         int64
	HostID        int64
	GoodsTypeID   int64
	Status        string
	Attempts      int
	NextRunAt     time.Time
	LastErrorCode string
	LastError     string
	LeaseOwner    string
	LeaseUntil    *time.Time
	Fence         int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// RefundCleanupJobStatus values.
const (
	RefundCleanupPending = "pending"
	RefundCleanupRunning = "running"
	RefundCleanupRetry   = "retry"
	RefundCleanupDone    = "done"
	RefundCleanupFailed  = "failed"
)
