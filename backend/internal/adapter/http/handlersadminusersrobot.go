package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
)

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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	var payload struct {
		Text      string `json:"text"`
		Sender    string `json:"sender"`
		Timestamp any    `json:"timestamp"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if h.adminSvc != nil || h.settingsSvc != nil {
		if enabled := strings.ToLower(h.getSettingValueByKey(c, "robot_webhook_enabled")); enabled == "false" {
			c.JSON(http.StatusForbidden, gin.H{"error": "robot webhook disabled"})
			return
		}
		secret := h.getSettingValueByKey(c, "robot_webhook_secret")
		if secret != "" {
			signature := c.GetHeader("X-Signature")
			if signature == "" {
				signature = c.GetHeader("X-Robot-Signature")
			}
			if signature == "" || !verifyHMAC(body, secret, signature) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
				return
			}
		}
	}
	text := strings.TrimSpace(payload.Text)
	if strings.HasPrefix(text, "通过订单") {
		rest := strings.TrimSpace(strings.TrimPrefix(text, "通过订单"))
		idStr := strings.Fields(rest)
		if len(idStr) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing order id"})
			return
		}
		orderID, err := strconv.ParseInt(idStr[0], 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing order id"})
			return
		}
		orderID, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
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
	c.JSON(http.StatusBadRequest, gin.H{"error": "unknown command"})
}

func (h *Handler) AdminLogin(c *gin.Context) {
	var payload struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	user, err := h.authSvc.Login(c, payload.Username, payload.Password)
	if err != nil || user.Role != domain.UserRoleAdmin {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	accessToken, err := h.signAuthToken(user.ID, string(user.Role), 24*time.Hour, "access")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sign token failed"})
		return
	}
	refreshToken, err := h.signAuthToken(user.ID, string(user.Role), 7*24*time.Hour, "refresh")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sign token failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    86400,
	})
}

func (h *Handler) AdminRefresh(c *gin.Context) {
	var payload struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	claims, err := h.parseRefreshToken(payload.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}
	userID, ok := parseMapInt64(claims["user_id"])
	if !ok || userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}
	role, _ := claims["role"].(string)
	if role != string(domain.UserRoleAdmin) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}
	if h.adminSvc != nil {
		user, err := h.adminSvc.GetUser(c, userID)
		if err != nil || user.Role != domain.UserRoleAdmin {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
			return
		}
	}
	accessToken, err := h.signAuthToken(userID, role, 24*time.Hour, "access")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sign token failed"})
		return
	}
	newRefreshToken, err := h.signAuthToken(userID, role, 7*24*time.Hour, "refresh")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sign token failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": newRefreshToken,
		"expires_in":    86400,
	})
}

func (h *Handler) AdminUsers(c *gin.Context) {
	limit, offset := paging(c)
	users, total, err := h.adminSvc.ListUsers(c, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": toUserDTOs(users), "total": total})
}

func (h *Handler) AdminUserDetail(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	user, err := h.adminSvc.GetUser(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if payload.Role != "" && strings.TrimSpace(payload.Role) != string(domain.UserRoleUser) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "admin role not allowed"})
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
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	user, err := h.adminSvc.GetUser(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "admin role not allowed"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	user, err := h.adminSvc.GetUser(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if user.Role == domain.UserRoleAdmin {
		c.JSON(http.StatusBadRequest, gin.H{"error": "admin user not editable"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	user, err := h.adminSvc.GetUser(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if user.Role == domain.UserRoleAdmin {
		c.JSON(http.StatusBadRequest, gin.H{"error": "admin user not editable"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "realname disabled"})
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var payload struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	}
	if err := bindJSON(c, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	user, err := h.adminSvc.GetUser(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if user.Role == domain.UserRoleAdmin {
		c.JSON(http.StatusBadRequest, gin.H{"error": "admin user not editable"})
		return
	}
	record, err := h.realnameSvc.Latest(c, id)
	if err != nil {
		if err == appshared.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "realname record not found"})
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
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if user.Role != domain.UserRoleUser {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not a user account"})
		return
	}
	if user.Status != domain.UserStatusActive {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user disabled"})
		return
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"role":    user.Role,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})
	signed, err := token.SignedString(h.jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sign token failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"access_token": signed, "expires_in": 86400, "user": gin.H{"id": user.ID, "username": user.Username, "role": user.Role}})
}

func (h *Handler) AdminQQAvatar(c *gin.Context) {
	qq := strings.TrimSpace(c.Param("qq"))
	if qq == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid qq"})
		return
	}
	for _, r := range qq {
		if r < '0' || r > '9' {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid qq"})
			return
		}
	}
	url := "https://q1.qlogo.cn/g?b=qq&nk=" + qq + "&s=100"
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "request failed"})
		return
	}
	client := &http.Client{Timeout: 6 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fetch failed"})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "avatar not found"})
		return
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}
	c.Header("Cache-Control", "public, max-age=86400")
	c.DataFromReader(http.StatusOK, resp.ContentLength, contentType, resp.Body, nil)
}
