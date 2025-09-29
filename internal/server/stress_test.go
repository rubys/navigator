//go:build integration || stress
// +build integration stress

package server

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rubys/navigator/internal/config"
	"github.com/rubys/navigator/internal/idle"
	"github.com/rubys/navigator/internal/process"
)

// TestHighLoadScenarios tests various high-load scenarios
func TestHighLoadScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress tests in short mode")
	}

	// Create multiple backend servers to simulate real load balancing
	backends := make([]*httptest.Server, 3)
	for i := range backends {
		backendID := i
		backends[i] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simulate varying response times
			delay := time.Duration(rand.Intn(100)) * time.Millisecond
			time.Sleep(delay)

			w.Header().Set("X-Backend-ID", fmt.Sprintf("backend-%d", backendID))
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf("Response from backend %d", backendID)))
		}))
	}
	defer func() {
		for _, backend := range backends {
			backend.Close()
		}
	}()

	cfg := &config.Config{
		Routes: config.RoutesConfig{
			ReverseProxies: []config.ProxyRoute{
				{
					Name:   "backend-0",
					Path:   "^/api/v1/",
					Target: backends[0].URL,
					Headers: map[string]string{
						"X-Load-Test": "true",
					},
				},
				{
					Name:   "backend-1",
					Path:   "^/api/v2/",
					Target: backends[1].URL,
					Headers: map[string]string{
						"X-Load-Test": "true",
					},
				},
				{
					Name:      "backend-2",
					Path:      "^/websocket/",
					Target:    backends[2].URL,
					WebSocket: true,
				},
			},
		},
	}

	appManager := &process.AppManager{}
	idleManager := &idle.Manager{}
	handler := CreateHandler(cfg, appManager, nil, idleManager)

	scenarios := []struct {
		name        string
		duration    time.Duration
		concurrency int
		requestRate int // requests per second per goroutine
		paths       []string
		description string
	}{
		{
			name:        "sustained_load",
			duration:    10 * time.Second,
			concurrency: 50,
			requestRate: 10,
			paths:       []string{"/api/v1/users", "/api/v2/posts", "/static/file.txt"},
			description: "Sustained load with moderate concurrency",
		},
		{
			name:        "burst_load",
			duration:    5 * time.Second,
			concurrency: 200,
			requestRate: 50,
			paths:       []string{"/api/v1/data", "/api/v2/metrics"},
			description: "High burst load with many concurrent connections",
		},
		{
			name:        "mixed_workload",
			duration:    15 * time.Second,
			concurrency: 30,
			requestRate: 15,
			paths: []string{
				"/api/v1/users", "/api/v1/posts", "/api/v1/comments",
				"/api/v2/analytics", "/api/v2/reports", "/api/v2/dashboard",
				"/static/app.js", "/static/styles.css", "/static/logo.png",
				"/websocket/chat", "/websocket/notifications",
			},
			description: "Mixed workload with various endpoint types",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Logf("Running stress scenario: %s", scenario.description)

			var (
				totalRequests int64
				successCount  int64
				errorCount    int64
				totalLatency  int64
				maxLatency    int64
				minLatency    int64 = 999999999 // Initialize to high value
			)

			var wg sync.WaitGroup
			startTime := time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), scenario.duration+5*time.Second)
			defer cancel()

			// Launch worker goroutines
			for i := 0; i < scenario.concurrency; i++ {
				wg.Add(1)
				go func(workerID int) {
					defer wg.Done()

					ticker := time.NewTicker(time.Second / time.Duration(scenario.requestRate))
					defer ticker.Stop()

					for {
						select {
						case <-ctx.Done():
							return
						case <-ticker.C:
							if time.Since(startTime) >= scenario.duration {
								return
							}

							// Choose random path
							path := scenario.paths[rand.Intn(len(scenario.paths))]

							// Make request
							reqStart := time.Now()
							req := httptest.NewRequest("GET", path, nil)
							req.Header.Set("X-Worker-ID", fmt.Sprintf("%d", workerID))
							recorder := httptest.NewRecorder()

							handler.ServeHTTP(recorder, req)

							reqDuration := time.Since(reqStart)
							latencyNs := reqDuration.Nanoseconds()

							// Update statistics atomically
							atomic.AddInt64(&totalRequests, 1)
							if recorder.Code >= 200 && recorder.Code < 400 {
								atomic.AddInt64(&successCount, 1)
							} else {
								atomic.AddInt64(&errorCount, 1)
							}

							atomic.AddInt64(&totalLatency, latencyNs)

							// Update min latency
							for {
								current := atomic.LoadInt64(&minLatency)
								if latencyNs >= current || atomic.CompareAndSwapInt64(&minLatency, current, latencyNs) {
									break
								}
							}

							// Update max latency
							for {
								current := atomic.LoadInt64(&maxLatency)
								if latencyNs <= current || atomic.CompareAndSwapInt64(&maxLatency, current, latencyNs) {
									break
								}
							}
						}
					}
				}(i)
			}

			wg.Wait()
			actualDuration := time.Since(startTime)

			// Calculate statistics
			total := atomic.LoadInt64(&totalRequests)
			success := atomic.LoadInt64(&successCount)
			errors := atomic.LoadInt64(&errorCount)
			avgLatency := time.Duration(atomic.LoadInt64(&totalLatency) / max(total, 1))
			minLat := time.Duration(atomic.LoadInt64(&minLatency))
			maxLat := time.Duration(atomic.LoadInt64(&maxLatency))

			successRate := float64(success) / float64(total) * 100
			rps := float64(total) / actualDuration.Seconds()

			// Log comprehensive results
			t.Logf("Scenario: %s", scenario.name)
			t.Logf("  Duration: %v (requested: %v)", actualDuration, scenario.duration)
			t.Logf("  Total Requests: %d", total)
			t.Logf("  Success Rate: %.2f%% (%d/%d)", successRate, success, total)
			t.Logf("  Error Count: %d", errors)
			t.Logf("  Requests/Second: %.2f", rps)
			t.Logf("  Latency - Avg: %v, Min: %v, Max: %v", avgLatency, minLat, maxLat)

			// Basic performance assertions
			if total == 0 {
				t.Errorf("No requests were made")
			}
			if successRate < 80 {
				t.Errorf("Success rate too low: %.2f%% (expected >= 80%%)", successRate)
			}
			if avgLatency > 5*time.Second {
				t.Errorf("Average latency too high: %v", avgLatency)
			}
			if time.Duration(maxLatency) > 30*time.Second {
				t.Errorf("Max latency too high: %v", time.Duration(maxLatency))
			}
		})
	}
}

// TestConcurrencyLimits tests the system's behavior at concurrency limits
func TestConcurrencyLimits(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency limits test in short mode")
	}

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate some work
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer backend.Close()

	cfg := &config.Config{
		Routes: config.RoutesConfig{
			ReverseProxies: []config.ProxyRoute{
				{
					Name:   "concurrency-test",
					Path:   "^/test/",
					Target: backend.URL,
				},
			},
		},
	}

	appManager := &process.AppManager{}
	idleManager := &idle.Manager{}
	handler := CreateHandler(cfg, appManager, nil, idleManager)

	// Test different concurrency levels
	concurrencyLevels := []int{10, 50, 100, 500, 1000}

	for _, concurrency := range concurrencyLevels {
		t.Run(fmt.Sprintf("concurrency_%d", concurrency), func(t *testing.T) {
			var wg sync.WaitGroup
			var successful, failed int64

			start := time.Now()

			for i := 0; i < concurrency; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()

					req := httptest.NewRequest("GET", fmt.Sprintf("/test/%d", id), nil)
					recorder := httptest.NewRecorder()

					// Add timeout per request
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()
					req = req.WithContext(ctx)

					handler.ServeHTTP(recorder, req)

					if recorder.Code == http.StatusOK {
						atomic.AddInt64(&successful, 1)
					} else {
						atomic.AddInt64(&failed, 1)
					}
				}(i)
			}

			wg.Wait()
			duration := time.Since(start)

			successCount := atomic.LoadInt64(&successful)
			failureCount := atomic.LoadInt64(&failed)
			successRate := float64(successCount) / float64(concurrency) * 100

			t.Logf("Concurrency %d: %d success, %d failed (%.1f%% success) in %v",
				concurrency, successCount, failureCount, successRate, duration)

			// Allow some flexibility for high concurrency
			minSuccessRate := 95.0
			if concurrency >= 500 {
				minSuccessRate = 90.0 // Allow more failures at very high concurrency
			}

			if successRate < minSuccessRate {
				t.Errorf("Success rate too low at concurrency %d: %.1f%% (expected >= %.1f%%)",
					concurrency, successRate, minSuccessRate)
			}
		})
	}
}

// TestMemoryUsageUnderLoad tests memory usage patterns under load
func TestMemoryUsageUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory usage test in short mode")
	}

	// Force garbage collection before test
	runtime.GC()
	runtime.GC()

	var initialStats, midStats, finalStats runtime.MemStats
	runtime.ReadMemStats(&initialStats)

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create some temporary data to test memory management
		data := make([]byte, 1024) // 1KB per request
		for i := range data {
			data[i] = byte(rand.Intn(256))
		}

		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}))
	defer backend.Close()

	cfg := &config.Config{
		Routes: config.RoutesConfig{
			ReverseProxies: []config.ProxyRoute{
				{
					Name:   "memory-test",
					Path:   "^/memory/",
					Target: backend.URL,
				},
			},
		},
	}

	appManager := &process.AppManager{}
	idleManager := &idle.Manager{}
	handler := CreateHandler(cfg, appManager, nil, idleManager)

	// Phase 1: Light load
	const lightRequests = 1000
	for i := 0; i < lightRequests; i++ {
		req := httptest.NewRequest("GET", fmt.Sprintf("/memory/test-%d", i), nil)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
	}

	runtime.GC()
	runtime.ReadMemStats(&midStats)

	// Phase 2: Heavy load
	const heavyRequests = 5000
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 50) // Limit concurrency

	for i := 0; i < heavyRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			req := httptest.NewRequest("GET", fmt.Sprintf("/memory/heavy-%d", id), nil)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)
		}(i)
	}

	wg.Wait()

	runtime.GC()
	runtime.GC() // Run GC twice to ensure cleanup
	runtime.ReadMemStats(&finalStats)

	// Calculate memory usage changes
	initialMB := float64(initialStats.Alloc) / 1024 / 1024
	midMB := float64(midStats.Alloc) / 1024 / 1024
	finalMB := float64(finalStats.Alloc) / 1024 / 1024

	t.Logf("Memory Usage:")
	t.Logf("  Initial: %.2f MB", initialMB)
	t.Logf("  After light load (%d requests): %.2f MB (+%.2f MB)", lightRequests, midMB, midMB-initialMB)
	t.Logf("  After heavy load (%d requests): %.2f MB (+%.2f MB)", heavyRequests, finalMB, finalMB-initialMB)
	t.Logf("  Total allocations: %d", finalStats.TotalAlloc-initialStats.TotalAlloc)
	t.Logf("  GC runs: %d", finalStats.NumGC-initialStats.NumGC)

	// Check for potential memory leaks
	memoryGrowth := finalMB - initialMB
	if memoryGrowth > 100 { // Allow up to 100MB growth
		t.Errorf("Potential memory leak detected: memory grew by %.2f MB", memoryGrowth)
	}

	// Check that memory usage after heavy load isn't dramatically higher than after light load
	heavyLoadGrowth := finalMB - midMB
	if heavyLoadGrowth > 50 { // Allow up to 50MB additional growth for heavy load
		t.Errorf("Excessive memory growth during heavy load: %.2f MB", heavyLoadGrowth)
	}
}

// TestResourceLeaks tests for various types of resource leaks
func TestResourceLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping resource leak test in short mode")
	}

	// Test goroutine leaks
	initialGoroutines := runtime.NumGoroutine()

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer backend.Close()

	cfg := &config.Config{
		Routes: config.RoutesConfig{
			ReverseProxies: []config.ProxyRoute{
				{
					Name:   "leak-test",
					Path:   "^/leak/",
					Target: backend.URL,
				},
			},
		},
	}

	appManager := &process.AppManager{}
	idleManager := &idle.Manager{}
	handler := CreateHandler(cfg, appManager, nil, idleManager)

	// Make many requests
	const numRequests = 2000
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 100)

	t.Logf("Testing resource leaks with %d requests", numRequests)

	start := time.Now()
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			req := httptest.NewRequest("GET", fmt.Sprintf("/leak/test-%d", id), nil)
			recorder := httptest.NewRecorder()

			// Add timeout to prevent hanging goroutines
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			req = req.WithContext(ctx)

			handler.ServeHTTP(recorder, req)
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	// Allow time for cleanup
	time.Sleep(2 * time.Second)
	runtime.GC()
	time.Sleep(1 * time.Second)

	finalGoroutines := runtime.NumGoroutine()

	t.Logf("Resource leak test completed in %v", duration)
	t.Logf("Goroutines: initial=%d, final=%d, difference=%d",
		initialGoroutines, finalGoroutines, finalGoroutines-initialGoroutines)

	// Allow some tolerance for goroutine count differences
	goroutineDifference := finalGoroutines - initialGoroutines
	if goroutineDifference > 10 {
		t.Errorf("Potential goroutine leak: %d extra goroutines", goroutineDifference)
	}
}

// TestFailureRecovery tests system recovery from various failure scenarios
func TestFailureRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping failure recovery test in short mode")
	}

	scenarios := []struct {
		name        string
		setupServer func() *httptest.Server
		description string
	}{
		{
			name: "backend_timeout",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Simulate timeout by sleeping longer than typical timeouts
					time.Sleep(10 * time.Second)
					w.WriteHeader(http.StatusOK)
				}))
			},
			description: "Backend server that times out",
		},
		{
			name: "backend_random_failures",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Randomly fail 30% of requests
					if rand.Float32() < 0.3 {
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("Success"))
				}))
			},
			description: "Backend with random failures",
		},
		{
			name: "backend_slow_response",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Simulate slow response
					delay := time.Duration(rand.Intn(2000)) * time.Millisecond
					time.Sleep(delay)
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("Slow response"))
				}))
			},
			description: "Slow backend responses",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			backend := scenario.setupServer()
			defer backend.Close()

			cfg := &config.Config{
				Routes: config.RoutesConfig{
					ReverseProxies: []config.ProxyRoute{
						{
							Name:   "recovery-test",
							Path:   "^/recovery/",
							Target: backend.URL,
						},
					},
				},
			}

			appManager := &process.AppManager{}
			idleManager := &idle.Manager{}
			handler := CreateHandler(cfg, appManager, nil, idleManager)

			t.Logf("Testing failure recovery: %s", scenario.description)

			var successful, failed, timeout int64
			const testRequests = 100
			var wg sync.WaitGroup

			start := time.Now()
			for i := 0; i < testRequests; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()

					req := httptest.NewRequest("GET", fmt.Sprintf("/recovery/test-%d", id), nil)
					recorder := httptest.NewRecorder()

					// Set reasonable timeout
					ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
					defer cancel()
					req = req.WithContext(ctx)

					handler.ServeHTTP(recorder, req)

					if ctx.Err() == context.DeadlineExceeded {
						atomic.AddInt64(&timeout, 1)
					} else if recorder.Code >= 200 && recorder.Code < 400 {
						atomic.AddInt64(&successful, 1)
					} else {
						atomic.AddInt64(&failed, 1)
					}
				}(i)
			}

			wg.Wait()
			duration := time.Since(start)

			successCount := atomic.LoadInt64(&successful)
			failureCount := atomic.LoadInt64(&failed)
			timeoutCount := atomic.LoadInt64(&timeout)

			t.Logf("Recovery test %s results:", scenario.name)
			t.Logf("  Duration: %v", duration)
			t.Logf("  Successful: %d", successCount)
			t.Logf("  Failed: %d", failureCount)
			t.Logf("  Timeout: %d", timeoutCount)

			// System should handle failures gracefully without crashing
			total := successCount + failureCount + timeoutCount
			if total != testRequests {
				t.Errorf("Request count mismatch: expected %d, got %d", testRequests, total)
			}

			// For timeout scenario, expect mostly timeouts
			if scenario.name == "backend_timeout" && timeoutCount < testRequests/2 {
				t.Errorf("Expected more timeouts for timeout scenario: got %d", timeoutCount)
			}
		})
	}
}

// max returns the maximum of two int64 values
func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
