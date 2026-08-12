package plugins

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/pkg/pluginsign"
)

// TestEnsureRunningRefusesSignatureMismatch verifies the runtime re-check
// (P0-11/S1): if the on-disk plugin no longer verifies as the signature status
// recorded at install time, starting it is refused before any execution side
// effect. The plugin is signed as OFFICIAL with a real trusted key, then the
// signature is re-issued by a DIFFERENT key (or dropped), so the recomputed
// status diverges while the content digest stays unchanged (the ledger files are
// excluded from the digest) — isolating the signature gate from the digest gate.
func TestEnsureRunningRefusesSignatureMismatch(t *testing.T) {
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

	// Sign the plugin with a real key: installed status = Official.
	trustedPub, trustedPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	writePluginChecksums(t, pluginDir, []string{"manifest.json", "plugin"}, trustedPriv)

	digest, err := pluginsign.ContentDigest(pluginDir)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := VerifySignature(pluginDir, []ed25519.PublicKey{trustedPub}); err != nil || got != domain.PluginSignatureOfficial {
		t.Fatalf("precondition: expected official signature, got %q err=%v", got, err)
	}

	repo := &fakePluginInstallationRepo{inst: domain.PluginInstallation{
		Category:        "automation",
		PluginID:        "demo",
		InstanceID:      "default",
		Enabled:         true,
		SignatureStatus: domain.PluginSignatureOfficial,
		ContentDigest:   digest,
	}}
	m := NewManager(base, repo, newTestCipher(t), []ed25519.PublicKey{trustedPub})

	// Re-issue the signature with a DIFFERENT key. The ledger files are excluded
	// from the content digest, so digest stays equal while the signature status
	// flips to Untrusted.
	otherPub, otherPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	_ = otherPub
	writePluginChecksums(t, pluginDir, []string{"manifest.json", "plugin"}, otherPriv)

	if status, _ := VerifySignature(pluginDir, []ed25519.PublicKey{trustedPub}); status != domain.PluginSignatureUntrusted {
		t.Fatalf("precondition: expected untrusted after re-signing with another key, got %q", status)
	}

	_, startErr := m.EnsureRunning(context.Background(), "automation", "demo", "default")
	if startErr == nil {
		t.Fatal("expected start to be refused when on-disk signature mismatches installed status")
	}
	if !strings.Contains(startErr.Error(), "signature changed") {
		t.Fatalf("expected a signature-change refusal, got: %v", startErr)
	}
}

// TestEnsureRunningAllowsMatchingUnsigned verifies the matching-signature path is
// not refused: a plugin recorded as "unsigned" whose on-disk state also verifies
// as "unsigned" is still allowed (the failure above is specifically about the
// status diverging, not about unsigned plugins being banned).
func TestEnsureRunningAllowsMatchingUnsigned(t *testing.T) {
	base := t.TempDir()
	pluginDir := filepath.Join(base, "automation", "demo")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(`{"plugin_id":"demo","name":"Demo","version":"1.0.0"}`), 0o644); err != nil {
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

	// The manifest declares no binaries, so runtime.Start fails at ResolveEntry
	// with "plugin binary not found" — but crucially NOT with a signature-mismatch
	// refusal. This proves the signature gate let the matching-status plugin
	// through to the resolution stage.
	_, err = m.EnsureRunning(context.Background(), "automation", "demo", "default")
	if err == nil {
		t.Fatal("expected entry resolution error (binary missing)")
	}
	if strings.Contains(err.Error(), "signature changed") {
		t.Fatalf("signature gate refused a matching unsigned plugin: %v", err)
	}
}
