package repo

import (
	"database/sql"
	"strings"
	"time"

	"xiaoheiplay/internal/domain"
)

func scanUser(row scanner) (domain.User, error) {
	var u domain.User
	var email sql.NullString
	var qq sql.NullString
	var avatar sql.NullString
	var phone sql.NullString
	var bio sql.NullString
	var intro sql.NullString
	var permissionGroupID sql.NullInt64
	var createdAt sql.NullTime
	var updatedAt sql.NullTime
	if err := row.Scan(&u.ID, &u.Username, &email, &qq, &avatar, &phone, &bio, &intro, &permissionGroupID, &u.PasswordHash, &u.Role, &u.Status, &createdAt, &updatedAt); err != nil {
		return domain.User{}, rEnsure(err)
	}
	if email.Valid {
		u.Email = email.String
	}
	if qq.Valid {
		u.QQ = qq.String
	}
	if avatar.Valid {
		u.Avatar = avatar.String
	}
	if phone.Valid {
		u.Phone = phone.String
	}
	if bio.Valid {
		u.Bio = bio.String
	}
	if intro.Valid {
		u.Intro = intro.String
	}
	if permissionGroupID.Valid {
		u.PermissionGroupID = &permissionGroupID.Int64
	}
	if createdAt.Valid {
		u.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		u.UpdatedAt = updatedAt.Time
	}
	return u, nil
}

func scanWalletOrder(row scanner) (domain.WalletOrder, error) {
	var order domain.WalletOrder
	var reviewed sql.NullInt64
	var reason sql.NullString
	var createdAt sql.NullTime
	var updatedAt sql.NullTime
	if err := row.Scan(&order.ID, &order.UserID, &order.Type, &order.Amount, &order.Currency, &order.Status, &order.Note, &order.MetaJSON, &reviewed, &reason, &createdAt, &updatedAt); err != nil {
		return domain.WalletOrder{}, rEnsure(err)
	}
	if reviewed.Valid {
		order.ReviewedBy = &reviewed.Int64
	}
	if reason.Valid {
		order.ReviewReason = reason.String
	}
	if createdAt.Valid {
		order.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		order.UpdatedAt = updatedAt.Time
	}
	return order, nil
}

func scanOrder(row scanner) (domain.Order, error) {
	var o domain.Order
	var idem sql.NullString
	var approvedBy sql.NullInt64
	var approvedAt sql.NullTime
	var rejectedReason sql.NullString
	var pendingReason sql.NullString
	if err := row.Scan(&o.ID, &o.UserID, &o.OrderNo, &o.Status, &o.TotalAmount, &o.Currency, &idem, &pendingReason, &approvedBy, &approvedAt, &rejectedReason, &o.CreatedAt, &o.UpdatedAt); err != nil {
		return domain.Order{}, rEnsure(err)
	}
	if idem.Valid {
		o.IdempotencyKey = idem.String
	}
	if approvedBy.Valid {
		o.ApprovedBy = &approvedBy.Int64
	}
	if approvedAt.Valid {
		o.ApprovedAt = &approvedAt.Time
	}
	if pendingReason.Valid {
		o.PendingReason = pendingReason.String
	}
	if rejectedReason.Valid {
		o.RejectedReason = rejectedReason.String
	}
	return o, nil
}

func scanOrderItem(row scanner) (domain.OrderItem, error) {
	var item domain.OrderItem
	if err := row.Scan(&item.ID, &item.OrderID, &item.PackageID, &item.SystemID, &item.SpecJSON, &item.Qty, &item.Amount, &item.Status, &item.GoodsTypeID, &item.AutomationInstanceID, &item.Action, &item.DurationMonths, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return domain.OrderItem{}, rEnsure(err)
	}
	return item, nil
}

func scanCartItem(row scanner) (domain.CartItem, error) {
	var item domain.CartItem
	if err := row.Scan(&item.ID, &item.UserID, &item.PackageID, &item.SystemID, &item.SpecJSON, &item.Qty, &item.Amount, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return domain.CartItem{}, rEnsure(err)
	}
	return item, nil
}

func scanSystemImage(row scanner) (domain.SystemImage, error) {
	var img domain.SystemImage
	var enabled int
	if err := row.Scan(&img.ID, &img.ImageID, &img.Name, &img.Type, &enabled, &img.CreatedAt, &img.UpdatedAt); err != nil {
		return domain.SystemImage{}, rEnsure(err)
	}
	img.Enabled = enabled == 1
	return img, nil
}

func scanVPSInstance(row scanner) (domain.VPSInstance, error) {
	var inst domain.VPSInstance
	var expire sql.NullTime
	var adminStatus sql.NullString
	var lastEmergency sql.NullTime
	var panelURL sql.NullString
	var accessInfo sql.NullString
	if err := row.Scan(&inst.ID, &inst.UserID, &inst.OrderItemID, &inst.AutomationInstanceID, &inst.GoodsTypeID, &inst.Name, &inst.Region, &inst.RegionID, &inst.LineID, &inst.PackageID, &inst.PackageName, &inst.CPU, &inst.MemoryGB, &inst.DiskGB, &inst.BandwidthMB, &inst.PortNum, &inst.MonthlyPrice, &inst.SpecJSON, &inst.SystemID, &inst.Status, &inst.AutomationState, &adminStatus, &expire, &panelURL, &accessInfo, &lastEmergency, &inst.CreatedAt, &inst.UpdatedAt); err != nil {
		return domain.VPSInstance{}, rEnsure(err)
	}
	if expire.Valid {
		inst.ExpireAt = &expire.Time
	}
	if adminStatus.Valid {
		inst.AdminStatus = domain.VPSAdminStatus(adminStatus.String)
	} else {
		inst.AdminStatus = domain.VPSAdminStatusNormal
	}
	if panelURL.Valid {
		inst.PanelURLCache = panelURL.String
	}
	if accessInfo.Valid {
		inst.AccessInfoJSON = accessInfo.String
	}
	if lastEmergency.Valid {
		inst.LastEmergencyRenewAt = &lastEmergency.Time
	}
	return inst, nil
}

func scanAPIKey(row scanner) (domain.APIKey, error) {
	var key domain.APIKey
	var lastUsed sql.NullTime
	var groupID sql.NullInt64
	if err := row.Scan(&key.ID, &key.Name, &key.KeyHash, &key.Status, &key.ScopesJSON, &groupID, &key.CreatedAt, &key.UpdatedAt, &lastUsed); err != nil {
		return domain.APIKey{}, rEnsure(err)
	}
	if groupID.Valid {
		v := groupID.Int64
		key.PermissionGroupID = &v
	}
	if lastUsed.Valid {
		key.LastUsedAt = &lastUsed.Time
	}
	return key, nil
}

func scanEmailTemplate(row scanner) (domain.EmailTemplate, error) {
	var tmpl domain.EmailTemplate
	var enabled int
	if err := row.Scan(&tmpl.ID, &tmpl.Name, &tmpl.Subject, &tmpl.Body, &enabled, &tmpl.CreatedAt, &tmpl.UpdatedAt); err != nil {
		return domain.EmailTemplate{}, rEnsure(err)
	}
	tmpl.Enabled = enabled == 1
	return tmpl, nil
}

func scanOrderPayment(row scanner) (domain.OrderPayment, error) {
	var pay domain.OrderPayment
	var reviewedBy sql.NullInt64
	var reviewReason sql.NullString
	var idem sql.NullString
	if err := row.Scan(&pay.ID, &pay.OrderID, &pay.UserID, &pay.Method, &pay.Amount, &pay.Currency, &pay.TradeNo, &pay.Note, &pay.ScreenshotURL, &pay.Status, &idem, &reviewedBy, &reviewReason, &pay.CreatedAt, &pay.UpdatedAt); err != nil {
		return domain.OrderPayment{}, rEnsure(err)
	}
	if idem.Valid {
		pay.IdempotencyKey = idem.String
	}
	if reviewedBy.Valid {
		pay.ReviewedBy = &reviewedBy.Int64
	}
	if reviewReason.Valid {
		pay.ReviewReason = reviewReason.String
	}
	return pay, nil
}

func scanRealNameVerification(row scanner) (domain.RealNameVerification, error) {
	var record domain.RealNameVerification
	var verifiedAt sql.NullTime
	if err := row.Scan(&record.ID, &record.UserID, &record.RealName, &record.IDNumber, &record.Status, &record.Provider, &record.Reason, &record.CreatedAt, &verifiedAt); err != nil {
		return domain.RealNameVerification{}, rEnsure(err)
	}
	if verifiedAt.Valid {
		record.VerifiedAt = &verifiedAt.Time
	}
	return record, nil
}

func scanBillingCycle(row scanner) (domain.BillingCycle, error) {
	var cycle domain.BillingCycle
	var active int
	if err := row.Scan(&cycle.ID, &cycle.Name, &cycle.Months, &cycle.Multiplier, &cycle.MinQty, &cycle.MaxQty, &active, &cycle.SortOrder, &cycle.CreatedAt, &cycle.UpdatedAt); err != nil {
		return domain.BillingCycle{}, rEnsure(err)
	}
	cycle.Active = active == 1
	return cycle, nil
}

func scanAutomationLog(row scanner) (domain.AutomationLog, error) {
	var logEntry domain.AutomationLog
	var success int
	if err := row.Scan(&logEntry.ID, &logEntry.OrderID, &logEntry.OrderItemID, &logEntry.Action, &logEntry.RequestJSON, &logEntry.ResponseJSON, &success, &logEntry.Message, &logEntry.CreatedAt); err != nil {
		return domain.AutomationLog{}, rEnsure(err)
	}
	logEntry.Success = success == 1
	return logEntry, nil
}

func scanProvisionJob(row scanner) (domain.ProvisionJob, error) {
	var job domain.ProvisionJob
	if err := row.Scan(&job.ID, &job.OrderID, &job.OrderItemID, &job.HostID, &job.HostName, &job.Status, &job.Attempts, &job.NextRunAt, &job.LastError, &job.CreatedAt, &job.UpdatedAt); err != nil {
		return domain.ProvisionJob{}, rEnsure(err)
	}
	return job, nil
}

func scanResizeTask(row scanner) (domain.ResizeTask, error) {
	var task domain.ResizeTask
	var scheduledAt sql.NullTime
	var startedAt sql.NullTime
	var finishedAt sql.NullTime
	if err := row.Scan(&task.ID, &task.VPSID, &task.OrderID, &task.OrderItemID, &task.Status, &scheduledAt, &startedAt, &finishedAt, &task.CreatedAt, &task.UpdatedAt); err != nil {
		return domain.ResizeTask{}, rEnsure(err)
	}
	if scheduledAt.Valid {
		task.ScheduledAt = &scheduledAt.Time
	}
	if startedAt.Valid {
		task.StartedAt = &startedAt.Time
	}
	if finishedAt.Valid {
		task.FinishedAt = &finishedAt.Time
	}
	return task, nil
}

func scanIntegrationLog(row scanner) (domain.IntegrationSyncLog, error) {
	var logEntry domain.IntegrationSyncLog
	if err := row.Scan(&logEntry.ID, &logEntry.Target, &logEntry.Mode, &logEntry.Status, &logEntry.Message, &logEntry.CreatedAt); err != nil {
		return domain.IntegrationSyncLog{}, rEnsure(err)
	}
	return logEntry, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func toUserRow(u domain.User) userRow {
	var email *string
	if strings.TrimSpace(u.Email) != "" {
		v := strings.TrimSpace(u.Email)
		email = &v
	}
	return userRow{
		ID:                   u.ID,
		Username:             u.Username,
		Email:                email,
		QQ:                   u.QQ,
		Avatar:               u.Avatar,
		Phone:                u.Phone,
		LastLoginIP:          u.LastLoginIP,
		LastLoginAt:          u.LastLoginAt,
		LastLoginCity:        u.LastLoginCity,
		LastLoginTZ:          u.LastLoginTZ,
		TOTPEnabled:          boolToInt(u.TOTPEnabled),
		TOTPSecretEnc:        u.TOTPSecretEnc,
		TOTPPendingSecretEnc: u.TOTPPendingSecretEnc,
		Bio:                  u.Bio,
		Intro:                u.Intro,
		PermissionGroupID:    u.PermissionGroupID,
		PasswordHash:         u.PasswordHash,
		Role:                 string(u.Role),
		Status:               string(u.Status),
		CreatedAt:            u.CreatedAt,
		UpdatedAt:            u.UpdatedAt,
	}
}

func fromUserRow(r userRow) domain.User {
	var email string
	if r.Email != nil {
		email = *r.Email
	}
	return domain.User{
		ID:                   r.ID,
		Username:             r.Username,
		Email:                email,
		QQ:                   r.QQ,
		Avatar:               r.Avatar,
		Phone:                r.Phone,
		LastLoginIP:          r.LastLoginIP,
		LastLoginAt:          r.LastLoginAt,
		LastLoginCity:        r.LastLoginCity,
		LastLoginTZ:          r.LastLoginTZ,
		TOTPEnabled:          r.TOTPEnabled == 1,
		TOTPSecretEnc:        r.TOTPSecretEnc,
		TOTPPendingSecretEnc: r.TOTPPendingSecretEnc,
		Bio:                  r.Bio,
		Intro:                r.Intro,
		PermissionGroupID:    r.PermissionGroupID,
		PasswordHash:         r.PasswordHash,
		Role:                 domain.UserRole(r.Role),
		Status:               domain.UserStatus(r.Status),
		CreatedAt:            r.CreatedAt,
		UpdatedAt:            r.UpdatedAt,
	}
}

func toCartItemRow(item domain.CartItem) cartItemRow {
	return cartItemRow{
		ID:        item.ID,
		UserID:    item.UserID,
		PackageID: item.PackageID,
		SystemID:  item.SystemID,
		SpecJSON:  item.SpecJSON,
		Qty:       item.Qty,
		Amount:    item.Amount,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}

func fromCartItemRow(r cartItemRow) domain.CartItem {
	return domain.CartItem{
		ID:        r.ID,
		UserID:    r.UserID,
		PackageID: r.PackageID,
		SystemID:  r.SystemID,
		SpecJSON:  r.SpecJSON,
		Qty:       r.Qty,
		Amount:    r.Amount,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

func toOrderRow(order domain.Order) orderRow {
	var idem *string
	if strings.TrimSpace(order.IdempotencyKey) != "" {
		v := strings.TrimSpace(order.IdempotencyKey)
		idem = &v
	}
	return orderRow{
		ID:             order.ID,
		UserID:         order.UserID,
		OrderNo:        order.OrderNo,
		Status:         string(order.Status),
		TotalAmount:    order.TotalAmount,
		Currency:       order.Currency,
		IdempotencyKey: idem,
		PendingReason:  order.PendingReason,
		ApprovedBy:     order.ApprovedBy,
		ApprovedAt:     order.ApprovedAt,
		RejectedReason: order.RejectedReason,
		CreatedAt:      order.CreatedAt,
		UpdatedAt:      order.UpdatedAt,
	}
}

func fromOrderRow(r orderRow) domain.Order {
	out := domain.Order{
		ID:             r.ID,
		UserID:         r.UserID,
		OrderNo:        r.OrderNo,
		Status:         domain.OrderStatus(r.Status),
		TotalAmount:    r.TotalAmount,
		Currency:       r.Currency,
		PendingReason:  r.PendingReason,
		ApprovedBy:     r.ApprovedBy,
		ApprovedAt:     r.ApprovedAt,
		RejectedReason: r.RejectedReason,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
	if r.IdempotencyKey != nil {
		out.IdempotencyKey = *r.IdempotencyKey
	}
	return out
}

func toOrderItemRow(item domain.OrderItem) orderItemRow {
	return orderItemRow{
		ID:                   item.ID,
		OrderID:              item.OrderID,
		PackageID:            item.PackageID,
		SystemID:             item.SystemID,
		SpecJSON:             item.SpecJSON,
		Qty:                  item.Qty,
		Amount:               item.Amount,
		Status:               string(item.Status),
		GoodsTypeID:          item.GoodsTypeID,
		AutomationInstanceID: item.AutomationInstanceID,
		Action:               item.Action,
		DurationMonths:       item.DurationMonths,
		CreatedAt:            item.CreatedAt,
		UpdatedAt:            item.UpdatedAt,
	}
}

func fromOrderItemRow(r orderItemRow) domain.OrderItem {
	return domain.OrderItem{
		ID:                   r.ID,
		OrderID:              r.OrderID,
		PackageID:            r.PackageID,
		SystemID:             r.SystemID,
		SpecJSON:             r.SpecJSON,
		Qty:                  r.Qty,
		Amount:               r.Amount,
		Status:               domain.OrderItemStatus(r.Status),
		GoodsTypeID:          r.GoodsTypeID,
		AutomationInstanceID: r.AutomationInstanceID,
		Action:               r.Action,
		DurationMonths:       r.DurationMonths,
		CreatedAt:            r.CreatedAt,
		UpdatedAt:            r.UpdatedAt,
	}
}

func toVPSInstanceRow(inst domain.VPSInstance) vpsInstanceRow {
	return vpsInstanceRow{
		ID:                   inst.ID,
		UserID:               inst.UserID,
		OrderItemID:          inst.OrderItemID,
		AutomationInstanceID: inst.AutomationInstanceID,
		GoodsTypeID:          inst.GoodsTypeID,
		Name:                 inst.Name,
		Region:               inst.Region,
		RegionID:             inst.RegionID,
		LineID:               inst.LineID,
		PackageID:            inst.PackageID,
		PackageName:          inst.PackageName,
		CPU:                  inst.CPU,
		MemoryGB:             inst.MemoryGB,
		DiskGB:               inst.DiskGB,
		BandwidthMbps:        inst.BandwidthMB,
		PortNum:              inst.PortNum,
		MonthlyPrice:         inst.MonthlyPrice,
		SpecJSON:             inst.SpecJSON,
		SystemID:             inst.SystemID,
		Status:               string(inst.Status),
		AutomationState:      inst.AutomationState,
		AdminStatus:          string(inst.AdminStatus),
		ExpireAt:             inst.ExpireAt,
		PanelURLCache:        inst.PanelURLCache,
		AccessInfoJSON:       inst.AccessInfoJSON,
		LastEmergencyRenewAt: inst.LastEmergencyRenewAt,
		CreatedAt:            inst.CreatedAt,
		UpdatedAt:            inst.UpdatedAt,
	}
}

func fromVPSInstanceRow(r vpsInstanceRow) domain.VPSInstance {
	return domain.VPSInstance{
		ID:                   r.ID,
		UserID:               r.UserID,
		OrderItemID:          r.OrderItemID,
		AutomationInstanceID: r.AutomationInstanceID,
		GoodsTypeID:          r.GoodsTypeID,
		Name:                 r.Name,
		Region:               r.Region,
		RegionID:             r.RegionID,
		LineID:               r.LineID,
		PackageID:            r.PackageID,
		PackageName:          r.PackageName,
		CPU:                  r.CPU,
		MemoryGB:             r.MemoryGB,
		DiskGB:               r.DiskGB,
		BandwidthMB:          r.BandwidthMbps,
		PortNum:              r.PortNum,
		MonthlyPrice:         r.MonthlyPrice,
		SpecJSON:             r.SpecJSON,
		SystemID:             r.SystemID,
		Status:               domain.VPSStatus(r.Status),
		AutomationState:      r.AutomationState,
		AdminStatus:          domain.VPSAdminStatus(r.AdminStatus),
		ExpireAt:             r.ExpireAt,
		PanelURLCache:        r.PanelURLCache,
		AccessInfoJSON:       r.AccessInfoJSON,
		LastEmergencyRenewAt: r.LastEmergencyRenewAt,
		CreatedAt:            r.CreatedAt,
		UpdatedAt:            r.UpdatedAt,
	}
}

func toOrderPaymentRow(pay domain.OrderPayment) orderPaymentRow {
	var note *string
	if strings.TrimSpace(pay.Note) != "" {
		v := pay.Note
		note = &v
	}
	var screenshot *string
	if strings.TrimSpace(pay.ScreenshotURL) != "" {
		v := pay.ScreenshotURL
		screenshot = &v
	}
	var idem *string
	if strings.TrimSpace(pay.IdempotencyKey) != "" {
		v := pay.IdempotencyKey
		idem = &v
	}
	return orderPaymentRow{
		ID:             pay.ID,
		OrderID:        pay.OrderID,
		UserID:         pay.UserID,
		Method:         pay.Method,
		Amount:         pay.Amount,
		Currency:       pay.Currency,
		TradeNo:        pay.TradeNo,
		Note:           note,
		ScreenshotURL:  screenshot,
		Status:         string(pay.Status),
		IdempotencyKey: idem,
		ReviewedBy:     pay.ReviewedBy,
		ReviewReason:   pay.ReviewReason,
		CreatedAt:      pay.CreatedAt,
		UpdatedAt:      pay.UpdatedAt,
	}
}

func fromOrderPaymentRow(r orderPaymentRow) domain.OrderPayment {
	out := domain.OrderPayment{
		ID:           r.ID,
		OrderID:      r.OrderID,
		UserID:       r.UserID,
		Method:       r.Method,
		Amount:       r.Amount,
		Currency:     r.Currency,
		TradeNo:      r.TradeNo,
		Status:       domain.PaymentStatus(r.Status),
		ReviewedBy:   r.ReviewedBy,
		ReviewReason: r.ReviewReason,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
	if r.Note != nil {
		out.Note = *r.Note
	}
	if r.ScreenshotURL != nil {
		out.ScreenshotURL = *r.ScreenshotURL
	}
	if r.IdempotencyKey != nil {
		out.IdempotencyKey = *r.IdempotencyKey
	}
	return out
}

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
