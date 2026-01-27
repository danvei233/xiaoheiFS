package payment

import (
	"context"
	"testing"

	"xiaoheiplay/internal/usecase"
)

func TestCustomProvider(t *testing.T) {
	provider := newCustomProvider()
	if err := provider.SetConfig("{"); err == nil {
		t.Fatalf("expected config error")
	}
	if err := provider.SetConfig(`{"pay_url":"https://pay.local","instructions":"pay here"}`); err != nil {
		t.Fatalf("set config: %v", err)
	}
	res, err := provider.CreatePayment(context.Background(), usecase.PaymentCreateRequest{OrderID: 1, Amount: 1000})
	if err != nil {
		t.Fatalf("create payment: %v", err)
	}
	if res.PayURL == "" || res.TradeNo == "" {
		t.Fatalf("expected pay url and trade no")
	}
	if res.Extra["instructions"] == "" {
		t.Fatalf("expected instructions")
	}
	if _, err := provider.VerifyNotify(context.Background(), map[string]string{}); err == nil {
		t.Fatalf("expected verify notify error")
	}
}
