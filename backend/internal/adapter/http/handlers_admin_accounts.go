package http

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"strings"
	"time"
	"xiaoheiplay/internal/domain"
)

func (h *Handler) AdminAdmins(c *gin.Context) {
	limit, offset := paging(c)
	status := strings.TrimSpace(c.Query("status"))
	if status == "" {
		status = "active"
	}
	admins, total, err := h.adminSvc.ListAdmins(c, status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": admins, "total": total})
}

func (h *Handler) AdminAdminCreate(c *gin.Context) {
	var payload struct {
		Username          string `json:"username" binding:"required"`
		Email             string `json:"email" binding:"required,email"`
		QQ                string `json:"qq"`
		Password          string `json:"password" binding:"required"`
		PermissionGroupID *int64 `json:"permission_group_id"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	if payload.QQ != "" && !isDigits(payload.QQ) {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrQqMustBeNumeric.Error()})
		return
	}
	admin, err := h.adminSvc.CreateAdmin(c, getUserID(c), payload.Username, payload.Email, payload.QQ, payload.Password, payload.PermissionGroupID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toUserDTO(admin))
}

func (h *Handler) AdminAdminUpdate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		Username          string `json:"username" binding:"required"`
		Email             string `json:"email" binding:"required,email"`
		QQ                string `json:"qq"`
		PermissionGroupID *int64 `json:"permission_group_id"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	if payload.QQ != "" && !isDigits(payload.QQ) {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrQqMustBeNumeric.Error()})
		return
	}
	if id == getUserID(c) {
		existing, err := h.adminSvc.GetUser(c, id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// Allow self-update requests that include the current permission_group_id.
		// Only block an actual attempt to switch permission group.
		if payload.PermissionGroupID != nil {
			existingGroupID := int64(0)
			if existing.PermissionGroupID != nil {
				existingGroupID = *existing.PermissionGroupID
			}
			if *payload.PermissionGroupID != existingGroupID {
				c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrCannotUpdatePermissionGroup.Error()})
				return
			}
		}
		payload.PermissionGroupID = existing.PermissionGroupID
	}
	if err := h.adminSvc.UpdateAdmin(c, getUserID(c), id, payload.Username, payload.Email, payload.QQ, payload.PermissionGroupID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminAdminStatus(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		Status string `json:"status"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	if id == getUserID(c) {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrCannotUpdateSelfStatus.Error()})
		return
	}
	status := strings.TrimSpace(payload.Status)
	if status != string(domain.UserStatusActive) && status != string(domain.UserStatusDisabled) {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidStatus.Error()})
		return
	}
	if err := h.adminSvc.UpdateAdminStatus(c, getUserID(c), id, domain.UserStatus(status)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminAdminDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.adminSvc.DeleteAdmin(c, getUserID(c), id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPermissionGroups(c *gin.Context) {
	groups, err := h.adminSvc.ListPermissionGroups(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": groups})
}

func (h *Handler) AdminPermissionGroupCreate(c *gin.Context) {
	var payload struct {
		Name        string   `json:"name" binding:"required"`
		Description string   `json:"description"`
		Permissions []string `json:"permissions" binding:"required"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	permJSON := mustJSON(payload.Permissions)
	group := &domain.PermissionGroup{
		Name:            payload.Name,
		Description:     payload.Description,
		PermissionsJSON: permJSON,
	}
	if err := h.adminSvc.CreatePermissionGroup(c, getUserID(c), group); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, group)
}

func (h *Handler) AdminPermissionGroupUpdate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		Name        string   `json:"name" binding:"required"`
		Description string   `json:"description"`
		Permissions []string `json:"permissions" binding:"required"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	permJSON := mustJSON(payload.Permissions)
	group := domain.PermissionGroup{
		ID:              id,
		Name:            payload.Name,
		Description:     payload.Description,
		PermissionsJSON: permJSON,
	}
	if err := h.adminSvc.UpdatePermissionGroup(c, getUserID(c), group); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminPermissionGroupDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.adminSvc.DeletePermissionGroup(c, getUserID(c), id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminProfile(c *gin.Context) {
	userID := getUserID(c)
	user, err := h.adminSvc.GetUser(c, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
		return
	}
	dto := toUserDTO(user)
	if h.permissionSvc != nil {
		if isPrimary, err := h.permissionSvc.IsPrimaryAdmin(c, userID); err == nil && isPrimary {
			dto.Permissions = []string{"*"}
			c.JSON(http.StatusOK, dto)
			return
		}
		perms, err := h.permissionSvc.GetUserPermissions(c, userID)
		if err == nil {
			dto.Permissions = perms
		}
	}
	c.JSON(http.StatusOK, dto)
}

func (h *Handler) AdminProfileUpdate(c *gin.Context) {
	var payload struct {
		Email string `json:"email" binding:"omitempty,email"`
		QQ    string `json:"qq"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	if err := h.adminSvc.UpdateProfile(c, getUserID(c), payload.Email, payload.QQ); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminProfileChangePassword(c *gin.Context) {
	var payload struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	if err := h.adminSvc.ChangePassword(c, getUserID(c), payload.OldPassword, payload.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminForgotPassword(c *gin.Context) {
	var payload struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	ip := strings.TrimSpace(c.ClientIP())
	if ip == "" {
		ip = "unknown"
	}
	if !forgotPwdLimiter.Allow("admin_forgot_password:ip:"+ip, 5, 15*time.Minute) ||
		!forgotPwdLimiter.Allow("admin_forgot_password:email:"+strings.ToLower(strings.TrimSpace(payload.Email)), 3, 15*time.Minute) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": domain.ErrTooManyRequests.Error()})
		return
	}
	if h.passwordReset == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	if err := h.passwordReset.RequestReset(c, payload.Email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminResetPassword(c *gin.Context) {
	var payload struct {
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	if h.passwordReset == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	if err := h.passwordReset.ResetPassword(c, payload.Token, payload.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
