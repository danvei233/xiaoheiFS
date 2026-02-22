package repo

import (
	"strings"

	"xiaoheiplay/internal/domain"
)

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
		UserTierGroupID:      u.UserTierGroupID,
		UserTierExpireAt:     u.UserTierExpireAt,
		PasswordHash:         u.PasswordHash,
		PasswordChangedAt:    u.PasswordChangedAt,
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
		UserTierGroupID:      r.UserTierGroupID,
		UserTierExpireAt:     r.UserTierExpireAt,
		PasswordHash:         r.PasswordHash,
		PasswordChangedAt:    r.PasswordChangedAt,
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
	source := strings.TrimSpace(order.Source)
	if source == "" {
		source = "user_ui"
	}
	return orderRow{
		ID:             order.ID,
		UserID:         order.UserID,
		OrderNo:        order.OrderNo,
		Source:         source,
		Status:         string(order.Status),
		TotalAmount:    order.TotalAmount,
		Currency:       order.Currency,
		CouponID:       order.CouponID,
		CouponCode:     order.CouponCode,
		CouponDiscount: order.CouponDiscount,
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
		Source:         r.Source,
		Status:         domain.OrderStatus(r.Status),
		TotalAmount:    r.TotalAmount,
		Currency:       r.Currency,
		CouponID:       r.CouponID,
		CouponCode:     r.CouponCode,
		CouponDiscount: r.CouponDiscount,
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
