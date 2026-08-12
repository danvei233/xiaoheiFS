package http

import (
	"context"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"xiaoheiplay/internal/domain"
)

const packageCapabilitiesSettingKey = "package_capabilities_json"
const goodsTypeCapabilitiesSettingKey = "goods_type_capabilities_json"

type packageCapabilityPolicy = domain.CapabilityPolicy

func capabilityPolicyScope(key string) (domain.CapabilityPolicyScope, bool) {
	switch strings.TrimSpace(key) {
	case packageCapabilitiesSettingKey:
		return domain.CapabilityPolicyScopePackage, true
	case goodsTypeCapabilitiesSettingKey:
		return domain.CapabilityPolicyScopeGoodsType, true
	default:
		return "", false
	}
}

func (h *Handler) loadAllCapabilityPolicies(ctx context.Context, key string) map[string]packageCapabilityPolicy {
	if h.adminSvc == nil {
		return map[string]packageCapabilityPolicy{}
	}
	scope, ok := capabilityPolicyScope(key)
	if !ok {
		return map[string]packageCapabilityPolicy{}
	}
	items, err := h.adminSvc.ListCapabilityPolicies(ctx, scope)
	if err != nil {
		return map[string]packageCapabilityPolicy{}
	}
	out := make(map[string]packageCapabilityPolicy, len(items))
	for id, policy := range items {
		out[strconv.FormatInt(id, 10)] = policy
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
	return h.getCapabilityPolicy(ctx, domain.CapabilityPolicyScopePackage, packageID)
}

func (h *Handler) getCapabilityPolicy(ctx context.Context, scope domain.CapabilityPolicyScope, itemID int64) packageCapabilityPolicy {
	if h.adminSvc == nil || itemID <= 0 {
		return packageCapabilityPolicy{}
	}
	item, err := h.adminSvc.GetCapabilityPolicy(ctx, scope, itemID)
	if err != nil {
		return packageCapabilityPolicy{}
	}
	return item
}

func (h *Handler) getGoodsTypeCapabilityPolicy(ctx context.Context, goodsTypeID int64) packageCapabilityPolicy {
	if goodsTypeID <= 0 {
		return packageCapabilityPolicy{}
	}
	return h.getCapabilityPolicy(ctx, domain.CapabilityPolicyScopeGoodsType, goodsTypeID)
}

func (h *Handler) saveCapabilityPolicy(c *gin.Context, settingKey string, itemID int64, policy packageCapabilityPolicy) error {
	if h.adminSvc == nil || itemID <= 0 {
		return domain.ErrNotSupported
	}
	scope, ok := capabilityPolicyScope(settingKey)
	if !ok {
		return domain.ErrInvalidInput
	}
	return h.adminSvc.UpsertCapabilityPolicy(c.Request.Context(), scope, itemID, policy)
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
	policy := h.effectiveCapabilityPolicy(c.Request.Context(), inst)
	switch feature {
	case "resize":
		if policy.ResizeEnabled != nil {
			allowed = *policy.ResizeEnabled
		}
	case "refund":
		if policy.RefundEnabled != nil {
			allowed = *policy.RefundEnabled
		}
	}
	if !allowed {
		return false
	}
	auto := h.resolveVPSAutomationCapability(c, inst)
	if feature == "resize" || feature == "refund" {
		// Strict AND semantics for resize/refund:
		// 1) package/goods-type/site policy must allow (checked above)
		// 2) plugin capability features must explicitly contain the feature
		return featureAllowedByCapability(auto, feature, false)
	}
	if auto != nil {
		allowed = featureAllowedByCapability(auto, feature, allowed)
	}
	return allowed
}

func (h *Handler) effectiveCapabilityPolicy(ctx context.Context, inst domain.VPSInstance) packageCapabilityPolicy {
	goodsType := h.getGoodsTypeCapabilityPolicy(ctx, h.capabilityPolicyGoodsTypeID(ctx, inst))
	pkg := h.getPackageCapabilityPolicy(ctx, inst.PackageID)
	return mergeHTTPPolicies(pkg, goodsType)
}

func mergeHTTPPolicies(specific, fallback packageCapabilityPolicy) packageCapabilityPolicy {
	merged := fallback
	if specific.ResizeEnabled != nil {
		merged.ResizeEnabled = specific.ResizeEnabled
	}
	if specific.RefundEnabled != nil {
		merged.RefundEnabled = specific.RefundEnabled
	}
	return merged
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
	ctx := c.Request.Context()
	pkgPolicy := h.getPackageCapabilityPolicy(ctx, packageID)
	goodsTypePolicy := packageCapabilityPolicy{}
	if h.catalogSvc != nil {
		if pkg, err := h.catalogSvc.GetPackage(ctx, packageID); err == nil {
			goodsTypePolicy = h.getGoodsTypeCapabilityPolicy(ctx, pkg.GoodsTypeID)
		}
	}
	switch key {
	case "resize":
		if pkgPolicy.ResizeEnabled != nil {
			return *pkgPolicy.ResizeEnabled, "package"
		}
		if goodsTypePolicy.ResizeEnabled != nil {
			return *goodsTypePolicy.ResizeEnabled, "goods_type"
		}
	case "refund":
		if pkgPolicy.RefundEnabled != nil {
			return *pkgPolicy.RefundEnabled, "package"
		}
		if goodsTypePolicy.RefundEnabled != nil {
			return *goodsTypePolicy.RefundEnabled, "goods_type"
		}
	}
	if v, ok := h.getSettingBool(c, globalKey); ok {
		return v, "global"
	}
	return defaultVal, "default"
}

func (h *Handler) goodsTypeCapabilityResolvedValue(c *gin.Context, goodsTypeID int64, key string, globalKey string, defaultVal bool) (bool, string) {
	policy := h.getGoodsTypeCapabilityPolicy(c.Request.Context(), goodsTypeID)
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
