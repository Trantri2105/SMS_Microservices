package handler

import (
	"VCS_SMS_Microservice/internal/auth-service/api/middleware"
	"bytes"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupTestContext(w *httptest.ResponseRecorder, method, path string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(method, path, nil)
	return c
}

func TestLogger_LoggingError(t *testing.T) {
	testCases := []struct {
		name                 string
		setupContext         func(c *gin.Context)
		err                  error
		errDescription       string
		logLevel             zapcore.Level
		expectedToContain    []string
		expectedToNotContain []string
		expectPanic          bool
	}{
		{
			name: "Success - Logs basic info when no claims in context",
			setupContext: func(c *gin.Context) {
			},
			err:            errors.New("database connection failed"),
			errDescription: "Failed to connect to the database",
			logLevel:       zapcore.ErrorLevel,
			expectedToContain: []string{
				`"level":"error"`,
				`"msg":"Failed to connect to the database"`,
				`"error":"database connection failed"`,
				`"http_method":"GET"`,
				`"http_path":"/test-path"`,
			},
			expectedToNotContain: []string{
				"user_id",
				"scopes",
			},
		},
		{
			name: "Success - Logs user info when claims are present in context",
			setupContext: func(c *gin.Context) {
				claims := jwt.MapClaims{
					"user_id": "user-123",
					"scopes":  []string{"read:data", "write:data"},
				}
				c.Set(middleware.JWTClaimsContextKey, claims)
			},
			err:            errors.New("permission denied"),
			errDescription: "User does not have required permissions",
			logLevel:       zapcore.WarnLevel,
			expectedToContain: []string{
				`"level":"warn"`,
				`"msg":"User does not have required permissions"`,
				`"error":"permission denied"`,
				`"http_method":"GET"`,
				`"http_path":"/test-path"`,
				`"user_id":"user-123"`,
				`"scopes":["read:data","write:data"]`,
			},
			expectedToNotContain: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buffer bytes.Buffer
			encoderConfig := zap.NewProductionEncoderConfig()
			core := zapcore.NewCore(
				zapcore.NewJSONEncoder(encoderConfig),
				zapcore.AddSync(&buffer),
				zapcore.DebugLevel,
			)
			testZapLogger := zap.New(core)
			logger := NewLogger(testZapLogger)

			w := httptest.NewRecorder()
			c := setupTestContext(w, "GET", "/test-path")
			tc.setupContext(c)

			logger.LoggingError(c, tc.err, tc.errDescription, tc.logLevel)
			logOutput := buffer.String()
			for _, expected := range tc.expectedToContain {
				assert.Contains(t, logOutput, expected)
			}
			for _, notExpected := range tc.expectedToNotContain {
				assert.NotContains(t, logOutput, notExpected)
			}
		})
	}
}
