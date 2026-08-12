package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	appprobe "xiaoheiplay/internal/app/probe"
)

// TestClientIPIgnoresSpoofedForwardedFor verifies the router trusts no proxy by
// default: c.ClientIP() returns the direct peer address, so an attacker-supplied
// X-Forwarded-For cannot spoof the IP that keys rate-limit budgets or callback
// URLs.
func TestClientIPIgnoresSpoofedForwardedFor(t *testing.T) {
	r := newGinEngine()
	r.GET("/whoami", func(c *gin.Context) {
		c.String(http.StatusOK, c.ClientIP())
	})
	req := httptest.NewRequest(http.MethodGet, "/whoami", nil)
	req.RemoteAddr = "203.0.113.7:12345"
	req.Header.Set("X-Forwarded-For", "198.51.100.99")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if got := rec.Body.String(); got != "203.0.113.7" {
		t.Fatalf("ClientIP = %q, want the peer IP (spoofed XFF ignored)", got)
	}
}

// TestProbeLogSessionOwnedByPrefersHub verifies the WS log-frame ownership
// check resolves against the hub's in-memory session map (no DB round-trip per
// frame) while a probeSvc is not even needed for the hub-known path.
func TestProbeLogSessionOwnedByPrefersHub(t *testing.T) {
	hub := appprobe.NewHub()
	hub.OpenLogSession("s1", 1, time.Minute)
	h := &Handler{probeHub: hub}
	if !h.probeLogSessionOwnedBy(context.Background(), "s1", 1) {
		t.Fatalf("hub-owned session rejected")
	}
	if h.probeLogSessionOwnedBy(context.Background(), "s1", 99) {
		t.Fatalf("foreign probe accepted via hub")
	}
	// A session unknown to the hub must fail closed without panicking.
	if h.probeLogSessionOwnedBy(context.Background(), "missing", 1) {
		t.Fatalf("missing session accepted")
	}
}

// TestTeardownProbeConnMarksOfflineOnlyWhenEmpty verifies the handler tears
// down exactly its own connection and marks the probe offline only when the hub
// reports (atomically) that no connection remains.
func TestTeardownProbeConnMarksOfflineOnlyWhenEmpty(t *testing.T) {
	hub := appprobe.NewHub()
	h := &Handler{probeHub: hub}

	// Two connections: tearing one down keeps the probe online and must NOT
	// trigger an offline transition (probeSvc is nil here; the guard is the hub
	// reporting a remaining connection).
	idA := hub.RegisterConn(7, func([]byte) error { return nil })
	idB := hub.RegisterConn(7, func([]byte) error { return nil })
	if !h.teardownProbeConn(7, idA) {
		t.Fatalf("teardown of a non-last connection must report still-online")
	}
	if !hub.IsOnline(7) {
		t.Fatalf("probe must stay online while a connection remains")
	}

	// Tearing the last connection reports offline.
	if h.teardownProbeConn(7, idB) {
		t.Fatalf("teardown of the last connection must report offline")
	}
	if hub.IsOnline(7) {
		t.Fatalf("probe must be offline after last connection teardown")
	}
}
