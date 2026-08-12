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
	ID             int64     `gorm:"primaryKey;autoIncrement;column:id"`
	UserID         int64     `gorm:"column:user_id;not null;index"`
	Amount         int64     `gorm:"column:amount;not null"`
	Type           string    `gorm:"column:type;not null"`
	RefType        string    `gorm:"column:ref_type;not null"`
	RefID          int64     `gorm:"column:ref_id;not null;default:0"`
	IdempotencyKey *string   `gorm:"size:191;column:idempotency_key;uniqueIndex:idx_wallet_transactions_idempotency_key"`
	Note           string    `gorm:"column:note;not null;default:''"`
	CreatedAt      time.Time `gorm:"column:created_at;not null;autoCreateTime"`
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

// scheduledTaskLeaseRow backs the database CAS lease that makes scheduled task
// execution exclusive across multiple server instances. task_key is the primary
// key so there is exactly one lease row per task. fencing_token is rotated on
// every acquisition/renewal and gates the final run-status write.
type scheduledTaskLeaseRow struct {
	TaskKey      string    `gorm:"column:task_key;primaryKey;size:191"`
	Owner        string    `gorm:"column:owner;size:191;not null;default:''"`
	LeaseUntil   time.Time `gorm:"column:lease_until;not null"`
	FencingToken int64     `gorm:"column:fencing_token;not null;default:0"`
}

func (scheduledTaskLeaseRow) TableName() string { return "scheduled_task_leases" }

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
	ID       int64  `gorm:"primaryKey;autoIncrement;column:id"`
	UserID   int64  `gorm:"column:user_id;not null;index"`
	RealName string `gorm:"column:real_name;not null"`
	IDNumber string `gorm:"column:id_number;not null"`
	Status   string `gorm:"column:status;not null"`
	Provider string `gorm:"column:provider;not null"`
	Reason   string `gorm:"column:reason;not null;default:''"`
	// ProviderState holds the versioned encrypted provider continuation state
	// (pending KYC token / redirect URL). It is never returned by the API.
	ProviderState string     `gorm:"column:provider_state;not null;default:''"`
	LastPolledAt  *time.Time `gorm:"column:last_polled_at"`
	PollCount     int        `gorm:"column:poll_count;not null;default:0"`
	CreatedAt     time.Time  `gorm:"column:created_at;not null;autoCreateTime"`
	VerifiedAt    *time.Time `gorm:"column:verified_at"`
}

func (realnameVerificationRow) TableName() string { return "realname_verifications" }

// refundCleanupJobRow is a durable outbox entry for deleting a refunded VPS
// host. It survives restarts, is claimed with a lease+fence by exactly one
// worker at a time, and is retried with bounded backoff until the host is gone
// and the local instance state is updated.
type refundCleanupJobRow struct {
	ID           int64      `gorm:"primaryKey;autoIncrement;column:id"`
	WalletOrderID int64     `gorm:"column:wallet_order_id;not null;uniqueIndex:idx_refund_cleanup_wallet_order"`
	UserID       int64      `gorm:"column:user_id;not null;index"`
	VPSID        int64      `gorm:"column:vps_id;not null;index"`
	HostID       int64      `gorm:"column:host_id;not null"`
	GoodsTypeID  int64      `gorm:"column:goods_type_id;not null;default:0"`
	Status       string     `gorm:"size:32;column:status;not null;default:pending;index"`
	Attempts     int        `gorm:"column:attempts;not null;default:0"`
	NextRunAt    time.Time  `gorm:"column:next_run_at;not null"`
	LastErrorCode string   `gorm:"size:32;column:last_error_code;not null;default:''"`
	LastError    string     `gorm:"column:last_error;not null;default:''"`
	LeaseOwner   string     `gorm:"size:128;column:lease_owner;not null;default:''"`
	LeaseUntil   *time.Time `gorm:"column:lease_until;index"`
	Fence        int        `gorm:"column:fence;not null;default:0"`
	CreatedAt    time.Time  `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt    time.Time  `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (refundCleanupJobRow) TableName() string { return "refund_cleanup_jobs" }

// realnameAttemptRow backs the database realname attempt budget. A unique
// (user_id, dimension, window_start) row is incremented atomically per attempt;
// the realname service rejects once the count exceeds the configured max, which
// is enforced consistently across instances and restarts (unlike a process map).
type realnameAttemptRow struct {
	ID          int64     `gorm:"primaryKey;autoIncrement;column:id"`
	UserID      int64     `gorm:"column:user_id;not null;uniqueIndex:uniq_realname_attempts_window,priority:1"`
	Dimension   string    `gorm:"size:191;column:dimension;not null;uniqueIndex:uniq_realname_attempts_window,priority:2"`
	WindowStart time.Time `gorm:"column:window_start;not null;uniqueIndex:uniq_realname_attempts_window,priority:3;index"`
	Count       int       `gorm:"column:count;not null;default:0"`
	UpdatedAt   time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (realnameAttemptRow) TableName() string { return "realname_attempts" }

// rateLimitBucketRow backs the DB atomic window budget used by cost-sensitive
// endpoints (SMS/SMTP test sends). A unique (bucket_key, window_start) row is
// incremented atomically per call; the caller rejects once the count exceeds
// the configured max. Being DB-backed it is consistent across instances and
// restarts, unlike a process map.
type rateLimitBucketRow struct {
	ID          int64     `gorm:"primaryKey;autoIncrement;column:id"`
	BucketKey   string    `gorm:"size:191;column:bucket_key;not null;uniqueIndex:uniq_rate_limit_window,priority:1"`
	WindowStart time.Time `gorm:"column:window_start;not null;uniqueIndex:uniq_rate_limit_window,priority:2;index"`
	Count       int       `gorm:"column:count;not null;default:0"`
	UpdatedAt   time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (rateLimitBucketRow) TableName() string { return "rate_limit_buckets" }
