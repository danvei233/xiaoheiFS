package order

import (
	"context"
	"math"
	"time"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/pkg/money"
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
	// The elapsed ratio must be measured against the current billing period, not
	// the instance creation time: renewals extend ExpireAt without touching
	// CreatedAt, so a lifetime-based ratio would systematically under-refund
	// every renewed instance. currentPeriod prefers the current_period_start/end
	// written into SpecJSON on renewal and falls back to CreatedAt/ExpireAt.
	periodStart, periodEnd, hasPeriod := currentPeriod(inst, now)
	var elapsedRatio float64
	if hasPeriod {
		total := periodEnd.Sub(periodStart)
		if total <= 0 {
			hasPeriod = false
		} else {
			elapsed := now.Sub(periodStart)
			switch {
			case elapsed <= 0:
				elapsedRatio = 0
			case elapsed >= total:
				elapsedRatio = 1
			default:
				elapsedRatio = float64(elapsed) / float64(total)
			}
		}
	}
	if !hasPeriod {
		elapsedRatio = refundElapsedRatio(inst, now)
	}
	if len(policy.Curve) > 0 {
		if ratio, ok := RefundCurveRatio(policy.Curve, elapsedRatio*100); ok {
			return int64(math.Round(float64(amount) * ratio))
		}
	}
	if !hasPeriod {
		// No usable billing period (ExpireAt nil or CreatedAt zero): fall back to
		// refund windows measured from the instance creation time.
		return refundAmountByAge(amount, now.Sub(inst.CreatedAt).Hours(), policy)
	}

	// Integer-exact proration against the current billing period, consistent
	// with money.ProrateCents used by resize pricing.
	fullNanos := refundThresholdNanos(policy.FullHours, policy.FullDays)
	prorateNanos := refundThresholdNanos(policy.ProrateHours, policy.ProrateDays)
	noRefundNanos := refundThresholdNanos(policy.NoRefundHours, policy.NoRefundDays)
	elapsedNanos := int64(now.Sub(periodStart))
	if elapsedNanos < 0 {
		elapsedNanos = 0
	}
	if noRefundNanos > 0 && elapsedNanos > noRefundNanos {
		return 0
	}
	if fullNanos > 0 && elapsedNanos <= fullNanos {
		return amount
	}
	if prorateNanos > 0 && elapsedNanos <= prorateNanos {
		return money.ProrateCents(amount, prorateNanos-elapsedNanos, prorateNanos)
	}
	return 0
}

// refundThresholdNanos converts a refund policy window expressed in hours and/or
// days into nanoseconds. Hours win when both are set, matching refundRatioThreshold.
func refundThresholdNanos(hours int, days int) int64 {
	h := int64(hours)
	if h <= 0 && days > 0 {
		h = int64(days) * 24
	}
	if h <= 0 {
		return 0
	}
	return h * int64(time.Hour)
}

// refundAmountByAge applies the day/hour refund windows against the instance age
// (in hours) for legacy instances that carry no usable billing period.
func refundAmountByAge(amount int64, ageHours float64, policy RefundPolicy) int64 {
	ageHoursInt := int(ageHours)
	ageDays := int(ageHours / 24)
	if policy.NoRefundHours > 0 && ageHoursInt > policy.NoRefundHours {
		return 0
	}
	if policy.NoRefundHours <= 0 && policy.NoRefundDays > 0 && ageDays > policy.NoRefundDays {
		return 0
	}
	if policy.FullHours > 0 && ageHoursInt <= policy.FullHours {
		return amount
	}
	if policy.FullHours <= 0 && policy.FullDays > 0 && ageDays <= policy.FullDays {
		return amount
	}
	if policy.ProrateHours > 0 && ageHoursInt <= policy.ProrateHours {
		return money.ProrateCents(amount, int64(policy.ProrateHours-ageHoursInt), int64(policy.ProrateHours))
	}
	if policy.ProrateHours <= 0 && policy.ProrateDays > 0 && ageDays <= policy.ProrateDays {
		return money.ProrateCents(amount, int64(policy.ProrateDays-ageDays), int64(policy.ProrateDays))
	}
	return 0
}
