package http

import "github.com/gin-gonic/gin"

type integrationRoutesRegistrar struct{}

func (integrationRoutesRegistrar) Register(r *gin.Engine, handler *Handler, middleware *Middleware) {
	integration := r.Group("/api/v1/integrations")
	integration.Use(middleware.RequireAPIKey())
	{
		integration.POST("/robot/webhook", handler.RobotWebhook)
	}
}
