package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"pingbot/internal/collector"
	"pingbot/internal/config"
	"pingbot/internal/service"
)

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

	svc := service.New(*cfgPath, cfg)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := svc.Run(ctx); err != nil {
		log.Fatalf("service exit: %v", err)
	}
}
