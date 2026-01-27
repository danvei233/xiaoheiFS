package http_test

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

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

func TestHandlers_AdminPaymentPluginUpload(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	groupID := ensureAdminGroup(t, env)
	admin := testutil.CreateAdmin(t, env.Repo, "adminplugin", "adminplugin@example.com", "pass", groupID)
	token := testutil.IssueJWT(t, env.JWTSecret, admin.ID, "admin", time.Hour)

	dir := t.TempDir()
	env.Handler.SetPaymentPluginConfig(dir, "qweasd123456")

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	_ = writer.WriteField("password", "qweasd123456")
	part, err := writer.CreateFormFile("file", "demo.exe")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := io.WriteString(part, "demo"); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/admin/api/v1/plugins/payment/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	env.Router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("payment plugin upload: %d", rec.Code)
	}
}
