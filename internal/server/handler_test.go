package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/rubys/navigator/internal/auth"
	"github.com/rubys/navigator/internal/config"
	"github.com/rubys/navigator/internal/idle"
	"github.com/rubys/navigator/internal/process"
)

func TestResponseRecorder(t *testing.T) {
	// Create a test ResponseWriter
	recorder := httptest.NewRecorder()

	// Create ResponseRecorder
	respRecorder := NewResponseRecorder(recorder, nil, nil)

	// Test WriteHeader
	respRecorder.WriteHeader(http.StatusCreated)
	if respRecorder.statusCode != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, respRecorder.statusCode)
	}

	// Test Write
	testData := []byte("test response")
	n, err := respRecorder.Write(testData)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(testData) {
		t.Errorf("Write returned %d bytes, expected %d", n, len(testData))
	}
	if respRecorder.size != len(testData) {
		t.Errorf("Size = %d, expected %d", respRecorder.size, len(testData))
	}

	// Test SetMetadata
	respRecorder.SetMetadata("test_key", "test_value")
	if respRecorder.metadata["test_key"] != "test_value" {
		t.Errorf("Metadata not set correctly")
	}

	// Verify underlying recorder received the data
	if recorder.Code != http.StatusCreated {
		t.Errorf("Underlying recorder code = %d, expected %d", recorder.Code, http.StatusCreated)
	}
	if recorder.Body.String() != string(testData) {
		t.Errorf("Underlying recorder body = %q, expected %q", recorder.Body.String(), string(testData))
	}
}

// TestFindBestLocation was removed - legacy locations functionality no longer exists
// Reverse proxy routing is now handled via Routes.ReverseProxies configuration

func TestHealthCheckHandler(t *testing.T) {
	t.Run("synthetic response", func(t *testing.T) {
		// Configure synthetic health check response
		cfg := &config.Config{}
		cfg.Server.HealthCheck = config.HealthCheckConfig{
			Path: "/up",
			Response: &config.HealthCheckResponse{
				Status: http.StatusOK,
				Body:   "OK",
				Headers: map[string]string{
					"Content-Type": "text/plain",
				},
			},
		}

		handler := &Handler{config: cfg, staticHandler: NewStaticFileHandler(cfg)}

		req := httptest.NewRequest("GET", "/up", nil)
		recorder := httptest.NewRecorder()

		handler.handleHealthCheck(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, recorder.Code)
		}

		expectedBody := "OK"
		if recorder.Body.String() != expectedBody {
			t.Errorf("Expected body %q, got %q", expectedBody, recorder.Body.String())
		}

		expectedContentType := "text/plain"
		if recorder.Header().Get("Content-Type") != expectedContentType {
			t.Errorf("Expected Content-Type %q, got %q", expectedContentType, recorder.Header().Get("Content-Type"))
		}
	})

	t.Run("custom synthetic response", func(t *testing.T) {
		// Configure custom synthetic health check response
		cfg := &config.Config{}
		cfg.Server.HealthCheck = config.HealthCheckConfig{
			Path: "/health",
			Response: &config.HealthCheckResponse{
				Status: http.StatusServiceUnavailable,
				Body:   "Service Unavailable",
				Headers: map[string]string{
					"Content-Type":    "application/json",
					"X-Custom-Header": "test-value",
				},
			},
		}

		handler := &Handler{config: cfg, staticHandler: NewStaticFileHandler(cfg)}

		req := httptest.NewRequest("GET", "/health", nil)
		recorder := httptest.NewRecorder()

		handler.handleHealthCheck(recorder, req)

		if recorder.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, recorder.Code)
		}

		expectedBody := "Service Unavailable"
		if recorder.Body.String() != expectedBody {
			t.Errorf("Expected body %q, got %q", expectedBody, recorder.Body.String())
		}

		expectedContentType := "application/json"
		if recorder.Header().Get("Content-Type") != expectedContentType {
			t.Errorf("Expected Content-Type %q, got %q", expectedContentType, recorder.Header().Get("Content-Type"))
		}

		expectedCustomHeader := "test-value"
		if recorder.Header().Get("X-Custom-Header") != expectedCustomHeader {
			t.Errorf("Expected X-Custom-Header %q, got %q", expectedCustomHeader, recorder.Header().Get("X-Custom-Header"))
		}
	})
}

func TestSetContentType(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"test.html", "text/html; charset=utf-8"},
		{"test.css", "text/css; charset=utf-8"},       // stdlib adds charset
		{"test.js", "text/javascript; charset=utf-8"}, // stdlib uses text/javascript
		{"test.json", "application/json"},             // stdlib omits charset for json
		{"test.png", "image/png"},
		{"test.jpg", "image/jpeg"},
		{"test.gif", "image/gif"},
		{"test.svg", "image/svg+xml"},
		{"test.pdf", "application/pdf"},
		{"test.txt", "text/plain; charset=utf-8"},
		{"test.unknown", ""}, // Default case returns empty
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			SetContentType(recorder, tt.filename)

			contentType := recorder.Header().Get("Content-Type")
			if contentType != tt.expected {
				t.Errorf("For %s, expected %s, got %s", tt.filename, tt.expected, contentType)
			}
		})
	}
}

// TestLocationMatching was removed - legacy locations functionality no longer exists
// Reverse proxy routing is now handled via Routes.ReverseProxies configuration

// TestTryFilesWithStaticDirectories was removed - deprecated static.directories functionality
// Try files now uses server.try_files configuration

// TestStaticDirectoryMatching was removed - deprecated static.directories functionality
// Static file serving now uses server.public_dir directly without directory mappings

// TestTryFilesConfigurationPriority was removed - deprecated static.try_files functionality
// Try files configuration now only uses server.try_files

func TestServeStaticFileWithRootPath(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.CreateTemp("", "navigator-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.Remove(tempDir.Name())
	tempDir.Close()

	publicDir := filepath.Dir(tempDir.Name())
	testPublicDir := filepath.Join(publicDir, "test-public")
	assetsDir := filepath.Join(testPublicDir, "assets", "controllers")

	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		t.Fatalf("Failed to create test directories: %v", err)
	}
	defer os.RemoveAll(testPublicDir)

	// Create test file
	testFile := filepath.Join(assetsDir, "live_scores_controller-3e78916c.js")
	testContent := "// Test JS file content\nconsole.log('test');"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name           string
		requestPath    string
		rootPath       string
		publicDir      string
		expectedStatus int
		expectContent  bool
	}{
		{
			name:           "static file with root path stripping",
			requestPath:    "/showcase/assets/controllers/live_scores_controller-3e78916c.js",
			rootPath:       "/showcase",
			publicDir:      testPublicDir,
			expectedStatus: http.StatusOK,
			expectContent:  true,
		},
		{
			name:           "static file with empty root path (no stripping)",
			requestPath:    "/showcase/assets/controllers/live_scores_controller-3e78916c.js",
			rootPath:       "",
			publicDir:      testPublicDir,
			expectedStatus: 0, // Should not be handled by static file serving
			expectContent:  false,
		},
		{
			name:           "static file without root path prefix",
			requestPath:    "/assets/controllers/live_scores_controller-3e78916c.js",
			rootPath:       "/showcase",
			publicDir:      testPublicDir,
			expectedStatus: http.StatusOK,
			expectContent:  true,
		},
		{
			name:           "non-static file extension",
			requestPath:    "/showcase/some/path/without/extension",
			rootPath:       "/showcase",
			publicDir:      testPublicDir,
			expectedStatus: 0, // Should not be handled by serveStaticFile
			expectContent:  false,
		},
		{
			name:           "static file not found",
			requestPath:    "/showcase/assets/nonexistent.js",
			rootPath:       "/showcase",
			publicDir:      testPublicDir,
			expectedStatus: 0, // Should not be handled (file doesn't exist)
			expectContent:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create config
			cfg := &config.Config{}
			cfg.Server.RootPath = tt.rootPath
			cfg.Server.Static.PublicDir = tt.publicDir

			// Create handler
			handler := &Handler{config: cfg, staticHandler: NewStaticFileHandler(cfg)}

			// Create request and response recorder
			req := httptest.NewRequest(http.MethodGet, tt.requestPath, nil)
			recorder := httptest.NewRecorder()
			respRecorder := NewResponseRecorder(recorder, nil, nil)

			// Test serveStaticFile
			handled := handler.staticHandler.ServeStatic(respRecorder, req)

			if tt.expectedStatus == 0 {
				// Should not be handled
				if handled {
					t.Errorf("Expected file not to be handled, but it was")
				}
			} else {
				// Should be handled
				if !handled {
					t.Errorf("Expected file to be handled, but it wasn't")
					return
				}

				if recorder.Code != tt.expectedStatus {
					t.Errorf("Expected status %d, got %d", tt.expectedStatus, recorder.Code)
				}

				if tt.expectContent {
					body := recorder.Body.String()
					if body != testContent {
						t.Errorf("Expected content %q, got %q", testContent, body)
					}

					// Check Content-Type header
					contentType := recorder.Header().Get("Content-Type")
					if !strings.Contains(contentType, "javascript") && !strings.Contains(contentType, "text/plain") {
						t.Errorf("Expected javascript or text content type, got %s", contentType)
					}
				}

				// Check metadata was set
				if respRecorder.metadata["response_type"] != "static" {
					t.Errorf("Expected response_type metadata to be 'static', got %v", respRecorder.metadata["response_type"])
				}
			}
		})
	}
}

func BenchmarkResponseRecorder(b *testing.B) {
	recorder := httptest.NewRecorder()
	testData := []byte("benchmark test data")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		respRecorder := NewResponseRecorder(recorder, nil, nil)
		respRecorder.WriteHeader(http.StatusOK)
		_, _ = respRecorder.Write(testData)
		respRecorder.SetMetadata("test", "value")
	}
}

// BenchmarkFindBestLocation was removed - legacy locations functionality no longer exists
// Reverse proxy routing is now handled via Routes.ReverseProxies configuration

func TestCreateHandler(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.Listen = "3000"
	cfg.Server.Hostname = "localhost"

	appManager := &process.AppManager{}
	idleManager := &idle.Manager{}

	tests := []struct {
		name string
		auth *auth.BasicAuth
	}{
		{"Without auth", nil},
		{"With auth", &auth.BasicAuth{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := CreateTestHandler(cfg, appManager, tt.auth, idleManager)

			if handler == nil {
				t.Fatal("CreateHandler returned nil")
			}

			// Verify it implements http.Handler
			var _ http.Handler = handler

			// Verify handler type
			h, ok := handler.(*Handler)
			if !ok {
				t.Fatal("CreateHandler did not return *Handler")
			}

			if h.config != cfg {
				t.Error("Handler config not set correctly")
			}
			if h.appManager != appManager {
				t.Error("Handler appManager not set correctly")
			}
			if h.auth != tt.auth {
				t.Error("Handler auth not set correctly")
			}
			if h.idleManager != idleManager {
				t.Error("Handler idleManager not set correctly")
			}
		})
	}
}

func TestHandler_ServeHTTP_HealthCheck(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.HealthCheck = config.HealthCheckConfig{
		Path: "/up",
		Response: &config.HealthCheckResponse{
			Status: http.StatusOK,
			Body:   "OK",
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
		},
	}
	handler := CreateTestHandler(cfg, nil, nil, nil)

	req := httptest.NewRequest("GET", "/up", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("Health check returned %d, expected %d", recorder.Code, http.StatusOK)
	}

	if recorder.Body.String() != "OK" {
		t.Errorf("Health check body = %q, expected %q", recorder.Body.String(), "OK")
	}

	contentType := recorder.Header().Get("Content-Type")
	if contentType != "text/plain" {
		t.Errorf("Health check Content-Type = %q, expected %q", contentType, "text/plain")
	}
}

func TestHandler_ServeHTTP_HealthCheckBeforeAuth(t *testing.T) {
	// This test verifies the critical fix: health checks must be processed BEFORE authentication
	cfg := &config.Config{}
	cfg.Server.HealthCheck = config.HealthCheckConfig{
		Path: "/up",
		Response: &config.HealthCheckResponse{
			Status: http.StatusOK,
			Body:   "OK",
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
		},
	}

	// Enable authentication
	cfg.Auth.Enabled = true
	cfg.Auth.Realm = "Test Realm"

	// Create basic auth (but don't provide credentials)
	basicAuth := &auth.BasicAuth{}

	handler := CreateTestHandler(cfg, nil, basicAuth, nil)

	req := httptest.NewRequest("GET", "/up", nil)
	// Deliberately NOT providing auth credentials
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	// Health check should succeed WITHOUT authentication
	if recorder.Code != http.StatusOK {
		t.Errorf("Health check with auth enabled returned %d, expected %d (should bypass auth)",
			recorder.Code, http.StatusOK)
	}

	if recorder.Body.String() != "OK" {
		t.Errorf("Health check body = %q, expected %q", recorder.Body.String(), "OK")
	}
}

func TestHandler_ServeHTTP_HealthCheckNotConfigured(t *testing.T) {
	// Test that requests to /up without health check config fall through to normal routing
	cfg := &config.Config{}
	// No health check configuration
	cfg.Server.HealthCheck = config.HealthCheckConfig{
		Path: "", // Empty path = no health check
	}

	handler := CreateTestHandler(cfg, nil, nil, nil)

	req := httptest.NewRequest("GET", "/up", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	// Should NOT be handled as health check (will return 404 or fall through to tenants)
	if recorder.Code == http.StatusOK && recorder.Body.String() == "OK" {
		t.Error("Request to /up should not be handled as health check when not configured")
	}
}

func TestHandler_ServeHTTP_HealthCheckDifferentPaths(t *testing.T) {
	tests := []struct {
		name           string
		configPath     string
		requestPath    string
		expectHealthOK bool
	}{
		{
			name:           "Exact match /up",
			configPath:     "/up",
			requestPath:    "/up",
			expectHealthOK: true,
		},
		{
			name:           "Exact match /health",
			configPath:     "/health",
			requestPath:    "/health",
			expectHealthOK: true,
		},
		{
			name:           "Different path should not match",
			configPath:     "/up",
			requestPath:    "/health",
			expectHealthOK: false,
		},
		{
			name:           "Trailing slash should not match",
			configPath:     "/up",
			requestPath:    "/up/",
			expectHealthOK: false,
		},
		{
			name:           "Prefix should not match",
			configPath:     "/up",
			requestPath:    "/upstream",
			expectHealthOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.Server.HealthCheck = config.HealthCheckConfig{
				Path: tt.configPath,
				Response: &config.HealthCheckResponse{
					Status: http.StatusOK,
					Body:   "Healthy",
					Headers: map[string]string{
						"Content-Type": "text/plain",
					},
				},
			}

			handler := CreateTestHandler(cfg, nil, nil, nil)

			req := httptest.NewRequest("GET", tt.requestPath, nil)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, req)

			if tt.expectHealthOK {
				if recorder.Code != http.StatusOK {
					t.Errorf("Expected health check to succeed with 200, got %d", recorder.Code)
				}
				if recorder.Body.String() != "Healthy" {
					t.Errorf("Expected health check body, got: %s", recorder.Body.String())
				}
			} else {
				// Should NOT return the health check response
				if recorder.Body.String() == "Healthy" {
					t.Error("Should not return health check response for non-matching path")
				}
			}
		})
	}
}

func TestHandler_ServeHTTP_HealthCheckCustomHeaders(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.HealthCheck = config.HealthCheckConfig{
		Path: "/health",
		Response: &config.HealthCheckResponse{
			Status: http.StatusOK,
			Body:   `{"status":"healthy"}`,
			Headers: map[string]string{
				"Content-Type":       "application/json",
				"X-Health-Check":     "synthetic",
				"Cache-Control":      "no-cache",
				"X-Response-Time-Ms": "0",
			},
		},
	}

	handler := CreateTestHandler(cfg, nil, nil, nil)

	req := httptest.NewRequest("GET", "/health", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	// Verify all custom headers are set
	expectedHeaders := map[string]string{
		"Content-Type":       "application/json",
		"X-Health-Check":     "synthetic",
		"Cache-Control":      "no-cache",
		"X-Response-Time-Ms": "0",
	}

	for key, expected := range expectedHeaders {
		actual := recorder.Header().Get(key)
		if actual != expected {
			t.Errorf("Header %s = %q, expected %q", key, actual, expected)
		}
	}

	// Verify body
	expectedBody := `{"status":"healthy"}`
	if recorder.Body.String() != expectedBody {
		t.Errorf("Body = %q, expected %q", recorder.Body.String(), expectedBody)
	}
}

func TestHandler_ServeHTTP_RequestID(t *testing.T) {
	cfg := &config.Config{}
	handler := CreateTestHandler(cfg, nil, nil, nil)

	tests := []struct {
		name               string
		existingRequestID  string
		expectNewRequestID bool
	}{
		{"No existing request ID", "", true},
		{"With existing request ID", "existing-123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/up", nil)
			if tt.existingRequestID != "" {
				req.Header.Set("X-Request-Id", tt.existingRequestID)
			}

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			requestID := req.Header.Get("X-Request-Id")

			if tt.expectNewRequestID {
				if requestID == "" {
					t.Error("Expected new request ID to be generated")
				}
			} else {
				if requestID != tt.existingRequestID {
					t.Errorf("Request ID changed: got %q, expected %q", requestID, tt.existingRequestID)
				}
			}
		})
	}
}

func TestHandler_ServeHTTP_Authentication(t *testing.T) {
	cfg := &config.Config{}
	cfg.Auth.Enabled = true
	cfg.Auth.PublicPaths = []string{"/public/*", "/assets/*"}

	// Create a basic auth instance
	basicAuth := &auth.BasicAuth{}
	// Enable auth by setting some values (simplified)

	handler := CreateTestHandler(cfg, nil, basicAuth, nil)

	tests := []struct {
		name           string
		path           string
		expectAuthSkip bool
	}{
		{"Public path excluded", "/public/logo.png", true},
		{"Assets path excluded", "/assets/app.css", true},
		{"Health check excluded", "/up", true},
		{"Regular path requires auth", "/admin/dashboard", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, req)

			// For public paths and health check, should not get 401
			// For protected paths with no auth, should get 401 or other response
			if tt.expectAuthSkip {
				if recorder.Code == http.StatusUnauthorized {
					t.Errorf("Path %s should not require auth but got 401", tt.path)
				}
			}
			// Note: We don't test the auth failure case since we'd need proper auth setup
		})
	}
}

func TestHandler_ServeHTTP_Routing(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.HealthCheck = config.HealthCheckConfig{
		Path: "/up",
		Response: &config.HealthCheckResponse{
			Status: http.StatusOK,
			Body:   "OK",
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
		},
	}

	handler := CreateTestHandler(cfg, &process.AppManager{}, nil, nil)

	tests := []struct {
		name         string
		method       string
		path         string
		expectStatus int
	}{
		{"Health check", "GET", "/up", http.StatusOK},
		{"API path (will fail proxy)", "GET", "/api/users", http.StatusBadGateway}, // Will fail to proxy but routes correctly
		{"Static path (will fail proxy)", "GET", "/static/app.css", http.StatusBadGateway},
		{"Unknown path", "GET", "/unknown", http.StatusBadGateway}, // Will try web app proxy and fail
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, req)

			// Note: We expect most routes to fail with 502 since we don't have actual backends
			// But this tests that the routing logic works correctly
			if tt.path == "/up" && recorder.Code != tt.expectStatus {
				t.Errorf("Path %s returned %d, expected %d", tt.path, recorder.Code, tt.expectStatus)
			}
		})
	}
}

func TestMaintenanceModeHandler(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "navigator-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a maintenance page
	maintenanceHTML := `<!DOCTYPE html>
<html>
<head><title>Maintenance</title></head>
<body><h1>Site Under Maintenance</h1></body>
</html>`

	maintenancePath := filepath.Join(tempDir, "503.html")
	if err := os.WriteFile(maintenancePath, []byte(maintenanceHTML), 0644); err != nil {
		t.Fatalf("Failed to create maintenance file: %v", err)
	}

	// Create test configuration for maintenance mode using rewrite rules
	cfg := &config.Config{
		Applications: config.Applications{
			Tenants: []config.Tenant{}, // Empty tenants for maintenance mode
		},
	}
	cfg.Server.Static.PublicDir = tempDir

	// Use rewrite rules to redirect all traffic to 503.html (proper maintenance mode)
	pattern := regexp.MustCompile("^.*$")
	cfg.Server.RewriteRules = []config.RewriteRule{
		{
			Pattern:     pattern,
			Replacement: "/503.html",
			Flag:        "last",
		},
	}

	// Create handler
	handler := CreateTestHandler(cfg, nil, nil, nil)

	// Test cases
	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Root path returns maintenance page via rewrite",
			path:           "/",
			expectedStatus: http.StatusOK,
			expectedBody:   "Site Under Maintenance",
		},
		{
			name:           "Random path returns maintenance page via rewrite",
			path:           "/some/random/path",
			expectedStatus: http.StatusOK,
			expectedBody:   "Site Under Maintenance",
		},
		{
			name:           "Path with query params returns maintenance page via rewrite",
			path:           "/test?param=value",
			expectedStatus: http.StatusOK,
			expectedBody:   "Site Under Maintenance",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req, err := http.NewRequest("GET", tt.path, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			// Create response recorder
			rr := httptest.NewRecorder()

			// Serve the request
			handler.ServeHTTP(rr, req)

			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			// Check response body contains expected text
			body := rr.Body.String()
			if tt.expectedBody != "" && !strings.Contains(body, tt.expectedBody) {
				t.Errorf("Expected body to contain '%s', got: %s", tt.expectedBody, body)
			}
		})
	}
}

func TestRewriteRulesWithMaintenanceConfig(t *testing.T) {
	// Create test configuration with rewrite rules (maintenance mode style)
	cfg := &config.Config{
		Applications: config.Applications{
			Tenants: []config.Tenant{}, // Empty tenants
		},
	}

	// Add a rewrite rule that matches everything and rewrites to /503.html
	pattern := regexp.MustCompile("^.*$")
	cfg.Server.RewriteRules = []config.RewriteRule{
		{
			Pattern:     pattern,
			Replacement: "/503.html",
			Flag:        "last",
		},
	}

	// Create handler
	handler := &Handler{
		config:        cfg,
		staticHandler: NewStaticFileHandler(cfg),
	}

	// Test rewrite handling
	req, err := http.NewRequest("GET", "/test/path", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()

	// Call handleRewrites
	result := handler.handleRewrites(rr, req)

	// For "last" flag, the function should not return true (continue processing)
	if result {
		t.Error("Expected handleRewrites to return false for 'last' flag")
	}

	// Check that the path was rewritten
	if req.URL.Path != "/503.html" {
		t.Errorf("Expected path to be rewritten to /503.html, got %s", req.URL.Path)
	}
}

func TestStaticFallbackWithNoTenants(t *testing.T) {
	// Create test configuration with no tenants
	cfg := &config.Config{
		Applications: config.Applications{
			Tenants: []config.Tenant{}, // Empty tenants
		},
	}

	// Create handler
	handler := &Handler{
		config:        cfg,
		staticHandler: NewStaticFileHandler(cfg),
	}

	// Test that non-existent paths return 404 when there are no tenants
	req := httptest.NewRequest("GET", "/any/path", nil)
	rr := httptest.NewRecorder()

	// Since ServeFallback is only called after all other handlers fail,
	// we test through ServeHTTP which will eventually return 404
	handler.ServeHTTP(rr, req)

	// Should return 404 for missing paths when no tenants configured
	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestMaintenanceModeWithStaticFiles(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "navigator-maintenance-static-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a static HTML file
	staticHTML := `<!DOCTYPE html>
<html>
<head><title>Static Page</title></head>
<body><h1>This is a static page</h1></body>
</html>`

	staticPath := filepath.Join(tempDir, "static.html")
	if err := os.WriteFile(staticPath, []byte(staticHTML), 0644); err != nil {
		t.Fatalf("Failed to create static file: %v", err)
	}

	// Create a maintenance page
	maintenanceHTML := `<!DOCTYPE html>
<html>
<head><title>Maintenance</title></head>
<body><h1>Site Under Maintenance</h1></body>
</html>`

	maintenancePath := filepath.Join(tempDir, "503.html")
	if err := os.WriteFile(maintenancePath, []byte(maintenanceHTML), 0644); err != nil {
		t.Fatalf("Failed to create maintenance file: %v", err)
	}

	// Create CSS file to test static extensions
	cssContent := "body { color: blue; }"
	cssPath := filepath.Join(tempDir, "style.css")
	if err := os.WriteFile(cssPath, []byte(cssContent), 0644); err != nil {
		t.Fatalf("Failed to create CSS file: %v", err)
	}

	// Create test configuration with maintenance mode enabled
	cfg := &config.Config{
		Applications: config.Applications{
			Tenants: []config.Tenant{
				{
					Path: "/app/",
					Var:  map[string]interface{}{},
				},
			},
		},
	}
	cfg.Server.Static.PublicDir = tempDir
	cfg.Server.Static.AllowedExtensions = []string{"html", "css", "js", "png", "jpg"}
	cfg.Maintenance.Enabled = true
	cfg.Maintenance.Page = "/503.html"

	// Create handler
	handler := CreateTestHandler(cfg, nil, nil, nil)

	// Test cases
	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedBody   string
		description    string
	}{
		{
			name:           "Static HTML file served during maintenance",
			path:           "/static.html",
			expectedStatus: http.StatusOK,
			expectedBody:   "This is a static page",
			description:    "Static files should be served even in maintenance mode",
		},
		{
			name:           "Static CSS file served during maintenance",
			path:           "/style.css",
			expectedStatus: http.StatusOK,
			expectedBody:   "body { color: blue; }",
			description:    "Static CSS files should be served even in maintenance mode",
		},
		{
			name:           "Dynamic request gets maintenance page",
			path:           "/app/dashboard",
			expectedStatus: http.StatusServiceUnavailable,
			expectedBody:   "Site Under Maintenance",
			description:    "Dynamic requests should get the maintenance page",
		},
		{
			name:           "Root path gets maintenance page",
			path:           "/",
			expectedStatus: http.StatusServiceUnavailable,
			expectedBody:   "Site Under Maintenance",
			description:    "Root path should get the maintenance page when no static file exists",
		},
		{
			name:           "Non-existent path gets maintenance page",
			path:           "/nonexistent",
			expectedStatus: http.StatusServiceUnavailable,
			expectedBody:   "Site Under Maintenance",
			description:    "Non-existent paths should get the maintenance page",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req, err := http.NewRequest("GET", tt.path, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			// Create response recorder
			rr := httptest.NewRecorder()

			// Serve the request
			handler.ServeHTTP(rr, req)

			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			// Check response body contains expected text
			body := rr.Body.String()
			if tt.expectedBody != "" && !strings.Contains(body, tt.expectedBody) {
				t.Errorf("Expected body to contain '%s', got: %s", tt.expectedBody, body)
			}
		})
	}
}

// TestAssetServingIntegration tests the complete HTTP request flow for asset serving
// with root path stripping functionality. This is an integration test that verifies
// the full handler chain works correctly for the original 404 asset issue.
func TestAssetServingIntegration(t *testing.T) {
	// Create temporary directory structure mimicking showcase app
	tempDir, err := os.MkdirTemp("", "navigator-asset-integration-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create assets directory and test files
	assetsDir := filepath.Join(tempDir, "assets")
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		t.Fatalf("Failed to create assets dir: %v", err)
	}

	jsContent := "// Test JavaScript controller\nconsole.log('Live scores controller loaded');"
	jsFile := filepath.Join(assetsDir, "controllers", "live_scores_controller-3e78916c.js")
	if err := os.MkdirAll(filepath.Dir(jsFile), 0755); err != nil {
		t.Fatalf("Failed to create controllers dir: %v", err)
	}
	if err := os.WriteFile(jsFile, []byte(jsContent), 0644); err != nil {
		t.Fatalf("Failed to write JS file: %v", err)
	}

	cssContent := "/* Test CSS styles */\n.live-scores { color: blue; }"
	cssFile := filepath.Join(assetsDir, "stylesheets", "application-1a2b3c4d.css")
	if err := os.MkdirAll(filepath.Dir(cssFile), 0755); err != nil {
		t.Fatalf("Failed to create stylesheets dir: %v", err)
	}
	if err := os.WriteFile(cssFile, []byte(cssContent), 0644); err != nil {
		t.Fatalf("Failed to write CSS file: %v", err)
	}

	// Test configurations: with explicit root_path
	testCases := []struct {
		name        string
		rootPath    string
		description string
	}{
		{
			name:        "with_configured_root_path",
			rootPath:    "/showcase",
			description: "root_path explicitly configured to '/showcase'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create config for this test case
			cfg := &config.Config{
				LocationConfigMutex: sync.RWMutex{},
			}
			cfg.Server.Static.PublicDir = tempDir
			cfg.Server.RootPath = tc.rootPath
			cfg.Auth.AuthPatterns = []config.AuthPattern{}
			cfg.Server.RewriteRules = []config.RewriteRule{}

			// Create handler with all required components
			handler := &Handler{
				config:        cfg,
				auth:          &auth.BasicAuth{},
				idleManager:   idle.NewManager(cfg, "", nil),
				appManager:    &process.AppManager{},
				staticHandler: NewStaticFileHandler(cfg),
			}
			// Test cases for asset requests that should succeed
			assetTests := []struct {
				path        string
				expectedExt string
				description string
			}{
				{
					path:        "/showcase/assets/controllers/live_scores_controller-3e78916c.js",
					expectedExt: ".js",
					description: "JavaScript controller asset that was originally failing with 404",
				},
				{
					path:        "/showcase/assets/stylesheets/application-1a2b3c4d.css",
					expectedExt: ".css",
					description: "CSS stylesheet asset",
				},
			}

			for _, assetTest := range assetTests {
				t.Run(assetTest.description, func(t *testing.T) {
					// Create HTTP request
					req := httptest.NewRequest("GET", assetTest.path, nil)
					rr := httptest.NewRecorder()

					// Process request through full handler
					handler.ServeHTTP(rr, req)

					// Verify successful response
					if rr.Code != http.StatusOK {
						t.Errorf("Expected status %d for %s, got %d (%s)",
							http.StatusOK, assetTest.path, rr.Code, tc.description)
					}

					// Verify content type is set correctly
					contentType := rr.Header().Get("Content-Type")
					if assetTest.expectedExt == ".js" {
						if !strings.Contains(contentType, "javascript") && !strings.Contains(contentType, "text/plain") {
							t.Errorf("Expected JS content type for %s, got: %s", assetTest.path, contentType)
						}
					} else if assetTest.expectedExt == ".css" {
						if !strings.Contains(contentType, "css") && !strings.Contains(contentType, "text/plain") {
							t.Errorf("Expected CSS content type for %s, got: %s", assetTest.path, contentType)
						}
					}

					// Verify file content is returned
					body := rr.Body.String()
					if len(body) == 0 {
						t.Errorf("Expected non-empty response body for %s", assetTest.path)
					}

					// Verify actual content matches expected
					if assetTest.expectedExt == ".js" {
						if !strings.Contains(body, "Live scores controller loaded") {
							t.Errorf("JS file content not found in response for %s", assetTest.path)
						}
					} else if assetTest.expectedExt == ".css" {
						if !strings.Contains(body, ".live-scores") {
							t.Errorf("CSS content not found in response for %s", assetTest.path)
						}
					}
				})
			}
		})
	}
}

// TestAssetServingIntegrationErrorCases tests error scenarios in the asset serving integration
func TestAssetServingIntegrationErrorCases(t *testing.T) {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "navigator-asset-error-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &config.Config{
		LocationConfigMutex: sync.RWMutex{},
	}
	cfg.Server.Static.PublicDir = tempDir
	cfg.Server.RootPath = "/showcase"
	cfg.Auth.AuthPatterns = []config.AuthPattern{}
	cfg.Server.RewriteRules = []config.RewriteRule{}

	handler := &Handler{
		config:        cfg,
		auth:          &auth.BasicAuth{},
		idleManager:   idle.NewManager(cfg, "", nil),
		appManager:    &process.AppManager{},
		staticHandler: NewStaticFileHandler(cfg),
	}

	// Test 404 for non-existent asset
	req := httptest.NewRequest("GET", "/showcase/assets/nonexistent-file.js", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Should return 404 for missing files
	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected 404 for non-existent asset, got %d", rr.Code)
	}
}

// TestAssetServingRootPathVariations tests different root path configurations
func TestAssetServingRootPathVariations(t *testing.T) {
	// Create temporary directory with test asset
	tempDir, err := os.MkdirTemp("", "navigator-rootpath-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test asset
	assetsDir := filepath.Join(tempDir, "assets")
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		t.Fatalf("Failed to create assets dir: %v", err)
	}

	testContent := "/* test css */"
	testFile := filepath.Join(assetsDir, "test.css")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	testCases := []struct {
		name        string
		rootPath    string
		requestPath string
		shouldWork  bool
		description string
	}{
		{
			name:        "showcase_prefix",
			rootPath:    "/showcase",
			requestPath: "/showcase/assets/test.css",
			shouldWork:  true,
			description: "Standard showcase root path",
		},
		{
			name:        "app_prefix",
			rootPath:    "/app",
			requestPath: "/app/assets/test.css",
			shouldWork:  true,
			description: "Custom app root path",
		},
		{
			name:        "empty_root_no_stripping",
			rootPath:    "",
			requestPath: "/showcase/assets/test.css",
			shouldWork:  false,
			description: "Empty root path should not strip any prefix",
		},
		{
			name:        "wrong_prefix",
			rootPath:    "/showcase",
			requestPath: "/wrong/assets/test.css",
			shouldWork:  false,
			description: "Wrong prefix should not work",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{
				LocationConfigMutex: sync.RWMutex{},
			}
			cfg.Server.Static.PublicDir = tempDir
			cfg.Server.RootPath = tc.rootPath
			cfg.Auth.AuthPatterns = []config.AuthPattern{}
			cfg.Server.RewriteRules = []config.RewriteRule{}

			handler := &Handler{
				config:        cfg,
				auth:          &auth.BasicAuth{},
				idleManager:   idle.NewManager(cfg, "", nil),
				appManager:    &process.AppManager{},
				staticHandler: NewStaticFileHandler(cfg),
			}

			req := httptest.NewRequest("GET", tc.requestPath, nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if tc.shouldWork {
				if rr.Code != http.StatusOK {
					t.Errorf("Expected 200 for %s (%s), got %d", tc.requestPath, tc.description, rr.Code)
				}
				body := rr.Body.String()
				if body != testContent {
					t.Errorf("Expected '%s' in body, got: %s", testContent, body)
				}
			} else {
				if rr.Code == http.StatusOK {
					t.Errorf("Expected non-200 for %s (%s), got %d", tc.requestPath, tc.description, rr.Code)
				}
			}
		})
	}
}

func TestJSONAccessLogging(t *testing.T) {
	// Create a test config
	cfg := &config.Config{}
	cfg.Server.Listen = "3000"
	cfg.Server.Hostname = "localhost"
	cfg.Server.Static.PublicDir = "public"

	// Create handler with logging enabled (not using CreateTestHandler)
	handler := CreateHandler(cfg, nil, nil, nil, nil, func() string { return "" }, func(string) {})

	// Capture stdout to test JSON log output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Update access log writer to use the pipe
	SetAccessLogWriter(w)

	// Create test request
	req := httptest.NewRequest("GET", "/test-path?param=value", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.1")
	req.Header.Set("User-Agent", "Test-Agent/1.0")
	req.Header.Set("Referer", "https://example.com/")
	req.Header.Set("X-Request-Id", "test-request-123")
	req.Header.Set("Fly-Request-Id", "test-fly-456")
	req.RemoteAddr = "192.0.2.1:45678"

	// Create response recorder
	rr := httptest.NewRecorder()

	// Make request (will return 404 but should log)
	handler.ServeHTTP(rr, req)

	// Close writer and restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Restore access log writer
	SetAccessLogWriter(oldStdout)

	// Read captured output
	output := make([]byte, 1024)
	n, _ := r.Read(output)
	logOutput := string(output[:n])

	// Verify JSON access log was output
	if !strings.Contains(logOutput, `"@timestamp"`) {
		t.Error("Expected JSON access log with @timestamp field")
	}
	if !strings.Contains(logOutput, `"client_ip":"203.0.113.1"`) {
		t.Error("Expected client_ip to be extracted from X-Forwarded-For header")
	}
	if !strings.Contains(logOutput, `"method":"GET"`) {
		t.Error("Expected method field in JSON log")
	}
	if !strings.Contains(logOutput, `"uri":"/test-path?param=value"`) {
		t.Error("Expected full URI with query parameters in JSON log")
	}
	if !strings.Contains(logOutput, `"protocol":"HTTP/1.1"`) {
		t.Error("Expected protocol field in JSON log")
	}
	if !strings.Contains(logOutput, `"status":404`) {
		t.Error("Expected status code in JSON log")
	}
	if !strings.Contains(logOutput, `"request_id":"test-request-123"`) {
		t.Error("Expected request_id from X-Request-Id header")
	}
	if !strings.Contains(logOutput, `"fly_request_id":"test-fly-456"`) {
		t.Error("Expected fly_request_id from Fly-Request-Id header")
	}
	if !strings.Contains(logOutput, `"user_agent":"Test-Agent/1.0"`) {
		t.Error("Expected user_agent field in JSON log")
	}
	if !strings.Contains(logOutput, `"referer":"https://example.com/"`) {
		t.Error("Expected referer field in JSON log")
	}
	if !strings.Contains(logOutput, `"remote_user":"-"`) {
		t.Error("Expected remote_user field (dash for no auth) in JSON log")
	}
	if !strings.Contains(logOutput, `"request_time"`) {
		t.Error("Expected request_time field in JSON log")
	}

	// Verify it's valid JSON format (basic check)
	if !strings.HasPrefix(strings.TrimSpace(logOutput), "{") || !strings.HasSuffix(strings.TrimSpace(logOutput), "}") {
		t.Errorf("JSON log output doesn't appear to be valid JSON format: %s", logOutput)
	}
}

func TestHandler_HandleRewritesFlyReplay(t *testing.T) {
	tests := []struct {
		name              string
		path              string
		rewriteRules      []config.RewriteRule
		expectHandled     bool
		expectFlyReplay   bool
		expectStatus      int
		expectContentType string
		flyAppName        string
	}{
		{
			name: "Region-based fly-replay",
			path: "/showcase/2025/coquitlam/medal-ball/",
			rewriteRules: []config.RewriteRule{
				{
					Pattern:     regexp.MustCompile(`^/showcase/(?:2023|2024|2025|2026)/(?:bellevue|coquitlam|edmonton|everett|folsom|fremont|honolulu|livermore|millbrae|montclair|monterey|petaluma|reno|salem|sanjose|slc|stockton|vegas)(?:/.*)?$`),
					Replacement: "/showcase/2025/coquitlam/medal-ball/",
					Flag:        "fly-replay:sjc:307",
				},
			},
			expectHandled:     true,
			expectFlyReplay:   true,
			expectStatus:      307,
			expectContentType: "application/vnd.fly.replay+json",
			flyAppName:        "smooth-nav",
		},
		{
			name: "App-based fly-replay",
			path: "/showcase/documents/test.pdf",
			rewriteRules: []config.RewriteRule{
				{
					Pattern:     regexp.MustCompile(`^/showcase/.+\.pdf$`),
					Replacement: "/showcase/documents/test.pdf",
					Flag:        "fly-replay:app=smooth-pdf:307",
				},
			},
			expectHandled:     true,
			expectFlyReplay:   true,
			expectStatus:      307,
			expectContentType: "application/vnd.fly.replay+json",
			flyAppName:        "smooth-nav",
		},
		{
			name: "Regular redirect (non-fly-replay)",
			path: "/old-path",
			rewriteRules: []config.RewriteRule{
				{
					Pattern:     regexp.MustCompile(`^/old-path$`),
					Replacement: "/new-path",
					Flag:        "redirect",
				},
			},
			expectHandled:   true,
			expectFlyReplay: false,
			expectStatus:    302,
		},
		{
			name: "No matching rewrite rule",
			path: "/no-match",
			rewriteRules: []config.RewriteRule{
				{
					Pattern:     regexp.MustCompile(`^/other-path$`),
					Replacement: "/replacement",
					Flag:        "redirect",
				},
			},
			expectHandled:   false,
			expectFlyReplay: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			if tt.flyAppName != "" {
				os.Setenv("FLY_APP_NAME", tt.flyAppName)
				defer os.Unsetenv("FLY_APP_NAME")
			}

			// Create handler with test configuration
			cfg := &config.Config{}
			cfg.Server.RewriteRules = tt.rewriteRules

			handler := &Handler{
				config: cfg,
			}

			// Create test request
			req := httptest.NewRequest("GET", tt.path, nil)
			recorder := httptest.NewRecorder()

			// Test the handleRewrites method
			handled := handler.handleRewrites(recorder, req)

			if handled != tt.expectHandled {
				t.Errorf("handleRewrites() returned %v, expected %v", handled, tt.expectHandled)
			}

			if !tt.expectHandled {
				return
			}

			// Check status code
			if recorder.Code != tt.expectStatus {
				t.Errorf("Status code = %d, expected %d", recorder.Code, tt.expectStatus)
			}

			// Check for fly-replay specific responses
			if tt.expectFlyReplay {
				if recorder.Header().Get("Content-Type") != tt.expectContentType {
					t.Errorf("Content-Type = %q, expected %q",
						recorder.Header().Get("Content-Type"), tt.expectContentType)
				}

				// Verify JSON response structure for fly-replay
				body := recorder.Body.String()
				if !strings.Contains(body, `"region"`) && !strings.Contains(body, `"app"`) {
					t.Error("Fly-replay response should contain either 'region' or 'app' field")
				}
			}
		})
	}
}

func TestHandler_HandleRewritesFlyReplayWithMethods(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.RewriteRules = []config.RewriteRule{
		{
			Pattern:     regexp.MustCompile(`^/api/`),
			Replacement: "/api/",
			Flag:        "fly-replay:us-west:307",
			Methods:     []string{"GET", "HEAD"}, // Only allow safe methods
		},
	}

	handler := &Handler{
		config:        cfg,
		staticHandler: NewStaticFileHandler(cfg),
	}

	tests := []struct {
		name          string
		method        string
		expectHandled bool
	}{
		{"GET allowed", "GET", true},
		{"HEAD allowed", "HEAD", true},
		{"POST not allowed", "POST", false},
		{"PUT not allowed", "PUT", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/endpoint", nil)
			recorder := httptest.NewRecorder()

			handled := handler.handleRewrites(recorder, req)

			if handled != tt.expectHandled {
				t.Errorf("handleRewrites() returned %v, expected %v for method %s",
					handled, tt.expectHandled, tt.method)
			}
		})
	}
}

func TestHandler_HandleRewritesFlyReplayLargeRequest(t *testing.T) {
	os.Setenv("FLY_APP_NAME", "testapp")
	defer os.Unsetenv("FLY_APP_NAME")

	cfg := &config.Config{}
	cfg.Server.Listen = "3000"
	cfg.Server.RewriteRules = []config.RewriteRule{
		{
			Pattern:     regexp.MustCompile(`^/upload/`),
			Replacement: "/upload/",
			Flag:        "fly-replay:us-west:307",
		},
	}

	handler := &Handler{
		config:        cfg,
		staticHandler: NewStaticFileHandler(cfg),
	}

	// Create a POST request with large content that should trigger fallback
	req := httptest.NewRequest("POST", "/upload/large-file", strings.NewReader("large content"))
	req.ContentLength = MaxFlyReplaySize + 1 // Exceeds limit
	recorder := httptest.NewRecorder()

	handled := handler.handleRewrites(recorder, req)

	if !handled {
		t.Error("handleRewrites() should have handled large request via fallback")
	}

	// Should not be a fly-replay JSON response
	contentType := recorder.Header().Get("Content-Type")
	if contentType == "application/vnd.fly.replay+json" {
		t.Error("Large request should not use fly-replay JSON response, should use fallback")
	}
}

// TestHandler_HandleRewritesBasicRedirect tests basic redirect functionality
func TestHandler_HandleRewritesBasicRedirect(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.RewriteRules = []config.RewriteRule{
		{
			Pattern:     regexp.MustCompile(`^/old-path/(.+)$`),
			Replacement: "/new-path/$1",
			Flag:        "redirect",
		},
	}

	handler := &Handler{
		config:        cfg,
		staticHandler: NewStaticFileHandler(cfg),
	}

	tests := []struct {
		name             string
		path             string
		expectRedirect   bool
		expectedLocation string
	}{
		{
			name:             "Matching path should redirect",
			path:             "/old-path/some-page",
			expectRedirect:   true,
			expectedLocation: "/new-path/some-page",
		},
		{
			name:           "Non-matching path should not redirect",
			path:           "/other-path/page",
			expectRedirect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			recorder := httptest.NewRecorder()

			handled := handler.handleRewrites(recorder, req)

			if tt.expectRedirect {
				if !handled {
					t.Error("Expected redirect to be handled")
				}
				if recorder.Code != http.StatusFound {
					t.Errorf("Expected status %d, got %d", http.StatusFound, recorder.Code)
				}
				location := recorder.Header().Get("Location")
				if location != tt.expectedLocation {
					t.Errorf("Expected Location header %q, got %q", tt.expectedLocation, location)
				}
			} else {
				if handled {
					t.Error("Expected redirect not to be handled")
				}
			}
		})
	}
}

// TestHandler_HandleRewritesInternalRewrite tests internal rewrite functionality
func TestHandler_HandleRewritesInternalRewrite(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.RewriteRules = []config.RewriteRule{
		{
			Pattern:     regexp.MustCompile(`^/api/v1/(.+)$`),
			Replacement: "/api/v2/$1",
			Flag:        "last",
		},
	}

	handler := &Handler{
		config:        cfg,
		staticHandler: NewStaticFileHandler(cfg),
	}

	tests := []struct {
		name          string
		path          string
		expectRewrite bool
		expectedPath  string
	}{
		{
			name:          "Matching path should be rewritten",
			path:          "/api/v1/users",
			expectRewrite: true,
			expectedPath:  "/api/v2/users",
		},
		{
			name:          "Non-matching path should not be rewritten",
			path:          "/other/path",
			expectRewrite: false,
			expectedPath:  "/other/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			recorder := httptest.NewRecorder()

			// For "last" flag, handleRewrites should return false (continue processing)
			handled := handler.handleRewrites(recorder, req)

			if handled {
				t.Error("Expected internal rewrite not to return true (should continue processing)")
			}

			// Check if the path was actually rewritten
			if req.URL.Path != tt.expectedPath {
				t.Errorf("Expected path %q, got %q", tt.expectedPath, req.URL.Path)
			}
		})
	}
}

// TestHandler_HandleRewritesRegexReplacement tests complex regex patterns
func TestHandler_HandleRewritesRegexReplacement(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.RewriteRules = []config.RewriteRule{
		{
			Pattern:     regexp.MustCompile(`^/user/(\d+)/profile/(.+)$`),
			Replacement: "/profile/$2?user_id=$1",
			Flag:        "redirect",
		},
		{
			Pattern:     regexp.MustCompile(`^/legacy/([^/]+)/(.+)$`),
			Replacement: "/modern/$1/$2",
			Flag:        "last",
		},
	}

	handler := &Handler{
		config:        cfg,
		staticHandler: NewStaticFileHandler(cfg),
	}

	tests := []struct {
		name           string
		path           string
		expectHandled  bool
		expectedResult string // Location header for redirects, URL.Path for rewrites
		isRedirect     bool
	}{
		{
			name:           "User profile redirect with capture groups",
			path:           "/user/123/profile/settings",
			expectHandled:  true,
			expectedResult: "/profile/settings?user_id=123",
			isRedirect:     true,
		},
		{
			name:           "Legacy path internal rewrite",
			path:           "/legacy/api/users",
			expectHandled:  false, // "last" flag doesn't return true
			expectedResult: "/modern/api/users",
			isRedirect:     false,
		},
		{
			name:           "Non-matching path",
			path:           "/other/path",
			expectHandled:  false,
			expectedResult: "/other/path",
			isRedirect:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			recorder := httptest.NewRecorder()

			handled := handler.handleRewrites(recorder, req)

			if handled != tt.expectHandled {
				t.Errorf("Expected handled=%v, got %v", tt.expectHandled, handled)
			}

			if tt.isRedirect && tt.expectHandled {
				if recorder.Code != http.StatusFound {
					t.Errorf("Expected status %d, got %d", http.StatusFound, recorder.Code)
				}
				location := recorder.Header().Get("Location")
				if location != tt.expectedResult {
					t.Errorf("Expected Location %q, got %q", tt.expectedResult, location)
				}
			} else if !tt.isRedirect {
				// Check URL path for internal rewrites
				if req.URL.Path != tt.expectedResult {
					t.Errorf("Expected path %q, got %q", tt.expectedResult, req.URL.Path)
				}
			}
		})
	}
}

// TestHandler_HandleRewritesMultipleRules tests multiple rewrite rules in sequence
func TestHandler_HandleRewritesMultipleRules(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.RewriteRules = []config.RewriteRule{
		{
			Pattern:     regexp.MustCompile(`^/step1/(.+)$`),
			Replacement: "/step2/$1",
			Flag:        "last",
		},
		{
			Pattern:     regexp.MustCompile(`^/step2/(.+)$`),
			Replacement: "/final/$1",
			Flag:        "last",
		},
		{
			Pattern:     regexp.MustCompile(`^/redirect-me/(.+)$`),
			Replacement: "/redirected/$1",
			Flag:        "redirect",
		},
	}

	handler := &Handler{
		config:        cfg,
		staticHandler: NewStaticFileHandler(cfg),
	}

	tests := []struct {
		name          string
		path          string
		expectHandled bool
		expectedPath  string
		isRedirect    bool
	}{
		{
			name:          "Multiple rewrite rules should chain",
			path:          "/step1/test",
			expectHandled: false,         // "last" doesn't return true
			expectedPath:  "/final/test", // Both rules apply: step1->step2->final
			isRedirect:    false,
		},
		{
			name:          "Redirect rule should terminate processing",
			path:          "/redirect-me/test",
			expectHandled: true,
			expectedPath:  "/redirected/test", // This will be in Location header
			isRedirect:    true,
		},
		{
			name:          "Non-matching path",
			path:          "/no-match/test",
			expectHandled: false,
			expectedPath:  "/no-match/test",
			isRedirect:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			recorder := httptest.NewRecorder()

			handled := handler.handleRewrites(recorder, req)

			if handled != tt.expectHandled {
				t.Errorf("Expected handled=%v, got %v", tt.expectHandled, handled)
			}

			if tt.isRedirect && tt.expectHandled {
				location := recorder.Header().Get("Location")
				if location != tt.expectedPath {
					t.Errorf("Expected Location %q, got %q", tt.expectedPath, location)
				}
			} else if !tt.isRedirect {
				if req.URL.Path != tt.expectedPath {
					t.Errorf("Expected path %q, got %q", tt.expectedPath, req.URL.Path)
				}
			}
		})
	}
}
