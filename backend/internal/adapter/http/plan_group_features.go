package http

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"xiaoheiplay/internal/domain"
)

// 线路（PlanGroup）级「插件功能开关」。
//
// 每个功能有三种运行模式：
//
//	"off"   关闭：能力列表移除，前端整块 tab 不显示，后端拒绝所有请求
//	"read"  只读：tab 可见可查看，但前端隐藏创建/修改/删除入口，后端拒绝写操作
//	"write" 完全可用
//
// 存储方式沿用 settings 表的 key-value，值形如：
//
//	{"12": {"nic": "read", "pci": "off"}}
//
// 只持久化「被显式改过默认值」的键；缺省表示跟随 automationFeatureCatalog 里的默认策略。
// 兼容旧数据：布尔值 true 视为 "write"，false 视为 "off"。
//
// 之所以用 settings 而不是给 plan_groups 加列：
//  1. 开关是运营侧灰度配置，与商品目录结构解耦，无需 schema 迁移；
//  2. 与既有的 goods_type / package 能力策略（package_capabilities_json）保持一致；
//  3. 修改后对存量实例立即生效，不需要重新下单或重建能力快照。
const planGroupFeatureFlagsSettingKey = "plan_group_feature_flags_json"

const (
	automationFeatureModeOff   = "off"
	automationFeatureModeRead  = "read"
	automationFeatureModeWrite = "write"
)

// automationFeatureSpec 单个可开关功能的元信息。
type automationFeatureSpec struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Hint  string `json:"hint"`
	// Default 未显式配置时的默认模式。
	Default string `json:"default"`
	// AllowReadonly 为 false 时该功能只支持 off/write，不允许配置为只读。
	AllowReadonly bool `json:"allow_readonly"`
}

// automationFeatureCatalog 可开关功能清单，顺序即前端展示顺序。
//
// 默认策略约定：
//   - pci / usb：默认关闭，需要运营显式开启才出现（直通涉及宿主机硬件，风险高）；
//   - nic / disk / iso / snapshot：默认只读，用户可查看现状，但不能自行增删改；
//   - 其余功能保持完全可用，与历史行为一致。
var automationFeatureCatalog = []automationFeatureSpec{
	{Key: "firewall", Label: "防火墙", Hint: "虚拟机防火墙规则的查看与增删改", Default: automationFeatureModeWrite, AllowReadonly: true},
	{Key: "port_mapping", Label: "端口映射", Hint: "NAT 端口转发的查看与增删", Default: automationFeatureModeWrite, AllowReadonly: true},
	{Key: "snapshot", Label: "快照", Hint: "平台原生快照；只读模式下仅可查看与回滚入口关闭", Default: automationFeatureModeRead, AllowReadonly: true},
	{Key: "backup", Label: "备份", Hint: "备份与还原", Default: automationFeatureModeWrite, AllowReadonly: true},
	{Key: "nic", Label: "网卡管理", Hint: "网卡的查看与增删改", Default: automationFeatureModeRead, AllowReadonly: true},
	{Key: "disk", Label: "数据盘", Hint: "数据盘的查看与挂载/卸载", Default: automationFeatureModeRead, AllowReadonly: true},
	{Key: "iso", Label: "光盘管理", Hint: "ISO 的查看与挂载/卸载", Default: automationFeatureModeRead, AllowReadonly: true},
	{Key: "reverse_proxy", Label: "反向代理", Hint: "域名反向代理的查看与增删改", Default: automationFeatureModeWrite, AllowReadonly: true},
	{Key: "boot_order", Label: "启动顺序", Hint: "启动项顺序的查看与调整", Default: automationFeatureModeWrite, AllowReadonly: true},
	{Key: "pci", Label: "PCI 直通", Hint: "PCI/GPU 直通，默认关闭", Default: automationFeatureModeOff, AllowReadonly: true},
	{Key: "usb", Label: "USB 直通", Hint: "USB 设备直通，默认关闭", Default: automationFeatureModeOff, AllowReadonly: true},
}

// automationFeatureSpecs 便于按 key 查询默认策略等元信息。
var automationFeatureSpecs = func() map[string]automationFeatureSpec {
	out := make(map[string]automationFeatureSpec, len(automationFeatureCatalog))
	for _, item := range automationFeatureCatalog {
		out[item.Key] = item
	}
	return out
}()

// automationFeatureKeys 返回全部可开关功能键，用于校验入参。
func automationFeatureKeys() map[string]struct{} {
	out := make(map[string]struct{}, len(automationFeatureCatalog))
	for _, item := range automationFeatureCatalog {
		out[item.Key] = struct{}{}
	}
	return out
}

// normalizeAutomationFeatureMode 归一化模式取值；非法值返回空串。
func normalizeAutomationFeatureMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case automationFeatureModeOff, "none", "disabled", "hide":
		return automationFeatureModeOff
	case automationFeatureModeRead, "readonly", "ro":
		return automationFeatureModeRead
	case automationFeatureModeWrite, "rw", "on", "full":
		return automationFeatureModeWrite
	default:
		return ""
	}
}

// featureFlagMap 兼容新旧两种 JSON 取值：布尔（历史数据）与字符串（新模式）。
type featureFlagMap map[string]string

func (m *featureFlagMap) UnmarshalJSON(data []byte) error {
	raw := map[string]json.RawMessage{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	out := make(featureFlagMap, len(raw))
	for key, value := range raw {
		k := strings.ToLower(strings.TrimSpace(key))
		if k == "" {
			continue
		}
		var text string
		if err := json.Unmarshal(value, &text); err == nil {
			if mode := normalizeAutomationFeatureMode(text); mode != "" {
				out[k] = mode
			}
			continue
		}
		var enabled bool
		if err := json.Unmarshal(value, &enabled); err == nil {
			// true → 完全可用（历史语义）；false → 关闭
			if enabled {
				out[k] = automationFeatureModeWrite
			} else {
				out[k] = automationFeatureModeOff
			}
		}
	}
	*m = out
	return nil
}

// loadAllPlanGroupFeatureFlags 读取全部线路的功能开关。
func (h *Handler) loadAllPlanGroupFeatureFlags(ctx context.Context) map[string]featureFlagMap {
	out := map[string]featureFlagMap{}
	setting, err := h.getSettingByContext(ctx, planGroupFeatureFlagsSettingKey)
	if err != nil {
		return out
	}
	raw := strings.TrimSpace(setting.ValueJSON)
	if raw == "" || raw == "{}" {
		return out
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil || out == nil {
		return map[string]featureFlagMap{}
	}
	return out
}

// planGroupFeatureFlags 读取单个线路的功能开关。
func (h *Handler) planGroupFeatureFlags(ctx context.Context, planGroupID int64) featureFlagMap {
	if planGroupID <= 0 {
		return nil
	}
	all := h.loadAllPlanGroupFeatureFlags(ctx)
	flags, ok := all[strconv.FormatInt(planGroupID, 10)]
	if !ok || len(flags) == 0 {
		return nil
	}
	return flags
}

// savePlanGroupFeatureFlags 保存单个线路的功能开关；flags 为空时删除该线路配置。
func (h *Handler) savePlanGroupFeatureFlags(c *gin.Context, planGroupID int64, flags featureFlagMap) error {
	if h.adminSvc == nil || planGroupID <= 0 {
		return domain.ErrNotSupported
	}
	allowed := automationFeatureKeys()
	cleaned := make(featureFlagMap, len(flags))
	for key, value := range flags {
		k := strings.ToLower(strings.TrimSpace(key))
		if _, ok := allowed[k]; !ok {
			continue
		}
		mode := normalizeAutomationFeatureMode(value)
		if mode == "" {
			continue
		}
		// 不允许配置只读的功能直接把 read 提升为 write，避免存下无意义的值。
		if mode == automationFeatureModeRead && !automationFeatureSpecs[k].AllowReadonly {
			mode = automationFeatureModeWrite
		}
		cleaned[k] = mode
	}
	all := h.loadAllPlanGroupFeatureFlags(c)
	id := strconv.FormatInt(planGroupID, 10)
	if len(cleaned) == 0 {
		delete(all, id)
	} else {
		all[id] = cleaned
	}
	raw, err := json.Marshal(all)
	if err != nil {
		return err
	}
	return h.adminSvc.UpdateSetting(c, getUserID(c), planGroupFeatureFlagsSettingKey, string(raw))
}

// planGroupIDForInstance 反查实例所属线路；取不到时返回 0。
func (h *Handler) planGroupIDForInstance(ctx context.Context, inst domain.VPSInstance) int64 {
	if h.catalogSvc == nil || inst.PackageID <= 0 {
		return 0
	}
	pkg, err := h.catalogSvc.GetPackage(ctx, inst.PackageID)
	if err != nil {
		return 0
	}
	return pkg.PlanGroupID
}

// applyPlanGroupFeatureFlags 把「默认策略 + 线路显式开关」叠加到能力列表上。
//
// 默认策略只在功能本身存在（插件已声明）时生效，避免凭空造出插件并不支持的能力；
// 线路显式配置则是运营的最终裁决，可以强制开启插件未声明的功能。
//
// 返回新的 DTO（不修改入参，避免影响其它调用方持有的共享切片）。
func applyPlanGroupFeatureFlags(cap *VPSAutomationCapabilityDTO, flags featureFlagMap) *VPSAutomationCapabilityDTO {
	featureSet := map[string]struct{}{}
	readonlySet := map[string]struct{}{}
	reasons := map[string]string{}
	if cap != nil {
		for _, f := range normalizeFeatureList(cap.Features) {
			featureSet[normalizeFeatureKey(f)] = struct{}{}
		}
		for _, f := range normalizeFeatureList(cap.ReadonlyFeatures) {
			readonlySet[normalizeFeatureKey(f)] = struct{}{}
		}
		for k, v := range normalizeReasons(cap.NotSupportedReasons) {
			reasons[k] = v
		}
	}

	for _, spec := range automationFeatureCatalog {
		key := normalizeFeatureKey(spec.Key)
		_, declared := featureSet[key]
		mode, explicit := flags[key]
		if !explicit || normalizeAutomationFeatureMode(mode) == "" {
			mode = spec.Default
			// 默认策略不创造能力：插件没声明的功能保持原样。
			if !declared {
				continue
			}
		}
		mode = normalizeAutomationFeatureMode(mode)

		switch mode {
		case automationFeatureModeOff:
			delete(featureSet, key)
			delete(readonlySet, key)
			if explicit && strings.TrimSpace(reasons[key]) == "" {
				reasons[key] = "线路未开启该功能"
			}
		case automationFeatureModeRead:
			featureSet[key] = struct{}{}
			readonlySet[key] = struct{}{}
			delete(reasons, key)
		default:
			featureSet[key] = struct{}{}
			delete(readonlySet, key)
			delete(reasons, key)
		}
	}

	out := &VPSAutomationCapabilityDTO{Features: sortedFeatureKeys(featureSet)}
	// 只保留确实在 features 里的只读项，避免前端出现「不在 features 里却只读」的矛盾状态。
	readonly := make([]string, 0, len(readonlySet))
	for f := range readonlySet {
		if _, ok := featureSet[f]; ok {
			readonly = append(readonly, f)
		}
	}
	if len(readonly) > 0 {
		sort.Strings(readonly)
		out.ReadonlyFeatures = readonly
	}
	if len(reasons) > 0 {
		out.NotSupportedReasons = reasons
	}
	return out
}

func sortedFeatureKeys(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for f := range set {
		out = append(out, f)
	}
	sort.Strings(out)
	return out
}

// AdminPlanGroupFeatures GET /api/v1/admin/plan-groups/:id/features
// 返回该线路显式配置的功能开关与全部可选项清单（含默认模式）。
func (h *Handler) AdminPlanGroupFeatures(c *gin.Context) {
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(400, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	c.JSON(200, gin.H{
		"items":   h.planGroupFeatureFlags(c, uri.ID),
		"catalog": automationFeatureCatalog,
	})
}

// AdminPlanGroupFeaturesUpdate PUT /api/v1/admin/plan-groups/:id/features
func (h *Handler) AdminPlanGroupFeaturesUpdate(c *gin.Context) {
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(400, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	var payload struct {
		FeatureFlags featureFlagMap `json:"feature_flags"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(400, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	if err := h.savePlanGroupFeatureFlags(c, uri.ID, payload.FeatureFlags); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true, "items": h.planGroupFeatureFlags(c, uri.ID)})
}

// fillPlanGroupFeatureFlags 为列表/单条响应的 DTO 填充 feature_flags。
func (h *Handler) fillPlanGroupFeatureFlags(ctx context.Context, items []PlanGroupDTO) {
	if len(items) == 0 {
		return
	}
	all := h.loadAllPlanGroupFeatureFlags(ctx)
	for i := range items {
		if flags, ok := all[strconv.FormatInt(items[i].ID, 10)]; ok && len(flags) > 0 {
			items[i].FeatureFlags = map[string]string(flags)
		}
	}
}

// effectiveAutomationFeatureMode 计算某实例下某个功能的生效模式。
// 返回值一定非空（默认 write），供后端写操作鉴权使用。
func (h *Handler) effectiveAutomationFeatureMode(ctx context.Context, inst domain.VPSInstance, feature string) string {
	spec, ok := automationFeatureSpecs[normalizeFeatureKey(feature)]
	if !ok {
		return automationFeatureModeWrite
	}
	flags := h.planGroupFeatureFlags(ctx, h.planGroupIDForInstance(ctx, inst))
	if mode, explicit := flags[normalizeFeatureKey(feature)]; explicit {
		if normalized := normalizeAutomationFeatureMode(mode); normalized != "" {
			return normalized
		}
	}
	return spec.Default
}

// denyIfFeatureReadonly 写操作前的只读校验；返回 true 表示已经写入错误响应。
func (h *Handler) denyIfFeatureReadonly(c *gin.Context, inst domain.VPSInstance, feature, label string) bool {
	if h.effectiveAutomationFeatureMode(c, inst, feature) != automationFeatureModeRead {
		return false
	}
	c.JSON(403, gin.H{"error": label + "当前为只读模式，无法修改"})
	return true
}
