package repo

import "time"

type couponProductGroupRow struct {
	ID          int64     `gorm:"primaryKey;autoIncrement;column:id"`
	Name        string    `gorm:"size:191;column:name;not null"`
	RulesJSON   string    `gorm:"type:text;column:rules_json;not null;default:[]"`
	Scope       string    `gorm:"size:64;column:scope;not null;index"`
	GoodsTypeID int64     `gorm:"column:goods_type_id;not null;default:0;index"`
	RegionID    int64     `gorm:"column:region_id;not null;default:0;index"`
	PlanGroupID int64     `gorm:"column:plan_group_id;not null;default:0;index"`
	PackageID   int64     `gorm:"column:package_id;not null;default:0;index"`
	AddonCore   int       `gorm:"column:addon_core;not null;default:-1"`
	AddonMemGB  int       `gorm:"column:addon_mem_gb;not null;default:-1"`
	AddonDiskGB int       `gorm:"column:addon_disk_gb;not null;default:-1"`
	AddonBWMbps int       `gorm:"column:addon_bw_mbps;not null;default:-1"`
	CreatedAt   time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt   time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (couponProductGroupRow) TableName() string { return "coupon_product_groups" }

type couponRow struct {
	ID               int64      `gorm:"primaryKey;autoIncrement;column:id"`
	Code             string     `gorm:"size:128;column:code;not null;uniqueIndex"`
	DiscountPermille int        `gorm:"column:discount_permille;not null;default:1000"`
	ProductGroupID   int64      `gorm:"column:product_group_id;not null;index"`
	TotalLimit       int        `gorm:"column:total_limit;not null;default:-1"`
	PerUserLimit     int        `gorm:"column:per_user_limit;not null;default:-1"`
	StartsAt         *time.Time `gorm:"column:starts_at;index"`
	EndsAt           *time.Time `gorm:"column:ends_at;index"`
	NewUserOnly      int        `gorm:"column:new_user_only;not null;default:0;index"`
	Active           int        `gorm:"column:active;not null;default:1;index"`
	Note             string     `gorm:"size:500;column:note;not null;default:''"`
	CreatedAt        time.Time  `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt        time.Time  `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (couponRow) TableName() string { return "coupons" }

type couponRedemptionRow struct {
	ID             int64     `gorm:"primaryKey;autoIncrement;column:id"`
	CouponID       int64     `gorm:"column:coupon_id;not null;index"`
	OrderID        int64     `gorm:"column:order_id;not null;uniqueIndex"`
	UserID         int64     `gorm:"column:user_id;not null;index"`
	Status         string    `gorm:"size:32;column:status;not null;index"`
	DiscountAmount int64     `gorm:"column:discount_amount;not null;default:0"`
	CreatedAt      time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt      time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (couponRedemptionRow) TableName() string { return "coupon_redemptions" }
