package handler

import (
	"VCS_SMS_Microservice/internal/auth-service/api/middleware"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger interface {
	LoggingError(c *gin.Context, err error, errDescription string, logLevel zapcore.Level)
}

type logger struct {
	log *zap.Logger
}

func (l *logger) LoggingError(c *gin.Context, err error, errDescription string, logLevel zapcore.Level) {
	var data []zapcore.Field
	data = append(data, zap.Error(err))
	data = append(data, zap.String("http_method", c.Request.Method))
	data = append(data, zap.String("http_path", c.Request.URL.Path))
	claims := c.Value(middleware.JWTClaimsContextKey)
	if claims != nil {
		m := claims.(jwt.MapClaims)
		data = append(data, zap.Any("user_id", m["user_id"]))
		data = append(data, zap.Any("scopes", m["scopes"]))
	}
	l.log.Log(logLevel, errDescription, data...)
}

func NewLogger(l *zap.Logger) Logger {
	return &logger{
		log: l,
	}
}
