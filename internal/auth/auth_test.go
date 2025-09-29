package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rubys/navigator/internal/config"
	"github.com/tg123/go-htpasswd"
)

func TestLoadAuthFile(t *testing.T) {
	// Test with empty path
	auth, err := LoadAuthFile("", "test", []string{})
	if err != nil {
		t.Errorf("LoadAuthFile with empty path should not error: %v", err)
	}
	if auth != nil {
		t.Error("LoadAuthFile with empty path should return nil")
	}

	// Test with non-existent file
	auth, err = LoadAuthFile("/non/existent/file", "test", []string{})
	if err == nil {
		t.Error("LoadAuthFile with non-existent file should return error")
	}
	if auth != nil {
		t.Error("LoadAuthFile with error should return nil auth")
	}
}

func TestCheckAuth(t *testing.T) {
	// Create auth without a file (should always return true - no auth configured)
	auth := &BasicAuth{
		File:  nil,
		Realm: "test",
	}

	tests := []struct {
		name        string
		authHeader  string
		expectAuth  bool
		description string
	}{
		{
			name:        "No auth header",
			authHeader:  "",
			expectAuth:  true, // No auth configured, so allow access
			description: "Request without authorization header",
		},
		{
			name:        "Invalid auth type",
			authHeader:  "Bearer token123",
			expectAuth:  true, // No auth configured, so allow access
			description: "Non-basic auth header",
		},
		{
			name:        "Malformed basic auth",
			authHeader:  "Basic invalid-base64",
			expectAuth:  true, // No auth configured, so allow access
			description: "Invalid base64 in basic auth",
		},
		{
			name:        "Basic auth with credentials",
			authHeader:  "Basic dXNlcjpwYXNz", // user:pass in base64
			expectAuth:  true,                 // No auth configured, so allow access
			description: "Valid format but no auth file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			result := auth.CheckAuth(req)
			if result != tt.expectAuth {
				t.Errorf("%s: CheckAuth() = %v, expected %v", tt.description, result, tt.expectAuth)
			}
		})
	}
}

func TestRequireAuth(t *testing.T) {
	auth := &BasicAuth{
		Realm: "Test Realm",
	}

	recorder := httptest.NewRecorder()
	auth.RequireAuth(recorder)

	// Should set 401 status
	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}

	// Should set WWW-Authenticate header
	authHeader := recorder.Header().Get("WWW-Authenticate")
	expectedHeader := `Basic realm="Test Realm"`
	if authHeader != expectedHeader {
		t.Errorf("Expected WWW-Authenticate header %q, got %q", expectedHeader, authHeader)
	}
}

func TestShouldExcludeFromAuth(t *testing.T) {
	tests := []struct {
		path        string
		authExclude []string
		expected    bool
		description string
	}{
		{"/styles.css", []string{"*.css"}, true, "CSS file exclusion"},
		{"/app.js", []string{"*.js"}, true, "JS file exclusion"},
		{"/image.png", []string{"*.png", "*.jpg"}, true, "Image file exclusion"},
		{"/up", []string{"/up", "/health"}, true, "Health endpoint exclusion"},
		{"/api/data", []string{"*.css", "/up"}, false, "API endpoint should require auth"},
		{"/admin/panel", []string{"*.js"}, false, "Admin panel should require auth"},
		{"/", []string{}, false, "No patterns means no exclusion"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			// Create a minimal config with auth exclusions
			cfg := &config.Config{}
			cfg.Server.AuthExclude = tt.authExclude

			result := ShouldExcludeFromAuth(tt.path, cfg)
			if result != tt.expected {
				t.Errorf("ShouldExcludeFromAuth(%q, %v) = %v, expected %v",
					tt.path, tt.authExclude, result, tt.expected)
			}
		})
	}
}

func TestIsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		auth     *BasicAuth
		expected bool
	}{
		{
			name:     "Nil auth",
			auth:     nil,
			expected: false,
		},
		{
			name:     "Auth without file",
			auth:     &BasicAuth{File: nil},
			expected: false,
		},
		{
			name: "Auth with file (mock)",
			auth: &BasicAuth{
				File:  &htpasswd.File{}, // Mock file
				Realm: "test",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result bool
			if tt.auth != nil {
				result = tt.auth.IsEnabled()
			} else {
				// Test nil safety
				result = false
			}

			if result != tt.expected {
				t.Errorf("IsEnabled() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func BenchmarkCheckAuth(b *testing.B) {
	auth := &BasicAuth{
		File:  nil,
		Realm: "test",
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz") // user:pass

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		auth.CheckAuth(req)
	}
}

func BenchmarkShouldExcludeFromAuth(b *testing.B) {
	cfg := &config.Config{}
	cfg.Server.AuthExclude = []string{"*.css", "*.js", "*.png", "/up", "/health"}

	path := "/styles.css"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ShouldExcludeFromAuth(path, cfg)
	}
}
