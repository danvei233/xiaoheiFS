package main

import (
	"net"
	"net/rpc"
	"strings"
	"testing"

	paymentplugin "xiaoheiplay/internal/adapter/plugins/payment"
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

// TestDemoPaymentProvider_VerifyNotifyV2_QueryAndFormBody guards the P3 fix: the
// versioned VerifyNotifyV2 must merge the raw query AND the form-urlencoded body
// (plus headers) into the params the demo handler verifies against — the old
// implementation only copied headers and silently dropped query/body.
func TestDemoPaymentProvider_VerifyNotifyV2_QueryAndFormBody(t *testing.T) {
	p := &DemoPaymentProvider{}
	notify, err := p.VerifyNotifyV2(shared.RawHTTPRequest{
		Method:   "POST",
		Path:     "/api/v1/payments/notify",
		RawQuery: "trade_no=TN-Q&status=paid",
		Headers:  map[string][]string{"Content-Type": {"application/x-www-form-urlencoded"}},
		Body:     []byte("sign=body-sign&amount=2500"),
	})
	if err != nil {
		t.Fatalf("verify notify v2: %v", err)
	}
	if notify.TradeNo != "TN-Q" || !notify.Paid {
		t.Fatalf("query params did not reach the demo handler: %+v", notify)
	}
	if notify.Raw["sign"] != "body-sign" || notify.Raw["amount"] != "2500" {
		t.Fatalf("form body params did not reach the demo handler: %+v", notify.Raw)
	}
}

// TestDemoPaymentProvider_VerifyNotifyV2_RPCRoundTrip verifies the versioned
// net/rpc contract end-to-end: a RawHTTPRequest carrying a query and a
// form-urlencoded body round-trips through net/rpc and both reach the demo
// handler's params (mirroring the host's versioned VerifyNotifyV2 transport).
func TestDemoPaymentProvider_VerifyNotifyV2_RPCRoundTrip(t *testing.T) {
	srv, err := (&paymentplugin.ProviderPlugin{Impl: &DemoPaymentProvider{}}).Server(nil)
	if err != nil {
		t.Fatalf("plugin server: %v", err)
	}
	server := rpc.NewServer()
	if err := server.RegisterName("Plugin", srv); err != nil {
		t.Fatalf("register plugin server: %v", err)
	}
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	go server.ServeConn(serverConn)
	client := rpc.NewClient(clientConn)
	defer client.Close()

	req := shared.RawHTTPRequest{
		Method:   "POST",
		Path:     "/api/v1/payments/notify",
		RawQuery: "trade_no=TN-RPC&status=paid",
		Headers: map[string][]string{
			"Content-Type": {"application/x-www-form-urlencoded"},
			"X-Sign":       {"sig-value"},
		},
		Body: []byte("sign=body-sign-rpc&amount=3100"),
	}
	var resp shared.PaymentNotifyResult
	if err := client.Call("Plugin.VerifyNotifyV2", paymentplugin.NotifyV2Args{Request: req}, &resp); err != nil {
		t.Fatalf("verify v2 over net/rpc: %v", err)
	}
	if resp.TradeNo != "TN-RPC" || !resp.Paid {
		t.Fatalf("query params did not reach the demo handler over net/rpc: %+v", resp)
	}
	if resp.Raw["sign"] != "body-sign-rpc" || resp.Raw["amount"] != "3100" {
		t.Fatalf("form body params did not reach the demo handler over net/rpc: %+v", resp.Raw)
	}
	if resp.Raw["x-sign"] != "sig-value" {
		t.Fatalf("headers did not reach the demo handler over net/rpc: %+v", resp.Raw)
	}
}
