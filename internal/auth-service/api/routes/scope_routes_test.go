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

func TestSetUpScopeRoutes(t *testing.T) {
	ctrl := gomock.NewController(t)

	emptySuccessHandler := func(c *gin.Context) {
		c.Status(http.StatusOK)
	}
	nextMiddleware := func(c *gin.Context) {
		c.Next()
	}

	mockHandler := mockhandler.NewMockScopeHandler(ctrl)
	mockMiddleware := mockmiddleware.NewMockAuthMiddleware(ctrl)
	mockMiddleware.EXPECT().ValidateAndExtractJwt().Return(nextMiddleware).AnyTimes()
	mockMiddleware.EXPECT().CheckUserPermission(gomock.Any()).Return(nextMiddleware).AnyTimes()
	mockHandler.EXPECT().GetScopes().Return(emptySuccessHandler)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	SetUpScopeRoutes(r, mockHandler, mockMiddleware)
	req, _ := http.NewRequest(http.MethodGet, "/scopes", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
