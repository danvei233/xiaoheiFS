package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"xiaoheiplay/internal/adapter/automation"
	"xiaoheiplay/internal/adapter/email"
	"xiaoheiplay/internal/adapter/event"
	"xiaoheiplay/internal/adapter/http"
	"xiaoheiplay/internal/adapter/payment"
	"xiaoheiplay/internal/adapter/realname"
	"xiaoheiplay/internal/adapter/repo"
	"xiaoheiplay/internal/adapter/robot"
	"xiaoheiplay/internal/adapter/seed"
	"xiaoheiplay/internal/adapter/sse"
	"xiaoheiplay/internal/adapter/system"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/pkg/config"
	"xiaoheiplay/internal/pkg/db"
	"xiaoheiplay/internal/pkg/permissions"
	"xiaoheiplay/internal/usecase"
)

func main() {
	cfg := config.Load()

	conn, err := db.Open(cfg)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	if err := repo.Migrate(conn.Gorm); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	initLockPath := filepath.Join(filepath.Dir(cfg.DBPath), "init.lock")
	_, initLockErr := os.Stat(initLockPath)
	initLocked := initLockErr == nil
	if initLockErr != nil && !os.IsNotExist(initLockErr) {
		log.Fatalf("init lock: %v", initLockErr)
	}
	if err := seed.EnsureSettings(conn.SQL, conn.Dialect); err != nil {
		log.Fatalf("seed settings: %v", err)
	}
	if err := seed.EnsurePermissionDefaults(conn.SQL, conn.Dialect); err != nil {
		log.Fatalf("seed permission defaults: %v", err)
	}
	if err := seed.EnsurePermissionGroups(conn.SQL, conn.Dialect); err != nil {
		log.Fatalf("seed permission groups: %v", err)
	}
	if !initLocked {
		if err := seed.EnsureCMSDefaults(conn.SQL, conn.Dialect); err != nil {
			log.Fatalf("seed cms defaults: %v", err)
		}
		if err := seed.SeedIfEmpty(conn.SQL); err != nil {
			log.Fatalf("seed: %v", err)
		}
		if err := os.MkdirAll(filepath.Dir(initLockPath), 0o755); err != nil {
			log.Fatalf("init lock dir: %v", err)
		}
		if err := os.WriteFile(initLockPath, []byte(time.Now().Format(time.RFC3339)), 0o644); err != nil {
			log.Fatalf("init lock write: %v", err)
		}
	}
	if err := os.MkdirAll("uploads", 0o755); err != nil {
		log.Fatalf("uploads dir: %v", err)
	}

	repoSQLite := repo.NewSQLiteRepo(conn.Gorm)
	if isInstalled() {
		if _, userSet := os.LookupEnv("ADMIN_USER"); userSet {
			if _, passSet := os.LookupEnv("ADMIN_PASS"); passSet {
				ensureAdminUser(repoSQLite, cfg.AdminUser, cfg.AdminPass)
			}
		}
	}
	if v, ok := os.LookupEnv("AUTOMATION_BASE_URL"); ok && v != "" {
		_ = repoSQLite.UpsertSetting(context.Background(), domain.Setting{Key: "automation_base_url", ValueJSON: v})
	}
	if v, ok := os.LookupEnv("AUTOMATION_API_KEY"); ok && v != "" {
		_ = repoSQLite.UpsertSetting(context.Background(), domain.Setting{Key: "automation_api_key", ValueJSON: v})
		_ = repoSQLite.UpsertSetting(context.Background(), domain.Setting{Key: "automation_enabled", ValueJSON: "true"})
	}
	if strings.TrimSpace(cfg.SiteName) != "" {
		_ = repoSQLite.UpsertSetting(context.Background(), domain.Setting{Key: "site_name", ValueJSON: strings.TrimSpace(cfg.SiteName)})
	}
	if strings.TrimSpace(cfg.SiteURL) != "" {
		_ = repoSQLite.UpsertSetting(context.Background(), domain.Setting{Key: "site_url", ValueJSON: strings.TrimSpace(cfg.SiteURL)})
	}

	catalogSvc := usecase.NewCatalogService(repoSQLite, repoSQLite, repoSQLite)
	cartSvc := usecase.NewCartService(repoSQLite, repoSQLite, repoSQLite)
	broker := sse.NewBroker(repoSQLite)
	automationClient := automation.NewDynamicClient(repoSQLite, cfg.AutomationBaseURL, cfg.AutomationAPIKey, repoSQLite)
	emailSender := email.NewSender(repoSQLite)
	robotNotifier := robot.NewWebhookNotifier(repoSQLite)
	eventBus := event.NewFanoutPublisher(broker, robotNotifier)
	realnameRegistry := realname.NewRegistry()
	realnameSvc := usecase.NewRealNameService(repoSQLite, realnameRegistry, repoSQLite)
	messageSvc := usecase.NewMessageCenterService(repoSQLite, repoSQLite)
	orderSvc := usecase.NewOrderService(repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, eventBus, automationClient, nil, repoSQLite, repoSQLite, emailSender, repoSQLite, repoSQLite, repoSQLite, repoSQLite, messageSvc, realnameSvc)
	vpsSvc := usecase.NewVPSService(repoSQLite, automationClient, repoSQLite)
	adminSvc := usecase.NewAdminService(repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite)
	adminVPSSvc := usecase.NewAdminVPSService(repoSQLite, automationClient, repoSQLite, repoSQLite, repoSQLite, messageSvc)
	apiKeySvc := usecase.NewAPIKeyService(repoSQLite)
	authSvc := usecase.NewAuthService(repoSQLite, repoSQLite)
	notifySvc := usecase.NewNotificationService(repoSQLite, repoSQLite, repoSQLite, emailSender, messageSvc)
	integrationSvc := usecase.NewIntegrationService(repoSQLite, repoSQLite, repoSQLite, automationClient, repoSQLite)
	reportSvc := usecase.NewReportService(repoSQLite, repoSQLite, repoSQLite)
	cmsSvc := usecase.NewCMSService(repoSQLite, repoSQLite, repoSQLite, messageSvc)
	ticketSvc := usecase.NewTicketService(repoSQLite, repoSQLite, repoSQLite, messageSvc)
	permissionSvc := usecase.NewPermissionService(repoSQLite, repoSQLite, repoSQLite)
	passwordResetSvc := usecase.NewPasswordResetService(repoSQLite, repoSQLite, emailSender, repoSQLite)
	walletSvc := usecase.NewWalletService(repoSQLite, repoSQLite)
	walletOrderSvc := usecase.NewWalletOrderService(repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, automationClient, repoSQLite)
	paymentRegistry := payment.NewRegistry(repoSQLite)
	paymentSvc := usecase.NewPaymentService(repoSQLite, repoSQLite, repoSQLite, paymentRegistry, repoSQLite, orderSvc, eventBus)
	statusSvc := usecase.NewServerStatusService(system.NewProvider())
	taskSvc := usecase.NewScheduledTaskService(repoSQLite, vpsSvc, orderSvc, notifySvc, repoSQLite)
	go taskSvc.Start(context.Background())

	pluginDir := getSettingValue(repoSQLite, "payment_plugin_dir")
	if pluginDir == "" {
		pluginDir = "plugins/payment"
	}
	pluginPassword := getSettingValue(repoSQLite, "payment_plugin_upload_password")
	_ = os.MkdirAll(pluginDir, 0o755)
	_ = paymentRegistry.StartWatcher(context.Background(), pluginDir)

	handler := http.NewHandlerWithServices(authSvc, catalogSvc, cartSvc, orderSvc, vpsSvc, adminSvc, adminVPSSvc, integrationSvc, reportSvc, cmsSvc, ticketSvc, walletSvc, walletOrderSvc, paymentSvc, messageSvc, statusSvc, realnameSvc, repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, broker, cfg.JWTSecret, automationClient, passwordResetSvc, permissionSvc, taskSvc)
	handler.SetPaymentPluginConfig(pluginDir, pluginPassword)
	middleware := http.NewMiddleware(cfg.JWTSecret, apiKeySvc, permissionSvc)
	server := http.NewServer(handler, middleware)

	routeDefinitions := permissions.BuildFromRoutes(server.Engine.Routes())
	permissions.SetDefinitions(routeDefinitions)
	if err := repoSQLite.RegisterPermissions(context.Background(), routeDefinitions); err != nil {
		log.Printf("permission register failed: %v", err)
	}

	log.Printf("listening on %s", cfg.Addr)
	if err := server.Engine.Run(cfg.Addr); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func isInstalled() bool {
	lockPath := strings.TrimSpace(os.Getenv("APP_INSTALL_LOCK_PATH"))
	if lockPath == "" {
		lockPath = "install.lock"
	}
	_, err := os.Stat(lockPath)
	return err == nil
}

func ensureAdminUser(repo *repo.SQLiteRepo, username, password string) {
	if username == "" || password == "" {
		return
	}
	ctx := context.Background()
	if _, err := repo.GetUserByUsernameOrEmail(ctx, username); err == nil {
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	superAdminGroupID := int64(1)
	user := &domain.User{
		Username:          username,
		Email:             username + "@local",
		PasswordHash:      string(hash),
		Role:              domain.UserRoleAdmin,
		Status:            domain.UserStatusActive,
		PermissionGroupID: &superAdminGroupID,
	}
	_ = repo.CreateUser(ctx, user)
}

func getSettingValue(repo *repo.SQLiteRepo, key string) string {
	setting, err := repo.GetSetting(context.Background(), key)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(setting.ValueJSON)
}
