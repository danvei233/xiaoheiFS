package repo_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func TestSQLiteRepo_OrderApprovalAndProvisionClaimsAreCAS(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	user := testutil.CreateUser(t, repo, "order-cas", "order-cas@example.com", "pass")
	order := domain.Order{UserID: user.ID, OrderNo: "ORD-CAS", Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	if err := repo.CreateOrderItems(ctx, []domain.OrderItem{{OrderID: order.ID, Amount: 1000, Status: domain.OrderItemStatusPendingPayment, Action: "create", SpecJSON: "{}"}}); err != nil {
		t.Fatalf("create order item: %v", err)
	}
	payment := domain.OrderPayment{OrderID: order.ID, UserID: user.ID, Method: "manual", Amount: order.TotalAmount, Currency: order.Currency, TradeNo: "ORD-CAS-PAY", Status: domain.PaymentStatusPendingReview}
	if err := repo.CreatePayment(ctx, &payment); err != nil {
		t.Fatalf("create payment: %v", err)
	}
	items, err := repo.ListOrderItems(ctx, order.ID)
	if err != nil || len(items) != 1 {
		t.Fatalf("list order items: len=%d err=%v", len(items), err)
	}

	approved := order
	approved.Status = domain.OrderStatusApproved
	now := time.Now()
	approved.ApprovedAt = &now
	itemIDs := []int64{items[0].ID}
	current := []domain.OrderStatus{domain.OrderStatusPendingPayment, domain.OrderStatusPendingReview, domain.OrderStatusRejected}

	var wg sync.WaitGroup
	results := make(chan bool, 2)
	errs := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			claimed, err := repo.ApproveOrderAtomic(ctx, approved, itemIDs, nil, current)
			results <- claimed
			errs <- err
		}()
	}
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
		if err != nil && err != appshared.ErrConflict {
			t.Fatalf("approve order CAS: %v", err)
		}
	}
	if winners != 1 {
		t.Fatalf("expected one approval winner, got %d", winners)
	}
	payments, err := repo.ListPaymentsByOrder(ctx, order.ID)
	if err != nil || len(payments) != 1 || payments[0].Status != domain.PaymentStatusApproved {
		t.Fatalf("approval did not atomically approve payment: payments=%+v err=%v", payments, err)
	}

	claimResults := make(chan bool, 2)
	claimErrs := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			claimed, err := repo.ClaimOrderItemForProvision(ctx, items[0].ID)
			if err != nil {
				claimErrs <- err
				return
			}
			claimResults <- claimed
		}()
	}
	wg.Wait()
	close(claimResults)
	close(claimErrs)
	claimWinners := 0
	for claimed := range claimResults {
		if claimed {
			claimWinners++
		}
	}
	if claimWinners != 1 {
		t.Fatalf("expected one provision claim winner, got %d", claimWinners)
	}
	for err := range claimErrs {
		t.Fatalf("claim order item: %v", err)
	}
}

func TestSQLiteRepo_OrderApproveRejectRaceStaysConsistent(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	user := testutil.CreateUser(t, repo, "order-decision-cas", "order-decision-cas@example.com", "pass")
	order := domain.Order{UserID: user.ID, OrderNo: "ORD-DECISION-CAS", Status: domain.OrderStatusPendingReview, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	if err := repo.CreateOrderItems(ctx, []domain.OrderItem{{OrderID: order.ID, Amount: 1000, Status: domain.OrderItemStatusPendingReview, Action: "create", SpecJSON: "{}"}}); err != nil {
		t.Fatalf("create order item: %v", err)
	}
	items, err := repo.ListOrderItems(ctx, order.ID)
	if err != nil || len(items) != 1 {
		t.Fatalf("list order items: len=%d err=%v", len(items), err)
	}

	approved := order
	approved.Status = domain.OrderStatusApproved
	rejected := order
	rejected.Status = domain.OrderStatusRejected
	rejected.RejectedReason = "declined"
	current := []domain.OrderStatus{domain.OrderStatusPendingReview}

	start := make(chan struct{})
	results := make(chan bool, 2)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		claimed, err := repo.ApproveOrderAtomic(ctx, approved, []int64{items[0].ID}, nil, current)
		results <- claimed
		errs <- err
	}()
	go func() {
		defer wg.Done()
		<-start
		claimed, err := repo.RejectOrderAtomic(ctx, rejected, 9, current)
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
			t.Fatalf("order decision race: %v", err)
		}
	}
	if winners != 1 {
		t.Fatalf("expected one order decision winner, got %d", winners)
	}
	storedOrder, err := repo.GetOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	storedItems, err := repo.ListOrderItems(ctx, order.ID)
	if err != nil || len(storedItems) != 1 {
		t.Fatalf("list stored items: len=%d err=%v", len(storedItems), err)
	}
	if storedOrder.Status == domain.OrderStatusApproved && storedItems[0].Status != domain.OrderItemStatusApproved {
		t.Fatalf("approved order has inconsistent item status %s", storedItems[0].Status)
	}
	if storedOrder.Status == domain.OrderStatusRejected && storedItems[0].Status != domain.OrderItemStatusRejected {
		t.Fatalf("rejected order has inconsistent item status %s", storedItems[0].Status)
	}
}

func TestSQLiteRepo_ApproveWalletOrderAtomicConcurrent(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	user := testutil.CreateUser(t, repo, "wallet-cas", "wallet-cas@example.com", "pass")
	order := domain.WalletOrder{UserID: user.ID, Type: domain.WalletOrderRecharge, Amount: 2500, Currency: "CNY", Status: domain.WalletOrderPendingReview, MetaJSON: "{}"}
	if err := repo.CreateWalletOrder(ctx, &order); err != nil {
		t.Fatalf("create wallet order: %v", err)
	}

	var wg sync.WaitGroup
	winners := make(chan bool, 2)
	errs := make(chan error, 2)
	for adminID := int64(1); adminID <= 2; adminID++ {
		wg.Add(1)
		go func(adminID int64) {
			defer wg.Done()
			_, claimed, err := repo.ApproveWalletOrderAtomic(ctx, order, adminID, order.Amount, "credit")
			winners <- claimed
			errs <- err
		}(adminID)
	}
	wg.Wait()
	close(winners)
	close(errs)
	claimedCount := 0
	for claimed := range winners {
		if claimed {
			claimedCount++
		}
	}
	for err := range errs {
		if err != nil && err != appshared.ErrConflict {
			t.Fatalf("approve wallet order: %v", err)
		}
	}
	if claimedCount != 1 {
		t.Fatalf("expected one wallet approval winner, got %d", claimedCount)
	}
	wallet, err := repo.GetWallet(ctx, user.ID)
	if err != nil {
		t.Fatalf("get wallet: %v", err)
	}
	if wallet.Balance != order.Amount {
		t.Fatalf("expected one credit of %d, got %d", order.Amount, wallet.Balance)
	}
	txs, total, err := repo.ListWalletTransactions(ctx, user.ID, 10, 0)
	if err != nil {
		t.Fatalf("list wallet transactions: %v", err)
	}
	if total != 1 || len(txs) != 1 || txs[0].RefType != "wallet_order" || txs[0].RefID != order.ID {
		t.Fatalf("expected one wallet-order transaction, total=%d txs=%+v", total, txs)
	}
}

func TestSQLiteRepo_WalletApproveRejectRaceStaysConsistent(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	user := testutil.CreateUser(t, repo, "wallet-decision-cas", "wallet-decision-cas@example.com", "pass")
	order := domain.WalletOrder{UserID: user.ID, Type: domain.WalletOrderRecharge, Amount: 2500, Currency: "CNY", Status: domain.WalletOrderPendingReview, MetaJSON: "{}"}
	if err := repo.CreateWalletOrder(ctx, &order); err != nil {
		t.Fatalf("create wallet order: %v", err)
	}

	start := make(chan struct{})
	results := make(chan bool, 2)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		_, claimed, err := repo.ApproveWalletOrderAtomic(ctx, order, 1, order.Amount, "credit")
		results <- claimed
		errs <- err
	}()
	go func() {
		defer wg.Done()
		<-start
		claimed, err := repo.UpdateWalletOrderStatusIfCurrent(ctx, order.ID, domain.WalletOrderPendingReview, domain.WalletOrderRejected, nil, "declined")
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
			t.Fatalf("wallet decision race: %v", err)
		}
	}
	if winners != 1 {
		t.Fatalf("expected one wallet decision winner, got %d", winners)
	}
	stored, err := repo.GetWalletOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("get wallet order: %v", err)
	}
	wallet, walletErr := repo.GetWallet(ctx, user.ID)
	if stored.Status == domain.WalletOrderApproved {
		if walletErr != nil || wallet.Balance != order.Amount {
			t.Fatalf("approved wallet order must have one credit: wallet=%+v err=%v", wallet, walletErr)
		}
	} else if stored.Status == domain.WalletOrderRejected {
		if walletErr == nil && wallet.Balance != 0 {
			t.Fatalf("rejected wallet order changed balance to %d", wallet.Balance)
		}
	} else {
		t.Fatalf("unexpected wallet order status %s", stored.Status)
	}
}

func TestSQLiteRepo_WalletOrderLedgerKeyBlocksDuplicateAfterStatusRegression(t *testing.T) {
	db, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	user := testutil.CreateUser(t, repo, "wallet-key", "wallet-key@example.com", "pass")
	order := domain.WalletOrder{UserID: user.ID, Type: domain.WalletOrderRecharge, Amount: 900, Currency: "CNY", Status: domain.WalletOrderPendingReview, MetaJSON: "{}"}
	if err := repo.CreateWalletOrder(ctx, &order); err != nil {
		t.Fatalf("create wallet order: %v", err)
	}
	if _, claimed, err := repo.ApproveWalletOrderAtomic(ctx, order, 1, order.Amount, "credit"); err != nil || !claimed {
		t.Fatalf("first approval: claimed=%v err=%v", claimed, err)
	}

	if _, err := db.Exec("UPDATE wallet_orders SET status = ? WHERE id = ?", domain.WalletOrderPendingReview, order.ID); err != nil {
		t.Fatalf("regress wallet order status: %v", err)
	}
	if _, claimed, err := repo.ApproveWalletOrderAtomic(ctx, order, 2, order.Amount, "credit"); err == nil || claimed {
		t.Fatalf("expected ledger key conflict after status regression, claimed=%v err=%v", claimed, err)
	}
	wallet, err := repo.GetWallet(ctx, user.ID)
	if err != nil {
		t.Fatalf("get wallet: %v", err)
	}
	if wallet.Balance != order.Amount {
		t.Fatalf("duplicate credit changed balance to %d", wallet.Balance)
	}
	_, total, err := repo.ListWalletTransactions(ctx, user.ID, 10, 0)
	if err != nil {
		t.Fatalf("list wallet transactions: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected one durable ledger row, got %d", total)
	}
}

func TestSQLiteRepo_WalletIdempotencyDoesNotBlockRepeatedAdminAdjustments(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	user := testutil.CreateUser(t, repo, "wallet-adjust", "wallet-adjust@example.com", "pass")

	for range 2 {
		if _, err := repo.AdjustWalletBalance(ctx, user.ID, 100, "credit", "admin_adjust", 7, "manual adjustment"); err != nil {
			t.Fatalf("adjust wallet: %v", err)
		}
	}
	wallet, err := repo.GetWallet(ctx, user.ID)
	if err != nil {
		t.Fatalf("get wallet: %v", err)
	}
	if wallet.Balance != 200 {
		t.Fatalf("expected both admin adjustments, got balance %d", wallet.Balance)
	}
	_, total, err := repo.ListWalletTransactions(ctx, user.ID, 10, 0)
	if err != nil {
		t.Fatalf("list wallet transactions: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected two admin adjustment transactions, got %d", total)
	}
}

func TestSQLiteRepo_PayOrderWithBalanceAtomicRollsBackOnPaymentFailure(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	user := testutil.CreateUser(t, repo, "balance-rollback", "balance-rollback@example.com", "pass")
	if _, err := repo.AdjustWalletBalance(ctx, user.ID, 5000, "credit", "seed", 1, "init"); err != nil {
		t.Fatalf("seed wallet: %v", err)
	}

	order := domain.Order{UserID: user.ID, OrderNo: "ORD-BAL-ROLLBACK", Status: domain.OrderStatusPendingPayment, TotalAmount: 2000, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	if err := repo.CreateOrderItems(ctx, []domain.OrderItem{{OrderID: order.ID, Amount: 2000, Status: domain.OrderItemStatusPendingPayment, Action: "create", SpecJSON: "{}"}}); err != nil {
		t.Fatalf("create order item: %v", err)
	}
	items, err := repo.ListOrderItems(ctx, order.ID)
	if err != nil || len(items) != 1 {
		t.Fatalf("list order items: len=%d err=%v", len(items), err)
	}

	other := domain.Order{UserID: user.ID, OrderNo: "ORD-BAL-CONFLICT", Status: domain.OrderStatusPendingPayment, TotalAmount: 1, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &other); err != nil {
		t.Fatalf("create other order: %v", err)
	}
	tradeNo := fmt.Sprintf("BAL-%d", order.ID)
	conflict := domain.OrderPayment{OrderID: other.ID, UserID: user.ID, Method: "balance", Amount: 1, Currency: "CNY", TradeNo: tradeNo, Status: domain.PaymentStatusApproved}
	if err := repo.CreatePayment(ctx, &conflict); err != nil {
		t.Fatalf("create conflicting payment: %v", err)
	}

	payment := domain.OrderPayment{OrderID: order.ID, UserID: user.ID, Method: "balance", Amount: order.TotalAmount, Currency: order.Currency, TradeNo: tradeNo, Status: domain.PaymentStatusApproved, IdempotencyKey: fmt.Sprintf("balance:%d", order.ID)}
	if _, claimed, err := repo.PayOrderWithBalanceAtomic(ctx, order, &payment); err == nil || claimed {
		t.Fatalf("expected payment insert failure and no claim, claimed=%v err=%v", claimed, err)
	}

	wallet, err := repo.GetWallet(ctx, user.ID)
	if err != nil {
		t.Fatalf("get wallet: %v", err)
	}
	if wallet.Balance != 5000 {
		t.Fatalf("balance changed despite rollback: %d", wallet.Balance)
	}
	gotOrder, err := repo.GetOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if gotOrder.Status != domain.OrderStatusPendingPayment {
		t.Fatalf("order status changed despite rollback: %s", gotOrder.Status)
	}
	gotItems, err := repo.ListOrderItems(ctx, order.ID)
	if err != nil || len(gotItems) != 1 {
		t.Fatalf("list items after rollback: len=%d err=%v", len(gotItems), err)
	}
	if gotItems[0].Status != domain.OrderItemStatusPendingPayment {
		t.Fatalf("item status changed despite rollback: %s", gotItems[0].Status)
	}
	hasTx, err := repo.HasWalletTransaction(ctx, user.ID, "order", order.ID)
	if err != nil {
		t.Fatalf("check wallet transaction: %v", err)
	}
	if hasTx {
		t.Fatalf("wallet transaction persisted despite rollback")
	}
}

func TestSQLiteRepo_PayOrderWithBalanceAtomicConcurrent(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	user := testutil.CreateUser(t, repo, "balance-cas", "balance-cas@example.com", "pass")
	if _, err := repo.AdjustWalletBalance(ctx, user.ID, 5000, "credit", "seed", 1, "init"); err != nil {
		t.Fatalf("seed wallet: %v", err)
	}
	order := domain.Order{UserID: user.ID, OrderNo: "ORD-BAL-CAS", Status: domain.OrderStatusPendingPayment, TotalAmount: 2000, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	if err := repo.CreateOrderItems(ctx, []domain.OrderItem{{OrderID: order.ID, Amount: 2000, Status: domain.OrderItemStatusPendingPayment, Action: "create", SpecJSON: "{}"}}); err != nil {
		t.Fatalf("create order item: %v", err)
	}

	var wg sync.WaitGroup
	claimedResults := make(chan bool, 2)
	errs := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			payment := domain.OrderPayment{
				OrderID: order.ID, UserID: user.ID, Method: "balance", Amount: order.TotalAmount, Currency: order.Currency,
				TradeNo: fmt.Sprintf("BAL-%d", order.ID), Status: domain.PaymentStatusApproved, IdempotencyKey: fmt.Sprintf("balance:%d", order.ID),
			}
			_, claimed, err := repo.PayOrderWithBalanceAtomic(ctx, order, &payment)
			claimedResults <- claimed
			errs <- err
		}()
	}
	wg.Wait()
	close(claimedResults)
	close(errs)
	winners := 0
	for claimed := range claimedResults {
		if claimed {
			winners++
		}
	}
	for err := range errs {
		if err != nil {
			t.Fatalf("pay with balance: %v", err)
		}
	}
	if winners != 1 {
		t.Fatalf("expected one balance payment winner, got %d", winners)
	}
	wallet, err := repo.GetWallet(ctx, user.ID)
	if err != nil {
		t.Fatalf("get wallet: %v", err)
	}
	if wallet.Balance != 3000 {
		t.Fatalf("expected one debit to leave 3000, got %d", wallet.Balance)
	}
	txs, total, err := repo.ListWalletTransactions(ctx, user.ID, 10, 0)
	if err != nil {
		t.Fatalf("list transactions: %v", err)
	}
	orderTxs := 0
	for _, tx := range txs {
		if tx.RefType == "order" && tx.RefID == order.ID {
			orderTxs++
		}
	}
	if total != 2 || orderTxs != 1 {
		t.Fatalf("expected seed plus one order transaction, total=%d order_txs=%d", total, orderTxs)
	}
	payments, err := repo.ListPaymentsByOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("list payments: %v", err)
	}
	if len(payments) != 1 || payments[0].Status != domain.PaymentStatusApproved {
		t.Fatalf("expected one approved payment, got %+v", payments)
	}
	gotOrder, err := repo.GetOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if gotOrder.Status != domain.OrderStatusPendingReview {
		t.Fatalf("expected durable pending-review handoff, got %s", gotOrder.Status)
	}
}

func TestSQLiteRepo_BalancePaymentAndRejectHaveSingleWinner(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	user := testutil.CreateUser(t, repo, "balance-reject-cas", "balance-reject-cas@example.com", "pass")
	if _, err := repo.AdjustWalletBalance(ctx, user.ID, 5000, "credit", "seed", 1, "init"); err != nil {
		t.Fatalf("seed wallet: %v", err)
	}
	order := domain.Order{UserID: user.ID, OrderNo: "ORD-BAL-REJECT-CAS", Status: domain.OrderStatusPendingPayment, TotalAmount: 2000, Currency: "CNY"}
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
		if wallet.Balance != 5000 || len(payments) != 0 {
			t.Fatalf("rejected order retained a debit: wallet=%d payments=%+v", wallet.Balance, payments)
		}
	default:
		t.Fatalf("unexpected order status: %s", stored.Status)
	}
}

func TestSQLiteRepo_PayOrderWithBalanceRejectsUnprovenIdempotencyRecord(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	user := testutil.CreateUser(t, repo, "balance-proof", "balance-proof@example.com", "pass")
	if _, err := repo.AdjustWalletBalance(ctx, user.ID, 5000, "credit", "seed", 1, "init"); err != nil {
		t.Fatalf("seed wallet: %v", err)
	}
	order := domain.Order{UserID: user.ID, OrderNo: "ORD-BAL-PROOF", Status: domain.OrderStatusPendingPayment, TotalAmount: 2000, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	if _, err := repo.AdjustWalletBalance(ctx, user.ID, -order.TotalAmount, "debit", "order", order.ID, "legacy debit without atomic settlement key"); err != nil {
		t.Fatalf("create unproven debit: %v", err)
	}
	idempotencyKey := fmt.Sprintf("balance:%d", order.ID)
	conflict := domain.OrderPayment{
		OrderID: order.ID, UserID: user.ID, Method: "balance", Amount: order.TotalAmount, Currency: order.Currency,
		TradeNo: "MANUAL-BALANCE-PROOF", Status: domain.PaymentStatusApproved, IdempotencyKey: idempotencyKey,
	}
	if err := repo.CreatePayment(ctx, &conflict); err != nil {
		t.Fatalf("create conflicting payment: %v", err)
	}
	payment := domain.OrderPayment{
		OrderID: order.ID, UserID: user.ID, Method: "balance", Amount: order.TotalAmount, Currency: order.Currency,
		TradeNo: fmt.Sprintf("BAL-%d", order.ID), Status: domain.PaymentStatusApproved, IdempotencyKey: idempotencyKey,
	}
	if _, claimed, err := repo.PayOrderWithBalanceAtomic(ctx, order, &payment); err != appshared.ErrConflict || claimed {
		t.Fatalf("expected conflict without balance settlement proof, claimed=%v err=%v", claimed, err)
	}
}
