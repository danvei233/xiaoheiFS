package repo

import (
	"time"
)

func (cmsCategoryRow) TableName() string { return "cms_categories" }

type cmsPostRow struct {
	ID          int64      `gorm:"primaryKey;autoIncrement;column:id"`
	CategoryID  int64      `gorm:"column:category_id;not null;index"`
	Title       string     `gorm:"column:title;not null"`
	Slug        string     `gorm:"size:191;column:slug;not null;uniqueIndex"`
	Summary     string     `gorm:"column:summary;not null;default:''"`
	ContentHTML string     `gorm:"type:longtext;column:content_html;not null"`
	CoverURL    string     `gorm:"column:cover_url;not null;default:''"`
	Lang        string     `gorm:"size:191;column:lang;not null;default:zh-CN;index"`
	Status      string     `gorm:"column:status;not null;default:draft"`
	Pinned      int        `gorm:"column:pinned;not null;default:0"`
	SortOrder   int        `gorm:"column:sort_order;not null;default:0"`
	PublishedAt *time.Time `gorm:"column:published_at"`
	CreatedAt   time.Time  `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (cmsPostRow) TableName() string { return "cms_posts" }

type cmsBlockRow struct {
	ID          int64     `gorm:"primaryKey;autoIncrement;column:id"`
	Page        string    `gorm:"size:191;column:page;not null;index"`
	Type        string    `gorm:"column:type;not null"`
	Title       string    `gorm:"column:title;not null;default:''"`
	Subtitle    string    `gorm:"column:subtitle;not null;default:''"`
	ContentJSON string    `gorm:"type:longtext;column:content_json;not null"`
	CustomHTML  string    `gorm:"type:longtext;column:custom_html;not null"`
	Lang        string    `gorm:"column:lang;not null;default:zh-CN"`
	Visible     int       `gorm:"column:visible;not null;default:1"`
	SortOrder   int       `gorm:"column:sort_order;not null;default:0"`
	CreatedAt   time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt   time.Time `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (cmsBlockRow) TableName() string { return "cms_blocks" }

type uploadRow struct {
	ID         int64     `gorm:"primaryKey;autoIncrement;column:id"`
	Name       string    `gorm:"column:name;not null"`
	Path       string    `gorm:"column:path;not null"`
	URL        string    `gorm:"column:url;not null"`
	Mime       string    `gorm:"column:mime;not null"`
	Size       int64     `gorm:"column:size;not null"`
	UploaderID int64     `gorm:"column:uploader_id;not null;index"`
	CreatedAt  time.Time `gorm:"column:created_at;not null;autoCreateTime"`
}

func (uploadRow) TableName() string { return "uploads" }

type ticketRow struct {
	ID            int64      `gorm:"primaryKey;autoIncrement;column:id"`
	UserID        int64      `gorm:"column:user_id;not null;index"`
	Subject       string     `gorm:"size:240;column:subject;not null"`
	Status        string     `gorm:"size:191;column:status;not null;default:open;index"`
	LastReplyAt   *time.Time `gorm:"column:last_reply_at"`
	LastReplyBy   *int64     `gorm:"column:last_reply_by"`
	LastReplyRole string     `gorm:"column:last_reply_role;not null;default:user"`
	ClosedAt      *time.Time `gorm:"column:closed_at"`
	CreatedAt     time.Time  `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (ticketRow) TableName() string { return "tickets" }

type ticketMessageRow struct {
	ID         int64     `gorm:"primaryKey;autoIncrement;column:id"`
	TicketID   int64     `gorm:"column:ticket_id;not null;index"`
	SenderID   int64     `gorm:"column:sender_id;not null"`
	SenderRole string    `gorm:"column:sender_role;not null"`
	SenderName string    `gorm:"column:sender_name"`
	SenderQQ   string    `gorm:"size:32;column:sender_qq"`
	Content    string    `gorm:"size:10000;column:content;not null"`
	CreatedAt  time.Time `gorm:"column:created_at;not null;autoCreateTime"`
}

func (ticketMessageRow) TableName() string { return "ticket_messages" }

type ticketResourceRow struct {
	ID           int64     `gorm:"primaryKey;autoIncrement;column:id"`
	TicketID     int64     `gorm:"column:ticket_id;not null;index"`
	ResourceType string    `gorm:"column:resource_type;not null"`
	ResourceID   int64     `gorm:"column:resource_id;not null;default:0"`
	ResourceName string    `gorm:"size:128;column:resource_name;not null;default:''"`
	CreatedAt    time.Time `gorm:"column:created_at;not null;autoCreateTime"`
}

func (ticketResourceRow) TableName() string { return "ticket_resources" }
