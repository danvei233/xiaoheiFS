package http

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"
	apppayment "xiaoheiplay/internal/app/payment"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

type UserDTO struct {
	ID                int64      `json:"id"`
	Username          string     `json:"username"`
	Email             string     `json:"email"`
	EmailMasked       string     `json:"email_masked,omitempty"`
	EmailBound        bool       `json:"email_bound,omitempty"`
	QQ                string     `json:"qq"`
	Phone             string     `json:"phone"`
	PhoneMasked       string     `json:"phone_masked,omitempty"`
	PhoneBound        bool       `json:"phone_bound,omitempty"`
	TOTPEnabled       bool       `json:"totp_enabled"`
	LastLoginIP       string     `json:"last_login_ip"`
	LastLoginAt       *time.Time `json:"last_login_at"`
	Bio               string     `json:"bio"`
	Intro             string     `json:"intro"`
	AvatarURL         string     `json:"avatar_url"`
	PermissionGroupID *int64     `json:"permission_group_id"`
	UserTierGroupID   *int64     `json:"user_tier_group_id"`
	UserTierExpireAt  *time.Time `json:"user_tier_expire_at"`
	Role              string     `json:"role"`
	Status            string     `json:"status"`
	Permissions       []string   `json:"permissions,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type RegionDTO struct {
	ID          int64  `json:"id"`
	GoodsTypeID int64  `json:"goods_type_id"`
	Code        string `json:"code"`
	Name        string `json:"name" binding:"required"`
	Active      bool   `json:"active"`
}

type PlanGroupDTO struct {
	ID                int64   `json:"id"`
	GoodsTypeID       int64   `json:"goods_type_id"`
	RegionID          int64   `json:"region_id"`
	Name              string  `json:"name" binding:"required"`
	LineID            int64   `json:"line_id"`
	UnitCore          float64 `json:"unit_core" binding:"gte=0"`
	UnitMem           float64 `json:"unit_mem" binding:"gte=0"`
	UnitDisk          float64 `json:"unit_disk" binding:"gte=0"`
	UnitBW            float64 `json:"unit_bw" binding:"gte=0"`
	AddCoreMin        int     `json:"add_core_min"`
	AddCoreMax        int     `json:"add_core_max"`
	AddCoreStep       int     `json:"add_core_step"`
	AddMemMin         int     `json:"add_mem_min"`
	AddMemMax         int     `json:"add_mem_max"`
	AddMemStep        int     `json:"add_mem_step"`
	AddDiskMin        int     `json:"add_disk_min"`
	AddDiskMax        int     `json:"add_disk_max"`
	AddDiskStep       int     `json:"add_disk_step"`
	AddBWMin          int     `json:"add_bw_min"`
	AddBWMax          int     `json:"add_bw_max"`
	AddBWStep         int     `json:"add_bw_step"`
	Active            bool    `json:"active"`
	Visible           bool    `json:"visible"`
	CapacityRemaining int     `json:"capacity_remaining"`
	SortOrder         int     `json:"sort_order"`
}

type PackageDTO struct {
	ID                   int64   `json:"id"`
	GoodsTypeID          int64   `json:"goods_type_id"`
	PlanGroupID          int64   `json:"plan_group_id"`
	ProductID            int64   `json:"product_id"`
	IntegrationPackageID int64   `json:"integration_package_id"`
	Name                 string  `json:"name" binding:"required"`
	Cores                int     `json:"cores"`
	MemoryGB             int     `json:"memory_gb"`
	DiskGB               int     `json:"disk_gb"`
	BandwidthMB          int     `json:"bandwidth_mbps"`
	CPUModel             string  `json:"cpu_model"`
	MonthlyPrice         float64 `json:"monthly_price" binding:"gte=0"`
	PortNum              int     `json:"port_num"`
	SortOrder            int     `json:"sort_order"`
	Active               bool    `json:"active"`
	Visible              bool    `json:"visible"`
	CapacityRemaining    int     `json:"capacity_remaining"`
}

type SystemImageDTO struct {
	ID        int64    `json:"id"`
	ImageID   int64    `json:"image_id"`
	Name      string   `json:"name" binding:"required"`
	Type      string   `json:"type"`
	Enabled   bool     `json:"enabled"`
	LineNames []string `json:"line_names,omitempty"`
}

type BillingCycleDTO struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name" binding:"required"`
	Months     int       `json:"months" binding:"gt=0"`
	Multiplier float64   `json:"multiplier" binding:"gt=0"`
	MinQty     int       `json:"min_qty"`
	MaxQty     int       `json:"max_qty"`
	Active     bool      `json:"active"`
	SortOrder  int       `json:"sort_order"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type CartItemDTO struct {
	ID        int64              `json:"id"`
	UserID    int64              `json:"user_id"`
	PackageID int64              `json:"package_id"`
	SystemID  int64              `json:"system_id"`
	Spec      appshared.CartSpec `json:"spec"`
	Qty       int                `json:"qty"`
	Amount    float64            `json:"amount"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
}

type OrderDTO struct {
	ID             int64      `json:"id"`
	UserID         int64      `json:"user_id"`
	OrderNo        string     `json:"order_no"`
	Source         string     `json:"source"`
	Status         string     `json:"status"`
	TotalAmount    float64    `json:"total_amount"`
	Currency       string     `json:"currency"`
	CouponID       *int64     `json:"coupon_id,omitempty"`
	CouponCode     string     `json:"coupon_code,omitempty"`
	CouponDiscount float64    `json:"coupon_discount,omitempty"`
	IdempotencyKey string     `json:"idempotency_key"`
	PendingReason  string     `json:"pending_reason"`
	ApprovedBy     *int64     `json:"approved_by"`
	ApprovedAt     *time.Time `json:"approved_at"`
	RejectedReason string     `json:"rejected_reason"`
	CanReview      bool       `json:"can_review"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type OrderItemDTO struct {
	ID                   int64           `json:"id"`
	OrderID              int64           `json:"order_id"`
	PackageID            int64           `json:"package_id"`
	SystemID             int64           `json:"system_id"`
	Spec                 json.RawMessage `json:"spec"`
	Qty                  int             `json:"qty"`
	Amount               float64         `json:"amount"`
	Status               string          `json:"status"`
	AutomationInstanceID string          `json:"automation_instance_id"`
	Action               string          `json:"action"`
	DurationMonths       int             `json:"duration_months"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
}

type OrderPaymentDTO struct {
	ID             int64     `json:"id"`
	OrderID        int64     `json:"order_id"`
	UserID         int64     `json:"user_id"`
	Method         string    `json:"method"`
	Amount         float64   `json:"amount"`
	Currency       string    `json:"currency"`
	TradeNo        string    `json:"trade_no"`
	Note           string    `json:"note"`
	ScreenshotURL  string    `json:"screenshot_url"`
	Status         string    `json:"status"`
	IdempotencyKey string    `json:"idempotency_key"`
	ReviewedBy     *int64    `json:"reviewed_by"`
	ReviewReason   string    `json:"review_reason"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type PaymentProviderDTO struct {
	Key           string `json:"key"`
	Name          string `json:"name"`
	Enabled       bool   `json:"enabled"`
	OrderEnabled  bool   `json:"order_enabled"`
	WalletEnabled bool   `json:"wallet_enabled"`
	SchemaJSON    string `json:"schema_json"`
	ConfigJSON    string `json:"config_json"`
}

type PaymentMethodDTO struct {
	Key          string  `json:"key"`
	Name         string  `json:"name"`
	Instructions string  `json:"instructions"`
	Notice       string  `json:"notice"`
	Balance      float64 `json:"balance"`
}

type PaymentSelectDTO struct {
	Method  string            `json:"method"`
	Status  string            `json:"status"`
	TradeNo string            `json:"trade_no"`
	PayURL  string            `json:"pay_url"`
	Extra   map[string]string `json:"extra"`
	Paid    bool              `json:"paid"`
	Message string            `json:"message"`
	Balance float64           `json:"balance"`
}

type WalletDTO struct {
	UserID    int64     `json:"user_id"`
	Balance   float64   `json:"balance"`
	UpdatedAt time.Time `json:"updated_at"`
}

type WalletTransactionDTO struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Amount    float64   `json:"amount"`
	Type      string    `json:"type"`
	RefType   string    `json:"ref_type"`
	RefID     int64     `json:"ref_id"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
}

type WalletOrderDTO struct {
	ID           int64          `json:"id"`
	UserID       int64          `json:"user_id"`
	Type         string         `json:"type"`
	Amount       float64        `json:"amount"`
	Currency     string         `json:"currency"`
	Status       string         `json:"status"`
	Note         string         `json:"note"`
	Meta         map[string]any `json:"meta"`
	ReviewedBy   *int64         `json:"reviewed_by,omitempty"`
	ReviewReason string         `json:"review_reason,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

type NotificationDTO struct {
	ID        int64      `json:"id"`
	UserID    int64      `json:"user_id"`
	Type      string     `json:"type"`
	Title     string     `json:"title"`
	Content   string     `json:"content"`
	ReadAt    *time.Time `json:"read_at"`
	CreatedAt time.Time  `json:"created_at"`
}

type RealNameVerificationDTO struct {
	ID          int64      `json:"id"`
	UserID      int64      `json:"user_id"`
	RealName    string     `json:"real_name"`
	IDNumber    string     `json:"id_number"`
	Status      string     `json:"status"`
	Provider    string     `json:"provider"`
	Reason      string     `json:"reason"`
	RedirectURL string     `json:"redirect_url,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	VerifiedAt  *time.Time `json:"verified_at"`
}

type ServerStatusDTO struct {
	Hostname        string  `json:"hostname"`
	OS              string  `json:"os"`
	Platform        string  `json:"platform"`
	KernelVersion   string  `json:"kernel_version"`
	UptimeSeconds   uint64  `json:"uptime_seconds"`
	CPUModel        string  `json:"cpu_model"`
	CPUCores        int     `json:"cpu_cores"`
	CPUUsagePercent float64 `json:"cpu_usage_percent"`
	MemTotal        uint64  `json:"mem_total"`
	MemUsed         uint64  `json:"mem_used"`
	MemUsedPercent  float64 `json:"mem_used_percent"`
	// Backward/forward compat: frontend expects *_usage_percent
	MemUsagePercent float64 `json:"mem_usage_percent"`
	DiskTotal       uint64  `json:"disk_total"`
	DiskUsed        uint64  `json:"disk_used"`
	DiskUsedPercent float64 `json:"disk_used_percent"`
	// Backward/forward compat: frontend expects *_usage_percent
	DiskUsagePercent float64 `json:"disk_usage_percent"`
}

type VPSInstanceDTO struct {
	ID                   int64               `json:"id"`
	UserID               int64               `json:"user_id"`
	OrderItemID          int64               `json:"order_item_id"`
	GoodsTypeID          int64               `json:"goods_type_id"`
	AutomationInstanceID string              `json:"automation_instance_id"`
	Name                 string              `json:"name"`
	Region               string              `json:"region"`
	RegionID             int64               `json:"region_id"`
	LineID               int64               `json:"line_id"`
	PackageID            int64               `json:"package_id"`
	PackageName          string              `json:"package_name"`
	CPU                  int                 `json:"cpu"`
	MemoryGB             int                 `json:"memory_gb"`
	DiskGB               int                 `json:"disk_gb"`
	BandwidthMB          int                 `json:"bandwidth_mbps"`
	PortNum              int                 `json:"port_num"`
	MonthlyPrice         float64             `json:"monthly_price"`
	Spec                 json.RawMessage     `json:"spec"`
	SystemID             int64               `json:"system_id"`
	Status               string              `json:"status"`
	AutomationState      int                 `json:"automation_state"`
	AdminStatus          string              `json:"admin_status"`
	ExpireAt             *time.Time          `json:"expire_at"`
	DestroyAt            *time.Time          `json:"destroy_at,omitempty"`
	DestroyInDays        *int                `json:"destroy_in_days,omitempty"`
	PanelURLCache        string              `json:"panel_url_cache"`
	AccessInfo           map[string]any      `json:"access_info,omitempty"`
	Capabilities         *VPSCapabilitiesDTO `json:"capabilities,omitempty"`
	LastEmergencyRenewAt *time.Time          `json:"last_emergency_renew_at"`
	CreatedAt            time.Time           `json:"created_at"`
	UpdatedAt            time.Time           `json:"updated_at"`
}

type VPSCapabilitiesDTO struct {
	Automation *VPSAutomationCapabilityDTO `json:"automation,omitempty"`
}

type VPSAutomationCapabilityDTO struct {
	Features            []string          `json:"features"`
	NotSupportedReasons map[string]string `json:"not_supported_reasons,omitempty"`
}

type OrderEventDTO struct {
	ID        int64           `json:"id"`
	OrderID   int64           `json:"order_id"`
	Seq       int64           `json:"seq"`
	Type      string          `json:"type"`
	Data      json.RawMessage `json:"data"`
	CreatedAt time.Time       `json:"created_at"`
}

type AdminAuditLogDTO struct {
	ID         int64           `json:"id"`
	AdminID    int64           `json:"admin_id"`
	Action     string          `json:"action"`
	TargetType string          `json:"target_type"`
	TargetID   string          `json:"target_id"`
	Detail     json.RawMessage `json:"detail"`
	CreatedAt  time.Time       `json:"created_at"`
}

type APIKeyDTO struct {
	ID                int64      `json:"id"`
	Name              string     `json:"name"`
	KeyHash           string     `json:"key_hash"`
	Status            string     `json:"status"`
	Scopes            []string   `json:"scopes"`
	PermissionGroupID *int64     `json:"permission_group_id"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	LastUsedAt        *time.Time `json:"last_used_at"`
}

type SettingDTO struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

type EmailTemplateDTO struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name" binding:"required"`
	Subject   string    `json:"subject" binding:"required"`
	Body      string    `json:"body"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type IntegrationSyncLogDTO struct {
	ID        int64     `json:"id"`
	Target    string    `json:"target"`
	Mode      string    `json:"mode"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

type AutomationLogDTO struct {
	ID           int64           `json:"id"`
	OrderID      int64           `json:"order_id"`
	OrderItemID  int64           `json:"order_item_id"`
	Action       string          `json:"action"`
	RequestJSON  json.RawMessage `json:"request_json"`
	ResponseJSON json.RawMessage `json:"response_json"`
	Success      bool            `json:"success"`
	Message      string          `json:"message"`
	CreatedAt    time.Time       `json:"created_at"`
}

type UserTierGroupDTO struct {
	ID                 int64     `json:"id"`
	Name               string    `json:"name"`
	Color              string    `json:"color"`
	Icon               string    `json:"icon"`
	Priority           int       `json:"priority"`
	AutoApproveEnabled bool      `json:"auto_approve_enabled"`
	IsDefault          bool      `json:"is_default"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type UserTierDiscountRuleDTO struct {
	ID               int64  `json:"id"`
	GroupID          int64  `json:"group_id"`
	Scope            string `json:"scope"`
	GoodsTypeID      int64  `json:"goods_type_id"`
	RegionID         int64  `json:"region_id"`
	PlanGroupID      int64  `json:"plan_group_id"`
	PackageID        int64  `json:"package_id"`
	DiscountPermille int    `json:"discount_permille"`
	// FixedPrice is an optional fixed monthly price in CENTS (分), not yuan.
	// Nil means "no fixed price — apply permille discounts instead".
	// Do NOT convert with centsToFloat/floatToCents; it is passed through to the
	// domain model and stored as integer cents (see UserTierDiscountRule.MonthlyPrice).
	FixedPrice      *int64 `json:"fixed_price"`
	AddCorePermille int    `json:"add_core_permille"`
	AddMemPermille  int    `json:"add_mem_permille"`
	AddDiskPermille int    `json:"add_disk_permille"`
	AddBWPermille   int    `json:"add_bw_permille"`
}

type UserTierAutoRuleDTO struct {
	ID             int64  `json:"id"`
	GroupID        int64  `json:"group_id"`
	DurationDays   int    `json:"duration_days"`
	ConditionsJSON string `json:"conditions_json"`
	SortOrder      int    `json:"sort_order"`
}

type CouponProductGroupDTO struct {
	ID          int64                  `json:"id"`
	Name        string                 `json:"name"`
	Rules       []CouponProductRuleDTO `json:"rules"`
	Scope       string                 `json:"scope"`
	GoodsTypeID int64                  `json:"goods_type_id"`
	RegionID    int64                  `json:"region_id"`
	PlanGroupID int64                  `json:"plan_group_id"`
	PackageID   int64                  `json:"package_id"`
	AddonCore   int                    `json:"addon_core"`
	AddonMemGB  int                    `json:"addon_mem_gb"`
	AddonDiskGB int                    `json:"addon_disk_gb"`
	AddonBWMbps int                    `json:"addon_bw_mbps"`
}

type CouponProductRuleDTO struct {
	Scope            string `json:"scope"`
	GoodsTypeID      int64  `json:"goods_type_id"`
	RegionID         int64  `json:"region_id"`
	PlanGroupID      int64  `json:"plan_group_id"`
	PackageID        int64  `json:"package_id"`
	AddonCoreEnabled bool   `json:"addon_core_enabled"`
	AddonMemEnabled  bool   `json:"addon_mem_enabled"`
	AddonDiskEnabled bool   `json:"addon_disk_enabled"`
	AddonBWEnabled   bool   `json:"addon_bw_enabled"`
}

type CouponDTO struct {
	ID               int64      `json:"id"`
	Code             string     `json:"code"`
	DiscountPermille int        `json:"discount_permille"`
	ProductGroupID   int64      `json:"product_group_id"`
	TotalLimit       int        `json:"total_limit"`
	PerUserLimit     int        `json:"per_user_limit"`
	StartsAt         *time.Time `json:"starts_at"`
	EndsAt           *time.Time `json:"ends_at"`
	NewUserOnly      bool       `json:"new_user_only"`
	Active           bool       `json:"active"`
	Note             string     `json:"note"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func toUserDTO(user domain.User) UserDTO {
	return UserDTO{
		ID:                user.ID,
		Username:          user.Username,
		Email:             user.Email,
		QQ:                user.QQ,
		Phone:             user.Phone,
		TOTPEnabled:       user.TOTPEnabled,
		LastLoginIP:       user.LastLoginIP,
		LastLoginAt:       user.LastLoginAt,
		Bio:               user.Bio,
		Intro:             user.Intro,
		AvatarURL:         resolveAvatarURL(user),
		PermissionGroupID: user.PermissionGroupID,
		UserTierGroupID:   user.UserTierGroupID,
		UserTierExpireAt:  user.UserTierExpireAt,
		Role:              string(user.Role),
		Status:            string(user.Status),
		CreatedAt:         user.CreatedAt,
		UpdatedAt:         user.UpdatedAt,
	}
}

func toUserTierGroupDTO(item domain.UserTierGroup) UserTierGroupDTO {
	return UserTierGroupDTO{
		ID:                 item.ID,
		Name:               item.Name,
		Color:              item.Color,
		Icon:               item.Icon,
		Priority:           item.Priority,
		AutoApproveEnabled: item.AutoApproveEnabled,
		IsDefault:          item.IsDefault,
		CreatedAt:          item.CreatedAt,
		UpdatedAt:          item.UpdatedAt,
	}
}

func toUserTierGroupDTOs(items []domain.UserTierGroup) []UserTierGroupDTO {
	out := make([]UserTierGroupDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toUserTierGroupDTO(item))
	}
	return out
}

func toUserTierDiscountRuleDTO(item domain.UserTierDiscountRule) UserTierDiscountRuleDTO {
	return UserTierDiscountRuleDTO{
		ID:               item.ID,
		GroupID:          item.GroupID,
		Scope:            string(item.Scope),
		GoodsTypeID:      item.GoodsTypeID,
		RegionID:         item.RegionID,
		PlanGroupID:      item.PlanGroupID,
		PackageID:        item.PackageID,
		DiscountPermille: item.DiscountPermille,
		FixedPrice:       item.FixedPrice,
		AddCorePermille:  item.AddCorePermille,
		AddMemPermille:   item.AddMemPermille,
		AddDiskPermille:  item.AddDiskPermille,
		AddBWPermille:    item.AddBWPermille,
	}
}

func toUserTierDiscountRuleDTOs(items []domain.UserTierDiscountRule) []UserTierDiscountRuleDTO {
	out := make([]UserTierDiscountRuleDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toUserTierDiscountRuleDTO(item))
	}
	return out
}

func toUserTierAutoRuleDTO(item domain.UserTierAutoRule) UserTierAutoRuleDTO {
	return UserTierAutoRuleDTO{
		ID:             item.ID,
		GroupID:        item.GroupID,
		DurationDays:   item.DurationDays,
		ConditionsJSON: item.ConditionsJSON,
		SortOrder:      item.SortOrder,
	}
}

func toUserTierAutoRuleDTOs(items []domain.UserTierAutoRule) []UserTierAutoRuleDTO {
	out := make([]UserTierAutoRuleDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toUserTierAutoRuleDTO(item))
	}
	return out
}

func toCouponProductGroupDTO(item domain.CouponProductGroup) CouponProductGroupDTO {
	return CouponProductGroupDTO{
		ID:          item.ID,
		Name:        item.Name,
		Rules:       toCouponProductRuleDTOs(parseCouponProductRules(item.RulesJSON)),
		Scope:       string(item.Scope),
		GoodsTypeID: item.GoodsTypeID,
		RegionID:    item.RegionID,
		PlanGroupID: item.PlanGroupID,
		PackageID:   item.PackageID,
		AddonCore:   item.AddonCore,
		AddonMemGB:  item.AddonMemGB,
		AddonDiskGB: item.AddonDiskGB,
		AddonBWMbps: item.AddonBWMbps,
	}
}

func toCouponProductGroupDTOs(items []domain.CouponProductGroup) []CouponProductGroupDTO {
	out := make([]CouponProductGroupDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toCouponProductGroupDTO(item))
	}
	return out
}

func toCouponDTO(item domain.Coupon) CouponDTO {
	return CouponDTO{
		ID:               item.ID,
		Code:             item.Code,
		DiscountPermille: item.DiscountPermille,
		ProductGroupID:   item.ProductGroupID,
		TotalLimit:       item.TotalLimit,
		PerUserLimit:     item.PerUserLimit,
		StartsAt:         item.StartsAt,
		EndsAt:           item.EndsAt,
		NewUserOnly:      item.NewUserOnly,
		Active:           item.Active,
		Note:             item.Note,
		CreatedAt:        item.CreatedAt,
		UpdatedAt:        item.UpdatedAt,
	}
}

func toCouponDTOs(items []domain.Coupon) []CouponDTO {
	out := make([]CouponDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toCouponDTO(item))
	}
	return out
}

func toCouponProductRuleDTO(item domain.CouponProductRule) CouponProductRuleDTO {
	return CouponProductRuleDTO{
		Scope:            string(item.Scope),
		GoodsTypeID:      item.GoodsTypeID,
		RegionID:         item.RegionID,
		PlanGroupID:      item.PlanGroupID,
		PackageID:        item.PackageID,
		AddonCoreEnabled: item.AddonCoreEnabled,
		AddonMemEnabled:  item.AddonMemEnabled,
		AddonDiskEnabled: item.AddonDiskEnabled,
		AddonBWEnabled:   item.AddonBWEnabled,
	}
}

func toCouponProductRuleDTOs(items []domain.CouponProductRule) []CouponProductRuleDTO {
	out := make([]CouponProductRuleDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toCouponProductRuleDTO(item))
	}
	return out
}

func toUserSelfDTO(user domain.User) UserDTO {
	dto := toUserDTO(user)
	dto.Email = ""
	dto.Phone = ""
	dto.EmailMasked = maskEmail(user.Email)
	dto.PhoneMasked = maskPhone(user.Phone)
	dto.EmailBound = strings.TrimSpace(user.Email) != ""
	dto.PhoneBound = strings.TrimSpace(user.Phone) != ""
	return dto
}

func toUserDTOs(items []domain.User) []UserDTO {
	out := make([]UserDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toUserDTO(item))
	}
	return out
}

func toRegionDTO(region domain.Region) RegionDTO {
	return RegionDTO{
		ID:          region.ID,
		GoodsTypeID: region.GoodsTypeID,
		Code:        region.Code,
		Name:        region.Name,
		Active:      region.Active,
	}
}

func toPlanGroupDTO(plan domain.PlanGroup) PlanGroupDTO {
	return PlanGroupDTO{
		ID:                plan.ID,
		GoodsTypeID:       plan.GoodsTypeID,
		RegionID:          plan.RegionID,
		Name:              plan.Name,
		LineID:            plan.LineID,
		UnitCore:          centsToFloat(plan.UnitCore),
		UnitMem:           centsToFloat(plan.UnitMem),
		UnitDisk:          centsToFloat(plan.UnitDisk),
		UnitBW:            centsToFloat(plan.UnitBW),
		AddCoreMin:        plan.AddCoreMin,
		AddCoreMax:        plan.AddCoreMax,
		AddCoreStep:       plan.AddCoreStep,
		AddMemMin:         plan.AddMemMin,
		AddMemMax:         plan.AddMemMax,
		AddMemStep:        plan.AddMemStep,
		AddDiskMin:        plan.AddDiskMin,
		AddDiskMax:        plan.AddDiskMax,
		AddDiskStep:       plan.AddDiskStep,
		AddBWMin:          plan.AddBWMin,
		AddBWMax:          plan.AddBWMax,
		AddBWStep:         plan.AddBWStep,
		Active:            plan.Active,
		Visible:           plan.Visible,
		CapacityRemaining: plan.CapacityRemaining,
		SortOrder:         plan.SortOrder,
	}
}

func toPackageDTO(pkg domain.Package) PackageDTO {
	return PackageDTO{
		ID:                   pkg.ID,
		GoodsTypeID:          pkg.GoodsTypeID,
		PlanGroupID:          pkg.PlanGroupID,
		ProductID:            pkg.ProductID,
		IntegrationPackageID: pkg.IntegrationPackageID,
		Name:                 pkg.Name,
		Cores:                pkg.Cores,
		MemoryGB:             pkg.MemoryGB,
		DiskGB:               pkg.DiskGB,
		BandwidthMB:          pkg.BandwidthMB,
		CPUModel:             pkg.CPUModel,
		MonthlyPrice:         centsToFloat(pkg.Monthly),
		PortNum:              pkg.PortNum,
		SortOrder:            pkg.SortOrder,
		Active:               pkg.Active,
		Visible:              pkg.Visible,
		CapacityRemaining:    pkg.CapacityRemaining,
	}
}

func toSystemImageDTO(img domain.SystemImage) SystemImageDTO {
	return SystemImageDTO{
		ID:      img.ID,
		ImageID: img.ImageID,
		Name:    img.Name,
		Type:    img.Type,
		Enabled: img.Enabled,
	}
}

func toBillingCycleDTO(cycle domain.BillingCycle) BillingCycleDTO {
	return BillingCycleDTO{
		ID:         cycle.ID,
		Name:       cycle.Name,
		Months:     cycle.Months,
		Multiplier: cycle.Multiplier,
		MinQty:     cycle.MinQty,
		MaxQty:     cycle.MaxQty,
		Active:     cycle.Active,
		SortOrder:  cycle.SortOrder,
		CreatedAt:  cycle.CreatedAt,
		UpdatedAt:  cycle.UpdatedAt,
	}
}

func toCartItemDTO(item domain.CartItem) CartItemDTO {
	return CartItemDTO{
		ID:        item.ID,
		UserID:    item.UserID,
		PackageID: item.PackageID,
		SystemID:  item.SystemID,
		Spec:      parseCartSpec(item.SpecJSON),
		Qty:       item.Qty,
		Amount:    centsToFloat(item.Amount),
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}

func toOrderDTO(order domain.Order) OrderDTO {
	return OrderDTO{
		ID:             order.ID,
		UserID:         order.UserID,
		OrderNo:        order.OrderNo,
		Source:         order.Source,
		Status:         string(order.Status),
		TotalAmount:    centsToFloat(order.TotalAmount),
		Currency:       order.Currency,
		CouponID:       order.CouponID,
		CouponCode:     order.CouponCode,
		CouponDiscount: centsToFloat(order.CouponDiscount),
		IdempotencyKey: order.IdempotencyKey,
		PendingReason:  order.PendingReason,
		ApprovedBy:     order.ApprovedBy,
		ApprovedAt:     order.ApprovedAt,
		RejectedReason: order.RejectedReason,
		CanReview:      order.Status == domain.OrderStatusPendingPayment || order.Status == domain.OrderStatusPendingReview || order.Status == domain.OrderStatusRejected,
		CreatedAt:      order.CreatedAt,
		UpdatedAt:      order.UpdatedAt,
	}
}

func toOrderItemDTO(item domain.OrderItem) OrderItemDTO {
	return OrderItemDTO{
		ID:                   item.ID,
		OrderID:              item.OrderID,
		PackageID:            item.PackageID,
		SystemID:             item.SystemID,
		Spec:                 normalizeOrderItemSpec(item.Action, item.SpecJSON),
		Qty:                  item.Qty,
		Amount:               centsToFloat(item.Amount),
		Status:               string(item.Status),
		AutomationInstanceID: item.AutomationInstanceID,
		Action:               item.Action,
		DurationMonths:       item.DurationMonths,
		CreatedAt:            item.CreatedAt,
		UpdatedAt:            item.UpdatedAt,
	}
}

func toOrderPaymentDTO(payment domain.OrderPayment) OrderPaymentDTO {
	return OrderPaymentDTO{
		ID:             payment.ID,
		OrderID:        payment.OrderID,
		UserID:         payment.UserID,
		Method:         payment.Method,
		Amount:         centsToFloat(payment.Amount),
		Currency:       payment.Currency,
		TradeNo:        payment.TradeNo,
		Note:           payment.Note,
		ScreenshotURL:  payment.ScreenshotURL,
		Status:         string(payment.Status),
		IdempotencyKey: payment.IdempotencyKey,
		ReviewedBy:     payment.ReviewedBy,
		ReviewReason:   payment.ReviewReason,
		CreatedAt:      payment.CreatedAt,
		UpdatedAt:      payment.UpdatedAt,
	}
}

func toPaymentProviderDTO(info appshared.PaymentProviderInfo) PaymentProviderDTO {
	return PaymentProviderDTO{
		Key:           info.Key,
		Name:          info.Name,
		Enabled:       info.Enabled,
		OrderEnabled:  info.OrderEnabled,
		WalletEnabled: info.WalletEnabled,
		SchemaJSON:    info.SchemaJSON,
		ConfigJSON:    redactPaymentConfigJSON(info.ConfigJSON, info.SchemaJSON),
	}
}

// redactPaymentConfigJSON masks secret-keyed fields in a payment provider's
// config JSON before it is echoed to an admin session (P2-13). The config can
// carry merchant keys, app secrets and API keys; only the masked form must be
// returned by the list endpoint.
func redactPaymentConfigJSON(configJSON string, schemaJSON ...string) string {
	raw := strings.TrimSpace(configJSON)
	if raw == "" {
		return raw
	}
	var config any
	if err := json.Unmarshal([]byte(raw), &config); err != nil {
		return "{}"
	}
	switch config.(type) {
	case map[string]any, []any:
	default:
		masked, _ := json.Marshal(apppayment.ConfigSecretMask)
		return string(masked)
	}
	secretPaths := paymentSchemaSecretPaths(schemaJSON)
	changed := redactPaymentConfigValue(config, "", secretPaths)
	if !changed {
		return raw
	}
	out, err := json.Marshal(config)
	if err != nil {
		return raw
	}
	return string(out)
}

func redactPaymentConfigValue(value any, path string, schemaSecrets map[string]struct{}) bool {
	changed := false
	switch node := value.(type) {
	case map[string]any:
		for key, child := range node {
			childPath := key
			if path != "" {
				childPath = path + "." + key
			}
			if isPaymentConfigSecretKey(key) || hasPaymentSchemaSecret(schemaSecrets, childPath) {
				node[key] = apppayment.ConfigSecretMask
				changed = true
				continue
			}
			changed = redactPaymentConfigValue(child, childPath, schemaSecrets) || changed
		}
	case []any:
		for index, child := range node {
			childPath := fmt.Sprintf("%s[%d]", path, index)
			if hasPaymentSchemaSecret(schemaSecrets, childPath) {
				node[index] = apppayment.ConfigSecretMask
				changed = true
				continue
			}
			changed = redactPaymentConfigValue(child, childPath, schemaSecrets) || changed
		}
	}
	return changed
}

func paymentSchemaSecretPaths(schemaJSON []string) map[string]struct{} {
	result := make(map[string]struct{})
	for _, raw := range schemaJSON {
		var root map[string]any
		if json.Unmarshal([]byte(raw), &root) != nil {
			continue
		}
		activeRefs := map[string]bool{}
		nodesVisited := 0
		var walk func(map[string]any, string, int)
		walk = func(node map[string]any, prefix string, depth int) {
			nodesVisited++
			if depth > 64 || nodesVisited > 4096 {
				if prefix != "" {
					result[prefix] = struct{}{}
				}
				return
			}
			if secret, ok := node["x-secret"].(bool); ok && secret && prefix != "" {
				result[prefix] = struct{}{}
			}
			if format, _ := node["format"].(string); strings.EqualFold(format, "password") && prefix != "" {
				result[prefix] = struct{}{}
			}
			properties, _ := node["properties"].(map[string]any)
			for key, child := range properties {
				childMap, ok := child.(map[string]any)
				if ok {
					childPrefix := key
					if prefix != "" {
						childPrefix = prefix + "." + key
					}
					walk(childMap, childPrefix, depth+1)
				}
			}
			if items, ok := node["items"].(map[string]any); ok {
				walk(items, prefix+"[]", depth+1)
			}
			if additional, ok := node["additionalProperties"].(map[string]any); ok {
				additionalPrefix := "*"
				if prefix != "" {
					additionalPrefix = prefix + ".*"
				}
				walk(additional, additionalPrefix, depth+1)
			}
			for _, composition := range []string{"allOf", "oneOf", "anyOf"} {
				variants, _ := node[composition].([]any)
				for _, variant := range variants {
					if variantMap, ok := variant.(map[string]any); ok {
						walk(variantMap, prefix, depth+1)
					}
				}
			}
			if ref, _ := node["$ref"].(string); strings.HasPrefix(ref, "#/$defs/") || strings.HasPrefix(ref, "#/definitions/") {
				if activeRefs[ref] {
					// Recursive schemas can describe arbitrarily deep config trees. Once
					// the same reference re-enters the current chain, mask that subtree
					// rather than recursing forever or leaking deeper credentials.
					if prefix != "" {
						result[prefix] = struct{}{}
					}
					return
				}
				activeRefs[ref] = true
				defer delete(activeRefs, ref)
				segments := strings.Split(strings.TrimPrefix(ref, "#/"), "/")
				var current any = root
				for _, segment := range segments {
					mapping, ok := current.(map[string]any)
					if !ok {
						current = nil
						break
					}
					current = mapping[segment]
				}
				if refMap, ok := current.(map[string]any); ok {
					walk(refMap, prefix, depth+1)
				}
			}
		}
		walk(root, "", 0)
	}
	return result
}

func hasPaymentSchemaSecret(paths map[string]struct{}, path string) bool {
	normalizedPath := normalizePaymentConfigPath(path)
	if _, ok := paths[normalizedPath]; ok {
		return true
	}
	actualSegments := strings.Split(normalizedPath, ".")
	for pattern := range paths {
		patternSegments := strings.Split(pattern, ".")
		if len(patternSegments) != len(actualSegments) {
			continue
		}
		matches := true
		for index := range patternSegments {
			if patternSegments[index] != "*" && patternSegments[index] != actualSegments[index] {
				matches = false
				break
			}
		}
		if matches {
			return true
		}
	}
	return false
}

func normalizePaymentConfigPath(path string) string {
	var builder strings.Builder
	for index := 0; index < len(path); index++ {
		if path[index] != '[' {
			builder.WriteByte(path[index])
			continue
		}
		end := strings.IndexByte(path[index:], ']')
		if end <= 0 {
			builder.WriteByte(path[index])
			continue
		}
		builder.WriteString("[]")
		index += end
	}
	return builder.String()
}

// isPaymentConfigSecretKey reports whether a payment-channel config field holds
// a credential. It reuses the settings secret-key patterns and additionally
// treats merchant signing fields (`_key` / `private_key` / `sign_key`) as
// secrets. Scoped to payment provider configs only, so the broader `_key`
// suffix is acceptable here — the common secret fields in channel configs are
// exactly those (merchant_key, app_key, api_key, sign_key, private_key).
func isPaymentConfigSecretKey(key string) bool {
	if isSecretSettingKey(key) {
		return true
	}
	k := strings.ToLower(strings.TrimSpace(key))
	normalized := strings.NewReplacer("_", "", "-", "", ".", "").Replace(k)
	// Provider schemas are extensible and commonly use camelCase rather than
	// the site's snake_case setting names. Keep this explicit list narrow enough
	// to avoid masking ordinary labels while covering the credential spellings
	// used by SDKs and payment gateways.
	for _, exact := range []string{
		"apikey", "appkey", "secretkey", "privatekey", "signingkey", "clientsecret",
		"clientidsecret", "accesstoken", "refreshtoken", "merchantkey", "merchantsecret",
		"password", "secret", "token", "webhooksecret", "hmackey",
	} {
		if normalized == exact {
			return true
		}
	}
	for _, suffix := range []string{"_key", "_private_key", "_sign_key", "_signing_key"} {
		if strings.HasSuffix(k, suffix) {
			return true
		}
	}
	return false
}

func toPaymentProviderDTOs(items []appshared.PaymentProviderInfo) []PaymentProviderDTO {
	out := make([]PaymentProviderDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toPaymentProviderDTO(item))
	}
	return out
}

func toPaymentMethodDTO(info appshared.PaymentMethodInfo) PaymentMethodDTO {
	return PaymentMethodDTO{
		Key:          info.Key,
		Name:         info.Name,
		Instructions: info.Instructions,
		Notice:       info.Notice,
		Balance:      centsToFloat(info.Balance),
	}
}

func toPaymentMethodDTOs(items []appshared.PaymentMethodInfo) []PaymentMethodDTO {
	out := make([]PaymentMethodDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toPaymentMethodDTO(item))
	}
	return out
}

func toPaymentSelectDTO(result appshared.PaymentSelectResult) PaymentSelectDTO {
	return PaymentSelectDTO{
		Method:  result.Method,
		Status:  result.Status,
		TradeNo: result.TradeNo,
		PayURL:  result.PayURL,
		Extra:   result.Extra,
		Paid:    result.Paid,
		Message: result.Message,
		Balance: centsToFloat(result.Balance),
	}
}

func toWalletDTO(wallet domain.Wallet) WalletDTO {
	return WalletDTO{
		UserID:    wallet.UserID,
		Balance:   centsToFloat(wallet.Balance),
		UpdatedAt: wallet.UpdatedAt,
	}
}

func toWalletTransactionDTO(item domain.WalletTransaction) WalletTransactionDTO {
	return WalletTransactionDTO{
		ID:        item.ID,
		UserID:    item.UserID,
		Amount:    centsToFloat(item.Amount),
		Type:      item.Type,
		RefType:   item.RefType,
		RefID:     item.RefID,
		Note:      item.Note,
		CreatedAt: item.CreatedAt,
	}
}

func toWalletOrderDTO(item domain.WalletOrder) WalletOrderDTO {
	return WalletOrderDTO{
		ID:           item.ID,
		UserID:       item.UserID,
		Type:         string(item.Type),
		Amount:       centsToFloat(item.Amount),
		Currency:     item.Currency,
		Status:       string(item.Status),
		Note:         item.Note,
		Meta:         parseMapJSON(item.MetaJSON),
		ReviewedBy:   item.ReviewedBy,
		ReviewReason: item.ReviewReason,
		CreatedAt:    item.CreatedAt,
		UpdatedAt:    item.UpdatedAt,
	}
}

func toWalletTransactionDTOs(items []domain.WalletTransaction) []WalletTransactionDTO {
	out := make([]WalletTransactionDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toWalletTransactionDTO(item))
	}
	return out
}

func toWalletOrderDTOs(items []domain.WalletOrder) []WalletOrderDTO {
	out := make([]WalletOrderDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toWalletOrderDTO(item))
	}
	return out
}

func toNotificationDTO(item domain.Notification) NotificationDTO {
	return NotificationDTO{
		ID:        item.ID,
		UserID:    item.UserID,
		Type:      item.Type,
		Title:     item.Title,
		Content:   item.Content,
		ReadAt:    item.ReadAt,
		CreatedAt: item.CreatedAt,
	}
}

func toRealNameVerificationDTO(item domain.RealNameVerification, redirectURL string) RealNameVerificationDTO {
	return RealNameVerificationDTO{
		ID:          item.ID,
		UserID:      item.UserID,
		RealName:    item.RealName,
		IDNumber:    maskIDNumber(item.IDNumber),
		Status:      item.Status,
		Provider:    item.Provider,
		Reason:      item.Reason,
		RedirectURL: redirectURL,
		CreatedAt:   item.CreatedAt,
		VerifiedAt:  item.VerifiedAt,
	}
}

// realNameRedirectURL resolves the face-verification redirect URL for a record:
// new records keep it in the encrypted provider_state (decrypted by the
// service), legacy records still embed it in Reason. Never returns the token.
func (h *Handler) realNameRedirectURL(ctx context.Context, record domain.RealNameVerification) string {
	if h.realNameSvc != nil {
		if u := h.realNameSvc.PendingRedirectURL(ctx, record); u != "" {
			return u
		}
	}
	return parsePendingRedirectURL(record.Reason)
}

func parsePendingRedirectURL(reason string) string {
	reason = strings.TrimSpace(reason)
	if !strings.HasPrefix(reason, "pending_face:") {
		return ""
	}
	parts := strings.SplitN(reason, ":", 4)
	if len(parts) < 4 {
		return ""
	}
	encoded := strings.TrimSpace(parts[3])
	if encoded == "" {
		return ""
	}
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return ""
	}
	u := strings.TrimSpace(string(raw))
	if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
		return u
	}
	return ""
}

func toServerStatusDTO(status appshared.ServerStatus) ServerStatusDTO {
	return ServerStatusDTO{
		Hostname:         status.Hostname,
		OS:               status.OS,
		Platform:         status.Platform,
		KernelVersion:    status.KernelVersion,
		UptimeSeconds:    status.UptimeSeconds,
		CPUModel:         status.CPUModel,
		CPUCores:         status.CPUCores,
		CPUUsagePercent:  status.CPUUsagePercent,
		MemTotal:         status.MemTotal,
		MemUsed:          status.MemUsed,
		MemUsedPercent:   status.MemUsedPercent,
		MemUsagePercent:  status.MemUsedPercent,
		DiskTotal:        status.DiskTotal,
		DiskUsed:         status.DiskUsed,
		DiskUsedPercent:  status.DiskUsedPercent,
		DiskUsagePercent: status.DiskUsedPercent,
	}
}

func toVPSInstanceDTO(inst domain.VPSInstance) VPSInstanceDTO {
	return VPSInstanceDTO{
		ID:                   inst.ID,
		UserID:               inst.UserID,
		OrderItemID:          inst.OrderItemID,
		GoodsTypeID:          inst.GoodsTypeID,
		AutomationInstanceID: inst.AutomationInstanceID,
		Name:                 inst.Name,
		Region:               inst.Region,
		RegionID:             inst.RegionID,
		LineID:               inst.LineID,
		PackageID:            inst.PackageID,
		PackageName:          inst.PackageName,
		CPU:                  inst.CPU,
		MemoryGB:             inst.MemoryGB,
		DiskGB:               inst.DiskGB,
		BandwidthMB:          inst.BandwidthMB,
		PortNum:              inst.PortNum,
		MonthlyPrice:         centsToFloat(inst.MonthlyPrice),
		Spec:                 parseRawJSON(inst.SpecJSON),
		SystemID:             inst.SystemID,
		Status:               string(inst.Status),
		AutomationState:      inst.AutomationState,
		AdminStatus:          string(inst.AdminStatus),
		ExpireAt:             inst.ExpireAt,
		PanelURLCache:        inst.PanelURLCache,
		AccessInfo:           parseMapJSON(inst.AccessInfoJSON),
		LastEmergencyRenewAt: inst.LastEmergencyRenewAt,
		CreatedAt:            inst.CreatedAt,
		UpdatedAt:            inst.UpdatedAt,
	}
}

// toUserVPSInstanceDTO is the user-facing VPS shape (P2-4): it mirrors
// toVPSInstanceDTO but NEVER carries AccessInfo — the map contains the panel,
// VNC and OS passwords and must not be echoed to list/detail/monitor responses.
// Credentials are served only by the explicit, audited credentials endpoint.
func toUserVPSInstanceDTO(inst domain.VPSInstance) VPSInstanceDTO {
	dto := toVPSInstanceDTO(inst)
	dto.AccessInfo = nil
	return dto
}

func toUserVPSInstanceDTOs(items []domain.VPSInstance) []VPSInstanceDTO {
	out := make([]VPSInstanceDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toUserVPSInstanceDTO(item))
	}
	return out
}

func toOrderEventDTO(event domain.OrderEvent) OrderEventDTO {
	return OrderEventDTO{
		ID:        event.ID,
		OrderID:   event.OrderID,
		Seq:       event.Seq,
		Type:      event.Type,
		Data:      parseRawJSON(event.DataJSON),
		CreatedAt: event.CreatedAt,
	}
}

func toAdminAuditLogDTO(log domain.AdminAuditLog) AdminAuditLogDTO {
	detail := parseRawJSON(log.DetailJSON)
	// Defensive redaction: historical settings.update rows may still carry a
	// plaintext secret even though new writes are masked at the service layer.
	if strings.EqualFold(strings.TrimSpace(log.Action), "settings.update") &&
		isSecretSettingKey(log.TargetID) {
		detail = json.RawMessage(`{"value":"***"}`)
	}
	return AdminAuditLogDTO{
		ID:         log.ID,
		AdminID:    log.AdminID,
		Action:     log.Action,
		TargetType: log.TargetType,
		TargetID:   log.TargetID,
		Detail:     detail,
		CreatedAt:  log.CreatedAt,
	}
}

func toAPIKeyDTO(key domain.APIKey) APIKeyDTO {
	return APIKeyDTO{
		ID:                key.ID,
		Name:              key.Name,
		KeyHash:           key.KeyHash,
		Status:            string(key.Status),
		Scopes:            parseStringArray(key.ScopesJSON),
		PermissionGroupID: key.PermissionGroupID,
		CreatedAt:         key.CreatedAt,
		UpdatedAt:         key.UpdatedAt,
		LastUsedAt:        key.LastUsedAt,
	}
}

func toSettingDTO(setting domain.Setting) SettingDTO {
	value := normalizeSettingValue(setting.Key, setting.ValueJSON)
	if isPluginUploadPasswordSetting(setting.Key) && strings.TrimSpace(value) != "" {
		value = pluginUploadPasswordMask
	}
	return SettingDTO{
		Key:       setting.Key,
		Value:     value,
		UpdatedAt: setting.UpdatedAt,
	}
}

func toEmailTemplateDTO(tmpl domain.EmailTemplate) EmailTemplateDTO {
	return EmailTemplateDTO{
		ID:        tmpl.ID,
		Name:      tmpl.Name,
		Subject:   tmpl.Subject,
		Body:      tmpl.Body,
		Enabled:   tmpl.Enabled,
		CreatedAt: tmpl.CreatedAt,
		UpdatedAt: tmpl.UpdatedAt,
	}
}

func toIntegrationSyncLogDTO(log domain.IntegrationSyncLog) IntegrationSyncLogDTO {
	return IntegrationSyncLogDTO{
		ID:        log.ID,
		Target:    log.Target,
		Mode:      log.Mode,
		Status:    log.Status,
		Message:   log.Message,
		CreatedAt: log.CreatedAt,
	}
}

func toAutomationLogDTO(log domain.AutomationLog) AutomationLogDTO {
	return AutomationLogDTO{
		ID:           log.ID,
		OrderID:      log.OrderID,
		OrderItemID:  log.OrderItemID,
		Action:       log.Action,
		RequestJSON:  parseRawJSON(log.RequestJSON),
		ResponseJSON: parseRawJSON(log.ResponseJSON),
		Success:      log.Success,
		Message:      log.Message,
		CreatedAt:    log.CreatedAt,
	}
}

func toRegionDTOs(items []domain.Region) []RegionDTO {
	out := make([]RegionDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toRegionDTO(item))
	}
	return out
}

func toPlanGroupDTOs(items []domain.PlanGroup) []PlanGroupDTO {
	out := make([]PlanGroupDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toPlanGroupDTO(item))
	}
	return out
}

func toPackageDTOs(items []domain.Package) []PackageDTO {
	out := make([]PackageDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toPackageDTO(item))
	}
	return out
}

func toSystemImageDTOs(items []domain.SystemImage) []SystemImageDTO {
	out := make([]SystemImageDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toSystemImageDTO(item))
	}
	return out
}

func toBillingCycleDTOs(items []domain.BillingCycle) []BillingCycleDTO {
	out := make([]BillingCycleDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toBillingCycleDTO(item))
	}
	return out
}

func toCartItemDTOs(items []domain.CartItem) []CartItemDTO {
	out := make([]CartItemDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toCartItemDTO(item))
	}
	return out
}

func toOrderDTOs(items []domain.Order) []OrderDTO {
	out := make([]OrderDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toOrderDTO(item))
	}
	return out
}

func toOrderItemDTOs(items []domain.OrderItem) []OrderItemDTO {
	out := make([]OrderItemDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toOrderItemDTO(item))
	}
	return out
}

func toOrderPaymentDTOs(items []domain.OrderPayment) []OrderPaymentDTO {
	out := make([]OrderPaymentDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toOrderPaymentDTO(item))
	}
	return out
}

func toOrderEventDTOs(items []domain.OrderEvent) []OrderEventDTO {
	out := make([]OrderEventDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toOrderEventDTO(item))
	}
	return out
}

func toAdminAuditLogDTOs(items []domain.AdminAuditLog) []AdminAuditLogDTO {
	out := make([]AdminAuditLogDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toAdminAuditLogDTO(item))
	}
	return out
}

func toAPIKeyDTOs(items []domain.APIKey) []APIKeyDTO {
	out := make([]APIKeyDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toAPIKeyDTO(item))
	}
	return out
}

func toSettingDTOs(items []domain.Setting) []SettingDTO {
	out := make([]SettingDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toSettingDTO(item))
	}
	return out
}

func toEmailTemplateDTOs(items []domain.EmailTemplate) []EmailTemplateDTO {
	out := make([]EmailTemplateDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toEmailTemplateDTO(item))
	}
	return out
}

func toIntegrationSyncLogDTOs(items []domain.IntegrationSyncLog) []IntegrationSyncLogDTO {
	out := make([]IntegrationSyncLogDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toIntegrationSyncLogDTO(item))
	}
	return out
}

func toAutomationLogDTOs(items []domain.AutomationLog) []AutomationLogDTO {
	out := make([]AutomationLogDTO, 0, len(items))
	for _, item := range items {
		out = append(out, toAutomationLogDTO(item))
	}
	return out
}

func maskIDNumber(idNumber string) string {
	if idNumber == "" {
		return ""
	}
	if len(idNumber) <= 4 {
		return "****"
	}
	if len(idNumber) <= 8 {
		return idNumber[:2] + "****"
	}
	return idNumber[:4] + "****" + idNumber[len(idNumber)-4:]
}

func parseCartSpec(specJSON string) appshared.CartSpec {
	if specJSON == "" {
		return appshared.CartSpec{}
	}
	var spec appshared.CartSpec
	if err := json.Unmarshal([]byte(specJSON), &spec); err != nil {
		return appshared.CartSpec{}
	}
	return spec
}

func parseRawJSON(payload string) json.RawMessage {
	trimmed := strings.TrimSpace(payload)
	if trimmed == "" {
		return nil
	}
	var raw json.RawMessage
	if err := json.Unmarshal([]byte(trimmed), &raw); err == nil {
		return raw
	}
	encoded, _ := json.Marshal(payload)
	return encoded
}

func normalizeOrderItemSpec(action, specJSON string) json.RawMessage {
	raw := parseRawJSON(specJSON)
	if len(raw) == 0 {
		return raw
	}
	action = strings.ToLower(strings.TrimSpace(action))
	if action != "resize" && action != "refund" {
		return raw
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return raw
	}

	changed := false
	for _, key := range []string{"current_monthly", "target_monthly", "charge_amount", "refund_amount"} {
		value, ok := payload[key]
		if !ok {
			continue
		}
		cents, ok := parseAnyInt64(value)
		if !ok {
			continue
		}
		payload[key] = centsToFloat(cents)
		changed = true
	}
	if !changed {
		return raw
	}
	normalized, err := json.Marshal(payload)
	if err != nil {
		return raw
	}
	return normalized
}

func parseAnyInt64(value any) (int64, bool) {
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int64:
		return v, true
	case int32:
		return int64(v), true
	case float64:
		// Order-item spec amounts are integer cents; if a fractional float slips in
		// (e.g. JSON 9.99), round to the nearest cent instead of truncating to 9.
		return int64(math.Round(v)), true
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		return n, err == nil
	default:
		return 0, false
	}
}

func parseMapJSON(payload string) map[string]any {
	if payload == "" {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(payload), &out); err != nil {
		return nil
	}
	return out
}

func encodeMapJSON(m map[string]any) ([]byte, error) {
	return json.Marshal(m)
}

func parseStringArray(payload string) []string {
	if payload == "" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(payload), &out); err != nil {
		return nil
	}
	return out
}

// settingsSecretMask 掩码占位：后端设置列表绝不回显密钥明文，编辑提交时值为
// 掩码/空 则保留原值不覆盖（web_auth_surface F6）。
const settingsSecretMask = "********"

var secretSettingKeys = map[string]bool{
	"smtp_pass":                      true,
	"payment_plugin_upload_password": true,
	"robot_webhook_secret":           true,
	"robot_webhook_key":              true,
	"robot_webhooks":                 true,
	"automation_api_key":             true,
	"auth_geetest_captcha_key":       true,
	"fcm_server_key":                 true,
	"fcm_service_account_json":       true,
}

// isSecretSettingKey reports whether a settings key holds a credential that
// must never be echoed back or written into the audit log. It matches an
// explicit allowlist plus common credential suffixes.
func isSecretSettingKey(key string) bool {
	k := strings.ToLower(strings.TrimSpace(key))
	if secretSettingKeys[k] {
		return true
	}
	for _, suffix := range []string{"_secret", "_token", "_password", "_pass", "_api_key", "apikey", "appkey"} {
		if strings.HasSuffix(k, suffix) {
			return true
		}
	}
	return false
}

func normalizeSettingValue(key, value string) string {
	// robot_webhooks is deep-redacted separately (redactRobotWebhooksValue
	// preserves URLs/names/events); masking the whole value would break the
	// robot config page.
	if strings.EqualFold(strings.TrimSpace(key), "robot_webhooks") {
		return value
	}
	if isSecretSettingKey(key) && value != "" {
		return settingsSecretMask
	}
	return value
}

// pluginUploadPasswordMask is returned in place of the real payment plugin
// upload password in the admin settings list. The update path treats this
// literal as "keep the existing value" so a read-modify-write round trip does
// not overwrite the secret with the mask.
const pluginUploadPasswordMask = "******"

// pluginUploadPasswordSettingKey is the settings key that gates plugin package
// uploads. It is a credential and must not be echoed in plaintext.
const pluginUploadPasswordSettingKey = "payment_plugin_upload_password"

// isPluginUploadPasswordSetting reports whether a settings key is the plugin
// upload password.
func isPluginUploadPasswordSetting(key string) bool {
	return strings.EqualFold(strings.TrimSpace(key), pluginUploadPasswordSettingKey)
}

func isLikelyJSONSettingKey(key string) bool {
	k := strings.ToLower(strings.TrimSpace(key))
	if k == "" {
		return false
	}
	if strings.HasSuffix(k, "_json") || strings.HasPrefix(k, "task.") {
		return true
	}
	// Named JSON settings keys are consolidated in app/shared (P3-4) so the
	// handler and the repo layer cannot drift from each other.
	return appshared.IsNamedJSONSettingKey(k)
}

func parseCouponProductRules(payload string) []domain.CouponProductRule {
	raw := strings.TrimSpace(payload)
	if raw == "" {
		return nil
	}
	var out []domain.CouponProductRule
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	items := make([]domain.CouponProductRule, 0, len(out))
	for _, item := range out {
		if strings.TrimSpace(string(item.Scope)) == "" {
			continue
		}
		items = append(items, item)
	}
	return items
}

func planGroupDTOToDomain(dto PlanGroupDTO) domain.PlanGroup {
	return domain.PlanGroup{
		ID:                dto.ID,
		GoodsTypeID:       dto.GoodsTypeID,
		RegionID:          dto.RegionID,
		Name:              dto.Name,
		LineID:            dto.LineID,
		UnitCore:          floatToCents(dto.UnitCore),
		UnitMem:           floatToCents(dto.UnitMem),
		UnitDisk:          floatToCents(dto.UnitDisk),
		UnitBW:            floatToCents(dto.UnitBW),
		AddCoreMin:        dto.AddCoreMin,
		AddCoreMax:        dto.AddCoreMax,
		AddCoreStep:       dto.AddCoreStep,
		AddMemMin:         dto.AddMemMin,
		AddMemMax:         dto.AddMemMax,
		AddMemStep:        dto.AddMemStep,
		AddDiskMin:        dto.AddDiskMin,
		AddDiskMax:        dto.AddDiskMax,
		AddDiskStep:       dto.AddDiskStep,
		AddBWMin:          dto.AddBWMin,
		AddBWMax:          dto.AddBWMax,
		AddBWStep:         dto.AddBWStep,
		Active:            dto.Active,
		Visible:           dto.Visible,
		CapacityRemaining: dto.CapacityRemaining,
		SortOrder:         dto.SortOrder,
	}
}

func packageDTOToDomain(dto PackageDTO) domain.Package {
	return domain.Package{
		ID:                   dto.ID,
		GoodsTypeID:          dto.GoodsTypeID,
		PlanGroupID:          dto.PlanGroupID,
		ProductID:            dto.ProductID,
		IntegrationPackageID: dto.IntegrationPackageID,
		Name:                 dto.Name,
		Cores:                dto.Cores,
		MemoryGB:             dto.MemoryGB,
		DiskGB:               dto.DiskGB,
		BandwidthMB:          dto.BandwidthMB,
		CPUModel:             dto.CPUModel,
		Monthly:              floatToCents(dto.MonthlyPrice),
		PortNum:              dto.PortNum,
		SortOrder:            dto.SortOrder,
		Active:               dto.Active,
		Visible:              dto.Visible,
		CapacityRemaining:    dto.CapacityRemaining,
	}
}

func systemImageDTOToDomain(dto SystemImageDTO) domain.SystemImage {
	return domain.SystemImage{
		ID:      dto.ID,
		ImageID: dto.ImageID,
		Name:    dto.Name,
		Type:    dto.Type,
		Enabled: dto.Enabled,
	}
}

func regionDTOToDomain(dto RegionDTO) domain.Region {
	return domain.Region{
		ID:          dto.ID,
		GoodsTypeID: dto.GoodsTypeID,
		Code:        dto.Code,
		Name:        dto.Name,
		Active:      dto.Active,
	}
}

func billingCycleDTOToDomain(dto BillingCycleDTO) domain.BillingCycle {
	return domain.BillingCycle{
		ID:         dto.ID,
		Name:       dto.Name,
		Months:     dto.Months,
		Multiplier: dto.Multiplier,
		MinQty:     dto.MinQty,
		MaxQty:     dto.MaxQty,
		Active:     dto.Active,
		SortOrder:  dto.SortOrder,
	}
}

func emailTemplateDTOToDomain(dto EmailTemplateDTO) domain.EmailTemplate {
	return domain.EmailTemplate{
		ID:      dto.ID,
		Name:    dto.Name,
		Subject: dto.Subject,
		Body:    dto.Body,
		Enabled: dto.Enabled,
	}
}

type CMSCategoryDTO struct {
	ID        int64     `json:"id"`
	Key       string    `json:"key"`
	Name      string    `json:"name"`
	Lang      string    `json:"lang"`
	SortOrder int       `json:"sort_order"`
	Visible   bool      `json:"visible"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CMSPostDTO struct {
	ID          int64      `json:"id"`
	CategoryID  int64      `json:"category_id"`
	Title       string     `json:"title"`
	Slug        string     `json:"slug"`
	Summary     string     `json:"summary"`
	ContentHTML string     `json:"content_html"`
	CoverURL    string     `json:"cover_url"`
	Lang        string     `json:"lang"`
	Status      string     `json:"status"`
	Pinned      bool       `json:"pinned"`
	SortOrder   int        `json:"sort_order"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type CMSBlockDTO struct {
	ID          int64     `json:"id"`
	Page        string    `json:"page"`
	Type        string    `json:"type"`
	Title       string    `json:"title"`
	Subtitle    string    `json:"subtitle"`
	ContentJSON string    `json:"content_json"`
	CustomHTML  string    `json:"custom_html"`
	Lang        string    `json:"lang"`
	Visible     bool      `json:"visible"`
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type TicketDTO struct {
	ID            int64      `json:"id"`
	UserID        int64      `json:"user_id"`
	Subject       string     `json:"subject"`
	Status        string     `json:"status"`
	ResourceCount int        `json:"resource_count"`
	LastReplyAt   *time.Time `json:"last_reply_at,omitempty"`
	LastReplyBy   *int64     `json:"last_reply_by,omitempty"`
	LastReplyRole string     `json:"last_reply_role"`
	ClosedAt      *time.Time `json:"closed_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type TicketMessageDTO struct {
	ID         int64     `json:"id"`
	TicketID   int64     `json:"ticket_id"`
	SenderID   int64     `json:"sender_id"`
	SenderRole string    `json:"sender_role"`
	SenderName string    `json:"sender_name,omitempty"`
	SenderQQ   string    `json:"sender_qq,omitempty"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"created_at"`
}

type TicketResourceDTO struct {
	ID           int64     `json:"id"`
	TicketID     int64     `json:"ticket_id"`
	ResourceType string    `json:"resource_type"`
	ResourceID   int64     `json:"resource_id"`
	ResourceName string    `json:"resource_name"`
	CreatedAt    time.Time `json:"created_at"`
}
type UploadDTO struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	URL        string    `json:"url"`
	Mime       string    `json:"mime"`
	Size       int64     `json:"size"`
	UploaderID int64     `json:"uploader_id"`
	CreatedAt  time.Time `json:"created_at"`
}

func toCMSCategoryDTO(item domain.CMSCategory) CMSCategoryDTO {
	return CMSCategoryDTO{
		ID:        item.ID,
		Key:       item.Key,
		Name:      item.Name,
		Lang:      item.Lang,
		SortOrder: item.SortOrder,
		Visible:   item.Visible,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}

func toCMSPostDTO(item domain.CMSPost) CMSPostDTO {
	return CMSPostDTO{
		ID:          item.ID,
		CategoryID:  item.CategoryID,
		Title:       item.Title,
		Slug:        item.Slug,
		Summary:     item.Summary,
		ContentHTML: item.ContentHTML,
		CoverURL:    item.CoverURL,
		Lang:        item.Lang,
		Status:      item.Status,
		Pinned:      item.Pinned,
		SortOrder:   item.SortOrder,
		PublishedAt: item.PublishedAt,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}
}

func toCMSBlockDTO(item domain.CMSBlock) CMSBlockDTO {
	return CMSBlockDTO{
		ID:          item.ID,
		Page:        item.Page,
		Type:        item.Type,
		Title:       item.Title,
		Subtitle:    item.Subtitle,
		ContentJSON: item.ContentJSON,
		CustomHTML:  item.CustomHTML,
		Lang:        item.Lang,
		Visible:     item.Visible,
		SortOrder:   item.SortOrder,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}
}

func toTicketDTO(item domain.Ticket) TicketDTO {
	return TicketDTO{
		ID:            item.ID,
		UserID:        item.UserID,
		Subject:       item.Subject,
		Status:        item.Status,
		ResourceCount: item.ResourceCount,
		LastReplyAt:   item.LastReplyAt,
		LastReplyBy:   item.LastReplyBy,
		LastReplyRole: item.LastReplyRole,
		ClosedAt:      item.ClosedAt,
		CreatedAt:     item.CreatedAt,
		UpdatedAt:     item.UpdatedAt,
	}
}

func toTicketMessageDTO(item domain.TicketMessage, senderName, senderQQ string) TicketMessageDTO {
	return TicketMessageDTO{
		ID:         item.ID,
		TicketID:   item.TicketID,
		SenderID:   item.SenderID,
		SenderRole: item.SenderRole,
		SenderName: senderName,
		SenderQQ:   senderQQ,
		Content:    item.Content,
		CreatedAt:  item.CreatedAt,
	}
}

func toTicketResourceDTO(item domain.TicketResource) TicketResourceDTO {
	return TicketResourceDTO{
		ID:           item.ID,
		TicketID:     item.TicketID,
		ResourceType: item.ResourceType,
		ResourceID:   item.ResourceID,
		ResourceName: item.ResourceName,
		CreatedAt:    item.CreatedAt,
	}
}
func toUploadDTO(item domain.Upload) UploadDTO {
	return UploadDTO{
		ID:         item.ID,
		Name:       item.Name,
		Path:       item.Path,
		URL:        item.URL,
		Mime:       item.Mime,
		Size:       item.Size,
		UploaderID: item.UploaderID,
		CreatedAt:  item.CreatedAt,
	}
}

func centsToFloat(cents int64) float64 {
	return float64(cents) / 100
}

func floatToCents(amount float64) int64 {
	return int64(math.Round(amount * 100))
}

func resolveAvatarURL(user domain.User) string {
	if user.Avatar != "" {
		return user.Avatar
	}
	if user.QQ != "" {
		return "https://q1.qlogo.cn/g?b=qq&nk=" + url.QueryEscape(user.QQ) + "&s=100"
	}
	seed := user.Username
	if seed == "" {
		seed = fmt.Sprintf("user-%d", user.ID)
	}
	return "https://api.dicebear.com/7.x/identicon/svg?seed=" + url.QueryEscape(seed)
}
