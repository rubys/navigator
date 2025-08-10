package proxy

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"testing"

	"github.com/rubys/navigator/internal/config"
)

func TestRouterHealthCheck(t *testing.T) {
	router := &Router{}
	
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	
	router.healthCheck(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	if w.Body.String() != "OK" {
		t.Errorf("Expected body 'OK', got %s", w.Body.String())
	}
	
	contentType := w.Header().Get("Content-Type")
	if contentType != "text/plain" {
		t.Errorf("Expected Content-Type 'text/plain', got %s", contentType)
	}
}

func TestRouterHandleRequestRedirect(t *testing.T) {
	router := &Router{
		URLPrefix: "/showcase",
		Showcases: &config.Showcases{},
	}
	
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	
	router.handleRequest(w, req)
	
	if w.Code != http.StatusFound {
		t.Errorf("Expected status 302, got %d", w.Code)
	}
	
	location := w.Header().Get("Location")
	if location != "/studios/" {
		t.Errorf("Expected redirect to /studios/, got %s", location)
	}
}

func TestRouterHandleRequestNotFound(t *testing.T) {
	router := &Router{
		Showcases: &config.Showcases{},
	}
	
	req := httptest.NewRequest("GET", "/nonexistent/path", nil)
	w := httptest.NewRecorder()
	
	router.handleRequest(w, req)
	
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestRouterGetOrCreateProxy(t *testing.T) {
	router := &Router{
		proxies: make(map[string]*httputil.ReverseProxy),
	}
	
	// Test creating new proxy
	proxy1 := router.getOrCreateProxy("test-tenant", 4000)
	if proxy1 == nil {
		t.Error("Expected proxy to be created")
	}
	
	// Test getting existing proxy
	proxy2 := router.getOrCreateProxy("test-tenant", 4000)
	if proxy1 != proxy2 {
		t.Error("Expected to get same proxy instance")
	}
	
	// Test different tenant gets different proxy
	proxy3 := router.getOrCreateProxy("other-tenant", 4001)
	if proxy1 == proxy3 {
		t.Error("Expected different proxy for different tenant")
	}
	
	// Verify proxy configuration
	if len(router.proxies) != 2 {
		t.Errorf("Expected 2 proxies, got %d", len(router.proxies))
	}
}

func TestRouterCacheMiddleware(t *testing.T) {
	router := &Router{}
	
	// Test non-GET request
	req := httptest.NewRequest("POST", "/assets/app.css", nil)
	w := httptest.NewRecorder()
	
	handled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handled = true
		w.WriteHeader(http.StatusOK)
	})
	
	middleware := router.cacheMiddleware(next)
	middleware.ServeHTTP(w, req)
	
	if !handled {
		t.Error("Next handler should have been called for non-GET request")
	}
	
	// Test non-asset path
	req = httptest.NewRequest("GET", "/admin/users", nil)
	w = httptest.NewRecorder()
	handled = false
	
	middleware.ServeHTTP(w, req)
	
	if !handled {
		t.Error("Next handler should have been called for non-asset path")
	}
}

func TestRouterStructuredLogging(t *testing.T) {
	router := &Router{}
	
	// Create a test handler that the middleware wraps
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})
	
	// Create the middleware
	middleware := router.structuredLogRequest(testHandler)
	
	// Test request
	req := httptest.NewRequest("GET", "/test/path", nil)
	req.Header.Set("User-Agent", "test-agent")
	w := httptest.NewRecorder()
	
	// Execute request through middleware
	middleware.ServeHTTP(w, req)
	
	// Verify response was handled
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	if w.Body.String() != "test response" {
		t.Errorf("Expected 'test response', got %s", w.Body.String())
	}
}

// Benchmarks for performance testing
func BenchmarkRouterGetOrCreateProxy(b *testing.B) {
	router := &Router{
		proxies: make(map[string]*httputil.ReverseProxy),
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		label := fmt.Sprintf("tenant-%d", i%10) // Simulate 10 different tenants
		port := 4000 + (i % 10)
		router.getOrCreateProxy(label, port)
	}
}