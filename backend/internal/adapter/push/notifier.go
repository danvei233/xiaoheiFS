package push

import (
	"context"

	appports "xiaoheiplay/internal/app/ports"
	apppush "xiaoheiplay/internal/app/push"
	"xiaoheiplay/internal/domain"
)

type OrderPushNotifier struct {
	orders appports.OrderRepository
	push   *apppush.Service
}

func NewOrderPushNotifier(orders appports.OrderRepository, push *apppush.Service) *OrderPushNotifier {
	return &OrderPushNotifier{
		orders: orders,
		push:   push,
	}
}

func (n *OrderPushNotifier) NotifyOrderEvent(ctx context.Context, ev domain.OrderEvent) error {
	if n.push == nil || n.orders == nil {
		return nil
	}
	if ev.Type != "order.pending_review" {
		return nil
	}
	order, err := n.orders.GetOrder(ctx, ev.OrderID)
	if err != nil {
		return err
	}
	return n.push.NotifyAdminsNewOrder(ctx, order)
}
