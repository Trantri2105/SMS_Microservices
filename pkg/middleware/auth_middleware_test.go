package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestNewAuthMiddleware(t *testing.T) {
	middleware := NewAuthMiddleware()

	assert.NotNil(t, middleware)
	assert.Implements(t, (*AuthMiddleware)(nil), middleware)
}

func TestCheckUserPermission(t *testing.T) {
	testCases := []struct {
		name           string
		requiredScope  string
		headerValue    string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Success",
			requiredScope:  "read:users",
			headerValue:    "read:products,read:users,write:products",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"ok"}`,
		},
		{
			name:           "Failure, no headers",
			requiredScope:  "read:users",
			headerValue:    "",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"message":"X-User-Scopes header is empty"}`,
		},
		{
			name:           "Failure, invalid scope",
			requiredScope:  "admin:all",
			headerValue:    "read:products,read:users",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"message":"Permission denied"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			w := httptest.NewRecorder()
			c, router := gin.CreateTestContext(w)

			m := NewAuthMiddleware()

			router.GET("/test", m.CheckUserPermission(tc.requiredScope), func(ctx *gin.Context) {
				ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
			})

			req := httptest.NewRequest("GET", "/test", nil)
			if tc.headerValue != "" {
				req.Header.Set("X-User-Scopes", tc.headerValue)
			}
			c.Request = req
			router.ServeHTTP(w, req)
			assert.Equal(t, tc.expectedStatus, w.Code)
			assert.JSONEq(t, tc.expectedBody, w.Body.String())
		})
	}
}
