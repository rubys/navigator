package proxy

import (
	"testing"
	"time"
)

// TestTrustProxyReload verifies that SetTrustProxy can be called multiple times
// to simulate configuration reload scenarios
func TestTrustProxyReload(t *testing.T) {
	// Simulate initial startup with trust_proxy: false (maintenance config)
	SetTrustProxy(false)
	if GetTrustProxy() != false {
		t.Error("Expected initial trust_proxy to be false")
	}

	// Simulate some requests being processed
	time.Sleep(10 * time.Millisecond)

	// Simulate config reload with trust_proxy: true (full config after ready hook)
	SetTrustProxy(true)
	if GetTrustProxy() != true {
		t.Error("Expected trust_proxy to be true after reload")
	}

	// Verify subsequent reads see the updated value
	time.Sleep(10 * time.Millisecond)
	if GetTrustProxy() != true {
		t.Error("Expected trust_proxy to remain true after reload")
	}

	// Simulate another reload back to false
	SetTrustProxy(false)
	if GetTrustProxy() != false {
		t.Error("Expected trust_proxy to be false after second reload")
	}
}

// TestTrustProxyConcurrentReload verifies thread-safety during config reload
func TestTrustProxyConcurrentReload(t *testing.T) {
	// Start with false
	SetTrustProxy(false)

	// Simulate concurrent reads while reload happens
	done := make(chan bool)

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			_ = GetTrustProxy() // Should never panic or race
			time.Sleep(1 * time.Millisecond)
		}
		done <- true
	}()

	// Reload goroutine (simulating SIGHUP or hook reload)
	go func() {
		for i := 0; i < 100; i++ {
			SetTrustProxy(i%2 == 0) // Toggle between true/false
			time.Sleep(1 * time.Millisecond)
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Test passes if no race condition or panic occurred
}

// TestDisableCompressionReload verifies that SetDisableCompression can be called multiple times
// to simulate configuration reload scenarios
func TestDisableCompressionReload(t *testing.T) {
	// Simulate initial startup with disable_compression: false (default)
	SetDisableCompression(false)
	if GetDisableCompression() != false {
		t.Error("Expected initial disable_compression to be false")
	}

	// Simulate some requests being processed
	time.Sleep(10 * time.Millisecond)

	// Simulate config reload with disable_compression: true
	SetDisableCompression(true)
	if GetDisableCompression() != true {
		t.Error("Expected disable_compression to be true after reload")
	}

	// Verify subsequent reads see the updated value
	time.Sleep(10 * time.Millisecond)
	if GetDisableCompression() != true {
		t.Error("Expected disable_compression to remain true after reload")
	}

	// Simulate another reload back to false
	SetDisableCompression(false)
	if GetDisableCompression() != false {
		t.Error("Expected disable_compression to be false after second reload")
	}
}

// TestDisableCompressionConcurrentReload verifies thread-safety during config reload
func TestDisableCompressionConcurrentReload(t *testing.T) {
	// Start with false
	SetDisableCompression(false)

	// Simulate concurrent reads while reload happens
	done := make(chan bool)

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			_ = GetDisableCompression() // Should never panic or race
			time.Sleep(1 * time.Millisecond)
		}
		done <- true
	}()

	// Reload goroutine (simulating SIGHUP or hook reload)
	go func() {
		for i := 0; i < 100; i++ {
			SetDisableCompression(i%2 == 0) // Toggle between true/false
			time.Sleep(1 * time.Millisecond)
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Test passes if no race condition or panic occurred
}
