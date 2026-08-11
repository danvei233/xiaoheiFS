package cryptox

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
	"testing"
)

func rawKey() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func TestNewAESGCMInvalidKeys(t *testing.T) {
	if _, err := NewAESGCM(""); err == nil {
		t.Fatal("expected error for empty key")
	}
	if _, err := NewAESGCM("short"); err == nil {
		t.Fatal("expected error for non-32-byte key")
	}
	if _, err := NewAESGCM("!@#$"); err == nil {
		t.Fatal("expected error for non-base64 key")
	}
}

func TestNewAESGCMAcceptsPaddedBase64(t *testing.T) {
	// Padded standard base64 (openssl rand -base64 32 output) must be accepted.
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	padded := base64.StdEncoding.EncodeToString(b)
	if !strings.Contains(padded, "=") {
		t.Fatal("test setup: expected padding")
	}
	if _, err := NewAESGCM(padded); err != nil {
		t.Fatalf("expected padded std base64 key accepted, got %v", err)
	}
}

func TestRoundTrip(t *testing.T) {
	c, err := NewAESGCM(rawKey())
	if err != nil {
		t.Fatal(err)
	}
	ct, err := c.EncryptToString([]byte("hello world"))
	if err != nil {
		t.Fatal(err)
	}
	if ct == "" {
		t.Fatal("expected non-empty ciphertext")
	}
	pt, err := c.DecryptString(ct)
	if err != nil {
		t.Fatal(err)
	}
	if string(pt) != "hello world" {
		t.Fatalf("round-trip mismatch: %q", pt)
	}
}

func TestUniqueNonces(t *testing.T) {
	c, err := NewAESGCM(rawKey())
	if err != nil {
		t.Fatal(err)
	}
	a, _ := c.EncryptToString([]byte("data"))
	b2, _ := c.EncryptToString([]byte("data"))
	if a == b2 {
		t.Fatal("expected distinct ciphertexts (fresh nonce per encryption)")
	}
}

func TestAuthenticationFailure(t *testing.T) {
	c, err := NewAESGCM(rawKey())
	if err != nil {
		t.Fatal(err)
	}
	ct, err := c.EncryptToString([]byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := base64.RawURLEncoding.DecodeString(ct)
	if err != nil {
		t.Fatal(err)
	}
	b[len(b)-1] ^= 0x01
	if _, err := c.DecryptString(base64.RawURLEncoding.EncodeToString(b)); err == nil {
		t.Fatal("expected authentication failure on tampered ciphertext")
	}
}

// TestDecryptStringEmpty documents the contract: empty ciphertext means "no
// data" and returns (nil, nil), mirroring the consumer that treats an empty
// ConfigCipher as "no config stored".
func TestDecryptStringEmpty(t *testing.T) {
	c, err := NewAESGCM(rawKey())
	if err != nil {
		t.Fatal(err)
	}
	pt, err := c.DecryptString("")
	if err != nil {
		t.Fatalf("empty ciphertext must not error, got %v", err)
	}
	if pt != nil {
		t.Fatalf("empty ciphertext must return nil, got %q", pt)
	}
}

func TestDecryptStringGarbage(t *testing.T) {
	c, err := NewAESGCM(rawKey())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.DecryptString("@@not-base64@@"); err == nil {
		t.Fatal("expected error for invalid base64")
	}
	if _, err := c.DecryptString("AAAA"); err == nil {
		t.Fatal("expected error for ciphertext shorter than nonce")
	}
}
