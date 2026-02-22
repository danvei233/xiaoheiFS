package http

import "github.com/gin-gonic/gin"

// RouteRegistrar owns one route module and registers it to the engine.
type RouteRegistrar interface {
	Register(r *gin.Engine, handler *Handler, middleware *Middleware)
}

func defaultRouteRegistrars() []RouteRegistrar {
	return []RouteRegistrar{
		publicRoutesRegistrar{},
		userRoutesRegistrar{},
		openRoutesRegistrar{},
		integrationRoutesRegistrar{},
		adminRoutesRegistrar{},
		adminPublicAuthRoutesRegistrar{},
	}
}
