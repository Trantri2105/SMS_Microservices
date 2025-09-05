package handler

import (
	"VCS_SMS_Microservice/internal/auth-service/api/middleware"
	apperrors "VCS_SMS_Microservice/internal/auth-service/errors"
	mockhandler "VCS_SMS_Microservice/internal/auth-service/mocks/api/handler"
	authmock "VCS_SMS_Microservice/internal/auth-service/mocks/service/auth"
	"VCS_SMS_Microservice/internal/auth-service/model"
	"VCS_SMS_Microservice/internal/auth-service/service"
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
	"time"
)

func TestAuthHandler_Register(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuthService := authmock.NewMockAuthService(ctrl)
	mockLogger := mockhandler.NewMockLogger(ctrl)
	handler := NewAuthHandler(mockAuthService, mockLogger)

	router := gin.New()
	router.POST("/register", handler.Register())

	testCases := []struct {
		name           string
		body           string
		mock           func()
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			body: `{"email":"test@example.com", "first_name":"test", "last_name":"test", "password":"password123", "roleIds":["role1"]}`,
			mock: func() {
				mockAuthService.EXPECT().Register(gomock.Any(), gomock.Any()).Return(model.User{
					ID:        "user1",
					Email:     "test@example.com",
					FirstName: "test",
					LastName:  "test",
					Roles:     []model.Role{{ID: "role1", Name: "Admin"}},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"id":"user1", "email":"test@example.com", "first_name":"test", "last_name":"test", "roles":[{"id":"role1", "name":"Admin"}]}`,
		},
		{
			name:           "Bad Request Invalid JSON",
			body:           `{"email":"test@example.com"`,
			mock:           func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"message":"Invalid request body"}`,
		},
		{
			name:           "Bad Request Validation Error",
			body:           `{"password":"password123"}`,
			mock:           func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"message":"The Email field is required"}`,
		},
		{
			name: "Conflict Email Already Exists",
			body: `{"email":"test@example.com", "first_name":"test", "last_name":"test", "password":"password123", "roleIds":["role1"]}`,
			mock: func() {
				mockAuthService.EXPECT().Register(gomock.Any(), gomock.Any()).Return(model.User{}, apperrors.ErrUserMailAlreadyExists)
			},
			expectedStatus: http.StatusConflict,
			expectedBody:   `{"message":"Email already exists"}`,
		},
		{
			name: "Internal Server Error",
			body: `{"email":"test@example.com", "first_name":"test", "last_name":"test", "password":"password123", "roleIds":["role1"]}`,
			mock: func() {
				mockAuthService.EXPECT().Register(gomock.Any(), gomock.Any()).Return(model.User{}, errors.New("db error"))
				mockLogger.EXPECT().LoggingError(gomock.Any(), gomock.Any(), "failed to register an user", zapcore.ErrorLevel)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"message":"Internal server error"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mock()
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
			assert.JSONEq(t, tc.expectedBody, w.Body.String())
		})
	}
}

func TestAuthHandler_Login(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuthService := authmock.NewMockAuthService(ctrl)
	mockLogger := mockhandler.NewMockLogger(ctrl)
	handler := NewAuthHandler(mockAuthService, mockLogger)

	router := gin.New()
	router.POST("/login", handler.Login())

	authResponse := service.AuthenticationResponse{
		AccessToken: "access.token", RefreshToken: "refresh.token", AccessTokenTTL: 15 * time.Minute, RefreshTokenTTL: 7 * 24 * time.Hour,
	}

	testCases := []struct {
		name           string
		body           string
		mock           func()
		expectedStatus int
		expectedBody   string
		expectCookie   bool
	}{
		{
			name: "Success",
			body: `{"email":"test@example.com", "password":"password123"}`,
			mock: func() {
				mockAuthService.EXPECT().Login(gomock.Any(), "test@example.com", "password123").Return(authResponse, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   fmt.Sprintf(`{"access_token":"access.token", "token_type":"Bearer", "expires_in":%d}`, int(authResponse.AccessTokenTTL.Seconds())),
			expectCookie:   true,
		},
		{
			name:           "Bad Request Invalid JSON",
			body:           `{"email":"test@example.com"`,
			mock:           func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"message":"Invalid request body"}`,
		},
		{
			name:           "Bad Request Validation Error",
			body:           `{"password":"password123"}`,
			mock:           func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"message":"The Email field is required"}`,
		},
		{
			name: "User Not Found",
			body: `{"email":"notfound@example.com", "password":"password123"}`,
			mock: func() {
				mockAuthService.EXPECT().Login(gomock.Any(), "notfound@example.com", "password123").Return(service.AuthenticationResponse{}, apperrors.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"message":"User not found"}`,
			expectCookie:   false,
		},
		{
			name: "Invalid Password",
			body: `{"email":"test@example.com", "password":"wrong"}`,
			mock: func() {
				mockAuthService.EXPECT().Login(gomock.Any(), "test@example.com", "wrong").Return(service.AuthenticationResponse{}, apperrors.ErrInvalidPassword)
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"message":"Invalid password"}`,
			expectCookie:   false,
		},
		{
			name: "Internal Server Error",
			body: `{"email":"test@example.com", "password":"password123"}`,
			mock: func() {
				mockAuthService.EXPECT().Login(gomock.Any(), "test@example.com", "password123").Return(service.AuthenticationResponse{}, errors.New("some error"))
				mockLogger.EXPECT().LoggingError(gomock.Any(), gomock.Any(), "failed to login", zapcore.ErrorLevel)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"message":"Internal server error"}`,
			expectCookie:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mock()
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
			assert.JSONEq(t, tc.expectedBody, w.Body.String())
			if tc.expectCookie {
				cookieHeader := w.Header().Get("Set-Cookie")
				assert.Contains(t, cookieHeader, "refresh_token=refresh.token")
				assert.Contains(t, cookieHeader, "HttpOnly")
			}
		})
	}
}

func TestAuthHandler_Logout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuthService := authmock.NewMockAuthService(ctrl)
	mockLogger := mockhandler.NewMockLogger(ctrl)
	handler := NewAuthHandler(mockAuthService, mockLogger)

	mockAuthMiddleware := func(c *gin.Context) {
		claims := jwt.MapClaims{"user_id": "user123"}
		c.Set(middleware.JWTClaimsContextKey, claims)
		c.Next()
	}

	router := gin.New()
	router.POST("/logout", mockAuthMiddleware, handler.Logout())

	testCases := []struct {
		name           string
		mock           func()
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			mock: func() {
				mockAuthService.EXPECT().Logout(gomock.Any(), "user123").Return(nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"Logout successfully"}`,
		},
		{
			name: "Internal Server Error",
			mock: func() {
				mockAuthService.EXPECT().Logout(gomock.Any(), "user123").Return(errors.New("redis failed"))
				mockLogger.EXPECT().LoggingError(gomock.Any(), gomock.Any(), "failed to logout", zapcore.ErrorLevel)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"message":"Internal server error"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mock()
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/logout", nil)
			router.ServeHTTP(w, req)
			assert.Equal(t, tc.expectedStatus, w.Code)
			assert.JSONEq(t, tc.expectedBody, w.Body.String())

			if tc.expectedStatus == http.StatusOK {
				cookieHeader := w.Header().Get("Set-Cookie")
				assert.Contains(t, cookieHeader, "Max-Age=0")
			}
		})
	}
}

func TestAuthHandler_Refresh(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuthService := authmock.NewMockAuthService(ctrl)
	mockLogger := mockhandler.NewMockLogger(ctrl)
	handler := NewAuthHandler(mockAuthService, mockLogger)

	router := gin.New()
	router.POST("/refresh", handler.Refresh())

	newAuthResponse := service.AuthenticationResponse{
		AccessToken: "new.access.token", RefreshToken: "new.refresh.token", AccessTokenTTL: 15 * time.Minute, RefreshTokenTTL: 7 * 24 * time.Hour,
	}

	testCases := []struct {
		name           string
		cookie         *http.Cookie
		mock           func()
		expectedStatus int
		expectedBody   string
	}{
		{
			name:   "Success",
			cookie: &http.Cookie{Name: "refresh_token", Value: "valid_token"},
			mock: func() {
				mockAuthService.EXPECT().Refresh(gomock.Any(), "valid_token").Return(newAuthResponse, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   fmt.Sprintf(`{"access_token":"new.access.token", "token_type":"Bearer", "expires_in":%d}`, int(newAuthResponse.AccessTokenTTL.Seconds())),
		},
		{
			name:           "Unauthorized No Cookie",
			cookie:         nil,
			mock:           func() {},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"message":"Cookie not found"}`,
		},
		{
			name:   "Unauthorized Invalid Token",
			cookie: &http.Cookie{Name: "refresh_token", Value: "invalid_token"},
			mock: func() {
				mockAuthService.EXPECT().Refresh(gomock.Any(), "invalid_token").Return(service.AuthenticationResponse{}, apperrors.ErrInvalidToken)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"message":"Invalid refresh token"}`,
		},
		{
			name:   "Refresh Token Not Found",
			cookie: &http.Cookie{Name: "refresh_token", Value: "abcdxyz"},
			mock: func() {
				mockAuthService.EXPECT().Refresh(gomock.Any(), "abcdxyz").Return(service.AuthenticationResponse{}, apperrors.ErrRefreshTokenNotFound)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"message":"Invalid refresh token"}`,
		},
		{
			name:   "User Not Found",
			cookie: &http.Cookie{Name: "refresh_token", Value: "abcdxyz"},
			mock: func() {
				mockAuthService.EXPECT().Refresh(gomock.Any(), "abcdxyz").Return(service.AuthenticationResponse{}, apperrors.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"message":"User not found"}`,
		},
		{
			name:   "Internal Server Error",
			cookie: &http.Cookie{Name: "refresh_token", Value: "abcdxyz"},
			mock: func() {
				mockAuthService.EXPECT().Refresh(gomock.Any(), "abcdxyz").Return(service.AuthenticationResponse{}, errors.New("redis failed"))
				mockLogger.EXPECT().LoggingError(gomock.Any(), gomock.Any(), gomock.Any(), zapcore.ErrorLevel)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"message":"Internal server error"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mock()
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/refresh", nil)
			if tc.cookie != nil {
				req.AddCookie(tc.cookie)
			}
			router.ServeHTTP(w, req)
			assert.Equal(t, tc.expectedStatus, w.Code)
			assert.JSONEq(t, tc.expectedBody, w.Body.String())
		})
	}
}

func TestAuthHandler_VerifyToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAuthHandler(nil, nil) // No dependencies needed for this handler

	claims := jwt.MapClaims{
		"user_id": "user-xyz",
		"scopes":  []interface{}{"scope1", "scope2"},
	}

	mockAuthMiddleware := func(c *gin.Context) {
		c.Set(middleware.JWTClaimsContextKey, claims)
		c.Next()
	}

	router := gin.New()
	router.GET("/verify", mockAuthMiddleware, handler.VerifyToken())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/verify", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "user-xyz", w.Header().Get("X-User-Id"))
	assert.Equal(t, "scope1,scope2", w.Header().Get("X-User-Scopes"))
}
