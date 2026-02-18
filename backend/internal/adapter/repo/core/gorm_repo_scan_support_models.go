package repo

import (
	"time"
)

type settingModel struct {
	Key       string    `gorm:"primaryKey;column:key"`
	ValueJSON string    `gorm:"column:value_json"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (settingModel) TableName() string { return "settings" }

type pluginInstallationModel struct {
	ID              int64     `gorm:"primaryKey;autoIncrement;column:id"`
	Category        string    `gorm:"column:category;uniqueIndex:idx_plugin_installations_cat_id_instance"`
	PluginID        string    `gorm:"column:plugin_id;uniqueIndex:idx_plugin_installations_cat_id_instance"`
	InstanceID      string    `gorm:"column:instance_id;uniqueIndex:idx_plugin_installations_cat_id_instance"`
	Enabled         int       `gorm:"column:enabled"`
	SignatureStatus string    `gorm:"column:signature_status"`
	ConfigCipher    string    `gorm:"column:config_cipher"`
	CreatedAt       time.Time `gorm:"column:created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at"`
}

func (pluginInstallationModel) TableName() string { return "plugin_installations" }

type pluginPaymentMethodModel struct {
	ID         int64     `gorm:"primaryKey;autoIncrement;column:id"`
	Category   string    `gorm:"column:category"`
	PluginID   string    `gorm:"column:plugin_id"`
	InstanceID string    `gorm:"column:instance_id"`
	Method     string    `gorm:"column:method"`
	Enabled    int       `gorm:"column:enabled"`
	CreatedAt  time.Time `gorm:"column:created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at"`
}

func (pluginPaymentMethodModel) TableName() string { return "plugin_payment_methods" }

type provisionJobModel struct {
	ID          int64     `gorm:"primaryKey;autoIncrement;column:id"`
	OrderID     int64     `gorm:"column:order_id"`
	OrderItemID int64     `gorm:"column:order_item_id;uniqueIndex:idx_provision_jobs_item"`
	HostID      int64     `gorm:"column:host_id"`
	HostName    string    `gorm:"column:host_name"`
	Status      string    `gorm:"column:status"`
	Attempts    int       `gorm:"column:attempts"`
	NextRunAt   time.Time `gorm:"column:next_run_at"`
	LastError   string    `gorm:"column:last_error"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (provisionJobModel) TableName() string { return "provision_jobs" }

type permissionModel struct {
	ID           int64     `gorm:"primaryKey;autoIncrement;column:id"`
	Code         string    `gorm:"column:code;uniqueIndex"`
	Name         string    `gorm:"column:name"`
	FriendlyName string    `gorm:"column:friendly_name"`
	Category     string    `gorm:"column:category"`
	ParentCode   string    `gorm:"column:parent_code"`
	SortOrder    int       `gorm:"column:sort_order"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (permissionModel) TableName() string { return "permissions" }

type walletModel struct {
	ID        int64     `gorm:"primaryKey;autoIncrement;column:id"`
	UserID    int64     `gorm:"column:user_id;uniqueIndex"`
	Balance   int64     `gorm:"column:balance"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (walletModel) TableName() string { return "user_wallets" }

type pushTokenModel struct {
	ID        int64     `gorm:"primaryKey;autoIncrement;column:id"`
	UserID    int64     `gorm:"column:user_id"`
	Platform  string    `gorm:"column:platform"`
	Token     string    `gorm:"column:token"`
	DeviceID  string    `gorm:"column:device_id"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (pushTokenModel) TableName() string { return "push_tokens" }

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
