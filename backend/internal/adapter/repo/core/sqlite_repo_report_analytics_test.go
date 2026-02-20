package repo_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func TestSQLiteRepo_ReportAnalyticsPaymentWindow(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()

	user := testutil.CreateUser(t, repo, "ra_repo", "ra_repo@example.com", "pass")
	order := domain.Order{UserID: user.ID, OrderNo: "ORD-RA-REPO", Status: domain.OrderStatusApproved, TotalAmount: 3000, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	pay := domain.OrderPayment{OrderID: order.ID, UserID: user.ID, Method: "manual", Amount: 3000, Currency: "CNY", TradeNo: "TRADE-RA-REPO", Status: domain.PaymentStatusApproved}
	if err := repo.CreatePayment(ctx, &pay); err != nil {
		t.Fatalf("create payment: %v", err)
	}

	from := time.Now().Add(-2 * time.Hour)
	to := time.Now().Add(2 * time.Hour)
	items, total, err := repo.ListPayments(ctx, appshared.PaymentFilter{Status: string(domain.PaymentStatusApproved), From: &from, To: &to}, 20, 0)
	if err != nil {
		t.Fatalf("list payments: %v", err)
	}
	if total == 0 || len(items) == 0 {
		t.Fatalf("expected approved payment in window")
	}
}

func TestSQLiteRepo_ReportAnalyticsDetailsOrdering(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()

	user := testutil.CreateUser(t, repo, "ra_detail", "ra_detail@example.com", "pass")
	gt := domain.GoodsType{Code: "vps", Name: "VPS", Active: true}
	if err := repo.CreateGoodsType(ctx, &gt); err != nil {
		t.Fatalf("create goods type: %v", err)
	}
	region := domain.Region{GoodsTypeID: gt.ID, Code: "cn", Name: "China", Active: true}
	if err := repo.CreateRegion(ctx, &region); err != nil {
		t.Fatalf("create region: %v", err)
	}
	plan := domain.PlanGroup{GoodsTypeID: gt.ID, RegionID: region.ID, LineID: 1001, Name: "BGP", UnitCore: 1, UnitMem: 1, UnitDisk: 1, UnitBW: 1, Active: true, Visible: true}
	if err := repo.CreatePlanGroup(ctx, &plan); err != nil {
		t.Fatalf("create plan group: %v", err)
	}
	pkg := domain.Package{GoodsTypeID: gt.ID, PlanGroupID: plan.ID, Name: "BASIC", Active: true, Visible: true}
	if err := repo.CreatePackage(ctx, &pkg); err != nil {
		t.Fatalf("create package: %v", err)
	}

	makeOrder := func(no string, amount int64) {
		order := domain.Order{
			UserID:      user.ID,
			OrderNo:     no,
			Status:      domain.OrderStatusApproved,
			TotalAmount: amount,
			Currency:    "CNY",
		}
		if err := repo.CreateOrder(ctx, &order); err != nil {
			t.Fatalf("create order: %v", err)
		}
		item := domain.OrderItem{
			OrderID:     order.ID,
			PackageID:   pkg.ID,
			GoodsTypeID: gt.ID,
			Amount:      amount,
			Qty:         1,
			Status:      domain.OrderItemStatusApproved,
			Action:      "create",
			SpecJSON:    "{}",
		}
		if err := repo.CreateOrderItems(ctx, []domain.OrderItem{item}); err != nil {
			t.Fatalf("create order item: %v", err)
		}
		pay := domain.OrderPayment{
			OrderID:  order.ID,
			UserID:   user.ID,
			Method:   "manual",
			Amount:   amount,
			Currency: "CNY",
			TradeNo:  fmt.Sprintf("TRADE-%s", no),
			Status:   domain.PaymentStatusApproved,
		}
		if err := repo.CreatePayment(ctx, &pay); err != nil {
			t.Fatalf("create payment: %v", err)
		}
	}

	makeOrder("ORD-RA-D1", 3000)
	time.Sleep(20 * time.Millisecond)
	makeOrder("ORD-RA-D2", 5000)

	from := time.Now().Add(-1 * time.Hour)
	to := time.Now().Add(1 * time.Hour)

	rowsByAmount, total, err := repo.ListRevenueAnalyticsDetails(ctx, from, to, "amount", "desc", 20, 0)
	if err != nil {
		t.Fatalf("list details by amount: %v", err)
	}
	if total != 2 || len(rowsByAmount) != 2 {
		t.Fatalf("unexpected total/details length: total=%d len=%d", total, len(rowsByAmount))
	}
	if rowsByAmount[0].Amount < rowsByAmount[1].Amount {
		t.Fatalf("expected amount desc ordering")
	}

	rowsByTime, _, err := repo.ListRevenueAnalyticsDetails(ctx, from, to, "paid_at", "desc", 20, 0)
	if err != nil {
		t.Fatalf("list details by time: %v", err)
	}
	if len(rowsByTime) != 2 {
		t.Fatalf("unexpected rows by time: %d", len(rowsByTime))
	}
	if rowsByTime[0].PaidAt.Before(rowsByTime[1].PaidAt) {
		t.Fatalf("expected paid_at desc ordering")
	}
}
