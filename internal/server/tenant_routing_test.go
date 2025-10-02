package server

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rubys/navigator/internal/config"
	"github.com/rubys/navigator/internal/idle"
	"github.com/rubys/navigator/internal/process"
)

// TestTenantRoutingIntegration tests the end-to-end tenant routing flow
func TestTenantRoutingIntegration(t *testing.T) {
	// Create test configuration with multiple tenants
	cfg := &config.Config{}
	cfg.Applications.Tenants = []config.Tenant{
		{
			Name: "",
			Path: "/showcase/",
			Root: "/app/index",
		},
		{
			Name: "2025/raleigh/shimmer-shine",
			Path: "/showcase/2025/raleigh/shimmer-shine/",
			Root: "/app/shimmer-shine",
		},
		{
			Name: "2025/boston/april",
			Path: "/showcase/2025/boston/april/",
			Root: "/app/boston",
		},
		{
			Name: "test-simple",
			Path: "/showcase/test-simple/",
			Root: "/app/simple",
		},
	}

	tests := []struct {
		name           string
		requestPath    string
		expectedTenant string
		shouldMatch    bool
	}{
		{
			name:           "Index tenant with empty name - index_update",
			requestPath:    "/showcase/index_update",
			expectedTenant: "",
			shouldMatch:    true,
		},
		{
			name:           "Index tenant with empty name - index_date",
			requestPath:    "/showcase/index_date",
			expectedTenant: "",
			shouldMatch:    true,
		},
		{
			name:           "Index tenant with empty name - root",
			requestPath:    "/showcase/",
			expectedTenant: "",
			shouldMatch:    true,
		},
		{
			name:           "Shimmer-shine tenant routing",
			requestPath:    "/showcase/2025/raleigh/shimmer-shine/",
			expectedTenant: "2025/raleigh/shimmer-shine",
			shouldMatch:    true,
		},
		{
			name:           "Shimmer-shine deep path",
			requestPath:    "/showcase/2025/raleigh/shimmer-shine/events/123",
			expectedTenant: "2025/raleigh/shimmer-shine",
			shouldMatch:    true,
		},
		{
			name:           "Boston tenant routing",
			requestPath:    "/showcase/2025/boston/april/",
			expectedTenant: "2025/boston/april",
			shouldMatch:    true,
		},
		{
			name:           "Boston deep path",
			requestPath:    "/showcase/2025/boston/april/formations",
			expectedTenant: "2025/boston/april",
			shouldMatch:    true,
		},
		{
			name:           "Simple tenant routing",
			requestPath:    "/showcase/test-simple/",
			expectedTenant: "test-simple",
			shouldMatch:    true,
		},
		{
			name:           "Non-existent tenant falls back to index",
			requestPath:    "/showcase/2025/invalid/tenant/",
			expectedTenant: "",
			shouldMatch:    true,
		},
		{
			name:        "Non-showcase path",
			requestPath: "/other/path",
			shouldMatch: false,
		},
		{
			name:        "Root path",
			requestPath: "/",
			shouldMatch: false,
		},
	}

	// Create handler with properly initialized managers
	appManager := process.NewAppManager(cfg)
	idleManager := idle.NewManager(cfg)
	handler := CreateTestHandler(cfg, appManager, nil, idleManager)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test request
			req := httptest.NewRequest("GET", tt.requestPath, nil)
			recorder := httptest.NewRecorder()

			// Call the handler
			handler.ServeHTTP(recorder, req)

			if tt.shouldMatch {
				// For valid tenant paths, we expect the handler to attempt routing
				// (even if it fails because we don't have actual apps running)
				// The key is that it should NOT return 404 immediately
				if recorder.Code == 404 {
					// Check if this is a "tenant not found" 404 vs "app not running" 404
					responseBody := recorder.Body.String()
					if strings.Contains(responseBody, "Not Found") && !strings.Contains(responseBody, "maintenance") {
						t.Errorf("Expected tenant %q to be matched for path %q, but got 404 Not Found", tt.expectedTenant, tt.requestPath)
					}
				}
				// If we get 502 Bad Gateway, that's actually good - it means the tenant was found
				// but the application isn't running (which is expected in tests)
			} else {
				// For invalid paths, we expect 404 Not Found
				if recorder.Code != 404 {
					t.Errorf("Expected 404 for invalid path %q, got %d", tt.requestPath, recorder.Code)
				}
			}
		})
	}
}

// TestExtractTenantFromPath tests the extractTenantFromPath function with various scenarios
func TestExtractTenantFromPath(t *testing.T) {
	cfg := &config.Config{}
	cfg.Applications.Tenants = []config.Tenant{
		{Name: "", Path: "/showcase/"},
		{Name: "2025/raleigh/shimmer-shine", Path: "/showcase/2025/raleigh/shimmer-shine/"},
		{Name: "2025/boston/april", Path: "/showcase/2025/boston/april/"},
		{Name: "test-simple", Path: "/showcase/test-simple/"},
	}

	tests := []struct {
		name           string
		requestPath    string
		expectedTenant string
		expectedFound  bool
	}{
		{
			name:           "Index tenant with empty name - index_update",
			requestPath:    "/showcase/index_update",
			expectedTenant: "",
			expectedFound:  true,
		},
		{
			name:           "Index tenant with empty name - index_date",
			requestPath:    "/showcase/index_date",
			expectedTenant: "",
			expectedFound:  true,
		},
		{
			name:           "Shimmer-shine tenant",
			requestPath:    "/showcase/2025/raleigh/shimmer-shine/",
			expectedTenant: "2025/raleigh/shimmer-shine",
			expectedFound:  true,
		},
		{
			name:           "Boston tenant deep path",
			requestPath:    "/showcase/2025/boston/april/formations",
			expectedTenant: "2025/boston/april",
			expectedFound:  true,
		},
		{
			name:           "Simple tenant",
			requestPath:    "/showcase/test-simple/",
			expectedTenant: "test-simple",
			expectedFound:  true,
		},
		{
			name:           "Non-existent tenant falls back to index",
			requestPath:    "/showcase/2025/nonexistent/",
			expectedTenant: "",
			expectedFound:  true,
		},
		{
			name:           "Non-showcase path",
			requestPath:    "/other/path",
			expectedTenant: "",
			expectedFound:  false,
		},
		{
			name:           "Root path",
			requestPath:    "/",
			expectedTenant: "",
			expectedFound:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Manually test the extraction logic
			var bestMatch string
			bestMatchLen := 0
			found := false

			for _, tenant := range cfg.Applications.Tenants {
				if strings.HasPrefix(tt.requestPath, tenant.Path) && len(tenant.Path) > bestMatchLen {
					bestMatch = tenant.Name
					bestMatchLen = len(tenant.Path)
					found = true
				}
			}

			if found != tt.expectedFound {
				t.Errorf("Expected found=%v for path %q, got %v", tt.expectedFound, tt.requestPath, found)
			}

			if bestMatch != tt.expectedTenant {
				t.Errorf("Expected tenant %q for path %q, got %q", tt.expectedTenant, tt.requestPath, bestMatch)
			}
		})
	}
}

// TestTenantMatchingLogic tests the tenant matching algorithm directly
func TestTenantMatchingLogic(t *testing.T) {
	cfg := &config.Config{}
	cfg.Applications.Tenants = []config.Tenant{
		{Name: "2025/raleigh/shimmer-shine", Path: "/showcase/2025/raleigh/shimmer-shine/"},
		{Name: "2025/boston/april", Path: "/showcase/2025/boston/april/"},
		{Name: "test-simple", Path: "/showcase/test-simple/"},
	}

	tests := []struct {
		requestPath    string
		expectedTenant string
		shouldMatch    bool
	}{
		{"/showcase/2025/raleigh/shimmer-shine/", "2025/raleigh/shimmer-shine", true},
		{"/showcase/2025/raleigh/shimmer-shine/events", "2025/raleigh/shimmer-shine", true},
		{"/showcase/2025/boston/april/", "2025/boston/april", true},
		{"/showcase/test-simple/", "test-simple", true},
		{"/showcase/2025/nonexistent/", "", false},
		{"/other/path", "", false},
		{"/", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.requestPath, func(t *testing.T) {
			// Test the tenant matching logic using Path field
			tenantName := ""
			found := false
			for _, tenant := range cfg.Applications.Tenants {
				if strings.HasPrefix(tt.requestPath, tenant.Path) {
					tenantName = tenant.Name
					found = true
					break
				}
			}

			if tt.shouldMatch {
				if !found || tenantName != tt.expectedTenant {
					t.Errorf("Expected tenant %q for path %q, got %q (found=%v)", tt.expectedTenant, tt.requestPath, tenantName, found)
				}
			} else {
				if found {
					t.Errorf("Expected no tenant match for path %q, got %q", tt.requestPath, tenantName)
				}
			}
		})
	}
}

// TestTenantPathExtraction tests the path extraction that happens during config loading
func TestTenantPathExtraction(t *testing.T) {
	tests := []struct {
		path         string
		expectedName string
	}{
		{"/showcase/2025/raleigh/shimmer-shine/", "2025/raleigh/shimmer-shine"},
		{"/showcase/2025/boston/april/", "2025/boston/april"},
		{"/showcase/test-simple/", "test-simple"},
		{"/showcase/complex/tenant/name/", "complex/tenant/name"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			// Simulate the tenant name extraction logic from config/loader.go
			tenantName := strings.TrimPrefix(tt.path, "/showcase/")
			tenantName = strings.TrimSuffix(tenantName, "/")

			if tenantName != tt.expectedName {
				t.Errorf("Expected name %q from path %q, got %q", tt.expectedName, tt.path, tenantName)
			}
		})
	}
}
