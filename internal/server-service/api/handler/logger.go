package handler

import (
	"github.com/gin-gonic/gin"
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
	l.log.Log(logLevel, errDescription, data...)
}

func NewLogger(l *zap.Logger) Logger {
	return &logger{
		log: l,
	}
}
