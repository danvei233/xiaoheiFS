package http_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
	adapterhttp "xiaoheiplay/internal/adapter/http"
	"xiaoheiplay/internal/domain"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/testutilhttp"
)

func TestHandlers_RobotWebhook(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/robot/webhook", bytes.NewBufferString(`{"text":"hi","sender":"bot"}`))
	ctx.Request = req

	env.Handler.RobotWebhook(ctx)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("robot webhook status: %d", rec.Code)
	}
}

// TestHandlers_AdminPaymentPluginUploadRouteRemoved guards the removal of the
// legacy raw-binary upload path: the route must no longer exist, so no debug
// mode + upload password can drop an executable into the watched payment plugin
// directory. Plugin binaries are installed exclusively through
// AdminPluginInstall (full package validation + trust confirmation).
func TestHandlers_AdminPaymentPluginUploadRouteRemoved(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	lockPath := filepath.Join(t.TempDir(), "install.lock")
	if err := os.WriteFile(lockPath, []byte("ok"), 0o644); err != nil {
		t.Fatalf("write lock file: %v", err)
	}
	adapterhttp.SetInstallLockPathForTest(lockPath)
	defer adapterhttp.SetInstallLockPathForTest("")
	previousMode := gin.Mode()
	gin.SetMode(gin.DebugMode)
	defer gin.SetMode(previousMode)

	groupID := ensureAdminGroup(t, env)
	admin := testutil.CreateAdmin(t, env.Repo, "adminplugin", "adminplugin@example.com", "pass", groupID)
	token := testutil.IssueJWT(t, env.JWTSecret, admin.ID, "admin", time.Hour)

	req := httptest.NewRequest(http.MethodPost, "/admin/api/v1/plugins/payment/upload", bytes.NewBuffer(nil))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	env.Router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("legacy payment plugin upload route must be gone, got status %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandlers_AdminPluginInstall_DisabledOutsideDebug(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	lockPath := filepath.Join(t.TempDir(), "install.lock")
	if err := os.WriteFile(lockPath, []byte("ok"), 0o644); err != nil {
		t.Fatalf("write lock file: %v", err)
	}
	adapterhttp.SetInstallLockPathForTest(lockPath)
	defer adapterhttp.SetInstallLockPathForTest("")
	previousMode := gin.Mode()
	gin.SetMode(gin.ReleaseMode)
	defer gin.SetMode(previousMode)

	groupID := ensureAdminGroup(t, env)
	admin := testutil.CreateAdmin(t, env.Repo, "adminplugin3", "adminplugin3@example.com", "pass", groupID)
	token := testutil.IssueJWT(t, env.JWTSecret, admin.ID, "admin", time.Hour)

	req := httptest.NewRequest(http.MethodPost, "/admin/api/v1/plugins/install", bytes.NewBuffer(nil))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	env.Router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("plugin install should be forbidden outside debug mode: %d", rec.Code)
	}
}

// TestHandlers_RobotWebhook_ReplayAndStaleRejected verifies the durable DB
// webhook replay ledger rejects a repeated signed delivery within the freshness
// window (409) and rejects stale or missing timestamps (400).
func TestHandlers_RobotWebhook_ReplayAndStaleRejected(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	ctx := context.Background()
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "robot_webhook_enabled", "true"); err != nil {
		t.Fatalf("enable robot webhook: %v", err)
	}
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "robot_webhook_secret", "webhook-test-secret"); err != nil {
		t.Fatalf("set robot webhook secret: %v", err)
	}

	send := func(body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		gc, _ := gin.CreateTestContext(rec)
		gc.Request = httptest.NewRequest(http.MethodPost, "/api/v1/integrations/robot/webhook", bytes.NewBufferString(body))
		gc.Request.Header.Set("X-Signature", hmacBodyHex([]byte(body), "webhook-test-secret"))
		env.Handler.RobotWebhook(gc)
		return rec
	}

	now := time.Now().UTC().Format(time.RFC3339)
	freshBody := `{"text":"通过订单 999","sender":"bot","timestamp":"` + now + `"}`
	first := send(freshBody)
	// Order-not-found is a normal outcome; replay and stale must not occur here.
	if first.Code == http.StatusUnauthorized {
		t.Fatalf("fresh signed webhook must pass signature check, got 401")
	}

	replay := send(freshBody)
	if replay.Code != http.StatusConflict {
		t.Fatalf("replayed webhook must be rejected with 409, got %d", replay.Code)
	}

	staleBody := `{"text":"通过订单 999","sender":"bot","timestamp":"` + time.Now().UTC().Add(-10*time.Minute).Format(time.RFC3339) + `"}`
	stale := send(staleBody)
	if stale.Code != http.StatusBadRequest {
		t.Fatalf("stale webhook must be rejected with 400, got %d", stale.Code)
	}

	missingTs := send(`{"text":"通过订单 999","sender":"bot"}`)
	if missingTs.Code != http.StatusBadRequest {
		t.Fatalf("webhook without timestamp must be rejected with 400, got %d", missingTs.Code)
	}
}

func hmacBodyHex(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return fmt.Sprintf("%x", mac.Sum(nil))
}

// TestHandlers_RobotWebhook_AuditOperatorNotZero verifies webhook business
// commands are attributed to the robot-webhook operator (not admin 0) in the
// audit log, so approvals/rejections issued via the webhook are traceable.
func TestHandlers_RobotWebhook_AuditOperatorNotZero(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	ctx := context.Background()
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "robot_webhook_enabled", "true"); err != nil {
		t.Fatalf("enable robot webhook: %v", err)
	}
	if err := env.AdminSvc.UpdateSetting(ctx, 1, "robot_webhook_secret", "wh-audit-secret"); err != nil {
		t.Fatalf("set robot webhook secret: %v", err)
	}
	user := testutil.CreateUser(t, env.Repo, "wh-audit", "wh-audit@example.com", "pass123")
	order := domain.Order{UserID: user.ID, OrderNo: "ORD-WH-REJECT", Status: domain.OrderStatusPendingReview, TotalAmount: 100, Currency: "CNY"}
	if err := env.Repo.CreateOrder(ctx, &order); err != nil {
		t.Fatalf("create order: %v", err)
	}

	body := `{"text":"驳回订单 ` + testutil.Itoa(order.ID) + ` 原因","sender":"bot","timestamp":"` + time.Now().UTC().Format(time.RFC3339) + `"}`
	rec := httptest.NewRecorder()
	gc, _ := gin.CreateTestContext(rec)
	gc.Request = httptest.NewRequest(http.MethodPost, "/api/v1/integrations/robot/webhook", bytes.NewBufferString(body))
	gc.Request.Header.Set("X-Signature", hmacBodyHex([]byte(body), "wh-audit-secret"))
	env.Handler.RobotWebhook(gc)
	if rec.Code != http.StatusOK {
		t.Fatalf("webhook reject should succeed, got %d: %s", rec.Code, rec.Body.String())
	}

	logs, _, err := env.Repo.ListAuditLogs(ctx, appshared.AuditLogFilter{}, 20, 0)
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}
	found := false
	for _, l := range logs {
		if l.Action == "order.reject" && l.TargetID == testutil.Itoa(order.ID) {
			found = true
			if l.AdminID == 0 {
				t.Fatalf("webhook audit operator must not be pinned at 0, got %d", l.AdminID)
			}
			if l.AdminID != domain.RobotWebhookOperatorID {
				t.Fatalf("webhook audit operator must be the robot sentinel, got %d", l.AdminID)
			}
		}
	}
	if !found {
		t.Fatalf("webhook reject action missing from audit log")
	}
}
