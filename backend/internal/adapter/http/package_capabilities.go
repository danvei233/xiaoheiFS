package http

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"xiaoheiplay/internal/domain"
)

const packageCapabilitiesSettingKey = "package_capabilities_json"
const goodsTypeCapabilitiesSettingKey = "goods_type_capabilities_json"

type packageCapabilityPolicy struct {
	ResizeEnabled *bool `json:"resize_enabled,omitempty"`
	RefundEnabled *bool `json:"refund_enabled,omitempty"`
}

func (h *Handler) loadAllCapabilityPolicies(ctx context.Context, key string) map[string]packageCapabilityPolicy {
	setting, err := h.getSettingByContext(ctx, key)
	if err != nil {
		return map[string]packageCapabilityPolicy{}
	}
	raw := strings.TrimSpace(setting.ValueJSON)
	if raw == "" || raw == "{}" {
		return map[string]packageCapabilityPolicy{}
	}
	var out map[string]packageCapabilityPolicy
	if err := json.Unmarshal([]byte(raw), &out); err != nil || out == nil {
		return map[string]packageCapabilityPolicy{}
	}
	return out
}

func (h *Handler) loadAllPackageCapabilityPolicies(ctx context.Context) map[string]packageCapabilityPolicy {
	return h.loadAllCapabilityPolicies(ctx, packageCapabilitiesSettingKey)
}

func (h *Handler) loadAllGoodsTypeCapabilityPolicies(ctx context.Context) map[string]packageCapabilityPolicy {
	return h.loadAllCapabilityPolicies(ctx, goodsTypeCapabilitiesSettingKey)
}

func (h *Handler) getPackageCapabilityPolicy(ctx context.Context, packageID int64) packageCapabilityPolicy {
	if packageID <= 0 {
		return packageCapabilityPolicy{}
	}
	all := h.loadAllPackageCapabilityPolicies(ctx)
	item, ok := all[strconv.FormatInt(packageID, 10)]
	if !ok {
		return packageCapabilityPolicy{}
	}
	return item
}

func (h *Handler) getGoodsTypeCapabilityPolicy(ctx context.Context, goodsTypeID int64) packageCapabilityPolicy {
	if goodsTypeID <= 0 {
		return packageCapabilityPolicy{}
	}
	all := h.loadAllGoodsTypeCapabilityPolicies(ctx)
	item, ok := all[strconv.FormatInt(goodsTypeID, 10)]
	if !ok {
		return packageCapabilityPolicy{}
	}
	return item
}

func (h *Handler) saveCapabilityPolicy(c *gin.Context, settingKey string, itemID int64, policy packageCapabilityPolicy) error {
	if h.adminSvc == nil || itemID <= 0 {
		return domain.ErrNotSupported
	}
	all := h.loadAllCapabilityPolicies(c, settingKey)
	key := strconv.FormatInt(itemID, 10)
	if policy.ResizeEnabled == nil && policy.RefundEnabled == nil {
		delete(all, key)
	} else {
		all[key] = policy
	}
	raw, err := json.Marshal(all)
	if err != nil {
		return err
	}
	return h.adminSvc.UpdateSetting(c, getUserID(c), settingKey, string(raw))
}

func (h *Handler) savePackageCapabilityPolicy(c *gin.Context, packageID int64, policy packageCapabilityPolicy) error {
	return h.saveCapabilityPolicy(c, packageCapabilitiesSettingKey, packageID, policy)
}

func (h *Handler) saveGoodsTypeCapabilityPolicy(c *gin.Context, goodsTypeID int64, policy packageCapabilityPolicy) error {
	return h.saveCapabilityPolicy(c, goodsTypeCapabilitiesSettingKey, goodsTypeID, policy)
}

func (h *Handler) capabilityPolicyGoodsTypeID(ctx context.Context, inst domain.VPSInstance) int64 {
	if inst.GoodsTypeID > 0 {
		return inst.GoodsTypeID
	}
	if h.catalogSvc == nil || inst.PackageID <= 0 {
		return 0
	}
	pkg, err := h.catalogSvc.GetPackage(ctx, inst.PackageID)
	if err != nil {
		return 0
	}
	return pkg.GoodsTypeID
}

func (h *Handler) packageFeatureAllowed(c *gin.Context, inst domain.VPSInstance, feature string, fallback bool) bool {
	allowed := fallback
	goodsTypePolicy := h.getGoodsTypeCapabilityPolicy(c, h.capabilityPolicyGoodsTypeID(c, inst))
	switch feature {
	case "resize":
		if goodsTypePolicy.ResizeEnabled != nil {
			allowed = *goodsTypePolicy.ResizeEnabled
		}
	case "refund":
		if goodsTypePolicy.RefundEnabled != nil {
			allowed = *goodsTypePolicy.RefundEnabled
		}
	}
	if auto := h.resolveVPSAutomationCapability(c, inst); auto != nil {
		allowed = featureAllowedByCapability(auto, feature, allowed)
	}
	return allowed
}

func featureAllowedByCapability(cap *VPSAutomationCapabilityDTO, feature string, fallback bool) bool {
	if cap == nil {
		return fallback
	}
	allowed := fallback
	featureSet := make(map[string]struct{}, len(cap.Features))
	for _, raw := range cap.Features {
		v := normalizeFeatureKey(raw)
		if v != "" {
			featureSet[v] = struct{}{}
		}
	}
	if len(featureSet) > 0 {
		_, allowed = featureSet[normalizeFeatureKey(feature)]
	}
	return allowed
}

func (h *Handler) packageCapabilityResolvedValue(c *gin.Context, packageID int64, key string, globalKey string, defaultVal bool) (bool, string) {
	policy := h.getPackageCapabilityPolicy(c, packageID)
	switch key {
	case "resize":
		if policy.ResizeEnabled != nil {
			return *policy.ResizeEnabled, "package"
		}
	case "refund":
		if policy.RefundEnabled != nil {
			return *policy.RefundEnabled, "package"
		}
	}
	if v, ok := h.getSettingBool(c, globalKey); ok {
		return v, "global"
	}
	return defaultVal, "default"
}

func (h *Handler) goodsTypeCapabilityResolvedValue(c *gin.Context, goodsTypeID int64, key string, globalKey string, defaultVal bool) (bool, string) {
	policy := h.getGoodsTypeCapabilityPolicy(c, goodsTypeID)
	switch key {
	case "resize":
		if policy.ResizeEnabled != nil {
			return *policy.ResizeEnabled, "goods_type"
		}
	case "refund":
		if policy.RefundEnabled != nil {
			return *policy.RefundEnabled, "goods_type"
		}
	}
	if v, ok := h.getSettingBool(c, globalKey); ok {
		return v, "global"
	}
	return defaultVal, "default"
}
