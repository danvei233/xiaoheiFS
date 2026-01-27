package usecase_test

import (
	"context"
	"testing"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/usecase"
)

func TestOrderService_SubmitPaymentRobotItems(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	user := domain.User{
		Username:     "buyer1",
		Email:        "buyer1@example.com",
		PasswordHash: "hash",
		Role:         domain.UserRoleUser,
		Status:       domain.UserStatusActive,
	}
	if err := repo.CreateUser(ctx, &user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	region := domain.Region{Code: "r1", Name: "Region1", Active: true}
	if err := repo.CreateRegion(ctx, &region); err != nil {
		t.Fatalf("create region: %v", err)
	}
	plan := domain.PlanGroup{RegionID: region.ID, Name: "Plan1", LineID: 1, UnitCore: 1, UnitMem: 1, UnitDisk: 1, UnitBW: 1, Active: true, Visible: true}
	if err := repo.CreatePlanGroup(ctx, &plan); err != nil {
		t.Fatalf("create plan group: %v", err)
	}
	pkg := domain.Package{PlanGroupID: plan.ID, Name: "Pkg1", Cores: 1, MemoryGB: 1, DiskGB: 10, BandwidthMB: 10, Monthly: 100, PortNum: 1, Active: true, Visible: true}
	if err := repo.CreatePackage(ctx, &pkg); err != nil {
		t.Fatalf("create package: %v", err)
	}
	img := domain.SystemImage{ImageID: 1, Name: "Ubuntu", Type: "linux", Enabled: true}
	if err := repo.CreateSystemImage(ctx, &img); err != nil {
		t.Fatalf("create image: %v", err)
	}

	order := domain.Order{
		UserID:      user.ID,
		OrderNo:     "ORD-ROBOT-1",
		Status:      domain.OrderStatusPendingPayment,
		TotalAmount: 1000,
		Currency:    "CNY",
	}
	if err := repo.CreateOrder(ctx, &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	if err := repo.CreateOrderItems(ctx, []domain.OrderItem{{
		OrderID:   order.ID,
		PackageID: pkg.ID,
		SystemID:  img.ID,
		SpecJSON:  "{}",
		Qty:       1,
		Amount:    1000,
		Status:    domain.OrderItemStatusPendingPayment,
		Action:    "create",
	}}); err != nil {
		t.Fatalf("create order items: %v", err)
	}

	robot := &testutil.FakeRobotNotifier{}
	orderSvc := usecase.NewOrderService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, nil, robot, repo, repo, nil, repo, repo, repo, nil, nil, nil)

	if _, err := orderSvc.SubmitPayment(ctx, user.ID, order.ID, usecase.PaymentInput{
		Method:   "manual",
		Amount:   1000,
		Currency: "CNY",
		TradeNo:  "TRADE-1",
	}, ""); err != nil {
		t.Fatalf("submit payment: %v", err)
	}
	if len(robot.Payload) != 1 || len(robot.Payload[0].Items) != 1 {
		t.Fatalf("robot payload items missing")
	}

	if _, total, err := orderSvc.ListOrders(ctx, usecase.OrderFilter{UserID: user.ID}, 10, 0); err != nil || total == 0 {
		t.Fatalf("order list: %v", err)
	}
}

func TestAutomationLogContext(t *testing.T) {
	ctx := context.Background()
	if _, ok := usecase.GetAutomationLogContext(ctx); ok {
		t.Fatalf("expected no context")
	}
	ctx = usecase.WithAutomationLogContext(ctx, 10, 20)
	logCtx, ok := usecase.GetAutomationLogContext(ctx)
	if !ok || logCtx.OrderID != 10 || logCtx.OrderItemID != 20 {
		t.Fatalf("context mismatch")
	}
}
