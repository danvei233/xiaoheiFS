package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"pingbot/internal/client"
)

func TestRedactSensitiveMasksBearerToken(t *testing.T) {
	line := `POST /api HTTP/1.1
Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.payload.signature
X-Other: 1`
	got := redactSensitive(line)
	if strings.Contains(got, "eyJhbGciOiJIUzI1NiJ9") {
		t.Fatalf("JWT leaked through redaction: %q", got)
	}
	if !strings.Contains(got, "[REDACTED]") {
		t.Fatalf("expected redaction marker, got %q", got)
	}
}

func TestRedactSensitiveMasksStandaloneBearer(t *testing.T) {
	got := redactSensitive("connecting with Bearer abc123.payload.token now")
	if strings.Contains(got, "abc123.payload.token") {
		t.Fatalf("standalone bearer token leaked: %q", got)
	}
	if !strings.Contains(got, "Bearer [REDACTED]") {
		t.Fatalf("expected Bearer [REDACTED] marker, got %q", got)
	}
}

func TestRedactSensitiveMasksPSK(t *testing.T) {
	line := "connection using psk_hUZ9CF5GWu1pw9RzysRCUam3nDK7H7J3QkwKWph45kDn8bAJ now"
	got := redactSensitive(line)
	if strings.Contains(got, "psk_hUZ9CF5GWu1pw9RzysRCUam3nDK7H7J3QkwKWph45kDn8bAJ") {
		t.Fatalf("psk secret leaked: %q", got)
	}
}

func TestClampRuntimeInterval(t *testing.T) {
	cases := []struct{ in, def, want int }{
		{0, 10, 10},
		{-5, 10, 10},
		{5, 10, 5},
		{50000, 10, maxRuntimeIntervalSec},
	}
	for _, tc := range cases {
		if got := clampRuntimeInterval(tc.in, tc.def); got != tc.want {
			t.Fatalf("clampRuntimeInterval(%d,%d) = %d, want %d", tc.in, tc.def, got, tc.want)
		}
	}
}

// TestApplyRuntimeConfigClampsHostileSetConfig verifies that a server set_config
// with an enormous interval (which would overflow time.Duration into a negative
// ticker and panic in time.NewTicker) is clamped to the runtime bound, and that
// a huge chunk cap is capped too.
func TestApplyRuntimeConfigClampsHostileSetConfig(t *testing.T) {
	prev := client.RuntimeConfig{HeartbeatIntervalSec: 10, SnapshotIntervalSec: 60, LogChunkMaxBytes: 16384}
	incoming := client.RuntimeConfig{
		HeartbeatIntervalSec: 9_000_000_000, // ~285 years; v*1e9 overflows int64
		SnapshotIntervalSec:  50000,
		LogChunkMaxBytes:     maxLogChunkBytes * 4,
	}
	got := applyRuntimeConfig(incoming, prev)
	if got.HeartbeatIntervalSec != maxRuntimeIntervalSec {
		t.Fatalf("heartbeat not clamped: got %d, want %d", got.HeartbeatIntervalSec, maxRuntimeIntervalSec)
	}
	if got.SnapshotIntervalSec != maxRuntimeIntervalSec {
		t.Fatalf("snapshot not clamped: got %d, want %d", got.SnapshotIntervalSec, maxRuntimeIntervalSec)
	}
	if got.LogChunkMaxBytes != maxLogChunkBytes {
		t.Fatalf("chunk not capped: got %d, want %d", got.LogChunkMaxBytes, maxLogChunkBytes)
	}
	// The clamped values must produce a valid (positive) ticker duration.
	if time.Duration(got.HeartbeatIntervalSec)*time.Second <= 0 {
		t.Fatalf("clamped heartbeat duration is non-positive: %v", time.Duration(got.HeartbeatIntervalSec)*time.Second)
	}
	// A zero/missing field keeps the previous value (no spurious restart).
	zero := client.RuntimeConfig{}
	kept := applyRuntimeConfig(zero, prev)
	if kept.HeartbeatIntervalSec != prev.HeartbeatIntervalSec || kept.SnapshotIntervalSec != prev.SnapshotIntervalSec {
		t.Fatalf("zero config should keep previous values, got %+v", kept)
	}
}

// TestSleepCtxRespondsToCancellation verifies the reconnect backoff returns
// immediately when the run context is cancelled, so shutdown is never blocked
// by a 30s sleep.
func TestSleepCtxRespondsToCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	if sleepCtx(ctx, 30*time.Second) {
		t.Fatalf("sleepCtx returned true for an already-cancelled context")
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("sleepCtx took %v to return on cancelled context, want ~immediate", elapsed)
	}
}

// TestSleepCtxWaitsForDuration verifies a non-cancelled context waits the full
// backoff (used by the retry path).
func TestSleepCtxWaitsForDuration(t *testing.T) {
	ctx := context.Background()
	start := time.Now()
	if !sleepCtx(ctx, 30*time.Millisecond) {
		t.Fatalf("sleepCtx returned false on a live context")
	}
	if elapsed := time.Since(start); elapsed < 20*time.Millisecond {
		t.Fatalf("sleepCtx returned too early: %v", elapsed)
	}
}

// TestSanitizeURLStripsCredentialsAndQuery verifies logged URLs never carry
// userinfo or a query string, both of which can contain secrets.
func TestSanitizeURLStripsCredentialsAndQuery(t *testing.T) {
	got := SanitizeURL("https://user:supersecret@example.com:8443/api?token=abc&x=1#frag")
	for _, leaked := range []string{"supersecret", "token=abc", "user:"} {
		if strings.Contains(got, leaked) {
			t.Fatalf("SanitizeURL leaked %q in %q", leaked, got)
		}
	}
	if !strings.Contains(got, "example.com:8443/api") {
		t.Fatalf("SanitizeURL dropped host/path: %q", got)
	}
}

// TestToWSURLClearsQuery verifies the WS URL derivation drops the original
// query (so it cannot be echoed in a dial log).
func TestToWSURLClearsQuery(t *testing.T) {
	ws, err := toWSURL("https://example.com/base?secret=1", "/api/v1/probe/ws")
	if err != nil {
		t.Fatalf("toWSURL: %v", err)
	}
	if strings.Contains(ws, "secret") {
		t.Fatalf("toWSURL preserved query: %q", ws)
	}
	if ws != "wss://example.com/api/v1/probe/ws" {
		t.Fatalf("toWSURL = %q", ws)
	}
}
