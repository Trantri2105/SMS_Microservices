package handler

import (
	apperrors "VCS_SMS_Microservice/internal/auth-service/errors"
	mockhandler "VCS_SMS_Microservice/internal/auth-service/mocks/api/handler"
	mockservice "VCS_SMS_Microservice/internal/auth-service/mocks/service"
	"VCS_SMS_Microservice/internal/auth-service/model"
	"bytes"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap/zapcore"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRoleHandler_CreateRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRoleService := mockservice.NewMockRoleService(ctrl)
	mockLogger := mockhandler.NewMockLogger(ctrl)
	handler := NewRoleHandler(mockRoleService, mockLogger)

	router := gin.New()
	router.POST("/roles", handler.CreateRole())

	successResponse := model.Role{
		ID:          "role-123",
		Name:        "Admin",
		Description: "Administrator Role",
		Scopes:      []model.Scope{{ID: "scope-abc", Name: "users:read"}},
	}

	testCases := []struct {
		name           string
		body           string
		mock           func()
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			body: `{"name":"Admin", "description":"Administrator Role", "scopeIds":["scope-abc"]}`,
			mock: func() {
				mockRoleService.EXPECT().CreateRole(gomock.Any(), gomock.Any()).Return(successResponse, nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   `{"id":"role-123", "name":"Admin", "description":"Administrator Role", "scopes":[{"id":"scope-abc", "name":"users:read"}]}`,
		},
		{
			name:           "Bad Request Invalid JSON",
			body:           `{"name":"Admin"`,
			mock:           func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"message":"Invalid request body"}`,
		},
		{
			name:           "Bad Request Validation Error",
			body:           `{"description":"Administrator Role"}`, // Thiếu Name
			mock:           func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"message":"The Name field is required"}`,
		},
		{
			name: "Bad Request - Invalid Scopes",
			body: `{"name":"Admin", "scopeIds":["invalid-scope"]}`,
			mock: func() {
				mockRoleService.EXPECT().CreateRole(gomock.Any(), gomock.Any()).Return(model.Role{}, apperrors.ErrInvalidScopes)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"message":"Invalid scopes"}`,
		},
		{
			name: "Bad Request Role Name Already Exists",
			body: `{"name":"Admin"}`,
			mock: func() {
				mockRoleService.EXPECT().CreateRole(gomock.Any(), gomock.Any()).Return(model.Role{}, apperrors.ErrRoleNameAlreadyExists)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"message":"Role name already exists"}`,
		},
		{
			name: "Internal Server Error",
			body: `{"name":"Admin"}`,
			mock: func() {
				mockRoleService.EXPECT().CreateRole(gomock.Any(), gomock.Any()).Return(model.Role{}, errors.New("db error"))
				mockLogger.EXPECT().LoggingError(gomock.Any(), gomock.Any(), "failed to create new role", zapcore.ErrorLevel)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"message":"Internal server error"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mock()
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/roles", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
			assert.JSONEq(t, tc.expectedBody, w.Body.String())
		})
	}
}

func TestRoleHandler_UpdateRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRoleService := mockservice.NewMockRoleService(ctrl)
	mockLogger := mockhandler.NewMockLogger(ctrl)
	handler := NewRoleHandler(mockRoleService, mockLogger)

	router := gin.New()
	router.PUT("/roles/:id", handler.UpdateRole())

	roleID := "role-xyz"

	testCases := []struct {
		name           string
		roleID         string
		body           string
		mock           func()
		expectedStatus int
		expectedBody   string
	}{
		{
			name:   "Success",
			roleID: roleID,
			body:   `{"name":"Updated Name"}`,
			mock: func() {
				mockRoleService.EXPECT().UpdateRoleByID(gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"Role updated"}`,
		},
		{
			name:           "Bad Request Invalid JSON",
			roleID:         roleID,
			body:           `{"name":"Updated Name",`,
			mock:           func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"message":"Invalid request body"}`,
		},
		{
			name:   "Bad Request Role Name Already Exists",
			roleID: roleID,
			body:   `{"name":"Existing Name"}`,
			mock: func() {
				mockRoleService.EXPECT().UpdateRoleByID(gomock.Any(), gomock.Any()).Return(apperrors.ErrRoleNameAlreadyExists)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"message":"Role name already exists"}`,
		},
		{
			name:   "Bad Request Invalid Scopes",
			roleID: roleID,
			body:   `{"scopeIds":["invalid-scope"]}`,
			mock: func() {
				mockRoleService.EXPECT().UpdateRoleByID(gomock.Any(), gomock.Any()).Return(apperrors.ErrInvalidScopes)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"message":"Invalid scopes"}`,
		},
		{
			name:   "Not Found Role Not Found",
			roleID: "not-found-id",
			body:   `{"name":"Updated Name"}`,
			mock: func() {
				mockRoleService.EXPECT().UpdateRoleByID(gomock.Any(), gomock.Any()).Return(apperrors.ErrRoleNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"message":"Role not found"}`,
		},
		{
			name:   "Internal Server Error",
			roleID: roleID,
			body:   `{"name":"Updated Name"}`,
			mock: func() {
				mockRoleService.EXPECT().UpdateRoleByID(gomock.Any(), gomock.Any()).Return(errors.New("db error"))
				mockLogger.EXPECT().LoggingError(gomock.Any(), gomock.Any(), "failed to update role", zapcore.ErrorLevel)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"message":"Internal server error"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mock()
			w := httptest.NewRecorder()
			url := fmt.Sprintf("/roles/%s", tc.roleID)
			req, _ := http.NewRequest(http.MethodPut, url, bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
			assert.JSONEq(t, tc.expectedBody, w.Body.String())
		})
	}
}

func TestRoleHandler_DeleteRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRoleService := mockservice.NewMockRoleService(ctrl)
	mockLogger := mockhandler.NewMockLogger(ctrl)
	handler := NewRoleHandler(mockRoleService, mockLogger)

	router := gin.New()
	router.DELETE("/roles/:id", handler.DeleteRole())

	roleID := "role-to-delete"

	testCases := []struct {
		name           string
		roleID         string
		mock           func()
		expectedStatus int
		expectedBody   string
	}{
		{
			name:   "Success",
			roleID: roleID,
			mock: func() {
				mockRoleService.EXPECT().DeleteRoleByID(gomock.Any(), roleID).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"Role deleted"}`,
		},
		{
			name:   "Internal Server Error",
			roleID: roleID,
			mock: func() {
				mockRoleService.EXPECT().DeleteRoleByID(gomock.Any(), roleID).Return(errors.New("db error"))
				mockLogger.EXPECT().LoggingError(gomock.Any(), gomock.Any(), "failed to delete role", zapcore.ErrorLevel)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"message":"Internal server error"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mock()
			w := httptest.NewRecorder()
			url := fmt.Sprintf("/roles/%s", tc.roleID)
			req, _ := http.NewRequest(http.MethodDelete, url, nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
			assert.JSONEq(t, tc.expectedBody, w.Body.String())
		})
	}
}

func TestRoleHandler_GetRoles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRoleService := mockservice.NewMockRoleService(ctrl)
	mockLogger := mockhandler.NewMockLogger(ctrl)
	handler := NewRoleHandler(mockRoleService, mockLogger)

	router := gin.New()
	router.GET("/roles", handler.GetRoles())

	rolesFromService := []model.Role{{ID: "role-1", Name: "Admin"}, {ID: "role-2", Name: "User"}}

	testCases := []struct {
		name           string
		url            string
		mock           func()
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success Default Params",
			url:  "/roles",
			mock: func() {
				mockRoleService.EXPECT().GetRoles(gomock.Any(), "", "created_at", "asc", 10, 0).Return(rolesFromService, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[{"id":"role-1", "name":"Admin"}, {"id":"role-2", "name":"User"}]`,
		},
		{
			name: "Success - With All Custom Parameters",
			url:  "/roles?role_name=Admin&offset=5&limit=20&sort_by=name&sort_order=desc",
			mock: func() {
				mockRoleService.EXPECT().GetRoles(gomock.Any(), "Admin", "name", "desc", 20, 5).Return(rolesFromService, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[{"id":"role-1", "name":"Admin"}, {"id":"role-2", "name":"User"}]`,
		},
		{
			name:           "Bad Request - Invalid Offset",
			url:            "/roles?offset=abc",
			mock:           func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"message":"Offset must be an integer"}`,
		},
		{
			name:           "Bad Request - Invalid Limit",
			url:            "/roles?limit=xyz",
			mock:           func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"message":"Limit must be an integer"}`,
		},
		{
			name:           "Bad Request Invalid Sort By",
			url:            "/roles?sort_by=id",
			mock:           func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"message":"Invalid sort by"}`,
		},
		{
			name:           "Bad Request Invalid Sort Order",
			url:            "/roles?sort_order=ascending",
			mock:           func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"message":"Invalid sort order"}`,
		},
		{
			name: "Success - With Negative Offset (should default to 0)",
			url:  "/roles?offset=-50",
			mock: func() {
				mockRoleService.EXPECT().GetRoles(gomock.Any(), "", "created_at", "asc", 10, 0).Return(rolesFromService, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[{"id":"role-1", "name":"Admin"}, {"id":"role-2", "name":"User"}]`,
		},
		{
			name: "Success - With Zero Limit (should default to 10)",
			url:  "/roles?limit=0",
			mock: func() {
				mockRoleService.EXPECT().GetRoles(gomock.Any(), "", "created_at", "asc", 10, 0).Return(rolesFromService, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[{"id":"role-1", "name":"Admin"}, {"id":"role-2", "name":"User"}]`,
		},
		{
			name: "Internal Server Error",
			url:  "/roles",
			mock: func() {
				mockRoleService.EXPECT().GetRoles(gomock.Any(), "", "created_at", "asc", 10, 0).Return(nil, errors.New("db error"))
				mockLogger.EXPECT().LoggingError(gomock.Any(), gomock.Any(), "failed to fetch roles", zapcore.ErrorLevel)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"message":"Internal server error"}`,
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

func TestRoleHandler_GetRoleByID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRoleService := mockservice.NewMockRoleService(ctrl)
	mockLogger := mockhandler.NewMockLogger(ctrl)
	handler := NewRoleHandler(mockRoleService, mockLogger)

	router := gin.New()
	router.GET("/roles/:id", handler.GetRoleByID())

	roleID := "role-abc"
	successResponse := model.Role{
		ID: roleID, Name: "Viewer", Scopes: []model.Scope{{ID: "scope-1", Name: "read"}},
	}

	testCases := []struct {
		name           string
		roleID         string
		mock           func()
		expectedStatus int
		expectedBody   string
	}{
		{
			name:   "Success",
			roleID: roleID,
			mock: func() {
				mockRoleService.EXPECT().GetRoleByID(gomock.Any(), roleID).Return(successResponse, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"id":"role-abc", "name":"Viewer", "scopes":[{"id":"scope-1", "name":"read"}]}`,
		},
		{
			name:   "Not Found",
			roleID: "not-found",
			mock: func() {
				mockRoleService.EXPECT().GetRoleByID(gomock.Any(), "not-found").Return(model.Role{}, apperrors.ErrRoleNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"message":"Role not found"}`,
		},
		{
			name:   "Internal Server Error",
			roleID: roleID,
			mock: func() {
				mockRoleService.EXPECT().GetRoleByID(gomock.Any(), roleID).Return(model.Role{}, errors.New("db error"))
				mockLogger.EXPECT().LoggingError(gomock.Any(), gomock.Any(), "failed to get role", zapcore.ErrorLevel)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"message":"Internal server error"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mock()
			w := httptest.NewRecorder()
			url := fmt.Sprintf("/roles/%s", tc.roleID)
			req, _ := http.NewRequest(http.MethodGet, url, nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
			assert.JSONEq(t, tc.expectedBody, w.Body.String())
		})
	}
}
