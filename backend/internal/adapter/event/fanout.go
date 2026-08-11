package event

import (
	"context"
	"log"
	"time"

	appports "xiaoheiplay/internal/app/ports"
	"xiaoheiplay/internal/domain"
)

// fanoutMaxConcurrent bounds how many sink goroutines may run at once. Sinks
// are external best-effort delivery (webhook/push) with a 3s deadline; without
// a cap a burst of order events could spawn unbounded goroutines.
const fanoutMaxConcurrent = 16

type EventSink interface {
	NotifyOrderEvent(ctx context.Context, ev domain.OrderEvent) error
}

type FanoutPublisher struct {
	primary appports.EventPublisher
	sinks   []EventSink
	sem     chan struct{}
}

func NewFanoutPublisher(primary appports.EventPublisher, sinks ...EventSink) *FanoutPublisher {
	return &FanoutPublisher{
		primary: primary,
		sinks:   sinks,
		sem:     make(chan struct{}, fanoutMaxConcurrent),
	}
}

func (p *FanoutPublisher) Publish(ctx context.Context, orderID int64, eventType string, payload any) (domain.OrderEvent, error) {
	ev, err := p.primary.Publish(ctx, orderID, eventType, payload)
	if err != nil {
		return ev, err
	}
	for _, sink := range p.sinks {
		if sink == nil {
			continue
		}
		// Do not block the request path on external sinks; keep best-effort
		// delivery. If the concurrency budget is exhausted, drop rather than
		// pile up goroutines.
		select {
		case p.sem <- struct{}{}:
		default:
			log.Printf("fanout: sink busy, dropping notification order_id=%d event=%s", orderID, eventType)
			continue
		}
		go func(s EventSink, event domain.OrderEvent) {
			defer func() { <-p.sem }()
			defer func() {
				if r := recover(); r != nil {
					// A panicking sink must not take down the process.
					log.Printf("fanout: sink panic recovered order_id=%d event=%s panic=%v", event.OrderID, event.Type, r)
				}
			}()
			sinkCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = s.NotifyOrderEvent(sinkCtx, event)
		}(sink, ev)
	}
	return ev, nil
}

var _ appports.EventPublisher = (*FanoutPublisher)(nil)
