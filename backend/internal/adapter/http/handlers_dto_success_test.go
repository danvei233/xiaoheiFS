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

func TestHandlers_AdminDTOListsAndDetail(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	groupID := ensureAdminGroup(t, env)
	admin := testutil.CreateAdmin(t, env.Repo, "admindto", "admindto@example.com", "pass", groupID)
	adminToken := testutil.IssueJWT(t, env.JWTSecret, admin.ID, "admin", time.Hour)

	user := testutil.CreateUser(t, env.Repo, "userdto", "userdto@example.com", "pass")
	catalog := testutil.SeedCatalog(t, env.Repo)

	order := domain.Order{
		UserID:      user.ID,
		OrderNo:     "ORD-DTO-1",
		Status:      domain.OrderStatusPendingPayment,
		TotalAmount: 3000,
		Currency:    "CNY",
	}
	if err := env.Repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	items := []domain.OrderItem{{
		OrderID:        order.ID,
		PackageID:      catalog.Package.ID,
		SystemID:       catalog.SystemImage.ID,
		SpecJSON:       `{"cpu":2}`,
		Qty:            1,
		Amount:         3000,
		Status:         domain.OrderItemStatusPendingPayment,
		Action:         "create",
		DurationMonths: 1,
	}}
	if err := env.Repo.CreateOrderItems(context.Background(), items); err != nil {
		t.Fatalf("create order items: %v", err)
	}
	payment := domain.OrderPayment{
		OrderID:  order.ID,
		UserID:   user.ID,
		Method:   "manual",
		Amount:   3000,
		Currency: "CNY",
		TradeNo:  "TRADE-DTO-1",
		Status:   domain.PaymentStatusPendingReview,
	}
	if err := env.Repo.CreatePayment(context.Background(), &payment); err != nil {
		t.Fatalf("create payment: %v", err)
	}
	if _, err := env.Repo.AppendEvent(context.Background(), order.ID, "created", `{"step":"init"}`); err != nil {
		t.Fatalf("append event: %v", err)
	}

	if err := env.Repo.AddAuditLog(context.Background(), domain.AdminAuditLog{
		AdminID:    admin.ID,
		Action:     "order.view",
		TargetType: "order",
		TargetID:   "order-1",
		DetailJSON: `{"ip":"127.0.0.1"}`,
	}); err != nil {
		t.Fatalf("add audit log: %v", err)
	}
	if err := env.Repo.UpsertSetting(context.Background(), domain.Setting{Key: "test_setting", ValueJSON: "demo"}); err != nil {
		t.Fatalf("upsert setting: %v", err)
	}
	apiKey := domain.APIKey{Name: "k1", KeyHash: "hash1", Status: domain.APIKeyStatusActive, ScopesJSON: `["order.view"]`}
	if err := env.Repo.CreateAPIKey(context.Background(), &apiKey); err != nil {
		t.Fatalf("create api key: %v", err)
	}
	env.PaymentReg.RegisterProvider(&testutil.FakePaymentProvider{KeyVal: "fake", NameVal: "Fake", Schema: `{"fields":[]}`}, true, `{"key":"v"}`)

	rec := testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/users", nil, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin users: %d", rec.Code)
	}
	var userList struct {
		Items []struct {
			ID       int64  `json:"id"`
			Username string `json:"username"`
		} `json:"items"`
		Total int `json:"total"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &userList)
	if userList.Total == 0 || len(userList.Items) == 0 {
		t.Fatalf("admin users list empty")
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/orders", nil, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin orders: %d", rec.Code)
	}
	var orderList struct {
		Items []struct {
			ID     int64  `json:"id"`
			Status string `json:"status"`
		} `json:"items"`
		Total int `json:"total"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &orderList)
	if orderList.Total == 0 || len(orderList.Items) == 0 || orderList.Items[0].Status == "" {
		t.Fatalf("admin orders list empty")
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/orders/"+testutil.Itoa(order.ID), nil, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin order detail: %d", rec.Code)
	}
	var orderDetail struct {
		Order struct {
			ID     int64  `json:"id"`
			Status string `json:"status"`
		} `json:"order"`
		Items []struct {
			ID        int64 `json:"id"`
			PackageID int64 `json:"package_id"`
		} `json:"items"`
		Payments []struct {
			ID      int64   `json:"id"`
			TradeNo string  `json:"trade_no"`
			Amount  float64 `json:"amount"`
		} `json:"payments"`
		Events []struct {
			ID   int64  `json:"id"`
			Type string `json:"type"`
		} `json:"events"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &orderDetail)
	if orderDetail.Order.ID != order.ID || len(orderDetail.Items) != 1 || len(orderDetail.Payments) != 1 || len(orderDetail.Events) != 1 {
		t.Fatalf("admin order detail mismatch")
	}
	if orderDetail.Payments[0].TradeNo == "" || orderDetail.Payments[0].Amount <= 0 {
		t.Fatalf("admin order payment fields missing")
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/settings", nil, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin settings: %d", rec.Code)
	}
	var settings struct {
		Items []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"items"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &settings)
	if len(settings.Items) == 0 || settings.Items[0].Key == "" {
		t.Fatalf("admin settings empty")
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/audit-logs", nil, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin audit logs: %d", rec.Code)
	}
	var auditLogs struct {
		Items []struct {
			ID     int64  `json:"id"`
			Action string `json:"action"`
		} `json:"items"`
		Total int `json:"total"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &auditLogs)
	if auditLogs.Total == 0 || len(auditLogs.Items) == 0 || auditLogs.Items[0].Action == "" {
		t.Fatalf("admin audit logs empty")
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/api-keys", nil, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin api keys: %d", rec.Code)
	}
	var apiKeys struct {
		Items []struct {
			ID      int64  `json:"id"`
			KeyHash string `json:"key_hash"`
		} `json:"items"`
		Total int `json:"total"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &apiKeys)
	if apiKeys.Total == 0 || len(apiKeys.Items) == 0 || apiKeys.Items[0].KeyHash == "" {
		t.Fatalf("admin api keys empty")
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/payments/providers", nil, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin payment providers: %d", rec.Code)
	}
	var providers struct {
		Items []struct {
			Key     string `json:"key"`
			Name    string `json:"name"`
			Enabled bool   `json:"enabled"`
		} `json:"items"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &providers)
	if len(providers.Items) == 0 || providers.Items[0].Key == "" || providers.Items[0].Name == "" {
		t.Fatalf("admin providers empty")
	}
}

func TestHandlers_UserDTOListsAndDetail(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	user := testutil.CreateUser(t, env.Repo, "userdto2", "userdto2@example.com", "pass")
	userToken := testutil.IssueJWT(t, env.JWTSecret, user.ID, "user", time.Hour)

	catalog := testutil.SeedCatalog(t, env.Repo)
	order := domain.Order{
		UserID:      user.ID,
		OrderNo:     "ORD-DTO-2",
		Status:      domain.OrderStatusPendingPayment,
		TotalAmount: 2000,
		Currency:    "CNY",
	}
	if err := env.Repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	items := []domain.OrderItem{{
		OrderID:        order.ID,
		PackageID:      catalog.Package.ID,
		SystemID:       catalog.SystemImage.ID,
		SpecJSON:       `{"cpu":1}`,
		Qty:            1,
		Amount:         2000,
		Status:         domain.OrderItemStatusPendingPayment,
		Action:         "create",
		DurationMonths: 1,
	}}
	if err := env.Repo.CreateOrderItems(context.Background(), items); err != nil {
		t.Fatalf("create order items: %v", err)
	}
	payment := domain.OrderPayment{
		OrderID:  order.ID,
		UserID:   user.ID,
		Method:   "manual",
		Amount:   2000,
		Currency: "CNY",
		TradeNo:  "TRADE-DTO-2",
		Status:   domain.PaymentStatusPendingReview,
	}
	if err := env.Repo.CreatePayment(context.Background(), &payment); err != nil {
		t.Fatalf("create payment: %v", err)
	}
	tx := domain.WalletTransaction{
		UserID:  user.ID,
		Amount:  990,
		Type:    "credit",
		RefType: "seed",
		RefID:   2,
		Note:    "init",
	}
	if err := env.Repo.AddWalletTransaction(context.Background(), &tx); err != nil {
		t.Fatalf("add wallet tx: %v", err)
	}
	walletOrder := domain.WalletOrder{
		UserID:   user.ID,
		Type:     domain.WalletOrderRecharge,
		Amount:   5000,
		Currency: "CNY",
		Status:   domain.WalletOrderPendingReview,
		Note:     "recharge",
		MetaJSON: `{"channel":"bank"}`,
	}
	if err := env.Repo.CreateWalletOrder(context.Background(), &walletOrder); err != nil {
		t.Fatalf("create wallet order: %v", err)
	}
	verifiedAt := time.Now().UTC()
	if err := env.Repo.CreateRealNameVerification(context.Background(), &domain.RealNameVerification{
		UserID:     user.ID,
		RealName:   "Alice",
		IDNumber:   "1234567890123456",
		Status:     "verified",
		Provider:   "fake",
		Reason:     "",
		VerifiedAt: &verifiedAt,
	}); err != nil {
		t.Fatalf("create realname: %v", err)
	}
	env.PaymentReg.RegisterProvider(&testutil.FakePaymentProvider{KeyVal: "balance", NameVal: "Balance", Schema: `{}`}, true, `{}`)

	rec := testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/orders", nil, userToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("user orders: %d", rec.Code)
	}
	var orderList struct {
		Items []struct {
			ID     int64  `json:"id"`
			Status string `json:"status"`
		} `json:"items"`
		Total int `json:"total"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &orderList)
	if orderList.Total == 0 || len(orderList.Items) == 0 || orderList.Items[0].Status == "" {
		t.Fatalf("user order list empty")
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/orders?status=pending_payment", nil, userToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("user orders status filter: %d", rec.Code)
	}
	var filtered struct {
		Items []struct {
			Status string `json:"status"`
		} `json:"items"`
		Total int `json:"total"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &filtered)
	if filtered.Total == 0 || len(filtered.Items) == 0 {
		t.Fatalf("user orders status filter empty")
	}
	for _, it := range filtered.Items {
		if it.Status != "pending_payment" {
			t.Fatalf("unexpected filtered status: %s", it.Status)
		}
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/orders?status=active", nil, userToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("user orders status filter(active): %d", rec.Code)
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &filtered)
	if filtered.Total != 0 || len(filtered.Items) != 0 {
		t.Fatalf("expected empty active orders for this user")
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/orders?status=__invalid__", nil, userToken)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid status, got: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/orders/"+testutil.Itoa(order.ID), nil, userToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("user order detail: %d", rec.Code)
	}
	var orderDetail struct {
		Order struct {
			ID int64 `json:"id"`
		} `json:"order"`
		Payments []struct {
			ID      int64  `json:"id"`
			TradeNo string `json:"trade_no"`
		} `json:"payments"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &orderDetail)
	if orderDetail.Order.ID != order.ID || len(orderDetail.Payments) != 1 || orderDetail.Payments[0].TradeNo == "" {
		t.Fatalf("user order detail mismatch")
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/wallet/transactions", nil, userToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("wallet transactions: %d", rec.Code)
	}
	var txList struct {
		Items []struct {
			ID     int64   `json:"id"`
			Amount float64 `json:"amount"`
		} `json:"items"`
		Total int `json:"total"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &txList)
	if txList.Total == 0 || len(txList.Items) == 0 || txList.Items[0].Amount == 0 {
		t.Fatalf("wallet transactions empty")
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/wallet/orders", nil, userToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("wallet orders: %d", rec.Code)
	}
	var walletOrders struct {
		Items []struct {
			ID     int64  `json:"id"`
			Status string `json:"status"`
		} `json:"items"`
		Total int `json:"total"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &walletOrders)
	if walletOrders.Total == 0 || len(walletOrders.Items) == 0 || walletOrders.Items[0].Status == "" {
		t.Fatalf("wallet orders empty")
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/payments/providers", nil, userToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("payment methods: %d", rec.Code)
	}
	var methods struct {
		Items []struct {
			Key     string  `json:"key"`
			Name    string  `json:"name"`
			Balance float64 `json:"balance"`
		} `json:"items"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &methods)
	if len(methods.Items) == 0 || methods.Items[0].Key == "" || methods.Items[0].Name == "" {
		t.Fatalf("payment methods empty")
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/realname/status", nil, userToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("realname status: %d", rec.Code)
	}
	var realname struct {
		Verification struct {
			IDNumber string `json:"id_number"`
			Status   string `json:"status"`
		} `json:"verification"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &realname)
	if realname.Verification.Status != "verified" || realname.Verification.IDNumber != "1234****3456" {
		t.Fatalf("realname verification mismatch")
	}
}
