package http

import (
	"strings"
	"sync"
	"time"
)

// defaultSecurityGuardMaxKeys bounds how many distinct keys the login and 2FA
// guards track. Without a bound an attacker rotating keys (spoofed IPs, random
// usernames) could grow the state maps until the process runs out of memory.
const defaultSecurityGuardMaxKeys = 100000

// emptyAdminSecurityKeyBucket is the bucket used for empty/whitespace keys.
// The guards used to silently ignore such keys, so a caller that produced an
// empty key (e.g. a trimmed username) never had its failures counted. Routing
// them to a fixed bucket keeps the guards fail-closed.
const emptyAdminSecurityKeyBucket = "<empty>"

type loginCooldownState struct {
	failures    int
	lockedUntil time.Time
}

type loginCooldownGuard struct {
	mu      sync.Mutex
	states  map[string]loginCooldownState
	maxKeys int
}

func newLoginCooldownGuard() *loginCooldownGuard {
	return &loginCooldownGuard{states: map[string]loginCooldownState{}, maxKeys: defaultSecurityGuardMaxKeys}
}

func (g *loginCooldownGuard) IsCoolingDown(key string, now time.Time) (bool, time.Time) {
	key = normalizeAdminSecurityKey(key)
	if key == "" {
		return false, time.Time{}
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	state, ok := g.states[key]
	if !ok {
		return false, time.Time{}
	}
	if state.lockedUntil.After(now) {
		return true, state.lockedUntil
	}
	if state.failures == 0 {
		delete(g.states, key)
		return false, time.Time{}
	}
	state.lockedUntil = time.Time{}
	g.states[key] = state
	return false, time.Time{}
}

func (g *loginCooldownGuard) RegisterFailure(key string, threshold int, cooldown time.Duration, now time.Time) (bool, time.Time, int) {
	key = normalizeAdminSecurityKey(key)
	if key == "" {
		return false, time.Time{}, 0
	}
	if threshold <= 0 || cooldown <= 0 {
		return false, time.Time{}, 0
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	state := g.states[key]
	if state.lockedUntil.After(now) {
		return true, state.lockedUntil, state.failures
	}
	if !state.lockedUntil.IsZero() {
		state.lockedUntil = time.Time{}
		state.failures = 0
	}
	state.failures++
	if state.failures >= threshold {
		state.lockedUntil = now.Add(cooldown)
		state.failures = 0
		g.states[key] = state
		return true, state.lockedUntil, threshold
	}
	g.states[key] = state
	g.evictOverflow()
	return false, time.Time{}, state.failures
}

func (g *loginCooldownGuard) Reset(key string) {
	key = normalizeAdminSecurityKey(key)
	if key == "" {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.states, key)
}

// evictOverflow keeps the state map bounded. Entries are only ever removed
// under memory pressure; the evicted key simply restarts its failure count.
func (g *loginCooldownGuard) evictOverflow() {
	for len(g.states) > g.maxKeys {
		for k := range g.states {
			delete(g.states, k)
			break
		}
	}
}

type consecutiveFailureGuard struct {
	mu      sync.Mutex
	counts  map[string]int
	maxKeys int
}

func newConsecutiveFailureGuard() *consecutiveFailureGuard {
	return &consecutiveFailureGuard{counts: map[string]int{}, maxKeys: defaultSecurityGuardMaxKeys}
}

func (g *consecutiveFailureGuard) RegisterFailure(key string) int {
	key = normalizeAdminSecurityKey(key)
	if key == "" {
		return 0
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.counts[key]++
	g.evictOverflow()
	return g.counts[key]
}

func (g *consecutiveFailureGuard) Reset(key string) {
	key = normalizeAdminSecurityKey(key)
	if key == "" {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.counts, key)
}

// evictOverflow keeps the count map bounded under key rotation.
func (g *consecutiveFailureGuard) evictOverflow() {
	for len(g.counts) > g.maxKeys {
		for k := range g.counts {
			delete(g.counts, k)
			break
		}
	}
}

func normalizeAdminSecurityKey(v string) string {
	key := strings.ToLower(strings.TrimSpace(v))
	if key == "" {
		return emptyAdminSecurityKeyBucket
	}
	return key
}
