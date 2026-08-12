package plugins

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/pkg/cryptox"
)

func newTestCipher(t *testing.T) *cryptox.AESGCM {
	t.Helper()
	key := base64.RawURLEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	cipher, err := cryptox.NewAESGCM(key)
	if err != nil {
		t.Fatal(err)
	}
	return cipher
}

// setupLightboatPluginDir writes a minimal resolvable lightboat plugin layout
// (root-fallback entry, no bin/ dir) into baseDir.
func setupLightboatPluginDir(t *testing.T, baseDir string) {
	t.Helper()
	pluginDir := filepath.Join(baseDir, "automation", "lightboat")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"plugin_id":"lightboat","name":"Lightboat","version":"1.0.0","capabilities":{"automation":{"features":["lifecycle"]}}}`
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	// Non-executable on purpose: for the enable-failure test the process must
	// fail to spawn.
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin"), []byte("#!/bin/sh\nexit 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

type fakeGoodsTypeRepo struct {
	listErr error
	created int
}

func (f *fakeGoodsTypeRepo) ListGoodsTypes(context.Context) ([]domain.GoodsType, error) {
	return nil, f.listErr
}
func (f *fakeGoodsTypeRepo) GetGoodsType(context.Context, int64) (domain.GoodsType, error) {
	return domain.GoodsType{}, errors.New("not found")
}
func (f *fakeGoodsTypeRepo) CreateGoodsType(_ context.Context, _ *domain.GoodsType) error {
	f.created++
	return nil
}
func (f *fakeGoodsTypeRepo) UpdateGoodsType(context.Context, domain.GoodsType) error { return nil }
func (f *fakeGoodsTypeRepo) DeleteGoodsType(context.Context, int64) error            { return nil }

// TestParseIntDefaultPreservesZero verifies an explicit "0" (valid for retry,
// schema minimum 0) is not converted to the default.
func TestParseIntDefaultPreservesZero(t *testing.T) {
	if got := parseIntDefault("0", 1); got != 0 {
		t.Fatalf("parseIntDefault(\"0\") = %d, want 0", got)
	}
	if got := parseIntDefault("", 1); got != 1 {
		t.Fatalf("parseIntDefault(\"\") = %d, want default 1", got)
	}
	if got := parseIntDefault("5", 1); got != 5 {
		t.Fatalf("parseIntDefault(\"5\") = %d, want 5", got)
	}
	if got := parseIntDefault("abc", 1); got != 1 {
		t.Fatalf("parseIntDefault(\"abc\") = %d, want default 1", got)
	}
}

// TestMigrateLegacyAutomationDoesNotMarkMigratedWhenEnableFails verifies that a
// failed plugin enable prevents the migration-complete marker from being
// written; otherwise automation silently stays disabled and never retries.
func TestMigrateLegacyAutomationDoesNotMarkMigratedWhenEnableFails(t *testing.T) {
	base := t.TempDir()
	setupLightboatPluginDir(t, base)
	m := NewManager(base, &fakePluginInstallationRepo{}, newTestCipher(t), nil)
	settings := &fakeSettingsRepo{settings: map[string]domain.Setting{
		"automation_base_url": {ValueJSON: "https://x"},
		"automation_api_key":  {ValueJSON: "secret"},
		"automation_enabled":  {ValueJSON: "true"},
	}}

	err := MigrateLegacyAutomationToPlugins(context.Background(), settings, nil, m)
	if err == nil {
		t.Fatal("expected error when plugin enable fails")
	}
	if _, ok := settings.settings["automation_migrated_to_plugins"]; ok {
		t.Fatal("migration must not be marked complete when enable fails")
	}
}

// TestMigrateLegacyAutomationRefusesAutoEnableUntrustedWithOfficialKeys guards
// the auto-enable trust gate (supply_chain F4): when a trusted official key is
// configured, the migration must NOT auto-create+configure+enable a pre-placed
// automation/lightboat directory whose content is not authenticated by that key.
// CreateInstance self-certifies whatever is on disk, so without this gate a
// malicious pre-placed directory would be enabled with no admin interaction.
func TestMigrateLegacyAutomationRefusesAutoEnableUntrustedWithOfficialKeys(t *testing.T) {
	base := t.TempDir()
	setupLightboatPluginDir(t, base)
	// A trusted official key that does NOT sign the on-disk plugin.
	trustedPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	m := NewManager(base, &fakePluginInstallationRepo{}, newTestCipher(t), []ed25519.PublicKey{trustedPub})
	settings := &fakeSettingsRepo{settings: map[string]domain.Setting{
		"automation_base_url": {ValueJSON: "https://x"},
		"automation_api_key":  {ValueJSON: "secret"},
		"automation_enabled":  {ValueJSON: "true"},
	}}

	err = MigrateLegacyAutomationToPlugins(context.Background(), settings, nil, m)
	if err == nil {
		t.Fatal("expected refusal to auto-enable an unverified automation plugin")
	}
	if !strings.Contains(err.Error(), "refusing to auto-enable") {
		t.Fatalf("expected the auto-enable refusal, got: %v", err)
	}
	if _, ok := settings.settings["automation_migrated_to_plugins"]; ok {
		t.Fatal("migration must not be marked complete when auto-enable is refused")
	}
}

// TestMigrateLegacyAutomationAutoEnablesOfficialPlugin verifies the gate does
// NOT block a plugin whose on-disk content IS authenticated by the trusted key:
// the migration proceeds past the gate (and fails later only because the test
// plugin binary cannot actually spawn, exactly like the nil-key test).
func TestMigrateLegacyAutomationAutoEnablesOfficialPlugin(t *testing.T) {
	base := t.TempDir()
	setupLightboatPluginDir(t, base)
	trustedPub, trustedPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	writePluginChecksums(t, filepath.Join(base, "automation", "lightboat"), []string{"manifest.json", "plugin"}, trustedPriv)

	m := NewManager(base, &fakePluginInstallationRepo{}, newTestCipher(t), []ed25519.PublicKey{trustedPub})
	settings := &fakeSettingsRepo{settings: map[string]domain.Setting{
		"automation_base_url": {ValueJSON: "https://x"},
		"automation_api_key":  {ValueJSON: "secret"},
		"automation_enabled":  {ValueJSON: "true"},
	}}

	err = MigrateLegacyAutomationToPlugins(context.Background(), settings, nil, m)
	// The gate must NOT refuse an officially-signed plugin; the only failure the
	// test can observe is the enable attempt failing because the shell-script
	// "binary" cannot complete the go-plugin handshake. If it refused with the
	// auto-enable message instead, the gate is over-blocking.
	if err != nil && strings.Contains(err.Error(), "refusing to auto-enable") {
		t.Fatalf("the trust gate over-blocked an officially signed plugin: %v", err)
	}
}

// TestMigrateLegacyAutomationSkipsGoodsTypeCreationOnListError verifies a read
// error from ListGoodsTypes does not trigger a default goods-type write, and
// that a fully successful migration clears the legacy plaintext api_key.
func TestMigrateLegacyAutomationSkipsGoodsTypeCreationOnListError(t *testing.T) {
	base := t.TempDir()
	setupLightboatPluginDir(t, base)
	m := NewManager(base, &fakePluginInstallationRepo{}, newTestCipher(t), nil)
	settings := &fakeSettingsRepo{settings: map[string]domain.Setting{
		"automation_base_url": {ValueJSON: "https://x"},
		"automation_api_key":  {ValueJSON: "secret"},
		"automation_enabled":  {ValueJSON: "false"},
	}}
	goods := &fakeGoodsTypeRepo{listErr: errors.New("db down")}

	err := MigrateLegacyAutomationToPlugins(context.Background(), settings, goods, m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if goods.created != 0 {
		t.Fatalf("expected no goods type created on list error, got %d", goods.created)
	}
	if _, ok := settings.settings["automation_migrated_to_plugins"]; !ok {
		t.Fatal("expected migration marker to be written")
	}
	if got := settings.settings["automation_api_key"].ValueJSON; got != "" {
		t.Fatalf("expected legacy api_key to be cleared after migration, got %q", got)
	}
}
