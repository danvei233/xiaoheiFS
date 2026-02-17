package walletorder_test

import (
	"context"
	"testing"
	"time"
	appshared "xiaoheiplay/internal/app/shared"
	appwalletorder "xiaoheiplay/internal/app/walletorder"
	"xiaoheiplay/internal/domain"
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
	nextID int64
	orders map[int64]domain.WalletOrder
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

func (f *fakeWalletOrderRepo) GetWalletOrder(ctx context.Context, id int64) (domain.WalletOrder, error) {
	if order, ok := f.orders[id]; ok {
		return order, nil
	}
	return domain.WalletOrder{}, appshared.ErrNotFound
}

func (f *fakeWalletOrderRepo) ListWalletOrders(ctx context.Context, userID int64, limit, offset int) ([]domain.WalletOrder, int, error) {
	return nil, 0, nil
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

type fakeSettingsRepo struct {
	values map[string]string
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
func (f *fakeVPSRepo) ListInstancesByUser(ctx context.Context, userID int64) ([]domain.VPSInstance, error) {
	return nil, nil
}
func (f *fakeVPSRepo) ListInstances(ctx context.Context, limit, offset int) ([]domain.VPSInstance, int, error) {
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

func TestWalletOrderService_RequestRefund(t *testing.T) {
	settings := &fakeSettingsRepo{values: map[string]string{"refund_requires_approval": "false"}}
	wallets := &fakeWalletRepo{}
	orders := &fakeWalletOrderRepo{}
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

func TestWalletOrderService_RequestRefund_UsesInstanceMonthlyPrice(t *testing.T) {
	settings := &fakeSettingsRepo{values: map[string]string{"refund_requires_approval": "false"}}
	wallets := &fakeWalletRepo{}
	orders := &fakeWalletOrderRepo{}
	item := domain.OrderItem{ID: 20, Amount: 200000, Status: domain.OrderItemStatusActive, Action: "create", SpecJSON: "{}"}
	inst := domain.VPSInstance{
		ID:                   2,
		UserID:               1,
		OrderItemID:          item.ID,
		AutomationInstanceID: "100",
		MonthlyPrice:         3000,
		CreatedAt:            time.Now(),
	}
	expire := time.Now().Add(30 * 24 * time.Hour)
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
	if refund.Amount != 3000 {
		t.Fatalf("expected refund based on instance monthly price 3000, got %d", refund.Amount)
	}
}
