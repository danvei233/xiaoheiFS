package testutilhttp

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"

	adapteremail "xiaoheiplay/internal/adapter/email"
	"xiaoheiplay/internal/adapter/http"
	"xiaoheiplay/internal/adapter/repo/core"
	"xiaoheiplay/internal/adapter/sse"
	appadmin "xiaoheiplay/internal/app/admin"
	appadminvps "xiaoheiplay/internal/app/adminvps"
	appauth "xiaoheiplay/internal/app/auth"
	appautomationlog "xiaoheiplay/internal/app/automationlog"
	appcart "xiaoheiplay/internal/app/cart"
	appcatalog "xiaoheiplay/internal/app/catalog"
	appcms "xiaoheiplay/internal/app/cms"
	appgoodstype "xiaoheiplay/internal/app/goodstype"
	appintegration "xiaoheiplay/internal/app/integration"
	appmessage "xiaoheiplay/internal/app/message"
	appnotification "xiaoheiplay/internal/app/notification"
	apporder "xiaoheiplay/internal/app/order"
	apporderevent "xiaoheiplay/internal/app/orderevent"
	apppayment "xiaoheiplay/internal/app/payment"
	apppermission "xiaoheiplay/internal/app/permission"
	apprealname "xiaoheiplay/internal/app/realname"
	appreport "xiaoheiplay/internal/app/report"
	appsecurityticket "xiaoheiplay/internal/app/securityticket"
	appsettings "xiaoheiplay/internal/app/settings"
	appticket "xiaoheiplay/internal/app/ticket"
	appupload "xiaoheiplay/internal/app/upload"
	appvps "xiaoheiplay/internal/app/vps"
	appwallet "xiaoheiplay/internal/app/wallet"
	appwalletorder "xiaoheiplay/internal/app/walletorder"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

type Env struct {
	Repo          *repo.GormRepo
	JWTSecret     string
	Automation    *testutil.FakeAutomationClient
	PaymentReg    *testutil.FakePaymentRegistry
	Email         *testutil.FakeEmailSender
	Robot         *testutil.FakeRobotNotifier
	RealnameReg   *testutil.FakeRealNameRegistry
	Broker        *sse.Broker
	AuthSvc       *appauth.Service
	CatalogSvc    *appcatalog.Service
	CartSvc       *appcart.Service
	OrderSvc      *apporder.Service
	VpsSvc        *appvps.Service
	AdminSvc      *appadmin.Service
	AdminVPSSvc   *appadminvps.Service
	PermissionSvc *apppermission.Service
	PaymentSvc    *apppayment.Service
	MessageSvc    *appmessage.Service
	WalletSvc     *appwallet.Service
	WalletOrder   *appwalletorder.Service
	NotifySvc     *appnotification.Service
	Handler       *http.Handler
	Router        *gin.Engine
}

func NewTestEnv(t *testing.T, withCMS bool) *Env {
	t.Helper()
	gin.SetMode(gin.TestMode)
	_, repoSQLite := testutil.NewTestDB(t, withCMS)
	// Keep test flows deterministic: disable register verification unless tests opt in.
	_ = repoSQLite.UpsertSetting(context.Background(), domain.Setting{Key: "auth_register_verify_type", ValueJSON: "none"})
	_ = repoSQLite.UpsertSetting(context.Background(), domain.Setting{Key: "auth_register_verify_channels", ValueJSON: `[]`})
	_ = repoSQLite.UpsertSetting(context.Background(), domain.Setting{Key: "auth_2fa_enabled", ValueJSON: "false"})
	automation := &testutil.FakeAutomationClient{}
	automationResolver := &testutil.FakeAutomationResolver{Client: automation}
	paymentReg := testutil.NewFakePaymentRegistry()
	email := &testutil.FakeEmailSender{}
	robot := &testutil.FakeRobotNotifier{}
	realnameReg := testutil.NewFakeRealNameRegistry()
	broker := sse.NewBroker(repoSQLite)

	catalogSvc := appcatalog.NewService(repoSQLite, repoSQLite, repoSQLite)
	goodsTypeSvc := appgoodstype.NewService(repoSQLite, repoSQLite)
	cartSvc := appcart.NewService(repoSQLite, repoSQLite, repoSQLite)
	messageSvc := appmessage.NewService(repoSQLite, repoSQLite)
	realnameSvc := apprealname.NewService(repoSQLite, realnameReg, repoSQLite)
	orderSvc := apporder.NewService(repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, broker, automationResolver, robot, repoSQLite, repoSQLite, email, repoSQLite, repoSQLite, repoSQLite, repoSQLite, messageSvc, realnameSvc)
	vpsSvc := appvps.NewService(repoSQLite, automationResolver, repoSQLite)
	adminSvc := appadmin.NewService(repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite)
	adminVPSSvc := appadminvps.NewService(repoSQLite, automationResolver, repoSQLite, repoSQLite, repoSQLite, messageSvc)
	authSvc := appauth.NewService(repoSQLite, repoSQLite, repoSQLite)
	permissionSvc := apppermission.NewService(repoSQLite, repoSQLite, repoSQLite)
	paymentSvc := apppayment.NewService(repoSQLite, repoSQLite, repoSQLite, paymentReg, repoSQLite, orderSvc, broker)
	walletSvc := appwallet.NewService(repoSQLite, repoSQLite)
	walletOrderSvc := appwalletorder.NewService(repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, automationResolver, repoSQLite)
	uploadSvc := appupload.NewService(repoSQLite)
	autoLogSvc := appautomationlog.NewService(repoSQLite)
	orderEventSvc := apporderevent.NewService(repoSQLite)
	securityTicketSvc := appsecurityticket.NewService(repoSQLite)
	settingsSvc := appsettings.NewService(repoSQLite)
	notifySvc := appnotification.NewService(repoSQLite, repoSQLite, repoSQLite, email, messageSvc)
	integrationSvc := appintegration.NewService(repoSQLite, repoSQLite, repoSQLite, repoSQLite, automationResolver, repoSQLite)
	reportSvc := appreport.NewService(repoSQLite, repoSQLite, repoSQLite)
	cmsSvc := appcms.NewService(repoSQLite, repoSQLite, repoSQLite, messageSvc)
	ticketSvc := appticket.NewService(repoSQLite, repoSQLite, repoSQLite, messageSvc)
	seedDefaultGoodsType(t, repoSQLite)

	jwtSecret := "test-secret"
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
		RealnameSvc:       realnameSvc,
		OrderEventSvc:     orderEventSvc,
		AutoLogSvc:        autoLogSvc,
		SettingsSvc:       settingsSvc,
		UploadSvc:         uploadSvc,
		Broker:            broker,
		JWTSecret:         jwtSecret,
		SecurityTicketSvc: securityTicketSvc,
		PermissionSvc:     permissionSvc,
		EmailSender:       adapteremail.NewSender(repoSQLite),
	})
	middleware := http.NewMiddleware(jwtSecret, nil, permissionSvc, authSvc, settingsSvc)
	server := http.NewServer(handler, middleware)

	return &Env{
		Repo:          repoSQLite,
		JWTSecret:     jwtSecret,
		Automation:    automation,
		PaymentReg:    paymentReg,
		Email:         email,
		Robot:         robot,
		RealnameReg:   realnameReg,
		Broker:        broker,
		AuthSvc:       authSvc,
		CatalogSvc:    catalogSvc,
		CartSvc:       cartSvc,
		OrderSvc:      orderSvc,
		VpsSvc:        vpsSvc,
		AdminSvc:      adminSvc,
		AdminVPSSvc:   adminVPSSvc,
		PermissionSvc: permissionSvc,
		PaymentSvc:    paymentSvc,
		MessageSvc:    messageSvc,
		WalletSvc:     walletSvc,
		WalletOrder:   walletOrderSvc,
		NotifySvc:     notifySvc,
		Handler:       handler,
		Router:        server.Engine,
	}
}

func seedDefaultGoodsType(t *testing.T, repoSQLite *repo.GormRepo) {
	t.Helper()
	ctx := context.Background()
	items, err := repoSQLite.ListGoodsTypes(ctx)
	if err == nil && len(items) > 0 {
		return
	}
	gt := domain.GoodsType{
		Code:                 "__env_default__",
		Name:                 "Env Default",
		Active:               true,
		SortOrder:            1,
		AutomationCategory:   "automation",
		AutomationPluginID:   "lightboat",
		AutomationInstanceID: "default",
	}
	if err := repoSQLite.CreateGoodsType(ctx, &gt); err != nil {
		t.Fatalf("seed default goods type: %v", err)
	}
}
