package http

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"

	"github.com/gin-gonic/gin"
)

func (h *Handler) parseAdminFromAuthorization(c *gin.Context) (int64, string, bool) {
	auth := strings.TrimSpace(c.GetHeader("Authorization"))
	if !strings.HasPrefix(auth, "Bearer ") {
		return 0, "", false
	}
	raw := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	claims, err := h.parseAccessToken(c.Request.Context(), raw)
	if err != nil {
		return 0, "", false
	}
	userID, ok := parseMapInt64(claims["user_id"])
	if !ok || userID <= 0 {
		return 0, "", false
	}
	role, _ := claims["role"].(string)
	if role != string(domain.UserRoleAdmin) {
		return 0, role, false
	}
	return userID, role, true
}

func (h *Handler) RobotApprove(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.orderSvc.ApproveOrder(c, 0, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) RobotReject(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		Reason string `json:"reason"`
	}
	if err := bindJSONOptional(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	if err := h.orderSvc.RejectOrder(c, 0, id, payload.Reason); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) RobotWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	var payload struct {
		Text      string `json:"text"`
		Sender    string `json:"sender"`
		Timestamp any    `json:"timestamp"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	if h.adminSvc != nil || h.settingsSvc != nil {
		if enabled := strings.ToLower(h.getSettingValueByKey(c, "robot_webhook_enabled")); enabled == "false" {
			c.JSON(http.StatusForbidden, gin.H{"error": domain.ErrRobotWebhookDisabled.Error()})
			return
		}
		secret := h.getSettingValueByKey(c, "robot_webhook_secret")
		if secret == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": domain.ErrRobotWebhookSecretRequired.Error()})
			return
		}
		signature := c.GetHeader("X-Signature")
		if signature == "" {
			signature = c.GetHeader("X-Robot-Signature")
		}
		if signature == "" || !verifyHMAC(body, secret, signature) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidSignature.Error()})
			return
		}
	}
	text := strings.TrimSpace(payload.Text)
	if strings.HasPrefix(text, "通过订单") {
		rest := strings.TrimSpace(strings.TrimPrefix(text, "通过订单"))
		idStr := strings.Fields(rest)
		if len(idStr) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrMissingOrderId.Error()})
			return
		}
		orderID, err := strconv.ParseInt(idStr[0], 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidOrderId.Error()})
			return
		}
		if err := h.orderSvc.ApproveOrder(c, 0, orderID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}
	if strings.HasPrefix(text, "驳回订单") {
		rest := strings.TrimSpace(strings.TrimPrefix(text, "驳回订单"))
		parts := strings.Fields(rest)
		if len(parts) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrMissingOrderId.Error()})
			return
		}
		orderID, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidOrderId.Error()})
			return
		}
		reason := ""
		if len(parts) > 1 {
			reason = strings.TrimSpace(strings.TrimPrefix(strings.Join(parts[1:], " "), "原因"))
		}
		if err := h.orderSvc.RejectOrder(c, 0, orderID, reason); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrUnknownCommand.Error()})
}

func (h *Handler) AdminLogin(c *gin.Context) {
	if h.authSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": domain.ErrNotInstalled.Error()})
		return
	}
	var payload struct {
		Username   string `json:"username"`
		Password   string `json:"password"`
		AdminPath  string `json:"admin_path"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	
	// 校验管理端路径
	requestPath := strings.TrimSpace(payload.AdminPath)
	if requestPath != "" {
		// 验证路径格式
		if err := ValidateAdminPath(requestPath); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	} else {
		requestPath = "admin"
	}
	
	// 获取配置的管理路径
	configuredPath := GetAdminPathFromSettings(h.settingsSvc)
	if configuredPath == "" {
		configuredPath = "admin"
	}
	
	// 验证路径是否匹配
	if requestPath != configuredPath {
		c.JSON(http.StatusForbidden, gin.H{"error": domain.ErrAdminPathMismatch.Error()})
		return
	}
	accountKey := normalizeAdminSecurityKey(payload.Username)
	now := time.Now()
	if cooling, lockedUntil := adminLoginGuard.IsCoolingDown(accountKey, now); cooling {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":        domain.ErrTooManyAttempts.Error(),
			"locked_until": lockedUntil.Unix(),
		})
		return
	}
	user, err := h.authSvc.Login(c, payload.Username, payload.Password)
	if err != nil || user.Role != domain.UserRoleAdmin {
		cooling, lockedUntil, _ := adminLoginGuard.RegisterFailure(accountKey, adminLoginFailureThreshold, adminLoginCooldown, now)
		if cooling {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":        domain.ErrTooManyAttempts.Error(),
				"locked_until": lockedUntil.Unix(),
			})
			return
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidCredentials.Error()})
		return
	}
	adminLoginGuard.Reset(accountKey)
	settings := h.loadAuthSettings(c)
	totpEnabled := user.TOTPEnabled
	mfaBindRequired := false
	mfaRequired := false
	mfaUnlocked := false
	if settings.TwoFAEnabled {
		if !totpEnabled {
			mfaBindRequired = true
		} else {
			mfaRequired = true
		}
	} else {
		mfaUnlocked = true
	}
	accessToken, err := h.signAuthTokenWithMFA(user.ID, string(user.Role), 24*time.Hour, "access", 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrSignTokenFailed.Error()})
		return
	}
	refreshToken, err := h.signAuthTokenWithMFA(user.ID, string(user.Role), 7*24*time.Hour, "refresh", 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrSignTokenFailed.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"access_token":      accessToken,
		"refresh_token":     refreshToken,
		"expires_in":        86400,
		"totp_enabled":      totpEnabled,
		"mfa_required":      mfaRequired,
		"mfa_bind_required": mfaBindRequired,
		"mfa_unlocked":      mfaUnlocked,
		"user":              gin.H{"id": user.ID, "username": user.Username, "role": user.Role},
	})
}

func (h *Handler) AdminRefresh(c *gin.Context) {
	var payload struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	claims, err := h.parseRefreshToken(payload.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidRefreshToken.Error()})
		return
	}
	userID, ok := parseMapInt64(claims["user_id"])
	if !ok || userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidRefreshToken.Error()})
		return
	}
	role, _ := claims["role"].(string)
	if role != string(domain.UserRoleAdmin) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidRefreshToken.Error()})
		return
	}
	mfa := 0
	if v, ok := parseMapInt64(claims["mfa"]); ok && v > 0 {
		mfa = 1
	}
	if h.adminSvc != nil {
		user, err := h.adminSvc.GetUser(c, userID)
		if err != nil || user.Role != domain.UserRoleAdmin {
			c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidRefreshToken.Error()})
			return
		}
		if tokenIssuedBeforePasswordChange(user, claims) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidRefreshToken.Error()})
			return
		}
	}
	accessToken, err := h.signAuthTokenWithMFA(userID, role, 24*time.Hour, "access", mfa)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrSignTokenFailed.Error()})
		return
	}
	newRefreshToken, err := h.signAuthTokenWithMFA(userID, role, 7*24*time.Hour, "refresh", mfa)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrSignTokenFailed.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": newRefreshToken,
		"expires_in":    86400,
	})
}

func (h *Handler) Admin2FAUnlock(c *gin.Context) {
	var payload struct {
		TOTPCode string `json:"totp_code"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	auth := strings.TrimSpace(c.GetHeader("Authorization"))
	if !strings.HasPrefix(auth, "Bearer ") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidToken.Error()})
		return
	}
	raw := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	claims, err := h.parseAccessToken(c.Request.Context(), raw)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidToken.Error()})
		return
	}
	userID, ok := parseMapInt64(claims["user_id"])
	if !ok || userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidToken.Error()})
		return
	}
	role, _ := claims["role"].(string)
	if role != string(domain.UserRoleAdmin) {
		c.JSON(http.StatusForbidden, gin.H{"error": domain.ErrAdminRequired.Error()})
		return
	}
	user, err := h.authSvc.GetUser(c, userID)
	if err != nil || !user.TOTPEnabled {
		c.JSON(http.StatusForbidden, gin.H{"error": domain.Err2faNotEnabled.Error()})
		return
	}
	if user.Status != domain.UserStatusActive {
		c.JSON(http.StatusForbidden, gin.H{"error": domain.ErrUserDisabled.Error()})
		return
	}
	accountKey := normalizeAdminSecurityKey(user.Username)
	if err := h.authSvc.VerifyTOTP(c, userID, payload.TOTPCode); err != nil {
		failures := admin2FAFailureGuard.RegisterFailure(accountKey)
		if failures >= admin2FAFailureThreshold {
			if h.adminSvc == nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrAdminServiceUnavailable.Error()})
				return
			}
			if updateErr := h.adminSvc.UpdateAdminStatus(c, userID, userID, domain.UserStatusDisabled); updateErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": updateErr.Error()})
				return
			}
			admin2FAFailureGuard.Reset(accountKey)
			c.JSON(http.StatusForbidden, gin.H{"error": domain.ErrUserDisabled.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalid2faCode.Error()})
		return
	}
	admin2FAFailureGuard.Reset(accountKey)
	accessToken, err := h.signAuthTokenWithMFA(userID, role, 24*time.Hour, "access", 1)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrSignTokenFailed.Error()})
		return
	}
	refreshToken, err := h.signAuthTokenWithMFA(userID, role, 7*24*time.Hour, "refresh", 1)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrSignTokenFailed.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    86400,
		"mfa_unlocked":  true,
	})
}

func (h *Handler) Admin2FASetup(c *gin.Context) {
	var payload struct {
		Password    string `json:"password"`
		CurrentCode string `json:"current_code"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	userID, _, ok := h.parseAdminFromAuthorization(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidToken.Error()})
		return
	}
	settings := h.loadAuthSettings(c)
	if !settings.TwoFAEnabled {
		c.JSON(http.StatusForbidden, gin.H{"error": domain.Err2faDisabled.Error()})
		return
	}
	user, err := h.authSvc.GetUser(c, userID)
	if err != nil || user.Role != domain.UserRoleAdmin {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrUserNotFound.Error()})
		return
	}
	if user.TOTPEnabled && !settings.TwoFARebindEnabled {
		c.JSON(http.StatusForbidden, gin.H{"error": domain.Err2faRebindDisabled.Error()})
		return
	}
	if !user.TOTPEnabled && !settings.TwoFABindEnabled {
		c.JSON(http.StatusForbidden, gin.H{"error": domain.Err2faBindDisabled.Error()})
		return
	}
	secret, otpURL, err := h.authSvc.SetupTOTP(c, userID, payload.Password, payload.CurrentCode)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"secret": secret, "otpauth_url": otpURL})
}

func (h *Handler) Admin2FAConfirm(c *gin.Context) {
	var payload struct {
		Code string `json:"code"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	userID, _, ok := h.parseAdminFromAuthorization(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidToken.Error()})
		return
	}
	if err := h.authSvc.ConfirmTOTP(c, userID, payload.Code); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminUsers(c *gin.Context) {
	limit, offset := paging(c)
	users, total, err := h.adminSvc.ListUsers(c, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrListError.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toUserDTOs(users), "total": total})
}

func (h *Handler) AdminUserDetail(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	user, err := h.adminSvc.GetUser(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrUserNotFound.Error()})
		return
	}
	c.JSON(http.StatusOK, toUserDTO(user))
}

func (h *Handler) AdminUserCreate(c *gin.Context) {
	var payload struct {
		Username          string `json:"username"`
		Email             string `json:"email"`
		QQ                string `json:"qq"`
		Phone             string `json:"phone"`
		Bio               string `json:"bio"`
		Intro             string `json:"intro"`
		Password          string `json:"password"`
		Role              string `json:"role"`
		Status            string `json:"status"`
		PermissionGroupID *int64 `json:"permission_group_id"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	if payload.Role != "" && strings.TrimSpace(payload.Role) != string(domain.UserRoleUser) {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrAdminRoleNotAllowed.Error()})
		return
	}
	user, err := h.adminSvc.CreateUser(c, getUserID(c), domain.User{
		Username:          payload.Username,
		Email:             payload.Email,
		QQ:                payload.QQ,
		Phone:             payload.Phone,
		Bio:               payload.Bio,
		Intro:             payload.Intro,
		PermissionGroupID: payload.PermissionGroupID,
		Role:              domain.UserRoleUser,
		Status:            domain.UserStatus(payload.Status),
	}, payload.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toUserDTO(user))
}

func (h *Handler) AdminUserUpdate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		Username          *string `json:"username"`
		Email             *string `json:"email"`
		QQ                *string `json:"qq"`
		Phone             *string `json:"phone"`
		Bio               *string `json:"bio"`
		Intro             *string `json:"intro"`
		Avatar            *string `json:"avatar"`
		Role              *string `json:"role"`
		Status            *string `json:"status"`
		PermissionGroupID *int64  `json:"permission_group_id"`
		UserTierGroupID   *int64  `json:"user_tier_group_id"`
		UserTierExpireAt  *string `json:"user_tier_expire_at"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	user, err := h.adminSvc.GetUser(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrUserNotFound.Error()})
		return
	}
	if payload.Username != nil {
		user.Username = strings.TrimSpace(*payload.Username)
	}
	if payload.Email != nil {
		user.Email = strings.TrimSpace(*payload.Email)
	}
	if payload.QQ != nil {
		user.QQ = strings.TrimSpace(*payload.QQ)
	}
	if payload.Phone != nil {
		user.Phone = strings.TrimSpace(*payload.Phone)
	}
	if payload.Bio != nil {
		user.Bio = *payload.Bio
	}
	if payload.Intro != nil {
		user.Intro = *payload.Intro
	}
	if payload.Avatar != nil {
		user.Avatar = strings.TrimSpace(*payload.Avatar)
	}
	if payload.Role != nil {
		role := strings.TrimSpace(*payload.Role)
		if role != "" && role != string(domain.UserRoleUser) {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrAdminRoleNotAllowed.Error()})
			return
		}
		user.Role = domain.UserRoleUser
	}
	if payload.Status != nil {
		user.Status = domain.UserStatus(strings.TrimSpace(*payload.Status))
	}
	if payload.PermissionGroupID != nil {
		user.PermissionGroupID = payload.PermissionGroupID
	}
	if payload.UserTierGroupID != nil {
		user.UserTierGroupID = payload.UserTierGroupID
	}
	if payload.UserTierExpireAt != nil {
		raw := strings.TrimSpace(*payload.UserTierExpireAt)
		if raw == "" {
			user.UserTierExpireAt = nil
		} else if v, err := time.Parse(time.RFC3339, raw); err == nil {
			user.UserTierExpireAt = &v
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidExpireAt.Error()})
			return
		}
	}
	if err := h.adminSvc.UpdateUser(c, getUserID(c), user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminUserResetPassword(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		Password string `json:"password"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	user, err := h.adminSvc.GetUser(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrUserNotFound.Error()})
		return
	}
	if user.Role == domain.UserRoleAdmin {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrAdminUserNotEditable.Error()})
		return
	}
	if err := h.adminSvc.ResetUserPassword(c, getUserID(c), id, payload.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminUserStatus(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		Status string `json:"status"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	user, err := h.adminSvc.GetUser(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrUserNotFound.Error()})
		return
	}
	if user.Role == domain.UserRoleAdmin {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrAdminUserNotEditable.Error()})
		return
	}
	status := domain.UserStatus(payload.Status)
	if err := h.adminSvc.UpdateUserStatus(c, getUserID(c), id, status); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminUserRealNameStatus(c *gin.Context) {
	if h.realnameSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrRealnameDisabled.Error()})
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidBody.Error()})
		return
	}
	user, err := h.adminSvc.GetUser(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrUserNotFound.Error()})
		return
	}
	if user.Role == domain.UserRoleAdmin {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrAdminUserNotEditable.Error()})
		return
	}
	record, err := h.realnameSvc.Latest(c, id)
	if err != nil {
		if err == appshared.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrRealnameRecordNotFound.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.realnameSvc.UpdateStatus(c, record.ID, payload.Status, payload.Reason); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updated, err := h.realnameSvc.Latest(c, id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toRealNameVerificationDTO(updated))
}

func (h *Handler) AdminUserImpersonate(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	user, err := h.adminSvc.GetUser(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrUserNotFound.Error()})
		return
	}
	if user.Role != domain.UserRoleUser {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrNotAUserAccount.Error()})
		return
	}
	if user.Status != domain.UserStatusActive {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrUserDisabled.Error()})
		return
	}
	operatorID := getUserID(c)
	log.Printf("[AUDIT] impersonate: admin_id=%d target_user_id=%d target_username=%s ip=%s", operatorID, user.ID, user.Username, c.ClientIP())
	signed, err := h.signAuthToken(user.ID, string(user.Role), 24*time.Hour, "access")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrSignTokenFailed.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"access_token": signed, "expires_in": 86400, "user": gin.H{"id": user.ID, "username": user.Username, "role": user.Role}})
}

func (h *Handler) AdminQQAvatar(c *gin.Context) {
	qq := strings.TrimSpace(c.Param("qq"))
	if qq == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidQq.Error()})
		return
	}
	for _, r := range qq {
		if r < '0' || r > '9' {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidQq.Error()})
			return
		}
	}
	url := "https://q1.qlogo.cn/g?b=qq&nk=" + qq + "&s=100"
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrRequestFailed.Error()})
		return
	}
	client := &http.Client{Timeout: 6 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrFetchFailed.Error()})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrAvatarNotFound.Error()})
		return
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}
	const maxAvatarSize = 10 << 20 // 10MB
	body := &io.LimitedReader{R: resp.Body, N: maxAvatarSize}
	c.Header("Cache-Control", "public, max-age=86400")
	c.DataFromReader(http.StatusOK, -1, contentType, body, nil)
}
