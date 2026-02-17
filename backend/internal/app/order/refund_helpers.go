package order

import (
	"context"
	"math"
	"time"

	"xiaoheiplay/internal/domain"
)

func loadRefundPolicy(ctx context.Context, settings SettingsRepository) RefundPolicy {
	policy := RefundPolicy{
		FullDays:           1,
		ProrateDays:        7,
		NoRefundDays:       30,
		FullHours:          0,
		ProrateHours:       0,
		NoRefundHours:      0,
		RequireApproval:    true,
		AutoRefundOnDelete: false,
	}
	if settings == nil {
		return policy
	}
	if v, ok := getSettingInt(ctx, settings, "refund_full_days"); ok {
		policy.FullDays = v
	}
	if v, ok := getSettingInt(ctx, settings, "refund_prorate_days"); ok {
		policy.ProrateDays = v
	}
	if v, ok := getSettingInt(ctx, settings, "refund_no_refund_days"); ok {
		policy.NoRefundDays = v
	}
	if v, ok := getSettingInt(ctx, settings, "refund_full_hours"); ok {
		policy.FullHours = v
	}
	if v, ok := getSettingInt(ctx, settings, "refund_prorate_hours"); ok {
		policy.ProrateHours = v
	}
	if v, ok := getSettingInt(ctx, settings, "refund_no_refund_hours"); ok {
		policy.NoRefundHours = v
	}
	if v, ok := getSettingBool(ctx, settings, "refund_requires_approval"); ok {
		policy.RequireApproval = v
	}
	if v, ok := getSettingBool(ctx, settings, "refund_on_admin_delete"); ok {
		policy.AutoRefundOnDelete = v
	}
	if curve, ok := LoadRefundCurve(ctx, settings); ok {
		policy.Curve = curve
	}
	return policy
}

func calculateRefundAmountForAmount(inst domain.VPSInstance, amount int64, policy RefundPolicy) int64 {
	if amount <= 0 {
		return 0
	}
	now := time.Now()
	if inst.ExpireAt != nil && !inst.ExpireAt.After(now) {
		return 0
	}
	elapsedRatio := refundElapsedRatio(inst, now)
	if len(policy.Curve) > 0 {
		if ratio, ok := RefundCurveRatio(policy.Curve, elapsedRatio*100); ok {
			return int64(math.Round(float64(amount) * ratio))
		}
	}
	totalHours, ok := refundPeriodHours(inst)
	if !ok {
		ageHoursFloat := now.Sub(inst.CreatedAt).Hours()
		ageHours := int(ageHoursFloat)
		ageDays := int(ageHoursFloat / 24)
		if policy.NoRefundHours > 0 && ageHours > policy.NoRefundHours {
			return 0
		}
		if policy.NoRefundHours <= 0 && policy.NoRefundDays > 0 && ageDays > policy.NoRefundDays {
			return 0
		}
		if policy.FullHours > 0 && ageHours <= policy.FullHours {
			return amount
		}
		if policy.FullHours <= 0 && policy.FullDays > 0 && ageDays <= policy.FullDays {
			return amount
		}
		if policy.ProrateHours > 0 && ageHours <= policy.ProrateHours {
			ratio := float64(policy.ProrateHours-ageHours) / float64(policy.ProrateHours)
			return int64(math.Round(float64(amount) * ratio))
		}
		if policy.ProrateHours <= 0 && policy.ProrateDays > 0 && ageDays <= policy.ProrateDays {
			ratio := float64(policy.ProrateDays-ageDays) / float64(policy.ProrateDays)
			return int64(math.Round(float64(amount) * ratio))
		}
		return 0
	}

	fullRatio := refundRatioThreshold(totalHours, policy.FullHours, policy.FullDays)
	prorateRatio := refundRatioThreshold(totalHours, policy.ProrateHours, policy.ProrateDays)
	noRefundRatio := refundRatioThreshold(totalHours, policy.NoRefundHours, policy.NoRefundDays)

	if noRefundRatio > 0 && elapsedRatio > noRefundRatio {
		return 0
	}
	if fullRatio > 0 && elapsedRatio <= fullRatio {
		return amount
	}
	if prorateRatio > 0 && elapsedRatio <= prorateRatio {
		ratio := (prorateRatio - elapsedRatio) / prorateRatio
		return int64(math.Round(float64(amount) * ratio))
	}
	return 0
}
