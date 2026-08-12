package main

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBuildCanonicalPayload pins the exact canonical payload that both the Go
// signer and the Dart verifier must produce. Any change here breaks the
// signing contract and MUST be mirrored in
// app/xiaoheifs_userapp/lib/core/update/manifest_verifier.dart and
// app/xiaoheifs_app/lib/services/manifest_verifier.dart.
func TestBuildCanonicalPayload(t *testing.T) {
	m := Manifest{
		HasUpdate:         true,
		KeyEpoch:          1,
		LatestVersion:     "1.2.3",
		LatestVersionCode: 4,
		ForceUpdate:       false,
		ApkURL:            "https://pkg.example.com/app-v1.2.3.apk",
		SHA256:            "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		Changelog:         "bug fixes",
		IssuedAt:          1754952800,
	}
	want := "" +
		"pkg=com.example.app\n" +
		"hasUpdate=1\n" +
		"epoch=1\n" +
		"version=1.2.3\n" +
		"versionCode=4\n" +
		"force=0\n" +
		"url=https://pkg.example.com/app-v1.2.3.apk\n" +
		"sha256=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef\n" +
		"notes=bug fixes\n" +
		"issuedAt=1754952800"
	got := BuildCanonicalPayload("com.example.app", m)
	if got != want {
		t.Fatalf("canonical payload mismatch:\n got: %q\nwant: %q", got, want)
	}
}

// TestBuildCanonicalPayloadEscaping covers control-character escaping that the
// Dart _escape helper also implements.
func TestBuildCanonicalPayloadEscaping(t *testing.T) {
	m := Manifest{
		HasUpdate:         true,
		KeyEpoch:          2,
		LatestVersion:     "1.0",
		LatestVersionCode: 1,
		ForceUpdate:       true,
		ApkURL:            "https://pkg.example.com/a\\b\nc",
		SHA256:            "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		Changelog:         "tab\there",
		IssuedAt:          1,
	}
	got := BuildCanonicalPayload("com.example.app", m)
	for _, want := range []string{
		"hasUpdate=1\n",
		"epoch=2\n",
		"force=1\n",
		`url=https://pkg.example.com/a\\b\nc` + "\n",
		`notes=tab\there` + "\n",
		"issuedAt=1",
	} {
		if !bytes.Contains([]byte(got), []byte(want)) {
			t.Fatalf("payload missing %q in:\n%s", want, got)
		}
	}
}

// TestGoldenVector pins a fixed key/manifest -> signature pair shared with the
// Dart verifier test so both implementations agree byte-for-byte on the exact
// bytes the clients will verify.
func TestGoldenVector(t *testing.T) {
	// Deterministic test-only seed: bytes 0..31. Public key hex must match the
	// key the Dart golden test derives from the same seed.
	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = byte(i)
	}
	priv := ed25519.NewKeyFromSeed(seed)
	pubHex := hex.EncodeToString(priv.Public().(ed25519.PublicKey))

	m := Manifest{
		HasUpdate:         true,
		KeyEpoch:          1,
		LatestVersion:     "1.2.3",
		LatestVersionCode: 4,
		ForceUpdate:       false,
		ApkURL:            "https://updates.example.com/app.apk",
		SHA256:            "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Changelog:         "bug fixes",
		IssuedAt:          1754952800,
	}
	canonical := BuildCanonicalPayload("com.example.app", m)
	sig := ed25519.Sign(priv, []byte(canonical))
	if !ed25519.Verify(priv.Public().(ed25519.PublicKey), []byte(canonical), sig) {
		t.Fatal("signature did not verify")
	}

	// Pin the exact public key and signature so the Dart side (which uses the
	// same seed) can verify the SAME signature bytes. The Dart golden test
	// (script/manifest-verifier-test) checks this exact base64 signature
	// against the same canonical payload; any drift breaks the contract.
	const goldenPubHex = "03a107bff3ce10be1d70dd18e74bc09967e4d6309ba50d5f1ddc8664125531b8"
	const goldenSignatureB64 = "ikau1RjAMHxlF6UsZKt6UjfeuFMdTFLnSQhGD8+2msb4gyH5WKy2kv9WWmiKwtFD/DIoMVw/3HizdHG4LyeCDw=="
	if pubHex != goldenPubHex {
		t.Fatalf("unexpected golden public key %s", pubHex)
	}
	if got := base64.StdEncoding.EncodeToString(sig); got != goldenSignatureB64 {
		t.Fatalf("golden signature drift: got %s want %s (Dart clients would reject)", got, goldenSignatureB64)
	}
	t.Logf("public_key=%s", pubHex)
	t.Logf("signature=%s", goldenSignatureB64)
}

// TestSignVerifyRoundTrip proves the publish-side flow: build canonical
// payload, sign with Ed25519, and verify with the embedded public key. This is
// the same operation the Flutter clients perform in ManifestVerifier.
func TestSignVerifyRoundTrip(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	m := Manifest{
		HasUpdate:         true,
		KeyEpoch:          1,
		LatestVersion:     "1.2.3",
		LatestVersionCode: 4,
		ForceUpdate:       false,
		ApkURL:            "https://pkg.example.com/app-v1.2.3.apk",
		SHA256:            "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		Changelog:         "bug fixes",
		IssuedAt:          1754952800,
	}
	canonical := BuildCanonicalPayload("com.example.app", m)
	sig := ed25519.Sign(priv, []byte(canonical))
	if !ed25519.Verify(pub, []byte(canonical), sig) {
		t.Fatal("signature did not verify against the canonical payload")
	}

	// A tampered signed field must fail verification. Every behavior field is
	// covered by the signature, including hasUpdate and keyEpoch.
	for _, tamper := range []struct{ from, to string }{
		{"hasUpdate=1", "hasUpdate=0"},
		{"epoch=1", "epoch=2"},
		{"versionCode=4", "versionCode=5"},
		{"force=0", "force=1"},
		{"issuedAt=1754952800", "issuedAt=1754952801"},
	} {
		if ed25519.Verify(pub, []byte(strings.ReplaceAll(canonical, tamper.from, tamper.to)), sig) {
			t.Fatalf("signature unexpectedly verified a tampered payload (%s -> %s)", tamper.from, tamper.to)
		}
	}
}

// TestDecodePrivateKey exercises seed vs full private key forms used by the
// release tooling.
func TestDecodePrivateKey(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodePrivateKey(hex.EncodeToString(priv)); err != nil {
		t.Fatalf("full private key hex: %v", err)
	}
	if _, err := decodePrivateKey(hex.EncodeToString(priv.Seed())); err != nil {
		t.Fatalf("seed hex: %v", err)
	}
	if _, err := decodePrivateKey(base64.StdEncoding.EncodeToString(priv)); err != nil {
		t.Fatalf("private key base64: %v", err)
	}
	if _, err := decodePrivateKey(base64.StdEncoding.EncodeToString(priv.Seed())); err != nil {
		t.Fatalf("seed base64: %v", err)
	}
	if _, err := decodePrivateKey("not-a-key"); err == nil {
		t.Fatal("expected error for garbage key")
	}
	_ = pub
}

// TestLoadPrivateKeyRejectsInsecureSources proves the release tooling never
// takes the key from argv and refuses world-readable key files.
func TestLoadPrivateKeyRejectsInsecureSources(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	keyHex := hex.EncodeToString(priv.Seed())

	// No source -> error.
	if _, err := loadPrivateKey("", false); err == nil {
		t.Fatal("expected error when no key source is given")
	}
	// More than one source -> error.
	t.Setenv("UPDATE_SIGN_PRIVATE_KEY", keyHex)
	if _, err := loadPrivateKey("some-file", true); err == nil {
		t.Fatal("expected error when multiple key sources are given")
	}
	os.Unsetenv("UPDATE_SIGN_PRIVATE_KEY")

	// Protected env secret works.
	t.Setenv("UPDATE_SIGN_PRIVATE_KEY", keyHex)
	if _, err := loadPrivateKey("", false); err != nil {
		t.Fatalf("env secret source: %v", err)
	}
	os.Unsetenv("UPDATE_SIGN_PRIVATE_KEY")

	// 0600 key file works; a world-readable file is rejected.
	dir := t.TempDir()
	good := filepath.Join(dir, "good.key")
	if err := os.WriteFile(good, []byte(keyHex+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadPrivateKey(good, false); err != nil {
		t.Fatalf("0600 key file: %v", err)
	}
	bad := filepath.Join(dir, "bad.key")
	if err := os.WriteFile(bad, []byte(keyHex+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadPrivateKey(bad, false); err == nil {
		t.Fatal("expected error for world-readable key file")
	}
}
