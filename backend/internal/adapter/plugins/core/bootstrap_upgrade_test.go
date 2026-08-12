package plugins

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	appports "xiaoheiplay/internal/app/ports"
	"xiaoheiplay/internal/domain"
)

// bootstrapRepo is an in-memory PluginInstallationRepository holding multiple
// installations, so an image upgrade can observe which new plugins got imported.
type bootstrapRepo struct {
	insts map[string]domain.PluginInstallation
}

func (r *bootstrapRepo) key(category, pluginID, instanceID string) string {
	return category + ":" + pluginID + ":" + instanceID
}

func (r *bootstrapRepo) UpsertPluginInstallation(_ context.Context, inst *domain.PluginInstallation) error {
	if inst == nil {
		return fmt.Errorf("nil installation")
	}
	r.insts[r.key(inst.Category, inst.PluginID, inst.InstanceID)] = *inst
	return nil
}

func (r *bootstrapRepo) GetPluginInstallation(_ context.Context, category, pluginID, instanceID string) (domain.PluginInstallation, error) {
	v, ok := r.insts[r.key(category, pluginID, instanceID)]
	if !ok {
		return domain.PluginInstallation{}, fmt.Errorf("not found")
	}
	return v, nil
}

func (r *bootstrapRepo) ListPluginInstallations(context.Context) ([]domain.PluginInstallation, error) {
	out := make([]domain.PluginInstallation, 0, len(r.insts))
	for _, v := range r.insts {
		out = append(out, v)
	}
	return out, nil
}

func (r *bootstrapRepo) DeletePluginInstallation(_ context.Context, category, pluginID, instanceID string) error {
	delete(r.insts, r.key(category, pluginID, instanceID))
	return nil
}

// bootstrapSettings is an in-memory SettingsRepository.
type bootstrapSettings struct {
	vals map[string]domain.Setting
}

func (s *bootstrapSettings) UpdateSettingAtomic(_ context.Context, key string, mutate func(current string) (string, error)) error {
	current, err := s.GetSetting(context.Background(), key)
	if err != nil {
		current = domain.Setting{}
	}
	next, err := mutate(current.ValueJSON)
	if err != nil {
		return err
	}
	return s.UpsertSetting(context.Background(), domain.Setting{Key: key, ValueJSON: next})
}

func (s *bootstrapSettings) GetSetting(_ context.Context, key string) (domain.Setting, error) {
	v, ok := s.vals[key]
	if !ok {
		return domain.Setting{}, fmt.Errorf("not found")
	}
	return v, nil
}

func (s *bootstrapSettings) UpsertSetting(_ context.Context, setting domain.Setting) error {
	s.vals[setting.Key] = setting
	return nil
}

func (s *bootstrapSettings) UpsertSettings(_ context.Context, settings []domain.Setting) error {
	for _, setting := range settings {
		s.vals[setting.Key] = setting
	}
	return nil
}

func (s *bootstrapSettings) ListSettings(context.Context) ([]domain.Setting, error) {
	out := make([]domain.Setting, 0, len(s.vals))
	for _, v := range s.vals {
		out = append(out, v)
	}
	return out, nil
}

func (s *bootstrapSettings) ListEmailTemplates(context.Context) ([]domain.EmailTemplate, error) {
	return nil, nil
}

func (s *bootstrapSettings) GetEmailTemplate(context.Context, int64) (domain.EmailTemplate, error) {
	return domain.EmailTemplate{}, fmt.Errorf("not found")
}

func (s *bootstrapSettings) UpsertEmailTemplate(context.Context, *domain.EmailTemplate) error { return nil }
func (s *bootstrapSettings) DeleteEmailTemplate(context.Context, int64) error                { return nil }

func writeBootstrapPluginDir(t *testing.T, dir, pluginID string, priv ed25519.PrivateKey) {
	t.Helper()
	_ = os.MkdirAll(filepath.Join(dir, "bin", "linux_amd64"), 0o755)
	manifest := fmt.Sprintf(`{"plugin_id":%q,"name":%q,"version":"1.0.0","binaries":{"linux_amd64":"bin/linux_amd64/plugin"}}`, pluginID, pluginID)
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bin", "linux_amd64", "plugin"), []byte("binary-"+pluginID), 0o755); err != nil {
		t.Fatal(err)
	}
	if priv == nil {
		return // unsigned
	}
	csBytes := buildTestChecksums(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "checksums.json"), csBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	sig := ed25519.Sign(priv, csBytes)
	if err := os.WriteFile(filepath.Join(dir, "signature.sig"), sig, 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestBootstrapFromDiskImageUpgrade pins the P2-65 release/upgrade contract:
//
//	1. first bootstrap imports every signed bundled plugin as Official;
//	2. on a subsequent image upgrade a NEW signed plugin is auto-imported;
//	3. a NEW unsigned plugin is logged-and-skipped, never silently imported.
func TestBootstrapFromDiskImageUpgrade(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	baseDir := t.TempDir()
	writeBootstrapPluginDir(t, filepath.Join(baseDir, "payment", "signed1"), "signed1", priv)

	repo := &bootstrapRepo{insts: map[string]domain.PluginInstallation{}}
	settings := &bootstrapSettings{vals: map[string]domain.Setting{}}
	m := NewManager(baseDir, repo, nil, []ed25519.PublicKey{pub})

	// Phase 1: first bootstrap imports the signed plugin and marks bootstrapped.
	if err := m.BootstrapFromDisk(context.Background(), settings); err != nil {
		t.Fatalf("first bootstrap: %v", err)
	}
	inst, err := repo.GetPluginInstallation(context.Background(), "payment", "signed1", "default")
	if err != nil {
		t.Fatalf("signed plugin not imported on first bootstrap: %v", err)
	}
	if inst.SignatureStatus != domain.PluginSignatureOfficial {
		t.Fatalf("first-bootstrap signature status = %v, want Official", inst.SignatureStatus)
	}
	if s, _ := settings.GetSetting(context.Background(), pluginsBootstrappedSettingKey); s.ValueJSON != "true" {
		t.Fatalf("plugins_bootstrapped not recorded after first bootstrap")
	}

	// Phase 2: image upgrade. New unsigned plugin must be logged-and-skipped;
	// a new signed plugin must be imported as Official.
	writeBootstrapPluginDir(t, filepath.Join(baseDir, "payment", "unsigned-new"), "unsigned-new", nil)
	writeBootstrapPluginDir(t, filepath.Join(baseDir, "payment", "signed2"), "signed2", priv)

	if err := m.BootstrapFromDisk(context.Background(), settings); err != nil {
		t.Fatalf("upgrade bootstrap: %v", err)
	}
	if _, err := repo.GetPluginInstallation(context.Background(), "payment", "unsigned-new", "default"); err == nil {
		t.Fatalf("unsigned plugin was auto-imported on upgrade (must be logged-and-skipped)")
	}
	inst2, err := repo.GetPluginInstallation(context.Background(), "payment", "signed2", "default")
	if err != nil {
		t.Fatalf("signed new plugin not imported on upgrade: %v", err)
	}
	if inst2.SignatureStatus != domain.PluginSignatureOfficial {
		t.Fatalf("upgrade-imported signature status = %v, want Official", inst2.SignatureStatus)
	}
	// The already-imported signed1 is untouched.
	if _, err := repo.GetPluginInstallation(context.Background(), "payment", "signed1", "default"); err != nil {
		t.Fatalf("existing signed plugin lost on upgrade: %v", err)
	}
}

// TestVerifySignatureRejectsTamperedBinary pins the installed-bytes binding:
// flipping a byte in the binary must make the whole bundle Untrusted even when
// the signature bytes still verify (checksum is over the actual files).
func TestVerifySignatureRejectsTamperedBinary(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	writeBootstrapPluginDir(t, dir, "tampered", priv)
	bin := filepath.Join(dir, "bin", "linux_amd64", "plugin")
	orig, _ := os.ReadFile(bin)
	if err := os.WriteFile(bin, append(orig, []byte("extra")...), 0o755); err != nil {
		t.Fatal(err)
	}
	status, err := VerifySignature(dir, []ed25519.PublicKey{pub})
	if err == nil {
		t.Fatal("expected checksum-mismatch error after tampering the binary")
	}
	if status != domain.PluginSignatureUntrusted {
		t.Fatalf("tampered signature status = %v, want Untrusted", status)
	}
}

// compile-time interface assertions for the fakes.
var _ appports.PluginInstallationRepository = (*bootstrapRepo)(nil)
var _ appports.SettingsRepository = (*bootstrapSettings)(nil)
