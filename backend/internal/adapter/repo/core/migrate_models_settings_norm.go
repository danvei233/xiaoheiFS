package repo

import "time"

type smsTemplateRow struct {
	ID        int64     `gorm:"primaryKey;autoIncrement;column:id"`
	Name      string    `gorm:"size:191;column:name;not null;uniqueIndex"`
	Content   string    `gorm:"type:text;column:content;not null"`
	Enabled   int       `gorm:"column:enabled;not null;default:1"`
	CreatedAt time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (smsTemplateRow) TableName() string { return "sms_templates" }

type settingListValueRow struct {
	ID         int64     `gorm:"primaryKey;autoIncrement;column:id"`
	SettingKey string    `gorm:"size:191;column:setting_key;not null;uniqueIndex:idx_setting_list_values_unique,priority:1;index:idx_setting_list_values_key_order,priority:1"`
	Value      string    `gorm:"size:191;column:value;not null;uniqueIndex:idx_setting_list_values_unique,priority:2"`
	SortOrder  int       `gorm:"column:sort_order;not null;default:0;index:idx_setting_list_values_key_order,priority:2"`
	CreatedAt  time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt  time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (settingListValueRow) TableName() string { return "setting_list_values" }

type robotWebhookRow struct {
	ID         int64     `gorm:"primaryKey;autoIncrement;column:id"`
	Name       string    `gorm:"size:191;column:name;not null;default:''"`
	URL        string    `gorm:"size:1024;column:url;not null;default:''"`
	Secret     string    `gorm:"size:512;column:secret;not null;default:''"`
	Enabled    int       `gorm:"column:enabled;not null;default:1"`
	EventsJSON string    `gorm:"type:text;column:events_json;not null;default:'[]'"`
	SortOrder  int       `gorm:"column:sort_order;not null;default:0"`
	CreatedAt  time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt  time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (robotWebhookRow) TableName() string { return "robot_webhooks" }

type scheduledTaskConfigRow struct {
	TaskKey     string    `gorm:"size:191;primaryKey;column:task_key"`
	Enabled     int       `gorm:"column:enabled;not null;default:1"`
	Strategy    string    `gorm:"size:32;column:strategy;not null;default:interval"`
	IntervalSec int       `gorm:"column:interval_sec;not null;default:60"`
	DailyAt     string    `gorm:"size:16;column:daily_at;not null;default:''"`
	UpdatedAt   time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (scheduledTaskConfigRow) TableName() string { return "scheduled_task_configs" }

type packageCapabilityRow struct {
	ID            int64     `gorm:"primaryKey;autoIncrement;column:id"`
	PackageID     int64     `gorm:"column:package_id;not null;uniqueIndex"`
	ResizeEnabled *int      `gorm:"column:resize_enabled"`
	RefundEnabled *int      `gorm:"column:refund_enabled"`
	CreatedAt     time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt     time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (packageCapabilityRow) TableName() string { return "package_capabilities" }

type permissionGroupPermissionRow struct {
	ID                int64     `gorm:"primaryKey;autoIncrement;column:id"`
	PermissionGroupID int64     `gorm:"column:permission_group_id;not null;index;uniqueIndex:idx_permission_group_permissions_unique,priority:1"`
	PermissionCode    string    `gorm:"size:191;column:permission_code;not null;uniqueIndex:idx_permission_group_permissions_unique,priority:2"`
	CreatedAt         time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt         time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (permissionGroupPermissionRow) TableName() string { return "permission_group_permissions" }
