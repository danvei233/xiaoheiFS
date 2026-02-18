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
)

type Checksums struct {
	Algo  string            `json:"algo"`
	Files map[string]string `json:"files"`
}

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
	// checksums.json must cover all binaries under bin/** (multi-platform packages).
	if err := ensureBinFilesCovered(pluginDir, cs.Files); err != nil {
		return domain.PluginSignatureUntrusted, err
	}
	for rel, wantHex := range cs.Files {
		rel = filepath.ToSlash(strings.TrimSpace(rel))
		if rel == "" {
			return domain.PluginSignatureUntrusted, fmt.Errorf("invalid checksums path")
		}
		if strings.HasPrefix(rel, "/") || strings.Contains(rel, "..") {
			return domain.PluginSignatureUntrusted, fmt.Errorf("invalid checksums path")
		}
		if strings.Contains(rel, ":") {
			return domain.PluginSignatureUntrusted, fmt.Errorf("invalid checksums path")
		}
		wantHex = strings.ToLower(strings.TrimSpace(wantHex))
		if len(wantHex) != 64 {
			return domain.PluginSignatureUntrusted, fmt.Errorf("invalid checksum")
		}
		if _, err := hex.DecodeString(wantHex); err != nil {
			return domain.PluginSignatureUntrusted, fmt.Errorf("invalid checksum")
		}
		full := filepath.Join(pluginDir, filepath.FromSlash(rel))
		sum, err := sha256File(full)
		if err != nil {
			return domain.PluginSignatureUntrusted, err
		}
		if wantHex != sum {
			return domain.PluginSignatureUntrusted, fmt.Errorf("checksum mismatch")
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
	sigBytes = bytesTrimSpace(sigBytes)
	if len(sigBytes) != ed25519.SignatureSize {
		decoded, derr := base64.StdEncoding.DecodeString(string(sigBytes))
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

func ensureBinFilesCovered(pluginDir string, files map[string]string) error {
	binDir := filepath.Join(pluginDir, "bin")
	st, err := os.Stat(binDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !st.IsDir() {
		return fmt.Errorf("bin is not a directory")
	}
	return filepath.WalkDir(binDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(pluginDir, path)
		rel = filepath.ToSlash(rel)
		if _, ok := files[rel]; !ok {
			return fmt.Errorf("%s", "checksums missing for bin file: "+rel)
		}
		return nil
	})
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
