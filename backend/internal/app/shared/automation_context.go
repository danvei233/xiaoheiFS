package shared

import "context"

type automationLogContextKey struct{}

func WithAutomationLogContext(ctx context.Context, orderID, orderItemID int64) context.Context {
	return context.WithValue(ctx, automationLogContextKey{}, AutomationLogContext{
		OrderID:     orderID,
		OrderItemID: orderItemID,
	})
}

func GetAutomationLogContext(ctx context.Context) (AutomationLogContext, bool) {
	value := ctx.Value(automationLogContextKey{})
	if value == nil {
		return AutomationLogContext{}, false
	}
	logCtx, ok := value.(AutomationLogContext)
	return logCtx, ok
}
