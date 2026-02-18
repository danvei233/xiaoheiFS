package repo

import (
	"database/sql"

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
