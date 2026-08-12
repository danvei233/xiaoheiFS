package repo

import (
	"testing"

	appshared "xiaoheiplay/internal/app/shared"
)

// TestNormalizedListSettingKeys_MatchesSharedSource locks P3-4: the repo's
// list-setting-key whitelist and the shared source of truth must be identical,
// so the repo layer and the HTTP handler (isLikelyJSONSettingKey) cannot drift.
func TestNormalizedListSettingKeys_MatchesSharedSource(t *testing.T) {
	shared := appshared.JSONSettingKeySet()
	if len(normalizedListSettingKeys) != len(shared) {
		t.Fatalf("repo list keys (%d) must match shared JSON setting keys (%d)", len(normalizedListSettingKeys), len(shared))
	}
	for k := range shared {
		if _, ok := normalizedListSettingKeys[k]; !ok {
			t.Fatalf("repo list keys missing shared JSON setting key %q", k)
		}
	}
	for k := range normalizedListSettingKeys {
		if _, ok := shared[k]; !ok {
			t.Fatalf("repo list key %q has no shared JSON setting counterpart", k)
		}
	}
}
