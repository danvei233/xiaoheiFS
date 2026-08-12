package http_test

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	adapterhttp "xiaoheiplay/internal/adapter/http"
	appsettings "xiaoheiplay/internal/app/settings"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/testutilhttp"
)

type fakeTestSMSSender struct {
	mu    sync.Mutex
	calls int
	err   error
}

func (f *fakeTestSMSSender) Send(ctx context.Context, pluginID, instanceID string, msg appshared.SMSMessage) (appshared.SMSDelivery, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.err != nil {
		return appshared.SMSDelivery{}, f.err
	}
	return appshared.SMSDelivery{MessageID: "test-1"}, nil
}

func smsTestHarness(t *testing.T, senderErr error) (*gin.Engine, *testutilhttp.Env, *fakeTestSMSSender, domain.User, string) {
	t.Helper()
	env := testutilhttp.NewTestEnv(t, false)
	admin := testutil.CreateAdmin(t, env.Repo, "smsadmin", "smsadmin@example.com", "pass", 0)
	admin.Phone = "13800000000"
	if err := env.Repo.UpdateUser(context.Background(), admin); err != nil {
		t.Fatalf("set admin phone: %v", err)
	}
	token := testutil.IssueJWT(t, env.JWTSecret, admin.ID, "admin", time.Hour)
	sender := &fakeTestSMSSender{err: senderErr}
	handler := adapterhttp.NewHandler(adapterhttp.HandlerDeps{
		JWTSecret:   env.JWTSecret,
		AuthSvc:     env.AuthSvc,
		AdminSvc:    env.AdminSvc,
		SettingsSvc: appsettings.NewService(env.Repo),
		SMSSender:   sender,
	})
	r := gin.New()
	// The real route is behind RequireAdmin which sets user_id/role; replicate
	// that identity for the bare router.
	r.Use(func(c *gin.Context) {
		c.Set("user_id", admin.ID)
		c.Set("role", "admin")
		c.Next()
	})
	r.POST("/sms-test", handler.AdminSMSTest)
	return r, env, sender, admin, token
}

func doSMS(t *testing.T, r *gin.Engine, token, phone string) int {
	t.Helper()
	rec := testutil.DoJSON(t, r, http.MethodPost, "/sms-test", map[string]any{
		"phone":       phone,
		"content":     "hello",
		"plugin_id":   "fake",
		"instance_id": "default",
	}, token)
	return rec.Code
}

// TestAdminSMSTestArbitraryTargetForbidden verifies an SMS test can only target
// the current admin's own verified phone.
func TestAdminSMSTestArbitraryTargetForbidden(t *testing.T) {
	r, _, _, _, token := smsTestHarness(t, nil)
	if code := doSMS(t, r, token, "13900000000"); code != http.StatusForbidden {
		t.Fatalf("arbitrary target: got %d, want 403", code)
	}
	// Sending to the admin's own phone is allowed.
	if code := doSMS(t, r, token, "13800000000"); code == http.StatusForbidden {
		t.Fatalf("own phone must not be forbidden")
	}
}

// TestAdminSMSTestBudgetHolds verifies the per-admin budget is enforced
// (the old admin+IP limiter was bypassable by switching IPs; the DB budget is
// per-admin regardless of the source IP).
func TestAdminSMSTestBudgetHolds(t *testing.T) {
	r, _, _, _, token := smsTestHarness(t, nil)
	sent := 0
	for i := 0; i < 10; i++ {
		if code := doSMS(t, r, token, "13800000000"); code == http.StatusOK {
			sent++
		}
	}
	if sent != 10 {
		t.Fatalf("expected 10 sends within budget, got %d", sent)
	}
	if code := doSMS(t, r, token, "13800000000"); code != http.StatusTooManyRequests {
		t.Fatalf("budget must hold: got %d, want 429", code)
	}
}

// TestAdminSMSTestProviderFailureStableCode verifies a provider error is mapped
// to a stable code and never echoes the raw error to the client.
func TestAdminSMSTestProviderFailureStableCode(t *testing.T) {
	r, _, _, _, token := smsTestHarness(t, domain.ErrNotFound)
	rec := testutil.DoJSON(t, r, http.MethodPost, "/sms-test", map[string]any{
		"phone":       "13800000000",
		"content":     "hello",
		"plugin_id":   "fake",
		"instance_id": "default",
	}, token)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("provider failure: got %d (%s), want 502", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "not found") {
		t.Fatalf("raw provider error leaked to client: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), domain.ErrSMSDeliveryFailed.Error()) {
		t.Fatalf("expected stable delivery error, got %s", rec.Body.String())
	}
}

// TestAdminSMSTestLongContentRejected verifies oversized content is rejected
// with a stable code.
func TestAdminSMSTestLongContentRejected(t *testing.T) {
	r, _, _, _, token := smsTestHarness(t, nil)
	rec := testutil.DoJSON(t, r, http.MethodPost, "/sms-test", map[string]any{
		"phone":   "13800000000",
		"content": strings.Repeat("x", 2000),
	}, token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("long content: got %d, want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), domain.ErrSMSContentTooLong.Error()) {
		t.Fatalf("expected stable content-too-long code, got %s", rec.Body.String())
	}
}

// TestAdminSMTPTestArbitraryTargetForbidden verifies an SMTP test can only
// target the current admin's own verified email.
func TestAdminSMTPTestArbitraryTargetForbidden(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	admin := testutil.CreateAdmin(t, env.Repo, "smtpadmin", "smtpadmin@example.com", "pass", 0)
	token := testutil.IssueJWT(t, env.JWTSecret, admin.ID, "admin", time.Hour)

	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/integrations/smtp/test", map[string]any{
		"to":      "victim@example.com",
		"subject": "hi",
		"body":    "hello",
	}, token)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("arbitrary smtp target: got %d (%s), want 403", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), domain.ErrSMTPTestTargetForbidden.Error()) {
		t.Fatalf("expected stable forbidden code, got %s", rec.Body.String())
	}
	// Admin's own email reaches the (unconfigured) SMTP sender; that is a
	// provider failure -> stable 502, not a 403.
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/integrations/smtp/test", map[string]any{
		"to":      "smtpadmin@example.com",
		"subject": "hi",
		"body":    "hello",
	}, token)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("own-email provider failure: got %d (%s), want 502", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), domain.ErrSMTPDeliveryFailed.Error()) {
		t.Fatalf("expected stable smtp delivery error, got %s", rec.Body.String())
	}
}

// TestAdminSMTPTestCRLFRejected verifies a CRLF-injected recipient is rejected.
func TestAdminSMTPTestCRLFRejected(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	admin := testutil.CreateAdmin(t, env.Repo, "smtpcrlf", "smtpcrlf@example.com", "pass", 0)
	token := testutil.IssueJWT(t, env.JWTSecret, admin.ID, "admin", time.Hour)
	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/integrations/smtp/test", map[string]any{
		"to":      "a@example.com\r\nBcc: victim@example.com",
		"subject": "hi",
		"body":    "hello",
	}, token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("crlf recipient: got %d (%s), want 400", rec.Code, rec.Body.String())
	}
}
