package admin

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
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
	users    appports.UserRepository
	orders   appports.OrderRepository
	vps      appports.VPSRepository
	keys     appports.APIKeyRepository
	settings appports.SettingsRepository
	audit    appports.AuditRepository
	groups   appports.PermissionGroupRepository
}

func NewService(users appports.UserRepository, orders appports.OrderRepository, vps appports.VPSRepository, keys appports.APIKeyRepository, settings appports.SettingsRepository, audit appports.AuditRepository, groups appports.PermissionGroupRepository) *Service {
	return &Service{users: users, orders: orders, vps: vps, keys: keys, settings: settings, audit: audit, groups: groups}
}

func (s *Service) ListUsers(ctx context.Context, limit, offset int) ([]domain.User, int, error) {
	return s.users.ListUsersByRoleStatus(ctx, string(domain.UserRoleUser), "", limit, offset)
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
	password, err = trimAndValidateRequired(password, maxLenPassword)
	if err != nil {
		return domain.User{}, appshared.ErrInvalidInput
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
	user.Username = username
	user.Email = email
	user.QQ = qq
	user.Phone = phone
	user.Bio = bio
	user.Intro = intro
	user.Avatar = avatar
	if err := s.users.UpdateUser(ctx, user); err != nil {
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

func (s *Service) ListAuditLogs(ctx context.Context, limit, offset int) ([]domain.AdminAuditLog, int, error) {
	if s.audit == nil {
		return nil, 0, nil
	}
	return s.audit.ListAuditLogs(ctx, limit, offset)
}

func (s *Service) Audit(ctx context.Context, adminID int64, action, targetType, targetID string, detail any) {
	if s.audit == nil {
		return
	}
	_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: action, TargetType: targetType, TargetID: targetID, DetailJSON: mustJSON(detail)})
}

func (s *Service) UpdateUserStatus(ctx context.Context, adminID int64, userID int64, status domain.UserStatus) error {
	if err := s.users.UpdateUserStatus(ctx, userID, status); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "user.status", TargetType: "user", TargetID: toString(userID), DetailJSON: mustJSON(map[string]any{"status": status})})
	}
	return nil
}

func (s *Service) ListInstances(ctx context.Context, limit, offset int) ([]domain.VPSInstance, int, error) {
	return s.vps.ListInstances(ctx, limit, offset)
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
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "settings.update", TargetType: "setting", TargetID: key, DetailJSON: valueJSON})
	}
	return nil
}

func shouldSanitizePlainSetting(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "site_name", "site_title", "site_subtitle", "site_description", "site_keywords",
		"company_name", "contact_phone", "contact_email", "contact_qq",
		"icp_number", "psbe_number", "maintenance_message":
		return true
	default:
		return false
	}
}

func (s *Service) ListSettings(ctx context.Context) ([]domain.Setting, error) {
	return s.settings.ListSettings(ctx)
}

func (s *Service) GetSetting(ctx context.Context, key string) (domain.Setting, error) {
	return s.settings.GetSetting(ctx, key)
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
	if err := s.groups.CreatePermissionGroup(ctx, group); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "permission_group.create", TargetType: "permission_group", TargetID: toString(group.ID), DetailJSON: mustJSON(map[string]any{"name": group.Name})})
	}
	return nil
}

func (s *Service) UpdatePermissionGroup(ctx context.Context, adminID int64, group domain.PermissionGroup) error {
	if err := s.groups.UpdatePermissionGroup(ctx, group); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.AddAuditLog(ctx, domain.AdminAuditLog{AdminID: adminID, Action: "permission_group.update", TargetType: "permission_group", TargetID: toString(group.ID), DetailJSON: mustJSON(map[string]any{"name": group.Name})})
	}
	return nil
}

func (s *Service) DeletePermissionGroup(ctx context.Context, adminID int64, id int64) error {
	if err := s.groups.DeletePermissionGroup(ctx, id); err != nil {
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

func (s *Service) CreateAdmin(ctx context.Context, adminID int64, username, email, qq, password string, permissionGroupID *int64) (domain.User, error) {
	username, err := trimAndValidateRequired(username, maxLenUsername)
	if err != nil {
		return domain.User{}, appshared.ErrInvalidInput
	}
	email, err = trimAndValidateRequired(email, maxLenEmail)
	if err != nil {
		return domain.User{}, appshared.ErrInvalidInput
	}
	qq, err = trimAndValidateOptional(qq, maxLenQQ)
	if err != nil {
		return domain.User{}, appshared.ErrInvalidInput
	}
	password, err = trimAndValidateRequired(password, maxLenPassword)
	if err != nil {
		return domain.User{}, appshared.ErrInvalidInput
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
	var err error
	username, err = trimAndValidateRequired(username, maxLenUsername)
	if err != nil {
		return appshared.ErrInvalidInput
	}
	email, err = trimAndValidateRequired(email, maxLenEmail)
	if err != nil {
		return appshared.ErrInvalidInput
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
	existing, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if existing.Role != domain.UserRoleAdmin {
		return appshared.ErrNotFound
	}
	if err := s.users.UpdateUserStatus(ctx, userID, status); err != nil {
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
	if err := s.users.UpdateUserStatus(ctx, userID, domain.UserStatusDisabled); err != nil {
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
