package http

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"xiaoheiplay/internal/adapter/sse"
	"xiaoheiplay/internal/usecase"
)

type Server struct {
	Engine *gin.Engine
}

func NewServer(handler *Handler, middleware *Middleware) *Server {
	r := gin.Default()
	r.Use(corsMiddleware())
	r.Static("/uploads", "./uploads")

	// Installer gate: before installation completes, redirect site traffic to /install and
	// block non-install API calls to avoid confusing errors.
	r.Use(installGateMiddleware(handler))

	// Serve built frontend assets from ./static (Vite/SPA).
	// - If a real file exists under ./static, serve it (e.g. /assets/*, /favicon.ico).
	// - Otherwise, for non-API routes, fall back to ./static/index.html so history-mode routing works.
	r.Use(spaStaticFileMiddleware("./static",
		[]string{"/api/", "/admin/api/", "/uploads/"},
	))
	r.NoRoute(spaIndexFallbackHandler("./static",
		[]string{"/api/", "/admin/api/", "/uploads/"},
	))

	public := r.Group("/api/v1")
	{
		public.GET("/install/status", handler.InstallStatus)
		public.POST("/install/db/check", handler.InstallDBCheck)
		public.POST("/install", handler.InstallRun)

		public.GET("/captcha", handler.Captcha)
		public.GET("/auth/settings", handler.AuthSettings)
		public.POST("/auth/register/code", handler.RegisterCode)
		public.POST("/auth/register", handler.Register)
		public.POST("/auth/login", handler.Login)
		public.POST("/auth/password-reset/options", handler.PasswordResetOptions)
		public.POST("/auth/password-reset/send-code", handler.PasswordResetSendCode)
		public.POST("/auth/password-reset/verify-code", handler.PasswordResetVerifyCode)
		public.POST("/auth/password-reset/confirm", handler.PasswordResetConfirm)
		public.POST("/auth/refresh", handler.Refresh)
		public.Any("/payments/notify/:provider", handler.PaymentNotify)
		public.GET("/site/settings", handler.SiteSettings)
		public.GET("/cms/blocks", handler.CMSBlocksPublic)
		public.GET("/cms/posts", handler.CMSPostsPublic)
		public.GET("/cms/posts/:slug", handler.CMSPostDetailPublic)
		public.POST("/probe/enroll", handler.ProbeEnroll)
		public.POST("/probe/auth/token", handler.ProbeAuthToken)
		public.GET("/probe/ws", handler.ProbeWS)
	}

	user := r.Group("/api/v1")
	user.Use(middleware.RequireUser())
	{
		user.GET("/me", handler.Me)
		user.PATCH("/me", handler.UpdateProfile)
		user.POST("/me/password/change", handler.MePasswordChange)
		user.GET("/me/security/contacts", handler.MeSecurityContacts)
		user.POST("/me/security/email/verify-2fa", handler.MeSecurityEmailVerify2FA)
		user.POST("/me/security/email/send-code", handler.MeSecurityEmailSendCode)
		user.POST("/me/security/email/confirm", handler.MeSecurityEmailConfirm)
		user.POST("/me/security/phone/verify-2fa", handler.MeSecurityPhoneVerify2FA)
		user.POST("/me/security/phone/send-code", handler.MeSecurityPhoneSendCode)
		user.POST("/me/security/phone/confirm", handler.MeSecurityPhoneConfirm)
		user.GET("/me/security/2fa/status", handler.MeTwoFAStatus)
		user.POST("/me/security/2fa/setup", handler.MeTwoFASetup)
		user.POST("/me/security/2fa/confirm", handler.MeTwoFAConfirm)
		user.GET("/realname/status", handler.RealNameStatus)
		user.POST("/realname/verify", handler.RealNameVerify)
		user.GET("/dashboard", handler.Dashboard)
		user.GET("/goods-types", handler.GoodsTypes)
		user.GET("/catalog", handler.Catalog)
		user.GET("/plan-groups", handler.PlanGroups)
		user.GET("/packages", handler.Packages)
		user.GET("/system-images", handler.SystemImages)
		user.GET("/billing-cycles", handler.BillingCycles)
		user.GET("/payments/providers", handler.PaymentMethods)
		user.POST("/auth/logout", handler.Logout)
		user.GET("/cart", handler.CartList)
		user.POST("/cart", handler.CartAdd)
		user.DELETE("/cart", handler.CartClear)
		user.PATCH("/cart/:id", handler.CartUpdate)
		user.DELETE("/cart/:id", handler.CartDelete)
		user.POST("/orders", handler.OrderCreate)
		user.POST("/orders/items", handler.OrderCreateItems)
		user.GET("/orders", handler.OrderList)
		user.GET("/orders/:id", handler.OrderDetail)
		user.POST("/orders/:id/pay", handler.OrderPay)
		user.POST("/orders/:id/payments", handler.OrderPayment)
		user.POST("/orders/:id/cancel", handler.OrderCancel)
		user.GET("/orders/:id/events", handler.OrderEvents)
		user.POST("/orders/:id/refresh", handler.OrderRefresh)
		user.POST("/tickets", handler.TicketCreate)
		user.GET("/tickets", handler.TicketList)
		user.GET("/tickets/:id", handler.TicketDetail)
		user.POST("/tickets/:id/messages", handler.TicketMessageCreate)
		user.POST("/tickets/:id/close", handler.TicketClose)
		user.GET("/notifications", handler.Notifications)
		user.GET("/notifications/unread-count", handler.NotificationsUnreadCount)
		user.POST("/notifications/:id/read", handler.NotificationRead)
		user.POST("/notifications/read-all", handler.NotificationReadAll)
		user.GET("/wallet", handler.WalletInfo)
		user.GET("/wallet/transactions", handler.WalletTransactions)
		user.POST("/wallet/recharge", handler.WalletRecharge)
		user.POST("/wallet/withdraw", handler.WalletWithdraw)
		user.GET("/wallet/orders", handler.WalletOrders)
		user.GET("/vps", handler.VPSList)
		user.GET("/vps/:id", handler.VPSDetail)
		user.POST("/vps/:id/refresh", handler.VPSRefresh)
		user.GET("/vps/:id/panel", handler.VPSPanel)
		user.GET("/vps/:id/monitor", handler.VPSMonitor)
		user.GET("/vps/:id/vnc", handler.VPSVNC)
		user.POST("/vps/:id/start", handler.VPSStart)
		user.POST("/vps/:id/shutdown", handler.VPSShutdown)
		user.POST("/vps/:id/reboot", handler.VPSReboot)
		user.POST("/vps/:id/reset-os", handler.VPSResetOS)
		user.POST("/vps/:id/reset-os-password", handler.VPSResetOSPassword)
		user.GET("/vps/:id/snapshots", handler.VPSSnapshots)
		user.POST("/vps/:id/snapshots", handler.VPSSnapshots)
		user.DELETE("/vps/:id/snapshots/:snapshotId", handler.VPSSnapshotDelete)
		user.POST("/vps/:id/snapshots/:snapshotId/restore", handler.VPSSnapshotRestore)
		user.GET("/vps/:id/backups", handler.VPSBackups)
		user.POST("/vps/:id/backups", handler.VPSBackups)
		user.DELETE("/vps/:id/backups/:backupId", handler.VPSBackupDelete)
		user.POST("/vps/:id/backups/:backupId/restore", handler.VPSBackupRestore)
		user.GET("/vps/:id/firewall", handler.VPSFirewallRules)
		user.POST("/vps/:id/firewall", handler.VPSFirewallRules)
		user.DELETE("/vps/:id/firewall/:ruleId", handler.VPSFirewallDelete)
		user.GET("/vps/:id/ports", handler.VPSPortMappings)
		user.POST("/vps/:id/ports", handler.VPSPortMappings)
		user.GET("/vps/:id/ports/candidates", handler.VPSPortCandidates)
		user.DELETE("/vps/:id/ports/:mappingId", handler.VPSPortMappingDelete)
		user.POST("/vps/:id/renew", handler.VPSRenewOrder)
		user.POST("/vps/:id/resize/quote", handler.VPSResizeQuote)
		user.POST("/vps/:id/resize", handler.VPSResizeOrder)
		user.POST("/vps/:id/emergency-renew", handler.VPSEmergencyRenew)
		user.POST("/vps/:id/refund", handler.VPSRefund)
	}

	integration := r.Group("/api/v1/integrations")
	integration.Use(middleware.RequireAPIKey())
	{
		integration.POST("/robot/webhook", handler.RobotWebhook)
	}

	admin := r.Group("/admin/api/v1")
	{
		admin.POST("/auth/login", handler.AdminLogin)
		admin.POST("/auth/refresh", handler.AdminRefresh)
		admin.GET("/avatar/qq/:qq", handler.AdminQQAvatar)
		admin.Use(middleware.RequireAdminPermissionAuto())
		admin.GET("/users", handler.AdminUsers)
		admin.POST("/users", handler.AdminUserCreate)
		admin.GET("/users/:id", handler.AdminUserDetail)
		admin.PATCH("/users/:id", handler.AdminUserUpdate)
		admin.POST("/users/:id/reset-password", handler.AdminUserResetPassword)
		admin.PATCH("/users/:id/status", handler.AdminUserStatus)
		admin.PATCH("/users/:id/realname-status", handler.AdminUserRealNameStatus)
		admin.POST("/users/:id/impersonate", handler.AdminUserImpersonate)
		admin.GET("/orders", handler.AdminOrders)
		admin.GET("/scheduled-tasks", handler.AdminScheduledTasks)
		admin.PATCH("/scheduled-tasks/:key", handler.AdminScheduledTaskUpdate)
		admin.GET("/scheduled-tasks/:key/runs", handler.AdminScheduledTaskRuns)
		admin.GET("/payments/providers", handler.AdminPaymentProviders)
		admin.PATCH("/payments/providers/:key", handler.AdminPaymentProviderUpdate)
		admin.POST("/plugins/payment/upload", handler.AdminPaymentPluginUpload)
		admin.GET("/plugins/payment-methods", handler.AdminPluginPaymentMethodsList)
		admin.PATCH("/plugins/payment-methods", handler.AdminPluginPaymentMethodsUpdate)
		admin.GET("/plugins", handler.AdminPluginsList)
		admin.GET("/plugins/discover", handler.AdminPluginsDiscover)
		admin.POST("/plugins/install", handler.AdminPluginInstall)
		admin.POST("/plugins/:category/:plugin_id/import", handler.AdminPluginImportFromDisk)
		admin.POST("/plugins/:category/:plugin_id/instances", handler.AdminPluginInstanceCreate)
		admin.POST("/plugins/:category/:plugin_id/:instance_id/enable", handler.AdminPluginInstanceEnable)
		admin.POST("/plugins/:category/:plugin_id/:instance_id/disable", handler.AdminPluginInstanceDisable)
		admin.DELETE("/plugins/:category/:plugin_id/:instance_id", handler.AdminPluginInstanceDelete)
		admin.GET("/plugins/:category/:plugin_id/:instance_id/config/schema", handler.AdminPluginInstanceConfigSchema)
		admin.GET("/plugins/:category/:plugin_id/:instance_id/config", handler.AdminPluginInstanceConfigGet)
		admin.PUT("/plugins/:category/:plugin_id/:instance_id/config", handler.AdminPluginInstanceConfigUpdate)
		admin.DELETE("/plugins/:category/:plugin_id/files", handler.AdminPluginDeleteFiles)
		admin.POST("/plugins/:category/:plugin_id/enable", handler.AdminPluginEnable)
		admin.POST("/plugins/:category/:plugin_id/disable", handler.AdminPluginDisable)
		admin.DELETE("/plugins/:category/:plugin_id", handler.AdminPluginUninstall)
		admin.GET("/plugins/:category/:plugin_id/config/schema", handler.AdminPluginConfigSchema)
		admin.GET("/plugins/:category/:plugin_id/config", handler.AdminPluginConfigGet)
		admin.PUT("/plugins/:category/:plugin_id/config", handler.AdminPluginConfigUpdate)
		admin.GET("/server/status", handler.AdminServerStatus)
		admin.GET("/debug/status", handler.AdminDebugStatus)
		admin.PATCH("/debug/status", handler.AdminDebugStatusUpdate)
		admin.GET("/debug/logs", handler.AdminDebugLogs)
		admin.GET("/orders/:id", handler.AdminOrderDetail)
		admin.POST("/orders/:id/approve", handler.AdminOrderApprove)
		admin.POST("/orders/:id/reject", handler.AdminOrderReject)
		admin.DELETE("/orders/:id", handler.AdminOrderDelete)
		admin.POST("/orders/:id/mark-paid", handler.AdminOrderMarkPaid)
		admin.POST("/orders/:id/retry", handler.AdminOrderRetry)
		admin.GET("/tickets", handler.AdminTickets)
		admin.GET("/tickets/:id", handler.AdminTicketDetail)
		admin.PATCH("/tickets/:id", handler.AdminTicketUpdate)
		admin.POST("/tickets/:id/messages", handler.AdminTicketMessageCreate)
		admin.DELETE("/tickets/:id", handler.AdminTicketDelete)
		admin.GET("/vps", handler.AdminVPSList)
		admin.POST("/vps", handler.AdminVPSCreate)
		admin.GET("/vps/:id", handler.AdminVPSDetail)
		admin.PATCH("/vps/:id", handler.AdminVPSUpdate)
		admin.POST("/vps/:id/lock", handler.AdminVPSLock)
		admin.POST("/vps/:id/unlock", handler.AdminVPSUnlock)
		admin.POST("/vps/:id/delete", handler.AdminVPSDelete)
		admin.POST("/vps/:id/resize", handler.AdminVPSResize)
		admin.POST("/vps/:id/status", handler.AdminVPSStatus)
		admin.POST("/vps/:id/emergency-renew", handler.AdminVPSEmergencyRenew)
		admin.POST("/vps/:id/refresh", handler.AdminVPSRefresh)
		admin.PATCH("/vps/:id/expire-at", handler.AdminVPSUpdateExpire)
		admin.GET("/audit-logs", handler.AdminAuditLogs)
		admin.GET("/regions", handler.AdminRegions)
		admin.POST("/regions", handler.AdminRegionCreate)
		admin.PATCH("/regions/:id", handler.AdminRegionUpdate)
		admin.DELETE("/regions/:id", handler.AdminRegionDelete)
		admin.POST("/regions/bulk-delete", handler.AdminRegionBulkDelete)
		admin.GET("/plan-groups", handler.AdminPlanGroups)
		admin.GET("/lines", handler.AdminLines)
		admin.POST("/plan-groups", handler.AdminPlanGroupCreate)
		admin.POST("/lines", handler.AdminLineCreate)
		admin.PATCH("/plan-groups/:id", handler.AdminPlanGroupUpdate)
		admin.PATCH("/lines/:id", handler.AdminLineUpdate)
		admin.POST("/lines/:id/system-images", handler.AdminLineSystemImages)
		admin.DELETE("/plan-groups/:id", handler.AdminPlanGroupDelete)
		admin.DELETE("/lines/:id", handler.AdminLineDelete)
		admin.POST("/plan-groups/bulk-delete", handler.AdminPlanGroupBulkDelete)
		admin.POST("/lines/bulk-delete", handler.AdminPlanGroupBulkDelete)
		admin.GET("/packages", handler.AdminPackages)
		admin.POST("/packages", handler.AdminPackageCreate)
		admin.PATCH("/packages/:id", handler.AdminPackageUpdate)
		admin.DELETE("/packages/:id", handler.AdminPackageDelete)
		admin.POST("/packages/bulk-delete", handler.AdminPackageBulkDelete)
		admin.GET("/billing-cycles", handler.AdminBillingCycles)
		admin.POST("/billing-cycles", handler.AdminBillingCycleCreate)
		admin.PATCH("/billing-cycles/:id", handler.AdminBillingCycleUpdate)
		admin.DELETE("/billing-cycles/:id", handler.AdminBillingCycleDelete)
		admin.POST("/billing-cycles/bulk-delete", handler.AdminBillingCycleBulkDelete)
		admin.GET("/system-images", handler.AdminSystemImages)
		admin.POST("/system-images", handler.AdminSystemImageCreate)
		admin.PATCH("/system-images/:id", handler.AdminSystemImageUpdate)
		admin.DELETE("/system-images/:id", handler.AdminSystemImageDelete)
		admin.POST("/system-images/bulk-delete", handler.AdminSystemImageBulkDelete)
		admin.POST("/system-images/sync", handler.AdminSystemImageSync)
		admin.GET("/integrations/automation", handler.AdminAutomationConfig)
		admin.PATCH("/integrations/automation", handler.AdminAutomationConfigUpdate)
		admin.POST("/integrations/automation/sync", handler.AdminAutomationSync)
		admin.GET("/integrations/automation/sync-logs", handler.AdminAutomationSyncLogs)

		admin.GET("/goods-types", handler.AdminGoodsTypes)
		admin.POST("/goods-types", handler.AdminGoodsTypeCreate)
		admin.POST("/goods-types/:id/sync-automation", handler.AdminGoodsTypeSyncAutomation)
		admin.PUT("/goods-types/:id", handler.AdminGoodsTypeUpdate)
		admin.DELETE("/goods-types/:id", handler.AdminGoodsTypeDelete)
		admin.GET("/integrations/robot", handler.AdminRobotConfig)
		admin.PATCH("/integrations/robot", handler.AdminRobotConfigUpdate)
		admin.POST("/integrations/robot/test", handler.AdminRobotTest)
		admin.GET("/realname/config", handler.AdminRealNameConfig)
		admin.PATCH("/realname/config", handler.AdminRealNameConfigUpdate)
		admin.GET("/realname/providers", handler.AdminRealNameProviders)
		admin.GET("/realname/records", handler.AdminRealNameRecords)
		admin.GET("/integrations/smtp", handler.AdminSMTPConfig)
		admin.PATCH("/integrations/smtp", handler.AdminSMTPConfigUpdate)
		admin.POST("/integrations/smtp/test", handler.AdminSMTPTest)
		admin.GET("/integrations/sms", handler.AdminSMSConfig)
		admin.PATCH("/integrations/sms", handler.AdminSMSConfigUpdate)
		admin.POST("/integrations/sms/test", handler.AdminSMSTest)
		admin.POST("/integrations/sms/preview", handler.AdminSMSPreview)
		// Backward-compatible aliases
		admin.GET("/integrations/sms/config", handler.AdminSMSConfig)
		admin.PATCH("/integrations/sms/config", handler.AdminSMSConfigUpdate)
		admin.POST("/integrations/sms/send-test", handler.AdminSMSTest)
		admin.GET("/api-keys", handler.AdminAPIKeys)
		admin.POST("/api-keys", handler.AdminAPIKeyCreate)
		admin.PATCH("/api-keys/:id", handler.AdminAPIKeyUpdate)
		admin.GET("/wallets/:user_id", handler.AdminWalletInfo)
		admin.POST("/wallets/:user_id/adjust", handler.AdminWalletAdjust)
		admin.GET("/wallets/:user_id/transactions", handler.AdminWalletTransactions)
		admin.GET("/wallet/orders", handler.AdminWalletOrders)
		admin.POST("/wallet/orders/:id/approve", handler.AdminWalletOrderApprove)
		admin.POST("/wallet/orders/:id/reject", handler.AdminWalletOrderReject)
		admin.GET("/settings", handler.AdminSettingsList)
		admin.PATCH("/settings", handler.AdminSettingsUpdate)
		admin.POST("/push-tokens", handler.AdminPushTokenRegister)
		admin.DELETE("/push-tokens", handler.AdminPushTokenDelete)
		admin.GET("/cms/categories", handler.AdminCMSCategories)
		admin.POST("/cms/categories", handler.AdminCMSCategoryCreate)
		admin.PATCH("/cms/categories/:id", handler.AdminCMSCategoryUpdate)
		admin.DELETE("/cms/categories/:id", handler.AdminCMSCategoryDelete)
		admin.GET("/cms/posts", handler.AdminCMSPosts)
		admin.POST("/cms/posts", handler.AdminCMSPostCreate)
		admin.PATCH("/cms/posts/:id", handler.AdminCMSPostUpdate)
		admin.DELETE("/cms/posts/:id", handler.AdminCMSPostDelete)
		admin.GET("/cms/blocks", handler.AdminCMSBlocks)
		admin.POST("/cms/blocks", handler.AdminCMSBlockCreate)
		admin.PATCH("/cms/blocks/:id", handler.AdminCMSBlockUpdate)
		admin.DELETE("/cms/blocks/:id", handler.AdminCMSBlockDelete)
		admin.GET("/uploads", handler.AdminUploads)
		admin.POST("/uploads", handler.AdminUploadCreate)
		admin.GET("/email-templates", handler.AdminEmailTemplates)
		admin.POST("/email-templates", handler.AdminEmailTemplateUpsert)
		admin.PATCH("/email-templates/:id", handler.AdminEmailTemplateUpsert)
		admin.DELETE("/email-templates/:id", handler.AdminEmailTemplateDelete)
		admin.GET("/sms-templates", handler.AdminSMSTemplates)
		admin.POST("/sms-templates", handler.AdminSMSTemplateUpsert)
		admin.PATCH("/sms-templates/:id", handler.AdminSMSTemplateUpsert)
		admin.DELETE("/sms-templates/:id", handler.AdminSMSTemplateDelete)
		// Backward-compatible aliases
		admin.GET("/sms/templates", handler.AdminSMSTemplates)
		admin.POST("/sms/templates", handler.AdminSMSTemplateUpsert)
		admin.PATCH("/sms/templates/:id", handler.AdminSMSTemplateUpsert)
		admin.DELETE("/sms/templates/:id", handler.AdminSMSTemplateDelete)
		admin.GET("/admins", handler.AdminAdmins)
		admin.POST("/admins", handler.AdminAdminCreate)
		admin.PATCH("/admins/:id", handler.AdminAdminUpdate)
		admin.PATCH("/admins/:id/status", handler.AdminAdminStatus)
		admin.DELETE("/admins/:id", handler.AdminAdminDelete)
		admin.GET("/permission-groups", handler.AdminPermissionGroups)
		admin.POST("/permission-groups", handler.AdminPermissionGroupCreate)
		admin.PATCH("/permission-groups/:id", handler.AdminPermissionGroupUpdate)
		admin.DELETE("/permission-groups/:id", handler.AdminPermissionGroupDelete)
		admin.GET("/permissions", handler.AdminPermissions)
		admin.GET("/permissions/list", handler.AdminPermissionsList)
		admin.GET("/permissions/:code", handler.AdminPermissionDetail)
		admin.PATCH("/permissions/:code", handler.AdminPermissionsUpdate)
		admin.POST("/permissions/sync", handler.AdminPermissionsSync)
		admin.GET("/profile", handler.AdminProfile)
		admin.PATCH("/profile", handler.AdminProfileUpdate)
		admin.POST("/profile/change-password", handler.AdminProfileChangePassword)
		admin.POST("/dashboard/overview", handler.AdminDashboardOverview)
		admin.POST("/dashboard/revenue", handler.AdminDashboardRevenue)
		admin.GET("/dashboard/vps-status", handler.AdminDashboardVPSStatus)
		admin.GET("/probes", handler.AdminProbes)
		admin.POST("/probes", handler.AdminProbeCreate)
		admin.GET("/probes/:id", handler.AdminProbeDetail)
		admin.PATCH("/probes/:id", handler.AdminProbeUpdate)
		admin.DELETE("/probes/:id", handler.AdminProbeDelete)
		admin.POST("/probes/:id/enroll-token/reset", handler.AdminProbeResetEnrollToken)
		admin.GET("/probes/:id/sla", handler.AdminProbeSLA)
		admin.POST("/probes/:id/port-check", handler.AdminProbePortCheck)
		admin.POST("/probes/:id/log-sessions", handler.AdminProbeLogSessionCreate)
		admin.GET("/probes/:id/log-sessions/:sid/stream", handler.AdminProbeLogSessionStream)
	}

	public.POST("/auth/forgot-password", handler.AdminForgotPassword)
	public.POST("/auth/reset-password", handler.AdminResetPassword)

	return &Server{Engine: r}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		if origin == "" || !isAllowedLocalOrigin(origin) {
			c.Next()
			return
		}

		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type,Accept,X-API-Key,X-API-Version")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == http.MethodOptions {
			c.Status(http.StatusNoContent)
			c.Abort()
			return
		}
		c.Next()
	}
}

func isAllowedLocalOrigin(origin string) bool {
	lower := strings.ToLower(origin)
	if strings.HasPrefix(lower, "http://localhost:") || strings.HasPrefix(lower, "https://localhost:") {
		return true
	}
	if strings.HasPrefix(lower, "http://127.0.0.1:") || strings.HasPrefix(lower, "https://127.0.0.1:") {
		return true
	}
	if strings.HasPrefix(lower, "http://[::1]:") || strings.HasPrefix(lower, "https://[::1]:") {
		return true
	}
	return false
}

func installGateMiddleware(handler *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		if handler == nil || handler.IsInstalled() {
			c.Next()
			return
		}

		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/v1/install") {
			c.Next()
			return
		}
		if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/admin/api/") {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "not installed"})
			return
		}

		// Allow uploads and static assets so the installer page can load.
		if strings.HasPrefix(path, "/uploads/") || strings.HasPrefix(path, "/assets/") || path == "/favicon.ico" {
			c.Next()
			return
		}

		// Allow direct access to installer page itself.
		if path == "/install" || strings.HasPrefix(path, "/install/") {
			c.Next()
			return
		}

		// Browser navigation: redirect to /install.
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead {
			c.Redirect(http.StatusFound, "/install")
			c.Abort()
			return
		}

		c.AbortWithStatus(http.StatusNotFound)
	}
}

func spaStaticFileMiddleware(staticDir string, excludedPrefixes []string) gin.HandlerFunc {
	staticAbs, staticAbsErr := filepath.Abs(staticDir)
	staticAbs = filepath.Clean(staticAbs)

	return func(c *gin.Context) {
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.Next()
			return
		}

		reqPath := c.Request.URL.Path
		for _, p := range excludedPrefixes {
			if strings.HasPrefix(reqPath, p) {
				c.Next()
				return
			}
		}

		// Map URL path -> filesystem path under staticDir.
		rel := strings.TrimPrefix(reqPath, "/")
		target := filepath.Join(staticDir, filepath.FromSlash(rel))
		targetAbs, err := filepath.Abs(target)
		if err != nil {
			c.Next()
			return
		}
		targetAbs = filepath.Clean(targetAbs)

		// Basic path traversal guard: only serve files within staticDir.
		if staticAbsErr == nil {
			if targetAbs != staticAbs && !strings.HasPrefix(targetAbs, staticAbs+string(os.PathSeparator)) {
				c.Next()
				return
			}
		}

		st, err := os.Stat(targetAbs)
		if err != nil || st.IsDir() {
			c.Next()
			return
		}

		c.File(targetAbs)
		c.Abort()
	}
}

func spaIndexFallbackHandler(staticDir string, excludedPrefixes []string) gin.HandlerFunc {
	indexPath := filepath.Join(staticDir, "index.html")

	return func(c *gin.Context) {
		reqPath := c.Request.URL.Path
		for _, p := range excludedPrefixes {
			if strings.HasPrefix(reqPath, p) {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
		}

		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		if _, err := os.Stat(indexPath); err == nil {
			c.File(indexPath)
			return
		}
		c.AbortWithStatus(http.StatusNotFound)
	}
}

func NewHandlerWithServices(auth *usecase.AuthService, catalog *usecase.CatalogService, goodsTypes *usecase.GoodsTypeService, cart *usecase.CartService, orders *usecase.OrderService, vps *usecase.VPSService, admin *usecase.AdminService, adminVPS *usecase.AdminVPSService, integration *usecase.IntegrationService, reportSvc *usecase.ReportService, cmsSvc *usecase.CMSService, ticketSvc *usecase.TicketService, walletSvc *usecase.WalletService, walletOrders *usecase.WalletOrderService, paymentSvc *usecase.PaymentService, messageSvc *usecase.MessageCenterService, statusSvc *usecase.ServerStatusService, realnameSvc *usecase.RealNameService, orderItems usecase.OrderItemRepository, users usecase.UserRepository, orderRepo usecase.OrderRepository, vpsRepo usecase.VPSRepository, payments usecase.PaymentRepository, events usecase.EventRepository, automationLogs usecase.AutomationLogRepository, settings usecase.SettingsRepository, permissions usecase.PermissionRepository, uploads usecase.UploadRepository, broker *sse.Broker, jwtSecret string, passwordReset *usecase.PasswordResetService, permissionSvc *usecase.PermissionService, taskSvc *usecase.ScheduledTaskService) *Handler {
	return NewHandler(auth, catalog, goodsTypes, cart, orders, vps, admin, adminVPS, integration, reportSvc, cmsSvc, ticketSvc, walletSvc, walletOrders, paymentSvc, messageSvc, statusSvc, realnameSvc, orderItems, users, orderRepo, vpsRepo, payments, events, automationLogs, settings, permissions, uploads, broker, jwtSecret, passwordReset, permissionSvc, taskSvc)
}
