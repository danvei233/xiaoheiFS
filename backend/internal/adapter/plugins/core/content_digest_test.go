package plugins

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/pkg/pluginsign"
)

// TestEnsureRunningRefusesContentTampering guards the tamper-anchor behavior:
// an unsigned/untrusted plugin (no official keys configured) must still be
// refused to start if its on-disk files were modified after install, because
// the recorded content digest no longer matches.
func TestEnsureRunningRefusesContentTampering(t *testing.T) {
	base := t.TempDir()
	pluginDir := filepath.Join(base, "automation", "demo")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(`{"plugin_id":"demo","name":"Demo","version":"1.0.0","capabilities":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
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
	m := NewManager(base, repo, newTestCipher(t), nil)

	// Tamper with a payload file: append a malicious line to the entry binary.
	f, err := os.OpenFile(filepath.Join(pluginDir, "plugin"), os.O_APPEND|os.O_WRONLY, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("# attacker\n"); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	if _, startErr := m.EnsureRunning(context.Background(), "automation", "demo", "default"); startErr == nil {
		t.Fatal("expected start to be refused after on-disk content was modified")
	} else if !strings.Contains(startErr.Error(), "content changed") {
		t.Fatalf("expected a content-change refusal, got: %v", startErr)
	}
}

// TestEnsureRunningRefusesTamperedDeadProcess guards the stale-running-entry
// window (exec_race F2): a plugin whose process has exited but whose entry still
// sits in the running map must NOT skip the tamper anchors. Previously
// verifyUntampered returned early for ANY running entry, so replacing the live
// binary while a dead entry lingered would respawn the tampered payload on the
// next EnsureRunning. The entry must be treated as "not running" once the client
// has exited, forcing the digest/signature re-check before any respawn.
func TestEnsureRunningRefusesTamperedDeadProcess(t *testing.T) {
	base := t.TempDir()
	writeTestPlugin(t, base, "automation", "demo", false)
	digest, err := pluginsign.ContentDigest(filepath.Join(base, "automation", "demo"))
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
	m := NewManager(base, repo, newTestCipher(t), nil)
	m.runtime.newClient = factory.new

	// Successful start: the plugin is running.
	if _, err := m.EnsureRunning(context.Background(), "automation", "demo", "default"); err != nil {
		t.Fatalf("initial start: %v", err)
	}
	if got := factory.spawnCount(); got != 1 {
		t.Fatalf("want 1 spawn, got %d", got)
	}

	// The plugin process dies underneath the runtime, leaving a stale running entry.
	factory.handles[0].exited.Store(true)

	// An attacker swaps the live entry binary for a tampered payload.
	if err := os.WriteFile(filepath.Join(base, "automation", "demo", "plugin"), []byte("#!/bin/sh\nattacker\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	// EnsureRunning must refuse with a content-change error instead of respawning
	// the tampered binary.
	if _, err := m.EnsureRunning(context.Background(), "automation", "demo", "default"); err == nil {
		t.Fatal("expected refusal after the dead process was replaced by tampered content")
	} else if !strings.Contains(err.Error(), "content changed") {
		t.Fatalf("expected a content-change refusal, got: %v", err)
	}
	if got := factory.spawnCount(); got != 1 {
		t.Fatalf("tampered dead process was respawned: %d spawns", got)
	}
}

// TestEnsureRunningRefusesWhenDigestMissing guards fail-closed behavior: an
// installation record without a recorded content digest cannot be proven
// untampered, so starting it must be refused.
func TestEnsureRunningRefusesWhenDigestMissing(t *testing.T) {
	base := t.TempDir()
	pluginDir := filepath.Join(base, "automation", "demo")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(`{"plugin_id":"demo","name":"Demo","version":"1.0.0","capabilities":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	repo := &fakePluginInstallationRepo{inst: domain.PluginInstallation{
		Category:        "automation",
		PluginID:        "demo",
		InstanceID:      "default",
		Enabled:         true,
		SignatureStatus: domain.PluginSignatureUnsigned,
		ContentDigest:   "",
	}}
	m := NewManager(base, repo, newTestCipher(t), nil)

	_, err := m.EnsureRunning(context.Background(), "automation", "demo", "default")
	if err == nil {
		t.Fatal("expected start to be refused when no content digest is recorded")
	}
	if !strings.Contains(err.Error(), "no recorded content digest") {
		t.Fatalf("expected a missing-digest refusal, got: %v", err)
	}
}

// TestInstallRecordsContentDigest verifies Install persists the canonical
// content digest of the freshly installed files.
func TestInstallRecordsContentDigest(t *testing.T) {
	platform := CurrentPlatformKey()
	pkg := buildTestZip(t, map[string]string{
		"plugins/automation/demo/manifest.json":               demoManifestJSON(platform),
		"plugins/automation/demo/bin/" + platform + "/plugin": "binary-content",
	})
	baseDir := t.TempDir()
	repo := &fakePluginInstallationRepo{}
	m := NewManager(baseDir, repo, testCipher(t), nil)

	inst, err := m.Install(context.Background(), "demo.zip", bytes.NewReader(pkg))
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if inst.ContentDigest == "" {
		t.Fatal("Install did not record a content digest")
	}
	// Recompute from the installed directory; must match.
	got, err := pluginsign.ContentDigest(filepath.Join(baseDir, "automation", "demo"))
	if err != nil {
		t.Fatal(err)
	}
	if got != inst.ContentDigest {
		t.Fatalf("recorded digest %q does not match installed content digest %q", inst.ContentDigest, got)
	}
}
