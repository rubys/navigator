package logging

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

// captureLog captures slog output for testing
func captureLog(fn func()) string {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	oldLogger := slog.Default()
	slog.SetDefault(slog.New(handler))
	defer slog.SetDefault(oldLogger)

	fn()
	return buf.String()
}

func TestLogRequest(t *testing.T) {
	output := captureLog(func() {
		LogRequest("GET", "/test", "req-123")
	})

	if !strings.Contains(output, "GET") {
		t.Error("Expected log to contain method")
	}
	if !strings.Contains(output, "/test") {
		t.Error("Expected log to contain path")
	}
	if !strings.Contains(output, "req-123") {
		t.Error("Expected log to contain request_id")
	}
}

func TestLogProcessExit(t *testing.T) {
	t.Run("Normal exit", func(t *testing.T) {
		output := captureLog(func() {
			LogProcessExit("test-process", nil)
		})

		if !strings.Contains(output, "normally") {
			t.Error("Expected log to indicate normal exit")
		}
	})

	t.Run("Error exit", func(t *testing.T) {
		output := captureLog(func() {
			LogProcessExit("test-process", errors.New("test error"))
		})

		if !strings.Contains(output, "error") {
			t.Error("Expected log to contain error")
		}
	})
}

func TestLogWebAppStart(t *testing.T) {
	output := captureLog(func() {
		LogWebAppStart("test-tenant", 4000, "ruby", "bin/rails", []string{"server"})
	})

	if !strings.Contains(output, "test-tenant") {
		t.Error("Expected log to contain tenant")
	}
	if !strings.Contains(output, "4000") {
		t.Error("Expected log to contain port")
	}
	if !strings.Contains(output, "ruby") {
		t.Error("Expected log to contain runtime")
	}
}

func TestLogConfigReload(t *testing.T) {
	output := captureLog(func() {
		LogConfigReload()
	})

	if !strings.Contains(output, "Reloading") {
		t.Error("Expected log to mention reloading")
	}
}

func TestLogServerStarting(t *testing.T) {
	output := captureLog(func() {
		LogServerStarting("localhost", 3000)
	})

	if !strings.Contains(output, "localhost") {
		t.Error("Expected log to contain host")
	}
	if !strings.Contains(output, "3000") {
		t.Error("Expected log to contain port")
	}
}

func TestLogHookExecution(t *testing.T) {
	output := captureLog(func() {
		LogHookExecution("start", "echo", []string{"test"}, "5s")
	})

	if !strings.Contains(output, "start") {
		t.Error("Expected log to contain hook type")
	}
	if !strings.Contains(output, "echo") {
		t.Error("Expected log to contain command")
	}
}

func TestLogProxyMatch(t *testing.T) {
	output := captureLog(func() {
		LogProxyMatch("/api", "http://backend:8080", false)
	})

	if !strings.Contains(output, "/api") {
		t.Error("Expected log to contain path")
	}
	if !strings.Contains(output, "backend") {
		t.Error("Expected log to contain target")
	}
}
