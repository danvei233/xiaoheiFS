package http

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"xiaoheiplay/internal/adapter/repo"
	"xiaoheiplay/internal/adapter/seed"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/pkg/config"
	"xiaoheiplay/internal/pkg/db"
)

const installLockEnvKey = "APP_INSTALL_LOCK_PATH"
const installConfigPath = "app.config.json"

func installLockPath() string {
	if v := strings.TrimSpace(os.Getenv(installLockEnvKey)); v != "" {
		return v
	}
	return "install.lock"
}

func (h *Handler) IsInstalled() bool {
	// Keep existing test suite behavior: tests run without an install.lock by default.
	// Installer behavior is covered by explicit tests that set APP_INSTALL_LOCK_PATH.
	if gin.Mode() == gin.TestMode && strings.TrimSpace(os.Getenv(installLockEnvKey)) == "" {
		return true
	}
	_, err := os.Stat(installLockPath())
	return err == nil
}

func (h *Handler) InstallStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"installed": h.IsInstalled()})
}

func (h *Handler) InstallDBCheck(c *gin.Context) {
	var payload struct {
		DB struct {
			Type string `json:"type"`
			Path string `json:"path"`
			DSN  string `json:"dsn"`
		} `json:"db"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	dbType := strings.ToLower(strings.TrimSpace(payload.DB.Type))
	switch dbType {
	case "", "sqlite":
		dbType = "sqlite"
	case "mysql":
		// ok
	case "postgres", "postgresql":
		c.JSON(http.StatusBadRequest, gin.H{"error": "db type not supported yet (postgresql disabled temporarily)"})
		return
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown db type"})
		return
	}

	cfg := config.Config{
		DBType: dbType,
		DBPath: strings.TrimSpace(payload.DB.Path),
		DBDSN:  strings.TrimSpace(payload.DB.DSN),
	}
	if cfg.DBType == "sqlite" && cfg.DBPath == "" {
		cfg.DBPath = "./data/app.db"
	}
	if cfg.DBType == "mysql" && cfg.DBDSN == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "db.dsn required for mysql"})
		return
	}

	conn, err := db.Open(cfg)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": err.Error()})
		return
	}
	defer conn.SQL.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := conn.SQL.PingContext(ctx); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) InstallRun(c *gin.Context) {
	if h.IsInstalled() {
		c.JSON(http.StatusConflict, gin.H{"error": "already installed"})
		return
	}

	var payload struct {
		DB struct {
			Type string `json:"type"`
			Path string `json:"path"`
			DSN  string `json:"dsn"`
		} `json:"db"`
		Site struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"site"`
		Admin struct {
			Username string `json:"username"`
			Password string `json:"password"`
		} `json:"admin"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	dbType := strings.ToLower(strings.TrimSpace(payload.DB.Type))
	if dbType == "" {
		dbType = "sqlite"
	}
	switch dbType {
	case "sqlite":
		// ok
	case "mysql":
		// ok
	case "postgres", "postgresql":
		c.JSON(http.StatusBadRequest, gin.H{"error": "db type not supported yet (postgresql disabled temporarily)"})
		return
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown db type"})
		return
	}

	siteName := strings.TrimSpace(payload.Site.Name)
	siteURL := strings.TrimSpace(payload.Site.URL)
	adminUser := strings.TrimSpace(payload.Admin.Username)
	adminPass := strings.TrimSpace(payload.Admin.Password)
	if siteName == "" || adminUser == "" || adminPass == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "site.name, admin.username, admin.password required"})
		return
	}
	if len(adminPass) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "admin.password too short"})
		return
	}

	cfg := config.Config{
		DBType: dbType,
		DBPath: strings.TrimSpace(payload.DB.Path),
		DBDSN:  strings.TrimSpace(payload.DB.DSN),
	}
	if cfg.DBType == "sqlite" && cfg.DBPath == "" {
		cfg.DBPath = "./data/app.db"
	}
	if cfg.DBType == "mysql" && cfg.DBDSN == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "db.dsn required for mysql"})
		return
	}

	conn, err := db.Open(cfg)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer conn.SQL.Close()

	if err := repo.Migrate(conn.Gorm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "migrate: " + err.Error()})
		return
	}
	if err := seed.EnsureSettings(conn.SQL, conn.Dialect); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "seed settings: " + err.Error()})
		return
	}
	if err := seed.EnsurePermissionDefaults(conn.SQL, conn.Dialect); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "seed permission defaults: " + err.Error()})
		return
	}
	if err := seed.EnsurePermissionGroups(conn.SQL, conn.Dialect); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "seed permission groups: " + err.Error()})
		return
	}
	// CMS defaults and base seed (only if empty) to keep install snappy and idempotent.
	if err := seed.EnsureCMSDefaults(conn.SQL, conn.Dialect); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "seed cms defaults: " + err.Error()})
		return
	}
	if err := seed.SeedIfEmpty(conn.SQL); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "seed: " + err.Error()})
		return
	}

	ctx := context.Background()
	repoAny := repo.NewSQLiteRepo(conn.Gorm)

	_ = repoAny.UpsertSetting(ctx, domain.Setting{Key: "site_name", ValueJSON: siteName})
	if siteURL != "" {
		_ = repoAny.UpsertSetting(ctx, domain.Setting{Key: "site_url", ValueJSON: siteURL})
	}

	if _, err := repoAny.GetUserByUsernameOrEmail(ctx, adminUser); err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "admin already exists"})
		return
	}

	var superAdminGroupID int64
	if err := conn.SQL.QueryRow(`SELECT id FROM permission_groups WHERE name = ?`, "超级管理员").Scan(&superAdminGroupID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "permission group missing"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(adminPass), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hash failed"})
		return
	}
	user := &domain.User{
		Username:          adminUser,
		Email:             adminUser + "@local",
		PasswordHash:      string(hash),
		Role:              domain.UserRoleAdmin,
		Status:            domain.UserStatusActive,
		PermissionGroupID: &superAdminGroupID,
	}
	if err := repoAny.CreateUser(ctx, user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Persist DB config so next boot uses the installed DB even without env vars.
	if b, err := json.MarshalIndent(map[string]any{
		"db_type": cfg.DBType,
		"db_path": cfg.DBPath,
		"db_dsn":  cfg.DBDSN,
	}, "", "  "); err == nil {
		_ = os.WriteFile(installConfigPath, b, 0o600)
	}

	lockPath := installLockPath()
	if dir := filepath.Dir(lockPath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "lock dir error"})
			return
		}
	}
	if err := os.WriteFile(lockPath, []byte(time.Now().Format(time.RFC3339)), 0o644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "write lock failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":               true,
		"restart_required": cfg.DBType != "sqlite",
		"config_file":      installConfigPath,
	})
}
