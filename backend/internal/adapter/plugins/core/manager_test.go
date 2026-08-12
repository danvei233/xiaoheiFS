package plugins

import (
	"context"
	"encoding/base64"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/pkg/cryptox"

	"xiaoheiplay/pkg/pluginsign"
	pluginv1 "xiaoheiplay/plugin/v1"
)

func testCipher(t *testing.T) *cryptox.AESGCM {
	t.Helper()
	key := base64.RawURLEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	cipher, err := cryptox.NewAESGCM(key)
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}
	return cipher
}

// failingUpsertRepo always fails the persistence step.
type failingUpsertRepo struct {
	inst domain.PluginInstallation
}

func (f *failingUpsertRepo) UpsertPluginInstallation(context.Context, *domain.PluginInstallation) error {
	return errors.New("db down")
}

func (f *failingUpsertRepo) GetPluginInstallation(_ context.Context, category, pluginID, instanceID string) (domain.PluginInstallation, error) {
	if f.inst.Category == category && f.inst.PluginID == pluginID && f.inst.InstanceID == instanceID {
		return f.inst, nil
	}
	return domain.PluginInstallation{}, errors.New("not found")
}

func (f *failingUpsertRepo) ListPluginInstallations(context.Context) ([]domain.PluginInstallation, error) {
	return []domain.PluginInstallation{f.inst}, nil
}

func (f *failingUpsertRepo) DeletePluginInstallation(context.Context, string, string, string) error {
	return nil
}

// --- F3: a failed reload must roll the persisted config back ----------------

func TestUpdateConfigInstanceReloadFailureRollsBackRepo(t *testing.T) {
	cipher := testCipher(t)
	oldCipher, err := cipher.EncryptToString([]byte(`{"base_url":"https://old.example.com"}`))
	if err != nil {
		t.Fatalf("encrypt old config: %v", err)
	}
	repo := &fakePluginInstallationRepo{inst: domain.PluginInstallation{
		Category:     "automation",
		PluginID:     "lightboat",
		InstanceID:   "default",
		Enabled:      true,
		ConfigCipher: oldCipher,
	}}
	m := NewManager(t.TempDir(), repo, cipher, nil)

	// Simulate a running plugin whose ReloadConfig rejects the new config.
	rp := &runningPlugin{
		category:   "automation",
		pluginID:   "lightboat",
		instanceID: "default",
		core: &fakeCoreClient{reloadFn: func(*pluginv1.ReloadConfigRequest) (*pluginv1.ReloadConfigResponse, error) {
			return &pluginv1.ReloadConfigResponse{Ok: false, Error: "rejected"}, nil
		}},
	}
	m.runtime.mu.Lock()
	m.runtime.running["automation:lightboat:default"] = rp
	m.runtime.mu.Unlock()

	err = m.UpdateConfigInstance(context.Background(), "automation", "lightboat", "default", `{"base_url":"https://new.example.com"}`)
	if err == nil {
		t.Fatal("expected reload failure error")
	}
	inst, err := repo.GetPluginInstallation(context.Background(), "automation", "lightboat", "default")
	if err != nil {
		t.Fatalf("get installation: %v", err)
	}
	if inst.ConfigCipher != oldCipher {
		t.Fatal("repository config was not rolled back after reload failure")
	}
}

func TestUpdateConfigInstanceReloadRPCErrorRollsBackRepo(t *testing.T) {
	cipher := testCipher(t)
	oldCipher, err := cipher.EncryptToString([]byte(`{"base_url":"https://old.example.com"}`))
	if err != nil {
		t.Fatalf("encrypt old config: %v", err)
	}
	repo := &fakePluginInstallationRepo{inst: domain.PluginInstallation{
		Category:     "automation",
		PluginID:     "lightboat",
		InstanceID:   "default",
		Enabled:      true,
		ConfigCipher: oldCipher,
	}}
	m := NewManager(t.TempDir(), repo, cipher, nil)
	rp := &runningPlugin{
		category:   "automation",
		pluginID:   "lightboat",
		instanceID: "default",
		core: &fakeCoreClient{reloadFn: func(*pluginv1.ReloadConfigRequest) (*pluginv1.ReloadConfigResponse, error) {
			return nil, errors.New("rpc unavailable")
		}},
	}
	m.runtime.mu.Lock()
	m.runtime.running["automation:lightboat:default"] = rp
	m.runtime.mu.Unlock()

	if err := m.UpdateConfigInstance(context.Background(), "automation", "lightboat", "default", `{"base_url":"https://new.example.com"}`); err == nil {
		t.Fatal("expected reload RPC error")
	}
	inst, err := repo.GetPluginInstallation(context.Background(), "automation", "lightboat", "default")
	if err != nil {
		t.Fatalf("get installation: %v", err)
	}
	if inst.ConfigCipher != oldCipher {
		t.Fatal("repository config was not rolled back after reload RPC failure")
	}
}

// --- F7: a persistence failure after Start must not orphan the process -------

func TestEnableInstanceStopsProcessWhenPersistFails(t *testing.T) {
	base := t.TempDir()
	// The fake client factory serves a manifest with plugin_id "demo", so the
	// on-disk manifest must match for Start to succeed.
	writeTestPlugin(t, base, "automation", "demo", false)
	digest, err := pluginsign.ContentDigest(filepath.Join(base, "automation", "demo"))
	if err != nil {
		t.Fatal(err)
	}
	factory := &fakeClientFactory{}
	repo := &failingUpsertRepo{inst: domain.PluginInstallation{
		Category:      "automation",
		PluginID:      "demo",
		InstanceID:    "default",
		Enabled:       false,
		ContentDigest: digest,
	}}
	m := NewManager(base, repo, testCipher(t), nil)
	m.runtime.newClient = factory.new

	if err := m.EnableInstance(context.Background(), "automation", "demo", "default"); err == nil {
		t.Fatal("expected persistence failure")
	}
	if _, ok := m.runtime.GetRunning("automation", "demo", "default"); ok {
		t.Fatal("plugin process should have been stopped after persistence failure")
	}
}

// TestGetConfigInstanceRedactsAutomationAPIKey guards m068 F1: automation config
// returned through the admin API must have its api_key redacted, not returned in
// plaintext.
func TestGetConfigInstanceRedactsAutomationAPIKey(t *testing.T) {
	cipher := newTestCipher(t)
	cfg := `{"base_url":"https://panel.example.com","api_key":"AUTOMATION-SECRET-KEY","timeout_sec":12}`
	cipherText, err := cipher.EncryptToString([]byte(cfg))
	if err != nil {
		t.Fatal(err)
	}
	repo := &fakePluginInstallationRepo{inst: domain.PluginInstallation{
		Category:        "automation",
		PluginID:        "lightboat",
		InstanceID:      "default",
		Enabled:         true,
		SignatureStatus: domain.PluginSignatureUnsigned,
		ConfigCipher:    cipherText,
	}}
	m := NewManager(t.TempDir(), repo, cipher, nil)

	got, err := m.GetConfigInstance(context.Background(), "automation", "lightboat", "default")
	if err != nil {
		t.Fatalf("GetConfigInstance: %v", err)
	}
	if strings.Contains(got, "AUTOMATION-SECRET-KEY") {
		t.Fatalf("automation api_key leaked in admin config response: %s", got)
	}
	if !strings.Contains(got, "base_url") {
		t.Fatalf("non-secret automation config should still be returned: %s", got)
	}
}
