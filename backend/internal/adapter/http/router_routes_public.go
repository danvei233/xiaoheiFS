package http

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"xiaoheiplay/internal/domain"
)

// notifyRateLimit applies a durable per-IP budget shared by every application
// replica. Payment gateways retry callbacks, so a process-local counter that
// resets on restart does not provide a meaningful flood boundary.
func (h *Handler) notifyRateLimit(c *gin.Context) {
	if h.settingsSvc == nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": domain.ErrServiceUnavailable.Error()})
		return
	}
	allowed, err := h.settingsSvc.AllowRateLimit(c.Request.Context(), "notify:"+strings.TrimSpace(c.ClientIP()), 600, 10*time.Minute)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": domain.ErrServiceUnavailable.Error()})
		return
	}
	if !allowed {
		c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": domain.ErrTooManyRequests.Error()})
		return
	}
	c.Next()
}

type publicRoutesRegistrar struct{}

func (publicRoutesRegistrar) Register(r *gin.Engine, handler *Handler, _ *Middleware) {
	public := r.Group("/api/v1")
	{
		public.GET("/install/status", handler.InstallStatus)
		public.POST("/install/db/check", handler.InstallDBCheck)
		public.POST("/install", handler.InstallRun)
		public.GET("/install/generate-admin-path", handler.InstallGenerateAdminPath)
		public.POST("/install/validate-admin-path", handler.ValidateAdminPathHandler)
		public.POST("/check-admin-path", handler.CheckAdminPath)

		public.GET("/captcha", handler.Captcha)
		public.GET("/auth/settings", handler.AuthSettings)
		public.POST("/auth/register/code", handler.RegisterCode)
		public.POST("/auth/register", handler.Register)
		public.POST("/auth/login", handler.Login)
		public.POST("/auth/password-reset/options", handler.PasswordResetOptions)
		public.POST("/auth/password-reset/send-code", handler.PasswordResetSendCode)
		public.POST("/auth/password-reset/verify-code", handler.PasswordResetVerifyCode)
		public.POST("/auth/password-reset/confirm", handler.PasswordResetConfirm)
		public.POST("/auth/refresh", handler.Refresh)
		public.Any("/payments/notify/:provider", handler.notifyRateLimit, handler.PaymentNotify)
		public.Any("/wallet/payments/notify/:provider", handler.notifyRateLimit, handler.WalletPaymentNotify)
		public.GET("/site/settings", handler.SiteSettings)
		public.GET("/cms/blocks", handler.CMSBlocksPublic)
		public.GET("/cms/posts", handler.CMSPostsPublic)
		public.GET("/cms/posts/:slug", handler.CMSPostDetailPublic)
		public.POST("/probe/enroll", handler.ProbeEnroll)
		public.POST("/probe/auth/token", handler.ProbeAuthToken)
		public.GET("/probe/ws", handler.ProbeWS)
	}
}
