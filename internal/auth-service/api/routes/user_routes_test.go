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

func TestSetUpUserRoutes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHandler := mockhandler.NewMockUserHandler(ctrl)
	mockMiddleware := mockmiddleware.NewMockAuthMiddleware(ctrl)

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
	mockHandler.EXPECT().GetUserByID().Return(emptySuccessHandler)
	mockHandler.EXPECT().GetMe().Return(emptySuccessHandler)
	mockHandler.EXPECT().UpdateUserRole().Return(emptySuccessHandler)
	mockHandler.EXPECT().UpdateUserPassword().Return(emptySuccessHandler)
	mockHandler.EXPECT().UpdateUserInfo().Return(emptySuccessHandler)
	mockHandler.EXPECT().GetUsers().Return(emptySuccessHandler)
	SetUpUserRoutes(r, mockHandler, mockMiddleware)

	testCases := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{
			name:           "Get User By ID Route",
			method:         http.MethodGet,
			path:           "/users/user-123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Get Me Route",
			method:         http.MethodGet,
			path:           "/users/me",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Update User Role Route",
			method:         http.MethodPut,
			path:           "/users/user-456/roles",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Update User Password Route",
			method:         http.MethodPut,
			path:           "/users/me/password",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Update User Info Route",
			method:         http.MethodPatch,
			path:           "/users/me",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Get Users Route",
			method:         http.MethodGet,
			path:           "/users",
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
