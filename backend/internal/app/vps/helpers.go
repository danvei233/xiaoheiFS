package vps

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"

	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

const (
	maxLenPassword        = 128
	maxLenPortMappingName = 64
)

var vpsFieldValidator = validator.New()

type emergencyRenewPolicy struct {
	Enabled       bool
	WindowDays    int
	RenewDays     int
	IntervalHours int
}

func loadEmergencyRenewPolicy(ctx context.Context, settings appports.SettingsRepository) emergencyRenewPolicy {
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

func getSettingInt(ctx context.Context, repo appports.SettingsRepository, key string) (int, bool) {
	if repo == nil {
		return 0, false
	}
	setting, err := repo.GetSetting(ctx, key)
	if err != nil {
		return 0, false
	}
	val, err := strconv.Atoi(strings.TrimSpace(setting.ValueJSON))
	if err != nil {
		return 0, false
	}
	return val, true
}

func getSettingBool(ctx context.Context, repo appports.SettingsRepository, key string) (bool, bool) {
	if repo == nil {
		return false, false
	}
	setting, err := repo.GetSetting(ctx, key)
	if err != nil {
		return false, false
	}
	raw := strings.TrimSpace(setting.ValueJSON)
	if raw == "" {
		return false, false
	}
	switch strings.ToLower(raw) {
	case "true", "1", "yes":
		return true, true
	case "false", "0", "no":
		return false, true
	default:
		return false, false
	}
}

func parseHostID(v string) int64 {
	id, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	return id
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func trimAndValidateRequired(value string, maxLen int) (string, error) {
	trimmed := strings.TrimSpace(value)
	if err := vpsFieldValidator.Var(trimmed, fmt.Sprintf("required,max=%d", maxLen)); err != nil {
		return "", appshared.ErrInvalidInput
	}
	return trimmed, nil
}

func trimAndValidateOptional(value string, maxLen int) (string, error) {
	trimmed := strings.TrimSpace(value)
	if err := vpsFieldValidator.Var(trimmed, fmt.Sprintf("omitempty,max=%d", maxLen)); err != nil {
		return "", appshared.ErrInvalidInput
	}
	return trimmed, nil
}

func MapAutomationState(state int) domain.VPSStatus {
	switch state {
	case 0, 1, 13:
		return domain.VPSStatusProvisioning
	case 2:
		return domain.VPSStatusRunning
	case 3:
		return domain.VPSStatusStopped
	case 4:
		return domain.VPSStatusReinstalling
	case 5:
		return domain.VPSStatusReinstallFailed
	case 10:
		return domain.VPSStatusLocked
	default:
		return domain.VPSStatusUnknown
	}
}

func mergeAccessInfo(existing string, info AutomationHostInfo) string {
	osPwd := ""
	remoteIP := ""
	panelPwd := ""
	vncPwd := ""
	if existing != "" {
		var current map[string]any
		if err := json.Unmarshal([]byte(existing), &current); err == nil {
			if v, ok := current["os_password"]; ok {
				osPwd = fmt.Sprintf("%v", v)
			}
			if v, ok := current["remote_ip"]; ok {
				remoteIP = fmt.Sprintf("%v", v)
			}
			if v, ok := current["panel_password"]; ok {
				panelPwd = fmt.Sprintf("%v", v)
			}
			if v, ok := current["vnc_password"]; ok {
				vncPwd = fmt.Sprintf("%v", v)
			}
		}
	}
	if info.RemoteIP != "" {
		remoteIP = info.RemoteIP
	}
	if info.PanelPassword != "" {
		panelPwd = info.PanelPassword
	}
	if info.VNCPassword != "" {
		vncPwd = info.VNCPassword
	}
	if info.OSPassword != "" {
		osPwd = info.OSPassword
	}
	payload := map[string]any{
		"remote_ip":      remoteIP,
		"panel_password": panelPwd,
		"vnc_password":   vncPwd,
	}
	if osPwd != "" {
		payload["os_password"] = osPwd
	}
	return mustJSON(payload)
}
