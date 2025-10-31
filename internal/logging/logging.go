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

// LogProxyInvalidURL logs an invalid proxy target URL
func LogProxyInvalidURL(target string, err error) {
	slog.Error("Invalid proxy target URL",
		"target", target,
		"error", err)
}

// LogProxyHTTPRequest logs an HTTP proxy request
func LogProxyHTTPRequest(method, path, target string) {
	slog.Debug("Proxying HTTP request",
		"method", method,
		"path", path,
		"target", target)
}

// LogProxyRetryExhausted logs when proxy retry attempts are exhausted
func LogProxyRetryExhausted(target string, attempts int, duration interface{}) {
	slog.Error("Proxy failed after max retry duration",
		"target", target,
		"attempts", attempts,
		"duration", duration)
}

// LogProxyRetry logs a proxy retry attempt
func LogProxyRetry(target string, attempt int, delay interface{}) {
	slog.Debug("Proxy retry",
		"target", target,
		"attempt", attempt,
		"delay", delay)
}

// LogProxyClientDisconnected logs client disconnection during proxy
func LogProxyClientDisconnected(target string, err error) {
	slog.Debug("Client disconnected during proxy",
		"target", target,
		"error", err)
}

// LogProxyResponseBufferDisabled logs when response buffering is disabled
func LogProxyResponseBufferDisabled(size int64) {
	slog.Debug("Disabling retry due to large response size",
		"size", size)
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

// WebSocket logging helpers

// LogWebSocketProxyStart logs WebSocket proxy connection start
func LogWebSocketProxyStart(client, target, path string) {
	slog.Debug("Proxying WebSocket connection",
		"client", client,
		"target", target,
		"path", path)
}

// LogWebSocketBackendConnectError logs WebSocket backend connection failure
func LogWebSocketBackendConnectError(target string, err error) {
	slog.Error("Failed to connect to backend WebSocket",
		"target", target,
		"error", err)
}

// LogWebSocketBackendResponse logs WebSocket backend response
func LogWebSocketBackendResponse(status int) {
	slog.Debug("Backend response",
		"status", status)
}

// LogWebSocketUpgradeError logs WebSocket client upgrade failure
func LogWebSocketUpgradeError(err error) {
	slog.Error("Failed to upgrade client connection",
		"error", err)
}

// LogWebSocketProxyEstablished logs successful WebSocket proxy establishment
func LogWebSocketProxyEstablished(client, target, path string) {
	slog.Info("WebSocket proxy established",
		"client", client,
		"target", target,
		"path", path)
}

// LogWebSocketProxyEnded logs WebSocket proxy end
func LogWebSocketProxyEnded(err error) {
	if err != nil {
		slog.Debug("WebSocket proxy ended with error",
			"error", err)
	} else {
		slog.Debug("WebSocket proxy closed normally")
	}
}

// LogWebSocketConnectionStarted logs WebSocket connection start
func LogWebSocketConnectionStarted(activeCount int32) {
	slog.Debug("WebSocket connection started",
		"activeWebSockets", activeCount)
}

// LogWebSocketConnectionEnded logs WebSocket connection end
func LogWebSocketConnectionEnded(activeCount int32) {
	slog.Debug("WebSocket connection ended",
		"activeWebSockets", activeCount)
}

// LogWebSocketConnectionClosed logs WebSocket connection close
func LogWebSocketConnectionClosed(activeCount int32) {
	slog.Debug("WebSocket connection closed",
		"activeWebSockets", activeCount)
}

// LogWebSocketHijacked logs when WebSocket hijacks HTTP request
func LogWebSocketHijacked() {
	slog.Debug("WebSocket hijacked, finishing HTTP request tracking")
}

// Static file logging helpers

// LogStaticFileCheck logs static file check start
func LogStaticFileCheck(method, path string) {
	slog.Debug("Checking static file",
		"method", method,
		"path", path)
}

// LogStaticFileStripRoot logs root path stripping
func LogStaticFileStripRoot(originalPath, rootPath, newPath string) {
	slog.Debug("Stripping root path",
		"originalPath", originalPath,
		"rootPath", rootPath)
	slog.Debug("Path after stripping",
		"newPath", newPath)
}

// LogStaticFileExistenceCheck logs file existence check
func LogStaticFileExistenceCheck(fsPath, originalPath string) {
	slog.Debug("Checking file existence",
		"fsPath", fsPath,
		"originalPath", originalPath)
}

// LogStaticFileNotFound logs when static file is not found
func LogStaticFileNotFound(fsPath string, err error) {
	slog.Debug("File not found or is directory",
		"fsPath", fsPath,
		"err", err)
}

// LogStaticFileServe logs static file serving
func LogStaticFileServe(path, fsPath string) {
	slog.Debug("Serving static file",
		"path", path,
		"fsPath", fsPath)
}

// LogTryFilesCheck logs try_files check
func LogTryFilesCheck(path string) {
	slog.Debug("tryFiles checking",
		"path", path)
}

// LogTryFilesSkipExtension logs try_files skip due to extension
func LogTryFilesSkipExtension() {
	slog.Debug("tryFiles skipping - path has extension")
}

// LogTryFilesDisabled logs try_files disabled
func LogTryFilesDisabled() {
	slog.Debug("tryFiles disabled - no suffixes configured")
}

// LogTryFilesSkipTenant logs try_files skip due to tenant match
func LogTryFilesSkipTenant(tenantPath string) {
	slog.Debug("tryFiles skipping - matches tenant path",
		"tenantPath", tenantPath)
}

// LogTryFilesSearching logs try_files search start
func LogTryFilesSearching(path string) {
	slog.Debug("Trying files in public directory",
		"path", path)
}

// LogTryFilesCheckingPath logs try_files path check
func LogTryFilesCheckingPath(fsPath string) {
	slog.Debug("tryFiles checking",
		"fsPath", fsPath)
}

// LogTryFilesServe logs try_files successful serve
func LogTryFilesServe(requestPath, fsPath string) {
	slog.Info("Serving file via tryFiles",
		"requestPath", requestPath,
		"fsPath", fsPath)
}

// LogDirectoryRedirect logs when a directory is redirected to include trailing slash
func LogDirectoryRedirect(path, redirectURL string) {
	slog.Info("Redirecting directory to trailing slash",
		"path", path,
		"redirectURL", redirectURL)
}

// Maintenance page logging helpers

// LogMaintenancePageCustom logs custom maintenance page served
func LogMaintenancePageCustom(file string) {
	slog.Debug("Served custom maintenance page",
		"file", file)
}

// LogMaintenancePageFallback logs fallback maintenance page served
func LogMaintenancePageFallback() {
	slog.Debug("Served fallback maintenance page")
}

// Fly replay logging helpers

// LogFlyReplayLargeContent logs fly-replay fallback due to large content
func LogFlyReplayLargeContent(contentLength int64, method string) {
	slog.Debug("Using reverse proxy due to large content length",
		"contentLength", contentLength,
		"method", method)
}

// LogFlyReplayMissingContentLength logs fly-replay fallback due to missing content length
func LogFlyReplayMissingContentLength(method string) {
	slog.Debug("Using reverse proxy due to missing content length on body method",
		"method", method)
}

// LogFlyReplayRetryDetected logs fly-replay retry detection
func LogFlyReplayRetryDetected(retryCount string, targetMachine string) {
	slog.Info("Retry detected, serving maintenance page",
		"retryCount", retryCount,
		"targetMachine", targetMachine)
}

// LogFlyReplayResponseBody logs fly-replay response body
func LogFlyReplayResponseBody(body []byte) {
	slog.Debug("Fly replay response body",
		"body", string(body))
}

// LogFlyReplayNoAppName logs missing FLY_APP_NAME
func LogFlyReplayNoAppName() {
	slog.Debug("FLY_APP_NAME not set, cannot construct fallback proxy URL")
}

// LogFlyReplayInvalidMachineTarget logs invalid machine target format
func LogFlyReplayInvalidMachineTarget(target string) {
	slog.Debug("Invalid machine target format",
		"target", target)
}

// LogFlyReplayFallbackURLError logs fallback URL parse error
func LogFlyReplayFallbackURLError(url string, err error) {
	slog.Error("Failed to parse fly-replay fallback URL",
		"url", url,
		"error", err)
}

// LogFlyReplayFallbackProxy logs automatic fallback to reverse proxy
func LogFlyReplayFallbackProxy(target, fallbackURL string) {
	slog.Info("Using automatic reverse proxy fallback for fly-replay",
		"originalTarget", target,
		"fallbackURL", fallbackURL)
}

// Request handling logging helpers

// LogTenantExtraction logs tenant extraction result
func LogTenantExtraction(tenantName string, found bool, path string) {
	slog.Debug("Tenant extraction result",
		"tenantName", tenantName,
		"found", found,
		"path", path)
}

// LogAppStartupTimeout logs app startup timeout
func LogAppStartupTimeout(tenant string, timeout interface{}) {
	slog.Info("App still starting after timeout, serving maintenance page",
		"tenant", tenant,
		"timeout", timeout)
}

// LogInvalidTimeout logs invalid timeout configuration
func LogInvalidTimeoutTenant(tenant, value string, err error) {
	slog.Warn("Invalid tenant startup_timeout, using default",
		"tenant", tenant,
		"value", value,
		"error", err)
}

// LogInvalidTimeoutGlobal logs invalid global timeout configuration
func LogInvalidTimeoutGlobal(value string, err error) {
	slog.Warn("Invalid global startup_timeout, using default",
		"value", value,
		"error", err)
}
