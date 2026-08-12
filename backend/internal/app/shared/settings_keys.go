package shared

import "strings"

// JSONSettingKeys is the single source of truth for settings whose value is a
// JSON document (object/array) and must be JSON-validated on write. Both the
// HTTP handler (isLikelyJSONSettingKey) and the repo layer
// (normalizedListSettingKeys) derive from this list so the two cannot drift
// (P3-4). Additional JSON-typed settings are matched structurally rather than
// listed here: keys ending in `_json` and keys prefixed with `task.`.
var JSONSettingKeys = []string{
	"site_nav_items",
	"auth_register_required_fields",
	"auth_register_verify_channels",
	"auth_login_notify_channels",
	"auth_password_reset_channels",
	"realname_block_actions",
	"robot_webhooks",
}

// IsNamedJSONSettingKey reports whether key is one of the named JSON settings
// keys (excluding the structural *_json / task. matches).
func IsNamedJSONSettingKey(key string) bool {
	k := strings.ToLower(strings.TrimSpace(key))
	for _, candidate := range JSONSettingKeys {
		if k == candidate {
			return true
		}
	}
	return false
}

// JSONSettingKeySet returns the named JSON settings keys as a set.
func JSONSettingKeySet() map[string]struct{} {
	out := make(map[string]struct{}, len(JSONSettingKeys))
	for _, k := range JSONSettingKeys {
		out[k] = struct{}{}
	}
	return out
}
