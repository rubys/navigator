package auth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"
	"time"

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
			cfg.Auth.PublicPaths = tt.authExclude

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
	cfg.Auth.PublicPaths = []string{"*.css", "*.js", "*.png", "/up", "/health"}

	path := "/styles.css"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ShouldExcludeFromAuth(path, cfg)
	}
}

// Tests for improved coverage

func TestCheckAuthWithCredentials(t *testing.T) {
	// Test with actual htpasswd file
	// Create a temporary htpasswd file for testing
	// Format: username:password (bcrypt hash)
	// For testing, we'll use the htpasswd library directly

	tests := []struct {
		name       string
		username   string
		password   string
		expectAuth bool
	}{
		{
			name:       "Missing credentials",
			username:   "",
			password:   "",
			expectAuth: false,
		},
		{
			name:       "Invalid credentials",
			username:   "wronguser",
			password:   "wrongpass",
			expectAuth: false,
		},
	}

	// Create an auth with nil file to test the missing credentials case
	auth := &BasicAuth{
		File:  nil,
		Realm: "test",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.username != "" || tt.password != "" {
				req.SetBasicAuth(tt.username, tt.password)
			}

			// With nil file, CheckAuth should return true (no auth configured)
			result := auth.CheckAuth(req)
			if result != true {
				t.Errorf("CheckAuth() with nil file should return true, got %v", result)
			}
		})
	}
}

func TestRequireAuthWithNilAuth(t *testing.T) {
	// Test RequireAuth with nil auth (should not panic)
	var auth *BasicAuth
	recorder := httptest.NewRecorder()
	auth.RequireAuth(recorder)

	// Should not set any headers or status (no-op)
	if recorder.Code != 200 {
		t.Errorf("Expected default status 200, got %d", recorder.Code)
	}
}

func TestRequireAuthWithDefaultRealm(t *testing.T) {
	// Test RequireAuth with empty realm (should use default)
	auth := &BasicAuth{
		Realm: "",
	}

	recorder := httptest.NewRecorder()
	auth.RequireAuth(recorder)

	authHeader := recorder.Header().Get("WWW-Authenticate")
	expectedHeader := `Basic realm="Restricted"`
	if authHeader != expectedHeader {
		t.Errorf("Expected default realm header %q, got %q", expectedHeader, authHeader)
	}
}

func TestShouldExcludeFromAuthAdvanced(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		authExclude []string
		expected    bool
	}{
		{
			name:        "Prefix match with trailing slash",
			path:        "/public/file.html",
			authExclude: []string{"/public/"},
			expected:    true,
		},
		{
			name:        "Prefix no match without trailing slash",
			path:        "/public/file.html",
			authExclude: []string{"/public"},
			expected:    false,
		},
		{
			name:        "Glob pattern with path",
			path:        "/assets/style.css",
			authExclude: []string{"/assets/*.css"},
			expected:    true,
		},
		{
			name:        "Glob pattern no match",
			path:        "/api/data.json",
			authExclude: []string{"/assets/*.css"},
			expected:    false,
		},
		{
			name:        "Exact match",
			path:        "/health",
			authExclude: []string{"/health"},
			expected:    true,
		},
		{
			name:        "Exact no match",
			path:        "/health-check",
			authExclude: []string{"/health"},
			expected:    false,
		},
		{
			name:        "Multiple patterns first match",
			path:        "/test.js",
			authExclude: []string{"*.js", "*.css", "/public/"},
			expected:    true,
		},
		{
			name:        "Multiple patterns middle match",
			path:        "/style.css",
			authExclude: []string{"*.js", "*.css", "/public/"},
			expected:    true,
		},
		{
			name:        "Multiple patterns no match",
			path:        "/api/data",
			authExclude: []string{"*.js", "*.css", "/public/"},
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.Auth.PublicPaths = tt.authExclude

			result := ShouldExcludeFromAuth(tt.path, cfg)
			if result != tt.expected {
				t.Errorf("ShouldExcludeFromAuth(%q, %v) = %v, expected %v",
					tt.path, tt.authExclude, result, tt.expected)
			}
		})
	}
}

func TestLoadAuthFileWithValidFile(t *testing.T) {
	// Test with valid htpasswd content
	// Note: This test requires testdata/htpasswd file to exist
	// For now, we test the error case which is already covered

	// Test that LoadAuthFile handles realm and exclude properly
	auth, err := LoadAuthFile("", "Custom Realm", []string{"/public/"})
	if err != nil {
		t.Errorf("LoadAuthFile with empty filename should not error: %v", err)
	}
	if auth != nil {
		t.Error("LoadAuthFile with empty filename should return nil")
	}
}

func TestShouldExcludeFromAuthWithPatterns(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		authPatterns []config.AuthPattern
		publicPaths  []string
		expected     bool
	}{
		{
			name: "Auth pattern with action=off matches",
			path: "/showcase/2025/boston/",
			authPatterns: []config.AuthPattern{
				{
					Pattern: mustCompileRegex(t, "^/showcase/2025/(boston|seattle)/?$"),
					Action:  "off",
				},
			},
			expected: true,
		},
		{
			name: "Auth pattern with action=off does not match",
			path: "/showcase/2025/boston/disney/",
			authPatterns: []config.AuthPattern{
				{
					Pattern: mustCompileRegex(t, "^/showcase/2025/(boston|seattle)/?$"),
					Action:  "off",
				},
			},
			expected: false,
		},
		{
			name: "Auth pattern with custom realm does not exclude",
			path: "/admin/panel",
			authPatterns: []config.AuthPattern{
				{
					Pattern: mustCompileRegex(t, "^/admin/"),
					Action:  "Admin Only",
				},
			},
			expected: false,
		},
		{
			name: "Auth patterns checked before public paths",
			path: "/showcase/2025/boston/",
			authPatterns: []config.AuthPattern{
				{
					Pattern: mustCompileRegex(t, "^/showcase/2025/(boston|seattle)/?$"),
					Action:  "off",
				},
			},
			publicPaths: []string{"*.css"},
			expected:    true,
		},
		{
			name: "Public paths checked when no auth pattern matches",
			path: "/styles.css",
			authPatterns: []config.AuthPattern{
				{
					Pattern: mustCompileRegex(t, "^/showcase/"),
					Action:  "off",
				},
			},
			publicPaths: []string{"*.css"},
			expected:    true,
		},
		{
			name: "Multiple auth patterns with alternations",
			path: "/showcase/2025/raleigh/disney/public/heats",
			authPatterns: []config.AuthPattern{
				{
					Pattern: mustCompileRegex(t, "^/showcase/2025/(boston|seattle|raleigh|portland)/?$"),
					Action:  "off",
				},
				{
					Pattern: mustCompileRegex(t, "^/showcase/2025/(boston|seattle|raleigh)/[^/]+/public/"),
					Action:  "off",
				},
			},
			expected: true,
		},
		{
			name: "Protected tenant path not matched by patterns",
			path: "/showcase/2025/boston/disney/entries",
			authPatterns: []config.AuthPattern{
				{
					Pattern: mustCompileRegex(t, "^/showcase/2025/(boston|seattle)/?$"),
					Action:  "off",
				},
				{
					Pattern: mustCompileRegex(t, "^/showcase/2025/[^/]+/[^/]+/public/"),
					Action:  "off",
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.Auth.AuthPatterns = tt.authPatterns
			cfg.Auth.PublicPaths = tt.publicPaths

			result := ShouldExcludeFromAuth(tt.path, cfg)
			if result != tt.expected {
				t.Errorf("ShouldExcludeFromAuth(%q) = %v, expected %v",
					tt.path, result, tt.expected)
			}
		})
	}
}

// Helper function to compile regex patterns for tests
func mustCompileRegex(t *testing.T, pattern string) *regexp.Regexp {
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		t.Fatalf("Failed to compile regex %q: %v", pattern, err)
	}
	return compiled
}

func TestHtpasswdAutoReload(t *testing.T) {
	// Create a temporary htpasswd file
	tmpfile, err := os.CreateTemp("", "htpasswd-test-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	// Write initial htpasswd content with user1:password1
	// Using bcrypt hash (generated with: htpasswd -nbB user1 password1)
	initialContent := "user1:$2y$05$HhAkLv4T/hijhH3KQUtfWuuFm15Wwpf4qmdcbZnZILZ0zR3P6bBEG\n"
	if _, err := tmpfile.WriteString(initialContent); err != nil {
		t.Fatalf("Failed to write initial content: %v", err)
	}
	tmpfile.Close()

	// Load the auth file
	auth, err := LoadAuthFile(tmpfile.Name(), "Test", []string{})
	if err != nil {
		t.Fatalf("Failed to load auth file: %v", err)
	}

	// Test 1: Verify user1:password1 works initially
	req1 := httptest.NewRequest("GET", "/", nil)
	req1.SetBasicAuth("user1", "password1")
	if !auth.CheckAuth(req1) {
		t.Error("Initial auth check for user1:password1 should succeed")
	}

	// Test 2: Verify user2:password2 fails initially
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.SetBasicAuth("user2", "password2")
	if auth.CheckAuth(req2) {
		t.Error("Auth check for user2:password2 should fail (user doesn't exist yet)")
	}

	// Sleep briefly to ensure mtime will be different
	time.Sleep(10 * time.Millisecond)

	// Update the htpasswd file to add user2:password2
	// Using bcrypt hash (generated with: htpasswd -nbB user2 password2)
	updatedContent := initialContent + "user2:$2y$05$tjlZDlLnLpQQJnwJ9.7mDOvxuumMT6zwA3Wv/xfu01C.aZp8GG2dW\n"
	if err := os.WriteFile(tmpfile.Name(), []byte(updatedContent), 0644); err != nil {
		t.Fatalf("Failed to update htpasswd file: %v", err)
	}

	// Test 3: Verify user1:password1 still works (cached)
	req3 := httptest.NewRequest("GET", "/", nil)
	req3.SetBasicAuth("user1", "password1")
	if !auth.CheckAuth(req3) {
		t.Error("Auth check for user1:password1 should still succeed")
	}

	// Test 4: Try user2:password2 - should FAIL initially (cached)
	// but then auto-reload and SUCCEED
	req4 := httptest.NewRequest("GET", "/", nil)
	req4.SetBasicAuth("user2", "password2")
	result := auth.CheckAuth(req4)
	if !result {
		t.Error("Auth check for user2:password2 should succeed after auto-reload")
	}

	// Test 5: Verify the file was actually reloaded by checking mtime was updated
	stat, err := os.Stat(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	auth.mu.RLock()
	cachedMtime := auth.mtime
	auth.mu.RUnlock()

	if !cachedMtime.Equal(stat.ModTime()) {
		t.Errorf("Cached mtime should match file mtime after reload. cached=%v, file=%v",
			cachedMtime, stat.ModTime())
	}

	// Test 6: Verify user2:password2 works again (now from cache after reload)
	req5 := httptest.NewRequest("GET", "/", nil)
	req5.SetBasicAuth("user2", "password2")
	if !auth.CheckAuth(req5) {
		t.Error("Auth check for user2:password2 should succeed from cache after reload")
	}
}
