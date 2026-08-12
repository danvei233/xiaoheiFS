package order_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"xiaoheiplay/internal/adapter/repo/core"
	appcart "xiaoheiplay/internal/app/cart"
	appcoupon "xiaoheiplay/internal/app/coupon"
	apporder "xiaoheiplay/internal/app/order"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

// flipBeforeStatusWriteRepo wraps the real repo and simulates a concurrent
// status transition that commits between a service's GetOrder read and its
// status write, by flipping the order to flipStatus just before the write is
// performed.
type flipBeforeStatusWriteRepo struct {
	*repo.GormRepo
	flipOrderID int64
	flipStatus  domain.OrderStatus
}

func (r *flipBeforeStatusWriteRepo) UpdateOrderMeta(ctx context.Context, order domain.Order) error {
	_ = r.GormRepo.UpdateOrderStatus(ctx, r.flipOrderID, r.flipStatus)
	return r.GormRepo.UpdateOrderMeta(ctx, order)
}

func (r *flipBeforeStatusWriteRepo) UpdateOrderStatusIfCurrent(ctx context.Context, id int64, status domain.OrderStatus, currentStatuses []domain.OrderStatus) (bool, error) {
	_ = r.GormRepo.UpdateOrderStatus(ctx, r.flipOrderID, r.flipStatus)
	return r.GormRepo.UpdateOrderStatusIfCurrent(ctx, id, status, currentStatuses)
}

func (r *flipBeforeStatusWriteRepo) CancelOrderAtomic(ctx context.Context, order domain.Order, currentStatuses []domain.OrderStatus) (bool, error) {
	_ = r.GormRepo.UpdateOrderStatus(ctx, r.flipOrderID, r.flipStatus)
	return r.GormRepo.CancelOrderAtomic(ctx, order, currentStatuses)
}

// F1: an attacker-controlled order line quantity must not be expanded
// unboundedly. Any qty above maxOrderItemQty is rejected before allocation.
func TestOrderService_CreateOrderFromItems_RejectsOversizedQty(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "qtycap", "qtycap@example.com", "pass")

	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, nil, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
	_, _, err := svc.CreateOrderFromItems(context.Background(), user.ID, "CNY", []appshared.OrderItemInput{
		{PackageID: seed.Package.ID, SystemID: seed.SystemImage.ID, Qty: 101},
	}, "idem-qty-cap", "")
	if err != appshared.ErrInvalidInput {
		t.Fatalf("expected invalid input for oversized qty, got %v", err)
	}
}

// F1: the cart path must enforce the same per-line qty cap.
func TestOrderService_CreateOrderFromCart_RejectsOversizedCartQty(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "cartqtycap", "cartqtycap@example.com", "pass")

	cartSvc := appcart.NewService(repo, repo, repo)
	if _, err := cartSvc.Add(context.Background(), user.ID, seed.Package.ID, seed.SystemImage.ID, appshared.CartSpec{}, 101); err != nil {
		t.Fatalf("add cart: %v", err)
	}

	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, nil, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
	if _, _, err := svc.CreateOrderFromCart(context.Background(), user.ID, "CNY", "idem-cart-qty-cap", ""); err != appshared.ErrInvalidInput {
		t.Fatalf("expected invalid input for oversized cart qty, got %v", err)
	}
}

// F1: many lines each at the per-line cap must still respect a global cap on
// expanded order items.
func TestOrderService_CreateOrderFromItems_RejectsExcessiveTotalItems(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "totalcap", "totalcap@example.com", "pass")

	inputs := make([]appshared.OrderItemInput, 0, 11)
	for i := 0; i < 11; i++ {
		inputs = append(inputs, appshared.OrderItemInput{PackageID: seed.Package.ID, SystemID: seed.SystemImage.ID, Qty: 100})
	}
	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, nil, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
	if _, _, err := svc.CreateOrderFromItems(context.Background(), user.ID, "CNY", inputs, "idem-total-cap", ""); err != appshared.ErrInvalidInput {
		t.Fatalf("expected invalid input for excessive total items, got %v", err)
	}
}

// F2: order_no must stay unique for rapid successive orders by the same user.
// A second-order-per-second timestamp was colliding on the global unique index.
func TestOrderService_CreateOrderFromItems_SameSecondOrdersDistinctOrderNo(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "orderno", "orderno@example.com", "pass")

	svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, nil, nil, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
	first, _, err := svc.CreateOrderFromItems(context.Background(), user.ID, "CNY", []appshared.OrderItemInput{
		{PackageID: seed.Package.ID, SystemID: seed.SystemImage.ID, Qty: 1},
	}, "idem-orderno-1", "")
	if err != nil {
		t.Fatalf("create first order: %v", err)
	}
	second, _, err := svc.CreateOrderFromItems(context.Background(), user.ID, "CNY", []appshared.OrderItemInput{
		{PackageID: seed.Package.ID, SystemID: seed.SystemImage.ID, Qty: 1},
	}, "idem-orderno-2", "")
	if err != nil {
		t.Fatalf("create second order collided on order_no: %v", err)
	}
	if first.OrderNo == second.OrderNo {
		t.Fatalf("expected distinct order_no, both %q", first.OrderNo)
	}
}

// F3: SubmitPayment must not resurrect an order that was canceled between its
// read of the order and its status write.
func TestOrderService_SubmitPayment_DoesNotResurrectCanceledOrder(t *testing.T) {
	_, base := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, base, "toctou", "toctou@example.com", "pass")
	order := domain.Order{UserID: user.ID, OrderNo: "ORD-TOCTOU", Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "CNY"}
	if err := base.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	if err := base.CreateOrderItems(context.Background(), []domain.OrderItem{{OrderID: order.ID, Amount: 1000, Status: domain.OrderItemStatusPendingPayment, Action: "create", SpecJSON: "{}"}}); err != nil {
		t.Fatalf("create items: %v", err)
	}

	orders := &flipBeforeStatusWriteRepo{GormRepo: base, flipOrderID: order.ID, flipStatus: domain.OrderStatusCanceled}
	svc := apporder.NewService(orders, base, base, base, base, base, base, base, base, nil, nil, nil, base, base, nil, base, base, base, nil, nil, nil)
	if _, err := svc.SubmitPayment(context.Background(), user.ID, order.ID, appshared.PaymentInput{
		Method: "manual", Amount: 1000, Currency: "CNY", TradeNo: "T-TOCTOU",
	}, "idem-toctou"); err != appshared.ErrConflict {
		t.Fatalf("expected conflict when order canceled concurrently, got %v", err)
	}
	updated, err := base.GetOrder(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if updated.Status != domain.OrderStatusCanceled {
		t.Fatalf("expected order to stay canceled, got %s", updated.Status)
	}
}

// F3: MarkPaid must not resurrect an order that was canceled between the admin
// read and the status write.
func TestOrderService_MarkPaid_DoesNotResurrectCanceledOrder(t *testing.T) {
	_, base := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, base, "markpaidtoctou", "markpaidtoctou@example.com", "pass")
	order := domain.Order{UserID: user.ID, OrderNo: "ORD-MP-TOCTOU", Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "CNY"}
	if err := base.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	if err := base.CreateOrderItems(context.Background(), []domain.OrderItem{{OrderID: order.ID, Amount: 1000, Status: domain.OrderItemStatusPendingPayment, Action: "create", SpecJSON: "{}"}}); err != nil {
		t.Fatalf("create items: %v", err)
	}

	orders := &flipBeforeStatusWriteRepo{GormRepo: base, flipOrderID: order.ID, flipStatus: domain.OrderStatusCanceled}
	svc := apporder.NewService(orders, base, base, base, base, base, base, base, base, nil, nil, nil, base, base, nil, base, base, base, nil, nil, nil)
	if _, err := svc.MarkPaid(context.Background(), 1, order.ID, appshared.PaymentInput{
		Method: "manual", Amount: 1000, Currency: "CNY", TradeNo: "T-MP-TOCTOU",
	}); err != appshared.ErrConflict {
		t.Fatalf("expected conflict when order canceled concurrently, got %v", err)
	}
	updated, err := base.GetOrder(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if updated.Status != domain.OrderStatusCanceled {
		t.Fatalf("expected order to stay canceled, got %s", updated.Status)
	}
}

// failCreateItemsRepo forces CreateOrderItems to fail, simulating a partial
// failure after the order row was inserted.
type failCreateItemsRepo struct {
	*repo.GormRepo
}

func (r *failCreateItemsRepo) CreateOrderItems(ctx context.Context, items []domain.OrderItem) error {
	return errors.New("items insert failed")
}

// replayIdemRepo simulates a concurrent request winning the idempotency-key
// race: on CreateOrder it first commits an order with the same key (the
// winner), then the real insert collides on the unique index.
type replayIdemRepo struct {
	*repo.GormRepo
	userID  int64
	idemKey string
}

func (r *replayIdemRepo) CreateOrder(ctx context.Context, order *domain.Order) error {
	winner := domain.Order{
		UserID:         r.userID,
		OrderNo:        "ORD-REPLAY-WINNER",
		Source:         order.Source,
		Status:         domain.OrderStatusPendingPayment,
		TotalAmount:    order.TotalAmount,
		Currency:       order.Currency,
		IdempotencyKey: r.idemKey,
	}
	if err := r.GormRepo.CreateOrder(ctx, &winner); err != nil {
		return err
	}
	return r.GormRepo.CreateOrder(ctx, order)
}

// replayPaymentRepo simulates a concurrent submit winning the payment
// idempotency-key race.
type replayPaymentRepo struct {
	*repo.GormRepo
	orderID int64
	idemKey string
}

func (r *replayPaymentRepo) CreatePayment(ctx context.Context, payment *domain.OrderPayment) error {
	winner := domain.OrderPayment{
		OrderID:       r.orderID,
		UserID:        payment.UserID,
		Method:        payment.Method,
		Amount:        payment.Amount,
		Currency:      payment.Currency,
		TradeNo:       payment.TradeNo,
		Status:        domain.PaymentStatusPendingReview,
		IdempotencyKey: r.idemKey,
	}
	if err := r.GormRepo.CreatePayment(ctx, &winner); err != nil {
		return err
	}
	return r.GormRepo.CreatePayment(ctx, payment)
}

// failingRedemptionCouponEngine accepts a coupon preview but fails to record
// the redemption, simulating a stock/race failure after the cart was cleared.
type failingRedemptionCouponEngine struct{}

func (failingRedemptionCouponEngine) PreviewDiscount(ctx context.Context, userID int64, code string, items []appcoupon.QuoteItem) (appcoupon.ApplyResult, error) {
	return appcoupon.ApplyResult{
		Coupon:        domain.Coupon{ID: 1, Code: code},
		TotalDiscount: 0,
		UnitDiscount:  []int64{0},
	}, nil
}

func (failingRedemptionCouponEngine) CreateRedemption(ctx context.Context, redemption *domain.CouponRedemption) error {
	return errors.New("redemption failed")
}

func (failingRedemptionCouponEngine) MarkOrderCanceled(ctx context.Context, orderID int64) error   { return nil }
func (failingRedemptionCouponEngine) MarkOrderConfirmed(ctx context.Context, orderID int64) error { return nil }

// F5: when order-item insertion fails after the order row was created, the
// order must be rolled back instead of leaving an itemless orphan that also
// burns the idempotency key.
func TestOrderService_CreateOrderFromItems_RollsBackOrderOnItemsFailure(t *testing.T) {
	_, base := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, base)
	user := testutil.CreateUser(t, base, "orphan", "orphan@example.com", "pass")

	itemsRepo := &failCreateItemsRepo{GormRepo: base}
	svc := apporder.NewService(base, itemsRepo, base, base, base, base, base, base, base, nil, nil, nil, base, base, nil, base, base, base, nil, nil, nil)
	if _, _, err := svc.CreateOrderFromItems(context.Background(), user.ID, "CNY", []appshared.OrderItemInput{
		{PackageID: seed.Package.ID, SystemID: seed.SystemImage.ID, Qty: 1},
	}, "idem-orphan", ""); err == nil {
		t.Fatal("expected items insertion error")
	}
	orders, _, err := base.ListOrders(context.Background(), appshared.OrderFilter{UserID: user.ID}, 10, 0)
	if err != nil {
		t.Fatalf("list orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("expected no orphan order, got %d", len(orders))
	}
}

// F5: the atomic cart->order path clears the cart before the coupon redemption
// is recorded; if the redemption then fails, the cart must be restored rather
// than silently wiped.
func TestOrderService_CreateOrderFromCart_RestoresCartOnRedemptionFailure(t *testing.T) {
	_, base := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, base)
	user := testutil.CreateUser(t, base, "couponfail", "couponfail@example.com", "pass")

	cartSvc := appcart.NewService(base, base, base)
	if _, err := cartSvc.Add(context.Background(), user.ID, seed.Package.ID, seed.SystemImage.ID, appshared.CartSpec{}, 1); err != nil {
		t.Fatalf("add cart: %v", err)
	}

	svc := apporder.NewService(base, base, base, base, base, base, base, base, base, nil, nil, nil, base, base, nil, base, base, base, nil, nil, nil)
	svc.SetCouponService(failingRedemptionCouponEngine{})
	if _, _, err := svc.CreateOrderFromCart(context.Background(), user.ID, "CNY", "idem-coupon-fail", "COUPON"); err == nil {
		t.Fatal("expected redemption failure")
	}
	cartItems, err := base.ListCartItems(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("list cart: %v", err)
	}
	if len(cartItems) != 1 {
		t.Fatalf("expected cart restored to 1 item, got %d", len(cartItems))
	}
}

// F10: when a concurrent create with the same idempotency key wins the insert
// race, the losing request must replay the winner's order instead of returning
// a raw unique-index error.
func TestOrderService_CreateOrderFromItems_ReplaysConcurrentIdemWinner(t *testing.T) {
	_, base := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, base)
	user := testutil.CreateUser(t, base, "idemrace", "idemrace@example.com", "pass")

	orders := &replayIdemRepo{GormRepo: base, userID: user.ID, idemKey: "idem-race"}
	svc := apporder.NewService(orders, base, base, base, base, base, base, base, base, nil, nil, nil, base, base, nil, base, base, base, nil, nil, nil)
	created, _, err := svc.CreateOrderFromItems(context.Background(), user.ID, "CNY", []appshared.OrderItemInput{
		{PackageID: seed.Package.ID, SystemID: seed.SystemImage.ID, Qty: 1},
	}, "idem-race", "")
	if err != nil {
		t.Fatalf("expected idempotent replay, got %v", err)
	}
	if created.OrderNo != "ORD-REPLAY-WINNER" {
		t.Fatalf("expected replay of the winner order, got %s", created.OrderNo)
	}
}

// F10: the payment submit path must replay a concurrently-created payment for
// the same idempotency key instead of surfacing a raw DB error.
func TestOrderService_SubmitPayment_ReplaysConcurrentIdemWinner(t *testing.T) {
	_, base := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, base, "payidemrace", "payidemrace@example.com", "pass")
	order := domain.Order{UserID: user.ID, OrderNo: "ORD-PAY-RACE", Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "CNY"}
	if err := base.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}

	payments := &replayPaymentRepo{GormRepo: base, orderID: order.ID, idemKey: "pay-idem-race"}
	svc := apporder.NewService(base, base, base, base, base, base, base, base, payments, nil, nil, nil, base, base, nil, base, base, base, nil, nil, nil)
	payment, err := svc.SubmitPayment(context.Background(), user.ID, order.ID, appshared.PaymentInput{
		Method: "manual", Amount: 1000, Currency: "CNY", TradeNo: "T-PAY-RACE",
	}, "pay-idem-race")
	if err != nil {
		t.Fatalf("expected idempotent replay, got %v", err)
	}
	if payment.IdempotencyKey != "pay-idem-race" {
		t.Fatalf("expected replayed payment with idem key, got %+v", payment)
	}
}

// F7: reusing a trade_no across two orders through MarkPaid must surface
// ErrConflict, not a raw unique-index DB error.
func TestOrderService_MarkPaid_RejectsCrossOrderTradeNoReuse(t *testing.T) {
	_, base := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, base, "markpaidcross", "markpaidcross@example.com", "pass")
	orderA := domain.Order{UserID: user.ID, OrderNo: "ORD-MP-A", Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "CNY"}
	orderB := domain.Order{UserID: user.ID, OrderNo: "ORD-MP-B", Status: domain.OrderStatusPendingPayment, TotalAmount: 2000, Currency: "CNY"}
	if err := base.CreateOrder(context.Background(), &orderA); err != nil {
		t.Fatalf("create order a: %v", err)
	}
	if err := base.CreateOrder(context.Background(), &orderB); err != nil {
		t.Fatalf("create order b: %v", err)
	}
	if err := base.CreateOrderItems(context.Background(), []domain.OrderItem{{OrderID: orderA.ID, Amount: 1000, Status: domain.OrderItemStatusPendingPayment, Action: "create", SpecJSON: "{}"}}); err != nil {
		t.Fatalf("create items a: %v", err)
	}
	if err := base.CreateOrderItems(context.Background(), []domain.OrderItem{{OrderID: orderB.ID, Amount: 2000, Status: domain.OrderItemStatusPendingPayment, Action: "create", SpecJSON: "{}"}}); err != nil {
		t.Fatalf("create items b: %v", err)
	}

	svc := apporder.NewService(base, base, base, base, base, base, base, base, base, nil, nil, nil, base, base, nil, base, base, base, nil, nil, nil)
	if _, err := svc.MarkPaid(context.Background(), 1, orderA.ID, appshared.PaymentInput{Method: "manual", Amount: 1000, Currency: "CNY", TradeNo: "T-MP-CROSS"}); err != nil {
		t.Fatalf("mark paid a: %v", err)
	}
	if _, err := svc.MarkPaid(context.Background(), 1, orderB.ID, appshared.PaymentInput{Method: "manual", Amount: 2000, Currency: "CNY", TradeNo: "T-MP-CROSS"}); err != appshared.ErrConflict {
		t.Fatalf("expected conflict for cross-order trade_no reuse, got %v", err)
	}
}

// F4: a second refund order for the same VPS must be rejected while a refund
// is already pending review, otherwise concurrent auto-approve could credit the
// wallet twice.
func TestOrderService_CreateRefundOrder_RejectsConcurrentPendingRefund(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "refundtwice", "refundtwice@example.com", "pass")
	_ = repo.UpsertSetting(context.Background(), domain.Setting{Key: "refund_requires_approval", ValueJSON: "true"})

	baseOrder := domain.Order{
		UserID:      user.ID,
		OrderNo:     "ORD-REFUND-TWICE-BASE",
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
		AutomationInstanceID: "2001",
		Name:                 "vm-refund-twice",
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
	first, _, err := svc.CreateRefundOrder(context.Background(), user.ID, inst.ID, "first refund")
	if err != nil {
		t.Fatalf("create first refund order: %v", err)
	}
	if first.Status != domain.OrderStatusPendingReview {
		t.Fatalf("expected first refund to stay pending review, got %s", first.Status)
	}
	if _, _, err := svc.CreateRefundOrder(context.Background(), user.ID, inst.ID, "second refund"); err != appshared.ErrConflict {
		t.Fatalf("expected conflict for concurrent pending refund, got %v", err)
	}
}

// F3: CancelOrder must not override an approval that committed between its read
// and write.
func TestOrderService_CancelOrder_DoesNotCancelConcurrentlyApprovedOrder(t *testing.T) {
	_, base := testutil.NewTestDB(t, false)
	user := testutil.CreateUser(t, base, "cancelafterapprove", "cancelafterapprove@example.com", "pass")
	order := domain.Order{UserID: user.ID, OrderNo: "ORD-CA-APPROVED", Status: domain.OrderStatusPendingReview, TotalAmount: 1000, Currency: "CNY"}
	if err := base.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	if err := base.CreateOrderItems(context.Background(), []domain.OrderItem{{OrderID: order.ID, Amount: 1000, Status: domain.OrderItemStatusPendingReview, Action: "create", SpecJSON: "{}"}}); err != nil {
		t.Fatalf("create items: %v", err)
	}

	// Simulate a concurrent admin approval by flipping the order to approved
	// right before CancelOrder's status write.
	orders := &flipBeforeStatusWriteRepo{GormRepo: base, flipOrderID: order.ID, flipStatus: domain.OrderStatusApproved}
	svc := apporder.NewService(orders, base, base, base, base, base, base, base, base, nil, nil, nil, base, base, nil, base, base, base, nil, nil, nil)
	if err := svc.CancelOrder(context.Background(), user.ID, order.ID); err != appshared.ErrConflict {
		t.Fatalf("expected conflict when order was approved concurrently, got %v", err)
	}
	updated, err := base.GetOrder(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if updated.Status == domain.OrderStatusCanceled {
		t.Fatalf("cancel must not override a concurrent approval")
	}
}
