package domain

import "time"

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
	ID                   int64
	GoodsTypeID          int64
	PlanGroupID          int64
	ProductID            int64
	IntegrationPackageID int64
	Name                 string
	Cores                int
	MemoryGB             int
	DiskGB               int
	BandwidthMB          int
	CPUModel             string
	Monthly              int64
	PortNum              int
	SortOrder            int
	Active               bool
	Visible              bool
	CapacityRemaining    int
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
