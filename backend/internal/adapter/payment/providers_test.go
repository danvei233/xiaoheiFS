package payment

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"xiaoheiplay/internal/usecase"
)

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
	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}
	raw := usecase.RawHTTPRequest{
		Method:   "POST",
		Path:     "/payments/notify/yipay",
		Headers:  map[string][]string{"Content-Type": {"application/x-www-form-urlencoded"}},
		Body:     []byte(form.Encode()),
		RawQuery: "",
	}
	notify, err := p.VerifyNotify(context.Background(), raw)
	if err != nil || !notify.Paid {
		t.Fatalf("verify notify: %v, %+v", err, notify)
	}
}

func TestSimpleProvider_Errors(t *testing.T) {
	p := newApprovalProvider()
	if _, err := p.CreatePayment(context.Background(), usecase.PaymentCreateRequest{}); err == nil {
		t.Fatalf("expected create error")
	}
	if _, err := p.VerifyNotify(context.Background(), usecase.RawHTTPRequest{}); err == nil {
		t.Fatalf("expected verify error")
	}
}
