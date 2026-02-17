package payment

import (
	"context"
	"testing"

	appshared "xiaoheiplay/internal/app/shared"
)

func TestSimpleProvider_Errors(t *testing.T) {
	p := newApprovalProvider()
	if _, err := p.CreatePayment(context.Background(), appshared.PaymentCreateRequest{}); err == nil {
		t.Fatalf("expected create error")
	}
	if _, err := p.VerifyNotify(context.Background(), appshared.RawHTTPRequest{}); err == nil {
		t.Fatalf("expected verify error")
	}
}
