package http_test

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	adapterhttp "xiaoheiplay/internal/adapter/http"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/testutilhttp"
)

// TestAdminSettingsListMasksPluginUploadPassword guards P0-20 (plugin upload
// password part): the admin settings list must not echo the payment plugin
// upload password in plaintext, and a read-modify-write round trip that sends
// the mask back must not overwrite the real secret.
func TestAdminSettingsListMasksPluginUploadPassword(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	lockPath := filepath.Join(t.TempDir(), "install.lock")
	if err := os.WriteFile(lockPath, []byte("ok"), 0o644); err != nil {
		t.Fatalf("write lock file: %v", err)
	}
	adapterhttp.SetInstallLockPathForTest(lockPath)
	defer adapterhttp.SetInstallLockPathForTest("")
	gin.SetMode(gin.TestMode)

	groupID := ensureAdminGroup(t, env)
	admin := testutil.CreateAdmin(t, env.Repo, "adminplugsecret", "adminplugsecret@example.com", "pass", groupID)
	token := testutil.IssueJWT(t, env.JWTSecret, admin.ID, "admin", time.Hour)

	const secret = "SUPER-SECRET-UPLOAD-PASSWORD"
	if err := env.Repo.UpsertSetting(context.Background(), domain.Setting{
		Key:       "payment_plugin_upload_password",
		ValueJSON: secret,
	}); err != nil {
		t.Fatalf("set upload password: %v", err)
	}

	rec := testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/settings", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("settings status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, secret) {
		t.Fatalf("settings list leaked the plugin upload password in plaintext")
	}
	if !strings.Contains(body, "******") {
		t.Fatalf("settings list did not return a mask for the plugin upload password")
	}

	// Simulate the frontend read-modify-write: send the mask back as the value.
	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/settings", map[string]any{
		"key":   "payment_plugin_upload_password",
		"value": "******",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("mask write-back status=%d body=%s", rec.Code, rec.Body.String())
	}
	stored, err := env.Repo.GetSetting(context.Background(), "payment_plugin_upload_password")
	if err != nil {
		t.Fatalf("get upload password: %v", err)
	}
	if stored.ValueJSON != secret {
		t.Fatalf("mask write-back overwrote the real secret: stored=%q", stored.ValueJSON)
	}

	// And an explicit new value must still be stored.
	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/settings", map[string]any{
		"key":   "payment_plugin_upload_password",
		"value": "new-secret",
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("update status=%d body=%s", rec.Code, rec.Body.String())
	}
	stored, err = env.Repo.GetSetting(context.Background(), "payment_plugin_upload_password")
	if err != nil {
		t.Fatalf("get upload password: %v", err)
	}
	if stored.ValueJSON != "new-secret" {
		t.Fatalf("explicit new value not stored: %q", stored.ValueJSON)
	}
}
