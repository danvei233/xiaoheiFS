package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/alipay"

	pluginv1 "xiaoheiplay/plugin/v1"
)

// TestNormalizeKeyPEMOrBase64_StripsPEMHeaders guards against regression of the
// "double-wrapped PEM" failure (M1): gopay's FormatAlipayPublicKey/PrivateKey
// unconditionally wraps the passed key in a PEM block, so any key that still
// carries PEM headers gets wrapped a second time and cannot be parsed by
// xpem.Decode*.
func TestNormalizeKeyPEMOrBase64_StripsPEMHeaders(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	pubDER, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatalf("MarshalPKIXPublicKey: %v", err)
	}
	pubB64 := base64.StdEncoding.EncodeToString(pubDER)

	pemKey := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}))
	got := normalizeKeyPEMOrBase64(pemKey)
	if got != pubB64 {
		t.Fatalf("normalizeKeyPEMOrBase64(PEM) = %q, want bare base64 %q", got, pubB64)
	}

	// Bare base64 body must be returned unchanged (no extra wrapping).
	if got := normalizeKeyPEMOrBase64(pubB64); got != pubB64 {
		t.Fatalf("normalizeKeyPEMOrBase64(base64) = %q, want unchanged %q", got, pubB64)
	}
}

// TestAlipay_VerifyNotify_PEMPublicKey ensures VerifyNotify still succeeds when
// the operator configures the Alipay public key with PEM headers (M1). Before
// the fix, gopay double-wraps the key and verification always fails.
func TestAlipay_VerifyNotify_PEMPublicKey(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	pubDER, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatalf("MarshalPKIXPublicKey: %v", err)
	}
	pemPub := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}))

	appID := "2026000000000000"
	sellerID := "2088000000000000"

	core := &coreServer{
		cfg: config{
			AppID:        appID,
			AliPublicKey: normalizeKeyPEMOrBase64(pemPub),
			SellerID:     sellerID,
		},
	}
	pay := &payServer{core: core}

	orderNo := "ORDER-123"
	aliTradeNo := "2026012900000000000000000000"

	bmSign := gopay.BodyMap{}
	bmSign.Set("out_trade_no", orderNo).
		Set("trade_no", aliTradeNo).
		Set("trade_status", "TRADE_SUCCESS").
		Set("total_amount", "12.34").
		Set("app_id", appID).
		Set("seller_id", sellerID)

	sign, err := alipay.GetRsaSign(bmSign, "RSA2", priv)
	if err != nil {
		t.Fatalf("GetRsaSign: %v", err)
	}
	bmSign.Set("sign_type", "RSA2")
	bmSign.Set("sign", sign)

	q := url.Values{}
	for k, v := range bmSign {
		q.Set(k, fmt.Sprint(v))
	}

	vr, err := pay.VerifyNotify(context.Background(), &pluginv1.VerifyNotifyRequest{
		Method: "alipay_wap",
		Raw: &pluginv1.RawHttpRequest{
			Method: "POST",
			Body:   []byte(q.Encode()),
			Headers: map[string]*pluginv1.StringList{
				"Content-Type": {Values: []string{"application/x-www-form-urlencoded"}},
			},
		},
	})
	if err != nil {
		t.Fatalf("VerifyNotify err: %v", err)
	}
	if !vr.GetOk() {
		t.Fatalf("VerifyNotify not ok: %v", vr.GetError())
	}
	if vr.GetOrderNo() != orderNo {
		t.Fatalf("unexpected order_no: %q", vr.GetOrderNo())
	}
}

// TestParseGMTTime_IsAsiaShanghai pins the timezone semantics of the Alipay
// gmt_payment field (M2). Alipay sends Asia/Shanghai (UTC+8) wall-clock time;
// parsing it with the host's local timezone shifts paid_at_unix on non-CST
// machines.
func TestParseGMTTime_IsAsiaShanghai(t *testing.T) {
	// Alipay gmt_payment sample: 2026-08-11 15:04:05 (Asia/Shanghai wall clock).
	got := parseGMTTime("2026-08-11 15:04:05")
	want := time.Date(2026, 8, 11, 15, 4, 5, 0, cstZone).Unix()
	if got != want {
		t.Fatalf("parseGMTTime = %d, want %d (Asia/Shanghai)", got, want)
	}
	if got == 0 {
		t.Fatalf("parseGMTTime returned 0 for a valid timestamp")
	}
	// Empty / malformed input must not panic and must return 0.
	if got := parseGMTTime(""); got != 0 {
		t.Fatalf("parseGMTTime(\"\") = %d, want 0", got)
	}
	if got := parseGMTTime("not-a-time"); got != 0 {
		t.Fatalf("parseGMTTime(invalid) = %d, want 0", got)
	}
}

func TestAlipay_VerifyNotify_RSA2(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	pubDER, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatalf("MarshalPKIXPublicKey: %v", err)
	}
	pubB64 := base64.StdEncoding.EncodeToString(pubDER)

	appID := "2026000000000000"
	sellerID := "2088000000000000"

	core := &coreServer{
		cfg: config{
			AppID:        appID,
			AliPublicKey: pubB64,
			SellerID:     sellerID,
		},
	}
	pay := &payServer{core: core}

	orderNo := "ORDER-123"
	aliTradeNo := "2026012900000000000000000000"

	// Signature string does NOT include sign/sign_type.
	bmSign := gopay.BodyMap{}
	bmSign.Set("out_trade_no", orderNo).
		Set("trade_no", aliTradeNo).
		Set("trade_status", "TRADE_SUCCESS").
		Set("total_amount", "12.34").
		Set("app_id", appID).
		Set("seller_id", sellerID)

	sign, err := alipay.GetRsaSign(bmSign, "RSA2", priv)
	if err != nil {
		t.Fatalf("GetRsaSign: %v", err)
	}
	bmSign.Set("sign_type", "RSA2")
	bmSign.Set("sign", sign)

	q := url.Values{}
	for k, v := range bmSign {
		q.Set(k, fmt.Sprint(v))
	}

	vr, err := pay.VerifyNotify(context.Background(), &pluginv1.VerifyNotifyRequest{
		Method: "alipay_wap",
		Raw: &pluginv1.RawHttpRequest{
			Method: "POST",
			Body:   []byte(q.Encode()),
			Headers: map[string]*pluginv1.StringList{
				"Content-Type": {Values: []string{"application/x-www-form-urlencoded"}},
			},
		},
	})
	if err != nil {
		t.Fatalf("VerifyNotify err: %v", err)
	}
	if !vr.GetOk() {
		t.Fatalf("VerifyNotify not ok: %v", vr.GetError())
	}
	if vr.GetTradeNo() != aliTradeNo {
		t.Fatalf("unexpected trade_no: %q", vr.GetTradeNo())
	}
	if vr.GetOrderNo() != orderNo {
		t.Fatalf("unexpected order_no: %q", vr.GetOrderNo())
	}
	if vr.GetAckBody() != "success" {
		t.Fatalf("unexpected ack_body: %q", vr.GetAckBody())
	}
}
