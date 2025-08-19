package proxy

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rubys/navigator/internal/config"
	"github.com/rubys/navigator/internal/logger"
	"github.com/rubys/navigator/internal/manager"
	"github.com/victorspringer/http-cache"
	"github.com/victorspringer/http-cache/adapter/memory"
)

// Router handles HTTP routing with chi
type Router struct {
	Manager     *manager.PumaManager
	Showcases   *config.Showcases
	RailsRoot   string
	DbPath      string
	StoragePath string
	URLPrefix   string
	htpasswd    *HtpasswdAuth
	proxies     map[string]*httputil.ReverseProxy
	cacheClient *cache.Client
	mu          sync.RWMutex
}

// NewRouter creates a new chi router with all routes configured
func NewRouter(cfg RouterConfig) *chi.Mux {
	// Initialize memory cache for static assets (10MB max, LRU eviction)
	memcached, err := memory.NewAdapter(
		memory.AdapterWithAlgorithm(memory.LRU),
		memory.AdapterWithStorageCapacity(10*1024*1024), // 10MB of memory
	)
	if err != nil {
		logger.WithField("error", err).Warn("Failed to initialize cache, proceeding without caching")
	}

	var cacheClient *cache.Client
	if memcached != nil {
		cacheClient, err = cache.NewClient(
			cache.ClientWithAdapter(memcached),
			cache.ClientWithTTL(1*time.Hour), // Default 1 hour TTL for assets
			cache.ClientWithRefreshKey("opn"),
		)
		if err != nil {
			logger.WithField("error", err).Warn("Failed to create cache client")
			cacheClient = nil
		} else {
			logger.Info("HTTP cache initialized with 100MB memory store and 1h TTL")
		}
	}

	h := &Router{
		Manager:     cfg.Manager,
		Showcases:   cfg.Showcases,
		RailsRoot:   cfg.RailsRoot,
		DbPath:      cfg.DbPath,
		StoragePath: cfg.StoragePath,
		URLPrefix:   cfg.URLPrefix,
		proxies:     make(map[string]*httputil.ReverseProxy),
		cacheClient: cacheClient,
	}

	// Load htpasswd if specified
	if cfg.HtpasswdFile != "" {
		logger.WithField("file", cfg.HtpasswdFile).Info("Loading htpasswd file")
		if auth, err := LoadHtpasswd(cfg.HtpasswdFile); err == nil {
			h.htpasswd = auth
			logger.Info("Successfully loaded htpasswd file")
		} else {
			logger.WithField("error", err).Warn("Failed to load htpasswd file")
		}
	}

	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(h.structuredLogRequest)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5, "text/html", "text/css", "text/javascript", "application/json"))

	// Add cache middleware for static assets
	if h.cacheClient != nil {
		r.Use(h.cacheMiddleware)
	}

	// Health checks
	r.Get("/up", h.healthCheck)
	r.Get("/health", h.healthCheck)

	// Static file server with try_files logic
	r.Get("/*", h.handleRequest)
	r.Post("/*", h.handleRequest)
	r.Put("/*", h.handleRequest)
	r.Delete("/*", h.handleRequest)
	r.Patch("/*", h.handleRequest)

	return r
}

// structuredLogRequest logs incoming requests with structured data
func (h *Router) structuredLogRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		duration := time.Since(start)

		logger.WithFields(map[string]interface{}{
			"method":      r.Method,
			"path":        r.URL.Path,
			"remote_addr": r.RemoteAddr,
			"status":      ww.Status(),
			"duration_ms": duration.Milliseconds(),
			"user_agent":  r.UserAgent(),
			"request_id":  middleware.GetReqID(r.Context()),
		}).Info("HTTP request")
	})
}

// cacheMiddleware handles HTTP caching for static assets
func (h *Router) cacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only cache GET requests for static assets
		if r.Method != http.MethodGet || !h.isCacheableAsset(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Use the http-cache middleware for static assets
		cacheHandler := h.cacheClient.Middleware(next)
		cacheHandler.ServeHTTP(w, r)
	})
}

// isCacheableAsset checks if the path is for a cacheable static asset
func (h *Router) isCacheableAsset(path string) bool {
	return h.isAssetPath(path) || strings.HasSuffix(path, ".ico") || strings.HasSuffix(path, ".png") ||
		strings.HasSuffix(path, ".jpg") || strings.HasSuffix(path, ".gif") || strings.HasSuffix(path, ".svg")
}

// isLongTermAsset checks if the asset should be cached for a longer period
func (h *Router) isLongTermAsset(path string) bool {
	// Assets with fingerprints in the name can be cached longer
	return strings.Contains(path, "/assets/") && (strings.Contains(path, "-") || strings.Contains(path, "_"))
}

// healthCheck handles health check endpoints
func (h *Router) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("OK"))
}

// handleRequest is the main request handler
func (h *Router) handleRequest(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Handle URL prefix if configured
	if h.URLPrefix != "" && strings.HasPrefix(path, h.URLPrefix) {
		path = strings.TrimPrefix(path, h.URLPrefix)
		if path == "" {
			path = "/"
		}
		r.URL.Path = path
	}

	// Special routes
	if path == "/" {
		http.Redirect(w, r, "/studios/", http.StatusFound)
		return
	}

	// Clean path for tenant lookup
	cleanPath := strings.TrimPrefix(path, "/")

	// Special handling for /studios/ paths - all should go to index tenant
	var tenant *config.Tenant
	if strings.HasPrefix(cleanPath, "studios/") || cleanPath == "studios" {
		tenant = h.Showcases.GetTenant("index")
		if tenant != nil {
			logger.WithFields(map[string]interface{}{
				"tenant": "index",
				"path":   cleanPath,
			}).Debug("Routing /studios/* to index tenant")
		}
	} else {
		// Try to serve static files first
		if h.serveStaticFile(w, r) {
			return
		}

		// Look up tenant
		tenant = h.Showcases.GetTenantByPath(cleanPath)
		if tenant != nil {
			logger.WithFields(map[string]interface{}{
				"tenant": tenant.Label,
				"scope":  tenant.Scope,
				"path":   cleanPath,
			}).Debug("Found tenant for request")
		}
	}

	if tenant == nil {
		http.NotFound(w, r)
		return
	}

	// Check authentication if required
	if h.htpasswd != nil && !h.isPublicPath(path) {
		if !h.htpasswd.Authenticate(r) {
			h.htpasswd.RequireAuth(w, r)
			return
		}
	}

	// Proxy to tenant
	h.proxyToTenant(w, r, tenant)
}

// proxyToTenant proxies the request to the appropriate Puma process
func (h *Router) proxyToTenant(w http.ResponseWriter, r *http.Request, tenant *config.Tenant) {
	// Get or start Puma process
	process, err := h.Manager.GetOrStart(tenant)
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"tenant": tenant.Label,
			"error":  err,
		}).Error("Failed to get Puma process")
		http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
		return
	}

	// Update last used time
	process.Touch()

	// Get or create proxy
	proxy := h.getOrCreateProxy(tenant.Label, process.Port)

	// Set X-Sendfile support headers
	r.Header.Set("X-Sendfile-Type", "X-Accel-Redirect")
	r.Header.Set("X-Accel-Mapping", fmt.Sprintf("%s/public/=/internal/", h.RailsRoot))

	// Create custom error handler for automatic recovery
	errorHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		logger.WithFields(map[string]interface{}{
			"path":   r.URL.Path,
			"tenant": tenant.Label,
			"error":  err,
		}).Error("Proxy error occurred")

		// Check if this is a connection refused error (process died)
		if strings.Contains(err.Error(), "connection refused") || strings.Contains(err.Error(), "dial tcp") {
			logger.WithField("tenant", tenant.Label).Warn("Connection refused, attempting to restart process")

			// Clear the cached proxy
			h.mu.Lock()
			delete(h.proxies, tenant.Label)
			h.mu.Unlock()

			// Attempt to restart the process
			if newProcess, restartErr := h.Manager.GetOrStart(tenant); restartErr == nil {
				// Process restarted successfully, create new proxy and retry
				newProxy := h.getOrCreateProxy(tenant.Label, newProcess.Port)
				logger.WithFields(map[string]interface{}{
					"tenant": tenant.Label,
					"port":   newProcess.Port,
				}).Info("Process restarted successfully, retrying request")
				newProxy.ServeHTTP(w, r)
				return
			}

			logger.WithField("tenant", tenant.Label).Error("Failed to restart process")
		}

		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}

	// Set the error handler
	proxy.ErrorHandler = errorHandler

	// Proxy the request
	proxy.ServeHTTP(w, r)
}

// getOrCreateProxy gets or creates a reverse proxy for a Puma instance
func (h *Router) getOrCreateProxy(label string, port int) *httputil.ReverseProxy {
	h.mu.RLock()
	proxy, exists := h.proxies[label]
	h.mu.RUnlock()

	if exists {
		return proxy
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Check again with write lock
	if proxy, exists = h.proxies[label]; exists {
		return proxy
	}

	// Create new proxy
	target := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("localhost:%d", port),
	}

	proxy = httputil.NewSingleHostReverseProxy(target)

	// Configure transport for better performance
	proxy.Transport = &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  true, // We handle compression ourselves
	}

	// Add response modifier for security headers and X-Accel-Redirect handling
	proxy.ModifyResponse = func(resp *http.Response) error {
		resp.Header.Set("X-Frame-Options", "SAMEORIGIN")
		resp.Header.Set("X-Content-Type-Options", "nosniff")
		resp.Header.Set("X-XSS-Protection", "1; mode=block")
		
		// Handle X-Accel-Redirect (nginx internal redirect)
		if accelRedirect := resp.Header.Get("X-Accel-Redirect"); accelRedirect != "" {
			// Remove the X-Accel-Redirect header from the response
			resp.Header.Del("X-Accel-Redirect")
			
			// Parse the internal redirect path
			// Format is typically /internal/path/to/file
			if strings.HasPrefix(accelRedirect, "/internal/") {
				filePath := strings.TrimPrefix(accelRedirect, "/internal/")
				fullPath := filepath.Join(h.RailsRoot, "public", filePath)
				
				// Open and read the file
				fileData, err := os.ReadFile(fullPath)
				if err != nil {
					logger.WithFields(map[string]interface{}{
						"path":  fullPath,
						"error": err,
					}).Error("Failed to read X-Accel-Redirect file")
					resp.StatusCode = http.StatusNotFound
					resp.Body = io.NopCloser(strings.NewReader("Not Found"))
					return nil
				}
				
				// Get file info for headers
				stat, err := os.Stat(fullPath)
				if err != nil {
					logger.WithFields(map[string]interface{}{
						"path":  fullPath,
						"error": err,
					}).Error("Failed to stat X-Accel-Redirect file")
					resp.StatusCode = http.StatusInternalServerError
					resp.Body = io.NopCloser(strings.NewReader("Internal Server Error"))
					return nil
				}
				
				// Replace the response body with the file content
				resp.Body = io.NopCloser(bytes.NewReader(fileData))
				resp.ContentLength = int64(len(fileData))
				resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(fileData)))
				
				// Set proper content type if not already set
				if resp.Header.Get("Content-Type") == "" || resp.Header.Get("Content-Type") == "text/html" {
					contentType := mime.TypeByExtension(filepath.Ext(fullPath))
					if contentType != "" {
						resp.Header.Set("Content-Type", contentType)
					}
				}
				
				// Set Last-Modified header
				resp.Header.Set("Last-Modified", stat.ModTime().UTC().Format(http.TimeFormat))
				
				logger.WithFields(map[string]interface{}{
					"redirect_path": accelRedirect,
					"file_path":     fullPath,
					"size":          stat.Size(),
				}).Debug("Serving X-Accel-Redirect file")
			}
		}
		
		return nil
	}

	h.proxies[label] = proxy
	return proxy
}

// serveStaticFile attempts to serve a static file with try_files logic
func (h *Router) serveStaticFile(w http.ResponseWriter, r *http.Request) bool {
	path := r.URL.Path

	// Security: prevent directory traversal
	if strings.Contains(path, "..") {
		return false
	}

	// Try files in order (nginx-style try_files behavior)
	tryPaths := []string{path}

	// If path doesn't have an extension, try with .html
	if filepath.Ext(path) == "" {
		tryPaths = append(tryPaths, path+".html")
		if strings.HasSuffix(path, "/") {
			tryPaths = append(tryPaths, path+"index.html")
		} else {
			tryPaths = append(tryPaths, path+"/index.html")
		}
	}

	for _, tryPath := range tryPaths {
		fullPath := filepath.Join(h.RailsRoot, "public", tryPath)
		if h.tryServeFile(w, r, fullPath) {
			return true
		}
	}

	return false
}

// tryServeFile attempts to serve a single file
func (h *Router) tryServeFile(w http.ResponseWriter, r *http.Request, path string) bool {
	// Use http.ServeFile for proper ETag, Content-Type, and range support
	if info, err := http.Dir(filepath.Dir(path)).Open(filepath.Base(path)); err == nil {
		defer info.Close()

		if stat, err := info.Stat(); err == nil && !stat.IsDir() {
			// Cache headers are now handled by the cache middleware
			http.ServeFile(w, r, path)
			return true
		}
	}
	return false
}

// isAssetPath checks if the path is for a static asset
func (h *Router) isAssetPath(path string) bool {
	return strings.HasPrefix(path, "/assets/") ||
		strings.HasPrefix(path, "/packs/") ||
		strings.Contains(path, ".css") ||
		strings.Contains(path, ".js") ||
		strings.Contains(path, ".png") ||
		strings.Contains(path, ".jpg") ||
		strings.Contains(path, ".ico")
}

// isPublicPath checks if the path should be publicly accessible
func (h *Router) isPublicPath(path string) bool {
	return h.isAssetPath(path) ||
		strings.HasSuffix(path, "/cable") ||
		strings.Contains(path, "/public/") ||
		strings.HasPrefix(path, "/password/") ||
		path == "/studios/" || path == "/studios"
}

// RouterConfig for the chi router
type RouterConfig struct {
	Manager      *manager.PumaManager
	Showcases    *config.Showcases
	RailsRoot    string
	DbPath       string
	StoragePath  string
	URLPrefix    string
	HtpasswdFile string
}
