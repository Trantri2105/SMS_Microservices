package routes

import (
	mock_handler "VCS_SMS_Microservice/internal/auth-service/mocks/api/handler"
	mock_middleware "VCS_SMS_Microservice/internal/auth-service/mocks/api/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSetUpAuthRoutes(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockHandler := mock_handler.NewMockAuthHandler(ctrl)
	mockMiddleware := mock_middleware.NewMockAuthMiddleware(ctrl)

	gin.SetMode(gin.TestMode)
	r := gin.New()

	emptySuccessHandler := func(c *gin.Context) {
		c.Status(http.StatusOK)
	}
	nextMiddleware := func(c *gin.Context) {
		c.Next()
	}
	mockMiddleware.EXPECT().ValidateAndExtractJwt().Return(nextMiddleware).AnyTimes()
	mockMiddleware.EXPECT().CheckUserPermission(gomock.Any()).Return(nextMiddleware).AnyTimes()
	mockHandler.EXPECT().Register().Return(emptySuccessHandler)
	mockHandler.EXPECT().Login().Return(emptySuccessHandler)
	mockHandler.EXPECT().Logout().Return(emptySuccessHandler)
	mockHandler.EXPECT().Refresh().Return(emptySuccessHandler)
	mockHandler.EXPECT().VerifyToken().Return(emptySuccessHandler)

	SetUpAuthRoutes(r, mockHandler, mockMiddleware)

	testCases := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{
			name:           "Register Route Success",
			method:         http.MethodPost,
			path:           "/auth/register",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Login Route Success",
			method:         http.MethodPost,
			path:           "/auth/login",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Logout Route Success",
			method:         http.MethodPost,
			path:           "/auth/logout",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Refresh Route Success",
			method:         http.MethodPost,
			path:           "/auth/refresh",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Verify Token Route Success",
			method:         http.MethodGet,
			path:           "/auth/verify",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}
