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

	// 密钥回写目标必须是「本次真正加载的那个配置文件」，而不是进程当前目录。
	// 否则当工作目录与配置文件目录不一致（如 systemd 未设置 WorkingDirectory、
	// 从其他目录启动二进制）时，每次启动都会读到不含密钥的配置、重新生成密钥，
	// 再写到另一个文件，导致重启后全部 JWT 失效——用户被迫重新登录。
	secretPath := resolveSecretPersistPath(configPath)
	if strings.TrimSpace(cfg.JWTSecret) == "" {
		cfg.JWTSecret = generateSecret()
		if err := persistSecretKey(secretPath, "jwt_secret", cfg.JWTSecret); err != nil {
			log.Printf("config: persist jwt_secret to %s failed: %v (existing sessions will be invalidated on next restart)", secretPath, err)
		}
	}
	if strings.TrimSpace(cfg.PluginMasterKey) == "" {
		cfg.PluginMasterKey = generateSecret()
		if err := persistSecretKey(secretPath, "plugin_master_key", cfg.PluginMasterKey); err != nil {
			log.Printf("config: persist plugin_master_key to %s failed: %v (plugin secrets become unreadable after restart)", secretPath, err)
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
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// resolveSecretPersistPath 决定自动生成的密钥往哪个配置文件回写。
//
// 优先写回 Load() 实际加载的文件；只有在完全没有配置文件时（安装引导阶段）
// 才回到当前目录，此时下一阶段的安装器会把 DB 配置合并进同一个文件。
func resolveSecretPersistPath(configPath string) string {
	if p := strings.TrimSpace(configPath); p != "" {
		return p
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

// persistSecretKey 把密钥合并写入指定配置文件，保留文件里已有的其他配置项。
func persistSecretKey(path, key, value string) error {
	if strings.TrimSpace(path) == "" {
		path = localConfigYAML
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	out := map[string]any{}
	if existing, err := os.ReadFile(path); err == nil {
		if strings.HasSuffix(strings.ToLower(path), ".json") {
			_ = json.Unmarshal(existing, &out)
		} else {
			_ = yaml.Unmarshal(existing, &out)
		}
	}
	if out == nil {
		out = map[string]any{}
	}
	out[key] = value

	if strings.HasSuffix(strings.ToLower(path), ".json") {
		b, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(path, b, 0o600)
	}
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
