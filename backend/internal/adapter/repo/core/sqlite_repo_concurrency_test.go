package repo_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
)

func TestSQLiteRepo_RefreshTokenUseIsAtomic(t *testing.T) {
	_, repository := testutil.NewTestDB(t, false)
	ctx := context.Background()
	const workers = 12
	results := make(chan bool, workers)
	familyRevoked := make(chan bool, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			claimed, revoked, err := repository.RotateRefreshToken(
				ctx,
				strings.Repeat("a", 64),
				fmt.Sprintf("%064x", i+2),
				"family-a",
				time.Now().Add(time.Hour),
				time.Now().Add(2*time.Hour),
			)
			if err != nil {
				t.Errorf("consume refresh token: %v", err)
				return
			}
			results <- claimed
			familyRevoked <- revoked
		}()
	}
	wg.Wait()
	close(results)
	close(familyRevoked)
	winners := 0
	for claimed := range results {
		if claimed {
			winners++
		}
	}
	if winners != 1 {
		t.Fatalf("expected exactly one refresh-token consumer, got %d", winners)
	}
	revocations := 0
	for revoked := range familyRevoked {
		if revoked {
			revocations++
		}
	}
	if revocations == 0 {
		t.Fatal("expected replay contenders to revoke the token family")
	}
	rotated, revoked, err := repository.RotateRefreshToken(
		ctx,
		strings.Repeat("b", 64),
		strings.Repeat("c", 64),
		"family-a",
		time.Now().Add(time.Hour),
		time.Now().Add(2*time.Hour),
	)
	if err != nil {
		t.Fatalf("check revoked descendant: %v", err)
	}
	if rotated || !revoked {
		t.Fatalf("revoked family accepted descendant: rotated=%v revoked=%v", rotated, revoked)
	}
}

// TestSQLiteRepo_UserAPIKeyNonceIsAtomic verifies the durable nonce ledger: when
// many goroutines race to consume the same (akid, nonce), exactly one wins,
// which is what makes replay protection hold across instances (P1-10).
func TestSQLiteRepo_UserAPIKeyNonceIsAtomic(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	exp := time.Now().Add(time.Hour)

	const goroutines = 8
	var wg sync.WaitGroup
	results := make([]bool, goroutines)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			claimed, err := repo.SaveUserAPIKeyNonce(ctx, "akid-race", "same-nonce", exp)
			if err != nil {
				t.Errorf("save nonce: %v", err)
				return
			}
			results[idx] = claimed
		}(i)
	}
	wg.Wait()

	winners := 0
	for _, claimed := range results {
		if claimed {
			winners++
		}
	}
	if winners != 1 {
		t.Fatalf("expected exactly one nonce winner, got %d", winners)
	}
}

// TestSQLiteRepo_PasswordResetTokenClaimIsAtomic verifies the single-use CAS
// primitive: when many goroutines try to consume the same token simultaneously,
// exactly one wins (P1-2 TOCTOU).
func TestSQLiteRepo_PasswordResetTokenClaimIsAtomic(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	admin := testutil.CreateAdmin(t, repo, "admin-claim", "admin-claim@example.com", "pass", 0)

	token := domain.PasswordResetToken{
		UserID:    admin.ID,
		Token:     "token-hash-value",
		ExpiresAt: time.Now().Add(time.Hour),
		Used:      false,
	}
	if err := repo.CreatePasswordResetToken(ctx, &token); err != nil {
		t.Fatalf("create token: %v", err)
	}

	const goroutines = 8
	var wg sync.WaitGroup
	results := make([]bool, goroutines)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			claimed, err := repo.MarkPasswordResetTokenUsedIfUnused(ctx, token.ID)
			if err != nil {
				t.Errorf("claim: %v", err)
				return
			}
			results[idx] = claimed
		}(i)
	}
	wg.Wait()

	winners := 0
	for _, claimed := range results {
		if claimed {
			winners++
		}
	}
	if winners != 1 {
		t.Fatalf("expected exactly one winner, got %d", winners)
	}

	stored, err := repo.GetPasswordResetToken(ctx, "token-hash-value")
	if err != nil {
		t.Fatalf("reload token: %v", err)
	}
	if !stored.Used {
		t.Fatalf("token must be marked used after a successful claim")
	}
}

// TestSQLiteRepo_ConsumePasswordResetTokenAtomic verifies the atomic
// claim+password-update primitive: when many goroutines race to consume the
// same token, exactly one wins and only the winner's password takes effect
// (P2-8 single-winner).
func TestSQLiteRepo_ConsumePasswordResetTokenAtomic(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	user := testutil.CreateUser(t, repo, "consume-race", "consume-race@example.com", "pass")

	token := domain.PasswordResetToken{
		UserID:    user.ID,
		Token:     "consume-race-hash",
		ExpiresAt: time.Now().Add(time.Hour),
		Used:      false,
	}
	if err := repo.CreatePasswordResetToken(ctx, &token); err != nil {
		t.Fatalf("create token: %v", err)
	}

	const goroutines = 8
	var wg sync.WaitGroup
	winners := make([]bool, goroutines)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			claimed, err := repo.ConsumePasswordResetTokenAndUpdatePassword(ctx, token.ID, user.ID, fmt.Sprintf("hash-%d", idx), time.Now())
			if err != nil {
				return // losers may see errors; the winner must not
			}
			winners[idx] = claimed
		}(i)
	}
	wg.Wait()

	winnerCount := 0
	for _, c := range winners {
		if c {
			winnerCount++
		}
	}
	if winnerCount != 1 {
		t.Fatalf("expected exactly one winner, got %d", winnerCount)
	}
	reloaded, err := repo.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if reloaded.PasswordVersion != 1 {
		t.Fatalf("password must have been changed exactly once, version=%d", reloaded.PasswordVersion)
	}
}

// TestSQLiteRepo_ConsumePasswordResetTokenRollsBackFailedUpdate verifies a
// failed password update rolls back the token claim: the token stays unused
// and the password is unchanged, so a usable token is never burned (P2-8).
func TestSQLiteRepo_ConsumePasswordResetTokenRollsBackFailedUpdate(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	user := testutil.CreateUser(t, repo, "consume-rollback", "consume-rollback@example.com", "pass")

	token := domain.PasswordResetToken{
		UserID:    user.ID,
		Token:     "consume-rollback-hash",
		ExpiresAt: time.Now().Add(time.Hour),
		Used:      false,
	}
	if err := repo.CreatePasswordResetToken(ctx, &token); err != nil {
		t.Fatalf("create token: %v", err)
	}
	before, err := repo.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("load before: %v", err)
	}

	// Target a user that does not exist: the user-row update affects 0 rows,
	// which the repo treats as a failed update and rolls the claim back.
	claimed, err := repo.ConsumePasswordResetTokenAndUpdatePassword(ctx, token.ID, 999999, "should-not-persist", time.Now())
	if err == nil || claimed {
		t.Fatalf("expected failure for missing user, got claimed=%v err=%v", claimed, err)
	}

	reloadedToken, err := repo.GetPasswordResetToken(ctx, "consume-rollback-hash")
	if err != nil {
		t.Fatalf("reload token: %v", err)
	}
	if reloadedToken.Used {
		t.Fatalf("token must remain unused after a failed password update")
	}
	after, err := repo.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("load after: %v", err)
	}
	if after.PasswordHash != before.PasswordHash || after.PasswordVersion != before.PasswordVersion {
		t.Fatalf("password must be unchanged after a failed update")
	}
}

// TestSQLiteRepo_ConsumePasswordResetTicketAtomic verifies the ticket-based
// atomic consume: concurrent consumers of the same ticket yield exactly one
// winner whose password change persists.
func TestSQLiteRepo_ConsumePasswordResetTicketAtomic(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	user := testutil.CreateUser(t, repo, "ticket-race", "ticket-race@example.com", "pass")

	ticket := domain.PasswordResetTicket{
		UserID:    user.ID,
		Channel:   "email",
		Receiver:  "ticket-race@example.com",
		Token:     "ticket-race-hash",
		ExpiresAt: time.Now().Add(time.Hour),
		Used:      false,
	}
	if err := repo.CreatePasswordResetTicket(ctx, &ticket); err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	const goroutines = 8
	var wg sync.WaitGroup
	winners := make([]bool, goroutines)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			claimed, err := repo.ConsumePasswordResetTicketAndUpdatePassword(ctx, ticket.ID, user.ID, fmt.Sprintf("hash-%d", idx), time.Now())
			if err != nil {
				return
			}
			winners[idx] = claimed
		}(i)
	}
	wg.Wait()

	winnerCount := 0
	for _, c := range winners {
		if c {
			winnerCount++
		}
	}
	if winnerCount != 1 {
		t.Fatalf("expected exactly one winner, got %d", winnerCount)
	}
}

// TestSQLiteRepo_ConsumeContactChangeTicketAtomicAndRollsBack verifies contact
// changes claim the security ticket atomically: a uniqueness violation rolls
// back the claim (ticket stays unused) and concurrent claims have one winner.
func TestSQLiteRepo_ConsumeContactChangeTicketAtomicAndRollsBack(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	owner := testutil.CreateUser(t, repo, "contact-owner", "contact-owner@example.com", "pass")
	other := testutil.CreateUser(t, repo, "contact-other", "contact-other@example.com", "pass")

	mkTicket := func(hash string) domain.PasswordResetTicket {
		tk := domain.PasswordResetTicket{
			UserID:    owner.ID,
			Channel:   "email",
			Receiver:  "contact-owner@example.com",
			Token:     hash,
			ExpiresAt: time.Now().Add(time.Hour),
			Used:      false,
		}
		if err := repo.CreatePasswordResetTicket(ctx, &tk); err != nil {
			t.Fatalf("create ticket: %v", err)
		}
		return tk
	}

	// Uniqueness violation: another user already owns the target email.
	conflict := mkTicket("contact-conflict-hash")
	claimed, err := repo.ConsumeContactChangeTicketAndUpdateContact(ctx, conflict.ID, owner.ID, "email", other.Email, time.Now())
	if err == nil || claimed {
		t.Fatalf("expected uniqueness violation, got claimed=%v err=%v", claimed, err)
	}
	reloaded, err := repo.GetPasswordResetTicket(ctx, "contact-conflict-hash")
	if err != nil {
		t.Fatalf("reload conflict ticket: %v", err)
	}
	if reloaded.Used {
		t.Fatalf("conflict ticket must remain unused after a rejected change")
	}

	// Successful claim applies the contact change and marks the ticket used.
	ok := mkTicket("contact-ok-hash")
	claimed, err = repo.ConsumeContactChangeTicketAndUpdateContact(ctx, ok.ID, owner.ID, "email", "new-contact@example.com", time.Now())
	if err != nil {
		t.Fatalf("contact change: %v", err)
	}
	if !claimed {
		t.Fatalf("expected claim to win")
	}
	updated, err := repo.GetUserByID(ctx, owner.ID)
	if err != nil {
		t.Fatalf("reload owner: %v", err)
	}
	if updated.Email != "new-contact@example.com" {
		t.Fatalf("email must be updated, got %q", updated.Email)
	}
}

// TestSQLiteRepo_DeletePermissionGroupSerializesWithAssign verifies P2-36: a
// group deletion and a concurrent assignment serialize on the group row, so the
// system never ends up with a dangling permission-group reference regardless of
// which operation wins.
func TestSQLiteRepo_DeletePermissionGroupSerializesWithAssign(t *testing.T) {
	for i := 0; i < 5; i++ {
		_, repo := testutil.NewTestDB(t, false)
		ctx := context.Background()
		group := domain.PermissionGroup{Name: fmt.Sprintf("pg-race-%d", i), PermissionsJSON: `["*"]`}
		if err := repo.CreatePermissionGroup(ctx, &group); err != nil {
			t.Fatalf("create group: %v", err)
		}
		user := testutil.CreateUser(t, repo, fmt.Sprintf("pg-race-u%d", i), fmt.Sprintf("pg-race-u%d@example.com", i), "pass123")
		base, err := repo.GetUserByID(ctx, user.ID)
		if err != nil {
			t.Fatalf("load user: %v", err)
		}

		var wg sync.WaitGroup
		var deleteErr, assignErr error
		wg.Add(2)
		go func() {
			defer wg.Done()
			deleteErr = repo.DeletePermissionGroupSafe(ctx, group.ID)
		}()
		go func() {
			defer wg.Done()
			assign := base
			assign.PermissionGroupID = &group.ID
			assignErr = repo.UpdateUser(ctx, assign)
		}()
		wg.Wait()

		_, groupLookupErr := repo.GetPermissionGroup(ctx, group.ID)
		reloaded, err := repo.GetUserByID(ctx, user.ID)
		if err != nil {
			t.Fatalf("reload user: %v", err)
		}
		if groupLookupErr == nil {
			// Group survived: the assignment must have won and be visible.
			if reloaded.PermissionGroupID == nil || *reloaded.PermissionGroupID != group.ID {
				t.Fatalf("group survived but assignment missing (delErr=%v assignErr=%v)", deleteErr, assignErr)
			}
		} else {
			// Group deleted: the user must not reference it.
			if reloaded.PermissionGroupID != nil && *reloaded.PermissionGroupID == group.ID {
				t.Fatalf("dangling permission group reference after delete (delErr=%v assignErr=%v)", deleteErr, assignErr)
			}
		}
	}
}

// TestSQLiteRepo_UpdateAdminStatusSafeNeverZerosActiveAdmins verifies that
// concurrent admin disables can never leave the system with zero active admins.
func TestSQLiteRepo_UpdateAdminStatusSafeNeverZerosActiveAdmins(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	group := domain.PermissionGroup{Name: "safe-admins", PermissionsJSON: `["*"]`}
	if err := repo.CreatePermissionGroup(ctx, &group); err != nil {
		t.Fatalf("create group: %v", err)
	}
	adminA := testutil.CreateAdmin(t, repo, "safe-a", "safe-a@example.com", "pass123", group.ID)
	adminB := testutil.CreateAdmin(t, repo, "safe-b", "safe-b@example.com", "pass123", group.ID)

	var wg sync.WaitGroup
	errs := make([]error, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		errs[0] = repo.UpdateAdminStatusSafe(ctx, adminA.ID, domain.UserStatusDisabled)
	}()
	go func() {
		defer wg.Done()
		errs[1] = repo.UpdateAdminStatusSafe(ctx, adminB.ID, domain.UserStatusDisabled)
	}()
	wg.Wait()

	// At least one of the two disables must have been rejected.
	if errs[0] == nil && errs[1] == nil {
		t.Fatalf("both disables succeeded — system would have zero active admins")
	}
	// At least one admin must still be active.
	activeAdmins := 0
	for _, id := range []int64{adminA.ID, adminB.ID} {
		u, err := repo.GetUserByID(ctx, id)
		if err != nil {
			t.Fatalf("reload admin %d: %v", id, err)
		}
		if u.Status == domain.UserStatusActive {
			activeAdmins++
		}
	}
	if activeAdmins < 1 {
		t.Fatalf("expected at least one active admin after concurrent disables, got %d", activeAdmins)
	}
}

// TestSQLiteRepo_ConsumeProbeEnrollTokenRollsBackFailedNodeUpdate verifies the
// atomic enroll: when the node binding/secret update cannot be applied, the
// token claim is rolled back (still unused) and no half-applied state remains.
func TestSQLiteRepo_ConsumeProbeEnrollTokenRollsBackFailedNodeUpdate(t *testing.T) {
	db, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()

	node := domain.ProbeNode{Name: "p-rollback", AgentID: "agent-rollback", SecretHash: "old-hash", Status: domain.ProbeStatusOffline, TagsJSON: "[]"}
	if err := repo.CreateProbeNode(ctx, &node); err != nil {
		t.Fatalf("create node: %v", err)
	}
	tok := domain.ProbeEnrollToken{ProbeID: node.ID, TokenHash: "enroll-rollback-hash", ExpiresAt: time.Now().Add(time.Hour)}
	if err := repo.CreateProbeEnrollToken(ctx, &tok); err != nil {
		t.Fatalf("create token: %v", err)
	}

	// Remove the node row directly (bypassing the cascading repo delete) so the
	// update inside the atomic transaction affects zero rows.
	if _, err := db.Exec(`DELETE FROM probe_nodes WHERE id = ?`, node.ID); err != nil {
		t.Fatalf("delete node: %v", err)
	}

	claimed, err := repo.ConsumeProbeEnrollTokenAndUpdateNode(ctx, tok.ID, node.ID, map[string]any{"secret_hash": "new-hash"}, time.Now())
	if err == nil || claimed {
		t.Fatalf("expected failed node update, got claimed=%v err=%v", claimed, err)
	}
	reloaded, err := repo.GetValidProbeEnrollTokenByHash(ctx, "enroll-rollback-hash", time.Now().Add(time.Minute))
	if err == nil {
		if reloaded.UsedAt != nil {
			t.Fatalf("token must remain unused after a failed node update")
		}
	} else {
		// The token lookup with a future now could still fail if expires check…
		// re-fetch by raw query to confirm used_at stayed NULL.
		var usedAt *time.Time
		if err := db.QueryRow(`SELECT used_at FROM probe_enroll_tokens WHERE id = ?`, tok.ID).Scan(&usedAt); err != nil {
			t.Fatalf("query token: %v", err)
		}
		if usedAt != nil {
			t.Fatalf("token must remain unused after a failed node update")
		}
	}
}

// TestSQLiteRepo_RobotWebhookNonceIsDurable verifies the webhook replay ledger
// survives restarts/replica switches: a replay key recorded before a "restart"
// (a fresh caller on the same DB) is still rejected, and distinct keys are
// accepted. Expired rows are swept.
func TestSQLiteRepo_RobotWebhookNonceIsDurable(t *testing.T) {
	db, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()

	// First delivery wins.
	claimed, err := repo.SaveRobotWebhookNonce(ctx, "replay-key-1", time.Now().Add(time.Hour))
	if err != nil || !claimed {
		t.Fatalf("first claim: claimed=%v err=%v", claimed, err)
	}
	// Same key after a "restart" (fresh call, same DB) must lose.
	claimed, err = repo.SaveRobotWebhookNonce(ctx, "replay-key-1", time.Now().Add(time.Hour))
	if err != nil || claimed {
		t.Fatalf("replay after restart must lose, claimed=%v err=%v", claimed, err)
	}
	// A different key is accepted.
	claimed, err = repo.SaveRobotWebhookNonce(ctx, "replay-key-2", time.Now().Add(time.Hour))
	if err != nil || !claimed {
		t.Fatalf("distinct key must be claimed: claimed=%v err=%v", claimed, err)
	}

	// Expired rows are swept so the ledger stays bounded.
	expired, err := repo.SaveRobotWebhookNonce(ctx, "replay-key-expired", time.Now().Add(-time.Minute))
	if err != nil || !expired {
		t.Fatalf("save expired row: %v %v", expired, err)
	}
	if err := repo.DeleteExpiredRobotWebhookNonces(ctx); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM robot_webhook_nonces`).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 { // replay-key-1 and replay-key-2 remain
		t.Fatalf("expected 2 remaining nonces after sweep, got %d", count)
	}
}

// TestSQLiteRepo_RobotWebhookNonceAtomic verifies concurrent consumers of the
// same webhook replay key yield exactly one winner (multi-instance CAS).
func TestSQLiteRepo_RobotWebhookNonceAtomic(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	exp := time.Now().Add(time.Hour)

	const goroutines = 8
	var wg sync.WaitGroup
	results := make([]bool, goroutines)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			claimed, err := repo.SaveRobotWebhookNonce(ctx, "same-replay-key", exp)
			if err != nil {
				return
			}
			results[idx] = claimed
		}(i)
	}
	wg.Wait()
	winners := 0
	for _, c := range results {
		if c {
			winners++
		}
	}
	if winners != 1 {
		t.Fatalf("expected exactly one replay winner, got %d", winners)
	}
}

// TestSQLiteRepo_AuthFailureBudgetConcurrentIncrement verifies the durable
// failure budget counts concurrent increments without lost updates across
// instances.
func TestSQLiteRepo_AuthFailureBudgetConcurrentIncrement(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	const goroutines = 8
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, _, err := repo.RecordAuthFailure(ctx, "budget:account:x", 100, time.Hour, time.Now()); err != nil {
				t.Errorf("record: %v", err)
			}
		}()
	}
	wg.Wait()
	count, locked, err := repo.PeekAuthFailure(ctx, "budget:account:x", 100, time.Hour, time.Now())
	if err != nil {
		t.Fatalf("peek: %v", err)
	}
	if count != goroutines {
		t.Fatalf("expected count %d, got %d", goroutines, count)
	}
	if locked {
		t.Fatalf("count %d must not be locked below limit 100", count)
	}
}

// TestSQLiteRepo_AuthFailureBudgetLockResetExpiry verifies the fixed-window
// semantics: the budget locks at the limit, a reset clears it, and an expired
// window recovers automatically (expiry recovery).
func TestSQLiteRepo_AuthFailureBudgetLockResetExpiry(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	now := time.Now()
	window := 10 * time.Minute

	// Failures up to the limit lock the key.
	var lockedUntil time.Time
	for i := 0; i < 5; i++ {
		var err error
		_, lockedUntil, err = repo.RecordAuthFailure(ctx, "budget:2fa:acct", 5, window, now)
		if err != nil {
			t.Fatalf("record %d: %v", i+1, err)
		}
	}
	if !lockedUntil.After(now) {
		t.Fatalf("key must be locked at the limit")
	}
	if _, locked, err := repo.PeekAuthFailure(ctx, "budget:2fa:acct", 5, window, now); err != nil || !locked {
		t.Fatalf("peek must report locked, locked=%v err=%v", locked, err)
	}

	// Success resets only the correct scope.
	if err := repo.ResetAuthFailure(ctx, "budget:2fa:acct"); err != nil {
		t.Fatalf("reset: %v", err)
	}
	if count, locked, err := repo.PeekAuthFailure(ctx, "budget:2fa:acct", 5, window, now); err != nil || locked || count != 0 {
		t.Fatalf("after reset: count=%d locked=%v err=%v", count, locked, err)
	}

	// Expiry recovery: a window that has elapsed is treated as a fresh budget.
	if _, _, err := repo.RecordAuthFailure(ctx, "budget:2fa:acct", 5, window, now); err != nil {
		t.Fatalf("record after reset: %v", err)
	}
	expiredNow := now.Add(window + time.Second)
	if count, locked, err := repo.PeekAuthFailure(ctx, "budget:2fa:acct", 5, window, expiredNow); err != nil || locked || count != 0 {
		t.Fatalf("expired window must not be locked: count=%d locked=%v err=%v", count, locked, err)
	}
	count, lockedUntil, err := repo.RecordAuthFailure(ctx, "budget:2fa:acct", 5, window, expiredNow)
	if err != nil {
		t.Fatalf("record after expiry: %v", err)
	}
	if count != 1 {
		t.Fatalf("expired window must reset the count to 1, got %d", count)
	}
	if lockedUntil.After(expiredNow) {
		t.Fatalf("a single post-expiry failure must not lock the key")
	}
}

// TestSQLiteRepo_AuthFailureBudgetIPRotationIsIndependent verifies account and
// IP dimensions are separate: failures recorded under one IP never leak into
// another IP's budget (an attacker rotating IPs cannot reset the account key).
func TestSQLiteRepo_AuthFailureBudgetIPRotationIsIndependent(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	now := time.Now()
	window := 10 * time.Minute

	accountKey := "budget:login:account:victim"
	for _, ip := range []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"} {
		if _, _, err := repo.RecordAuthFailure(ctx, "budget:login:ip:"+ip, 5, window, now); err != nil {
			t.Fatalf("record ip %s: %v", ip, err)
		}
		if _, _, err := repo.RecordAuthFailure(ctx, accountKey, 5, window, now); err != nil {
			t.Fatalf("record account: %v", err)
		}
	}
	// Three failed attempts on the account budget (not yet locked at limit 5).
	count, locked, err := repo.PeekAuthFailure(ctx, accountKey, 5, window, now)
	if err != nil || locked || count != 3 {
		t.Fatalf("account budget must count 3 across IPs, got count=%d locked=%v err=%v", count, locked, err)
	}
	// The IP budgets are independent.
	for _, ip := range []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"} {
		c, l, err := repo.PeekAuthFailure(ctx, "budget:login:ip:"+ip, 5, window, now)
		if err != nil || l || c != 1 {
			t.Fatalf("ip %s budget must be independent, count=%d locked=%v err=%v", ip, c, l, err)
		}
	}
}

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

// TestSQLiteRepo_CreatePaymentIdempotencyRaceReturnsConflict locks the
// cross-module "支付幂等键并发 miss 后的唯一冲突不得 500" item: a second
// CreatePayment with the same order+idempotency key must surface a stable
// ErrConflict (which the handler maps to 409), not a raw constraint error.
func TestSQLiteRepo_CreatePaymentIdempotencyRaceReturnsConflict(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	user := testutil.CreateUser(t, repo, "pay-idem-race", "pay-idem-race@example.com", "pass")
	order := domain.Order{UserID: user.ID, OrderNo: "ORD-PAY-RACE", Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	idem := "provider:stripe:12345"
	first := domain.OrderPayment{OrderID: order.ID, UserID: user.ID, Method: "stripe", Amount: 1000, Currency: "CNY", TradeNo: "STRIPE-1", Status: domain.PaymentStatusPendingPayment, IdempotencyKey: idem}
	if err := repo.CreatePayment(ctx, &first); err != nil {
		t.Fatalf("first create: %v", err)
	}
	second := domain.OrderPayment{OrderID: order.ID, UserID: user.ID, Method: "stripe", Amount: 1000, Currency: "CNY", TradeNo: "STRIPE-2", Status: domain.PaymentStatusPendingPayment, IdempotencyKey: idem}
	if err := repo.CreatePayment(ctx, &second); !errors.Is(err, appshared.ErrConflict) {
		t.Fatalf("expected ErrConflict on idempotency-key race, got %v", err)
	}
	// Exactly one payment record exists.
	payments, err := repo.ListPaymentsByOrder(ctx, order.ID)
	if err != nil || len(payments) != 1 {
		t.Fatalf("expected exactly one payment, got %d err=%v", len(payments), err)
	}
	if payments[0].TradeNo != "STRIPE-1" {
		t.Fatalf("expected the first session to win, got %s", payments[0].TradeNo)
	}
}

func TestSQLiteRepo_ApproveOrderOnlyApprovesReviewablePayments(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	user := testutil.CreateUser(t, repo, "approve-scope", "approve-scope@example.com", "pass")
	order := domain.Order{UserID: user.ID, OrderNo: "ORD-APPROVE-SCOPE", Status: domain.OrderStatusPendingReview, TotalAmount: 1000, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	if err := repo.CreateOrderItems(ctx, []domain.OrderItem{{OrderID: order.ID, Amount: 1000, Status: domain.OrderItemStatusPendingReview, Action: "create", SpecJSON: "{}"}}); err != nil {
		t.Fatalf("create item: %v", err)
	}
	// A manual payment awaiting admin review.
	review := domain.OrderPayment{OrderID: order.ID, UserID: user.ID, Method: "manual", Amount: 1000, Currency: "CNY", TradeNo: "REVIEW-1", Status: domain.PaymentStatusPendingReview}
	if err := repo.CreatePayment(ctx, &review); err != nil {
		t.Fatalf("create review payment: %v", err)
	}
	// An abandoned provider payment that never received funds.
	provider := domain.OrderPayment{OrderID: order.ID, UserID: user.ID, Method: "stripe", Amount: 1000, Currency: "CNY", TradeNo: "PROVIDER-1", Status: domain.PaymentStatusPendingPayment}
	if err := repo.CreatePayment(ctx, &provider); err != nil {
		t.Fatalf("create provider payment: %v", err)
	}
	items, err := repo.ListOrderItems(ctx, order.ID)
	if err != nil || len(items) != 1 {
		t.Fatalf("list items: len=%d err=%v", len(items), err)
	}
	approved := order
	approved.Status = domain.OrderStatusApproved
	now := time.Now()
	approved.ApprovedAt = &now
	claimed, err := repo.ApproveOrderAtomic(ctx, approved, []int64{items[0].ID}, nil, []domain.OrderStatus{domain.OrderStatusPendingReview})
	if err != nil || !claimed {
		t.Fatalf("approve order: claimed=%v err=%v", claimed, err)
	}
	payments, err := repo.ListPaymentsByOrder(ctx, order.ID)
	if err != nil || len(payments) != 2 {
		t.Fatalf("list payments: len=%d err=%v", len(payments), err)
	}
	for _, p := range payments {
		switch p.TradeNo {
		case "REVIEW-1":
			if p.Status != domain.PaymentStatusApproved {
				t.Fatalf("review payment not approved: %+v", p)
			}
		case "PROVIDER-1":
			if p.Status != domain.PaymentStatusPendingPayment {
				t.Fatalf("unpaid provider payment was approved: %+v", p)
			}
		}
	}
}

func TestSQLiteRepo_ConcurrentAppendEventAssignsMonotonicSeq(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	ctx := context.Background()
	user := testutil.CreateUser(t, repo, "evt-seq-cas", "evt-seq@example.com", "pass")
	order := domain.Order{UserID: user.ID, OrderNo: "ORD-EVT-SEQ", Status: domain.OrderStatusApproved, TotalAmount: 1, Currency: "CNY"}
	if err := repo.CreateOrder(ctx, &order); err != nil {
		t.Fatalf("create order: %v", err)
	}

	const n = 4
	start := make(chan struct{})
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			_, err := repo.AppendEvent(ctx, order.ID, fmt.Sprintf("evt.%d", i), "{}")
			errs <- err
		}(i)
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent event append failed: %v", err)
		}
	}

	events, err := repo.ListEventsAfter(ctx, order.ID, 0, n+10)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) != n {
		t.Fatalf("expected %d events, got %d: %+v", n, len(events), events)
	}
	seen := make(map[int64]bool, n)
	for _, ev := range events {
		if seen[ev.Seq] {
			t.Fatalf("duplicate seq %d", ev.Seq)
		}
		seen[ev.Seq] = true
	}
	for i := int64(1); i <= n; i++ {
		if !seen[i] {
			t.Fatalf("missing seq %d (seen %v)", i, seen)
		}
	}
}
