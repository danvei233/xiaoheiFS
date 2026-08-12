package order_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	appcart "xiaoheiplay/internal/app/cart"
	apporder "xiaoheiplay/internal/app/order"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func TestOrderService_CreateOrderFromItems(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "buyer", "buyer@example.com", "pass")

	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, nil, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
	order, items, err := svc.CreateOrderFromItems(context.Background(), user.ID, "CNY", []appshared.OrderItemInput{
		{PackageID: seed.Package.ID, SystemID: seed.SystemImage.ID, Qty: 1},
	}, "idem-1", "")
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	if order.ID == 0 || len(items) != 1 {
		t.Fatalf("expected order and one item")
	}
	if order.Status != domain.OrderStatusPendingPayment {
		t.Fatalf("expected pending payment")
	}
}

func TestOrderService_SubmitPaymentIdempotent(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "pay", "pay@example.com", "pass")
	order := domain.Order{
		UserID:      user.ID,
		OrderNo:     "ORD-100",
		Status:      domain.OrderStatusPendingPayment,
		TotalAmount: 1000,
		Currency:    "CNY",
	}
	if err := repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	item := domain.OrderItem{
		OrderID:  order.ID,
		Amount:   1000,
		Status:   domain.OrderItemStatusPendingPayment,
		Action:   "create",
		SpecJSON: "{}",
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{item}); err != nil {
		t.Fatalf("create items: %v", err)
	}

	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, nil, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
	payment, err := svc.SubmitPayment(context.Background(), user.ID, order.ID, appshared.PaymentInput{
		Method:   "manual",
		Amount:   1000,
		Currency: "CNY",
		TradeNo:  "T-1",
	}, "idem-1")
	if err != nil {
		t.Fatalf("submit payment: %v", err)
	}
	payment2, err := svc.SubmitPayment(context.Background(), user.ID, order.ID, appshared.PaymentInput{
		Method:   "manual",
		Amount:   1000,
		Currency: "CNY",
		TradeNo:  "T-1",
	}, "idem-1")
	if err != nil {
		t.Fatalf("submit payment 2: %v", err)
	}
	if payment.ID != payment2.ID {
		t.Fatalf("expected idempotent payment")
	}
}

func TestOrderService_RejectsPaymentAmountMismatchWithoutMutation(t *testing.T) {
	for _, tc := range []struct {
		name string
		pay  func(*apporder.OrderService, context.Context, int64, int64, appshared.PaymentInput) error
	}{
		{
			name: "user submit",
			pay: func(svc *apporder.OrderService, ctx context.Context, actorID, orderID int64, input appshared.PaymentInput) error {
				_, err := svc.SubmitPayment(ctx, actorID, orderID, input, "mismatch-idem")
				return err
			},
		},
		{
			name: "admin mark paid",
			pay: func(svc *apporder.OrderService, ctx context.Context, actorID, orderID int64, input appshared.PaymentInput) error {
				_, err := svc.MarkPaid(ctx, actorID, orderID, input)
				return err
			},
		},
	} {
		for _, amount := range []int64{999, 1001} {
			t.Run(tc.name+"/amount_"+fmt.Sprint(amount), func(t *testing.T) {
				_, repo := testutil.NewTestDB(t, false)
				user := testutil.CreateUser(t, repo, "amount-check", "amount-check@example.com", "pass")
				ctx := context.Background()
				order := domain.Order{
					UserID:      user.ID,
					OrderNo:     fmt.Sprintf("ORD-AMOUNT-%d", amount),
					Status:      domain.OrderStatusPendingPayment,
					TotalAmount: 1000,
					Currency:    "CNY",
				}
				if err := repo.CreateOrder(ctx, &order); err != nil {
					t.Fatalf("create order: %v", err)
				}
				if err := repo.CreateOrderItems(ctx, []domain.OrderItem{{
					OrderID: order.ID, Amount: 1000, Status: domain.OrderItemStatusPendingPayment,
					Action: "create", SpecJSON: "{}",
				}}); err != nil {
					t.Fatalf("create item: %v", err)
				}

				svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, nil, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
				err := tc.pay(svc, ctx, user.ID, order.ID, appshared.PaymentInput{
					Method: "manual", Amount: amount, Currency: "CNY", TradeNo: fmt.Sprintf("TN-%d", amount),
				})
				if err != appshared.ErrInvalidInput {
					t.Fatalf("expected ErrInvalidInput, got %v", err)
				}

				storedOrder, err := repo.GetOrder(ctx, order.ID)
				if err != nil {
					t.Fatalf("get order: %v", err)
				}
				if storedOrder.Status != domain.OrderStatusPendingPayment {
					t.Fatalf("order status mutated to %q", storedOrder.Status)
				}
				items, err := repo.ListOrderItems(ctx, order.ID)
				if err != nil {
					t.Fatalf("list items: %v", err)
				}
				if len(items) != 1 || items[0].Status != domain.OrderItemStatusPendingPayment {
					t.Fatalf("order items mutated: %#v", items)
				}
				payments, err := repo.ListPaymentsByOrder(ctx, order.ID)
				if err != nil {
					t.Fatalf("list payments: %v", err)
				}
				if len(payments) != 0 {
					t.Fatalf("mismatched amount created payments: %#v", payments)
				}
			})
		}
	}
}

func TestOrderService_SubmitPayment_RejectsCrossOrderTradeNoReuse(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, repo, "pay2", "pay2@example.com", "pass")
	orderA := domain.Order{UserID: user.ID, OrderNo: "ORD-A", Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "CNY"}
	orderB := domain.Order{UserID: user.ID, OrderNo: "ORD-B", Status: domain.OrderStatusPendingPayment, TotalAmount: 2000, Currency: "CNY"}
	if err := repo.CreateOrder(context.Background(), &orderA); err != nil {
		t.Fatalf("create order a: %v", err)
	}
	if err := repo.CreateOrder(context.Background(), &orderB); err != nil {
		t.Fatalf("create order b: %v", err)
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{{OrderID: orderA.ID, Amount: 1000, Status: domain.OrderItemStatusPendingPayment, Action: "create", SpecJSON: "{}"}}); err != nil {
		t.Fatalf("create items a: %v", err)
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{{OrderID: orderB.ID, Amount: 2000, Status: domain.OrderItemStatusPendingPayment, Action: "create", SpecJSON: "{}"}}); err != nil {
		t.Fatalf("create items b: %v", err)
	}
	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, nil, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
	if _, err := svc.SubmitPayment(context.Background(), user.ID, orderA.ID, appshared.PaymentInput{
		Method:   "approval",
		Amount:   1000,
		Currency: "CNY",
		TradeNo:  "TN-CROSS",
	}, "idem-a"); err != nil {
		t.Fatalf("submit payment a: %v", err)
	}
	if _, err := svc.SubmitPayment(context.Background(), user.ID, orderB.ID, appshared.PaymentInput{
		Method:   "approval",
		Amount:   2000,
		Currency: "CNY",
		TradeNo:  "TN-CROSS",
	}, "idem-b"); err != appshared.ErrConflict {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestOrderService_CreateOrderFromCartAndCancel(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "cartuser", "cartuser@example.com", "pass")

	cartSvc := appcart.NewService(repo, repo, repo)
	if _, err := cartSvc.Add(context.Background(), user.ID, seed.Package.ID, seed.SystemImage.ID, appshared.CartSpec{}, 1); err != nil {
		t.Fatalf("add cart: %v", err)
	}

	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, nil, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
	order, items, err := svc.CreateOrderFromCart(context.Background(), user.ID, "CNY", "idem-cart", "")
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("expected items")
	}

	if err := svc.CancelOrder(context.Background(), user.ID, order.ID); err != nil {
		t.Fatalf("cancel order: %v", err)
	}
	updated, err := repo.GetOrder(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if updated.Status != domain.OrderStatusCanceled {
		t.Fatalf("expected canceled")
	}
}

func TestOrderService_CreateRefundOrder_UsesItemAmount(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "refundbase", "refundbase@example.com", "pass")

	baseOrder := domain.Order{
		UserID:      user.ID,
		OrderNo:     "ORD-REFUND-BASE-MP",
		Status:      domain.OrderStatusActive,
		TotalAmount: 200000,
		Currency:    "CNY",
	}
	if err := repo.CreateOrder(context.Background(), &baseOrder); err != nil {
		t.Fatalf("create base order: %v", err)
	}
	baseItem := domain.OrderItem{
		OrderID:   baseOrder.ID,
		PackageID: seed.Package.ID,
		SystemID:  seed.SystemImage.ID,
		Amount:    200000,
		Status:    domain.OrderItemStatusActive,
		Action:    "create",
		SpecJSON:  "{}",
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{baseItem}); err != nil {
		t.Fatalf("create base item: %v", err)
	}
	items, err := repo.ListOrderItems(context.Background(), baseOrder.ID)
	if err != nil || len(items) == 0 {
		t.Fatalf("list base items: %v", err)
	}
	inst := domain.VPSInstance{
		UserID:               user.ID,
		OrderItemID:          items[0].ID,
		AutomationInstanceID: "999",
		Name:                 "vm-refund-base",
		PackageID:            seed.Package.ID,
		PackageName:          seed.Package.Name,
		MonthlyPrice:         3000,
		SpecJSON:             "{}",
		Status:               domain.VPSStatusRunning,
		CreatedAt:            time.Now(),
	}
	expire := time.Now().Add(30 * 24 * time.Hour)
	inst.ExpireAt = &expire
	if err := repo.CreateInstance(context.Background(), &inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}

	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, nil, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
	refundOrder, amount, err := svc.CreateRefundOrder(context.Background(), user.ID, inst.ID, "test")
	if err != nil {
		t.Fatalf("create refund order: %v", err)
	}
	// The refund base must be the amount actually paid for the item (the
	// prepaid TotalAmount 200000), not the single-month MonthlyPrice (3000).
	// Within the full-refund window the whole prepaid amount is returned.
	if amount != 200000 {
		t.Fatalf("expected refund amount based on item amount 200000, got %d", amount)
	}
	if refundOrder.TotalAmount != -200000 {
		t.Fatalf("expected refund order total -200000, got %d", refundOrder.TotalAmount)
	}
}

// TestOrderService_CreateRefundOrder_MultiMonthPrepayLocks the refund base for
// a multi-month prepay: an instance paid 3 months in advance refunds the full
// prepaid amount (3 * monthly) inside the full-refund window, not a single
// month. This is the order-path counterpart of the walletorder regression
// (TestWalletOrderService_RequestRefund_MultiMonthPrepayProrated) and pins the
// two refund paths to the same base.
func TestOrderService_CreateRefundOrder_MultiMonthPrepay(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "refundprepay", "refundprepay@example.com", "pass")

	baseOrder := domain.Order{
		UserID:      user.ID,
		OrderNo:     "ORD-REFUND-PREPAY",
		Status:      domain.OrderStatusActive,
		TotalAmount: 9000,
		Currency:    "CNY",
	}
	if err := repo.CreateOrder(context.Background(), &baseOrder); err != nil {
		t.Fatalf("create base order: %v", err)
	}
	baseItem := domain.OrderItem{
		OrderID:        baseOrder.ID,
		PackageID:      seed.Package.ID,
		SystemID:       seed.SystemImage.ID,
		Amount:         9000,
		DurationMonths: 3,
		Status:         domain.OrderItemStatusActive,
		Action:         "create",
		SpecJSON:       "{}",
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{baseItem}); err != nil {
		t.Fatalf("create base item: %v", err)
	}
	items, err := repo.ListOrderItems(context.Background(), baseOrder.ID)
	if err != nil || len(items) == 0 {
		t.Fatalf("list base items: %v", err)
	}
	inst := domain.VPSInstance{
		UserID:               user.ID,
		OrderItemID:          items[0].ID,
		AutomationInstanceID: "1002",
		Name:                 "vm-refund-prepay",
		PackageID:            seed.Package.ID,
		PackageName:          seed.Package.Name,
		MonthlyPrice:         3000,
		SpecJSON:             "{}",
		Status:               domain.VPSStatusRunning,
		CreatedAt:            time.Now().Add(-time.Hour),
	}
	expire := time.Now().Add(90*24*time.Hour - time.Hour)
	inst.ExpireAt = &expire
	if err := repo.CreateInstance(context.Background(), &inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}

	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, nil, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
	refundOrder, amount, err := svc.CreateRefundOrder(context.Background(), user.ID, inst.ID, "prepay refund")
	if err != nil {
		t.Fatalf("create refund order: %v", err)
	}
	// Full-refund window (~1h elapsed of 90d): refund equals the full prepaid
	// 3-month amount, not a single monthly price.
	if amount != 9000 {
		t.Fatalf("expected full refund of prepaid amount 9000, got %d", amount)
	}
	if refundOrder.TotalAmount != -9000 {
		t.Fatalf("expected refund order total -9000, got %d", refundOrder.TotalAmount)
	}
}

func TestOrderService_CreateRenewOrder_UsesInstanceMonthlyPrice(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "renewprice", "renewprice@example.com", "pass")

	baseOrder := domain.Order{
		UserID:      user.ID,
		OrderNo:     "ORD-RENEW-BASE-MP",
		Status:      domain.OrderStatusActive,
		TotalAmount: seed.Package.Monthly,
		Currency:    "CNY",
	}
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
	items, err := repo.ListOrderItems(context.Background(), baseOrder.ID)
	if err != nil || len(items) == 0 {
		t.Fatalf("list base items: %v", err)
	}

	inst := domain.VPSInstance{
		UserID:               user.ID,
		OrderItemID:          items[0].ID,
		AutomationInstanceID: "1009",
		Name:                 "vm-renew-price",
		PackageID:            seed.Package.ID,
		PackageName:          seed.Package.Name,
		MonthlyPrice:         1234,
		SpecJSON:             "{}",
		Status:               domain.VPSStatusRunning,
		CreatedAt:            time.Now(),
	}
	expire := time.Now().Add(30 * 24 * time.Hour)
	inst.ExpireAt = &expire
	if err := repo.CreateInstance(context.Background(), &inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}

	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, nil, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
	order, err := svc.CreateRenewOrder(context.Background(), user.ID, inst.ID, 0, 2)
	if err != nil {
		t.Fatalf("create renew order: %v", err)
	}
	if order.TotalAmount != 2468 {
		t.Fatalf("expected renew order total 2468 from instance monthly price, got %d", order.TotalAmount)
	}
	if order.Status != domain.OrderStatusPendingPayment {
		t.Fatalf("expected pending payment renew order, got %s", order.Status)
	}

	renewItems, err := repo.ListOrderItems(context.Background(), order.ID)
	if err != nil || len(renewItems) != 1 {
		t.Fatalf("list renew items: %v", err)
	}
	if renewItems[0].Amount != 2468 {
		t.Fatalf("expected renew item amount 2468, got %d", renewItems[0].Amount)
	}
}

func TestOrderService_CreateRenewOrder_RejectsOverflowInputs(t *testing.T) {
	tests := []struct {
		name           string
		monthlyPrice   int64
		durationMonths int
	}{
		{
			name:           "amount overflow",
			monthlyPrice:   31,
			durationMonths: int((int64(^uint64(0)>>1) / 31) + 1),
		},
		{
			name:           "renew days overflow",
			monthlyPrice:   1,
			durationMonths: int((int(^uint(0)>>1) / 30) + 1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, repo := testutil.NewTestDB(t, false)
			seed := testutil.SeedCatalog(t, repo)
			user := testutil.CreateUser(t, repo, "renewoverflow", "renewoverflow@example.com", "pass")

			baseOrder := domain.Order{
				UserID:      user.ID,
				OrderNo:     "ORD-RENEW-OVERFLOW",
				Status:      domain.OrderStatusActive,
				TotalAmount: seed.Package.Monthly,
				Currency:    "CNY",
			}
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
			items, err := repo.ListOrderItems(context.Background(), baseOrder.ID)
			if err != nil || len(items) == 0 {
				t.Fatalf("list base items: %v", err)
			}

			inst := domain.VPSInstance{
				UserID:               user.ID,
				OrderItemID:          items[0].ID,
				AutomationInstanceID: "1010",
				Name:                 "vm-renew-overflow",
				PackageID:            seed.Package.ID,
				PackageName:          seed.Package.Name,
				MonthlyPrice:         tt.monthlyPrice,
				SpecJSON:             "{}",
				Status:               domain.VPSStatusRunning,
				CreatedAt:            time.Now(),
			}
			expire := time.Now().Add(30 * 24 * time.Hour)
			inst.ExpireAt = &expire
			if err := repo.CreateInstance(context.Background(), &inst); err != nil {
				t.Fatalf("create instance: %v", err)
			}

			svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, nil, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
			if _, err := svc.CreateRenewOrder(context.Background(), user.ID, inst.ID, 0, tt.durationMonths); err != appshared.ErrInvalidInput {
				t.Fatalf("expected invalid input, got %v", err)
			}
		})
	}
}

// TestOrderService_CreateRefundOrder_UserAPIKeySourceRespectsApproval locks
// the F5 fix: an API-key refund must NOT bypass refund_requires_approval. With
// approval required the refund order stays pending_review and no wallet credit
// happens, so a stolen/errant key cannot delete the VPS and pocket a refund
// without an admin review.
func TestOrderService_CreateRefundOrder_UserAPIKeySourceRespectsApproval(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "refundapikey", "refundapikey@example.com", "pass")
	_ = repo.UpsertSetting(context.Background(), domain.Setting{Key: "refund_requires_approval", ValueJSON: "true"})

	baseOrder := domain.Order{
		UserID:      user.ID,
		OrderNo:     "ORD-REFUND-APIKEY-BASE",
		Status:      domain.OrderStatusActive,
		TotalAmount: 200000,
		Currency:    "CNY",
	}
	if err := repo.CreateOrder(context.Background(), &baseOrder); err != nil {
		t.Fatalf("create base order: %v", err)
	}
	baseItem := domain.OrderItem{
		OrderID:   baseOrder.ID,
		PackageID: seed.Package.ID,
		SystemID:  seed.SystemImage.ID,
		Amount:    200000,
		Status:    domain.OrderItemStatusActive,
		Action:    "create",
		SpecJSON:  "{}",
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{baseItem}); err != nil {
		t.Fatalf("create base item: %v", err)
	}
	items, err := repo.ListOrderItems(context.Background(), baseOrder.ID)
	if err != nil || len(items) == 0 {
		t.Fatalf("list base items: %v", err)
	}
	inst := domain.VPSInstance{
		UserID:               user.ID,
		OrderItemID:          items[0].ID,
		GoodsTypeID:          1,
		AutomationInstanceID: "1001",
		Name:                 "vm-refund-apikey",
		PackageID:            seed.Package.ID,
		PackageName:          seed.Package.Name,
		MonthlyPrice:         3000,
		SpecJSON:             "{}",
		Status:               domain.VPSStatusRunning,
		CreatedAt:            time.Now(),
	}
	expire := time.Now().Add(30 * 24 * time.Hour)
	inst.ExpireAt = &expire
	if err := repo.CreateInstance(context.Background(), &inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}

	wBefore, err := repo.GetWallet(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("wallet before: %v", err)
	}

	automationResolver := &testutil.FakeAutomationResolver{Client: &testutil.FakeAutomationClient{}}
	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, automationResolver, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
	ctx := apporder.WithOrderSource(context.Background(), apporder.OrderSourceUserAPIKey)
	refundOrder, amount, err := svc.CreateRefundOrder(ctx, user.ID, inst.ID, "apikey refund")
	if err != nil {
		t.Fatalf("create refund order: %v", err)
	}
	if amount != 200000 {
		t.Fatalf("expected refund amount based on item amount 200000, got %d", amount)
	}
	if refundOrder.Status != domain.OrderStatusPendingReview {
		t.Fatalf("expected API-key refund to stay pending_review under refund_requires_approval=true, got %s", refundOrder.Status)
	}

	wAfter, err := repo.GetWallet(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("wallet after: %v", err)
	}
	if wAfter.Balance-wBefore.Balance != 0 {
		t.Fatalf("expected no wallet credit before admin approval, delta=%d", wAfter.Balance-wBefore.Balance)
	}
}

// TestOrderService_CreateRefundOrder_UserAPIKeySourceAutoApprovesWhenConfigured
// is the positive counterpart: when refund_requires_approval=false the refund
// auto-approves for an API-key request too (the exemption must follow the
// configuration, not the request source).
func TestOrderService_CreateRefundOrder_UserAPIKeySourceAutoApprovesWhenConfigured(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "refundapikey2", "refundapikey2@example.com", "pass")
	_ = repo.UpsertSetting(context.Background(), domain.Setting{Key: "refund_requires_approval", ValueJSON: "false"})

	baseOrder := domain.Order{
		UserID:      user.ID,
		OrderNo:     "ORD-REFUND-APIKEY2-BASE",
		Status:      domain.OrderStatusActive,
		TotalAmount: 200000,
		Currency:    "CNY",
	}
	if err := repo.CreateOrder(context.Background(), &baseOrder); err != nil {
		t.Fatalf("create base order: %v", err)
	}
	baseItem := domain.OrderItem{
		OrderID:   baseOrder.ID,
		PackageID: seed.Package.ID,
		SystemID:  seed.SystemImage.ID,
		Amount:    200000,
		Status:    domain.OrderItemStatusActive,
		Action:    "create",
		SpecJSON:  "{}",
	}
	if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{baseItem}); err != nil {
		t.Fatalf("create base item: %v", err)
	}
	items, err := repo.ListOrderItems(context.Background(), baseOrder.ID)
	if err != nil || len(items) == 0 {
		t.Fatalf("list base items: %v", err)
	}
	inst := domain.VPSInstance{
		UserID:               user.ID,
		OrderItemID:          items[0].ID,
		GoodsTypeID:          1,
		AutomationInstanceID: "1003",
		Name:                 "vm-refund-apikey2",
		PackageID:            seed.Package.ID,
		PackageName:          seed.Package.Name,
		MonthlyPrice:         3000,
		SpecJSON:             "{}",
		Status:               domain.VPSStatusRunning,
		CreatedAt:            time.Now(),
	}
	expire := time.Now().Add(30 * 24 * time.Hour)
	inst.ExpireAt = &expire
	if err := repo.CreateInstance(context.Background(), &inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}

	automationResolver := &testutil.FakeAutomationResolver{Client: &testutil.FakeAutomationClient{}}
	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, automationResolver, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
	ctx := apporder.WithOrderSource(context.Background(), apporder.OrderSourceUserAPIKey)
	refundOrder, _, err := svc.CreateRefundOrder(ctx, user.ID, inst.ID, "apikey refund")
	if err != nil {
		t.Fatalf("create refund order: %v", err)
	}
	if refundOrder.Status == domain.OrderStatusPendingReview {
		t.Fatalf("expected API-key refund auto-approved when refund_requires_approval=false, got pending_review")
	}
}

// TestOrderService_RejectBalancePaidOrderRefundsWallet locks the F2b fix: an
// order whose balance payment is already approved (order in pending_review
// before auto-approval) must be rejectable, and the wallet debit must be
// returned idempotently — the NOT EXISTS guard in RejectOrderAtomic must only
// protect external (provider) approved payments, not local balance debits.
func TestOrderService_RejectBalancePaidOrderRefundsWallet(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	user := testutil.CreateUser(t, repo, "rejectbalance", "rejectbalance@example.com", "pass")
	if _, err := repo.AdjustWalletBalance(ctx, user.ID, 5000, "credit", "seed", 1, "init"); err != nil {
		t.Fatalf("seed wallet: %v", err)
	}
	order := domain.Order{UserID: user.ID, OrderNo: "ORD-REJ-BAL", Status: domain.OrderStatusPendingReview, TotalAmount: 2000, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	if err := repo.CreateOrderItems(ctx, []domain.OrderItem{{
		OrderID:  order.ID,
		Amount:   2000,
		Status:   domain.OrderItemStatusPendingReview,
		Action:   "create",
		SpecJSON: "{}",
	}}); err != nil {
		t.Fatalf("create items: %v", err)
	}
	// Simulate the committed balance payment: wallet debited with ref_type=order,
	// approved balance payment record.
	if _, err := repo.AdjustWalletBalance(ctx, user.ID, -2000, "debit", "order", order.ID, "balance payment"); err != nil {
		t.Fatalf("debit wallet: %v", err)
	}
	payment := domain.OrderPayment{
		OrderID:  order.ID,
		UserID:   user.ID,
		Method:   "balance",
		Amount:   2000,
		Currency: "CNY",
		TradeNo:  "BAL-REJ",
		Status:   domain.PaymentStatusApproved,
	}
	if err := repo.CreatePayment(ctx, &payment); err != nil {
		t.Fatalf("create payment: %v", err)
	}

	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, nil, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
	if err := svc.RejectOrder(ctx, 9, order.ID, "declined"); err != nil {
		t.Fatalf("reject balance-paid order: %v", err)
	}

	updated, err := repo.GetOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if updated.Status != domain.OrderStatusRejected {
		t.Fatalf("expected rejected, got %s", updated.Status)
	}
	wallet, err := repo.GetWallet(ctx, user.ID)
	if err != nil {
		t.Fatalf("get wallet: %v", err)
	}
	if wallet.Balance != 5000 {
		t.Fatalf("expected balance refunded to 5000, got %d", wallet.Balance)
	}
	hasRefund, err := repo.HasWalletTransaction(ctx, user.ID, "balance_refund", order.ID)
	if err != nil || !hasRefund {
		t.Fatalf("expected balance_refund transaction, has=%v err=%v", hasRefund, err)
	}

	// Rejecting again must not double-credit (idempotency key).
	if err := svc.RejectOrder(ctx, 9, order.ID, "declined again"); err != nil {
		t.Fatalf("reject again: %v", err)
	}
	wallet, err = repo.GetWallet(ctx, user.ID)
	if err != nil {
		t.Fatalf("get wallet after second reject: %v", err)
	}
	if wallet.Balance != 5000 {
		t.Fatalf("expected balance still 5000 after idempotent reject, got %d", wallet.Balance)
	}
}

// TestOrderService_RejectProviderPaidOrderStillBlocked locks the flip side of
// F2b: an order with an approved EXTERNAL (provider) payment must still be
// protected from rejection — that guard prevents stranding real provider money
// without an external refund path.
func TestOrderService_RejectProviderPaidOrderStillBlocked(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	user := testutil.CreateUser(t, repo, "rejectprovider", "rejectprovider@example.com", "pass")
	order := domain.Order{UserID: user.ID, OrderNo: "ORD-REJ-PROV", Status: domain.OrderStatusPendingReview, TotalAmount: 2000, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	if err := repo.CreateOrderItems(ctx, []domain.OrderItem{{
		OrderID:  order.ID,
		Amount:   2000,
		Status:   domain.OrderItemStatusPendingReview,
		Action:   "create",
		SpecJSON: "{}",
	}}); err != nil {
		t.Fatalf("create items: %v", err)
	}
	payment := domain.OrderPayment{
		OrderID:  order.ID,
		UserID:   user.ID,
		Method:   "wechat",
		Amount:   2000,
		Currency: "CNY",
		TradeNo:  "WX-REJ",
		Status:   domain.PaymentStatusApproved,
	}
	if err := repo.CreatePayment(ctx, &payment); err != nil {
		t.Fatalf("create payment: %v", err)
	}

	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, nil, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
	if err := svc.RejectOrder(ctx, 9, order.ID, "declined"); err != appshared.ErrConflict {
		t.Fatalf("expected conflict rejecting an order with an approved provider payment, got %v", err)
	}
}
