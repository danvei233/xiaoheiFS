package event

import (
	"context"
	"time"

	appports "xiaoheiplay/internal/app/ports"
	"xiaoheiplay/internal/domain"
)

type EventSink interface {
	NotifyOrderEvent(ctx context.Context, ev domain.OrderEvent) error
}

type FanoutPublisher struct {
	primary appports.EventPublisher
	sinks   []EventSink
}

func NewFanoutPublisher(primary appports.EventPublisher, sinks ...EventSink) *FanoutPublisher {
	return &FanoutPublisher{primary: primary, sinks: sinks}
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
		// Do not block request path on external sinks (webhook/push); keep best-effort delivery.
		go func(s EventSink, event domain.OrderEvent) {
			sinkCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = s.NotifyOrderEvent(sinkCtx, event)
		}(sink, ev)
	}
	return ev, nil
}

var _ appports.EventPublisher = (*FanoutPublisher)(nil)
