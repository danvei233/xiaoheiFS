package http_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"

	adapterhttp "xiaoheiplay/internal/adapter/http"
	"xiaoheiplay/internal/adapter/repo/core"
	"xiaoheiplay/internal/pkg/config"
	"xiaoheiplay/internal/pkg/db"
	"xiaoheiplay/internal/testutil"
)

func TestInstallRun_SQLite_FullFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tmp := chdirTemp(t)
	lockPath := filepath.Join(tmp, "install.lock")

	adapterhttp.SetInstallLockPathForTest(lockPath)
	t.Cleanup(func() { adapterhttp.SetInstallLockPathForTest("") })

	server := adapterhttp.NewInstallBootstrapServer("test-secret")
	sqlitePath := filepath.Join(tmp, "data", "install.db")
	payload := map[string]any{
		"db": map[string]any{
			"type": "sqlite",
			"path": sqlitePath,
		},
		"site": map[string]any{
			"name": "Install Test",
			"url":  "http://localhost:8080",
		},
		"admin": map[string]any{
			"username": "installer_admin",
			"password": "password123",
		},
	}

	recCheck := testutil.DoJSON(t, server.Engine, "POST", "/api/v1/install/db/check", map[string]any{
		"db": map[string]any{"type": "sqlite", "path": sqlitePath},
	}, "")
	if recCheck.Code != 200 {
		t.Fatalf("db check status=%d body=%s", recCheck.Code, recCheck.Body.String())
	}

	rec := testutil.DoJSON(t, server.Engine, "POST", "/api/v1/install", payload, "")
	if rec.Code != 200 {
		t.Fatalf("install status=%d body=%s", rec.Code, rec.Body.String())
	}

	var resp struct {
		OK          bool   `json:"ok"`
		ConfigFile  string `json:"config_file"`
		RestartNeed bool   `json:"restart_required"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode install response: %v", err)
	}
	if !resp.OK {
		t.Fatalf("install response not ok: %s", rec.Body.String())
	}
	if resp.RestartNeed {
		t.Fatalf("sqlite should not require restart")
	}
	if strings.TrimSpace(resp.ConfigFile) == "" {
		t.Fatalf("install response missing config_file")
	}

	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("install lock not created: %v", err)
	}

	cfgRaw, err := os.ReadFile(resp.ConfigFile)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var cfg map[string]any
	if err := yaml.Unmarshal(cfgRaw, &cfg); err != nil {
		t.Fatalf("parse config yaml: %v", err)
	}
	dbCfg, ok := cfg["db"].(map[string]any)
	if !ok {
		t.Fatalf("config missing db section: %v", cfg)
	}
	if strings.TrimSpace(asString(dbCfg["type"])) != "sqlite" {
		t.Fatalf("unexpected db.type: %v", dbCfg["type"])
	}
	if strings.TrimSpace(asString(dbCfg["path"])) != sqlitePath {
		t.Fatalf("unexpected db.path: %v", dbCfg["path"])
	}

	conn, err := db.Open(config.Config{DBType: "sqlite", DBPath: sqlitePath})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = conn.SQL.Close() })
	r := repo.NewGormRepo(conn.Gorm)
	admin, err := r.GetUserByUsernameOrEmail(context.Background(), "installer_admin")
	if err != nil {
		t.Fatalf("query admin: %v", err)
	}
	if admin.Role != "admin" || admin.Status != "active" {
		t.Fatalf("unexpected admin role/status: %+v", admin)
	}

	recStatus := testutil.DoJSON(t, server.Engine, "GET", "/api/v1/install/status", nil, "")
	if recStatus.Code != 200 {
		t.Fatalf("status code=%d body=%s", recStatus.Code, recStatus.Body.String())
	}
	var statusResp struct {
		Installed bool `json:"installed"`
	}
	if err := json.Unmarshal(recStatus.Body.Bytes(), &statusResp); err != nil {
		t.Fatalf("decode status response: %v", err)
	}
	if !statusResp.Installed {
		t.Fatalf("expected installed=true")
	}
}

func TestInstallRun_MySQL_FullFlow(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("XIAOHEI_TEST_MYSQL_DSN"))
	if dsn == "" {
		t.Skip("set XIAOHEI_TEST_MYSQL_DSN to run MySQL install integration test")
	}
	gin.SetMode(gin.TestMode)
	tmp := chdirTemp(t)
	lockPath := filepath.Join(tmp, "install.lock")

	adapterhttp.SetInstallLockPathForTest(lockPath)
	t.Cleanup(func() { adapterhttp.SetInstallLockPathForTest("") })

	server := adapterhttp.NewInstallBootstrapServer("test-secret")
	payload := map[string]any{
		"db": map[string]any{
			"type": "mysql",
			"dsn":  dsn,
		},
		"site": map[string]any{
			"name": "Install Test MySQL",
			"url":  "http://localhost:8080",
		},
		"admin": map[string]any{
			"username": "installer_admin_mysql",
			"password": "password123",
		},
	}

	recCheck := testutil.DoJSON(t, server.Engine, "POST", "/api/v1/install/db/check", map[string]any{
		"db": map[string]any{"type": "mysql", "dsn": dsn},
	}, "")
	if recCheck.Code != 200 {
		t.Fatalf("mysql db check status=%d body=%s", recCheck.Code, recCheck.Body.String())
	}

	rec := testutil.DoJSON(t, server.Engine, "POST", "/api/v1/install", payload, "")
	if rec.Code != 200 {
		t.Fatalf("mysql install status=%d body=%s", rec.Code, rec.Body.String())
	}

	conn, err := db.Open(config.Config{DBType: "mysql", DBDSN: dsn})
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	t.Cleanup(func() { _ = conn.SQL.Close() })
	r := repo.NewGormRepo(conn.Gorm)
	admin, err := r.GetUserByUsernameOrEmail(context.Background(), "installer_admin_mysql")
	if err != nil {
		t.Fatalf("query mysql admin: %v", err)
	}
	if admin.Role != "admin" || admin.Status != "active" {
		t.Fatalf("unexpected mysql admin role/status: %+v", admin)
	}
}

func chdirTemp(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})
	return tmp
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
