package http

import "github.com/gin-gonic/gin"

type openRoutesRegistrar struct{}

func (openRoutesRegistrar) Register(r *gin.Engine, handler *Handler, middleware *Middleware) {
	openJWT := r.Group("/api/v1/open")
	openJWT.Use(middleware.RequireUser())
	{
		openJWT.GET("/me/api-keys", handler.OpenUserAPIKeys)
		openJWT.POST("/me/api-keys", handler.OpenUserAPIKeyCreate)
		openJWT.PATCH("/me/api-keys/:id", handler.OpenUserAPIKeyPatch)
		openJWT.DELETE("/me/api-keys/:id", handler.OpenUserAPIKeyDelete)
	}

	openSigned := r.Group("/api/v1/open")
	openSigned.Use(middleware.RequireUserAPIKeySigned())
	{
		openSigned.POST("/orders/instant/create", handler.OpenInstantOrderCreate)
		openSigned.POST("/orders/instant/renew", handler.OpenInstantOrderRenew)
		openSigned.POST("/orders/instant/resize", handler.OpenInstantOrderResize)
		openSigned.POST("/orders/instant/refund", handler.OpenInstantOrderRefund)

		openSigned.GET("/vps", handler.VPSList)
		openSigned.GET("/vps/:id", handler.VPSDetail)
		openSigned.POST("/vps/:id/refresh", handler.VPSRefresh)
		openSigned.GET("/vps/:id/panel", handler.VPSPanel)
		openSigned.GET("/vps/:id/monitor", handler.VPSMonitor)
		openSigned.GET("/vps/:id/vnc", handler.VPSVNC)
		openSigned.POST("/vps/:id/start", handler.VPSStart)
		openSigned.POST("/vps/:id/shutdown", handler.VPSShutdown)
		openSigned.POST("/vps/:id/reboot", handler.VPSReboot)
		openSigned.POST("/vps/:id/reset-os", handler.VPSResetOS)
		openSigned.POST("/vps/:id/reset-os-password", handler.VPSResetOSPassword)
		openSigned.GET("/vps/:id/snapshots", handler.VPSSnapshots)
		openSigned.POST("/vps/:id/snapshots", handler.VPSSnapshots)
		openSigned.DELETE("/vps/:id/snapshots/:snapshotId", handler.VPSSnapshotDelete)
		openSigned.POST("/vps/:id/snapshots/:snapshotId/restore", handler.VPSSnapshotRestore)
		openSigned.GET("/vps/:id/backups", handler.VPSBackups)
		openSigned.POST("/vps/:id/backups", handler.VPSBackups)
		openSigned.DELETE("/vps/:id/backups/:backupId", handler.VPSBackupDelete)
		openSigned.POST("/vps/:id/backups/:backupId/restore", handler.VPSBackupRestore)
		openSigned.GET("/vps/:id/firewall", handler.VPSFirewallRules)
		openSigned.POST("/vps/:id/firewall", handler.VPSFirewallRules)
		openSigned.DELETE("/vps/:id/firewall/:ruleId", handler.VPSFirewallDelete)
		openSigned.GET("/vps/:id/ports", handler.VPSPortMappings)
		openSigned.POST("/vps/:id/ports", handler.VPSPortMappings)
		openSigned.GET("/vps/:id/ports/candidates", handler.VPSPortCandidates)
		openSigned.DELETE("/vps/:id/ports/:mappingId", handler.VPSPortMappingDelete)
	}
}
