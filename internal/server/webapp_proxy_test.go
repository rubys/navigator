package server

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/rubys/navigator/internal/proxy"
)

// TestProxyNoRetryIntegration tests that proxy requests don't retry on gateway failures
// Health checks ensure apps are ready before proxying, eliminating need for retries
func TestProxyNoRetryIntegration(t *testing.T) {
	// Track number of attempts to backend
	var attempts int32

	// Create a backend that always succeeds
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Success"))
	}))
	defer backend.Close()

	// Make request through ProxyWithWebSocketSupport (what tenant apps use)
	req := httptest.NewRequest("GET", "/test", nil)
	recorder := httptest.NewRecorder()

	// This should succeed immediately (no retries)
	proxy.ProxyWithWebSocketSupport(recorder, req, backend.URL, nil)

	// Verify only one attempt was made (no retries)
	finalAttempts := atomic.LoadInt32(&attempts)
	if finalAttempts != 1 {
		t.Errorf("Expected exactly 1 attempt (no retries), got %d", finalAttempts)
	}

	// Verify request succeeded
	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", recorder.Code)
	}

	if recorder.Body.String() != "Success" {
		t.Errorf("Expected 'Success' in response body, got %q", recorder.Body.String())
	}

	t.Logf("Proxy no-retry test passed: %d attempts made", finalAttempts)
}

// Note: TestProxyNoRetryForUnsafeMethods and TestProxyRetryTimeout moved to
// webapp_proxy_integration_test.go because they take 20s+ to run

// TestHandleProxyNoRetry tests that HandleProxy (without retry) fails immediately
// This documents the difference between HandleProxy and ProxyWithWebSocketSupport
func TestHandleProxyNoRetry(t *testing.T) {
	var attempts int32

	// Create backend that always fails
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		hj, ok := w.(http.Hijacker)
		if ok {
			conn, _, _ := hj.Hijack()
			if conn != nil {
				conn.Close()
			}
		}
	}))
	defer backend.Close()

	// Make request through HandleProxy (no retry)
	req := httptest.NewRequest("GET", "/test", nil)
	recorder := httptest.NewRecorder()

	proxy.HandleProxy(recorder, req, backend.URL)

	// Should only attempt once (no retry)
	finalAttempts := atomic.LoadInt32(&attempts)
	if finalAttempts > 1 {
		t.Errorf("HandleProxy should not retry, but got %d attempts", finalAttempts)
	}

	// Should fail immediately with 502
	if recorder.Code != http.StatusBadGateway {
		t.Errorf("Expected 502, got %d", recorder.Code)
	}

	t.Logf("HandleProxy (no retry): %d attempt", finalAttempts)
}
