package order

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	appports "xiaoheiplay/internal/app/ports"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func TestNormalizeOrderQty(t *testing.T) {
	if qty, err := normalizeOrderQty(0); err != nil || qty != 1 {
		t.Fatalf("expected default 1, got %d err=%v", qty, err)
	}
	if qty, err := normalizeOrderQty(-3); err != nil || qty != 1 {
		t.Fatalf("expected default 1 for negative, got %d err=%v", qty, err)
	}
	if _, err := normalizeOrderQty(maxOrderItemQty + 1); err != ErrInvalidInput {
		t.Fatalf("expected qty cap rejection, got %v", err)
	}
	if qty, err := normalizeOrderQty(maxOrderItemQty); err != nil || qty != maxOrderItemQty {
		t.Fatalf("expected max allowed qty, got %d err=%v", qty, err)
	}
}

func TestAddLineTotal(t *testing.T) {
	total, err := addLineTotal(100, 10, 5)
	if err != nil || total != 150 {
		t.Fatalf("expected 150, got %d err=%v", total, err)
	}
	// per-line product overflow
	if _, err := addLineTotal(0, math.MaxInt64, 2); err != ErrInvalidInput {
		t.Fatalf("expected product overflow rejection, got %v", err)
	}
	// running total wrap
	if _, err := addLineTotal(math.MaxInt64, 1, 1); err != ErrInvalidInput {
		t.Fatalf("expected total wrap rejection, got %v", err)
	}
	// qty cap enforced even when unitTotal is zero
	if _, err := addLineTotal(0, 0, maxOrderItemQty+1); err != ErrInvalidInput {
		t.Fatalf("expected qty cap rejection, got %v", err)
	}
	// zero-priced line is free and does not move the total
	total, err = addLineTotal(50, 0, 3)
	if err != nil || total != 50 {
		t.Fatalf("expected unchanged total 50, got %d err=%v", total, err)
	}
}

func TestRenewalNextDue(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	t.Run("prefers recorded period end over expire at", func(t *testing.T) {
		periodEnd := now.Add(10 * 24 * time.Hour)
		inst := domain.VPSInstance{
			ExpireAt: &periodEnd,
			SpecJSON: mustJSON(map[string]any{"current_period_end": periodEnd.UTC().Format(time.RFC3339)}),
		}
		next := renewalNextDue(inst, 30, now)
		want := periodEnd.Add(30 * 24 * time.Hour)
		if !next.Equal(want) {
			t.Fatalf("expected %v, got %v", want, next)
		}
	})

	t.Run("retry while original period still active reasserts advanced expiry", func(t *testing.T) {
		originalEnd := now.Add(5 * 24 * time.Hour)
		advanced := originalEnd.Add(30 * 24 * time.Hour)
		inst := domain.VPSInstance{
			ExpireAt: &advanced,
			SpecJSON: mustJSON(map[string]any{"current_period_end": originalEnd.UTC().Format(time.RFC3339)}),
		}
		next := renewalNextDue(inst, 30, now)
		if !next.Equal(advanced) {
			t.Fatalf("expected idempotent due %v, got %v", advanced, next)
		}
	})

	t.Run("retry after original period ended reasserts advanced expiry", func(t *testing.T) {
		originalEnd := now.Add(-2 * 24 * time.Hour)
		advanced := originalEnd.Add(30 * 24 * time.Hour) // = now + 28d
		inst := domain.VPSInstance{
			ExpireAt: &advanced,
			SpecJSON: mustJSON(map[string]any{"current_period_end": originalEnd.UTC().Format(time.RFC3339)}),
		}
		next := renewalNextDue(inst, 30, now)
		if !next.Equal(advanced) {
			t.Fatalf("expected idempotent due %v, got %v", advanced, next)
		}
	})

	t.Run("falls back to now when instance already expired", func(t *testing.T) {
		expired := now.Add(-5 * 24 * time.Hour)
		inst := domain.VPSInstance{ExpireAt: &expired, SpecJSON: "{}"}
		next := renewalNextDue(inst, 30, now)
		want := now.Add(30 * 24 * time.Hour)
		if !next.Equal(want) {
			t.Fatalf("expected %v, got %v", want, next)
		}
	})
}

// F11: automation payload redaction must cover token/secret/api_key keys, not
// only passwords.
func TestRedactAutomationPayload_RedactsTokensSecretsAndAPIKeys(t *testing.T) {
	payload := map[string]any{
		"access_token":  "tok-123",
		"client_secret": "sec-456",
		"api_key":       "key-789",
		"apiKey":        "key-abc",
		"host_id":       1001,
		"nested": map[string]any{
			"refresh_token": "ref-tok",
			"os_password":   "pw",
			"label":         "keep-me",
		},
	}
	serialized, err := json.Marshal(redactAutomationPayload(payload))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, secret := range []string{"tok-123", "sec-456", "key-789", "key-abc", "ref-tok", "pw"} {
		if strings.Contains(string(serialized), secret) {
			t.Fatalf("redacted payload still contains %q: %s", secret, serialized)
		}
	}
	if !strings.Contains(string(serialized), "keep-me") {
		t.Fatalf("expected non-sensitive value preserved: %s", serialized)
	}
	if !strings.Contains(string(serialized), "[REDACTED]") {
		t.Fatalf("expected redaction marker: %s", serialized)
	}
}

// fakeTierPricingResolver returns a fixed member monthly price.
type fakeTierPricingResolver struct{ monthly int64 }

func (f fakeTierPricingResolver) ResolvePackagePricing(ctx context.Context, userID, packageID int64) (domain.UserTierPriceCache, int64, error) {
	return domain.UserTierPriceCache{MonthlyPrice: f.monthly}, 0, nil
}

// F9: a resize must keep the member-tier monthly price instead of resetting it
// to the catalog price (which would overcharge later renewals).
func TestHandleResize_PreservesTierMonthlyPrice(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "tierresize", "tierresize@example.com", "pass")

	expire := time.Now().Add(30 * 24 * time.Hour)
	inst := domain.VPSInstance{
		UserID:               user.ID,
		OrderItemID:          1,
		AutomationInstanceID: "3001",
		Name:                 "vm-tier-resize",
		PackageID:            seed.Package.ID,
		PackageName:          seed.Package.Name,
		GoodsTypeID:          seed.Package.GoodsTypeID,
		MonthlyPrice:         5000,
		SpecJSON:             "{}",
		Status:               domain.VPSStatusRunning,
		ExpireAt:             &expire,
	}
	if err := repo.CreateInstance(context.Background(), &inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}

	automation := &testutil.FakeAutomationClient{}
	svc := &OrderService{catalog: repo, vps: repo, automation: &testutil.FakeAutomationResolver{Client: automation}}
	svc.SetUserTierPricingResolver(fakeTierPricingResolver{monthly: 4000})

	item := domain.OrderItem{
		OrderID: 1, ID: 1, Action: "resize",
		SpecJSON: mustJSON(map[string]any{
			"vps_id":            inst.ID,
			"target_package_id": seed.Package.ID,
			"target_cpu":        2,
			"target_mem_gb":     2,
			"target_disk_gb":    20,
			"target_bw_mbps":    5,
			"spec": map[string]any{
				"add_cores": 0, "add_mem_gb": 0, "add_disk_gb": 0, "add_bw_mbps": 0,
			},
		}),
	}
	if err := svc.handleResize(context.Background(), item); err != nil {
		t.Fatalf("handleResize: %v", err)
	}
	updated, err := repo.GetInstance(context.Background(), inst.ID)
	if err != nil {
		t.Fatalf("get instance: %v", err)
	}
	if updated.MonthlyPrice != 4000 {
		t.Fatalf("expected member monthly price 4000 after resize, got %d", updated.MonthlyPrice)
	}
}

// renewRetryVPSRepo simulates a renewal whose first attempt advances the local
// ExpireAt but fails the spec update, so a retry re-reads an already-advanced
// ExpireAt (the double-extend scenario).
type renewRetryVPSRepo struct {
	inst       domain.VPSInstance
	specCalls  int
	specFail   bool
	expireSets []time.Time
}

func (f *renewRetryVPSRepo) CreateInstance(ctx context.Context, inst *domain.VPSInstance) error {
	return nil
}
func (f *renewRetryVPSRepo) GetInstance(ctx context.Context, id int64) (domain.VPSInstance, error) {
	return f.inst, nil
}
func (f *renewRetryVPSRepo) GetInstanceByOrderItem(ctx context.Context, orderItemID int64) (domain.VPSInstance, error) {
	return domain.VPSInstance{}, ErrNotFound
}
func (f *renewRetryVPSRepo) ListInstancesByUserFiltered(ctx context.Context, userID int64, filter appports.VPSUserInstanceFilter, limit, offset int) ([]domain.VPSInstance, int, error) {
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
func (f *renewRetryVPSRepo) ListInstancesByUser(ctx context.Context, userID int64) ([]domain.VPSInstance, error) {
	return nil, nil
}
func (f *renewRetryVPSRepo) ListInstances(ctx context.Context, limit, offset int) ([]domain.VPSInstance, int, error) {
	return nil, 0, nil
}
func (f *renewRetryVPSRepo) ListInstancesFiltered(ctx context.Context, filter appports.VPSInstanceFilter, limit, offset int) ([]domain.VPSInstance, int, error) {
	return nil, 0, nil
}
func (f *renewRetryVPSRepo) ListInstancesExpiring(ctx context.Context, before time.Time) ([]domain.VPSInstance, error) {
	return nil, nil
}
func (f *renewRetryVPSRepo) DeleteInstance(ctx context.Context, id int64) error { return nil }
func (f *renewRetryVPSRepo) UpdateInstanceStatus(ctx context.Context, id int64, status domain.VPSStatus, automationState int) error {
	return nil
}
func (f *renewRetryVPSRepo) UpdateInstanceAdminStatus(ctx context.Context, id int64, status domain.VPSAdminStatus) error {
	return nil
}
func (f *renewRetryVPSRepo) UpdateInstanceExpireAt(ctx context.Context, id int64, expireAt time.Time) error {
	f.expireSets = append(f.expireSets, expireAt)
	f.inst.ExpireAt = &expireAt
	return nil
}
func (f *renewRetryVPSRepo) UpdateInstancePanelCache(ctx context.Context, id int64, panelURL string) error {
	return nil
}
func (f *renewRetryVPSRepo) UpdateInstanceSpec(ctx context.Context, id int64, specJSON string) error {
	f.specCalls++
	if f.specFail && f.specCalls == 1 {
		return errors.New("spec update failed")
	}
	f.inst.SpecJSON = specJSON
	return nil
}
func (f *renewRetryVPSRepo) UpdateInstanceAccessInfo(ctx context.Context, id int64, accessJSON string) error {
	return nil
}
func (f *renewRetryVPSRepo) UpdateInstanceEmergencyRenewAt(ctx context.Context, id int64, at time.Time) error {
	return nil
}
func (f *renewRetryVPSRepo) UpdateInstanceLocal(ctx context.Context, inst domain.VPSInstance) error {
	f.inst = inst
	return nil
}

// F6: retrying a renewal after a remote-success/local-failure must re-assert the
// same due date instead of extending the already-advanced ExpireAt again.
func TestHandleRenew_RetryReassertsSameDueDate(t *testing.T) {
	now := time.Now()
	expire := now.Add(10 * 24 * time.Hour)
	vpsRepo := &renewRetryVPSRepo{
		inst: domain.VPSInstance{
			ID: 7, UserID: 9, GoodsTypeID: 1, AutomationInstanceID: "1007",
			ExpireAt: &expire, AdminStatus: domain.VPSAdminStatusNormal, Status: domain.VPSStatusRunning,
			SpecJSON: mustJSON(map[string]any{"current_period_end": expire.UTC().Format(time.RFC3339)}),
		},
		specFail: true,
	}
	automation := &fakeLifecycleAutomationClient{}
	svc := NewOrderService(&fakeLifecycleOrderRepo{}, &fakeLifecycleOrderItemRepo{}, nil, nil, nil, nil, vpsRepo, nil, nil, nil, automation, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	item := domain.OrderItem{OrderID: 20, ID: 21, Action: "renew", SpecJSON: `{"vps_id":7,"renew_days":30}`}

	if err := svc.handleRenew(context.Background(), item); err == nil {
		t.Fatal("expected spec update failure on first attempt")
	}
	vpsRepo.specFail = false
	if err := svc.handleRenew(context.Background(), item); err != nil {
		t.Fatalf("retry handleRenew: %v", err)
	}
	if len(automation.RenewCalls) != 2 {
		t.Fatalf("expected 2 renew calls, got %d", len(automation.RenewCalls))
	}
	if !automation.RenewCalls[0].Next.Equal(automation.RenewCalls[1].Next) {
		t.Fatalf("expected retry to reassert due %v, got %v", automation.RenewCalls[0].Next, automation.RenewCalls[1].Next)
	}
}

// adminStatusTrackingVPSRepo wraps renewRetryVPSRepo and records local
// admin_status/status writes so the renewal path can be asserted to reset the
// stale auto-lock markers.
type adminStatusTrackingVPSRepo struct {
	renewRetryVPSRepo
	adminStatusSets []domain.VPSAdminStatus
	statusSets      []domain.VPSStatus
}

func (f *adminStatusTrackingVPSRepo) UpdateInstanceAdminStatus(ctx context.Context, id int64, status domain.VPSAdminStatus) error {
	f.adminStatusSets = append(f.adminStatusSets, status)
	f.inst.AdminStatus = status
	return nil
}

func (f *adminStatusTrackingVPSRepo) UpdateInstanceStatus(ctx context.Context, id int64, status domain.VPSStatus, automationState int) error {
	f.statusSets = append(f.statusSets, status)
	f.inst.Status = status
	return nil
}

// TestHandleRenew_ResetsStaleLockMarkers locks the P3-29 fix: the production
// order-path renewal must reset the locally persisted admin_status/status that
// the auto-lock job wrote when the instance expired, otherwise the next expiry
// is skipped forever (status=locked is a permanent exemption in AutoLockExpired).
func TestHandleRenew_ResetsStaleLockMarkers(t *testing.T) {
	expire := time.Now().Add(10 * 24 * time.Hour)
	vpsRepo := &adminStatusTrackingVPSRepo{
		renewRetryVPSRepo: renewRetryVPSRepo{
			inst: domain.VPSInstance{
				ID: 8, UserID: 9, GoodsTypeID: 1, AutomationInstanceID: "1008",
				ExpireAt: &expire, AdminStatus: domain.VPSAdminStatusLocked, Status: domain.VPSStatusExpiredLocked,
				SpecJSON: mustJSON(map[string]any{"current_period_end": expire.UTC().Format(time.RFC3339)}),
			},
		},
	}
	automation := &fakeLifecycleAutomationClient{}
	svc := NewOrderService(&fakeLifecycleOrderRepo{}, &fakeLifecycleOrderItemRepo{}, nil, nil, nil, nil, vpsRepo, nil, nil, nil, automation, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	item := domain.OrderItem{OrderID: 20, ID: 21, Action: "renew", SpecJSON: `{"vps_id":8,"renew_days":30}`}

	if err := svc.handleRenew(context.Background(), item); err != nil {
		t.Fatalf("handleRenew: %v", err)
	}
	found := false
	for _, st := range vpsRepo.adminStatusSets {
		if st == domain.VPSAdminStatusNormal {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected local admin_status reset to normal after renewal, sets=%v", vpsRepo.adminStatusSets)
	}
	for _, st := range vpsRepo.statusSets {
		if st != domain.VPSStatusExpiredLocked {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected expired_locked status cleared after renewal, sets=%v", vpsRepo.statusSets)
	}
}

// recordingEventPublisher captures published order events for assertions.
type recordingEventPublisher struct {
	events []struct {
		OrderID int64
		Type    string
	}
}

func (p *recordingEventPublisher) Publish(ctx context.Context, orderID int64, eventType string, payload any) (domain.OrderEvent, error) {
	p.events = append(p.events, struct {
		OrderID int64
		Type    string
	}{orderID, eventType})
	return domain.OrderEvent{OrderID: orderID, Type: eventType}, nil
}

// TestRefreshOrderStatus_DoesNotPublishCompletedOnFailure locks P3-30: an order
// transitioning to failed must not emit the "order.completed" event (downstream
// consumers treat it as success).
func TestRefreshOrderStatus_DoesNotPublishCompletedOnFailure(t *testing.T) {
	orders := &fakeLifecycleOrderRepo{orders: map[int64]domain.Order{
		1: {ID: 1, Status: domain.OrderStatusProvisioning},
	}}
	items := &fakeLifecycleOrderItemRepo{items: []domain.OrderItem{
		{ID: 1, OrderID: 1, Status: domain.OrderItemStatusFailed},
		{ID: 2, OrderID: 1, Status: domain.OrderItemStatusActive},
	}}
	events := &recordingEventPublisher{}
	svc := NewOrderService(orders, items, nil, nil, nil, nil, &fakeLifecycleVPSRepo{}, nil, nil, events, &fakeLifecycleAutomationClient{}, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	svc.refreshOrderStatus(context.Background(), 1)

	completed := false
	failed := false
	for _, e := range events.events {
		switch e.Type {
		case "order.completed":
			completed = true
		case "order.failed":
			failed = true
		}
	}
	if completed {
		t.Fatalf("order.completed must not fire when the order fails: %+v", events.events)
	}
	if !failed {
		t.Fatalf("expected order.failed event, got %+v", events.events)
	}
}
