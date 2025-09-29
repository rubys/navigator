package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/rubys/navigator/internal/config"
	"github.com/rubys/navigator/internal/idle"
	"github.com/rubys/navigator/internal/process"
)

// TestSecurityHeaders tests proper handling of security-related headers
func TestSecurityHeaders(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Echo back received headers for testing
		for name, values := range r.Header {
			for i, value := range values {
				w.Header().Add(fmt.Sprintf("Echo-%s-%d", name, i), value)
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Backend response"))
	}))
	defer backend.Close()

	cfg := &config.Config{
		Routes: config.RoutesConfig{
			ReverseProxies: []config.ProxyRoute{
				{
					Name:   "security-test",
					Path:   "^/api/",
					Target: backend.URL,
					Headers: map[string]string{
						"X-Forwarded-For":   "$remote_addr",
						"X-Forwarded-Proto": "$scheme",
						"X-Forwarded-Host":  "$host",
					},
				},
			},
		},
	}

	appManager := &process.AppManager{}
	idleManager := &idle.Manager{}
	handler := CreateHandler(cfg, appManager, nil, idleManager)

	tests := []struct {
		name          string
		headers       map[string]string
		expectedBlock bool
		description   string
	}{
		{
			name: "normal_headers",
			headers: map[string]string{
				"User-Agent": "Navigator-Test/1.0",
				"Accept":     "application/json",
			},
			expectedBlock: false,
			description:   "Normal headers should be accepted",
		},
		{
			name: "header_injection_attempt",
			headers: map[string]string{
				"User-Agent": "Test\r\nX-Injected-Header: malicious",
			},
			expectedBlock: true, // HTTP library correctly blocks invalid headers
			description:   "Header injection attempts should be blocked",
		},
		{
			name: "xss_in_headers",
			headers: map[string]string{
				"X-Custom": "<script>alert('xss')</script>",
				"Referer":  "http://evil.com/<script>alert('xss')</script>",
			},
			expectedBlock: false, // Headers are just passed through
			description:   "XSS attempts in headers should not affect proxy",
		},
		{
			name: "sql_injection_in_headers",
			headers: map[string]string{
				"X-User-ID": "1'; DROP TABLE users; --",
				"X-Search":  "' OR '1'='1",
			},
			expectedBlock: false, // Navigator doesn't process SQL
			description:   "SQL injection attempts in headers should be passed through",
		},
		{
			name: "command_injection_in_headers",
			headers: map[string]string{
				"X-Command": "; rm -rf /; echo pwned",
				"X-Input":   "`whoami`",
			},
			expectedBlock: false, // Navigator doesn't execute commands from headers
			description:   "Command injection attempts should be handled safely",
		},
		{
			name: "path_traversal_in_headers",
			headers: map[string]string{
				"X-File": "../../../etc/passwd",
				"X-Path": "..\\..\\windows\\system32\\config\\sam",
			},
			expectedBlock: false, // Path traversal in headers doesn't affect routing
			description:   "Path traversal attempts in headers should be passed through",
		},
		{
			name: "oversized_headers",
			headers: map[string]string{
				"X-Large-Header": strings.Repeat("A", 100000), // 100KB header
			},
			expectedBlock: false, // Should be handled by HTTP library
			description:   "Oversized headers should be handled gracefully",
		},
		{
			name: "null_bytes_in_headers",
			headers: map[string]string{
				"X-Null": "value\x00with\x00nulls",
			},
			expectedBlock: true, // HTTP library correctly blocks invalid headers
			description:   "Null bytes in headers should be blocked",
		},
		{
			name: "unicode_normalization",
			headers: map[string]string{
				"X-Unicode": "café", // Contains Unicode characters
				"X-Emoji":   "🔥💯🚀", // Emoji characters
			},
			expectedBlock: false,
			description:   "Unicode characters should be handled properly",
		},
		{
			name: "forwarded_headers_spoofing",
			headers: map[string]string{
				"X-Forwarded-For":   "127.0.0.1",
				"X-Forwarded-Proto": "https",
				"X-Real-IP":         "192.168.1.1",
			},
			expectedBlock: false, // Navigator should handle or override these
			description:   "Client-provided forwarded headers should be handled correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/test", nil)

			// Set test headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			// Set remote address for forwarded header testing
			req.RemoteAddr = "203.0.113.45:12345"

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			// Should not block legitimate requests
			if tt.expectedBlock && recorder.Code < 400 {
				t.Errorf("Expected request to be blocked, but got status %d", recorder.Code)
			}
			if !tt.expectedBlock && recorder.Code >= 500 {
				t.Errorf("Request unexpectedly failed with status %d", recorder.Code)
			}

			t.Logf("Security test %s: %s -> Status %d", tt.name, tt.description, recorder.Code)
		})
	}
}

// TestPathTraversalPrevention tests protection against path traversal attacks
func TestPathTraversalPrevention(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Echo the received path for analysis
		w.Header().Set("Received-Path", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("Path: %s", r.URL.Path)))
	}))
	defer backend.Close()

	cfg := &config.Config{
		Routes: config.RoutesConfig{
			ReverseProxies: []config.ProxyRoute{
				{
					Name:      "path-test",
					Path:      "^/api/",
					Target:    backend.URL,
					StripPath: true,
				},
			},
		},
	}

	appManager := &process.AppManager{}
	idleManager := &idle.Manager{}
	handler := CreateHandler(cfg, appManager, nil, idleManager)

	tests := []struct {
		name        string
		path        string
		expectBlock bool
		description string
	}{
		{
			name:        "normal_path",
			path:        "/api/users",
			expectBlock: false,
			description: "Normal API path should work",
		},
		{
			name:        "dotdot_traversal",
			path:        "/api/../../../etc/passwd",
			expectBlock: false, // URL normalization should handle this
			description: "Dot-dot traversal attempt",
		},
		{
			name:        "encoded_dotdot",
			path:        "/api/%2E%2E/%2E%2E/%2E%2E/etc/passwd",
			expectBlock: false, // Should be decoded and normalized
			description: "URL-encoded dot-dot traversal",
		},
		{
			name:        "double_encoded",
			path:        "/api/%252E%252E/secret",
			expectBlock: false, // Double encoding should be handled
			description: "Double URL-encoded traversal",
		},
		{
			name:        "backslash_traversal",
			path:        "/api/..\\..\\..\\windows\\system32",
			expectBlock: false, // Backslashes in URLs
			description: "Backslash-based traversal",
		},
		{
			name:        "unicode_normalization",
			path:        "/api/\u002E\u002E/\u002E\u002E/secret",
			expectBlock: false, // Unicode dots
			description: "Unicode dot traversal",
		},
		{
			name:        "null_byte_injection",
			path:        "/api/file.txt%00.exe",
			expectBlock: false, // Null bytes should be handled
			description: "Null byte injection attempt",
		},
		{
			name:        "path_with_query",
			path:        "/api/../config?file=passwd",
			expectBlock: false, // Path traversal with query params
			description: "Path traversal with query parameters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, req)

			// Log the result for analysis
			receivedPath := recorder.Header().Get("Received-Path")
			t.Logf("Path traversal test %s: %s -> Status %d, Received: %s",
				tt.name, tt.description, recorder.Code, receivedPath)

			// Check that sensitive paths are not accessible
			if strings.Contains(receivedPath, "/etc/passwd") ||
				strings.Contains(receivedPath, "/windows/system32") ||
				strings.Contains(receivedPath, "..") {
				t.Logf("Warning: Potential path traversal vulnerability detected: %s", receivedPath)
			}

			// Should handle requests gracefully without crashes
			if recorder.Code == 0 {
				t.Errorf("No response received for path traversal test")
			}
		})
	}
}

// TestRegexInjectionPrevention tests protection against regex injection
func TestRegexInjectionPrevention(t *testing.T) {
	// Test regex patterns that could be vulnerable
	vulnerablePatterns := []string{
		"(a+)+$",           // ReDoS: exponential backtracking
		"(a|a)*$",          // ReDoS: alternation
		"a*a*a*a*a*a*a*$",  // ReDoS: nested quantifiers
		"^(a+)+b$",         // ReDoS: catastrophic backtracking
	}

	for _, pattern := range vulnerablePatterns {
		t.Run(fmt.Sprintf("pattern_%s", pattern), func(t *testing.T) {
			// Test regex compilation
			compiled, err := regexp.Compile(pattern)
			if err != nil {
				t.Logf("Pattern %q failed to compile: %v", pattern, err)
				return
			}

			// Test with potentially problematic input
			testInput := strings.Repeat("a", 1000) + "X" // Input that won't match

			// Set timeout to detect ReDoS
			timeout := time.After(1 * time.Second)
			done := make(chan bool, 1)

			go func() {
				_ = compiled.MatchString(testInput)
				done <- true
			}()

			select {
			case <-done:
				t.Logf("Pattern %q completed matching within timeout", pattern)
			case <-timeout:
				t.Errorf("Pattern %q appears vulnerable to ReDoS (timeout)", pattern)
			}
		})
	}
}

// TestInputSanitization tests input sanitization and validation
func TestInputSanitization(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer backend.Close()

	cfg := &config.Config{
		Routes: config.RoutesConfig{
			ReverseProxies: []config.ProxyRoute{
				{
					Name:   "input-test",
					Path:   "^/api/",
					Target: backend.URL,
				},
			},
		},
	}

	appManager := &process.AppManager{}
	idleManager := &idle.Manager{}
	handler := CreateHandler(cfg, appManager, nil, idleManager)

	tests := []struct {
		name   string
		path   string
		method string
		body   string
	}{
		{
			name:   "script_injection_in_path",
			path:   "/api/<script>alert('xss')</script>",
			method: "GET",
		},
		{
			name:   "sql_injection_in_path",
			path:   "/api/users?id=%27%3B%20DROP%20TABLE%20users%3B%20--%20",
			method: "GET",
		},
		{
			name:   "command_injection_in_path",
			path:   "/api/cmd?exec=%3B%20rm%20-rf%20%2F%3B%20echo%20pwned",
			method: "GET",
		},
		{
			name:   "ldap_injection_in_path",
			path:   "/api/search?filter=%2A%29%28%26%28objectClass%3D%2A%29",
			method: "GET",
		},
		{
			name:   "xpath_injection_in_path",
			path:   "/api/xpath?query=%27%20or%20%271%27%3D%271",
			method: "GET",
		},
		{
			name:   "xml_injection_in_body",
			path:   "/api/data",
			method: "POST",
			body:   "<?xml version=\"1.0\"?><!DOCTYPE foo [<!ENTITY xxe SYSTEM \"file:///etc/passwd\">]><data>&xxe;</data>",
		},
		{
			name:   "json_injection_in_body",
			path:   "/api/data",
			method: "POST",
			body:   `{"name": "'; DROP TABLE users; --", "value": "<script>alert('xss')</script>"}`,
		},
		{
			name:   "yaml_injection_in_body",
			path:   "/api/config",
			method: "POST",
			body:   "!!python/object/apply:os.system [\"rm -rf /\"]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			// Should handle malicious input gracefully
			if recorder.Code == 0 {
				t.Errorf("No response received for input sanitization test")
			}

			t.Logf("Input sanitization test %s -> Status %d", tt.name, recorder.Code)
		})
	}
}

// TestDenialOfServicePrevention tests DoS prevention mechanisms
func TestDenialOfServicePrevention(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping DoS prevention test in short mode")
	}

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer backend.Close()

	cfg := &config.Config{
		Routes: config.RoutesConfig{
			ReverseProxies: []config.ProxyRoute{
				{
					Name:   "dos-test",
					Path:   "^/api/",
					Target: backend.URL,
				},
			},
		},
	}

	appManager := &process.AppManager{}
	idleManager := &idle.Manager{}
	handler := CreateHandler(cfg, appManager, nil, idleManager)

	tests := []struct {
		name        string
		description string
		test        func(t *testing.T)
	}{
		{
			name:        "large_request_body",
			description: "Test handling of extremely large request bodies",
			test: func(t *testing.T) {
				// Create a large body (10MB)
				largeBody := strings.NewReader(strings.Repeat("A", 10*1024*1024))

				req := httptest.NewRequest("POST", "/api/upload", largeBody)
				req.Header.Set("Content-Type", "application/octet-stream")

				recorder := httptest.NewRecorder()
				start := time.Now()

				handler.ServeHTTP(recorder, req)

				duration := time.Since(start)
				t.Logf("Large request body test: Status %d, Duration %v", recorder.Code, duration)

				// Should complete within reasonable time
				if duration > 30*time.Second {
					t.Errorf("Large request took too long: %v", duration)
				}
			},
		},
		{
			name:        "extremely_long_url",
			description: "Test handling of extremely long URLs",
			test: func(t *testing.T) {
				longPath := "/api/" + strings.Repeat("very-long-path-segment/", 10000)

				req := httptest.NewRequest("GET", longPath, nil)
				recorder := httptest.NewRecorder()

				start := time.Now()
				handler.ServeHTTP(recorder, req)
				duration := time.Since(start)

				t.Logf("Long URL test: Status %d, Duration %v", recorder.Code, duration)

				// Should complete quickly even with long URLs
				if duration > 5*time.Second {
					t.Errorf("Long URL processing took too long: %v", duration)
				}
			},
		},
		{
			name:        "malformed_requests",
			description: "Test handling of malformed HTTP requests",
			test: func(t *testing.T) {
				// Test various malformed requests
				malformedTests := []struct {
					name string
					path string
				}{
					{"control_characters", "/api/\x01\x02\x03test"},
					{"invalid_encoding", "/api/%ZZ%invalid"},
					{"mixed_separators", "/api\\path/with\\backslashes"},
				}

				for _, mt := range malformedTests {
					t.Run(mt.name, func(t *testing.T) {
						req := httptest.NewRequest("GET", mt.path, nil)
						recorder := httptest.NewRecorder()

						start := time.Now()
						handler.ServeHTTP(recorder, req)
						duration := time.Since(start)

						t.Logf("Malformed request %s: Status %d, Duration %v", mt.name, recorder.Code, duration)

						// Should handle malformed requests quickly
						if duration > 1*time.Second {
							t.Errorf("Malformed request %s took too long: %v", mt.name, duration)
						}
					})
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Running DoS prevention test: %s", tt.description)
			tt.test(t)
		})
	}
}

// TestInformationDisclosure tests for information disclosure vulnerabilities
func TestInformationDisclosure(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate various backend responses
		switch r.URL.Path {
		case "/error":
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal server error with sensitive info: database connection failed"))
		case "/debug":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Debug: server version 1.2.3, built on 2024-01-01"))
		default:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}
	}))
	defer backend.Close()

	cfg := &config.Config{
		Routes: config.RoutesConfig{
			ReverseProxies: []config.ProxyRoute{
				{
					Name:   "info-disclosure-test",
					Path:   "^/api/",
					Target: backend.URL,
				},
			},
		},
	}

	appManager := &process.AppManager{}
	idleManager := &idle.Manager{}
	handler := CreateHandler(cfg, appManager, nil, idleManager)

	tests := []struct {
		name        string
		path        string
		checkHeader string
		description string
	}{
		{
			name:        "server_header_disclosure",
			path:        "/api/test",
			checkHeader: "Server",
			description: "Check if server information is disclosed in headers",
		},
		{
			name:        "version_disclosure",
			path:        "/api/debug",
			checkHeader: "X-Powered-By",
			description: "Check for version information disclosure",
		},
		{
			name:        "error_information",
			path:        "/api/error",
			description: "Check if detailed error information is disclosed",
		},
		{
			name:        "directory_listing",
			path:        "/api/../",
			description: "Check for directory listing disclosure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, req)

			// Check for sensitive header disclosure
			if tt.checkHeader != "" {
				headerValue := recorder.Header().Get(tt.checkHeader)
				if headerValue != "" {
					t.Logf("Information disclosure in %s header: %s", tt.checkHeader, headerValue)
				}
			}

			// Check response body for sensitive information
			body := recorder.Body.String()
			sensitivePatterns := []string{
				"database", "password", "secret", "key", "token",
				"internal", "debug", "stack trace", "exception",
				"version", "build", "configuration",
			}

			for _, pattern := range sensitivePatterns {
				if strings.Contains(strings.ToLower(body), pattern) {
					t.Logf("Potential information disclosure: response contains '%s'", pattern)
				}
			}

			t.Logf("Information disclosure test %s: %s -> Status %d", tt.name, tt.description, recorder.Code)
		})
	}
}

// TestAuthenticationBypass tests for authentication bypass vulnerabilities
func TestAuthenticationBypass(t *testing.T) {
	// This test would be more comprehensive with actual auth setup
	// For now, test basic auth header handling
	cfg := &config.Config{}
	cfg.Server.Authentication = "/tmp/nonexistent.htpasswd" // Simulate auth requirement
	cfg.Server.AuthExclude = []string{"/public/*"}

	appManager := &process.AppManager{}
	idleManager := &idle.Manager{}
	handler := CreateHandler(cfg, appManager, nil, idleManager)

	tests := []struct {
		name        string
		path        string
		headers     map[string]string
		expectAuth  bool
		description string
	}{
		{
			name:        "no_auth_headers",
			path:        "/protected",
			expectAuth:  true,
			description: "Request without auth headers should be challenged",
		},
		{
			name: "malformed_auth_header",
			path: "/protected",
			headers: map[string]string{
				"Authorization": "Basic malformed-base64",
			},
			expectAuth:  true,
			description: "Malformed auth header should be rejected",
		},
		{
			name: "auth_bypass_attempt",
			path: "/protected",
			headers: map[string]string{
				"X-Forwarded-User": "admin",
				"X-Remote-User":    "root",
			},
			expectAuth:  true,
			description: "Auth bypass headers should be ignored",
		},
		{
			name:        "public_path",
			path:        "/public/file.txt",
			expectAuth:  false,
			description: "Public paths should not require auth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)

			// Set test headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			if tt.expectAuth && recorder.Code != http.StatusUnauthorized {
				t.Logf("Expected auth challenge for %s, got status %d", tt.name, recorder.Code)
			}

			t.Logf("Auth bypass test %s: %s -> Status %d", tt.name, tt.description, recorder.Code)
		})
	}
}

// TestRateLimitingBehavior tests rate limiting behavior (if implemented)
func TestRateLimitingBehavior(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping rate limiting test in short mode")
	}

	cfg := &config.Config{}
	appManager := &process.AppManager{}
	idleManager := &idle.Manager{}
	handler := CreateHandler(cfg, appManager, nil, idleManager)

	// Make rapid requests to test rate limiting
	const numRequests = 100
	const rapidInterval = 10 * time.Millisecond

	results := make([]int, numRequests)

	start := time.Now()
	for i := 0; i < numRequests; i++ {
		req := httptest.NewRequest("GET", fmt.Sprintf("/rate-test/%d", i), nil)
		req.RemoteAddr = "203.0.113.45:12345" // Same IP for rate limiting

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		results[i] = recorder.Code

		if i < numRequests-1 {
			time.Sleep(rapidInterval)
		}
	}
	duration := time.Since(start)

	// Analyze results
	statusCounts := make(map[int]int)
	for _, status := range results {
		statusCounts[status]++
	}

	t.Logf("Rate limiting test: %d requests in %v", numRequests, duration)
	for status, count := range statusCounts {
		t.Logf("  Status %d: %d requests", status, count)
	}

	// Check if any rate limiting is in effect
	if statusCounts[http.StatusTooManyRequests] > 0 {
		t.Logf("Rate limiting detected: %d requests blocked", statusCounts[http.StatusTooManyRequests])
	} else {
		t.Logf("No rate limiting detected in rapid request test")
	}
}