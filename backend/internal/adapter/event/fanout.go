package event

import (
	"context"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/usecase"
)

type WebhookSink interface {
	NotifyOrderEvent(ctx context.Context, ev domain.OrderEvent) error
}

type FanoutPublisher struct {
	primary  usecase.EventPublisher
	webhooks WebhookSink
}

func NewFanoutPublisher(primary usecase.EventPublisher, webhooks WebhookSink) *FanoutPublisher {
	return &FanoutPublisher{primary: primary, webhooks: webhooks}
}

func (p *FanoutPublisher) Publish(ctx context.Context, orderID int64, eventType string, payload any) (domain.OrderEvent, error) {
	ev, err := p.primary.Publish(ctx, orderID, eventType, payload)
	if err != nil {
		return ev, err
	}
	if p.webhooks != nil {
		_ = p.webhooks.NotifyOrderEvent(ctx, ev)
	}
	return ev, nil
}

var _ usecase.EventPublisher = (*FanoutPublisher)(nil)
