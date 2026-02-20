package http

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"strconv"
	"strings"
	appapikey "xiaoheiplay/internal/app/apikey"
	apppermission "xiaoheiplay/internal/app/permission"
	appshared "xiaoheiplay/internal/app/shared"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/pkg/permissions"
)

type Middleware struct {
	jwtSecret     []byte
	apiKeys       *appapikey.Service
	permissionSvc *apppermission.Service
	authSvc       AuthService
	settingsSvc   SettingsService
}

func NewMiddleware(jwtSecret string, apiKeys *appapikey.Service, permissionSvc *apppermission.Service, authSvc AuthService, settingsSvc SettingsService) *Middleware {
	return &Middleware{
		jwtSecret:     []byte(jwtSecret),
		apiKeys:       apiKeys,
		permissionSvc: permissionSvc,
		authSvc:       authSvc,
		settingsSvc:   settingsSvc,
	}
}

func (m *Middleware) RequireUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := m.parseToken(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidToken.Error()})
			return
		}
		userID, ok := toInt64Claim(claims["user_id"])
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidToken.Error()})
			return
		}
		c.Set("user_id", userID)
		c.Set("role", claims["role"])
		c.Next()
	}
}

func (m *Middleware) RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := m.parseToken(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidToken.Error()})
			return
		}
		role, _ := claims["role"].(string)
		if role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": domain.ErrAdminRequired.Error()})
			return
		}
		userID, ok := toInt64Claim(claims["user_id"])
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidToken.Error()})
			return
		}
		c.Set("user_id", userID)
		c.Set("role", role)
		c.Next()
	}
}

func (m *Middleware) RequireAdminPermissionAuto() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		perm, permOK := permissions.InferPermissionCode(c.Request.Method, path)

		auth := c.GetHeader("Authorization")
		if strings.HasPrefix(auth, "Bearer ") {
			token := strings.TrimPrefix(auth, "Bearer ")
			if !looksLikeJWT(token) {
				m.requireAdminPermissionByAPIKey(c, token, perm, permOK)
				return
			}
		} else if key := c.GetHeader("X-API-Key"); key != "" {
			m.requireAdminPermissionByAPIKey(c, key, perm, permOK)
			return
		}

		claims, err := m.parseToken(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidToken.Error()})
			return
		}
		role, _ := claims["role"].(string)
		if role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": domain.ErrAdminRequired.Error()})
			return
		}
		userID, ok := toInt64Claim(claims["user_id"])
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidToken.Error()})
			return
		}
		c.Set("user_id", userID)
		c.Set("role", role)

		if !m.isAdmin2FAAllowlisted(path) {
			m.enforceAdmin2FAGate(c, userID, claims)
			if c.IsAborted() {
				return
			}
		}

		if m.permissionSvc == nil {
			c.Next()
			return
		}
		isPrimary, err := m.permissionSvc.IsPrimaryAdmin(c, userID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": domain.ErrPermissionCheckFailed.Error()})
			return
		}
		if isPrimary {
			c.Next()
			return
		}

		if !permOK {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": domain.ErrPermissionDenied.Error()})
			return
		}
		if isAdminSelfPass(c, perm, userID) {
			c.Next()
			return
		}
		has, err := m.permissionSvc.HasPermission(c, userID, perm)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": domain.ErrPermissionCheckFailed.Error()})
			return
		}
		if !has {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": domain.ErrPermissionDenied.Error()})
			return
		}
		c.Next()
	}
}

func (m *Middleware) requireAdminPermissionByAPIKey(c *gin.Context, rawKey string, perm string, permOK bool) {
	if strings.TrimSpace(rawKey) == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": domain.ErrMissingApiKey.Error()})
		return
	}
	if m.apiKeys == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": domain.ErrApiKeyDisabled.Error()})
		return
	}
	key, err := m.apiKeys.Validate(c, rawKey)
	if err != nil {
		status := http.StatusUnauthorized
		if errors.Is(err, appshared.ErrForbidden) {
			status = http.StatusForbidden
		}
		c.AbortWithStatusJSON(status, gin.H{"error": domain.ErrInvalidApiKey.Error()})
		return
	}

	c.Set("user_id", int64(0))
	c.Set("role", "admin")
	c.Set("api_key_id", key.ID)

	if m.permissionSvc == nil {
		c.Next()
		return
	}
	if key.PermissionGroupID == nil || *key.PermissionGroupID <= 0 {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": domain.ErrPermissionDenied.Error()})
		return
	}

	if !permOK {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": domain.ErrPermissionDenied.Error()})
		return
	}
	has, err := m.permissionSvc.HasPermissionForGroup(c, *key.PermissionGroupID, perm)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": domain.ErrPermissionCheckFailed.Error()})
		return
	}
	if !has {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": domain.ErrPermissionDenied.Error()})
		return
	}
	c.Next()
}

func (m *Middleware) RequireAPIKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := extractAPIKey(c)
		if key == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": domain.ErrMissingApiKey.Error()})
			return
		}
		if m.apiKeys == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": domain.ErrApiKeyDisabled.Error()})
			return
		}
		_, err := m.apiKeys.Validate(c, key)
		if err != nil {
			status := http.StatusUnauthorized
			if errors.Is(err, appshared.ErrForbidden) {
				status = http.StatusForbidden
			}
			c.AbortWithStatusJSON(status, gin.H{"error": domain.ErrInvalidApiKey.Error()})
			return
		}
		c.Next()
	}
}

func (m *Middleware) RequirePermission(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := m.parseToken(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidToken.Error()})
			return
		}
		role, _ := claims["role"].(string)
		if role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": domain.ErrAdminRequired.Error()})
			return
		}
		userID, ok := toInt64Claim(claims["user_id"])
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidToken.Error()})
			return
		}

		if m.permissionSvc == nil {
			c.Set("user_id", userID)
			c.Set("role", role)
			c.Next()
			return
		}

		if len(permissions) == 0 {
			c.Set("user_id", userID)
			c.Set("role", role)
			c.Next()
			return
		}

		for _, perm := range permissions {
			has, err := m.permissionSvc.HasPermission(c, userID, perm)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": domain.ErrPermissionCheckFailed.Error()})
				return
			}
			if has {
				c.Set("user_id", userID)
				c.Set("role", role)
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": domain.ErrPermissionDenied.Error()})
	}
}

func (m *Middleware) parseToken(c *gin.Context) (jwt.MapClaims, error) {
	auth := c.GetHeader("Authorization")
	var tokenStr string
	if strings.HasPrefix(auth, "Bearer ") {
		tokenStr = strings.TrimPrefix(auth, "Bearer ")
	} else {
		tokenStr = m.tokenFromQuery(c)
	}
	if tokenStr == "" {
		return nil, domain.ErrEmptyToken
	}
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
		if token.Method == nil || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, domain.ErrUnexpectedSigningMethod
		}
		return m.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, domain.ErrInvalidToken
	}
	if m.authSvc != nil {
		userID, ok := toInt64Claim(claims["user_id"])
		if ok && userID > 0 {
			user, userErr := m.authSvc.GetUser(c, userID)
			if userErr != nil {
				return nil, domain.ErrInvalidToken
			}
			if user.Role != domain.UserRoleAdmin && user.PasswordChangedAt != nil {
				iatSeconds, hasIAT := toFloat64Claim(claims["iat"])
				if !hasIAT || iatSeconds <= 0 {
					return nil, domain.ErrInvalidToken
				}
				changedAt := float64(user.PasswordChangedAt.UnixNano()) / 1e9
				if iatSeconds <= changedAt {
					return nil, domain.ErrInvalidToken
				}
			}
		}
	}
	return claims, nil
}

func (m *Middleware) tokenFromQuery(c *gin.Context) string {
	path := c.Request.URL.Path
	if path == "" {
		return ""
	}
	if !(strings.HasSuffix(path, "/panel") || strings.HasSuffix(path, "/vnc")) {
		return ""
	}
	token := c.Query("token")
	if token == "" {
		token = c.Query("access_token")
	}
	return token
}

func extractAPIKey(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return c.GetHeader("X-API-Key")
}

func looksLikeJWT(token string) bool {
	// JWT should have 2 dots: header.payload.signature
	return strings.Count(token, ".") >= 2
}

func (m *Middleware) isAdmin2FAAllowlisted(path string) bool {
	switch path {
	case "/admin/api/v1/auth/login",
		"/admin/api/v1/auth/refresh",
		"/admin/api/v1/auth/2fa/setup",
		"/admin/api/v1/auth/2fa/confirm",
		"/admin/api/v1/auth/2fa/unlock",
		"/admin/api/v1/avatar/qq/:qq":
		return true
	default:
		return false
	}
}

func (m *Middleware) enforceAdmin2FAGate(c *gin.Context, userID int64, claims jwt.MapClaims) {
	if m.settingsSvc == nil || m.authSvc == nil {
		return
	}

	auth2faEnabled := m.getSettingBool(c.Request.Context(), "auth_2fa_enabled", true)
	if !auth2faEnabled {
		return
	}

	mfa := 0
	if v, ok := toInt64Claim(claims["mfa"]); ok && v > 0 {
		mfa = 1
	}
	if mfa > 0 {
		return
	}

	user, err := m.authSvc.GetUser(c, userID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidToken.Error()})
		return
	}
	if !user.TOTPEnabled {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": domain.Err2faBindRequired.Error(), "code": "admin_2fa_bind_required"})
		return
	}
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": domain.Err2faRequired.Error(), "code": "admin_2fa_required"})
}

func (m *Middleware) getSettingValue(ctx context.Context, key string) string {
	if m.settingsSvc == nil {
		return ""
	}
	item, err := m.settingsSvc.Get(ctx, key)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(item.ValueJSON)
}

func (m *Middleware) getSettingBool(ctx context.Context, key string, def bool) bool {
	val := strings.ToLower(strings.TrimSpace(m.getSettingValue(ctx, key)))
	if val == "" {
		return def
	}
	return val == "true" || val == "1" || val == "yes"
}

func getUserID(c *gin.Context) int64 {
	val, _ := c.Get("user_id")
	id, _ := val.(int64)
	return id
}

func toInt64Claim(v any) (int64, bool) {
	switch t := v.(type) {
	case float64:
		return int64(t), true
	case int64:
		return t, true
	case json.Number:
		if i, err := t.Int64(); err == nil {
			return i, true
		}
	}
	return 0, false
}

func toFloat64Claim(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case int64:
		return float64(t), true
	case int:
		return float64(t), true
	case json.Number:
		if f, err := t.Float64(); err == nil {
			return f, true
		}
	case string:
		if f, err := strconv.ParseFloat(strings.TrimSpace(t), 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

func isAdminSelfPass(c *gin.Context, permission string, userID int64) bool {
	switch permission {
	case "profile.view", "profile.update", "profile.change_password":
		return true
	case "dashboard.overview", "dashboard.vps_status":
		return true
	case "permission_group.list", "permission.list", "permission.tree":
		if c.Request.Method == http.MethodGet {
			return true
		}
		return false
	case "admin.view", "admin.update":
		id := c.Param("id")
		if id == "" {
			return false
		}
		parsed, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return false
		}
		return parsed == userID
	default:
		return false
	}
}
