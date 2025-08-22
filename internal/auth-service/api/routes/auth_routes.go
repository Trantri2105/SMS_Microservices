package routes

import (
	"VCS_SMS_Microservice/internal/auth-service/api/handler"
	"VCS_SMS_Microservice/internal/auth-service/api/middleware"

	"github.com/gin-gonic/gin"
)

var (
	ScopeUsersCreate = "users:create"
)

func SetUpAuthRoutes(r *gin.Engine, handler handler.AuthHandler, m middleware.AuthMiddleware) {
	authRoutes := r.Group("/auth")
	authRoutes.POST("/register", m.ValidateAndExtractJwt(), m.CheckUserPermission(ScopeUsersCreate), handler.Register())
	authRoutes.POST("/login", handler.Login())
	authRoutes.POST("/logout", m.ValidateAndExtractJwt(), handler.Logout())
	authRoutes.POST("/refresh", handler.Refresh())
	authRoutes.GET("/verify", m.ValidateAndExtractJwt(), handler.VerifyToken())
}
