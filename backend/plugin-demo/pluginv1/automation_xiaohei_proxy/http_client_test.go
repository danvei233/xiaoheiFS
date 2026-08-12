package main

import (
	"testing"
	"time"
)

// TestSanitizeHeaders covers H5: emitLog must not record static admin API keys
// or request signatures. X-API-Key, X-AKID, X-Signature and Authorization
// values must be masked in the http_trace returned to the host.
func TestSanitizeHeaders(t *testing.T) {
	in := map[string]string{
		"X-API-Key":    "admin-secret-key",
		"X-AKID":       "open-akid",
		"X-Signature":  "deadbeef-signature",
		"Authorization": "Bearer token-secret",
		"X-Timestamp":  "2026-08-11T00:00:00Z",
		"X-Nonce":      "nonce123",
		"Content-Type": "application/json",
	}
	got := sanitizeHeaders(in)
	for _, key := range []string{"X-API-Key", "X-AKID", "X-Signature", "Authorization"} {
		if v, ok := got[key]; !ok || v != "***" {
			t.Fatalf("sanitizeHeaders(%s) = %q, want masked ***", key, v)
		}
	}
	// Non-sensitive headers must be preserved.
	if got["Content-Type"] != "application/json" {
		t.Fatalf("Content-Type = %q, want preserved", got["Content-Type"])
	}
	if got["X-Timestamp"] == "" || got["X-Nonce"] == "" {
		t.Fatalf("X-Timestamp/X-Nonce dropped: %v", got)
	}
}

// TestPickNewestVPS covers M9: the preferredName branch of findNewestVPS was
// dead code — both branches returned the first candidate, so a requested
// hostname could never actually be matched. pickNewestVPS must prefer an exact
// hostname match and otherwise fall back to the newest package+system match.
func TestPickNewestVPS(t *testing.T) {
	mk := func(id int64, name string, pkg, sys int64, created time.Time) vpsDTO {
		return vpsDTO{ID: id, Name: name, PackageID: pkg, SystemID: sys, CreatedAt: created}
	}
	base := time.Unix(1700000000, 0)
	items := []vpsDTO{
		mk(1, "other", 10, 20, base.Add(3*time.Minute)), // newest, but not the requested name
		mk(2, "requested-name", 10, 20, base.Add(1*time.Minute)),
		mk(3, "unrelated", 99, 99, base.Add(9*time.Minute)), // different package
	}

	// With a preferred name, the exact match must win even though it is older.
	v, err := pickNewestVPS(items, 10, 20, "requested-name")
	if err != nil {
		t.Fatalf("pickNewestVPS: %v", err)
	}
	if v.ID != 2 {
		t.Fatalf("pickNewestVPS with preferredName = id %d, want 2 (exact name match)", v.ID)
	}

	// Without a preferred name, the newest matching package+system wins.
	v, err = pickNewestVPS(items, 10, 20, "")
	if err != nil {
		t.Fatalf("pickNewestVPS: %v", err)
	}
	if v.ID != 1 {
		t.Fatalf("pickNewestVPS without preferredName = id %d, want 1 (newest)", v.ID)
	}

	// Preferred name set but no exact match -> newest matching fallback.
	v, err = pickNewestVPS(items, 10, 20, "no-such-name")
	if err != nil {
		t.Fatalf("pickNewestVPS: %v", err)
	}
	if v.ID != 1 {
		t.Fatalf("pickNewestVPS with unmatched preferredName = id %d, want 1 (newest fallback)", v.ID)
	}

	// No match at all.
	if _, err := pickNewestVPS(items, 999, 999, ""); err == nil {
		t.Fatalf("pickNewestVPS with no match returned no error")
	}
}
