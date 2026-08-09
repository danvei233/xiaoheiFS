package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/testutilhttp"
)

func TestHandlers_CatalogCartOrderFlow(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	seed := testutil.SeedCatalog(t, env.Repo)
	user := testutil.CreateUser(t, env.Repo, "buyer", "buyer@example.com", "pass")
	token := testutil.IssueJWT(t, env.JWTSecret, user.ID, "user", time.Hour)

	if err := env.Repo.CreateBillingCycle(context.Background(), &domain.BillingCycle{Name: "monthly", Months: 1, Multiplier: 1, MinQty: 1, MaxQty: 12, Active: true, SortOrder: 1}); err != nil {
		t.Fatalf("create cycle: %v", err)
	}

	rec := testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/catalog", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("catalog code: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/cart", map[string]any{
		"package_id": seed.Package.ID,
		"system_id":  seed.SystemImage.ID,
		"spec":       map[string]any{},
		"qty":        1,
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("cart add code: %d", rec.Code)
	}
	var cartResp struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &cartResp); err != nil {
		t.Fatalf("decode cart: %v", err)
	}
	if cartResp.ID == 0 {
		t.Fatalf("expected cart id")
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/api/v1/cart/"+testutil.Itoa(cartResp.ID), map[string]any{
		"spec": map[string]any{"add_cores": 1},
		"qty":  2,
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("cart update code: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/cart", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("cart list code: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodDelete, "/api/v1/cart/"+testutil.Itoa(cartResp.ID), nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("cart delete code: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodDelete, "/api/v1/cart", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("cart clear code: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/orders/items", map[string]any{
		"items": []map[string]any{
			{"package_id": seed.Package.ID, "system_id": seed.SystemImage.ID, "qty": 1},
		},
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("order create code: %d", rec.Code)
	}
	var orderResp struct {
		Order struct {
			ID int64 `json:"id"`
		} `json:"order"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &orderResp); err != nil {
		t.Fatalf("decode order: %v", err)
	}
	if orderResp.Order.ID == 0 {
		t.Fatalf("expected order id")
	}

	if _, err := env.Repo.AdjustWalletBalance(context.Background(), user.ID, 100, "credit", "seed", 1, "init"); err != nil {
		t.Fatalf("seed wallet: %v", err)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/orders/"+testutil.Itoa(orderResp.Order.ID)+"/pay", map[string]any{
		"method": "balance",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("order pay code: %d", rec.Code)
	}
}

func TestHandlers_PaymentNotifyIdempotent(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	user := testutil.CreateUser(t, env.Repo, "n1", "n1@example.com", "pass")
	order := domain.Order{UserID: user.ID, OrderNo: "ORD-1", Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "CNY"}
	if err := env.Repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	if err := env.Repo.CreateOrderItems(context.Background(), []domain.OrderItem{{
		OrderID:  order.ID,
		Amount:   1000,
		Status:   domain.OrderItemStatusPendingPayment,
		Action:   "create",
		SpecJSON: "{}",
	}}); err != nil {
		t.Fatalf("create items: %v", err)
	}
	if err := env.Repo.CreatePayment(context.Background(), &domain.OrderPayment{
		OrderID:  order.ID,
		UserID:   user.ID,
		Method:   "fake",
		Amount:   1000,
		Currency: "CNY",
		TradeNo:  "TN-1",
		Status:   domain.PaymentStatusPendingPayment,
	}); err != nil {
		t.Fatalf("create payment: %v", err)
	}
	env.PaymentReg.RegisterProvider(&testutil.FakePaymentProvider{
		KeyVal:    "fake",
		NameVal:   "Fake",
		VerifyRes: shared.PaymentNotifyResult{TradeNo: "TN-1", Paid: true, Amount: 1000},
	}, true, "")

	form := "trade_no=TN-1"
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/notify/fake", bytes.NewBufferString(form))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		env.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("notify code: %d", rec.Code)
		}
	}

	updated, err := env.Repo.GetPaymentByTradeNo(context.Background(), "TN-1")
	if err != nil {
		t.Fatalf("get payment: %v", err)
	}
	if updated.Status != domain.PaymentStatusApproved {
		t.Fatalf("expected approved")
	}
}

func TestHandlers_WalletPaymentNotifyIdempotent(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	user := testutil.CreateUser(t, env.Repo, "wn1", "wn1@example.com", "pass")
	order := domain.WalletOrder{
		UserID:   user.ID,
		Type:     domain.WalletOrderRecharge,
		Amount:   1200,
		Currency: "CNY",
		Status:   domain.WalletOrderPendingReview,
		MetaJSON: `{"payment_method":"fake","payment_order_no":"WO-1","payment_trade_no":"WT-1"}`,
		Note:     "recharge",
	}
	if err := env.Repo.CreateWalletOrder(context.Background(), &order); err != nil {
		t.Fatalf("create wallet order: %v", err)
	}
	env.PaymentReg.RegisterProvider(&testutil.FakePaymentProvider{
		KeyVal:  "fake",
		NameVal: "Fake",
		VerifyRes: shared.PaymentNotifyResult{
			OrderNo: "WO-1",
			TradeNo: "WT-1",
			Paid:    true,
			Amount:  1200,
		},
	}, true, "")

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet/payments/notify/fake", bytes.NewBufferString("trade_no=WT-1"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		env.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("notify code: %d", rec.Code)
		}
	}

	updated, err := env.Repo.GetWalletOrder(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("get wallet order: %v", err)
	}
	if updated.Status != domain.WalletOrderApproved {
		t.Fatalf("expected approved status, got=%s", updated.Status)
	}
}

func TestHandlers_WalletPaymentNotifyDoesNotAcknowledgeRejectedOrder(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	user := testutil.CreateUser(t, env.Repo, "wn-rejected", "wn-rejected@example.com", "pass")
	order := domain.WalletOrder{
		UserID:   user.ID,
		Type:     domain.WalletOrderRecharge,
		Amount:   1200,
		Currency: "CNY",
		Status:   domain.WalletOrderRejected,
		MetaJSON: `{"payment_method":"fake","payment_order_no":"WO-REJECTED","payment_trade_no":"WT-REJECTED"}`,
		Note:     "rejected recharge",
	}
	if err := env.Repo.CreateWalletOrder(context.Background(), &order); err != nil {
		t.Fatalf("create wallet order: %v", err)
	}
	env.PaymentReg.RegisterProvider(&testutil.FakePaymentProvider{
		KeyVal:  "fake",
		NameVal: "Fake",
		VerifyRes: shared.PaymentNotifyResult{
			OrderNo: "WO-REJECTED",
			TradeNo: "WT-REJECTED",
			Paid:    true,
			Amount:  1200,
		},
	}, true, "")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet/payments/notify/fake", bytes.NewBufferString("trade_no=WT-REJECTED"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	env.Router.ServeHTTP(rec, req)
	if rec.Code == http.StatusOK {
		t.Fatalf("rejected wallet order callback must not be acknowledged")
	}

	updated, err := env.Repo.GetWalletOrder(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("get wallet order: %v", err)
	}
	if updated.Status != domain.WalletOrderRejected {
		t.Fatalf("expected rejected status, got=%s", updated.Status)
	}
}

func TestHandlers_WalletPaymentNotify_MatchByDerivedOrderNo(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	user := testutil.CreateUser(t, env.Repo, "wn2", "wn2@example.com", "pass")
	order := domain.WalletOrder{
		UserID:   user.ID,
		Type:     domain.WalletOrderRecharge,
		Amount:   900,
		Currency: "CNY",
		Status:   domain.WalletOrderPendingReview,
		MetaJSON: `{"payment_method":"fake"}`,
		Note:     "retry pay",
	}
	if err := env.Repo.CreateWalletOrder(context.Background(), &order); err != nil {
		t.Fatalf("create wallet order: %v", err)
	}
	derivedOrderNo := fmt.Sprintf("WALLET-ORDER-%d", order.ID)
	env.PaymentReg.RegisterProvider(&testutil.FakePaymentProvider{
		KeyVal:  "fake",
		NameVal: "Fake",
		VerifyRes: shared.PaymentNotifyResult{
			OrderNo: derivedOrderNo,
			Paid:    true,
			Amount:  900,
		},
	}, true, "")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet/payments/notify/fake", bytes.NewBufferString("out_trade_no="+derivedOrderNo))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	env.Router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("notify code: %d", rec.Code)
	}

	updated, err := env.Repo.GetWalletOrder(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("get wallet order: %v", err)
	}
	if updated.Status != domain.WalletOrderApproved {
		t.Fatalf("expected approved status, got=%s", updated.Status)
	}
}
