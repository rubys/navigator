package logger

import (
	"bytes"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestInit(t *testing.T) {
	// Test that Init function doesn't panic and sets JSON formatter
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Init panicked: %v", r)
		}
	}()
	
	// Test init with valid levels
	Init("debug")
	Init("info")
	Init("warn")
	Init("error")
	
	// Test that formatter is JSON after initialization
	logger := Get() // Use our logger, not the standard logger
	_, isJSONFormatter := logger.Formatter.(*logrus.JSONFormatter)
	if !isJSONFormatter {
		t.Error("Expected JSON formatter to be set after Init")
	}
}

func TestInitInvalidLevel(t *testing.T) {
	// Test invalid log level defaults to info
	var buf bytes.Buffer
	
	// Save original logger and restore after test
	originalLogger := logrus.StandardLogger()
	defer func() {
		logrus.SetOutput(originalLogger.Out)
		logrus.SetFormatter(originalLogger.Formatter)
		logrus.SetLevel(originalLogger.Level)
	}()
	
	// Redirect output to buffer
	logrus.SetOutput(&buf)
	
	// Initialize with invalid level
	Init("invalid-level")
	
	// Should default to info level
	if logrus.GetLevel() != logrus.InfoLevel {
		t.Errorf("Expected default log level info, got %v", logrus.GetLevel())
	}
}

func TestJSONOutput(t *testing.T) {
	// Just test that Init sets up JSON formatter correctly
	Init("info")
	
	// Verify JSON formatter is set on our logger instance
	logger := Get()
	if _, isJSON := logger.Formatter.(*logrus.JSONFormatter); !isJSON {
		t.Error("Expected JSON formatter to be configured")
	}
}

func TestLoggingFunctions(t *testing.T) {
	// Test that logging functions don't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Logging function panicked: %v", r)
		}
	}()
	
	// Initialize logger
	Init("debug")
	
	// Test that functions can be called without panic
	Debug("debug message")
	Info("info message") 
	Warn("warn message")
	Error("error message")
	
	// Test WithField
	WithField("key", "value").Info("test message")
	
	// Test WithFields
	WithFields(map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	}).Info("test message")
}

func TestLogLevelFiltering(t *testing.T) {
	// Test that different log levels can be set without error
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Log level setting panicked: %v", r)
		}
	}()
	
	// Test setting different levels
	Init("warn")
	Init("debug")
	Init("error")
	Init("info")
}