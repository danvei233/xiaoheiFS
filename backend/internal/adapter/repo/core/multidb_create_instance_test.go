package repo_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

// TestMultiDialect_CreateInstanceConcurrentSameOrderItem runs N concurrent
// CreateInstance calls against the same order_item_id. The order_item_id unique
// index must admit exactly one persisted VPS row: replays of the winning
// automation host are idempotent, and a different automation host under the same
// order item is surfaced as ErrConflict rather than silently returning a
// different instance.
func TestMultiDialect_CreateInstanceConcurrentSameOrderItem(t *testing.T) {
	for _, dialect := range availableDialects(t) {
		t.Run(dialect, func(t *testing.T) {
			repo := openDialectRepo(t, dialect)
			ctx := context.Background()
			run := fmt.Sprintf("%s-%d", dialect, time.Now().UnixNano())
			user := testutil.CreateUser(t, repo, "inst-conc-"+run, "inst-conc-"+run+"@example.com", "pass")
			order := domain.Order{UserID: user.ID, OrderNo: "ORD-INST-CONC-" + run, Status: domain.OrderStatusApproved, TotalAmount: 1000, Currency: "CNY"}
			if err := repo.CreateOrder(ctx, &order); err != nil {
				t.Fatalf("create order: %v", err)
			}
			if err := repo.CreateOrderItems(ctx, []domain.OrderItem{{OrderID: order.ID, Amount: 1000, Status: domain.OrderItemStatusApproved, Action: "create", SpecJSON: "{}"}}); err != nil {
				t.Fatalf("create order item: %v", err)
			}
			items, err := repo.ListOrderItems(ctx, order.ID)
			if err != nil || len(items) != 1 {
				t.Fatalf("list items: len=%d err=%v", len(items), err)
			}

			const n = 8
			start := make(chan struct{})
			var wg sync.WaitGroup
			ids := make(chan int64, n)
			errs := make(chan error, n)
			for i := 0; i < n; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					<-start
					inst := domain.VPSInstance{
						UserID: user.ID, OrderItemID: items[0].ID,
						AutomationInstanceID: fmt.Sprintf("host-%d", i),
						Name: fmt.Sprintf("vm-%d", i), Status: domain.VPSStatusRunning,
						AdminStatus: domain.VPSAdminStatusNormal, SpecJSON: "{}",
					}
					err := repo.CreateInstance(ctx, &inst)
					if err != nil {
						errs <- err
						return
					}
					ids <- inst.ID
				}(i)
			}
			close(start)
			wg.Wait()
			close(ids)
			close(errs)

			inserted := 0
			conflicts := 0
			instanceIDs := map[int64]struct{}{}
			for id := range ids {
				inserted++
				instanceIDs[id] = struct{}{}
			}
			for err := range errs {
				if err == domain.ErrConflict {
					conflicts++
					continue
				}
				t.Fatalf("create instance: %v", err)
			}
			if inserted+conflicts != n {
				t.Fatalf("expected %d total outcomes, got inserted=%d conflicts=%d", n, inserted, conflicts)
			}
			if inserted != 1 {
				t.Fatalf("expected exactly one successful insert, got %d", inserted)
			}
			if len(instanceIDs) != 1 {
				t.Fatalf("expected all survivors to agree on one instance, got %v", instanceIDs)
			}
			// The stored instance count for the order item must be exactly one.
			stored, err := repo.GetInstanceByOrderItem(ctx, items[0].ID)
			if err != nil {
				t.Fatalf("get stored instance: %v", err)
			}
			if stored.OrderItemID != items[0].ID || stored.UserID != user.ID {
				t.Fatalf("stored instance does not match the order item/user: %+v", stored)
			}
		})
	}
}

// TestMultiDialect_CreateInstanceConcurrentSameHostReplay verifies that a full
// set of concurrent replays of the SAME automation host under one order item is
// idempotent: no conflicts, one persisted row, all calls see the same instance.
func TestMultiDialect_CreateInstanceConcurrentSameHostReplay(t *testing.T) {
	for _, dialect := range availableDialects(t) {
		t.Run(dialect, func(t *testing.T) {
			repo := openDialectRepo(t, dialect)
			ctx := context.Background()
			run := fmt.Sprintf("%s-%d", dialect, time.Now().UnixNano())
			user := testutil.CreateUser(t, repo, "inst-replay-"+run, "inst-replay-"+run+"@example.com", "pass")
			order := domain.Order{UserID: user.ID, OrderNo: "ORD-INST-REPLAY-" + run, Status: domain.OrderStatusApproved, TotalAmount: 1000, Currency: "CNY"}
			if err := repo.CreateOrder(ctx, &order); err != nil {
				t.Fatalf("create order: %v", err)
			}
			if err := repo.CreateOrderItems(ctx, []domain.OrderItem{{OrderID: order.ID, Amount: 1000, Status: domain.OrderItemStatusApproved, Action: "create", SpecJSON: "{}"}}); err != nil {
				t.Fatalf("create order item: %v", err)
			}
			items, err := repo.ListOrderItems(ctx, order.ID)
			if err != nil || len(items) != 1 {
				t.Fatalf("list items: len=%d err=%v", len(items), err)
			}

			const n = 8
			start := make(chan struct{})
			var wg sync.WaitGroup
			ids := make(chan int64, n)
			errs := make(chan error, n)
			for i := 0; i < n; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					<-start
					inst := domain.VPSInstance{
						UserID: user.ID, OrderItemID: items[0].ID,
						AutomationInstanceID: "same-host", Name: "same-vm", Status: domain.VPSStatusRunning,
						AdminStatus: domain.VPSAdminStatusNormal, SpecJSON: "{}",
					}
					err := repo.CreateInstance(ctx, &inst)
					if err != nil {
						errs <- err
						return
					}
					ids <- inst.ID
				}()
			}
			close(start)
			wg.Wait()
			close(ids)
			close(errs)

			inserted := 0
			instanceIDs := map[int64]struct{}{}
			for id := range ids {
				inserted++
				instanceIDs[id] = struct{}{}
			}
			for err := range errs {
				t.Fatalf("idempotent replay must not conflict: %v", err)
			}
			if inserted != n {
				t.Fatalf("expected all %d calls to succeed idempotently, got %d", n, inserted)
			}
			if len(instanceIDs) != 1 {
				t.Fatalf("expected all replays to resolve to one instance, got %v", instanceIDs)
			}
			stored, err := repo.GetInstanceByOrderItem(ctx, items[0].ID)
			if err != nil {
				t.Fatalf("expected exactly one persisted instance for the order item: %v", err)
			}
			if stored.UserID != user.ID || stored.AutomationInstanceID != "same-host" {
				t.Fatalf("stored instance mismatch: %+v", stored)
			}
		})
	}
}
