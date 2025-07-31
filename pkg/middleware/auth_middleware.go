package middleware

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"slices"
	"strings"
)

type AuthMiddleware interface {
	CheckUserPermission(requiredScope string) gin.HandlerFunc
}

type authMiddleware struct {
}

func (a *authMiddleware) CheckUserPermission(requiredScope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		scopesHeader := c.Request.Header.Get("X-User-Scopes")
		if len(scopesHeader) == 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "Scopes header is empty",
			})
			return
		}
		scopes := strings.Split(scopesHeader, ",")
		if !slices.Contains(scopes, requiredScope) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "Permission denied",
			})
			return
		}
		c.Next()
	}
}

func NewAuthMiddleware() AuthMiddleware {
	return &authMiddleware{}
}
