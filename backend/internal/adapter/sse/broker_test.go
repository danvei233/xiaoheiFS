package sse

import (
	"bytes"
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"xiaoheiplay/internal/domain"
)

type testWriter struct {
	mu     sync.Mutex
	header http.Header
	body   bytes.Buffer
	status int
}

func newTestWriter() *testWriter {
	return &testWriter{header: http.Header{}, status: http.StatusOK}
}

func (w *testWriter) Header() http.Header { return w.header }
func (w *testWriter) WriteHeader(statusCode int) {
	w.status = statusCode
}
func (w *testWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.body.Write(p)
}
func (w *testWriter) Flush() {}

type fakeEventRepo struct{}

func (f fakeEventRepo) AppendEvent(ctx context.Context, orderID int64, eventType string, dataJSON string) (domain.OrderEvent, error) {
	return domain.OrderEvent{OrderID: orderID, Seq: 1, Type: eventType, DataJSON: dataJSON}, nil
}
func (f fakeEventRepo) ListEventsAfter(ctx context.Context, orderID int64, afterSeq int64, limit int) ([]domain.OrderEvent, error) {
	return nil, nil
}

func TestBroker_StreamHeaders(t *testing.T) {
	broker := NewBroker(fakeEventRepo{})
	w := newTestWriter()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = broker.Stream(ctx, w, 1, 0)
		close(done)
	}()
	time.Sleep(10 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("stream did not return")
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("expected event-stream, got %q", ct)
	}
}

// TestBroker_UnsubscribeRemovesEmptyMap guards F17: once the last subscriber of
// an order goes away, its empty map must be removed from the broker instead of
// leaking for the lifetime of the process.
func TestBroker_UnsubscribeRemovesEmptyMap(t *testing.T) {
	broker := NewBroker(fakeEventRepo{})
	ch := broker.Subscribe(1)
	broker.Unsubscribe(1, ch)
	broker.mu.RLock()
	_, exists := broker.subs[1]
	broker.mu.RUnlock()
	if exists {
		t.Fatalf("expected empty order map to be removed from subs")
	}
}

// TestBroker_UnsubscribeWithOtherSubsNoPanic guards F21: a repeated
// Unsubscribe for the same channel must not panic (close of closed channel)
// even when other subscribers remain on the same order.
func TestBroker_UnsubscribeWithOtherSubsNoPanic(t *testing.T) {
	broker := NewBroker(fakeEventRepo{})
	ch1 := broker.Subscribe(1)
	ch2 := broker.Subscribe(1)
	broker.Unsubscribe(1, ch1)
	broker.Unsubscribe(1, ch1) // must not panic
	broker.Unsubscribe(1, ch2)
}

// TestBroker_UnsubscribeIdempotent guards F21 for the single-subscriber case.
func TestBroker_UnsubscribeIdempotent(t *testing.T) {
	broker := NewBroker(fakeEventRepo{})
	ch := broker.Subscribe(1)
	broker.Unsubscribe(1, ch)
	broker.Unsubscribe(1, ch) // must not panic
}

func TestBroker_PublishAndWriteEvent(t *testing.T) {
	broker := NewBroker(fakeEventRepo{})
	ch := broker.Subscribe(1)
	defer broker.Unsubscribe(1, ch)

	ev, err := broker.Publish(context.Background(), 1, "paid", map[string]any{"ok": true})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	select {
	case got := <-ch:
		if got.Type != ev.Type {
			t.Fatalf("unexpected event type")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("expected event")
	}

	w := newTestWriter()
	writeEvent(w, ev)
	if w.body.Len() == 0 {
		t.Fatalf("expected event data")
	}
}

var _ http.Flusher = (*testWriter)(nil)
var _ interface {
	AppendEvent(ctx context.Context, orderID int64, eventType string, dataJSON string) (domain.OrderEvent, error)
	ListEventsAfter(ctx context.Context, orderID int64, afterSeq int64, limit int) ([]domain.OrderEvent, error)
} = fakeEventRepo{}
