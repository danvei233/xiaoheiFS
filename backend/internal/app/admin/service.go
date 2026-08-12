package admin

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/mail"
	"strings"
	"sync"
	"time"

	appports "xiaoheiplay/internal/app/ports"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"

	"golang.org/x/crypto/bcrypt"
)

type (
	OrderFilter = appshared.OrderFilter
)

type Service struct {
	users            appports.UserRepository
	orders           appports.OrderRepository
	vps              appports.VPSRepository
	keys             appports.APIKeyRepository
	settings         appports.SettingsRepository
	capabilityPolicy appports.CapabilityPolicyRepository
	capabilityCache  map[domain.CapabilityPolicyScope]capabilityPolicyCacheEntry
	capabilityMu     sync.Mutex
	audit            appports.AuditRepository
	groups           appports.PermissionGroupRepository
	userTierAssigner userTierAssigner
}

type userTierAssigner interface {
	EnsureUserHasGroup(ctx context.Context, userID int64) error
}

type capabilityPolicyCacheEntry struct {
	items     map[int64]domain.CapabilityPolicy
	expiresAt time.Time
}

const capabilityPolicyCacheTTL = time.Second

func NewService(users appports.UserRepository, orders appports.OrderRepository, vps appports.VPSRepository, keys appports.APIKeyRepository, settings appports.SettingsRepository, audit appports.AuditRepository, groups appports.PermissionGroupRepository) *Service {
	service := &Service{users: users, orders: orders, vps: vps, keys: keys, settings: settings, audit: audit, groups: groups}
	service.capabilityPolicy, _ = settings.(appports.CapabilityPolicyRepository)
	service.capabilityCache = map[domain.CapabilityPolicyScope]capabilityPolicyCacheEntry{}
	return service
}

func (s *Service) SetUserTierAssigner(assigner userTierAssigner) {
	s.userTierAssigner = assigner
}

func (s *Service) ListUsers(ctx context.Context, limit, offset int) ([]domain.User, int, error) {
	return s.users.ListUsersFiltered(ctx, "", "", limit, offset)
}

// ListUsersFiltered 管理端用户列表，支持 keyword / status 服务端过滤
// （web_api_contract F5）。
func (s *Service) ListUsersFiltered(ctx context.Context, keyword, status string, limit, offset int) ([]domain.User, int, error) {
	return s.users.ListUsersFiltered(ctx, keyword, status, limit, offset)
}

func (s *Service) GetUser(ctx context.Context, id int64) (domain.User, error) {
	return s.users.GetUserByID(ctx, id)
}

func (s *Service) CreateUser(ctx context.Context, adminID int64, user domain.User, password string) (domain.User, error) {
	username, err := trimAndValidateRequired(user.Username, maxLenUsername)
	if err != nil {
		return domain.User{}, appshared.ErrInvalidInput
	}
	email, err := trimAndValidateRequired(user.Email, maxLenEmail)
	if err != nil {
		return domain.User{}, appshared.ErrInvalidInput
	}
	if err := validateEmailFormat(email); err != nil {
		return domain.User{}, err
	}
	password, err = trimAndValidateRequired(password, maxLenPassword)
	if err != nil {
		return domain.User{}, appshared.ErrInvalidInput
	}
	if err := validatePasswordStrength(password); err != nil {
		return domain.User{}, err
	}
	qq, err := trimAndValidateOptional(user.QQ, maxLenQQ)
	if err != nil {
		return domain.User{}, appshared.ErrInvalidInput
	}
	phone, err := trimAndValidateOptional(user.Phone, maxLenPhone)
	if err != nil {
		return domain.User{}, appshared.ErrInvalidInput
	}
	bio, err := trimAndValidateOptional(user.Bio, maxLenBio)
	if err != nil {
		return domain.User{}, appshared.ErrInvalidInput
	}
	intro, err := trimAndValidateOptional(user.Intro, maxLenIntro)
	if err != nil {
		return domain.User{}, appshared.ErrInvalidInput
	}
	avatar, err := trimAndValidateOptional(user.Avatar, maxLenAvatarURL)
	if err != nil {
		return domain.User{}, appshared.ErrInvalidInput
	}
	if _, err := s.users.GetUserByUsernameOrEmail(ctx, username); err == nil {
		return domain.User{}, appshared.ErrConflict
	}
	if _, err := s.users.GetUserByUsernameOrEmail(ctx, email); err == nil {
		return domain.User{}, appshared.ErrConflict
	}
	// A phone number is a unique account identifier; reject a duplicate non-empty
	// phone here and let the DB unique index be the atomic backstop (Issue #51).
	if phone != "" {
		if _, err := s.users.GetUserByPhone(ctx, phone); err == nil {
			return domain.User{}, domain.ErrPhoneAlreadyExists
		}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return domain.User{}, err
	}
	user.Username = username
	user.Email = email
	user.QQ = qq
	user.Phone = phone
	user.Bio = bio
	user.Intro = intro
	user.Avatar = avatar
	user.PasswordHash = string(hash)
	// CreateUser is exclusively for regular users: a caller cannot escalate a
	// new account to an admin role through this method, and the status must be
	// a valid enum (service-layer guard, not just handler tags).
	user.Role = domain.UserRoleUser
	if user.Status == "" {
		user.Status = domain.UserStatusActive
	}
	if err := validateUserStatus(user.Status); err != nil {
		return domain.User{}, err
	}
	if err := s.users.CreateUser(ctx, &user); err != nil {
		return domain.User{}, err
	}
	if s.userTierAssigner != nil && user.Role == domain.UserRoleUser && user.ID > 0 {
		if err := s.userTierAssigner.EnsureUserHasGroup(ctx, user.ID); err != nil {
			return domain.User{}, err
		}
		if refreshed, err := s.users.GetUserByID(ctx, user.ID); err == nil {
			user = refreshed
		}
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "user.create", TargetType: "user", TargetID: toString(user.ID), DetailJSON: mustJSON(map[string]any{"role": user.Role})})
	}
	return user, nil
}

func (s *Service) UpdateUser(ctx context.Context, adminID int64, user domain.User) error {
	if user.ID == 0 {
		return appshared.ErrInvalidInput
	}
	username, err := trimAndValidateRequired(user.Username, maxLenUsername)
	if err != nil {
		return appshared.ErrInvalidInput
	}
	email, err := trimAndValidateRequired(user.Email, maxLenEmail)
	if err != nil {
		return appshared.ErrInvalidInput
	}
	if err := validateEmailFormat(email); err != nil {
		return err
	}
	qq, err := trimAndValidateOptional(user.QQ, maxLenQQ)
	if err != nil {
		return appshared.ErrInvalidInput
	}
	phone, err := trimAndValidateOptional(user.Phone, maxLenPhone)
	if err != nil {
		return appshared.ErrInvalidInput
	}
	bio, err := trimAndValidateOptional(user.Bio, maxLenBio)
	if err != nil {
		return appshared.ErrInvalidInput
	}
	intro, err := trimAndValidateOptional(user.Intro, maxLenIntro)
	if err != nil {
		return appshared.ErrInvalidInput
	}
	avatar, err := trimAndValidateOptional(user.Avatar, maxLenAvatarURL)
	if err != nil {
		return appshared.ErrInvalidInput
	}
	// A full-entity update must never change role or status: only the profile
	// fields below are writable, so a caller cannot escalate a user to an admin
	// or flip a status by passing a modified entity (P2-7 service guard).
	existing, err := s.users.GetUserByID(ctx, user.ID)
	if err != nil {
		return err
	}
	// An admin account cannot be modified through the generic user route even by
	// a direct service caller: only the target admin themself (self-edit of
	// profile fields) or the primary admin may touch an admin, and role/status/
	// permission-group are never writable here (P0-5 service guard). Without
	// this, a sub-admin holding `user.update` could demote or disable the primary
	// admin through UpdateUser.
	if existing.Role == domain.UserRoleAdmin {
		if user.Role != existing.Role || user.Status != existing.Status ||
			(user.PermissionGroupID == nil) != (existing.PermissionGroupID == nil) ||
			(user.PermissionGroupID != nil && existing.PermissionGroupID != nil && *user.PermissionGroupID != *existing.PermissionGroupID) {
			return domain.ErrAdminUserNotEditable
		}
		if adminID != existing.ID {
			primary, err := s.isPrimaryAdmin(ctx, adminID)
			if err != nil {
				return err
			}
			if !primary {
				return domain.ErrAdminUserNotEditable
			}
		}
	}
	// A phone change must not collide with another account's phone; the DB unique
	// index is the atomic backstop for concurrent writers (Issue #51 phone binding).
	if phone != "" && strings.TrimSpace(existing.Phone) != phone {
		if owner, err := s.users.GetUserByPhone(ctx, phone); err == nil && owner.ID != user.ID {
			return domain.ErrPhoneAlreadyExists
		}
	}
	existing.Username = username
	existing.Email = email
	existing.QQ = qq
	existing.Phone = phone
	existing.Bio = bio
	existing.Intro = intro
	existing.Avatar = avatar
	if err := s.users.UpdateUser(ctx, existing); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "user.update", TargetType: "user", TargetID: toString(user.ID), DetailJSON: mustJSON(map[string]any{"role": user.Role, "status": user.Status})})
	}
	return nil
}

func (s *Service) ResetUserPassword(ctx context.Context, adminID int64, userID int64, password string) error {
	password, err := trimAndValidateRequired(password, maxLenPassword)
	if userID == 0 || err != nil {
		return appshared.ErrInvalidInput
	}
	if err := validatePasswordStrength(password); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.users.UpdateUserPassword(ctx, userID, string(hash)); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "user.reset_password", TargetType: "user", TargetID: toString(userID), DetailJSON: "{}"})
	}
	return nil
}

func (s *Service) ListOrders(ctx context.Context, filter OrderFilter, limit, offset int) ([]domain.Order, int, error) {
	return s.orders.ListOrders(ctx, filter, limit, offset)
}

func (s *Service) DeleteOrder(ctx context.Context, adminID int64, orderID int64) error {
	if orderID == 0 {
		return appshared.ErrInvalidInput
	}
	order, err := s.orders.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}
	if order.Status == domain.OrderStatusApproved {
		return appshared.ErrConflict
	}
	if err := s.orders.DeleteOrder(ctx, orderID); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{
			AdminID:    adminID,
			Action:     "order.delete",
			TargetType: "order",
			TargetID:   toString(orderID),
			DetailJSON: mustJSON(map[string]any{"order_no": order.OrderNo, "status": order.Status}),
		})
	}
	return nil
}

func (s *Service) ListAuditLogs(ctx context.Context, filter appshared.AuditLogFilter, limit, offset int) ([]domain.AdminAuditLog, int, error) {
	if s.audit == nil {
		return nil, 0, nil
	}
	return s.audit.ListAuditLogs(ctx, filter, limit, offset)
}

func (s *Service) Audit(ctx context.Context, adminID int64, action, targetType, targetID string, detail any) {
	if s.audit == nil {
		return
	}
	_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: action, TargetType: targetType, TargetID: targetID, DetailJSON: mustJSON(detail)})
}

// AuditRequired records a security-sensitive event and reports persistence
// failure to the caller. Handlers that disclose credentials or issue an
// impersonation token must fail closed when this write cannot be guaranteed.
func (s *Service) AuditRequired(ctx context.Context, adminID int64, action, targetType, targetID string, detail any) error {
	if s.audit == nil {
		return domain.ErrAuditWriteFailed
	}
	if err := s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: action, TargetType: targetType, TargetID: targetID, DetailJSON: mustJSON(detail)}); err != nil {
		return fmt.Errorf("%w: %w", domain.ErrAuditWriteFailed, err)
	}
	return nil
}

func (s *Service) UpdateUserStatus(ctx context.Context, adminID int64, userID int64, status domain.UserStatus) error {
	if err := validateUserStatus(status); err != nil {
		return err
	}
	if err := s.users.UpdateUserStatus(ctx, userID, status); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "user.status", TargetType: "user", TargetID: toString(userID), DetailJSON: mustJSON(map[string]any{"status": status})})
	}
	return nil
}

func (s *Service) ListInstances(ctx context.Context, limit, offset int) ([]domain.VPSInstance, int, error) {
	return s.vps.ListInstancesFiltered(ctx, appports.VPSInstanceFilter{}, limit, offset)
}

// ListInstancesFiltered 管理端 VPS 列表，支持服务端 admin_status / keyword 过滤
// （web_refresh_perf F4）。
func (s *Service) ListInstancesFiltered(ctx context.Context, filter appports.VPSInstanceFilter, limit, offset int) ([]domain.VPSInstance, int, error) {
	return s.vps.ListInstancesFiltered(ctx, filter, limit, offset)
}

func (s *Service) CreateAPIKey(ctx context.Context, adminID int64, name string, permissionGroupID *int64, scopes []string) (string, domain.APIKey, error) {
	raw := "ak_live_" + randomToken(24)
	hash := hashKey(raw)
	key := domain.APIKey{Name: name, KeyHash: hash, Status: domain.APIKeyStatusActive, ScopesJSON: mustJSON(scopes), PermissionGroupID: permissionGroupID}
	if err := s.keys.CreateAPIKey(ctx, &key); err != nil {
		return "", domain.APIKey{}, err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "api_key.create", TargetType: "api_key", TargetID: toString(key.ID), DetailJSON: mustJSON(map[string]any{"name": name})})
	}
	return raw, key, nil
}

func (s *Service) ListAPIKeys(ctx context.Context, limit, offset int) ([]domain.APIKey, int, error) {
	return s.keys.ListAPIKeys(ctx, limit, offset)
}

func (s *Service) UpdateAPIKeyStatus(ctx context.Context, adminID int64, id int64, status domain.APIKeyStatus) error {
	if err := s.keys.UpdateAPIKeyStatus(ctx, id, status); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "api_key.status", TargetType: "api_key", TargetID: toString(id), DetailJSON: mustJSON(map[string]any{"status": status})})
	}
	return nil
}

func (s *Service) UpdateSetting(ctx context.Context, adminID int64, key string, valueJSON string) error {
	if shouldSanitizePlainSetting(key) {
		valueJSON = strings.TrimSpace(appshared.SanitizePlainText(valueJSON))
	}
	if err := s.settings.UpsertSetting(ctx, domain.Setting{Key: key, ValueJSON: valueJSON, UpdatedAt: time.Now()}); err != nil {
		return err
	}
	s.auditSettingUpdate(ctx, adminID, key, valueJSON)
	return nil
}

// UpdateSettings persists a batch of settings atomically (all or nothing) and
// then writes one audit entry per key. The settings are sanitized before the
// single transactional write, so a failure on the k-th key rolls back the
// earlier keys instead of leaving the configuration half-updated.
func (s *Service) UpdateSettings(ctx context.Context, adminID int64, items []domain.Setting) error {
	for i := range items {
		if shouldSanitizePlainSetting(items[i].Key) {
			items[i].ValueJSON = strings.TrimSpace(appshared.SanitizePlainText(items[i].ValueJSON))
		}
		items[i].UpdatedAt = time.Now()
	}
	if err := s.settings.UpsertSettings(ctx, items); err != nil {
		return err
	}
	if s.audit != nil {
		for _, item := range items {
			s.auditSettingUpdate(ctx, adminID, item.Key, item.ValueJSON)
		}
	}
	return nil
}

func (s *Service) auditSettingUpdate(ctx context.Context, adminID int64, key string, valueJSON string) {
	if s.audit == nil {
		return
	}
	detail := valueJSON
	if isSensitiveSettingKey(key) {
		detail = `"***"`
	}
	_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "settings.update", TargetType: "setting", TargetID: key, DetailJSON: detail})
}

// isSensitiveSettingKey reports whether a settings key holds a credential that
// must never be written to the audit log.
func isSensitiveSettingKey(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	switch key {
	case "smtp_pass", "robot_webhook_secret", "robot_webhooks", "payment_plugin_upload_password",
		"fcm_server_key", "fcm_service_account_json", "automation_api_key",
		"auth_geetest_captcha_key":
		return true
	}
	for _, suffix := range []string{"_secret", "_token", "_password", "_pass", "_api_key", "apikey", "appkey"} {
		if strings.HasSuffix(key, suffix) {
			return true
		}
	}
	return false
}

func shouldSanitizePlainSetting(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "site_name", "site_title", "site_subtitle", "site_description", "site_keywords",
		"company_name", "contact_phone", "contact_email", "contact_qq",
		"icp_number", "psbe_number", "maintenance_message", "copyright_text":
		return true
	default:
		return false
	}
}

func (s *Service) ListSettings(ctx context.Context) ([]domain.Setting, error) {
	return s.settings.ListSettings(ctx)
}

// UpdateSettingAtomic applies a read-modify-write to a settings key with a DB
// row lock, so concurrent updates of a shared JSON document (e.g. capability
// policy) do not last-write-wins across instances (P2-8).
func (s *Service) UpdateSettingAtomic(ctx context.Context, key string, mutate func(current string) (string, error)) error {
	return s.settings.UpdateSettingAtomic(ctx, key, mutate)
}

func (s *Service) GetSetting(ctx context.Context, key string) (domain.Setting, error) {
	return s.settings.GetSetting(ctx, key)
}

func (s *Service) ListCapabilityPolicies(ctx context.Context, scope domain.CapabilityPolicyScope) (map[int64]domain.CapabilityPolicy, error) {
	if s.capabilityPolicy == nil {
		return nil, domain.ErrNotSupported
	}
	now := time.Now()
	s.capabilityMu.Lock()
	entry, ok := s.capabilityCache[scope]
	if ok && now.Before(entry.expiresAt) {
		items := cloneCapabilityPolicies(entry.items)
		s.capabilityMu.Unlock()
		return items, nil
	}
	s.capabilityMu.Unlock()
	// Only display/admin reads are cached. Execution paths use the repository
	// port directly so a cross-instance policy disable takes effect immediately.
	items, err := s.capabilityPolicy.ListCapabilityPolicies(ctx, scope)
	if err != nil {
		return nil, err
	}
	s.capabilityMu.Lock()
	s.capabilityCache[scope] = capabilityPolicyCacheEntry{items: cloneCapabilityPolicies(items), expiresAt: now.Add(capabilityPolicyCacheTTL)}
	s.capabilityMu.Unlock()
	return items, nil
}

func (s *Service) GetCapabilityPolicy(ctx context.Context, scope domain.CapabilityPolicyScope, itemID int64) (domain.CapabilityPolicy, error) {
	if s.capabilityPolicy == nil {
		return domain.CapabilityPolicy{}, domain.ErrNotSupported
	}
	items, err := s.ListCapabilityPolicies(ctx, scope)
	if err != nil {
		return domain.CapabilityPolicy{}, err
	}
	item, ok := items[itemID]
	if !ok {
		return domain.CapabilityPolicy{}, appshared.ErrNotFound
	}
	return item, nil
}

func (s *Service) UpsertCapabilityPolicy(ctx context.Context, scope domain.CapabilityPolicyScope, itemID int64, policy domain.CapabilityPolicy) error {
	if s.capabilityPolicy == nil {
		return domain.ErrNotSupported
	}
	if err := s.capabilityPolicy.UpsertCapabilityPolicy(ctx, scope, itemID, policy); err != nil {
		return err
	}
	s.capabilityMu.Lock()
	delete(s.capabilityCache, scope)
	s.capabilityMu.Unlock()
	return nil
}

func cloneCapabilityPolicies(items map[int64]domain.CapabilityPolicy) map[int64]domain.CapabilityPolicy {
	out := make(map[int64]domain.CapabilityPolicy, len(items))
	for id, item := range items {
		cloned := domain.CapabilityPolicy{}
		if item.ResizeEnabled != nil {
			value := *item.ResizeEnabled
			cloned.ResizeEnabled = &value
		}
		if item.RefundEnabled != nil {
			value := *item.RefundEnabled
			cloned.RefundEnabled = &value
		}
		out[id] = cloned
	}
	return out
}

func (s *Service) UpsertEmailTemplate(ctx context.Context, adminID int64, tmpl *domain.EmailTemplate) error {
	if err := s.settings.UpsertEmailTemplate(ctx, tmpl); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "email_template.upsert", TargetType: "email_template", TargetID: toString(tmpl.ID), DetailJSON: mustJSON(map[string]any{"name": tmpl.Name})})
	}
	return nil
}

func (s *Service) ListEmailTemplates(ctx context.Context) ([]domain.EmailTemplate, error) {
	return s.settings.ListEmailTemplates(ctx)
}

func (s *Service) DeleteEmailTemplate(ctx context.Context, adminID int64, id int64) error {
	if id == 0 {
		return appshared.ErrInvalidInput
	}
	if err := s.settings.DeleteEmailTemplate(ctx, id); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "email_template.delete", TargetType: "email_template", TargetID: toString(id), DetailJSON: "{}"})
	}
	return nil
}

func (s *Service) ListPermissionGroups(ctx context.Context) ([]domain.PermissionGroup, error) {
	return s.groups.ListPermissionGroups(ctx)
}

func (s *Service) GetPermissionGroup(ctx context.Context, id int64) (domain.PermissionGroup, error) {
	return s.groups.GetPermissionGroup(ctx, id)
}

func (s *Service) CreatePermissionGroup(ctx context.Context, adminID int64, group *domain.PermissionGroup) error {
	if err := s.requirePrimaryAdmin(ctx, adminID); err != nil {
		return err
	}
	if err := s.groups.CreatePermissionGroup(ctx, group); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "permission_group.create", TargetType: "permission_group", TargetID: toString(group.ID), DetailJSON: mustJSON(map[string]any{"name": group.Name})})
	}
	return nil
}

func (s *Service) UpdatePermissionGroup(ctx context.Context, adminID int64, group domain.PermissionGroup) error {
	if err := s.requirePrimaryAdmin(ctx, adminID); err != nil {
		return err
	}
	if err := s.groups.UpdatePermissionGroup(ctx, group); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "permission_group.update", TargetType: "permission_group", TargetID: toString(group.ID), DetailJSON: mustJSON(map[string]any{"name": group.Name})})
	}
	return nil
}

func (s *Service) DeletePermissionGroup(ctx context.Context, adminID int64, id int64) error {
	if err := s.requirePrimaryAdmin(ctx, adminID); err != nil {
		return err
	}
	// The reference re-check and the delete happen in one repository transaction
	// that locks the group row, closing the count-then-delete TOCTOU: a
	// concurrent assignment either commits before the delete (and is counted) or
	// fails after it, so no dangling references are ever created (P2-36).
	if err := s.groups.DeletePermissionGroupSafe(ctx, id); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "permission_group.delete", TargetType: "permission_group", TargetID: toString(id), DetailJSON: "{}"})
	}
	return nil
}

func (s *Service) ListAdmins(ctx context.Context, status string, limit, offset int) ([]domain.User, int, error) {
	if status == "all" {
		status = ""
	}
	return s.users.ListUsersByRoleStatus(ctx, string(domain.UserRoleAdmin), status, limit, offset)
}

// isPrimaryAdmin reports whether adminID is the first-created (primary) admin.
// Only the primary admin may create, modify, disable or delete other admins;
// a sub-admin granted admin.create/update must not be able to take over or
// demote the primary admin or manage sibling admins (Issue #51 P0-5).
func (s *Service) isPrimaryAdmin(ctx context.Context, adminID int64) (bool, error) {
	if adminID <= 0 {
		return false, nil
	}
	minID, err := s.users.GetMinUserIDByRole(ctx, string(domain.UserRoleAdmin))
	if err != nil {
		return false, err
	}
	return minID == adminID, nil
}

// requirePrimaryAdmin returns ErrPrimaryAdminRequired unless the actor is the
// primary admin.
func (s *Service) requirePrimaryAdmin(ctx context.Context, adminID int64) error {
	primary, err := s.isPrimaryAdmin(ctx, adminID)
	if err != nil {
		return err
	}
	if !primary {
		return domain.ErrPrimaryAdminRequired
	}
	return nil
}

func (s *Service) CreateAdmin(ctx context.Context, adminID int64, username, email, qq, password string, permissionGroupID *int64) (domain.User, error) {
	if err := s.requirePrimaryAdmin(ctx, adminID); err != nil {
		return domain.User{}, err
	}
	username, err := trimAndValidateRequired(username, maxLenUsername)
	if err != nil {
		return domain.User{}, appshared.ErrInvalidInput
	}
	email, err = trimAndValidateRequired(email, maxLenEmail)
	if err != nil {
		return domain.User{}, appshared.ErrInvalidInput
	}
	if err := validateEmailFormat(email); err != nil {
		return domain.User{}, err
	}
	qq, err = trimAndValidateOptional(qq, maxLenQQ)
	if err != nil {
		return domain.User{}, appshared.ErrInvalidInput
	}
	password, err = trimAndValidateRequired(password, maxLenPassword)
	if err != nil {
		return domain.User{}, appshared.ErrInvalidInput
	}
	if err := validatePasswordStrength(password); err != nil {
		return domain.User{}, err
	}
	if _, err := s.users.GetUserByUsernameOrEmail(ctx, username); err == nil {
		return domain.User{}, appshared.ErrConflict
	}
	if _, err := s.users.GetUserByUsernameOrEmail(ctx, email); err == nil {
		return domain.User{}, appshared.ErrConflict
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return domain.User{}, err
	}
	user := domain.User{Username: username, Email: email, QQ: qq, PermissionGroupID: permissionGroupID, PasswordHash: string(hash), Role: domain.UserRoleAdmin, Status: domain.UserStatusActive}
	if err := s.users.CreateUser(ctx, &user); err != nil {
		return domain.User{}, err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "admin.create", TargetType: "admin", TargetID: toString(user.ID), DetailJSON: mustJSON(map[string]any{"username": username, "email": email})})
	}
	return user, nil
}

func (s *Service) UpdateAdmin(ctx context.Context, adminID int64, userID int64, username, email, qq string, permissionGroupID *int64) error {
	if userID == 0 {
		return appshared.ErrInvalidInput
	}
	// Only the primary admin may modify another admin. Self-edits are allowed
	// so an admin can correct their own contact details.
	if adminID != userID {
		if err := s.requirePrimaryAdmin(ctx, adminID); err != nil {
			return err
		}
	}
	var err error
	username, err = trimAndValidateRequired(username, maxLenUsername)
	if err != nil {
		return appshared.ErrInvalidInput
	}
	email, err = trimAndValidateRequired(email, maxLenEmail)
	if err != nil {
		return appshared.ErrInvalidInput
	}
	if err := validateEmailFormat(email); err != nil {
		return err
	}
	qq, err = trimAndValidateOptional(qq, maxLenQQ)
	if err != nil {
		return appshared.ErrInvalidInput
	}
	existing, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if existing.Role != domain.UserRoleAdmin {
		return appshared.ErrInvalidInput
	}
	// A non-primary admin's self-edit is restricted to non-privileged profile
	// fields: the server never trusts a permissionGroupID (or any role/status)
	// supplied for self-edit, so a sub-admin cannot escalate their own group
	// (Issue #51 P0-5 / P2-7).
	if adminID == userID {
		primary, err := s.isPrimaryAdmin(ctx, adminID)
		if err != nil {
			return err
		}
		if !primary {
			permissionGroupID = existing.PermissionGroupID
		}
	}
	if existing.Username != username {
		if _, err := s.users.GetUserByUsernameOrEmail(ctx, username); err == nil {
			return appshared.ErrConflict
		}
	}
	if existing.Email != email {
		if _, err := s.users.GetUserByUsernameOrEmail(ctx, email); err == nil {
			return appshared.ErrConflict
		}
	}
	existing.Username = username
	existing.Email = email
	existing.QQ = qq
	existing.PermissionGroupID = permissionGroupID
	if err := s.users.UpdateUser(ctx, existing); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "admin.update", TargetType: "admin", TargetID: toString(userID), DetailJSON: mustJSON(map[string]any{"username": username, "email": email})})
	}
	return nil
}

func (s *Service) UpdateAdminStatus(ctx context.Context, adminID int64, userID int64, status domain.UserStatus) error {
	if userID == 0 {
		return appshared.ErrInvalidInput
	}
	if err := validateUserStatus(status); err != nil {
		return err
	}
	if err := s.requirePrimaryAdmin(ctx, adminID); err != nil {
		return err
	}
	// The primary admin cannot disable or block their own account — that would
	// strand the panel without any administrator able to manage it.
	if adminID == userID && status != domain.UserStatusActive {
		return domain.ErrCannotUpdateSelfStatus
	}
	existing, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if existing.Role != domain.UserRoleAdmin {
		return appshared.ErrNotFound
	}
	// The last-active-admin constraint is enforced atomically at the database:
	// concurrent disable attempts serialize on the active-admin rows so the
	// system can never end up with zero active admins.
	if err := s.users.UpdateAdminStatusSafe(ctx, userID, status); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "admin.status", TargetType: "admin", TargetID: toString(userID), DetailJSON: mustJSON(map[string]any{"status": status})})
	}
	return nil
}

func (s *Service) DeleteAdmin(ctx context.Context, adminID int64, userID int64) error {
	if userID == 0 {
		return appshared.ErrInvalidInput
	}
	if err := s.requirePrimaryAdmin(ctx, adminID); err != nil {
		return err
	}
	existing, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if existing.Role != domain.UserRoleAdmin {
		return appshared.ErrInvalidInput
	}
	if existing.ID == adminID {
		return domain.ErrCannotDeleteSelf
	}
	// Delete is a disable; the last-active-admin constraint is enforced
	// atomically so a concurrent delete cannot zero out all active admins.
	if err := s.users.UpdateAdminStatusSafe(ctx, userID, domain.UserStatusDisabled); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "admin.delete", TargetType: "admin", TargetID: toString(userID), DetailJSON: "{}"})
	}
	return nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID int64, email, qq string) error {
	if userID == 0 {
		return appshared.ErrInvalidInput
	}
	if email != "" {
		var err error
		email, err = trimAndValidateRequired(email, maxLenEmail)
		if err != nil {
			return appshared.ErrInvalidInput
		}
	}
	if qq != "" {
		var err error
		qq, err = trimAndValidateOptional(qq, maxLenQQ)
		if err != nil {
			return appshared.ErrInvalidInput
		}
	}
	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if email != "" && email != user.Email {
		if existing, err := s.users.GetUserByUsernameOrEmail(ctx, email); err == nil && existing.ID != user.ID {
			return appshared.ErrConflict
		}
		user.Email = email
	}
	if qq != "" {
		user.QQ = qq
	}
	return s.users.UpdateUser(ctx, user)
}

func (s *Service) ChangePassword(ctx context.Context, userID int64, oldPassword, newPassword string) error {
	oldPassword, oldErr := trimAndValidateRequired(oldPassword, maxLenPassword)
	newPassword, newErr := trimAndValidateRequired(newPassword, maxLenPassword)
	if userID == 0 || oldErr != nil || newErr != nil {
		return appshared.ErrInvalidInput
	}
	if err := validatePasswordStrength(newPassword); err != nil {
		return err
	}
	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return domain.ErrInvalidOldPassword
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.users.UpdateUserPassword(ctx, userID, string(hash))
}

// minAdminPasswordLen is the service-level password policy floor. Handlers may
// additionally enforce settings-driven rules, but the service must never accept
// a weaker password than this.
const minAdminPasswordLen = 6

// validatePasswordStrength enforces the service-level password policy so admin
// account creation and resets cannot accept trivial passwords even when a
// handler is bypassed.
func validatePasswordStrength(v string) error {
	if len(strings.TrimSpace(v)) < minAdminPasswordLen {
		return appshared.ErrInvalidInput
	}
	return nil
}

// validateUserStatus enforces the service-level status enum so callers cannot
// persist an arbitrary status value (Issue #51 P2-7 / P3-15 service bypass).
func validateUserStatus(s domain.UserStatus) error {
	switch s {
	case domain.UserStatusActive, domain.UserStatusDisabled, domain.UserStatusBlocked:
		return nil
	default:
		return domain.ErrInvalidStatus
	}
}

// validateEmailFormat enforces a basic email shape at the service boundary.
func validateEmailFormat(v string) error {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" || len(trimmed) > 254 {
		return appshared.ErrInvalidInput
	}
	addr, err := mail.ParseAddress(trimmed)
	if err != nil || addr.Address != trimmed {
		return appshared.ErrInvalidInput
	}
	return nil
}

func hashKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func toString(id int64) string {
	return fmt.Sprintf("%d", id)
}

func randomToken(n int) string {
	letters := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	buf := make([]byte, n)
	_, _ = rand.Read(buf)
	for i := range buf {
		buf[i] = letters[int(buf[i])%len(letters)]
	}
	return string(buf)
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
