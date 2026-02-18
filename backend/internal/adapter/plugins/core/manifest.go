package plugins

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Manifest struct {
	PluginID     string            `json:"plugin_id"`
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Description  string            `json:"description,omitempty"`
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
		} `json:"automation,omitempty"`
	} `json:"capabilities"`
}

func ReadManifest(dir string) (Manifest, error) {
	b, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return Manifest{}, err
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return Manifest{}, err
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
