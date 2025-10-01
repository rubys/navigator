//go:build integration
// +build integration

package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rubys/navigator/internal/config"
	"github.com/rubys/navigator/internal/idle"
	"github.com/rubys/navigator/internal/process"
	"github.com/rubys/navigator/internal/proxy"
)

// TestProxyNoRetryForUnsafeMethods tests that unsafe methods (POST/PUT/DELETE) are not retried
// This is an integration test because it waits for the full retry timeout (~10s per method)
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
// This is an integration test because it waits for the full 10s retry timeout
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

	// Should respect timeout (default is 10 seconds for proxy retry)
	if elapsed > 12*time.Second {
		t.Errorf("Retry took too long: %v (expected < 12s)", elapsed)
	}

	// Should eventually give up with 502
	if recorder.Code != http.StatusBadGateway {
		t.Errorf("Expected 502 after retry timeout, got %d", recorder.Code)
	}

	t.Logf("Retry timeout test: %d attempts in %v", finalAttempts, elapsed)
}

// TestSlowRequests tests handling of slow backend responses
// This is an integration test because it intentionally delays 2-4 seconds per test case
func TestSlowRequests(t *testing.T) {
	// Create a slow backend server
	slowBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Slow response"))
	}))
	defer slowBackend.Close()

	cfg := &config.Config{
		Routes: config.RoutesConfig{
			ReverseProxies: []config.ProxyRoute{
				{
					Name:   "slow-backend",
					Path:   "^/slow/",
					Target: slowBackend.URL,
				},
			},
		},
	}

	appManager := &process.AppManager{}
	idleManager := &idle.Manager{}
	handler := CreateTestHandler(cfg, appManager, nil, idleManager)

	tests := []struct {
		name        string
		path        string
		timeout     time.Duration
		expectError bool
	}{
		{
			name:        "slow request with sufficient timeout",
			path:        "/slow/test",
			timeout:     5 * time.Second,
			expectError: false,
		},
		{
			name:        "slow request with insufficient timeout",
			path:        "/slow/test",
			timeout:     500 * time.Millisecond,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			recorder := httptest.NewRecorder()

			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()
			req = req.WithContext(ctx)

			start := time.Now()
			handler.ServeHTTP(recorder, req)
			duration := time.Since(start)

			if tt.expectError {
				// Should timeout or return error status
				if duration >= tt.timeout && ctx.Err() == context.DeadlineExceeded {
					t.Logf("Request correctly timed out after %v", duration)
				} else if recorder.Code >= 500 {
					t.Logf("Request failed with status %d (expected for timeout)", recorder.Code)
				}
			} else {
				// Should complete successfully
				if recorder.Code == 0 {
					t.Errorf("No response received within timeout")
				}
				t.Logf("Slow request completed in %v with status %d", duration, recorder.Code)
			}
		})
	}
}
