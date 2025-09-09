package routes

import (
	mockmiddleware "VCS_SMS_Microservice/internal/auth-service/mocks/api/middleware"
	mockhandler "VCS_SMS_Microservice/internal/server-service/mocks/api/handler"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestSetUpServerRoutes(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockHandler := mockhandler.NewMockServerHandler(ctrl)
	mockMiddleware := mockmiddleware.NewMockAuthMiddleware(ctrl)

	gin.SetMode(gin.TestMode)
	r := gin.New()

	emptySuccessHandler := func(c *gin.Context) {
		c.Status(http.StatusOK)
	}
	nextMiddleware := func(c *gin.Context) {
		c.Next()
	}

	mockMiddleware.EXPECT().CheckUserPermission(gomock.Any()).Return(nextMiddleware).AnyTimes()

	mockHandler.EXPECT().CreateServer().Return(emptySuccessHandler).AnyTimes()
	mockHandler.EXPECT().GetServers().Return(emptySuccessHandler).AnyTimes()
	mockHandler.EXPECT().UpdateServer().Return(emptySuccessHandler).AnyTimes()
	mockHandler.EXPECT().DeleteServer().Return(emptySuccessHandler).AnyTimes()
	mockHandler.EXPECT().ImportServersFromExcelFile().Return(emptySuccessHandler).AnyTimes()
	mockHandler.EXPECT().ExportServersToExcelFile().Return(emptySuccessHandler).AnyTimes()
	mockHandler.EXPECT().ReportAllServersHealthInfo().Return(emptySuccessHandler).AnyTimes()
	mockHandler.EXPECT().GetServerUptimePercentage().Return(emptySuccessHandler).AnyTimes()

	SetUpServerRoutes(r, mockHandler, mockMiddleware)

	testCases := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{
			name:           "Create Server Route",
			method:         http.MethodPost,
			path:           "/servers",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Get Servers Route",
			method:         http.MethodGet,
			path:           "/servers",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Update Server Route",
			method:         http.MethodPatch,
			path:           "/servers/some-id",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Delete Server Route",
			method:         http.MethodDelete,
			path:           "/servers/some-id",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Import Servers Route",
			method:         http.MethodPost,
			path:           "/servers/import",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Export Servers Route",
			method:         http.MethodGet,
			path:           "/servers/export",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Report Servers Route",
			method:         http.MethodPost,
			path:           "/servers/reports",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Get Uptime Route",
			method:         http.MethodGet,
			path:           "/servers/some-id/uptime",
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
