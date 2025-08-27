package routes

import (
	"VCS_SMS_Microservice/internal/auth-service/api/handler"
	"VCS_SMS_Microservice/internal/auth-service/api/middleware"

	"github.com/gin-gonic/gin"
)

var (
	ScopeUserRead        = "users:read"
	ScopeUserCreate      = "users:create"
	ScopeUserRolesUpdate = "users:roles:update"
)

func SetUpUserRoutes(r *gin.Engine, h handler.UserHandler, m middleware.AuthMiddleware) {
	userRoutes := r.Group("/users")
	userRoutes.GET("/:id", m.ValidateAndExtractJwt(), m.CheckUserPermission(ScopeUserRead), h.GetUserByID())
	userRoutes.GET("/me", m.ValidateAndExtractJwt(), h.GetMe())
	userRoutes.PUT("/:id/roles", m.ValidateAndExtractJwt(), m.CheckUserPermission(ScopeUserRolesUpdate), h.UpdateUserRole())
	userRoutes.PUT("/me/password", m.ValidateAndExtractJwt(), h.UpdateUserPassword())
	userRoutes.PATCH("/me", m.ValidateAndExtractJwt(), h.UpdateUserInfo())
	userRoutes.GET("", m.ValidateAndExtractJwt(), m.CheckUserPermission(ScopeUserRead), h.GetUsers())
}
