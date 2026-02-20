package shared

import (
	"context"
	"time"

	"xiaoheiplay/internal/domain"
)

type CartSpec struct {
	AddCores       int   `json:"add_cores"`
	AddMemGB       int   `json:"add_mem_gb"`
	AddDiskGB      int   `json:"add_disk_gb"`
	AddBWMbps      int   `json:"add_bw_mbps"`
	BillingCycleID int64 `json:"billing_cycle_id"`
	CycleQty       int   `json:"cycle_qty"`
	DurationMonths int   `json:"duration_months"`
}

type RegisterInput struct {
	Username        string
	Email           string
	QQ              string
	Phone           string
	Password        string
	CaptchaID       string
	CaptchaCode     string
	CaptchaRequired bool
}

type UpdateProfileInput struct {
	Username string
	Email    string
	QQ       string
	Phone    string
	Bio      string
	Intro    string
	Password string
}

type AutomationLogContext struct {
	OrderID     int64
	OrderItemID int64
}

type PasswordResetTicketRepository interface {
	CreatePasswordResetTicket(ctx context.Context, ticket *domain.PasswordResetTicket) error
	GetPasswordResetTicket(ctx context.Context, token string) (domain.PasswordResetTicket, error)
	MarkPasswordResetTicketUsed(ctx context.Context, ticketID int64) error
	DeleteExpiredPasswordResetTickets(ctx context.Context) error
}

type OrderFilter struct {
	Status string
	UserID int64
	From   *time.Time
	To     *time.Time
}

type PaymentFilter struct {
	Status string
	From   *time.Time
	To     *time.Time
}

type NotificationFilter struct {
	UserID *int64
	Status string
	Limit  int
	Offset int
}

type TicketFilter struct {
	UserID  *int64
	Status  string
	Keyword string
	Limit   int
	Offset  int
}

type CMSPostFilter struct {
	CategoryID    *int64
	CategoryKey   string
	Status        string
	Lang          string
	PublishedOnly bool
	Limit         int
	Offset        int
}

type ProbeNodeFilter struct {
	Keyword string
	Status  string
}

type CouponFilter struct {
	Keyword        string
	ProductGroupID int64
	Active         *bool
}

type OrderItemInput struct {
	PackageID int64    `json:"package_id"`
	SystemID  int64    `json:"system_id"`
	Spec      CartSpec `json:"spec"`
	Qty       int      `json:"qty"`
}

type PaymentInput struct {
	Method        string `json:"method"`
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	TradeNo       string `json:"trade_no"`
	Note          string `json:"note"`
	ScreenshotURL string `json:"screenshot_url"`
}

type PaymentSelectInput struct {
	Method    string
	ReturnURL string
	NotifyURL string
	Extra     map[string]string
}

type PaymentProviderInfo struct {
	Key        string
	Name       string
	Enabled    bool
	SchemaJSON string
	ConfigJSON string
}

type PaymentMethodInfo struct {
	Key        string
	Name       string
	SchemaJSON string
	ConfigJSON string
	Balance    int64
}

type PaymentSelectResult struct {
	Method  string
	Status  string
	TradeNo string
	PayURL  string
	Extra   map[string]string
	Paid    bool
	Message string
	Balance int64
}

type PaymentCreateRequest struct {
	OrderID   int64
	OrderNo   string
	UserID    int64
	Amount    int64
	Currency  string
	Subject   string
	ReturnURL string
	NotifyURL string
	Extra     map[string]string
}

type PaymentCreateResult struct {
	TradeNo string
	PayURL  string
	Extra   map[string]string
}

type PaymentNotifyResult struct {
	OrderNo string
	TradeNo string
	Paid    bool
	Amount  int64
	Raw     map[string]string
	AckBody string
}

type RawHTTPRequest struct {
	Method   string
	Path     string
	RawQuery string
	Headers  map[string][]string
	Body     []byte
}

type PaymentProvider interface {
	Key() string
	Name() string
	SchemaJSON() string
	CreatePayment(ctx context.Context, req PaymentCreateRequest) (PaymentCreateResult, error)
	VerifyNotify(ctx context.Context, req RawHTTPRequest) (PaymentNotifyResult, error)
}

type ConfigurablePaymentProvider interface {
	PaymentProvider
	SetConfig(configJSON string) error
}

type PaymentProviderRegistry interface {
	ListProviders(ctx context.Context, includeDisabled bool) ([]PaymentProvider, error)
	GetProvider(ctx context.Context, key string) (PaymentProvider, error)
	GetProviderConfig(ctx context.Context, key string) (string, bool, error)
	UpdateProviderConfig(ctx context.Context, key string, enabled bool, configJSON string) error
}

type AutomationCreateHostRequest struct {
	LineID     int64
	OS         string
	CPU        int
	MemoryGB   int
	DiskGB     int
	Bandwidth  int
	ExpireTime time.Time
	HostName   string
	SysPwd     string
	VNCPwd     string
	PortNum    int
	Snapshot   int
	Backups    int
}

type AutomationCreateHostResult struct {
	HostID int64
	Raw    map[string]any
}

type AutomationHostInfo struct {
	HostID        int64
	HostName      string
	State         int
	CPU           int
	MemoryGB      int
	DiskGB        int
	Bandwidth     int
	PanelPassword string
	VNCPassword   string
	OSPassword    string
	RemoteIP      string
	ExpireAt      *time.Time
}

type AutomationHostSimple struct {
	ID       int64
	HostName string
	IP       string
}

type AutomationElasticUpdateRequest struct {
	HostID    int64
	CPU       *int
	MemoryGB  *int
	DiskGB    *int
	Bandwidth *int
	PortNum   *int
}

type AutomationImage struct {
	ImageID int64
	Name    string
	Type    string
}

type AutomationLine struct {
	ID     int64
	Name   string
	AreaID int64
	State  int
}

type AutomationArea struct {
	ID    int64
	Name  string
	State int
}

type AutomationProduct struct {
	ID        int64
	Name      string
	CPU       int
	MemoryGB  int
	DiskGB    int
	Bandwidth int
	Price     int64
	PortNum   int
}

type AutomationMonitor struct {
	CPUPercent     int   `json:"cpu"`
	MemoryPercent  int   `json:"memory"`
	BytesIn        int64 `json:"bytes_in"`
	BytesOut       int64 `json:"bytes_out"`
	StoragePercent int   `json:"storage"`
}

type AutomationSnapshot map[string]any
type AutomationBackup map[string]any
type AutomationFirewallRule map[string]any
type AutomationPortMapping map[string]any

type AutomationFirewallRuleCreate struct {
	HostID    int64
	Direction string
	Protocol  string
	Method    string
	Port      string
	IP        string
	Priority  int
}

type AutomationPortMappingCreate struct {
	HostID int64
	Name   string
	Sport  string
	Dport  int64
}

type AutomationClient interface {
	CreateHost(ctx context.Context, req AutomationCreateHostRequest) (AutomationCreateHostResult, error)
	GetHostInfo(ctx context.Context, hostID int64) (AutomationHostInfo, error)
	ListHostSimple(ctx context.Context, searchTag string) ([]AutomationHostSimple, error)
	ElasticUpdate(ctx context.Context, req AutomationElasticUpdateRequest) error
	RenewHost(ctx context.Context, hostID int64, nextDueDate time.Time) error
	LockHost(ctx context.Context, hostID int64) error
	UnlockHost(ctx context.Context, hostID int64) error
	DeleteHost(ctx context.Context, hostID int64) error
	StartHost(ctx context.Context, hostID int64) error
	ShutdownHost(ctx context.Context, hostID int64) error
	RebootHost(ctx context.Context, hostID int64) error
	ResetOS(ctx context.Context, hostID int64, templateID int64, password string) error
	ResetOSPassword(ctx context.Context, hostID int64, password string) error
	ListSnapshots(ctx context.Context, hostID int64) ([]AutomationSnapshot, error)
	CreateSnapshot(ctx context.Context, hostID int64) error
	DeleteSnapshot(ctx context.Context, hostID int64, snapshotID int64) error
	RestoreSnapshot(ctx context.Context, hostID int64, snapshotID int64) error
	ListBackups(ctx context.Context, hostID int64) ([]AutomationBackup, error)
	CreateBackup(ctx context.Context, hostID int64) error
	DeleteBackup(ctx context.Context, hostID int64, backupID int64) error
	RestoreBackup(ctx context.Context, hostID int64, backupID int64) error
	ListFirewallRules(ctx context.Context, hostID int64) ([]AutomationFirewallRule, error)
	AddFirewallRule(ctx context.Context, req AutomationFirewallRuleCreate) error
	DeleteFirewallRule(ctx context.Context, hostID int64, ruleID int64) error
	ListPortMappings(ctx context.Context, hostID int64) ([]AutomationPortMapping, error)
	AddPortMapping(ctx context.Context, req AutomationPortMappingCreate) error
	DeletePortMapping(ctx context.Context, hostID int64, mappingID int64) error
	FindPortCandidates(ctx context.Context, hostID int64, keywords string) ([]int64, error)
	GetPanelURL(ctx context.Context, hostName string, panelPassword string) (string, error)
	ListAreas(ctx context.Context) ([]AutomationArea, error)
	ListImages(ctx context.Context, lineID int64) ([]AutomationImage, error)
	ListLines(ctx context.Context) ([]AutomationLine, error)
	ListProducts(ctx context.Context, lineID int64) ([]AutomationProduct, error)
	GetMonitor(ctx context.Context, hostID int64) (AutomationMonitor, error)
	GetVNCURL(ctx context.Context, hostID int64) (string, error)
}

type RealNameVerifyInput struct {
	RealName string
	IDNumber string
	Phone    string
	// CallbackURL is used by face-flow KYC providers that require an upstream callback URL.
	CallbackURL string
}

type RealNameProvider interface {
	Key() string
	Name() string
	Verify(ctx context.Context, realName string, idNumber string) (bool, string, error)
}

type RealNameProviderWithInput interface {
	VerifyWithInput(ctx context.Context, in RealNameVerifyInput) (bool, string, error)
}

type RealNameProviderPendingPoller interface {
	QueryPending(ctx context.Context, token string, provider string) (status string, reason string, err error)
}

type RealNameProviderRegistry interface {
	GetProvider(key string) (RealNameProvider, error)
	ListProviders() []RealNameProvider
}

type EmailSender interface {
	Send(ctx context.Context, to string, subject string, body string) error
}

type RobotOrderPayload struct {
	OrderNo    string
	UserID     int64
	Username   string
	Email      string
	QQ         string
	Amount     int64
	Currency   string
	Items      []RobotOrderItem
	ApproveURL string
}

type RobotOrderItem struct {
	PackageName string
	SystemName  string
	SpecJSON    string
	Amount      int64
}

type RobotNotifier interface {
	NotifyOrderPending(ctx context.Context, payload RobotOrderPayload) error
}

type PushPayload struct {
	Title string
	Body  string
	Data  map[string]string
}

type PushConfig struct {
	ProjectID          string
	ServiceAccountJSON string
	LegacyServerKey    string
}

type WalletOrderCreateInput struct {
	Amount   int64
	Currency string
	Note     string
	Meta     map[string]any
}

type TaskStrategy string

const (
	TaskStrategyInterval TaskStrategy = "interval"
	TaskStrategyDaily    TaskStrategy = "daily"
)

type ScheduledTaskUpdate struct {
	Enabled     *bool        `json:"enabled"`
	Strategy    TaskStrategy `json:"strategy"`
	IntervalSec *int         `json:"interval_sec"`
	DailyAt     *string      `json:"daily_at"`
}

type AutomationConfig struct {
	BaseURL    string `json:"base_url"`
	APIKey     string `json:"api_key"`
	Enabled    bool   `json:"enabled"`
	TimeoutSec int    `json:"timeout_sec"`
	Retry      int    `json:"retry"`
	DryRun     bool   `json:"dry_run"`
}

type AdminVPSCreateInput struct {
	UserID               int64
	OrderItemID          int64
	AutomationInstanceID string
	GoodsTypeID          int64
	Name                 string
	Region               string
	RegionID             int64
	SystemID             int64
	Status               domain.VPSStatus
	AutomationState      int
	AdminStatus          domain.VPSAdminStatus
	ExpireAt             *time.Time
	PanelURLCache        string
	SpecJSON             string
	AccessInfoJSON       string
	Provision            bool
	LineID               int64
	PackageID            int64
	PackageName          string
	OS                   string
	CPU                  int
	MemoryGB             int
	DiskGB               int
	BandwidthMB          int
	PortNum              int
	MonthlyPrice         int64
}

type AdminVPSUpdateInput struct {
	PackageID      *int64
	PackageName    *string
	MonthlyPrice   *int64
	SystemID       *int64
	SpecJSON       *string
	PanelURLCache  *string
	AccessInfoJSON *string
	Status         *domain.VPSStatus
	AdminStatus    *domain.VPSAdminStatus
	CPU            *int
	MemoryGB       *int
	DiskGB         *int
	BandwidthMB    *int
	PortNum        *int
	SyncMode       string
}

type ResizeQuote struct {
	ChargeAmount     int64
	RefundAmount     int64
	RefundToWallet   bool
	CurrentPackageID int64
	CurrentCPU       int
	CurrentMemGB     int
	CurrentDiskGB    int
	CurrentBWMbps    int
	TargetPackageID  int64
	TargetCPU        int
	TargetMemGB      int
	TargetDiskGB     int
	TargetBWMbps     int
	TargetMonthly    int64
	CurrentMonthly   int64
}

func (q ResizeQuote) ToPayload(vpsID int64, spec CartSpec) map[string]any {
	return map[string]any{
		"vps_id":             vpsID,
		"spec":               spec,
		"current_package_id": q.CurrentPackageID,
		"current_cpu":        q.CurrentCPU,
		"current_mem_gb":     q.CurrentMemGB,
		"current_disk_gb":    q.CurrentDiskGB,
		"current_bw_mbps":    q.CurrentBWMbps,
		"target_package_id":  q.TargetPackageID,
		"target_cpu":         q.TargetCPU,
		"target_mem_gb":      q.TargetMemGB,
		"target_disk_gb":     q.TargetDiskGB,
		"target_bw_mbps":     q.TargetBWMbps,
		"current_monthly":    q.CurrentMonthly,
		"target_monthly":     q.TargetMonthly,
		"charge_amount":      q.ChargeAmount,
		"refund_amount":      q.RefundAmount,
		"refund_to_wallet":   q.RefundToWallet,
	}
}

type ServerStatus struct {
	Hostname        string
	OS              string
	Platform        string
	KernelVersion   string
	UptimeSeconds   uint64
	CPUModel        string
	CPUCores        int
	CPUUsagePercent float64
	MemTotal        uint64
	MemUsed         uint64
	MemUsedPercent  float64
	DiskTotal       uint64
	DiskUsed        uint64
	DiskUsedPercent float64
}

type SystemInfoProvider interface {
	Status(ctx context.Context) (ServerStatus, error)
}

type RobotWebhookConfig struct {
	Name    string   `json:"name"`
	URL     string   `json:"url"`
	Secret  string   `json:"secret"`
	Enabled bool     `json:"enabled"`
	Events  []string `json:"events"`
}

const (
	CodeComplexityDigits  = "digits"
	CodeComplexityLetters = "letters"
	CodeComplexityAlnum   = "alnum"
)
