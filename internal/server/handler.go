package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
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
		config:        cfg,
		appManager:    appManager,
		auth:          basicAuth,
		idleManager:   idleManager,
		staticHandler: NewStaticFileHandler(cfg),
	}
}

// CreateTestHandler creates a handler with logging disabled for tests
func CreateTestHandler(cfg *config.Config, appManager *process.AppManager, basicAuth *auth.BasicAuth, idleManager *idle.Manager) http.Handler {
	return &Handler{
		config:        cfg,
		appManager:    appManager,
		auth:          basicAuth,
		idleManager:   idleManager,
		staticHandler: NewStaticFileHandler(cfg),
		disableLog:    true,
	}
}

// Handler is the main HTTP handler for Navigator
type Handler struct {
	config        *config.Config
	appManager    *process.AppManager
	auth          *auth.BasicAuth
	idleManager   *idle.Manager
	staticHandler *StaticFileHandler
	disableLog    bool // When true, suppresses access log output (for tests)
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
	recorder.disableLog = h.disableLog
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
	if h.staticHandler.ServeStatic(recorder, r) {
		return
	}

	// Try files for public paths
	if isPublic && h.staticHandler.TryFiles(recorder, r) {
		return
	}

	// Handle web application proxy
	if len(h.config.Applications.Tenants) > 0 {
		h.handleWebAppProxy(recorder, r)
	} else {
		// No tenants configured - check for static fallback (maintenance mode)
		h.staticHandler.ServeFallback(recorder, r)
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
// serveStaticFile removed - use staticHandler.ServeStatic instead
// tryFiles removed - use staticHandler.TryFiles instead
// handleStandaloneProxy removed - use Routes.ReverseProxies instead

// extractTenantFromPath extracts the tenant name from the URL path
func (h *Handler) extractTenantFromPath(path string) string {
	for _, tenant := range h.config.Applications.Tenants {
		lookingFor := "/showcase/" + tenant.Name + "/"
		if strings.HasPrefix(path, lookingFor) {
			return tenant.Name
		}
	}
	return ""
}

// handleWebAppProxy proxies requests to web applications
func (h *Handler) handleWebAppProxy(w http.ResponseWriter, r *http.Request) {
	recorder := w.(*ResponseRecorder)

	// Extract tenant name from path
	tenantName := h.extractTenantFromPath(r.URL.Path)

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
	disableLog  bool // When true, suppresses access log output
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

// NewTestResponseRecorder creates a response recorder with logging disabled for tests
func NewTestResponseRecorder(w http.ResponseWriter, idleManager *idle.Manager) *ResponseRecorder {
	recorder := NewResponseRecorder(w, idleManager)
	recorder.disableLog = true
	return recorder
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
	// Skip logging if disabled (e.g., during tests)
	if r.disableLog {
		return
	}

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
