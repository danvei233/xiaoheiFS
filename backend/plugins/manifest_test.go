package plugins

import (
	"encoding/json"
	"os"
	"testing"
)

// TestMockpayManifestIsMarkedTestOnly guards that the test-only payment plugin
// carries an explicit structured marker so tooling can gate it out of release
// bundles independently of the description string.
func TestMockpayManifestIsMarkedTestOnly(t *testing.T) {
	b, err := os.ReadFile("payment/mockpay/manifest.json")
	if err != nil {
		t.Fatalf("read mockpay manifest: %v", err)
	}
	var m struct {
		TestOnly bool `json:"test_only"`
	}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("parse mockpay manifest: %v", err)
	}
	if !m.TestOnly {
		t.Error(`mockpay manifest must declare "test_only": true (release-smoke excludes it from distribution)`)
	}
}
