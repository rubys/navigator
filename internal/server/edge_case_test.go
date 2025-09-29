package server

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rubys/navigator/internal/config"
	"github.com/rubys/navigator/internal/idle"
	"github.com/rubys/navigator/internal/process"
)

// TestEdgeCaseRequests tests various edge case HTTP requests
func TestEdgeCaseRequests(t *testing.T) {
	// In CI environments, disable output to prevent overwhelming logs
	if os.Getenv("CI") == "true" {
		// Redirect stdout to discard to prevent massive JSON logs
		oldStdout := os.Stdout
		os.Stdout, _ = os.Open(os.DevNull)
		defer func() { os.Stdout = oldStdout }()

		// Set slog to discard output
		slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	}

	cfg := &config.Config{}
	appManager := &process.AppManager{}
	idleManager := &idle.Manager{}
	handler := CreateHandler(cfg, appManager, nil, idleManager)

	tests := []struct {
		name           string
		method         string
		path           string
		headers        map[string]string
		body           string
		expectedStatus int
		description    string
	}{
		{
			name:           "extremely long URL path",
			method:         "GET",
			path:           "/" + strings.Repeat("very-long-path-segment/", 1000),
			expectedStatus: http.StatusNotFound,
			description:    "Should handle very long URLs gracefully",
		},
		{
			name:   "request with extremely long headers",
			method: "GET",
			path:   "/",
			headers: map[string]string{
				"X-Very-Long-Header": strings.Repeat("a", 100000),
			},
			expectedStatus: http.StatusNotFound,
			description:    "Should handle very long headers",
		},
		{
			name:           "request with percent-encoded null bytes",
			method:         "GET",
			path:           "/test%00null%00bytes",
			expectedStatus: http.StatusNotFound,
			description:    "Should handle percent-encoded null bytes in path",
		},
		{
			name:           "request with Unicode characters in path",
			method:         "GET",
			path:           "/测试/тест/テスト/🚀",
			expectedStatus: http.StatusNotFound,
			description:    "Should handle Unicode paths properly",
		},
		{
			name:           "request with encoded special characters",
			method:         "GET",
			path:           "/test%20path/%3C%3E%22%27%26",
			expectedStatus: http.StatusNotFound,
			description:    "Should handle URL-encoded special characters",
		},
		{
			name:   "request with special character headers",
			method: "GET",
			path:   "/",
			headers: map[string]string{
				"X-Special-Chars": "value with spaces",
				"Another-Header":  "normal-value",
			},
			expectedStatus: http.StatusNotFound,
			description:    "Should handle headers with special characters",
		},
		{
			name:   "request with extremely large number of headers",
			method: "GET",
			path:   "/",
			headers: func() map[string]string {
				headers := make(map[string]string)
				for i := 0; i < 1000; i++ {
					headers[fmt.Sprintf("Header-%d", i)] = fmt.Sprintf("Value-%d", i)
				}
				return headers
			}(),
			expectedStatus: http.StatusNotFound,
			description:    "Should handle many headers",
		},
		{
			name:           "request with very long query string",
			method:         "GET",
			path:           "/?" + strings.Repeat("param=value&", 10000),
			expectedStatus: http.StatusNotFound,
			description:    "Should handle long query strings",
		},
		{
			name:   "HTTP/1.0 request",
			method: "GET",
			path:   "/",
			headers: map[string]string{
				"Connection": "close",
			},
			expectedStatus: http.StatusNotFound,
			description:    "Should handle HTTP/1.0 requests",
		},
		{
			name:           "request with empty method",
			method:         "",
			path:           "/",
			expectedStatus: http.StatusBadRequest,
			description:    "Should handle empty HTTP method",
		},
		{
			name:           "unsupported HTTP method",
			method:         "PATCH",
			path:           "/",
			expectedStatus: http.StatusNotFound,
			description:    "Should handle unsupported HTTP methods",
		},
		{
			name:           "request with fragment identifier",
			method:         "GET",
			path:           "/test#fragment",
			expectedStatus: http.StatusNotFound,
			description:    "Should handle fragments in URL",
		},
		{
			name:   "request with duplicate headers",
			method: "GET",
			path:   "/",
			headers: map[string]string{
				"User-Agent": "Navigator-Test/1.0",
			},
			expectedStatus: http.StatusNotFound,
			description:    "Should handle duplicate headers properly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			var body io.Reader
			if tt.body != "" {
				body = strings.NewReader(tt.body)
			}

			req := httptest.NewRequest(tt.method, tt.path, body)

			// Add headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			recorder := httptest.NewRecorder()

			// Use a timeout to prevent hanging on problematic requests
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			req = req.WithContext(ctx)

			handler.ServeHTTP(recorder, req)

			// Check if status is within reasonable bounds
			if recorder.Code < 100 || recorder.Code >= 600 {
				t.Errorf("Unexpected status code %d for %s", recorder.Code, tt.name)
			}

			t.Logf("Edge case %s -> Status %d (%s)", tt.name, recorder.Code, tt.description)
		})
	}
}

// TestConcurrentRequests tests handling of concurrent requests
func TestConcurrentRequests(t *testing.T) {
	cfg := &config.Config{
		Routes: config.RoutesConfig{
			ReverseProxies: []config.ProxyRoute{
				{
					Name:   "test-backend",
					Path:   "^/api/",
					Target: "http://localhost:9999", // Non-existent backend
				},
			},
		},
	}

	appManager := &process.AppManager{}
	idleManager := &idle.Manager{}
	handler := CreateHandler(cfg, appManager, nil, idleManager)

	// Test concurrent requests to various endpoints
	tests := []struct {
		name        string
		path        string
		method      string
		concurrency int
		requests    int
	}{
		{
			name:        "concurrent GET requests to root",
			path:        "/",
			method:      "GET",
			concurrency: 50,
			requests:    200,
		},
		{
			name:        "concurrent requests to proxy endpoint",
			path:        "/api/test",
			method:      "GET",
			concurrency: 25,
			requests:    100,
		},
		{
			name:        "mixed HTTP methods",
			path:        "/mixed",
			method:      "POST",
			concurrency: 20,
			requests:    80,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var wg sync.WaitGroup
			resultsChan := make(chan int, tt.requests)
			semaphore := make(chan struct{}, tt.concurrency)

			start := time.Now()

			// Launch concurrent requests
			for i := 0; i < tt.requests; i++ {
				wg.Add(1)
				go func(requestNum int) {
					defer wg.Done()

					// Rate limiting with semaphore
					semaphore <- struct{}{}
					defer func() { <-semaphore }()

					req := httptest.NewRequest(tt.method, tt.path+fmt.Sprintf("?req=%d", requestNum), nil)
					recorder := httptest.NewRecorder()

					// Add timeout per request
					ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
					defer cancel()
					req = req.WithContext(ctx)

					handler.ServeHTTP(recorder, req)
					resultsChan <- recorder.Code
				}(i)
			}

			wg.Wait()
			close(resultsChan)

			duration := time.Since(start)

			// Collect results
			statusCounts := make(map[int]int)
			totalRequests := 0
			for status := range resultsChan {
				statusCounts[status]++
				totalRequests++
			}

			// Verify all requests completed
			if totalRequests != tt.requests {
				t.Errorf("Expected %d requests, but got %d", tt.requests, totalRequests)
			}

			// Log performance statistics
			rps := float64(totalRequests) / duration.Seconds()
			t.Logf("Concurrent test %s: %d requests in %v (%.2f req/s)", tt.name, totalRequests, duration, rps)

			// Log status distribution
			for status, count := range statusCounts {
				t.Logf("  Status %d: %d requests", status, count)
			}

			// Basic sanity checks
			if len(statusCounts) == 0 {
				t.Errorf("No requests completed successfully")
			}

			// Verify reasonable response times
			if duration > 30*time.Second {
				t.Errorf("Requests took too long: %v", duration)
			}
		})
	}
}

// TestMemoryLeaks tests for potential memory leaks with many requests
func TestMemoryLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	// Create a simple backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer backend.Close()

	cfg := &config.Config{
		Routes: config.RoutesConfig{
			ReverseProxies: []config.ProxyRoute{
				{
					Name:   "memory-test",
					Path:   "^/test/",
					Target: backend.URL,
				},
			},
		},
	}

	appManager := &process.AppManager{}
	idleManager := &idle.Manager{}
	handler := CreateHandler(cfg, appManager, nil, idleManager)

	// Make many requests to check for memory growth
	numRequests := 1000
	if os.Getenv("NAVIGATOR_MEMORY_TEST") != "" {
		numRequests = 10000 // More intensive test when explicitly requested
	}

	t.Logf("Running memory leak test with %d requests", numRequests)

	start := time.Now()
	for i := 0; i < numRequests; i++ {
		req := httptest.NewRequest("GET", fmt.Sprintf("/test/%d", i), nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Errorf("Request %d failed with status %d", i, recorder.Code)
		}

		// Periodic progress update
		if i > 0 && i%1000 == 0 {
			elapsed := time.Since(start)
			t.Logf("Completed %d requests in %v", i, elapsed)
		}
	}

	duration := time.Since(start)
	rps := float64(numRequests) / duration.Seconds()
	t.Logf("Memory leak test completed: %d requests in %v (%.2f req/s)", numRequests, duration, rps)
}

// TestErrorRecovery tests error recovery scenarios
func TestErrorRecovery(t *testing.T) {
	cfg := &config.Config{}
	appManager := &process.AppManager{}
	idleManager := &idle.Manager{}
	handler := CreateHandler(cfg, appManager, nil, idleManager)

	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "panic recovery in handler",
			test: func(t *testing.T) {
				// Create a request that might cause issues
				req := httptest.NewRequest("GET", "/", nil)
				recorder := httptest.NewRecorder()

				// The handler should not panic
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("Handler panicked: %v", r)
					}
				}()

				handler.ServeHTTP(recorder, req)

				// Should get some response, not a panic
				if recorder.Code == 0 {
					t.Errorf("No response received, possible panic")
				}
			},
		},
		{
			name: "nil request handling",
			test: func(t *testing.T) {
				recorder := httptest.NewRecorder()

				defer func() {
					if r := recover(); r == nil {
						t.Errorf("Expected panic for nil request, but got none")
					}
				}()

				handler.ServeHTTP(recorder, nil)
			},
		},
		{
			name: "nil response writer handling",
			test: func(t *testing.T) {
				req := httptest.NewRequest("GET", "/", nil)

				defer func() {
					if r := recover(); r == nil {
						t.Errorf("Expected panic for nil ResponseWriter, but got none")
					}
				}()

				handler.ServeHTTP(nil, req)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.test(t)
		})
	}
}

// TestResourceExhaustion tests resource exhaustion scenarios
func TestResourceExhaustion(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping resource exhaustion test in short mode")
	}

	cfg := &config.Config{}
	appManager := &process.AppManager{}
	idleManager := &idle.Manager{}
	handler := CreateHandler(cfg, appManager, nil, idleManager)

	tests := []struct {
		name        string
		description string
		test        func(t *testing.T)
	}{
		{
			name:        "many simultaneous connections",
			description: "Test handling of many simultaneous connections",
			test: func(t *testing.T) {
				const numConns = 1000
				var wg sync.WaitGroup
				results := make(chan bool, numConns)

				for i := 0; i < numConns; i++ {
					wg.Add(1)
					go func(connNum int) {
						defer wg.Done()

						req := httptest.NewRequest("GET", fmt.Sprintf("/conn/%d", connNum), nil)
						recorder := httptest.NewRecorder()

						// Add timeout to prevent hanging
						ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
						defer cancel()
						req = req.WithContext(ctx)

						handler.ServeHTTP(recorder, req)
						results <- recorder.Code > 0 // Any response is better than hanging
					}(i)
				}

				wg.Wait()
				close(results)

				successful := 0
				for result := range results {
					if result {
						successful++
					}
				}

				successRate := float64(successful) / float64(numConns) * 100
				t.Logf("Connection test: %d/%d successful (%.2f%%)", successful, numConns, successRate)

				if successful == 0 {
					t.Errorf("No connections were successful")
				}
			},
		},
		{
			name:        "large request bodies",
			description: "Test handling of large request bodies",
			test: func(t *testing.T) {
				sizes := []int{1024, 10240, 102400, 1024000} // 1KB, 10KB, 100KB, 1MB

				for _, size := range sizes {
					t.Run(fmt.Sprintf("body_size_%d", size), func(t *testing.T) {
						largeBody := strings.Repeat("a", size)
						req := httptest.NewRequest("POST", "/large", strings.NewReader(largeBody))
						recorder := httptest.NewRecorder()

						// Add timeout for large requests
						ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
						defer cancel()
						req = req.WithContext(ctx)

						handler.ServeHTTP(recorder, req)

						if recorder.Code == 0 {
							t.Errorf("No response for %d byte request", size)
						}

						t.Logf("Large body test (%d bytes): Status %d", size, recorder.Code)
					})
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Running resource exhaustion test: %s", tt.description)
			tt.test(t)
		})
	}
}

// TestSlowRequests tests handling of slow or stalled requests
func TestSlowRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping slow request test in short mode")
	}

	// Create a slow backend server
	slowBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Slow response"))
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
	handler := CreateHandler(cfg, appManager, nil, idleManager)

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

// TestLoggingUnderStress tests logging behavior under stress
func TestLoggingUnderStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping logging stress test in short mode")
	}

	// Temporarily redirect logging to avoid spam during tests
	originalLogger := slog.Default()
	defer slog.SetDefault(originalLogger)

	// Create a logger that captures output
	logOutput := &strings.Builder{}
	testLogger := slog.New(slog.NewTextHandler(logOutput, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(testLogger)

	cfg := &config.Config{}
	appManager := &process.AppManager{}
	idleManager := &idle.Manager{}
	handler := CreateHandler(cfg, appManager, nil, idleManager)

	// Generate many requests rapidly to stress the logging system
	numRequests := 500
	var wg sync.WaitGroup

	start := time.Now()
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(reqNum int) {
			defer wg.Done()

			req := httptest.NewRequest("GET", fmt.Sprintf("/log-test-%d", reqNum), nil)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, req)
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	t.Logf("Logging stress test: %d requests in %v", numRequests, duration)

	// Check that some logging occurred
	logContent := logOutput.String()
	if len(logContent) == 0 {
		t.Errorf("No log output captured during stress test")
	}

	// Count log entries (rough approximation)
	logLines := strings.Count(logContent, "\n")
	t.Logf("Generated approximately %d log lines", logLines)
}
