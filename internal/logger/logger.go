package logger

import (
	"context"
	"os"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"
)

var log *logrus.Logger

// Init initializes the logger with the specified level
func Init(level string) {
	log = logrus.New()

	// Set output format
	log.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	})

	// Set output destination
	log.SetOutput(os.Stdout)

	// Parse and set log level
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		log.Warn("Invalid log level, using info")
		logLevel = logrus.InfoLevel
	}
	log.SetLevel(logLevel)

	log.WithField("level", level).Info("Logger initialized")
}

// Get returns the global logger instance
func Get() *logrus.Logger {
	if log == nil {
		Init("info") // Default fallback
	}
	return log
}

// WithFields creates a new entry with the specified fields
func WithFields(fields logrus.Fields) *logrus.Entry {
	return Get().WithFields(fields)
}

// WithField creates a new entry with a single field
func WithField(key string, value interface{}) *logrus.Entry {
	return Get().WithField(key, value)
}

// WithTenant creates a new entry with tenant information
func WithTenant(tenant string) *logrus.Entry {
	return WithField("tenant", tenant)
}

// WithRequest creates a new entry with request information from context
func WithRequest(ctx context.Context) *logrus.Entry {
	fields := logrus.Fields{}

	// Extract request ID if available
	if requestID := middleware.GetReqID(ctx); requestID != "" {
		fields["request_id"] = requestID
	}

	return WithFields(fields)
}

// Info logs an info message
func Info(msg string) {
	Get().Info(msg)
}

// Infof logs a formatted info message
func Infof(format string, args ...interface{}) {
	Get().Infof(format, args...)
}

// Warn logs a warning message
func Warn(msg string) {
	Get().Warn(msg)
}

// Warnf logs a formatted warning message
func Warnf(format string, args ...interface{}) {
	Get().Warnf(format, args...)
}

// Error logs an error message
func Error(msg string) {
	Get().Error(msg)
}

// Errorf logs a formatted error message
func Errorf(format string, args ...interface{}) {
	Get().Errorf(format, args...)
}

// Debug logs a debug message
func Debug(msg string) {
	Get().Debug(msg)
}

// Debugf logs a formatted debug message
func Debugf(format string, args ...interface{}) {
	Get().Debugf(format, args...)
}

// Fatal logs a fatal message and exits
func Fatal(msg string) {
	Get().Fatal(msg)
}

// Fatalf logs a formatted fatal message and exits
func Fatalf(format string, args ...interface{}) {
	Get().Fatalf(format, args...)
}
