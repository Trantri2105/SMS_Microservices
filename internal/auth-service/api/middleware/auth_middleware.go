package middleware

import (
	"VCS_SMS_Microservice/internal/auth-service/jwt"
	"VCS_SMS_Microservice/internal/server-service/api/dto/response"
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	jwt2 "github.com/golang-jwt/jwt"
)

type AuthMiddleware interface {
	ValidateAndExtractJwt() gin.HandlerFunc
	CheckUserPermission(requiredScope string) gin.HandlerFunc
}

const (
	JWTClaimsContextKey = "JWTClaimsContextKey"
)

type authMiddleware struct {
	jwt jwt.Utils
}

func (a *authMiddleware) ValidateAndExtractJwt() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if len(authHeader) == 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Response{Message: "Authorization header is empty"})
			return
		}
		header := strings.Fields(authHeader)
		if len(header) != 2 && header[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Response{Message: "Authorization header is invalid"})
			return
		}
		accessToken := header[1]
		claims, err := a.jwt.VerifyToken(accessToken)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Response{Message: "Invalid access token"})
			return
		}
		c.Set(JWTClaimsContextKey, claims)
		c.Next()
	}
}

func (a *authMiddleware) CheckUserPermission(requiredScope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := c.Value(JWTClaimsContextKey).(jwt2.MapClaims)
		scopesList := claims["scopes"].([]interface{})
		scopes := make([]string, len(scopesList))
		for i, scope := range scopesList {
			scopes[i] = scope.(string)
		}
		if !slices.Contains(scopes, requiredScope) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Response{Message: "Permission denied"})
			return
		}
		c.Next()
	}
}

func NewAuthMiddleware(jwt jwt.Utils) AuthMiddleware {
	return &authMiddleware{jwt: jwt}
}
