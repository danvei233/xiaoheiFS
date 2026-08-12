package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
	"xiaoheiplay/internal/adapter/repo/core"
	"xiaoheiplay/internal/adapter/seed"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/pkg/config"
	"xiaoheiplay/internal/pkg/db"

	"github.com/gin-gonic/gin"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

const installConfigPath = "app.config.yaml"

// installCompletedSettingKey records a successful install inside the database
// itself. Unlike install.lock (a plain file that can be lost to CWD changes,
// volume restore or deletion), this marker survives and lets a replayed install
// be refused even when the lock file has gone missing.
const installCompletedSettingKey = "install_completed_at"

// Fixed error messages returned to the unauthenticated client on DB failures.
// The detailed driver/GORM errors (host, port, db user, DSN fragments) are
// logged server-side instead of being echoed back. The db connect/migrate/seed
// messages alias the centralized domain sentinels (ops hardening) so handlers
// and tests share one spelling.
var (
	errDBCheckFailed   = domain.ErrDbConnectFailed.Error()
	errDBMigrateFailed = strings.TrimSpace(domain.ErrMigrate.Error())
	errDBSeedFailed    = strings.TrimSpace(domain.ErrSeedSettings.Error())
	errDBClaimFailed   = "database claim failed"
)

// validateInstallMySQLDSN reports whether a MySQL DSN is a plausible install
// target. It rejects DSNs that do not parse and DSNs whose network is not TCP or
// whose host is empty. This keeps the pre-auth install check endpoints from being
// used as free-form outbound dialers (P0-8 SSRF surface). Because the endpoint
// is public before installation, only public TCP database addresses are valid.
var installLookupIP = net.DefaultResolver.LookupIPAddr
var installDialerSequence atomic.Uint64

func publicInstallIP(ip net.IP) bool {
	if ip == nil || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() {
		return false
	}
	// Global-unicast is a routing property, not a guarantee that an address is
	// safe to dial from an unauthenticated installer. Reject shared CGNAT,
	// metadata/special-use, documentation, benchmark, multicast and reserved
	// ranges explicitly so a public pre-install endpoint cannot probe them.
	for _, network := range installDeniedIPNetworks {
		if network.Contains(ip) {
			return false
		}
	}
	return true
}

var installDeniedIPNetworks = func() []*net.IPNet {
	cidrs := []string{
		"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24",
		"198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "224.0.0.0/4",
		"240.0.0.0/4", "2001:db8::/32",
	}
	result := make([]*net.IPNet, 0, len(cidrs))
	for _, cidr := range cidrs {
		_, network, err := net.ParseCIDR(cidr)
		if err == nil {
			result = append(result, network)
		}
	}
	return result
}()

// pinInstallMySQLDSN resolves a hostname once, validates every returned address,
// and registers a request-scoped driver network that dials the checked address.
// The DSN retains the original hostname so TLS still validates the database
// certificate against that name, while the custom dialer prevents DNS rebinding.
func pinInstallMySQLDSN(ctx context.Context, dsn string) (string, func(), bool) {
	noop := func() {}
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return "", noop, false
	}
	cfg, err := mysqlDriver.ParseDSN(dsn)
	if err != nil {
		return "", noop, false
	}
	// The public installer must never grant a remote MySQL server permission to
	// read local files or downgrade authentication. In particular,
	// allowAllFiles lets a malicious server choose any process-readable path via
	// LOCAL INFILE before the application has been installed or authenticated.
	if cfg.AllowAllFiles || cfg.AllowCleartextPasswords || cfg.AllowFallbackToPlaintext || cfg.AllowOldPasswords {
		return "", noop, false
	}
	originalNetwork := strings.ToLower(cfg.Net)
	switch originalNetwork {
	case "tcp", "tcp4", "tcp6":
	default:
		return "", noop, false
	}
	// The address must name a real host, not a bare ":3306" or "()".
	host, port, err := net.SplitHostPort(cfg.Addr)
	if err != nil {
		return "", noop, false
	}
	host = strings.Trim(strings.TrimSpace(host), "[]")
	if host == "" {
		return "", noop, false
	}
	// This endpoint is unauthenticated and opens an outbound database
	// connection, so local/private destinations are never valid install
	// targets.  Keep the check on the parsed address rather than the original
	// DSN to cover IPv4, IPv6 and IPv4-mapped IPv6 spellings.
	hostLower := strings.TrimSuffix(strings.ToLower(host), ".")
	if hostLower == "localhost" || strings.HasSuffix(hostLower, ".localhost") {
		return "", noop, false
	}
	if ip := net.ParseIP(host); ip != nil {
		if !publicInstallIP(ip) {
			return "", noop, false
		}
		return cfg.FormatDSN(), noop, true
	}

	lookupCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	addresses, err := installLookupIP(lookupCtx, host)
	if err != nil || len(addresses) == 0 {
		return "", noop, false
	}
	var selected net.IP
	for _, address := range addresses {
		ip := address.IP
		if !publicInstallIP(ip) {
			// Reject mixed public/private answers as well. Picking only the public
			// answer would leave failover and rebinding behavior ambiguous.
			return "", noop, false
		}
		if selected == nil && (originalNetwork != "tcp4" || ip.To4() != nil) && (originalNetwork != "tcp6" || ip.To4() == nil) {
			selected = ip
		}
	}
	if selected == nil {
		return "", noop, false
	}
	pinnedAddress := net.JoinHostPort(selected.String(), port)
	dialerName := fmt.Sprintf("xiaohei_install_%d", installDialerSequence.Add(1))
	mysqlDriver.RegisterDialContext(dialerName, func(dialCtx context.Context, _ string) (net.Conn, error) {
		var dialer net.Dialer
		return dialer.DialContext(dialCtx, originalNetwork, pinnedAddress)
	})
	cfg.Net = dialerName
	cleanup := func() { mysqlDriver.DeregisterDialContext(dialerName) }
	return cfg.FormatDSN(), cleanup, true
}

func validateInstallMySQLDSN(dsn string) bool {
	_, cleanup, ok := pinInstallMySQLDSN(context.Background(), dsn)
	cleanup()
	return ok
}

var installLockPathOverride string

// Rate limits for the unauthenticated install endpoints. The installer accepts an
// arbitrary DB DSN and echoes reachability checks, so without a limit it doubles
// as a cheap port scanner / SSRF probe and a brute-force oracle for the admin
// path. Limits are deliberately generous for a human operator iterating on a DSN.
var (
	installRateLimiter    = newRateLimiter()
	checkAdminPathLimiter = newRateLimiter()
)

type installDatabaseInput struct {
	Type string `json:"type"`
	Path string `json:"path"`
	DSN  string `json:"dsn"`
}

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

func (h *Handler) InstallGenerateAdminPath(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) ValidateAdminPathHandler(c *gin.Context) {
	var payload struct {
		Path string `json:"path"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}

	if err := ValidateAdminPath(payload.Path); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": true})
}

func (h *Handler) CheckAdminPath(c *gin.Context) {
	if !checkAdminPathLimiter.Allow("check-admin-path:"+strings.TrimSpace(c.ClientIP()), 60, 10*time.Minute) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": domain.ErrTooManyRequests.Error()})
		return
	}
	var payload struct {
		Path string `json:"path"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}

	path := strings.TrimSpace(payload.Path)
	if path == "" {
		c.JSON(http.StatusOK, gin.H{"is_admin": false})
		return
	}

	// 只接受与 ValidateAdminPath 一致的 [a-zA-Z0-9]+ 单段路径：绝对 URL、协议
	// 相对、userinfo、多段路径一律返回 false、不进入比较，避免端点成为任意
	// 字符串的枚举 oracle，也不泄露格式判定（统一 is_admin:false）。
	if err := ValidateAdminPath(path); err != nil {
		c.JSON(http.StatusOK, gin.H{"is_admin": false})
		return
	}

	// 获取配置的管理端路径
	configuredPath := GetAdminPathFromSettings(h.settingsSvc)

	// 如果没有配置，默认是 admin
	if configuredPath == "" {
		configuredPath = "admin"
	}

	isAdmin := strings.EqualFold(path, configuredPath)
	c.JSON(http.StatusOK, gin.H{"is_admin": isAdmin})
}

func (h *Handler) InstallDBCheck(c *gin.Context) {
	if h.IsInstalled() {
		c.JSON(http.StatusConflict, gin.H{"error": domain.ErrAlreadyInstalled.Error()})
		return
	}
	if !installRateLimiter.Allow("install:check:"+strings.TrimSpace(c.ClientIP()), 30, 10*time.Minute) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": domain.ErrTooManyRequests.Error()})
		return
	}
	var payload struct {
		DB installDatabaseInput `json:"db"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}

	dbType := strings.ToLower(strings.TrimSpace(payload.DB.Type))
	switch dbType {
	case "":
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrDbTypeRequired.Error()})
		return
	case "sqlite":
		// ok
	case "mysql":
		// ok
	case "postgres", "postgresql":
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrDbTypeNotSupportedYetPostgresqlDisabledTemporarily.Error()})
		return
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrUnknownDbType.Error()})
		return
	}

	cfg := config.Config{
		DBType: dbType,
		DBPath: strings.TrimSpace(payload.DB.Path),
		DBDSN:  strings.TrimSpace(payload.DB.DSN),
	}
	if cfg.DBType == "sqlite" && cfg.DBPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrDbPathRequiredForSqlite.Error()})
		return
	}
	if cfg.DBType == "mysql" && cfg.DBDSN == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrDbDsnRequiredForMysql.Error()})
		return
	}
	if cfg.DBType == "mysql" {
		pinnedDSN, cleanup, ok := pinInstallMySQLDSN(c.Request.Context(), cfg.DBDSN)
		if !ok {
			// A DSN that does not parse, uses a non-tcp network, or carries an empty
			// host is rejected up front: it cannot be a usable install target and
			// accepting it turns the pre-auth check endpoint into a free-form
			// outbound dialer (P0-8 SSRF surface).
			c.JSON(http.StatusBadRequest, gin.H{"error": errDBCheckFailed})
			return
		}
		defer cleanup()
		checkCfg := cfg
		checkCfg.DBDSN = pinnedDSN
		cfg = checkCfg
	}

	conn, err := db.Open(cfg)
	if err != nil {
		log.Printf("install: db open failed type=%s err=%v", cfg.DBType, err)
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": errDBCheckFailed})
		return
	}
	defer conn.SQL.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := conn.SQL.PingContext(ctx); err != nil {
		log.Printf("install: db ping failed type=%s err=%v", cfg.DBType, err)
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": errDBCheckFailed})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) InstallRun(c *gin.Context) {
	if h.IsInstalled() {
		c.JSON(http.StatusConflict, gin.H{"error": domain.ErrAlreadyInstalled.Error()})
		return
	}
	if !installRateLimiter.Allow("install:run:"+strings.TrimSpace(c.ClientIP()), 10, 10*time.Minute) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": domain.ErrTooManyRequests.Error()})
		return
	}

	var payload struct {
		DB   installDatabaseInput `json:"db"`
		Site struct {
			Name      string `json:"name"`
			URL       string `json:"url"`
			AdminPath string `json:"admin_path"`
		} `json:"site"`
		Admin struct {
			Username string `json:"username"`
			Password string `json:"password"`
		} `json:"admin"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}

	dbType := strings.ToLower(strings.TrimSpace(payload.DB.Type))
	switch dbType {
	case "":
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrDbTypeRequired.Error()})
		return
	case "sqlite":
		// ok
	case "mysql":
		// ok
	case "postgres", "postgresql":
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrDbTypeNotSupportedYetPostgresqlDisabledTemporarily.Error()})
		return
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrUnknownDbType.Error()})
		return
	}

	siteName := strings.TrimSpace(payload.Site.Name)
	siteURL := strings.TrimSpace(payload.Site.URL)
	adminPath := strings.TrimSpace(payload.Site.AdminPath)
	adminUser := strings.TrimSpace(payload.Admin.Username)
	adminPass := strings.TrimSpace(payload.Admin.Password)
	if siteName == "" || adminUser == "" || adminPass == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrSiteNameAdminUsernameAdminPasswordRequired.Error()})
		return
	}
	if len(adminPass) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrAdminPasswordTooShort.Error()})
		return
	}
	// The installer derives the admin email as <username>@local (there is no
	// separate email field), so a username containing '@' or whitespace would
	// produce an invalid address. Reject those up front.
	if strings.ContainsAny(adminUser, " \t@") {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	// site.url is optional but, when given, must be an absolute http(s) URL so it
	// is not persisted verbatim and break later frontend use.
	if siteURL != "" {
		u, err := url.Parse(siteURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
			return
		}
	}

	// 校验管理端路径
	if err := ValidateAdminPath(adminPath); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cfg := config.Config{
		DBType: dbType,
		DBPath: strings.TrimSpace(payload.DB.Path),
		DBDSN:  strings.TrimSpace(payload.DB.DSN),
	}
	if cfg.DBType == "sqlite" && cfg.DBPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrDbPathRequiredForSqlite.Error()})
		return
	}
	if cfg.DBType == "mysql" && cfg.DBDSN == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrDbDsnRequiredForMysql.Error()})
		return
	}
	dbCfg := cfg
	if cfg.DBType == "mysql" {
		pinnedDSN, cleanup, ok := pinInstallMySQLDSN(c.Request.Context(), cfg.DBDSN)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": errDBCheckFailed})
			return
		}
		defer cleanup()
		dbCfg.DBDSN = pinnedDSN
	}

	// Atomically claim the install lock before any side effect (db open, migrate,
	// seed, admin creation, config write). O_EXCL turns the create into a
	// test-and-set: a concurrent request that also passed the IsInstalled() check
	// above loses the race here and is rejected with 409 instead of re-initializing
	// the system and overwriting the admin credentials.
	lockPath := installLockPath()
	if dir := filepath.Dir(lockPath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrLockDirError.Error()})
			return
		}
	}
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if os.IsExist(err) {
			c.JSON(http.StatusConflict, gin.H{"error": domain.ErrAlreadyInstalled.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrWriteLockFailed.Error()})
		return
	}
	_, _ = lockFile.Write([]byte("installing " + time.Now().Format(time.RFC3339)))
	_ = lockFile.Close()

	// A failed install must not leave the system marked as installed, otherwise
	// the operator could never retry: the lock is removed so the next attempt can
	// claim it again, and any DB completion marker THIS attempt claimed is cleared
	// (only a marker this attempt inserted — never a pre-existing one — so a
	// replayed install can never reset the admin password).
	installed := false
	markerClaimed := false
	var repoAny *repo.GormRepo
	// Registered first, runs LAST (LIFO): remove the lock on any failure. DB
	// connections are closed by the later defer, so this must not touch the DB.
	defer func() {
		if !installed {
			_ = os.Remove(lockPath)
		}
	}()

	conn, err := db.Open(dbCfg)
	if err != nil {
		log.Printf("install: db open failed type=%s err=%v", dbCfg.DBType, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": errDBCheckFailed})
		return
	}
	defer conn.SQL.Close()

	if err := repo.Migrate(conn.Gorm); err != nil {
		log.Printf("install: migrate failed err=%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": errDBMigrateFailed})
		return
	}
	if err := seed.EnsureSettings(conn.Gorm); err != nil {
		log.Printf("install: seed settings failed err=%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": errDBSeedFailed})
		return
	}
	if err := seed.EnsurePermissionDefaults(conn.Gorm); err != nil {
		log.Printf("install: seed permission defaults failed err=%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": errDBSeedFailed})
		return
	}
	if err := seed.EnsurePermissionGroups(conn.Gorm); err != nil {
		log.Printf("install: seed permission groups failed err=%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": errDBSeedFailed})
		return
	}
	// CMS defaults and base seed (only if empty) to keep install snappy and idempotent.
	if err := seed.EnsureCMSDefaults(conn.Gorm); err != nil {
		log.Printf("install: seed cms defaults failed err=%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": errDBSeedFailed})
		return
	}
	if err := seed.SeedIfEmpty(conn.Gorm); err != nil {
		log.Printf("install: seed failed err=%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": errDBSeedFailed})
		return
	}

	ctx := context.Background()
	repoAny = repo.NewGormRepo(conn.Gorm)

	// Registered after conn.SQL.Close() so it runs BEFORE the DB is closed (LIFO):
	// the marker claim is cleared while the connection is still usable. It only
	// clears a marker this attempt inserted (markerClaimed), never a pre-existing
	// completion marker, so a replayed install stays refused even on rollback.
	defer func() {
		if installed || !markerClaimed || repoAny == nil {
			return
		}
		_ = repoAny.DeleteSetting(context.Background(), installCompletedSettingKey)
	}()

	// Atomically claim the DB-side completion marker. The claim is a unique-key
	// insert: only one host sharing this database (and only one replay after the
	// lock file is lost) can win it, so the "installer creates/resets the admin"
	// side effects below can never run twice on the same DB. A read-then-check
	// would let two hosts both pass; the atomic insert makes the claim the
	// serialization point.
	claimed, err := repoAny.ClaimSettingIfAbsent(ctx, installCompletedSettingKey, "claimed")
	if err != nil {
		log.Printf("install: claim completion marker failed err=%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": errDBClaimFailed})
		return
	}
	if !claimed {
		c.JSON(http.StatusConflict, gin.H{"error": domain.ErrAlreadyInstalled.Error()})
		return
	}
	markerClaimed = true

	installSettings := []domain.Setting{{Key: "site_name", ValueJSON: siteName}}
	if siteURL != "" {
		installSettings = append(installSettings, domain.Setting{Key: "site_url", ValueJSON: siteURL})
	}
	if adminPath != "" {
		installSettings = append(installSettings, domain.Setting{Key: "admin_path", ValueJSON: adminPath})
	}
	if err := repoAny.UpsertSettings(ctx, installSettings); err != nil {
		log.Printf("install: persist initial settings failed err=%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrPersistConfigFailed.Error()})
		return
	}

	existingAdmin, adminLookupErr := repoAny.GetUserByUsernameOrEmail(ctx, adminUser)
	if adminLookupErr != nil && !errors.Is(adminLookupErr, appshared.ErrNotFound) {
		log.Printf("install: query admin failed err=%v", adminLookupErr)
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrQueryAdminFailed.Error()})
		return
	}
	// Reject an install that would promote an existing ordinary user (matched by
	// username or email) to super-admin and hand it an attacker-chosen password.
	if adminLookupErr == nil && existingAdmin.Role != domain.UserRoleAdmin {
		c.JSON(http.StatusConflict, gin.H{"error": domain.ErrAlreadyInstalled.Error()})
		return
	}

	groups, err := repoAny.ListPermissionGroups(ctx)
	if err != nil {
		log.Printf("install: list permission groups failed err=%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrPermissionGroupQueryFailed.Error()})
		return
	}
	superAdminGroupID, ok := findSuperAdminGroupID(groups)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrPermissionGroupMissing.Error()})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(adminPass), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrHashFailed.Error()})
		return
	}
	if adminLookupErr == nil {
		existingAdmin.Username = adminUser
		if strings.TrimSpace(existingAdmin.Email) == "" {
			existingAdmin.Email = adminUser + "@local"
		}
		existingAdmin.Role = domain.UserRoleAdmin
		existingAdmin.Status = domain.UserStatusActive
		existingAdmin.PermissionGroupID = &superAdminGroupID
		if err := repoAny.UpdateUser(ctx, existingAdmin); err != nil {
			log.Printf("install: update admin failed err=%v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrUpdateAdminFailed.Error()})
			return
		}
		if err := repoAny.UpdateUserPassword(ctx, existingAdmin.ID, string(hash)); err != nil {
			log.Printf("install: update admin password failed err=%v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrUpdateAdminPasswordFailed.Error()})
			return
		}
	} else {
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
				adminNow, getErr := repoAny.GetUserByUsernameOrEmail(ctx, adminUser)
				if getErr == nil && adminNow.Role == domain.UserRoleAdmin {
					adminNow.Role = domain.UserRoleAdmin
					adminNow.Status = domain.UserStatusActive
					adminNow.PermissionGroupID = &superAdminGroupID
					if strings.TrimSpace(adminNow.Email) == "" {
						adminNow.Email = adminUser + "@local"
					}
					if updateErr := repoAny.UpdateUser(ctx, adminNow); updateErr == nil {
						if passErr := repoAny.UpdateUserPassword(ctx, adminNow.ID, string(hash)); passErr == nil {
							goto persistConfig
						}
					}
				}
			}
			log.Printf("install: create admin failed err=%v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrUpdateAdminFailed.Error()})
			return
		}
	}

persistConfig:
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
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Printf("install: create config directory failed err=%v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrPersistConfigFailed.Error()})
			return
		}
	}

	out := map[string]any{}
	if existing, err := os.ReadFile(configPath); err == nil {
		if err := yaml.Unmarshal(existing, &out); err != nil {
			log.Printf("install: parse existing config failed err=%v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrPersistConfigFailed.Error()})
			return
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		log.Printf("install: read existing config failed err=%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrPersistConfigFailed.Error()})
		return
	}
	// These values are NOT persisted in the config file. They are created/managed via install + DB settings.
	delete(out, "admin")
	delete(out, "automation")
	out["db"] = map[string]any{
		"type": cfg.DBType,
		"path": cfg.DBPath,
		"dsn":  cfg.DBDSN,
	}

	b, err := yaml.Marshal(&out)
	if err != nil {
		log.Printf("install: encode config failed path=%s err=%v", configPath, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrPersistConfigFailed.Error()})
		return
	}
	if writeErr := writeInstallConfigAtomic(configPath, b); writeErr != nil {
		log.Printf("install: persist config failed path=%s err=%v", configPath, writeErr)
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrPersistConfigFailed.Error()})
		return
	}

	// Persist the completion marker inside the DB so the installed state does not
	// depend solely on the lock file. This is part of the install commit: failure
	// must not report success or leave a lock that makes recovery impossible.
	if err := repoAny.UpsertSetting(ctx, domain.Setting{Key: installCompletedSettingKey, ValueJSON: time.Now().Format(time.RFC3339)}); err != nil {
		log.Printf("install: failed to persist completion marker: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrPersistConfigFailed.Error()})
		return
	}

	// Overwrite the lock claimed at the start with a completion marker.
	if err := os.WriteFile(lockPath, []byte("installed "+time.Now().Format(time.RFC3339)), 0o600); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrWriteLockFailed.Error()})
		return
	}
	installed = true

	c.JSON(http.StatusOK, gin.H{
		"ok":               true,
		"restart_required": cfg.DBType == "sqlite",
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

func writeInstallConfigAtomic(path string, data []byte) (resultErr error) {
	dir := filepath.Dir(path)
	if dir == "" {
		dir = "."
	}
	tmp, err := os.CreateTemp(dir, ".xiaohei-install-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = tmp.Close()
		if resultErr != nil {
			_ = os.Remove(tmpPath)
		}
	}()
	if err := tmp.Chmod(0o600); err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	directory, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
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
