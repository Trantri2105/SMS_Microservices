package handler

import (
	mockhandler "VCS_SMS_Microservice/internal/auth-service/mocks/api/handler"
	mockservice "VCS_SMS_Microservice/internal/auth-service/mocks/service"
	"VCS_SMS_Microservice/internal/auth-service/model"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap/zapcore"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetScopes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockScopeService := mockservice.NewMockScopeService(ctrl)
	mockLogger := mockhandler.NewMockLogger(ctrl)
	handler := NewScopeHandler(mockScopeService, mockLogger)

	router := gin.New()
	router.GET("/scopes", handler.GetScopes())

	scopesFromService := []model.Scope{
		{ID: "scope-1", Name: "read:users", Description: "Read all users"},
		{ID: "scope-2", Name: "write:users", Description: "Write all users"},
	}

	testCases := []struct {
		name           string
		url            string
		mock           func()
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success - With Default Parameters",
			url:  "/scopes",
			mock: func() {
				mockScopeService.EXPECT().GetScopesList(gomock.Any(), "", "created_at", "asc", 10, 0).Return(scopesFromService, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[{"id":"scope-1","name":"read:users","description":"Read all users"},{"id":"scope-2","name":"write:users","description":"Write all users"}]`,
		},
		{
			name: "Success - With All Custom Parameters",
			url:  "/scopes?scope_name=read&offset=5&limit=20&sort_by=name&sort_order=desc",
			mock: func() {
				mockScopeService.EXPECT().GetScopesList(gomock.Any(), "read", "name", "desc", 20, 5).Return(scopesFromService, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[{"id":"scope-1","name":"read:users","description":"Read all users"},{"id":"scope-2","name":"write:users","description":"Write all users"}]`,
		},
		{
			name:           "Bad Request - Invalid Offset",
			url:            "/scopes?offset=abc",
			mock:           func() {}, // Service không được gọi
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"message":"Offset must be an integer"}`,
		},
		{
			name:           "Bad Request - Invalid Limit",
			url:            "/scopes?limit=xyz",
			mock:           func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"message":"Limit must be an integer"}`,
		},
		{
			name:           "Bad Request - Invalid Sort By",
			url:            "/scopes?sort_by=id",
			mock:           func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"message":"Invalid sort by"}`,
		},
		{
			name:           "Bad Request - Invalid Sort Order",
			url:            "/scopes?sort_order=ascending",
			mock:           func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"message":"Invalid sort order"}`,
		},
		{
			name: "Success - With Negative Offset (should default to 0)",
			url:  "/scopes?offset=-10",
			mock: func() {
				mockScopeService.EXPECT().GetScopesList(gomock.Any(), "", "created_at", "asc", 10, 0).Return(scopesFromService, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[{"id":"scope-1","name":"read:users","description":"Read all users"},{"id":"scope-2","name":"write:users","description":"Write all users"}]`,
		},
		{
			name: "Success - With Zero Limit (should default to 10)",
			url:  "/scopes?limit=0",
			mock: func() {
				mockScopeService.EXPECT().GetScopesList(gomock.Any(), "", "created_at", "asc", 10, 0).Return(scopesFromService, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[{"id":"scope-1","name":"read:users","description":"Read all users"},{"id":"scope-2","name":"write:users","description":"Write all users"}]`,
		},
		{
			name: "Internal Server Error - Service Fails",
			url:  "/scopes",
			mock: func() {
				mockScopeService.EXPECT().GetScopesList(gomock.Any(), "", "created_at", "asc", 10, 0).Return(nil, errors.New("database is down"))
				mockLogger.EXPECT().LoggingError(gomock.Any(), gomock.Any(), "failed to get scopes", zapcore.ErrorLevel)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"message":"Internal Server Error"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mock()

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, tc.url, nil)

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
			assert.JSONEq(t, tc.expectedBody, w.Body.String())
		})
	}
}
