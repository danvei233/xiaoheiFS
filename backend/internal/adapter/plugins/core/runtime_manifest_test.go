package plugins

import (
	"strings"
	"testing"

	pluginv1 "xiaoheiplay/plugin/v1"
)

func TestValidateManifestConsistencyCatalogReadonlyMatch(t *testing.T) {
	jsonM := Manifest{
		PluginID: "demo",
		Name:     "Demo Plugin",
		Version:  "1.0.0",
	}
	jsonM.Capabilities.Automation = &struct {
		Features           []string          `json:"features"`
		NotSupportedReason map[string]string `json:"not_supported_reasons,omitempty"`
		CatalogReadonly    bool              `json:"catalog_readonly,omitempty"`
	}{
		Features:        []string{"catalog_sync"},
		CatalogReadonly: true,
	}

	grpcM := &pluginv1.Manifest{
		PluginId: "demo",
		Name:     "Demo Plugin",
		Version:  "1.0.0",
		Automation: &pluginv1.AutomationCapability{
			Features:        []pluginv1.AutomationFeature{pluginv1.AutomationFeature_AUTOMATION_FEATURE_CATALOG_SYNC},
			CatalogReadonly: true,
		},
	}

	if err := validateManifestConsistency(jsonM, grpcM); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateManifestConsistencyCatalogReadonlyMismatch(t *testing.T) {
	jsonM := Manifest{
		PluginID: "demo",
		Name:     "Demo Plugin",
		Version:  "1.0.0",
	}
	jsonM.Capabilities.Automation = &struct {
		Features           []string          `json:"features"`
		NotSupportedReason map[string]string `json:"not_supported_reasons,omitempty"`
		CatalogReadonly    bool              `json:"catalog_readonly,omitempty"`
	}{
		Features:        []string{"catalog_sync"},
		CatalogReadonly: true,
	}

	grpcM := &pluginv1.Manifest{
		PluginId: "demo",
		Name:     "Demo Plugin",
		Version:  "1.0.0",
		Automation: &pluginv1.AutomationCapability{
			Features:        []pluginv1.AutomationFeature{pluginv1.AutomationFeature_AUTOMATION_FEATURE_CATALOG_SYNC},
			CatalogReadonly: false,
		},
	}

	err := validateManifestConsistency(jsonM, grpcM)
	if err == nil {
		t.Fatalf("expected mismatch error")
	}
	if !strings.Contains(err.Error(), "automation.catalog_readonly") {
		t.Fatalf("unexpected error: %v", err)
	}
}
