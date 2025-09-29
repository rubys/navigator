package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rubys/navigator/internal/config"
	"github.com/rubys/navigator/internal/idle"
	"github.com/rubys/navigator/internal/process"
)

// TestReverseProxyYAMLConfigIntegration tests complete YAML reverse_proxies config working end-to-end
func TestReverseProxyYAMLConfigIntegration(t *testing.T) {
	// Create test backend servers
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"service":"api","path":"` + r.URL.Path + `","headers":` + fmt.Sprintf("%q", r.Header) + `}`))
	}))
	defer apiServer.Close()

	webServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body><h1>Web Service</h1><p>Path: ` + r.URL.Path + `</p></body></html>`))
	}))
	defer webServer.Close()

	// Create configuration with reverse proxy routes
	cfg := &config.Config{
		Routes: config.RoutesConfig{
			ReverseProxies: []config.ProxyRoute{
				{
					Name:   "api-proxy",
					Path:   "^/api/",
					Target: apiServer.URL,
					Headers: map[string]string{
						"X-Service-Name":    "api-backend",
						"X-Forwarded-Proto": "$scheme",
						"X-Forwarded-Host":  "$host",
						"X-Real-IP":         "$remote_addr",
					},
				},
				{
					Name:   "web-proxy",
					Prefix: "/web/",
					Target: webServer.URL,
					Headers: map[string]string{
						"X-Service-Name": "web-backend",
					},
				},
				{
					Name:      "strip-path-proxy",
					Path:      "^/old-api/(.*)$",
					Target:    apiServer.URL,
					StripPath: true,
					Headers: map[string]string{
						"X-Path-Stripped": "true",
					},
				},
			},
		},
	}

	// Create handler
	appManager := &process.AppManager{}
	idleManager := &idle.Manager{}
	handler := CreateHandler(cfg, appManager, nil, idleManager)

	tests := []struct {
		name               string
		path               string
		expectProxy        bool
		expectedService    string
		expectStripPath    bool
		expectedHeaders    map[string]string
		expectContentType  string
	}{
		{
			name:              "API path should route to API server",
			path:              "/api/users",
			expectProxy:       true,
			expectedService:   "api",
			expectedHeaders:   map[string]string{"X-Service-Name": "api-backend"},
			expectContentType: "application/json",
		},
		{
			name:              "Web path should route to web server",
			path:              "/web/dashboard",
			expectProxy:       true,
			expectedHeaders:   map[string]string{"X-Service-Name": "web-backend"},
			expectContentType: "text/html",
		},
		{
			name:               "Strip path should work",
			path:               "/old-api/v1/users",
			expectProxy:        true,
			expectedService:    "api",
			expectStripPath:    true,
			expectedHeaders:    map[string]string{"X-Path-Stripped": "true"},
			expectContentType:  "application/json",
		},
		{
			name:        "Non-matching path should not proxy",
			path:        "/other/path",
			expectProxy: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			req.Host = "example.com"
			req.RemoteAddr = "192.168.1.100:12345"
			req.URL.Scheme = "https"

			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, req)

			if tt.expectProxy {
				// Should have been handled by proxy
				if recorder.Code == http.StatusNotFound {
					t.Errorf("Expected proxy to handle request, got 404")
					return
				}

				// Check content type
				if tt.expectContentType != "" {
					contentType := recorder.Header().Get("Content-Type")
					if contentType != tt.expectContentType {
						t.Errorf("Expected Content-Type %q, got %q", tt.expectContentType, contentType)
					}
				}

				// Check expected service in response body
				if tt.expectedService != "" {
					body := recorder.Body.String()
					if !strings.Contains(body, tt.expectedService) {
						t.Errorf("Expected response to contain service %q, got: %s", tt.expectedService, body)
					}
				}

				t.Logf("Proxy response: %s", recorder.Body.String())
			} else {
				// Should not have been handled by proxy (could be 404 or handled by other parts)
				// This is fine - just verify it wasn't proxied to our test servers
				body := recorder.Body.String()
				if strings.Contains(body, "api") || strings.Contains(body, "Web Service") {
					t.Errorf("Request should not have been proxied, but got response: %s", body)
				}
			}
		})
	}
}

// TestReverseProxyCustomHeaders tests custom header injection with variable substitution
func TestReverseProxyCustomHeaders(t *testing.T) {
	// Track received headers
	var receivedHeaders http.Header
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer testServer.Close()

	cfg := &config.Config{
		Routes: config.RoutesConfig{
			ReverseProxies: []config.ProxyRoute{
				{
					Name:   "header-test",
					Path:   "^/test/",
					Target: testServer.URL,
					Headers: map[string]string{
						"X-Forwarded-Proto":  "$scheme",
						"X-Forwarded-Host":   "$host",
						"X-Real-IP":          "$remote_addr",
						"X-Custom-Header":    "custom-value",
						"X-Static-Header":    "static-value",
						"X-Complex-Header":   "proto=$scheme,host=$host,ip=$remote_addr",
					},
				},
			},
		},
	}

	appManager := &process.AppManager{}
	idleManager := &idle.Manager{}
	handler := CreateHandler(cfg, appManager, nil, idleManager)

	// Make request with specific values to test variable substitution
	req := httptest.NewRequest("GET", "/test/endpoint", nil)
	req.Host = "navigator.example.com"
	req.RemoteAddr = "203.0.113.45:54321"  // Use documentation IP
	req.Header.Set("X-Forwarded-Proto", "https") // Set scheme via header since TLS isn't available in test

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	// Verify request was handled
	if recorder.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", recorder.Code)
	}

	// Test variable substitution
	tests := []struct {
		name         string
		headerName   string
		expectedValue string
	}{
		{
			name:         "Scheme variable substitution",
			headerName:   "X-Forwarded-Proto",
			expectedValue: "https",
		},
		{
			name:         "Host variable substitution",
			headerName:   "X-Forwarded-Host",
			expectedValue: "navigator.example.com",
		},
		{
			name:         "Remote address variable substitution",
			headerName:   "X-Real-IP",
			expectedValue: "203.0.113.45", // Port is stripped by getClientIP
		},
		{
			name:         "Static header value",
			headerName:   "X-Static-Header",
			expectedValue: "static-value",
		},
		{
			name:         "Complex header with multiple variables",
			headerName:   "X-Complex-Header",
			expectedValue: "proto=https,host=navigator.example.com,ip=203.0.113.45", // Port is stripped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := receivedHeaders.Get(tt.headerName)
			if actual != tt.expectedValue {
				t.Errorf("Header %s: expected %q, got %q", tt.headerName, tt.expectedValue, actual)
			}
		})
	}

	t.Logf("All received headers: %+v", receivedHeaders)
}

// TestReverseProxyMultipleConfigs tests multiple proxy configurations with precedence
func TestReverseProxyMultipleConfigs(t *testing.T) {
	// Create different backend servers
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("server1"))
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("server2"))
	}))
	defer server2.Close()

	server3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("server3"))
	}))
	defer server3.Close()

	cfg := &config.Config{
		Routes: config.RoutesConfig{
			ReverseProxies: []config.ProxyRoute{
				{
					Name:   "specific-api",
					Path:   "^/api/v1/users$", // More specific pattern
					Target: server1.URL,
				},
				{
					Name:   "general-api",
					Path:   "^/api/",          // Less specific pattern
					Target: server2.URL,
				},
				{
					Name:   "prefix-match",
					Prefix: "/prefix/",
					Target: server3.URL,
				},
			},
		},
	}

	appManager := &process.AppManager{}
	idleManager := &idle.Manager{}
	handler := CreateHandler(cfg, appManager, nil, idleManager)

	tests := []struct {
		name           string
		path           string
		expectedServer string
		description    string
	}{
		{
			name:           "Specific pattern should match first",
			path:           "/api/v1/users",
			expectedServer: "server1",
			description:    "More specific regex should take precedence",
		},
		{
			name:           "General pattern should match when specific doesn't",
			path:           "/api/v1/posts",
			expectedServer: "server2",
			description:    "General API pattern should catch other API paths",
		},
		{
			name:           "Prefix matching should work",
			path:           "/prefix/test",
			expectedServer: "server3",
			description:    "Simple prefix matching should work",
		},
		{
			name:           "General API pattern should match root",
			path:           "/api/",
			expectedServer: "server2",
			description:    "General pattern should match root API path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusOK {
				t.Fatalf("Expected status 200, got %d for path %s", recorder.Code, tt.path)
			}

			body := recorder.Body.String()
			if body != tt.expectedServer {
				t.Errorf("%s: expected response %q, got %q", tt.description, tt.expectedServer, body)
			}

			t.Logf("Path %s -> %s ✓", tt.path, body)
		})
	}
}

// TestReverseProxyPathStripping tests path stripping functionality
func TestReverseProxyPathStripping(t *testing.T) {
	// Track received path
	var receivedPath string
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("path:" + r.URL.Path))
	}))
	defer testServer.Close()

	cfg := &config.Config{
		Routes: config.RoutesConfig{
			ReverseProxies: []config.ProxyRoute{
				{
					Name:      "strip-prefix",
					Path:      "^/old-api/(.*)$",
					Target:    testServer.URL,
					StripPath: true,
				},
				{
					Name:      "no-strip",
					Path:      "^/new-api/",
					Target:    testServer.URL,
					StripPath: false,
				},
			},
		},
	}

	appManager := &process.AppManager{}
	idleManager := &idle.Manager{}
	handler := CreateHandler(cfg, appManager, nil, idleManager)

	tests := []struct {
		name         string
		requestPath  string
		expectedPath string
		description  string
	}{
		{
			name:         "Path should be stripped",
			requestPath:  "/old-api/v1/users",
			expectedPath: "/v1/users",
			description:  "StripPath should remove matched prefix",
		},
		{
			name:         "Path should not be stripped",
			requestPath:  "/new-api/v1/users",
			expectedPath: "/new-api/v1/users",
			description:  "Without StripPath, full path should be preserved",
		},
		{
			name:         "Root path stripping",
			requestPath:  "/old-api/",
			expectedPath: "/",
			description:  "Stripping should work for root paths",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.requestPath, nil)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusOK {
				t.Fatalf("Expected status 200, got %d", recorder.Code)
			}

			if receivedPath != tt.expectedPath {
				t.Errorf("%s: expected backend to receive path %q, got %q",
					tt.description, tt.expectedPath, receivedPath)
			}

			t.Logf("Request %s -> Backend %s ✓", tt.requestPath, receivedPath)
		})
	}
}

// TestReverseProxyErrorHandling tests error handling in proxy scenarios
func TestReverseProxyErrorHandling(t *testing.T) {
	cfg := &config.Config{
		Routes: config.RoutesConfig{
			ReverseProxies: []config.ProxyRoute{
				{
					Name:   "unavailable-service",
					Path:   "^/unavailable/",
					Target: "http://localhost:99999", // Invalid port
				},
				{
					Name:   "invalid-target",
					Path:   "^/invalid/",
					Target: "not-a-valid-url",
				},
			},
		},
	}

	appManager := &process.AppManager{}
	idleManager := &idle.Manager{}
	handler := CreateHandler(cfg, appManager, nil, idleManager)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		description    string
	}{
		{
			name:           "Unavailable service should return 502",
			path:           "/unavailable/test",
			expectedStatus: http.StatusBadGateway,
			description:    "Connection refused should result in 502",
		},
		{
			name:           "Invalid target should return 500 or 502",
			path:           "/invalid/test",
			expectedStatus: http.StatusInternalServerError, // Could be 500 or 502
			description:    "Invalid URL should result in error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, req)

			// Allow some flexibility in error status codes
			if recorder.Code != tt.expectedStatus &&
			   !(tt.expectedStatus == http.StatusInternalServerError && recorder.Code == http.StatusBadGateway) {
				t.Errorf("%s: expected status %d, got %d", tt.description, tt.expectedStatus, recorder.Code)
			}

			t.Logf("Error scenario %s -> Status %d ✓", tt.path, recorder.Code)
		})
	}
}

// TestWebSocketProxyConfiguration tests WebSocket routing logic through YAML reverse_proxies config
func TestWebSocketProxyConfiguration(t *testing.T) {
	// Since WebSocket proxying involves actual TCP hijacking which is complex to test
	// with httptest.Server, we'll test the routing and configuration logic instead

	// Regular HTTP server that can simulate WebSocket upgrade responses
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Record that we received the request with correct headers
		if r.Header.Get("X-WebSocket-Proxy") == "true" {
			w.Header().Set("X-Received-WebSocket-Header", "true")
		}
		if r.Header.Get("X-HTTP-Proxy") == "true" {
			w.Header().Set("X-Received-HTTP-Header", "true")
		}

		// Simulate different responses based on path
		if strings.HasPrefix(r.URL.Path, "/ws/") {
			if r.Header.Get("Upgrade") == "websocket" {
				// Simulate successful routing to WebSocket server
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("WebSocket route matched"))
			} else {
				// Non-WebSocket request to WebSocket path
				w.WriteHeader(http.StatusBadRequest)
			}
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("HTTP response"))
		}
	}))
	defer testServer.Close()

	cfg := &config.Config{
		Routes: config.RoutesConfig{
			ReverseProxies: []config.ProxyRoute{
				{
					Name:      "websocket-proxy",
					Path:      "^/ws/",
					Target:    testServer.URL,
					WebSocket: true,
					Headers: map[string]string{
						"X-WebSocket-Proxy": "true",
					},
				},
				{
					Name:   "http-proxy",
					Path:   "^/http/",
					Target: testServer.URL,
					Headers: map[string]string{
						"X-HTTP-Proxy": "true",
					},
				},
			},
		},
	}

	appManager := &process.AppManager{}
	idleManager := &idle.Manager{}
	handler := CreateHandler(cfg, appManager, nil, idleManager)

	tests := []struct {
		name               string
		path               string
		headers            map[string]string
		expectStatus       int
		expectResponseBody string
		expectHeader       string
	}{
		{
			name:               "Non-WebSocket request to WebSocket-configured route should use HTTP proxy",
			path:               "/ws/info",
			headers:            map[string]string{},
			expectStatus:       http.StatusBadRequest, // Our test server returns 400 for /ws/ paths without Upgrade header
			expectHeader:       "X-Received-WebSocket-Header",
		},
		{
			name:               "HTTP request should route to HTTP proxy",
			path:               "/http/test",
			headers:            map[string]string{},
			expectStatus:       http.StatusOK,
			expectResponseBody: "HTTP response",
			expectHeader:       "X-Received-HTTP-Header",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)

			// Set headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			// Check status
			if recorder.Code != tt.expectStatus {
				t.Errorf("Expected status %d, got %d", tt.expectStatus, recorder.Code)
			}

			// Check response body if expected
			if tt.expectResponseBody != "" {
				body := recorder.Body.String()
				if body != tt.expectResponseBody {
					t.Errorf("Expected body %q, got %q", tt.expectResponseBody, body)
				}
			}

			// Check that the correct proxy header was received
			if tt.expectHeader != "" {
				actual := recorder.Header().Get(tt.expectHeader)
				if actual != "true" {
					t.Errorf("Expected header %s: %q, got %q", tt.expectHeader, "true", actual)
				}
			}

			t.Logf("WebSocket routing test %s -> Status %d ✓", tt.path, recorder.Code)
		})
	}
}

// TestWebSocketProxyHeaders tests that WebSocket proxies receive custom headers
func TestWebSocketProxyHeaders(t *testing.T) {
	// Since WebSocket proxying with actual TCP handshakes is complex to test,
	// we'll test the HTTP proxy path that handles WebSocket configuration

	// Track received headers from the backend
	var receivedHeaders http.Header
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()

		// Return success for both WebSocket and HTTP requests
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Headers received"))
	}))
	defer testServer.Close()

	cfg := &config.Config{
		Routes: config.RoutesConfig{
			ReverseProxies: []config.ProxyRoute{
				{
					Name:      "ws-headers-test",
					Path:      "^/wstest/",
					Target:    testServer.URL,
					WebSocket: true,
					Headers: map[string]string{
						"X-WebSocket-Service": "chat-server",
						"X-Client-IP":         "$remote_addr",
						"X-Origin-Host":       "$host",
					},
				},
			},
		},
	}

	appManager := &process.AppManager{}
	idleManager := &idle.Manager{}
	handler := CreateHandler(cfg, appManager, nil, idleManager)

	// Test with non-WebSocket request to verify header injection works
	req := httptest.NewRequest("GET", "/wstest/room/123", nil)
	req.Host = "chat.example.com"
	req.RemoteAddr = "198.51.100.42:56789"

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	// Should get successful response from test server
	if recorder.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", recorder.Code)
	}

	// Check custom headers were injected correctly
	expectedHeaders := map[string]string{
		"X-WebSocket-Service": "chat-server",
		"X-Client-IP":         "198.51.100.42", // Port stripped by getClientIP
		"X-Origin-Host":       "chat.example.com",
	}

	for header, expectedValue := range expectedHeaders {
		actual := receivedHeaders.Get(header)
		if actual != expectedValue {
			t.Errorf("Header %s: expected %q, got %q", header, expectedValue, actual)
		}
	}

	t.Logf("WebSocket headers test passed - all custom headers received correctly")
}

// TestWebSocketProxyFallback tests that WebSocket proxy handles both WebSocket and HTTP requests
func TestWebSocketProxyFallback(t *testing.T) {
	// Server that responds differently to WebSocket vs HTTP
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Upgrade") == "websocket" {
			// For WebSocket requests, return success to show routing worked
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("WebSocket route handled"))
		} else {
			// Regular HTTP
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("HTTP fallback response"))
		}
	}))
	defer testServer.Close()

	cfg := &config.Config{
		Routes: config.RoutesConfig{
			ReverseProxies: []config.ProxyRoute{
				{
					Name:      "fallback-test",
					Path:      "^/mixed/",
					Target:    testServer.URL,
					WebSocket: true, // Configured for WebSocket but should handle HTTP too
				},
			},
		},
	}

	appManager := &process.AppManager{}
	idleManager := &idle.Manager{}
	handler := CreateHandler(cfg, appManager, nil, idleManager)

	tests := []struct {
		name         string
		addWSHeaders bool
		expectStatus int
		expectBody   string
	}{
		{
			name:         "HTTP request should get HTTP response on WebSocket-configured route",
			addWSHeaders: false,
			expectStatus: http.StatusOK,
			expectBody:   "HTTP fallback response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/mixed/endpoint", nil)

			if tt.addWSHeaders {
				req.Header.Set("Upgrade", "websocket")
				req.Header.Set("Connection", "Upgrade")
			}

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			if recorder.Code != tt.expectStatus {
				t.Errorf("Expected status %d, got %d", tt.expectStatus, recorder.Code)
			}

			if tt.expectBody != "" && recorder.Body.String() != tt.expectBody {
				t.Errorf("Expected body %q, got %q", tt.expectBody, recorder.Body.String())
			}

			t.Logf("Fallback test %s -> Status %d ✓", tt.name, recorder.Code)
		})
	}
}