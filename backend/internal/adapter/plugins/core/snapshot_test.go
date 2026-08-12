package plugins

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/pkg/pluginsign"
)

// TestEnsureRunningExecutesFromVerifiedSnapshot is the verify-to-spawn binding
// regression: after verification, the runtime must exec the plugin from the
// immutable content-addressed snapshot (a copy made and verified in one step),
// never from the live plugin dir. Replacing the entry binary in the live dir
// after the snapshot exists must not change what is executed.
func TestEnsureRunningExecutesFromVerifiedSnapshot(t *testing.T) {
	base := t.TempDir()
	pluginDir := filepath.Join(base, "automation", "demo")
	if err := os.MkdirAll(filepath.Join(pluginDir, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	orig := "#!/bin/sh\noriginal-binary\n"
	// The fake client factory serves Name "Demo Plugin", so the on-disk manifest
	// must match for manifest-consistency validation to pass.
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(`{"plugin_id":"demo","name":"Demo Plugin","version":"1.0.0","capabilities":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin"), []byte(orig), 0o755); err != nil {
		t.Fatal(err)
	}
	digest, err := pluginsign.ContentDigest(pluginDir)
	if err != nil {
		t.Fatal(err)
	}

	repo := &fakePluginInstallationRepo{inst: domain.PluginInstallation{
		Category:        "automation",
		PluginID:        "demo",
		InstanceID:      "default",
		Enabled:         true,
		SignatureStatus: domain.PluginSignatureUnsigned,
		ContentDigest:   digest,
	}}
	factory := &fakeClientFactory{}
	m := NewManager(base, repo, testCipher(t), nil)
	m.runtime.newClient = factory.new

	if _, err := m.EnsureRunning(context.Background(), "automation", "demo", "default"); err != nil {
		t.Fatalf("EnsureRunning: %v", err)
	}
	if len(factory.configs) == 0 {
		t.Fatal("no plugin client config captured")
	}
	cmdPath := filepath.Clean(factory.configs[0].Cmd.Path)
	snapRoot := filepath.Clean(filepath.Join(base, snapshotRootName))
	if !strings.HasPrefix(cmdPath, snapRoot+string(os.PathSeparator)) {
		t.Fatalf("exec target %q is outside the verified snapshot root %q; the plugin would run from the live dir", cmdPath, snapRoot)
	}

	// The snapshot must contain the ORIGINAL verified entry binary, and it must
	// be the bytes that were validated (not a copy of the live dir made later).
	snapPlugin := filepath.Join(snapRoot, "automation", "demo", digest, "plugin")
	got, err := os.ReadFile(snapPlugin)
	if err != nil {
		t.Fatalf("snapshot entry binary missing: %v", err)
	}
	if string(got) != orig {
		t.Fatalf("snapshot entry binary differs from the verified original")
	}
}

// TestEnsureRunningRefusesLiveSwapAfterSnapshot guards the fail-closed side of
// the same guarantee: even though a verified snapshot exists, a live-dir swap is
// still detected on the next cold start and the plugin is refused, so tampering
// cannot go unnoticed or run from the live path.
func TestEnsureRunningRefusesLiveSwapAfterSnapshot(t *testing.T) {
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
	digest, err := pluginsign.ContentDigest(pluginDir)
	if err != nil {
		t.Fatal(err)
	}

	repo := &fakePluginInstallationRepo{inst: domain.PluginInstallation{
		Category:        "automation",
		PluginID:        "demo",
		InstanceID:      "default",
		Enabled:         true,
		SignatureStatus: domain.PluginSignatureUnsigned,
		ContentDigest:   digest,
	}}
	m := NewManager(base, repo, testCipher(t), nil)

	// Create the verified snapshot, then swap the live entry binary to an attack
	// payload, then cold-start: the live digest check must refuse to start.
	snap, err := m.verifiedSnapshot(context.Background(), repo.inst)
	if err != nil {
		t.Fatalf("verifiedSnapshot: %v", err)
	}
	if _, err := os.Stat(filepath.Join(snap, "plugin")); err != nil {
		t.Fatalf("snapshot plugin missing: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin"), []byte("#!/bin/sh\nattacker\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	_, err = m.EnsureRunning(context.Background(), "automation", "demo", "default")
	if err == nil {
		t.Fatal("expected start to be refused after the live entry binary was swapped")
	}
	if !strings.Contains(err.Error(), "content changed") {
		t.Fatalf("expected a content-change refusal, got: %v", err)
	}
}

// TestGetConfigSchemaInstanceRefusesUninstalledPlugin guards the dialCore trust
// gate (plugin_supply_chain F1): a plugin directory on disk with NO installation
// record must not be spawned via the config-schema path. dialCore resolves the
// installation record first and refuses unknown instances, so an uninstalled
// directory cannot be executed by opening its config page.
func TestGetConfigSchemaInstanceRefusesUninstalledPlugin(t *testing.T) {
	base := t.TempDir()
	// The automation category is host-managed (built-in schema, never dials the
	// plugin), so use the sms category which exercises the dialCore gate.
	writeTestPlugin(t, base, "sms", "demo", false)
	m := NewManager(base, &fakePluginInstallationRepo{}, newTestCipher(t), nil)

	if _, _, err := m.GetConfigSchemaInstance(context.Background(), "sms", "demo", "default"); err == nil {
		t.Fatal("expected config schema to be refused for an uninstalled plugin")
	}
}

// TestGetConfigSchemaInstanceRefusesTamperedPlugin guards the dialCore verify
// binding (plugin_supply_chain F1 / exec_race F4): after the entry binary on
// disk is tampered with, the config-schema path must refuse to spawn it with a
// content-change error instead of executing the modified bytes.
func TestGetConfigSchemaInstanceRefusesTamperedPlugin(t *testing.T) {
	base := t.TempDir()
	pluginDir := filepath.Join(base, "sms", "demo")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(`{"plugin_id":"demo","name":"Demo","version":"1.0.0","capabilities":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin"), []byte("#!/bin/sh\noriginal\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	digest, err := pluginsign.ContentDigest(pluginDir)
	if err != nil {
		t.Fatal(err)
	}
	repo := &fakePluginInstallationRepo{inst: domain.PluginInstallation{
		Category:        "sms",
		PluginID:        "demo",
		InstanceID:      "default",
		Enabled:         true,
		SignatureStatus: domain.PluginSignatureUnsigned,
		ContentDigest:   digest,
	}}
	m := NewManager(base, repo, newTestCipher(t), nil)

	if err := os.WriteFile(filepath.Join(pluginDir, "plugin"), []byte("#!/bin/sh\nattacker\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	if _, _, err := m.GetConfigSchemaInstance(context.Background(), "sms", "demo", "default"); err == nil {
		t.Fatal("expected config schema to be refused after the entry binary was tampered")
	} else if !strings.Contains(err.Error(), "content changed") {
		t.Fatalf("expected a content-change refusal, got: %v", err)
	}
}

// TestVerifiedSnapshotReuseRejectsTamperedSnapshot is the reuse-path regression:
// a same-user attacker who unlinks/replaces the verified snapshot between
// startups must be refused on the next start — the snapshot is re-verified, so
// attacker bytes are never silently executed from a reused path.
func TestVerifiedSnapshotReuseRejectsTamperedSnapshot(t *testing.T) {
	base := t.TempDir()
	pluginDir := filepath.Join(base, "automation", "demo")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	orig := "#!/bin/sh\noriginal\n"
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(`{"plugin_id":"demo","name":"Demo","version":"1.0.0","capabilities":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin"), []byte(orig), 0o755); err != nil {
		t.Fatal(err)
	}
	digest, err := pluginsign.ContentDigest(pluginDir)
	if err != nil {
		t.Fatal(err)
	}

	repo := &fakePluginInstallationRepo{inst: domain.PluginInstallation{
		Category:        "automation",
		PluginID:        "demo",
		InstanceID:      "default",
		Enabled:         true,
		SignatureStatus: domain.PluginSignatureUnsigned,
		ContentDigest:   digest,
	}}
	m := NewManager(base, repo, testCipher(t), nil)

	// First start materializes the verified snapshot.
	snap, err := m.verifiedSnapshot(context.Background(), repo.inst)
	if err != nil {
		t.Fatalf("verifiedSnapshot: %v", err)
	}
	if _, err := os.Stat(filepath.Join(snap, "plugin")); err != nil {
		t.Fatalf("snapshot plugin missing: %v", err)
	}

	// Same-user attacker unlinks and replaces the snapshot entry binary after the
	// snapshot was created (the snapshot dir is owner-writable so removal works).
	snapPlugin := filepath.Join(snap, "plugin")
	if err := os.Remove(snapPlugin); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(snapPlugin, []byte("#!/bin/sh\nattacker\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Reuse must recompute the digest and refuse the tampered snapshot.
	if _, err := m.verifiedSnapshot(context.Background(), repo.inst); err == nil {
		t.Fatal("expected reuse to refuse a tampered snapshot")
	} else if !strings.Contains(err.Error(), "snapshot re-verification failed") {
		t.Fatalf("expected a snapshot re-verification refusal, got: %v", err)
	}
}

// TestEnsureRunningRefusesTamperedSnapshot guards the same invariant through the
// startup path: a cold start after the verified snapshot was swapped on disk
// must refuse instead of executing the swapped bytes.
func TestEnsureRunningRefusesTamperedSnapshot(t *testing.T) {
	base := t.TempDir()
	pluginDir := filepath.Join(base, "automation", "demo")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(`{"plugin_id":"demo","name":"Demo Plugin","version":"1.0.0","capabilities":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin"), []byte("#!/bin/sh\noriginal\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	digest, err := pluginsign.ContentDigest(pluginDir)
	if err != nil {
		t.Fatal(err)
	}

	repo := &fakePluginInstallationRepo{inst: domain.PluginInstallation{
		Category:        "automation",
		PluginID:        "demo",
		InstanceID:      "default",
		Enabled:         true,
		SignatureStatus: domain.PluginSignatureUnsigned,
		ContentDigest:   digest,
	}}
	factory := &fakeClientFactory{}
	m := NewManager(base, repo, testCipher(t), nil)
	m.runtime.newClient = factory.new

	if _, err := m.EnsureRunning(context.Background(), "automation", "demo", "default"); err != nil {
		t.Fatalf("EnsureRunning: %v", err)
	}

	// The plugin is now running; simulate a process exit so the next start goes
	// through the snapshot reuse path.
	rp, ok := m.runtime.GetRunning("automation", "demo", "default")
	if !ok || rp == nil {
		t.Fatal("plugin not running after start")
	}
	rp.mu.Lock()
	if rp.client != nil {
		rp.client.Kill()
	}
	rp.mu.Unlock()
	m.runtime.remove("automation", "demo", "default", rp)

	// Swap the verified snapshot payload (the live dir is left untouched, so the
	// live-dir digest check alone would not catch this).
	snap := filepath.Join(base, snapshotRootName, "automation", "demo", digest)
	if err := os.Remove(filepath.Join(snap, "plugin")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(snap, "plugin"), []byte("#!/bin/sh\nattacker\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	_, err = m.EnsureRunning(context.Background(), "automation", "demo", "default")
	if err == nil {
		t.Fatal("expected the second startup to refuse the tampered snapshot")
	}
	if !strings.Contains(err.Error(), "snapshot re-verification failed") {
		t.Fatalf("expected a snapshot re-verification refusal, got: %v", err)
	}
}

// TestVerifiedSnapshotRenameFailureReusesOnlyVerifiedSnapshot is the
// rename-failure-path regression: a same-user attacker who plants a directory at
// the content-addressed final path between the earlier existence check and the
// os.Rename forces the rename to fail, and the fallback that would then "reuse"
// that path must re-verify it. Without the fix, the code returned the planted
// (never-verified) snapshot to the caller and the attacker bytes would be
// executed on the next start.
func TestVerifiedSnapshotRenameFailureReusesOnlyVerifiedSnapshot(t *testing.T) {
	base := t.TempDir()
	pluginDir := filepath.Join(base, "automation", "demo")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	orig := "#!/bin/sh\noriginal\n"
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(`{"plugin_id":"demo","name":"Demo","version":"1.0.0","capabilities":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin"), []byte(orig), 0o755); err != nil {
		t.Fatal(err)
	}
	digest, err := pluginsign.ContentDigest(pluginDir)
	if err != nil {
		t.Fatal(err)
	}

	repo := &fakePluginInstallationRepo{inst: domain.PluginInstallation{
		Category:        "automation",
		PluginID:        "demo",
		InstanceID:      "default",
		Enabled:         true,
		SignatureStatus: domain.PluginSignatureUnsigned,
		ContentDigest:   digest,
	}}
	m := NewManager(base, repo, testCipher(t), nil)

	// The attacker wins the race: between the earlier os.Stat(final) (which did
	// not find a snapshot) and the os.Rename, a non-empty directory is planted at
	// the content-addressed final path. The rename then fails and the fallback
	// reuse path must re-verify the planted tree and refuse it.
	final := filepath.Join(base, snapshotRootName, "automation", "demo", digest)
	m.beforeRename = func() {
		if err := os.MkdirAll(final, 0o700); err != nil {
			t.Errorf("plant snapshot dir: %v", err)
			return
		}
		if err := os.WriteFile(filepath.Join(final, "plugin"), []byte("#!/bin/sh\nattacker\n"), 0o755); err != nil {
			t.Errorf("plant snapshot payload: %v", err)
		}
	}

	if _, err := m.verifiedSnapshot(context.Background(), repo.inst); err == nil {
		t.Fatal("expected the planted (never-verified) snapshot to be refused after the rename failed")
	} else if !strings.Contains(err.Error(), "snapshot re-verification failed") {
		t.Fatalf("expected a snapshot re-verification refusal, got: %v", err)
	}
}

// TestVerifiedSnapshotRenameFailureReusesConcurrentWinner confirms the
// rename-failure fallback still serves the normal concurrent-caller case: when
// the directory at final is a valid snapshot (a concurrent caller won the race),
// re-verification passes and the path is returned.
func TestVerifiedSnapshotRenameFailureReusesConcurrentWinner(t *testing.T) {
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
	digest, err := pluginsign.ContentDigest(pluginDir)
	if err != nil {
		t.Fatal(err)
	}

	repo := &fakePluginInstallationRepo{inst: domain.PluginInstallation{
		Category:        "automation",
		PluginID:        "demo",
		InstanceID:      "default",
		Enabled:         true,
		SignatureStatus: domain.PluginSignatureUnsigned,
		ContentDigest:   digest,
	}}
	m := NewManager(base, repo, testCipher(t), nil)

	// A concurrent caller (simulated by the hook) wins the race and plants a
	// VALID snapshot tree at final right before our rename. The rename fails, and
	// the fallback re-verification must accept it: same verified bytes, so the
	// content-addressed path is safe to return.
	final := filepath.Join(base, snapshotRootName, "automation", "demo", digest)
	m.beforeRename = func() {
		if err := copyDir(pluginDir, final); err != nil {
			t.Errorf("plant valid snapshot tree: %v", err)
		}
	}
	snap, err := m.verifiedSnapshot(context.Background(), repo.inst)
	if err != nil {
		t.Fatalf("concurrent-winner reuse must be accepted after re-verification: %v", err)
	}
	if snap != final {
		t.Fatalf("expected the content-addressed path %q, got %q", final, snap)
	}
}
func TestVerifiedSnapshotReuseAcceptsUnchangedSnapshot(t *testing.T) {
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
	digest, err := pluginsign.ContentDigest(pluginDir)
	if err != nil {
		t.Fatal(err)
	}

	repo := &fakePluginInstallationRepo{inst: domain.PluginInstallation{
		Category:        "automation",
		PluginID:        "demo",
		InstanceID:      "default",
		Enabled:         true,
		SignatureStatus: domain.PluginSignatureUnsigned,
		ContentDigest:   digest,
	}}
	m := NewManager(base, repo, testCipher(t), nil)

	snap1, err := m.verifiedSnapshot(context.Background(), repo.inst)
	if err != nil {
		t.Fatalf("verifiedSnapshot: %v", err)
	}
	snap2, err := m.verifiedSnapshot(context.Background(), repo.inst)
	if err != nil {
		t.Fatalf("verifiedSnapshot reuse: %v", err)
	}
	if snap1 != snap2 {
		t.Fatalf("reuse returned a different snapshot path: %q vs %q", snap1, snap2)
	}
}

// TestSnapshotsRemovedOnDelete guards cleanup: removing the last instance also
// removes the plugin's verified snapshots so they do not accumulate.
func TestSnapshotsRemovedOnDelete(t *testing.T) {
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
	digest, err := pluginsign.ContentDigest(pluginDir)
	if err != nil {
		t.Fatal(err)
	}

	repo := &fakePluginInstallationRepo{inst: domain.PluginInstallation{
		Category:        "automation",
		PluginID:        "demo",
		InstanceID:      "default",
		Enabled:         true,
		SignatureStatus: domain.PluginSignatureUnsigned,
		ContentDigest:   digest,
	}}
	m := NewManager(base, repo, testCipher(t), nil)
	if _, err := m.verifiedSnapshot(context.Background(), repo.inst); err != nil {
		t.Fatalf("verifiedSnapshot: %v", err)
	}
	snapDir := filepath.Join(base, snapshotRootName, "automation", "demo")
	if _, err := os.Stat(snapDir); err != nil {
		t.Fatalf("snapshot dir missing: %v", err)
	}
	m.removeSnapshots("automation", "demo")
	if _, err := os.Stat(snapDir); !os.IsNotExist(err) {
		t.Fatalf("snapshot dir still present after removal: %v", err)
	}
}
