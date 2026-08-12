package order

import (
	"context"

	"xiaoheiplay/internal/domain"
)

type orderSourceKey struct{}

const (
	// OrderSourceUserUI and OrderSourceUserAPIKey are aliases of the canonical
	// domain constants (moved there for a single definition site).
	OrderSourceUserUI     = domain.OrderSourceUserUI
	OrderSourceUserAPIKey = domain.OrderSourceUserAPIKey
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
