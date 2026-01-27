package payment

import (
	"context"
	"strings"
	"testing"

	"xiaoheiplay/internal/usecase"
)

func TestCustomProvider_CreatePayment(t *testing.T) {
	p := newCustomProvider()
	if err := p.SetConfig(`{"pay_url":"https://pay.local","instructions":"hello"}`); err != nil {
		t.Fatalf("set config: %v", err)
	}
	res, err := p.CreatePayment(context.Background(), usecase.PaymentCreateRequest{
		OrderID: 1,
		Amount:  990,
	})
	if err != nil {
		t.Fatalf("create payment: %v", err)
	}
	if res.PayURL != "https://pay.local" || res.TradeNo == "" {
		t.Fatalf("unexpected result: %#v", res)
	}
}

func TestYipayProvider_SignAndVerify(t *testing.T) {
	p := newYipayProvider()
	if err := p.SetConfig(`{"base_url":"https://yipay.local","pid":"100","key":"k","pay_type":"alipay","sign_type":"MD5"}`); err != nil {
		t.Fatalf("set config: %v", err)
	}
	res, err := p.CreatePayment(context.Background(), usecase.PaymentCreateRequest{
		OrderID: 1,
		Amount:  1000,
		Subject: "test",
	})
	if err != nil {
		t.Fatalf("create payment: %v", err)
	}
	if !strings.Contains(res.PayURL, "sign=") {
		t.Fatalf("expected sign in url")
	}
	params := map[string]string{
		"out_trade_no": res.TradeNo,
		"money":        "10.00",
		"trade_status": "TRADE_SUCCESS",
		"sign_type":    "MD5",
	}
	params["sign"] = p.sign(params)
	notify, err := p.VerifyNotify(context.Background(), params)
	if err != nil || !notify.Paid {
		t.Fatalf("verify notify: %v, %+v", err, notify)
	}
}

func TestSimpleProvider_Errors(t *testing.T) {
	p := newApprovalProvider()
	if _, err := p.CreatePayment(context.Background(), usecase.PaymentCreateRequest{}); err == nil {
		t.Fatalf("expected create error")
	}
	if _, err := p.VerifyNotify(context.Background(), map[string]string{}); err == nil {
		t.Fatalf("expected verify error")
	}
}
