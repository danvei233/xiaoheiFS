package http_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/testutilhttp"
	"xiaoheiplay/internal/usecase"
)

func TestHandlers_UserOpsMore(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	seed := testutil.SeedCatalog(t, env.Repo)
	user := testutil.CreateUser(t, env.Repo, "ops", "ops@example.com", "pass")
	token := testutil.IssueJWT(t, env.JWTSecret, user.ID, "user", time.Hour)

	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/auth/logout", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("logout: %d", rec.Code)
	}

	orderCancel := domain.Order{UserID: user.ID, OrderNo: "ORD-CAN", Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "CNY"}
	if err := env.Repo.CreateOrder(context.Background(), &orderCancel); err != nil {
		t.Fatalf("create cancel order: %v", err)
	}
	if err := env.Repo.CreateOrderItems(context.Background(), []domain.OrderItem{{OrderID: orderCancel.ID, Amount: 1000, Status: domain.OrderItemStatusPendingPayment, Action: "create", SpecJSON: "{}"}}); err != nil {
		t.Fatalf("create cancel item: %v", err)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/orders/"+testutil.Itoa(orderCancel.ID)+"/cancel", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("order cancel: %d", rec.Code)
	}

	orderRefresh := domain.Order{UserID: user.ID, OrderNo: "ORD-REF", Status: domain.OrderStatusApproved, TotalAmount: 1000, Currency: "CNY"}
	if err := env.Repo.CreateOrder(context.Background(), &orderRefresh); err != nil {
		t.Fatalf("create refresh order: %v", err)
	}
	item := domain.OrderItem{
		OrderID:   orderRefresh.ID,
		PackageID: seed.Package.ID,
		SystemID:  seed.SystemImage.ID,
		Amount:    1000,
		Status:    domain.OrderItemStatusActive,
		Action:    "create",
		SpecJSON:  "{}",
	}
	if err := env.Repo.CreateOrderItems(context.Background(), []domain.OrderItem{item}); err != nil {
		t.Fatalf("create refresh item: %v", err)
	}
	items, _ := env.Repo.ListOrderItems(context.Background(), orderRefresh.ID)
	altImage := domain.SystemImage{ImageID: 2, Name: "Debian", Type: "linux", Enabled: true}
	if err := env.Repo.CreateSystemImage(context.Background(), &altImage); err != nil {
		t.Fatalf("create alt image: %v", err)
	}
	if err := env.Repo.SetLineSystemImages(context.Background(), seed.PlanGroup.LineID, []int64{seed.SystemImage.ID, altImage.ID}); err != nil {
		t.Fatalf("set line images: %v", err)
	}
	inst := domain.VPSInstance{
		UserID:               user.ID,
		OrderItemID:          items[0].ID,
		AutomationInstanceID: "123",
		Name:                 "vm-refresh",
		PackageID:            seed.Package.ID,
		SystemID:             seed.SystemImage.ID,
		Status:               domain.VPSStatusRunning,
		SpecJSON:             "{}",
		ExpireAt:             ptrTime(time.Now().Add(24 * time.Hour)),
	}
	if err := env.Repo.CreateInstance(context.Background(), &inst); err != nil {
		t.Fatalf("create instance: %v", err)
	}
	env.Automation.HostInfo = map[int64]usecase.AutomationHostInfo{
		123: {HostID: 123, HostName: "vm-refresh", State: 2, PanelPassword: "pass"},
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/orders/"+testutil.Itoa(orderRefresh.ID)+"/refresh", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("order refresh: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/vps/"+testutil.Itoa(inst.ID)+"/refresh", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("vps refresh: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/vps/"+testutil.Itoa(inst.ID)+"/panel", nil, token)
	if rec.Code != http.StatusFound {
		t.Fatalf("vps panel: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/vps/"+testutil.Itoa(inst.ID)+"/monitor", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("vps monitor: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/vps/"+testutil.Itoa(inst.ID)+"/vnc", nil, token)
	if rec.Code != http.StatusFound {
		t.Fatalf("vps vnc: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/vps/"+testutil.Itoa(inst.ID)+"/start", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("vps start: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/vps/"+testutil.Itoa(inst.ID)+"/shutdown", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("vps shutdown: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/vps/"+testutil.Itoa(inst.ID)+"/reset-os", map[string]any{
		"template_id": altImage.ID,
		"password":    "Pass123!",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("vps reset-os: %d", rec.Code)
	}
	updatedInst, err := env.Repo.GetInstance(context.Background(), inst.ID)
	if err != nil {
		t.Fatalf("get updated vps: %v", err)
	}
	if updatedInst.SystemID != altImage.ID {
		t.Fatalf("expected system_id updated to %d, got %d", altImage.ID, updatedInst.SystemID)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/vps/"+testutil.Itoa(inst.ID)+"/emergency-renew", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("vps emergency renew: %d", rec.Code)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/vps/"+testutil.Itoa(inst.ID)+"/refund", map[string]any{
		"reason": "not needed",
	}, token)
	if rec.Code != http.StatusConflict {
		t.Fatalf("vps refund: %d", rec.Code)
	}

	ticket := domain.Ticket{UserID: user.ID, Subject: "Help", Status: "open"}
	msg := domain.TicketMessage{SenderID: user.ID, SenderRole: "user", Content: "hello"}
	if err := env.Repo.CreateTicketWithDetails(context.Background(), &ticket, &msg, nil); err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/tickets", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("ticket list: %d", rec.Code)
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
