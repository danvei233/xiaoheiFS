package main

import (
	"context"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/auth/verifiers"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"

	pluginv1 "xiaoheiplay/plugin/v1"
)

// TestMapWechatTradeState_SurfacesTransactionID covers M3: QueryPayment must
// return the WeChat transaction_id so the host can refund by it.
func TestMapWechatTradeState_SurfacesTransactionID(t *testing.T) {
	state := "SUCCESS"
	transactionID := "42000012022022200000000000000000"
	amount := int64(1234)
	txn := &payments.Transaction{
		TradeState:    &state,
		TransactionId: &transactionID,
		Amount:        &payments.TransactionAmount{Total: &amount},
	}
	ps, gotAmount, gotTradeNo, raw := mapWechatTradeState(txn)
	if gotTradeNo != transactionID {
		t.Fatalf("trade_no = %q, want %q", gotTradeNo, transactionID)
	}
	if gotAmount != amount {
		t.Fatalf("amount = %d, want %d", gotAmount, amount)
	}
	if ps == pluginv1.PaymentStatus_PAYMENT_STATUS_UNSPECIFIED {
		t.Fatalf("status should be mapped")
	}
	if raw == "" {
		t.Fatalf("raw_json empty")
	}
	// nil transaction must not panic and must return empty trade_no.
	_, _, gotTradeNo2, _ := mapWechatTradeState(nil)
	if gotTradeNo2 != "" {
		t.Fatalf("mapWechatTradeState(nil) trade_no = %q, want empty", gotTradeNo2)
	}
}

// TestValidateNotifyIdentity covers L6: a signed WeChat notify must belong to
// the configured appid/mchid when the transaction carries them, so a notify
// for a different app under the same merchant cannot be accepted.
func TestValidateNotifyIdentity(t *testing.T) {
	appid := "wx-app-1"
	mchid := "1234567890"
	cfg := config{AppID: appid, MchID: mchid}

	// Matching identity is accepted.
	txn := &payments.Transaction{Appid: core.String(appid), Mchid: core.String(mchid)}
	if err := validateNotifyIdentity(txn, cfg); err != nil {
		t.Fatalf("validateNotifyIdentity(match) = %v", err)
	}

	// Mismatched appid is rejected.
	txn2 := &payments.Transaction{Appid: core.String("wx-app-other"), Mchid: core.String(mchid)}
	if err := validateNotifyIdentity(txn2, cfg); err == nil {
		t.Fatalf("expected appid mismatch error")
	}

	// Mismatched mchid is rejected.
	txn3 := &payments.Transaction{Appid: core.String(appid), Mchid: core.String("0000000000")}
	if err := validateNotifyIdentity(txn3, cfg); err == nil {
		t.Fatalf("expected mchid mismatch error")
	}

	// Missing fields are skipped (backward compatible).
	txn4 := &payments.Transaction{}
	if err := validateNotifyIdentity(txn4, cfg); err != nil {
		t.Fatalf("validateNotifyIdentity(empty) = %v", err)
	}
}

func TestWeChatPayV3_VerifyNotify_Success(t *testing.T) {
	// platform cert for signature verification (test-only)
	platPriv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &x509.Certificate{
		SerialNumber: big.NewInt(123456),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}, &x509.Certificate{
		SerialNumber: big.NewInt(123456),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}, &platPriv.PublicKey, platPriv)
	if err != nil {
		t.Fatalf("CreateCertificate: %v", err)
	}
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatalf("ParseCertificate: %v", err)
	}
	serial := utils.GetCertificateSerialNumber(*cert)

	apiV3Key := "01234567890123456789012345678901" // 32 bytes
	associatedData := "transaction"
	resourceNonce := "0123456789AB" // 12 bytes

	outTradeNo := "ORDER-123"
	transactionID := "PLAT-999"
	state := "SUCCESS"
	total := int64(1234)
	txn := payments.Transaction{
		OutTradeNo:    &outTradeNo,
		TransactionId: &transactionID,
		TradeState:    &state,
		Amount: &payments.TransactionAmount{
			Total:    &total,
			Currency: core.String("CNY"),
		},
	}
	plain, err := json.Marshal(&txn)
	if err != nil {
		t.Fatalf("Marshal txn: %v", err)
	}

	block, err := aes.NewCipher([]byte(apiV3Key))
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("NewGCM: %v", err)
	}
	ciphertext := gcm.Seal(nil, []byte(resourceNonce), plain, []byte(associatedData))
	ciphertextB64 := base64.StdEncoding.EncodeToString(ciphertext)

	bodyObj := map[string]any{
		"id":            "test-id",
		"create_time":   time.Now().Format(time.RFC3339),
		"resource_type": "encrypt-resource",
		"event_type":    "TRANSACTION.SUCCESS",
		"summary":       "支付成功",
		"resource": map[string]any{
			"original_type":   "transaction",
			"algorithm":       "AEAD_AES_256_GCM",
			"ciphertext":      ciphertextB64,
			"associated_data": associatedData,
			"nonce":           resourceNonce,
		},
	}
	bodyBytes, err := json.Marshal(bodyObj)
	if err != nil {
		t.Fatalf("Marshal body: %v", err)
	}
	body := string(bodyBytes)

	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	nonceHeader := "sig-nonce"
	msg := strings.Join([]string{timestamp, nonceHeader, body, ""}, "\n")
	sum := sha256.Sum256([]byte(msg))
	sig, err := rsa.SignPKCS1v15(rand.Reader, platPriv, crypto.SHA256, sum[:])
	if err != nil {
		t.Fatalf("SignPKCS1v15: %v", err)
	}
	sigB64 := base64.StdEncoding.EncodeToString(sig)

	ver := verifiers.NewSHA256WithRSAVerifier(core.NewCertificateMapWithList([]*x509.Certificate{cert}))
	coreSrv := &coreServer{
		cfg: config{
			APIv3Key:   apiV3Key,
			TimeoutSec: 10,
		},
		verifier: ver,
	}
	pay := &payServer{core: coreSrv}

	res, err := pay.VerifyNotify(context.Background(), &pluginv1.VerifyNotifyRequest{
		Method: "wechat_native",
		Raw: &pluginv1.RawHttpRequest{
			Method: "POST",
			Body:   bodyBytes,
			Headers: map[string]*pluginv1.StringList{
				"Content-Type":        {Values: []string{"application/json"}},
				"Wechatpay-Timestamp": {Values: []string{timestamp}},
				"Wechatpay-Nonce":     {Values: []string{nonceHeader}},
				"Wechatpay-Serial":    {Values: []string{serial}},
				"Wechatpay-Signature": {Values: []string{sigB64}},
			},
		},
	})
	if err != nil {
		t.Fatalf("VerifyNotify err: %v", err)
	}
	if !res.GetOk() {
		t.Fatalf("VerifyNotify not ok: %v", res.GetError())
	}
	if res.GetTradeNo() != transactionID {
		t.Fatalf("unexpected trade_no: %q", res.GetTradeNo())
	}
	if res.GetOrderNo() != outTradeNo {
		t.Fatalf("unexpected order_no: %q", res.GetOrderNo())
	}
	if res.GetAckBody() == "" {
		t.Fatalf("ack_body empty")
	}
}
