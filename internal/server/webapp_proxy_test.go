package server

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rubys/navigator/internal/proxy"
)

// TestProxyRetryIntegration tests that proxy requests retry on gateway failures
// This test prevents regression of the bug where tenant app proxy didn't use retry logic
func TestProxyRetryIntegration(t *testing.T) {
	// Track number of attempts to backend
	var attempts int32

	// Create a backend that fails a few times then succeeds
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptNum := atomic.AddInt32(&attempts, 1)

		// Fail first 2 attempts with connection-like errors
		if attemptNum <= 2 {
			// Simulate connection refused by immediately closing connection
			hj, ok := w.(http.Hijacker)
			if ok {
				conn, _, _ := hj.Hijack()
				if conn != nil {
					conn.Close()
				}
			}
			return
		}

		// Succeed on 3rd attempt
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Success after retries"))
	}))
	defer backend.Close()

	// Make request through ProxyWithWebSocketSupport (what tenant apps should use)
	req := httptest.NewRequest("GET", "/test", nil)
	recorder := httptest.NewRecorder()

	// This should retry and eventually succeed
	proxy.ProxyWithWebSocketSupport(recorder, req, backend.URL, nil)

	// Verify retry happened
	finalAttempts := atomic.LoadInt32(&attempts)
	if finalAttempts < 2 {
		t.Errorf("Expected at least 2 retry attempts for ProxyWithWebSocketSupport, got %d", finalAttempts)
	}

	// Verify request eventually succeeded after retries
	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200 after retries, got %d", recorder.Code)
	}

	if recorder.Body.String() != "Success after retries" {
		t.Errorf("Expected 'Success after retries' in response body, got %q", recorder.Body.String())
	}

	t.Logf("Proxy retry test passed: %d attempts made", finalAttempts)
}

// TestProxyNoRetryForUnsafeMethods tests that unsafe methods (POST/PUT/DELETE) are not retried
func TestProxyNoRetryForUnsafeMethods(t *testing.T) {
	tests := []struct {
		method      string
		expectRetry bool
	}{
		{"GET", true},
		{"HEAD", true},
		{"POST", false},
		{"PUT", false},
		{"DELETE", false},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			var attempts int32

			// Create backend that always fails
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&attempts, 1)
				// Always fail with connection error
				hj, ok := w.(http.Hijacker)
				if ok {
					conn, _, _ := hj.Hijack()
					if conn != nil {
						conn.Close()
					}
				}
			}))
			defer backend.Close()

			// Make request
			req := httptest.NewRequest(tt.method, "/test", nil)
			recorder := httptest.NewRecorder()

			proxy.ProxyWithWebSocketSupport(recorder, req, backend.URL, nil)

			// Check attempt count
			finalAttempts := atomic.LoadInt32(&attempts)

			if tt.expectRetry {
				if finalAttempts < 2 {
					t.Errorf("Method %s should retry, but only got %d attempts", tt.method, finalAttempts)
				}
			} else {
				if finalAttempts > 1 {
					t.Errorf("Method %s should NOT retry, but got %d attempts", tt.method, finalAttempts)
				}
			}

			t.Logf("Method %s: %d attempts (retry=%v)", tt.method, finalAttempts, tt.expectRetry)
		})
	}
}

// TestProxyRetryTimeout tests that retry logic respects timeout duration
func TestProxyRetryTimeout(t *testing.T) {
	var attempts int32
	startTime := time.Now()

	// Create backend that always fails quickly
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

	// Make request
	req := httptest.NewRequest("GET", "/test", nil)
	recorder := httptest.NewRecorder()

	proxy.ProxyWithWebSocketSupport(recorder, req, backend.URL, nil)

	elapsed := time.Since(startTime)
	finalAttempts := atomic.LoadInt32(&attempts)

	// Should have multiple attempts
	if finalAttempts < 2 {
		t.Errorf("Expected multiple retry attempts, got %d", finalAttempts)
	}

	// Should respect timeout (default is 3 seconds for proxy retry)
	if elapsed > 5*time.Second {
		t.Errorf("Retry took too long: %v (expected < 5s)", elapsed)
	}

	// Should eventually give up with 502
	if recorder.Code != http.StatusBadGateway {
		t.Errorf("Expected 502 after retry timeout, got %d", recorder.Code)
	}

	t.Logf("Retry timeout test: %d attempts in %v", finalAttempts, elapsed)
}

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
