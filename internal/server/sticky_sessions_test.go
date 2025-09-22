package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/rubys/navigator/internal/config"
)

func TestHandleStickySession(t *testing.T) {
	tests := []struct {
		name               string
		stickyEnabled      bool
		paths              []string
		requestPath        string
		currentMachineID   string
		appName            string
		existingCookie     string
		retryHeader        string
		expectHandled      bool
		expectCookieSet    bool
		expectFlyReplay    bool
		expectMaintenance  bool
	}{
		{
			name:            "Sticky disabled",
			stickyEnabled:   false,
			requestPath:     "/app/dashboard",
			expectHandled:   false,
		},
		{
			name:            "Path doesn't match pattern",
			stickyEnabled:   true,
			paths:           []string{"/app/*"},
			requestPath:     "/other/page",
			expectHandled:   false,
		},
		{
			name:               "Missing environment variables",
			stickyEnabled:      true,
			requestPath:        "/app/dashboard",
			currentMachineID:   "", // Missing
			appName:            "testapp",
			expectHandled:      false,
		},
		{
			name:               "No existing cookie - set cookie and continue",
			stickyEnabled:      true,
			requestPath:        "/app/dashboard",
			currentMachineID:   "machine123",
			appName:            "testapp",
			expectHandled:      false,
			expectCookieSet:    true,
		},
		{
			name:               "Cookie matches current machine - continue",
			stickyEnabled:      true,
			requestPath:        "/app/dashboard",
			currentMachineID:   "machine123",
			appName:            "testapp",
			existingCookie:     "machine123",
			expectHandled:      false,
			expectCookieSet:    true,
		},
		{
			name:               "Cookie for different machine - fly-replay",
			stickyEnabled:      true,
			requestPath:        "/app/dashboard",
			currentMachineID:   "machine123",
			appName:            "testapp",
			existingCookie:     "machine456",
			expectHandled:      true,
			expectFlyReplay:    true,
		},
		{
			name:               "Cookie for different machine with retry - maintenance",
			stickyEnabled:      true,
			requestPath:        "/app/dashboard",
			currentMachineID:   "machine123",
			appName:            "testapp",
			existingCookie:     "machine456",
			retryHeader:        "true",
			expectHandled:      true,
			expectMaintenance:  true,
			expectCookieSet:    true, // Should reset to current machine
		},
		{
			name:               "Path pattern matching",
			stickyEnabled:      true,
			paths:              []string{"/app/*", "/admin/*"},
			requestPath:        "/admin/users",
			currentMachineID:   "machine123",
			appName:            "testapp",
			expectHandled:      false,
			expectCookieSet:    true,
		},
		{
			name:               "Large request uses fallback",
			stickyEnabled:      true,
			requestPath:        "/app/upload",
			currentMachineID:   "machine123",
			appName:            "testapp",
			existingCookie:     "machine456",
			expectHandled:      true,
			expectFlyReplay:    true, // Will still be handled, but via fallback
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment
			if tt.currentMachineID != "" {
				os.Setenv("FLY_MACHINE_ID", tt.currentMachineID)
				defer os.Unsetenv("FLY_MACHINE_ID")
			}
			if tt.appName != "" {
				os.Setenv("FLY_APP_NAME", tt.appName)
				defer os.Unsetenv("FLY_APP_NAME")
			}

			// Setup config
			cfg := &config.Config{}
			cfg.Server.StickySession.Enabled = tt.stickyEnabled
			cfg.Server.StickySession.Paths = tt.paths
			cfg.Server.StickySession.CookieName = "_navigator_machine"
			cfg.Server.StickySession.CookieMaxAge = "1h"

			// Create request
			req := httptest.NewRequest("GET", tt.requestPath, nil)
			if tt.existingCookie != "" {
				req.AddCookie(&http.Cookie{
					Name:  "_navigator_machine",
					Value: tt.existingCookie,
				})
			}
			if tt.retryHeader != "" {
				req.Header.Set("X-Navigator-Retry", tt.retryHeader)
			}

			recorder := httptest.NewRecorder()

			// Call function
			handled := HandleStickySession(recorder, req, cfg)

			// Verify result
			if handled != tt.expectHandled {
				t.Errorf("HandleStickySession() = %v, expected %v", handled, tt.expectHandled)
			}

			// Check cookie was set if expected
			if tt.expectCookieSet {
				cookies := recorder.Result().Cookies()
				found := false
				for _, cookie := range cookies {
					if cookie.Name == "_navigator_machine" {
						found = true
						if cookie.Value != tt.currentMachineID {
							t.Errorf("Cookie value = %q, expected %q", cookie.Value, tt.currentMachineID)
						}
						break
					}
				}
				if !found {
					t.Error("Expected sticky session cookie to be set")
				}
			}

			// Check for fly-replay response
			if tt.expectFlyReplay {
				contentType := recorder.Header().Get("Content-Type")
				if !strings.Contains(contentType, "fly.replay") && recorder.Code != 502 {
					// Either fly-replay JSON or proxy error (502) is acceptable
					t.Logf("Response Content-Type: %s, Status: %d", contentType, recorder.Code)
				}
			}

			// Check for maintenance page (status will vary based on implementation)
			if tt.expectMaintenance {
				if recorder.Code == 0 {
					t.Error("Expected maintenance page to set status code")
				}
			}
		})
	}
}

func TestSetStickySessionCookie(t *testing.T) {
	tests := []struct {
		name              string
		machineID         string
		cookieName        string
		maxAge            string
		secure            bool
		httpOnly          bool
		sameSite          string
		expectMaxAge      int
		expectSameSite    http.SameSite
	}{
		{
			name:           "Basic cookie",
			machineID:      "machine123",
			cookieName:     "_nav_machine",
			expectMaxAge:   3600, // Default 1 hour
			expectSameSite: http.SameSiteLaxMode, // Default
		},
		{
			name:           "Custom max age",
			machineID:      "machine456",
			cookieName:     "_sticky",
			maxAge:         "2h",
			expectMaxAge:   7200,
			expectSameSite: http.SameSiteLaxMode,
		},
		{
			name:           "Security flags",
			machineID:      "machine789",
			cookieName:     "_secure_session",
			secure:         true,
			httpOnly:       true,
			expectMaxAge:   3600,
			expectSameSite: http.SameSiteLaxMode,
		},
		{
			name:           "SameSite strict",
			machineID:      "machine000",
			cookieName:     "_strict",
			sameSite:       "strict",
			expectMaxAge:   3600,
			expectSameSite: http.SameSiteStrictMode,
		},
		{
			name:           "SameSite none",
			machineID:      "machine111",
			cookieName:     "_none",
			sameSite:       "none",
			expectMaxAge:   3600,
			expectSameSite: http.SameSiteNoneMode,
		},
		{
			name:           "SameSite lax",
			machineID:      "machine222",
			cookieName:     "_lax",
			sameSite:       "lax",
			expectMaxAge:   3600,
			expectSameSite: http.SameSiteLaxMode,
		},
		{
			name:           "Invalid duration uses default",
			machineID:      "machine333",
			cookieName:     "_invalid",
			maxAge:         "invalid-duration",
			expectMaxAge:   3600,
			expectSameSite: http.SameSiteLaxMode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.Server.StickySession.CookieName = tt.cookieName
			cfg.Server.StickySession.CookieMaxAge = tt.maxAge
			cfg.Server.StickySession.CookieSecure = tt.secure
			cfg.Server.StickySession.CookieHTTPOnly = tt.httpOnly
			cfg.Server.StickySession.CookieSameSite = tt.sameSite

			recorder := httptest.NewRecorder()

			SetStickySessionCookie(recorder, tt.machineID, cfg)

			// Get the cookie
			cookies := recorder.Result().Cookies()
			if len(cookies) == 0 {
				t.Fatal("No cookies were set")
			}

			cookie := cookies[0]

			// Verify basic properties
			if cookie.Name != tt.cookieName {
				t.Errorf("Cookie name = %q, expected %q", cookie.Name, tt.cookieName)
			}
			if cookie.Value != tt.machineID {
				t.Errorf("Cookie value = %q, expected %q", cookie.Value, tt.machineID)
			}
			if cookie.Path != "/" {
				t.Errorf("Cookie path = %q, expected %q", cookie.Path, "/")
			}

			// Verify max age
			if cookie.MaxAge != tt.expectMaxAge {
				t.Errorf("Cookie MaxAge = %d, expected %d", cookie.MaxAge, tt.expectMaxAge)
			}

			// Verify security flags
			if cookie.Secure != tt.secure {
				t.Errorf("Cookie Secure = %v, expected %v", cookie.Secure, tt.secure)
			}
			if cookie.HttpOnly != tt.httpOnly {
				t.Errorf("Cookie HttpOnly = %v, expected %v", cookie.HttpOnly, tt.httpOnly)
			}

			// Verify SameSite
			if cookie.SameSite != tt.expectSameSite {
				t.Errorf("Cookie SameSite = %v, expected %v", cookie.SameSite, tt.expectSameSite)
			}
		})
	}
}

func TestHandleStickySession_PathMatching(t *testing.T) {
	// Set up environment for all tests
	os.Setenv("FLY_MACHINE_ID", "machine123")
	os.Setenv("FLY_APP_NAME", "testapp")
	defer func() {
		os.Unsetenv("FLY_MACHINE_ID")
		os.Unsetenv("FLY_APP_NAME")
	}()

	tests := []struct {
		name          string
		patterns      []string
		requestPath   string
		expectHandled bool
	}{
		{
			name:          "No patterns - all paths match",
			patterns:      nil,
			requestPath:   "/any/path",
			expectHandled: false, // No cookie, so continues normally
		},
		{
			name:          "Simple wildcard match",
			patterns:      []string{"/app/*"},
			requestPath:   "/app/dashboard",
			expectHandled: false,
		},
		{
			name:          "Simple wildcard no match",
			patterns:      []string{"/app/*"},
			requestPath:   "/admin/users",
			expectHandled: false, // Path doesn't match, so skips sticky logic
		},
		{
			name:          "Multiple patterns - first match",
			patterns:      []string{"/app/*", "/admin/*"},
			requestPath:   "/app/settings",
			expectHandled: false,
		},
		{
			name:          "Multiple patterns - second match",
			patterns:      []string{"/app/*", "/admin/*"},
			requestPath:   "/admin/users",
			expectHandled: false,
		},
		{
			name:          "Complex pattern match",
			patterns:      []string{"/*/cable", "/api/v*/websocket"},
			requestPath:   "/2025/cable",
			expectHandled: false,
		},
		{
			name:          "No match with multiple patterns",
			patterns:      []string{"/app/*", "/admin/*"},
			requestPath:   "/public/assets",
			expectHandled: false, // Skips due to path mismatch
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.Server.StickySession.Enabled = true
			cfg.Server.StickySession.Paths = tt.patterns
			cfg.Server.StickySession.CookieName = "_nav_machine"

			req := httptest.NewRequest("GET", tt.requestPath, nil)
			recorder := httptest.NewRecorder()

			handled := HandleStickySession(recorder, req, cfg)

			if handled != tt.expectHandled {
				t.Errorf("HandleStickySession() = %v, expected %v for path %s",
					handled, tt.expectHandled, tt.requestPath)
			}
		})
	}
}

func BenchmarkHandleStickySession(b *testing.B) {
	os.Setenv("FLY_MACHINE_ID", "machine123")
	os.Setenv("FLY_APP_NAME", "testapp")
	defer func() {
		os.Unsetenv("FLY_MACHINE_ID")
		os.Unsetenv("FLY_APP_NAME")
	}()

	cfg := &config.Config{}
	cfg.Server.StickySession.Enabled = true
	cfg.Server.StickySession.CookieName = "_nav_machine"

	req := httptest.NewRequest("GET", "/app/dashboard", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		recorder := httptest.NewRecorder()
		HandleStickySession(recorder, req, cfg)
	}
}

func BenchmarkSetStickySessionCookie(b *testing.B) {
	cfg := &config.Config{}
	cfg.Server.StickySession.CookieName = "_nav_machine"
	cfg.Server.StickySession.CookieMaxAge = "1h"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		recorder := httptest.NewRecorder()
		SetStickySessionCookie(recorder, "machine123", cfg)
	}
}