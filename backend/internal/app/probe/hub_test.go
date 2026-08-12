package probe_test

import (
	"testing"
	"time"
	appprobe "xiaoheiplay/internal/app/probe"
)

func TestProbeHubSubscribeAfterLogEndReplaysBacklog(t *testing.T) {
	hub := appprobe.NewHub()
	sessionID := "s1"
	hub.OpenLogSession(sessionID, 1, time.Minute)

	hub.PublishLogChunk(1, sessionID, "req1", "line1")
	hub.PublishLogEnd(1, sessionID, "req1")

	ch, cancel, err := hub.SubscribeLogSession(sessionID)
	if err != nil {
		t.Fatalf("SubscribeLogSession returned error: %v", err)
	}
	defer cancel()

	msg1 := mustRecvLogMsg(t, ch)
	if msg1.Type != "log_chunk" || msg1.Data != "line1" {
		t.Fatalf("unexpected first message: %#v", msg1)
	}
	msg2 := mustRecvLogMsg(t, ch)
	if msg2.Type != "log_end" {
		t.Fatalf("unexpected second message: %#v", msg2)
	}
	if _, ok := <-ch; ok {
		t.Fatalf("expected closed channel after replay")
	}
}

func TestProbeHubLiveSubscriberReceivesChunk(t *testing.T) {
	hub := appprobe.NewHub()
	sessionID := "s2"
	hub.OpenLogSession(sessionID, 1, time.Minute)

	ch, cancel, err := hub.SubscribeLogSession(sessionID)
	if err != nil {
		t.Fatalf("SubscribeLogSession returned error: %v", err)
	}
	defer cancel()

	hub.PublishLogChunk(1, sessionID, "req2", "line2")
	msg := mustRecvLogMsg(t, ch)
	if msg.Type != "log_chunk" || msg.Data != "line2" {
		t.Fatalf("unexpected message: %#v", msg)
	}
}

func TestHubReconnectDoesNotLoseNewConn(t *testing.T) {
	hub := appprobe.NewHub()
	idA := hub.RegisterConn(7, func([]byte) error { return nil })
	// New connection (B) replaces old (A) in delivery, but old unregister must
	// not delete the newer connection.
	hub.RegisterConn(7, func([]byte) error { return nil })
	hub.UnregisterConn(7, idA)
	if !hub.IsOnline(7) {
		t.Fatalf("expected probe to stay online after stale connection teardown")
	}
}

func TestHubLastConnTeardownMarksOffline(t *testing.T) {
	hub := appprobe.NewHub()
	idA := hub.RegisterConn(7, func([]byte) error { return nil })
	idB := hub.RegisterConn(7, func([]byte) error { return nil })
	hub.UnregisterConn(7, idA)
	if !hub.IsOnline(7) {
		t.Fatalf("expected probe online while one connection remains")
	}
	hub.UnregisterConn(7, idB)
	if hub.IsOnline(7) {
		t.Fatalf("expected probe offline after last connection teardown")
	}
}

func TestHubUnregisterReportsRemainingConnections(t *testing.T) {
	hub := appprobe.NewHub()
	// First connection: probe is not yet registered, so registering keeps it
	// online, but unregistering the last one must report offline.
	idA := hub.RegisterConn(7, func([]byte) error { return nil })
	if hub.UnregisterConn(7, idA) {
		t.Fatalf("expected UnregisterConn to report offline after removing the only connection")
	}
	if hub.IsOnline(7) {
		t.Fatalf("expected probe offline after last connection teardown")
	}

	// Two connections: removing one keeps the probe online.
	idB := hub.RegisterConn(7, func([]byte) error { return nil })
	idC := hub.RegisterConn(7, func([]byte) error { return nil })
	if !hub.UnregisterConn(7, idB) {
		t.Fatalf("expected UnregisterConn to report still-online while another connection remains")
	}
	if !hub.IsOnline(7) {
		t.Fatalf("expected probe online while one connection remains")
	}
	if hub.UnregisterConn(7, idC) {
		t.Fatalf("expected UnregisterConn to report offline after removing the last connection")
	}
}

// TestHubPublishLogConcurrentCloseNoPanic exercises the publish/cancel/close
// races that previously panicked with "send on closed channel": the publish
// loop must never send to a channel a concurrent cancel/CloseLogSession/gcLoop
// has closed. The fix holds the hub lock during the (non-blocking) sends.
func TestHubPublishLogConcurrentCloseNoPanic(t *testing.T) {
	hub := appprobe.NewHub()
	sid := "s-race-close"
	hub.OpenLogSession(sid, 1, time.Minute)
	_, cancel, err := hub.SubscribeLogSession(sid)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	ch2, cancel2, err := hub.SubscribeLogSession(sid)
	if err != nil {
		t.Fatalf("subscribe2: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		// No recover here on purpose: a "send on closed channel" panic would
		// crash the test process, which is the loudest possible failure.
		for i := 0; i < 500; i++ {
			hub.PublishLogChunk(1, sid, "req", "line")
		}
	}()

	// Concurrently tear down the subscriptions and the whole session while the
	// publisher is mid-flight.
	cancel()
	cancel2()
	hub.CloseLogSession(sid)

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatalf("publish did not finish")
	}
	// ch2 must be closed after cancel2; reading it (drain) must not panic.
	for range ch2 {
	}
}

// TestHubLogSessionOwnedByMemory verifies the in-memory ownership check used by
// the WS handler for log frames, which must not require a DB round-trip.
func TestHubLogSessionOwnedByMemory(t *testing.T) {
	hub := appprobe.NewHub()
	sid := "s-owner"
	hub.OpenLogSession(sid, 1, time.Minute)

	if !hub.LogSessionOwnedBy(sid, 1) {
		t.Fatalf("owner probe 1 should be reported as owner")
	}
	if hub.LogSessionOwnedBy(sid, 99) {
		t.Fatalf("foreign probe must not be reported as owner")
	}
	if hub.LogSessionOwnedBy("missing", 1) {
		t.Fatalf("missing session must fail closed")
	}
	hub.CloseLogSession(sid)
	if hub.LogSessionOwnedBy(sid, 1) {
		t.Fatalf("closed session must fail closed")
	}
}

func TestHubLogPublishRejectsForeignProbe(t *testing.T) {
	hub := appprobe.NewHub()
	sessionID := "s3"
	hub.OpenLogSession(sessionID, 1, time.Minute)

	// A different probe publishing into probe 1's session must be dropped.
	hub.PublishLogChunk(99, sessionID, "req", "foreign")
	if hub.HasLogSession(sessionID) {
		ch, cancel, err := hub.SubscribeLogSession(sessionID)
		if err != nil {
			t.Fatalf("subscribe: %v", err)
		}
		defer cancel()
		select {
		case msg := <-ch:
			t.Fatalf("foreign probe chunk leaked into session: %#v", msg)
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func mustRecvLogMsg(t *testing.T, ch <-chan appprobe.ProbeLogMessage) appprobe.ProbeLogMessage {
	t.Helper()
	select {
	case msg, ok := <-ch:
		if !ok {
			t.Fatalf("channel closed unexpectedly")
		}
		return msg
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting log message")
		return appprobe.ProbeLogMessage{}
	}
}
