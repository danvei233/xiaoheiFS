package domain

import "time"

type AdminAuditLog struct {
	ID         int64
	AdminID    int64
	Action     string
	TargetType string
	TargetID   string
	DetailJSON string
	CreatedAt  time.Time
}

type EmailTemplate struct {
	ID        int64
	Name      string
	Subject   string
	Body      string
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Setting struct {
	Key       string
	ValueJSON string
	UpdatedAt time.Time
}

type AutomationLog struct {
	ID           int64
	OrderID      int64
	OrderItemID  int64
	Action       string
	RequestJSON  string
	ResponseJSON string
	Success      bool
	Message      string
	CreatedAt    time.Time
}

type ScheduledTaskRun struct {
	ID          int64
	TaskKey     string
	Status      string
	StartedAt   time.Time
	FinishedAt  *time.Time
	DurationSec int
	Message     string
	CreatedAt   time.Time
}

type IntegrationSyncLog struct {
	ID        int64
	Target    string
	Mode      string
	Status    string
	Message   string
	CreatedAt time.Time
}

type PermissionGroup struct {
	ID              int64
	Name            string
	Description     string
	PermissionsJSON string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Permission struct {
	ID           int64
	Code         string
	Name         string
	FriendlyName string
	Category     string
	ParentCode   string
	SortOrder    int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type PermissionDefinition struct {
	Code         string
	Name         string
	FriendlyName string
	Category     string
	ParentCode   string
	SortOrder    int
}

type PermissionTree struct {
	Code         string
	Name         string
	FriendlyName string
	Category     string
	Children     []PermissionTree
}
