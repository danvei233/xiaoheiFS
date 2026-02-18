package http

import "github.com/gin-gonic/gin"

type adminPublicAuthRoutesRegistrar struct{}

func (adminPublicAuthRoutesRegistrar) Register(r *gin.Engine, handler *Handler, _ *Middleware) {
	public := r.Group("/api/v1")
	public.POST("/auth/forgot-password", handler.AdminForgotPassword)
	public.POST("/auth/reset-password", handler.AdminResetPassword)
}
