package http

import (
	"context"
	"encoding/json"
	"errors"
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
	"xiaoheiplay/internal/usecase"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"gopkg.in/yaml.v3"
)

const installConfigPath = "app.config.yaml"

var installLockPathOverride string

func installLockPath() string {
	if strings.TrimSpace(installLockPathOverride) != "" {
		return strings.TrimSpace(installLockPathOverride)
	}
	if configPath := strings.TrimSpace(config.LocalConfigPath()); configPath != "" {
		return filepath.Join(filepath.Dir(configPath), "install.lock")
	}
	return "install.lock"
}

func (h *Handler) IsInstalled() bool {
	// Keep existing test suite behavior: tests run without an install.lock by default.
	// Installer behavior is covered by explicit tests that override the lock path.
	if gin.Mode() == gin.TestMode && strings.TrimSpace(installLockPathOverride) == "" {
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
	case "":
		c.JSON(http.StatusBadRequest, gin.H{"error": "db.type required"})
		return
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

	cfg := config.Config{
		DBType: dbType,
		DBPath: strings.TrimSpace(payload.DB.Path),
		DBDSN:  strings.TrimSpace(payload.DB.DSN),
	}
	if cfg.DBType == "sqlite" && cfg.DBPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "db.path required for sqlite"})
		return
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
	switch dbType {
	case "":
		c.JSON(http.StatusBadRequest, gin.H{"error": "db.type required"})
		return
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "db.path required for sqlite"})
		return
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
	if err := seed.EnsureSettingsGorm(conn.Gorm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "seed settings: " + err.Error()})
		return
	}
	if err := seed.EnsurePermissionDefaultsGorm(conn.Gorm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "seed permission defaults: " + err.Error()})
		return
	}
	if err := seed.EnsurePermissionGroupsGorm(conn.Gorm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "seed permission groups: " + err.Error()})
		return
	}
	// CMS defaults and base seed (only if empty) to keep install snappy and idempotent.
	if err := seed.EnsureCMSDefaultsGorm(conn.Gorm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "seed cms defaults: " + err.Error()})
		return
	}
	if err := seed.SeedIfEmptyGorm(conn.Gorm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "seed: " + err.Error()})
		return
	}

	ctx := context.Background()
	repoAny := repo.NewGormRepo(conn.Gorm)

	_ = repoAny.UpsertSetting(ctx, domain.Setting{Key: "site_name", ValueJSON: siteName})
	if siteURL != "" {
		_ = repoAny.UpsertSetting(ctx, domain.Setting{Key: "site_url", ValueJSON: siteURL})
	}

	if _, err := repoAny.GetUserByUsernameOrEmail(ctx, adminUser); err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "admin already exists"})
		return
	} else if !errors.Is(err, usecase.ErrNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query admin failed: " + err.Error()})
		return
	}

	groups, err := repoAny.ListPermissionGroups(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "permission group query failed: " + err.Error()})
		return
	}
	superAdminGroupID, ok := findSuperAdminGroupID(groups)
	if !ok {
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
		if isDuplicateEntryError(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "admin already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Persist DB config to the same config file that loader currently resolves,
	// so subsequent restarts do not fall back to sqlite due to CWD differences.
	configPath := strings.TrimSpace(config.LocalConfigPath())
	if configPath == "" {
		if exePath, err := os.Executable(); err == nil {
			configPath = filepath.Join(filepath.Dir(exePath), installConfigPath)
		} else {
			configPath = installConfigPath
		}
	}
	if dir := filepath.Dir(configPath); dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0o755)
	}

	out := map[string]any{}
	if existing, err := os.ReadFile(configPath); err == nil {
		_ = yaml.Unmarshal(existing, &out)
	}
	// These values are NOT persisted in the config file. They are created/managed via install + DB settings.
	delete(out, "admin")
	delete(out, "automation")
	out["db"] = map[string]any{
		"type": cfg.DBType,
		"path": cfg.DBPath,
		"dsn":  cfg.DBDSN,
	}

	if b, err := yaml.Marshal(&out); err == nil {
		_ = os.WriteFile(configPath, b, 0o600)
	} else if b, err := json.MarshalIndent(map[string]any{
		"db_type": cfg.DBType,
		"db_path": cfg.DBPath,
		"db_dsn":  cfg.DBDSN,
	}, "", "  "); err == nil {
		_ = os.WriteFile("app.config.json", b, 0o600)
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
		"config_file":      configPath,
	})
	if cfg.DBType != "sqlite" && gin.Mode() != gin.TestMode {
		// Trigger process recycle so the newly persisted DB config is applied immediately.
		go func() {
			time.Sleep(300 * time.Millisecond)
			os.Exit(0)
		}()
	}
}

func isDuplicateEntryError(err error) bool {
	if err == nil {
		return false
	}
	var me *mysqlDriver.MySQLError
	if errors.As(err, &me) {
		return me.Number == 1062
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate entry") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "constraint failed")
}

func findSuperAdminGroupID(groups []domain.PermissionGroup) (int64, bool) {
	if len(groups) == 0 {
		return 0, false
	}
	for _, group := range groups {
		var perms []string
		if json.Unmarshal([]byte(group.PermissionsJSON), &perms) != nil {
			continue
		}
		for _, p := range perms {
			if strings.TrimSpace(p) == "*" {
				return group.ID, true
			}
		}
	}
	for _, group := range groups {
		name := strings.ToLower(strings.TrimSpace(group.Name))
		if strings.Contains(name, "admin") || strings.Contains(name, "管理员") {
			return group.ID, true
		}
	}
	return groups[0].ID, true
}
