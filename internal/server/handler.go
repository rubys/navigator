package server

import (
	"bufio"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/rubys/navigator/internal/auth"
	"github.com/rubys/navigator/internal/cgi"
	"github.com/rubys/navigator/internal/config"
	"github.com/rubys/navigator/internal/idle"
	"github.com/rubys/navigator/internal/logging"
	"github.com/rubys/navigator/internal/process"
	"github.com/rubys/navigator/internal/proxy"
	"github.com/rubys/navigator/internal/utils"
)

// CreateHandler creates the main HTTP handler for Navigator
func CreateHandler(cfg *config.Config, appManager *process.AppManager, basicAuth *auth.BasicAuth, idleManager *idle.Manager, currentConfigFn func() string, triggerReloadFn func(string)) http.Handler {
	h := &Handler{
		config:        cfg,
		appManager:    appManager,
		auth:          basicAuth,
		idleManager:   idleManager,
		staticHandler: NewStaticFileHandler(cfg),
	}
	h.setupCGIHandlers(currentConfigFn, triggerReloadFn)
	return h
}

// CreateTestHandler creates a handler with logging disabled for tests
func CreateTestHandler(cfg *config.Config, appManager *process.AppManager, basicAuth *auth.BasicAuth, idleManager *idle.Manager) http.Handler {
	h := &Handler{
		config:        cfg,
		appManager:    appManager,
		auth:          basicAuth,
		idleManager:   idleManager,
		staticHandler: NewStaticFileHandler(cfg),
		disableLog:    true,
	}
	h.setupCGIHandlers(nil, nil) // No reload support in tests
	return h
}

// Handler is the main HTTP handler for Navigator
type Handler struct {
	config        *config.Config
	appManager    *process.AppManager
	auth          *auth.BasicAuth
	idleManager   *idle.Manager
	staticHandler *StaticFileHandler
	cgiHandlers   map[string]*cgiRoute // Path -> CGI handler mapping
	disableLog    bool                 // When true, suppresses access log output (for tests)
}

// cgiRoute represents a CGI route with method filtering
type cgiRoute struct {
	handler *cgi.Handler
	method  string // Empty string means all methods
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
	logging.LogRequest(r.Method, r.URL.Path, requestID)

	// Handle health check endpoint
	if r.URL.Path == "/up" {
		h.handleHealthCheck(recorder, r)
		return
	}

	// Check authentication EARLY - before any routing decisions
	// This prevents authentication bypass via reverse proxies, fly-replay, etc.
	isPublic := auth.ShouldExcludeFromAuth(r.URL.Path, h.config)
	needsAuth := h.auth.IsEnabled() && !isPublic

	if needsAuth && !h.auth.CheckAuth(r) {
		recorder.SetMetadata("response_type", "auth-failure")
		h.auth.RequireAuth(recorder)
		return
	}

	// Handle rewrites and redirects
	if h.handleRewrites(recorder, r) {
		return
	}

	// Handle CGI scripts
	if h.handleCGI(recorder, r) {
		return
	}

	// Handle reverse proxies (including WebSockets)
	if h.handleReverseProxies(recorder, r) {
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
	_, _ = w.Write([]byte("OK"))
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

// getStartupTimeout determines the startup timeout for a tenant
// Priority: tenant-specific > global applications config > default
func (h *Handler) getStartupTimeout(tenant *config.Tenant) time.Duration {
	// 1. Check tenant-specific override
	if tenant != nil && tenant.StartupTimeout != "" {
		timeout := utils.ParseDurationWithContext(tenant.StartupTimeout, 0, map[string]interface{}{
			"tenant": tenant.Name,
			"type":   "startup_timeout",
		})
		if timeout > 0 {
			return timeout
		}
	}

	// 2. Check global applications config
	if h.config.Applications.StartupTimeout != "" {
		timeout := utils.ParseDurationWithDefault(h.config.Applications.StartupTimeout, 0)
		if timeout > 0 {
			return timeout
		}
	}

	// 3. Use default
	return config.DefaultStartupTimeout
}

// extractTenantFromPath extracts the tenant name from the URL path
// Returns (tenantName, found) where found indicates if a tenant was matched
func (h *Handler) extractTenantFromPath(path string) (string, bool) {
	// Find the longest matching tenant path (most specific match)
	var bestMatch string
	bestMatchLen := 0
	found := false

	for _, tenant := range h.config.Applications.Tenants {
		if strings.HasPrefix(path, tenant.Path) && len(tenant.Path) > bestMatchLen {
			bestMatch = tenant.Name
			bestMatchLen = len(tenant.Path)
			found = true
		}
	}
	return bestMatch, found
}

// handleWebAppProxy proxies requests to web applications
func (h *Handler) handleWebAppProxy(w http.ResponseWriter, r *http.Request) {
	recorder := w.(*ResponseRecorder)

	// Extract tenant name from path
	tenantName, found := h.extractTenantFromPath(r.URL.Path)

	logging.LogTenantExtraction(tenantName, found, r.URL.Path)

	if !found {
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

	// Determine startup timeout (tenant-specific override, then global, then default)
	startupTimeout := h.getStartupTimeout(app.Tenant)

	// Wait for app to be ready (with timeout)
	select {
	case <-app.ReadyChan():
		// App is ready, but check if client is still connected
		if r.Context().Err() != nil {
			// Client closed connection while waiting (similar to nginx 499)
			recorder.SetMetadata("tenant", tenantName)
			recorder.SetMetadata("response_type", "client_closed")
			w.WriteHeader(499) // Use nginx convention for client closed connection
			return
		}
		// Client still connected, continue with proxy
	case <-time.After(startupTimeout):
		// Timeout waiting for app to be ready, serve maintenance page
		logging.LogAppStartupTimeout(tenantName, startupTimeout)
		recorder.SetMetadata("tenant", tenantName)
		recorder.SetMetadata("response_type", "maintenance")
		ServeMaintenancePage(w, r, h.config)
		return
	}

	// Set metadata for logging
	recorder.SetMetadata("tenant", tenantName)
	recorder.SetMetadata("response_type", "proxy")
	recorder.SetMetadata("proxy_backend", fmt.Sprintf("tenant:%s", tenantName))

	// Determine if WebSocket tracking is enabled for this tenant
	var wsPtr *int32
	if app.ShouldTrackWebSockets(h.config.Applications.TrackWebSockets) {
		wsPtr = app.GetActiveWebSocketsPtr()
	}

	// Proxy to the web app with retry support and optional WebSocket tracking
	targetURL := fmt.Sprintf("http://localhost:%d", app.Port)
	proxy.ProxyWithWebSocketSupport(w, r, targetURL, wsPtr)
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
			logging.LogWebSocketHijacked()
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

	// Log the request using the access logging module
	LogRequest(req, r.statusCode, r.size, r.startTime, r.metadata, r.disableLog)
}

// setupCGIHandlers initializes CGI handlers from configuration
func (h *Handler) setupCGIHandlers(currentConfigFn func() string, triggerReloadFn func(string)) {
	if len(h.config.Server.CGIScripts) == 0 {
		return
	}

	h.cgiHandlers = make(map[string]*cgiRoute)

	for i, scriptCfg := range h.config.Server.CGIScripts {
		handler, err := cgi.NewHandler(&scriptCfg, currentConfigFn, triggerReloadFn)
		if err != nil {
			slog.Error("Failed to create CGI handler",
				"index", i,
				"path", scriptCfg.Path,
				"script", scriptCfg.Script,
				"error", err)
			continue
		}

		h.cgiHandlers[scriptCfg.Path] = &cgiRoute{
			handler: handler,
			method:  scriptCfg.Method,
		}

		slog.Info("Registered CGI script",
			"path", scriptCfg.Path,
			"script", scriptCfg.Script,
			"method", scriptCfg.Method,
			"user", scriptCfg.User)
	}
}

// handleCGI handles CGI script requests
func (h *Handler) handleCGI(w http.ResponseWriter, r *http.Request) bool {
	if len(h.cgiHandlers) == 0 {
		return false
	}

	route, exists := h.cgiHandlers[r.URL.Path]
	if !exists {
		return false
	}

	// Check method if specified
	if route.method != "" && !strings.EqualFold(r.Method, route.method) {
		slog.Debug("CGI method mismatch",
			"path", r.URL.Path,
			"expected", route.method,
			"got", r.Method)
		return false
	}

	// Execute CGI script
	w.(*ResponseRecorder).SetMetadata("response_type", "cgi")
	route.handler.ServeHTTP(w, r)
	return true
}
