package order

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

const (
	packageCapabilitiesSettingKey   = "package_capabilities_json"
	goodsTypeCapabilitiesSettingKey = "goods_type_capabilities_json"
)

type packageCapabilityPolicy = domain.CapabilityPolicy

func loadCapabilityPolicy(
	ctx context.Context,
	policies appports.CapabilityPolicyRepository,
	settings SettingsRepository,
	packageID int64,
	goodsTypeID int64,
) (packageCapabilityPolicy, error) {
	if policies == nil {
		goodsType := loadCapabilityPolicyByKey(ctx, settings, goodsTypeCapabilitiesSettingKey, goodsTypeID)
		pkg := loadCapabilityPolicyByKey(ctx, settings, packageCapabilitiesSettingKey, packageID)
		return mergeCapabilityPolicies(pkg, goodsType), nil
	}
	goodsType, err := getCapabilityPolicyOrEmpty(ctx, policies, domain.CapabilityPolicyScopeGoodsType, goodsTypeID)
	if err != nil {
		return packageCapabilityPolicy{}, err
	}
	pkg, err := getCapabilityPolicyOrEmpty(ctx, policies, domain.CapabilityPolicyScopePackage, packageID)
	if err != nil {
		return packageCapabilityPolicy{}, err
	}
	return mergeCapabilityPolicies(pkg, goodsType), nil
}

func getCapabilityPolicyOrEmpty(ctx context.Context, repo appports.CapabilityPolicyRepository, scope domain.CapabilityPolicyScope, itemID int64) (packageCapabilityPolicy, error) {
	if itemID <= 0 {
		return packageCapabilityPolicy{}, nil
	}
	policy, err := repo.GetCapabilityPolicy(ctx, scope, itemID)
	if errors.Is(err, appshared.ErrNotFound) {
		return packageCapabilityPolicy{}, nil
	}
	return policy, err
}

func mergeCapabilityPolicies(specific, fallback packageCapabilityPolicy) packageCapabilityPolicy {
	merged := fallback
	if specific.ResizeEnabled != nil {
		merged.ResizeEnabled = specific.ResizeEnabled
	}
	if specific.RefundEnabled != nil {
		merged.RefundEnabled = specific.RefundEnabled
	}
	return merged
}

func loadCapabilityPolicyByKey(ctx context.Context, repo SettingsRepository, key string, itemID int64) packageCapabilityPolicy {
	if repo == nil || itemID <= 0 {
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
	return all[strconv.FormatInt(itemID, 10)]
}
