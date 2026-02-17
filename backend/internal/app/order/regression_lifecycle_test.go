package order

import (
	"context"
	"strings"
	"testing"
	"time"

	apprealname "xiaoheiplay/internal/app/realname"
	appvps "xiaoheiplay/internal/app/vps"
	"xiaoheiplay/internal/domain"
)

type fakeLifecycleSettingsRepo struct {
	values map[string]string
}

func (f *fakeLifecycleSettingsRepo) GetSetting(ctx context.Context, key string) (domain.Setting, error) {
	if f.values == nil {
		return domain.Setting{}, ErrNotFound
	}
	val, ok := f.values[key]
	if !ok {
		return domain.Setting{}, ErrNotFound
	}
	return domain.Setting{Key: key, ValueJSON: val, UpdatedAt: time.Now()}, nil
}

func (f *fakeLifecycleSettingsRepo) UpsertSetting(ctx context.Context, setting domain.Setting) error {
	if f.values == nil {
		f.values = map[string]string{}
	}
	f.values[setting.Key] = setting.ValueJSON
	return nil
}

func (f *fakeLifecycleSettingsRepo) ListSettings(ctx context.Context) ([]domain.Setting, error) {
	items := make([]domain.Setting, 0, len(f.values))
	for key, val := range f.values {
		items = append(items, domain.Setting{Key: key, ValueJSON: val, UpdatedAt: time.Now()})
	}
	return items, nil
}

func (f *fakeLifecycleSettingsRepo) ListEmailTemplates(ctx context.Context) ([]domain.EmailTemplate, error) {
	return nil, nil
}

func (f *fakeLifecycleSettingsRepo) GetEmailTemplate(ctx context.Context, id int64) (domain.EmailTemplate, error) {
	return domain.EmailTemplate{}, ErrNotFound
}

func (f *fakeLifecycleSettingsRepo) UpsertEmailTemplate(ctx context.Context, tmpl *domain.EmailTemplate) error {
	return nil
}

func (f *fakeLifecycleSettingsRepo) DeleteEmailTemplate(ctx context.Context, id int64) error {
	return nil
}

type fakeLifecycleVPSRepo struct {
	inst          domain.VPSInstance
	updates       []time.Time
	deletes       []int64
	expiring      []domain.VPSInstance
	statusUpdates []struct {
		ID              int64
		Status          domain.VPSStatus
		AutomationState int
	}
	adminUpdates []struct {
		ID     int64
		Status domain.VPSAdminStatus
	}
}

func (f *fakeLifecycleVPSRepo) CreateInstance(ctx context.Context, inst *domain.VPSInstance) error {
	return nil
}
func (f *fakeLifecycleVPSRepo) GetInstance(ctx context.Context, id int64) (domain.VPSInstance, error) {
	if f.inst.ID == id {
		return f.inst, nil
	}
	return domain.VPSInstance{}, ErrNotFound
}
func (f *fakeLifecycleVPSRepo) GetInstanceByOrderItem(ctx context.Context, orderItemID int64) (domain.VPSInstance, error) {
	return domain.VPSInstance{}, ErrNotFound
}
func (f *fakeLifecycleVPSRepo) ListInstancesByUser(ctx context.Context, userID int64) ([]domain.VPSInstance, error) {
	return nil, nil
}
func (f *fakeLifecycleVPSRepo) ListInstances(ctx context.Context, limit, offset int) ([]domain.VPSInstance, int, error) {
	return nil, 0, nil
}
func (f *fakeLifecycleVPSRepo) ListInstancesExpiring(ctx context.Context, before time.Time) ([]domain.VPSInstance, error) {
	return f.expiring, nil
}
func (f *fakeLifecycleVPSRepo) DeleteInstance(ctx context.Context, id int64) error {
	f.deletes = append(f.deletes, id)
	return nil
}
func (f *fakeLifecycleVPSRepo) UpdateInstanceStatus(ctx context.Context, id int64, status domain.VPSStatus, automationState int) error {
	f.statusUpdates = append(f.statusUpdates, struct {
		ID              int64
		Status          domain.VPSStatus
		AutomationState int
	}{ID: id, Status: status, AutomationState: automationState})
	return nil
}
func (f *fakeLifecycleVPSRepo) UpdateInstanceAdminStatus(ctx context.Context, id int64, status domain.VPSAdminStatus) error {
	f.adminUpdates = append(f.adminUpdates, struct {
		ID     int64
		Status domain.VPSAdminStatus
	}{ID: id, Status: status})
	return nil
}
func (f *fakeLifecycleVPSRepo) UpdateInstanceExpireAt(ctx context.Context, id int64, expireAt time.Time) error {
	f.updates = append(f.updates, expireAt)
	return nil
}
func (f *fakeLifecycleVPSRepo) UpdateInstancePanelCache(ctx context.Context, id int64, panelURL string) error {
	return nil
}
func (f *fakeLifecycleVPSRepo) UpdateInstanceSpec(ctx context.Context, id int64, specJSON string) error {
	return nil
}
func (f *fakeLifecycleVPSRepo) UpdateInstanceAccessInfo(ctx context.Context, id int64, accessJSON string) error {
	return nil
}
func (f *fakeLifecycleVPSRepo) UpdateInstanceEmergencyRenewAt(ctx context.Context, id int64, at time.Time) error {
	return nil
}
func (f *fakeLifecycleVPSRepo) UpdateInstanceLocal(ctx context.Context, inst domain.VPSInstance) error {
	return nil
}

type fakeLifecycleOrderRepo struct {
	nextID int64
	orders map[int64]domain.Order
}

func (f *fakeLifecycleOrderRepo) CreateOrder(ctx context.Context, order *domain.Order) error {
	if f.orders == nil {
		f.orders = map[int64]domain.Order{}
	}
	f.nextID++
	order.ID = f.nextID
	f.orders[order.ID] = *order
	return nil
}
func (f *fakeLifecycleOrderRepo) GetOrder(ctx context.Context, id int64) (domain.Order, error) {
	if order, ok := f.orders[id]; ok {
		return order, nil
	}
	return domain.Order{}, ErrNotFound
}
func (f *fakeLifecycleOrderRepo) GetOrderByNo(ctx context.Context, orderNo string) (domain.Order, error) {
	return domain.Order{}, ErrNotFound
}
func (f *fakeLifecycleOrderRepo) GetOrderByIdempotencyKey(ctx context.Context, userID int64, key string) (domain.Order, error) {
	return domain.Order{}, ErrNotFound
}
func (f *fakeLifecycleOrderRepo) UpdateOrderStatus(ctx context.Context, id int64, status domain.OrderStatus) error {
	order, ok := f.orders[id]
	if !ok {
		return ErrNotFound
	}
	order.Status = status
	f.orders[id] = order
	return nil
}
func (f *fakeLifecycleOrderRepo) UpdateOrderMeta(ctx context.Context, order domain.Order) error {
	if f.orders == nil {
		f.orders = map[int64]domain.Order{}
	}
	f.orders[order.ID] = order
	return nil
}
func (f *fakeLifecycleOrderRepo) ListOrders(ctx context.Context, filter OrderFilter, limit, offset int) ([]domain.Order, int, error) {
	return nil, 0, nil
}
func (f *fakeLifecycleOrderRepo) DeleteOrder(ctx context.Context, id int64) error { return nil }

type fakeLifecycleOrderItemRepo struct {
	items        []domain.OrderItem
	nextID       int64
	pendingRenew bool
}

func (f *fakeLifecycleOrderItemRepo) CreateOrderItems(ctx context.Context, items []domain.OrderItem) error {
	for i := range items {
		if items[i].ID == 0 {
			f.nextID++
			items[i].ID = f.nextID
		}
		f.items = append(f.items, items[i])
	}
	return nil
}
func (f *fakeLifecycleOrderItemRepo) ListOrderItems(ctx context.Context, orderID int64) ([]domain.OrderItem, error) {
	out := make([]domain.OrderItem, 0)
	for _, item := range f.items {
		if item.OrderID == orderID {
			out = append(out, item)
		}
	}
	return out, nil
}
func (f *fakeLifecycleOrderItemRepo) GetOrderItem(ctx context.Context, id int64) (domain.OrderItem, error) {
	for _, item := range f.items {
		if item.ID == id {
			return item, nil
		}
	}
	return domain.OrderItem{}, ErrNotFound
}
func (f *fakeLifecycleOrderItemRepo) UpdateOrderItemStatus(ctx context.Context, id int64, status domain.OrderItemStatus) error {
	for i := range f.items {
		if f.items[i].ID == id {
			f.items[i].Status = status
			return nil
		}
	}
	return ErrNotFound
}
func (f *fakeLifecycleOrderItemRepo) UpdateOrderItemAutomation(ctx context.Context, id int64, automationID string) error {
	return nil
}
func (f *fakeLifecycleOrderItemRepo) HasPendingRenewOrder(ctx context.Context, userID, vpsID int64) (bool, error) {
	return f.pendingRenew, nil
}
func (f *fakeLifecycleOrderItemRepo) HasPendingResizeOrder(ctx context.Context, userID, vpsID int64) (bool, error) {
	return false, nil
}
func (f *fakeLifecycleOrderItemRepo) HasPendingRefundOrder(ctx context.Context, userID, vpsID int64) (bool, error) {
	return false, nil
}

type fakeLifecycleRealNameRepo struct {
	latest domain.RealNameVerification
	has    bool
}

func (f *fakeLifecycleRealNameRepo) CreateRealNameVerification(ctx context.Context, record *domain.RealNameVerification) error {
	f.latest = *record
	f.has = true
	return nil
}

func (f *fakeLifecycleRealNameRepo) GetLatestRealNameVerification(ctx context.Context, userID int64) (domain.RealNameVerification, error) {
	if !f.has || f.latest.UserID != userID {
		return domain.RealNameVerification{}, ErrNotFound
	}
	return f.latest, nil
}

func (f *fakeLifecycleRealNameRepo) ListRealNameVerifications(ctx context.Context, userID *int64, limit, offset int) ([]domain.RealNameVerification, int, error) {
	if !f.has {
		return nil, 0, nil
	}
	if userID != nil && f.latest.UserID != *userID {
		return nil, 0, nil
	}
	return []domain.RealNameVerification{f.latest}, 1, nil
}

func (f *fakeLifecycleRealNameRepo) UpdateRealNameStatus(ctx context.Context, id int64, status string, reason string, verifiedAt *time.Time) error {
	if !f.has || f.latest.ID != id {
		return ErrNotFound
	}
	f.latest.Status = status
	f.latest.Reason = reason
	f.latest.VerifiedAt = verifiedAt
	return nil
}

type fakeLifecycleAutomationClient struct {
	RenewCalls []struct {
		HostID int64
		Next   time.Time
	}
	DeleteCalls []int64
	LockCalls   []int64
}

func (f *fakeLifecycleAutomationClient) ClientForGoodsType(ctx context.Context, goodsTypeID int64) (AutomationClient, error) {
	_ = ctx
	_ = goodsTypeID
	return f, nil
}

func (f *fakeLifecycleAutomationClient) CreateHost(ctx context.Context, req AutomationCreateHostRequest) (AutomationCreateHostResult, error) {
	return AutomationCreateHostResult{}, nil
}
func (f *fakeLifecycleAutomationClient) GetHostInfo(ctx context.Context, hostID int64) (AutomationHostInfo, error) {
	return AutomationHostInfo{}, nil
}
func (f *fakeLifecycleAutomationClient) ListHostSimple(ctx context.Context, searchTag string) ([]AutomationHostSimple, error) {
	return nil, nil
}
func (f *fakeLifecycleAutomationClient) ElasticUpdate(ctx context.Context, req AutomationElasticUpdateRequest) error {
	return nil
}
func (f *fakeLifecycleAutomationClient) RenewHost(ctx context.Context, hostID int64, nextDueDate time.Time) error {
	f.RenewCalls = append(f.RenewCalls, struct {
		HostID int64
		Next   time.Time
	}{HostID: hostID, Next: nextDueDate})
	return nil
}
func (f *fakeLifecycleAutomationClient) LockHost(ctx context.Context, hostID int64) error {
	f.LockCalls = append(f.LockCalls, hostID)
	return nil
}
func (f *fakeLifecycleAutomationClient) UnlockHost(ctx context.Context, hostID int64) error {
	return nil
}
func (f *fakeLifecycleAutomationClient) DeleteHost(ctx context.Context, hostID int64) error {
	f.DeleteCalls = append(f.DeleteCalls, hostID)
	return nil
}
func (f *fakeLifecycleAutomationClient) StartHost(ctx context.Context, hostID int64) error {
	return nil
}
func (f *fakeLifecycleAutomationClient) ShutdownHost(ctx context.Context, hostID int64) error {
	return nil
}
func (f *fakeLifecycleAutomationClient) RebootHost(ctx context.Context, hostID int64) error {
	return nil
}
func (f *fakeLifecycleAutomationClient) ResetOS(ctx context.Context, hostID int64, templateID int64, password string) error {
	return nil
}
func (f *fakeLifecycleAutomationClient) ResetOSPassword(ctx context.Context, hostID int64, password string) error {
	return nil
}
func (f *fakeLifecycleAutomationClient) ListSnapshots(ctx context.Context, hostID int64) ([]AutomationSnapshot, error) {
	return nil, nil
}
func (f *fakeLifecycleAutomationClient) CreateSnapshot(ctx context.Context, hostID int64) error {
	return nil
}
func (f *fakeLifecycleAutomationClient) DeleteSnapshot(ctx context.Context, hostID int64, snapshotID int64) error {
	return nil
}
func (f *fakeLifecycleAutomationClient) RestoreSnapshot(ctx context.Context, hostID int64, snapshotID int64) error {
	return nil
}
func (f *fakeLifecycleAutomationClient) ListBackups(ctx context.Context, hostID int64) ([]AutomationBackup, error) {
	return nil, nil
}
func (f *fakeLifecycleAutomationClient) CreateBackup(ctx context.Context, hostID int64) error {
	return nil
}
func (f *fakeLifecycleAutomationClient) DeleteBackup(ctx context.Context, hostID int64, backupID int64) error {
	return nil
}
func (f *fakeLifecycleAutomationClient) RestoreBackup(ctx context.Context, hostID int64, backupID int64) error {
	return nil
}
func (f *fakeLifecycleAutomationClient) ListFirewallRules(ctx context.Context, hostID int64) ([]AutomationFirewallRule, error) {
	return nil, nil
}
func (f *fakeLifecycleAutomationClient) AddFirewallRule(ctx context.Context, req AutomationFirewallRuleCreate) error {
	return nil
}
func (f *fakeLifecycleAutomationClient) DeleteFirewallRule(ctx context.Context, hostID int64, ruleID int64) error {
	return nil
}
func (f *fakeLifecycleAutomationClient) ListPortMappings(ctx context.Context, hostID int64) ([]AutomationPortMapping, error) {
	return nil, nil
}
func (f *fakeLifecycleAutomationClient) AddPortMapping(ctx context.Context, req AutomationPortMappingCreate) error {
	return nil
}
func (f *fakeLifecycleAutomationClient) DeletePortMapping(ctx context.Context, hostID int64, mappingID int64) error {
	return nil
}
func (f *fakeLifecycleAutomationClient) FindPortCandidates(ctx context.Context, hostID int64, keywords string) ([]int64, error) {
	return nil, nil
}
func (f *fakeLifecycleAutomationClient) GetPanelURL(ctx context.Context, hostName string, panelPassword string) (string, error) {
	return "", nil
}
func (f *fakeLifecycleAutomationClient) ListAreas(ctx context.Context) ([]AutomationArea, error) {
	return nil, nil
}
func (f *fakeLifecycleAutomationClient) ListImages(ctx context.Context, lineID int64) ([]AutomationImage, error) {
	return nil, nil
}
func (f *fakeLifecycleAutomationClient) ListLines(ctx context.Context) ([]AutomationLine, error) {
	return nil, nil
}
func (f *fakeLifecycleAutomationClient) ListProducts(ctx context.Context, lineID int64) ([]AutomationProduct, error) {
	return nil, nil
}
func (f *fakeLifecycleAutomationClient) GetMonitor(ctx context.Context, hostID int64) (AutomationMonitor, error) {
	return AutomationMonitor{}, nil
}
func (f *fakeLifecycleAutomationClient) GetVNCURL(ctx context.Context, hostID int64) (string, error) {
	return "", nil
}

func TestCreateEmergencyRenewOrder_AutoApproveAndZeroAmount(t *testing.T) {
	now := time.Now()
	expire := now.Add(48 * time.Hour)
	inst := domain.VPSInstance{
		ID:                   1,
		UserID:               9,
		AutomationInstanceID: "1001",
		ExpireAt:             &expire,
		AdminStatus:          domain.VPSAdminStatusNormal,
		Status:               domain.VPSStatusRunning,
	}
	settings := &fakeLifecycleSettingsRepo{
		values: map[string]string{
			"emergency_renew_enabled":        "true",
			"emergency_renew_window_days":    "7",
			"emergency_renew_days":           "1",
			"emergency_renew_interval_hours": "24",
		},
	}
	vpsRepo := &fakeLifecycleVPSRepo{inst: inst}
	orders := &fakeLifecycleOrderRepo{}
	items := &fakeLifecycleOrderItemRepo{}
	automation := &fakeLifecycleAutomationClient{}
	svc := NewOrderService(orders, items, nil, nil, nil, nil, vpsRepo, nil, nil, nil, automation, nil, nil, nil, nil, settings, nil, nil, nil, nil, nil)

	order, err := svc.CreateEmergencyRenewOrder(context.Background(), inst.UserID, inst.ID)
	if err != nil {
		t.Fatalf("create emergency renew: %v", err)
	}
	if order.TotalAmount != 0 {
		t.Fatalf("expected zero amount, got %v", order.TotalAmount)
	}
	if !strings.HasPrefix(order.OrderNo, "EMR-") {
		t.Fatalf("unexpected order no: %s", order.OrderNo)
	}
	if len(items.items) != 1 || items.items[0].Action != "emergency_renew" {
		t.Fatalf("expected emergency renew item: %+v", items.items)
	}
}

func TestHandleEmergencyRenew_RespectsWindowAndUpdatesExpire(t *testing.T) {
	now := time.Now()
	expire := now.Add(24 * time.Hour)
	inst := domain.VPSInstance{
		ID:                   2,
		UserID:               9,
		AutomationInstanceID: "1002",
		ExpireAt:             &expire,
		AdminStatus:          domain.VPSAdminStatusNormal,
		Status:               domain.VPSStatusRunning,
	}
	settings := &fakeLifecycleSettingsRepo{
		values: map[string]string{
			"emergency_renew_enabled":        "true",
			"emergency_renew_window_days":    "7",
			"emergency_renew_days":           "2",
			"emergency_renew_interval_hours": "24",
		},
	}
	vpsRepo := &fakeLifecycleVPSRepo{inst: inst}
	automation := &fakeLifecycleAutomationClient{}
	svc := NewOrderService(&fakeLifecycleOrderRepo{}, &fakeLifecycleOrderItemRepo{}, nil, nil, nil, nil, vpsRepo, nil, nil, nil, automation, nil, nil, nil, nil, settings, nil, nil, nil, nil, nil)

	item := domain.OrderItem{
		OrderID:  10,
		ID:       11,
		Action:   "emergency_renew",
		SpecJSON: `{"vps_id":2,"renew_days":2}`,
	}
	if err := svc.handleEmergencyRenew(context.Background(), item); err != nil {
		t.Fatalf("handle emergency renew: %v", err)
	}
	if len(automation.RenewCalls) != 1 {
		t.Fatalf("expected renew call, got %d", len(automation.RenewCalls))
	}
	if len(vpsRepo.updates) != 1 {
		t.Fatalf("expected expire update")
	}
}

func TestCreateEmergencyRenewOrder_ForbiddenOutsideWindow(t *testing.T) {
	now := time.Now()
	expire := now.Add(30 * 24 * time.Hour)
	inst := domain.VPSInstance{
		ID:                   3,
		UserID:               9,
		AutomationInstanceID: "1003",
		ExpireAt:             &expire,
		AdminStatus:          domain.VPSAdminStatusNormal,
		Status:               domain.VPSStatusRunning,
	}
	settings := &fakeLifecycleSettingsRepo{
		values: map[string]string{
			"emergency_renew_enabled":        "true",
			"emergency_renew_window_days":    "7",
			"emergency_renew_days":           "1",
			"emergency_renew_interval_hours": "24",
		},
	}
	vpsRepo := &fakeLifecycleVPSRepo{inst: inst}
	svc := NewOrderService(&fakeLifecycleOrderRepo{}, &fakeLifecycleOrderItemRepo{}, nil, nil, nil, nil, vpsRepo, nil, nil, nil, &fakeLifecycleAutomationClient{}, nil, nil, nil, nil, settings, nil, nil, nil, nil, nil)

	if _, err := svc.CreateEmergencyRenewOrder(context.Background(), inst.UserID, inst.ID); err != ErrForbidden {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestCreateEmergencyRenewOrder_RequiresRealNameWhenRenewBlocked(t *testing.T) {
	now := time.Now()
	expire := now.Add(24 * time.Hour)
	inst := domain.VPSInstance{
		ID:                   4,
		UserID:               9,
		AutomationInstanceID: "1004",
		ExpireAt:             &expire,
		AdminStatus:          domain.VPSAdminStatusNormal,
		Status:               domain.VPSStatusRunning,
	}
	settings := &fakeLifecycleSettingsRepo{
		values: map[string]string{
			"emergency_renew_enabled":        "true",
			"emergency_renew_window_days":    "7",
			"emergency_renew_days":           "1",
			"emergency_renew_interval_hours": "24",
			"realname_enabled":               "true",
			"realname_provider":              "fake",
			"realname_block_actions":         `["renew_vps"]`,
		},
	}
	realnameSvc := apprealname.NewService(&fakeLifecycleRealNameRepo{}, nil, settings)
	vpsRepo := &fakeLifecycleVPSRepo{inst: inst}
	svc := NewOrderService(&fakeLifecycleOrderRepo{}, &fakeLifecycleOrderItemRepo{}, nil, nil, nil, nil, vpsRepo, nil, nil, nil, &fakeLifecycleAutomationClient{}, nil, nil, nil, nil, settings, nil, nil, nil, nil, realnameSvc)

	if _, err := svc.CreateEmergencyRenewOrder(context.Background(), inst.UserID, inst.ID); err != ErrRealNameRequired {
		t.Fatalf("expected real name required, got %v", err)
	}
}

func TestAutoDeleteExpired(t *testing.T) {
	now := time.Now()
	expired := now.Add(-48 * time.Hour)
	active := now.Add(24 * time.Hour)
	vpsRepo := &fakeLifecycleVPSRepo{
		expiring: []domain.VPSInstance{
			{ID: 1, UserID: 1, AutomationInstanceID: "1001", ExpireAt: &expired, Name: "expired"},
			{ID: 2, UserID: 1, AutomationInstanceID: "1002", ExpireAt: &active, Name: "active"},
		},
	}
	settings := &fakeLifecycleSettingsRepo{
		values: map[string]string{
			"auto_delete_enabled": "true",
			"auto_delete_days":    "0",
		},
	}
	automation := &fakeLifecycleAutomationClient{}
	svc := appvps.NewService(vpsRepo, automation, settings)
	if err := svc.AutoDeleteExpired(context.Background()); err != nil {
		t.Fatalf("auto delete: %v", err)
	}
	if len(automation.DeleteCalls) != 1 || automation.DeleteCalls[0] != 1001 {
		t.Fatalf("expected delete host 1001, got %+v", automation.DeleteCalls)
	}
	if len(vpsRepo.deletes) != 1 || vpsRepo.deletes[0] != 1 {
		t.Fatalf("expected delete instance 1, got %+v", vpsRepo.deletes)
	}
}

func TestAutoLockExpired(t *testing.T) {
	now := time.Now()
	expired := now.Add(-2 * time.Hour)
	active := now.Add(24 * time.Hour)
	vpsRepo := &fakeLifecycleVPSRepo{
		expiring: []domain.VPSInstance{
			{ID: 1, UserID: 1, GoodsTypeID: 1, AutomationInstanceID: "1001", ExpireAt: &expired, Status: domain.VPSStatusRunning, AdminStatus: domain.VPSAdminStatusNormal, Name: "expired-running"},
			{ID: 2, UserID: 1, GoodsTypeID: 1, AutomationInstanceID: "1002", ExpireAt: &expired, Status: domain.VPSStatusLocked, AdminStatus: domain.VPSAdminStatusLocked, Name: "expired-locked"},
			{ID: 3, UserID: 1, GoodsTypeID: 1, AutomationInstanceID: "1003", ExpireAt: &active, Status: domain.VPSStatusRunning, AdminStatus: domain.VPSAdminStatusNormal, Name: "active"},
		},
	}
	automation := &fakeLifecycleAutomationClient{}
	svc := appvps.NewService(vpsRepo, automation, nil)
	if err := svc.AutoLockExpired(context.Background()); err != nil {
		t.Fatalf("auto lock: %v", err)
	}
	if len(automation.LockCalls) != 1 || automation.LockCalls[0] != 1001 {
		t.Fatalf("expected lock host 1001, got %+v", automation.LockCalls)
	}
	if len(vpsRepo.statusUpdates) != 1 || vpsRepo.statusUpdates[0].Status != domain.VPSStatusExpiredLocked {
		t.Fatalf("expected expired_locked update, got %+v", vpsRepo.statusUpdates)
	}
	if len(vpsRepo.adminUpdates) != 1 || vpsRepo.adminUpdates[0].Status != domain.VPSAdminStatusLocked {
		t.Fatalf("expected admin locked update, got %+v", vpsRepo.adminUpdates)
	}
}
