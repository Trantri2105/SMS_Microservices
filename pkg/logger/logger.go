package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

func NewLogger(logLevel string, fileSyncer *ReopenableWriteSyncer) *zap.Logger {
	encodeConfig := zap.NewProductionConfig()
	encodeConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encodeConfig.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encodeConfig.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	var level zapcore.Level
	switch logLevel {
	case "debug":
		level = zap.DebugLevel
	case "info":
		level = zap.InfoLevel
	case "warn":
		level = zap.WarnLevel
	case "error":
		level = zap.ErrorLevel
	case "fatal":
		level = zap.FatalLevel
	default:
		level = zap.InfoLevel
	}

	core := zapcore.NewCore(zapcore.NewJSONEncoder(encodeConfig.EncoderConfig), zapcore.NewMultiWriteSyncer(
		fileSyncer, os.Stderr), level)
	logger := zap.New(core, zap.AddCaller())

	return logger
}
