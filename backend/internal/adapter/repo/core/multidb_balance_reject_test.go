package repo_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"xiaoheiplay/internal/adapter/repo/core"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

// runBalancePaymentAndRejectIsolation exercises the payment/reject race under a
// given dialect's transaction isolation level. Exactly one of the concurrent
// balance-payment commit and order-reject may win; the losers must not deduct
// balance, and the survivors must be consistent across payment/order/ledger.
// A rejected order must never show a deducted balance ("已扣余额但 rejected").
func runBalancePaymentAndRejectIsolation(t *testing.T, repo *repo.GormRepo, suffix string) {
	t.Helper()
	ctx := context.Background()
	// A shared external database persists rows across runs, so the identities
	// must be unique per invocation.
	run := fmt.Sprintf("%s-%d", suffix, time.Now().UnixNano())
	user := testutil.CreateUser(t, repo, "bal-reject-"+run, "bal-reject-"+run+"@example.com", "pass")
	if _, err := repo.AdjustWalletBalance(ctx, user.ID, 5000, "credit", "seed", 1, "init"); err != nil {
		t.Fatalf("seed wallet: %v", err)
	}
	order := domain.Order{UserID: user.ID, OrderNo: "ORD-BAL-REJECT-" + run, Status: domain.OrderStatusPendingPayment, TotalAmount: 2000, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	if err := repo.CreateOrderItems(ctx, []domain.OrderItem{{OrderID: order.ID, Amount: order.TotalAmount, Status: domain.OrderItemStatusPendingPayment, Action: "create", SpecJSON: "{}"}}); err != nil {
		t.Fatalf("create order item: %v", err)
	}

	start := make(chan struct{})
	results := make(chan bool, 2)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		payment := domain.OrderPayment{
			OrderID: order.ID, UserID: user.ID, Method: "balance", Amount: order.TotalAmount, Currency: order.Currency,
			TradeNo: fmt.Sprintf("BAL-%d", order.ID), Status: domain.PaymentStatusApproved, IdempotencyKey: fmt.Sprintf("balance:%d", order.ID),
		}
		_, claimed, err := repo.PayOrderWithBalanceAtomic(ctx, order, &payment)
		results <- claimed
		errs <- err
	}()
	go func() {
		defer wg.Done()
		<-start
		rejected := order
		rejected.Status = domain.OrderStatusRejected
		rejected.RejectedReason = "declined"
		claimed, err := repo.RejectOrderAtomic(ctx, rejected, 9, []domain.OrderStatus{domain.OrderStatusPendingPayment})
		results <- claimed
		errs <- err
	}()
	close(start)
	wg.Wait()
	close(results)
	close(errs)

	winners := 0
	for claimed := range results {
		if claimed {
			winners++
		}
	}
	for err := range errs {
		if err != nil {
			t.Fatalf("balance payment/reject race: %v", err)
		}
	}
	if winners != 1 {
		t.Fatalf("expected one decision winner, got %d", winners)
	}

	stored, err := repo.GetOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	wallet, err := repo.GetWallet(ctx, user.ID)
	if err != nil {
		t.Fatalf("get wallet: %v", err)
	}
	payments, err := repo.ListPaymentsByOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("list payments: %v", err)
	}
	switch stored.Status {
	case domain.OrderStatusPendingReview:
		if wallet.Balance != 3000 || len(payments) != 1 || payments[0].Status != domain.PaymentStatusApproved {
			t.Fatalf("paid order is inconsistent: wallet=%d payments=%+v", wallet.Balance, payments)
		}
	case domain.OrderStatusRejected:
		// The rejected branch must not have persisted a balance debit, payment,
		// or review handoff: money in, no deduction.
		if wallet.Balance != 5000 || len(payments) != 0 {
			t.Fatalf("rejected order retained a debit: wallet=%d payments=%+v", wallet.Balance, payments)
		}
	default:
		t.Fatalf("unexpected order status: %s", stored.Status)
	}
}

func TestMultiDialect_BalancePaymentAndRejectIsolation(t *testing.T) {
	// SQLite is already covered by TestSQLiteRepo_BalancePaymentAndRejectHaveSingleWinner.
	// This test exists to prove the same invariants hold under the MySQL and
	// PostgreSQL isolation levels (the SQLite single-writer model can also raise
	// transient database-is-locked in this race, which is not the behavior under test).
	for _, dialect := range []string{"mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			repo := openDialectRepo(t, dialect)
			runBalancePaymentAndRejectIsolation(t, repo, dialect)
		})
	}
}
