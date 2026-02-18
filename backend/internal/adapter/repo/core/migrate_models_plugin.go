package repo

import (
	"time"
)

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
