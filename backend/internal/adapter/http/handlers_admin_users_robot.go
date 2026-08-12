package http

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
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

// robotWebhookTimestampWindow is how old a robot webhook timestamp may be.
const robotWebhookTimestampWindow = 5 * time.Minute

// robotWebhookBodyBytes bounds the public robot approval webhook payload; the
// body is a small JSON document, so a generous cap still prevents buffering a
// large scripted payload.
const robotWebhookBodyBytes int64 = 64 << 10

// robotWebhookReplayKey derives a bounded, deterministic replay key for a
// signed webhook command. It is a SHA-256 digest of the normalized timestamp
// plus the command text, so identical deliveries within the window map to the
// same row in the durable ledger.
func robotWebhookReplayKey(text string, ts time.Time) string {
	sum := sha256.Sum256([]byte(ts.Format(time.RFC3339Nano) + "|" + strings.TrimSpace(text)))
	return hex.EncodeToString(sum[:])
}

// parseRobotWebhookTimestamp accepts RFC3339 or a Unix seconds number.
func parseRobotWebhookTimestamp(v any) (time.Time, bool) {
	switch t := v.(type) {
	case string:
		s := strings.TrimSpace(t)
		if ts, err := time.Parse(time.RFC3339Nano, s); err == nil {
			return ts, true
		}
		if sec, err := strconv.ParseInt(s, 10, 64); err == nil && sec > 0 {
			return time.Unix(sec, 0), true
		}
	case float64:
		if t > 0 {
			return time.Unix(int64(t), 0), true
		}
	case json.Number:
		if sec, err := t.Int64(); err == nil && sec > 0 {
			return time.Unix(sec, 0), true
		}
	}
	return time.Time{}, false
}

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

func (h *Handler) RobotWebhook(c *gin.Context) {
	// Public webhook: cap the body well below the global limit so a scripted
	// client cannot make us buffer a huge JSON document. Overflow returns 413.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, robotWebhookBodyBytes)
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": domain.ErrRequestBodyTooLarge.Error()})
			return
		}
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
	// Signature verification is unconditional (P1-3): even when the settings
	// service is unavailable this route must not fall through without proving
	// knowledge of the shared secret.
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

	// Freshness + replay protection: the signed body carries a timestamp that
	// must be close to now, and an identical command+timestamp may only be
	// accepted once within the window.
	ts, ok := parseRobotWebhookTimestamp(payload.Timestamp)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidTimestamp.Error()})
		return
	}
	if delta := time.Since(ts); delta < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrStaleTimestamp.Error()})
		return
	} else if delta > robotWebhookTimestampWindow {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrStaleTimestamp.Error()})
		return
	}
	if h.robotWebhookReplay == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": domain.ErrNotSupported.Error()})
		return
	}
	// Durable, multi-instance replay protection: the replay key is claimed via a
	// database unique insert so a restart or a replica switch cannot replay a
	// captured webhook. Expired rows are swept lazily to keep the ledger bounded.
	_ = h.robotWebhookReplay.DeleteExpiredRobotWebhookNonces(c.Request.Context())
	claimed, err := h.robotWebhookReplay.SaveRobotWebhookNonce(c.Request.Context(), robotWebhookReplayKey(payload.Text, ts), time.Now().Add(robotWebhookTimestampWindow))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrSignatureCheckFailed.Error()})
		return
	}
	if !claimed {
		c.JSON(http.StatusConflict, gin.H{"error": domain.ErrReplayedWebhook.Error()})
		return
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
		if err := h.orderSvc.ApproveOrder(c.Request.Context(), domain.RobotWebhookOperatorID, orderID); err != nil {
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
		if err := h.orderSvc.RejectOrder(c.Request.Context(), domain.RobotWebhookOperatorID, orderID, reason); err != nil {
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
		Username  string `json:"username"`
		Password  string `json:"password"`
		AdminPath string `json:"admin_path"`
	}
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
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
	if !strings.EqualFold(requestPath, configuredPath) {
		c.JSON(http.StatusForbidden, gin.H{"error": domain.ErrAdminPathMismatch.Error()})
		return
	}
	accountKey := "admin_login:account:" + normalizeAdminSecurityKey(payload.Username)
	ip := strings.TrimSpace(c.ClientIP())
	if ip == "" {
		ip = "unknown"
	}
	now := time.Now()
	// Account, IP and global budgets are enforced separately and durably so a
	// single source (or a distributed spray) rotating usernames or instances
	// cannot reset another dimension's budget (P1-1 follow-up).
	loginDims := []budgetDim{
		{key: accountKey, limit: adminLoginFailureThreshold, window: adminLoginCooldown},
		{key: "admin_login:ip:" + ip, limit: adminLoginIPMax, window: adminLoginIPWindow},
		{key: "admin_login:global", limit: adminLoginGlobalMax, window: adminLoginGlobalWindow},
	}
	if _, locked := h.peekAuthBudgets(c.Request.Context(), loginDims, now); locked {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": domain.ErrTooManyAttempts.Error()})
		return
	}
	user, err := h.authSvc.Login(c.Request.Context(), payload.Username, payload.Password)
	if err != nil || user.Role != domain.UserRoleAdmin {
		lockedUntil, locked := h.recordAuthFailures(c.Request.Context(), loginDims, now)
		if locked {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":        domain.ErrTooManyAttempts.Error(),
				"locked_until": lockedUntil.Unix(),
			})
			return
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidCredentials.Error()})
		return
	}
	// On success reset the account and IP dimensions only — the global budget
	// is left intact so a single success cannot wipe a distributed-spray budget.
	h.resetAuthFailures(c.Request.Context(), accountKey, "admin_login:ip:"+ip)
	settings := h.loadAuthSettings(c.Request.Context())
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
	familyID := newTokenID()
	accessToken, err := h.signAuthTokenWithMFA(user.ID, string(user.Role), user.PasswordVersion, 24*time.Hour, "access", 0, familyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrSignTokenFailed.Error()})
		return
	}
	refreshToken, err := h.signAuthTokenWithMFA(user.ID, string(user.Role), user.PasswordVersion, 7*24*time.Hour, "refresh", 0, familyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrSignTokenFailed.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"access_token":      accessToken,
		"refresh_token":     refreshToken,
		"expires_in":        tokenExpirySeconds,
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
		writeBodyError(c, err)
		return
	}
	claims, err := h.parseRefreshToken(c.Request.Context(), payload.RefreshToken)
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
	familyID, _ := claims["fid"].(string)
	if strings.TrimSpace(familyID) == "" {
		familyID, _ = claims["jti"].(string)
		if strings.TrimSpace(familyID) == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidRefreshToken.Error()})
			return
		}
	}
	mfa := 0
	if v, ok := parseMapInt64(claims["mfa"]); ok && v > 0 {
		mfa = 1
	}
	var pvv int64
	if h.adminSvc != nil {
		user, err := h.adminSvc.GetUser(c.Request.Context(), userID)
		if err != nil || user.Role != domain.UserRoleAdmin || user.Status != domain.UserStatusActive {
			c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidRefreshToken.Error()})
			return
		}
		if tokenIssuedBeforePasswordChange(user, claims) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidRefreshToken.Error()})
			return
		}
		pvv = user.PasswordVersion
	}
	accessToken, err := h.signAuthTokenWithMFA(userID, role, pvv, 24*time.Hour, "access", mfa, familyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrSignTokenFailed.Error()})
		return
	}
	newRefreshToken, err := h.signAuthTokenWithMFA(userID, role, pvv, 7*24*time.Hour, "refresh", mfa, familyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrSignTokenFailed.Error()})
		return
	}
	expiresAt, err := claims.GetExpirationTime()
	if err != nil || expiresAt == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidRefreshToken.Error()})
		return
	}
	rotated, familyRevoked, err := h.authSvc.RotateRefreshToken(
		c.Request.Context(), payload.RefreshToken, newRefreshToken, familyID,
		expiresAt.Time, time.Now().Add(7*24*time.Hour),
	)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": domain.ErrServiceUnavailable.Error()})
		return
	}
	if !rotated || familyRevoked {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidRefreshToken.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": newRefreshToken,
		"expires_in":    tokenExpirySeconds,
	})
}

func (h *Handler) Admin2FAUnlock(c *gin.Context) {
	var payload struct {
		TOTPCode string `json:"totp_code"`
	}
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
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
	user, err := h.authSvc.GetUser(c.Request.Context(), userID)
	if err != nil || !user.TOTPEnabled {
		c.JSON(http.StatusForbidden, gin.H{"error": domain.Err2faNotEnabled.Error()})
		return
	}
	if user.Status != domain.UserStatusActive {
		c.JSON(http.StatusForbidden, gin.H{"error": domain.ErrUserDisabled.Error()})
		return
	}
	// Account and IP dimensions are enforced separately so a stolen session
	// cannot be used to brute-force unlimited TOTP codes and so rotating the
	// source IP cannot reset the account budget (P0-3 follow-up).
	ip := strings.TrimSpace(c.ClientIP())
	if ip == "" {
		ip = "unknown"
	}
	accountKey := "admin_2fa:account:" + normalizeAdminSecurityKey(user.Username)
	ipKey := "admin_2fa:ip:" + ip
	now := time.Now()
	faDims := []budgetDim{
		{key: accountKey, limit: admin2FAAttemptThreshold, window: admin2FACooldown},
		{key: ipKey, limit: admin2FAAttemptThreshold * 4, window: admin2FACooldown},
	}
	if _, locked := h.peekAuthBudgets(c.Request.Context(), faDims, now); locked {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error": domain.ErrTooManyAttempts.Error(),
		})
		return
	}
	if err := h.authSvc.VerifyTOTP(c.Request.Context(), userID, payload.TOTPCode); err != nil {
		// Temporary cooldown only. The admin account is deliberately never
		// disabled by failed 2FA attempts: permanent disable is an operator
		// denial-of-service (P0-3).
		lockedUntil, locked := h.recordAuthFailures(c.Request.Context(), faDims, now)
		if locked {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":        domain.ErrTooManyAttempts.Error(),
				"locked_until": lockedUntil.Unix(),
			})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalid2faCode.Error()})
		return
	}
	h.resetAuthFailures(c.Request.Context(), accountKey, ipKey)
	oldFamilyID, _ := claims["fid"].(string)
	if strings.TrimSpace(oldFamilyID) == "" {
		oldFamilyID, _ = claims["jti"].(string)
	}
	if strings.TrimSpace(oldFamilyID) == "" || h.authSvc.RevokeRefreshTokenFamily(c.Request.Context(), oldFamilyID, time.Now().Add(8*24*time.Hour)) != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": domain.ErrServiceUnavailable.Error()})
		return
	}
	familyID := newTokenID()
	accessToken, err := h.signAuthTokenWithMFA(userID, role, user.PasswordVersion, 24*time.Hour, "access", 1, familyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrSignTokenFailed.Error()})
		return
	}
	refreshToken, err := h.signAuthTokenWithMFA(userID, role, user.PasswordVersion, 7*24*time.Hour, "refresh", 1, familyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrSignTokenFailed.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    tokenExpirySeconds,
		"mfa_unlocked":  true,
	})
}

func (h *Handler) Admin2FASetup(c *gin.Context) {
	var payload struct {
		Password    string `json:"password"`
		CurrentCode string `json:"current_code"`
	}
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	userID, _, ok := h.parseAdminFromAuthorization(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidToken.Error()})
		return
	}
	settings := h.loadAuthSettings(c.Request.Context())
	if !settings.TwoFAEnabled {
		c.JSON(http.StatusForbidden, gin.H{"error": domain.Err2faDisabled.Error()})
		return
	}
	user, err := h.authSvc.GetUser(c.Request.Context(), userID)
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
	secret, otpURL, err := h.authSvc.SetupTOTP(c.Request.Context(), userID, payload.Password, payload.CurrentCode)
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
		writeBodyError(c, err)
		return
	}
	userID, _, ok := h.parseAdminFromAuthorization(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidToken.Error()})
		return
	}
	if err := h.authSvc.ConfirmTOTP(c.Request.Context(), userID, payload.Code); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminUsers(c *gin.Context) {
	limit, offset := paging(c)
	var query struct {
		Keyword string `form:"keyword"`
		Status  string `form:"status"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}
	// web_api_contract F5: keyword/status server-side filter, the frontend no
	// longer pulls 500 rows/page and filters client-side.
	users, total, err := h.adminSvc.ListUsersFiltered(c.Request.Context(), strings.TrimSpace(query.Keyword), strings.TrimSpace(query.Status), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrListError.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toUserDTOs(users), "total": total})
}

func (h *Handler) AdminUserDetail(c *gin.Context) {
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	user, err := h.adminSvc.GetUser(c.Request.Context(), uri.ID)
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
		writeBodyError(c, err)
		return
	}
	if payload.Role != "" && strings.TrimSpace(payload.Role) != string(domain.UserRoleUser) {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrAdminRoleNotAllowed.Error()})
		return
	}
	user, err := h.adminSvc.CreateUser(c.Request.Context(), getUserID(c), domain.User{
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
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	var payload struct {
		Username          *string `json:"username"`
		Email             *string `json:"email"`
		QQ                *string `json:"qq"`
		Phone             *string `json:"phone"`
		Bio               *string `json:"bio"`
		Intro             *string `json:"intro"`
		Avatar            *string `json:"avatar"`
		Role              *string `json:"role" binding:"omitempty,oneof=user"`
		Status            *string `json:"status" binding:"omitempty,oneof=active disabled blocked"`
		PermissionGroupID *int64  `json:"permission_group_id"`
		UserTierGroupID   *int64  `json:"user_tier_group_id"`
		UserTierExpireAt  *string `json:"user_tier_expire_at"`
	}
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	user, err := h.adminSvc.GetUser(c.Request.Context(), uri.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrUserNotFound.Error()})
		return
	}
	// An admin account is never editable through the generic user route: role,
	// status and permission-group changes go through /admins/* with the
	// primary-admin guard, and profile edits on an admin are only allowed for the
	// target admin themself or the primary admin. Otherwise a sub-admin holding
	// `user.update` could demote or disable the primary admin (P0-5).
	if user.Role == domain.UserRoleAdmin {
		if payload.Role != nil || payload.Status != nil || payload.PermissionGroupID != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrAdminUserNotEditable.Error()})
			return
		}
		actorID := getUserID(c)
		if actorID != user.ID {
			primary, pErr := h.isPrimaryAdminActor(c, actorID)
			if pErr != nil || !primary {
				c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrAdminUserNotEditable.Error()})
				return
			}
		}
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
	// role / status / permission_group_id are intentionally never applied here:
	// the generic user route must not be able to change them (P0-5 / P2-7), and
	// the service layer enforces the same rule. The DTO binding tags still reject
	// invalid role/status enums, the admin-target guard above rejects any attempt
	// to send these fields for an admin account, and the dedicated
	// /users/:id/status endpoint is the only path that mutates status.
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
	if err := h.adminSvc.UpdateUser(c.Request.Context(), getUserID(c), user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminUserResetPassword(c *gin.Context) {
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	var payload struct {
		Password string `json:"password"`
	}
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	user, err := h.adminSvc.GetUser(c.Request.Context(), uri.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrUserNotFound.Error()})
		return
	}
	if user.Role == domain.UserRoleAdmin {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrAdminUserNotEditable.Error()})
		return
	}
	if err := h.adminSvc.ResetUserPassword(c.Request.Context(), getUserID(c), uri.ID, payload.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminUserStatus(c *gin.Context) {
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	var payload struct {
		Status string `json:"status" binding:"required,oneof=active disabled blocked"`
	}
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	user, err := h.adminSvc.GetUser(c.Request.Context(), uri.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrUserNotFound.Error()})
		return
	}
	if user.Role == domain.UserRoleAdmin {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrAdminUserNotEditable.Error()})
		return
	}
	// The enum is enforced declaratively by the validator above; the service
	// layer re-validates so direct callers cannot bypass it either.
	status := domain.UserStatus(strings.TrimSpace(payload.Status))
	if err := h.adminSvc.UpdateUserStatus(c.Request.Context(), getUserID(c), uri.ID, status); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminUserRealNameStatus(c *gin.Context) {
	if h.realNameSvc == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrRealnameDisabled.Error()})
		return
	}
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	var payload struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	}
	if err := bindJSON(c, &payload); err != nil {
		writeBodyError(c, err)
		return
	}
	user, err := h.adminSvc.GetUser(c.Request.Context(), uri.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrUserNotFound.Error()})
		return
	}
	if user.Role == domain.UserRoleAdmin {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrAdminUserNotEditable.Error()})
		return
	}
	record, err := h.realNameSvc.Latest(c.Request.Context(), uri.ID)
	if err != nil {
		if err == appshared.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": domain.ErrRealnameRecordNotFound.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.realNameSvc.UpdateStatus(c.Request.Context(), record.ID, payload.Status, payload.Reason); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updated, err := h.realNameSvc.Latest(c.Request.Context(), uri.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toRealNameVerificationDTO(updated, h.realNameRedirectURL(c.Request.Context(), updated)))
}

func (h *Handler) AdminUserImpersonate(c *gin.Context) {
	var uri adminIDURI
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidId.Error()})
		return
	}
	operatorID := getUserID(c)
	if h.adminSvc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": domain.ErrAdminServiceUnavailable.Error()})
		return
	}
	primary, err := h.isPrimaryAdminActor(c.Request.Context(), operatorID)
	if err != nil || !primary {
		c.JSON(http.StatusForbidden, gin.H{"error": domain.ErrPrimaryAdminRequired.Error()})
		return
	}
	user, err := h.adminSvc.GetUser(c.Request.Context(), uri.ID)
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
	// The token carries the operator's admin id in the `imp` claim so every
	// subsequent request surfaces the impersonator in RequestActor and audit logs
	// can attribute the action to the admin who issued it, not to the impersonated
	// user (Issue #51 impersonation audit). A DB audit record is written so the
	// impersonation itself is traceable, not just the stdlib log line.
	if err := h.adminSvc.AuditRequired(c.Request.Context(), operatorID, "admin.impersonate", "user", strconv.FormatInt(user.ID, 10), map[string]any{
		"target_username": user.Username,
		"ip":              c.ClientIP(),
	}); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": domain.ErrAuditWriteFailed.Error()})
		return
	}
	log.Printf("[AUDIT] impersonate: admin_id=%d target_user_id=%d target_username=%s ip=%s", operatorID, user.ID, user.Username, c.ClientIP())
	operator, err := h.adminSvc.GetUser(c.Request.Context(), operatorID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidToken.Error()})
		return
	}
	signed, err := h.signImpersonationToken(operatorID, operator.PasswordVersion, user.ID, string(user.Role), user.PasswordVersion, 24*time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": domain.ErrSignTokenFailed.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"access_token": signed, "expires_in": tokenExpirySeconds, "user": gin.H{"id": user.ID, "username": user.Username, "role": user.Role}})
}
