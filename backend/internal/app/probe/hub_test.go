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

	hub.PublishLogChunk(sessionID, "req1", "line1")
	hub.PublishLogEnd(sessionID, "req1")

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

	hub.PublishLogChunk(sessionID, "req2", "line2")
	msg := mustRecvLogMsg(t, ch)
	if msg.Type != "log_chunk" || msg.Data != "line2" {
		t.Fatalf("unexpected message: %#v", msg)
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
