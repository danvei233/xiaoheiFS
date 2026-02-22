package order

import "context"

type orderSourceKey struct{}

const (
	OrderSourceUserUI     = "user_ui"
	OrderSourceUserAPIKey = "user_apikey"
)

func WithOrderSource(ctx context.Context, source string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, orderSourceKey{}, source)
}

func resolveOrderSource(ctx context.Context) string {
	if ctx == nil {
		return OrderSourceUserUI
	}
	if v, ok := ctx.Value(orderSourceKey{}).(string); ok {
		if v == OrderSourceUserAPIKey {
			return v
		}
	}
	return OrderSourceUserUI
}
