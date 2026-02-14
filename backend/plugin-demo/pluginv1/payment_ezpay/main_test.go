package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pluginv1 "xiaoheiplay/plugin/v1"
)

func TestEZPay_CreatePayment_FormHTML_PlainSignModeMatchesPHPSDK(t *testing.T) {
	core := &coreServer{}
	core.cfg = config{
		GatewayBaseURL: "https://www.ezfpy.cn",
		SubmitPath:     "submit.php",
		PID:            "10001",
		MerchantKey:    "testkey",
		SignType:       "MD5",
		SignKeyMode:    "plain",
	}
	pay := &payServer{core: core}

	resp, err := pay.CreatePayment(context.Background(), &pluginv1.CreatePaymentRpcRequest{
		Method: "wxpay",
		Request: &pluginv1.PaymentCreateRequest{
			OrderNo:   "ORDER-123",
			UserId:    "1",
			Amount:    1234,
			Currency:  "CNY",
			Subject:   "test",
			NotifyUrl: "https://host.example/api/v1/payments/notify/ezpay.wxpay",
			ReturnUrl: "https://host.example/return",
			Extra:     map[string]string{"client_ip": "127.0.0.1", "device": "pc"},
		},
	})
	if err != nil {
		t.Fatalf("CreatePayment err: %v", err)
	}
	if !resp.GetOk() {
		t.Fatalf("CreatePayment not ok: %v", resp.GetError())
	}
	if resp.GetPayUrl() != "" {
		t.Fatalf("expected empty pay_url (POST form flow), got %q", resp.GetPayUrl())
	}
	formHTML := resp.GetExtra()["form_html"]
	if formHTML == "" {
		t.Fatalf("expected extra.form_html")
	}
	if want := "ORDER-123-wx-"; !contains(formHTML, want) {
		t.Fatalf("form_html missing out_trade_no %q", want)
	}
	if !contains(formHTML, "/submit.php") {
		t.Fatalf("form_html missing submit.php")
	}
	outTradeNo := extractInputValue(formHTML, "out_trade_no")
	expectedSign := signEZPay(map[string]string{
		"pid":          "10001",
		"type":         "wxpay",
		"out_trade_no": outTradeNo,
		"notify_url":   "https://host.example/api/v1/payments/notify/ezpay.wxpay",
		"return_url":   "https://host.example/return",
		"name":         "test",
		"money":        "12.34",
		"clientip":     "127.0.0.1",
		"device":       "pc",
		"sign_type":    "MD5",
	}, "testkey", "plain")
	if got := extractInputValue(formHTML, "sign"); got != expectedSign {
		t.Fatalf("unexpected sign: %q want %q", got, expectedSign)
	}
	if got := extractInputValue(formHTML, "clientip"); got != "127.0.0.1" {
		t.Fatalf("unexpected clientip: %q", got)
	}
	if got := extractInputValue(formHTML, "device"); got != "pc" {
		t.Fatalf("unexpected device: %q", got)
	}
}

func TestEZPay_CreatePayment_FormHTML_AmpKeySignModeMatchesPHPSDK(t *testing.T) {
	core := &coreServer{}
	core.cfg = config{
		GatewayBaseURL: "https://www.ezfpy.cn",
		SubmitPath:     "submit.php",
		PID:            "10001",
		MerchantKey:    "testkey",
		SignType:       "MD5",
		SignKeyMode:    "amp_key",
	}
	pay := &payServer{core: core}

	resp, err := pay.CreatePayment(context.Background(), &pluginv1.CreatePaymentRpcRequest{
		Method: "wxpay",
		Request: &pluginv1.PaymentCreateRequest{
			OrderNo:   "ORDER-123",
			UserId:    "1",
			Amount:    1234,
			Currency:  "CNY",
			Subject:   "test",
			NotifyUrl: "https://host.example/api/v1/payments/notify/ezpay.wxpay",
			ReturnUrl: "https://host.example/return",
			Extra:     map[string]string{"client_ip": "127.0.0.1", "device": "pc"},
		},
	})
	if err != nil {
		t.Fatalf("CreatePayment err: %v", err)
	}
	if !resp.GetOk() {
		t.Fatalf("CreatePayment not ok: %v", resp.GetError())
	}
	formHTML := resp.GetExtra()["form_html"]
	outTradeNo := extractInputValue(formHTML, "out_trade_no")
	expectedSign := signEZPay(map[string]string{
		"pid":          "10001",
		"type":         "wxpay",
		"out_trade_no": outTradeNo,
		"notify_url":   "https://host.example/api/v1/payments/notify/ezpay.wxpay",
		"return_url":   "https://host.example/return",
		"name":         "test",
		"money":        "12.34",
		"clientip":     "127.0.0.1",
		"device":       "pc",
		"sign_type":    "MD5",
	}, "testkey", "amp_key")
	if got := extractInputValue(formHTML, "sign"); got != expectedSign {
		t.Fatalf("unexpected sign: %q want %q", got, expectedSign)
	}
}

func TestEZPay_CreatePayment_MAPI_ReturnsQR(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if got := r.FormValue("clientip"); got != "127.0.0.1" {
			t.Fatalf("unexpected clientip: %q", got)
		}
		if got := r.FormValue("device"); got != "mobile" {
			t.Fatalf("unexpected device: %q", got)
		}
		if got := r.FormValue("name"); got == "" {
			t.Fatalf("expected name")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":1,"trade_no":"TN-1","qrcode":"weixin://wxpay/bizpayurl?pr=abc"}`))
	}))
	defer ts.Close()

	core := &coreServer{}
	core.cfg = config{
		SubmitURL:      ts.URL + "/mapi.php",
		PID:            "10001",
		MerchantKey:    "testkey",
		SignType:       "MD5",
		SignKeyMode:    "plain",
		TimeoutSec:     5,
		GatewayBaseURL: "",
	}
	pay := &payServer{core: core}
	resp, err := pay.CreatePayment(context.Background(), &pluginv1.CreatePaymentRpcRequest{
		Method: "wxpay",
		Request: &pluginv1.PaymentCreateRequest{
			OrderNo:   "ORDER-123",
			UserId:    "1",
			Amount:    1234,
			Currency:  "CNY",
			Subject:   "",
			NotifyUrl: "https://host.example/api/v1/payments/notify/ezpay.wxpay",
			ReturnUrl: "https://host.example/return",
			Extra:     map[string]string{"client_ip": "127.0.0.1", "device": "mobile"},
		},
	})
	if err != nil {
		t.Fatalf("CreatePayment err: %v", err)
	}
	if !resp.GetOk() {
		t.Fatalf("CreatePayment not ok")
	}
	if got := resp.GetExtra()["pay_kind"]; got != "qr" {
		t.Fatalf("unexpected pay_kind: %q", got)
	}
	if got := resp.GetExtra()["code_url"]; got == "" {
		t.Fatalf("missing code_url")
	}
	if got := resp.GetExtra()["form_html"]; got != "" {
		t.Fatalf("mapi flow should not return form_html")
	}
}

func TestEZPay_VerifyNotify_MD5_TwoKeyModes(t *testing.T) {
	core := &coreServer{}
	core.cfg = config{
		GatewayBaseURL: "https://www.ezfpy.cn",
		PID:            "10001",
		MerchantKey:    "testkey",
		SignType:       "MD5",
	}
	pay := &payServer{core: core}

	base := map[string]string{
		"pid":          core.cfg.PID,
		"out_trade_no": "ORDER-123-zfb",
		"trade_no":     "PLAT-999",
		"type":         "alipay",
		"money":        "12.34",
		"trade_status": "TRADE_SUCCESS",
		"sign_type":    "MD5",
	}

	cases := []struct {
		name string
		sign string
	}{
		{name: "plain", sign: signEZPay(base, core.cfg.MerchantKey, "plain")},
		{name: "amp_key", sign: signEZPay(base, core.cfg.MerchantKey, "amp_key")},
	}
	for _, tc := range cases {
		params := map[string]string{}
		for k, v := range base {
			params[k] = v
		}
		params["sign"] = tc.sign
		q := url.Values{}
		for k, v := range params {
			q.Set(k, v)
		}
		vr, err := pay.VerifyNotify(context.Background(), &pluginv1.VerifyNotifyRequest{
			Method: "alipay",
			Raw: &pluginv1.RawHttpRequest{
				Method:   "GET",
				RawQuery: q.Encode(),
			},
		})
		if err != nil {
			t.Fatalf("VerifyNotify err (%s): %v", tc.name, err)
		}
		if !vr.GetOk() {
			t.Fatalf("VerifyNotify not ok (%s): %v", tc.name, vr.GetError())
		}
		if vr.GetOrderNo() != "ORDER-123" {
			t.Fatalf("unexpected order_no: %q", vr.GetOrderNo())
		}
		if vr.GetTradeNo() != "PLAT-999" {
			t.Fatalf("unexpected trade_no: %q", vr.GetTradeNo())
		}
		if vr.GetAckBody() != "success" {
			t.Fatalf("unexpected ack_body: %q", vr.GetAckBody())
		}
		if vr.GetAmount() != 1234 {
			t.Fatalf("unexpected amount: %d", vr.GetAmount())
		}
	}
}

func TestEZPay_CreatePayment_RequiresClientIP(t *testing.T) {
	core := &coreServer{}
	core.cfg = config{
		GatewayBaseURL: "https://www.ezfpy.cn",
		SubmitPath:     "mapi.php",
		PID:            "10001",
		MerchantKey:    "testkey",
		SignType:       "MD5",
		SignKeyMode:    "plain",
	}
	pay := &payServer{core: core}

	_, err := pay.CreatePayment(context.Background(), &pluginv1.CreatePaymentRpcRequest{
		Method: "wxpay",
		Request: &pluginv1.PaymentCreateRequest{
			OrderNo:   "ORDER-123",
			UserId:    "1",
			Amount:    1234,
			Currency:  "CNY",
			Subject:   "test",
			NotifyUrl: "https://host.example/api/v1/payments/notify/ezpay.wxpay",
			ReturnUrl: "https://host.example/return",
		},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
	if st.Message() != "client_ip required" {
		t.Fatalf("unexpected error message: %q", st.Message())
	}
}

func TestEZPay_CreatePayment_RequiresDevice(t *testing.T) {
	core := &coreServer{}
	core.cfg = config{
		GatewayBaseURL: "https://www.ezfpy.cn",
		SubmitPath:     "submit.php",
		PID:            "10001",
		MerchantKey:    "testkey",
		SignType:       "MD5",
		SignKeyMode:    "plain",
	}
	pay := &payServer{core: core}
	_, err := pay.CreatePayment(context.Background(), &pluginv1.CreatePaymentRpcRequest{
		Method: "wxpay",
		Request: &pluginv1.PaymentCreateRequest{
			OrderNo:   "ORDER-123",
			UserId:    "1",
			Amount:    1234,
			Currency:  "CNY",
			Subject:   "test",
			NotifyUrl: "https://host.example/api/v1/payments/notify/ezpay.wxpay",
			ReturnUrl: "https://host.example/return",
			Extra:     map[string]string{"client_ip": "127.0.0.1"},
		},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
	if st.Message() != "device required" {
		t.Fatalf("unexpected error message: %q", st.Message())
	}
}

func TestEZPay_VerifyNotify_InvalidSign(t *testing.T) {
	core := &coreServer{}
	core.cfg = config{
		GatewayBaseURL: "https://www.ezfpy.cn",
		PID:            "10001",
		MerchantKey:    "testkey",
		SignType:       "MD5",
	}
	pay := &payServer{core: core}

	params := url.Values{}
	params.Set("pid", core.cfg.PID)
	params.Set("out_trade_no", "ORDER-123-wx")
	params.Set("trade_no", "PLAT-999")
	params.Set("type", "wxpay")
	params.Set("money", "12.34")
	params.Set("trade_status", "TRADE_SUCCESS")
	params.Set("sign_type", "MD5")
	params.Set("sign", "bad")

	_, err := pay.VerifyNotify(context.Background(), &pluginv1.VerifyNotifyRequest{
		Method: "wxpay",
		Raw: &pluginv1.RawHttpRequest{
			Method:   "GET",
			RawQuery: params.Encode(),
		},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}

func TestEZPay_ParseMoneyToCentsStrict(t *testing.T) {
	cases := map[string]int64{
		"1.00":   100,
		"0.01":   1,
		"100":    10000,
		"100.10": 10010,
		"000.10": 10,
		"12.3":   1230,
		"12.30":  1230,
		"12.340": 1234,
	}
	for in, want := range cases {
		got, err := parseMoneyToCentsStrict(in)
		if err != nil {
			t.Fatalf("parseMoneyToCentsStrict(%q) err: %v", in, err)
		}
		if got != want {
			t.Fatalf("parseMoneyToCentsStrict(%q)=%d want %d", in, got, want)
		}
	}
}

func TestEZPay_CreatePayment_MAPI_CacheByOrderAndMethod(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":1,"trade_no":"TN-1","payurl":"https://pay.example/redirect/1"}`))
	}))
	defer ts.Close()

	core := &coreServer{}
	core.cfg = config{
		SubmitURL:      ts.URL + "/mapi.php",
		PID:            "10001",
		MerchantKey:    "testkey",
		SignType:       "MD5",
		SignKeyMode:    "plain",
		TimeoutSec:     5,
		GatewayBaseURL: "",
	}
	pay := &payServer{core: core}

	makeReq := func() *pluginv1.CreatePaymentRpcRequest {
		return &pluginv1.CreatePaymentRpcRequest{
			Method: "wxpay",
			Request: &pluginv1.PaymentCreateRequest{
				OrderNo:   "ORDER-123",
				UserId:    "1",
				Amount:    1234,
				Currency:  "CNY",
				Subject:   "test",
				NotifyUrl: "https://host.example/api/v1/payments/notify/ezpay.wxpay",
				ReturnUrl: "https://host.example/return",
				Extra:     map[string]string{"client_ip": "127.0.0.1", "device": "mobile"},
			},
		}
	}

	first, err := pay.CreatePayment(context.Background(), makeReq())
	if err != nil {
		t.Fatalf("first CreatePayment err: %v", err)
	}
	second, err := pay.CreatePayment(context.Background(), makeReq())
	if err != nil {
		t.Fatalf("second CreatePayment err: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected exactly 1 upstream call, got %d", callCount)
	}
	firstOutNo := first.GetExtra()["out_trade_no"]
	secondOutNo := second.GetExtra()["out_trade_no"]
	if !contains(firstOutNo, "ORDER-123-wx-") {
		t.Fatalf("unexpected first out_trade_no: %q", firstOutNo)
	}
	if firstOutNo != secondOutNo {
		t.Fatalf("expected same out_trade_no in same window, first=%q second=%q", firstOutNo, secondOutNo)
	}
	if first.GetExtra()["pay_url"] != second.GetExtra()["pay_url"] {
		t.Fatalf("expected cached pay_url to match, first=%q second=%q", first.GetExtra()["pay_url"], second.GetExtra()["pay_url"])
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && (stringIndex(s, sub) >= 0))
}

func stringIndex(s, sub string) int {
	// tiny helper to avoid importing strings in tests (keep explicit deps minimal)
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func extractInputValue(html string, name string) string {
	needle := "name=\"" + name + "\" value=\""
	idx := stringIndex(html, needle)
	if idx < 0 {
		return ""
	}
	start := idx + len(needle)
	end := start
	for end < len(html) && html[end] != '"' {
		end++
	}
	if end <= start {
		return ""
	}
	return html[start:end]
}
