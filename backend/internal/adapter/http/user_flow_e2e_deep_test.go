package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/testutilhttp"
)

func TestUserFlow_E2E_Deep(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	seed := testutil.SeedCatalog(t, env.Repo)
	if err := env.Repo.CreateBillingCycle(context.Background(), &domain.BillingCycle{Name: "monthly", Months: 1, Multiplier: 1, MinQty: 1, MaxQty: 12, Active: true, SortOrder: 1}); err != nil {
		t.Fatalf("create billing cycle: %v", err)
	}
	adminToken := createAdminToken(t, env)

	env.PaymentReg.RegisterProvider(&testutil.FakePaymentProvider{KeyVal: "balance", NameVal: "Balance"}, true, "")
	methods := testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/payments/providers", nil, "")
	if methods.Code != http.StatusUnauthorized {
		t.Fatalf("methods unauthorized: %d", methods.Code)
	}

	captcha, code, err := env.AuthSvc.CreateCaptcha(context.Background(), time.Minute)
	if err != nil {
		t.Fatalf("captcha: %v", err)
	}
	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"username":     "deep",
		"email":        "deep@example.com",
		"password":     "pass123",
		"captcha_id":   captcha.ID,
		"captcha_code": code,
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("register: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"username": "deep",
		"password": "pass123",
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login: %d", rec.Code)
	}
	var loginResp struct {
		AccessToken string `json:"access_token"`
		User        struct {
			ID int64 `json:"id"`
		} `json:"user"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	token := loginResp.AccessToken
	userID := loginResp.User.ID

	methods = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/payments/providers", nil, token)
	if methods.Code != http.StatusOK {
		t.Fatalf("methods: %d", methods.Code)
	}
	var methodResp struct {
		Items []struct {
			Key string `json:"key"`
		} `json:"items"`
	}
	_ = json.Unmarshal(methods.Body.Bytes(), &methodResp)
	if len(methodResp.Items) == 0 {
		t.Fatalf("expected payment methods")
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/cart", map[string]any{
		"package_id": seed.Package.ID,
		"system_id":  seed.SystemImage.ID,
		"spec":       map[string]any{},
		"qty":        1,
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("cart add: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/orders", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("create order: %d", rec.Code)
	}
	var orderResp struct {
		Order struct {
			ID int64 `json:"id"`
		} `json:"order"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &orderResp); err != nil {
		t.Fatalf("decode order: %v", err)
	}
	orderID := orderResp.Order.ID

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/wallet/recharge", map[string]any{
		"amount":   100,
		"currency": "CNY",
		"note":     "topup",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("wallet recharge: %d", rec.Code)
	}
	var walletOrderResp struct {
		Order struct {
			ID int64 `json:"id"`
		} `json:"order"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &walletOrderResp)
	approve := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/wallet/orders/"+testutil.Itoa(walletOrderResp.Order.ID)+"/approve", nil, adminToken)
	if approve.Code != http.StatusOK {
		t.Fatalf("wallet approve: %d", approve.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/orders/"+testutil.Itoa(orderID)+"/pay", map[string]any{
		"method": "balance",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("pay: %d", rec.Code)
	}
	if _, err := env.Repo.AppendEvent(context.Background(), orderID, "order.paid", `{"ok":true}`); err != nil {
		t.Fatalf("append event: %v", err)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/orders", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("orders list: %d", rec.Code)
	}
	var listResp struct {
		Items []struct {
			ID int64 `json:"id"`
		} `json:"items"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("decode orders list: %v", err)
	}
	if listResp.Total == 0 || len(listResp.Items) == 0 {
		t.Fatalf("orders list empty")
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/orders/"+testutil.Itoa(orderID), nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("order detail: %d", rec.Code)
	}
	var detail struct {
		Order struct {
			ID int64 `json:"id"`
		} `json:"order"`
		Payments []struct {
			ID int64 `json:"id"`
		} `json:"payments"`
		Items []struct {
			ID int64 `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode order detail: %v", err)
	}
	if detail.Order.ID != orderID || len(detail.Payments) == 0 || len(detail.Items) == 0 {
		t.Fatalf("order detail missing data")
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/wallet/transactions", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("wallet transactions: %d", rec.Code)
	}
	var txResp struct {
		Items []struct {
			ID int64 `json:"id"`
		} `json:"items"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &txResp)
	if len(txResp.Items) == 0 {
		t.Fatalf("wallet transactions empty")
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/wallet/orders", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("wallet orders: %d", rec.Code)
	}
	var woResp struct {
		Items []struct {
			ID int64 `json:"id"`
		} `json:"items"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &woResp)
	if len(woResp.Items) == 0 {
		t.Fatalf("wallet orders empty")
	}

	record := domain.RealNameVerification{UserID: userID, RealName: "Test", IDNumber: "11010519491231002X", Status: "verified", Provider: "idcard_cn"}
	if err := env.Repo.CreateRealNameVerification(context.Background(), &record); err != nil {
		t.Fatalf("realname record: %v", err)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/realname/status", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("realname status: %d", rec.Code)
	}
	var rnResp struct {
		Verification map[string]any `json:"verification"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &rnResp)
	if rnResp.Verification == nil {
		t.Fatalf("expected verification")
	}
}
