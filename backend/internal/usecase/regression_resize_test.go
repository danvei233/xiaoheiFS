package usecase_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/pkg/money"
	"xiaoheiplay/internal/usecase"
)

type fakeResizeCatalogRepo struct {
	packages map[int64]domain.Package
	plans    map[int64]domain.PlanGroup
	regions  map[int64]domain.Region
}

func (f *fakeResizeCatalogRepo) ListRegions(ctx context.Context) ([]domain.Region, error) {
	return nil, nil
}
func (f *fakeResizeCatalogRepo) ListPlanGroups(ctx context.Context) ([]domain.PlanGroup, error) {
	return nil, nil
}
func (f *fakeResizeCatalogRepo) ListPackages(ctx context.Context) ([]domain.Package, error) {
	return nil, nil
}
func (f *fakeResizeCatalogRepo) GetPackage(ctx context.Context, id int64) (domain.Package, error) {
	if pkg, ok := f.packages[id]; ok {
		return pkg, nil
	}
	return domain.Package{}, usecase.ErrNotFound
}
func (f *fakeResizeCatalogRepo) GetPlanGroup(ctx context.Context, id int64) (domain.PlanGroup, error) {
	if plan, ok := f.plans[id]; ok {
		return plan, nil
	}
	return domain.PlanGroup{}, usecase.ErrNotFound
}
func (f *fakeResizeCatalogRepo) GetRegion(ctx context.Context, id int64) (domain.Region, error) {
	if region, ok := f.regions[id]; ok {
		return region, nil
	}
	return domain.Region{}, usecase.ErrNotFound
}
func (f *fakeResizeCatalogRepo) CreateRegion(ctx context.Context, region *domain.Region) error {
	return nil
}
func (f *fakeResizeCatalogRepo) UpdateRegion(ctx context.Context, region domain.Region) error {
	return nil
}
func (f *fakeResizeCatalogRepo) DeleteRegion(ctx context.Context, id int64) error {
	return nil
}
func (f *fakeResizeCatalogRepo) CreatePlanGroup(ctx context.Context, plan *domain.PlanGroup) error {
	return nil
}
func (f *fakeResizeCatalogRepo) UpdatePlanGroup(ctx context.Context, plan domain.PlanGroup) error {
	return nil
}
func (f *fakeResizeCatalogRepo) DeletePlanGroup(ctx context.Context, id int64) error {
	return nil
}
func (f *fakeResizeCatalogRepo) CreatePackage(ctx context.Context, pkg *domain.Package) error {
	return nil
}
func (f *fakeResizeCatalogRepo) UpdatePackage(ctx context.Context, pkg domain.Package) error {
	return nil
}

func mustJSON(t *testing.T, payload any) string {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return string(raw)
}
func (f *fakeResizeCatalogRepo) DeletePackage(ctx context.Context, id int64) error {
	return nil
}

type fakeResizeSettingsRepo struct {
	values map[string]string
}

func (f *fakeResizeSettingsRepo) GetSetting(ctx context.Context, key string) (domain.Setting, error) {
	if v, ok := f.values[key]; ok {
		return domain.Setting{Key: key, ValueJSON: v}, nil
	}
	return domain.Setting{}, usecase.ErrNotFound
}
func (f *fakeResizeSettingsRepo) UpsertSetting(ctx context.Context, setting domain.Setting) error {
	if f.values == nil {
		f.values = map[string]string{}
	}
	f.values[setting.Key] = setting.ValueJSON
	return nil
}
func (f *fakeResizeSettingsRepo) ListSettings(ctx context.Context) ([]domain.Setting, error) {
	return nil, nil
}
func (f *fakeResizeSettingsRepo) ListEmailTemplates(ctx context.Context) ([]domain.EmailTemplate, error) {
	return nil, nil
}
func (f *fakeResizeSettingsRepo) GetEmailTemplate(ctx context.Context, id int64) (domain.EmailTemplate, error) {
	return domain.EmailTemplate{}, usecase.ErrNotFound
}
func (f *fakeResizeSettingsRepo) UpsertEmailTemplate(ctx context.Context, tmpl *domain.EmailTemplate) error {
	return nil
}
func (f *fakeResizeSettingsRepo) DeleteEmailTemplate(ctx context.Context, id int64) error {
	return nil
}

type fakeResizeVPSRepo struct {
	inst domain.VPSInstance
}

func (f *fakeResizeVPSRepo) CreateInstance(ctx context.Context, inst *domain.VPSInstance) error {
	return nil
}
func (f *fakeResizeVPSRepo) GetInstance(ctx context.Context, id int64) (domain.VPSInstance, error) {
	if f.inst.ID == id {
		return f.inst, nil
	}
	return domain.VPSInstance{}, usecase.ErrNotFound
}
func (f *fakeResizeVPSRepo) GetInstanceByOrderItem(ctx context.Context, orderItemID int64) (domain.VPSInstance, error) {
	if f.inst.OrderItemID == orderItemID {
		return f.inst, nil
	}
	return domain.VPSInstance{}, usecase.ErrNotFound
}
func (f *fakeResizeVPSRepo) ListInstancesByUser(ctx context.Context, userID int64) ([]domain.VPSInstance, error) {
	return nil, nil
}
func (f *fakeResizeVPSRepo) ListInstances(ctx context.Context, limit, offset int) ([]domain.VPSInstance, int, error) {
	return nil, 0, nil
}
func (f *fakeResizeVPSRepo) ListInstancesExpiring(ctx context.Context, before time.Time) ([]domain.VPSInstance, error) {
	return nil, nil
}
func (f *fakeResizeVPSRepo) DeleteInstance(ctx context.Context, id int64) error { return nil }
func (f *fakeResizeVPSRepo) UpdateInstanceStatus(ctx context.Context, id int64, status domain.VPSStatus, automationState int) error {
	return nil
}
func (f *fakeResizeVPSRepo) UpdateInstanceAdminStatus(ctx context.Context, id int64, status domain.VPSAdminStatus) error {
	return nil
}
func (f *fakeResizeVPSRepo) UpdateInstanceExpireAt(ctx context.Context, id int64, expireAt time.Time) error {
	return nil
}
func (f *fakeResizeVPSRepo) UpdateInstancePanelCache(ctx context.Context, id int64, panelURL string) error {
	return nil
}
func (f *fakeResizeVPSRepo) UpdateInstanceSpec(ctx context.Context, id int64, specJSON string) error {
	return nil
}
func (f *fakeResizeVPSRepo) UpdateInstanceAccessInfo(ctx context.Context, id int64, accessJSON string) error {
	return nil
}
func (f *fakeResizeVPSRepo) UpdateInstanceEmergencyRenewAt(ctx context.Context, id int64, at time.Time) error {
	return nil
}
func (f *fakeResizeVPSRepo) UpdateInstanceLocal(ctx context.Context, inst domain.VPSInstance) error {
	return nil
}

type fakeResizeOrderRepo struct {
	nextID int64
	orders map[int64]domain.Order
}

func (f *fakeResizeOrderRepo) CreateOrder(ctx context.Context, order *domain.Order) error {
	if f.orders == nil {
		f.orders = map[int64]domain.Order{}
	}
	f.nextID++
	order.ID = f.nextID
	f.orders[order.ID] = *order
	return nil
}
func (f *fakeResizeOrderRepo) GetOrder(ctx context.Context, id int64) (domain.Order, error) {
	if order, ok := f.orders[id]; ok {
		return order, nil
	}
	return domain.Order{}, usecase.ErrNotFound
}
func (f *fakeResizeOrderRepo) GetOrderByNo(ctx context.Context, orderNo string) (domain.Order, error) {
	return domain.Order{}, usecase.ErrNotFound
}
func (f *fakeResizeOrderRepo) GetOrderByIdempotencyKey(ctx context.Context, userID int64, key string) (domain.Order, error) {
	return domain.Order{}, usecase.ErrNotFound
}
func (f *fakeResizeOrderRepo) UpdateOrderStatus(ctx context.Context, id int64, status domain.OrderStatus) error {
	order, ok := f.orders[id]
	if !ok {
		return usecase.ErrNotFound
	}
	order.Status = status
	f.orders[id] = order
	return nil
}
func (f *fakeResizeOrderRepo) UpdateOrderMeta(ctx context.Context, order domain.Order) error {
	if f.orders == nil {
		f.orders = map[int64]domain.Order{}
	}
	f.orders[order.ID] = order
	return nil
}
func (f *fakeResizeOrderRepo) ListOrders(ctx context.Context, filter usecase.OrderFilter, limit, offset int) ([]domain.Order, int, error) {
	return nil, 0, nil
}
func (f *fakeResizeOrderRepo) DeleteOrder(ctx context.Context, id int64) error { return nil }

type fakeResizeOrderItemRepo struct {
	items         []domain.OrderItem
	pendingResize bool
	pendingRefund bool
	nextID        int64
}

func (f *fakeResizeOrderItemRepo) CreateOrderItems(ctx context.Context, items []domain.OrderItem) error {
	for i := range items {
		if items[i].ID == 0 {
			f.nextID++
			items[i].ID = f.nextID
		}
		f.items = append(f.items, items[i])
	}
	return nil
}
func (f *fakeResizeOrderItemRepo) ListOrderItems(ctx context.Context, orderID int64) ([]domain.OrderItem, error) {
	var out []domain.OrderItem
	for _, item := range f.items {
		if item.OrderID == orderID {
			out = append(out, item)
		}
	}
	return out, nil
}
func (f *fakeResizeOrderItemRepo) GetOrderItem(ctx context.Context, id int64) (domain.OrderItem, error) {
	for _, item := range f.items {
		if item.ID == id {
			return item, nil
		}
	}
	return domain.OrderItem{}, usecase.ErrNotFound
}
func (f *fakeResizeOrderItemRepo) UpdateOrderItemStatus(ctx context.Context, id int64, status domain.OrderItemStatus) error {
	for i := range f.items {
		if f.items[i].ID == id {
			f.items[i].Status = status
			break
		}
	}
	return nil
}
func (f *fakeResizeOrderItemRepo) UpdateOrderItemAutomation(ctx context.Context, id int64, automationID string) error {
	return nil
}
func (f *fakeResizeOrderItemRepo) HasPendingRenewOrder(ctx context.Context, userID, vpsID int64) (bool, error) {
	return false, nil
}
func (f *fakeResizeOrderItemRepo) HasPendingResizeOrder(ctx context.Context, userID, vpsID int64) (bool, error) {
	return f.pendingResize, nil
}
func (f *fakeResizeOrderItemRepo) HasPendingRefundOrder(ctx context.Context, userID, vpsID int64) (bool, error) {
	return f.pendingRefund, nil
}

type fakeResizeTaskRepo struct {
	nextID  int64
	pending bool
	tasks   []domain.ResizeTask
}

func (f *fakeResizeTaskRepo) CreateResizeTask(ctx context.Context, task *domain.ResizeTask) error {
	f.nextID++
	task.ID = f.nextID
	f.tasks = append(f.tasks, *task)
	return nil
}

func (f *fakeResizeTaskRepo) GetResizeTask(ctx context.Context, id int64) (domain.ResizeTask, error) {
	for _, task := range f.tasks {
		if task.ID == id {
			return task, nil
		}
	}
	return domain.ResizeTask{}, usecase.ErrNotFound
}

func (f *fakeResizeTaskRepo) UpdateResizeTask(ctx context.Context, task domain.ResizeTask) error {
	for i := range f.tasks {
		if f.tasks[i].ID == task.ID {
			f.tasks[i] = task
			return nil
		}
	}
	f.tasks = append(f.tasks, task)
	return nil
}

func (f *fakeResizeTaskRepo) ListDueResizeTasks(ctx context.Context, limit int) ([]domain.ResizeTask, error) {
	return nil, nil
}

func (f *fakeResizeTaskRepo) HasPendingResizeTask(ctx context.Context, vpsID int64) (bool, error) {
	return f.pending, nil
}

func newResizeOrderService(inst domain.VPSInstance, catalog *fakeResizeCatalogRepo, settings *fakeResizeSettingsRepo, orders *fakeResizeOrderRepo, items *fakeResizeOrderItemRepo, tasks *fakeResizeTaskRepo) *usecase.OrderService {
	vps := &fakeResizeVPSRepo{inst: inst}
	return usecase.NewOrderService(
		orders,
		items,
		nil,
		catalog,
		nil,
		nil,
		vps,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		settings,
		nil,
		nil,
		tasks,
		nil,
		nil,
	)
}

func TestResizeOrder_RejectsSamePackage(t *testing.T) {
	plan := domain.PlanGroup{ID: 10, RegionID: 1}
	pkg := domain.Package{ID: 1, PlanGroupID: plan.ID, ProductID: 11, Name: "A", Cores: 2, MemoryGB: 4, DiskGB: 40, BandwidthMB: 100, Monthly: 10000}
	catalog := &fakeResizeCatalogRepo{
		packages: map[int64]domain.Package{pkg.ID: pkg},
		plans:    map[int64]domain.PlanGroup{plan.ID: plan},
	}
	inst := domain.VPSInstance{ID: 1, UserID: 2, PackageID: pkg.ID, SpecJSON: "{}", CreatedAt: time.Now()}
	svc := newResizeOrderService(inst, catalog, &fakeResizeSettingsRepo{}, &fakeResizeOrderRepo{}, &fakeResizeOrderItemRepo{}, &fakeResizeTaskRepo{})

	_, _, err := svc.CreateResizeOrder(context.Background(), inst.UserID, inst.ID, nil, pkg.ID, false, nil)
	if err != usecase.ErrResizeSamePlan {
		t.Fatalf("expected same plan error, got %v", err)
	}
}

func TestResizeOrder_RejectsEquivalentPlanFingerprint(t *testing.T) {
	plan := domain.PlanGroup{ID: 10, RegionID: 1}
	current := domain.Package{ID: 1, PlanGroupID: plan.ID, ProductID: 11, Name: "A", Cores: 2, MemoryGB: 4, DiskGB: 40, BandwidthMB: 100, Monthly: 10000}
	other := domain.Package{ID: 2, PlanGroupID: plan.ID, ProductID: 22, Name: "B", Cores: 2, MemoryGB: 4, DiskGB: 40, BandwidthMB: 100, Monthly: 10000}
	catalog := &fakeResizeCatalogRepo{
		packages: map[int64]domain.Package{current.ID: current, other.ID: other},
		plans:    map[int64]domain.PlanGroup{plan.ID: plan},
	}
	inst := domain.VPSInstance{ID: 1, UserID: 2, PackageID: current.ID, SpecJSON: "{}", CreatedAt: time.Now()}
	svc := newResizeOrderService(inst, catalog, &fakeResizeSettingsRepo{}, &fakeResizeOrderRepo{}, &fakeResizeOrderItemRepo{}, &fakeResizeTaskRepo{})

	_, _, err := svc.CreateResizeOrder(context.Background(), inst.UserID, inst.ID, nil, other.ID, false, nil)
	if err != usecase.ErrResizeSamePlan {
		t.Fatalf("expected same plan error for equivalent plan, got %v", err)
	}
}

func TestResizeOrder_RejectsWhenDisabled(t *testing.T) {
	plan := domain.PlanGroup{ID: 10, RegionID: 1}
	current := domain.Package{ID: 1, PlanGroupID: plan.ID, ProductID: 11, Name: "A", Cores: 2, MemoryGB: 4, DiskGB: 40, BandwidthMB: 100, Monthly: 10000}
	target := domain.Package{ID: 2, PlanGroupID: plan.ID, ProductID: 22, Name: "B", Cores: 4, MemoryGB: 8, DiskGB: 80, BandwidthMB: 200, Monthly: 20000}
	catalog := &fakeResizeCatalogRepo{
		packages: map[int64]domain.Package{current.ID: current, target.ID: target},
		plans:    map[int64]domain.PlanGroup{plan.ID: plan},
	}
	settings := &fakeResizeSettingsRepo{values: map[string]string{"resize_enabled": "false"}}
	inst := domain.VPSInstance{ID: 1, UserID: 2, PackageID: current.ID, SpecJSON: "{}", CreatedAt: time.Now()}
	svc := newResizeOrderService(inst, catalog, settings, &fakeResizeOrderRepo{}, &fakeResizeOrderItemRepo{}, &fakeResizeTaskRepo{})

	_, _, err := svc.CreateResizeOrder(context.Background(), inst.UserID, inst.ID, nil, target.ID, false, nil)
	if err != usecase.ErrResizeDisabled {
		t.Fatalf("expected resize disabled error, got %v", err)
	}
}

func TestResizeOrder_ConflictWhenPendingResizeOrder(t *testing.T) {
	plan := domain.PlanGroup{ID: 10, RegionID: 1}
	current := domain.Package{ID: 1, PlanGroupID: plan.ID, ProductID: 11, Name: "A", Cores: 2, MemoryGB: 4, DiskGB: 40, BandwidthMB: 100, Monthly: 10000}
	target := domain.Package{ID: 2, PlanGroupID: plan.ID, ProductID: 22, Name: "B", Cores: 4, MemoryGB: 8, DiskGB: 80, BandwidthMB: 200, Monthly: 20000}
	catalog := &fakeResizeCatalogRepo{
		packages: map[int64]domain.Package{current.ID: current, target.ID: target},
		plans:    map[int64]domain.PlanGroup{plan.ID: plan},
	}
	inst := domain.VPSInstance{ID: 1, UserID: 2, PackageID: current.ID, SpecJSON: "{}", CreatedAt: time.Now()}
	items := &fakeResizeOrderItemRepo{pendingResize: true}
	svc := newResizeOrderService(inst, catalog, &fakeResizeSettingsRepo{}, &fakeResizeOrderRepo{}, items, &fakeResizeTaskRepo{})

	_, _, err := svc.CreateResizeOrder(context.Background(), inst.UserID, inst.ID, nil, target.ID, false, nil)
	if err != usecase.ErrResizeInProgress {
		t.Fatalf("expected resize in progress with pending resize order, got %v", err)
	}
}

func TestResizeOrder_ConflictWhenPendingResizeTask(t *testing.T) {
	plan := domain.PlanGroup{ID: 10, RegionID: 1}
	current := domain.Package{ID: 1, PlanGroupID: plan.ID, ProductID: 11, Name: "A", Cores: 2, MemoryGB: 4, DiskGB: 40, BandwidthMB: 100, Monthly: 10000}
	target := domain.Package{ID: 2, PlanGroupID: plan.ID, ProductID: 22, Name: "B", Cores: 4, MemoryGB: 8, DiskGB: 80, BandwidthMB: 200, Monthly: 20000}
	catalog := &fakeResizeCatalogRepo{
		packages: map[int64]domain.Package{current.ID: current, target.ID: target},
		plans:    map[int64]domain.PlanGroup{plan.ID: plan},
	}
	inst := domain.VPSInstance{ID: 1, UserID: 2, PackageID: current.ID, SpecJSON: "{}", CreatedAt: time.Now()}
	items := &fakeResizeOrderItemRepo{}
	tasks := &fakeResizeTaskRepo{pending: true}
	svc := newResizeOrderService(inst, catalog, &fakeResizeSettingsRepo{}, &fakeResizeOrderRepo{}, items, tasks)

	_, _, err := svc.CreateResizeOrder(context.Background(), inst.UserID, inst.ID, nil, target.ID, false, nil)
	if err != usecase.ErrResizeInProgress {
		t.Fatalf("expected resize in progress with pending resize task, got %v", err)
	}
}

func TestResizeProrationCharge(t *testing.T) {
	plan := domain.PlanGroup{ID: 10, RegionID: 1}
	current := domain.Package{ID: 1, PlanGroupID: plan.ID, ProductID: 11, Name: "A", Cores: 2, MemoryGB: 4, DiskGB: 40, BandwidthMB: 100, Monthly: 10000}
	target := domain.Package{ID: 2, PlanGroupID: plan.ID, ProductID: 22, Name: "B", Cores: 4, MemoryGB: 8, DiskGB: 80, BandwidthMB: 200, Monthly: 20000}
	catalog := &fakeResizeCatalogRepo{
		packages: map[int64]domain.Package{current.ID: current, target.ID: target},
		plans:    map[int64]domain.PlanGroup{plan.ID: plan},
	}

	tests := []struct {
		name   string
		expire *time.Time
		months int
	}{
		{
			name:   "remaining one day",
			expire: ptrTime(time.Now().Add(24 * time.Hour)),
			months: 1,
		},
		{
			name:   "remaining zero",
			expire: ptrTime(time.Now().Add(-1 * time.Hour)),
			months: 1,
		},
		{
			name:   "just started",
			expire: ptrTime(time.Now().AddDate(0, 1, 0)),
			months: 1,
		},
	}

	for _, tc := range tests {
		now := time.Now()
		periodEnd := tc.expire
		if periodEnd == nil {
			periodEnd = ptrTime(now.AddDate(0, tc.months, 0))
		}
		periodEndValue := *periodEnd
		periodStart := periodEndValue.AddDate(0, -tc.months, 0)
		inst := domain.VPSInstance{
			ID:        1,
			UserID:    2,
			PackageID: current.ID,
			SpecJSON: mustJSON(t, map[string]any{
				"duration_months":      tc.months,
				"current_period_start": periodStart.UTC().Format(time.RFC3339),
				"current_period_end":   periodEndValue.UTC().Format(time.RFC3339),
			}),
			CreatedAt: now.Add(-365 * 24 * time.Hour),
			ExpireAt:  tc.expire,
		}
		expected := int64(0)
		if periodEnd != nil {
			remaining := periodEndValue.Sub(now)
			total := periodEndValue.Sub(periodStart)
			if remaining > 0 && total > 0 {
				expected = money.ProrateCents(10000, remaining.Nanoseconds(), total.Nanoseconds())
			}
		}
		svc := newResizeOrderService(inst, catalog, &fakeResizeSettingsRepo{}, &fakeResizeOrderRepo{}, &fakeResizeOrderItemRepo{}, &fakeResizeTaskRepo{})
		_, quote, err := svc.CreateResizeOrder(context.Background(), inst.UserID, inst.ID, nil, target.ID, false, nil)
		if err != nil {
			t.Fatalf("%s: create resize order: %v", tc.name, err)
		}
		if quote.ChargeAmount != expected {
			t.Fatalf("%s: expected charge %d, got %d", tc.name, expected, quote.ChargeAmount)
		}
	}
}

func TestResizeQuote_NoOrderCreated(t *testing.T) {
	plan := domain.PlanGroup{ID: 10, RegionID: 1}
	current := domain.Package{ID: 1, PlanGroupID: plan.ID, ProductID: 11, Name: "A", Cores: 2, MemoryGB: 4, DiskGB: 40, BandwidthMB: 100, Monthly: 1000}
	target := domain.Package{ID: 2, PlanGroupID: plan.ID, ProductID: 22, Name: "B", Cores: 4, MemoryGB: 8, DiskGB: 80, BandwidthMB: 200, Monthly: 1500}
	catalog := &fakeResizeCatalogRepo{
		packages: map[int64]domain.Package{current.ID: current, target.ID: target},
		plans:    map[int64]domain.PlanGroup{plan.ID: plan},
	}
	now := time.Now()
	inst := domain.VPSInstance{
		ID:        1,
		UserID:    2,
		PackageID: current.ID,
		SpecJSON: mustJSON(t, map[string]any{
			"duration_months":      12,
			"current_period_start": now.AddDate(0, -1, 0).UTC().Format(time.RFC3339),
			"current_period_end":   now.AddDate(0, 1, 0).UTC().Format(time.RFC3339),
		}),
		CreatedAt: now.AddDate(-1, 0, 0),
		ExpireAt:  ptrTime(now.AddDate(0, 3, 0)),
	}
	orders := &fakeResizeOrderRepo{}
	svc := newResizeOrderService(inst, catalog, &fakeResizeSettingsRepo{}, orders, &fakeResizeOrderItemRepo{}, &fakeResizeTaskRepo{})

	quote, _, err := svc.QuoteResizeOrder(context.Background(), inst.UserID, inst.ID, nil, target.ID, false)
	if err != nil {
		t.Fatalf("quote resize: %v", err)
	}
	if quote.ChargeAmount == 0 {
		t.Fatalf("expected non-zero quote")
	}
	if len(orders.orders) != 0 {
		t.Fatalf("expected no order created, got %d", len(orders.orders))
	}
}

func TestResizeApproveCreatesTask(t *testing.T) {
	plan := domain.PlanGroup{ID: 10, RegionID: 1}
	current := domain.Package{ID: 1, PlanGroupID: plan.ID, ProductID: 11, Name: "A", Cores: 2, MemoryGB: 4, DiskGB: 40, BandwidthMB: 100, Monthly: 10000}
	target := domain.Package{ID: 2, PlanGroupID: plan.ID, ProductID: 22, Name: "B", Cores: 4, MemoryGB: 8, DiskGB: 80, BandwidthMB: 200, Monthly: 20000}
	catalog := &fakeResizeCatalogRepo{
		packages: map[int64]domain.Package{current.ID: current, target.ID: target},
		plans:    map[int64]domain.PlanGroup{plan.ID: plan},
	}
	inst := domain.VPSInstance{ID: 1, UserID: 2, PackageID: current.ID, SpecJSON: "{}", CreatedAt: time.Now()}
	orders := &fakeResizeOrderRepo{}
	items := &fakeResizeOrderItemRepo{}
	tasks := &fakeResizeTaskRepo{}
	svc := newResizeOrderService(inst, catalog, &fakeResizeSettingsRepo{}, orders, items, tasks)

	order := domain.Order{UserID: inst.UserID, OrderNo: "UPG-1", Status: domain.OrderStatusPendingReview, TotalAmount: 0, Currency: "CNY"}
	if err := orders.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	specJSON, err := json.Marshal(map[string]any{
		"vps_id":       inst.ID,
		"scheduled_at": time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("spec json: %v", err)
	}
	item := domain.OrderItem{
		OrderID:  order.ID,
		Qty:      1,
		Amount:   0,
		Status:   domain.OrderItemStatusPendingReview,
		Action:   "resize",
		SpecJSON: string(specJSON),
	}
	if err := items.CreateOrderItems(context.Background(), []domain.OrderItem{item}); err != nil {
		t.Fatalf("create item: %v", err)
	}

	if err := svc.ApproveOrder(context.Background(), 1, order.ID); err != nil {
		t.Fatalf("approve order: %v", err)
	}
	updated, err := orders.GetOrder(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if updated.Status != domain.OrderStatusApproved {
		t.Fatalf("expected order approved, got %v", updated.Status)
	}
	if len(tasks.tasks) != 1 {
		t.Fatalf("expected resize task created, got %d", len(tasks.tasks))
	}
	if tasks.tasks[0].OrderID != order.ID || tasks.tasks[0].VPSID != inst.ID {
		t.Fatalf("unexpected resize task: %+v", tasks.tasks[0])
	}
	if items.items[0].Status != domain.OrderItemStatusApproved {
		t.Fatalf("expected order item approved, got %v", items.items[0].Status)
	}
}

func TestRefundCurveUsesElapsedRatio(t *testing.T) {
	settings := &fakeResizeSettingsRepo{
		values: map[string]string{
			"refund_curve_json": `[{"hours":0,"ratio":1},{"hours":50,"ratio":0.5},{"hours":100,"ratio":0}]`,
		},
	}
	orders := &fakeWalletOrderRepo{}
	items := &fakeResizeOrderItemRepo{}
	vps := &fakeResizeVPSRepo{}
	svc := usecase.NewWalletOrderService(orders, nil, settings, vps, items, nil, nil)

	now := time.Now()
	periods := []time.Duration{30 * 24 * time.Hour, 365 * 24 * time.Hour}
	for _, period := range periods {
		item := domain.OrderItem{ID: 10, Amount: 10000, Status: domain.OrderItemStatusActive, Action: "create", SpecJSON: "{}"}
		inst := domain.VPSInstance{
			ID:          1,
			UserID:      2,
			OrderItemID: item.ID,
			CreatedAt:   now.Add(-period / 2),
			ExpireAt:    ptrTime(now.Add(period / 2)),
		}
		vps.inst = inst
		items.items = []domain.OrderItem{item}
		refund, _, err := svc.RequestRefund(context.Background(), inst.UserID, inst.ID, "test")
		if err != nil {
			t.Fatalf("refund curve period %v: %v", period, err)
		}
		if refund.Amount != 5000 {
			t.Fatalf("refund curve period %v: expected 5000, got %d", period, refund.Amount)
		}
	}
}

func ptrTime(val time.Time) *time.Time {
	return &val
}
