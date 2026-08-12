package plugins

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"xiaoheiplay/internal/domain"
)

// writePluginDir writes a set of files (rel -> content) into a fresh temp dir.
func writePluginDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range files {
		full := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// writePluginChecksums writes checksums.json covering exactly the given rel
// paths and, when priv is non-nil, a signature.sig over the checksum bytes.
func writePluginChecksums(t *testing.T, dir string, rels []string, priv ed25519.PrivateKey) {
	t.Helper()
	cs := Checksums{Algo: "sha256", Files: map[string]string{}}
	for _, rel := range rels {
		sum, err := sha256File(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		cs.Files[rel] = sum
	}
	csBytes, err := json.Marshal(cs)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "checksums.json"), csBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	if priv != nil {
		if err := os.WriteFile(filepath.Join(dir, "signature.sig"), ed25519.Sign(priv, csBytes), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// TestVerifySignatureRawSignatureWithWhitespaceByte guards against trimming
// raw ed25519 signature bytes. A valid raw signature can legitimately start
// with an ASCII whitespace octet; trimming corrupts it and mislabels an
// official plugin as untrusted.
func TestVerifySignatureRawSignatureWithWhitespaceByte(t *testing.T) {
	platform := CurrentPlatformKey()
	dir := writePluginDir(t, map[string]string{
		"manifest.json":               `{"plugin_id":"demo","name":"Demo","version":"1.0.0","binaries":{"` + platform + `":"bin/` + platform + `/plugin"}}`,
		"bin/" + platform + "/plugin": "#!/bin/sh\nexit 0\n",
	})
	writePluginChecksums(t, dir, []string{
		"manifest.json",
		"bin/" + platform + "/plugin",
	}, nil)

	csBytes, err := os.ReadFile(filepath.Join(dir, "checksums.json"))
	if err != nil {
		t.Fatal(err)
	}

	var pub ed25519.PublicKey
	var sig []byte
	found := false
	for i := 0; i < 5000; i++ {
		p, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		s := ed25519.Sign(priv, csBytes)
		if isASCIIWhitespaceByte(s[0]) {
			pub, sig = p, s
			found = true
			break
		}
	}
	if !found {
		t.Skip("could not generate a signature starting with a whitespace byte")
	}
	if err := os.WriteFile(filepath.Join(dir, "signature.sig"), sig, 0o644); err != nil {
		t.Fatal(err)
	}

	status, err := VerifySignature(dir, []ed25519.PublicKey{pub})
	if err != nil {
		t.Fatalf("VerifySignature returned error: %v", err)
	}
	if status != domain.PluginSignatureOfficial {
		t.Fatalf("expected official signature status, got %q", status)
	}
}

func isASCIIWhitespaceByte(b byte) bool {
	switch b {
	case '\t', '\n', '\v', '\f', '\r', ' ':
		return true
	}
	return false
}

// TestVerifySignatureBase64Signature ensures base64-encoded signatures still
// verify after the raw-signature parsing change.
func TestVerifySignatureBase64Signature(t *testing.T) {
	platform := CurrentPlatformKey()
	dir := writePluginDir(t, map[string]string{
		"manifest.json":               `{"plugin_id":"demo","name":"Demo","version":"1.0.0","binaries":{"` + platform + `":"bin/` + platform + `/plugin"}}`,
		"bin/" + platform + "/plugin": "#!/bin/sh\nexit 0\n",
	})
	writePluginChecksums(t, dir, []string{
		"manifest.json",
		"bin/" + platform + "/plugin",
	}, nil)

	csBytes, err := os.ReadFile(filepath.Join(dir, "checksums.json"))
	if err != nil {
		t.Fatal(err)
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	sig := ed25519.Sign(priv, csBytes)
	if err := os.WriteFile(filepath.Join(dir, "signature.sig"), []byte(base64.StdEncoding.EncodeToString(sig)), 0o644); err != nil {
		t.Fatal(err)
	}

	status, err := VerifySignature(dir, []ed25519.PublicKey{pub})
	if err != nil {
		t.Fatalf("VerifySignature returned error: %v", err)
	}
	if status != domain.PluginSignatureOfficial {
		t.Fatalf("expected official signature status, got %q", status)
	}
}

// TestVerifySignatureRequiresManifestChecksum verifies that manifest.json must
// be part of the signed checksums; otherwise its identity/capabilities could be
// altered without invalidating the signature.
func TestVerifySignatureRequiresManifestChecksum(t *testing.T) {
	platform := CurrentPlatformKey()
	dir := writePluginDir(t, map[string]string{
		"manifest.json":               `{"plugin_id":"demo","name":"Demo","version":"1.0.0","binaries":{"` + platform + `":"bin/` + platform + `/plugin"}}`,
		"bin/" + platform + "/plugin": "#!/bin/sh\nexit 0\n",
	})
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	// Checksums deliberately omit manifest.json.
	writePluginChecksums(t, dir, []string{
		"bin/" + platform + "/plugin",
	}, priv)

	status, err := VerifySignature(dir, []ed25519.PublicKey{pub})
	if err == nil {
		t.Fatal("expected an error describing the missing manifest.json checksum")
	}
	if status != domain.PluginSignatureUntrusted {
		t.Fatalf("expected untrusted when manifest.json is not checksummed, got %q", status)
	}
}

// TestVerifySignatureRequiresEntryCoverage verifies the legacy root-fallback
// executable is covered by checksums: the binary that actually gets executed
// must be part of the signed payload.
func TestVerifySignatureRequiresEntryCoverage(t *testing.T) {
	dir := writePluginDir(t, map[string]string{
		"manifest.json": `{"plugin_id":"demo","name":"Demo","version":"1.0.0"}`,
		"plugin":        "#!/bin/sh\nexit 0\n",
	})
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	// Checksums cover manifest.json but not the root entry binary.
	writePluginChecksums(t, dir, []string{
		"manifest.json",
	}, priv)

	status, err := VerifySignature(dir, []ed25519.PublicKey{pub})
	if err == nil {
		t.Fatal("expected an error describing the missing entry checksum")
	}
	if status != domain.PluginSignatureUntrusted {
		t.Fatalf("expected untrusted when entry binary is not checksummed, got %q", status)
	}
}

// buildTestChecksums mirrors backend/cmd/tools/pluginsign.buildChecksums so the
// test signs a plugin exactly like the release tooling does (P2-65).
func buildTestChecksums(t *testing.T, dir string) []byte {
	t.Helper()
	cs := Checksums{Algo: "sha256", Files: map[string]string{}}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if strings.EqualFold(name, "signature.sig") || strings.EqualFold(name, "checksums.json") {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		sum, err := sha256File(path)
		if err != nil {
			return err
		}
		cs.Files[filepath.ToSlash(rel)] = sum
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	b, err := json.MarshalIndent(cs, "", "  ")
	if err != nil {
		t.Fatalf("marshal checksums: %v", err)
	}
	return b
}

func writeSignedPlugin(t *testing.T, dir string, priv ed25519.PrivateKey) {
	t.Helper()
	_ = os.MkdirAll(filepath.Join(dir, "bin", "linux_amd64"), 0o755)
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{"plugin_id":"p1","name":"P1","version":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bin", "linux_amd64", "plugin"), []byte("binary-bytes"), 0o755); err != nil {
		t.Fatal(err)
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

// TestSignAndVerifyPlugin pins the release signing contract: a plugin signed by
// the release tooling is accepted as Official by the host verifier.
func TestSignAndVerifyPlugin(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	writeSignedPlugin(t, dir, priv)

	status, err := VerifySignature(dir, []ed25519.PublicKey{pub})
	if err != nil {
		t.Fatalf("VerifySignature err: %v", err)
	}
	if status != domain.PluginSignatureOfficial {
		t.Fatalf("signature status = %v, want Official", status)
	}

	// Tampering with the binary must flip to Untrusted.
	binPath := filepath.Join(dir, "bin", "linux_amd64", "plugin")
	_ = os.WriteFile(binPath, []byte("tampered"), 0o755)
	status, err = VerifySignature(dir, []ed25519.PublicKey{pub})
	if err == nil {
		t.Fatal("expected a checksum-mismatch error after tampering")
	}
	if status != domain.PluginSignatureUntrusted {
		t.Fatalf("tampered signature status = %v, want Untrusted", status)
	}
}

// TestVerifySignatureUnsignedWhenNoChecksums pins the fail-closed baseline: a
// plugin with no checksums.json/signature.sig is Unsigned, not Official, so the
// host refuses to auto-import it on image upgrade.
func TestVerifySignatureUnsignedWhenNoChecksums(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{"plugin_id":"p1"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	status, err := VerifySignature(dir, nil)
	if err != nil {
		t.Fatalf("VerifySignature err: %v", err)
	}
	if status != domain.PluginSignatureUnsigned {
		t.Fatalf("signature status = %v, want Unsigned", status)
	}
}
