//go:build integration
// +build integration

package proxy

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rubys/navigator/internal/config"
)

// TestRetryLogicEndToEnd tests the complete retry logic with timing verification
func TestRetryLogicEndToEnd(t *testing.T) {
	// Track request attempts and timing
	var requestCount int
	var requestTimes []time.Time
	var mu sync.Mutex

	// Create a test server that returns 502 errors for the first few requests
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		requestTimes = append(requestTimes, time.Now())
		count := requestCount
		mu.Unlock()

		// Return 502 for first 3 requests, then succeed
		if count <= 3 {
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte("Bad Gateway"))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Success"))
		}
	}))
	defer testServer.Close()

	// Create a request to proxy
	req := httptest.NewRequest("GET", "/test", nil)
	recorder := httptest.NewRecorder()

	startTime := time.Now()

	// Test the retry logic
	HandleProxyWithRetry(recorder, req, testServer.URL, config.ProxyRetryTimeout)

	endTime := time.Now()
	totalDuration := endTime.Sub(startTime)

	mu.Lock()
	finalCount := requestCount
	times := make([]time.Time, len(requestTimes))
	copy(times, requestTimes)
	mu.Unlock()

	// Verify the request succeeded after retries
	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", recorder.Code)
	}

	if recorder.Body.String() != "Success" {
		t.Errorf("Expected 'Success', got %q", recorder.Body.String())
	}

	// Verify we made the expected number of requests (3 failures + 1 success)
	if finalCount != 4 {
		t.Errorf("Expected 4 requests, got %d", finalCount)
	}

	// Verify total duration is reasonable (should be around 700ms for 3 retries)
	// 100ms + 200ms + 400ms = 700ms + processing time
	minExpected := 600 * time.Millisecond
	maxExpected := 1200 * time.Millisecond
	if totalDuration < minExpected || totalDuration > maxExpected {
		t.Errorf("Expected total duration between %v and %v, got %v",
			minExpected, maxExpected, totalDuration)
	}

	t.Logf("Total requests: %d, Total duration: %v", finalCount, totalDuration)
}

// TestRetryExponentialBackoff tests the exponential backoff timing
func TestRetryExponentialBackoff(t *testing.T) {
	var requestTimes []time.Time
	var mu sync.Mutex

	// Create a server that always returns 502
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestTimes = append(requestTimes, time.Now())
		mu.Unlock()

		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("Always fails"))
	}))
	defer testServer.Close()

	req := httptest.NewRequest("GET", "/test", nil)
	recorder := httptest.NewRecorder()

	// Test with short timeout to ensure we get multiple retries
	HandleProxyWithRetry(recorder, req, testServer.URL, 2*time.Second)

	mu.Lock()
	times := make([]time.Time, len(requestTimes))
	copy(times, requestTimes)
	mu.Unlock()

	// Should have at least 2 requests (initial + 1 retry)
	if len(times) < 2 {
		t.Fatalf("Expected at least 2 requests, got %d", len(times))
	}

	// Verify exponential backoff timing between requests
	expectedDelays := []time.Duration{
		config.ProxyRetryInitialDelay,     // First retry after 100ms
		config.ProxyRetryInitialDelay * 2, // Second retry after 200ms
		config.ProxyRetryInitialDelay * 4, // Third retry after 400ms
		config.ProxyRetryMaxDelay,         // Fourth retry after 500ms (capped)
	}

	for i := 1; i < len(times) && i <= len(expectedDelays); i++ {
		actualDelay := times[i].Sub(times[i-1])
		expectedDelay := expectedDelays[i-1]

		// Allow 20% tolerance for timing variations
		minDelay := time.Duration(float64(expectedDelay) * 0.8)
		maxDelay := time.Duration(float64(expectedDelay) * 1.3)

		if actualDelay < minDelay || actualDelay > maxDelay {
			t.Errorf("Retry %d: expected delay between %v and %v, got %v",
				i, minDelay, maxDelay, actualDelay)
		}

		t.Logf("Retry %d: delay %v (expected ~%v)", i, actualDelay, expectedDelay)
	}
}

// TestRetryOnSpecific502Errors tests that we only retry on 502 errors
func TestRetryOnSpecific502Errors(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		expectRetry    bool
		expectedResult int
	}{
		{
			name:           "502 Bad Gateway should retry",
			statusCode:     502,
			expectRetry:    true,
			expectedResult: 502, // Will eventually give up and return 502
		},
		{
			name:           "500 Internal Server Error should not retry",
			statusCode:     500,
			expectRetry:    false,
			expectedResult: 500,
		},
		{
			name:           "404 Not Found should not retry",
			statusCode:     404,
			expectRetry:    false,
			expectedResult: 404,
		},
		{
			name:           "200 OK should not retry",
			statusCode:     200,
			expectRetry:    false,
			expectedResult: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requestCount int
			var mu sync.Mutex

			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				requestCount++
				mu.Unlock()

				w.WriteHeader(tt.statusCode)
				w.Write([]byte(fmt.Sprintf("Status: %d", tt.statusCode)))
			}))
			defer testServer.Close()

			req := httptest.NewRequest("GET", "/test", nil)
			recorder := httptest.NewRecorder()

			startTime := time.Now()
			HandleProxyWithRetry(recorder, req, testServer.URL, 1*time.Second)
			duration := time.Since(startTime)

			mu.Lock()
			finalCount := requestCount
			mu.Unlock()

			if recorder.Code != tt.expectedResult {
				t.Errorf("Expected status %d, got %d", tt.expectedResult, recorder.Code)
			}

			if tt.expectRetry {
				// Should have made multiple requests
				if finalCount <= 1 {
					t.Errorf("Expected multiple requests for retry, got %d", finalCount)
				}
				// Should have taken some time (at least one retry delay)
				if duration < 80*time.Millisecond {
					t.Errorf("Expected retry delay, but completed in %v", duration)
				}
			} else {
				// Should have made only one request
				if finalCount != 1 {
					t.Errorf("Expected exactly 1 request (no retry), got %d", finalCount)
				}
				// Should have completed quickly (no retry delay)
				if duration > 100*time.Millisecond {
					t.Errorf("Expected quick completion, but took %v", duration)
				}
			}

			t.Logf("Status %d: %d requests in %v", tt.statusCode, finalCount, duration)
		})
	}
}

// TestRetryTimeoutBehavior tests that retries respect the timeout limit
func TestRetryTimeoutBehavior(t *testing.T) {
	var requestCount int
	var mu sync.Mutex

	// Server that always returns 502
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		mu.Unlock()

		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("Always fails"))
	}))
	defer testServer.Close()

	req := httptest.NewRequest("GET", "/test", nil)
	recorder := httptest.NewRecorder()

	// Use a short timeout to test timeout behavior
	timeout := 300 * time.Millisecond
	startTime := time.Now()

	HandleProxyWithRetry(recorder, req, testServer.URL, timeout)

	duration := time.Since(startTime)

	mu.Lock()
	finalCount := requestCount
	mu.Unlock()

	// Should have stopped retrying due to timeout
	if recorder.Code != http.StatusBadGateway {
		t.Errorf("Expected final status 502, got %d", recorder.Code)
	}

	// Duration should be close to timeout (allowing some overhead)
	maxExpectedDuration := timeout + 100*time.Millisecond
	if duration > maxExpectedDuration {
		t.Errorf("Expected duration <= %v, got %v", maxExpectedDuration, duration)
	}

	// Should have made at least 2 requests (initial + at least 1 retry)
	if finalCount < 2 {
		t.Errorf("Expected at least 2 requests, got %d", finalCount)
	}

	t.Logf("Timeout test: %d requests in %v (timeout: %v)", finalCount, duration, timeout)
}

// TestRetryLargeRequestBypass tests that large requests bypass retry logic
func TestRetryLargeRequestBypass(t *testing.T) {
	var requestCount int
	var mu sync.Mutex

	// Server that always returns 502
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		mu.Unlock()

		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("Bad Gateway"))
	}))
	defer testServer.Close()

	// Create a request with large body (>1MB)
	largeBody := strings.NewReader(strings.Repeat("x", 1024*1024+1))
	req := httptest.NewRequest("POST", "/test", largeBody)
	req.ContentLength = 1024*1024 + 1

	recorder := httptest.NewRecorder()

	startTime := time.Now()
	HandleProxyWithRetry(recorder, req, testServer.URL, 2*time.Second)
	duration := time.Since(startTime)

	mu.Lock()
	finalCount := requestCount
	mu.Unlock()

	// Large requests should bypass retry, so only 1 request
	if finalCount != 1 {
		t.Errorf("Expected exactly 1 request (no retry for large body), got %d", finalCount)
	}

	// Should complete quickly without retry delays
	if duration > 100*time.Millisecond {
		t.Errorf("Expected quick completion for large request, took %v", duration)
	}

	// Should still return the 502 error
	if recorder.Code != http.StatusBadGateway {
		t.Errorf("Expected status 502, got %d", recorder.Code)
	}

	t.Logf("Large request test: %d requests in %v", finalCount, duration)
}

// TestRetryWithGETRequests tests that GET requests are safely retried
func TestRetryWithGETRequests(t *testing.T) {
	var requestCount int
	var mu sync.Mutex

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		count := requestCount
		mu.Unlock()

		// Succeed on second attempt
		if count == 1 {
			w.WriteHeader(http.StatusBadGateway)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Success after retry"))
		}
	}))
	defer testServer.Close()

	req := httptest.NewRequest("GET", "/test", nil)
	recorder := httptest.NewRecorder()

	HandleProxyWithRetry(recorder, req, testServer.URL, 2*time.Second)

	mu.Lock()
	finalCount := requestCount
	mu.Unlock()

	// Should have retried and succeeded
	if finalCount != 2 {
		t.Errorf("Expected 2 requests (1 failure + 1 retry), got %d", finalCount)
	}

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", recorder.Code)
	}

	if !strings.Contains(recorder.Body.String(), "Success") {
		t.Errorf("Expected success message, got %q", recorder.Body.String())
	}
}