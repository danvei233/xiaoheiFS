package config

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	localConfigYAML = "app.config.yaml"
	localConfigYML  = "app.config.yml"
	localConfigJSON = "app.config.json"
)

type Config struct {
	Addr               string
	DBType             string
	DBPath             string
	DBDSN              string
	JWTSecret          string
	PluginMasterKey    string
	PluginOfficialKeys []string
	PluginsDir         string
	APIBase            string
	SiteName           string
	SiteURL            string
	AutomationBaseURL  string
	AutomationAPIKey   string
	// ConfigDir is the directory of the loaded config file (app.config.*).
	// When PluginsDir/DBPath are relative, they are resolved from ConfigDir.
	ConfigDir string
}

type fileConfig struct {
	Addr    string `json:"addr" yaml:"addr"`
	APIBase string `json:"api_base_url" yaml:"api_base_url"`

	JWTSecret          string   `json:"jwt_secret" yaml:"jwt_secret"`
	PluginMasterKey    string   `json:"plugin_master_key" yaml:"plugin_master_key"`
	PluginOfficialKeys []string `json:"plugin_official_ed25519_pubkeys" yaml:"plugin_official_ed25519_pubkeys"`
	PluginsDir         string   `json:"plugins_dir" yaml:"plugins_dir"`

	DB struct {
		Type string `json:"type" yaml:"type"`
		Path string `json:"path" yaml:"path"`
		DSN  string `json:"dsn" yaml:"dsn"`
	} `json:"db" yaml:"db"`
}

func Load() Config {
	cfg := Config{
		Addr:               ":8080",
		DBType:             "",
		DBPath:             "",
		DBDSN:              "",
		JWTSecret:          "",
		PluginMasterKey:    "",
		PluginOfficialKeys: nil,
		PluginsDir:         "plugins",
		APIBase:            "http://localhost:8080",
		SiteName:           "",
		SiteURL:            "",
		AutomationBaseURL:  "",
		AutomationAPIKey:   "",
		ConfigDir:          "",
	}

	fc, configPath := readLocalConfig()
	applyFileConfig(&cfg, fc)
	if strings.TrimSpace(configPath) != "" {
		cfg.ConfigDir = filepath.Dir(configPath)
	} else {
		if abs, err := filepath.Abs("."); err == nil {
			cfg.ConfigDir = abs
		} else {
			cfg.ConfigDir = "."
		}
	}

	// Resolve relative paths from ConfigDir so running from different CWDs (e.g. air) is stable.
	if strings.TrimSpace(cfg.ConfigDir) != "" {
		if strings.TrimSpace(cfg.DBPath) != "" && !filepath.IsAbs(cfg.DBPath) {
			cfg.DBPath = filepath.Join(cfg.ConfigDir, cfg.DBPath)
		}
		if strings.TrimSpace(cfg.PluginsDir) != "" && !filepath.IsAbs(cfg.PluginsDir) {
			cfg.PluginsDir = filepath.Join(cfg.ConfigDir, cfg.PluginsDir)
		}
	}

	if strings.TrimSpace(cfg.JWTSecret) == "" {
		cfg.JWTSecret = generateSecret()
		_ = persistJWTSecret(cfg.JWTSecret)
	}
	if strings.TrimSpace(cfg.PluginMasterKey) == "" {
		cfg.PluginMasterKey = generateSecret()
		_ = persistPluginMasterKey(cfg.PluginMasterKey)
	}

	return cfg
}

// LocalConfigPath returns the absolute path to the local app.config.* file if found.
// It is used by installer logic to place/read install.lock next to the config.
func LocalConfigPath() string {
	_, configPath := readLocalConfig()
	return configPath
}

func readLocalConfig() (*fileConfig, string) {
	candidates := []string{
		localConfigYAML,
		localConfigYML,
		localConfigJSON,
		filepath.Join("..", localConfigYAML),
		filepath.Join("..", localConfigYML),
		filepath.Join("..", localConfigJSON),
		filepath.Join("backend", localConfigYAML),
		filepath.Join("backend", localConfigYML),
		filepath.Join("backend", localConfigJSON),
	}
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidates = append(candidates,
			filepath.Join(exeDir, localConfigYAML),
			filepath.Join(exeDir, localConfigYML),
			filepath.Join(exeDir, localConfigJSON),
		)
	}

	var fallbackFC *fileConfig
	var fallbackPath string

	for _, p := range candidates {
		if strings.TrimSpace(p) == "" {
			continue
		}
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}

		var fc fileConfig
		switch {
		case strings.HasSuffix(strings.ToLower(p), ".yaml") || strings.HasSuffix(strings.ToLower(p), ".yml"):
			if yaml.Unmarshal(b, &fc) == nil {
				hasDB := strings.TrimSpace(fc.DB.Type) != "" || strings.TrimSpace(fc.DB.Path) != "" || strings.TrimSpace(fc.DB.DSN) != ""
				if abs, err := filepath.Abs(p); err == nil {
					if hasDB {
						return &fc, abs
					}
					if fallbackFC == nil {
						cp := fc
						fallbackFC = &cp
						fallbackPath = abs
					}
					continue
				}
				if hasDB {
					return &fc, p
				}
				if fallbackFC == nil {
					cp := fc
					fallbackFC = &cp
					fallbackPath = p
				}
				continue
			}
		case strings.HasSuffix(strings.ToLower(p), ".json"):
			// Support both new nested JSON and the legacy flat JSON produced by older installers.
			if json.Unmarshal(b, &fc) == nil {
				hasDB := strings.TrimSpace(fc.DB.Type) != "" || strings.TrimSpace(fc.DB.Path) != "" || strings.TrimSpace(fc.DB.DSN) != ""
				if abs, err := filepath.Abs(p); err == nil {
					if hasDB {
						return &fc, abs
					}
					if fallbackFC == nil {
						cp := fc
						fallbackFC = &cp
						fallbackPath = abs
					}
					continue
				}
				if hasDB {
					return &fc, p
				}
				if fallbackFC == nil {
					cp := fc
					fallbackFC = &cp
					fallbackPath = p
				}
				continue
			}
			var legacy struct {
				DBType string `json:"db_type"`
				DBPath string `json:"db_path"`
				DBDSN  string `json:"db_dsn"`
			}
			if json.Unmarshal(b, &legacy) == nil {
				fc.DB.Type = legacy.DBType
				fc.DB.Path = legacy.DBPath
				fc.DB.DSN = legacy.DBDSN
				hasDB := strings.TrimSpace(fc.DB.Type) != "" || strings.TrimSpace(fc.DB.Path) != "" || strings.TrimSpace(fc.DB.DSN) != ""
				if abs, err := filepath.Abs(p); err == nil {
					if hasDB {
						return &fc, abs
					}
					if fallbackFC == nil {
						cp := fc
						fallbackFC = &cp
						fallbackPath = abs
					}
					continue
				}
				if hasDB {
					return &fc, p
				}
				if fallbackFC == nil {
					cp := fc
					fallbackFC = &cp
					fallbackPath = p
				}
				continue
			}
		default:
			// If extension is unknown, try YAML then JSON.
			if yaml.Unmarshal(b, &fc) == nil {
				hasDB := strings.TrimSpace(fc.DB.Type) != "" || strings.TrimSpace(fc.DB.Path) != "" || strings.TrimSpace(fc.DB.DSN) != ""
				if abs, err := filepath.Abs(p); err == nil {
					if hasDB {
						return &fc, abs
					}
					if fallbackFC == nil {
						cp := fc
						fallbackFC = &cp
						fallbackPath = abs
					}
					continue
				}
				if hasDB {
					return &fc, p
				}
				if fallbackFC == nil {
					cp := fc
					fallbackFC = &cp
					fallbackPath = p
				}
				continue
			}
			if json.Unmarshal(b, &fc) == nil {
				hasDB := strings.TrimSpace(fc.DB.Type) != "" || strings.TrimSpace(fc.DB.Path) != "" || strings.TrimSpace(fc.DB.DSN) != ""
				if abs, err := filepath.Abs(p); err == nil {
					if hasDB {
						return &fc, abs
					}
					if fallbackFC == nil {
						cp := fc
						fallbackFC = &cp
						fallbackPath = abs
					}
					continue
				}
				if hasDB {
					return &fc, p
				}
				if fallbackFC == nil {
					cp := fc
					fallbackFC = &cp
					fallbackPath = p
				}
			}
		}
	}
	if fallbackFC != nil {
		return fallbackFC, fallbackPath
	}
	return nil, ""
}

func applyFileConfig(cfg *Config, fc *fileConfig) {
	if cfg == nil || fc == nil {
		return
	}
	if strings.TrimSpace(fc.Addr) != "" {
		cfg.Addr = strings.TrimSpace(fc.Addr)
	}
	if strings.TrimSpace(fc.APIBase) != "" {
		cfg.APIBase = strings.TrimSpace(fc.APIBase)
	}
	if strings.TrimSpace(fc.JWTSecret) != "" {
		cfg.JWTSecret = strings.TrimSpace(fc.JWTSecret)
	}
	if strings.TrimSpace(fc.PluginMasterKey) != "" {
		cfg.PluginMasterKey = strings.TrimSpace(fc.PluginMasterKey)
	}
	if len(fc.PluginOfficialKeys) > 0 {
		cfg.PluginOfficialKeys = fc.PluginOfficialKeys
	}
	if strings.TrimSpace(fc.PluginsDir) != "" {
		cfg.PluginsDir = strings.TrimSpace(fc.PluginsDir)
	}
	if strings.TrimSpace(fc.DB.Type) != "" {
		cfg.DBType = strings.TrimSpace(fc.DB.Type)
	}
	if strings.TrimSpace(fc.DB.Path) != "" {
		cfg.DBPath = strings.TrimSpace(fc.DB.Path)
	}
	if strings.TrimSpace(fc.DB.DSN) != "" {
		cfg.DBDSN = strings.TrimSpace(fc.DB.DSN)
	}
}

func generateSecret() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func persistJWTSecret(secret string) error {
	path := localConfigYAML
	switch {
	case fileExists(localConfigYAML):
		path = localConfigYAML
	case fileExists(localConfigYML):
		path = localConfigYML
	case fileExists(localConfigJSON):
		path = localConfigJSON
	default:
		path = localConfigYAML
	}

	if strings.HasSuffix(strings.ToLower(path), ".json") {
		out := map[string]any{}
		if existing, err := os.ReadFile(path); err == nil {
			_ = json.Unmarshal(existing, &out)
		}
		out["jwt_secret"] = secret
		b, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(path, b, 0o600)
	}

	out := map[string]any{}
	if existing, err := os.ReadFile(path); err == nil {
		_ = yaml.Unmarshal(existing, &out)
	}
	out["jwt_secret"] = secret
	b, err := yaml.Marshal(&out)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

func persistPluginMasterKey(key string) error {
	path := localConfigYAML
	switch {
	case fileExists(localConfigYAML):
		path = localConfigYAML
	case fileExists(localConfigYML):
		path = localConfigYML
	case fileExists(localConfigJSON):
		path = localConfigJSON
	default:
		path = localConfigYAML
	}

	if strings.HasSuffix(strings.ToLower(path), ".json") {
		out := map[string]any{}
		if existing, err := os.ReadFile(path); err == nil {
			_ = json.Unmarshal(existing, &out)
		}
		out["plugin_master_key"] = key
		b, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(path, b, 0o600)
	}

	out := map[string]any{}
	if existing, err := os.ReadFile(path); err == nil {
		_ = yaml.Unmarshal(existing, &out)
	}
	out["plugin_master_key"] = key
	b, err := yaml.Marshal(&out)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
