package http

import "testing"

func TestValidateSettingJSONValue(t *testing.T) {
	if err := validateSettingJSONValue("site_name", "abc"); err != nil {
		t.Fatalf("plain setting should pass: %v", err)
	}
	if err := validateSettingJSONValue("site_nav_items", `[{"label":"A","url":"/a"}]`); err != nil {
		t.Fatalf("json array should pass: %v", err)
	}
	if err := validateSettingJSONValue("site_nav_items", `"{\"label\":\"A\"}"`); err == nil {
		t.Fatalf("double-encoded json should be rejected")
	}
}

func TestValidatePluginConfigJSON(t *testing.T) {
	if err := validatePluginConfigJSON(`{"a":1}`); err != nil {
		t.Fatalf("valid json should pass: %v", err)
	}
	if err := validatePluginConfigJSON(`"{\"a\":1}"`); err == nil {
		t.Fatalf("double-encoded config json should be rejected")
	}
}
