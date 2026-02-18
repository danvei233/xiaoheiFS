package repo

import (
	"time"
)

type probeNodeRow struct {
	ID               int64      `gorm:"primaryKey;autoIncrement;column:id"`
	Name             string     `gorm:"column:name;not null"`
	AgentID          string     `gorm:"size:191;column:agent_id;not null;uniqueIndex"`
	SecretHash       string     `gorm:"size:191;column:secret_hash;not null"`
	Status           string     `gorm:"size:32;column:status;not null;default:offline;index:idx_probe_nodes_status"`
	OSType           string     `gorm:"size:32;column:os_type;not null;default:''"`
	TagsJSON         string     `gorm:"column:tags_json;not null;default:'[]'"`
	LastHeartbeatAt  *time.Time `gorm:"column:last_heartbeat_at;index:idx_probe_nodes_heartbeat"`
	LastSnapshotAt   *time.Time `gorm:"column:last_snapshot_at"`
	LastSnapshotJSON string     `gorm:"type:longtext;column:last_snapshot_json;not null"`
	CreatedAt        time.Time  `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt        time.Time  `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (probeNodeRow) TableName() string { return "probe_nodes" }

type probeEnrollTokenRow struct {
	ID        int64      `gorm:"primaryKey;autoIncrement;column:id"`
	ProbeID   int64      `gorm:"column:probe_id;not null;index:idx_probe_enroll_tokens_probe"`
	TokenHash string     `gorm:"size:191;column:token_hash;not null;uniqueIndex"`
	ExpiresAt time.Time  `gorm:"column:expires_at;not null;index:idx_probe_enroll_tokens_expires"`
	UsedAt    *time.Time `gorm:"column:used_at"`
	CreatedAt time.Time  `gorm:"column:created_at;not null;autoCreateTime"`
}

func (probeEnrollTokenRow) TableName() string { return "probe_enroll_tokens" }

type probeStatusEventRow struct {
	ID        int64     `gorm:"primaryKey;autoIncrement;column:id"`
	ProbeID   int64     `gorm:"column:probe_id;not null;index:idx_probe_status_events_probe_at,priority:1"`
	Status    string    `gorm:"size:32;column:status;not null"`
	At        time.Time `gorm:"column:at;not null;index:idx_probe_status_events_probe_at,priority:2"`
	Reason    string    `gorm:"column:reason;not null;default:''"`
	CreatedAt time.Time `gorm:"column:created_at;not null;autoCreateTime"`
}

func (probeStatusEventRow) TableName() string { return "probe_status_events" }

type probeLogSessionRow struct {
	ID         int64      `gorm:"primaryKey;autoIncrement;column:id"`
	ProbeID    int64      `gorm:"column:probe_id;not null;index:idx_probe_log_sessions_probe"`
	OperatorID int64      `gorm:"column:operator_id;not null;default:0"`
	Source     string     `gorm:"column:source;not null;default:''"`
	Status     string     `gorm:"size:32;column:status;not null;default:running"`
	StartedAt  time.Time  `gorm:"column:started_at;not null"`
	EndedAt    *time.Time `gorm:"column:ended_at"`
	CreatedAt  time.Time  `gorm:"column:created_at;not null;autoCreateTime"`
}

func (probeLogSessionRow) TableName() string { return "probe_log_sessions" }
