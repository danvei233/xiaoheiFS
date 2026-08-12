package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
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

// capturingPaymentProvider records the last CreatePayment request so tests can
// assert which fields the handler forwarded to the payment gateway.
type capturingPaymentProvider struct {
	mu      sync.Mutex
	key     string
	name    string
	lastReq *shared.PaymentCreateRequest
}

func (p *capturingPaymentProvider) Key() string        { return p.key }
func (p *capturingPaymentProvider) Name() string       { return p.name }
func (p *capturingPaymentProvider) SchemaJSON() string { return "" }
func (p *capturingPaymentProvider) CreatePayment(ctx context.Context, req shared.PaymentCreateRequest) (shared.PaymentCreateResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	cp := req
	p.lastReq = &cp
	return shared.PaymentCreateResult{TradeNo: "CAP-TN-1", PayURL: "https://pay.example/cap"}, nil
}
func (p *capturingPaymentProvider) VerifyNotify(ctx context.Context, req shared.RawHTTPRequest) (shared.PaymentNotifyResult, error) {
	return shared.PaymentNotifyResult{}, nil
}

func (p *capturingPaymentProvider) Last() *shared.PaymentCreateRequest {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.lastReq == nil {
		return nil
	}
	cp := *p.lastReq
	return &cp
}

// Regression: an authenticated user must not be able to create an order whose
// per-line qty expands into an unbounded number of order rows (OOM / DoS).
func TestHandlers_OrderCreateItemsRejectsOversizedQty(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	seed := testutil.SeedCatalog(t, env.Repo)
	user := testutil.CreateUser(t, env.Repo, "qty", "qty@example.com", "pass")
	token := testutil.IssueJWT(t, env.JWTSecret, user.ID, "user", time.Hour)

	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/orders/items", map[string]any{
		"items": []map[string]any{
			{"package_id": seed.Package.ID, "system_id": seed.SystemImage.ID, "qty": 1001},
		},
	}, token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("oversized qty: want 400, got %d (body=%s)", rec.Code, rec.Body.String())
	}
}

// Regression: the items array length must be bounded to avoid request fan-out.
func TestHandlers_OrderCreateItemsRejectsOversizedItemsArray(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	seed := testutil.SeedCatalog(t, env.Repo)
	user := testutil.CreateUser(t, env.Repo, "many", "many@example.com", "pass")
	token := testutil.IssueJWT(t, env.JWTSecret, user.ID, "user", time.Hour)

	items := make([]map[string]any, 0, 101)
	for i := 0; i < 101; i++ {
		items = append(items, map[string]any{"package_id": seed.Package.ID, "system_id": seed.SystemImage.ID, "qty": 1})
	}
	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/orders/items", map[string]any{
		"items": items,
	}, token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("oversized items array: want 400, got %d", rec.Code)
	}
}

// A zero qty has always been treated as qty=1 by the order service; the
// handler validation must not regress that for existing clients.
func TestHandlers_OrderCreateItemsZeroQtyDefaultsToOne(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	seed := testutil.SeedCatalog(t, env.Repo)
	user := testutil.CreateUser(t, env.Repo, "qty0", "qty0@example.com", "pass")
	token := testutil.IssueJWT(t, env.JWTSecret, user.ID, "user", time.Hour)

	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/orders/items", map[string]any{
		"items": []map[string]any{
			{"package_id": seed.Package.ID, "system_id": seed.SystemImage.ID, "qty": 0},
		},
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("zero qty: want 200, got %d (body=%s)", rec.Code, rec.Body.String())
	}
	var resp struct {
		Items []json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("zero qty should expand to 1 item, got %d", len(resp.Items))
	}
}

// Regression: POST /orders with a chunked (ContentLength == -1) JSON body must
// bind the line items instead of silently falling back to "order from cart".
func TestHandlers_OrderCreateBindsChunkedBody(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	seed := testutil.SeedCatalog(t, env.Repo)
	user := testutil.CreateUser(t, env.Repo, "chunk", "chunk@example.com", "pass")
	token := testutil.IssueJWT(t, env.JWTSecret, user.ID, "user", time.Hour)

	body := fmt.Sprintf(`{"items":[{"package_id":%d,"system_id":%d,"qty":1}]}`, seed.Package.ID, seed.SystemImage.ID)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.ContentLength = -1 // chunked transfer encoding
	rec := httptest.NewRecorder()
	env.Router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("chunked body with items: want 200, got %d (body=%s)", rec.Code, rec.Body.String())
	}
}

// Regression: OrderPay must not let the client spoof the client_ip/device
// forwarded to the payment gateway; the server-derived values must win.
func TestHandlers_OrderPayUsesServerClientIP(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	user := testutil.CreateUser(t, env.Repo, "payip", "payip@example.com", "pass")
	token := testutil.IssueJWT(t, env.JWTSecret, user.ID, "user", time.Hour)
	order := domain.Order{UserID: user.ID, OrderNo: "ORD-PAYIP", Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "CNY"}
	if err := env.Repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	prov := &capturingPaymentProvider{key: "capture", name: "Capture"}
	env.PaymentReg.RegisterProvider(prov, true, "")

	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/orders/"+testutil.Itoa(order.ID)+"/pay", map[string]any{
		"method": "capture",
		"extra":  map[string]string{"client_ip": "1.2.3.4", "device": "pc"},
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("order pay: want 200, got %d (body=%s)", rec.Code, rec.Body.String())
	}
	got := prov.Last()
	if got == nil {
		t.Fatal("no provider payment created")
	}
	if got.Extra["client_ip"] == "1.2.3.4" {
		t.Errorf("client_ip must be server-derived, got spoofed value %q", got.Extra["client_ip"])
	}
	if got.Extra["device"] == "pc" {
		t.Errorf("device must be server-derived, got spoofed value %q", got.Extra["device"])
	}
}

// Regression: WalletRecharge must not forward a client-supplied client_ip.
func TestHandlers_WalletRechargeUsesServerClientIP(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	user := testutil.CreateUser(t, env.Repo, "wrip", "wrip@example.com", "pass")
	token := testutil.IssueJWT(t, env.JWTSecret, user.ID, "user", time.Hour)
	prov := &capturingPaymentProvider{key: "capture", name: "Capture"}
	env.PaymentReg.RegisterProvider(prov, true, "")

	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/wallet/recharge", map[string]any{
		"amount": 1000,
		"method": "capture",
		"extra":  map[string]string{"client_ip": "1.2.3.4"},
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("wallet recharge: want 200, got %d (body=%s)", rec.Code, rec.Body.String())
	}
	got := prov.Last()
	if got == nil {
		t.Fatal("no provider payment created")
	}
	if got.Extra["client_ip"] == "1.2.3.4" {
		t.Errorf("client_ip must be server-derived, got spoofed value %q", got.Extra["client_ip"])
	}
}

// Regression: WalletOrderPay must not forward a client-supplied client_ip.
func TestHandlers_WalletOrderPayUsesServerClientIP(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	user := testutil.CreateUser(t, env.Repo, "wopip", "wopip@example.com", "pass")
	token := testutil.IssueJWT(t, env.JWTSecret, user.ID, "user", time.Hour)
	prov := &capturingPaymentProvider{key: "capture", name: "Capture"}
	env.PaymentReg.RegisterProvider(prov, true, "")
	order := domain.WalletOrder{
		UserID:   user.ID,
		Type:     domain.WalletOrderRecharge,
		Amount:   100000,
		Currency: "CNY",
		Status:   domain.WalletOrderPendingReview,
		MetaJSON: `{"payment_method":"capture"}`,
		Note:     "recharge",
	}
	if err := env.Repo.CreateWalletOrder(context.Background(), &order); err != nil {
		t.Fatalf("create wallet order: %v", err)
	}

	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/wallet/orders/"+testutil.Itoa(order.ID)+"/pay", map[string]any{
		"extra": map[string]string{"client_ip": "1.2.3.4"},
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("wallet order pay: want 200, got %d (body=%s)", rec.Code, rec.Body.String())
	}
	got := prov.Last()
	if got == nil {
		t.Fatal("no provider payment created")
	}
	if got.Extra["client_ip"] == "1.2.3.4" {
		t.Errorf("client_ip must be server-derived, got spoofed value %q", got.Extra["client_ip"])
	}
}

// TestHandlers_WalletPaymentNotify_FindsOrderBeyondScanLimit locks the P1-4
// fix: the notify handler previously resolved recharge orders by scanning
// paginated lists capped at 4000 rows per status, so a recharge beyond that
// window (or deep in a large table) was never found and the gateway retried
// forever. The server-derived WALLET-ORDER-<id> order number must resolve the
// target order by ID directly.
func TestHandlers_WalletPaymentNotify_FindsOrderBeyondScanLimit(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	user := testutil.CreateUser(t, env.Repo, "wn_deep", "wn_deep@example.com", "pass")

	ctx := context.Background()
	// The target is the OLDEST order (lowest id). The old notify path scanned
	// newest-first (id DESC) with a 4000-row cap per status, so an old order
	// beyond that window was never found. Create the target first, then more
	// than 4000 filler orders so the scan cap is exceeded before reaching it.
	target := domain.WalletOrder{
		UserID:   user.ID,
		Type:     domain.WalletOrderRecharge,
		Amount:   777,
		Currency: "CNY",
		Status:   domain.WalletOrderPendingReview,
		MetaJSON: `{"payment_method":"fake"}`,
		Note:     "deep target",
	}
	if err := env.Repo.CreateWalletOrder(ctx, &target); err != nil {
		t.Fatalf("create target wallet order: %v", err)
	}
	for i := 0; i < 4010; i++ {
		order := domain.WalletOrder{
			UserID:   user.ID,
			Type:     domain.WalletOrderRecharge,
			Amount:   100,
			Currency: "CNY",
			Status:   domain.WalletOrderPendingReview,
			MetaJSON: `{"payment_method":"fake"}`,
			Note:     "bulk",
		}
		if err := env.Repo.CreateWalletOrder(ctx, &order); err != nil {
			t.Fatalf("create wallet order %d: %v", i, err)
		}
	}
	targetID := target.ID

	derivedOrderNo := fmt.Sprintf("WALLET-ORDER-%d", targetID)
	env.PaymentReg.RegisterProvider(&testutil.FakePaymentProvider{
		KeyVal:  "fake",
		NameVal: "Fake",
		VerifyRes: shared.PaymentNotifyResult{
			OrderNo: derivedOrderNo,
			Paid:    true,
			Amount:  777,
		},
	}, true, "")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/wallet/payments/notify/fake", bytes.NewBufferString("out_trade_no="+derivedOrderNo))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	env.Router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("notify code: %d body=%s", rec.Code, rec.Body.String())
	}

	updated, err := env.Repo.GetWalletOrder(ctx, targetID)
	if err != nil {
		t.Fatalf("get target wallet order: %v", err)
	}
	if updated.Status != domain.WalletOrderApproved {
		t.Fatalf("expected target order approved, got=%s", updated.Status)
	}
}

// TestHandlers_CatalogScopesImagesToVisiblePlanLines locks P2-32: the public
// catalog image list must be scoped to the lines referenced by visible plans,
// not the full enabled set.
func TestHandlers_CatalogScopesImagesToVisiblePlanLines(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	ctx := context.Background()
	seed := testutil.SeedCatalog(t, env.Repo)
	user := testutil.CreateUser(t, env.Repo, "catscope", "catscope@example.com", "pass")
	token := testutil.IssueJWT(t, env.JWTSecret, user.ID, "user", time.Hour)

	// A second image that belongs to line 2, which has no visible plan in the
	// catalog. Under the old behavior it appeared because it was enabled.
	img2 := domain.SystemImage{ImageID: 2, Name: "Debian", Type: "linux", Enabled: true}
	if err := env.Repo.CreateSystemImage(ctx, &img2); err != nil {
		t.Fatalf("create line-2 image: %v", err)
	}
	if err := env.Repo.SetLineSystemImages(ctx, 2, []int64{img2.ID}); err != nil {
		t.Fatalf("link line-2 image: %v", err)
	}

	rec := testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/catalog", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("catalog code: %d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Images []struct {
			ID int64 `json:"id"`
		} `json:"system_images"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode catalog: %v", err)
	}
	if len(resp.Images) != 1 {
		t.Fatalf("expected exactly the line-1 image, got %d", len(resp.Images))
	}
	if resp.Images[0].ID != seed.SystemImage.ID {
		t.Fatalf("expected image %d, got %d", seed.SystemImage.ID, resp.Images[0].ID)
	}
}

// TestHandlers_CatalogImagesScopedToMultipleVisibleLines covers the N+1-free
// batch path: images linked to any line referenced by a visible plan appear
// exactly once, even when a plan on line 2 exists in addition to line 1.
func TestHandlers_CatalogImagesScopedToMultipleVisibleLines(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	ctx := context.Background()
	seed := testutil.SeedCatalog(t, env.Repo)
	user := testutil.CreateUser(t, env.Repo, "catmulti", "catmulti@example.com", "pass")
	token := testutil.IssueJWT(t, env.JWTSecret, user.ID, "user", time.Hour)

	// A second visible plan on line 2, plus an image on that line.
	plan2 := domain.PlanGroup{
		RegionID:          seed.Region.ID,
		Name:              "Plan-2",
		LineID:            2,
		UnitCore:          1,
		UnitMem:           1,
		UnitDisk:          1,
		UnitBW:            1,
		Active:            true,
		Visible:           true,
		CapacityRemaining: -1,
		SortOrder:         1,
	}
	if err := env.Repo.CreatePlanGroup(ctx, &plan2); err != nil {
		t.Fatalf("create line-2 plan: %v", err)
	}
	pkg2 := domain.Package{
		PlanGroupID:       plan2.ID,
		Name:              "Basic-2",
		Cores:             2,
		MemoryGB:          4,
		DiskGB:            40,
		BandwidthMB:       10,
		Monthly:           10,
		PortNum:           30,
		Active:            true,
		Visible:           true,
		CapacityRemaining: -1,
	}
	if err := env.Repo.CreatePackage(ctx, &pkg2); err != nil {
		t.Fatalf("create line-2 package: %v", err)
	}
	img2 := domain.SystemImage{ImageID: 2, Name: "Debian", Type: "linux", Enabled: true}
	if err := env.Repo.CreateSystemImage(ctx, &img2); err != nil {
		t.Fatalf("create line-2 image: %v", err)
	}
	if err := env.Repo.SetLineSystemImages(ctx, 2, []int64{img2.ID}); err != nil {
		t.Fatalf("link line-2 image: %v", err)
	}
	// Associate the line-1 image with line 2 as well (duplicate association):
	// it must still be listed exactly once.
	if err := env.Repo.SetLineSystemImages(ctx, 2, []int64{img2.ID, seed.SystemImage.ID}); err != nil {
		t.Fatalf("relink line-2 images: %v", err)
	}

	rec := testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/catalog", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("catalog code: %d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Images []struct {
			ID int64 `json:"id"`
		} `json:"system_images"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode catalog: %v", err)
	}
	if len(resp.Images) != 2 {
		t.Fatalf("expected images from both visible lines (deduped), got %d: %+v", len(resp.Images), resp.Images)
	}
	got := map[int64]bool{}
	for _, img := range resp.Images {
		got[img.ID] = true
	}
	if !got[seed.SystemImage.ID] || !got[img2.ID] {
		t.Fatalf("expected line-1 and line-2 images, got %+v", resp.Images)
	}
}

// TestHandlers_CatalogImageQueryFailureReturnsStableError locks the fail-closed
// behavior of the line-scoped image query: when the batch line query cannot
// complete, the handler must return a stable 500 instead of silently serving a
// truncated catalog.
func TestHandlers_CatalogImageQueryFailureReturnsStableError(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	ctx := context.Background()
	seed := testutil.SeedCatalog(t, env.Repo)
	user := testutil.CreateUser(t, env.Repo, "catfail", "catfail@example.com", "pass")
	token := testutil.IssueJWT(t, env.JWTSecret, user.ID, "user", time.Hour)

	// Drop the association table the batch line query joins on. ListAllSystemImages
	// (no join) still succeeds, so the failure is isolated to the line-scoped batch.
	if _, err := env.SQLDB.ExecContext(ctx, "DROP TABLE IF EXISTS line_system_images"); err != nil {
		t.Fatalf("drop line_system_images: %v", err)
	}

	_ = seed
	rec := testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/catalog", nil, token)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when the line image query fails, got %d body=%s", rec.Code, rec.Body.String())
	}
}
