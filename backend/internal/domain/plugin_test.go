package domain

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// TestPluginInstallationConfigCipherNotSerialized guards against the plugin
// configuration ciphertext (which may embed payment-gateway keys) leaking into
// HTTP responses. handlers_admin_plugins.go serializes domain.PluginInstallation
// directly via c.JSON for install/import/create-instance, so the field must
// never be marshaled to JSON.
func TestPluginInstallationConfigCipherNotSerialized(t *testing.T) {
	inst := PluginInstallation{
		ID:              1,
		Category:        "payment",
		PluginID:        "stripe",
		InstanceID:      "default",
		Enabled:         true,
		SignatureStatus: PluginSignatureOfficial,
		ConfigCipher:    "SUPERSECRET_CIPHERTEXT_VALUE",
		CreatedAt:       time.Unix(1700000000, 0),
		UpdatedAt:       time.Unix(1700000000, 0),
	}

	raw, err := json.Marshal(inst)
	if err != nil {
		t.Fatalf("marshal PluginInstallation: %v", err)
	}
	if strings.Contains(string(raw), "SUPERSECRET_CIPHERTEXT_VALUE") {
		t.Fatalf("PluginInstallation.ConfigCipher leaked into JSON output: %s", raw)
	}
}
