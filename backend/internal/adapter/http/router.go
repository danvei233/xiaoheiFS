package http

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net"
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

// defaultTrustedProxiesEnv is the env var operators can set to declare the
// CIDRs of reverse proxies that may supply X-Forwarded-For. When unset or
// invalid, no proxy is trusted and gin resolves ClientIP from the direct peer
// address — otherwise every IP-keyed rate limiter and cooldown guard would be
// bypassable by spoofing the header (Issue #51 P0-1).
const defaultTrustedProxiesEnv = "TRUSTED_PROXIES"

// ConfigureTrustedProxies applies a fail-closed trusted-proxy policy to r,
// reading the operator-provided list from TRUSTED_PROXIES. An empty or unset
// value trusts no proxies.
func ConfigureTrustedProxies(r *gin.Engine) error {
	raw := strings.TrimSpace(os.Getenv(defaultTrustedProxiesEnv))
	if raw == "" {
		return r.SetTrustedProxies(nil)
	}
	var cidrs []string
	for _, part := range strings.Split(raw, ",") {
		if p := strings.TrimSpace(part); p != "" {
			cidrs = append(cidrs, p)
		}
	}
	return ConfigureTrustedProxiesWith(r, cidrs)
}

// ConfigureTrustedProxiesWith validates cidrs and installs them on r. Any
// invalid entry is an error and leaves the engine untrusted (fail closed).
func ConfigureTrustedProxiesWith(r *gin.Engine, cidrs []string) error {
	if len(cidrs) == 0 {
		return r.SetTrustedProxies(nil)
	}
	validated := make([]string, 0, len(cidrs))
	for _, cidr := range cidrs {
		trimmed := strings.TrimSpace(cidr)
		_, _, err := net.ParseCIDR(trimmed)
		if err != nil {
			// Fail closed: one malformed entry must not revert us to the
			// insecure trust-everything default.
			_ = r.SetTrustedProxies(nil)
			return fmt.Errorf("invalid trusted proxy CIDR %q: %w", cidr, err)
		}
		validated = append(validated, trimmed)
	}
	return r.SetTrustedProxies(validated)
}

// globalMaxBodyBytes bounds request bodies globally (P3-5). Streaming
// responses are GETs and carry no body; handlers that need a tighter cap (e.g.
// uploads, webhooks, signed requests) wrap the body again with their own
// MaxBytesReader and map overflow to a stable 413.
const globalMaxBodyBytes = 32 << 20

func bodySizeLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			// A declared Content-Length over the cap is rejected up front with a
			// stable 413 before the handler reads anything, so oversized JSON is
			// never partially parsed or logged. Chunked/unknown-length bodies are
			// enforced by MaxBytesReader during the read.
			if c.Request.ContentLength > globalMaxBodyBytes {
				c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": domain.ErrRequestBodyTooLarge.Error()})
				return
			}
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, globalMaxBodyBytes)
		}
		c.Next()
	}
}

// newGinEngine builds the router with gin's default middleware and trusts no
// proxy: gin's ClientIP() then returns the direct peer address, so an
// attacker-supplied X-Forwarded-For cannot spoof the IP that keys rate-limit
// budgets or callback URLs. Deployments behind a real reverse proxy must pin
// the trusted proxy CIDRs explicitly.
func newGinEngine() *gin.Engine {
	r := gin.Default()
	// gin's default TrustedProxies is "0.0.0.0/0","::/0", i.e. it trusts any
	// X-Forwarded-For header. That lets an unauthenticated client spoof the peer
	// address used by every c.ClientIP()-keyed rate limiter (installer, login,
	// password reset, admin path oracle) and rotate it per request to bypass the
	// limit. Trust no proxy by default so ClientIP() is the real TCP peer, which
	// cannot be forged. Deployments behind a reverse proxy that needs
	// X-Forwarded-For should call SetTrustedProxies with the proxy's address.
	if err := r.SetTrustedProxies(nil); err != nil {
		// SetTrustedProxies only errors when given nil for a nil slice is a no-op
		// in gin; treat any unexpected error as fatal rather than silently serving
		// with the spoofable default.
		panic(err)
	}
	return r
}

func NewServer(handler *Handler, middleware *Middleware) *Server {
	r := newGinEngine()
	if err := ConfigureTrustedProxies(r); err != nil {
		// Never fail to boot over a proxy config error — but the engine is
		// already left in the fail-closed (trust-none) state by the helper.
		log.Printf("warning: trusted proxy configuration rejected: %v", err)
	}
	r.Use(securityHeadersMiddleware())
	r.Use(bodySizeLimitMiddleware())
	r.Use(corsMiddleware())
	r.Static("/uploads", "./uploads")

	// Installer gate: before installation completes, redirect site traffic to /install and
	// block non-install API calls to avoid confusing errors.
	r.Use(installGateMiddleware(handler))

	// Serve the built frontend from ./static.
	//
	// The web tier is a single history-mode SPA carrying the public site, the auth
	// pages, the user console at /console, the installer at /install and the admin
	// console at the operator-configured admin_path. Requests that resolve to a file
	// are served as-is; every other non-API GET falls back to index.html and the
	// client router decides what to render, including which admin path is valid.
	spaExcludedPrefixes := []string{"/api/", "/admin/api/", "/uploads/"}
	r.Use(spaStaticFileMiddleware("./static", spaExcludedPrefixes))
	r.NoRoute(spaIndexFallbackHandler("./static", spaExcludedPrefixes))
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

func spaStaticFileMiddleware(staticDir string, excludedPrefixes []string) gin.HandlerFunc {
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

		selectedDir := staticDir
		rel := strings.TrimPrefix(reqPath, "/")

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

func spaIndexFallbackHandler(staticDir string, excludedPrefixes []string) gin.HandlerFunc {
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

		indexPath := filepath.Join(staticDir, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			applySPACacheHeaders(c, "index.html", true)
			c.File(indexPath)
			return
		}
		c.AbortWithStatus(http.StatusNotFound)
	}
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
