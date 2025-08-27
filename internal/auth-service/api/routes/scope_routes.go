package routes

import (
	"VCS_SMS_Microservice/internal/auth-service/api/handler"
	"VCS_SMS_Microservice/internal/auth-service/api/middleware"

	"github.com/gin-gonic/gin"
)

var (
	ScopeScopesRead = "scopes:read"
)

func SetUpScopeRoutes(r *gin.Engine, h handler.ScopeHandler, m middleware.AuthMiddleware) {
	scopeRoutes := r.Group("/scopes")
	scopeRoutes.GET("", m.ValidateAndExtractJwt(), m.CheckUserPermission(ScopeScopesRead), h.GetScopes())
}
