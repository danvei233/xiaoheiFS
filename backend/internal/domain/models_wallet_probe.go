package domain

import "time"

type Wallet struct {
	ID        int64
	UserID    int64
	Balance   int64
	UpdatedAt time.Time
}

type WalletTransaction struct {
	ID        int64
	UserID    int64
	Amount    int64
	Type      string
	RefType   string
	RefID     int64
	Note      string
	CreatedAt time.Time
}

type WalletOrder struct {
	ID           int64
	UserID       int64
	Type         WalletOrderType
	Amount       int64
	Currency     string
	Status       WalletOrderStatus
	Note         string
	MetaJSON     string
	ReviewedBy   *int64
	ReviewReason string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type ProbeStatus string

const (
	ProbeStatusOffline ProbeStatus = "offline"
	ProbeStatusOnline  ProbeStatus = "online"
)

type ProbeNode struct {
	ID               int64
	Name             string
	AgentID          string
	SecretHash       string
	Status           ProbeStatus
	OSType           string
	TagsJSON         string
	LastHeartbeatAt  *time.Time
	LastSnapshotAt   *time.Time
	LastSnapshotJSON string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type ProbeEnrollToken struct {
	ID        int64
	ProbeID   int64
	TokenHash string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

type ProbeStatusEvent struct {
	ID        int64
	ProbeID   int64
	Status    ProbeStatus
	At        time.Time
	Reason    string
	CreatedAt time.Time
}

type ProbeLogSession struct {
	ID         int64
	ProbeID    int64
	OperatorID int64
	Source     string
	Status     string
	StartedAt  time.Time
	EndedAt    *time.Time
	CreatedAt  time.Time
}
