package plugins

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Manifest struct {
	PluginID     string            `json:"plugin_id"`
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Description  string            `json:"description,omitempty"`
	TestOnly     bool              `json:"test_only"`
	Binaries     map[string]string `json:"binaries,omitempty"`
	Capabilities struct {
		SMS *struct {
			Send bool `json:"send"`
		} `json:"sms,omitempty"`
		Payment *struct {
			Methods []string `json:"methods"`
		} `json:"payment,omitempty"`
		KYC *struct {
			Start       bool `json:"start"`
			QueryResult bool `json:"query_result"`
		} `json:"kyc,omitempty"`
		Automation *struct {
			Features           []string          `json:"features"`
			NotSupportedReason map[string]string `json:"not_supported_reasons,omitempty"`
			CatalogReadonly    bool              `json:"catalog_readonly,omitempty"`
		} `json:"automation,omitempty"`
	} `json:"capabilities"`
}

// RejectTestOnly fails closed when a manifest declares `test_only: true`.
// Packaging scripts exclude test-only plugins from releases, but the runtime
// must not rely on that alone: an admin uploading a test-only package (e.g.
// mockpay, which lets any order be marked paid) must be refused here too.
// A build-time flag can deliberately allow test-only plugins in dev/test
// deployments; the default (allow=false) is production-safe.
func (m Manifest) RejectTestOnly(allow bool) error {
	if m.TestOnly && !allow {
		return fmt.Errorf("plugin %q is test-only and not allowed to be installed/enabled", m.PluginID)
	}
	return nil
}

func ReadManifest(dir string) (Manifest, error) {
	b, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return Manifest{}, err
	}
	var m Manifest
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		return Manifest{}, err
	}
	// Reject trailing content after the top-level object (json.Unmarshal was
	// strict about this; a streaming Decode alone would silently ignore it).
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return Manifest{}, fmt.Errorf("invalid manifest")
	}
	m.PluginID = strings.TrimSpace(m.PluginID)
	m.Name = strings.TrimSpace(m.Name)
	m.Version = strings.TrimSpace(m.Version)
	if m.PluginID == "" || m.Name == "" || m.Version == "" {
		return Manifest{}, fmt.Errorf("invalid manifest")
	}
	if len(m.Binaries) > 0 {
		clean := map[string]string{}
		for k, v := range m.Binaries {
			key := strings.TrimSpace(k)
			val := filepath.ToSlash(strings.TrimSpace(v))
			if key == "" || val == "" {
				return Manifest{}, fmt.Errorf("invalid manifest binaries")
			}
			// Platform keys are lookup keys (e.g. "linux_amd64") surfaced to the
			// UI; reject separators and traversal components that would only make
			// sense as filesystem paths.
			if strings.ContainsAny(key, "/\\:") || strings.Contains(key, "..") || strings.HasPrefix(key, ".") {
				return Manifest{}, fmt.Errorf("invalid manifest binaries platform")
			}
			if strings.HasPrefix(val, "/") || strings.Contains(val, "..") || strings.Contains(val, ":") {
				return Manifest{}, fmt.Errorf("invalid manifest binaries path")
			}
			prefix := "bin/" + key + "/"
			if !strings.HasPrefix(val, prefix) {
				return Manifest{}, fmt.Errorf("%s", "invalid manifest binaries path (must be "+prefix+"...)")
			}
			base := filepath.Base(filepath.FromSlash(val))
			if base != "plugin" && base != "plugin.exe" {
				return Manifest{}, fmt.Errorf("invalid manifest binaries filename (must be plugin or plugin.exe)")
			}
			clean[key] = val
		}
		m.Binaries = clean
	}
	return m, nil
}
