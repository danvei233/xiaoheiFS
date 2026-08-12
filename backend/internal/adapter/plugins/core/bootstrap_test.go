package plugins

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	appports "xiaoheiplay/internal/app/ports"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/pkg/pluginsign"
)

type fakeSettingsRepo struct {
	mu       sync.Mutex
	settings map[string]domain.Setting
}

func newFakeSettingsRepo() *fakeSettingsRepo {
	return &fakeSettingsRepo{settings: map[string]domain.Setting{}}
}

func (f *fakeSettingsRepo) UpdateSettingAtomic(_ context.Context, key string, mutate func(current string) (string, error)) error {
	current, err := f.GetSetting(context.Background(), key)
	if err != nil {
		current = domain.Setting{}
	}
	next, err := mutate(current.ValueJSON)
	if err != nil {
		return err
	}
	return f.UpsertSetting(context.Background(), domain.Setting{Key: key, ValueJSON: next})
}

func (f *fakeSettingsRepo) GetSetting(_ context.Context, key string) (domain.Setting, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if s, ok := f.settings[key]; ok {
		return s, nil
	}
	return domain.Setting{}, errors.New("not found")
}

func (f *fakeSettingsRepo) UpsertSetting(_ context.Context, s domain.Setting) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.settings[s.Key] = s
	return nil
}

func (f *fakeSettingsRepo) UpsertSettings(_ context.Context, settings []domain.Setting) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, s := range settings {
		f.settings[s.Key] = s
	}
	return nil
}

func (f *fakeSettingsRepo) ListSettings(context.Context) ([]domain.Setting, error) { return nil, nil }
func (f *fakeSettingsRepo) ListEmailTemplates(context.Context) ([]domain.EmailTemplate, error) {
	return nil, nil
}
func (f *fakeSettingsRepo) GetEmailTemplate(context.Context, int64) (domain.EmailTemplate, error) {
	return domain.EmailTemplate{}, errors.New("not found")
}
func (f *fakeSettingsRepo) UpsertEmailTemplate(context.Context, *domain.EmailTemplate) error {
	return nil
}
func (f *fakeSettingsRepo) DeleteEmailTemplate(context.Context, int64) error { return nil }

// recordingRepo records every upserted installation.
type recordingRepo struct {
	insts []domain.PluginInstallation
}

func (r *recordingRepo) UpsertPluginInstallation(_ context.Context, inst *domain.PluginInstallation) error {
	r.insts = append(r.insts, *inst)
	return nil
}

func (r *recordingRepo) GetPluginInstallation(_ context.Context, category, pluginID, instanceID string) (domain.PluginInstallation, error) {
	for _, inst := range r.insts {
		if inst.Category == category && inst.PluginID == pluginID && inst.InstanceID == instanceID {
			return inst, nil
		}
	}
	return domain.PluginInstallation{}, errors.New("not found")
}

func (r *recordingRepo) ListPluginInstallations(context.Context) ([]domain.PluginInstallation, error) {
	return r.insts, nil
}

func (r *recordingRepo) DeletePluginInstallation(context.Context, string, string, string) error {
	return nil
}

// writeCorruptPlugin lays down a manifest whose checksum does not match the
// entry binary, so VerifySignature fails.
func writeCorruptPlugin(t *testing.T, baseDir, category, pluginID string) {
	t.Helper()
	dir := filepath.Join(baseDir, category, pluginID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir plugin dir: %v", err)
	}
	manifestJSON := `{"plugin_id":"` + pluginID + `","name":"Bad Plugin","version":"1.0.0","capabilities":{}}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifestJSON), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plugin"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write plugin binary: %v", err)
	}
	// A deliberately wrong sha256 forces VerifySignature to error.
	bad := `{"algo":"sha256","files":{"plugin":"` + zeros64 + `"}}`
	if err := os.WriteFile(filepath.Join(dir, "checksums.json"), []byte(bad), 0o644); err != nil {
		t.Fatalf("write checksums: %v", err)
	}
}

const zeros64 = "0000000000000000000000000000000000000000000000000000000000000000"

// --- F9: bootstrap must surface import failures instead of swallowing them ----

func TestBootstrapFromDiskReturnsUpsertError(t *testing.T) {
	base := t.TempDir()
	writeTestPlugin(t, base, "automation", "demo", false)
	repo := &failingUpsertRepo{} // zero inst so the on-disk plugin is not seen as existing
	m := NewManager(base, repo, testCipher(t), nil)

	if err := m.BootstrapFromDisk(context.Background(), nil); err == nil {
		t.Fatal("expected BootstrapFromDisk to surface the upsert failure")
	}
}

func TestBootstrapFromDiskSurfacesVerifyErrorButSetsMarker(t *testing.T) {
	base := t.TempDir()
	writeTestPlugin(t, base, "automation", "good", false) // imports cleanly
	writeCorruptPlugin(t, base, "automation", "bad")      // fails verification
	repo := &recordingRepo{}
	settings := newFakeSettingsRepo()
	m := NewManager(base, repo, testCipher(t), nil)

	// The corrupt sibling is surfaced as an error (so the caller can alert) but
	// must NOT block convergence: the marker is still set and the clean plugin is
	// still imported.
	if err := m.BootstrapFromDisk(context.Background(), settings); err == nil {
		t.Fatal("expected the corrupt plugin failure to be surfaced")
	}

	// The clean plugin must still have been imported.
	found := false
	for _, inst := range repo.insts {
		if inst.Category == "automation" && inst.PluginID == "good" {
			found = true
		}
	}
	if !found {
		t.Fatal("clean plugin was not imported despite the corrupt sibling")
	}
	// The corrupt plugin must NOT have been imported.
	for _, inst := range repo.insts {
		if inst.Category == "automation" && inst.PluginID == "bad" {
			t.Fatal("corrupt plugin was imported despite failing verification")
		}
	}

	// The bootstrapped marker MUST be set so a later restart engages the
	// official-only auto-import filter instead of re-importing every untrusted
	// plugin found on disk (supply_chain F5).
	settings.mu.Lock()
	marker, markerSet := settings.settings[pluginsBootstrappedSettingKey]
	settings.mu.Unlock()
	if !markerSet || marker.ValueJSON != "true" {
		t.Fatalf("bootstrapped marker was not set despite import failures: %+v", marker)
	}
}

func TestBootstrapFromDiskSuccessImportsAndSetsMarker(t *testing.T) {
	base := t.TempDir()
	writeTestPlugin(t, base, "automation", "demo", false)
	repo := &recordingRepo{}
	settings := newFakeSettingsRepo()
	m := NewManager(base, repo, testCipher(t), nil)

	if err := m.BootstrapFromDisk(context.Background(), settings); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.insts) != 1 {
		t.Fatalf("expected 1 imported installation, got %d", len(repo.insts))
	}
	inst := repo.insts[0]
	if inst.PluginID != "demo" || inst.Enabled {
		t.Fatalf("unexpected installation: %+v", inst)
	}
	settings.mu.Lock()
	marker, ok := settings.settings[pluginsBootstrappedSettingKey]
	settings.mu.Unlock()
	if !ok || marker.ValueJSON != "true" {
		t.Fatalf("expected bootstrapped marker to be set, got %+v", marker)
	}
}

// TestBootstrapOrphanRowDoesNotBlockMarker guards the orphan-row case
// (plugin_install F7): a pre-digest installation whose plugin directory has been
// deleted must not permanently fail the bootstrap (which would keep
// plugins_bootstrapped unset and re-import every untrusted disk plugin on every
// restart). The orphan row is kept disabled and surfaced, the valid on-disk
// plugin is imported, and the marker is set.
func TestBootstrapOrphanRowDoesNotBlockMarker(t *testing.T) {
	base := t.TempDir()
	// One valid on-disk plugin that should be imported.
	writeTestPlugin(t, base, "automation", "good", false)
	// An orphan pre-digest installation whose directory was deleted.
	existing := []domain.PluginInstallation{{
		Category:        "automation",
		PluginID:        "orphan",
		InstanceID:      DefaultInstanceID,
		Enabled:         true,
		SignatureStatus: domain.PluginSignatureUnsigned,
		ContentDigest:   "",
	}}
	repo := &seededRecordingRepo{insts: existing}
	settings := newFakeSettingsRepo()
	m := NewManager(base, repo, testCipher(t), nil)

	if err := m.BootstrapFromDisk(context.Background(), settings); err != nil {
		t.Fatalf("bootstrap must not fail on an orphan row: %v", err)
	}

	// The valid plugin is imported.
	var gotGood bool
	for _, inst := range repo.insts {
		if inst.PluginID == "good" {
			gotGood = true
		}
	}
	if !gotGood {
		t.Fatal("valid on-disk plugin was not imported")
	}

	// The orphan row is kept disabled (never backfilled with a bogus digest).
	for _, inst := range repo.insts {
		if inst.PluginID == "orphan" {
			if inst.Enabled {
				t.Fatal("orphan row must be kept disabled")
			}
			if inst.ContentDigest != "" {
				t.Fatal("orphan row must not be backfilled with the missing-dir digest")
			}
		}
	}

	// The marker is set so subsequent startups use the official-only filter.
	settings.mu.Lock()
	marker, markerSet := settings.settings[pluginsBootstrappedSettingKey]
	settings.mu.Unlock()
	if !markerSet || marker.ValueJSON != "true" {
		t.Fatalf("bootstrapped marker not set despite an orphan row: %+v", marker)
	}
}

// TestBootstrapFirstImportOfficialOnlyWhenKeysConfigured guards the first-boot
// trust gate (supply_chain F4): when a trusted official key is configured, the
// first bootstrap auto-imports ONLY official plugins. An untrusted plugin found
// on disk stays out of the installation list (it is surfaced via DiscoverOnDisk
// and requires an explicit operator import with the admin second factor), so
// pre-placing a malicious directory cannot silently seed the system.
func TestBootstrapFirstImportOfficialOnlyWhenKeysConfigured(t *testing.T) {
	base := t.TempDir()
	// Official plugin (signed with the trusted key).
	officialDir := filepath.Join(base, "automation", "official")
	if err := os.MkdirAll(officialDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(officialDir, "manifest.json"), []byte(`{"plugin_id":"official","name":"Official","version":"1.0.0","capabilities":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(officialDir, "plugin"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	trustedPub, trustedPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	writePluginChecksums(t, officialDir, []string{"manifest.json", "plugin"}, trustedPriv)

	// Untrusted plugin (unsigned).
	writeTestPlugin(t, base, "automation", "rogue", false)

	repo := &recordingRepo{}
	settings := newFakeSettingsRepo()
	m := NewManager(base, repo, testCipher(t), []ed25519.PublicKey{trustedPub})

	if err := m.BootstrapFromDisk(context.Background(), settings); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	var gotOfficial, gotRogue bool
	for _, inst := range repo.insts {
		if inst.PluginID == "official" {
			gotOfficial = true
		}
		if inst.PluginID == "rogue" {
			gotRogue = true
		}
	}
	if !gotOfficial {
		t.Fatal("official plugin was not auto-imported on first boot")
	}
	if gotRogue {
		t.Fatal("untrusted plugin must not be auto-imported when official keys are configured")
	}
}

// TestBootstrapFailClosesOnMissingDigestWithoutOfficialKey guards the upgrade
// path: a pre-digest installation that is NOT authenticated by a trusted
// official signature must not be backfilled with the current disk content (that
// would ratify any tampering that happened before the upgrade). It is failed
// closed: kept disabled, no digest written, so the operator must explicitly
// re-import/authorize it.
func TestBootstrapFailClosesOnMissingDigestWithoutOfficialKey(t *testing.T) {
	base := t.TempDir()
	writeTestPlugin(t, base, "automation", "demo", false)
	existing := []domain.PluginInstallation{{
		Category:        "automation",
		PluginID:        "demo",
		InstanceID:      DefaultInstanceID,
		Enabled:         true,
		SignatureStatus: domain.PluginSignatureUnsigned,
		ConfigCipher:    "",
	}}
	repo := &seededRecordingRepo{insts: existing}
	settings := newFakeSettingsRepo()
	// No official keys configured: nothing can authenticate the on-disk bytes.
	m := NewManager(base, repo, testCipher(t), nil)

	if err := m.BootstrapFromDisk(context.Background(), settings); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if len(repo.insts) != 1 {
		t.Fatalf("expected 1 installation, got %d", len(repo.insts))
	}
	got := repo.insts[0]
	if got.ContentDigest != "" {
		t.Fatal("bootstrap must not backfill a content digest without a trusted anchor")
	}
	if got.Enabled {
		t.Fatal("a pre-digest plugin without a trusted anchor must be kept disabled")
	}
}

// TestBootstrapBackfillsOfficialSignedDigest: a pre-digest installation whose
// on-disk content verifies as OFFICIAL against the current trusted keys can be
// anchored from those verified bytes — the signature is the authenticator, so
// backfilling is safe and the plugin may keep running.
func TestBootstrapBackfillsOfficialSignedDigest(t *testing.T) {
	base := t.TempDir()
	writeTestPlugin(t, base, "automation", "demo", false)
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	writePluginChecksums(t, filepath.Join(base, "automation", "demo"), []string{"manifest.json", "plugin"}, priv)

	existing := []domain.PluginInstallation{{
		Category:        "automation",
		PluginID:        "demo",
		InstanceID:      DefaultInstanceID,
		Enabled:         true,
		SignatureStatus: domain.PluginSignatureOfficial,
		ConfigCipher:    "",
	}}
	repo := &seededRecordingRepo{insts: existing}
	settings := newFakeSettingsRepo()
	m := NewManager(base, repo, testCipher(t), []ed25519.PublicKey{pub})

	if err := m.BootstrapFromDisk(context.Background(), settings); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	got := repo.insts[0]
	if got.ContentDigest == "" {
		t.Fatal("official signed plugin should have its digest backfilled from the verified bytes")
	}
	if !got.Enabled {
		t.Fatal("official signed plugin must keep its enabled state")
	}
}

// TestBootstrapFailClosesTamperedWithoutOfficialSignature: a pre-digest
// installation whose on-disk content no longer matches an official signature
// (tampered) must not be backfilled — keep it disabled and surfaced.
func TestBootstrapFailClosesTamperedWithoutOfficialSignature(t *testing.T) {
	base := t.TempDir()
	writeTestPlugin(t, base, "automation", "demo", false)
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	writePluginChecksums(t, filepath.Join(base, "automation", "demo"), []string{"manifest.json", "plugin"}, priv)
	// Tamper with the entry binary after signing.
	if err := os.WriteFile(filepath.Join(base, "automation", "demo", "plugin"), []byte("#!/bin/sh\nevil\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	existing := []domain.PluginInstallation{{
		Category:        "automation",
		PluginID:        "demo",
		InstanceID:      DefaultInstanceID,
		Enabled:         true,
		SignatureStatus: domain.PluginSignatureOfficial,
		ConfigCipher:    "",
	}}
	repo := &seededRecordingRepo{insts: existing}
	settings := newFakeSettingsRepo()
	m := NewManager(base, repo, testCipher(t), []ed25519.PublicKey{pub})

	if err := m.BootstrapFromDisk(context.Background(), settings); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	got := repo.insts[0]
	if got.ContentDigest != "" {
		t.Fatal("tampered plugin must not be backfilled")
	}
	if got.Enabled {
		t.Fatal("tampered pre-digest plugin must be kept disabled")
	}
}

// TestBootstrapReAnchorsOfficialUpgrade guards the upgrade path for a
// digest-anchored installation (m068 F3): a legitimate image rebuild replaces a
// bundled plugin's on-disk bytes, which at start time would otherwise be
// indistinguishable from tampering. Bootstrap re-anchors the digest when the new
// content is authenticated by a current trusted official key, so an upgraded
// official plugin keeps its enabled state instead of silently failing to start.
func TestBootstrapReAnchorsOfficialUpgrade(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "automation", "demo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{"plugin_id":"demo","name":"Demo","version":"1.0.0","capabilities":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plugin"), []byte("#!/bin/sh\nold\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	origDigest, err := pluginsign.ContentDigest(dir)
	if err != nil {
		t.Fatal(err)
	}

	// The upgrade replaces the entry binary with new content and signs it with
	// the trusted key.
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plugin"), []byte("#!/bin/sh\nnew-version\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	writePluginChecksums(t, dir, []string{"manifest.json", "plugin"}, priv)
	newDigest, err := pluginsign.ContentDigest(dir)
	if err != nil {
		t.Fatal(err)
	}
	if newDigest == origDigest {
		t.Fatal("test precondition failed: upgraded content must hash differently")
	}

	existing := []domain.PluginInstallation{{
		Category:        "automation",
		PluginID:        "demo",
		InstanceID:      DefaultInstanceID,
		Enabled:         true,
		SignatureStatus: domain.PluginSignatureOfficial,
		ContentDigest:   origDigest,
	}}
	repo := &seededRecordingRepo{insts: existing}
	settings := newFakeSettingsRepo()
	m := NewManager(base, repo, testCipher(t), []ed25519.PublicKey{pub})

	if err := m.BootstrapFromDisk(context.Background(), settings); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	got := repo.insts[0]
	if got.ContentDigest != newDigest {
		t.Fatalf("official upgrade was not re-anchored: want %q got %q", newDigest, got.ContentDigest)
	}
	if !got.Enabled {
		t.Fatal("official upgraded plugin must keep its enabled state")
	}
	if got.SignatureStatus != domain.PluginSignatureOfficial {
		t.Fatalf("re-anchored plugin signature status: %q", got.SignatureStatus)
	}
}

// TestBootstrapFailClosesChangedUnsignedDigest guards the tamper path for a
// digest-anchored installation: on-disk content that changed WITHOUT a trusted
// official signature (unsigned plugin, or a signed plugin whose signature no
// longer verifies) is never re-anchored. The recorded digest stays pinned and
// the plugin is disabled so it cannot auto-start with unverified bytes.
func TestBootstrapFailClosesChangedUnsignedDigest(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "automation", "demo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{"plugin_id":"demo","name":"Demo","version":"1.0.0","capabilities":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plugin"), []byte("#!/bin/sh\noriginal\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	origDigest, err := pluginsign.ContentDigest(dir)
	if err != nil {
		t.Fatal(err)
	}

	existing := []domain.PluginInstallation{{
		Category:        "automation",
		PluginID:        "demo",
		InstanceID:      DefaultInstanceID,
		Enabled:         true,
		SignatureStatus: domain.PluginSignatureUnsigned,
		ContentDigest:   origDigest,
	}}
	repo := &seededRecordingRepo{insts: existing}
	settings := newFakeSettingsRepo()
	// No official keys configured: nothing can authenticate the changed bytes.
	m := NewManager(base, repo, testCipher(t), nil)

	// Attacker replaces the payload after install.
	if err := os.WriteFile(filepath.Join(dir, "plugin"), []byte("#!/bin/sh\nattacker\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := m.BootstrapFromDisk(context.Background(), settings); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	got := repo.insts[0]
	if got.ContentDigest != origDigest {
		t.Fatal("changed unsigned content must not be re-anchored")
	}
	if got.Enabled {
		t.Fatal("changed unsigned plugin must be kept disabled")
	}
}

// TestBootstrapExplicitReimportAuthorizes the migration path: after a pre-digest
// plugin is failed closed, an explicit ImportFromDisk re-authorizes the current
// content by pinning a fresh digest, after which the plugin can be enabled.
func TestBootstrapExplicitReimportAuthorizes(t *testing.T) {
	base := t.TempDir()
	writeTestPlugin(t, base, "automation", "demo", false)
	existing := []domain.PluginInstallation{{
		Category:        "automation",
		PluginID:        "demo",
		InstanceID:      DefaultInstanceID,
		Enabled:         true,
		SignatureStatus: domain.PluginSignatureUnsigned,
		ConfigCipher:    "",
	}}
	repo := &seededRecordingRepo{insts: existing}
	settings := newFakeSettingsRepo()
	m := NewManager(base, repo, testCipher(t), nil)

	if err := m.BootstrapFromDisk(context.Background(), settings); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if repo.insts[0].Enabled {
		t.Fatal("pre-digest plugin should be failed closed at bootstrap")
	}

	// The operator explicitly re-imports, authorizing the current content. The
	// preview digest binds the authorization; passing it through confirms the
	// directory did not change in between.
	preview, err := m.PreviewImportFromDisk(context.Background(), "automation", "demo")
	if err != nil {
		t.Fatalf("preview re-import: %v", err)
	}
	inst, err := m.ImportFromDisk(context.Background(), "automation", "demo", preview.ContentDigest)
	if err != nil {
		t.Fatalf("explicit re-import failed: %v", err)
	}
	if inst.ContentDigest == "" {
		t.Fatal("explicit re-import must pin a content digest")
	}
	if inst.Enabled {
		t.Fatal("re-import must not silently enable the plugin")
	}
}

// seededRecordingRepo is a recordingRepo that starts with existing installations.
type seededRecordingRepo struct {
	insts []domain.PluginInstallation
}

func (r *seededRecordingRepo) UpsertPluginInstallation(_ context.Context, inst *domain.PluginInstallation) error {
	if inst == nil {
		return errors.New("nil installation")
	}
	for i := range r.insts {
		if r.insts[i].Category == inst.Category && r.insts[i].PluginID == inst.PluginID && r.insts[i].InstanceID == inst.InstanceID {
			r.insts[i] = *inst
			return nil
		}
	}
	r.insts = append(r.insts, *inst)
	return nil
}

func (r *seededRecordingRepo) GetPluginInstallation(_ context.Context, category, pluginID, instanceID string) (domain.PluginInstallation, error) {
	for _, inst := range r.insts {
		if inst.Category == category && inst.PluginID == pluginID && inst.InstanceID == instanceID {
			return inst, nil
		}
	}
	return domain.PluginInstallation{}, errors.New("not found")
}

func (r *seededRecordingRepo) ListPluginInstallations(context.Context) ([]domain.PluginInstallation, error) {
	return r.insts, nil
}

func (r *seededRecordingRepo) DeletePluginInstallation(context.Context, string, string, string) error {
	return nil
}

var _ appports.SettingsRepository = (*fakeSettingsRepo)(nil)

// TestImportFromDiskRejectsDirectorySwap is the CAS regression for the on-disk
// import path: the admin's second factor is granted for the preview's content
// digest; if the directory is swapped before the commit, the import must refuse
// instead of silently importing the replacement.
func TestImportFromDiskRejectsDirectorySwap(t *testing.T) {
	base := t.TempDir()
	pluginDir := filepath.Join(base, "automation", "demo")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(`{"plugin_id":"demo","name":"Demo","version":"1.0.0","capabilities":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin"), []byte("#!/bin/sh\noriginal\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	repo := &seededRecordingRepo{}
	m := NewManager(base, repo, testCipher(t), nil)

	// Admin inspects the directory and grants a second factor bound to its digest.
	preview, err := m.PreviewImportFromDisk(context.Background(), "automation", "demo")
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if preview.SignatureStatus != domain.PluginSignatureUnsigned {
		t.Fatalf("expected unsigned, got %q", preview.SignatureStatus)
	}

	// Attacker replaces the directory with a different plugin before the commit.
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin"), []byte("#!/bin/sh\nattacker\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	_, err = m.ImportFromDisk(context.Background(), "automation", "demo", preview.ContentDigest)
	if err == nil {
		t.Fatal("expected the swapped directory import to be refused")
	}
	if !strings.Contains(err.Error(), "changed during import") {
		t.Fatalf("expected a changed-during-import conflict, got: %v", err)
	}
	// Nothing may have been imported for the replacement.
	if _, getErr := repo.GetPluginInstallation(context.Background(), "automation", "demo", "default"); getErr == nil {
		t.Fatal("the swapped plugin must not have been imported")
	}
}

// TestImportFromDiskConcurrentSameDigest drives concurrent imports of the same
// directory: because the digest is unchanged, all may succeed, but the imported
// record must be complete and consistent (no corruption from interleaving).
func TestImportFromDiskConcurrentSameDigest(t *testing.T) {
	base := t.TempDir()
	pluginDir := filepath.Join(base, "automation", "demo")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(`{"plugin_id":"demo","name":"Demo","version":"1.0.0","capabilities":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin"), []byte("#!/bin/sh\noriginal\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	repo := &seededRecordingRepo{}
	m := NewManager(base, repo, testCipher(t), nil)
	preview, err := m.PreviewImportFromDisk(context.Background(), "automation", "demo")
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	errs := make([]error, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = m.ImportFromDisk(context.Background(), "automation", "demo", preview.ContentDigest)
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("concurrent import %d failed: %v", i, err)
		}
	}
	inst, err := repo.GetPluginInstallation(context.Background(), "automation", "demo", "default")
	if err != nil {
		t.Fatalf("get imported installation: %v", err)
	}
	if inst.ContentDigest != preview.ContentDigest {
		t.Fatalf("imported digest %q does not match the preview %q", inst.ContentDigest, preview.ContentDigest)
	}
}
