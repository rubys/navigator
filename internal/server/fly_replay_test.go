package server

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/rubys/navigator/internal/config"
)

func TestShouldUseFlyReplay(t *testing.T) {
	tests := []struct {
		name          string
		method        string
		contentLength int64
		expectReplay  bool
	}{
		// Safe methods (should use replay regardless of content)
		{"GET request", "GET", -1, true},
		{"HEAD request", "HEAD", -1, true},
		{"OPTIONS request", "OPTIONS", -1, true},
		{"DELETE request", "DELETE", -1, true},

		// Methods with small content (should use replay)
		{"POST with small content", "POST", 1024, true},
		{"PUT with small content", "PUT", 1024, true},
		{"PATCH with small content", "PATCH", 1024, true},

		// Methods with large content (should use proxy)
		{"POST with large content", "POST", MaxFlyReplaySize, false},
		{"PUT with large content", "PUT", MaxFlyReplaySize + 1, false},
		{"PATCH with large content", "PATCH", MaxFlyReplaySize * 2, false},

		// Methods with missing content length (should use proxy for body methods)
		{"POST with missing content length", "POST", -1, false},
		{"PUT with missing content length", "PUT", -1, false},
		{"PATCH with missing content length", "PATCH", -1, false},

		// Edge cases
		{"POST with exactly 1MB", "POST", MaxFlyReplaySize, false},
		{"POST with just under 1MB", "POST", MaxFlyReplaySize - 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/test", nil)
			req.ContentLength = tt.contentLength

			result := ShouldUseFlyReplay(req)

			if result != tt.expectReplay {
				t.Errorf("ShouldUseFlyReplay() = %v, expected %v", result, tt.expectReplay)
			}
		})
	}
}

func TestHandleFlyReplay(t *testing.T) {
	tests := []struct {
		name                 string
		target               string
		status               string
		currentApp           string
		retryHeader          string
		expectHandled        bool
		expectStatus         int
		expectContentType    string
		expectMaintenancePage bool
	}{
		{
			name:              "Machine-based replay same app",
			target:            "machine=machine123:myapp",
			status:            "307",
			currentApp:        "myapp",
			expectHandled:     true,
			expectStatus:      307,
			expectContentType: "application/vnd.fly.replay+json",
		},
		{
			name:              "Machine-based replay different app",
			target:            "machine=machine123:otherapp",
			status:            "307",
			currentApp:        "myapp",
			expectHandled:     true,
			expectStatus:      307,
			expectContentType: "application/vnd.fly.replay+json",
		},
		{
			name:              "App-based replay",
			target:            "app=targetapp",
			status:            "302",
			currentApp:        "myapp",
			expectHandled:     true,
			expectStatus:      302,
			expectContentType: "application/vnd.fly.replay+json",
		},
		{
			name:              "Region-based replay",
			target:            "us-west",
			status:            "307",
			currentApp:        "myapp",
			expectHandled:     true,
			expectStatus:      307,
			expectContentType: "application/vnd.fly.replay+json",
		},
		{
			name:                  "Retry detected - maintenance page",
			target:                "us-west",
			status:                "307",
			currentApp:            "myapp",
			retryHeader:           "true",
			expectHandled:         true,
			expectMaintenancePage: true,
		},
		{
			name:              "Default status code",
			target:            "us-east",
			status:            "invalid",
			currentApp:        "myapp",
			expectHandled:     true,
			expectStatus:      307, // Default TemporaryRedirect
			expectContentType: "application/vnd.fly.replay+json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable for current app
			if tt.currentApp != "" {
				os.Setenv("FLY_APP_NAME", tt.currentApp)
				defer os.Unsetenv("FLY_APP_NAME")
			}

			cfg := &config.Config{}
			req := httptest.NewRequest("GET", "/test", nil) // GET to ensure ShouldUseFlyReplay returns true
			if tt.retryHeader != "" {
				req.Header.Set("X-Navigator-Retry", tt.retryHeader)
			}

			recorder := httptest.NewRecorder()

			handled := HandleFlyReplay(recorder, req, tt.target, tt.status, cfg)

			if handled != tt.expectHandled {
				t.Errorf("HandleFlyReplay() returned %v, expected %v", handled, tt.expectHandled)
			}

			if tt.expectMaintenancePage {
				// For maintenance page, we expect either 503 or specific maintenance content
				// The exact behavior depends on ServeMaintenancePage implementation
				return
			}

			if recorder.Code != tt.expectStatus {
				t.Errorf("Status code = %d, expected %d", recorder.Code, tt.expectStatus)
			}

			if recorder.Header().Get("Content-Type") != tt.expectContentType {
				t.Errorf("Content-Type = %q, expected %q",
					recorder.Header().Get("Content-Type"), tt.expectContentType)
			}

			// Verify JSON response structure
			var response map[string]interface{}
			err := json.Unmarshal(recorder.Body.Bytes(), &response)
			if err != nil {
				t.Fatalf("Failed to parse JSON response: %v", err)
			}

			// Verify response contains expected fields based on target type
			if strings.HasPrefix(tt.target, "machine=") {
				if _, ok := response["app"]; !ok {
					t.Error("Machine-based replay should contain 'app' field")
				}
				if _, ok := response["prefer_instance"]; !ok {
					t.Error("Machine-based replay should contain 'prefer_instance' field")
				}
			} else if strings.HasPrefix(tt.target, "app=") {
				if _, ok := response["app"]; !ok {
					t.Error("App-based replay should contain 'app' field")
				}
			} else {
				// Region-based
				if _, ok := response["region"]; !ok {
					t.Error("Region-based replay should contain 'region' field")
				}
			}

			// Check for retry header transform (only for same app or region replays)
			shouldHaveRetryTransform := false
			if strings.HasPrefix(tt.target, "machine=") {
				parts := strings.Split(strings.TrimPrefix(tt.target, "machine="), ":")
				shouldHaveRetryTransform = len(parts) == 2 && parts[1] == tt.currentApp
			} else if strings.HasPrefix(tt.target, "app=") {
				appName := strings.TrimPrefix(tt.target, "app=")
				shouldHaveRetryTransform = appName == tt.currentApp
			} else {
				// Region-based always has retry transform
				shouldHaveRetryTransform = true
			}

			if shouldHaveRetryTransform {
				if _, ok := response["transform"]; !ok {
					t.Error("Should contain retry transform for same-app/region replay")
				}
			}
		})
	}
}

func TestHandleFlyReplay_LargeRequest(t *testing.T) {
	// Set FLY_APP_NAME for fallback to work
	os.Setenv("FLY_APP_NAME", "testapp")
	defer os.Unsetenv("FLY_APP_NAME")

	cfg := &config.Config{}
	cfg.Server.Listen = "3000"

	// Create a request that should trigger fallback (large content)
	req := httptest.NewRequest("POST", "/test", strings.NewReader("large body"))
	req.ContentLength = MaxFlyReplaySize + 1

	recorder := httptest.NewRecorder()

	// This should call HandleFlyReplayFallback instead
	handled := HandleFlyReplay(recorder, req, "us-west", "307", cfg)

	if !handled {
		t.Error("HandleFlyReplay should have been handled (via fallback)")
	}

	// The exact behavior depends on HandleFlyReplayFallback implementation
	// but it should not be a fly-replay response
	contentType := recorder.Header().Get("Content-Type")
	if contentType == "application/vnd.fly.replay+json" {
		t.Error("Large request should not use fly-replay JSON response")
	}
}

func TestServeMaintenancePage(t *testing.T) {
	cfg := &config.Config{}
	req := httptest.NewRequest("GET", "/test", nil)
	recorder := httptest.NewRecorder()

	ServeMaintenancePage(recorder, req, cfg)

	// Should return some kind of maintenance response
	// The exact status and content depend on the implementation
	if recorder.Code == 0 {
		t.Error("ServeMaintenancePage should set a status code")
	}

	if recorder.Body.Len() == 0 {
		t.Error("ServeMaintenancePage should write some content")
	}
}

func BenchmarkShouldUseFlyReplay(b *testing.B) {
	req := httptest.NewRequest("POST", "/test", nil)
	req.ContentLength = 1024

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ShouldUseFlyReplay(req)
	}
}

func BenchmarkHandleFlyReplay(b *testing.B) {
	cfg := &config.Config{}
	req := httptest.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		recorder := httptest.NewRecorder()
		HandleFlyReplay(recorder, req, "us-west", "307", cfg)
	}
}