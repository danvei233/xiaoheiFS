package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

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
}

func NewMiddleware(jwtSecret string, apiKeys *appapikey.Service, permissionSvc *apppermission.Service) *Middleware {
	return &Middleware{jwtSecret: []byte(jwtSecret), apiKeys: apiKeys, permissionSvc: permissionSvc}
}

func (m *Middleware) RequireUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := m.parseToken(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		userID, ok := toInt64Claim(claims["user_id"])
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
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
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		role, _ := claims["role"].(string)
		if role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin required"})
			return
		}
		userID, ok := toInt64Claim(claims["user_id"])
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set("user_id", userID)
		c.Set("role", role)
		c.Next()
	}
}

func (m *Middleware) RequireAdminPermissionAuto() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if strings.HasPrefix(auth, "Bearer ") {
			token := strings.TrimPrefix(auth, "Bearer ")
			if !looksLikeJWT(token) {
				m.requireAdminPermissionByAPIKey(c, token)
				return
			}
		} else if key := c.GetHeader("X-API-Key"); key != "" {
			m.requireAdminPermissionByAPIKey(c, key)
			return
		}

		claims, err := m.parseToken(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		role, _ := claims["role"].(string)
		if role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin required"})
			return
		}
		userID, ok := toInt64Claim(claims["user_id"])
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set("user_id", userID)
		c.Set("role", role)

		if m.permissionSvc == nil {
			c.Next()
			return
		}
		isPrimary, err := m.permissionSvc.IsPrimaryAdmin(c, userID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "permission check failed"})
			return
		}
		if isPrimary {
			c.Next()
			return
		}

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		perm, ok := permissions.InferPermissionCode(c.Request.Method, path)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "permission denied"})
			return
		}
		if isAdminSelfPass(c, perm, userID) {
			c.Next()
			return
		}
		has, err := m.permissionSvc.HasPermission(c, userID, perm)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "permission check failed"})
			return
		}
		if !has {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "permission denied"})
			return
		}
		c.Next()
	}
}

func (m *Middleware) requireAdminPermissionByAPIKey(c *gin.Context, rawKey string) {
	if strings.TrimSpace(rawKey) == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing api key"})
		return
	}
	if m.apiKeys == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "api key disabled"})
		return
	}
	key, err := m.apiKeys.Validate(c, rawKey)
	if err != nil {
		status := http.StatusUnauthorized
		if errors.Is(err, appshared.ErrForbidden) {
			status = http.StatusForbidden
		}
		c.AbortWithStatusJSON(status, gin.H{"error": "invalid api key"})
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
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	path := c.FullPath()
	if path == "" {
		path = c.Request.URL.Path
	}
	perm, ok := permissions.InferPermissionCode(c.Request.Method, path)
	if !ok {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}
	has, err := m.permissionSvc.HasPermissionForGroup(c, *key.PermissionGroupID, perm)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "permission check failed"})
		return
	}
	if !has {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}
	c.Next()
}

func (m *Middleware) RequireAPIKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := extractAPIKey(c)
		if key == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing api key"})
			return
		}
		if m.apiKeys == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "api key disabled"})
			return
		}
		_, err := m.apiKeys.Validate(c, key)
		if err != nil {
			status := http.StatusUnauthorized
			if errors.Is(err, appshared.ErrForbidden) {
				status = http.StatusForbidden
			}
			c.AbortWithStatusJSON(status, gin.H{"error": "invalid api key"})
			return
		}
		c.Next()
	}
}

func (m *Middleware) RequirePermission(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := m.parseToken(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		role, _ := claims["role"].(string)
		if role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin required"})
			return
		}
		userID, ok := toInt64Claim(claims["user_id"])
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
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
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "permission check failed"})
				return
			}
			if has {
				c.Set("user_id", userID)
				c.Set("role", role)
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "permission denied"})
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
