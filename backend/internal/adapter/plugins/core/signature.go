package plugins

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"fmt"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/pkg/pluginsign"
)

type Checksums struct {
	Algo  string            `json:"algo"`
	Files map[string]string `json:"files"`
}

// VerifySignature validates the checksums.json / signature.sig pair of an
// on-disk plugin directory. The signer and this verifier share one canonical
// manifest (pluginsign.CanonicalManifest), so a signature covers every regular
// file in the package except the two bookkeeping files themselves. That closes
// the gap where a plugin could add or replace a schema, template, script or
// shared library that was not part of the signed payload.
//
// Returns:
//   - PluginSignatureOfficial  when a signature verifies against an official key
//   - PluginSignatureUnsigned  when no checksums.json (or no signature.sig) exists
//   - PluginSignatureUntrusted when checksums are present but incomplete,
//     mismatched, or not signed by an official key
func VerifySignature(pluginDir string, officialKeys []ed25519.PublicKey) (domain.PluginSignatureStatus, error) {
	checksumsPath := filepath.Join(pluginDir, "checksums.json")
	checksumsBytes, err := os.ReadFile(checksumsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return domain.PluginSignatureUnsigned, nil
		}
		return domain.PluginSignatureUntrusted, err
	}

	var cs Checksums
	if err := json.Unmarshal(checksumsBytes, &cs); err != nil {
		return domain.PluginSignatureUntrusted, err
	}
	if strings.TrimSpace(cs.Algo) == "" {
		cs.Algo = "sha256"
	}
	if strings.ToLower(strings.TrimSpace(cs.Algo)) != "sha256" {
		return domain.PluginSignatureUntrusted, fmt.Errorf("unsupported checksum algo")
	}
	if len(cs.Files) == 0 {
		return domain.PluginSignatureUntrusted, fmt.Errorf("empty checksums")
	}

	canonical, err := pluginsign.CanonicalManifest(pluginDir)
	if err != nil {
		return domain.PluginSignatureUntrusted, err
	}
	canonicalSums := pluginsign.FilesMap(canonical)

	// 1) Every listed checksum must correspond to a real, regular file and must
	//    match its content. A checksum that resolves to a non-existent file or
	//    collides with another entry through case/path normalization is rejected.
	for rel, wantHex := range cs.Files {
		rel = strings.TrimSpace(rel)
		if err := validateChecksumPath(rel); err != nil {
			return domain.PluginSignatureUntrusted, err
		}
		wantHex = strings.ToLower(strings.TrimSpace(wantHex))
		if len(wantHex) != 64 {
			return domain.PluginSignatureUntrusted, fmt.Errorf("invalid checksum")
		}
		if _, err := hex.DecodeString(wantHex); err != nil {
			return domain.PluginSignatureUntrusted, fmt.Errorf("invalid checksum")
		}
		sum, ok := canonicalSums[rel]
		if !ok {
			return domain.PluginSignatureUntrusted, fmt.Errorf("checksums reference a non-existent file: %s", rel)
		}
		if wantHex != sum {
			return domain.PluginSignatureUntrusted, fmt.Errorf("checksum mismatch: %s", rel)
		}
	}

	// 2) Every regular file in the package must be covered by the checksums.
	for rel := range canonicalSums {
		if _, ok := cs.Files[rel]; !ok {
			return domain.PluginSignatureUntrusted, fmt.Errorf("checksums missing for file: %s", rel)
		}
	}

	sigPath := filepath.Join(pluginDir, "signature.sig")
	sigBytes, err := os.ReadFile(sigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return domain.PluginSignatureUnsigned, nil
		}
		return domain.PluginSignatureUntrusted, err
	}
	// signature.sig is stored as raw ed25519 bytes by the official signer, but
	// some tooling may write base64. Accept either. Raw signatures must NOT be
	// whitespace-trimmed: a valid signature can legitimately start or end with
	// an ASCII whitespace octet, and trimming would corrupt it.
	trimmed := bytesTrimSpace(sigBytes)
	switch {
	case len(sigBytes) == ed25519.SignatureSize:
		// raw signature, use as-is
	case len(trimmed) == ed25519.SignatureSize:
		// raw signature with surrounding whitespace (e.g. a trailing newline)
		sigBytes = trimmed
	default:
		decoded, derr := base64.StdEncoding.DecodeString(string(trimmed))
		if derr != nil || len(decoded) != ed25519.SignatureSize {
			return domain.PluginSignatureUntrusted, fmt.Errorf("invalid signature")
		}
		sigBytes = decoded
	}
	for _, pub := range officialKeys {
		if len(pub) != ed25519.PublicKeySize {
			continue
		}
		if ed25519.Verify(pub, checksumsBytes, sigBytes) {
			return domain.PluginSignatureOfficial, nil
		}
	}
	return domain.PluginSignatureUntrusted, nil
}

// validateChecksumPath rejects checksum keys that could alias a real file or
// escape the plugin directory once joined onto it: separators, traversal
// components, drive-colon prefixes and backslash separators are all refused.
func validateChecksumPath(rel string) error {
	if rel == "" {
		return fmt.Errorf("invalid checksums path")
	}
	if strings.HasPrefix(rel, "/") || strings.Contains(rel, "..") || strings.Contains(rel, ":") || strings.Contains(rel, `\`) {
		return fmt.Errorf("invalid checksums path")
	}
	return nil
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func bytesTrimSpace(b []byte) []byte {
	return []byte(strings.TrimSpace(string(b)))
}
