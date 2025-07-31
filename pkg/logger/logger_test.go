package logger

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"path/filepath"
	"testing"
)

func TestNewReopenableWriteSyncer(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-logs")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	logFilePath := filepath.Join(tempDir, "app.log")
	t.Run("successful creation", func(t *testing.T) {
		ws, err := NewReopenableWriteSyncer(logFilePath)
		require.NoError(t, err)
		require.NotNil(t, ws)
		defer ws.Close()
		_, err = os.Stat(logFilePath)
		assert.NoError(t, err)
	})
	t.Run("path is a directory", func(t *testing.T) {
		ws, err := NewReopenableWriteSyncer(tempDir)
		assert.Error(t, err)
		assert.Nil(t, ws)
	})
}

func TestReopenableWriteSyncer_WriteAndReload(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-reload")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	logFilePath := filepath.Join(tempDir, "app.log")
	reloadedLogFilePath := filepath.Join(tempDir, "app.log.1")

	ws, err := NewReopenableWriteSyncer(logFilePath)
	require.NoError(t, err)
	defer ws.Close()

	_, err = ws.Write([]byte("firstLine\n"))
	require.NoError(t, err)

	err = os.Rename(logFilePath, reloadedLogFilePath)
	require.NoError(t, err)

	err = ws.Reload()
	require.NoError(t, err)

	_, err = ws.Write([]byte("secondLine\n"))
	require.NoError(t, err)
	ws.Sync()

	contentOld, err := os.ReadFile(reloadedLogFilePath)
	require.NoError(t, err)
	assert.Equal(t, "firstLine\n", string(contentOld))

	contentNew, err := os.ReadFile(logFilePath)
	require.NoError(t, err)
	assert.Equal(t, "secondLine\n", string(contentNew))
}

func TestNewLogger(t *testing.T) {
	ws, err := NewReopenableWriteSyncer(os.DevNull)
	require.NoError(t, err)
	defer ws.Close()

	testCases := []struct {
		name          string
		logLevel      string
		expectedLevel zapcore.Level
	}{
		{"debug level", "debug", zap.DebugLevel},
		{"info level", "info", zap.InfoLevel},
		{"warn level", "warn", zap.WarnLevel},
		{"error level", "error", zap.ErrorLevel},
		{"fatal level", "fatal", zap.FatalLevel},
		{"invalid level", "invalid", zap.InfoLevel},
		{"empty level", "", zap.InfoLevel},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logger := NewLogger(tc.logLevel, ws)
			require.NotNil(t, logger)

			isEnabled := logger.Core().Enabled(tc.expectedLevel)
			assert.True(t, isEnabled, "expected level %s should be enabled", tc.expectedLevel)
		})
	}
}
