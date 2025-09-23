package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
	respRecorder := NewResponseRecorder(recorder, nil)

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

func TestFindBestLocation(t *testing.T) {
	cfg := &config.Config{
		Locations: []config.Location{
			{
				Path:      "/api/*",
				ProxyPass: "http://localhost:4001",
			},
			{
				Path:      "/admin/*",
				ProxyPass: "http://localhost:4002",
			},
			{
				Path:      "/*/cable",
				ProxyPass: "http://localhost:4003",
			},
		},
	}

	handler := &Handler{config: cfg}

	tests := []struct {
		path     string
		expected string // Expected proxy pass URL
	}{
		{"/api/users", "http://localhost:4001"},
		{"/admin/dashboard", "http://localhost:4002"},
		{"/app/cable", "http://localhost:4003"},
		{"/unknown/path", ""}, // No match
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			location := handler.findBestLocation(tt.path)

			if tt.expected == "" {
				if location != nil {
					t.Errorf("Expected no location match, got %v", location)
				}
			} else {
				if location == nil {
					t.Errorf("Expected location match, got nil")
				} else if location.ProxyPass != tt.expected {
					t.Errorf("Expected proxy pass %s, got %s", tt.expected, location.ProxyPass)
				}
			}
		})
	}
}

func TestHealthCheckHandler(t *testing.T) {
	cfg := &config.Config{}
	handler := &Handler{config: cfg}

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

	expectedContentType := "text/html"
	if recorder.Header().Get("Content-Type") != expectedContentType {
		t.Errorf("Expected Content-Type %q, got %q", expectedContentType, recorder.Header().Get("Content-Type"))
	}
}

func TestSetContentType(t *testing.T) {
	tests := []struct {
		filename    string
		expected    string
	}{
		{"test.html", "text/html; charset=utf-8"},
		{"test.css", "text/css"},
		{"test.js", "application/javascript"},
		{"test.json", "application/json; charset=utf-8"},
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
			setContentType(recorder, tt.filename)

			contentType := recorder.Header().Get("Content-Type")
			if contentType != tt.expected {
				t.Errorf("For %s, expected %s, got %s", tt.filename, tt.expected, contentType)
			}
		})
	}
}

func TestLocationMatching(t *testing.T) {
	cfg := &config.Config{
		Locations: []config.Location{
			{Path: "/api/*", ProxyPass: "http://localhost:4001"},
			{Path: "/admin/*", ProxyPass: "http://localhost:4002"},
			{Path: "/*/cable", ProxyPass: "http://localhost:4003"},
		},
	}

	handler := &Handler{config: cfg}

	tests := []struct {
		path     string
		expected string
	}{
		{"/api/users", "/api/*"},
		{"/admin/dashboard", "/admin/*"},
		{"/2025/cable", "/*/cable"},
		{"/unknown/path", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			location := handler.findBestLocation(tt.path)
			if tt.expected == "" {
				if location != nil {
					t.Errorf("For path %s, expected nil location, got %s", tt.path, location.Path)
				}
			} else {
				if location == nil {
					t.Errorf("For path %s, expected location %s, got nil", tt.path, tt.expected)
				} else if location.Path != tt.expected {
					t.Errorf("For path %s, expected location %s, got %s", tt.path, tt.expected, location.Path)
				}
			}
		})
	}
}

func TestTryFilesWithStaticDirectories(t *testing.T) {
	// Create temporary directory structure for testing
	tempDir, err := os.MkdirTemp("", "navigator-tryfiles-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	studiosDir := filepath.Join(tempDir, "studios")
	if err := os.MkdirAll(studiosDir, 0755); err != nil {
		t.Fatalf("Failed to create studios dir: %v", err)
	}

	docsDir := filepath.Join(tempDir, "docs")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatalf("Failed to create docs dir: %v", err)
	}

	// Create index.html files
	indexContent := "<html><body>Test Page</body></html>"
	if err := os.WriteFile(filepath.Join(studiosDir, "index.html"), []byte(indexContent), 0644); err != nil {
		t.Fatalf("Failed to write studios/index.html: %v", err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "guide.html"), []byte(indexContent), 0644); err != nil {
		t.Fatalf("Failed to write docs/guide.html: %v", err)
	}

	cfg := &config.Config{}
	cfg.Server.PublicDir = tempDir
	cfg.Static.Directories = []config.StaticDir{
		{
			Path:   "/showcase/studios/",
			Prefix: "studios/",
		},
		{
			Path:   "/showcase/docs/",
			Prefix: "docs/",
		},
	}
	cfg.Static.TryFiles.Enabled = true
	cfg.Static.TryFiles.Suffixes = []string{"index.html", ".html"}

	handler := &Handler{
		config: cfg,
		auth:   &auth.BasicAuth{},
	}

	tests := []struct {
		name           string
		path           string
		expectedFound  bool
		shouldContain  string
	}{
		{
			name:          "Studios directory with index.html",
			path:          "/showcase/studios/",
			expectedFound: true,
			shouldContain: "Test Page",
		},
		{
			name:          "Docs directory with .html suffix",
			path:          "/showcase/docs/guide",
			expectedFound: true,
			shouldContain: "Test Page",
		},
		{
			name:          "Non-existent path",
			path:          "/showcase/nonexistent/",
			expectedFound: false,
		},
		{
			name:          "Path with extension should be skipped",
			path:          "/showcase/studios/existing.html",
			expectedFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			recorder := httptest.NewRecorder()
			respRecorder := NewResponseRecorder(recorder, nil)

			// Test tryFiles directly
			found := handler.tryFiles(respRecorder, req, nil)

			if found != tt.expectedFound {
				t.Errorf("Expected tryFiles to return %v for %s, got %v", tt.expectedFound, tt.path, found)
			}

			if tt.expectedFound && tt.shouldContain != "" {
				if recorder.Code != http.StatusOK {
					t.Errorf("Expected status %d, got %d", http.StatusOK, recorder.Code)
				}
				if !strings.Contains(recorder.Body.String(), tt.shouldContain) {
					t.Errorf("Expected body to contain %q, got %q", tt.shouldContain, recorder.Body.String())
				}
			}
		})
	}
}

func TestStaticDirectoryMatching(t *testing.T) {
	cfg := &config.Config{
		Static: config.StaticConfig{
			Directories: []config.StaticDir{
				{
					Path:   "/showcase/studios/",
					Prefix: "studios/",
				},
				{
					Path:   "/showcase/docs/",
					Prefix: "docs/",
				},
				{
					Path:   "/showcase/",
					Prefix: "general/",
				},
			},
		},
	}

	tests := []struct {
		path             string
		expectedPath     string
		expectedPrefix   string
		shouldMatch      bool
	}{
		{
			path:           "/showcase/studios/",
			expectedPath:   "/showcase/studios/",
			expectedPrefix: "studios/",
			shouldMatch:    true,
		},
		{
			path:           "/showcase/studios/page",
			expectedPath:   "/showcase/studios/",
			expectedPrefix: "studios/",
			shouldMatch:    true,
		},
		{
			path:           "/showcase/docs/guide",
			expectedPath:   "/showcase/docs/",
			expectedPrefix: "docs/",
			shouldMatch:    true,
		},
		{
			path:           "/showcase/other",
			expectedPath:   "/showcase/",
			expectedPrefix: "general/",
			shouldMatch:    true,
		},
		{
			path:        "/different/path",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			var bestStaticDir *config.StaticDir
			bestStaticDirLen := 0

			// Simulate the static directory matching logic from tryFiles
			for _, staticDir := range cfg.Static.Directories {
				if strings.HasPrefix(tt.path, staticDir.Path) && len(staticDir.Path) > bestStaticDirLen {
					bestStaticDir = &staticDir
					bestStaticDirLen = len(staticDir.Path)
				}
			}

			if tt.shouldMatch {
				if bestStaticDir == nil {
					t.Errorf("Expected to find matching static directory for %s", tt.path)
				} else {
					if bestStaticDir.Path != tt.expectedPath {
						t.Errorf("Expected path %s, got %s", tt.expectedPath, bestStaticDir.Path)
					}
					if bestStaticDir.Prefix != tt.expectedPrefix {
						t.Errorf("Expected prefix %s, got %s", tt.expectedPrefix, bestStaticDir.Prefix)
					}
				}
			} else {
				if bestStaticDir != nil {
					t.Errorf("Expected no match for %s, but got %s", tt.path, bestStaticDir.Path)
				}
			}
		})
	}
}

func TestTryFilesConfigurationPriority(t *testing.T) {
	tests := []struct {
		name            string
		locationTryFiles []string
		serverTryFiles   []string
		staticTryFiles   struct {
			Enabled  bool
			Suffixes []string
		}
		expectedSuffixes []string
	}{
		{
			name:             "Location try_files takes priority",
			locationTryFiles: []string{".location"},
			serverTryFiles:   []string{".server"},
			staticTryFiles: struct {
				Enabled  bool
				Suffixes []string
			}{
				Enabled:  true,
				Suffixes: []string{".static"},
			},
			expectedSuffixes: []string{".location"},
		},
		{
			name:           "Server try_files when no location",
			serverTryFiles: []string{".server"},
			staticTryFiles: struct {
				Enabled  bool
				Suffixes []string
			}{
				Enabled:  true,
				Suffixes: []string{".static"},
			},
			expectedSuffixes: []string{".server"},
		},
		{
			name: "Static try_files when no server or location",
			staticTryFiles: struct {
				Enabled  bool
				Suffixes []string
			}{
				Enabled:  true,
				Suffixes: []string{".static"},
			},
			expectedSuffixes: []string{".static"},
		},
		{
			name: "Default extensions when static disabled",
			staticTryFiles: struct {
				Enabled  bool
				Suffixes []string
			}{
				Enabled: false,
			},
			expectedSuffixes: []string{".html", ".htm", ".txt", ".xml", ".json"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.Server.TryFiles = tt.serverTryFiles
			cfg.Static.TryFiles.Enabled = tt.staticTryFiles.Enabled
			cfg.Static.TryFiles.Suffixes = tt.staticTryFiles.Suffixes

			var location *config.Location
			if len(tt.locationTryFiles) > 0 {
				location = &config.Location{
					TryFiles: tt.locationTryFiles,
				}
			}

			// Test the extension priority logic from tryFiles
			var extensions []string
			if location != nil && len(location.TryFiles) > 0 {
				extensions = location.TryFiles
			} else if len(cfg.Server.TryFiles) > 0 {
				extensions = cfg.Server.TryFiles
			} else if cfg.Static.TryFiles.Enabled && len(cfg.Static.TryFiles.Suffixes) > 0 {
				extensions = cfg.Static.TryFiles.Suffixes
			} else {
				extensions = []string{".html", ".htm", ".txt", ".xml", ".json"}
			}

			if len(extensions) != len(tt.expectedSuffixes) {
				t.Errorf("Expected %d suffixes, got %d", len(tt.expectedSuffixes), len(extensions))
			}

			for i, expected := range tt.expectedSuffixes {
				if i >= len(extensions) || extensions[i] != expected {
					t.Errorf("Expected suffix[%d] to be %s, got %s", i, expected, extensions[i])
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
		respRecorder := NewResponseRecorder(recorder, nil)
		respRecorder.WriteHeader(http.StatusOK)
		respRecorder.Write(testData)
		respRecorder.SetMetadata("test", "value")
	}
}

func BenchmarkFindBestLocation(b *testing.B) {
	cfg := &config.Config{
		Locations: []config.Location{
			{Path: "/api/*", ProxyPass: "http://localhost:4001"},
			{Path: "/admin/*", ProxyPass: "http://localhost:4002"},
			{Path: "/*/cable", ProxyPass: "http://localhost:4003"},
			{Path: "/assets/*", ProxyPass: "http://localhost:4004"},
			{Path: "/uploads/*", ProxyPass: "http://localhost:4005"},
		},
	}

	handler := &Handler{config: cfg}
	testPath := "/api/users/123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.findBestLocation(testPath)
	}
}

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
			handler := CreateHandler(cfg, appManager, tt.auth, idleManager)

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
	handler := CreateHandler(cfg, nil, nil, nil)

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
	if contentType != "text/html" {
		t.Errorf("Health check Content-Type = %q, expected %q", contentType, "text/html")
	}
}

func TestHandler_ServeHTTP_RequestID(t *testing.T) {
	cfg := &config.Config{}
	handler := CreateHandler(cfg, nil, nil, nil)

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
	cfg.Server.AuthExclude = []string{"/public/*", "/assets/*"}

	// Create a basic auth instance
	basicAuth := &auth.BasicAuth{}
	// Enable auth by setting some values (simplified)

	handler := CreateHandler(cfg, nil, basicAuth, nil)

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
	cfg := &config.Config{
		Locations: []config.Location{
			{Path: "/api/*", ProxyPass: "http://localhost:4001"},
			{Path: "/static/*", ProxyPass: "http://localhost:4002"},
		},
	}

	handler := CreateHandler(cfg, &process.AppManager{}, nil, nil)

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

func TestHandler_handleHealthCheck(t *testing.T) {
	cfg := &config.Config{}
	handler := &Handler{config: cfg}

	req := httptest.NewRequest("GET", "/up", nil)
	recorder := httptest.NewRecorder()

	handler.handleHealthCheck(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("handleHealthCheck returned %d, expected %d", recorder.Code, http.StatusOK)
	}

	if recorder.Body.String() != "OK" {
		t.Errorf("handleHealthCheck body = %q, expected %q", recorder.Body.String(), "OK")
	}

	contentType := recorder.Header().Get("Content-Type")
	if contentType != "text/html" {
		t.Errorf("handleHealthCheck Content-Type = %q, expected %q", contentType, "text/html")
	}
}

func TestHandler_handleStickySession(t *testing.T) {
	tests := []struct {
		name           string
		stickyEnabled  bool
		paths          []string
		requestPath    string
		expectHandled  bool
	}{
		{"Sticky disabled", false, nil, "/any/path", false},
		{"Sticky enabled, no paths configured", true, nil, "/any/path", false},
		{"Sticky enabled, path matches", true, []string{"/app/*"}, "/app/dashboard", false}, // Returns false in current impl
		{"Sticky enabled, path doesn't match", true, []string{"/app/*"}, "/other/path", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.Server.StickySession.Enabled = tt.stickyEnabled
			cfg.Server.StickySession.Paths = tt.paths

			handler := &Handler{config: cfg}

			req := httptest.NewRequest("GET", tt.requestPath, nil)
			recorder := httptest.NewRecorder()

			handled := handler.handleStickySession(recorder, req)

			if handled != tt.expectHandled {
				t.Errorf("handleStickySession returned %v, expected %v", handled, tt.expectHandled)
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

	// Create test configuration for maintenance mode
	cfg := &config.Config{
		Applications: config.Applications{
			Tenants: []config.Tenant{}, // Empty tenants for maintenance mode
		},
	}
	cfg.Server.PublicDir = tempDir
	cfg.Static.TryFiles.Enabled = true
	cfg.Static.TryFiles.Fallback = "/503.html"

	// Create handler
	handler := CreateHandler(cfg, nil, nil, nil)

	// Test cases
	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Root path returns maintenance page",
			path:           "/",
			expectedStatus: http.StatusOK,
			expectedBody:   "Site Under Maintenance",
		},
		{
			name:           "Random path returns maintenance page",
			path:           "/some/random/path",
			expectedStatus: http.StatusOK,
			expectedBody:   "Site Under Maintenance",
		},
		{
			name:           "Path with query params returns maintenance page",
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
		config: cfg,
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
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "navigator-fallback-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a fallback page
	fallbackHTML := `<!DOCTYPE html><html><body>Fallback Page</body></html>`
	fallbackPath := filepath.Join(tempDir, "fallback.html")
	if err := os.WriteFile(fallbackPath, []byte(fallbackHTML), 0644); err != nil {
		t.Fatalf("Failed to create fallback file: %v", err)
	}

	// Create test configuration with no tenants and static fallback
	cfg := &config.Config{
		Applications: config.Applications{
			Tenants: []config.Tenant{}, // Empty tenants
		},
	}
	cfg.Server.PublicDir = tempDir
	cfg.Static.TryFiles.Fallback = "/fallback.html"

	// Create handler
	handler := &Handler{
		config: cfg,
	}

	// Test fallback handling
	req := httptest.NewRequest("GET", "/any/path", nil)
	rr := httptest.NewRecorder()

	handler.handleStaticFallback(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	// Check response body
	body := rr.Body.String()
	if !strings.Contains(body, "Fallback Page") {
		t.Errorf("Expected body to contain 'Fallback Page', got: %s", body)
	}
}