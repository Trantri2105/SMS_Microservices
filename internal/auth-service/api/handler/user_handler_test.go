package handler

import (
	"VCS_SMS_Microservice/internal/auth-service/api/middleware"
	apperrors "VCS_SMS_Microservice/internal/auth-service/errors"
	mockhandler "VCS_SMS_Microservice/internal/auth-service/mocks/api/handler"
	mockservice "VCS_SMS_Microservice/internal/auth-service/mocks/service"
	"VCS_SMS_Microservice/internal/auth-service/model"
	"bytes"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap/zapcore"
	"net/http"
	"net/http/httptest"
	"testing"
)

func mockAuthMiddleware(claims jwt.MapClaims) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(middleware.JWTClaimsContextKey, claims)
		c.Next()
	}
}

func TestUserHandler_GetMe(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserService := mockservice.NewMockUserService(ctrl)
	mockLogger := mockhandler.NewMockLogger(ctrl)
	handler := NewUserHandler(mockUserService, mockLogger)

	claims := jwt.MapClaims{"user_id": "user-me-123"}

	router := gin.New()
	router.GET("/me", mockAuthMiddleware(claims), handler.GetMe())

	userFromService := model.User{
		ID: "user-me-123", Email: "me@example.com",
		Roles: []model.Role{
			{ID: "role1", Name: "Admin", Scopes: []model.Scope{{ID: "s1", Name: "read"}}},
			{ID: "role2", Name: "Viewer", Scopes: []model.Scope{{ID: "s1", Name: "read"}}},
		},
	}

	testCases := []struct {
		name           string
		mock           func()
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			mock: func() {
				mockUserService.EXPECT().GetUserById(gomock.Any(), "user-me-123").Return(userFromService, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"id":"user-me-123", "email":"me@example.com", "roles":[{"id":"role1", "name":"Admin"}, {"id":"role2", "name":"Viewer"}], "scopes":[{"id":"s1", "name":"read"}]}`,
		},
		{
			name: "User Not Found",
			mock: func() {
				mockUserService.EXPECT().GetUserById(gomock.Any(), "user-me-123").Return(model.User{}, apperrors.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"message":"User not found"}`,
		},
		{
			name: "Internal Server Error",
			mock: func() {
				mockUserService.EXPECT().GetUserById(gomock.Any(), "user-me-123").Return(model.User{}, errors.New("db error"))
				mockLogger.EXPECT().LoggingError(gomock.Any(), gomock.Any(), "failed to get user info by id", zapcore.ErrorLevel)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"message":"Internal Server Error"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mock()
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/me", nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
			assert.JSONEq(t, tc.expectedBody, w.Body.String())
		})
	}
}

func TestUserHandler_GetUserByID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserService := mockservice.NewMockUserService(ctrl)
	mockLogger := mockhandler.NewMockLogger(ctrl)
	handler := NewUserHandler(mockUserService, mockLogger)

	router := gin.New()
	router.GET("/users/:id", handler.GetUserByID())

	userID := "user-xyz-456"
	userFromService := model.User{ID: userID, Email: "test@example.com"}

	testCases := []struct {
		name           string
		userID         string
		mock           func()
		expectedStatus int
		expectedBody   string
	}{
		{
			name:   "Success",
			userID: userID,
			mock: func() {
				mockUserService.EXPECT().GetUserById(gomock.Any(), userID).Return(userFromService, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"id":"user-xyz-456", "email":"test@example.com"}`,
		},
		{
			name:   "User Not Found",
			userID: "not-found",
			mock: func() {
				mockUserService.EXPECT().GetUserById(gomock.Any(), "not-found").Return(model.User{}, apperrors.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"message":"User not found"}`,
		},
		{
			name:   "Internal Server Error",
			userID: userID,
			mock: func() {
				mockUserService.EXPECT().GetUserById(gomock.Any(), userID).Return(model.User{}, errors.New("db error"))
				mockLogger.EXPECT().LoggingError(gomock.Any(), gomock.Any(), gomock.Any(), zapcore.ErrorLevel)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"message":"Internal Server Error"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mock()
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/users/%s", tc.userID), nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
			assert.JSONEq(t, tc.expectedBody, w.Body.String())
		})
	}
}

func TestUserHandler_UpdateUserRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserService := mockservice.NewMockUserService(ctrl)
	mockLogger := mockhandler.NewMockLogger(ctrl)
	handler := NewUserHandler(mockUserService, mockLogger)

	router := gin.New()
	router.PUT("/users/:id/role", handler.UpdateUserRole())

	userID := "user-to-update"

	testCases := []struct {
		name           string
		userID         string
		body           string
		mock           func()
		expectedStatus int
		expectedBody   string
	}{
		{
			name:   "Success",
			userID: userID,
			body:   `{"role_ids":["21da6ffc-0495-4692-9fc1-e18777b7aa5a", "63355491-6f1f-4cba-8dd8-4962fff6d5fb"]}`,
			mock: func() {
				expectedUser := model.User{ID: userID, Roles: []model.Role{{ID: "21da6ffc-0495-4692-9fc1-e18777b7aa5a"}, {ID: "63355491-6f1f-4cba-8dd8-4962fff6d5fb"}}}
				mockUserService.EXPECT().UpdateUserByID(gomock.Any(), expectedUser).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"User role updated"}`,
		},
		{
			name:   "Bad Request Invalid Roles",
			userID: userID,
			body:   `{"role_ids":["63355491-6f1f-4cba-8dd8-4962fff6d5fb"]}`,
			mock: func() {
				mockUserService.EXPECT().UpdateUserByID(gomock.Any(), gomock.Any()).Return(apperrors.ErrInvalidRoles)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"message":"Invalid roles"}`,
		},
		{
			name:   "User Not Found",
			userID: "not-found",
			body:   `{"role_ids":["63355491-6f1f-4cba-8dd8-4962fff6d5fb"]}`,
			mock: func() {
				mockUserService.EXPECT().UpdateUserByID(gomock.Any(), gomock.Any()).Return(apperrors.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"message":"User not found"}`,
		},
		{
			name:   "Internal Server Error",
			userID: userID,
			body:   `{"role_ids":["63355491-6f1f-4cba-8dd8-4962fff6d5fb"]}`,
			mock: func() {
				mockUserService.EXPECT().UpdateUserByID(gomock.Any(), gomock.Any()).Return(errors.New("db error"))
				mockLogger.EXPECT().LoggingError(gomock.Any(), gomock.Any(), gomock.Any(), zapcore.ErrorLevel)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"message":"Internal Server Error"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mock()
			w := httptest.NewRecorder()
			url := fmt.Sprintf("/users/%s/role", tc.userID)
			req, _ := http.NewRequest(http.MethodPut, url, bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
			assert.JSONEq(t, tc.expectedBody, w.Body.String())
		})
	}
}

func TestUserHandler_UpdateUserPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserService := mockservice.NewMockUserService(ctrl)
	mockLogger := mockhandler.NewMockLogger(ctrl)
	handler := NewUserHandler(mockUserService, mockLogger)

	claims := jwt.MapClaims{"user_id": "user-pass-update"}

	router := gin.New()
	router.PUT("/user/password", mockAuthMiddleware(claims), handler.UpdateUserPassword())

	testCases := []struct {
		name           string
		body           string
		mock           func()
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			body: `{"current_password":"old-pass", "new_password":"new-pass"}`,
			mock: func() {
				mockUserService.EXPECT().UpdateUserPassword(gomock.Any(), "user-pass-update", "old-pass", "new-pass").Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"Password updated successfully"}`,
		},
		{
			name: "Bad Request Invalid Current Password",
			body: `{"current_password":"wrong-pass", "new_password":"new-pass"}`,
			mock: func() {
				mockUserService.EXPECT().UpdateUserPassword(gomock.Any(), "user-pass-update", "wrong-pass", "new-pass").Return(apperrors.ErrInvalidPassword)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"message":"Invalid password"}`,
		},
		{
			name: "Internal Server Error",
			body: `{"current_password":"old-pass", "new_password":"new-pass"}`,
			mock: func() {
				mockUserService.EXPECT().UpdateUserPassword(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("db error"))
				mockLogger.EXPECT().LoggingError(gomock.Any(), gomock.Any(), "failed to update user password", zapcore.ErrorLevel)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"message":"Internal server error"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mock()
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPut, "/user/password", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
			assert.JSONEq(t, tc.expectedBody, w.Body.String())
		})
	}
}

func TestUserHandler_UpdateUserInfo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserService := mockservice.NewMockUserService(ctrl)
	mockLogger := mockhandler.NewMockLogger(ctrl)
	handler := NewUserHandler(mockUserService, mockLogger)

	userID := "user-info-update-123"
	claims := jwt.MapClaims{"user_id": userID}

	router := gin.New()
	router.PUT("/user/info", mockAuthMiddleware(claims), handler.UpdateUserInfo())

	testCases := []struct {
		name           string
		body           string
		mock           func()
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			body: `{"email":"new.email@example.com", "first_name":"New", "last_name":"Name"}`,
			mock: func() {
				expectedUser := model.User{
					ID:        userID,
					Email:     "new.email@example.com",
					FirstName: "New",
					LastName:  "Name",
				}
				mockUserService.EXPECT().UpdateUserByID(gomock.Any(), expectedUser).Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"User info updated successfully"}`,
		},
		{
			name:           "Bad Request Invalid JSON",
			body:           `{"email":"bad@json"`,
			mock:           func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"message":"Invalid request body"}`,
		},
		{
			name:           "Bad Request Validation Error (Invalid Email)",
			body:           `{"email":"invalid-email", "first_name":"Test"}`,
			mock:           func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"message":"The Email field is not a valid email"}`,
		},
		{
			name: "User Not Found",
			body: `{"first_name":"Test"}`,
			mock: func() {
				mockUserService.EXPECT().UpdateUserByID(gomock.Any(), gomock.Any()).Return(apperrors.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"message":"User not found"}`,
		},
		{
			name: "Conflict Email Already Exists",
			body: `{"email":"exists@example.com"}`,
			mock: func() {
				mockUserService.EXPECT().UpdateUserByID(gomock.Any(), gomock.Any()).Return(apperrors.ErrUserMailAlreadyExists)
			},
			expectedStatus: http.StatusConflict,
			expectedBody:   `{"message":"User mail already exists"}`,
		},
		{
			name: "Internal Server Error",
			body: `{"first_name":"Test"}`,
			mock: func() {
				mockUserService.EXPECT().UpdateUserByID(gomock.Any(), gomock.Any()).Return(errors.New("db connection failed"))
				mockLogger.EXPECT().LoggingError(gomock.Any(), gomock.Any(), "failed to update user info by id", zapcore.ErrorLevel)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"message":"Internal server error"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mock()
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPut, "/user/info", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
			assert.JSONEq(t, tc.expectedBody, w.Body.String())
		})
	}
}

func TestUserHandler_GetUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserService := mockservice.NewMockUserService(ctrl)
	mockLogger := mockhandler.NewMockLogger(ctrl)
	handler := NewUserHandler(mockUserService, mockLogger)

	router := gin.New()
	router.GET("/users", handler.GetUsers())

	usersFromService := []model.User{
		{ID: "user-1", Email: "test1@example.com", FirstName: "A"},
		{ID: "user-2", Email: "test2@example.com", FirstName: "B"},
	}

	testCases := []struct {
		name           string
		url            string
		mock           func()
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success With Default Parameters",
			url:  "/users",
			mock: func() {
				mockUserService.EXPECT().GetUsers(gomock.Any(), "", "asc", 10, 0).Return(usersFromService, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[{"id":"user-1","email":"test1@example.com","first_name":"A"},{"id":"user-2","email":"test2@example.com","first_name":"B"}]`,
		},
		{
			name: "Success With All Custom Parameters",
			url:  "/users?email=test&offset=5&limit=20&sort_order=desc",
			mock: func() {
				mockUserService.EXPECT().GetUsers(gomock.Any(), "test", "desc", 20, 5).Return(usersFromService, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[{"id":"user-1","email":"test1@example.com","first_name":"A"},{"id":"user-2","email":"test2@example.com","first_name":"B"}]`,
		},
		{
			name:           "Bad Request Invalid Offset",
			url:            "/users?offset=abc",
			mock:           func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"message":"Offset must be an integer"}`,
		},
		{
			name:           "Bad Request Invalid Limit",
			url:            "/users?limit=xyz",
			mock:           func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"message":"Limit must be an integer"}`,
		},
		{
			name:           "Bad Request Invalid Sort Order",
			url:            "/users?sort_order=descending",
			mock:           func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"message":"Invalid sort order"}`,
		},
		{
			name: "Success With Negative Offset (defaults to 0)",
			url:  "/users?offset=-100",
			mock: func() {
				mockUserService.EXPECT().GetUsers(gomock.Any(), "", "asc", 10, 0).Return(usersFromService, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[{"id":"user-1","email":"test1@example.com","first_name":"A"},{"id":"user-2","email":"test2@example.com","first_name":"B"}]`,
		},
		{
			name: "Success With Zero Limit (defaults to 10)",
			url:  "/users?limit=0",
			mock: func() {
				mockUserService.EXPECT().GetUsers(gomock.Any(), "", "asc", 10, 0).Return(usersFromService, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[{"id":"user-1","email":"test1@example.com","first_name":"A"},{"id":"user-2","email":"test2@example.com","first_name":"B"}]`,
		},
		{
			name: "Internal Server Error",
			url:  "/users",
			mock: func() {
				mockUserService.EXPECT().GetUsers(gomock.Any(), "", "asc", 10, 0).Return(nil, errors.New("db query failed"))
				mockLogger.EXPECT().LoggingError(gomock.Any(), gomock.Any(), "failed to get users", zapcore.ErrorLevel)
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
