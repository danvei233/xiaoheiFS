package cryptox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

type AESGCM struct {
	aead cipher.AEAD
}

func NewAESGCM(base64Key string) (*AESGCM, error) {
	if base64Key == "" {
		return nil, fmt.Errorf("missing key")
	}
	key, err := base64.RawURLEncoding.DecodeString(base64Key)
	if err != nil {
		// Accept padded standard base64 too (e.g. `openssl rand -base64 32`),
		// which the raw URL alphabet rejects because of `+ / =`.
		key, err = base64.StdEncoding.DecodeString(base64Key)
	}
	if err != nil {
		return nil, err
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("invalid key length")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &AESGCM{aead: aead}, nil
}

func (c *AESGCM) EncryptToString(plaintext []byte) (string, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	out := c.aead.Seal(nonce, nonce, plaintext, nil)
	return base64.RawURLEncoding.EncodeToString(out), nil
}

// DecryptString decrypts a base64.RawURLEncoding ciphertext produced by
// EncryptToString. An empty ciphertext means "no data stored" and returns
// (nil, nil) — consumers rely on this to distinguish an empty plugin config
// from a decrypt failure.
func (c *AESGCM) DecryptString(ciphertext string) ([]byte, error) {
	if ciphertext == "" {
		return nil, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, err
	}
	ns := c.aead.NonceSize()
	if len(b) < ns {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce := b[:ns]
	ct := b[ns:]
	return c.aead.Open(nil, nonce, ct, nil)
}
