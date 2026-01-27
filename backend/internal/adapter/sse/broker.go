package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/usecase"
)

type Broker struct {
	events usecase.EventRepository
	mu     sync.RWMutex
	subs   map[int64]map[chan domain.OrderEvent]struct{}
}

func NewBroker(events usecase.EventRepository) *Broker {
	return &Broker{events: events, subs: make(map[int64]map[chan domain.OrderEvent]struct{})}
}

func (b *Broker) Publish(ctx context.Context, orderID int64, eventType string, payload any) (domain.OrderEvent, error) {
	data, _ := json.Marshal(payload)
	ev, err := b.events.AppendEvent(ctx, orderID, eventType, string(data))
	if err != nil {
		return domain.OrderEvent{}, err
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.subs[orderID] {
		select {
		case ch <- ev:
		default:
		}
	}
	return ev, nil
}

func (b *Broker) Subscribe(orderID int64) chan domain.OrderEvent {
	ch := make(chan domain.OrderEvent, 16)
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.subs[orderID] == nil {
		b.subs[orderID] = make(map[chan domain.OrderEvent]struct{})
	}
	b.subs[orderID][ch] = struct{}{}
	return ch
}

func (b *Broker) Unsubscribe(orderID int64, ch chan domain.OrderEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.subs[orderID] != nil {
		delete(b.subs[orderID], ch)
		close(ch)
	}
}

func (b *Broker) Stream(ctx context.Context, w http.ResponseWriter, orderID int64, lastSeq int64) error {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming unsupported")
	}

	if lastSeq > 0 {
		events, err := b.events.ListEventsAfter(ctx, orderID, lastSeq, 200)
		if err != nil {
			return err
		}
		for _, ev := range events {
			writeEvent(w, ev)
		}
		flusher.Flush()
	}

	ch := b.Subscribe(orderID)
	defer b.Unsubscribe(orderID, ch)

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case ev := <-ch:
			writeEvent(w, ev)
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

func writeEvent(w http.ResponseWriter, ev domain.OrderEvent) {
	fmt.Fprintf(w, "id: %d\n", ev.Seq)
	fmt.Fprintf(w, "event: %s\n", ev.Type)
	fmt.Fprintf(w, "data: %s\n\n", ev.DataJSON)
}
