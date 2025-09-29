package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rubys/navigator/internal/auth"
	"github.com/rubys/navigator/internal/config"
	"github.com/rubys/navigator/internal/idle"
	"github.com/rubys/navigator/internal/process"
	"github.com/rubys/navigator/internal/proxy"
	"github.com/rubys/navigator/internal/utils"
)

// AccessLogEntry represents a structured access log entry matching nginx format
type AccessLogEntry struct {
	Timestamp     string `json:"@timestamp"`
	ClientIP      string `json:"client_ip"`
	RemoteUser    string `json:"remote_user"`
	Method        string `json:"method"`
	URI           string `json:"uri"`
	Protocol      string `json:"protocol"`
	Status        int    `json:"status"`
	BodyBytesSent int    `json:"body_bytes_sent"`
	RequestID     string `json:"request_id"`
	RequestTime   string `json:"request_time"`
	Referer       string `json:"referer"`
	UserAgent     string `json:"user_agent"`
	FlyRequestID  string `json:"fly_request_id"`
	Tenant        string `json:"tenant,omitempty"`
	ResponseType  string `json:"response_type,omitempty"` // Type of response: proxy, static, redirect, fly-replay, auth-failure, error
	Destination   string `json:"destination,omitempty"`   // For fly-replay or redirect responses
	ProxyBackend  string `json:"proxy_backend,omitempty"` // For proxy responses
	FilePath      string `json:"file_path,omitempty"`     // For static file responses
	ErrorMessage  string `json:"error_message,omitempty"` // For error responses
}

// CreateHandler creates the main HTTP handler for Navigator
func CreateHandler(cfg *config.Config, appManager *process.AppManager, basicAuth *auth.BasicAuth, idleManager *idle.Manager) http.Handler {
	return &Handler{
		config:      cfg,
		appManager:  appManager,
		auth:        basicAuth,
		idleManager: idleManager,
	}
}

// Handler is the main HTTP handler for Navigator
type Handler struct {
	config      *config.Config
	appManager  *process.AppManager
	auth        *auth.BasicAuth
	idleManager *idle.Manager
}

// getPublicDir returns the configured public directory or the default
func (h *Handler) getPublicDir() string {
	if h.config.Server.PublicDir != "" {
		return h.config.Server.PublicDir
	}
	return config.DefaultPublicDir
}

// ServeHTTP handles all incoming HTTP requests
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Generate request ID if not present
	requestID := r.Header.Get("X-Request-Id")
	if requestID == "" {
		requestID = utils.GenerateRequestID()
		r.Header.Set("X-Request-Id", requestID)
	}

	// Create response recorder for logging and tracking
	recorder := NewResponseRecorder(w, h.idleManager)
	defer recorder.Finish(r)

	// Start idle tracking
	recorder.StartTracking()

	// Log request start
	slog.Debug("Request received",
		"method", r.Method,
		"path", r.URL.Path,
		"request_id", requestID)

	// Handle health check endpoint
	if r.URL.Path == "/up" {
		h.handleHealthCheck(recorder, r)
		return
	}

	// Handle sticky sessions (for Fly.io)
	if h.handleStickySession(recorder, r) {
		return
	}

	// Handle rewrites and redirects
	if h.handleRewrites(recorder, r) {
		return
	}

	// Handle reverse proxies (including WebSockets)
	if h.handleReverseProxies(recorder, r) {
		return
	}

	// Check authentication
	isPublic := auth.ShouldExcludeFromAuth(r.URL.Path, h.config)
	needsAuth := h.auth.IsEnabled() && !isPublic

	if needsAuth && !h.auth.CheckAuth(r) {
		recorder.SetMetadata("response_type", "auth-failure")
		h.auth.RequireAuth(recorder)
		return
	}

	// Try to serve static files
	if h.serveStaticFile(recorder, r) {
		return
	}

	// Try files for public paths
	if isPublic && h.tryFiles(recorder, r) {
		return
	}

	// Handle web application proxy
	if len(h.config.Applications.Tenants) > 0 {
		h.handleWebAppProxy(recorder, r)
	} else {
		// No tenants configured - check for static fallback (maintenance mode)
		h.handleStaticFallback(recorder, r)
	}
}

// handleHealthCheck handles the /up health check endpoint
func (h *Handler) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleStickySession handles sticky session routing for Fly.io
func (h *Handler) handleStickySession(w http.ResponseWriter, r *http.Request) bool {
	if !h.config.Server.StickySession.Enabled {
		return false
	}

	// Check if path matches sticky session paths
	matched := false
	for _, path := range h.config.Server.StickySession.Paths {
		if strings.HasPrefix(r.URL.Path, path) {
			matched = true
			break
		}
	}

	if !matched && len(h.config.Server.StickySession.Paths) > 0 {
		return false
	}

	// Implementation would check for sticky session cookie
	// and handle Fly-Replay if needed
	// This is a simplified version
	return false
}

// handleRewrites processes rewrite rules
func (h *Handler) handleRewrites(w http.ResponseWriter, r *http.Request) bool {
	for _, rule := range h.config.Server.RewriteRules {
		if !rule.Pattern.MatchString(r.URL.Path) {
			continue
		}

		// Check method restrictions
		if len(rule.Methods) > 0 {
			allowed := false
			for _, method := range rule.Methods {
				if r.Method == method {
					allowed = true
					break
				}
			}
			if !allowed {
				continue
			}
		}

		// Handle different rewrite flags
		switch {
		case rule.Flag == "redirect":
			newPath := rule.Pattern.ReplaceAllString(r.URL.Path, rule.Replacement)
			http.Redirect(w, r, newPath, http.StatusFound)
			return true

		case strings.HasPrefix(rule.Flag, "fly-replay:"):
			// Parse fly-replay flag: fly-replay:target:status
			parts := strings.Split(rule.Flag, ":")
			if len(parts) == 3 {
				target := parts[1]
				status := parts[2]

				// Use the full fly-replay implementation
				return HandleFlyReplay(w, r, target, status, h.config)
			}
			return false

		case rule.Flag == "last":
			// Internal rewrite
			r.URL.Path = rule.Pattern.ReplaceAllString(r.URL.Path, rule.Replacement)
			// Continue processing with new path
		}
	}

	return false
}

// findBestLocation removed - use Routes.ReverseProxies instead

// serveStaticFile attempts to serve a static file
func (h *Handler) serveStaticFile(w http.ResponseWriter, r *http.Request) bool {
	// Check if this is a request for static assets
	path := r.URL.Path

	slog.Debug("Checking static file",
		"path", path,
		"publicDir", h.config.Server.PublicDir,
		"rootPath", h.config.Server.RootPath)

	// Strip the root path if configured (e.g., "/showcase" prefix)
	rootPath := h.config.Server.RootPath

	if rootPath != "" && strings.HasPrefix(path, rootPath) {
		slog.Debug("Stripping root path", "originalPath", path, "rootPath", rootPath, "configured", h.config.Server.RootPath != "")
		path = strings.TrimPrefix(path, rootPath)
		if path == "" {
			path = "/"
		}
		slog.Debug("Path after stripping", "newPath", path)
	}

	// Check if file has a static extension (common static extensions)
	isStatic := false
	ext := strings.TrimPrefix(filepath.Ext(path), ".")
	if ext != "" {
		// Common static file extensions
		staticExts := []string{"js", "css", "png", "jpg", "jpeg", "gif", "svg", "ico", "pdf", "txt", "xml", "json", "woff", "woff2", "ttf", "eot"}
		for _, staticExt := range staticExts {
			if ext == staticExt {
				isStatic = true
				break
			}
		}
	}

	if !isStatic {
		return false
	}

	// Use server-level public directory (location-based serving removed)
	fsPath := filepath.Join(h.getPublicDir(), path)

	// Check if file exists
	slog.Debug("Checking file existence", "fsPath", fsPath, "originalPath", path)
	if info, err := os.Stat(fsPath); os.IsNotExist(err) || info.IsDir() {
		slog.Debug("File not found or is directory", "fsPath", fsPath, "err", err)
		return false
	}

	// Set response metadata for logging
	if recorder, ok := w.(*ResponseRecorder); ok {
		recorder.SetMetadata("response_type", "static")
		recorder.SetMetadata("file_path", fsPath)
	}

	// Set content type and serve the file
	setContentType(w, fsPath)
	http.ServeFile(w, r, fsPath)
	slog.Debug("Serving static file", "path", path, "fsPath", fsPath)
	return true
}

// tryFiles attempts to find and serve files with different extensions
func (h *Handler) tryFiles(w http.ResponseWriter, r *http.Request) bool {
	path := r.URL.Path

	slog.Debug("tryFiles checking", "path", path)

	// Only try files for paths that don't already have an extension
	if filepath.Ext(path) != "" {
		slog.Debug("tryFiles skipping - path has extension")
		return false
	}

	// Get try_files suffixes from config (location-based removed)
	var extensions []string
	if len(h.config.Server.TryFiles) > 0 {
		extensions = h.config.Server.TryFiles
	} else if h.config.Static.TryFiles.Enabled && len(h.config.Static.TryFiles.Suffixes) > 0 {
		// Use static try_files configuration (like the original navigator)
		extensions = h.config.Static.TryFiles.Suffixes
	} else {
		// Default extensions if not configured
		extensions = []string{".html", ".htm", ".txt", ".xml", ".json"}
	}

	// Skip if no extensions configured
	if len(extensions) == 0 {
		slog.Debug("tryFiles disabled - no suffixes configured")
		return false
	}

	// First, check static directories from config (like the original navigator)
	var bestStaticDir *config.StaticDir
	bestStaticDirLen := 0
	slog.Debug("Static directory matching", "path", path, "numDirectories", len(h.config.Static.Directories))
	for i, staticDir := range h.config.Static.Directories {
		hasPrefix := strings.HasPrefix(path, staticDir.Path)
		isLonger := len(staticDir.Path) > bestStaticDirLen
		slog.Debug("Checking static directory",
			"index", i,
			"staticPath", staticDir.Path,
			"dir", staticDir.Dir,
			"hasPrefix", hasPrefix,
			"pathLen", len(staticDir.Path),
			"bestLen", bestStaticDirLen,
			"isLonger", isLonger)
		if hasPrefix && isLonger {
			slog.Debug("New best match found", "staticPath", staticDir.Path, "dir", staticDir.Dir)
			bestStaticDir = &staticDir
			bestStaticDirLen = len(staticDir.Path)
		}
	}

	// If we found a matching static directory, try to serve from there
	if bestStaticDir != nil {
		slog.Debug("Found matching static directory", "path", path, "staticPath", bestStaticDir.Path, "dir", bestStaticDir.Dir)

		// Remove the URL prefix to get the relative path
		relativePath := strings.TrimPrefix(path, bestStaticDir.Path)
		if relativePath == "" {
			relativePath = "/"
		}
		if relativePath[0] != '/' {
			relativePath = "/" + relativePath
		}

		// Use server public directory as base
		publicDir := h.getPublicDir()

		// Try each extension
		for _, ext := range extensions {
			// Build the full filesystem path using static directory dir
			fsPath := filepath.Join(publicDir, bestStaticDir.Dir, relativePath+ext)
			slog.Debug("tryFiles checking static", "fsPath", fsPath)
			if info, err := os.Stat(fsPath); err == nil && !info.IsDir() {
				return h.serveFile(w, r, fsPath, path+ext)
			}
		}
		return false
	}

	slog.Debug("No static directory match found, using fallback", "path", path)
	// Fallback: check location-based public directory
	var publicDir string
	// Location-based public directory removed
	if h.config.Server.PublicDir != "" {
		publicDir = h.config.Server.PublicDir
		// Strip the root path if configured (e.g., "/showcase" prefix)
		if h.config.Server.RootPath != "" && strings.HasPrefix(path, h.config.Server.RootPath) {
			path = strings.TrimPrefix(path, h.config.Server.RootPath)
			if path == "" {
				path = "/"
			}
		}
	} else {
		// Default to public directory in current working directory
		publicDir = h.getPublicDir()
		// Strip the root path if configured (e.g., "/showcase" prefix)
		if h.config.Server.RootPath != "" && strings.HasPrefix(path, h.config.Server.RootPath) {
			path = strings.TrimPrefix(path, h.config.Server.RootPath)
			if path == "" {
				path = "/"
			}
		}
	}

	// Try each extension
	for _, ext := range extensions {
		// Build the full filesystem path
		fsPath := filepath.Join(publicDir, path+ext)
		slog.Debug("tryFiles checking", "fsPath", fsPath)
		if info, err := os.Stat(fsPath); err == nil && !info.IsDir() {
			return h.serveFile(w, r, fsPath, path+ext)
		}
	}

	return false
}

// handleStandaloneProxy removed - use Routes.ReverseProxies instead

// handleWebAppProxy proxies requests to web applications
func (h *Handler) handleWebAppProxy(w http.ResponseWriter, r *http.Request) {
	recorder := w.(*ResponseRecorder)

	// Extract tenant name from path
	tenantName := ""
	for _, tenant := range h.config.Applications.Tenants {
		lookingFor := "/showcase/" + tenant.Name + "/"
		if strings.HasPrefix(r.URL.Path, lookingFor) {
			tenantName = tenant.Name
			break
		}
	}

	slog.Debug("Tenant extraction result", "tenantName", tenantName, "path", r.URL.Path)

	if tenantName == "" {
		http.NotFound(w, r)
		return
	}

	// Get or start the web app
	app, err := h.appManager.GetOrStartApp(tenantName)
	if err != nil {
		recorder.SetMetadata("response_type", "error")
		recorder.SetMetadata("error_message", err.Error())
		http.Error(w, "Failed to start application", http.StatusInternalServerError)
		return
	}

	// Set metadata for logging
	recorder.SetMetadata("tenant", tenantName)
	recorder.SetMetadata("response_type", "proxy")
	recorder.SetMetadata("proxy_backend", fmt.Sprintf("tenant:%s", tenantName))

	// Proxy to the web app
	targetURL := fmt.Sprintf("http://localhost:%d", app.Port)
	proxy.HandleProxy(w, r, targetURL)
}

// ResponseRecorder wraps http.ResponseWriter to capture response details
type ResponseRecorder struct {
	http.ResponseWriter
	statusCode  int
	size        int
	startTime   time.Time
	metadata    map[string]interface{}
	idleManager *idle.Manager
	tracked     bool
}

// NewResponseRecorder creates a new response recorder
func NewResponseRecorder(w http.ResponseWriter, idleManager *idle.Manager) *ResponseRecorder {
	return &ResponseRecorder{
		ResponseWriter: w,
		statusCode:     200,
		startTime:      time.Now(),
		metadata:       make(map[string]interface{}),
		idleManager:    idleManager,
	}
}

// WriteHeader captures the status code
func (r *ResponseRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

// Write captures the response size
func (r *ResponseRecorder) Write(data []byte) (int, error) {
	n, err := r.ResponseWriter.Write(data)
	r.size += n
	return n, err
}

// SetMetadata sets metadata for logging
func (r *ResponseRecorder) SetMetadata(key string, value interface{}) {
	r.metadata[key] = value
}

// Hijack implements the http.Hijacker interface for WebSocket support
func (r *ResponseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := r.ResponseWriter.(http.Hijacker); ok {
		conn, rw, err := hijacker.Hijack()
		if err == nil {
			// WebSocket connection hijacked successfully
			// Finish tracking the HTTP request since it's now handled by WebSocket
			slog.Debug("WebSocket hijacked, finishing HTTP request tracking")
			r.finishTracking()
		}
		return conn, rw, err
	}
	return nil, nil, fmt.Errorf("ResponseWriter does not support hijacking")
}

func (r *ResponseRecorder) finishTracking() {
	if r.idleManager != nil && r.tracked {
		r.idleManager.RequestFinished()
		r.tracked = false
	}
}

// StartTracking starts idle tracking
func (r *ResponseRecorder) StartTracking() {
	if r.idleManager != nil && !r.tracked {
		r.idleManager.RequestStarted()
		r.tracked = true
	}
}

// Finish completes the request and logs it
func (r *ResponseRecorder) Finish(req *http.Request) {
	if r.idleManager != nil && r.tracked {
		r.idleManager.RequestFinished()
	}

	// Generate access log entry in JSON format matching the logger expectations
	r.logNavigatorRequest(req)
}

// logNavigatorRequest logs the request in JSON format matching nginx/legacy navigator format
func (r *ResponseRecorder) logNavigatorRequest(req *http.Request) {
	// Get client IP (prefer X-Forwarded-For if available)
	clientIP := req.Header.Get("X-Forwarded-For")
	if clientIP == "" {
		clientIP = req.RemoteAddr
	}
	// Clean up client IP (remove port if present)
	if idx := strings.LastIndex(clientIP, ":"); idx > 0 && strings.Count(clientIP, ":") == 1 {
		clientIP = clientIP[:idx]
	}

	// Get remote user from basic auth or headers
	remoteUser := "-"
	if user, _, ok := req.BasicAuth(); ok && user != "" {
		remoteUser = user
	} else if user := req.Header.Get("X-Remote-User"); user != "" {
		remoteUser = user
	}

	// Calculate request duration
	duration := time.Since(r.startTime)
	requestTime := fmt.Sprintf("%.3f", duration.Seconds())

	// Get request ID from headers
	requestID := req.Header.Get("X-Request-Id")
	if requestID == "" {
		requestID = req.Header.Get("X-Amzn-Trace-Id")
	}

	// Get Fly request ID
	flyRequestID := req.Header.Get("Fly-Request-Id")

	// Build URI including query string
	uri := req.URL.Path
	if req.URL.RawQuery != "" {
		uri += "?" + req.URL.RawQuery
	}

	// Create access log entry
	entry := AccessLogEntry{
		Timestamp:     time.Now().Format("2006-01-02T15:04:05.000Z07:00"),
		ClientIP:      clientIP,
		RemoteUser:    remoteUser,
		Method:        req.Method,
		URI:           uri,
		Protocol:      req.Proto,
		Status:        r.statusCode,
		BodyBytesSent: r.size,
		RequestID:     requestID,
		RequestTime:   requestTime,
		Referer:       req.Header.Get("Referer"),
		UserAgent:     req.Header.Get("User-Agent"),
		FlyRequestID:  flyRequestID,
	}

	// Add metadata from the recorder
	if tenant, ok := r.metadata["tenant"].(string); ok {
		entry.Tenant = tenant
	}
	if responseType, ok := r.metadata["response_type"].(string); ok {
		entry.ResponseType = responseType
	}
	if destination, ok := r.metadata["destination"].(string); ok {
		entry.Destination = destination
	}
	if proxyBackend, ok := r.metadata["proxy_backend"].(string); ok {
		entry.ProxyBackend = proxyBackend
	}
	if filePath, ok := r.metadata["file_path"].(string); ok {
		entry.FilePath = filePath
	}
	if errorMessage, ok := r.metadata["error_message"].(string); ok {
		entry.ErrorMessage = errorMessage
	}

	// Output JSON log entry (matching nginx/rails format)
	data, _ := json.Marshal(entry)
	fmt.Fprintln(os.Stdout, string(data))
}

// serveFile serves a specific file with appropriate headers
func (h *Handler) serveFile(w http.ResponseWriter, r *http.Request, fsPath, requestPath string) bool {
	// Set metadata for static file response
	if recorder, ok := w.(*ResponseRecorder); ok {
		recorder.SetMetadata("response_type", "static")
		recorder.SetMetadata("file_path", fsPath)
	}

	// Set appropriate content type
	setContentType(w, fsPath)

	// Serve the file
	http.ServeFile(w, r, fsPath)
	slog.Info("Serving file via tryFiles", "requestPath", requestPath, "fsPath", fsPath)
	return true
}

// handleStaticFallback handles requests when no tenants are configured (maintenance mode)
func (h *Handler) handleStaticFallback(w http.ResponseWriter, r *http.Request) {
	// Check if static fallback is configured
	if h.config.Static.TryFiles.Fallback != "" {
		fallbackPath := h.config.Static.TryFiles.Fallback

		// Build the filesystem path
		publicDir := h.getPublicDir()
		fsPath := filepath.Join(publicDir, fallbackPath)

		// Check if the fallback file exists
		if info, err := os.Stat(fsPath); err == nil && !info.IsDir() {
			if recorder, ok := w.(*ResponseRecorder); ok {
				recorder.SetMetadata("response_type", "static-fallback")
				recorder.SetMetadata("file_path", fsPath)
			}

			setContentType(w, fsPath)
			http.ServeFile(w, r, fsPath)
			slog.Info("Serving static fallback", "path", r.URL.Path, "fallback", fallbackPath, "fsPath", fsPath)
			return
		}
	}

	// No fallback configured or file not found
	http.NotFound(w, r)
}

// setContentType sets the appropriate Content-Type header based on file extension
func setContentType(w http.ResponseWriter, fsPath string) {
	ext := filepath.Ext(fsPath)
	switch ext {
	case ".js":
		w.Header().Set("Content-Type", "application/javascript")
	case ".css":
		w.Header().Set("Content-Type", "text/css")
	case ".html", ".htm":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case ".txt":
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	case ".xml":
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	case ".json":
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".gif":
		w.Header().Set("Content-Type", "image/gif")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	case ".ico":
		w.Header().Set("Content-Type", "image/x-icon")
	case ".pdf":
		w.Header().Set("Content-Type", "application/pdf")
	case ".xlsx":
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	case ".woff":
		w.Header().Set("Content-Type", "font/woff")
	case ".woff2":
		w.Header().Set("Content-Type", "font/woff2")
	case ".ttf":
		w.Header().Set("Content-Type", "font/ttf")
	case ".eot":
		w.Header().Set("Content-Type", "application/vnd.ms-fontobject")
	}
}
