package order_test

import (
	"context"
	"sync"
	"testing"
	"time"

	apporder "xiaoheiplay/internal/app/order"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

// recordingEvents captures the event types published during provisioning so the
// completion-event contract can be asserted.
type recordingEvents struct {
	mu     sync.Mutex
	events []string
}

func (r *recordingEvents) Publish(ctx context.Context, orderID int64, eventType string, payload any) (domain.OrderEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, eventType)
	return domain.OrderEvent{OrderID: orderID, Type: eventType}, nil
}

func (r *recordingEvents) has(t string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range r.events {
		if e == t {
			return true
		}
	}
	return false
}

// waitForOrder settles until the order reaches a terminal status or the timeout
// elapses, mirroring how the synchronous provision tests poll the goroutine that
// ApproveOrder launches.
func waitForOrder(t *testing.T, repo interface {
	GetOrder(ctx context.Context, id int64) (domain.Order, error)
}, orderID int64, want domain.OrderStatus) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		order, err := repo.GetOrder(context.Background(), orderID)
		if err == nil && order.Status == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	order, _ := repo.GetOrder(context.Background(), orderID)
	t.Fatalf("order did not reach %s, got %s", want, order.Status)
}

// TestProvisionOrder_PublishesCompletedOnlyWhenActive locks the completion-event
// contract: an order that provisions successfully emits order.completed, and an
// order that fails emits order.failed — never order.completed — while an order
// still provisioning emits neither.
func TestProvisionOrder_PublishesCompletedOnlyWhenActive(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	seed := testutil.SeedCatalog(t, repo)
	user := testutil.CreateUser(t, repo, "prov-evt", "prov-evt@example.com", "pass")

	newOrder := func(orderNo string, itemStatus domain.OrderItemStatus, automationID string) domain.Order {
		order := domain.Order{UserID: user.ID, OrderNo: orderNo, Status: domain.OrderStatusApproved, TotalAmount: 1000, Currency: "CNY"}
		if err := repo.CreateOrder(context.Background(), &order); err != nil {
			t.Fatalf("create order: %v", err)
		}
		if err := repo.CreateOrderItems(context.Background(), []domain.OrderItem{{
			OrderID: order.ID, PackageID: seed.Package.ID, SystemID: seed.SystemImage.ID, Amount: 1000,
			Status: itemStatus, Action: "create", SpecJSON: "{}", AutomationInstanceID: automationID,
		}}); err != nil {
			t.Fatalf("create item: %v", err)
		}
		return order
	}

	t.Run("active publishes completed", func(t *testing.T) {
		order := newOrder("ORD-EVT-OK", domain.OrderItemStatusApproved, "")
		recorder := &recordingEvents{}
		fakeAuto := &testutil.FakeAutomationClient{
			CreateHostResult: appshared.AutomationCreateHostResult{HostID: 1001},
			HostInfo:         map[int64]appshared.AutomationHostInfo{1001: {HostID: 1001, HostName: "host", State: 2}},
		}
		svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, recorder, &testutil.FakeAutomationResolver{Client: fakeAuto}, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
		if err := svc.ApproveOrder(context.Background(), 1, order.ID); err != nil {
			t.Fatalf("approve: %v", err)
		}
		waitForOrder(t, repo, order.ID, domain.OrderStatusActive)
		if !recorder.has("order.completed") {
			t.Fatalf("expected order.completed for an active order, got %v", recorder.events)
		}
		if recorder.has("order.failed") {
			t.Fatalf("unexpected order.failed for an active order: %v", recorder.events)
		}
	})

	t.Run("failed publishes failed not completed", func(t *testing.T) {
		order := newOrder("ORD-EVT-FAIL", domain.OrderItemStatusApproved, "")
		recorder := &recordingEvents{}
		fakeAuto := &testutil.FakeAutomationClient{
			CreateHostErr: context.DeadlineExceeded,
		}
		svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, recorder, &testutil.FakeAutomationResolver{Client: fakeAuto}, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
		if err := svc.ApproveOrder(context.Background(), 1, order.ID); err != nil {
			t.Fatalf("approve: %v", err)
		}
		waitForOrder(t, repo, order.ID, domain.OrderStatusFailed)
		if recorder.has("order.completed") {
			t.Fatalf("order.completed must not fire when the order fails: %v", recorder.events)
		}
		if !recorder.has("order.failed") {
			t.Fatalf("expected order.failed for a failed order, got %v", recorder.events)
		}
	})

	t.Run("still provisioning publishes neither", func(t *testing.T) {
		// The host comes up but never reaches a ready state, so the create intent
		// stays provisioning and the order must NOT emit a completion event.
		order := newOrder("ORD-EVT-PEND", domain.OrderItemStatusProvisioning, "")
		recorder := &recordingEvents{}
		fakeAuto := &testutil.FakeAutomationClient{
			CreateHostResult: appshared.AutomationCreateHostResult{HostID: 1003},
			HostInfo:         map[int64]appshared.AutomationHostInfo{1003: {HostID: 1003, State: 0}},
		}
		svc := apporder.NewService(repo, repo, repo, repo, repo, repo, repo, repo, repo, recorder, &testutil.FakeAutomationResolver{Client: fakeAuto}, nil, repo, repo, nil, repo, repo, repo, nil, nil, nil)
		if err := svc.ApproveOrder(context.Background(), 1, order.ID); err != nil {
			t.Fatalf("approve: %v", err)
		}
		deadline := time.Now().Add(500 * time.Millisecond)
		for time.Now().Before(deadline) {
			time.Sleep(10 * time.Millisecond)
		}
		if recorder.has("order.completed") {
			t.Fatalf("order.completed must not fire while the order is still provisioning: %v", recorder.events)
		}
		if recorder.has("order.failed") {
			t.Fatalf("order.failed must not fire while the order is still provisioning: %v", recorder.events)
		}
	})
}
