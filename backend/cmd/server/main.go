package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"xiaoheiplay/internal/adapter/email"
	"xiaoheiplay/internal/adapter/event"
	"xiaoheiplay/internal/adapter/http"
	"xiaoheiplay/internal/adapter/payment"
	"xiaoheiplay/internal/adapter/plugins/automation"
	"xiaoheiplay/internal/adapter/plugins/core"
	"xiaoheiplay/internal/adapter/push"
	"xiaoheiplay/internal/adapter/realname"
	"xiaoheiplay/internal/adapter/repo/core"
	"xiaoheiplay/internal/adapter/robot"
	"xiaoheiplay/internal/adapter/seed"
	"xiaoheiplay/internal/adapter/sse"
	"xiaoheiplay/internal/adapter/system"
	appadmin "xiaoheiplay/internal/app/admin"
	appadminvps "xiaoheiplay/internal/app/adminvps"
	appapikey "xiaoheiplay/internal/app/apikey"
	appauth "xiaoheiplay/internal/app/auth"
	appautomationlog "xiaoheiplay/internal/app/automationlog"
	appcart "xiaoheiplay/internal/app/cart"
	appcatalog "xiaoheiplay/internal/app/catalog"
	appcms "xiaoheiplay/internal/app/cms"
	appcoupon "xiaoheiplay/internal/app/coupon"
	appgoodstype "xiaoheiplay/internal/app/goodstype"
	appintegration "xiaoheiplay/internal/app/integration"
	appmessage "xiaoheiplay/internal/app/message"
	appnotification "xiaoheiplay/internal/app/notification"
	apporder "xiaoheiplay/internal/app/order"
	apporderevent "xiaoheiplay/internal/app/orderevent"
	apppasswordreset "xiaoheiplay/internal/app/passwordreset"
	apppayment "xiaoheiplay/internal/app/payment"
	apppermission "xiaoheiplay/internal/app/permission"
	apppluginadmin "xiaoheiplay/internal/app/pluginadmin"
	appprobe "xiaoheiplay/internal/app/probe"
	apppush "xiaoheiplay/internal/app/push"
	apprealname "xiaoheiplay/internal/app/realname"
	appreport "xiaoheiplay/internal/app/report"
	appscheduledtask "xiaoheiplay/internal/app/scheduledtask"
	appsecurityticket "xiaoheiplay/internal/app/securityticket"
	appsettings "xiaoheiplay/internal/app/settings"
	appsystemstatus "xiaoheiplay/internal/app/systemstatus"
	appticket "xiaoheiplay/internal/app/ticket"
	appupload "xiaoheiplay/internal/app/upload"
	appusertier "xiaoheiplay/internal/app/usertier"
	appvps "xiaoheiplay/internal/app/vps"
	appwallet "xiaoheiplay/internal/app/wallet"
	appwalletorder "xiaoheiplay/internal/app/walletorder"
	"xiaoheiplay/internal/pkg/config"
	"xiaoheiplay/internal/pkg/cryptox"
	"xiaoheiplay/internal/pkg/db"
	"xiaoheiplay/internal/pkg/permissions"
)

func main() {
	cfg := config.Load()
	if strings.TrimSpace(cfg.DBType) == "" {
		log.Printf("db config missing; entering install bootstrap mode on %s", cfg.Addr)
		server := http.NewInstallBootstrapServer(cfg.JWTSecret)
		if err := server.Engine.Run(cfg.Addr); err != nil {
			log.Fatalf("server: %v", err)
		}
		return
	}

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
	if err := seed.EnsureSettings(conn.Gorm); err != nil {
		log.Fatalf("seed settings: %v", err)
	}
	if err := seed.EnsurePermissionDefaults(conn.Gorm); err != nil {
		log.Fatalf("seed permission defaults: %v", err)
	}
	if err := seed.EnsurePermissionGroups(conn.Gorm); err != nil {
		log.Fatalf("seed permission groups: %v", err)
	}
	if !initLocked {
		if err := seed.EnsureCMSDefaults(conn.Gorm); err != nil {
			log.Fatalf("seed cms defaults: %v", err)
		}
		if err := seed.SeedIfEmpty(conn.Gorm); err != nil {
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

	repoSQLite := repo.NewGormRepo(conn.Gorm)

	pluginCipher, err := cryptox.NewAESGCM(cfg.PluginMasterKey)
	if err != nil {
		log.Fatalf("plugin cipher: %v", err)
	}
	if strings.TrimSpace(cfg.PluginsDir) == "" {
		log.Fatalf("plugins_dir is empty in config")
	}
	pluginMgr := plugins.NewManager(cfg.PluginsDir, repoSQLite, pluginCipher, plugins.ParseEd25519PublicKeys(cfg.PluginOfficialKeys))
	pluginSMSSender := plugins.NewSMSSender(pluginMgr)
	pluginAdminSvc := apppluginadmin.NewService(plugins.NewAdminManager(pluginMgr), repoSQLite, repoSQLite)
	_ = pluginMgr.BootstrapFromDisk(context.Background(), repoSQLite)
	pluginMgr.StartEnabled(context.Background())

	catalogSvc := appcatalog.NewService(repoSQLite, repoSQLite, repoSQLite)
	goodsTypeSvc := appgoodstype.NewService(repoSQLite, repoSQLite)
	cartSvc := appcart.NewService(repoSQLite, repoSQLite, repoSQLite)
	broker := sse.NewBroker(repoSQLite)
	automationResolver := automation.NewResolver(repoSQLite, pluginMgr, repoSQLite, repoSQLite)
	emailSender := email.NewSender(repoSQLite)
	robotNotifier := robot.NewWebhookNotifier(repoSQLite)
	pushSender := push.NewFCMSender()
	pushSvc := apppush.NewService(repoSQLite, repoSQLite, repoSQLite, pushSender)
	pushNotifier := push.NewOrderPushNotifier(repoSQLite, pushSvc)
	eventBus := event.NewFanoutPublisher(broker, robotNotifier, pushNotifier)
	realnameRegistry := realname.NewRegistry(repoSQLite)
	realnameRegistry.SetPluginManager(pluginMgr)
	realnameSvc := apprealname.NewService(repoSQLite, realnameRegistry, repoSQLite)
	messageSvc := appmessage.NewService(repoSQLite, repoSQLite)
	orderSvc := apporder.NewService(repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, eventBus, automationResolver, nil, repoSQLite, repoSQLite, emailSender, repoSQLite, repoSQLite, repoSQLite, repoSQLite, messageSvc, realnameSvc)
	vpsSvc := appvps.NewService(repoSQLite, automationResolver, repoSQLite)
	adminSvc := appadmin.NewService(repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite)
	adminVPSSvc := appadminvps.NewService(repoSQLite, automationResolver, repoSQLite, repoSQLite, repoSQLite, messageSvc)
	apiKeySvc := appapikey.NewService(repoSQLite)
	authSvc := appauth.NewService(repoSQLite, repoSQLite, repoSQLite)
	notifySvc := appnotification.NewService(repoSQLite, repoSQLite, repoSQLite, emailSender, messageSvc)
	integrationSvc := appintegration.NewService(repoSQLite, repoSQLite, repoSQLite, repoSQLite, automationResolver, repoSQLite)
	reportSvc := appreport.NewService(repoSQLite, repoSQLite, repoSQLite)
	cmsSvc := appcms.NewService(repoSQLite, repoSQLite, repoSQLite, messageSvc)
	ticketSvc := appticket.NewService(repoSQLite, repoSQLite, repoSQLite, messageSvc)
	permissionSvc := apppermission.NewService(repoSQLite, repoSQLite, repoSQLite)
	passwordResetSvc := apppasswordreset.NewService(repoSQLite, repoSQLite, emailSender, repoSQLite)
	walletSvc := appwallet.NewService(repoSQLite, repoSQLite)
	walletOrderSvc := appwalletorder.NewService(repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, automationResolver, repoSQLite)
	couponSvc := appcoupon.NewService(repoSQLite, repoSQLite)
	userTierSvc := appusertier.NewService(repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite)
	_, _ = userTierSvc.EnsureDefaultGroup(context.Background())
	authSvc.SetUserTierAssigner(userTierSvc)
	adminSvc.SetUserTierAssigner(userTierSvc)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		updated, err := userTierSvc.BackfillUsersWithoutGroup(ctx, 500)
		if err != nil {
			log.Printf("user tier backfill failed: %v", err)
			return
		}
		if updated > 0 {
			log.Printf("user tier backfill completed: updated=%d", updated)
		}
	}()
	cartSvc.SetUserTierPricingResolver(userTierSvc)
	orderSvc.SetUserTierPricingResolver(userTierSvc)
	orderSvc.SetUserTierAutoApprover(userTierSvc)
	orderSvc.SetCouponService(couponSvc)
	walletOrderSvc.SetUserTierAutoApprover(userTierSvc)
	uploadSvc := appupload.NewService(repoSQLite)
	autoLogSvc := appautomationlog.NewService(repoSQLite)
	orderEventSvc := apporderevent.NewService(repoSQLite)
	securityTicketSvc := appsecurityticket.NewService(repoSQLite)
	settingsSvc := appsettings.NewService(repoSQLite)

	paymentRegistry := payment.NewRegistry(repoSQLite)
	paymentRegistry.SetPluginManager(pluginMgr)
	paymentRegistry.SetPluginPaymentMethodRepo(repoSQLite)
	paymentSvc := apppayment.NewService(repoSQLite, repoSQLite, repoSQLite, paymentRegistry, repoSQLite, orderSvc, eventBus)
	statusSvc := appsystemstatus.NewService(system.NewProvider())
	taskSvc := appscheduledtask.NewService(repoSQLite, vpsSvc, orderSvc, notifySvc, repoSQLite, realnameSvc)
	taskSvc.SetUserTierService(userTierSvc)
	probeHub := appprobe.NewHub()
	probeSvc := appprobe.NewService(repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite)
	go taskSvc.Start(context.Background())
	go probeSvc.StartOfflineWatcher(context.Background())

	pluginDir := pluginAdminSvc.ResolveUploadDir(context.Background(), "")
	_ = os.MkdirAll(pluginDir, 0o755)
	_ = paymentRegistry.StartWatcher(context.Background(), pluginDir)

	handler := http.NewHandler(http.HandlerDeps{
		AuthSvc:           authSvc,
		CatalogSvc:        catalogSvc,
		GoodsTypes:        goodsTypeSvc,
		CartSvc:           cartSvc,
		OrderSvc:          orderSvc,
		VPSSvc:            vpsSvc,
		AdminSvc:          adminSvc,
		AdminVPS:          adminVPSSvc,
		Integration:       integrationSvc,
		ReportSvc:         reportSvc,
		CMSSvc:            cmsSvc,
		TicketSvc:         ticketSvc,
		WalletSvc:         walletSvc,
		WalletOrder:       walletOrderSvc,
		PaymentSvc:        paymentSvc,
		MessageSvc:        messageSvc,
		PushSvc:           pushSvc,
		StatusSvc:         statusSvc,
		RealnameSvc:       realnameSvc,
		OrderEventSvc:     orderEventSvc,
		AutoLogSvc:        autoLogSvc,
		SettingsSvc:       settingsSvc,
		UploadSvc:         uploadSvc,
		Broker:            broker,
		JWTSecret:         cfg.JWTSecret,
		PasswordReset:     passwordResetSvc,
		SecurityTicketSvc: securityTicketSvc,
		PermissionSvc:     permissionSvc,
		PluginAdmin:       pluginAdminSvc,
		UserTierSvc:       userTierSvc,
		CouponSvc:         couponSvc,
		SMSSender:         pluginSMSSender,
		TaskSvc:           taskSvc,
		ProbeSvc:          probeSvc,
		ProbeHub:          probeHub,
		EmailSender:       emailSender,
		RobotNotifier:     robotNotifier,
	})
	middleware := http.NewMiddleware(cfg.JWTSecret, apiKeySvc, permissionSvc, authSvc, settingsSvc)
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
