package repo

import (
	"time"
)

type userRow struct {
	ID                   int64      `gorm:"primaryKey;autoIncrement;column:id"`
	Username             string     `gorm:"size:191;column:username;not null;uniqueIndex"`
	Email                *string    `gorm:"size:191;column:email;uniqueIndex"`
	QQ                   string     `gorm:"size:32;column:qq"`
	PasswordHash         string     `gorm:"column:password_hash;not null"`
	Role                 string     `gorm:"column:role;not null"`
	Status               string     `gorm:"column:status;not null"`
	Avatar               string     `gorm:"size:1024;column:avatar"`
	Phone                string     `gorm:"size:32;column:phone"`
	LastLoginIP          string     `gorm:"size:64;column:last_login_ip"`
	LastLoginAt          *time.Time `gorm:"column:last_login_at"`
	LastLoginCity        string     `gorm:"size:128;column:last_login_city"`
	LastLoginTZ          string     `gorm:"size:64;column:last_login_tz"`
	TOTPEnabled          int        `gorm:"column:totp_enabled;not null;default:0"`
	TOTPSecretEnc        string     `gorm:"type:text;column:totp_secret_enc"`
	TOTPPendingSecretEnc string     `gorm:"type:text;column:totp_pending_secret_enc"`
	Bio                  string     `gorm:"size:512;column:bio"`
	Intro                string     `gorm:"size:1024;column:intro"`
	PermissionGroupID    *int64     `gorm:"column:permission_group_id"`
	UserTierGroupID      *int64     `gorm:"column:user_tier_group_id;index"`
	UserTierExpireAt     *time.Time `gorm:"column:user_tier_expire_at;index"`
	CreatedAt            time.Time  `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt            time.Time  `gorm:"column:updated_at;not null;autoUpdateTime"`
	PasswordChangedAt    *time.Time `gorm:"column:password_changed_at"`
}

func (userRow) TableName() string { return "users" }

type captchaRow struct {
	ID        string    `gorm:"size:191;primaryKey;column:id"`
	CodeHash  string    `gorm:"column:code_hash;not null"`
	ExpiresAt time.Time `gorm:"column:expires_at;not null"`
	CreatedAt time.Time `gorm:"column:created_at;not null;autoCreateTime"`
}

func (captchaRow) TableName() string { return "captchas" }

type verificationCodeRow struct {
	ID        int64     `gorm:"primaryKey;autoIncrement;column:id"`
	Channel   string    `gorm:"size:191;column:channel;not null;index:idx_verification_codes_receiver,priority:1"`
	Receiver  string    `gorm:"size:191;column:receiver;not null;index:idx_verification_codes_receiver,priority:2"`
	Purpose   string    `gorm:"size:191;column:purpose;not null;index:idx_verification_codes_receiver,priority:3"`
	CodeHash  string    `gorm:"column:code_hash;not null"`
	ExpiresAt time.Time `gorm:"column:expires_at;not null;index:idx_verification_codes_expires"`
	CreatedAt time.Time `gorm:"column:created_at;not null;autoCreateTime"`
}

func (verificationCodeRow) TableName() string { return "verification_codes" }

type goodsTypeRow struct {
	ID                   int64     `gorm:"primaryKey;autoIncrement;column:id"`
	Code                 string    `gorm:"size:191;column:code;uniqueIndex:idx_goods_types_code_unique,where:code <> ''"`
	Name                 string    `gorm:"column:name;not null"`
	Active               int       `gorm:"column:active;not null;default:1"`
	SortOrder            int       `gorm:"column:sort_order;not null;default:0"`
	AutomationCategory   string    `gorm:"size:191;column:automation_category;not null;default:automation;uniqueIndex:idx_goods_types_automation_unique"`
	AutomationPluginID   string    `gorm:"size:191;column:automation_plugin_id;not null;default:'';uniqueIndex:idx_goods_types_automation_unique"`
	AutomationInstanceID string    `gorm:"size:191;column:automation_instance_id;not null;default:'';uniqueIndex:idx_goods_types_automation_unique"`
	CreatedAt            time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt            time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (goodsTypeRow) TableName() string { return "goods_types" }

type regionRow struct {
	ID          int64     `gorm:"primaryKey;autoIncrement;column:id"`
	GoodsTypeID int64     `gorm:"column:goods_type_id;not null;default:0;index;uniqueIndex:idx_regions_gt_code_unique"`
	Code        string    `gorm:"size:191;column:code;not null;uniqueIndex:idx_regions_gt_code_unique"`
	Name        string    `gorm:"column:name;not null"`
	Active      int       `gorm:"column:active;not null;default:1"`
	CreatedAt   time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt   time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (regionRow) TableName() string { return "regions" }

type planGroupRow struct {
	ID                int64     `gorm:"primaryKey;autoIncrement;column:id"`
	GoodsTypeID       int64     `gorm:"column:goods_type_id;not null;default:0;index;uniqueIndex:idx_plan_groups_gt_line_unique,where:line_id > 0"`
	RegionID          int64     `gorm:"column:region_id;not null;index"`
	Name              string    `gorm:"column:name;not null"`
	LineID            int64     `gorm:"column:line_id;not null;default:0;index;uniqueIndex:idx_plan_groups_gt_line_unique,where:line_id > 0"`
	UnitCore          int64     `gorm:"column:unit_core;not null"`
	UnitMem           int64     `gorm:"column:unit_mem;not null"`
	UnitDisk          int64     `gorm:"column:unit_disk;not null"`
	UnitBW            int64     `gorm:"column:unit_bw;not null"`
	AddCoreMin        int       `gorm:"column:add_core_min;not null;default:0"`
	AddCoreMax        int       `gorm:"column:add_core_max;not null;default:0"`
	AddCoreStep       int       `gorm:"column:add_core_step;not null;default:1"`
	AddMemMin         int       `gorm:"column:add_mem_min;not null;default:0"`
	AddMemMax         int       `gorm:"column:add_mem_max;not null;default:0"`
	AddMemStep        int       `gorm:"column:add_mem_step;not null;default:1"`
	AddDiskMin        int       `gorm:"column:add_disk_min;not null;default:0"`
	AddDiskMax        int       `gorm:"column:add_disk_max;not null;default:0"`
	AddDiskStep       int       `gorm:"column:add_disk_step;not null;default:1"`
	AddBWMin          int       `gorm:"column:add_bw_min;not null;default:0"`
	AddBWMax          int       `gorm:"column:add_bw_max;not null;default:0"`
	AddBWStep         int       `gorm:"column:add_bw_step;not null;default:1"`
	Active            int       `gorm:"column:active;not null;default:1"`
	Visible           int       `gorm:"column:visible;not null;default:1"`
	CapacityRemaining int       `gorm:"column:capacity_remaining;not null;default:-1"`
	SortOrder         int       `gorm:"column:sort_order;not null;default:0"`
	CreatedAt         time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt         time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (planGroupRow) TableName() string { return "plan_groups" }

type packageRow struct {
	ID                   int64     `gorm:"primaryKey;autoIncrement;column:id"`
	GoodsTypeID          int64     `gorm:"column:goods_type_id;not null;default:0;index;uniqueIndex:idx_packages_gt_product_unique,where:product_id > 0"`
	PlanGroupID          int64     `gorm:"column:plan_group_id;not null;index;uniqueIndex:idx_packages_gt_product_unique,where:product_id > 0"`
	ProductID            int64     `gorm:"column:product_id;not null;default:0;uniqueIndex:idx_packages_gt_product_unique,where:product_id > 0"`
	IntegrationPackageID int64     `gorm:"column:integration_package_id;not null;default:0;index;uniqueIndex:idx_packages_gt_integration_unique,where:integration_package_id > 0"`
	Name                 string    `gorm:"column:name;not null"`
	Cores                int       `gorm:"column:cores;not null"`
	MemoryGB             int       `gorm:"column:memory_gb;not null"`
	DiskGB               int       `gorm:"column:disk_gb;not null"`
	BandwidthMbps        int       `gorm:"column:bandwidth_mbps;not null"`
	CPUModel             string    `gorm:"column:cpu_model;not null"`
	MonthlyPrice         int64     `gorm:"column:monthly_price;not null"`
	PortNum              int       `gorm:"column:port_num;not null;default:30"`
	SortOrder            int       `gorm:"column:sort_order;not null;default:0"`
	Active               int       `gorm:"column:active;not null;default:1"`
	Visible              int       `gorm:"column:visible;not null;default:1"`
	CapacityRemaining    int       `gorm:"column:capacity_remaining;not null;default:-1"`
	CreatedAt            time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt            time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (packageRow) TableName() string { return "packages" }

type systemImageRow struct {
	ID        int64     `gorm:"primaryKey;autoIncrement;column:id"`
	ImageID   int64     `gorm:"column:image_id;not null;default:0;index"`
	Name      string    `gorm:"column:name;not null"`
	Type      string    `gorm:"column:type;not null"`
	Enabled   int       `gorm:"column:enabled;not null;default:1"`
	CreatedAt time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (systemImageRow) TableName() string { return "system_images" }

type lineSystemImageRow struct {
	ID            int64     `gorm:"primaryKey;autoIncrement;column:id"`
	LineID        int64     `gorm:"column:line_id;not null;index;uniqueIndex:idx_line_system_images_unique"`
	SystemImageID int64     `gorm:"column:system_image_id;not null;uniqueIndex:idx_line_system_images_unique"`
	CreatedAt     time.Time `gorm:"column:created_at;not null;autoCreateTime"`
}

func (lineSystemImageRow) TableName() string { return "line_system_images" }

type cartItemRow struct {
	ID        int64     `gorm:"primaryKey;autoIncrement;column:id"`
	UserID    int64     `gorm:"column:user_id;not null;index"`
	PackageID int64     `gorm:"column:package_id;not null"`
	SystemID  int64     `gorm:"column:system_id;not null"`
	SpecJSON  string    `gorm:"column:spec_json;not null"`
	Qty       int       `gorm:"column:qty;not null;default:1"`
	Amount    int64     `gorm:"column:amount;not null"`
	CreatedAt time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (cartItemRow) TableName() string { return "cart_items" }

type orderRow struct {
	ID             int64      `gorm:"primaryKey;autoIncrement;column:id"`
	UserID         int64      `gorm:"column:user_id;not null;index;uniqueIndex:idx_orders_idem"`
	OrderNo        string     `gorm:"size:191;column:order_no;not null;uniqueIndex"`
	Source         string     `gorm:"size:64;column:source;not null;default:user_ui;index"`
	Status         string     `gorm:"column:status;not null"`
	TotalAmount    int64      `gorm:"column:total_amount;not null"`
	Currency       string     `gorm:"column:currency;not null"`
	CouponID       *int64     `gorm:"column:coupon_id;index"`
	CouponCode     string     `gorm:"size:128;column:coupon_code;not null;default:'';index"`
	CouponDiscount int64      `gorm:"column:coupon_discount;not null;default:0"`
	IdempotencyKey *string    `gorm:"size:191;column:idempotency_key;uniqueIndex:idx_orders_idem"`
	PendingReason  string     `gorm:"size:1000;column:pending_reason"`
	ApprovedBy     *int64     `gorm:"column:approved_by"`
	ApprovedAt     *time.Time `gorm:"column:approved_at"`
	RejectedReason string     `gorm:"size:1000;column:rejected_reason"`
	CreatedAt      time.Time  `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (orderRow) TableName() string { return "orders" }

type orderItemRow struct {
	ID                   int64     `gorm:"primaryKey;autoIncrement;column:id"`
	OrderID              int64     `gorm:"column:order_id;not null;index"`
	PackageID            int64     `gorm:"column:package_id;not null;default:0"`
	SystemID             int64     `gorm:"column:system_id;not null;default:0"`
	SpecJSON             string    `gorm:"column:spec_json;not null"`
	Qty                  int       `gorm:"column:qty;not null;default:1"`
	Amount               int64     `gorm:"column:amount;not null"`
	Status               string    `gorm:"column:status;not null"`
	GoodsTypeID          int64     `gorm:"column:goods_type_id;not null;default:0;index"`
	AutomationInstanceID string    `gorm:"column:automation_instance_id"`
	Action               string    `gorm:"column:action;not null;default:create"`
	DurationMonths       int       `gorm:"column:duration_months;not null;default:1"`
	CreatedAt            time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt            time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (orderItemRow) TableName() string { return "order_items" }

type vpsInstanceRow struct {
	ID                   int64      `gorm:"primaryKey;autoIncrement;column:id"`
	UserID               int64      `gorm:"column:user_id;not null;index"`
	OrderItemID          int64      `gorm:"column:order_item_id;not null;index"`
	AutomationInstanceID string     `gorm:"column:automation_instance_id;not null"`
	GoodsTypeID          int64      `gorm:"column:goods_type_id;not null;default:0;index"`
	Name                 string     `gorm:"size:128;column:name;not null"`
	Region               string     `gorm:"column:region"`
	RegionID             int64      `gorm:"column:region_id;not null;default:0"`
	LineID               int64      `gorm:"column:line_id;not null;default:0"`
	PackageID            int64      `gorm:"column:package_id;not null;default:0"`
	PackageName          string     `gorm:"column:package_name;not null;default:''"`
	CPU                  int        `gorm:"column:cpu;not null;default:0"`
	MemoryGB             int        `gorm:"column:memory_gb;not null;default:0"`
	DiskGB               int        `gorm:"column:disk_gb;not null;default:0"`
	BandwidthMbps        int        `gorm:"column:bandwidth_mbps;not null;default:0"`
	PortNum              int        `gorm:"column:port_num;not null;default:0"`
	MonthlyPrice         int64      `gorm:"column:monthly_price;not null;default:0"`
	SpecJSON             string     `gorm:"column:spec_json;not null"`
	SystemID             int64      `gorm:"column:system_id;not null"`
	Status               string     `gorm:"column:status;not null"`
	AutomationState      int        `gorm:"column:automation_state;not null;default:0"`
	AdminStatus          string     `gorm:"column:admin_status;not null;default:normal"`
	ExpireAt             *time.Time `gorm:"column:expire_at"`
	PanelURLCache        string     `gorm:"column:panel_url_cache"`
	AccessInfoJSON       string     `gorm:"column:access_info_json"`
	LastEmergencyRenewAt *time.Time `gorm:"column:last_emergency_renew_at"`
	CreatedAt            time.Time  `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt            time.Time  `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (vpsInstanceRow) TableName() string { return "vps_instances" }

type orderEventRow struct {
	ID        int64     `gorm:"primaryKey;autoIncrement;column:id"`
	OrderID   int64     `gorm:"column:order_id;not null;uniqueIndex:idx_order_events_seq"`
	Seq       int64     `gorm:"column:seq;not null;uniqueIndex:idx_order_events_seq"`
	Type      string    `gorm:"column:type;not null"`
	DataJSON  string    `gorm:"column:data_json;not null"`
	CreatedAt time.Time `gorm:"column:created_at;not null;autoCreateTime"`
}

func (orderEventRow) TableName() string { return "order_events" }

type adminAuditLogRow struct {
	ID         int64     `gorm:"primaryKey;autoIncrement;column:id"`
	AdminID    int64     `gorm:"column:admin_id;not null"`
	Action     string    `gorm:"column:action;not null"`
	TargetType string    `gorm:"column:target_type;not null"`
	TargetID   string    `gorm:"column:target_id;not null"`
	DetailJSON string    `gorm:"column:detail_json;not null"`
	CreatedAt  time.Time `gorm:"column:created_at;not null;autoCreateTime"`
}

func (adminAuditLogRow) TableName() string { return "admin_audit_logs" }

type apiKeyRow struct {
	ID                int64      `gorm:"primaryKey;autoIncrement;column:id"`
	Name              string     `gorm:"column:name;not null"`
	KeyHash           string     `gorm:"size:191;column:key_hash;not null;uniqueIndex"`
	Status            string     `gorm:"column:status;not null"`
	ScopesJSON        string     `gorm:"column:scopes_json;not null"`
	PermissionGroupID *int64     `gorm:"column:permission_group_id"`
	CreatedAt         time.Time  `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt         time.Time  `gorm:"column:updated_at;not null;autoUpdateTime"`
	LastUsedAt        *time.Time `gorm:"column:last_used_at"`
}

func (apiKeyRow) TableName() string { return "api_keys" }

type userAPIKeyRow struct {
	ID         int64      `gorm:"primaryKey;autoIncrement;column:id"`
	UserID     int64      `gorm:"column:user_id;not null;index"`
	Name       string     `gorm:"column:name;not null"`
	AKID       string     `gorm:"size:191;column:akid;not null;uniqueIndex"`
	KeyHash    string     `gorm:"size:191;column:key_hash;not null;uniqueIndex"`
	Status     string     `gorm:"column:status;not null"`
	ScopesJSON string     `gorm:"column:scopes_json;not null"`
	CreatedAt  time.Time  `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt  time.Time  `gorm:"column:updated_at;not null;autoUpdateTime"`
	LastUsedAt *time.Time `gorm:"column:last_used_at"`
}

func (userAPIKeyRow) TableName() string { return "user_api_keys" }

type settingRow struct {
	Key       string    `gorm:"size:191;primaryKey;column:key"`
	ValueJSON string    `gorm:"column:value_json;not null"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (settingRow) TableName() string { return "settings" }

type emailTemplateRow struct {
	ID        int64     `gorm:"primaryKey;autoIncrement;column:id"`
	Name      string    `gorm:"size:191;column:name;not null;uniqueIndex"`
	Subject   string    `gorm:"column:subject;not null"`
	Body      string    `gorm:"column:body;not null"`
	Enabled   int       `gorm:"column:enabled;not null;default:1"`
	CreatedAt time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (emailTemplateRow) TableName() string { return "email_templates" }

type orderPaymentRow struct {
	ID             int64     `gorm:"primaryKey;autoIncrement;column:id"`
	OrderID        int64     `gorm:"column:order_id;not null;index;uniqueIndex:idx_order_payments_idem"`
	UserID         int64     `gorm:"column:user_id;not null"`
	Method         string    `gorm:"size:64;column:method;not null"`
	Amount         int64     `gorm:"column:amount;not null"`
	Currency       string    `gorm:"column:currency;not null"`
	TradeNo        string    `gorm:"size:191;column:trade_no;not null;uniqueIndex:idx_order_payments_trade_no"`
	Note           *string   `gorm:"size:1000;column:note"`
	ScreenshotURL  *string   `gorm:"size:1024;column:screenshot_url"`
	Status         string    `gorm:"column:status;not null"`
	IdempotencyKey *string   `gorm:"size:191;column:idempotency_key;uniqueIndex:idx_order_payments_idem"`
	ReviewedBy     *int64    `gorm:"column:reviewed_by"`
	ReviewReason   string    `gorm:"size:1000;column:review_reason"`
	CreatedAt      time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt      time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (orderPaymentRow) TableName() string { return "order_payments" }

type billingCycleRow struct {
	ID         int64     `gorm:"primaryKey;autoIncrement;column:id"`
	Name       string    `gorm:"column:name;not null"`
	Months     int       `gorm:"column:months;not null"`
	Multiplier float64   `gorm:"column:multiplier;not null"`
	MinQty     int       `gorm:"column:min_qty;not null;default:1"`
	MaxQty     int       `gorm:"column:max_qty;not null;default:36"`
	Active     int       `gorm:"column:active;not null;default:1"`
	SortOrder  int       `gorm:"column:sort_order;not null;default:0"`
	CreatedAt  time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt  time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (billingCycleRow) TableName() string { return "billing_cycles" }

type automationLogRow struct {
	ID           int64     `gorm:"primaryKey;autoIncrement;column:id"`
	OrderID      int64     `gorm:"column:order_id;not null"`
	OrderItemID  int64     `gorm:"column:order_item_id;not null"`
	Action       string    `gorm:"column:action;not null"`
	RequestJSON  string    `gorm:"column:request_json;not null"`
	ResponseJSON string    `gorm:"column:response_json;not null"`
	Success      int       `gorm:"column:success;not null;default:0"`
	Message      string    `gorm:"column:message;not null"`
	CreatedAt    time.Time `gorm:"column:created_at;not null;autoCreateTime"`
}

func (automationLogRow) TableName() string { return "automation_logs" }

type provisionJobRow struct {
	ID          int64     `gorm:"primaryKey;autoIncrement;column:id"`
	OrderID     int64     `gorm:"column:order_id;not null"`
	OrderItemID int64     `gorm:"column:order_item_id;not null;uniqueIndex:idx_provision_jobs_item"`
	HostID      int64     `gorm:"column:host_id;not null"`
	HostName    string    `gorm:"column:host_name;not null"`
	Status      string    `gorm:"column:status;not null"`
	Attempts    int       `gorm:"column:attempts;not null;default:0"`
	NextRunAt   time.Time `gorm:"column:next_run_at;not null"`
	LastError   string    `gorm:"column:last_error;not null;default:''"`
	CreatedAt   time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt   time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (provisionJobRow) TableName() string { return "provision_jobs" }

type resizeTaskRow struct {
	ID          int64      `gorm:"primaryKey;autoIncrement;column:id"`
	VPSID       int64      `gorm:"column:vps_id;not null;index"`
	OrderID     int64      `gorm:"column:order_id;not null"`
	OrderItemID int64      `gorm:"column:order_item_id;not null"`
	Status      string     `gorm:"size:191;column:status;not null;index"`
	ScheduledAt *time.Time `gorm:"column:scheduled_at"`
	StartedAt   *time.Time `gorm:"column:started_at"`
	FinishedAt  *time.Time `gorm:"column:finished_at"`
	CreatedAt   time.Time  `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (resizeTaskRow) TableName() string { return "resize_tasks" }

type integrationSyncLogRow struct {
	ID        int64     `gorm:"primaryKey;autoIncrement;column:id"`
	Target    string    `gorm:"column:target;not null"`
	Mode      string    `gorm:"column:mode;not null"`
	Status    string    `gorm:"column:status;not null"`
	Message   string    `gorm:"column:message;not null"`
	CreatedAt time.Time `gorm:"column:created_at;not null;autoCreateTime"`
}

func (integrationSyncLogRow) TableName() string { return "integration_sync_logs" }

type permissionGroupRow struct {
	ID              int64     `gorm:"primaryKey;autoIncrement;column:id"`
	Name            string    `gorm:"size:191;column:name;not null;uniqueIndex"`
	Description     string    `gorm:"column:description"`
	PermissionsJSON string    `gorm:"column:permissions_json;not null"`
	CreatedAt       time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt       time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (permissionGroupRow) TableName() string { return "permission_groups" }

type passwordResetTokenRow struct {
	ID        int64     `gorm:"primaryKey;autoIncrement;column:id"`
	UserID    int64     `gorm:"column:user_id;not null;index"`
	Token     string    `gorm:"size:191;column:token;not null;uniqueIndex"`
	ExpiresAt time.Time `gorm:"column:expires_at;not null"`
	Used      int       `gorm:"column:used;not null;default:0"`
	CreatedAt time.Time `gorm:"column:created_at;not null;autoCreateTime"`
}

func (passwordResetTokenRow) TableName() string { return "password_reset_tokens" }

type passwordResetTicketRow struct {
	ID        int64     `gorm:"primaryKey;autoIncrement;column:id"`
	UserID    int64     `gorm:"column:user_id;not null;index"`
	Channel   string    `gorm:"size:32;column:channel;not null;index:idx_password_reset_tickets_token,priority:1"`
	Receiver  string    `gorm:"size:191;column:receiver;not null"`
	Token     string    `gorm:"size:191;column:token;not null;uniqueIndex:idx_password_reset_tickets_token,priority:2"`
	ExpiresAt time.Time `gorm:"column:expires_at;not null;index"`
	Used      int       `gorm:"column:used;not null;default:0"`
	CreatedAt time.Time `gorm:"column:created_at;not null;autoCreateTime"`
}

func (passwordResetTicketRow) TableName() string { return "password_reset_tickets" }

type permissionRow struct {
	ID           int64     `gorm:"primaryKey;autoIncrement;column:id"`
	Code         string    `gorm:"size:191;column:code;not null;uniqueIndex"`
	Name         string    `gorm:"column:name;not null"`
	FriendlyName string    `gorm:"column:friendly_name"`
	Category     string    `gorm:"column:category;not null"`
	ParentCode   string    `gorm:"column:parent_code"`
	SortOrder    int       `gorm:"column:sort_order;not null;default:0"`
	CreatedAt    time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt    time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (permissionRow) TableName() string { return "permissions" }

type cmsCategoryRow struct {
	ID        int64     `gorm:"primaryKey;autoIncrement;column:id"`
	Key       string    `gorm:"size:191;column:key;not null;uniqueIndex:idx_cms_categories_key_lang"`
	Name      string    `gorm:"column:name;not null"`
	Lang      string    `gorm:"size:191;column:lang;not null;default:zh-CN;uniqueIndex:idx_cms_categories_key_lang"`
	SortOrder int       `gorm:"column:sort_order;not null;default:0"`
	Visible   int       `gorm:"column:visible;not null;default:1"`
	CreatedAt time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}
