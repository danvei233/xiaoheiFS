package testutilhttp

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"

	"xiaoheiplay/internal/adapter/http"
	"xiaoheiplay/internal/adapter/repo"
	"xiaoheiplay/internal/adapter/sse"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/usecase"
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
	AuthSvc       *usecase.AuthService
	CatalogSvc    *usecase.CatalogService
	CartSvc       *usecase.CartService
	OrderSvc      *usecase.OrderService
	VpsSvc        *usecase.VPSService
	AdminSvc      *usecase.AdminService
	AdminVPSSvc   *usecase.AdminVPSService
	PermissionSvc *usecase.PermissionService
	PaymentSvc    *usecase.PaymentService
	MessageSvc    *usecase.MessageCenterService
	WalletSvc     *usecase.WalletService
	WalletOrder   *usecase.WalletOrderService
	NotifySvc     *usecase.NotificationService
	Handler       *http.Handler
	Router        *gin.Engine
}

func NewTestEnv(t *testing.T, withCMS bool) *Env {
	t.Helper()
	gin.SetMode(gin.TestMode)
	_, repoSQLite := testutil.NewTestDB(t, withCMS)
	automation := &testutil.FakeAutomationClient{}
	automationResolver := &testutil.FakeAutomationResolver{Client: automation}
	paymentReg := testutil.NewFakePaymentRegistry()
	email := &testutil.FakeEmailSender{}
	robot := &testutil.FakeRobotNotifier{}
	realnameReg := testutil.NewFakeRealNameRegistry()
	broker := sse.NewBroker(repoSQLite)

	catalogSvc := usecase.NewCatalogService(repoSQLite, repoSQLite, repoSQLite)
	goodsTypeSvc := usecase.NewGoodsTypeService(repoSQLite, repoSQLite)
	cartSvc := usecase.NewCartService(repoSQLite, repoSQLite, repoSQLite)
	messageSvc := usecase.NewMessageCenterService(repoSQLite, repoSQLite)
	realnameSvc := usecase.NewRealNameService(repoSQLite, realnameReg, repoSQLite)
	orderSvc := usecase.NewOrderService(repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, broker, automationResolver, robot, repoSQLite, repoSQLite, email, repoSQLite, repoSQLite, repoSQLite, repoSQLite, messageSvc, realnameSvc)
	vpsSvc := usecase.NewVPSService(repoSQLite, automationResolver, repoSQLite)
	adminSvc := usecase.NewAdminService(repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite)
	adminVPSSvc := usecase.NewAdminVPSService(repoSQLite, automationResolver, repoSQLite, repoSQLite, repoSQLite, messageSvc)
	authSvc := usecase.NewAuthService(repoSQLite, repoSQLite, repoSQLite)
	permissionSvc := usecase.NewPermissionService(repoSQLite, repoSQLite, repoSQLite)
	paymentSvc := usecase.NewPaymentService(repoSQLite, repoSQLite, repoSQLite, paymentReg, repoSQLite, orderSvc, broker)
	walletSvc := usecase.NewWalletService(repoSQLite, repoSQLite)
	walletOrderSvc := usecase.NewWalletOrderService(repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, automationResolver, repoSQLite)
	notifySvc := usecase.NewNotificationService(repoSQLite, repoSQLite, repoSQLite, email, messageSvc)
	integrationSvc := usecase.NewIntegrationService(repoSQLite, repoSQLite, repoSQLite, repoSQLite, automationResolver, repoSQLite)
	reportSvc := usecase.NewReportService(repoSQLite, repoSQLite, repoSQLite)
	cmsSvc := usecase.NewCMSService(repoSQLite, repoSQLite, repoSQLite, messageSvc)
	ticketSvc := usecase.NewTicketService(repoSQLite, repoSQLite, repoSQLite, messageSvc)
	seedDefaultGoodsType(t, repoSQLite)

	jwtSecret := "test-secret"
	handler := http.NewHandler(
		authSvc, catalogSvc, goodsTypeSvc, cartSvc, orderSvc, vpsSvc,
		adminSvc, adminVPSSvc, integrationSvc, reportSvc, cmsSvc, ticketSvc, walletSvc, walletOrderSvc,
		paymentSvc, messageSvc, nil, realnameSvc, repoSQLite, repoSQLite, repoSQLite,
		repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite, repoSQLite,
		broker, jwtSecret, nil, permissionSvc, nil,
	)
	middleware := http.NewMiddleware(jwtSecret, nil, permissionSvc)
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
