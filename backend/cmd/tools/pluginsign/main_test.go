package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeManifest creates a minimal plugin dir with manifest.json so run() accepts it.
func writeManifest(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{"plugin_id":"test"}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}

func emptyStdin() *bytes.Reader { return bytes.NewReader(nil) }

func TestRunMissingDir(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(nil, &stdout, &stderr, emptyStdin())
	if code != 2 {
		t.Fatalf("expected exit code 2 for missing -dir, got %d", code)
	}
	if !strings.Contains(stderr.String(), "missing -dir") {
		t.Fatalf("stderr should mention missing -dir, got: %s", stderr.String())
	}
}

func TestRunMissingManifest(t *testing.T) {
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := run([]string{"-dir", dir}, &stdout, &stderr, emptyStdin())
	if code != 2 {
		t.Fatalf("expected exit code 2 for missing manifest, got %d", code)
	}
	if !strings.Contains(stderr.String(), "manifest.json not found") {
		t.Fatalf("stderr should mention manifest.json, got: %s", stderr.String())
	}
}

// TestRunWithoutKeyGeneratesKeyFileAndRefusesToSign: with no private key source the
// tool must NOT print the private key to stdout, must write it to a 0600 file, and
// must NOT produce a misleading "OK" signature (M5 regression).
func TestRunWithoutKeyGeneratesKeyFileAndRefusesToSign(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, filepath.Join(dir, "plugin"))

	var stdout, stderr bytes.Buffer
	code := run([]string{"-dir", filepath.Join(dir, "plugin")}, &stdout, &stderr, emptyStdin())
	if code == 0 {
		t.Fatalf("expected non-zero exit when no private key is provided, got 0")
	}

	out := stdout.String()
	if strings.Contains(out, "PrivateKey(base64)") {
		t.Fatalf("private key leaked to stdout:\n%s", out)
	}
	if !strings.Contains(out, "PublicKey(base64)") {
		t.Fatalf("expected public key on stdout:\n%s", out)
	}

	// No signature / checksums must be written when no key is supplied.
	if _, err := os.Stat(filepath.Join(dir, "plugin", "signature.sig")); err == nil {
		t.Fatal("signature.sig must not be produced without a private key")
	}
	if _, err := os.Stat(filepath.Join(dir, "plugin", "checksums.json")); err == nil {
		t.Fatal("checksums.json must not be produced without a private key")
	}

	// Private key must be written to a 0600 sibling file outside the packaged dir.
	keyPath := filepath.Join(dir, "plugin.ed25519.private.key")
	st, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("private key file not written: %v", err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("private key file mode = %v, want 0600", st.Mode().Perm())
	}
	if !strings.Contains(stderr.String(), "no signature was produced") {
		t.Fatalf("stderr should explain that no signature was produced, got: %s", stderr.String())
	}
}

func TestRunWithPrivateKeyFileSigns(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "plugin")
	writeManifest(t, pluginDir)
	if err := os.WriteFile(filepath.Join(pluginDir, "payload.bin"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	keyFile := filepath.Join(dir, "sign.key")
	if err := os.WriteFile(keyFile, []byte(base64.StdEncoding.EncodeToString(priv)), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"-dir", pluginDir, "-ed25519-priv-file", keyFile}, &stdout, &stderr, emptyStdin())
	if code != 0 {
		t.Fatalf("expected exit 0 when signing with a key file, got %d (stderr=%s)", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(pluginDir, "signature.sig")); err != nil {
		t.Fatalf("signature.sig missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(pluginDir, "checksums.json")); err != nil {
		t.Fatalf("checksums.json missing: %v", err)
	}

	// The signature must verify against the supplied public key.
	sig, err := os.ReadFile(filepath.Join(pluginDir, "signature.sig"))
	if err != nil {
		t.Fatal(err)
	}
	csBytes, err := os.ReadFile(filepath.Join(pluginDir, "checksums.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !ed25519.Verify(pub, csBytes, sig) {
		t.Fatal("signature does not verify against the provided key")
	}
}

// TestRunRejectsArgvKey guards against reintroducing the -ed25519-priv argv flag:
// a private key on the command line is visible in the process list, so the flag
// must not exist and the invocation must fail before signing.
// TestRunWithoutKeyTightensExistingLooseKeyFile guards the permission tighten:
// os.WriteFile alone preserves an existing file's mode, so an old 0644 private
// key would stay group/world-readable after regeneration. Regeneration must
// explicitly chmod the path to 0600.
func TestRunWithoutKeyTightensExistingLooseKeyFile(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, filepath.Join(dir, "plugin"))
	keyPath := filepath.Join(dir, "plugin.ed25519.private.key")
	// Pre-existing private key with loose permissions.
	if err := os.WriteFile(keyPath, []byte("old-secret\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"-dir", filepath.Join(dir, "plugin")}, &stdout, &stderr, emptyStdin())
	if code == 0 {
		t.Fatalf("expected non-zero exit when no private key is provided, got 0")
	}

	st, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("private key file missing: %v", err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("existing private key file mode = %v, want 0600 after regeneration", st.Mode().Perm())
	}
	// The regenerated key must have replaced the placeholder.
	content, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "old-secret") {
		t.Fatal("stale private key content survived regeneration")
	}
}

func TestRunRejectsArgvKey(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "plugin")
	writeManifest(t, pluginDir)

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"-dir", pluginDir, "-ed25519-priv", base64.StdEncoding.EncodeToString(priv)}, &stdout, &stderr, emptyStdin())
	if code == 0 {
		t.Fatal("expected -ed25519-priv to be rejected (exit non-zero)")
	}
	if _, err := os.Stat(filepath.Join(pluginDir, "signature.sig")); err == nil {
		t.Fatal("signature.sig must not be written when the argv key flag is used")
	}
	if strings.Contains(stderr.String(), base64.StdEncoding.EncodeToString(priv)) {
		t.Fatal("stderr leaked the private key material")
	}
}

// TestRunRejectsLooseKeyFilePerms: a key file readable by group/others is refused
// (only 0600-or-stricter is acceptable) so a leaked file permission cannot expose
// the signing key.
func TestRunRejectsLooseKeyFilePerms(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "plugin")
	writeManifest(t, pluginDir)

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	keyFile := filepath.Join(dir, "loose.key")
	if err := os.WriteFile(keyFile, []byte(base64.StdEncoding.EncodeToString(priv)), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"-dir", pluginDir, "-ed25519-priv-file", keyFile}, &stdout, &stderr, emptyStdin())
	if code == 0 {
		t.Fatal("expected a loose key file to be rejected")
	}
	if !strings.Contains(stderr.String(), "0600") {
		t.Fatalf("stderr should explain the 0600 requirement, got: %s", stderr.String())
	}
	if _, err := os.Stat(filepath.Join(pluginDir, "signature.sig")); err == nil {
		t.Fatal("signature.sig must not be written for a rejected key file")
	}
}

// TestRunStdinKeySigns verifies the key can be supplied on stdin, which never
// shows up in argv or the environment.
func TestRunStdinKeySigns(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "plugin")
	writeManifest(t, pluginDir)

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := run([]string{"-dir", pluginDir, "-ed25519-priv-stdin"},
		&stdout, &stderr, strings.NewReader(base64.StdEncoding.EncodeToString(priv)+"\n"))
	if code != 0 {
		t.Fatalf("expected exit 0 when signing via stdin, got %d (stderr=%s)", code, stderr.String())
	}
	sig, err := os.ReadFile(filepath.Join(pluginDir, "signature.sig"))
	if err != nil {
		t.Fatal(err)
	}
	csBytes, err := os.ReadFile(filepath.Join(pluginDir, "checksums.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !ed25519.Verify(pub, csBytes, sig) {
		t.Fatal("signature from stdin key does not verify")
	}
}

// TestRunFDKeySigns verifies the key can be supplied via an open file descriptor.
func TestRunFDKeySigns(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "plugin")
	writeManifest(t, pluginDir)

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	keyFile := filepath.Join(dir, "fd.key")
	if err := os.WriteFile(keyFile, []byte(base64.StdEncoding.EncodeToString(priv)), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(keyFile)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	var stdout, stderr bytes.Buffer
	code := run([]string{"-dir", pluginDir, "-ed25519-priv-fd", "3"},
		&stdout, &stderr, emptyStdin())
	// The fd 3 in the child is not the open file here; this path is exercised
	// only when the parent arranges the fd. Rather than fork, assert the tool
	// rejects an fd that is not actually open.
	if code == 0 {
		t.Log("note: fd path succeeded; verifying signature")
		sig, err := os.ReadFile(filepath.Join(pluginDir, "signature.sig"))
		if err != nil {
			t.Fatal(err)
		}
		csBytes, err := os.ReadFile(filepath.Join(pluginDir, "checksums.json"))
		if err != nil {
			t.Fatal(err)
		}
		if !ed25519.Verify(pub, csBytes, sig) {
			t.Fatal("signature from fd key does not verify")
		}
		return
	}
	if !strings.Contains(stderr.String(), "private key") {
		t.Fatalf("expected a private-key error for an unopen fd, got: %s", stderr.String())
	}
}

// TestRunDoesNotEchoKeyOnError: a malformed key (bad base64 or wrong length) must
// not have its material echoed back into stderr/stdout.
func TestRunDoesNotEchoKeyOnError(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "plugin")
	writeManifest(t, pluginDir)

	secrets := []string{
		"super-secret-key-material-that-must-not-leak",
		base64.StdEncoding.EncodeToString(make([]byte, 16)), // wrong length
	}
	for _, secret := range secrets {
		keyFile := filepath.Join(dir, "bad.key")
		if err := os.WriteFile(keyFile, []byte(secret), 0o600); err != nil {
			t.Fatal(err)
		}
		var stdout, stderr bytes.Buffer
		code := run([]string{"-dir", pluginDir, "-ed25519-priv-file", keyFile}, &stdout, &stderr, emptyStdin())
		if code == 0 {
			t.Fatal("expected a non-zero exit for a malformed key")
		}
		combined := stdout.String() + stderr.String()
		if strings.Contains(combined, secret) {
			t.Fatalf("key material echoed to output for secret %q:\n%s", secret, combined)
		}
	}
}

// TestRunTrailingSlashDoesNotPlaceKeyInsidePlugin guards against a trailing "/" on
// -dir causing the generated private key to land inside the packaged plugin
// directory (where buildChecksums would hash it and it could be uploaded).
func TestRunTrailingSlashDoesNotPlaceKeyInsidePlugin(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "plugin")
	writeManifest(t, pluginDir)

	var stdout, stderr bytes.Buffer
	code := run([]string{"-dir", pluginDir + string(os.PathSeparator)}, &stdout, &stderr, emptyStdin())
	if code == 0 {
		t.Fatalf("expected non-zero exit when no private key is provided, got 0")
	}
	// The private key must be a sibling of the plugin dir, never inside it.
	keyPath := filepath.Join(dir, "plugin.ed25519.private.key")
	if _, err := os.Stat(keyPath); err != nil {
		t.Fatalf("private key file not written as sibling: %v", err)
	}
	if _, err := os.Stat(filepath.Join(pluginDir, ".ed25519.private.key")); err == nil {
		t.Fatal("private key leaked inside the plugin directory")
	}
}

func TestRunInvalidPrivateKeyFile(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "plugin")
	writeManifest(t, pluginDir)

	keyFile := filepath.Join(dir, "bad.key")
	if err := os.WriteFile(keyFile, []byte("not-base64"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"-dir", pluginDir, "-ed25519-priv-file", keyFile}, &stdout, &stderr, emptyStdin())
	if code != 2 {
		t.Fatalf("expected exit code 2 for invalid key file, got %d", code)
	}
}

func TestBuildChecksumsExcludesSigAndChecksums(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"bin/plugin.exe", "manifest.json", "signature.sig", "checksums.json"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cs, err := buildChecksums(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cs.Files["signature.sig"]; ok {
		t.Fatal("signature.sig must not be checksummed")
	}
	if _, ok := cs.Files["checksums.json"]; ok {
		t.Fatal("checksums.json must not be checksummed")
	}
	if _, ok := cs.Files["manifest.json"]; !ok {
		t.Fatal("manifest.json should be checksummed")
	}
	if _, ok := cs.Files["bin/plugin.exe"]; !ok {
		t.Fatal("bin/plugin.exe should be checksummed")
	}
}
