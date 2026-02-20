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

type Order struct {
	ID             int64
	UserID         int64
	OrderNo        string
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
