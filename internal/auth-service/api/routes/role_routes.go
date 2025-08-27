package routes

import (
	"VCS_SMS_Microservice/internal/auth-service/api/handler"
	"VCS_SMS_Microservice/internal/auth-service/api/middleware"

	"github.com/gin-gonic/gin"
)

var (
	ScopeRoleCreate = "roles:create"
	ScopeRoleUpdate = "roles:update"
	ScopeRoleDelete = "roles:delete"
	ScopeRoleRead   = "roles:read"
)

func SetUpRoleRoutes(r *gin.Engine, h handler.RoleHandler, m middleware.AuthMiddleware) {
	roleRoutes := r.Group("/roles")
	roleRoutes.POST("", m.ValidateAndExtractJwt(), m.CheckUserPermission(ScopeRoleCreate), h.CreateRole())
	roleRoutes.PUT("", m.ValidateAndExtractJwt(), m.CheckUserPermission(ScopeRoleUpdate), h.UpdateRole())
	roleRoutes.DELETE("/:id", m.ValidateAndExtractJwt(), m.CheckUserPermission(ScopeRoleDelete), h.DeleteRole())
	roleRoutes.GET("", m.ValidateAndExtractJwt(), m.CheckUserPermission(ScopeRoleRead), h.GetRoles())
	roleRoutes.GET("/:id", m.ValidateAndExtractJwt(), m.CheckUserPermission(ScopeRoleRead), h.GetRoleByID())
}
