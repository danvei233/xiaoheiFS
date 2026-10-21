package http

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"xiaoheiplay/internal/domain"
)

type Server struct {
	Engine *gin.Engine
}

const (
	spaIndexCacheControl   = "no-store"
	spaDefaultCacheControl = "no-cache"
	spaAssetCacheControl   = "public, max-age=31536000, immutable"
)

func securityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	}
}

func NewServer(handler *Handler, middleware *Middleware) *Server {
	r := gin.Default()
	r.Use(securityHeadersMiddleware())
	r.Use(corsMiddleware())
	r.Static("/uploads", "./uploads")

	// Installer gate: before installation completes, redirect site traffic to /install and
	// block non-install API calls to avoid confusing errors.
	r.Use(installGateMiddleware(handler))

	// Serve built frontend assets from ./static (Vite/SPA).
	// - Public site assets are served from ./static.
	// - Admin SPA assets are served from ./static-admin when the request path matches admin_path.
	// - Otherwise, for non-API routes, fall back to the matching SPA index.html so routing works.
	adminPathResolver := func() string {
		if handler == nil {
			return ""
		}
		return GetAdminPathFromSettings(handler.settingsSvc)
	}
	r.Use(spaStaticFileMiddleware("./static", "./static-admin", adminPathResolver,
		[]string{"/api/", "/admin/api/", "/uploads/"},
	))
	r.NoRoute(spaIndexFallbackHandler("./static", "./static-admin", adminPathResolver,
		[]string{"/api/", "/admin/api/", "/uploads/"},
	))
	for _, registrar := range defaultRouteRegistrars() {
		registrar.Register(r, handler, middleware)
	}

	return &Server{Engine: r}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		if origin == "" || !isAllowedLocalOrigin(origin) {
			c.Next()
			return
		}

		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type,Accept,X-API-Key,X-API-Version,X-AKID,X-Timestamp,X-Nonce,X-Signature,Idempotency-Key")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == http.MethodOptions {
			c.Status(http.StatusNoContent)
			c.Abort()
			return
		}
		c.Next()
	}
}

func isAllowedLocalOrigin(origin string) bool {
	lower := strings.ToLower(origin)
	if strings.HasPrefix(lower, "http://localhost:") || strings.HasPrefix(lower, "https://localhost:") {
		return true
	}
	if strings.HasPrefix(lower, "http://127.0.0.1:") || strings.HasPrefix(lower, "https://127.0.0.1:") {
		return true
	}
	if strings.HasPrefix(lower, "http://[::1]:") || strings.HasPrefix(lower, "https://[::1]:") {
		return true
	}
	return false
}

func installGateMiddleware(handler *Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		if handler == nil || handler.IsInstalled() {
			c.Next()
			return
		}

		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/v1/install") {
			c.Next()
			return
		}
		if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/admin/api/") {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": domain.ErrNotInstalled.Error()})
			return
		}

		// Allow uploads and static assets so the installer page can load.
		if strings.HasPrefix(path, "/uploads/") || strings.HasPrefix(path, "/assets/") || path == "/favicon.ico" {
			c.Next()
			return
		}

		// Allow direct access to installer page itself.
		if path == "/install" {
			c.Redirect(http.StatusTemporaryRedirect, "/install/")
			c.Abort()
			return
		}
		if strings.HasPrefix(path, "/install/") {
			c.Next()
			return
		}

		// Browser navigation: redirect to /install/.
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead {
			c.Redirect(http.StatusFound, "/install/")
			c.Abort()
			return
		}

		c.AbortWithStatus(http.StatusNotFound)
	}
}

func spaStaticFileMiddleware(
	publicStaticDir string,
	adminStaticDir string,
	adminPathResolver func() string,
	excludedPrefixes []string,
) gin.HandlerFunc {

	return func(c *gin.Context) {
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.Next()
			return
		}

		reqPath := c.Request.URL.Path
		for _, p := range excludedPrefixes {
			if strings.HasPrefix(reqPath, p) {
				c.Next()
				return
			}
		}

		selectedDir, rel := resolveSPATarget(reqPath, publicStaticDir, adminStaticDir, adminPathResolver)
		if selectedDir == "" {
			c.Next()
			return
		}

		target := filepath.Join(selectedDir, filepath.FromSlash(rel))
		targetAbs, err := filepath.Abs(target)
		if err != nil {
			c.Next()
			return
		}
		targetAbs = filepath.Clean(targetAbs)

		staticAbs, staticAbsErr := filepath.Abs(selectedDir)
		staticAbs = filepath.Clean(staticAbs)

		// Basic path traversal guard: only serve files within the selected SPA dir.
		if staticAbsErr == nil {
			if targetAbs != staticAbs && !strings.HasPrefix(targetAbs, staticAbs+string(os.PathSeparator)) {
				c.Next()
				return
			}
		}

		st, err := os.Stat(targetAbs)
		if err != nil || st.IsDir() {
			c.Next()
			return
		}

		applySPACacheHeaders(c, rel, false)
		c.File(targetAbs)
		c.Abort()
	}
}

func spaIndexFallbackHandler(
	publicStaticDir string,
	adminStaticDir string,
	adminPathResolver func() string,
	excludedPrefixes []string,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqPath := c.Request.URL.Path
		for _, p := range excludedPrefixes {
			if strings.HasPrefix(reqPath, p) {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": domain.ErrNotFound.Error()})
				return
			}
		}

		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		adminPath := ""
		if adminPathResolver != nil {
			adminPath = normalizeAdminRequestPath(adminPathResolver())
		}
		if adminPath != "" && dirExists(adminStaticDir) {
			if redirectTarget := buildAdminSPARedirectTarget(reqPath, adminPath, c.Request.URL.RawQuery); redirectTarget != "" {
				c.Redirect(http.StatusTemporaryRedirect, redirectTarget)
				return
			}
		}

		selectedDir, _ := resolveSPATarget(reqPath, publicStaticDir, adminStaticDir, adminPathResolver)
		indexPath := filepath.Join(selectedDir, "index.html")

		if _, err := os.Stat(indexPath); err == nil {
			applySPACacheHeaders(c, "index.html", true)
			c.File(indexPath)
			return
		}
		c.AbortWithStatus(http.StatusNotFound)
	}
}

func resolveSPATarget(
	reqPath string,
	publicStaticDir string,
	adminStaticDir string,
	adminPathResolver func() string,
) (string, string) {
	if isInstallerSPAPrefix(reqPath) && dirExists(adminStaticDir) {
		return adminStaticDir, trimInstallerSPAPrefix(reqPath)
	}

	adminPath := ""
	if adminPathResolver != nil {
		adminPath = normalizeAdminRequestPath(adminPathResolver())
	}

	if adminPath != "" && hasAdminSPAPrefix(reqPath, adminPath) && dirExists(adminStaticDir) {
		adminPrefix := "/" + adminPath
		rel := strings.TrimPrefix(reqPath, adminPrefix)
		rel = strings.TrimPrefix(rel, "/")
		return adminStaticDir, rel
	}

	return publicStaticDir, strings.TrimPrefix(reqPath, "/")
}

func isInstallerSPAPrefix(reqPath string) bool {
	return reqPath == "/install" || reqPath == "/install/" || strings.HasPrefix(reqPath, "/install/")
}

func trimInstallerSPAPrefix(reqPath string) string {
	rel := strings.TrimPrefix(reqPath, "/install")
	return strings.TrimPrefix(rel, "/")
}

func normalizeAdminRequestPath(path string) string {
	return strings.Trim(strings.TrimSpace(path), "/")
}

func hasAdminSPAPrefix(reqPath string, adminPath string) bool {
	if adminPath == "" {
		return false
	}

	adminPrefix := "/" + adminPath
	return reqPath == adminPrefix || strings.HasPrefix(reqPath, adminPrefix+"/")
}

func dirExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}

	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func buildAdminSPARedirectTarget(reqPath string, adminPath string, rawQuery string) string {
	adminPrefix := "/" + adminPath
	switch {
	case reqPath == adminPrefix:
		return appendQuery(adminPrefix+"/", rawQuery)
	case !strings.HasPrefix(reqPath, adminPrefix+"/"):
		return ""
	}

	rel := strings.TrimPrefix(reqPath, adminPrefix)
	if rel == "/" || looksLikeStaticAssetPath(rel) {
		return ""
	}

	target := adminPrefix + "/#" + rel
	if rawQuery != "" {
		target += "?" + rawQuery
	}
	return target
}

func looksLikeStaticAssetPath(rel string) bool {
	trimmed := strings.TrimPrefix(rel, "/")
	if trimmed == "" {
		return false
	}
	if strings.HasPrefix(trimmed, "assets/") {
		return true
	}
	return strings.Contains(filepath.Base(trimmed), ".")
}

func appendQuery(path string, rawQuery string) string {
	if rawQuery == "" {
		return path
	}
	return path + "?" + rawQuery
}

func applySPACacheHeaders(c *gin.Context, rel string, isIndex bool) {
	if isIndex {
		c.Header("Cache-Control", spaIndexCacheControl)
		return
	}

	if isImmutableAssetPath(rel) {
		c.Header("Cache-Control", spaAssetCacheControl)
		return
	}

	c.Header("Cache-Control", spaDefaultCacheControl)
}

func isImmutableAssetPath(rel string) bool {
	trimmed := strings.TrimPrefix(rel, "/")
	if !strings.HasPrefix(trimmed, "assets/") {
		return false
	}

	base := filepath.Base(trimmed)
	if base == "" {
		return false
	}

	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	if ext == ".gz" {
		innerExt := filepath.Ext(name)
		name = strings.TrimSuffix(name, innerExt)
		ext = innerExt
	}

	if ext == "" {
		return false
	}

	dash := strings.LastIndex(name, "-")
	return dash > 0 && dash < len(name)-1
}
