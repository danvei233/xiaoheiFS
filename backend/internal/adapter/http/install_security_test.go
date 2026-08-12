package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"

	adapterhttp "xiaoheiplay/internal/adapter/http"
	"xiaoheiplay/internal/adapter/repo/core"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/pkg/config"
	"xiaoheiplay/internal/pkg/db"
	"xiaoheiplay/internal/testutil"
)

func installPayload(sqlitePath, username, password string) map[string]any {
	return map[string]any{
		"db": map[string]any{
			"type": "sqlite",
			"path": sqlitePath,
		},
		"site": map[string]any{
			"name":       "Install Security",
			"url":        "http://localhost:8080",
			"admin_path": "InstallSecurity",
		},
		"admin": map[string]any{
			"username": username,
			"password": password,
		},
	}
}

// resetInstallLimiters clears the shared package-level install rate-limit state
// so a test that deliberately exhausts a limit (or bursts many concurrent
// requests) does not leak its exhaustion into the next test.
func resetInstallLimiters(t *testing.T) {
	t.Helper()
	adapterhttp.ResetInstallRateLimitersForTest()
	t.Cleanup(adapterhttp.ResetInstallRateLimitersForTest)
}

// TestInstallRun_ConcurrentOnlyOneSucceeds guards against the TOCTOU race where
// concurrent install requests all pass the initial IsInstalled() check before
// the install.lock is created, causing duplicate initialization (admin password
// overwrites, config corruption, double process exit). Exactly one request may
// succeed; every other concurrent request must receive a 409.
func TestInstallRun_ConcurrentOnlyOneSucceeds(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tmp := chdirTemp(t)
	lockPath := filepath.Join(tmp, "install.lock")

	adapterhttp.SetInstallLockPathForTest(lockPath)
	t.Cleanup(func() { adapterhttp.SetInstallLockPathForTest("") })
	resetInstallLimiters(t)

	server := adapterhttp.NewInstallBootstrapServer("test-secret")
	sqlitePath := filepath.Join(tmp, "data", "install.db")

	const n = 8
	start := make(chan struct{})
	codes := make([]int, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			payload := map[string]any{
				"db": map[string]any{
					"type": "sqlite",
					"path": sqlitePath,
				},
				"site": map[string]any{
					"name":       "Concurrent",
					"url":        "http://localhost:8080",
					"admin_path": "InstallConcurrent",
				},
				"admin": map[string]any{
					"username": "installer_concurrent",
					"password": "password123",
				},
			}
			var buf bytes.Buffer
			_ = json.NewEncoder(&buf).Encode(payload)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/install", &buf)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			server.Engine.ServeHTTP(rec, req)
			codes[idx] = rec.Code
		}(i)
	}
	close(start)
	wg.Wait()

	okCount, conflictCount := 0, 0
	for _, c := range codes {
		switch c {
		case http.StatusOK:
			okCount++
		case http.StatusConflict:
			conflictCount++
		}
	}
	if okCount != 1 || conflictCount != n-1 {
		t.Fatalf("expected exactly 1 success and %d conflicts, got ok=%d conflicts=%d codes=%v",
			n-1, okCount, conflictCount, codes)
	}
}

// TestInstallRun_ReplayAfterCompletedRefused ensures a completed install cannot be
// replayed to reset the admin password once install.lock is lost (CWD change,
// config volume restored, deleted lock). The DB keeps a completion marker that is
// independent of the lock file, so a replayed install is refused with 409 and the
// stored password hash is untouched.
func TestInstallRun_ReplayAfterCompletedRefused(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tmp := chdirTemp(t)
	lockPath := filepath.Join(tmp, "install.lock")

	adapterhttp.SetInstallLockPathForTest(lockPath)
	t.Cleanup(func() { adapterhttp.SetInstallLockPathForTest("") })
	resetInstallLimiters(t)

	server := adapterhttp.NewInstallBootstrapServer("test-secret")
	sqlitePath := filepath.Join(tmp, "data", "install.db")

	rec := testutil.DoJSON(t, server.Engine, "POST", "/api/v1/install", installPayload(sqlitePath, "installer_admin", "password123"), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("first install status=%d body=%s", rec.Code, rec.Body.String())
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
	originalHash := admin.PasswordHash

	// Simulate lock loss (deployment CWD change, config volume restore, manual delete).
	if err := os.Remove(lockPath); err != nil {
		t.Fatalf("remove lock: %v", err)
	}

	// Replay the install with a different password; must be refused.
	rec = testutil.DoJSON(t, server.Engine, "POST", "/api/v1/install", installPayload(sqlitePath, "installer_admin", "attacker-password"), "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("replayed install status=%d body=%s, want 409", rec.Code, rec.Body.String())
	}

	adminNow, err := r.GetUserByUsernameOrEmail(context.Background(), "installer_admin")
	if err != nil {
		t.Fatalf("re-query admin: %v", err)
	}
	if adminNow.PasswordHash != originalHash {
		t.Fatalf("replayed install changed admin password hash")
	}
	if adminNow.Role != domain.UserRoleAdmin {
		t.Fatalf("admin role changed by replay: %q", adminNow.Role)
	}
}

// TestInstallRun_ReplayTwiceStillRefused extends the single-replay test with the
// two-step replay regression (plugin_install F5): the DB completion marker must
// SURVIVE a refused replay (the rollback must not clear a pre-existing marker),
// so a second replay is refused just like the first and the stored admin hash
// never changes.
func TestInstallRun_ReplayTwiceStillRefused(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tmp := chdirTemp(t)
	lockPath := filepath.Join(tmp, "install.lock")

	adapterhttp.SetInstallLockPathForTest(lockPath)
	t.Cleanup(func() { adapterhttp.SetInstallLockPathForTest("") })
	resetInstallLimiters(t)

	server := adapterhttp.NewInstallBootstrapServer("test-secret")
	sqlitePath := filepath.Join(tmp, "data", "install.db")

	rec := testutil.DoJSON(t, server.Engine, "POST", "/api/v1/install", installPayload(sqlitePath, "installer_admin", "password123"), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("first install status=%d body=%s", rec.Code, rec.Body.String())
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
	originalHash := admin.PasswordHash

	if err := os.Remove(lockPath); err != nil {
		t.Fatalf("remove lock: %v", err)
	}

	for attempt := 1; attempt <= 2; attempt++ {
		rec = testutil.DoJSON(t, server.Engine, "POST", "/api/v1/install", installPayload(sqlitePath, "installer_admin", "attacker-password"), "")
		if rec.Code != http.StatusConflict {
			t.Fatalf("replay #%d status=%d body=%s, want 409", attempt, rec.Code, rec.Body.String())
		}
		adminNow, err := r.GetUserByUsernameOrEmail(context.Background(), "installer_admin")
		if err != nil {
			t.Fatalf("re-query admin after replay #%d: %v", attempt, err)
		}
		if adminNow.PasswordHash != originalHash {
			t.Fatalf("replay #%d changed admin password hash", attempt)
		}
	}
}

// TestInstallRun_RefusesPromotingExistingNonAdmin ensures an install re-run that
// matches an existing ordinary user (by username) does not silently promote that
// user to super-admin with an attacker-chosen password.
func TestInstallRun_RefusesPromotingExistingNonAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tmp := chdirTemp(t)
	lockPath := filepath.Join(tmp, "install.lock")

	adapterhttp.SetInstallLockPathForTest(lockPath)
	t.Cleanup(func() { adapterhttp.SetInstallLockPathForTest("") })
	resetInstallLimiters(t)

	sqlitePath := filepath.Join(tmp, "data", "install.db")
	conn, err := db.Open(config.Config{DBType: "sqlite", DBPath: sqlitePath})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = conn.SQL.Close() })
	if err := repo.Migrate(conn.Gorm); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	r := repo.NewGormRepo(conn.Gorm)
	bob := &domain.User{
		Username:     "bob",
		Email:        "bob@example.com",
		PasswordHash: "pre-existing-hash",
		Role:         domain.UserRoleUser,
		Status:       domain.UserStatusActive,
	}
	if err := r.CreateUser(context.Background(), bob); err != nil {
		t.Fatalf("create normal user: %v", err)
	}

	server := adapterhttp.NewInstallBootstrapServer("test-secret")
	rec := testutil.DoJSON(t, server.Engine, "POST", "/api/v1/install", installPayload(sqlitePath, "bob", "attacker-password"), "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("install matched normal user status=%d body=%s, want 409", rec.Code, rec.Body.String())
	}

	bobNow, err := r.GetUserByUsernameOrEmail(context.Background(), "bob")
	if err != nil {
		t.Fatalf("re-query bob: %v", err)
	}
	if bobNow.Role == domain.UserRoleAdmin {
		t.Fatalf("install escalated ordinary user to admin")
	}
	if bobNow.PasswordHash != "pre-existing-hash" {
		t.Fatalf("install changed ordinary user password")
	}

	// The refused attempt claimed the completion marker but must have cleared it
	// on rollback, so a legitimate retry (after resolving the conflicting user)
	// is not blocked as a replay.
	marker, markerErr := r.GetSetting(context.Background(), "install_completed_at")
	if markerErr == nil && strings.TrimSpace(marker.ValueJSON) != "" {
		t.Fatalf("refused install left the completion marker behind: %+v", marker)
	}
}

// TestInstallDBCheck_DoesNotLeakConnectionDetails ensures raw driver errors
// (which contain host, port, db user and driver/version details) are not echoed
// back to the unauthenticated client that triggered the connection attempt.
func TestInstallDBCheck_DoesNotLeakConnectionDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tmp := chdirTemp(t)
	lockPath := filepath.Join(tmp, "install.lock")

	adapterhttp.SetInstallLockPathForTest(lockPath)
	t.Cleanup(func() { adapterhttp.SetInstallLockPathForTest("") })
	resetInstallLimiters(t)

	server := adapterhttp.NewInstallBootstrapServer("test-secret")
	// Port 1 on loopback is essentially always closed -> immediate connection
	// refused with a "dial tcp 127.0.0.1:1" style message.
	rec := testutil.DoJSON(t, server.Engine, http.MethodPost, "/api/v1/install/db/check", map[string]any{
		"db": map[string]any{"type": "mysql", "dsn": "baduser:badpass@tcp(127.0.0.1:1)/nope"},
	}, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("db check status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, leaked := range []string{"dial tcp", "127.0.0.1", ":1", "baduser", "badpass", "connection refused"} {
		if strings.Contains(body, leaked) {
			t.Fatalf("db check leaked %q in response: %s", leaked, body)
		}
	}
}

// TestInstallRun_DoesNotLeakDBSettingsError ensures InstallRun's DB open failure
// also returns a fixed message rather than the raw DSN connection error.
func TestInstallRun_DoesNotLeakDBSettingsError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tmp := chdirTemp(t)
	lockPath := filepath.Join(tmp, "install.lock")

	adapterhttp.SetInstallLockPathForTest(lockPath)
	t.Cleanup(func() { adapterhttp.SetInstallLockPathForTest("") })
	resetInstallLimiters(t)

	server := adapterhttp.NewInstallBootstrapServer("test-secret")
	rec := testutil.DoJSON(t, server.Engine, http.MethodPost, "/api/v1/install", map[string]any{
		"db": map[string]any{"type": "mysql", "dsn": "baduser:badpass@tcp(127.0.0.1:1)/nope"},
		"site": map[string]any{
			"name":       "Leak Test",
			"url":        "http://localhost:8080",
			"admin_path": "InstallLeak",
		},
		"admin": map[string]any{"username": "installer_leak", "password": "password123"},
	}, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("install status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, leaked := range []string{"dial tcp", "127.0.0.1", "baduser", "badpass"} {
		if strings.Contains(body, leaked) {
			t.Fatalf("install leaked %q in response: %s", leaked, body)
		}
	}
	// The failed attempt must not leave the install lock behind.
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatalf("failed install left lock file behind: %v", err)
	}
}

// TestInstallDBCheck_RateLimited guards against the unauthenticated installer
// being used as a cheap port scanner / SSRF probe: sustained probing must
// eventually be refused with 429.
func TestInstallDBCheck_RateLimited(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tmp := chdirTemp(t)
	lockPath := filepath.Join(tmp, "install.lock")

	adapterhttp.SetInstallLockPathForTest(lockPath)
	t.Cleanup(func() { adapterhttp.SetInstallLockPathForTest("") })
	resetInstallLimiters(t)
	adapterhttp.ResetInstallRateLimitersForTest()

	server := adapterhttp.NewInstallBootstrapServer("test-secret")
	const maxAttempts = 80
	var saw429 bool
	for i := 0; i < maxAttempts; i++ {
		// Invalid db.type: consumes a rate-limit slot, fails fast on validation,
		// no DB side effect.
		rec := testutil.DoJSON(t, server.Engine, http.MethodPost, "/api/v1/install/db/check", map[string]any{
			"db": map[string]any{"type": ""},
		}, "")
		if rec.Code == http.StatusTooManyRequests {
			saw429 = true
			break
		}
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("unexpected db check code %d: %s", rec.Code, rec.Body.String())
		}
	}
	if !saw429 {
		t.Fatalf("install db/check was not rate limited within %d requests", maxAttempts)
	}
}

// TestInstallDBCheck_RateLimitedWithSpoofedForwardedFor is the regression for
// the ClientIP spoofing bypass: gin's default TrustedProxies trusts ANY
// X-Forwarded-For header, so without an explicit SetTrustedProxies(nil) an
// attacker could rotate a forged header per request and never hit the install
// rate limit (turning db/check into an unlimited port scanner). The server must
// key the limiter on the real TCP peer, making the forged header ineffective.
func TestInstallDBCheck_RateLimitedWithSpoofedForwardedFor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tmp := chdirTemp(t)
	lockPath := filepath.Join(tmp, "install.lock")

	adapterhttp.SetInstallLockPathForTest(lockPath)
	t.Cleanup(func() { adapterhttp.SetInstallLockPathForTest("") })
	resetInstallLimiters(t)
	adapterhttp.ResetInstallRateLimitersForTest()

	server := adapterhttp.NewInstallBootstrapServer("test-secret")
	const maxAttempts = 80
	var saw429 bool
	for i := 0; i < maxAttempts; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/install/db/check",
			bytes.NewBufferString(`{"db":{"type":""}}`))
		req.Header.Set("Content-Type", "application/json")
		// Rotate a forged X-Forwarded-For every request; it must not reset the
		// rate-limit budget keyed on the real peer.
		req.Header.Set("X-Forwarded-For", fmt.Sprintf("198.51.100.%d", i))
		rec := httptest.NewRecorder()
		server.Engine.ServeHTTP(rec, req)
		if rec.Code == http.StatusTooManyRequests {
			saw429 = true
			break
		}
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("unexpected db check code %d: %s", rec.Code, rec.Body.String())
		}
	}
	if !saw429 {
		t.Fatalf("install db/check was not rate limited even while rotating X-Forwarded-For")
	}
}

// TestCheckAdminPath_RateLimited guards the unauthenticated admin-path boolean
// oracle (used to confirm/enumerate the admin path) against brute force.
func TestCheckAdminPath_RateLimited(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tmp := chdirTemp(t)
	lockPath := filepath.Join(tmp, "install.lock")

	adapterhttp.SetInstallLockPathForTest(lockPath)
	t.Cleanup(func() { adapterhttp.SetInstallLockPathForTest("") })
	resetInstallLimiters(t)
	adapterhttp.ResetInstallRateLimitersForTest()

	// check-admin-path is only reachable after installation completes: the
	// install gate (router.go) 503s all non-install APIs while not installed.
	if err := os.WriteFile(lockPath, []byte("installed"), 0o600); err != nil {
		t.Fatalf("write install lock: %v", err)
	}

	server := adapterhttp.NewInstallBootstrapServer("test-secret")
	const maxAttempts = 120
	var saw429 bool
	for i := 0; i < maxAttempts; i++ {
		rec := testutil.DoJSON(t, server.Engine, http.MethodPost, "/api/v1/check-admin-path", map[string]any{"path": "guess"}, "")
		if rec.Code == http.StatusTooManyRequests {
			saw429 = true
			break
		}
	}
	if !saw429 {
		t.Fatalf("check-admin-path was not rate limited within %d requests", maxAttempts)
	}
}

// TestInstallRun_RejectsInvalidAdminUsername ensures an admin username that would
// produce a broken derived email (adminUser@local) is rejected: the installer has
// no separate email field, so usernames containing '@' or whitespace would
// otherwise create an invalid admin email.
func TestInstallRun_RejectsInvalidAdminUsername(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tmp := chdirTemp(t)
	lockPath := filepath.Join(tmp, "install.lock")

	adapterhttp.SetInstallLockPathForTest(lockPath)
	t.Cleanup(func() { adapterhttp.SetInstallLockPathForTest("") })
	resetInstallLimiters(t)

	server := adapterhttp.NewInstallBootstrapServer("test-secret")
	sqlitePath := filepath.Join(tmp, "data", "install.db")

	for _, bad := range []string{"a b", "user@host", "user\tname"} {
		rec := testutil.DoJSON(t, server.Engine, http.MethodPost, "/api/v1/install", installPayload(sqlitePath, bad, "password123"), "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("username %q accepted (status=%d) body=%s", bad, rec.Code, rec.Body.String())
		}
	}
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatalf("rejected install left lock file behind: %v", err)
	}
}

// TestInstallRun_RejectsInvalidSiteURL ensures a non-URL site.url value is
// rejected instead of being persisted verbatim and breaking later frontend use.
func TestInstallRun_RejectsInvalidSiteURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tmp := chdirTemp(t)
	lockPath := filepath.Join(tmp, "install.lock")

	adapterhttp.SetInstallLockPathForTest(lockPath)
	t.Cleanup(func() { adapterhttp.SetInstallLockPathForTest("") })
	resetInstallLimiters(t)

	server := adapterhttp.NewInstallBootstrapServer("test-secret")
	sqlitePath := filepath.Join(tmp, "data", "install.db")

	payload := installPayload(sqlitePath, "installer_admin", "password123")
	payload["site"].(map[string]any)["url"] = "not-a-url"
	rec := testutil.DoJSON(t, server.Engine, http.MethodPost, "/api/v1/install", payload, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("site.url %q accepted (status=%d) body=%s", "not-a-url", rec.Code, rec.Body.String())
	}
}
