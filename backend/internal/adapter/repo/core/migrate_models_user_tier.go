package repo

import "time"

type userTierGroupRow struct {
	ID                 int64     `gorm:"primaryKey;autoIncrement;column:id"`
	Name               string    `gorm:"size:191;column:name;not null;uniqueIndex"`
	Color              string    `gorm:"size:32;column:color;not null;default:#1677ff"`
	Icon               string    `gorm:"size:128;column:icon;not null;default:badge"`
	Priority           int       `gorm:"column:priority;not null;default:0;index"`
	AutoApproveEnabled int       `gorm:"column:auto_approve_enabled;not null;default:0;index"`
	IsDefault          int       `gorm:"column:is_default;not null;default:0;index"`
	CreatedAt          time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt          time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (userTierGroupRow) TableName() string { return "user_tier_groups" }

type userTierDiscountRuleRow struct {
	ID               int64     `gorm:"primaryKey;autoIncrement;column:id"`
	GroupID          int64     `gorm:"column:group_id;not null;index;uniqueIndex:idx_user_tier_rule_scope_unique,priority:1"`
	Scope            string    `gorm:"size:64;column:scope;not null;uniqueIndex:idx_user_tier_rule_scope_unique,priority:2"`
	GoodsTypeID      int64     `gorm:"column:goods_type_id;not null;default:0;uniqueIndex:idx_user_tier_rule_scope_unique,priority:3"`
	RegionID         int64     `gorm:"column:region_id;not null;default:0;uniqueIndex:idx_user_tier_rule_scope_unique,priority:4"`
	PlanGroupID      int64     `gorm:"column:plan_group_id;not null;default:0;uniqueIndex:idx_user_tier_rule_scope_unique,priority:5"`
	PackageID        int64     `gorm:"column:package_id;not null;default:0;uniqueIndex:idx_user_tier_rule_scope_unique,priority:6"`
	DiscountPermille int       `gorm:"column:discount_permille;not null;default:0"`
	FixedPrice       *int64    `gorm:"column:fixed_price"`
	AddCorePermille  int       `gorm:"column:add_core_permille;not null;default:0"`
	AddMemPermille   int       `gorm:"column:add_mem_permille;not null;default:0"`
	AddDiskPermille  int       `gorm:"column:add_disk_permille;not null;default:0"`
	AddBWPermille    int       `gorm:"column:add_bw_permille;not null;default:0"`
	CreatedAt        time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt        time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (userTierDiscountRuleRow) TableName() string { return "user_tier_discount_rules" }

type userTierAutoRuleRow struct {
	ID             int64     `gorm:"primaryKey;autoIncrement;column:id"`
	GroupID        int64     `gorm:"column:group_id;not null;index"`
	DurationDays   int       `gorm:"column:duration_days;not null;default:-1"`
	ConditionsJSON string    `gorm:"type:text;column:conditions_json;not null"`
	SortOrder      int       `gorm:"column:sort_order;not null;default:0"`
	CreatedAt      time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt      time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (userTierAutoRuleRow) TableName() string { return "user_tier_auto_rules" }

type userTierMembershipRow struct {
	UserID    int64      `gorm:"primaryKey;column:user_id"`
	GroupID   int64      `gorm:"column:group_id;not null;index"`
	Source    string     `gorm:"size:32;column:source;not null;default:auto"`
	ExpiresAt *time.Time `gorm:"column:expires_at;index"`
	UpdatedAt time.Time  `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (userTierMembershipRow) TableName() string { return "user_tier_memberships" }

type userTierPriceCacheRow struct {
	ID           int64     `gorm:"primaryKey;autoIncrement;column:id"`
	GroupID      int64     `gorm:"column:group_id;not null;index;uniqueIndex:idx_user_tier_price_cache_unique,priority:1"`
	PackageID    int64     `gorm:"column:package_id;not null;index;uniqueIndex:idx_user_tier_price_cache_unique,priority:2"`
	MonthlyPrice int64     `gorm:"column:monthly_price;not null;default:0"`
	UnitCore     int64     `gorm:"column:unit_core;not null;default:0"`
	UnitMem      int64     `gorm:"column:unit_mem;not null;default:0"`
	UnitDisk     int64     `gorm:"column:unit_disk;not null;default:0"`
	UnitBW       int64     `gorm:"column:unit_bw;not null;default:0"`
	UpdatedAt    time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (userTierPriceCacheRow) TableName() string { return "user_tier_price_cache" }
