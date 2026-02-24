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

type packageCapabilityPolicy struct {
	ResizeEnabled *bool `json:"resize_enabled,omitempty"`
	RefundEnabled *bool `json:"refund_enabled,omitempty"`
}

func (h *Handler) loadAllPackageCapabilityPolicies(ctx context.Context) map[string]packageCapabilityPolicy {
	setting, err := h.getSettingByContext(ctx, packageCapabilitiesSettingKey)
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

func (h *Handler) savePackageCapabilityPolicy(c *gin.Context, packageID int64, policy packageCapabilityPolicy) error {
	if h.adminSvc == nil || packageID <= 0 {
		return domain.ErrNotSupported
	}
	all := h.loadAllPackageCapabilityPolicies(c)
	key := strconv.FormatInt(packageID, 10)
	if policy.ResizeEnabled == nil && policy.RefundEnabled == nil {
		delete(all, key)
	} else {
		all[key] = policy
	}
	raw, err := json.Marshal(all)
	if err != nil {
		return err
	}
	return h.adminSvc.UpdateSetting(c, getUserID(c), packageCapabilitiesSettingKey, string(raw))
}

func (h *Handler) packageFeatureAllowed(c *gin.Context, inst domain.VPSInstance, feature string, fallback bool) bool {
	allowed := fallback
	policy := h.getPackageCapabilityPolicy(c, inst.PackageID)
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
