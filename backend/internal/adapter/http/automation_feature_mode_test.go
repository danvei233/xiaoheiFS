package http

import (
	"encoding/json"
	"testing"
)

func featureIndex(items []string) map[string]struct{} {
	out := make(map[string]struct{}, len(items))
	for _, item := range items {
		out[item] = struct{}{}
	}
	return out
}

func allDeclaredFeatures() *VPSAutomationCapabilityDTO {
	return &VPSAutomationCapabilityDTO{Features: []string{
		"firewall", "port_mapping", "snapshot", "backup",
		"nic", "disk", "iso", "reverse_proxy", "boot_order", "pci", "usb",
	}}
}

// 默认策略：PCI/USB 不出现；网卡/数据盘/光盘/快照只读；其余完全可用。
func TestApplyPlanGroupFeatureFlags_Defaults(t *testing.T) {
	got := applyPlanGroupFeatureFlags(allDeclaredFeatures(), nil)

	features := featureIndex(got.Features)
	readonly := featureIndex(got.ReadonlyFeatures)

	for _, key := range []string{"pci", "usb"} {
		if _, ok := features[key]; ok {
			t.Fatalf("expected %s hidden by default", key)
		}
	}
	for _, key := range []string{"nic", "disk", "iso", "snapshot"} {
		if _, ok := features[key]; !ok {
			t.Fatalf("expected %s visible by default", key)
		}
		if _, ok := readonly[key]; !ok {
			t.Fatalf("expected %s readonly by default", key)
		}
	}
	for _, key := range []string{"firewall", "port_mapping", "backup", "reverse_proxy", "boot_order"} {
		if _, ok := features[key]; !ok {
			t.Fatalf("expected %s visible by default", key)
		}
		if _, ok := readonly[key]; ok {
			t.Fatalf("expected %s writable by default", key)
		}
	}
}

// 线路显式开启写权限后，该功能不再只读。
func TestApplyPlanGroupFeatureFlags_ExplicitWrite(t *testing.T) {
	got := applyPlanGroupFeatureFlags(allDeclaredFeatures(), featureFlagMap{
		"nic": automationFeatureModeWrite,
		"pci": automationFeatureModeWrite,
	})
	features := featureIndex(got.Features)
	readonly := featureIndex(got.ReadonlyFeatures)

	if _, ok := readonly["nic"]; ok {
		t.Fatalf("expected nic writable after explicit write")
	}
	if _, ok := features["pci"]; !ok {
		t.Fatalf("expected pci visible after explicit enable")
	}
}

// 线路显式关闭后端能力列表移除，并写入原因。
func TestApplyPlanGroupFeatureFlags_ExplicitOff(t *testing.T) {
	got := applyPlanGroupFeatureFlags(allDeclaredFeatures(), featureFlagMap{
		"firewall": automationFeatureModeOff,
	})
	if _, ok := featureIndex(got.Features)["firewall"]; ok {
		t.Fatalf("expected firewall removed after explicit off")
	}
	if got.NotSupportedReasons["firewall"] == "" {
		t.Fatalf("expected reason recorded for disabled firewall")
	}
}

// 显式配置可以强制开启插件未声明的功能（历史 true 语义）。
func TestApplyPlanGroupFeatureFlags_ExplicitEnableUndeclared(t *testing.T) {
	got := applyPlanGroupFeatureFlags(&VPSAutomationCapabilityDTO{Features: []string{"lifecycle"}}, featureFlagMap{
		"usb": automationFeatureModeRead,
	})
	if _, ok := featureIndex(got.Features)["usb"]; !ok {
		t.Fatalf("expected usb force-enabled by explicit config")
	}
	if _, ok := featureIndex(got.ReadonlyFeatures)["usb"]; !ok {
		t.Fatalf("expected usb readonly per explicit config")
	}
}

// 旧数据里的布尔开关必须继续可用：true → write，false → off。
func TestFeatureFlagMap_UnmarshalLegacyBool(t *testing.T) {
	var flags featureFlagMap
	if err := json.Unmarshal([]byte(`{"nic":true,"pci":false}`), &flags); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if flags["nic"] != automationFeatureModeWrite {
		t.Fatalf("expected true mapped to write, got %q", flags["nic"])
	}
	if flags["pci"] != automationFeatureModeOff {
		t.Fatalf("expected false mapped to off, got %q", flags["pci"])
	}
}

func TestFeatureFlagMap_UnmarshalModes(t *testing.T) {
	var flags featureFlagMap
	if err := json.Unmarshal([]byte(`{"nic":"read","disk":"off","iso":"write","bad":"whatever"}`), &flags); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if flags["nic"] != automationFeatureModeRead || flags["disk"] != automationFeatureModeOff || flags["iso"] != automationFeatureModeWrite {
		t.Fatalf("unexpected modes: %#v", flags)
	}
	if _, ok := flags["bad"]; ok {
		t.Fatalf("expected invalid mode dropped")
	}
}

// 只读模式下的写操作必须被拒绝。
func TestDenyIfFeatureReadonly(t *testing.T) {
	if automationFeatureSpecs["nic"].Default != automationFeatureModeRead {
		t.Fatalf("expected nic default readonly")
	}
	if automationFeatureSpecs["pci"].Default != automationFeatureModeOff {
		t.Fatalf("expected pci default off")
	}
}
