package config

import (
	"encoding/json"
	"os"
)

const localConfigJSON = "app.config.json"

type Config struct {
	Addr              string
	DBType            string
	DBPath            string
	DBDSN             string
	AdminUser         string
	AdminPass         string
	JWTSecret         string
	APIBase           string
	AutomationBaseURL string
	AutomationAPIKey  string
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
		AutomationBaseURL: "https://idc.duncai.top/index.php/api/cloud",
		AutomationAPIKey:  "zPVhku8TueXcQbTcsdcu",
	}

	type localCfg struct {
		DBType string `json:"db_type"`
		DBPath string `json:"db_path"`
		DBDSN  string `json:"db_dsn"`
	}
	if b, err := os.ReadFile(localConfigJSON); err == nil {
		var lc localCfg
		if json.Unmarshal(b, &lc) == nil {
			if lc.DBType != "" {
				cfg.DBType = lc.DBType
			}
			if lc.DBPath != "" {
				cfg.DBPath = lc.DBPath
			}
			if lc.DBDSN != "" {
				cfg.DBDSN = lc.DBDSN
			}
		}
	}

	cfg.Addr = getEnv("APP_ADDR", cfg.Addr)
	cfg.DBType = getEnv("APP_DB_TYPE", cfg.DBType)
	cfg.DBPath = getEnv("APP_DB_PATH", cfg.DBPath)
	cfg.DBDSN = getEnv("APP_DB_DSN", cfg.DBDSN)
	cfg.AdminUser = getEnv("ADMIN_USER", cfg.AdminUser)
	cfg.AdminPass = getEnv("ADMIN_PASS", cfg.AdminPass)
	cfg.JWTSecret = getEnv("ADMIN_JWT_SECRET", cfg.JWTSecret)
	cfg.APIBase = getEnv("API_BASE_URL", cfg.APIBase)
	cfg.AutomationBaseURL = getEnv("AUTOMATION_BASE_URL", cfg.AutomationBaseURL)
	cfg.AutomationAPIKey = getEnv("AUTOMATION_API_KEY", cfg.AutomationAPIKey)

	return cfg
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
