package domain

import "time"

type CouponGroupScope string

const (
	CouponGroupScopeAll             CouponGroupScope = "all"
	CouponGroupScopeAllAddons       CouponGroupScope = "all_addons"
	CouponGroupScopeGoodsType       CouponGroupScope = "goods_type"
	CouponGroupScopeGoodsTypeRegion CouponGroupScope = "goods_type_region"
	CouponGroupScopePlanGroup       CouponGroupScope = "plan_group"
	CouponGroupScopePackage         CouponGroupScope = "package"
	CouponGroupScopeAddonConfig     CouponGroupScope = "addon_config"

	CouponRedemptionStatusApplied   string = "applied"
	CouponRedemptionStatusConfirmed string = "confirmed"
	CouponRedemptionStatusCanceled  string = "canceled"
)

type CouponProductGroup struct {
	ID          int64
	Name        string
	RulesJSON   string
	Scope       CouponGroupScope
	GoodsTypeID int64
	RegionID    int64
	PlanGroupID int64
	PackageID   int64
	AddonCore   int
	AddonMemGB  int
	AddonDiskGB int
	AddonBWMbps int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CouponProductRule struct {
	Scope            CouponGroupScope `json:"scope"`
	GoodsTypeID      int64            `json:"goods_type_id,omitempty"`
	RegionID         int64            `json:"region_id,omitempty"`
	PlanGroupID      int64            `json:"plan_group_id,omitempty"`
	PackageID        int64            `json:"package_id,omitempty"`
	AddonCoreEnabled bool             `json:"addon_core_enabled,omitempty"`
	AddonMemEnabled  bool             `json:"addon_mem_enabled,omitempty"`
	AddonDiskEnabled bool             `json:"addon_disk_enabled,omitempty"`
	AddonBWEnabled   bool             `json:"addon_bw_enabled,omitempty"`
}

type Coupon struct {
	ID               int64
	Code             string
	DiscountPermille int
	ProductGroupID   int64
	TotalLimit       int
	PerUserLimit     int
	StartsAt         *time.Time
	EndsAt           *time.Time
	NewUserOnly      bool
	Active           bool
	Note             string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type CouponRedemption struct {
	ID             int64
	CouponID       int64
	OrderID        int64
	UserID         int64
	Status         string
	DiscountAmount int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
