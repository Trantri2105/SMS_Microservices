package middleware

import (
	mockjwt "VCS_SMS_Microservice/internal/auth-service/mocks/jwt"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupTestContext(w *httptest.ResponseRecorder) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	return c
}

func TestAuthMiddleware_ValidateAndExtractJwt(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockJwtUtils := mockjwt.NewMockUtils(ctrl)
	a := NewAuthMiddleware(mockJwtUtils)

	nextHandler := func(c *gin.Context) {
		c.Status(http.StatusOK)
	}

	testCases := []struct {
		name           string
		setupRequest   func(req *http.Request)
		mock           func()
		expectedStatus int
		expectedBody   string
		expectNextCall bool
	}{
		{
			name:           "Failure No Authorization Header",
			setupRequest:   func(req *http.Request) {},
			mock:           func() {},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"message": "Authorization header is empty"}`,
			expectNextCall: false,
		},
		{
			name: "Failure Invalid Header Format",
			setupRequest: func(req *http.Request) {
				req.Header.Set("Authorization", "invalid-token")
			},
			mock:           func() {},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"message": "Authorization header is invalid"}`,
			expectNextCall: false,
		},
		{
			name: "Failure Invalid Header Format Wrong Scheme",
			setupRequest: func(req *http.Request) {
				req.Header.Set("Authorization", "Basic some-token")
			},
			mock:           func() {},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"message": "Authorization header is invalid"}`,
			expectNextCall: false,
		},
		{
			name: "Failure Verify Token Error",
			setupRequest: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer invalid-token")
			},
			mock: func() {
				mockJwtUtils.EXPECT().VerifyToken("invalid-token").Return(nil, errors.New("token expired")).Times(1)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"message": "Invalid access token"}`,
			expectNextCall: false,
		},
		{
			name: "Success",
			setupRequest: func(req *http.Request) {
				req.Header.Set("Authorization", "Bearer valid-token")
			},
			mock: func() {
				claims := jwt.MapClaims{"user_id": "123", "scopes": []string{"read"}}
				mockJwtUtils.EXPECT().VerifyToken("valid-token").Return(claims, nil).Times(1)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "",
			expectNextCall: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c := setupTestContext(w)
			tc.setupRequest(c.Request)
			tc.mock()
			router := gin.New()
			router.GET("/", a.ValidateAndExtractJwt(), nextHandler)

			router.ServeHTTP(w, c.Request)

			assert.Equal(t, tc.expectedStatus, w.Code)
			if tc.expectedBody != "" {
				assert.JSONEq(t, tc.expectedBody, w.Body.String())
			}
			if tc.expectNextCall {
				assert.Equal(t, http.StatusOK, w.Code)
			}
		})
	}
}

func TestAuthMiddleware_CheckUserPermission(t *testing.T) {
	a := NewAuthMiddleware(nil)
	requiredScope := "users:write"
	nextHandler := func(c *gin.Context) {
		c.Status(http.StatusOK)
	}

	testCases := []struct {
		name           string
		setupContext   func(c *gin.Context)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success Has Required Scope",
			setupContext: func(c *gin.Context) {
				claims := jwt.MapClaims{"scopes": []interface{}{"users:read", "users:write"}}
				c.Set(JWTClaimsContextKey, claims)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name: "Failure Missing Required Scope",
			setupContext: func(c *gin.Context) {
				claims := jwt.MapClaims{"scopes": []interface{}{"users:read"}}
				c.Set(JWTClaimsContextKey, claims)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"message": "Permission denied"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c := setupTestContext(w)
			handler := a.CheckUserPermission(requiredScope)

			tc.setupContext(c)
			handler(c)
			if !c.IsAborted() {
				nextHandler(c)
			}
			assert.Equal(t, tc.expectedStatus, w.Code)
			if tc.expectedBody != "" {
				assert.JSONEq(t, tc.expectedBody, w.Body.String())
			}
		})
	}
}
