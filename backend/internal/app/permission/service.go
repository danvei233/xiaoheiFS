package permission

import (
	"context"
	"encoding/json"
	"strings"

	appports "xiaoheiplay/internal/app/ports"
	"xiaoheiplay/internal/domain"
)

type Service struct {
	users       appports.UserRepository
	groups      appports.PermissionGroupRepository
	permissions appports.PermissionRepository
}

func NewService(users appports.UserRepository, groups appports.PermissionGroupRepository, permissions appports.PermissionRepository) *Service {
	return &Service{users: users, groups: groups, permissions: permissions}
}

func (s *Service) HasPermission(ctx context.Context, userID int64, permission string) (bool, error) {
	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return false, err
	}
	if user.Role != domain.UserRoleAdmin {
		return false, nil
	}
	return s.checkUserPermissions(ctx, user, permission)
}

func (s *Service) HasPermissionForGroup(ctx context.Context, groupID int64, permission string) (bool, error) {
	group, err := s.groups.GetPermissionGroup(ctx, groupID)
	if err != nil {
		return false, err
	}
	perms, err := parseGroupPermissions(group.PermissionsJSON)
	if err != nil {
		return false, err
	}
	return permissionListAllows(perms, permission), nil
}

func (s *Service) IsPrimaryAdmin(ctx context.Context, userID int64) (bool, error) {
	if userID == 0 {
		return false, nil
	}
	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return false, err
	}
	if user.Role != domain.UserRoleAdmin {
		return false, nil
	}
	minID, err := s.users.GetMinUserIDByRole(ctx, string(domain.UserRoleAdmin))
	if err != nil {
		return false, err
	}
	return minID == userID, nil
}

func (s *Service) HasAnyPermission(ctx context.Context, userID int64, permissions []string) (bool, error) {
	for _, perm := range permissions {
		has, err := s.HasPermission(ctx, userID, perm)
		if err != nil {
			return false, err
		}
		if has {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) HasAllPermissions(ctx context.Context, userID int64, permissions []string) (bool, error) {
	for _, perm := range permissions {
		has, err := s.HasPermission(ctx, userID, perm)
		if err != nil {
			return false, err
		}
		if !has {
			return false, nil
		}
	}
	return true, nil
}

func (s *Service) GetUserPermissions(ctx context.Context, userID int64) ([]string, error) {
	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.Role != domain.UserRoleAdmin {
		return nil, nil
	}
	return s.getUserPermissionList(ctx, user)
}

func (s *Service) ListPermissions(ctx context.Context) ([]domain.Permission, error) {
	return s.permissions.ListPermissions(ctx)
}

func (s *Service) GetPermissionByCode(ctx context.Context, code string) (domain.Permission, error) {
	return s.permissions.GetPermissionByCode(ctx, code)
}

func (s *Service) UpsertPermission(ctx context.Context, perm *domain.Permission) error {
	return s.permissions.UpsertPermission(ctx, perm)
}

func (s *Service) RegisterPermissions(ctx context.Context, perms []domain.PermissionDefinition) error {
	return s.permissions.RegisterPermissions(ctx, perms)
}

func (s *Service) checkUserPermissions(ctx context.Context, user domain.User, requiredPermission string) (bool, error) {
	if user.PermissionGroupID == nil {
		return false, nil
	}
	group, err := s.groups.GetPermissionGroup(ctx, *user.PermissionGroupID)
	if err != nil {
		return false, err
	}
	perms, err := parseGroupPermissions(group.PermissionsJSON)
	if err != nil {
		return false, err
	}
	return permissionListAllows(perms, requiredPermission), nil
}

func (s *Service) getUserPermissionList(ctx context.Context, user domain.User) ([]string, error) {
	if user.PermissionGroupID == nil {
		return []string{}, nil
	}
	group, err := s.groups.GetPermissionGroup(ctx, *user.PermissionGroupID)
	if err != nil {
		return nil, err
	}
	return parseGroupPermissions(group.PermissionsJSON)
}

func parseGroupPermissions(raw string) ([]string, error) {
	var perms []string
	if err := json.Unmarshal([]byte(raw), &perms); err != nil {
		return nil, err
	}
	return perms, nil
}

func permissionListAllows(permissions []string, requiredPermission string) bool {
	for _, perm := range permissions {
		if perm == "*" || perm == requiredPermission {
			return true
		}
		if strings.HasSuffix(perm, "*") {
			prefix := strings.TrimSuffix(perm, "*")
			if strings.HasPrefix(requiredPermission, prefix) {
				return true
			}
		}
	}
	return false
}
