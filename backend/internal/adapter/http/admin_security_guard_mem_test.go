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

func TestConsecutiveFailureGuard_CountsPerKey(t *testing.T) {
	g := newConsecutiveFailureGuard()
	if c := g.RegisterFailure("admin"); c != 1 {
		t.Fatalf("expected count 1, got %d", c)
	}
	if c := g.RegisterFailure("admin"); c != 2 {
		t.Fatalf("expected count 2, got %d", c)
	}
	g.Reset("admin")
	if c := g.RegisterFailure("admin"); c != 1 {
		t.Fatalf("expected count 1 after reset, got %d", c)
	}
}

// M-7: empty-key failures must be counted, not silently dropped.
func TestConsecutiveFailureGuard_EmptyKeyIsCounted(t *testing.T) {
	g := newConsecutiveFailureGuard()
	for i := 1; i <= 3; i++ {
		if c := g.RegisterFailure(" \t"); c != i {
			t.Fatalf("expected count %d for empty-key bucket, got %d", i, c)
		}
	}
}

// H-2: the consecutive-failure map must stay bounded too.
func TestConsecutiveFailureGuard_BoundsCount(t *testing.T) {
	const maxKeys = 64
	g := &consecutiveFailureGuard{counts: map[string]int{}, maxKeys: maxKeys}
	for i := 0; i < 10000; i++ {
		g.RegisterFailure(fmt.Sprintf("user-%d", i))
	}
	if got := len(g.counts); got > maxKeys {
		t.Fatalf("consecutive failure guard map exceeded bound: got %d keys, want <= %d", got, maxKeys)
	}
}
