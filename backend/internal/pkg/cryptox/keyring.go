package cryptox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
)

// KeyRing holds a set of versioned AES-GCM keys. The active key encrypts new
// values; decryption-only keys are accepted so rotated-out keys can still read
// existing ciphertext. Every ciphertext carries the key id it was sealed with,
// so a key rotation never needs to guess which key a value was written with.
//
// The envelope format is a JSON object:
//
//	{"k":"<keyid>","c":"<base64url nonce||ciphertext>"}
//
// Decryption fails (never falls back to the plaintext) when the key id is
// unknown, the envelope is malformed, or the authentication tag does not match.
type KeyRing struct {
	keys   map[string]*AESGCM
	active string
}

// NewKeyRing builds a KeyRing. activeID must reference an entry in keys.
// Each key material must be exactly 32 bytes (AES-256).
func NewKeyRing(activeID string, keys map[string][]byte) (*KeyRing, error) {
	if activeID == "" || len(keys) == 0 {
		return nil, fmt.Errorf("cryptox: at least one key with an active id is required")
	}
	ring := &KeyRing{keys: map[string]*AESGCM{}, active: activeID}
	for id, material := range keys {
		if len(material) != 32 {
			return nil, fmt.Errorf("cryptox: key %q must be 32 bytes, got %d", id, len(material))
		}
		block, err := aes.NewCipher(material)
		if err != nil {
			return nil, fmt.Errorf("cryptox: key %q: %w", id, err)
		}
		aead, err := cipher.NewGCM(block)
		if err != nil {
			return nil, fmt.Errorf("cryptox: key %q: %w", id, err)
		}
		ring.keys[id] = &AESGCM{aead: aead}
	}
	if _, ok := ring.keys[activeID]; !ok {
		return nil, fmt.Errorf("cryptox: active key %q is not present", activeID)
	}
	return ring, nil
}

// ActiveKeyID returns the id of the key used for new encryptions.
func (k *KeyRing) ActiveKeyID() string {
	return k.active
}

// EncryptString seals plaintext with the active key and returns a versioned
// envelope. An empty plaintext produces an empty envelope (no data stored).
func (k *KeyRing) EncryptString(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	aead := k.keys[k.active]
	nonce := make([]byte, aead.aead.NonceSize())
	if err := readFullRandom(nonce); err != nil {
		return "", fmt.Errorf("cryptox: nonce: %w", err)
	}
	ct := aead.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	env := keyringEnvelope{Key: k.active, Cipher: base64.RawURLEncoding.EncodeToString(ct)}
	raw, err := json.Marshal(env)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// DecryptString unwraps a versioned envelope and returns the plaintext. It
// returns an error when the envelope is malformed, the key id is unknown, or
// the authentication check fails — it never returns the input as the secret.
func (k *KeyRing) DecryptString(envelope string) (string, error) {
	if envelope == "" {
		return "", nil
	}
	var env keyringEnvelope
	if err := json.Unmarshal([]byte(envelope), &env); err != nil {
		return "", fmt.Errorf("cryptox: malformed envelope: %w", err)
	}
	aead, ok := k.keys[env.Key]
	if !ok {
		return "", fmt.Errorf("cryptox: unknown key id %q", env.Key)
	}
	ct, err := base64.RawURLEncoding.DecodeString(env.Cipher)
	if err != nil {
		return "", fmt.Errorf("cryptox: bad ciphertext: %w", err)
	}
	ns := aead.aead.NonceSize()
	if len(ct) < ns {
		return "", fmt.Errorf("cryptox: ciphertext too short")
	}
	plain, err := aead.aead.Open(nil, ct[:ns], ct[ns:], nil)
	if err != nil {
		return "", fmt.Errorf("cryptox: decrypt: %w", err)
	}
	return string(plain), nil
}

type keyringEnvelope struct {
	Key    string `json:"k"`
	Cipher string `json:"c"`
}

// readFullRandom fills buf from the crypto/rand reader, propagating any
// failure so callers never continue with a zero-filled (guessable) nonce.
func readFullRandom(buf []byte) error {
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return err
	}
	return nil
}
