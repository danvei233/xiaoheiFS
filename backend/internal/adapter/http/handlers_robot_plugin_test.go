package http_test

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
	adapterhttp "xiaoheiplay/internal/adapter/http"
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

func TestHandlers_AdminPaymentPluginUpload_DisabledOutsideDebug(t *testing.T) {
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
	admin := testutil.CreateAdmin(t, env.Repo, "adminplugin2", "adminplugin2@example.com", "pass", groupID)
	token := testutil.IssueJWT(t, env.JWTSecret, admin.ID, "admin", time.Hour)

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
	if rec.Code != http.StatusForbidden {
		t.Fatalf("payment plugin upload should be forbidden outside debug mode: %d", rec.Code)
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
