package http

import "github.com/gin-gonic/gin"

type publicRoutesRegistrar struct{}

func (publicRoutesRegistrar) Register(r *gin.Engine, handler *Handler, _ *Middleware) {
	public := r.Group("/api/v1")
	{
		public.GET("/install/status", handler.InstallStatus)
		public.POST("/install/db/check", handler.InstallDBCheck)
		public.POST("/install", handler.InstallRun)

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
		public.Any("/payments/notify/:provider", handler.PaymentNotify)
		public.GET("/site/settings", handler.SiteSettings)
		public.GET("/cms/blocks", handler.CMSBlocksPublic)
		public.GET("/cms/posts", handler.CMSPostsPublic)
		public.GET("/cms/posts/:slug", handler.CMSPostDetailPublic)
		public.POST("/probe/enroll", handler.ProbeEnroll)
		public.POST("/probe/auth/token", handler.ProbeAuthToken)
		public.GET("/probe/ws", handler.ProbeWS)
	}
}
