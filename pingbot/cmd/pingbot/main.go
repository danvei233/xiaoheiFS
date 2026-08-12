package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"pingbot/internal/collector"
	"pingbot/internal/config"
	"pingbot/internal/service"
)

// allowInsecureEnv is the explicit dev-only gate for TLS certificate
// verification bypass. The YAML flag alone is not enough; the operator must
// also set this environment variable, so a production install cannot silently
// skip verification.
const allowInsecureEnv = "PINGBOT_ALLOW_INSECURE"

func main() {
	defaultPath := collector.DefaultConfigPath()
	cfgPath := flag.String("config", defaultPath, "path to config yaml")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if cfg.ServerURL == "" {
		log.Fatalf("server_url is empty")
	}
	if cfg.TLSInsecureSkipVerify && strings.TrimSpace(os.Getenv(allowInsecureEnv)) != "1" {
		log.Fatalf("tls_insecure_skip_verify is set but %s is not '1'; refusing to start insecurely", allowInsecureEnv)
	}
	if cfg.TLSInsecureSkipVerify {
		log.Printf("[SECURITY] TLS certificate verification is DISABLED (tls_insecure_skip_verify + %s=1). "+
			"This is for development only and makes the connection vulnerable to MITM.", allowInsecureEnv)
	}
	log.Printf("pingbot starting config=%s server=%s probe_id=%d", *cfgPath, service.SanitizeURL(cfg.ServerURL), cfg.ProbeID)

	svc := service.New(*cfgPath, cfg)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := svc.Run(ctx); err != nil {
		log.Fatalf("service exit: %v", err)
	}
}
