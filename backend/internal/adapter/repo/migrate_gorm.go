package repo

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// migrateGorm creates the schema for non-sqlite databases.
// The runtime repository uses database/sql with portable SQL, so column names must match the queries.
func migrateGorm(db *gorm.DB) error {
	models := []any{
		&userRow{},
		&captchaRow{},
		&verificationCodeRow{},
		&goodsTypeRow{},
		&regionRow{},
		&planGroupRow{},
		&packageRow{},
		&systemImageRow{},
		&lineSystemImageRow{},
		&cartItemRow{},
		&orderRow{},
		&orderItemRow{},
		&vpsInstanceRow{},
		&orderEventRow{},
		&adminAuditLogRow{},
		&apiKeyRow{},
		&settingRow{},
		&emailTemplateRow{},
		&orderPaymentRow{},
		&billingCycleRow{},
		&automationLogRow{},
		&provisionJobRow{},
		&resizeTaskRow{},
		&integrationSyncLogRow{},
		&permissionGroupRow{},
		&passwordResetTokenRow{},
		&permissionRow{},
		&cmsCategoryRow{},
		&cmsPostRow{},
		&cmsBlockRow{},
		&uploadRow{},
		&ticketRow{},
		&ticketMessageRow{},
		&ticketResourceRow{},
		&walletRow{},
		&walletTransactionRow{},
		&walletOrderRow{},
		&scheduledTaskRunRow{},
		&notificationRow{},
		&pushTokenRow{},
		&realnameVerificationRow{},
		&pluginInstallationRow{},
		&pluginPaymentMethodRow{},
		&probeNodeRow{},
		&probeEnrollTokenRow{},
		&probeStatusEventRow{},
		&probeLogSessionRow{},
	}
	if err := db.AutoMigrate(models...); err != nil {
		return err
	}
	if db.Dialector != nil && db.Dialector.Name() == "mysql" {
		if err := fixMySQLPartialUniqueIndexes(db); err != nil {
			return err
		}
		if err := fixMySQLTextColumns(db); err != nil {
			return err
		}
	}
	if err := repairTimestampNulls(db, models); err != nil {
		return err
	}
	return nil
}

func repairTimestampNulls(db *gorm.DB, models []any) error {
	for _, model := range models {
		if db.Migrator().HasColumn(model, "created_at") {
			if err := db.Model(model).
				Where("created_at IS NULL").
				Update("created_at", clause.Expr{SQL: "CURRENT_TIMESTAMP"}).Error; err != nil {
				return err
			}
		}
		if db.Migrator().HasColumn(model, "updated_at") {
			if err := db.Model(model).
				Where("updated_at IS NULL").
				Update("updated_at", clause.Expr{SQL: "CURRENT_TIMESTAMP"}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// MySQL does not support partial unique indexes. We keep the same index names but make them non-unique.
func fixMySQLPartialUniqueIndexes(db *gorm.DB) error {
	if db.Migrator().HasIndex(&goodsTypeRow{}, "idx_goods_types_code_unique") {
		if err := db.Exec("DROP INDEX idx_goods_types_code_unique ON goods_types").Error; err != nil {
			return err
		}
	}
	if err := db.Exec("CREATE INDEX idx_goods_types_code_unique ON goods_types(code)").Error; err != nil {
		return err
	}

	if db.Migrator().HasIndex(&planGroupRow{}, "idx_plan_groups_gt_line_unique") {
		if err := db.Exec("DROP INDEX idx_plan_groups_gt_line_unique ON plan_groups").Error; err != nil {
			return err
		}
	}
	if err := db.Exec("CREATE INDEX idx_plan_groups_gt_line_unique ON plan_groups(goods_type_id, line_id)").Error; err != nil {
		return err
	}

	if db.Migrator().HasIndex(&packageRow{}, "idx_packages_gt_product_unique") {
		if err := db.Exec("DROP INDEX idx_packages_gt_product_unique ON packages").Error; err != nil {
			return err
		}
	}
	if err := db.Exec("CREATE INDEX idx_packages_gt_product_unique ON packages(goods_type_id, plan_group_id, product_id)").Error; err != nil {
		return err
	}

	return nil
}

func fixMySQLTextColumns(db *gorm.DB) error {
	stmts := []string{
		"ALTER TABLE cms_blocks MODIFY COLUMN content_json LONGTEXT NOT NULL",
		"ALTER TABLE cms_blocks MODIFY COLUMN custom_html LONGTEXT NOT NULL",
		"ALTER TABLE cms_posts MODIFY COLUMN content_html LONGTEXT NOT NULL",
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

type userRow struct {
	ID                int64     `gorm:"primaryKey;autoIncrement;column:id"`
	Username          string    `gorm:"size:191;column:username;not null;uniqueIndex"`
	Email             string    `gorm:"size:191;column:email;not null;uniqueIndex"`
	QQ                string    `gorm:"size:32;column:qq"`
	PasswordHash      string    `gorm:"column:password_hash;not null"`
	Role              string    `gorm:"column:role;not null"`
	Status            string    `gorm:"column:status;not null"`
	Avatar            string    `gorm:"size:1024;column:avatar"`
	Phone             string    `gorm:"size:32;column:phone"`
	Bio               string    `gorm:"size:512;column:bio"`
	Intro             string    `gorm:"size:1024;column:intro"`
	PermissionGroupID *int64    `gorm:"column:permission_group_id"`
	CreatedAt         time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt         time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
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
	ID                int64     `gorm:"primaryKey;autoIncrement;column:id"`
	GoodsTypeID       int64     `gorm:"column:goods_type_id;not null;default:0;index;uniqueIndex:idx_packages_gt_product_unique,where:product_id > 0"`
	PlanGroupID       int64     `gorm:"column:plan_group_id;not null;index;uniqueIndex:idx_packages_gt_product_unique,where:product_id > 0"`
	ProductID         int64     `gorm:"column:product_id;not null;default:0;uniqueIndex:idx_packages_gt_product_unique,where:product_id > 0"`
	Name              string    `gorm:"column:name;not null"`
	Cores             int       `gorm:"column:cores;not null"`
	MemoryGB          int       `gorm:"column:memory_gb;not null"`
	DiskGB            int       `gorm:"column:disk_gb;not null"`
	BandwidthMbps     int       `gorm:"column:bandwidth_mbps;not null"`
	CPUModel          string    `gorm:"column:cpu_model;not null"`
	MonthlyPrice      int64     `gorm:"column:monthly_price;not null"`
	PortNum           int       `gorm:"column:port_num;not null;default:30"`
	SortOrder         int       `gorm:"column:sort_order;not null;default:0"`
	Active            int       `gorm:"column:active;not null;default:1"`
	Visible           int       `gorm:"column:visible;not null;default:1"`
	CapacityRemaining int       `gorm:"column:capacity_remaining;not null;default:-1"`
	CreatedAt         time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt         time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
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
	Status         string     `gorm:"column:status;not null"`
	TotalAmount    int64      `gorm:"column:total_amount;not null"`
	Currency       string     `gorm:"column:currency;not null"`
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

func (cmsCategoryRow) TableName() string { return "cms_categories" }

type cmsPostRow struct {
	ID          int64      `gorm:"primaryKey;autoIncrement;column:id"`
	CategoryID  int64      `gorm:"column:category_id;not null;index"`
	Title       string     `gorm:"column:title;not null"`
	Slug        string     `gorm:"size:191;column:slug;not null;uniqueIndex"`
	Summary     string     `gorm:"column:summary;not null;default:''"`
	ContentHTML string     `gorm:"type:longtext;column:content_html;not null"`
	CoverURL    string     `gorm:"column:cover_url;not null;default:''"`
	Lang        string     `gorm:"size:191;column:lang;not null;default:zh-CN;index"`
	Status      string     `gorm:"column:status;not null;default:draft"`
	Pinned      int        `gorm:"column:pinned;not null;default:0"`
	SortOrder   int        `gorm:"column:sort_order;not null;default:0"`
	PublishedAt *time.Time `gorm:"column:published_at"`
	CreatedAt   time.Time  `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (cmsPostRow) TableName() string { return "cms_posts" }

type cmsBlockRow struct {
	ID          int64     `gorm:"primaryKey;autoIncrement;column:id"`
	Page        string    `gorm:"size:191;column:page;not null;index"`
	Type        string    `gorm:"column:type;not null"`
	Title       string    `gorm:"column:title;not null;default:''"`
	Subtitle    string    `gorm:"column:subtitle;not null;default:''"`
	ContentJSON string    `gorm:"type:longtext;column:content_json;not null"`
	CustomHTML  string    `gorm:"type:longtext;column:custom_html;not null"`
	Lang        string    `gorm:"column:lang;not null;default:zh-CN"`
	Visible     int       `gorm:"column:visible;not null;default:1"`
	SortOrder   int       `gorm:"column:sort_order;not null;default:0"`
	CreatedAt   time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt   time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (cmsBlockRow) TableName() string { return "cms_blocks" }

type uploadRow struct {
	ID         int64     `gorm:"primaryKey;autoIncrement;column:id"`
	Name       string    `gorm:"column:name;not null"`
	Path       string    `gorm:"column:path;not null"`
	URL        string    `gorm:"column:url;not null"`
	Mime       string    `gorm:"column:mime;not null"`
	Size       int64     `gorm:"column:size;not null"`
	UploaderID int64     `gorm:"column:uploader_id;not null;index"`
	CreatedAt  time.Time `gorm:"column:created_at;not null;autoCreateTime"`
}

func (uploadRow) TableName() string { return "uploads" }

type ticketRow struct {
	ID            int64      `gorm:"primaryKey;autoIncrement;column:id"`
	UserID        int64      `gorm:"column:user_id;not null;index"`
	Subject       string     `gorm:"size:240;column:subject;not null"`
	Status        string     `gorm:"size:191;column:status;not null;default:open;index"`
	LastReplyAt   *time.Time `gorm:"column:last_reply_at"`
	LastReplyBy   *int64     `gorm:"column:last_reply_by"`
	LastReplyRole string     `gorm:"column:last_reply_role;not null;default:user"`
	ClosedAt      *time.Time `gorm:"column:closed_at"`
	CreatedAt     time.Time  `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (ticketRow) TableName() string { return "tickets" }

type ticketMessageRow struct {
	ID         int64     `gorm:"primaryKey;autoIncrement;column:id"`
	TicketID   int64     `gorm:"column:ticket_id;not null;index"`
	SenderID   int64     `gorm:"column:sender_id;not null"`
	SenderRole string    `gorm:"column:sender_role;not null"`
	SenderName string    `gorm:"column:sender_name"`
	SenderQQ   string    `gorm:"size:32;column:sender_qq"`
	Content    string    `gorm:"size:10000;column:content;not null"`
	CreatedAt  time.Time `gorm:"column:created_at;not null;autoCreateTime"`
}

func (ticketMessageRow) TableName() string { return "ticket_messages" }

type ticketResourceRow struct {
	ID           int64     `gorm:"primaryKey;autoIncrement;column:id"`
	TicketID     int64     `gorm:"column:ticket_id;not null;index"`
	ResourceType string    `gorm:"column:resource_type;not null"`
	ResourceID   int64     `gorm:"column:resource_id;not null;default:0"`
	ResourceName string    `gorm:"size:128;column:resource_name;not null;default:''"`
	CreatedAt    time.Time `gorm:"column:created_at;not null;autoCreateTime"`
}

func (ticketResourceRow) TableName() string { return "ticket_resources" }

type walletRow struct {
	ID        int64     `gorm:"primaryKey;autoIncrement;column:id"`
	UserID    int64     `gorm:"column:user_id;not null;uniqueIndex"`
	Balance   int64     `gorm:"column:balance;not null;default:0"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (walletRow) TableName() string { return "user_wallets" }

type walletTransactionRow struct {
	ID        int64     `gorm:"primaryKey;autoIncrement;column:id"`
	UserID    int64     `gorm:"column:user_id;not null;index"`
	Amount    int64     `gorm:"column:amount;not null"`
	Type      string    `gorm:"column:type;not null"`
	RefType   string    `gorm:"column:ref_type;not null"`
	RefID     int64     `gorm:"column:ref_id;not null;default:0"`
	Note      string    `gorm:"column:note;not null;default:''"`
	CreatedAt time.Time `gorm:"column:created_at;not null;autoCreateTime"`
}

func (walletTransactionRow) TableName() string { return "wallet_transactions" }

type walletOrderRow struct {
	ID           int64     `gorm:"primaryKey;autoIncrement;column:id"`
	UserID       int64     `gorm:"column:user_id;not null;index"`
	Type         string    `gorm:"column:type;not null"`
	Amount       int64     `gorm:"column:amount;not null"`
	Currency     string    `gorm:"column:currency;not null;default:CNY"`
	Status       string    `gorm:"column:status;not null"`
	Note         string    `gorm:"size:1000;column:note;not null;default:''"`
	MetaJSON     string    `gorm:"column:meta_json;not null;default:''"`
	ReviewedBy   *int64    `gorm:"column:reviewed_by"`
	ReviewReason string    `gorm:"size:1000;column:review_reason"`
	CreatedAt    time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt    time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (walletOrderRow) TableName() string { return "wallet_orders" }

type scheduledTaskRunRow struct {
	ID          int64      `gorm:"primaryKey;autoIncrement;column:id"`
	TaskKey     string     `gorm:"column:task_key;not null"`
	Status      string     `gorm:"column:status;not null"`
	StartedAt   time.Time  `gorm:"column:started_at;not null"`
	FinishedAt  *time.Time `gorm:"column:finished_at"`
	DurationSec int        `gorm:"column:duration_sec;not null;default:0"`
	Message     string     `gorm:"column:message;not null;default:''"`
	CreatedAt   time.Time  `gorm:"column:created_at;not null;autoCreateTime"`
}

func (scheduledTaskRunRow) TableName() string { return "scheduled_task_runs" }

type notificationRow struct {
	ID        int64      `gorm:"primaryKey;autoIncrement;column:id"`
	UserID    int64      `gorm:"column:user_id;not null;index"`
	Type      string     `gorm:"column:type;not null"`
	Title     string     `gorm:"column:title;not null"`
	Content   string     `gorm:"column:content;not null"`
	ReadAt    *time.Time `gorm:"column:read_at"`
	CreatedAt time.Time  `gorm:"column:created_at;not null;autoCreateTime"`
}

func (notificationRow) TableName() string { return "notifications" }

type pushTokenRow struct {
	ID        int64     `gorm:"primaryKey;autoIncrement;column:id"`
	UserID    int64     `gorm:"column:user_id;not null;uniqueIndex:idx_push_tokens_user_token,priority:1;index:idx_push_tokens_user"`
	Platform  string    `gorm:"column:platform;not null"`
	Token     string    `gorm:"size:191;column:token;not null;uniqueIndex:idx_push_tokens_user_token,priority:2"`
	DeviceID  string    `gorm:"column:device_id;not null;default:''"`
	CreatedAt time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (pushTokenRow) TableName() string { return "push_tokens" }

type realnameVerificationRow struct {
	ID         int64      `gorm:"primaryKey;autoIncrement;column:id"`
	UserID     int64      `gorm:"column:user_id;not null;index"`
	RealName   string     `gorm:"column:real_name;not null"`
	IDNumber   string     `gorm:"column:id_number;not null"`
	Status     string     `gorm:"column:status;not null"`
	Provider   string     `gorm:"column:provider;not null"`
	Reason     string     `gorm:"column:reason;not null;default:''"`
	CreatedAt  time.Time  `gorm:"column:created_at;not null;autoCreateTime"`
	VerifiedAt *time.Time `gorm:"column:verified_at"`
}

func (realnameVerificationRow) TableName() string { return "realname_verifications" }

type pluginInstallationRow struct {
	ID              int64     `gorm:"primaryKey;autoIncrement;column:id"`
	Category        string    `gorm:"size:191;column:category;not null;uniqueIndex:idx_plugin_installations_cat_id_instance"`
	PluginID        string    `gorm:"size:191;column:plugin_id;not null;uniqueIndex:idx_plugin_installations_cat_id_instance"`
	InstanceID      string    `gorm:"size:191;column:instance_id;not null;uniqueIndex:idx_plugin_installations_cat_id_instance"`
	Enabled         int       `gorm:"column:enabled;not null;default:0"`
	SignatureStatus string    `gorm:"column:signature_status;not null;default:unsigned"`
	ConfigCipher    string    `gorm:"column:config_cipher;not null"`
	CreatedAt       time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt       time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (pluginInstallationRow) TableName() string { return "plugin_installations" }

type pluginPaymentMethodRow struct {
	ID         int64     `gorm:"primaryKey;autoIncrement;column:id"`
	Category   string    `gorm:"size:191;column:category;not null;uniqueIndex:idx_plugin_payment_methods_unique"`
	PluginID   string    `gorm:"size:191;column:plugin_id;not null;uniqueIndex:idx_plugin_payment_methods_unique"`
	InstanceID string    `gorm:"size:191;column:instance_id;not null;uniqueIndex:idx_plugin_payment_methods_unique"`
	Method     string    `gorm:"size:191;column:method;not null;uniqueIndex:idx_plugin_payment_methods_unique"`
	Enabled    int       `gorm:"column:enabled;not null;default:1"`
	CreatedAt  time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt  time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (pluginPaymentMethodRow) TableName() string { return "plugin_payment_methods" }

type probeNodeRow struct {
	ID               int64      `gorm:"primaryKey;autoIncrement;column:id"`
	Name             string     `gorm:"column:name;not null"`
	AgentID          string     `gorm:"size:191;column:agent_id;not null;uniqueIndex"`
	SecretHash       string     `gorm:"size:191;column:secret_hash;not null"`
	Status           string     `gorm:"size:32;column:status;not null;default:offline;index:idx_probe_nodes_status"`
	OSType           string     `gorm:"size:32;column:os_type;not null;default:''"`
	TagsJSON         string     `gorm:"column:tags_json;not null;default:'[]'"`
	LastHeartbeatAt  *time.Time `gorm:"column:last_heartbeat_at;index:idx_probe_nodes_heartbeat"`
	LastSnapshotAt   *time.Time `gorm:"column:last_snapshot_at"`
	LastSnapshotJSON string     `gorm:"column:last_snapshot_json;not null;default:'{}'"`
	CreatedAt        time.Time  `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt        time.Time  `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (probeNodeRow) TableName() string { return "probe_nodes" }

type probeEnrollTokenRow struct {
	ID        int64      `gorm:"primaryKey;autoIncrement;column:id"`
	ProbeID   int64      `gorm:"column:probe_id;not null;index:idx_probe_enroll_tokens_probe"`
	TokenHash string     `gorm:"size:191;column:token_hash;not null;uniqueIndex"`
	ExpiresAt time.Time  `gorm:"column:expires_at;not null;index:idx_probe_enroll_tokens_expires"`
	UsedAt    *time.Time `gorm:"column:used_at"`
	CreatedAt time.Time  `gorm:"column:created_at;not null;autoCreateTime"`
}

func (probeEnrollTokenRow) TableName() string { return "probe_enroll_tokens" }

type probeStatusEventRow struct {
	ID        int64     `gorm:"primaryKey;autoIncrement;column:id"`
	ProbeID   int64     `gorm:"column:probe_id;not null;index:idx_probe_status_events_probe_at,priority:1"`
	Status    string    `gorm:"size:32;column:status;not null"`
	At        time.Time `gorm:"column:at;not null;index:idx_probe_status_events_probe_at,priority:2"`
	Reason    string    `gorm:"column:reason;not null;default:''"`
	CreatedAt time.Time `gorm:"column:created_at;not null;autoCreateTime"`
}

func (probeStatusEventRow) TableName() string { return "probe_status_events" }

type probeLogSessionRow struct {
	ID         int64      `gorm:"primaryKey;autoIncrement;column:id"`
	ProbeID    int64      `gorm:"column:probe_id;not null;index:idx_probe_log_sessions_probe"`
	OperatorID int64      `gorm:"column:operator_id;not null;default:0"`
	Source     string     `gorm:"column:source;not null;default:''"`
	Status     string     `gorm:"size:32;column:status;not null;default:running"`
	StartedAt  time.Time  `gorm:"column:started_at;not null"`
	EndedAt    *time.Time `gorm:"column:ended_at"`
	CreatedAt  time.Time  `gorm:"column:created_at;not null;autoCreateTime"`
}

func (probeLogSessionRow) TableName() string { return "probe_log_sessions" }
