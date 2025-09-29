package server

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/rubys/navigator/internal/config"
	"github.com/rubys/navigator/internal/idle"
	"github.com/rubys/navigator/internal/process"
)

func TestFlyReplayIntegration_EndToEnd(t *testing.T) {
	// Create test configuration manually
	cfg := &config.Config{}

	// Add fly_replay rules as rewrite rules (simulating what the parser would do)
	cfg.Server.RewriteRules = []config.RewriteRule{
		{
			Pattern:     regexp.MustCompile(`^/showcase/.+\.pdf$`),
			Replacement: "/showcase/.+.pdf",
			Flag:        "fly-replay:app=smooth-pdf:307",
		},
		{
			Pattern:     regexp.MustCompile(`^/showcase/(?:2023|2024|2025|2026)/(?:bellevue|coquitlam|edmonton|everett|folsom|fremont|honolulu|livermore|millbrae|montclair|monterey|petaluma|reno|salem|sanjose|slc|stockton|vegas)(?:/.*)?$`),
			Replacement: "/showcase/sjc-route",
			Flag:        "fly-replay:sjc:307",
		},
		{
			Pattern:     regexp.MustCompile(`^/showcase/(?:2023|2024|2025)/(?:boston|cranford|laval|manchester|marlton|montreal|ottawa|princeton)(?:/.*)?$`),
			Replacement: "/showcase/ewr-route",
			Flag:        "fly-replay:ewr:307",
		},
	}

	// Set up test environment
	os.Setenv("FLY_APP_NAME", "smooth-nav")
	defer os.Unsetenv("FLY_APP_NAME")

	// Create handler
	appManager := &process.AppManager{}
	idleManager := &idle.Manager{}
	handler := CreateHandler(cfg, appManager, nil, idleManager)

	tests := []struct {
		name                  string
		path                  string
		expectFlyReplay       bool
		expectedRegion        string
		expectedApp           string
		expectedStatus        int
		expectedContentType   string
	}{
		{
			name:                "Coquitlam should route to SJC region",
			path:                "/showcase/2025/coquitlam/medal-ball/",
			expectFlyReplay:     true,
			expectedRegion:      "sjc",
			expectedStatus:      307,
			expectedContentType: "application/vnd.fly.replay+json",
		},
		{
			name:                "PDF should route to smooth-pdf app",
			path:                "/showcase/documents/report.pdf",
			expectFlyReplay:     true,
			expectedApp:         "smooth-pdf",
			expectedStatus:      307,
			expectedContentType: "application/vnd.fly.replay+json",
		},
		{
			name:                "Boston should route to EWR region",
			path:                "/showcase/2024/boston/spring-showcase/",
			expectFlyReplay:     true,
			expectedRegion:      "ewr",
			expectedStatus:      307,
			expectedContentType: "application/vnd.fly.replay+json",
		},
		{
			name:            "Non-matching path should not trigger fly-replay",
			path:            "/showcase/2025/unknown-location/",
			expectFlyReplay: false,
		},
		{
			name:            "Regular path should not trigger fly-replay",
			path:            "/other/path/",
			expectFlyReplay: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, req)

			if tt.expectFlyReplay {
				// Should be a fly-replay response
				if recorder.Code != tt.expectedStatus {
					t.Errorf("Status code = %d, expected %d", recorder.Code, tt.expectedStatus)
				}

				if recorder.Header().Get("Content-Type") != tt.expectedContentType {
					t.Errorf("Content-Type = %q, expected %q",
						recorder.Header().Get("Content-Type"), tt.expectedContentType)
				}

				// Parse JSON response
				var response map[string]interface{}
				err := json.Unmarshal(recorder.Body.Bytes(), &response)
				if err != nil {
					t.Fatalf("Failed to parse JSON response: %v", err)
				}

				// Check for expected fields based on type
				if tt.expectedRegion != "" {
					regionField, ok := response["region"].(string)
					if !ok {
						t.Error("Response should contain 'region' field")
					} else {
						// Region field format is "region,any"
						if !strings.HasPrefix(regionField, tt.expectedRegion+",") {
							t.Errorf("Region field = %q, expected to start with %q", regionField, tt.expectedRegion)
						}
					}
				}

				if tt.expectedApp != "" {
					appField, ok := response["app"].(string)
					if !ok {
						t.Error("Response should contain 'app' field")
					} else if appField != tt.expectedApp {
						t.Errorf("App field = %q, expected %q", appField, tt.expectedApp)
					}
				}

			} else {
				// Should NOT be a fly-replay response
				if recorder.Header().Get("Content-Type") == "application/vnd.fly.replay+json" {
					t.Error("Should not be a fly-replay response")
				}
				// Should either be 404 or some other response (depending on whether tenants are configured)
			}
		})
	}
}

func TestFlyReplayIntegration_LargeRequestFallback(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.Listen = "3000"
	cfg.Server.RewriteRules = []config.RewriteRule{
		{
			Pattern:     regexp.MustCompile(`^/upload/`),
			Replacement: "/upload/",
			Flag:        "fly-replay:us-west:307",
		},
	}

	os.Setenv("FLY_APP_NAME", "testapp")
	defer os.Unsetenv("FLY_APP_NAME")

	appManager := &process.AppManager{}
	idleManager := &idle.Manager{}
	handler := CreateHandler(cfg, appManager, nil, idleManager)

	// Create a large POST request that should trigger fallback
	req := httptest.NewRequest("POST", "/upload/large-file", strings.NewReader("large body content"))
	req.ContentLength = MaxFlyReplaySize + 1
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	// Should NOT be a fly-replay JSON response due to size
	if recorder.Header().Get("Content-Type") == "application/vnd.fly.replay+json" {
		t.Error("Large request should not use fly-replay JSON response")
	}

	// Should have been handled by fallback (exact behavior depends on proxy implementation)
	if recorder.Code == 0 {
		t.Error("Request should have been handled")
	}
}

func TestFlyReplayIntegration_RetryHandling(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.RewriteRules = []config.RewriteRule{
		{
			Pattern:     regexp.MustCompile(`^/retry-test/`),
			Replacement: "/retry-test/",
			Flag:        "fly-replay:us-east:307",
		},
	}

	appManager := &process.AppManager{}
	idleManager := &idle.Manager{}
	handler := CreateHandler(cfg, appManager, nil, idleManager)

	// Test request with retry header
	req := httptest.NewRequest("GET", "/retry-test/endpoint", nil)
	req.Header.Set("X-Navigator-Retry", "true")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	// Should serve maintenance page, not fly-replay
	if recorder.Header().Get("Content-Type") == "application/vnd.fly.replay+json" {
		t.Error("Retry request should not use fly-replay, should serve maintenance page")
	}

	// Should have some content (maintenance page)
	if recorder.Body.Len() == 0 {
		t.Error("Retry request should serve maintenance page content")
	}
}

func TestFlyReplayIntegration_MethodFiltering(t *testing.T) {
	cfg := &config.Config{}

	// Manually add a fly-replay rule with method restrictions
	cfg.Server.RewriteRules = append(cfg.Server.RewriteRules, config.RewriteRule{
		Pattern:     mustCompile(`^/api/`),
		Replacement: "/api/",
		Flag:        "fly-replay:us-west:307",
		Methods:     []string{"GET", "HEAD"},
	})

	appManager := &process.AppManager{}
	idleManager := &idle.Manager{}
	handler := CreateHandler(cfg, appManager, nil, idleManager)

	tests := []struct {
		method          string
		expectFlyReplay bool
	}{
		{"GET", true},
		{"HEAD", true},
		{"POST", false},
		{"PUT", false},
		{"DELETE", false},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/endpoint", nil)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, req)

			isFlyReplay := recorder.Header().Get("Content-Type") == "application/vnd.fly.replay+json"

			if tt.expectFlyReplay && !isFlyReplay {
				t.Errorf("Method %s should trigger fly-replay", tt.method)
			}

			if !tt.expectFlyReplay && isFlyReplay {
				t.Errorf("Method %s should not trigger fly-replay", tt.method)
			}
		})
	}
}

// Helper function
func mustCompile(pattern string) *regexp.Regexp {
	re, err := regexp.Compile(pattern)
	if err != nil {
		panic(err)
	}
	return re
}