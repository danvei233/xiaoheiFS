package usecase_test

import (
	"context"
	"testing"
	"time"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/usecase"
)

func TestOrderService_CreateOrderFromCartInvalid(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "cartbad", "cartbad@example.com", "pass")

	svc := usecase.NewOrderService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, nil, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
	if _, _, err := svc.CreateOrderFromCart(context.Background(), user.ID, "CNY", ""); err != usecase.ErrInvalidInput {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestOrderService_CreateOrderFromItemsInvalidSpec(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "specbad", "specbad@example.com", "pass")

	svc := usecase.NewOrderService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, nil, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
	_, _, err := svc.CreateOrderFromItems(context.Background(), user.ID, "CNY", []usecase.OrderItemInput{
		{PackageID: seed.Package.ID, SystemID: seed.SystemImage.ID, Qty: 1, Spec: usecase.CartSpec{AddCores: -1}},
	}, "")
	if err != usecase.ErrInvalidInput {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestOrderService_CreateOrderFromItemsInvalidBillingCycle(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "cyclebad", "cyclebad@example.com", "pass")

	cycle := domain.BillingCycle{Name: "bad", Months: 1, Multiplier: 1, MinQty: 2, MaxQty: 3, Active: false, SortOrder: 1}
	if err := repo.CreateBillingCycle(context.Background(), &cycle); err != nil {
		t.Fatalf("create cycle: %v", err)
	}

	svc := usecase.NewOrderService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, nil, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
	_, _, err := svc.CreateOrderFromItems(context.Background(), user.ID, "CNY", []usecase.OrderItemInput{
		{PackageID: seed.Package.ID, SystemID: seed.SystemImage.ID, Qty: 1, Spec: usecase.CartSpec{BillingCycleID: cycle.ID, CycleQty: 1}},
	}, "")
	if err != usecase.ErrInvalidInput {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestOrderService_SubmitPaymentConflicts(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "paybad", "paybad@example.com", "pass")
	other := testutil.CreateUser(t, repo, "other", "other@example.com", "pass")

	order := domain.Order{UserID: user.ID, OrderNo: "ORD-PB", Status: domain.OrderStatusApproved, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{{OrderID: order.ID, Amount: 1000, Status: domain.OrderItemStatusApproved, Action: "create", SpecJSON: "{}"}}); err != nil {
		t.Fatalf("create items: %v", err)
	}

	svc := usecase.NewOrderService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, nil, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
	if _, err := svc.SubmitPayment(context.Background(), other.ID, order.ID, usecase.PaymentInput{Method: "manual", Amount: 1000}, ""); err != usecase.ErrForbidden {
		t.Fatalf("expected forbidden, got %v", err)
	}
	if _, err := svc.SubmitPayment(context.Background(), user.ID, order.ID, usecase.PaymentInput{Method: "manual", Amount: 1000}, ""); err != usecase.ErrConflict {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestOrderService_RenewAndResizeFlow(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "renewuser", "renewuser@example.com", "pass")

	baseOrder := domain.Order{UserID: user.ID, OrderNo: "ORD-BASE", Status: domain.OrderStatusActive, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(context.Background(), &baseOrder); err != nil {
		t.Fatalf("create base order: %v", err)
	}
	baseItem := domain.OrderItem{
		OrderID:   baseOrder.ID,
		PackageID: seed.Package.ID,
		SystemID:  seed.SystemImage.ID,
		Amount:    1000,
		Status:    domain.OrderItemStatusActive,
		Action:    "create",
		SpecJSON:  "{}",
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{baseItem}); err != nil {
		t.Fatalf("create base item: %v", err)
	}
	baseItems, err := repo.ListOrderItems(context.Background(), baseOrder.ID)
	if err != nil || len(baseItems) == 0 {
		t.Fatalf("list base items: %v", err)
	}

	inst := domain.VPSInstance{
		UserID:               user.ID,
		OrderItemID:          baseItems[0].ID,
		AutomationInstanceID: "123",
		Name:                 "vm1",
		PackageID:            seed.Package.ID,
		PackageName:          seed.Package.Name,
		MonthlyPrice:         seed.Package.Monthly,
		SpecJSON:             "{}",
		Status:               domain.VPSStatusRunning,
		ExpireAt:             timePtr(time.Now().Add(24 * time.Hour)),
	}
	if err := repo.CreateInstance(context.Background(), &inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}

	renewOrder := domain.Order{UserID: user.ID, OrderNo: "REN-1", Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(context.Background(), &renewOrder); err != nil {
		t.Fatalf("create renew order: %v", err)
	}
	renewItem := domain.OrderItem{
		OrderID:  renewOrder.ID,
		Amount:   1000,
		Status:   domain.OrderItemStatusPendingPayment,
		Action:   "renew",
		SpecJSON: `{"vps_id":` + testutil.Itoa(inst.ID) + `,"renew_days":30}`,
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{renewItem}); err != nil {
		t.Fatalf("create renew item: %v", err)
	}

	fakeAuto := &testutil.FakeAutomationClient{}
	autoResolver := &testutil.FakeAutomationResolver{Client: fakeAuto}
	svc := usecase.NewOrderService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, autoResolver, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
	if err := svc.ApproveOrder(context.Background(), 1, renewOrder.ID); err != nil {
		t.Fatalf("approve renew: %v", err)
	}
	waitForOrderStatus(t, repo, renewOrder.ID, domain.OrderStatusActive)

	updatedInst, err := repo.GetInstance(context.Background(), inst.ID)
	if err != nil || updatedInst.ExpireAt == nil || !updatedInst.ExpireAt.After(*inst.ExpireAt) {
		t.Fatalf("expected expire extended: %v", err)
	}

	resizeOrder := domain.Order{UserID: user.ID, OrderNo: "UPG-1", Status: domain.OrderStatusPendingPayment, TotalAmount: 0, Currency: "CNY"}
	if err := repo.CreateOrder(context.Background(), &resizeOrder); err != nil {
		t.Fatalf("create resize order: %v", err)
	}
	resizeItem := domain.OrderItem{
		OrderID: resizeOrder.ID,
		Amount:  0,
		Status:  domain.OrderItemStatusPendingPayment,
		Action:  "resize",
		SpecJSON: `{"vps_id":` + testutil.Itoa(inst.ID) + `,"target_cpu":2,"target_mem_gb":4,"target_disk_gb":20,"target_bw_mbps":50,` +
			`"refund_amount":5,"refund_to_wallet":true,"spec":{"add_cores":0,"add_mem_gb":0,"add_disk_gb":0,"add_bw_mbps":0}}`,
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{resizeItem}); err != nil {
		t.Fatalf("create resize item: %v", err)
	}
	if err := svc.ApproveOrder(context.Background(), 1, resizeOrder.ID); err != nil {
		t.Fatalf("approve resize: %v", err)
	}
	waitForOrderStatus(t, repo, resizeOrder.ID, domain.OrderStatusActive)

	wallet, err := repo.GetWallet(context.Background(), user.ID)
	if err != nil || wallet.Balance < 5 {
		t.Fatalf("expected refund wallet balance, got %v %v", wallet.Balance, err)
	}
	if len(fakeAuto.ElasticUpdates) == 0 {
		t.Fatalf("expected elastic update call")
	}
}

func waitForOrderStatus(t *testing.T, repo usecase.OrderRepository, orderID int64, status domain.OrderStatus) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		order, err := repo.GetOrder(context.Background(), orderID)
		if err == nil && order.Status == status {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("order %d status not %s", orderID, status)
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func TestOrderService_CreateResizeOrder_NegativeAmountAutoApproves(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "resizeauto", "resizeauto@example.com", "pass")

	baseOrder := domain.Order{UserID: user.ID, OrderNo: "ORD-RESIZE-AUTO", Status: domain.OrderStatusActive, TotalAmount: seed.Package.Monthly, Currency: "CNY"}
	if err := repo.CreateOrder(context.Background(), &baseOrder); err != nil {
		t.Fatalf("create base order: %v", err)
	}
	baseItem := domain.OrderItem{
		OrderID:   baseOrder.ID,
		PackageID: seed.Package.ID,
		SystemID:  seed.SystemImage.ID,
		Amount:    seed.Package.Monthly,
		Status:    domain.OrderItemStatusActive,
		Action:    "create",
		SpecJSON:  "{}",
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{baseItem}); err != nil {
		t.Fatalf("create base item: %v", err)
	}
	baseItems, _ := repo.ListOrderItems(context.Background(), baseOrder.ID)
	inst := domain.VPSInstance{
		UserID:               user.ID,
		OrderItemID:          baseItems[0].ID,
		AutomationInstanceID: "123",
		GoodsTypeID:          seed.Package.GoodsTypeID,
		Name:                 "vm-auto",
		PackageID:            seed.Package.ID,
		PackageName:          seed.Package.Name,
		MonthlyPrice:         seed.Package.Monthly,
		SpecJSON:             "{}",
		Status:               domain.VPSStatusRunning,
		ExpireAt:             timePtr(time.Now().Add(30 * 24 * time.Hour)),
	}
	if err := repo.CreateInstance(context.Background(), &inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}

	target := domain.Package{
		PlanGroupID:       seed.Package.PlanGroupID,
		Name:              "lower-plan",
		Cores:             1,
		MemoryGB:          1,
		DiskGB:            10,
		BandwidthMB:       10,
		PortNum:           1,
		Monthly:           1,
		Active:            true,
		Visible:           true,
		SortOrder:         99,
		GoodsTypeID:       seed.Package.GoodsTypeID,
		ProductID:         seed.Package.ProductID + 1000,
		CPUModel:          seed.Package.CPUModel,
		CapacityRemaining: 100,
	}
	if err := repo.CreatePackage(context.Background(), &target); err != nil {
		t.Fatalf("create target package: %v", err)
	}

	fakeAuto := &testutil.FakeAutomationClient{}
	autoResolver := &testutil.FakeAutomationResolver{Client: fakeAuto}
	svc := usecase.NewOrderService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, autoResolver, nil, repo, repo, nil, repo, repo, repo, repo, nil, nil)

	order, quote, err := svc.CreateResizeOrder(context.Background(), user.ID, inst.ID, nil, target.ID, false, nil)
	if err != nil {
		t.Fatalf("create resize order: %v", err)
	}
	if quote.ChargeAmount >= 0 {
		t.Fatalf("expected negative charge amount for downgrade, got %d", quote.ChargeAmount)
	}
	updated, err := repo.GetOrder(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if updated.Status == domain.OrderStatusPendingReview || updated.Status == domain.OrderStatusPendingPayment {
		t.Fatalf("expected auto-approved flow, got %s", updated.Status)
	}
}
