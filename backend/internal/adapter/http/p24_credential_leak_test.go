package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/testutilhttp"
)

// seedVPSWithCredentials creates a user + VPS whose AccessInfoJSON carries
// panel/VNC/OS passwords.
func seedVPSWithCredentials(t *testing.T, env *testutilhttp.Env, user domain.User) (int64, string) {
	t.Helper()
	inst := domain.VPSInstance{
		UserID:         user.ID,
		Name:           "p2-4-vps",
		Status:         domain.VPSStatusRunning,
		AccessInfoJSON: `{"panel_url":"https://panel.example.com","panel_password":"PANEL-SECRET","vnc_password":"VNC-SECRET","os_password":"OS-SECRET"}`,
	}
	if err := env.Repo.CreateInstance(context.Background(), &inst); err != nil {
		t.Fatalf("create vps: %v", err)
	}
	token := testutil.IssueJWT(t, env.JWTSecret, user.ID, "user", time.Hour)
	return inst.ID, token
}

// TestP2_4_UserVPSResponsesDoNotLeakAccessInfo pins the least-privilege rule:
// user-facing VPS list / detail / monitor responses must not carry the
// AccessInfo map (panel/VNC/OS passwords).
func TestP2_4_UserVPSResponsesDoNotLeakAccessInfo(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	user := testutil.CreateUser(t, env.Repo, "p24user", "p24user@example.com", "pass123")
	vpsID, token := seedVPSWithCredentials(t, env, user)

	// List response must not leak.
	rec := testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/vps", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("vps list: %d", rec.Code)
	}
	var list struct {
		Items []struct {
			ID         int64                    `json:"id"`
			AccessInfo map[string]any           `json:"access_info"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	for _, item := range list.Items {
		if item.ID == vpsID && len(item.AccessInfo) > 0 {
			t.Fatalf("list response leaked AccessInfo: %#v", item.AccessInfo)
		}
	}
	if strings.Contains(rec.Body.String(), "PANEL-SECRET") || strings.Contains(rec.Body.String(), "OS-SECRET") {
		t.Fatalf("list response leaked a credential")
	}

	// Detail response must not leak.
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/vps/"+testutil.Itoa(vpsID), nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("vps detail: %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "PANEL-SECRET") || strings.Contains(rec.Body.String(), "VNC-SECRET") {
		t.Fatalf("detail response leaked a credential: %s", rec.Body.String())
	}
	var detail struct {
		AccessInfo map[string]any `json:"access_info"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	if len(detail.AccessInfo) > 0 {
		t.Fatalf("detail response leaked AccessInfo: %#v", detail.AccessInfo)
	}

	// Monitor response must not leak.
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/vps/"+testutil.Itoa(vpsID)+"/monitor", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("vps monitor: %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "PANEL-SECRET") || strings.Contains(rec.Body.String(), "VNC-SECRET") || strings.Contains(rec.Body.String(), "OS-SECRET") {
		t.Fatalf("monitor response leaked a credential: %s", rec.Body.String())
	}
	var mon struct {
		AccessInfo map[string]any `json:"access_info"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &mon); err != nil {
		t.Fatalf("decode monitor: %v", err)
	}
	if len(mon.AccessInfo) > 0 {
		t.Fatalf("monitor response leaked AccessInfo: %#v", mon.AccessInfo)
	}
}

// TestP2_4_OrderRefreshDoesNotLeakAccessInfo pins the P2-4 least-privilege rule
// on POST /api/v1/orders/:id/refresh: the order-refresh path used the admin DTO
// variant and echoed the panel/VNC/OS passwords back to the user in every
// response, bypassing the explicit, audited credentials endpoint.
func TestP2_4_OrderRefreshDoesNotLeakAccessInfo(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	user := testutil.CreateUser(t, env.Repo, "p24refresh", "p24refresh@example.com", "pass123")

	order := domain.Order{UserID: user.ID, OrderNo: "O-REFRESH", Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "USD"}
	if err := env.Repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	items := []domain.OrderItem{{OrderID: order.ID, SpecJSON: "{}", Qty: 1, Amount: 1000, Status: domain.OrderItemStatusPendingPayment, AutomationInstanceID: "1001", Action: "create", DurationMonths: 1}}
	if err := env.Repo.CreateOrderItems(context.Background(), items); err != nil {
		t.Fatalf("create order items: %v", err)
	}
	itemID := items[0].ID

	inst := domain.VPSInstance{
		UserID:               user.ID,
		Name:                 "refresh-vps",
		OrderItemID:          itemID,
		AutomationInstanceID: "1001",
		Status:               domain.VPSStatusRunning,
		AccessInfoJSON:       `{"panel_url":"https://panel.example.com","panel_password":"PANEL-SECRET","vnc_password":"VNC-SECRET","os_password":"OS-SECRET"}`,
	}
	if err := env.Repo.CreateInstance(context.Background(), &inst); err != nil {
		t.Fatalf("create vps: %v", err)
	}

	token := testutil.IssueJWT(t, env.JWTSecret, user.ID, "user", time.Hour)
	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/api/v1/orders/"+testutil.Itoa(order.ID)+"/refresh", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("order refresh: %d %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "PANEL-SECRET") || strings.Contains(rec.Body.String(), "VNC-SECRET") || strings.Contains(rec.Body.String(), "OS-SECRET") {
		t.Fatalf("order refresh response leaked a credential: %s", rec.Body.String())
	}
	var body struct {
		Items []struct {
			AccessInfo map[string]any `json:"access_info"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode refresh response: %v", err)
	}
	for _, item := range body.Items {
		if len(item.AccessInfo) > 0 {
			t.Fatalf("order refresh response leaked AccessInfo: %#v", item.AccessInfo)
		}
	}

	// The explicit credentials endpoint still serves the owner on request.
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/vps/"+testutil.Itoa(inst.ID)+"/credentials", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("credentials endpoint after refresh: %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "PANEL-SECRET") {
		t.Fatalf("credentials endpoint did not serve the owner: %s", rec.Body.String())
	}
}

// TestP2_4_CredentialsEndpointServesOnExplicitRequest verifies the explicit
// least-privilege endpoint returns the credentials only to the owner and is
// audited.
func TestP2_4_CredentialsEndpointServesOnExplicitRequest(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	user := testutil.CreateUser(t, env.Repo, "p24creds", "p24creds@example.com", "pass123")
	vpsID, token := seedVPSWithCredentials(t, env, user)

	rec := testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/vps/"+testutil.Itoa(vpsID)+"/credentials", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("credentials endpoint: %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "PANEL-SECRET") || !strings.Contains(rec.Body.String(), "OS-SECRET") {
		t.Fatalf("credentials endpoint did not return the credentials: %s", rec.Body.String())
	}

	// A different user must not get the credentials (owner-only via vpsSvc.Get).
	other := testutil.CreateUser(t, env.Repo, "p24other", "p24other@example.com", "pass123")
	otherToken := testutil.IssueJWT(t, env.JWTSecret, other.ID, "user", time.Hour)
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/vps/"+testutil.Itoa(vpsID)+"/credentials", nil, otherToken)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("non-owner credentials request must 404, got %d", rec.Code)
	}
}
