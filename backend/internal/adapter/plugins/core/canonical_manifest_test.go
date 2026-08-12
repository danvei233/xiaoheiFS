package plugins

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/pkg/pluginsign"
)

// TestCanonicalManifestCoversNestedBookkeepingPayload guards the root-only
// exclusion rule: only the plugin-root checksums.json/signature.sig ledger files
// are excluded from the signed payload. A same-named file deeper in the tree is
// ordinary payload and must appear in the canonical manifest (and therefore in
// the checksums and the persisted content digest).
func TestCanonicalManifestCoversNestedBookkeepingPayload(t *testing.T) {
	dir := writePluginDir(t, map[string]string{
		"manifest.json":           `{"plugin_id":"demo","name":"Demo","version":"1.0.0","capabilities":{}}`,
		"sub/checksums.json":      `{"nested":true}`,
		"sub/signature.sig":       "nested-ledger-payload",
		"sub/another/checksums.json": "deep",
	})
	entries, err := pluginsign.CanonicalManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	rels := map[string]bool{}
	for _, e := range entries {
		rels[e.Rel] = true
	}
	for _, want := range []string{"manifest.json", "sub/checksums.json", "sub/signature.sig", "sub/another/checksums.json"} {
		if !rels[want] {
			t.Errorf("canonical manifest is missing nested payload %q; entries=%v", want, rels)
		}
	}
}

// TestVerifySignatureRejectsTamperedNestedBookkeepingPayload is the tamper
// regression for the nested same-name case: an attacker adding or replacing a
// nested checksums.json/signature.sig that is not part of the signed checksum
// set must be refused, because that payload could be loaded by the plugin.
func TestVerifySignatureRejectsTamperedNestedBookkeepingPayload(t *testing.T) {
	platform := CurrentPlatformKey()
	dir := writePluginDir(t, map[string]string{
		"manifest.json":               `{"plugin_id":"demo","name":"Demo","version":"1.0.0","binaries":{"` + platform + `":"bin/` + platform + `/plugin"}}`,
		"bin/" + platform + "/plugin": "#!/bin/sh\nexit 0\n",
	})
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	writePluginChecksums(t, dir, []string{
		"manifest.json",
		"bin/" + platform + "/plugin",
	}, priv)

	// Attacker plants a nested checksums.json payload that is not covered.
	if err := os.MkdirAll(filepath.Join(dir, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "lib", "checksums.json"), []byte(`{"evil":true}`), 0o644); err != nil {
		t.Fatal(err)
	}

	status, err := VerifySignature(dir, []ed25519.PublicKey{pub})
	if err == nil {
		t.Fatal("expected an error for a tampered nested checksums.json payload")
	}
	if status != domain.PluginSignatureUntrusted {
		t.Fatalf("expected untrusted for a nested checksums.json payload, got %q", status)
	}
	if !strings.Contains(err.Error(), "lib/checksums.json") {
		t.Fatalf("error should name the uncovered nested file, got: %v", err)
	}
}

// TestCanonicalManifestRejectsCaseCollidingPaths guards cross-platform aliasing:
// two paths that differ only by letter case are distinct on Linux but collide on
// case-insensitive filesystems (Windows NTFS, macOS HFS+/APFS). The canonical
// manifest must reject the package up front so a signature made on Linux cannot
// be re-interpreted on Windows where one file shadows the other.
func TestCanonicalManifestRejectsCaseCollidingPaths(t *testing.T) {
	dir := writePluginDir(t, map[string]string{
		"manifest.json":    `{"plugin_id":"demo","name":"Demo","version":"1.0.0","capabilities":{}}`,
		"lib/Helper.txt":   "one",
		"lib/helper.txt":   "two",
	})
	_, err := pluginsign.CanonicalManifest(dir)
	if err == nil {
		t.Fatal("expected a case-collision rejection")
	}
	if !strings.Contains(err.Error(), "case-colliding") {
		t.Fatalf("expected a case-collision error, got: %v", err)
	}
}

// TestCanonicalManifestRejectsWindowsPathNormalizationCollisions guards the
// trailing-dot/space folding of Windows: "lib/helper." and "lib/helper" (or
// "lib/helper..txt" after trimming) would alias on NTFS. The canonical manifest
// must reject the package so a manifest verified on Linux cannot be
// re-interpreted on Windows where one path shadows the other.
func TestCanonicalManifestRejectsWindowsPathNormalizationCollisions(t *testing.T) {
	dir := writePluginDir(t, map[string]string{
		"manifest.json": `{"plugin_id":"demo","name":"Demo","version":"1.0.0","capabilities":{}}`,
		"lib/helper":    "one",
		"lib/helper.":   "two",
	})
	_, err := pluginsign.CanonicalManifest(dir)
	if err == nil {
		t.Fatal("expected a windows path-normalization collision rejection")
	}
	if !strings.Contains(err.Error(), "case-colliding") {
		t.Fatalf("expected a case-collision error, got: %v", err)
	}
}

// TestCanonicalManifestRejectsBackslashPath guards a Linux-signed path that
// would mean a different file layout on Windows: a literal backslash in a
// filename is a separator on Windows, so the package must be rejected up front
// rather than verified one way and extracted another.
func TestCanonicalManifestRejectsBackslashPath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{"plugin_id":"demo","name":"Demo","version":"1.0.0","capabilities":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	// A single file whose name literally contains a backslash (legal on Linux).
	if err := os.WriteFile(filepath.Join(dir, `weird\name.txt`), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := pluginsign.CanonicalManifest(dir)
	if err == nil {
		t.Fatal("expected a backslash-path rejection")
	}
	if !strings.Contains(err.Error(), "backslash") {
		t.Fatalf("expected a backslash-separator error, got: %v", err)
	}
}

// TestVerifySignatureRejectsCaseCollidingChecksums guards the verifier against a
// checksums.json that lists two keys differing only by case for one real file:
// on a case-insensitive filesystem both keys would resolve to the same file and
// the signature would be ambiguous.
func TestVerifySignatureRejectsCaseCollidingChecksums(t *testing.T) {
	platform := CurrentPlatformKey()
	dir := writePluginDir(t, map[string]string{
		"manifest.json":               `{"plugin_id":"demo","name":"Demo","version":"1.0.0","binaries":{"` + platform + `":"bin/` + platform + `/plugin"}}`,
		"bin/" + platform + "/plugin": "#!/bin/sh\nexit 0\n",
	})
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	sum := mustFileSum(t, filepath.Join(dir, "manifest.json"))
	cs := Checksums{Algo: "sha256", Files: map[string]string{
		"manifest.json":               sum,
		"Manifest.JSON":               sum,
		"bin/" + platform + "/plugin": mustFileSum(t, filepath.Join(dir, "bin", platform, "plugin")),
	}}
	csBytes, err := json.Marshal(cs)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "checksums.json"), csBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "signature.sig"), ed25519.Sign(priv, csBytes), 0o644); err != nil {
		t.Fatal(err)
	}

	status, err := VerifySignature(dir, []ed25519.PublicKey{pub})
	if err == nil {
		t.Fatal("expected an error for case-colliding checksum keys")
	}
	if status != domain.PluginSignatureUntrusted {
		t.Fatalf("expected untrusted for case-colliding checksum keys, got %q", status)
	}
}

// TestContentDigestCoversNestedBookkeepingPayload guards the persisted digest
// anchor: replacing a nested checksums.json/signature.sig payload must change the
// canonical content digest so a start-time verifyUntampered refuses the swap.
func TestContentDigestCoversNestedBookkeepingPayload(t *testing.T) {
	dir := writePluginDir(t, map[string]string{
		"manifest.json":      `{"plugin_id":"demo","name":"Demo","version":"1.0.0","capabilities":{}}`,
		"lib/checksums.json": `{"v":1}`,
	})
	before, err := pluginsign.ContentDigest(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "lib", "checksums.json"), []byte(`{"v":2}`), 0o644); err != nil {
		t.Fatal(err)
	}
	after, err := pluginsign.ContentDigest(dir)
	if err != nil {
		t.Fatal(err)
	}
	if after == before {
		t.Fatal("content digest did not change when a nested checksums.json payload was replaced")
	}
}
