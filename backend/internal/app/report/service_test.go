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

func TestOverviewRevenueUsesOrderStatusScope(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()

	user := testutil.CreateUser(t, repo, "overview_user", "overview_user@example.com", "pass")
	orders := []domain.Order{
		{UserID: user.ID, OrderNo: "ORD-OV-1", Status: domain.OrderStatusApproved, TotalAmount: 24000, Currency: "CNY"},
		{UserID: user.ID, OrderNo: "ORD-OV-2", Status: domain.OrderStatusPendingReview, TotalAmount: 12000, Currency: "CNY"},
		{UserID: user.ID, OrderNo: "ORD-OV-3", Status: domain.OrderStatusFailed, TotalAmount: 5000, Currency: "CNY"},
		{UserID: user.ID, OrderNo: "ORD-OV-4", Status: domain.OrderStatusPendingPayment, TotalAmount: 8000, Currency: "CNY"},
		{UserID: user.ID, OrderNo: "ORD-OV-5", Status: domain.OrderStatusApproved, TotalAmount: -3000, Currency: "CNY"},
		{UserID: user.ID, OrderNo: "ORD-OV-6", Status: domain.OrderStatusApproved, TotalAmount: 6000, Currency: "CNY"},
	}
	for i := range orders {
		if err := repo.CreateOrder(ctx, &orders[i]); err != nil {
			t.Fatalf("create order %d: %v", i, err)
		}
	}
	// ORD-OV-1 is approved with a confirmed payment: it counts as revenue.
	// ORD-OV-6 is approved but has NO approved payment (e.g. an admin approval
	// of a pending_payment order whose provider session was never confirmed):
	// it must NOT count, or the dashboard disagrees with the analytics module.
	if err := repo.CreatePayment(ctx, &domain.OrderPayment{
		OrderID: orders[0].ID, UserID: user.ID, Method: "alipay", Amount: 24000, Currency: "CNY",
		Status: domain.PaymentStatusApproved,
	}); err != nil {
		t.Fatalf("create approved payment: %v", err)
	}

	svc := appreport.NewService(repo, repo, repo, repo, repo, repo)
	overview, err := svc.Overview(ctx)
	if err != nil {
		t.Fatalf("overview: %v", err)
	}

	// Include approved orders with a confirmed payment and keep refunds
	// (negative); exclude pending_review (money not yet confirmed), failed,
	// pending_payment, and approved orders without an approved payment.
	wantRevenue := int64(24000 - 3000)
	if overview.Revenue != wantRevenue {
		t.Fatalf("unexpected revenue: got %d want %d", overview.Revenue, wantRevenue)
	}
	if overview.PendingReview != 1 {
		t.Fatalf("unexpected pending review count: got %d", overview.PendingReview)
	}
}

// TestRevenueDetailsRequiresApprovedPayment locks the F3 fix: a positive order
// with no approved payment (e.g. pending_review with an unconfirmed manual
// proof) must not appear in revenue details as an approved payment record, and
// must not inflate the total.
func TestRevenueDetailsRequiresApprovedPayment(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()

	user := testutil.CreateUser(t, repo, "rev_nopay", "rev_nopay@example.com", "pass")
	gt := domain.GoodsType{Code: "vps2", Name: "云主机", Active: true}
	if err := repo.CreateGoodsType(ctx, &gt); err != nil {
		t.Fatalf("create goods type: %v", err)
	}
	region := domain.Region{GoodsTypeID: gt.ID, Code: "cn-hz2", Name: "华东", Active: true}
	if err := repo.CreateRegion(ctx, &region); err != nil {
		t.Fatalf("create region: %v", err)
	}
	plan := domain.PlanGroup{GoodsTypeID: gt.ID, RegionID: region.ID, LineID: 2001, Name: "BGP", UnitCore: 1, UnitMem: 1, UnitDisk: 1, UnitBW: 1, Active: true, Visible: true}
	if err := repo.CreatePlanGroup(ctx, &plan); err != nil {
		t.Fatalf("create plan: %v", err)
	}
	pkg := domain.Package{GoodsTypeID: gt.ID, PlanGroupID: plan.ID, Name: "1C1G", Active: true, Visible: true}
	if err := repo.CreatePackage(ctx, &pkg); err != nil {
		t.Fatalf("create package: %v", err)
	}

	// Order A: pending_review, positive, NO approved payment → excluded.
	orderNoPay := domain.Order{UserID: user.ID, OrderNo: "ORD-NOPAY", Status: domain.OrderStatusPendingReview, TotalAmount: 12000, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &orderNoPay); err != nil {
		t.Fatalf("create no-pay order: %v", err)
	}
	if err := repo.CreateOrderItems(ctx, []domain.OrderItem{{OrderID: orderNoPay.ID, PackageID: pkg.ID, GoodsTypeID: gt.ID, Amount: 12000, Qty: 1, Status: domain.OrderItemStatusPendingReview, Action: "create", SpecJSON: "{}"}}); err != nil {
		t.Fatalf("create no-pay items: %v", err)
	}

	// Order B: approved, positive, WITH an approved payment → included.
	orderPaid := domain.Order{UserID: user.ID, OrderNo: "ORD-PAID", Status: domain.OrderStatusApproved, TotalAmount: 5000, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &orderPaid); err != nil {
		t.Fatalf("create paid order: %v", err)
	}
	if err := repo.CreateOrderItems(ctx, []domain.OrderItem{{OrderID: orderPaid.ID, PackageID: pkg.ID, GoodsTypeID: gt.ID, Amount: 5000, Qty: 1, Status: domain.OrderItemStatusApproved, Action: "create", SpecJSON: "{}"}}); err != nil {
		t.Fatalf("create paid items: %v", err)
	}
	pay := domain.OrderPayment{OrderID: orderPaid.ID, UserID: user.ID, Method: "manual", Amount: 5000, Currency: "CNY", TradeNo: "TRADE-PAID", Status: domain.PaymentStatusApproved}
	if err := repo.CreatePayment(ctx, &pay); err != nil {
		t.Fatalf("create payment: %v", err)
	}

	// Order C: approved, negative refund (no order_payment row) → still reduces revenue.
	orderRefund := domain.Order{UserID: user.ID, OrderNo: "ORD-REF", Status: domain.OrderStatusApproved, TotalAmount: -2000, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &orderRefund); err != nil {
		t.Fatalf("create refund order: %v", err)
	}
	if err := repo.CreateOrderItems(ctx, []domain.OrderItem{{OrderID: orderRefund.ID, PackageID: pkg.ID, GoodsTypeID: gt.ID, Amount: -2000, Qty: 1, Status: domain.OrderItemStatusApproved, Action: "refund", SpecJSON: "{}"}}); err != nil {
		t.Fatalf("create refund items: %v", err)
	}

	svc := appreport.NewService(repo, repo, repo, repo, repo, repo)
	query := appreport.RevenueAnalyticsQuery{
		FromAt:      time.Now().Add(-24 * time.Hour),
		ToAt:        time.Now().Add(24 * time.Hour),
		Level:       appreport.RevenueLevelGoodsType,
		GoodsTypeID: gt.ID,
	}
	details, total, err := svc.RevenueAnalyticsDetails(ctx, query)
	if err != nil {
		t.Fatalf("details: %v", err)
	}
	// `total` here is the number of detail rows (paid order + refund order).
	if total != 2 {
		t.Fatalf("expected 2 detail rows, got %d", total)
	}
	var sum int64
	for _, d := range details {
		if d.OrderID == orderNoPay.ID {
			t.Fatalf("pending_review order without an approved payment must not appear in revenue details")
		}
		if d.PaymentID < 0 && d.OrderID == orderPaid.ID {
			t.Fatalf("paid order must reference its real approved payment, got fabricated id %d", d.PaymentID)
		}
		sum += d.AmountCents
	}
	if sum != 3000 {
		t.Fatalf("expected revenue 5000-2000=3000, got %d", sum)
	}
}

// TestOverviewRevenueExcludesRejectedAndDraft locks the P1-15 revenue scope:
// rejected and draft orders must never count toward revenue (they inflate the
// report with money that was never received).
func TestOverviewRevenueExcludesRejectedAndDraft(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()

	user := testutil.CreateUser(t, repo, "rev_reject", "rev_reject@example.com", "pass")
	orders := []domain.Order{
		{UserID: user.ID, OrderNo: "ORD-RJ-1", Status: domain.OrderStatusRejected, TotalAmount: 50000, Currency: "CNY"},
		{UserID: user.ID, OrderNo: "ORD-DR-1", Status: domain.OrderStatusDraft, TotalAmount: 40000, Currency: "CNY"},
		{UserID: user.ID, OrderNo: "ORD-OK-1", Status: domain.OrderStatusApproved, TotalAmount: 10000, Currency: "CNY"},
	}
	for i := range orders {
		if err := repo.CreateOrder(ctx, &orders[i]); err != nil {
			t.Fatalf("create order %d: %v", i, err)
		}
	}
	// ORD-OK-1 is the only order that settled (has an approved payment).
	if err := repo.CreatePayment(ctx, &domain.OrderPayment{
		OrderID: orders[2].ID, UserID: user.ID, Method: "alipay", Amount: 10000, Currency: "CNY",
		Status: domain.PaymentStatusApproved,
	}); err != nil {
		t.Fatalf("create approved payment: %v", err)
	}

	svc := appreport.NewService(repo, repo, repo, repo, repo, repo)
	overview, err := svc.Overview(ctx)
	if err != nil {
		t.Fatalf("overview: %v", err)
	}
	if overview.Revenue != 10000 {
		t.Fatalf("expected revenue to include only the approved order (10000), got %d", overview.Revenue)
	}
}
