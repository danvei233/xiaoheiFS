package config

import (
	"encoding/json"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	localConfigYAML = "app.config.yaml"
	localConfigYML  = "app.config.yml"
	localConfigJSON = "app.config.json"
)

type Config struct {
	Addr              string
	DBType            string
	DBPath            string
	DBDSN             string
	AdminUser         string
	AdminPass         string
	JWTSecret         string
	APIBase           string
	SiteName          string
	SiteURL           string
	AutomationBaseURL string
	AutomationAPIKey  string
}

type fileConfig struct {
	Addr      string `json:"addr" yaml:"addr"`
	APIBase   string `json:"api_base_url" yaml:"api_base_url"`
	JWTSecret string `json:"jwt_secret" yaml:"jwt_secret"`

	Admin struct {
		User string `json:"user" yaml:"user"`
		Pass string `json:"pass" yaml:"pass"`
	} `json:"admin" yaml:"admin"`

	DB struct {
		Type string `json:"type" yaml:"type"`
		Path string `json:"path" yaml:"path"`
		DSN  string `json:"dsn" yaml:"dsn"`
	} `json:"db" yaml:"db"`

	Site struct {
		Name string `json:"name" yaml:"name"`
		URL  string `json:"url" yaml:"url"`
	} `json:"site" yaml:"site"`

	Automation struct {
		BaseURL string `json:"base_url" yaml:"base_url"`
		APIKey  string `json:"api_key" yaml:"api_key"`
	} `json:"automation" yaml:"automation"`
}

func Load() Config {
	cfg := Config{
		Addr:              ":8080",
		DBType:            "sqlite",
		DBPath:            "./data/app.db",
		DBDSN:             "",
		AdminUser:         "admin",
		AdminPass:         "admin123",
		JWTSecret:         "dev_secret",
		APIBase:           "http://localhost:8080",
		SiteName:          "",
		SiteURL:           "",
		AutomationBaseURL: "https://idc.duncai.top/index.php/api/cloud",
		AutomationAPIKey:  "zPVhku8TueXcQbTcsdcu",
	}

	applyFileConfig(&cfg, readLocalConfig())

	cfg.Addr = getEnv("APP_ADDR", cfg.Addr)
	cfg.DBType = getEnv("APP_DB_TYPE", cfg.DBType)
	cfg.DBPath = getEnv("APP_DB_PATH", cfg.DBPath)
	cfg.DBDSN = getEnv("APP_DB_DSN", cfg.DBDSN)
	cfg.AdminUser = getEnv("ADMIN_USER", cfg.AdminUser)
	cfg.AdminPass = getEnv("ADMIN_PASS", cfg.AdminPass)
	cfg.JWTSecret = getEnv("ADMIN_JWT_SECRET", cfg.JWTSecret)
	cfg.APIBase = getEnv("API_BASE_URL", cfg.APIBase)
	cfg.SiteName = getEnv("SITE_NAME", cfg.SiteName)
	cfg.SiteURL = getEnv("SITE_URL", cfg.SiteURL)
	cfg.AutomationBaseURL = getEnv("AUTOMATION_BASE_URL", cfg.AutomationBaseURL)
	cfg.AutomationAPIKey = getEnv("AUTOMATION_API_KEY", cfg.AutomationAPIKey)

	return cfg
}

func readLocalConfig() *fileConfig {
	configPath := strings.TrimSpace(os.Getenv("APP_CONFIG_PATH"))
	candidates := []string{configPath, localConfigYAML, localConfigYML, localConfigJSON}

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
				return &fc
			}
		case strings.HasSuffix(strings.ToLower(p), ".json"):
			// Support both new nested JSON and the legacy flat JSON produced by older installers.
			if json.Unmarshal(b, &fc) == nil {
				return &fc
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
				return &fc
			}
		default:
			// If extension is unknown, try YAML then JSON.
			if yaml.Unmarshal(b, &fc) == nil {
				return &fc
			}
			if json.Unmarshal(b, &fc) == nil {
				return &fc
			}
		}
	}
	return nil
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
	if strings.TrimSpace(fc.Admin.User) != "" {
		cfg.AdminUser = strings.TrimSpace(fc.Admin.User)
	}
	if strings.TrimSpace(fc.Admin.Pass) != "" {
		cfg.AdminPass = strings.TrimSpace(fc.Admin.Pass)
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
	if strings.TrimSpace(fc.Site.Name) != "" {
		cfg.SiteName = strings.TrimSpace(fc.Site.Name)
	}
	if strings.TrimSpace(fc.Site.URL) != "" {
		cfg.SiteURL = strings.TrimSpace(fc.Site.URL)
	}
	if strings.TrimSpace(fc.Automation.BaseURL) != "" {
		cfg.AutomationBaseURL = strings.TrimSpace(fc.Automation.BaseURL)
	}
	if strings.TrimSpace(fc.Automation.APIKey) != "" {
		cfg.AutomationAPIKey = strings.TrimSpace(fc.Automation.APIKey)
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
