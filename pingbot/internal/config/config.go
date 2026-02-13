package config

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ServerURL             string `yaml:"server_url"`
	EnrollToken           string `yaml:"enroll_token"`
	ProbeID               int64  `yaml:"probe_id"`
	ProbeSecret           string `yaml:"probe_secret"`
	HostnameAlias         string `yaml:"hostname_alias"`
	TLSInsecureSkipVerify bool   `yaml:"tls_insecure_skip_verify"`
}

func Load(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return Config{}, err
	}
	cfg.ServerURL = strings.TrimRight(strings.TrimSpace(cfg.ServerURL), "/")
	return cfg, nil
}

func Save(path string, cfg Config) error {
	cfg.ServerURL = strings.TrimRight(strings.TrimSpace(cfg.ServerURL), "/")
	b, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}
