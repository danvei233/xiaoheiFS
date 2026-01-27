package payment

import (
	"context"
	"testing"

	"xiaoheiplay/internal/usecase"
)

func TestSimpleProviders(t *testing.T) {
	for _, provider := range []*simpleProvider{newApprovalProvider(), newBalanceProvider()} {
		if err := provider.SetConfig(""); err != nil {
			t.Fatalf("set config: %v", err)
		}
		if _, err := provider.CreatePayment(context.Background(), usecase.PaymentCreateRequest{}); err == nil {
			t.Fatalf("expected create payment error")
		}
		if _, err := provider.VerifyNotify(context.Background(), map[string]string{}); err == nil {
			t.Fatalf("expected verify notify error")
		}
	}
}
