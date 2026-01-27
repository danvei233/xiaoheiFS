package payment

import (
	"context"
	"testing"

	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/usecase"
)

func TestRegistry_ListAndUpdate(t *testing.T) {
	_, repo := testutil.NewTestDB(t, false)
	reg := NewRegistry(repo)
	ctx := context.Background()

	providers, err := reg.ListProviders(ctx, true)
	if err != nil {
		t.Fatalf("list providers: %v", err)
	}
	if len(providers) == 0 {
		t.Fatalf("expected providers")
	}

	cfg, enabled, err := reg.GetProviderConfig(ctx, "custom")
	if err != nil {
		t.Fatalf("get provider config: %v", err)
	}
	if cfg == "" || !enabled {
		t.Fatalf("expected default config enabled")
	}

	if err := reg.UpdateProviderConfig(ctx, "custom", true, `{"pay_url":"https://pay.local","instructions":"hi"}`); err != nil {
		t.Fatalf("update provider config: %v", err)
	}
	provider, err := reg.GetProvider(ctx, "custom")
	if err != nil {
		t.Fatalf("get provider: %v", err)
	}
	result, err := provider.CreatePayment(ctx, usecase.PaymentCreateRequest{OrderID: 1, UserID: 2})
	if err != nil {
		t.Fatalf("create payment: %v", err)
	}
	if result.PayURL == "" {
		t.Fatalf("expected pay url")
	}

	if err := reg.UpdateProviderConfig(ctx, "custom", false, ``); err != nil {
		t.Fatalf("disable provider: %v", err)
	}
	if _, err := reg.GetProvider(ctx, "custom"); err == nil {
		t.Fatalf("expected forbidden")
	}
}
