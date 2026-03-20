package server

import (
	"encoding/json"
	"net/http/httptest"
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

		// Methods with large content (should not use replay)
		{"POST with large content", "POST", MaxFlyReplaySize, false},
		{"PUT with large content", "PUT", MaxFlyReplaySize + 1, false},
		{"PATCH with large content", "PATCH", MaxFlyReplaySize * 2, false},

		// Methods with missing content length (should not use replay for body methods)
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
		name                  string
		target                string
		status                string
		currentApp            string
		failedHeader          string
		expectHandled         bool
		expectStatus          int
		expectContentType     string
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
			name:                  "Failed replay - maintenance page",
			target:                "us-west",
			status:                "307",
			currentApp:            "myapp",
			failedHeader:          "connection_timeout",
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
				t.Setenv("FLY_APP_NAME", tt.currentApp)
			}

			cfg := &config.Config{}
			req := httptest.NewRequest("GET", "/test", nil) // GET to ensure ShouldUseFlyReplay returns true
			if tt.failedHeader != "" {
				req.Header.Set("fly-replay-failed", tt.failedHeader)
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

			// Verify timeout and fallback fields are present
			if _, ok := response["timeout"]; !ok {
				t.Error("Response should contain 'timeout' field")
			}
			if _, ok := response["fallback"]; !ok {
				t.Error("Response should contain 'fallback' field")
			}
			if response["timeout"] != DefaultFlyReplayTimeout {
				t.Errorf("timeout = %v, expected %v", response["timeout"], DefaultFlyReplayTimeout)
			}
			if response["fallback"] != DefaultFlyReplayFallback {
				t.Errorf("fallback = %v, expected %v", response["fallback"], DefaultFlyReplayFallback)
			}

			// Check for transform headers (only for same app or region replays with Authorization)
			// Without Authorization header, no transform should be present
			if strings.HasPrefix(tt.target, "machine=") {
				parts := strings.Split(strings.TrimPrefix(tt.target, "machine="), ":")
				if len(parts) == 2 && parts[1] != tt.currentApp {
					if _, ok := response["transform"]; ok {
						t.Error("Different-app replay should not contain transform")
					}
				}
			} else if strings.HasPrefix(tt.target, "app=") {
				appName := strings.TrimPrefix(tt.target, "app=")
				if appName != tt.currentApp {
					if _, ok := response["transform"]; ok {
						t.Error("Different-app replay should not contain transform")
					}
				}
			}
		})
	}
}

func TestHandleFlyReplay_WithAuthorization(t *testing.T) {
	t.Setenv("FLY_APP_NAME", "myapp")

	cfg := &config.Config{}
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer test-token")

	recorder := httptest.NewRecorder()
	HandleFlyReplay(recorder, req, "us-west", "307", cfg)

	var response map[string]interface{}
	err := json.Unmarshal(recorder.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Region-based replay with Authorization should have transform headers
	transform, ok := response["transform"].(map[string]interface{})
	if !ok {
		t.Fatal("Region-based replay with Authorization should contain transform")
	}

	headers, ok := transform["set_headers"].([]interface{})
	if !ok {
		t.Fatal("Transform should contain set_headers")
	}

	// Should have Authorization header but NOT X-Navigator-Retry
	foundAuth := false
	for _, h := range headers {
		header := h.(map[string]interface{})
		if header["name"] == "X-Navigator-Retry" {
			t.Error("Should not contain X-Navigator-Retry header")
		}
		if header["name"] == "Authorization" {
			foundAuth = true
			if header["value"] != "Bearer test-token" {
				t.Errorf("Authorization value = %v, expected 'Bearer test-token'", header["value"])
			}
		}
	}
	if !foundAuth {
		t.Error("Transform should contain Authorization header")
	}
}

func TestHandleFlyReplay_LargeRequest(t *testing.T) {
	cfg := &config.Config{}

	// Create a request that exceeds the size limit
	req := httptest.NewRequest("POST", "/test", strings.NewReader("large body"))
	req.ContentLength = MaxFlyReplaySize + 1

	recorder := httptest.NewRecorder()

	// Large requests should return false (not handled) so the caller can route normally
	handled := HandleFlyReplay(recorder, req, "us-west", "307", cfg)

	if handled {
		t.Error("HandleFlyReplay should return false for large requests")
	}

	// Should not be a fly-replay response
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
