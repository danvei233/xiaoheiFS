package cryptox

import (
	"crypto/sha256"
	"strings"
	"testing"
)

func mkKey(t *testing.T, seed string) []byte {
	t.Helper()
	sum := sha256.Sum256([]byte(seed))
	return sum[:]
}

func TestKeyRing_RoundTrip(t *testing.T) {
	ring, err := NewKeyRing("v1", map[string][]byte{"v1": mkKey(t, "active")})
	if err != nil {
		t.Fatalf("new ring: %v", err)
	}
	enc, err := ring.EncryptString("secret-totp")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if enc == "secret-totp" {
		t.Fatalf("ciphertext must never equal the plaintext")
	}
	plain, err := ring.DecryptString(enc)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if plain != "secret-totp" {
		t.Fatalf("round trip mismatch: %q", plain)
	}
}

// TestKeyRing_Rotation verifies a value sealed with an older key remains
// readable after rotation when the old key is retained as decryption-only.
func TestKeyRing_Rotation(t *testing.T) {
	old, err := NewKeyRing("v1", map[string][]byte{"v1": mkKey(t, "old-key")})
	if err != nil {
		t.Fatalf("old ring: %v", err)
	}
	enc, err := old.EncryptString("legacy-value")
	if err != nil {
		t.Fatalf("encrypt with old: %v", err)
	}

	// Rotate: v2 becomes active, v1 retained for reads.
	rotated, err := NewKeyRing("v2", map[string][]byte{
		"v2": mkKey(t, "new-key"),
		"v1": mkKey(t, "old-key"),
	})
	if err != nil {
		t.Fatalf("rotated ring: %v", err)
	}
	plain, err := rotated.DecryptString(enc)
	if err != nil {
		t.Fatalf("old value must stay readable after rotation: %v", err)
	}
	if plain != "legacy-value" {
		t.Fatalf("rotation mismatch: %q", plain)
	}

	// New writes use v2 and must not be readable by the old ring.
	newEnc, err := rotated.EncryptString("fresh-value")
	if err != nil {
		t.Fatalf("encrypt with new: %v", err)
	}
	if _, err := old.DecryptString(newEnc); err == nil {
		t.Fatalf("a value sealed with v2 must not be readable by the retired v1-only ring")
	}
}

// TestKeyRing_CorruptCiphertextIsNeverPlaintext verifies a damaged envelope is
// an error, not the plaintext secret.
func TestKeyRing_CorruptCiphertextIsNeverPlaintext(t *testing.T) {
	ring, err := NewKeyRing("v1", map[string][]byte{"v1": mkKey(t, "k")})
	if err != nil {
		t.Fatalf("new ring: %v", err)
	}
	for _, bad := range []string{
		"not-json",
		`{"k":"v1","c":"!!!not-base64"}`,
		`{"k":"unknown","c":"AAECAwQFBgcICQoLDA0ODxAREhMUFQ"}`,
		`{"k":"v1","c":"AAEC"}`,
		"{}",
	} {
		if plain, err := ring.DecryptString(bad); err == nil {
			t.Fatalf("corrupt input %q must error, got plaintext %q", bad, plain)
		}
	}
}

// TestKeyRing_RequiresActiveKey verifies a ring cannot be built with an empty
// or unknown active key id.
func TestKeyRing_RequiresActiveKey(t *testing.T) {
	if _, err := NewKeyRing("", map[string][]byte{"v1": mkKey(t, "k")}); err == nil {
		t.Fatalf("empty active id must be rejected")
	}
	if _, err := NewKeyRing("v1", map[string][]byte{}); err == nil {
		t.Fatalf("empty key set must be rejected")
	}
	if _, err := NewKeyRing("v9", map[string][]byte{"v1": mkKey(t, "k")}); err == nil || !strings.Contains(err.Error(), "active key") {
		t.Fatalf("unknown active id must be rejected, got %v", err)
	}
	if _, err := NewKeyRing("v1", map[string][]byte{"v1": []byte("short")}); err == nil {
		t.Fatalf("a short key material must be rejected")
	}
}
