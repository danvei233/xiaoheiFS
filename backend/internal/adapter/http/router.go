package http

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

type Server struct {
	Engine *gin.Engine
}

func NewServer(handler *Handler, middleware *Middleware) *Server {
	r := gin.Default()
	r.Use(corsMiddleware())
	r.Static("/uploads", "./uploads")

	// Installer gate: before installation completes, redirect site traffic to /install and
	// block non-install API calls to avoid confusing errors.
	r.Use(installGateMiddleware(handler))

	// Serve built frontend assets from ./static (Vite/SPA).
	// - If a real file exists under ./static, serve it (e.g. /assets/*, /favicon.ico).
	// - Otherwise, for non-API routes, fall back to ./static/index.html so history-mode routing works.
	r.Use(spaStaticFileMiddleware("./static",
		[]string{"/api/", "/admin/api/", "/uploads/"},
	))
	r.NoRoute(spaIndexFallbackHandler("./static",
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
		c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type,Accept,X-API-Key,X-API-Version")
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
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "not installed"})
			return
		}

		// Allow uploads and static assets so the installer page can load.
		if strings.HasPrefix(path, "/uploads/") || strings.HasPrefix(path, "/assets/") || path == "/favicon.ico" {
			c.Next()
			return
		}

		// Allow direct access to installer page itself.
		if path == "/install" || strings.HasPrefix(path, "/install/") {
			c.Next()
			return
		}

		// Browser navigation: redirect to /install.
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead {
			c.Redirect(http.StatusFound, "/install")
			c.Abort()
			return
		}

		c.AbortWithStatus(http.StatusNotFound)
	}
}

func spaStaticFileMiddleware(staticDir string, excludedPrefixes []string) gin.HandlerFunc {
	staticAbs, staticAbsErr := filepath.Abs(staticDir)
	staticAbs = filepath.Clean(staticAbs)

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

		// Map URL path -> filesystem path under staticDir.
		rel := strings.TrimPrefix(reqPath, "/")
		target := filepath.Join(staticDir, filepath.FromSlash(rel))
		targetAbs, err := filepath.Abs(target)
		if err != nil {
			c.Next()
			return
		}
		targetAbs = filepath.Clean(targetAbs)

		// Basic path traversal guard: only serve files within staticDir.
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

		c.File(targetAbs)
		c.Abort()
	}
}

func spaIndexFallbackHandler(staticDir string, excludedPrefixes []string) gin.HandlerFunc {
	indexPath := filepath.Join(staticDir, "index.html")

	return func(c *gin.Context) {
		reqPath := c.Request.URL.Path
		for _, p := range excludedPrefixes {
			if strings.HasPrefix(reqPath, p) {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
		}

		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		if _, err := os.Stat(indexPath); err == nil {
			c.File(indexPath)
			return
		}
		c.AbortWithStatus(http.StatusNotFound)
	}
}
