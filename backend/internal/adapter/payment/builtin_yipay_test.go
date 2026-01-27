package payment

import (
	"context"
	"strings"
	"testing"

	"xiaoheiplay/internal/usecase"
)

func TestYipayProvider(t *testing.T) {
	provider := newYipayProvider()
	if err := provider.SetConfig(""); err != nil {
		t.Fatalf("set config: %v", err)
	}
	if _, err := provider.CreatePayment(context.Background(), usecase.PaymentCreateRequest{}); err == nil {
		t.Fatalf("expected config error")
	}
	cfg := `{"base_url":"https://pay.local","pid":"pid","key":"secret","pay_type":"alipay","notify_url":"https://notify","return_url":"https://return"}`
	if err := provider.SetConfig(cfg); err != nil {
		t.Fatalf("set config: %v", err)
	}
	res, err := provider.CreatePayment(context.Background(), usecase.PaymentCreateRequest{OrderID: 1, Amount: 1000, Subject: "Order"})
	if err != nil {
		t.Fatalf("create payment: %v", err)
	}
	if !strings.Contains(res.PayURL, "sign=") {
		t.Fatalf("expected sign in pay url")
	}
	if _, err := provider.VerifyNotify(context.Background(), map[string]string{}); err == nil {
		t.Fatalf("expected missing sign error")
	}
	params := map[string]string{
		"pid":          "pid",
		"type":         "alipay",
		"out_trade_no": "TN1",
		"name":         "Order",
		"money":        "10.00",
		"trade_status": "TRADE_SUCCESS",
	}
	params["sign"] = provider.sign(params)
	result, err := provider.VerifyNotify(context.Background(), params)
	if err != nil || !result.Paid {
		t.Fatalf("verify notify: %v %v", result, err)
	}
	result2, err := provider.VerifyNotify(context.Background(), params)
	if err != nil || result2.TradeNo != result.TradeNo {
		t.Fatalf("verify notify again: %v %v", result2, err)
	}
}
