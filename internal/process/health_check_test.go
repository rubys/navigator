package process

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rubys/navigator/internal/config"
)

func TestGetHealthCheckEndpoint(t *testing.T) {
	tests := []struct {
		name           string
		globalHealth   string
		tenantHealth   string
		expectedResult string
	}{
		{
			name:           "Default to root when no health check configured",
			globalHealth:   "",
			tenantHealth:   "",
			expectedResult: "/",
		},
		{
			name:           "Use global health check when configured",
			globalHealth:   "/up",
			tenantHealth:   "",
			expectedResult: "/up",
		},
		{
			name:           "Tenant health check overrides global",
			globalHealth:   "/up",
			tenantHealth:   "/health",
			expectedResult: "/health",
		},
		{
			name:           "Tenant health check used when no global",
			globalHealth:   "",
			tenantHealth:   "/status",
			expectedResult: "/status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Applications: config.Applications{
					HealthCheck: tt.globalHealth,
				},
			}

			ps := NewProcessStarter(cfg)

			tenant := &config.Tenant{
				Name:        "test-tenant",
				HealthCheck: tt.tenantHealth,
			}

			result := ps.getHealthCheckEndpoint(tenant)

			if result != tt.expectedResult {
				t.Errorf("getHealthCheckEndpoint() = %q, want %q", result, tt.expectedResult)
			}
		})
	}
}

func TestGetHealthCheckEndpoint_NilTenant(t *testing.T) {
	cfg := &config.Config{
		Applications: config.Applications{
			HealthCheck: "/up",
		},
	}

	ps := NewProcessStarter(cfg)
	result := ps.getHealthCheckEndpoint(nil)

	if result != "/up" {
		t.Errorf("getHealthCheckEndpoint(nil) = %q, want %q", result, "/up")
	}
}

func TestWaitForReadyWithHealthCheck(t *testing.T) {
	// Create a test HTTP server that responds on a health check endpoint
	healthCheckCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/up" {
			healthCheckCalled = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Extract port from server URL
	var port int
	_, _ = fmt.Sscanf(server.URL, "http://127.0.0.1:%d", &port)

	cfg := &config.Config{
		Applications: config.Applications{
			HealthCheck: "/up",
		},
	}

	ps := NewProcessStarter(cfg)

	tenant := &config.Tenant{
		Name: "test-tenant",
		Env:  map[string]string{},
	}

	app := &WebApp{
		Port:     port,
		Tenant:   tenant,
		Starting: true,
	}

	// Set test mode to skip the check, we're testing the health check logic directly
	err := ps.waitForReady(app, "test-tenant", "test")

	if err != nil {
		t.Errorf("waitForReady() returned error: %v", err)
	}

	if !healthCheckCalled {
		t.Error("Health check endpoint was not called")
	}
}

func TestWaitForReadyAcceptsAnyStatusCode(t *testing.T) {
	// Test that any HTTP response (even errors) is considered "ready"
	statusCodes := []int{
		http.StatusOK,
		http.StatusNotFound,
		http.StatusInternalServerError,
		http.StatusServiceUnavailable,
	}

	for _, statusCode := range statusCodes {
		t.Run(fmt.Sprintf("Status_%d", statusCode), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(statusCode)
			}))
			defer server.Close()

			var port int
			_, _ = fmt.Sscanf(server.URL, "http://127.0.0.1:%d", &port)

			cfg := &config.Config{
				Applications: config.Applications{
					HealthCheck: "/",
				},
			}

			ps := NewProcessStarter(cfg)

			tenant := &config.Tenant{
				Name: "test-tenant",
				Env:  map[string]string{},
			}

			app := &WebApp{
				Port:     port,
				Tenant:   tenant,
				Starting: true,
			}

			err := ps.waitForReady(app, "test-tenant", "test")

			if err != nil {
				t.Errorf("waitForReady() with status %d returned error: %v", statusCode, err)
			}
		})
	}
}
