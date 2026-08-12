package http_test

import (
	"crypto/sha256"
	"testing"

	"xiaoheiplay/internal/pkg/cryptox"
)

// testTOTPKeySource must match the key source used by testutilhttp.NewTestEnv.
const testTOTPKeySource = "test-totp-encryption-key"

// encryptTOTPSecretForTest seals a TOTP secret the same way the production
// auth service does, so handler tests can store a validly-encrypted secret.
func encryptTOTPSecretForTest(t *testing.T, secret string) string {
	t.Helper()
	key := sha256.Sum256([]byte(testTOTPKeySource))
	ring, err := cryptox.NewKeyRing("v1", map[string][]byte{"v1": key[:]})
	if err != nil {
		t.Fatalf("build test keyring: %v", err)
	}
	enc, err := ring.EncryptString(secret)
	if err != nil {
		t.Fatalf("encrypt totp secret: %v", err)
	}
	return enc
}
