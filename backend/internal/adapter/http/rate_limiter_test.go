package http

import (
	"fmt"
	"testing"
	"time"
)

func TestRateLimiter_AllowEnforcesWindow(t *testing.T) {
	r := newRateLimiter()
	if !r.Allow("k", 2, time.Hour) {
		t.Fatalf("first request should be allowed")
	}
	if !r.Allow("k", 2, time.Hour) {
		t.Fatalf("second request should be allowed")
	}
	if r.Allow("k", 2, time.Hour) {
		t.Fatalf("third request within window should be rejected")
	}
}

// H-2: the limiter must keep its key map bounded. Previously a key was only
// pruned when hit again, so an attacker generating unique keys could grow the
// map without limit until the process ran out of memory.
func TestRateLimiter_BoundsKeyCount(t *testing.T) {
	const maxKeys = 64
	r := &rateLimiter{m: map[string][]time.Time{}, maxKeys: maxKeys}
	for i := 0; i < 10000; i++ {
		r.Allow(fmt.Sprintf("key-%d", i), 10, time.Minute)
	}
	if got := len(r.m); got > maxKeys {
		t.Fatalf("rate limiter map exceeded bound: got %d keys, want <= %d", got, maxKeys)
	}
}

func TestRateLimiter_ExpiredKeysStillPrunedPerKey(t *testing.T) {
	r := &rateLimiter{m: map[string][]time.Time{}, maxKeys: 100}
	// First hit is old enough to fall outside the window; Allow must not count it.
	old := time.Now().Add(-2 * time.Minute)
	r.m["k"] = []time.Time{old}
	if !r.Allow("k", 1, time.Minute) {
		t.Fatalf("expired hit should not count against the limit")
	}
}
