package http

import (
	"strings"
	"sync"
	"time"
)

type loginCooldownState struct {
	failures    int
	lockedUntil time.Time
}

type loginCooldownGuard struct {
	mu     sync.Mutex
	states map[string]loginCooldownState
}

func newLoginCooldownGuard() *loginCooldownGuard {
	return &loginCooldownGuard{states: map[string]loginCooldownState{}}
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

type consecutiveFailureGuard struct {
	mu     sync.Mutex
	counts map[string]int
}

func newConsecutiveFailureGuard() *consecutiveFailureGuard {
	return &consecutiveFailureGuard{counts: map[string]int{}}
}

func (g *consecutiveFailureGuard) RegisterFailure(key string) int {
	key = normalizeAdminSecurityKey(key)
	if key == "" {
		return 0
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.counts[key]++
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

func normalizeAdminSecurityKey(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}
