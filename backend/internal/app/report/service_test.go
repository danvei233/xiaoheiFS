package report_test

import (
	"context"
	"testing"
	"time"

	appreport "xiaoheiplay/internal/app/report"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func TestRevenueAnalyticsOverviewAndDetails(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()

	user := testutil.CreateUser(t, repo, "ra_user", "ra_user@example.com", "pass")
	gt := domain.GoodsType{Code: "vps", Name: "云主机", Active: true}
	if err := repo.CreateGoodsType(ctx, &gt); err != nil {
		t.Fatalf("create goods type: %v", err)
	}
	region := domain.Region{GoodsTypeID: gt.ID, Code: "cn-hz", Name: "华东", Active: true}
	if err := repo.CreateRegion(ctx, &region); err != nil {
		t.Fatalf("create region: %v", err)
	}
	plan := domain.PlanGroup{GoodsTypeID: gt.ID, RegionID: region.ID, LineID: 1001, Name: "BGP", UnitCore: 1, UnitMem: 1, UnitDisk: 1, UnitBW: 1, Active: true, Visible: true}
	if err := repo.CreatePlanGroup(ctx, &plan); err != nil {
		t.Fatalf("create plan: %v", err)
	}
	pkg := domain.Package{GoodsTypeID: gt.ID, PlanGroupID: plan.ID, Name: "1C1G", Active: true, Visible: true}
	if err := repo.CreatePackage(ctx, &pkg); err != nil {
		t.Fatalf("create package: %v", err)
	}

	order := domain.Order{UserID: user.ID, OrderNo: "ORD-RA-1", Status: domain.OrderStatusApproved, TotalAmount: 5000, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	if err := repo.CreateOrderItems(ctx, []domain.OrderItem{{OrderID: order.ID, PackageID: pkg.ID, GoodsTypeID: gt.ID, Amount: 5000, Qty: 1, Status: domain.OrderItemStatusApproved, Action: "create", SpecJSON: "{}"}}); err != nil {
		t.Fatalf("create order items: %v", err)
	}
	pay := domain.OrderPayment{OrderID: order.ID, UserID: user.ID, Method: "manual", Amount: 5000, Currency: "CNY", TradeNo: "TRADE-RA-1", Status: domain.PaymentStatusApproved}
	if err := repo.CreatePayment(ctx, &pay); err != nil {
		t.Fatalf("create payment: %v", err)
	}

	svc := appreport.NewService(repo, repo, repo, repo, repo, repo)
	query := appreport.RevenueAnalyticsQuery{
		FromAt:      time.Now().Add(-24 * time.Hour),
		ToAt:        time.Now().Add(24 * time.Hour),
		Level:       appreport.RevenueLevelGoodsType,
		GoodsTypeID: gt.ID,
	}

	overview, err := svc.RevenueAnalyticsOverview(ctx, query)
	if err != nil {
		t.Fatalf("overview: %v", err)
	}
	if overview.Summary.TotalRevenueCents <= 0 {
		t.Fatalf("expected positive revenue")
	}

	details, total, err := svc.RevenueAnalyticsDetails(ctx, query)
	if err != nil {
		t.Fatalf("details: %v", err)
	}
	if total == 0 || len(details) == 0 {
		t.Fatalf("expected detail rows")
	}

	overallQuery := appreport.RevenueAnalyticsQuery{
		FromAt: time.Now().Add(-24 * time.Hour),
		ToAt:   time.Now().Add(24 * time.Hour),
		Level:  appreport.RevenueLevelOverall,
	}
	overall, err := svc.RevenueAnalyticsOverview(ctx, overallQuery)
	if err != nil {
		t.Fatalf("overall overview: %v", err)
	}
	if overall.Summary.TotalRevenueCents <= 0 {
		t.Fatalf("expected overall positive revenue")
	}
}
