package http

import (
	"sync"
	"time"
)

// defaultRateLimiterMaxKeys bounds how many distinct keys a rateLimiter tracks.
// Without a bound, an attacker generating unique keys (spoofed IPs, random
// accounts) would grow the map without limit until the process OOMs.
const defaultRateLimiterMaxKeys = 100000

type rateLimiter struct {
	mu      sync.Mutex
	m       map[string][]time.Time
	maxKeys int
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{m: map[string][]time.Time{}, maxKeys: defaultRateLimiterMaxKeys}
}

func (r *rateLimiter) Allow(key string, limit int, window time.Duration) bool {
	if limit <= 0 {
		return true
	}
	if window <= 0 {
		return true
	}
	now := time.Now()
	cutoff := now.Add(-window)

	r.mu.Lock()
	defer r.mu.Unlock()

	hits := r.m[key]
	dst := hits[:0]
	for _, t := range hits {
		if t.After(cutoff) {
			dst = append(dst, t)
		}
	}
	hits = dst
	if len(hits) >= limit {
		r.m[key] = hits
		return false
	}
	hits = append(hits, now)
	r.m[key] = hits
	r.evictOverflow()
	return true
}

// evictOverflow keeps the key map bounded. Each overflow call removes only the
// entries in excess of maxKeys (the map is restored to maxKeys on every call),
// so the cost is O(1) per request while memory stays bounded regardless of how
// many distinct keys arrive. Under pressure arbitrary entries are evicted; the
// affected key simply restarts its rate-limit window.
func (r *rateLimiter) evictOverflow() {
	for len(r.m) > r.maxKeys {
		for k := range r.m {
			delete(r.m, k)
			break
		}
	}
}
