package domain

import "testing"

// TestWalletTransactionCarriesIdempotencyKey asserts the domain model exposes
// the idempotency key that wallet_transactions already persists
// (idempotency_key column + unique index idx_wallet_transactions_idempotency_key).
// Repo AdjustWalletBalanceIdempotent relies on that key to make recharge/payment
// callbacks idempotent, so the domain model — the authority for the domain —
// must be able to carry it; otherwise reporting/audit code reading a
// WalletTransaction can never tell an idempotent replay from a fresh write.
func TestWalletTransactionCarriesIdempotencyKey(t *testing.T) {
	var tx WalletTransaction
	const key = "recharge-callback-abc-123"
	tx.IdempotencyKey = key
	if tx.IdempotencyKey != key {
		t.Fatalf("WalletTransaction.IdempotencyKey not carried: %+v", tx)
	}
}
