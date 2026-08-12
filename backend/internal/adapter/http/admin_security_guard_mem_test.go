package http

import (
	"fmt"
	"testing"
	"time"
)

func TestLoginCooldownGuard_RegisterFailureLocksAtThreshold(t *testing.T) {
	g := newLoginCooldownGuard()
	now := time.Now()
	for i := 0; i < 4; i++ {
		if cooling, _, _ := g.RegisterFailure("admin", 5, time.Minute, now); cooling {
			t.Fatalf("unexpected lock before threshold at attempt %d", i+1)
		}
	}
	cooling, lockedUntil, _ := g.RegisterFailure("admin", 5, time.Minute, now)
	if !cooling || !lockedUntil.After(now) {
		t.Fatalf("expected lock at the 5th failure, cooling=%v locked=%v", cooling, lockedUntil)
	}
}

// M-7: empty/whitespace keys must not silently bypass the guard. A failure on
// an empty key has to count toward a shared bucket instead of being ignored.
func TestLoginCooldownGuard_EmptyKeyIsBucketed(t *testing.T) {
	g := newLoginCooldownGuard()
	now := time.Now()
	var cooling bool
	var lockedUntil time.Time
	for i := 0; i < 10; i++ {
		cooling, lockedUntil, _ = g.RegisterFailure("   ", 10, time.Minute, now)
	}
	if !cooling || !lockedUntil.After(now) {
		t.Fatalf("expected the 10th empty-key failure to lock, cooling=%v locked=%v", cooling, lockedUntil)
	}
	if cooling, _ := g.IsCoolingDown("", now); !cooling {
		t.Fatalf("empty key and whitespace key should share the same bucket")
	}
}

// H-2: the guard state map must stay bounded even when an attacker rotates keys.
func TestLoginCooldownGuard_BoundsStateCount(t *testing.T) {
	const maxKeys = 64
	g := &loginCooldownGuard{states: map[string]loginCooldownState{}, maxKeys: maxKeys}
	now := time.Now()
	for i := 0; i < 10000; i++ {
		g.RegisterFailure(fmt.Sprintf("user-%d", i), 10, time.Minute, now)
	}
	if got := len(g.states); got > maxKeys {
		t.Fatalf("login cooldown guard map exceeded bound: got %d keys, want <= %d", got, maxKeys)
	}
}

// TestLoginCooldownGuard_CooldownExpires verifies the temporary lockout decays:
// after the cooldown elapses the key is usable again, so repeated 2FA/login
// failures can never permanently disable an account (P0-3).
func TestLoginCooldownGuard_CooldownExpires(t *testing.T) {
	g := newLoginCooldownGuard()
	now := time.Now()
	var cooling bool
	var lockedUntil time.Time
	for i := 0; i < 5; i++ {
		cooling, lockedUntil, _ = g.RegisterFailure("admin", 5, 10*time.Minute, now)
	}
	if !cooling {
		t.Fatalf("expected lock at threshold")
	}
	// Before expiry the key must remain locked.
	if c, _ := g.IsCoolingDown("admin", now.Add(9*time.Minute)); !c {
		t.Fatalf("expected still cooling down before expiry")
	}
	// After expiry the key must be usable again.
	if c, _ := g.IsCoolingDown("admin", now.Add(11*time.Minute)); c {
		t.Fatalf("expected cooldown to expire")
	}
	if c, _, _ := g.RegisterFailure("admin", 5, 10*time.Minute, now.Add(11*time.Minute)); c {
		t.Fatalf("expected first failure after expiry not to lock immediately")
	}
	_ = lockedUntil
}
