package config

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log"
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

// fatalConfigLoad is the hook that refuses to start on an unparseable primary
// config (P2-55). Tests swap it to capture the call instead of exiting.
var fatalConfigLoad = func(format string, args ...any) {
	log.Fatalf(format, args...)
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
	if fc == nil && lastConfigParseFailed {
		fatalConfigLoad("config: a config file exists but none could be parsed; refusing to start on defaults (P2-55)")
	}
	applyFileConfig(&cfg, fc)
	applyEnvConfig(&cfg)
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
		// Persist to the same config file that was loaded (configPath may live
		// outside the process CWD). Writing to a CWD shadow file would rotate
		// the secret on every restart.
		if err := persistJWTSecret(cfg.JWTSecret, configPath); err != nil {
			log.Printf("config: failed to persist jwt_secret: %v", err)
		}
	}
	if strings.TrimSpace(cfg.PluginMasterKey) == "" {
		cfg.PluginMasterKey = generateSecret()
		if err := persistPluginMasterKey(cfg.PluginMasterKey, configPath); err != nil {
			log.Printf("config: failed to persist plugin_master_key: %v", err)
		}
	}

	return cfg
}

// LocalConfigPath returns the absolute path to the local app.config.* file if found.
// It is used by installer logic to place/read install.lock next to the config.
func LocalConfigPath() string {
	_, configPath := readLocalConfig()
	return configPath
}

// lastConfigParseFailed records that a candidate config file existed on disk
// but failed to parse. Load() refuses to start in that case (instead of
// silently running on defaults), because a corrupt config can point the server
// at the wrong database or trigger a fresh JWT secret (P2-55).
var lastConfigParseFailed bool

func readLocalConfig() (*fileConfig, string) {
	lastConfigParseFailed = false
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
			if err := yaml.Unmarshal(b, &fc); err == nil {
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
			} else {
				lastConfigParseFailed = true
				log.Printf("config: ignoring unparseable config file %s: %v", p, err)
			}
		case strings.HasSuffix(strings.ToLower(p), ".json"):
			// Support both new nested JSON and the legacy flat JSON produced by older installers.
			if err := json.Unmarshal(b, &fc); err == nil {
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
			} else {
				lastConfigParseFailed = true
				log.Printf("config: ignoring unparseable config file %s: %v", p, err)
			}
			var legacy struct {
				DBType string `json:"db_type"`
				DBPath string `json:"db_path"`
				DBDSN  string `json:"db_dsn"`
			}
			if err := json.Unmarshal(b, &legacy); err == nil {
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
			} else {
				log.Printf("config: ignoring unparseable config file %s: %v", p, err)
			}
		default:
			// If extension is unknown, try YAML then JSON.
			if err := yaml.Unmarshal(b, &fc); err == nil {
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
			} else {
				log.Printf("config: ignoring unparseable config file %s: %v", p, err)
			}
			if err := json.Unmarshal(b, &fc); err == nil {
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
			} else {
				log.Printf("config: ignoring unparseable config file %s: %v", p, err)
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

func applyEnvConfig(cfg *Config) {
	if cfg == nil {
		return
	}
	if v, ok := getEnvTrimmed("APP_ADDR"); ok {
		cfg.Addr = v
	}
	if v, ok := getEnvTrimmed("APP_API_BASE_URL"); ok {
		cfg.APIBase = v
	}
	if v, ok := getEnvTrimmed("APP_DB_TYPE"); ok {
		cfg.DBType = v
	}
	if v, ok := getEnvTrimmed("APP_DB_PATH"); ok {
		cfg.DBPath = v
	}
	if v, ok := getEnvTrimmed("APP_DB_DSN"); ok {
		cfg.DBDSN = v
	}
	if v, ok := getEnvTrimmed("APP_JWT_SECRET"); ok {
		cfg.JWTSecret = v
	}
	if v, ok := getEnvTrimmed("APP_PLUGIN_MASTER_KEY"); ok {
		cfg.PluginMasterKey = v
	}
	if v, ok := getEnvTrimmed("APP_PLUGINS_DIR"); ok {
		cfg.PluginsDir = v
	}
}

func getEnvTrimmed(key string) (string, bool) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return "", false
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return "", false
	}
	return v, true
}

func generateSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand is guaranteed to fill b on Go 1.24+; a failure here is
		// unrecoverable and must not silently yield an all-zero key.
		panic("config: crypto/rand unavailable: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// persistJWTSecret writes the secret into the given config file (the one the
// loader actually resolved). When configPath is empty it falls back to a CWD
// app.config.* file for first-run installs.
func persistJWTSecret(secret, configPath string) error {
	path := resolvePersistPath(configPath)

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
		return writeConfigFile(path, b)
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
	return writeConfigFile(path, b)
}

func persistPluginMasterKey(key, configPath string) error {
	path := resolvePersistPath(configPath)

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
		return writeConfigFile(path, b)
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
	return writeConfigFile(path, b)
}

// resolvePersistPath returns the config file to persist secrets into: the
// loaded config path when known, otherwise the first CWD app.config.* that
// exists, defaulting to app.config.yaml.
func resolvePersistPath(configPath string) string {
	if strings.TrimSpace(configPath) != "" {
		return configPath
	}
	switch {
	case fileExists(localConfigYAML):
		return localConfigYAML
	case fileExists(localConfigYML):
		return localConfigYML
	case fileExists(localConfigJSON):
		return localConfigJSON
	default:
		return localConfigYAML
	}
}

// writeConfigFile writes data with 0o600 and tightens permissions on a
// pre-existing file that may have been created with a looser mode.
func writeConfigFile(path string, data []byte) error {
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return err
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
