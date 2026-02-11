package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"xiaoheiplay/internal/domain"

	"golang.org/x/crypto/bcrypt"
)

type AdminService struct {
	users    UserRepository
	orders   OrderRepository
	vps      VPSRepository
	keys     APIKeyRepository
	settings SettingsRepository
	audit    AuditRepository
	groups   PermissionGroupRepository
}

func NewAdminService(users UserRepository, orders OrderRepository, vps VPSRepository, keys APIKeyRepository, settings SettingsRepository, audit AuditRepository, groups PermissionGroupRepository) *AdminService {
	return &AdminService{users: users, orders: orders, vps: vps, keys: keys, settings: settings, audit: audit, groups: groups}
}

func (s *AdminService) ListUsers(ctx context.Context, limit, offset int) ([]domain.User, int, error) {
	return s.users.ListUsersByRoleStatus(ctx, string(domain.UserRoleUser), "", limit, offset)
}

func (s *AdminService) GetUser(ctx context.Context, id int64) (domain.User, error) {
	return s.users.GetUserByID(ctx, id)
}

func (s *AdminService) CreateUser(ctx context.Context, adminID int64, user domain.User, password string) (domain.User, error) {
	if user.Username == "" || user.Email == "" || password == "" {
		return domain.User{}, ErrInvalidInput
	}
	if _, err := s.users.GetUserByUsernameOrEmail(ctx, user.Username); err == nil {
		return domain.User{}, ErrConflict
	}
	if _, err := s.users.GetUserByUsernameOrEmail(ctx, user.Email); err == nil {
		return domain.User{}, ErrConflict
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return domain.User{}, err
	}
	user.PasswordHash = string(hash)
	if user.Role == "" {
		user.Role = domain.UserRoleUser
	}
	if user.Status == "" {
		user.Status = domain.UserStatusActive
	}
	if err := s.users.CreateUser(ctx, &user); err != nil {
		return domain.User{}, err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "user.create", TargetType: "user", TargetID: toString(user.ID), DetailJSON: mustJSON(map[string]any{"role": user.Role})})
	}
	return user, nil
}

func (s *AdminService) UpdateUser(ctx context.Context, adminID int64, user domain.User) error {
	if user.ID == 0 {
		return ErrInvalidInput
	}
	if err := s.users.UpdateUser(ctx, user); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "user.update", TargetType: "user", TargetID: toString(user.ID), DetailJSON: mustJSON(map[string]any{"role": user.Role, "status": user.Status})})
	}
	return nil
}

func (s *AdminService) ResetUserPassword(ctx context.Context, adminID int64, userID int64, password string) error {
	if userID == 0 || password == "" {
		return ErrInvalidInput
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

func (s *AdminService) ListOrders(ctx context.Context, filter OrderFilter, limit, offset int) ([]domain.Order, int, error) {
	return s.orders.ListOrders(ctx, filter, limit, offset)
}

func (s *AdminService) DeleteOrder(ctx context.Context, adminID int64, orderID int64) error {
	if orderID == 0 {
		return ErrInvalidInput
	}
	order, err := s.orders.GetOrder(ctx, orderID)
	if err != nil {
		return err
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

func (s *AdminService) ListAuditLogs(ctx context.Context, limit, offset int) ([]domain.AdminAuditLog, int, error) {
	if s.audit == nil {
		return nil, 0, nil
	}
	return s.audit.ListAuditLogs(ctx, limit, offset)
}

func (s *AdminService) Audit(ctx context.Context, adminID int64, action, targetType, targetID string, detail any) {
	if s.audit == nil {
		return
	}
	_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{
		AdminID:    adminID,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		DetailJSON: mustJSON(detail),
	})
}

func (s *AdminService) UpdateUserStatus(ctx context.Context, adminID int64, userID int64, status domain.UserStatus) error {
	if err := s.users.UpdateUserStatus(ctx, userID, status); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "user.status", TargetType: "user", TargetID: toString(userID), DetailJSON: mustJSON(map[string]any{"status": status})})
	}
	return nil
}

func (s *AdminService) ListInstances(ctx context.Context, limit, offset int) ([]domain.VPSInstance, int, error) {
	return s.vps.ListInstances(ctx, limit, offset)
}

func (s *AdminService) CreateAPIKey(ctx context.Context, adminID int64, name string, permissionGroupID *int64, scopes []string) (string, domain.APIKey, error) {
	raw := "ak_live_" + randomToken(24)
	hash := hashKey(raw)
	key := domain.APIKey{
		Name:              name,
		KeyHash:           hash,
		Status:            domain.APIKeyStatusActive,
		ScopesJSON:        mustJSON(scopes),
		PermissionGroupID: permissionGroupID,
	}
	if err := s.keys.CreateAPIKey(ctx, &key); err != nil {
		return "", domain.APIKey{}, err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "api_key.create", TargetType: "api_key", TargetID: toString(key.ID), DetailJSON: mustJSON(map[string]any{"name": name})})
	}
	return raw, key, nil
}

func (s *AdminService) ListAPIKeys(ctx context.Context, limit, offset int) ([]domain.APIKey, int, error) {
	return s.keys.ListAPIKeys(ctx, limit, offset)
}

func (s *AdminService) UpdateAPIKeyStatus(ctx context.Context, adminID int64, id int64, status domain.APIKeyStatus) error {
	if err := s.keys.UpdateAPIKeyStatus(ctx, id, status); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "api_key.status", TargetType: "api_key", TargetID: toString(id), DetailJSON: mustJSON(map[string]any{"status": status})})
	}
	return nil
}

func (s *AdminService) UpdateSetting(ctx context.Context, adminID int64, key string, valueJSON string) error {
	if err := s.settings.UpsertSetting(ctx, domain.Setting{Key: key, ValueJSON: valueJSON, UpdatedAt: time.Now()}); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "settings.update", TargetType: "setting", TargetID: key, DetailJSON: valueJSON})
	}
	return nil
}

func (s *AdminService) ListSettings(ctx context.Context) ([]domain.Setting, error) {
	return s.settings.ListSettings(ctx)
}

func (s *AdminService) UpsertEmailTemplate(ctx context.Context, adminID int64, tmpl *domain.EmailTemplate) error {
	if err := s.settings.UpsertEmailTemplate(ctx, tmpl); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "email_template.upsert", TargetType: "email_template", TargetID: toString(tmpl.ID), DetailJSON: mustJSON(map[string]any{"name": tmpl.Name})})
	}
	return nil
}

func (s *AdminService) ListEmailTemplates(ctx context.Context) ([]domain.EmailTemplate, error) {
	return s.settings.ListEmailTemplates(ctx)
}

func (s *AdminService) ListPermissionGroups(ctx context.Context) ([]domain.PermissionGroup, error) {
	return s.groups.ListPermissionGroups(ctx)
}

func (s *AdminService) GetPermissionGroup(ctx context.Context, id int64) (domain.PermissionGroup, error) {
	return s.groups.GetPermissionGroup(ctx, id)
}

func (s *AdminService) CreatePermissionGroup(ctx context.Context, adminID int64, group *domain.PermissionGroup) error {
	if err := s.groups.CreatePermissionGroup(ctx, group); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "permission_group.create", TargetType: "permission_group", TargetID: toString(group.ID), DetailJSON: mustJSON(map[string]any{"name": group.Name})})
	}
	return nil
}

func (s *AdminService) UpdatePermissionGroup(ctx context.Context, adminID int64, group domain.PermissionGroup) error {
	if err := s.groups.UpdatePermissionGroup(ctx, group); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "permission_group.update", TargetType: "permission_group", TargetID: toString(group.ID), DetailJSON: mustJSON(map[string]any{"name": group.Name})})
	}
	return nil
}

func (s *AdminService) DeletePermissionGroup(ctx context.Context, adminID int64, id int64) error {
	if err := s.groups.DeletePermissionGroup(ctx, id); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "permission_group.delete", TargetType: "permission_group", TargetID: toString(id), DetailJSON: "{}"})
	}
	return nil
}

func (s *AdminService) ListAdmins(ctx context.Context, status string, limit, offset int) ([]domain.User, int, error) {
	if status == "all" {
		status = ""
	}
	return s.users.ListUsersByRoleStatus(ctx, string(domain.UserRoleAdmin), status, limit, offset)
}

func (s *AdminService) CreateAdmin(ctx context.Context, adminID int64, username, email, qq, password string, permissionGroupID *int64) (domain.User, error) {
	if username == "" || email == "" || password == "" {
		return domain.User{}, ErrInvalidInput
	}
	if _, err := s.users.GetUserByUsernameOrEmail(ctx, username); err == nil {
		return domain.User{}, ErrConflict
	}
	if _, err := s.users.GetUserByUsernameOrEmail(ctx, email); err == nil {
		return domain.User{}, ErrConflict
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return domain.User{}, err
	}
	user := domain.User{
		Username:          username,
		Email:             email,
		QQ:                qq,
		PermissionGroupID: permissionGroupID,
		PasswordHash:      string(hash),
		Role:              domain.UserRoleAdmin,
		Status:            domain.UserStatusActive,
	}
	if err := s.users.CreateUser(ctx, &user); err != nil {
		return domain.User{}, err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "admin.create", TargetType: "admin", TargetID: toString(user.ID), DetailJSON: mustJSON(map[string]any{"username": username, "email": email})})
	}
	return user, nil
}

func (s *AdminService) UpdateAdmin(ctx context.Context, adminID int64, userID int64, username, email, qq string, permissionGroupID *int64) error {
	if userID == 0 || username == "" || email == "" {
		return ErrInvalidInput
	}
	existing, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if existing.Role != domain.UserRoleAdmin {
		return ErrInvalidInput
	}
	if existing.Username != username {
		if _, err := s.users.GetUserByUsernameOrEmail(ctx, username); err == nil {
			return ErrConflict
		}
	}
	if existing.Email != email {
		if _, err := s.users.GetUserByUsernameOrEmail(ctx, email); err == nil {
			return ErrConflict
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

func (s *AdminService) UpdateAdminStatus(ctx context.Context, adminID int64, userID int64, status domain.UserStatus) error {
	if userID == 0 {
		return ErrInvalidInput
	}
	existing, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if existing.Role != domain.UserRoleAdmin {
		return ErrNotFound
	}
	if err := s.users.UpdateUserStatus(ctx, userID, status); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "admin.status", TargetType: "admin", TargetID: toString(userID), DetailJSON: mustJSON(map[string]any{"status": status})})
	}
	return nil
}

func (s *AdminService) DeleteAdmin(ctx context.Context, adminID int64, userID int64) error {
	if userID == 0 {
		return ErrInvalidInput
	}
	existing, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if existing.Role != domain.UserRoleAdmin {
		return ErrInvalidInput
	}
	if existing.ID == adminID {
		return errors.New("cannot delete yourself")
	}
	if err := s.users.UpdateUserStatus(ctx, userID, domain.UserStatusDisabled); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "admin.delete", TargetType: "admin", TargetID: toString(userID), DetailJSON: "{}"})
	}
	return nil
}

func (s *AdminService) UpdateProfile(ctx context.Context, userID int64, email, qq string) error {
	if userID == 0 {
		return ErrInvalidInput
	}
	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if email != "" && email != user.Email {
		if existing, err := s.users.GetUserByUsernameOrEmail(ctx, email); err == nil && existing.ID != user.ID {
			return ErrConflict
		}
		user.Email = email
	}
	if qq != "" {
		user.QQ = qq
	}
	if err := s.users.UpdateUser(ctx, user); err != nil {
		return err
	}
	return nil
}

func (s *AdminService) ChangePassword(ctx context.Context, userID int64, oldPassword, newPassword string) error {
	if userID == 0 || oldPassword == "" || newPassword == "" {
		return ErrInvalidInput
	}
	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return errors.New("invalid old password")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.users.UpdateUserPassword(ctx, userID, string(hash)); err != nil {
		return err
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
