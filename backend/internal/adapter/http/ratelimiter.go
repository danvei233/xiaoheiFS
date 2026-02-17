package http

import (
	"sync"
	"time"
)

type rateLimiter struct {
	mu sync.Mutex
	m  map[string][]time.Time
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{m: map[string][]time.Time{}}
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
	return true
}
