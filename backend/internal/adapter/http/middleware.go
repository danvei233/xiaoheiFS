package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	appapikey "xiaoheiplay/internal/app/apikey"
	apppermission "xiaoheiplay/internal/app/permission"
	appshared "xiaoheiplay/internal/app/shared"
	appuserapikey "xiaoheiplay/internal/app/userapikey"
	"xiaoheiplay/internal/domain"
	"xiaoheiplay/internal/pkg/permissions"
)

const maxSignedRequestBodyBytes int64 = 1 << 20

// signedAPIKeyRateWindow / RateLimit bound how many signed requests a single
// API key may issue per window. The count is derived from the durable nonce
// ledger, so it holds across instances and restarts (P1-10).
const (
	signedAPIKeyRateWindow = time.Minute
	signedAPIKeyRateLimit  = 60
)

type Middleware struct {
	jwtSecret          []byte
	apiKeys            *appapikey.Service
	userAPIKeys        *appuserapikey.Service
	permissionSvc      *apppermission.Service
	authSvc            AuthService
	settingsSvc        SettingsService
	impersonationAudit ImpersonationAuditService
}

func NewMiddleware(jwtSecret string, apiKeys *appapikey.Service, userAPIKeys *appuserapikey.Service, permissionSvc *apppermission.Service, authSvc AuthService, settingsSvc SettingsService, impersonationAudit ...ImpersonationAuditService) *Middleware {
	middleware := &Middleware{
		jwtSecret:     []byte(jwtSecret),
		apiKeys:       apiKeys,
		userAPIKeys:   userAPIKeys,
		permissionSvc: permissionSvc,
		authSvc:       authSvc,
		settingsSvc:   settingsSvc,
	}
	if len(impersonationAudit) > 0 {
		middleware.impersonationAudit = impersonationAudit[0]
	}
	return middleware
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
		role, _ := claims["role"].(string)
		if role != string(domain.UserRoleUser) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": domain.ErrForbidden.Error()})
			return
		}
		if impersonatorID := impersonatorIDFromClaims(claims); impersonatorID > 0 {
			if m.impersonationAudit == nil || m.impersonationAudit.AuditRequired(
				c.Request.Context(), impersonatorID, "admin.impersonated_request", "http_request",
				strconv.FormatInt(userID, 10), map[string]any{"method": c.Request.Method, "path": c.FullPath()},
			) != nil {
				c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": domain.ErrAuditWriteFailed.Error()})
				return
			}
		}
		c.Set("user_id", userID)
		c.Set("role", role)
		setTokenFamilyContext(c, claims)
		c.Set("actor_mode", string(domain.RequestActorModeUserJWT))
		c.Set("request_actor", domain.RequestActor{
			Mode:           domain.RequestActorModeUserJWT,
			UserID:         userID,
			Role:           role,
			ImpersonatorID: impersonatorIDFromClaims(claims),
		})
		c.Next()
	}
}

// impersonatorIDFromClaims extracts the `imp` claim (admin id that issued an
// impersonation token) when present; zero otherwise.
func impersonatorIDFromClaims(claims jwt.MapClaims) int64 {
	imp, _ := toInt64Claim(claims["imp"])
	if imp <= 0 {
		return 0
	}
	return imp
}

func setTokenFamilyContext(c *gin.Context, claims jwt.MapClaims) {
	familyID, _ := claims["fid"].(string)
	if familyID = strings.TrimSpace(familyID); familyID != "" {
		c.Set("token_family_id", familyID)
	}
}

// OptionalUser identifies the caller when a valid user token is present and
// otherwise lets the request through anonymously.
//
// It exists for the storefront: the catalog is browsable without an account, but
// a signed-in visitor must still see their tier pricing. Handlers read the caller
// with getUserID, which returns 0 for guests, so they need no special casing.
//
// Anything that mutates state or exposes per-user data must keep using
// RequireUser — this middleware never rejects.
func (m *Middleware) OptionalUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := m.parseToken(c)
		if err != nil {
			c.Next()
			return
		}
		userID, ok := toInt64Claim(claims["user_id"])
		if !ok {
			c.Next()
			return
		}
		role, _ := claims["role"].(string)
		if role != string(domain.UserRoleUser) {
			c.Next()
			return
		}
		if impersonatorID := impersonatorIDFromClaims(claims); impersonatorID > 0 {
			if m.impersonationAudit == nil || m.impersonationAudit.AuditRequired(
				c.Request.Context(), impersonatorID, "admin.impersonated_request", "http_request",
				strconv.FormatInt(userID, 10), map[string]any{"method": c.Request.Method, "path": c.FullPath()},
			) != nil {
				// Optional auth remains optional for ordinary invalid credentials, but
				// a valid impersonation capability must never become unaudited. Fail
				// closed instead of silently downgrading it to a guest request.
				c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": domain.ErrAuditWriteFailed.Error()})
				return
			}
		}
		c.Set("user_id", userID)
		c.Set("role", role)
		setTokenFamilyContext(c, claims)
		c.Set("actor_mode", string(domain.RequestActorModeUserJWT))
		c.Set("request_actor", domain.RequestActor{
			Mode:           domain.RequestActorModeUserJWT,
			UserID:         userID,
			Role:           role,
			ImpersonatorID: impersonatorIDFromClaims(claims),
		})
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
		setTokenFamilyContext(c, claims)
		c.Set("actor_mode", string(domain.RequestActorModeAdminJWT))
		c.Set("request_actor", domain.RequestActor{
			Mode:   domain.RequestActorModeAdminJWT,
			UserID: userID,
			Role:   "admin",
		})
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
		setTokenFamilyContext(c, claims)
		c.Set("actor_mode", string(domain.RequestActorModeAdminJWT))
		c.Set("request_actor", domain.RequestActor{
			Mode:   domain.RequestActorModeAdminJWT,
			UserID: userID,
			Role:   "admin",
		})

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
		isPrimary, err := m.permissionSvc.IsPrimaryAdmin(c.Request.Context(), userID)
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
		has, err := m.permissionSvc.HasPermission(c.Request.Context(), userID, perm)
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
	key, err := m.apiKeys.Validate(c.Request.Context(), rawKey)
	if err != nil {
		status := http.StatusUnauthorized
		if errors.Is(err, appshared.ErrForbidden) {
			status = http.StatusForbidden
		}
		c.AbortWithStatusJSON(status, gin.H{"error": domain.ErrInvalidApiKey.Error()})
		return
	}

	// The audit actor for an admin API-key request is the derived negative id so
	// audit logs are attributable (type+id) instead of silently collapsing to
	// admin_id=0 (Issue #51 P2-22). The encoding keeps API-key actors disjoint
	// from admin/user ids (>0) and from the robot webhook sentinel (-1).
	actorID := domain.AdminAPIKeyActorID(key.ID)
	c.Set("user_id", actorID)
	c.Set("role", "admin")
	c.Set("api_key_id", key.ID)
	c.Set("actor_mode", string(domain.RequestActorModeAdminAPIKey))
	c.Set("request_actor", domain.RequestActor{
		Mode:          domain.RequestActorModeAdminAPIKey,
		UserID:        actorID,
		Role:          "admin",
		AdminAPIKeyID: key.ID,
	})

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
	has, err := m.permissionSvc.HasPermissionForGroup(c.Request.Context(), *key.PermissionGroupID, perm)
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
		_, err := m.apiKeys.Validate(c.Request.Context(), key)
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

func (m *Middleware) RequireUserAPIKeySigned() gin.HandlerFunc {
	return func(c *gin.Context) {
		if m.userAPIKeys == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": domain.ErrApiKeyDisabled.Error()})
			return
		}
		akid := strings.TrimSpace(c.GetHeader("X-AKID"))
		tsRaw := strings.TrimSpace(c.GetHeader("X-Timestamp"))
		nonce := strings.TrimSpace(c.GetHeader("X-Nonce"))
		signature := strings.TrimSpace(c.GetHeader("X-Signature"))
		if akid == "" || tsRaw == "" || nonce == "" || signature == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidSignature.Error()})
			return
		}
		if _, err := appuserapikey.ParseAndCheckTimestamp(tsRaw, time.Now(), appuserapikey.DefaultSignatureWindowSec); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidSignature.Error()})
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSignedRequestBodyBytes)
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			var maxBytesErr *http.MaxBytesError
			if errors.As(err, &maxBytesErr) {
				c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": domain.ErrRequestBodyTooLarge.Error()})
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidSignature.Error()})
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
		path := c.Request.URL.Path
		canonical := appuserapikey.BuildCanonical(c.Request.Method, path, c.Request.URL.RawQuery, tsRaw, nonce, body)
		key, err := m.userAPIKeys.ValidateSignature(c.Request.Context(), akid, tsRaw, nonce, signature, canonical)
		if err != nil {
			status := http.StatusUnauthorized
			if errors.Is(err, appshared.ErrForbidden) {
				status = http.StatusForbidden
			}
			c.AbortWithStatusJSON(status, gin.H{"error": domain.ErrInvalidSignature.Error()})
			return
		}

		// Enforce the route's required scope (P0-7).
		routePath := c.FullPath()
		if routePath == "" {
			routePath = path
		}
		required, ok := appuserapikey.RequiredScope(c.Request.Method, routePath)
		if !ok || !appuserapikey.HasScope(key, required) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": domain.ErrPermissionDenied.Error()})
			return
		}

		// Durable, multi-instance replay protection and request budget.
		claimed, count, err := m.userAPIKeys.ConsumeNonce(c.Request.Context(), akid, nonce, time.Now(), signedAPIKeyRateWindow, signedAPIKeyRateLimit)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": domain.ErrSignatureCheckFailed.Error()})
			return
		}
		if !claimed {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": domain.ErrInvalidSignature.Error()})
			return
		}
		if count > signedAPIKeyRateLimit {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": domain.ErrTooManyRequests.Error()})
			return
		}
		_ = m.userAPIKeys.RecordUse(c.Request.Context(), key.ID)
		c.Set("user_id", key.UserID)
		c.Set("role", "user")
		c.Set("user_api_key_id", key.ID)
		c.Set("actor_mode", string(domain.RequestActorModeUserAPIKey))
		c.Set("request_actor", domain.RequestActor{
			Mode:         domain.RequestActorModeUserAPIKey,
			UserID:       key.UserID,
			Role:         "user",
			UserAPIKeyID: key.ID,
		})
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
			has, err := m.permissionSvc.HasPermission(c.Request.Context(), userID, perm)
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
	}
	if tokenStr == "" {
		return nil, domain.ErrEmptyToken
	}
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims,
		func(token *jwt.Token) (any, error) {
			if token.Method == nil || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, domain.ErrUnexpectedSigningMethod
			}
			return m.jwtSecret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(jwtIssuer),
		jwt.WithAudience(jwtAudience),
	)
	if err != nil || !token.Valid {
		return nil, domain.ErrInvalidToken
	}
	tokenType, _ := claims["type"].(string)
	if tokenType != "access" {
		return nil, domain.ErrInvalidTokenType
	}
	if err := validateCurrentAuthClaims(c.Request.Context(), m.authSvc, claims); err != nil {
		return nil, err
	}
	if purpose, _ := claims["purpose"].(string); purpose == "impersonation" {
		impersonatorID := impersonatorIDFromClaims(claims)
		if m.permissionSvc == nil || impersonatorID <= 0 {
			return nil, domain.ErrInvalidToken
		}
		primary, err := m.permissionSvc.IsPrimaryAdmin(c.Request.Context(), impersonatorID)
		if err != nil || !primary {
			return nil, domain.ErrInvalidToken
		}
	}
	return claims, nil
}

func validateCurrentAuthClaims(ctx context.Context, authSvc AuthService, claims jwt.MapClaims) error {
	if authSvc == nil {
		return nil
	}
	userID, ok := toInt64Claim(claims["user_id"])
	if !ok || userID <= 0 {
		return domain.ErrInvalidToken
	}
	role, ok := claims["role"].(string)
	if !ok || role == "" {
		return domain.ErrInvalidToken
	}
	user, err := authSvc.GetUser(ctx, userID)
	if err != nil || user.Status != domain.UserStatusActive || role != string(user.Role) {
		return domain.ErrInvalidToken
	}
	if tokenIssuedBeforePasswordChange(user, claims) {
		return domain.ErrInvalidToken
	}
	// Impersonation is a delegated capability, not a permanent user token. The
	// issuing administrator must still be an active primary admin on every
	// request; revocation therefore takes effect immediately for already-issued
	// 24-hour impersonation tokens.
	if purpose, _ := claims["purpose"].(string); purpose == "impersonation" {
		impersonatorID := impersonatorIDFromClaims(claims)
		if impersonatorID <= 0 {
			return domain.ErrInvalidToken
		}
		operator, err := authSvc.GetUser(ctx, impersonatorID)
		if err != nil || operator.Status != domain.UserStatusActive || operator.Role != domain.UserRoleAdmin {
			return domain.ErrInvalidToken
		}
		issuedOperatorPVV, ok := toInt64Claim(claims["imp_pvv"])
		if !ok || issuedOperatorPVV != operator.PasswordVersion {
			return domain.ErrInvalidToken
		}
	}
	return nil
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
		"/admin/api/v1/auth/2fa/unlock":
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

	user, err := m.authSvc.GetUser(c.Request.Context(), userID)
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
