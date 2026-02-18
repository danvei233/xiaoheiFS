package repo

import (
	"time"
)

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
