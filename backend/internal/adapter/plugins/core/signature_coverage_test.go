package plugins

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/pkg/pluginsign"
)

// TestVerifySignatureRejectsUncoveredFile guards the full-coverage requirement
// (m066/m074 F-line residual): a plugin can load schema, template, script or
// shared-library payload in addition to bin/** and manifest.json. Any regular
// file that is not listed in checksums.json must make verification fail, or an
// attacker could add a malicious payload without breaking the signature.
func TestVerifySignatureRejectsUncoveredFile(t *testing.T) {
	platform := CurrentPlatformKey()
	dir := writePluginDir(t, map[string]string{
		"manifest.json":               `{"plugin_id":"demo","name":"Demo","version":"1.0.0","binaries":{"` + platform + `":"bin/` + platform + `/plugin"}}`,
		"bin/" + platform + "/plugin": "#!/bin/sh\nexit 0\n",
		"schemas/config.schema.json":  `{"type":"object"}`,
	})
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	// Checksums cover the manifest and binary but omit the schema payload.
	writePluginChecksums(t, dir, []string{
		"manifest.json",
		"bin/" + platform + "/plugin",
	}, priv)

	status, err := VerifySignature(dir, []ed25519.PublicKey{pub})
	if err == nil {
		t.Fatal("expected an error for an uncovered payload file")
	}
	if status != domain.PluginSignatureUntrusted {
		t.Fatalf("expected untrusted when a payload file is not checksummed, got %q", status)
	}
	if !strings.Contains(err.Error(), "schemas/config.schema.json") {
		t.Fatalf("error should name the uncovered file, got: %v", err)
	}
}

// TestVerifySignatureRejectsExtraChecksum guards the reverse direction: a
// checksums.json entry that does not correspond to a real regular file (a stale
// or planted digest) must be rejected rather than ignored.
func TestVerifySignatureRejectsExtraChecksum(t *testing.T) {
	platform := CurrentPlatformKey()
	dir := writePluginDir(t, map[string]string{
		"manifest.json":               `{"plugin_id":"demo","name":"Demo","version":"1.0.0","binaries":{"` + platform + `":"bin/` + platform + `/plugin"}}`,
		"bin/" + platform + "/plugin": "#!/bin/sh\nexit 0\n",
	})
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	// The map lists a file that does not exist on disk ("ghost.txt").
	cs := Checksums{Algo: "sha256", Files: map[string]string{
		"manifest.json":               mustFileSum(t, filepath.Join(dir, "manifest.json")),
		"bin/" + platform + "/plugin": mustFileSum(t, filepath.Join(dir, "bin", platform, "plugin")),
		"ghost.txt":                   zeros64,
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
		t.Fatal("expected an error for a checksum referencing a non-existent file")
	}
	if status != domain.PluginSignatureUntrusted {
		t.Fatalf("expected untrusted for an extra checksum entry, got %q", status)
	}
}

// TestVerifySignatureRejectsSymlink guards the non-regular-file rejection: a
// symlink inside the plugin directory could point outside the package and be
// executed/loaded as payload that the walk would otherwise not follow.
func TestVerifySignatureRejectsSymlink(t *testing.T) {
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

	if err := os.Symlink("/etc/passwd", filepath.Join(dir, "evil.so")); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}

	status, err := VerifySignature(dir, []ed25519.PublicKey{pub})
	if err == nil {
		t.Fatal("expected an error for a symlink inside the plugin dir")
	}
	if status != domain.PluginSignatureUntrusted {
		t.Fatalf("expected untrusted when a symlink is present, got %q", status)
	}
}

// TestVerifySignatureRejectsHardlink guards the hardlink rejection on platforms
// where link counts are observable: two directory entries sharing one inode can
// smuggle an extra path past a content-only walk.
func TestVerifySignatureRejectsHardlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("hardlink link-count is not observable the same way on windows")
	}
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

	if err := os.Link(filepath.Join(dir, "manifest.json"), filepath.Join(dir, "manifest_alias.json")); err != nil {
		t.Skipf("cannot create hardlink: %v", err)
	}

	status, err := VerifySignature(dir, []ed25519.PublicKey{pub})
	if err == nil {
		t.Fatal("expected an error for a hardlinked file inside the plugin dir")
	}
	if status != domain.PluginSignatureUntrusted {
		t.Fatalf("expected untrusted when a hardlink is present, got %q", status)
	}
}

// TestVerifySignatureRejectsBackslashChecksumPath guards path-normalization
// collisions: a checksum key containing a backslash must not be silently
// normalized into a real file's path.
func TestVerifySignatureRejectsBackslashChecksumPath(t *testing.T) {
	platform := CurrentPlatformKey()
	dir := writePluginDir(t, map[string]string{
		"manifest.json":               `{"plugin_id":"demo","name":"Demo","version":"1.0.0","binaries":{"` + platform + `":"bin/` + platform + `/plugin"}}`,
		"bin/" + platform + "/plugin": "#!/bin/sh\nexit 0\n",
	})
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	cs := Checksums{Algo: "sha256", Files: map[string]string{
		"manifest.json":               mustFileSum(t, filepath.Join(dir, "manifest.json")),
		"bin/" + platform + "/plugin": mustFileSum(t, filepath.Join(dir, "bin", platform, "plugin")),
		`bin\` + platform + `\plugin`: mustFileSum(t, filepath.Join(dir, "bin", platform, "plugin")),
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
		t.Fatal("expected an error for a backslash checksum path")
	}
	if status != domain.PluginSignatureUntrusted {
		t.Fatalf("expected untrusted for a backslash checksum path, got %q", status)
	}
}

// TestSignerOutputVerifiesWithServer proves the signer and the verifier share
// one canonical manifest: whatever the signing tool writes into checksums.json
// is accepted by VerifySignature.
func TestSignerOutputVerifiesWithServer(t *testing.T) {
	platform := CurrentPlatformKey()
	dir := writePluginDir(t, map[string]string{
		"manifest.json":               `{"plugin_id":"demo","name":"Demo","version":"1.0.0","binaries":{"` + platform + `":"bin/` + platform + `/plugin"}}`,
		"bin/" + platform + "/plugin": "#!/bin/sh\nexit 0\n",
		"templates/notify.txt":        "hello",
		"schemas/config.schema.json":  `{"type":"object"}`,
	})

	// Replicate exactly what cmd/tools/pluginsign writes.
	entries, err := pluginsign.CanonicalManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	files := pluginsign.FilesMap(entries)
	cs := Checksums{Algo: "sha256", Files: files}
	csBytes, err := json.Marshal(cs)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "checksums.json"), csBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "signature.sig"), ed25519.Sign(priv, csBytes), 0o644); err != nil {
		t.Fatal(err)
	}

	status, err := VerifySignature(dir, []ed25519.PublicKey{pub})
	if err != nil {
		t.Fatalf("VerifySignature rejected the signer's own output: %v", err)
	}
	if status != domain.PluginSignatureOfficial {
		t.Fatalf("expected official status for signer output, got %q", status)
	}
}

func mustFileSum(t *testing.T, path string) string {
	t.Helper()
	sum, err := sha256File(path)
	if err != nil {
		t.Fatal(err)
	}
	return sum
}
