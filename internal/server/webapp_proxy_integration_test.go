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

// TestProxyNoRetryAnyMethod tests that NO methods are retried for tenant apps
// Health checks ensure apps are ready before proxying, eliminating need for retries
func TestProxyNoRetryAnyMethod(t *testing.T) {
	methods := []string{"GET", "HEAD", "POST", "PUT", "DELETE"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
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
			req := httptest.NewRequest(method, "/test", nil)
			recorder := httptest.NewRecorder()

			proxy.ProxyWithWebSocketSupport(recorder, req, backend.URL, nil)

			// Check attempt count - should be exactly 1 (no retries)
			finalAttempts := atomic.LoadInt32(&attempts)

			if finalAttempts != 1 {
				t.Errorf("Method %s should NOT retry, but got %d attempts", method, finalAttempts)
			}

			// Should fail immediately with 502
			if recorder.Code != http.StatusBadGateway {
				t.Errorf("Expected 502, got %d", recorder.Code)
			}

			t.Logf("Method %s: %d attempt (no retry)", method, finalAttempts)
		})
	}
}

// TestProxyFailsImmediately tests that proxy failures return immediately without retry
// Health checks ensure tenant apps are ready, so no retry logic needed
func TestProxyFailsImmediately(t *testing.T) {
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

	// Should only make one attempt (no retry)
	if finalAttempts != 1 {
		t.Errorf("Expected exactly 1 attempt (no retry), got %d", finalAttempts)
	}

	// Should fail quickly (no retry delays)
	if elapsed > 1*time.Second {
		t.Errorf("Proxy should fail immediately, took %v", elapsed)
	}

	// Should fail with 502
	if recorder.Code != http.StatusBadGateway {
		t.Errorf("Expected 502, got %d", recorder.Code)
	}

	t.Logf("Proxy failed immediately: %d attempt in %v", finalAttempts, elapsed)
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
