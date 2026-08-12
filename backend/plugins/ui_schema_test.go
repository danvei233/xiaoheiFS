package plugins

import (
	"os"
	"path/filepath"
	"testing"
)

// TestUISchemaKeysReferenceSchemaProperties guards against a config.ui.json
// referencing a field that does not exist in the paired config.schema.json.
// Regression for the ezpay UI schema binding "key" while the schema and the
// plugin process expose "merchant_key".
func TestUISchemaKeysReferenceSchemaProperties(t *testing.T) {
	plugins := []struct {
		category string
		plugin   string
	}{
		{"automation", "lightboat"},
		{"automation", "mofang_openapi"},
		{"automation", "openidc_default"},
		{"automation", "xiaohei_proxy"},
		{"payment", "ezpay"},
	}
	for _, p := range plugins {
		uiRel := filepath.Join(p.category, p.plugin, "schemas", "config.ui.json")
		if _, err := os.Stat(uiRel); err != nil {
			continue
		}
		schema := loadSchema(t, filepath.Join(p.category, p.plugin, "schemas", "config.schema.json"))
		for key := range loadUISchema(t, uiRel) {
			if _, ok := schema.Properties[key]; !ok {
				t.Errorf("%s: ui key %q is not defined in config.schema.json properties", uiRel, key)
			}
		}
	}
}
