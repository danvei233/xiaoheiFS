package event

import (
	"context"
	"sync"
	"testing"
	"time"

	"xiaoheiplay/internal/domain"
)

type fakePrimary struct{}

func (fakePrimary) Publish(ctx context.Context, orderID int64, eventType string, payload any) (domain.OrderEvent, error) {
	return domain.OrderEvent{OrderID: orderID, Seq: 1, Type: eventType, DataJSON: `{}`}, nil
}

type eventSinkFunc func(ctx context.Context, ev domain.OrderEvent) error

func (f eventSinkFunc) NotifyOrderEvent(ctx context.Context, ev domain.OrderEvent) error { return f(ctx, ev) }

type panicSink struct{}

func (panicSink) NotifyOrderEvent(ctx context.Context, ev domain.OrderEvent) error {
	panic("sink exploded")
}

// TestFanoutPublisher_RecoversSinkPanic guards F5: a panicking sink goroutine
// must be recovered and must not crash the whole process.
func TestFanoutPublisher_RecoversSinkPanic(t *testing.T) {
	p := NewFanoutPublisher(fakePrimary{}, panicSink{})
	ev, err := p.Publish(context.Background(), 1, "order.pending_review", map[string]any{"ok": true})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if ev.Seq != 1 {
		t.Fatalf("unexpected event")
	}
	// Give the sink goroutine time to run and (with the fix) recover.
	time.Sleep(100 * time.Millisecond)
}

// TestFanoutPublisher_BoundsConcurrency guards F6: concurrent sink goroutines
// must never exceed the configured cap even under a burst of events.
func TestFanoutPublisher_BoundsConcurrency(t *testing.T) {
	const events = 50
	var mu sync.Mutex
	active, maxActive := 0, 0
	sink := eventSinkFunc(func(ctx context.Context, ev domain.OrderEvent) error {
		mu.Lock()
		active++
		if active > maxActive {
			maxActive = active
		}
		mu.Unlock()
		time.Sleep(300 * time.Millisecond)
		mu.Lock()
		active--
		mu.Unlock()
		return nil
	})
	p := NewFanoutPublisher(fakePrimary{}, sink)
	for i := 0; i < events; i++ {
		if _, err := p.Publish(context.Background(), int64(i), "order.pending_review", nil); err != nil {
			t.Fatalf("publish %d: %v", i, err)
		}
	}
	// Allow acquired goroutines to start before inspecting the peak.
	time.Sleep(100 * time.Millisecond)
	if maxActive > fanoutMaxConcurrent {
		t.Fatalf("concurrency exceeded cap: maxActive=%d cap=%d", maxActive, fanoutMaxConcurrent)
	}
}

// TestFanoutPublisher_SkipsNilSink verifies nil sinks are ignored.
func TestFanoutPublisher_SkipsNilSink(t *testing.T) {
	p := NewFanoutPublisher(fakePrimary{}, nil)
	if _, err := p.Publish(context.Background(), 1, "order.pending_review", nil); err != nil {
		t.Fatalf("publish: %v", err)
	}
}
