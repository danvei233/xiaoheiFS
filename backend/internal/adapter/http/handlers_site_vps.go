package http

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

func (h *Handler) SiteSettings(c *gin.Context) {
	if h.settingsSvc == nil && h.adminSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	allowed := map[string]bool{
		"site_name":                true,
		"site_url":                 true,
		"logo_url":                 true,
		"favicon_url":              true,
		"site_description":         true,
		"site_keywords":            true,
		"company_name":             true,
		"contact_phone":            true,
		"contact_email":            true,
		"contact_qq":               true,
		"wechat_qrcode":            true,
		"icp_number":               true,
		"psbe_number":              true,
		"maintenance_mode":         true,
		"maintenance_message":      true,
		"analytics_code":           true,
		"site_nav_items":           true,
		"site_logo":                true,
		"site_icp":                 true,
		"site_maintenance_mode":    true,
		"site_maintenance_message": true,
	}
	aliases := map[string]string{
		"site_logo":                "logo_url",
		"site_icp":                 "icp_number",
		"site_maintenance_mode":    "maintenance_mode",
		"site_maintenance_message": "maintenance_message",
	}
	items, err := h.listSettings(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrListError.Error()})
		return
	}
	filtered := make([]domain.Setting, 0)
	indexed := make(map[string]domain.Setting)
	for _, item := range items {
		if allowed[item.Key] {
			filtered = append(filtered, item)
			indexed[item.Key] = item
		}
	}
	for legacy, current := range aliases {
		if _, ok := indexed[current]; ok {
			continue
		}
		if legacyItem, ok := indexed[legacy]; ok {
			filtered = append(filtered, domain.Setting{Key: current, ValueJSON: legacyItem.ValueJSON})
		}
	}
	c.JSON(http.StatusOK, gin.H{"items": toSettingDTOs(filtered)})
}

func (h *Handler) toVPSInstanceDTOWithLifecycle(c *gin.Context, inst domain.VPSInstance) VPSInstanceDTO {
	dto := toVPSInstanceDTO(inst)
	destroyAt, destroyInDays := h.lifecycleDestroyInfo(c, inst.ExpireAt)
	dto.DestroyAt = destroyAt
	dto.DestroyInDays = destroyInDays
	dto.Capabilities = h.resolveVPSCapabilities(c, inst)
	return dto
}

func (h *Handler) toVPSInstanceDTOsWithLifecycle(c *gin.Context, items []domain.VPSInstance) []VPSInstanceDTO {
	out := make([]VPSInstanceDTO, 0, len(items))
	for _, item := range items {
		out = append(out, h.toVPSInstanceDTOWithLifecycle(c, item))
	}
	return out
}

func (h *Handler) lifecycleDestroyInfo(c *gin.Context, expireAt *time.Time) (*time.Time, *int) {
	if expireAt == nil || (h.settingsSvc == nil && h.adminSvc == nil) {
		return nil, nil
	}
	enabled, ok := h.getSettingBool(c, "auto_delete_enabled")
	if !ok || !enabled {
		return nil, nil
	}
	days, ok := h.getSettingInt(c, "auto_delete_days")
	if !ok {
		days = 0
	}
	if days < 0 {
		days = 0
	}
	destroyAt := expireAt.Add(time.Duration(days) * 24 * time.Hour)
	inDays := int(math.Ceil(destroyAt.Sub(time.Now()).Hours() / 24))
	return &destroyAt, &inDays
}

func (h *Handler) resolveVPSCapabilities(c *gin.Context, inst domain.VPSInstance) *VPSCapabilitiesDTO {
	auto := h.resolveVPSAutomationCapability(c, inst)
	if auto == nil {
		return nil
	}
	return &VPSCapabilitiesDTO{Automation: auto}
}

func (h *Handler) resolveVPSAutomationCapability(c *gin.Context, inst domain.VPSInstance) *VPSAutomationCapabilityDTO {
	staticCap := h.resolveVPSAutomationCapabilityStatic(c, inst)
	dynamicCap := parseDynamicAutomationCapability(inst.AccessInfoJSON)
	merged := mergeAutomationCapabilities(staticCap, dynamicCap)
	return h.applyPackageCapabilityPolicy(c, inst, merged)
}

func toVPSAutomationCapabilityDTO(cap *appshared.PluginAutomationCapability) *VPSAutomationCapabilityDTO {
	if cap == nil {
		return nil
	}
	features := make([]string, 0, len(cap.Features))
	seen := make(map[string]struct{}, len(cap.Features))
	for _, raw := range cap.Features {
		v := strings.ToLower(strings.TrimSpace(raw))
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		features = append(features, v)
	}
	reasons := make(map[string]string, len(cap.NotSupportedReasons))
	for k, v := range cap.NotSupportedReasons {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		reasons[key] = v
	}
	out := &VPSAutomationCapabilityDTO{Features: features}
	if len(reasons) > 0 {
		out.NotSupportedReasons = reasons
	}
	return out
}

type dynamicAutomationCapability struct {
	HasFeatures         bool
	Features            []string
	AddFeatures         []string
	RemoveFeatures      []string
	DisabledFeatures    []string
	DenyFeatures        []string
	NotSupportedReasons map[string]string
}

func (h *Handler) resolveVPSAutomationCapabilityStatic(c *gin.Context, inst domain.VPSInstance) *VPSAutomationCapabilityDTO {
	if h.goodsTypes == nil || h.pluginAdmin == nil || inst.GoodsTypeID <= 0 {
		return nil
	}
	gt, err := h.goodsTypes.Get(c, inst.GoodsTypeID)
	if err != nil {
		return nil
	}
	category := strings.ToLower(strings.TrimSpace(gt.AutomationCategory))
	pluginID := strings.TrimSpace(gt.AutomationPluginID)
	instanceID := strings.TrimSpace(gt.AutomationInstanceID)
	if category == "" {
		category = "automation"
	}
	if category != "automation" || pluginID == "" || instanceID == "" {
		return nil
	}

	items, err := h.pluginAdmin.List(c)
	if err != nil {
		return nil
	}
	for _, item := range items {
		if strings.ToLower(strings.TrimSpace(item.Category)) != category {
			continue
		}
		if strings.TrimSpace(item.PluginID) != pluginID || strings.TrimSpace(item.InstanceID) != instanceID {
			continue
		}
		return toVPSAutomationCapabilityDTO(item.Capabilities.Capabilities.Automation)
	}
	return nil
}

func parseDynamicAutomationCapability(accessInfoJSON string) *dynamicAutomationCapability {
	raw := strings.TrimSpace(accessInfoJSON)
	if raw == "" || raw == "{}" {
		return nil
	}
	var envelope struct {
		Capabilities struct {
			Automation json.RawMessage `json:"automation"`
		} `json:"capabilities"`
	}
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil || len(envelope.Capabilities.Automation) == 0 {
		return nil
	}

	var payloadMap map[string]json.RawMessage
	if err := json.Unmarshal(envelope.Capabilities.Automation, &payloadMap); err != nil {
		return nil
	}

	var payload struct {
		Features            []string          `json:"features"`
		AddFeatures         []string          `json:"add_features"`
		RemoveFeatures      []string          `json:"remove_features"`
		DisabledFeatures    []string          `json:"disabled_features"`
		DenyFeatures        []string          `json:"deny_features"`
		NotSupportedReasons map[string]string `json:"not_supported_reasons"`
	}
	if err := json.Unmarshal(envelope.Capabilities.Automation, &payload); err != nil {
		return nil
	}

	out := &dynamicAutomationCapability{
		Features:            normalizeFeatureList(payload.Features),
		AddFeatures:         normalizeFeatureList(payload.AddFeatures),
		RemoveFeatures:      normalizeFeatureList(payload.RemoveFeatures),
		DisabledFeatures:    normalizeFeatureList(payload.DisabledFeatures),
		DenyFeatures:        normalizeFeatureList(payload.DenyFeatures),
		NotSupportedReasons: normalizeReasons(payload.NotSupportedReasons),
	}
	_, out.HasFeatures = payloadMap["features"]
	if !out.HasFeatures && len(out.AddFeatures) == 0 && len(out.RemoveFeatures) == 0 && len(out.DisabledFeatures) == 0 && len(out.DenyFeatures) == 0 && len(out.NotSupportedReasons) == 0 {
		return nil
	}
	return out
}

func mergeAutomationCapabilities(staticCap *VPSAutomationCapabilityDTO, dynamicCap *dynamicAutomationCapability) *VPSAutomationCapabilityDTO {
	if staticCap == nil && dynamicCap == nil {
		return nil
	}

	features := make(map[string]struct{}, 8)
	reasons := make(map[string]string, 8)

	if staticCap != nil {
		for _, f := range normalizeFeatureList(staticCap.Features) {
			features[f] = struct{}{}
		}
		for k, v := range normalizeReasons(staticCap.NotSupportedReasons) {
			reasons[k] = v
		}
	}

	if dynamicCap != nil {
		if dynamicCap.HasFeatures {
			features = make(map[string]struct{}, len(dynamicCap.Features))
			for _, f := range dynamicCap.Features {
				features[f] = struct{}{}
			}
		}
		for _, f := range dynamicCap.AddFeatures {
			features[f] = struct{}{}
		}
		removeAll := append([]string{}, dynamicCap.RemoveFeatures...)
		removeAll = append(removeAll, dynamicCap.DisabledFeatures...)
		removeAll = append(removeAll, dynamicCap.DenyFeatures...)
		for _, f := range removeAll {
			delete(features, f)
		}
		for k, v := range dynamicCap.NotSupportedReasons {
			reasons[k] = v
		}
	}

	outFeatures := make([]string, 0, len(features))
	for f := range features {
		outFeatures = append(outFeatures, f)
	}
	sort.Strings(outFeatures)

	out := &VPSAutomationCapabilityDTO{Features: outFeatures}
	if len(reasons) > 0 {
		out.NotSupportedReasons = reasons
	}
	return out
}

func normalizeFeatureList(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, raw := range items {
		v := strings.ToLower(strings.TrimSpace(raw))
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func normalizeReasons(items map[string]string) map[string]string {
	if len(items) == 0 {
		return nil
	}
	out := make(map[string]string, len(items))
	for k, v := range items {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		out[key] = strings.TrimSpace(v)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (h *Handler) applyPackageCapabilityPolicy(c *gin.Context, inst domain.VPSInstance, cap *VPSAutomationCapabilityDTO) *VPSAutomationCapabilityDTO {
	if cap == nil {
		cap = &VPSAutomationCapabilityDTO{Features: []string{}}
	}
	featureSet := map[string]struct{}{}
	for _, f := range normalizeFeatureList(cap.Features) {
		featureSet[normalizeFeatureKey(f)] = struct{}{}
	}
	reasons := normalizeReasons(cap.NotSupportedReasons)
	if reasons == nil {
		reasons = map[string]string{}
	}

	policy := h.getGoodsTypeCapabilityPolicy(c, h.capabilityPolicyGoodsTypeID(c, inst))

	resizeAllowed := true
	if v, ok := h.getSettingBool(c, "resize_enabled"); ok {
		resizeAllowed = v
	}
	if policy.ResizeEnabled != nil {
		resizeAllowed = *policy.ResizeEnabled
	}
	if resizeAllowed {
		if _, ok := featureSet["resize"]; !ok && strings.TrimSpace(reasons["resize"]) == "" {
			reasons["resize"] = "插件未声明支持升降配"
		}
	} else {
		delete(featureSet, "resize")
		if strings.TrimSpace(reasons["resize"]) == "" {
			if policy.ResizeEnabled != nil {
				reasons["resize"] = "商品类型未开启升降配"
			} else {
				reasons["resize"] = "站点未开启升降配"
			}
		}
	}

	refundAllowed := true
	if v, ok := h.getSettingBool(c, "refund_enabled"); ok {
		refundAllowed = v
	}
	if policy.RefundEnabled != nil {
		refundAllowed = *policy.RefundEnabled
	}
	if refundAllowed {
		if _, ok := featureSet["refund"]; !ok && strings.TrimSpace(reasons["refund"]) == "" {
			reasons["refund"] = "插件未声明支持退款"
		}
	} else {
		delete(featureSet, "refund")
		if strings.TrimSpace(reasons["refund"]) == "" {
			if policy.RefundEnabled != nil {
				reasons["refund"] = "商品类型未开启退款"
			} else {
				reasons["refund"] = "站点未开启退款"
			}
		}
	}
	outFeatures := make([]string, 0, len(featureSet))
	for f := range featureSet {
		outFeatures = append(outFeatures, f)
	}
	sort.Strings(outFeatures)
	out := &VPSAutomationCapabilityDTO{Features: outFeatures}
	if len(reasons) > 0 {
		out.NotSupportedReasons = reasons
	}
	return out
}

func normalizeFeatureKey(value string) string {
	v := strings.ToLower(strings.TrimSpace(value))
	switch v {
	case "upgrade", "downgrade":
		return "resize"
	case "refund_request":
		return "refund"
	default:
		return v
	}
}

func (h *Handler) getSettingInt(c *gin.Context, key string) (int, bool) {
	if h.settingsSvc == nil && h.adminSvc == nil {
		return 0, false
	}
	setting, err := h.getSetting(c, key)
	if err != nil {
		return 0, false
	}
	val, err := strconv.Atoi(strings.TrimSpace(setting.ValueJSON))
	if err != nil {
		return 0, false
	}
	return val, true
}

func (h *Handler) getSettingBool(c *gin.Context, key string) (bool, bool) {
	if h.settingsSvc == nil && h.adminSvc == nil {
		return false, false
	}
	setting, err := h.getSetting(c, key)
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
