package usecase

import (
	"context"
	"encoding/json"
	"strings"

	"xiaoheiplay/internal/domain"
)

type PermissionService struct {
	users       UserRepository
	groups      PermissionGroupRepository
	permissions PermissionRepository
}

func NewPermissionService(users UserRepository, groups PermissionGroupRepository, permissions PermissionRepository) *PermissionService {
	return &PermissionService{users: users, groups: groups, permissions: permissions}
}

func (s *PermissionService) HasPermission(ctx context.Context, userID int64, permission string) (bool, error) {
	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return false, err
	}

	if user.Role != domain.UserRoleAdmin {
		return false, nil
	}

	return s.checkUserPermissions(ctx, user, permission)
}

func (s *PermissionService) HasPermissionForGroup(ctx context.Context, groupID int64, permission string) (bool, error) {
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

func (s *PermissionService) IsPrimaryAdmin(ctx context.Context, userID int64) (bool, error) {
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

func (s *PermissionService) HasAnyPermission(ctx context.Context, userID int64, permissions []string) (bool, error) {
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

func (s *PermissionService) HasAllPermissions(ctx context.Context, userID int64, permissions []string) (bool, error) {
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

func (s *PermissionService) GetUserPermissions(ctx context.Context, userID int64) ([]string, error) {
	user, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if user.Role != domain.UserRoleAdmin {
		return nil, nil
	}

	return s.getUserPermissionList(ctx, user)
}

func (s *PermissionService) checkUserPermissions(ctx context.Context, user domain.User, requiredPermission string) (bool, error) {
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

func (s *PermissionService) getUserPermissionList(ctx context.Context, user domain.User) ([]string, error) {
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
