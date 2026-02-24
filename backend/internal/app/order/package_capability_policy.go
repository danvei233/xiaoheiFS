package order

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
)

const packageCapabilitiesSettingKey = "package_capabilities_json"

type packageCapabilityPolicy struct {
	ResizeEnabled *bool `json:"resize_enabled,omitempty"`
	RefundEnabled *bool `json:"refund_enabled,omitempty"`
}

func loadPackageCapabilityPolicy(ctx context.Context, repo SettingsRepository, packageID int64) packageCapabilityPolicy {
	if repo == nil || packageID <= 0 {
		return packageCapabilityPolicy{}
	}
	setting, err := repo.GetSetting(ctx, packageCapabilitiesSettingKey)
	if err != nil {
		return packageCapabilityPolicy{}
	}
	raw := strings.TrimSpace(setting.ValueJSON)
	if raw == "" || raw == "{}" {
		return packageCapabilityPolicy{}
	}
	var all map[string]packageCapabilityPolicy
	if err := json.Unmarshal([]byte(raw), &all); err != nil {
		return packageCapabilityPolicy{}
	}
	item, ok := all[strconv.FormatInt(packageID, 10)]
	if !ok {
		return packageCapabilityPolicy{}
	}
	return item
}
