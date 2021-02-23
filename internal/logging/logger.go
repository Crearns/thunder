package logging

import (
	"go.uber.org/zap"
	"os"
	"strings"
)

var (
	DefaultLogger Logger
	zapLogger     *zap.Logger
)


func init() {
	switch strings.ToLower(os.Getenv("LOGGING_MODE")) {
	case "prod":
		zapLogger, _ = zap.NewProduction()
	default:
		zapLogger, _ = zap.NewDevelopment()
	}
	DefaultLogger = zapLogger.Sugar()
}

func Cleanup() {
	_ = zapLogger.Sync()
}

// Logger is used for logging formatted messages.
type Logger interface {
	// Debugf logs messages at DEBUG level.
	Debugf(format string, args ...interface{})
	// Infof logs messages at INFO level.
	Infof(format string, args ...interface{})
	// Warnf logs messages at WARN level.
	Warnf(format string, args ...interface{})
	// Errorf logs messages at ERROR level.
	Errorf(format string, args ...interface{})
	// Fatalf logs messages at FATAL level.
	Fatalf(format string, args ...interface{})
}
