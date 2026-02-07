# Bug Report - Core Business Logic Issues

> Date: 2026-02-07
> Project: Cloud VPS Console Backend
> Severity Classification: Critical | High | Medium | Low

---

## Executive Summary

This report documents **12 significant bugs** found in the core business logic of the Cloud VPS Console backend. Among these, **3 are critical** and require immediate attention as they can lead to data inconsistency and financial losses.

---

## Critical Severity Bugs

### 1. Race Condition in Wallet Balance Adjustment

**Location:** `internal/adapter/repo/sqlite_repo.go:2724-2762`

**Function:** `AdjustWalletBalance()`

**Description:**
The wallet balance adjustment uses a check-then-update pattern that is vulnerable to race conditions. Two concurrent transactions could both read the same balance before either writes, leading to incorrect final balances.

**Vulnerable Code:**
```go
row := tx.QueryRowContext(ctx, `SELECT id, user_id, balance, updated_at FROM user_wallets WHERE user_id = ?`, userID)
if err = row.Scan(&wallet.ID, &wallet.UserID, &wallet.Balance, &wallet.UpdatedAt); err != nil {
    // ... handle missing wallet
}
newBalance := wallet.Balance + amount
if newBalance < 0 {
    err = usecase.ErrInsufficientBalance
    return domain.Wallet{}, err
}
if _, err = tx.ExecContext(ctx, `UPDATE user_wallets SET balance = ?, updated_at = CURRENT_TIMESTAMP WHERE user_id = ?`, newBalance, userID); err != nil {
    return domain.Wallet{}, err
}
```

**Attack Scenario:**
1. User has balance of 100
2. Two concurrent requests each try to debit 50
3. Both transactions read balance=100
4. Both pass the `newBalance < 0` check (100-50=50 >= 0)
5. First updates balance to 50
6. Second updates balance to 50
7. Final balance: 50 (should be 0)
8. User gained 50 units of currency!

**Recommended Fix:**
```sql
UPDATE user_wallets
SET balance = balance + ?,
    updated_at = CURRENT_TIMESTAMP
WHERE user_id = ? AND balance + ? >= 0
```

Then check `RowsAffected()` to verify the update succeeded.

---

### 2. Silent Error Handling in Payment Approval

**Location:** `internal/usecase/payment_service.go:230-242`

**Function:** `NotifyPayment()`

**Description:**
When processing payment notifications, errors during order approval are silently ignored (`_ = s.approver.ApproveOrder(...)`). This can leave payments marked as approved while orders remain unapproved, causing state inconsistency.

**Vulnerable Code:**
```go
if s.approver != nil {
    _ = s.approver.ApproveOrder(ctx, 0, payment.OrderID)  // Line 234 - ERROR IGNORED!
}
```

**Impact:**
- Payment marked as "paid" but order still in "pending_review" status
- User funds deducted but service not provisioned
- Manual intervention required to fix

**Recommended Fix:**
```go
if s.approver != nil {
    if err := s.approver.ApproveOrder(ctx, 0, payment.OrderID); err != nil {
        // Rollback payment or mark as needs_retry
        return result, fmt.Errorf("approve order after payment: %w", err)
    }
}
```

---

### 3. Double Refund Vulnerability

**Location:** `internal/usecase/order_service.go:1258-1306`

**Function:** `createAndApproveWalletRefund()`

**Description:**
The duplicate refund check is vulnerable to race conditions. Multiple concurrent refund approval requests for the same order could all pass the `HasWalletTransaction` check before any transaction is recorded.

**Vulnerable Code:**
```go
if txRefType != "" && txRefID > 0 {
    exists, err := s.wallets.HasWalletTransaction(ctx, userID, txRefType, txRefID)
    if err != nil {
        return err
    }
    if exists {
        return nil  // No atomicity here!
    }
}
// ... wallet balance adjustment happens next
```

**Attack Scenario:**
1. User has an order eligible for 100 refund
2. User sends two concurrent refund requests
3. Both requests check `HasWalletTransaction` - both return false
4. Both requests proceed to create refund
5. User receives 200 total refund (double payment!)

**Recommended Fix:**
- Use database unique constraint on `(user_id, tx_ref_type, tx_ref_id)`
- Or use `SELECT FOR UPDATE` / `BEGIN IMMEDIATE` to lock rows

---

## High Priority Bugs

### 4. Goroutine Leaks with Background() Context

**Location:** `internal/usecase/order_service.go:615, 619, 623, 639`

**Functions:**
- `executeResizeTask()`
- `ApproveOrder()`
- `RejectOrder()`
- `CancelOrder()`

**Description:**
Provisioning goroutines are spawned with `context.Background()`, ignoring the parent context's cancellation. If the HTTP request is cancelled, goroutines continue running, potentially causing:
- Duplicate provisioning attempts
- Resource leaks
- Ghost operations

**Vulnerable Code:**
```go
go s.executeResizeTask(context.Background(), *task)  // Ignores parent context!
go s.provisionOrder(order.ID)  // No context at all!
```

**Recommended Fix:**
```go
// Create a detached context with timeout for async operations
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
defer cancel()
go s.executeResizeTask(ctx, *task)
```

---

### 5. TOCTOU Race Conditions in Duplicate Order Checks

**Location:** `internal/usecase/order_service.go:1798, 1880, 1943, 1950`

**Functions:**
- `HasPendingRenewOrder()`
- `HasPendingResizeOrder()`
- `HasPendingChangeOrder()`

**Description:**
Time-of-check to time-of-use (TOCTOU) race between checking for pending operations and creating new orders. Between the check and order creation, another request could create a pending order.

**Vulnerable Code:**
```go
if pending, err := s.items.HasPendingRenewOrder(ctx, userID, vpsID); err != nil {
    return domain.Order{}, err
} else if pending {
    return domain.Order{}, fmt.Errorf("已有待处理续费订单，请先处理或撤销: %w", ErrConflict)
}
// ... later ...
if err := s.orders.CreateOrder(ctx, &order); err != nil {  // Race window!
```

**Recommended Fix:**
- Use database unique constraints on `(user_id, vps_id, status)` for pending orders
- Or use `INSERT ... ON CONFLICT` with proper error handling

---

## Medium Priority Bugs

### 6. Silent Failures in Batch Updates

**Location:** Multiple locations (20+ occurrences)

**Pattern:** `_ = s.items.UpdateOrderItemStatus(ctx, item.ID, newStatus)`

**Description:**
Errors when updating order item statuses are silently ignored throughout the codebase. If some items update successfully but others fail, the order state becomes inconsistent with its items.

**Affected Lines:**
- `order_service.go:333, 381, 432, 482, 484, 487, 494, 496, 574, 587, 601, 723, 732, 738, 739, 744, 748, 754, 759, 763, 769, 774, 778, 784, 789, 793, 799`

**Recommended Fix:**
```go
// Collect errors and handle them
var errs []error
for _, item := range items {
    if err := s.items.UpdateOrderItemStatus(ctx, item.ID, newStatus); err != nil {
        errs = append(errs, err)
    }
}
if len(errs) > 0 {
    // Log or handle the errors appropriately
}
```

---

### 7. Non-Atomic Order Creation (Fallback Path)

**Location:** `internal/usecase/order_service.go:150-163`

**Description:**
When the atomic creator interface isn't available, order creation, item creation, and cart clearing happen in separate non-atomic operations. If later operations fail, the system is left in an inconsistent state.

**Vulnerable Code:**
```go
} else {
    if err := s.orders.CreateOrder(ctx, &order); err != nil {
        return domain.Order{}, nil, err
    }
    for i := range orderItems {
        orderItems[i].OrderID = order.ID
    }
    if err := s.items.CreateOrderItems(ctx, orderItems); err != nil {
        return domain.Order{}, nil, err  // Order exists but has no items!
    }
    if err := s.cart.ClearCart(ctx, userID); err != nil {
        return domain.Order{}, nil, err  // Order+items exist but cart not cleared!
    }
}
```

**Impact:**
- Orphaned orders without items
- Users able to resubmit the same cart
- Data inconsistency

**Recommended Fix:**
Always use the atomic creator interface, or implement proper transaction handling.

---

### 8. Payment Status Not Atomic with Order Status

**Location:** `internal/usecase/order_service.go:326-334`

**Description:**
Payment creation and order status update happen in separate non-atomic operations. If the order update fails, the payment exists but the order status doesn't reflect the payment.

**Vulnerable Code:**
```go
if err := s.payments.CreatePayment(ctx, &payment); err != nil {
    return domain.OrderPayment{}, err
}
order.Status = domain.OrderStatusPendingReview
order.PendingReason = ""
if err := s.orders.UpdateOrderMeta(ctx, order); err != nil {
    return domain.OrderPayment{}, err  // Payment created but order not updated!
}
```

**Recommended Fix:**
Implement a transactional payment creation method that also updates the order atomically.

---

### 9. Missing Idempotency in Wallet Refund Creation

**Location:** `internal/usecase/order_service.go:1287-1295`

**Description:**
After passing the duplicate check, `createAndApproveWalletRefund` creates a wallet order and adjusts balance without proper idempotency protection.

**Vulnerable Code:**
```go
if err := walletOrders.CreateWalletOrder(ctx, &order); err != nil {
    return err
}
if _, err := s.wallets.AdjustWalletBalance(ctx, userID, amount, "credit", txRefType, txRefID, fmt.Sprintf("refund wallet order %d", order.ID)); err != nil {
    return err  // Wallet order created but balance not adjusted!
}
```

**Impact:**
- Wallet order exists but no actual refund given
- User expects refund but doesn't receive it
- Manual reconciliation needed

**Recommended Fix:**
Implement proper compensation logic (rollback wallet order creation if balance adjustment fails).

---

### 10. Duplicate Payment Check Race Condition

**Location:** `internal/usecase/payment_service.go:193-200`

**Description:**
Multiple concurrent payment notifications with the same `trade_no` could pass the duplicate check and create duplicate payment records.

**Vulnerable Code:**
```go
if strings.TrimSpace(result.TradeNo) == "" || items[i].TradeNo == result.TradeNo || strings.TrimSpace(items[i].TradeNo) == "" {
    payment = items[i]
    break
}
```

**Recommended Fix:**
Use a unique constraint on `trade_no` in the database and handle `ErrConstraintUnique` properly.

---

## Lower Priority Issues

### 11. Order ID Enumeration Attack

**Location:** `internal/usecase/order_service.go:257-270`

**Description:**
The `GetOrder` function returns `ErrNotFound` before checking ownership, allowing attackers to enumerate valid order IDs.

**Vulnerable Code:**
```go
order, err := s.orders.GetOrder(ctx, orderID)
if err != nil {
    return domain.Order{}, nil, err  // Returns ErrNotFound, leaking info
}
if order.UserID != userID {
    return domain.Order{}, nil, ErrForbidden
}
```

**Impact:**
- Information disclosure about order existence
- Could be used for competitive intelligence

**Recommended Fix:**
Return a generic error message, or check ownership in the database query itself.

---

### 12. Missing Transaction Context Propagation

**Location:** Various repository methods

**Description:**
Some operations that should be transactional are not passing context properly or not using transactions where needed.

**Recommended Fix:**
Audit all multi-step operations and ensure proper transaction handling.

---

## Recommended Action Plan

### Phase 1: Immediate (Within 1 Week)
1. **Fix wallet balance race condition** - Implement atomic UPDATE with CHECK constraint
2. **Fix silent error in payment approval** - Properly handle and propagate errors
3. **Fix double refund vulnerability** - Add unique constraint or row locking

### Phase 2: High Priority (Within 2 Weeks)
4. **Fix goroutine leaks** - Implement proper context propagation for async operations
5. **Fix TOCTOU races** - Add database constraints for duplicate prevention

### Phase 3: Medium Priority (Within 1 Month)
6. **Fix silent batch update failures** - Implement proper error collection and handling
7. **Make order creation atomic** - Always use transactions
8. **Fix payment/order state consistency** - Implement transactional payment creation
9. **Fix wallet refund idempotency** - Add compensation logic

### Phase 4: Lower Priority (Within 2 Months)
10. **Prevent order enumeration** - Implement generic error messages
11. **Audit transaction handling** - Comprehensive review of all transactional code
12. **Add integration tests** - Specifically for concurrent scenarios

---

## Additional Recommendations

### 1. Database Constraints
Add unique constraints to prevent race conditions at the database level:
- `UNIQUE(user_id, tx_ref_type, tx_ref_id)` on wallet_transactions
- `UNIQUE(trade_no)` on payments
- `UNIQUE(user_id, vps_id, status)` on order_items (for pending operations)

### 2. Monitoring & Alerting
- Add metrics for failed operations
- Alert on payment/order state mismatches
- Monitor for duplicate transaction attempts

### 3. Testing
- Add concurrent load tests for all financial operations
- Implement property-based testing for state transitions
- Add integration tests for race conditions

### 4. Code Review Practices
- Never ignore errors with `_` in critical paths
- Always use transactions for multi-step state changes
- Prefer atomic operations over check-then-update patterns

---

## Conclusion

The identified bugs primarily stem from:
1. **Insufficient transaction handling** in multi-step operations
2. **Race conditions** from check-then-update patterns
3. **Silent error handling** that ignores critical failures
4. **Missing idempotency** in financial operations

Addressing these issues requires both immediate fixes to critical vulnerabilities and systemic improvements to the codebase's approach to concurrency and error handling.

---

*Report generated by Claude Code*
*For questions or clarifications, please refer to the specific file locations mentioned above.*
