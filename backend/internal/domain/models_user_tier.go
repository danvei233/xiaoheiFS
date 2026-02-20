package domain

import "time"

type UserTierScope string

const (
	UserTierScopeAll               UserTierScope = "all"
	UserTierScopeAllAddons         UserTierScope = "all_addons"
	UserTierScopeGoodsType         UserTierScope = "goods_type"
	UserTierScopeGoodsTypeArea     UserTierScope = "goods_type_region"
	UserTierScopePlanGroup         UserTierScope = "plan_group"
	UserTierScopePackage           UserTierScope = "package"
	UserTierScopeAddonConfig       UserTierScope = "addon_config"
	UserTierMembershipSourceAuto   string        = "auto"
	UserTierMembershipSourceManual string        = "manual"
)

type UserTierGroup struct {
	ID                 int64
	Name               string
	Color              string
	Icon               string
	Priority           int
	AutoApproveEnabled bool
	IsDefault          bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type UserTierDiscountRule struct {
	ID               int64
	GroupID          int64
	Scope            UserTierScope
	GoodsTypeID      int64
	RegionID         int64
	PlanGroupID      int64
	PackageID        int64
	DiscountPermille int
	FixedPrice       *int64
	AddCorePermille  int
	AddMemPermille   int
	AddDiskPermille  int
	AddBWPermille    int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type UserTierAutoRule struct {
	ID             int64
	GroupID        int64
	DurationDays   int
	ConditionsJSON string
	SortOrder      int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type UserTierMembership struct {
	UserID    int64
	GroupID   int64
	Source    string
	ExpiresAt *time.Time
	UpdatedAt time.Time
}

type UserTierPriceCache struct {
	ID           int64
	GroupID      int64
	PackageID    int64
	MonthlyPrice int64
	UnitCore     int64
	UnitMem      int64
	UnitDisk     int64
	UnitBW       int64
	UpdatedAt    time.Time
}
