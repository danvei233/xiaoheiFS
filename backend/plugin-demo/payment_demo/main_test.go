package main

import (
	"strings"
	"testing"

	"xiaoheiplay/internal/app/shared"
)

func TestDemoPaymentProvider(t *testing.T) {
	p := &DemoPaymentProvider{}
	if p.Key() != "demo_pay" {
		t.Fatalf("unexpected key: %s", p.Key())
	}
	if p.Name() == "" {
		t.Fatalf("expected name")
	}
	if err := p.SetConfig(`{"base_url":"https://pay.local","api_key":"k","note":"hello"}`); err != nil {
		t.Fatalf("set config: %v", err)
	}
	result, err := p.CreatePayment(shared.PaymentCreateRequest{OrderID: 10, UserID: 20})
	if err != nil {
		t.Fatalf("create payment: %v", err)
	}
	if !strings.HasPrefix(result.TradeNo, "demo-") {
		t.Fatalf("unexpected trade no: %s", result.TradeNo)
	}
	if !strings.Contains(result.PayURL, "https://pay.local") {
		t.Fatalf("unexpected pay url: %s", result.PayURL)
	}
	if result.Extra["note"] != "hello" {
		t.Fatalf("unexpected extra note")
	}

	notify, err := p.VerifyNotify(map[string]string{"trade_no": "t1", "status": "paid"})
	if err != nil {
		t.Fatalf("verify notify: %v", err)
	}
	if !notify.Paid || notify.TradeNo != "t1" {
		t.Fatalf("unexpected notify result")
	}

	if err := p.SetConfig(""); err != nil {
		t.Fatalf("set empty config: %v", err)
	}
	_, _ = p.CreatePayment(shared.PaymentCreateRequest{OrderID: 1, UserID: 1})
	_, _ = p.VerifyNotify(map[string]string{"trade_no": "t2", "status": "paid"})
}
