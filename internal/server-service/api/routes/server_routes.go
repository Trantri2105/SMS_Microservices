package routes

import (
	"VCS_SMS_Microservice/internal/server-service/api/handler"
	"VCS_SMS_Microservice/pkg/middleware"
	"github.com/gin-gonic/gin"
)

const (
	ScopeServersRead   = "servers:read"
	ScopeServersCreate = "servers:create"
	ScopeServersUpdate = "servers:update"
	ScopeServersDelete = "servers:delete"
)

func AddServerRoutes(r *gin.Engine, handler handler.ServerHandler, m middleware.AuthMiddleware) {
	serverRoutes := r.Group("/servers")
	serverRoutes.POST("", m.CheckUserPermission(ScopeServersCreate), handler.CreateServer())
	serverRoutes.GET("", m.CheckUserPermission(ScopeServersRead), handler.GetServers())
	serverRoutes.PATCH("/:id", m.CheckUserPermission(ScopeServersUpdate), handler.UpdateServer())
	serverRoutes.DELETE("/:id", m.CheckUserPermission(ScopeServersDelete), handler.DeleteServer())
	serverRoutes.POST("/import", m.CheckUserPermission(ScopeServersCreate), handler.ImportServersFromExcelFile())
	serverRoutes.GET("/export", m.CheckUserPermission(ScopeServersRead), handler.ExportServersToExcelFile())
	serverRoutes.POST("/reports", m.CheckUserPermission(ScopeServersRead), handler.ReportAllServersHealthInfo())
	serverRoutes.GET("/:id/uptime", m.CheckUserPermission(ScopeServersRead), handler.GetServerUptimePercentage())
}
