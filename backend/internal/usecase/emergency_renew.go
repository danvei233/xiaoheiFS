package usecase

import (
	"context"
	"time"
)

type emergencyRenewPolicy struct {
	Enabled       bool
	WindowDays    int
	RenewDays     int
	IntervalHours int
}

func loadEmergencyRenewPolicy(ctx context.Context, settings SettingsRepository) emergencyRenewPolicy {
	policy := emergencyRenewPolicy{
		Enabled:       true,
		WindowDays:    7,
		RenewDays:     1,
		IntervalHours: 720,
	}
	if v, ok := getSettingBool(ctx, settings, "emergency_renew_enabled"); ok {
		policy.Enabled = v
	}
	if v, ok := getSettingInt(ctx, settings, "emergency_renew_window_days"); ok {
		policy.WindowDays = v
	}
	if v, ok := getSettingInt(ctx, settings, "emergency_renew_days"); ok {
		policy.RenewDays = v
	}
	if v, ok := getSettingInt(ctx, settings, "emergency_renew_interval_hours"); ok {
		policy.IntervalHours = v
	}
	if policy.WindowDays < 0 {
		policy.WindowDays = 0
	}
	if policy.RenewDays <= 0 {
		policy.RenewDays = 1
	}
	if policy.IntervalHours <= 0 {
		policy.IntervalHours = 24
	}
	return policy
}

func emergencyRenewInWindow(now time.Time, expireAt *time.Time, windowDays int) bool {
	if expireAt == nil {
		return false
	}
	if now.After(*expireAt) {
		return false
	}
	if windowDays <= 0 {
		return true
	}
	windowStart := expireAt.Add(-time.Duration(windowDays) * 24 * time.Hour)
	return !now.Before(windowStart)
}
