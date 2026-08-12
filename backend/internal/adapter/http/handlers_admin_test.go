package http_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/testutil"
	"xiaoheiplay/internal/testutilhttp"
)

func TestHandlers_AdminRoutesSmoke(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)

	group := domain.PermissionGroup{Name: "all", PermissionsJSON: `["*"]`}
	if err := env.Repo.CreatePermissionGroup(context.Background(), &group); err != nil {
		t.Fatalf("create group: %v", err)
	}
	admin := testutil.CreateAdmin(t, env.Repo, "admin", "admin@example.com", "pass", group.ID)
	adminToken := testutil.IssueJWT(t, env.JWTSecret, admin.ID, "admin", time.Hour)

	user := testutil.CreateUser(t, env.Repo, "u1", "u1@example.com", "pass")
	userToken := testutil.IssueJWT(t, env.JWTSecret, user.ID, "user", time.Hour)

	rec := testutil.DoJSON(t, env.Router, http.MethodGet, "/admin/api/v1/users", nil, userToken)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/auth/login", map[string]any{
		"username": "admin",
		"password": "pass",
	}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("admin login code: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/users/"+testutil.Itoa(user.ID), map[string]any{
		"username": user.Username,
		"email":    "u1-new@example.com",
		"qq":       "",
	}, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin user update code: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/users/"+testutil.Itoa(user.ID)+"/status", map[string]any{
		"status": "disabled",
	}, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin user status code: %d", rec.Code)
	}

	order := domain.Order{UserID: user.ID, OrderNo: "ORD-DEL", Status: domain.OrderStatusPendingPayment, TotalAmount: 1000, Currency: "CNY"}
	if err := env.Repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodDelete, "/admin/api/v1/orders/"+testutil.Itoa(order.ID), nil, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin order delete code: %d", rec.Code)
	}
	approved := domain.Order{UserID: user.ID, OrderNo: "ORD-DEL-APPROVED", Status: domain.OrderStatusApproved, TotalAmount: 1000, Currency: "CNY"}
	if err := env.Repo.CreateOrder(context.Background(), &approved); err != nil {
		t.Fatalf("create approved order: %v", err)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodDelete, "/admin/api/v1/orders/"+testutil.Itoa(approved.ID), nil, adminToken)
	if rec.Code != http.StatusConflict {
		t.Fatalf("approved order delete should be 409, got %d", rec.Code)
	}

	perm := domain.Permission{Code: "order.view", Name: "Order View", Category: "order"}
	if err := env.Repo.UpsertPermission(context.Background(), &perm); err != nil {
		t.Fatalf("create permission: %v", err)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/permissions/order.view", map[string]any{
		"name":     "Order View",
		"category": "order",
	}, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("permissions update code: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/profile", map[string]any{
		"email": "admin2@example.com",
	}, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("profile update code: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/settings", map[string]any{
		"key":        "site_name",
		"value_json": "Test Site",
	}, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("settings update code: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/payments/providers/fake", map[string]any{
		"enabled":     true,
		"config_json": `{"pay_url":"x"}`,
	}, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("payment provider update code: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/scheduled-tasks/task.vps_refresh", map[string]any{
		"enabled": false,
	}, adminToken)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("scheduled task update code: %d", rec.Code)
	}
}

func TestHandlers_AdminVPSUpdate(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	group := domain.PermissionGroup{Name: "vps", PermissionsJSON: `["vps.*"]`}
	if err := env.Repo.CreatePermissionGroup(context.Background(), &group); err != nil {
		t.Fatalf("create group: %v", err)
	}
	admin := testutil.CreateAdmin(t, env.Repo, "adminvps", "adminvps@example.com", "pass", group.ID)
	adminToken := testutil.IssueJWT(t, env.JWTSecret, admin.ID, "admin", time.Hour)
	user := testutil.CreateUser(t, env.Repo, "u2", "u2@example.com", "pass")

	order := domain.Order{UserID: user.ID, OrderNo: "ORD-VPS", Status: domain.OrderStatusApproved, TotalAmount: 1000, Currency: "CNY"}
	if err := env.Repo.CreateOrder(context.Background(), &order); err != nil {
		t.Fatalf("create order: %v", err)
	}
	item := domain.OrderItem{OrderID: order.ID, Amount: 1000, Status: domain.OrderItemStatusApproved, Action: "create", SpecJSON: "{}"}
	if err := env.Repo.CreateOrderItems(context.Background(), []domain.OrderItem{item}); err != nil {
		t.Fatalf("create items: %v", err)
	}
	items, _ := env.Repo.ListOrderItems(context.Background(), order.ID)
	vps := domain.VPSInstance{
		UserID:               user.ID,
		OrderItemID:          items[0].ID,
		AutomationInstanceID: "123",
		Name:                 "vm",
		SystemID:             1,
		Status:               domain.VPSStatusUnknown,
		SpecJSON:             "{}",
	}
	if err := env.Repo.CreateInstance(context.Background(), &vps); err != nil {
		t.Fatalf("create vps: %v", err)
	}

	rec := testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/vps/"+testutil.Itoa(vps.ID), map[string]any{
		"status": "running",
	}, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("vps update code: %d", rec.Code)
	}

	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/vps/"+testutil.Itoa(vps.ID)+"/expire-at", map[string]any{
		"expire_at": time.Now().Add(24 * time.Hour).Format("2006-01-02"),
	}, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("vps expire update code: %d", rec.Code)
	}
}

// TestHandlers_AdminUserStatus_ValidatorFirst verifies P3-15 regression: the
// status enum is enforced declaratively by the shared validator, so a malformed
// value is rejected at binding (400) and a valid value is accepted. The
// service-layer guard is exercised separately in admin service tests.
func TestHandlers_AdminUserStatus_ValidatorFirst(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	groupID := ensureAdminGroup(t, env)
	admin := testutil.CreateAdmin(t, env.Repo, "status-admin", "status-admin@example.com", "pass123", groupID)
	adminToken := testutil.IssueJWT(t, env.JWTSecret, admin.ID, "admin", time.Hour)
	user := testutil.CreateUser(t, env.Repo, "status-user", "status-user@example.com", "pass123")

	// Invalid enum -> 400 from the validator (not a 200 that silently accepts).
	rec := testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/users/"+testutil.Itoa(user.ID)+"/status", map[string]any{
		"status": "bogus",
	}, adminToken)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid status must be rejected by binding, got %d", rec.Code)
	}
	reloaded, err := env.Repo.GetUserByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if reloaded.Status != domain.UserStatusActive {
		t.Fatalf("invalid status must not mutate the user, got %s", reloaded.Status)
	}

	// Valid enum -> 200.
	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/users/"+testutil.Itoa(user.ID)+"/status", map[string]any{
		"status": "disabled",
	}, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("valid status must be accepted, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestHandlers_AdminUserUpdate_SubAdminCannotDemotePrimary verifies P0-5 at the
// HTTP layer: a sub-admin holding only `user.update` must be rejected when
// trying to demote, disable or re-group the primary admin through
// PATCH /admin/api/v1/users/:id, and the primary admin's row must stay intact.
func TestHandlers_AdminUserUpdate_SubAdminCannotDemotePrimary(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	ctx := context.Background()

	group := domain.PermissionGroup{Name: "user-mgmt", PermissionsJSON: `["user.update"]`}
	if err := env.Repo.CreatePermissionGroup(ctx, &group); err != nil {
		t.Fatalf("create group: %v", err)
	}
	primary := testutil.CreateAdmin(t, env.Repo, "primary-http", "primary-http@example.com", "pass123", group.ID)
	sub := testutil.CreateAdmin(t, env.Repo, "sub-http", "sub-http@example.com", "pass123", group.ID)
	subToken := testutil.IssueJWT(t, env.JWTSecret, sub.ID, "admin", time.Hour)

	// Attempt to demote the primary admin to a regular user.
	rec := testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/users/"+testutil.Itoa(primary.ID), map[string]any{
		"username": primary.Username,
		"email":    primary.Email,
		"role":     "user",
	}, subToken)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("sub-admin demote primary must be rejected, got %d: %s", rec.Code, rec.Body.String())
	}

	// Attempt to disable the primary admin.
	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/users/"+testutil.Itoa(primary.ID), map[string]any{
		"username": primary.Username,
		"email":    primary.Email,
		"status":   "disabled",
	}, subToken)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("sub-admin disable primary must be rejected, got %d: %s", rec.Code, rec.Body.String())
	}

	// Attempt to move the primary admin to a different permission group.
	otherGroup := domain.PermissionGroup{Name: "other-http", PermissionsJSON: `["user.view"]`}
	if err := env.Repo.CreatePermissionGroup(ctx, &otherGroup); err != nil {
		t.Fatalf("create other group: %v", err)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/users/"+testutil.Itoa(primary.ID), map[string]any{
		"username":            primary.Username,
		"email":               primary.Email,
		"permission_group_id": otherGroup.ID,
	}, subToken)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("sub-admin re-group primary must be rejected, got %d: %s", rec.Code, rec.Body.String())
	}

	// The primary admin row is unchanged.
	reloaded, err := env.Repo.GetUserByID(ctx, primary.ID)
	if err != nil {
		t.Fatalf("reload primary: %v", err)
	}
	if reloaded.Role != domain.UserRoleAdmin || reloaded.Status != domain.UserStatusActive {
		t.Fatalf("primary admin must be unchanged, got role=%s status=%s", reloaded.Role, reloaded.Status)
	}

	// A profile-only self-edit by the primary admin is still allowed.
	primaryToken := testutil.IssueJWT(t, env.JWTSecret, primary.ID, "admin", time.Hour)
	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/users/"+testutil.Itoa(primary.ID), map[string]any{
		"username": primary.Username,
		"email":    "primary-http-new@example.com",
	}, primaryToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("primary self-edit profile must be allowed, got %d: %s", rec.Code, rec.Body.String())
	}
	reloaded, err = env.Repo.GetUserByID(ctx, primary.ID)
	if err != nil {
		t.Fatalf("reload primary after self-edit: %v", err)
	}
	if reloaded.Email != "primary-http-new@example.com" || reloaded.Role != domain.UserRoleAdmin {
		t.Fatalf("self-edit must apply profile only, got email=%s role=%s", reloaded.Email, reloaded.Role)
	}
}

// TestHandlers_AdminUserUpdate_SubAdminCannotReemailPrimary verifies the exact
// Finding-1 takeover vector: a sub-admin must not be able to change the primary
// admin's email to an attacker-controlled address and then drive the public
// forgot-password flow. The email change must be rejected and the primary
// admin's email must stay intact.
func TestHandlers_AdminUserUpdate_SubAdminCannotReemailPrimary(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	ctx := context.Background()

	group := domain.PermissionGroup{Name: "user-mgmt", PermissionsJSON: `["user.update"]`}
	if err := env.Repo.CreatePermissionGroup(ctx, &group); err != nil {
		t.Fatalf("create group: %v", err)
	}
	primary := testutil.CreateAdmin(t, env.Repo, "primary-mail", "primary-mail@example.com", "pass123", group.ID)
	sub := testutil.CreateAdmin(t, env.Repo, "sub-mail", "sub-mail@example.com", "pass123", group.ID)
	subToken := testutil.IssueJWT(t, env.JWTSecret, sub.ID, "admin", time.Hour)

	rec := testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/users/"+testutil.Itoa(primary.ID), map[string]any{
		"username": primary.Username,
		"email":    "attacker@evil.example",
	}, subToken)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("sub-admin re-email primary must be rejected, got %d: %s", rec.Code, rec.Body.String())
	}
	reloaded, err := env.Repo.GetUserByID(ctx, primary.ID)
	if err != nil {
		t.Fatalf("reload primary: %v", err)
	}
	if reloaded.Email != primary.Email || reloaded.Role != domain.UserRoleAdmin {
		t.Fatalf("primary email/role must be unchanged, got email=%s role=%s", reloaded.Email, reloaded.Role)
	}
}

// TestHandlers_AdminUserImpersonateCarriesImpersonator verifies Finding 7: the
// impersonation token must carry the operator admin's id in the `imp` claim and
// a DB audit record (action admin.impersonate) must be written, so impersonated
// actions are attributable to the admin who issued them instead of landing on
// the impersonated user with no trace.
func TestHandlers_AdminUserImpersonateCarriesImpersonator(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	ctx := context.Background()

	group := domain.PermissionGroup{Name: "admin-group", PermissionsJSON: `["*"]`}
	if err := env.Repo.CreatePermissionGroup(ctx, &group); err != nil {
		t.Fatalf("create group: %v", err)
	}
	primary := testutil.CreateAdmin(t, env.Repo, "primary-imp", "primary-imp@example.com", "pass", group.ID)
	primaryToken := testutil.IssueJWT(t, env.JWTSecret, primary.ID, "admin", time.Hour)
	target := testutil.CreateUser(t, env.Repo, "target-imp", "target-imp@example.com", "pass123")

	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/users/"+testutil.Itoa(target.ID)+"/impersonate", nil, primaryToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("impersonate must succeed, got %d: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal impersonate response: %v", err)
	}
	if body.AccessToken == "" {
		t.Fatalf("expected access token in impersonate response")
	}

	// The issued token must carry the operator's admin id in the imp claim.
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(body.AccessToken, claims, func(token *jwt.Token) (any, error) { return []byte(env.JWTSecret), nil })
	if err != nil {
		t.Fatalf("parse impersonation token: %v", err)
	}
	imp, _ := claims["imp"].(float64)
	if int64(imp) != primary.ID {
		t.Fatalf("impersonation token must carry operator admin id %d, got imp=%v", primary.ID, claims["imp"])
	}
	if claims["user_id"] == nil {
		t.Fatalf("impersonation token must still carry the target user id")
	}

	// The impersonation itself must be recorded in the DB audit log.
	logs, _, err := env.Repo.ListAuditLogs(ctx, appshared.AuditLogFilter{}, 50, 0)
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}
	found := false
	for _, l := range logs {
		if l.Action == "admin.impersonate" && l.TargetID == testutil.Itoa(target.ID) && l.AdminID == primary.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected admin.impersonate audit log for target user, got %+v", logs)
	}

	// The impersonated token must be usable on a user endpoint (it still passes
	// the user middleware).
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/me", nil, body.AccessToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("impersonated token must work on user endpoint, got %d: %s", rec.Code, rec.Body.String())
	}
	logs, _, err = env.Repo.ListAuditLogs(ctx, appshared.AuditLogFilter{Action: "admin.impersonated_request"}, 50, 0)
	if err != nil || len(logs) == 0 || logs[0].AdminID != primary.ID || logs[0].TargetID != testutil.Itoa(target.ID) {
		t.Fatalf("every impersonated request must be attributed to admin %d, got logs=%+v err=%v", primary.ID, logs, err)
	}

	inst := domain.VPSInstance{UserID: target.ID, Name: "imp-audit-vps", Status: domain.VPSStatusRunning, AdminStatus: domain.VPSAdminStatusNormal, SpecJSON: `{}`, AccessInfoJSON: `{"os_password":"secret"}`}
	if err := env.Repo.CreateInstance(ctx, &inst); err != nil {
		t.Fatalf("create impersonated vps: %v", err)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/vps/"+testutil.Itoa(inst.ID)+"/credentials", nil, body.AccessToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("impersonated credential reveal must succeed: %d %s", rec.Code, rec.Body.String())
	}
	logs, _, err = env.Repo.ListAuditLogs(ctx, appshared.AuditLogFilter{Action: "vps.credentials"}, 50, 0)
	if err != nil || len(logs) != 1 || logs[0].AdminID != primary.ID {
		t.Fatalf("credential reveal must be attributed to impersonating admin %d, got logs=%+v err=%v", primary.ID, logs, err)
	}

	// Changing the issuing administrator's password revokes the delegated token
	// just like it revokes that administrator's own sessions.
	if err := env.Repo.UpdateUserPassword(ctx, primary.ID, "new-hash"); err != nil {
		t.Fatalf("rotate impersonator password version: %v", err)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/me", nil, body.AccessToken)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("password-changed impersonator's token must be revoked: %d %s", rec.Code, rec.Body.String())
	}

	// Reissue a token before testing status revocation independently.
	primary, err = env.Repo.GetUserByID(ctx, primary.ID)
	if err != nil {
		t.Fatalf("reload primary: %v", err)
	}
	primaryToken = testutil.IssueJWTWithPVV(t, env.JWTSecret, primary.ID, "admin", primary.PasswordVersion, time.Hour)
	rec = testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/users/"+testutil.Itoa(target.ID)+"/impersonate", nil, primaryToken)
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &body) != nil {
		t.Fatalf("reissue impersonation after password change: %d %s", rec.Code, rec.Body.String())
	}

	// Revoking the issuing administrator must invalidate an already-issued
	// impersonation capability immediately, without waiting for its 24h expiry.
	if err := env.Repo.UpdateUserStatus(ctx, primary.ID, domain.UserStatusDisabled); err != nil {
		t.Fatalf("disable impersonator: %v", err)
	}
	rec = testutil.DoJSON(t, env.Router, http.MethodGet, "/api/v1/me", nil, body.AccessToken)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("disabled impersonator's existing token must be revoked: %d %s", rec.Code, rec.Body.String())
	}
}

func TestHandlers_AdminUserImpersonateRequiresPrimaryAdmin(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	ctx := context.Background()
	fullGroupID := ensureAdminGroup(t, env)
	_ = testutil.CreateAdmin(t, env.Repo, "imp-primary-guard", "imp-primary-guard@example.com", "pass", fullGroupID)
	limitedGroup := domain.PermissionGroup{Name: "imp-update-only", PermissionsJSON: `["user.update"]`}
	if err := env.Repo.CreatePermissionGroup(ctx, &limitedGroup); err != nil {
		t.Fatalf("create limited group: %v", err)
	}
	limited := testutil.CreateAdmin(t, env.Repo, "imp-limited", "imp-limited@example.com", "pass", limitedGroup.ID)
	target := testutil.CreateUser(t, env.Repo, "imp-denied-target", "imp-denied-target@example.com", "pass")
	token := testutil.IssueJWT(t, env.JWTSecret, limited.ID, "admin", time.Hour)
	rec := testutil.DoJSON(t, env.Router, http.MethodPost, "/admin/api/v1/users/"+testutil.Itoa(target.ID)+"/impersonate", nil, token)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("ordinary user.update permission must not allow impersonation: %d %s", rec.Code, rec.Body.String())
	}
}

// TestHandlers_AdminUserUpdate_StatusEnumGuard verifies Finding 6: PATCH
// /admin/api/v1/users/:id must reject an invalid status enum (validator-first)
// and must never silently disable an account through an empty or valid-looking
// status value — the generic route does not mutate status (only
// PATCH /users/:id/status does).
func TestHandlers_AdminUserUpdate_StatusEnumGuard(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	ctx := context.Background()

	group := domain.PermissionGroup{Name: "admin-group", PermissionsJSON: `["*"]`}
	if err := env.Repo.CreatePermissionGroup(ctx, &group); err != nil {
		t.Fatalf("create group: %v", err)
	}
	admin := testutil.CreateAdmin(t, env.Repo, "status-admin", "status-admin@example.com", "pass", group.ID)
	adminToken := testutil.IssueJWT(t, env.JWTSecret, admin.ID, "admin", time.Hour)
	user := testutil.CreateUser(t, env.Repo, "status-user", "status-user@example.com", "pass123")

	base := map[string]any{"username": user.Username, "email": user.Email}

	// An invalid enum value must be rejected at the binding layer.
	rec := testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/users/"+testutil.Itoa(user.ID), map[string]any{
		"username": user.Username, "email": user.Email, "status": "bogus",
	}, adminToken)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid status enum must be 400, got %d: %s", rec.Code, rec.Body.String())
	}

	// An empty status must not silently disable the account.
	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/users/"+testutil.Itoa(user.ID), map[string]any{
		"username": user.Username, "email": user.Email, "status": "",
	}, adminToken)
	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusOK {
		t.Fatalf("empty status must be 400 or a no-op 200, got %d", rec.Code)
	}

	// A valid status value on the generic route must not mutate the DB status.
	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/users/"+testutil.Itoa(user.ID), map[string]any{
		"username": user.Username, "email": user.Email, "status": "disabled",
	}, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("generic update with valid status must be 200, got %d: %s", rec.Code, rec.Body.String())
	}
	reloaded, err := env.Repo.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if reloaded.Status != domain.UserStatusActive {
		t.Fatalf("generic update must not change status, got %s", reloaded.Status)
	}

	// Profile-only update still works.
	rec = testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/users/"+testutil.Itoa(user.ID), base, adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("profile-only update must be 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestHandlers_AdminAPIKeyAuditActorAttribution verifies P2-22: an admin action
// performed through an admin API key must be attributed in the audit log to a
// distinct negative actor id (type+id preserved), never collapsed to admin_id=0.
func TestHandlers_AdminAPIKeyAuditActorAttribution(t *testing.T) {
	env := testutilhttp.NewTestEnv(t, false)
	ctx := context.Background()

	group := domain.PermissionGroup{Name: "user-mgmt", PermissionsJSON: `["user.update"]`}
	if err := env.Repo.CreatePermissionGroup(ctx, &group); err != nil {
		t.Fatalf("create group: %v", err)
	}
	rawKey := "ak_live_p22attribution"
	sum := sha256.Sum256([]byte(rawKey))
	key := domain.APIKey{Name: "p22", KeyHash: hex.EncodeToString(sum[:]), Status: domain.APIKeyStatusActive, ScopesJSON: `[]`, PermissionGroupID: &group.ID}
	if err := env.Repo.CreateAPIKey(ctx, &key); err != nil {
		t.Fatalf("create api key: %v", err)
	}

	target := testutil.CreateUser(t, env.Repo, "p22-target", "p22-target@example.com", "pass123")
	rec := testutil.DoJSON(t, env.Router, http.MethodPatch, "/admin/api/v1/users/"+testutil.Itoa(target.ID), map[string]any{
		"username": target.Username,
		"email":    "p22-target-new@example.com",
	}, rawKey)
	if rec.Code != http.StatusOK {
		t.Fatalf("api key user update must succeed, got %d: %s", rec.Code, rec.Body.String())
	}

	logs, _, err := env.Repo.ListAuditLogs(ctx, appshared.AuditLogFilter{}, 50, 0)
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}
	wantActor := domain.AdminAPIKeyActorID(key.ID)
	found := false
	for _, l := range logs {
		if l.Action == "user.update" && l.TargetID == testutil.Itoa(target.ID) {
			found = true
			if l.AdminID != wantActor {
				t.Fatalf("api key audit actor must be the derived negative id %d, got %d (admin_id=0 or a user id would lose attribution)", wantActor, l.AdminID)
			}
		}
	}
	if !found {
		t.Fatalf("expected user.update audit log for the API-key action")
	}
}
