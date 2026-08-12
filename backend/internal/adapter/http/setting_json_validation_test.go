package http

import (
	"testing"

	appshared "xiaoheiplay/internal/app/shared"
)

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

// TestIsLikelyJSONSettingKey_MatchesSharedSource locks P3-4: the handler's named
// JSON-settings keys and the shared source of truth (app/shared.JSONSettingKeys)
// must agree, so the handler list can no longer drift from the repo whitelist.
func TestIsLikelyJSONSettingKey_MatchesSharedSource(t *testing.T) {
	shared := appshared.JSONSettingKeySet()
	for _, key := range appshared.JSONSettingKeys {
		if !isLikelyJSONSettingKey(key) {
			t.Fatalf("isLikelyJSONSettingKey must match shared JSON setting key %q", key)
		}
	}
	// No named key outside the shared set may be matched: a stray literal in the
	// handler (or a stray removal from the repo) is a contract violation.
	for _, key := range []string{
		"site_name", "smtp_port", "emergency_renew_window_days",
		"auth_register_verify_ttl_sec", "refund_full_days",
	} {
		if isLikelyJSONSettingKey(key) {
			t.Fatalf("plain setting key %q must not be treated as JSON", key)
		}
	}
	_ = shared
	if len(shared) != len(appshared.JSONSettingKeys) {
		t.Fatalf("shared JSONSettingKeys must have unique entries")
	}
}

// TestValidateAndNormalizeSettingValue_RejectsNonPositiveEmergencyWindow locks
// P3-27: the config validation must reject 0/negative emergency_renew_window_days
// instead of silently opening the emergency renew window forever.
func TestValidateAndNormalizeSettingValue_RejectsNonPositiveEmergencyWindow(t *testing.T) {
	for _, raw := range []string{"0", "-1", "-7", "abc"} {
		if _, err := validateAndNormalizeSettingValue("emergency_renew_window_days", raw); err == nil {
			t.Fatalf("emergency_renew_window_days=%q must be rejected", raw)
		}
	}
	if _, err := validateAndNormalizeSettingValue("emergency_renew_window_days", "7"); err != nil {
		t.Fatalf("emergency_renew_window_days=7 must be accepted: %v", err)
	}
}

// TestValidateAndNormalizeSettingValue_RejectsMalformedJSONForJSONKeys locks
// P2-1: the default branch of validateAndNormalizeSettingValue must run JSON
// validation for keys flagged by isLikelyJSONSettingKey (the validator used to
// be dead code and malformed *_json values passed straight through).
func TestValidateAndNormalizeSettingValue_RejectsMalformedJSONForJSONKeys(t *testing.T) {
	if _, err := validateAndNormalizeSettingValue("refund_curve_json", `[]`); err != nil {
		t.Fatalf("valid json array should pass: %v", err)
	}
	if _, err := validateAndNormalizeSettingValue("refund_curve_json", `{"a":1}`); err != nil {
		t.Fatalf("valid json object should pass: %v", err)
	}
	if _, err := validateAndNormalizeSettingValue("refund_curve_json", `not json`); err == nil {
		t.Fatalf("malformed json must be rejected for a *_json key")
	}
	if _, err := validateAndNormalizeSettingValue("refund_curve_json", `"{\"a\":1}"`); err == nil {
		t.Fatalf("double-encoded json must be rejected for a *_json key")
	}
	if _, err := validateAndNormalizeSettingValue("some_unknown_json", `[1,2]`); err != nil {
		t.Fatalf("structural *_json key with valid array should pass: %v", err)
	}
	if _, err := validateAndNormalizeSettingValue("site_name", "plain"); err != nil {
		t.Fatalf("plain key must pass unchanged: %v", err)
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
