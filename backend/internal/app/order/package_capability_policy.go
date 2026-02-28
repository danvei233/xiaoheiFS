package order

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
)

const goodsTypeCapabilitiesSettingKey = "goods_type_capabilities_json"

type packageCapabilityPolicy struct {
	ResizeEnabled *bool `json:"resize_enabled,omitempty"`
	RefundEnabled *bool `json:"refund_enabled,omitempty"`
}

func loadCapabilityPolicy(ctx context.Context, repo SettingsRepository, _ int64, goodsTypeID int64) packageCapabilityPolicy {
	if repo == nil {
		return packageCapabilityPolicy{}
	}
	goodsType := loadCapabilityPolicyByKey(ctx, repo, goodsTypeCapabilitiesSettingKey, goodsTypeID)
	return goodsType
}

func loadCapabilityPolicyByKey(ctx context.Context, repo SettingsRepository, key string, itemID int64) packageCapabilityPolicy {
	if itemID <= 0 {
		return packageCapabilityPolicy{}
	}
	setting, err := repo.GetSetting(ctx, key)
	if err != nil {
		return packageCapabilityPolicy{}
	}
	raw := strings.TrimSpace(setting.ValueJSON)
	if raw == "" || raw == "{}" {
		return packageCapabilityPolicy{}
	}
	var all map[string]packageCapabilityPolicy
	if err := json.Unmarshal([]byte(raw), &all); err != nil || all == nil {
		return packageCapabilityPolicy{}
	}
	item, ok := all[strconv.FormatInt(itemID, 10)]
	if !ok {
		return packageCapabilityPolicy{}
	}
	return item
}
