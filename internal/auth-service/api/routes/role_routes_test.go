package routes

import (
	mockhandler "VCS_SMS_Microservice/internal/auth-service/mocks/api/handler"
	mockmiddleware "VCS_SMS_Microservice/internal/auth-service/mocks/api/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSetUpRoleRoutes(t *testing.T) {
	ctrl := gomock.NewController(t)

	emptySuccessHandler := func(c *gin.Context) {
		c.Status(http.StatusOK)
	}
	nextMiddleware := func(c *gin.Context) {
		c.Next()
	}

	mockHandler := mockhandler.NewMockRoleHandler(ctrl)
	mockMiddleware := mockmiddleware.NewMockAuthMiddleware(ctrl)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	mockMiddleware.EXPECT().ValidateAndExtractJwt().Return(nextMiddleware).AnyTimes()
	mockMiddleware.EXPECT().CheckUserPermission(gomock.Any()).Return(nextMiddleware).AnyTimes()
	mockHandler.EXPECT().CreateRole().Return(emptySuccessHandler)
	mockHandler.EXPECT().UpdateRole().Return(emptySuccessHandler)
	mockHandler.EXPECT().DeleteRole().Return(emptySuccessHandler)
	mockHandler.EXPECT().GetRoles().Return(emptySuccessHandler)
	mockHandler.EXPECT().GetRoleByID().Return(emptySuccessHandler)
	SetUpRoleRoutes(r, mockHandler, mockMiddleware)

	testCases := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{
			name:           "Create Role Route",
			method:         http.MethodPost,
			path:           "/roles",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Update Role Route",
			method:         http.MethodPatch,
			path:           "/roles/123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Delete Role Route",
			method:         http.MethodDelete,
			path:           "/roles/456",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Get Roles Route",
			method:         http.MethodGet,
			path:           "/roles",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Get Role By ID Route",
			method:         http.MethodGet,
			path:           "/roles/789",
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
