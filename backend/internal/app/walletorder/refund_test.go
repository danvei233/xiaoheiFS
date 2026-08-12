package walletorder_test

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"testing"
	"time"
	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	appwalletorder "xiaoheiplay/internal/app/walletorder"
	"xiaoheiplay/internal/domain"

	"gorm.io/gorm"
	"xiaoheiplay/internal/testutil"
)

type fakeWalletRepo struct {
	balances map[int64]int64
}

func (f *fakeWalletRepo) GetWallet(ctx context.Context, userID int64) (domain.Wallet, error) {
	return domain.Wallet{UserID: userID, Balance: f.balances[userID]}, nil
}

func (f *fakeWalletRepo) UpsertWallet(ctx context.Context, wallet *domain.Wallet) error {
	if f.balances == nil {
		f.balances = map[int64]int64{}
	}
	f.balances[wallet.UserID] = wallet.Balance
	return nil
}

func (f *fakeWalletRepo) AddWalletTransaction(ctx context.Context, tx *domain.WalletTransaction) error {
	return nil
}

func (f *fakeWalletRepo) ListWalletTransactions(ctx context.Context, userID int64, limit, offset int) ([]domain.WalletTransaction, int, error) {
	return nil, 0, nil
}

func (f *fakeWalletRepo) AdjustWalletBalance(ctx context.Context, userID int64, amount int64, txType, refType string, refID int64, note string) (domain.Wallet, error) {
	if f.balances == nil {
		f.balances = map[int64]int64{}
	}
	f.balances[userID] += amount
	return domain.Wallet{UserID: userID, Balance: f.balances[userID]}, nil
}

func (f *fakeWalletRepo) HasWalletTransaction(ctx context.Context, userID int64, refType string, refID int64) (bool, error) {
	return false, nil
}

type fakeWalletOrderRepo struct {
	nextID  int64
	orders  map[int64]domain.WalletOrder
	wallets *fakeWalletRepo
}

func (f *fakeWalletOrderRepo) CreateWalletOrder(ctx context.Context, order *domain.WalletOrder) error {
	if f.orders == nil {
		f.orders = map[int64]domain.WalletOrder{}
	}
	f.nextID++
	order.ID = f.nextID
	f.orders[order.ID] = *order
	return nil
}

func (f *fakeWalletOrderRepo) GetWalletOrderByPaymentTradeNo(ctx context.Context, tradeNo string) (domain.WalletOrder, error) {
	return domain.WalletOrder{}, gorm.ErrRecordNotFound
}

func (f *fakeWalletOrderRepo) GetWalletOrder(ctx context.Context, id int64) (domain.WalletOrder, error) {
	if order, ok := f.orders[id]; ok {
		return order, nil
	}
	return domain.WalletOrder{}, appshared.ErrNotFound
}

func (f *fakeWalletOrderRepo) ListWalletOrders(ctx context.Context, userID int64, limit, offset int) ([]domain.WalletOrder, int, error) {
	if f.orders == nil {
		return nil, 0, nil
	}
	var all []domain.WalletOrder
	for _, o := range f.orders {
		if o.UserID == userID {
			all = append(all, o)
		}
	}
	// Deterministic order so pagination is stable in tests.
	sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })
	if offset >= len(all) {
		return nil, len(all), nil
	}
	end := offset + limit
	if limit <= 0 || end > len(all) {
		end = len(all)
	}
	return all[offset:end], len(all), nil
}

func (f *fakeWalletOrderRepo) ListWalletOrdersFiltered(ctx context.Context, userID int64, status, orderType string, limit, offset int) ([]domain.WalletOrder, int, error) {
	return nil, 0, nil
}
func (f *fakeWalletOrderRepo) SummarizeWalletOrders(context.Context, int64, time.Time, time.Time) (domain.WalletOrderSummary, error) {
	return domain.WalletOrderSummary{}, nil
}

func (f *fakeWalletOrderRepo) ListAllWalletOrders(ctx context.Context, status string, limit, offset int) ([]domain.WalletOrder, int, error) {
	return nil, 0, nil
}

func (f *fakeWalletOrderRepo) UpdateWalletOrderStatus(ctx context.Context, id int64, status domain.WalletOrderStatus, reviewedBy *int64, reason string) error {
	order, ok := f.orders[id]
	if !ok {
		return appshared.ErrNotFound
	}
	order.Status = status
	f.orders[id] = order
	return nil
}

func (f *fakeWalletOrderRepo) UpdateWalletOrderStatusIfCurrent(ctx context.Context, id int64, currentStatus, targetStatus domain.WalletOrderStatus, reviewedBy *int64, reason string) (bool, error) {
	order, ok := f.orders[id]
	if !ok {
		return false, appshared.ErrNotFound
	}
	if order.Status != currentStatus {
		return false, nil
	}
	order.Status = targetStatus
	f.orders[id] = order
	return true, nil
}

func (f *fakeWalletOrderRepo) ApproveWalletOrderAtomic(ctx context.Context, order domain.WalletOrder, adminID int64, amount int64, txType string) (domain.Wallet, bool, error) {
	stored, ok := f.orders[order.ID]
	if !ok {
		return domain.Wallet{}, false, appshared.ErrNotFound
	}
	if stored.Status != domain.WalletOrderPendingReview {
		return domain.Wallet{}, false, nil
	}
	stored.Status = domain.WalletOrderApproved
	stored.ReviewedBy = &adminID
	f.orders[order.ID] = stored
	if f.wallets == nil {
		return domain.Wallet{UserID: order.UserID, Balance: amount}, true, nil
	}
	wallet, err := f.wallets.AdjustWalletBalance(ctx, order.UserID, amount, txType, "wallet_order", order.ID, string(order.Type))
	return wallet, true, err
}

func (f *fakeWalletOrderRepo) UpdateWalletOrderMeta(ctx context.Context, id int64, metaJSON string) error {
	order, ok := f.orders[id]
	if !ok {
		return appshared.ErrNotFound
	}
	order.MetaJSON = metaJSON
	f.orders[id] = order
	return nil
}

type fakeSettingsRepo struct {
	values map[string]string
}

func (f *fakeSettingsRepo) UpdateSettingAtomic(ctx context.Context, key string, mutate func(current string) (string, error)) error {
	current, err := f.GetSetting(ctx, key)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			current = domain.Setting{}
		} else {
			return err
		}
	}
	next, err := mutate(current.ValueJSON)
	if err != nil {
		return err
	}
	return f.UpsertSetting(ctx, domain.Setting{Key: key, ValueJSON: next})
}
func (f *fakeSettingsRepo) GetSetting(ctx context.Context, key string) (domain.Setting, error) {
	if v, ok := f.values[key]; ok {
		return domain.Setting{Key: key, ValueJSON: v}, nil
	}
	return domain.Setting{}, appshared.ErrNotFound
}

func (f *fakeSettingsRepo) UpsertSetting(ctx context.Context, setting domain.Setting) error {
	if f.values == nil {
		f.values = map[string]string{}
	}
	f.values[setting.Key] = setting.ValueJSON
	return nil
}

func (f *fakeSettingsRepo) UpsertSettings(ctx context.Context, settings []domain.Setting) error {
	for _, setting := range settings {
		if err := f.UpsertSetting(ctx, setting); err != nil {
			return err
		}
	}
	return nil
}

func (f *fakeSettingsRepo) ListSettings(ctx context.Context) ([]domain.Setting, error) {
	return nil, nil
}

func (f *fakeSettingsRepo) ListEmailTemplates(ctx context.Context) ([]domain.EmailTemplate, error) {
	return nil, nil
}

func (f *fakeSettingsRepo) GetEmailTemplate(ctx context.Context, id int64) (domain.EmailTemplate, error) {
	return domain.EmailTemplate{}, appshared.ErrNotFound
}

func (f *fakeSettingsRepo) UpsertEmailTemplate(ctx context.Context, tmpl *domain.EmailTemplate) error {
	return nil
}

func (f *fakeSettingsRepo) DeleteEmailTemplate(ctx context.Context, id int64) error {
	return nil
}

type fakeVPSRepo struct {
	inst domain.VPSInstance
}

func (f *fakeVPSRepo) CreateInstance(ctx context.Context, inst *domain.VPSInstance) error { return nil }
func (f *fakeVPSRepo) GetInstance(ctx context.Context, id int64) (domain.VPSInstance, error) {
	if f.inst.ID == id {
		return f.inst, nil
	}
	return domain.VPSInstance{}, appshared.ErrNotFound
}
func (f *fakeVPSRepo) GetInstanceByOrderItem(ctx context.Context, orderItemID int64) (domain.VPSInstance, error) {
	if f.inst.OrderItemID == orderItemID {
		return f.inst, nil
	}
	return domain.VPSInstance{}, appshared.ErrNotFound
}
func (f *fakeVPSRepo) ListInstancesByUserFiltered(ctx context.Context, userID int64, filter appports.VPSUserInstanceFilter, limit, offset int) ([]domain.VPSInstance, int, error) {
	if limit <= 0 {
		limit = 20
	}
	items, err := f.ListInstancesByUser(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	start := offset
	if start > len(items) {
		start = len(items)
	}
	end := start + limit
	if end > len(items) {
		end = len(items)
	}
	return items[start:end], len(items), nil
}
func (f *fakeVPSRepo) ListInstancesByUser(ctx context.Context, userID int64) ([]domain.VPSInstance, error) {
	return nil, nil
}
func (f *fakeVPSRepo) ListInstances(ctx context.Context, limit, offset int) ([]domain.VPSInstance, int, error) {
	return nil, 0, nil
}
func (f *fakeVPSRepo) ListInstancesFiltered(ctx context.Context, filter appports.VPSInstanceFilter, limit, offset int) ([]domain.VPSInstance, int, error) {
	return nil, 0, nil
}
func (f *fakeVPSRepo) ListInstancesExpiring(ctx context.Context, before time.Time) ([]domain.VPSInstance, error) {
	return nil, nil
}
func (f *fakeVPSRepo) DeleteInstance(ctx context.Context, id int64) error { return nil }
func (f *fakeVPSRepo) UpdateInstanceStatus(ctx context.Context, id int64, status domain.VPSStatus, automationState int) error {
	return nil
}
func (f *fakeVPSRepo) UpdateInstanceAdminStatus(ctx context.Context, id int64, status domain.VPSAdminStatus) error {
	return nil
}
func (f *fakeVPSRepo) UpdateInstanceExpireAt(ctx context.Context, id int64, expireAt time.Time) error {
	return nil
}
func (f *fakeVPSRepo) UpdateInstancePanelCache(ctx context.Context, id int64, panelURL string) error {
	return nil
}
func (f *fakeVPSRepo) UpdateInstanceSpec(ctx context.Context, id int64, specJSON string) error {
	return nil
}
func (f *fakeVPSRepo) UpdateInstanceAccessInfo(ctx context.Context, id int64, accessJSON string) error {
	return nil
}
func (f *fakeVPSRepo) UpdateInstanceEmergencyRenewAt(ctx context.Context, id int64, at time.Time) error {
	return nil
}
func (f *fakeVPSRepo) UpdateInstanceLocal(ctx context.Context, inst domain.VPSInstance) error {
	return nil
}

type fakeOrderItemRepo struct {
	item domain.OrderItem
}

func (f *fakeOrderItemRepo) CreateOrderItems(ctx context.Context, items []domain.OrderItem) error {
	return nil
}
func (f *fakeOrderItemRepo) ListOrderItems(ctx context.Context, orderID int64) ([]domain.OrderItem, error) {
	return nil, nil
}
func (f *fakeOrderItemRepo) GetOrderItem(ctx context.Context, id int64) (domain.OrderItem, error) {
	if f.item.ID == id {
		return f.item, nil
	}
	return domain.OrderItem{}, appshared.ErrNotFound
}
func (f *fakeOrderItemRepo) UpdateOrderItemStatus(ctx context.Context, id int64, status domain.OrderItemStatus) error {
	return nil
}
func (f *fakeOrderItemRepo) ClaimOrderItemForProvision(ctx context.Context, id int64) (bool, error) {
	if f.item.ID != id {
		return false, appshared.ErrNotFound
	}
	if f.item.Status != domain.OrderItemStatusApproved && f.item.Status != domain.OrderItemStatusFailed {
		return false, nil
	}
	f.item.Status = domain.OrderItemStatusProvisioning
	return true, nil
}
func (f *fakeOrderItemRepo) UpdateOrderItemAutomation(ctx context.Context, id int64, automationID string) error {
	return nil
}
func (f *fakeOrderItemRepo) HasPendingRenewOrder(ctx context.Context, userID, vpsID int64) (bool, error) {
	return false, nil
}
func (f *fakeOrderItemRepo) HasPendingResizeOrder(ctx context.Context, userID, vpsID int64) (bool, error) {
	return false, nil
}
func (f *fakeOrderItemRepo) HasPendingRefundOrder(ctx context.Context, userID, vpsID int64) (bool, error) {
	return false, nil
}

func TestWalletOrderService_NormalizeRefundCurve_RejectsOutOfRangePercent(t *testing.T) {
	points := appwalletorder.NormalizeRefundCurve([]appwalletorder.RefundCurvePoint{
		{Percent: -1, Ratio: 1},
		{Percent: 50, Ratio: 0.5},
		{Percent: 720, Ratio: 0.2}, // e.g. an admin entering hours as if percent
	})
	if len(points) != 1 || points[0].Percent != 50 {
		t.Fatalf("expected only the in-range point, got %+v", points)
	}
}

func TestWalletOrderService_RequestRefund(t *testing.T) {
	settings := &fakeSettingsRepo{values: map[string]string{"refund_requires_approval": "false"}}
	wallets := &fakeWalletRepo{}
	orders := &fakeWalletOrderRepo{wallets: wallets}
	item := domain.OrderItem{ID: 10, Amount: 200000, Status: domain.OrderItemStatusActive, Action: "create", SpecJSON: "{}"}
	inst := domain.VPSInstance{
		ID:                   1,
		UserID:               1,
		OrderItemID:          item.ID,
		AutomationInstanceID: "100",
		CreatedAt:            time.Now(),
	}
	expire := time.Now().Add(30 * 24 * time.Hour)
	inst.ExpireAt = &expire

	vps := &fakeVPSRepo{inst: inst}
	items := &fakeOrderItemRepo{item: item}
	auto := &testutil.FakeAutomationClient{}
	autoResolver := &testutil.FakeAutomationResolver{Client: auto}
	svc := appwalletorder.NewService(orders, wallets, settings, vps, items, autoResolver, nil)

	if _, _, err := svc.RequestRefund(context.Background(), 2, inst.ID, "no"); err != appshared.ErrForbidden {
		t.Fatalf("expected forbidden, got %v", err)
	}
	refund, wallet, err := svc.RequestRefund(context.Background(), inst.UserID, inst.ID, "test")
	if err != nil {
		t.Fatalf("refund: %v", err)
	}
	if refund.Status != domain.WalletOrderApproved || wallet == nil || wallet.Balance <= 0 {
		t.Fatalf("expected auto-approved refund")
	}
}

func TestWalletOrderService_RequestRefund_UsesItemAmount(t *testing.T) {
	settings := &fakeSettingsRepo{values: map[string]string{"refund_requires_approval": "false"}}
	wallets := &fakeWalletRepo{}
	orders := &fakeWalletOrderRepo{wallets: wallets}
	item := domain.OrderItem{ID: 20, Amount: 200000, DurationMonths: 3, Status: domain.OrderItemStatusActive, Action: "create", SpecJSON: "{}"}
	inst := domain.VPSInstance{
		ID:                   2,
		UserID:               1,
		OrderItemID:          item.ID,
		AutomationInstanceID: "100",
		MonthlyPrice:         3000,
		CreatedAt:            time.Now().Add(-time.Hour),
	}
	expire := time.Now().Add(90*24*time.Hour - time.Hour)
	inst.ExpireAt = &expire

	vps := &fakeVPSRepo{inst: inst}
	items := &fakeOrderItemRepo{item: item}
	auto := &testutil.FakeAutomationClient{}
	autoResolver := &testutil.FakeAutomationResolver{Client: auto}
	svc := appwalletorder.NewService(orders, wallets, settings, vps, items, autoResolver, nil)

	refund, _, err := svc.RequestRefund(context.Background(), inst.UserID, inst.ID, "test")
	if err != nil {
		t.Fatalf("refund: %v", err)
	}
	// The refund base must be the amount actually paid for the item (which
	// covers the prepaid DurationMonths), not the single-month price.
	if refund.Amount != 200000 {
		t.Fatalf("expected refund based on item amount 200000, got %d", refund.Amount)
	}
}

func TestWalletOrderService_RequestRefund_MultiMonthPrepayProrated(t *testing.T) {
	settings := &fakeSettingsRepo{values: map[string]string{"refund_requires_approval": "false"}}
	wallets := &fakeWalletRepo{}
	orders := &fakeWalletOrderRepo{wallets: wallets}
	// 3-month prepay: paid 3 * monthly, ~1 hour elapsed (full-refund window).
	item := domain.OrderItem{ID: 30, Amount: 9000, DurationMonths: 3, Status: domain.OrderItemStatusActive, Action: "create", SpecJSON: "{}"}
	inst := domain.VPSInstance{
		ID:                   3,
		UserID:               1,
		OrderItemID:          item.ID,
		AutomationInstanceID: "100",
		MonthlyPrice:         3000,
		CreatedAt:            time.Now().Add(-time.Hour),
	}
	expire := time.Now().Add(90*24*time.Hour - time.Hour)
	inst.ExpireAt = &expire

	vps := &fakeVPSRepo{inst: inst}
	items := &fakeOrderItemRepo{item: item}
	auto := &testutil.FakeAutomationClient{}
	autoResolver := &testutil.FakeAutomationResolver{Client: auto}
	svc := appwalletorder.NewService(orders, wallets, settings, vps, items, autoResolver, nil)

	refund, _, err := svc.RequestRefund(context.Background(), inst.UserID, inst.ID, "test")
	if err != nil {
		t.Fatalf("refund: %v", err)
	}
	// Full-refund window: refund equals the full prepaid amount (3 months),
	// not just one monthly price.
	if refund.Amount != 9000 {
		t.Fatalf("expected full refund of prepaid amount 9000, got %d", refund.Amount)
	}
}

// TestWalletOrderService_RequestRefund_UsesCurrentBillingPeriod locks the
// walletorder refund-copy regression (m016 F2 sync): the elapsed ratio must be
// measured against the current billing period recorded in SpecJSON, not the
// instance lifetime. A renewal 10 days ago for a 30-day period refunds in full
// when the full-refund window is 15 days; the lifetime-based copy measured 60
// elapsed days and returned zero.
// TestWalletOrderService_AutoRefundOnAdminDeleteIdempotent locks the F4/F5
// fix: retrying an admin delete after an external DeleteHost failure must not
// create a second refund order / double-credit the wallet. The second call
// returns the existing admin_delete refund instead of a new one.
func TestWalletOrderService_AutoRefundOnAdminDeleteIdempotent(t *testing.T) {
	settings := &fakeSettingsRepo{values: map[string]string{
		"refund_on_admin_delete":   "true",
		"refund_requires_approval": "false",
	}}
	wallets := &fakeWalletRepo{balances: map[int64]int64{1: 0}}
	orders := &fakeWalletOrderRepo{wallets: wallets}
	item := domain.OrderItem{ID: 50, Amount: 9000, DurationMonths: 3, Status: domain.OrderItemStatusActive, Action: "create", SpecJSON: "{}"}
	inst := domain.VPSInstance{
		ID:                   5,
		UserID:               1,
		OrderItemID:          item.ID,
		AutomationInstanceID: "200",
		MonthlyPrice:         3000,
		CreatedAt:            time.Now().Add(-time.Hour),
	}
	expire := time.Now().Add(90*24*time.Hour - time.Hour)
	inst.ExpireAt = &expire
	vps := &fakeVPSRepo{inst: inst}
	items := &fakeOrderItemRepo{item: item}
	auto := &testutil.FakeAutomationClient{}
	autoResolver := &testutil.FakeAutomationResolver{Client: auto}
	svc := appwalletorder.NewService(orders, wallets, settings, vps, items, autoResolver, nil)

	first, _, err := svc.AutoRefundOnAdminDelete(context.Background(), 9, inst.ID, "delete")
	if err != nil {
		t.Fatalf("first auto refund: %v", err)
	}
	if first.Status != domain.WalletOrderApproved {
		t.Fatalf("expected first refund auto-approved, got %s", first.Status)
	}
	if wallets.balances[1] != 9000 {
		t.Fatalf("expected wallet credited 9000 once, got %d", wallets.balances[1])
	}

	// Second call (simulated retry after DeleteHost failed) must reuse the
	// existing approved admin_delete refund, not create a new one.
	second, _, err := svc.AutoRefundOnAdminDelete(context.Background(), 9, inst.ID, "delete retry")
	if err != nil {
		t.Fatalf("second auto refund: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("expected the same refund order on retry, got %d != %d", second.ID, first.ID)
	}
	if wallets.balances[1] != 9000 {
		t.Fatalf("expected wallet still 9000 after idempotent retry, got %d", wallets.balances[1])
	}
	count := 0
	for _, o := range orders.orders {
		if o.Type == domain.WalletOrderRefund {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly one refund order, got %d", count)
	}
}

func TestWalletOrderService_RequestRefund_UsesCurrentBillingPeriod(t *testing.T) {
	settings := &fakeSettingsRepo{values: map[string]string{
		"refund_requires_approval": "false",
		"refund_full_days":         "15",
		"refund_prorate_days":      "15",
		"refund_no_refund_days":    "15",
	}}
	wallets := &fakeWalletRepo{}
	orders := &fakeWalletOrderRepo{wallets: wallets}
	item := domain.OrderItem{ID: 40, Amount: 30000, DurationMonths: 1, Status: domain.OrderItemStatusActive, Action: "create", SpecJSON: "{}"}

	now := time.Now()
	periodStart := now.Add(-10 * 24 * time.Hour)
	periodEnd := now.Add(20 * 24 * time.Hour)
	specJSON, _ := jsonMarshal(map[string]any{
		"current_period_start": periodStart.UTC().Format(time.RFC3339),
		"current_period_end":   periodEnd.UTC().Format(time.RFC3339),
	})
	inst := domain.VPSInstance{
		ID:                   4,
		UserID:               1,
		OrderItemID:          item.ID,
		AutomationInstanceID: "100",
		MonthlyPrice:         3000,
		CreatedAt:            now.Add(-60 * 24 * time.Hour),
		SpecJSON:             specJSON,
	}
	expire := now.Add(20 * 24 * time.Hour)
	inst.ExpireAt = &expire

	vps := &fakeVPSRepo{inst: inst}
	items := &fakeOrderItemRepo{item: item}
	auto := &testutil.FakeAutomationClient{}
	autoResolver := &testutil.FakeAutomationResolver{Client: auto}
	svc := appwalletorder.NewService(orders, wallets, settings, vps, items, autoResolver, nil)

	refund, _, err := svc.RequestRefund(context.Background(), inst.UserID, inst.ID, "test")
	if err != nil {
		t.Fatalf("refund: %v", err)
	}
	if refund.Amount != 30000 {
		t.Fatalf("expected full refund anchored to current billing period (30000), got %d", refund.Amount)
	}
}

func jsonMarshal(v any) (string, error) {
	b, err := json.Marshal(v)
	return string(b), err
}
