package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/testutilhttp"
)

func TestUserFlow_E2E(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	seed := testutil.SeedCatalog(t, env.Repo)
	if err := env.Repo.CreateBillingCycle(context.Background(), &domain.BillingCycle{Name: "monthly", Months: 1, Multiplier: 1, MinQty: 1, MaxQty: 12, Active: true, SortOrder: 1}); err != nil {
		t.Fatalf("create billing cycle: %v", err)
	}
	_ = env.Repo.UpsertSetting(context.Background(), domain.Setting{Key: "site_name", ValueJSON: "Test"})
	_ = env.Repo.UpsertSetting(context.Background(), domain.Setting{Key: "resize_price_mode", ValueJSON: "remaining"})

	adminToken := createAdminToken(t, env)

	captcha, code, err := env.AuthSvc.CreateCaptcha(context.Background(), time.Minute)
	if err != nil {
		t.Fatalf("captcha: %v", err)
	}
	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"username":     "flow",
		"email":        "flow@example.com",
		"password":     "pass123",
		"captcha_id":   captcha.ID,
		"captcha_code": code,
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("register: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"username": "flow",
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

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/catalog", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("catalog: %d", rec.Code)
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
	items, err := env.Repo.ListCartItems(context.Background(), userID)
	if err != nil || len(items) != 1 {
		t.Fatalf("cart items: %v %v", items, err)
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
	order, err := env.Repo.GetOrder(context.Background(), orderResp.Order.ID)
	if err != nil || order.Status != domain.OrderStatusPendingPayment {
		t.Fatalf("order status: %v %v", order, err)
	}

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
			ID     int64   `json:"id"`
			Amount float64 `json:"amount"`
		} `json:"order"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &walletOrderResp); err != nil {
		t.Fatalf("decode wallet order: %v", err)
	}
	approve := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/wallet/orders/"+testutil.Itoa(walletOrderResp.Order.ID)+"/approve", nil, adminToken)
	if approve.Code != http.StatusOK {
		t.Fatalf("wallet approve: %d", approve.Code)
	}
	wallet, err := env.Repo.GetWallet(context.Background(), userID)
	if err != nil || wallet.Balance < 100 {
		t.Fatalf("wallet balance: %v %v", wallet, err)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/orders/"+testutil.Itoa(order.ID)+"/pay", map[string]any{
		"method": "balance",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("pay with balance: %d", rec.Code)
	}
	walletAfter, _ := env.Repo.GetWallet(context.Background(), userID)
	if walletAfter.Balance >= wallet.Balance {
		t.Fatalf("expected wallet debited")
	}

	waitForOrderActive(t, env, order.ID, 5*time.Second)
	vps := waitForVPS(t, env, userID, 5*time.Second)

	notifyOrderID := createDirectOrder(t, env, token, userID, seed.Package.ID, seed.SystemImage.ID)
	tradeNo := payWithProvider(t, env, token, notifyOrderID, "fake")
	beforeCalls := len(env.Automation.CreateHostRequests)
	for i := 0; i < 3; i++ {
		notifyRec := sendNotify(t, env, "fake", tradeNo, "10")
		if notifyRec.Code != http.StatusOK {
			t.Fatalf("notify: %d", notifyRec.Code)
		}
	}
	waitForOrderActive(t, env, notifyOrderID, 5*time.Second)
	afterCalls := len(env.Automation.CreateHostRequests)
	if afterCalls-beforeCalls != 1 {
		t.Fatalf("expected single automation call, got %d", afterCalls-beforeCalls)
	}
	payment, err := env.Repo.GetPaymentByTradeNo(context.Background(), tradeNo)
	if err != nil || payment.Status != domain.PaymentStatusApproved {
		t.Fatalf("payment status: %v %v", payment, err)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/vps", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("vps list: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/vps/"+testutil.Itoa(vps.ID), nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("vps detail: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/vps/"+testutil.Itoa(vps.ID)+"/reboot", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("vps reboot: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/vps/"+testutil.Itoa(vps.ID)+"/renew", map[string]any{
		"renew_days": 30,
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("vps renew: %d", rec.Code)
	}

	altPkg := domain.Package{
		PlanGroupID: seed.PlanGroup.ID,
		Name:        "Alt",
		Cores:       4,
		MemoryGB:    8,
		DiskGB:      40,
		BandwidthMB: 10,
		CPUModel:    "x",
		Monthly:     2000,
		PortNum:     30,
		Active:      true,
		Visible:     true,
	}
	if err := env.Repo.CreatePackage(context.Background(), &altPkg); err != nil {
		t.Fatalf("create alt package: %v", err)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/vps/"+testutil.Itoa(vps.ID)+"/resize", map[string]any{
		"target_package_id": altPkg.ID,
		"spec":              map[string]any{},
	}, token)
	if rec.Code != http.StatusOK && rec.Code != http.StatusConflict {
		t.Fatalf("vps resize: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/tickets", map[string]any{
		"subject": "help",
		"content": "need support",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("ticket create: %d", rec.Code)
	}
	var ticketResp struct {
		Ticket struct {
			ID int64 `json:"id"`
		} `json:"ticket"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &ticketResp)
	other := testutil.CreateUser(t, env.Repo, "other", "other@example.com", "pass")
	otherToken := testutil.IssueJWT(t, env.JWTSecret, other.ID, "user", time.Hour)
	deny := testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/tickets/"+testutil.Itoa(ticketResp.Ticket.ID), nil, otherToken)
	if deny.Code != http.StatusForbidden {
		t.Fatalf("ticket forbidden: %d", deny.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/tickets/"+testutil.Itoa(ticketResp.Ticket.ID)+"/messages", map[string]any{
		"content": "reply",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("ticket reply: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/tickets/"+testutil.Itoa(ticketResp.Ticket.ID)+"/close", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("ticket close: %d", rec.Code)
	}

	n := domain.Notification{UserID: userID, Type: "info", Title: "hi", Content: "msg"}
	if err := env.Repo.CreateNotification(context.Background(), &n); err != nil {
		t.Fatalf("create notification: %v", err)
	}
	unread := testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/notifications/unread-count", nil, token)
	if unread.Code != http.StatusOK {
		t.Fatalf("unread count: %d", unread.Code)
	}
	readAll := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/notifications/read-all", nil, token)
	if readAll.Code != http.StatusOK {
		t.Fatalf("read all: %d", readAll.Code)
	}
}

func createAdminToken(t *testing.T, env *testutilhttp.Env) string {
	groups, err := env.Repo.ListPermissionGroups(context.Background())
	if err != nil || len(groups) == 0 {
		t.Fatalf("permission groups: %v", err)
	}
	groupID := groups[0].ID
	for _, g := range groups {
		if g.PermissionsJSON == `["*"]` {
			groupID = g.ID
			break
		}
	}
	admin := testutil.CreateAdmin(t, env.Repo, "admin-flow", "admin-flow@example.com", "pass", groupID)
	return testutil.IssueJWT(t, env.JWTSecret, admin.ID, "admin", time.Hour)
}

func createDirectOrder(t *testing.T, env *testutilhttp.Env, token string, userID int64, pkgID int64, systemID int64) int64 {
	add := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/cart", map[string]any{
		"package_id": pkgID,
		"system_id":  systemID,
		"spec":       map[string]any{},
		"qty":        1,
	}, token)
	if add.Code != http.StatusOK {
		t.Fatalf("add cart for order: %d", add.Code)
	}
	items, err := env.Repo.ListCartItems(context.Background(), userID)
	if err == nil && len(items) == 0 {
		t.Fatalf("expected cart items")
	}
	time.Sleep(time.Second)
	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/orders", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("create order from cart: %d %s", rec.Code, rec.Body.String())
	}
	var orderResp struct {
		Order struct {
			ID int64 `json:"id"`
		} `json:"order"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &orderResp); err != nil {
		t.Fatalf("decode order: %v", err)
	}
	return orderResp.Order.ID
}

func payWithProvider(t *testing.T, env *testutilhttp.Env, token string, orderID int64, provider string) string {
	env.PaymentReg.RegisterProvider(&testutil.FakePaymentProvider{
		KeyVal:  provider,
		NameVal: "Fake",
		CreateRes: shared.PaymentCreateResult{
			PayURL:  "https://pay.local",
			TradeNo: "TN-E2E",
		},
		VerifyFunc: func(req shared.RawHTTPRequest) (shared.PaymentNotifyResult, error) {
			return shared.PaymentNotifyResult{TradeNo: "TN-E2E", Paid: true, Amount: 1000}, nil
		},
	}, true, "")
	pay := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/orders/"+testutil.Itoa(orderID)+"/pay", map[string]any{
		"method": provider,
	}, token)
	if pay.Code != http.StatusOK {
		t.Fatalf("pay with provider: %d", pay.Code)
	}
	var payResp struct {
		TradeNo string `json:"trade_no"`
	}
	if err := json.Unmarshal(pay.Body.Bytes(), &payResp); err != nil {
		t.Fatalf("decode pay: %v", err)
	}
	if payResp.TradeNo == "" {
		payResp.TradeNo = "TN-E2E"
	}
	return payResp.TradeNo
}

func sendNotify(t *testing.T, env *testutilhttp.Env, provider, tradeNo, amount string) *httptest.ResponseRecorder {
	form := "trade_no=" + tradeNo + "&amount=" + amount
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/notify/"+provider, bytes.NewBufferString(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	env.Router.ServeHTTP(rec, req)
	return rec
}

func waitForVPS(t *testing.T, env *testutilhttp.Env, userID int64, timeout time.Duration) domain.VPSInstance {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		list, err := env.Repo.ListInstancesByUser(context.Background(), userID)
		if err == nil && len(list) > 0 {
			return list[0]
		}
		time.Sleep(50 * time.Millisecond)
	}
	if len(env.Automation.CreateHostRequests) == 0 {
		t.Fatalf("automation not called")
	}
	return domain.VPSInstance{}
}

func waitForOrderActive(t *testing.T, env *testutilhttp.Env, orderID int64, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		order, err := env.Repo.GetOrder(context.Background(), orderID)
		if err == nil && order.Status == domain.OrderStatusActive {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	order, _ := env.Repo.GetOrder(context.Background(), orderID)
	t.Fatalf("order not active: %v", order.Status)
}
