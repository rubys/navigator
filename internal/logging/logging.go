package logging

import "log/slog"

// Request logging helpers

// LogRequest logs an incoming HTTP request
func LogRequest(method, path, requestID string) {
	slog.Debug("Request received",
		"method", method,
		"path", path,
		"request_id", requestID)
}

// LogRequestWithClient logs an incoming HTTP request with client information
func LogRequestWithClient(method, path, requestID, clientIP string) {
	slog.Debug("Request received",
		"method", method,
		"path", path,
		"request_id", requestID,
		"client_ip", clientIP)
}

// Proxy logging helpers

// LogProxyMatch logs a successful proxy route match
func LogProxyMatch(path, target string, isWebSocket bool) {
	slog.Debug("Matched reverse proxy route",
		"path", path,
		"target", target,
		"websocket", isWebSocket)
}

// LogProxyRequest logs a proxied request
func LogProxyRequest(method, path, target string) {
	slog.Debug("Proxying request",
		"method", method,
		"path", path,
		"target", target)
}

// LogProxyError logs a proxy error
func LogProxyError(target string, err error) {
	slog.Error("Proxy error",
		"target", target,
		"error", err)
}

// Process logging helpers

// LogProcessStart logs process startup
func LogProcessStart(name, command string, args []string) {
	slog.Info("Starting process",
		"name", name,
		"command", command,
		"args", args)
}

// LogProcessExit logs process exit (normal or error)
func LogProcessExit(name string, err error) {
	if err != nil {
		slog.Error("Process exited with error",
			"name", name,
			"error", err)
	} else {
		slog.Info("Process exited normally",
			"name", name)
	}
}

// LogProcessRestart logs automatic process restart
func LogProcessRestart(name string, delay int) {
	slog.Info("Auto-restarting process",
		"name", name,
		"delay_seconds", delay)
}

// LogProcessStop logs process stop
func LogProcessStop(name string) {
	slog.Info("Stopping process",
		"name", name)
}

// Web app logging helpers

// LogWebAppStart logs web application startup
func LogWebAppStart(tenant string, port int, runtime, server string, args []string) {
	slog.Info("Starting web app",
		"tenant", tenant,
		"port", port,
		"runtime", runtime,
		"server", server,
		"args", args)
}

// LogWebAppReady logs when web app is ready
func LogWebAppReady(tenant string, port int) {
	slog.Info("Web app is ready",
		"tenant", tenant,
		"port", port)
}

// LogWebAppStop logs web app shutdown
func LogWebAppStop(tenant string) {
	slog.Info("Stopping web app",
		"tenant", tenant)
}

// LogWebAppIdle logs idle web app shutdown
func LogWebAppIdle(tenant string, idleTime string) {
	slog.Info("Stopping idle web app",
		"tenant", tenant,
		"idleTime", idleTime)
}

// Config logging helpers

// LogConfigReload logs configuration reload
func LogConfigReload() {
	slog.Info("Reloading configuration")
}

// LogConfigLoaded logs successful configuration load
func LogConfigLoaded(path string) {
	slog.Info("Configuration loaded",
		"path", path)
}

// LogConfigUpdate logs configuration update
func LogConfigUpdate(component string, details ...interface{}) {
	attrs := []interface{}{"component", component}
	attrs = append(attrs, details...)
	slog.Info("Updated configuration", attrs...)
}

// Server lifecycle logging helpers

// LogServerStarting logs server startup
func LogServerStarting(host string, port int) {
	slog.Info("Starting server",
		"host", host,
		"port", port)
}

// LogServerReady logs when server is ready
func LogServerReady(host string, port int) {
	slog.Info("Server is ready",
		"host", host,
		"port", port)
}

// LogServerShutdown logs server shutdown
func LogServerShutdown() {
	slog.Info("Shutting down server")
}

// LogServerGracefulShutdown logs graceful shutdown completion
func LogServerGracefulShutdown() {
	slog.Info("Server gracefully shut down")
}

// Hook execution logging helpers

// LogHookExecution logs hook execution start
func LogHookExecution(hookType, command string, args []string, timeout string) {
	slog.Info("Executing hook",
		"type", hookType,
		"command", command,
		"args", args,
		"timeout", timeout)
}

// LogHookError logs hook execution failure
func LogHookError(hookType, command string, err error, output string) {
	slog.Error("Hook execution failed",
		"type", hookType,
		"command", command,
		"error", err,
		"output", output)
}

// Cleanup logging helpers

// LogCleanup logs cleanup operation start
func LogCleanup(component string) {
	slog.Info("Cleaning up",
		"component", component)
}

// LogCleanupComplete logs cleanup completion
func LogCleanupComplete(component string) {
	slog.Info("Cleanup complete",
		"component", component)
}
