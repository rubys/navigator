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

// Tests for previously uncovered logging helpers

func TestLogRequestWithClient(t *testing.T) {
	output := captureLog(func() {
		LogRequestWithClient("POST", "/api/users", "req-456", "192.168.1.1")
	})

	if !strings.Contains(output, "POST") {
		t.Error("Expected log to contain method")
	}
	if !strings.Contains(output, "/api/users") {
		t.Error("Expected log to contain path")
	}
	if !strings.Contains(output, "req-456") {
		t.Error("Expected log to contain request_id")
	}
	if !strings.Contains(output, "192.168.1.1") {
		t.Error("Expected log to contain client_ip")
	}
}

func TestLogProxyRequest(t *testing.T) {
	output := captureLog(func() {
		LogProxyRequest("GET", "/api/data", "http://backend:8080")
	})

	if !strings.Contains(output, "Proxying") {
		t.Error("Expected log to mention proxying")
	}
	if !strings.Contains(output, "GET") {
		t.Error("Expected log to contain method")
	}
	if !strings.Contains(output, "/api/data") {
		t.Error("Expected log to contain path")
	}
	if !strings.Contains(output, "backend") {
		t.Error("Expected log to contain target")
	}
}

func TestLogProxyError(t *testing.T) {
	output := captureLog(func() {
		LogProxyError("http://backend:8080", errors.New("connection refused"))
	})

	if !strings.Contains(output, "Proxy error") {
		t.Error("Expected log to mention proxy error")
	}
	if !strings.Contains(output, "backend") {
		t.Error("Expected log to contain target")
	}
	if !strings.Contains(output, "connection refused") {
		t.Error("Expected log to contain error message")
	}
}

func TestLogProcessStart(t *testing.T) {
	output := captureLog(func() {
		LogProcessStart("redis", "redis-server", []string{"--port", "6379"})
	})

	if !strings.Contains(output, "Starting process") {
		t.Error("Expected log to mention starting")
	}
	if !strings.Contains(output, "redis") {
		t.Error("Expected log to contain process name")
	}
	if !strings.Contains(output, "redis-server") {
		t.Error("Expected log to contain command")
	}
}

func TestLogProcessRestart(t *testing.T) {
	output := captureLog(func() {
		LogProcessRestart("worker", 5)
	})

	if !strings.Contains(output, "Auto-restarting") {
		t.Error("Expected log to mention auto-restart")
	}
	if !strings.Contains(output, "worker") {
		t.Error("Expected log to contain process name")
	}
	if !strings.Contains(output, "5") {
		t.Error("Expected log to contain delay")
	}
}

func TestLogProcessStop(t *testing.T) {
	output := captureLog(func() {
		LogProcessStop("sidekiq")
	})

	if !strings.Contains(output, "Stopping") {
		t.Error("Expected log to mention stopping")
	}
	if !strings.Contains(output, "sidekiq") {
		t.Error("Expected log to contain process name")
	}
}

func TestLogWebAppReady(t *testing.T) {
	output := captureLog(func() {
		LogWebAppReady("tenant1", 4001)
	})

	if !strings.Contains(output, "ready") {
		t.Error("Expected log to mention ready")
	}
	if !strings.Contains(output, "tenant1") {
		t.Error("Expected log to contain tenant")
	}
	if !strings.Contains(output, "4001") {
		t.Error("Expected log to contain port")
	}
}

func TestLogWebAppStop(t *testing.T) {
	output := captureLog(func() {
		LogWebAppStop("tenant2")
	})

	if !strings.Contains(output, "Stopping") {
		t.Error("Expected log to mention stopping")
	}
	if !strings.Contains(output, "tenant2") {
		t.Error("Expected log to contain tenant")
	}
}

func TestLogWebAppIdle(t *testing.T) {
	output := captureLog(func() {
		LogWebAppIdle("tenant3", "5m30s")
	})

	if !strings.Contains(output, "idle") {
		t.Error("Expected log to mention idle")
	}
	if !strings.Contains(output, "tenant3") {
		t.Error("Expected log to contain tenant")
	}
	if !strings.Contains(output, "5m30s") {
		t.Error("Expected log to contain idle time")
	}
}

func TestLogConfigLoaded(t *testing.T) {
	output := captureLog(func() {
		LogConfigLoaded("/etc/navigator/config.yml")
	})

	if !strings.Contains(output, "loaded") {
		t.Error("Expected log to mention loaded")
	}
	if !strings.Contains(output, "config.yml") {
		t.Error("Expected log to contain config path")
	}
}

func TestLogConfigUpdate(t *testing.T) {
	output := captureLog(func() {
		LogConfigUpdate("AppManager", "idleTimeout", "10m", "portRange", "4000-4099")
	})

	if !strings.Contains(output, "Updated") {
		t.Error("Expected log to mention updated")
	}
	if !strings.Contains(output, "AppManager") {
		t.Error("Expected log to contain component")
	}
	if !strings.Contains(output, "idleTimeout") {
		t.Error("Expected log to contain detail key")
	}
}

func TestLogServerReady(t *testing.T) {
	output := captureLog(func() {
		LogServerReady("0.0.0.0", 8080)
	})

	if !strings.Contains(output, "ready") {
		t.Error("Expected log to mention ready")
	}
	if !strings.Contains(output, "0.0.0.0") {
		t.Error("Expected log to contain host")
	}
	if !strings.Contains(output, "8080") {
		t.Error("Expected log to contain port")
	}
}

func TestLogServerShutdown(t *testing.T) {
	output := captureLog(func() {
		LogServerShutdown()
	})

	if !strings.Contains(output, "Shutting down") {
		t.Error("Expected log to mention shutting down")
	}
}

func TestLogServerGracefulShutdown(t *testing.T) {
	output := captureLog(func() {
		LogServerGracefulShutdown()
	})

	if !strings.Contains(output, "gracefully") {
		t.Error("Expected log to mention graceful shutdown")
	}
}

func TestLogHookError(t *testing.T) {
	output := captureLog(func() {
		LogHookError("start", "failed-command", errors.New("exit code 1"), "error output")
	})

	if !strings.Contains(output, "Hook execution failed") {
		t.Error("Expected log to mention hook failure")
	}
	if !strings.Contains(output, "start") {
		t.Error("Expected log to contain hook type")
	}
	if !strings.Contains(output, "failed-command") {
		t.Error("Expected log to contain command")
	}
}

func TestLogCleanup(t *testing.T) {
	output := captureLog(func() {
		LogCleanup("web applications")
	})

	if !strings.Contains(output, "Cleaning up") {
		t.Error("Expected log to mention cleaning up")
	}
	if !strings.Contains(output, "web applications") {
		t.Error("Expected log to contain component")
	}
}

func TestLogCleanupComplete(t *testing.T) {
	output := captureLog(func() {
		LogCleanupComplete("managed processes")
	})

	if !strings.Contains(output, "Cleanup complete") {
		t.Error("Expected log to mention cleanup complete")
	}
	if !strings.Contains(output, "managed processes") {
		t.Error("Expected log to contain component")
	}
}
