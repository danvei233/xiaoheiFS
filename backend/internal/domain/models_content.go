package domain

import "time"

type CMSCategory struct {
	ID        int64
	Key       string
	Name      string
	Lang      string
	SortOrder int
	Visible   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CMSPost struct {
	ID          int64
	CategoryID  int64
	Title       string
	Slug        string
	Summary     string
	ContentHTML string
	CoverURL    string
	Lang        string
	Status      string
	Pinned      bool
	SortOrder   int
	PublishedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CMSBlock struct {
	ID          int64
	Page        string
	Type        string
	Title       string
	Subtitle    string
	ContentJSON string
	CustomHTML  string
	Lang        string
	Visible     bool
	SortOrder   int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Upload struct {
	ID         int64
	Name       string
	Path       string
	URL        string
	Mime       string
	Size       int64
	UploaderID int64
	CreatedAt  time.Time
}

type Ticket struct {
	ID            int64
	UserID        int64
	Subject       string
	Status        string
	ResourceCount int
	LastReplyAt   *time.Time
	LastReplyBy   *int64
	LastReplyRole string
	ClosedAt      *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type TicketMessage struct {
	ID         int64
	TicketID   int64
	SenderID   int64
	SenderRole string
	SenderName string
	SenderQQ   string
	Content    string
	CreatedAt  time.Time
}

type TicketResource struct {
	ID           int64
	TicketID     int64
	ResourceType string
	ResourceID   int64
	ResourceName string
	CreatedAt    time.Time
}

type Notification struct {
	ID        int64
	UserID    int64
	Type      string
	Title     string
	Content   string
	ReadAt    *time.Time
	CreatedAt time.Time
}

type PushToken struct {
	ID        int64
	UserID    int64
	Platform  string
	Token     string `json:"-"`
	DeviceID  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type RealNameVerification struct {
	ID       int64
	UserID   int64
	RealName string
	IDNumber string
	Status   string
	Provider string
	Reason   string
	// ProviderState holds the versioned encrypted provider continuation state
	// (pending KYC token / redirect URL). It is internal and never returned by
	// the API; only the realname service can decrypt it.
	ProviderState string
	// LastPolledAt and PollCount bound how often a pending record is re-queried
	// against the paid provider, and when a record gives up (PollCount cap).
	LastPolledAt *time.Time
	PollCount    int
	CreatedAt    time.Time
	VerifiedAt   *time.Time
}
