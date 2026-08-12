package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/testutilhttp"
)

func TestPaymentProviders_UserResponseWhitelistsPublicFields(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	user := testutil.CreateUser(t, env.Repo, "provider-user", "provider-user@example.com", "pass")
	userToken := testutil.IssueJWT(t, env.JWTSecret, user.ID, "user", time.Hour)
	group := domain.PermissionGroup{Name: "payment-admin", PermissionsJSON: `["*"]`}
	if err := env.Repo.CreatePermissionGroup(context.Background(), &group); err != nil {
		t.Fatalf("create permission group: %v", err)
	}
	admin := testutil.CreateAdmin(t, env.Repo, "provider-admin", "provider-admin@example.com", "pass", group.ID)
	adminToken := testutil.IssueJWT(t, env.JWTSecret, admin.ID, "admin", time.Hour)

	const schema = `{"type":"object","properties":{"merchant_secret":{"type":"string"}}}`
	const config = `{"merchant_id":"merchant-1","merchant_secret":"secret-value","instructions":"Scan to pay","notice":"Settlement may be delayed"}`
	env.PaymentReg.RegisterProvider(&testutil.FakePaymentProvider{
		KeyVal:  "secure-pay",
		NameVal: "Secure Pay",
		Schema:  schema,
	}, true, config)

	userResponse := testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/payments/providers", nil, userToken)
	if userResponse.Code != http.StatusOK {
		t.Fatalf("user providers: code=%d body=%s", userResponse.Code, userResponse.Body.String())
	}
	var userPayload struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(userResponse.Body.Bytes(), &userPayload); err != nil {
		t.Fatalf("decode user providers: %v", err)
	}
	if len(userPayload.Items) != 1 {
		t.Fatalf("user provider count: %d", len(userPayload.Items))
	}
	item := userPayload.Items[0]
	if _, ok := item["schema_json"]; ok {
		t.Fatal("user response exposed provider schema")
	}
	if _, ok := item["config_json"]; ok {
		t.Fatal("user response exposed provider config")
	}
	if item["instructions"] != "Scan to pay" || item["notice"] != "Settlement may be delayed" {
		t.Fatalf("unexpected public fields: %#v", item)
	}
	if body := userResponse.Body.String(); strings.Contains(body, "merchant_secret") || strings.Contains(body, "secret-value") {
		t.Fatalf("user response exposed merchant secret: %s", body)
	}

	adminResponse := testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/payments/providers", nil, adminToken)
	if adminResponse.Code != http.StatusOK {
		t.Fatalf("admin providers: code=%d body=%s", adminResponse.Code, adminResponse.Body.String())
	}
	adminBody := adminResponse.Body.String()
	// The admin list must keep the schema and the config shape, but a secret
	// field (merchant_secret) is masked (P2-13) so a read-only admin session
	// does not echo the channel ciphertext.
	if !strings.Contains(adminBody, "schema_json") || !strings.Contains(adminBody, "config_json") || !strings.Contains(adminBody, "merchant_id") {
		t.Fatalf("admin response lost provider configuration shape: %s", adminBody)
	}
	if strings.Contains(adminBody, "secret-value") {
		t.Fatalf("admin response leaked the merchant secret: %s", adminBody)
	}
	if !strings.Contains(adminBody, "********") {
		t.Fatalf("admin response must mask the merchant secret: %s", adminBody)
	}
}

func TestWalletPaymentNotifyRejectsAmountMismatchWithoutCredit(t *testing.T) {
	tests := []struct {
		name   string
		amount int64
	}{
		{name: "underpayment", amount: 1199},
		{name: "zero amount", amount: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := testutilhttp.NewTestEnv(t, false)
			user := testutil.CreateUser(t, env.Repo, "wallet-amount", "wallet-amount@example.com", "pass")
			if _, err := env.Repo.AdjustWalletBalance(context.Background(), user.ID, 500, "credit", "seed", 1, "initial"); err != nil {
				t.Fatalf("seed wallet: %v", err)
			}
			order := domain.WalletOrder{
				UserID:   user.ID,
				Type:     domain.WalletOrderRecharge,
				Amount:   1200,
				Currency: "CNY",
				Status:   domain.WalletOrderPendingReview,
				MetaJSON: `{"payment_method":"fake","payment_order_no":"WO-AMOUNT","payment_trade_no":"WT-AMOUNT"}`,
			}
			if err := env.Repo.CreateWalletOrder(context.Background(), &order); err != nil {
				t.Fatalf("create wallet order: %v", err)
			}
			env.PaymentReg.RegisterProvider(&testutil.FakePaymentProvider{
				KeyVal:  "fake",
				NameVal: "Fake",
				VerifyRes: appshared.PaymentNotifyResult{
					OrderNo: "WO-AMOUNT",
					TradeNo: "WT-AMOUNT",
					Paid:    true,
					Amount:  tt.amount,
				},
			}, true, "")

			req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet/payments/notify/fake", strings.NewReader("trade_no=WT-AMOUNT"))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rec := httptest.NewRecorder()
			env.Router.ServeHTTP(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("notify code=%d body=%s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), domain.ErrPaymentAmountMismatch.Error()) {
				t.Fatalf("unexpected notify error: %s", rec.Body.String())
			}

			updatedOrder, err := env.Repo.GetWalletOrder(context.Background(), order.ID)
			if err != nil {
				t.Fatalf("get wallet order: %v", err)
			}
			if updatedOrder.Status != domain.WalletOrderPendingReview {
				t.Fatalf("wallet order status changed: %s", updatedOrder.Status)
			}
			wallet, err := env.Repo.GetWallet(context.Background(), user.ID)
			if err != nil {
				t.Fatalf("get wallet: %v", err)
			}
			if wallet.Balance != 500 {
				t.Fatalf("wallet balance changed: %d", wallet.Balance)
			}
		})
	}
}

func TestPublicPaymentCallbacksRejectOversizedBody(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "order", path: "/api/v1/payments/notify/fake"},
		{name: "wallet", path: "/api/v1/wallet/payments/notify/fake"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := testutilhttp.NewTestEnv(t, false)
			providerCalled := false
			env.PaymentReg.RegisterProvider(&testutil.FakePaymentProvider{
				KeyVal:  "fake",
				NameVal: "Fake",
				VerifyFunc: func(req appshared.RawHTTPRequest) (appshared.PaymentNotifyResult, error) {
					providerCalled = true
					return appshared.PaymentNotifyResult{}, nil
				},
			}, true, "")

			body := bytes.Repeat([]byte("x"), 2<<20)
			req := httptest.NewRequest(http.MethodPost, tt.path, bytes.NewReader(body))
			rec := httptest.NewRecorder()
			env.Router.ServeHTTP(rec, req)
			if rec.Code != http.StatusRequestEntityTooLarge {
				t.Fatalf("callback code=%d body=%s", rec.Code, rec.Body.String())
			}
			if providerCalled {
				t.Fatal("provider was called for oversized request body")
			}
			if !strings.Contains(rec.Body.String(), domain.ErrRequestBodyTooLarge.Error()) {
				t.Fatalf("callback did not return controlled error: %s", rec.Body.String())
			}
		})
	}
}
