package idle

import (
	"testing"
	"time"

	"github.com/rubys/navigator/internal/config"
)

func TestNewManager(t *testing.T) {
	tests := []struct {
		name     string
		action   string
		timeout  string
		expected bool
	}{
		{"Valid suspend config", "suspend", "20m", true},
		{"Valid stop config", "stop", "30m", true},
		{"Empty action", "", "20m", false},
		{"Invalid timeout", "suspend", "invalid", true}, // Falls back to default timeout
		{"Empty timeout", "suspend", "", true},          // Falls back to default timeout
		{"Invalid action", "invalid", "20m", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.Server.Idle.Action = tt.action
			cfg.Server.Idle.Timeout = tt.timeout
			manager := NewManager(cfg)

			if tt.expected {
				if manager == nil {
					t.Error("Expected valid IdleManager, got nil")
				} else {
					if !manager.IsEnabled() {
						t.Error("Expected IdleManager to be enabled")
					}
				}
			} else {
				if manager != nil && manager.IsEnabled() {
					t.Error("Expected IdleManager to be disabled or nil")
				}
			}
		})
	}
}

func TestIdleManagerBasicOperations(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.Idle.Action = "suspend"
	cfg.Server.Idle.Timeout = "100ms"
	manager := NewManager(cfg)
	if manager == nil {
		t.Fatal("Failed to create IdleManager")
	}
	defer manager.Stop() // Clean up timers

	// Enable test mode to prevent signals
	manager.EnableTestMode()

	// Test enabled state
	if !manager.IsEnabled() {
		t.Error("IdleManager should be enabled")
	}

	// Test request tracking
	manager.RequestStarted()
	manager.RequestFinished()

	// Test multiple concurrent requests
	for i := 0; i < 5; i++ {
		manager.RequestStarted()
	}
	for i := 0; i < 5; i++ {
		manager.RequestFinished()
	}
}

func TestIdleManagerDisabled(t *testing.T) {
	// Test with empty action (should be disabled)
	cfg := &config.Config{}
	cfg.Server.Idle.Action = ""
	cfg.Server.Idle.Timeout = "20m"
	manager := NewManager(cfg)

	if manager != nil && manager.IsEnabled() {
		t.Error("IdleManager with empty action should be disabled")
	}

	// Test operations on disabled manager (should not panic)
	if manager != nil {
		manager.RequestStarted()
		manager.RequestFinished()
		manager.Stop()
	}
}

func TestIdleManagerTimeout(t *testing.T) {
	// Test very short timeout with test mode to avoid signals
	cfg := &config.Config{}
	cfg.Server.Idle.Action = "suspend"
	cfg.Server.Idle.Timeout = "10ms"
	manager := NewManager(cfg)
	if manager == nil {
		t.Fatal("Failed to create IdleManager")
	}
	defer manager.Stop()

	// Enable test mode to prevent actual signals
	manager.EnableTestMode()

	// Start and finish a request
	manager.RequestStarted()
	manager.RequestFinished()

	// Wait for potential idle timeout (this is hard to test precisely)
	time.Sleep(50 * time.Millisecond)

	// The test mainly verifies no panics occur and timers work
}

func TestIdleManagerConcurrency(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.Idle.Action = "suspend"
	cfg.Server.Idle.Timeout = "1s"
	manager := NewManager(cfg)
	if manager == nil {
		t.Fatal("Failed to create IdleManager")
	}
	defer manager.Stop()

	// Enable test mode to prevent signals
	manager.EnableTestMode()

	// Test concurrent request tracking
	done := make(chan bool, 10)

	// Start multiple goroutines
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				manager.RequestStarted()
				time.Sleep(time.Microsecond) // Tiny delay
				manager.RequestFinished()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestIdleManagerStop(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.Idle.Action = "suspend"
	cfg.Server.Idle.Timeout = "1s"
	manager := NewManager(cfg)
	if manager == nil {
		t.Fatal("Failed to create IdleManager")
	}

	// Enable test mode to prevent signals
	manager.EnableTestMode()

	// Test that Stop doesn't panic
	manager.Stop()

	// Test that operations after Stop don't panic
	manager.RequestStarted()
	manager.RequestFinished()
}

func TestIdleManagerActions(t *testing.T) {
	actions := []string{"suspend", "stop"}

	for _, action := range actions {
		t.Run("Action: "+action, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.Server.Idle.Action = action
			cfg.Server.Idle.Timeout = "1m"
			manager := NewManager(cfg)
			if manager == nil {
				t.Errorf("Failed to create IdleManager for action: %s", action)
			} else {
				manager.EnableTestMode() // Prevent signals during test
				if !manager.IsEnabled() {
					t.Errorf("IdleManager should be enabled for action: %s", action)
				}
				manager.Stop()
			}
		})
	}
}

func TestIdleManagerTimeoutParsing(t *testing.T) {
	validTimeouts := []string{
		"1s", "30s", "1m", "5m", "1h", "2h30m", "90m",
	}

	for _, timeout := range validTimeouts {
		t.Run("Timeout: "+timeout, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.Server.Idle.Action = "suspend"
			cfg.Server.Idle.Timeout = timeout
			manager := NewManager(cfg)
			if manager == nil {
				t.Errorf("Failed to create IdleManager for timeout: %s", timeout)
			} else {
				manager.Stop()
			}
		})
	}

	invalidTimeouts := []string{
		"", "invalid", "1x", "abc",
	}

	for _, timeout := range invalidTimeouts {
		t.Run("Invalid timeout: "+timeout, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.Server.Idle.Action = "suspend"
			cfg.Server.Idle.Timeout = timeout
			manager := NewManager(cfg)
			// Invalid timeouts fall back to default timeout, so manager should still be enabled
			if manager == nil || !manager.IsEnabled() {
				t.Errorf("IdleManager should be enabled even with invalid timeout (falls back to default): %s", timeout)
			} else {
				manager.Stop()
			}
		})
	}

	// Test negative timeout separately since it's parsed successfully but is unusual
	t.Run("Invalid timeout: -5m", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.Server.Idle.Action = "suspend"
		cfg.Server.Idle.Timeout = "-5m"
		manager := NewManager(cfg)
		// -5m is a valid duration but negative, so it should still enable
		if manager == nil || !manager.IsEnabled() {
			t.Error("IdleManager should be enabled even with negative timeout")
		} else {
			manager.Stop()
		}
	})
}

func TestIdleManagerLongRunning(t *testing.T) {
	// Test that long-running requests don't trigger idle
	cfg := &config.Config{}
	cfg.Server.Idle.Action = "suspend"
	cfg.Server.Idle.Timeout = "50ms"
	manager := NewManager(cfg)
	if manager == nil {
		t.Fatal("Failed to create IdleManager")
	}
	defer manager.Stop()

	// Enable test mode to prevent signals
	manager.EnableTestMode()

	// Start a request
	manager.RequestStarted()

	// Wait longer than timeout
	time.Sleep(100 * time.Millisecond)

	// Should still be active due to ongoing request
	// (This is more of a behavioral test)

	// Finish the request
	manager.RequestFinished()
}

func BenchmarkIdleManagerRequestTracking(b *testing.B) {
	cfg := &config.Config{}
	cfg.Server.Idle.Action = "suspend"
	cfg.Server.Idle.Timeout = "1m"
	manager := NewManager(cfg)
	if manager == nil {
		b.Fatal("Failed to create IdleManager")
	}
	defer manager.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.RequestStarted()
		manager.RequestFinished()
	}
}

func BenchmarkIdleManagerConcurrentTracking(b *testing.B) {
	cfg := &config.Config{}
	cfg.Server.Idle.Action = "suspend"
	cfg.Server.Idle.Timeout = "1m"
	manager := NewManager(cfg)
	if manager == nil {
		b.Fatal("Failed to create IdleManager")
	}
	defer manager.Stop()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			manager.RequestStarted()
			manager.RequestFinished()
		}
	})
}

// Tests for previously uncovered functions

func TestGetStats(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.Idle.Action = "suspend"
	cfg.Server.Idle.Timeout = "1m"
	manager := NewManager(cfg)
	if manager == nil {
		t.Fatal("Failed to create IdleManager")
	}
	defer manager.Stop()

	// Initially should have 0 active requests
	activeRequests, lastActivity := manager.GetStats()
	if activeRequests != 0 {
		t.Errorf("Expected 0 active requests, got %d", activeRequests)
	}
	if lastActivity.IsZero() {
		t.Error("Expected lastActivity to be set")
	}

	// Start some requests
	manager.RequestStarted()
	manager.RequestStarted()

	activeRequests, _ = manager.GetStats()
	if activeRequests != 2 {
		t.Errorf("Expected 2 active requests, got %d", activeRequests)
	}

	// Finish requests
	manager.RequestFinished()
	activeRequests, _ = manager.GetStats()
	if activeRequests != 1 {
		t.Errorf("Expected 1 active request, got %d", activeRequests)
	}
}

func TestUpdateConfig(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.Idle.Action = "suspend"
	cfg.Server.Idle.Timeout = "1m"
	manager := NewManager(cfg)
	if manager == nil {
		t.Fatal("Failed to create IdleManager")
	}
	defer manager.Stop()

	manager.EnableTestMode()

	// Update config with new timeout
	newCfg := &config.Config{}
	newCfg.Server.Idle.Action = "suspend"
	newCfg.Server.Idle.Timeout = "5m"

	manager.UpdateConfig(newCfg)

	// Verify manager is still enabled
	if !manager.IsEnabled() {
		t.Error("Expected manager to still be enabled after config update")
	}

	// Update config to disable
	disabledCfg := &config.Config{}
	disabledCfg.Server.Idle.Action = ""
	disabledCfg.Server.Idle.Timeout = "1m"

	manager.UpdateConfig(disabledCfg)

	// Verify manager is now disabled
	if manager.IsEnabled() {
		t.Error("Expected manager to be disabled after config update")
	}
}

func TestStopMachine(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.Idle.Action = "stop"
	cfg.Server.Idle.Timeout = "10ms"
	manager := NewManager(cfg)
	if manager == nil {
		t.Fatal("Failed to create IdleManager")
	}
	defer manager.Stop()

	// Enable test mode to prevent actual SIGTERM
	manager.EnableTestMode()

	// Manually invoke stopMachine (in test mode it won't send signals)
	manager.stopMachine()

	// Should not panic - test mode prevents actual signal
	// This mainly verifies the test mode protection works
}

func TestSuspend(t *testing.T) {
	t.Run("Suspend with valid config", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.Server.Idle.Action = "suspend"
		cfg.Server.Idle.Timeout = "1m"
		manager := NewManager(cfg)
		if manager == nil {
			t.Fatal("Failed to create IdleManager")
		}
		defer manager.Stop()

		manager.EnableTestMode()

		// Should succeed
		err := manager.Suspend()
		if err != nil {
			t.Errorf("Expected Suspend to succeed, got error: %v", err)
		}
	})

	t.Run("Suspend with stop action (should fail)", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.Server.Idle.Action = "stop"
		cfg.Server.Idle.Timeout = "1m"
		manager := NewManager(cfg)
		if manager == nil {
			t.Fatal("Failed to create IdleManager")
		}
		defer manager.Stop()

		manager.EnableTestMode()

		// Should fail because action is "stop" not "suspend"
		err := manager.Suspend()
		if err == nil {
			t.Error("Expected Suspend to fail with stop action")
		}
	})

	t.Run("Suspend when disabled", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.Server.Idle.Action = ""
		cfg.Server.Idle.Timeout = "1m"
		manager := NewManager(cfg)

		// Manager should be nil or disabled
		if manager != nil && manager.IsEnabled() {
			err := manager.Suspend()
			if err == nil {
				t.Error("Expected Suspend to fail when disabled")
			}
		}
	})
}

func TestRequestStartedWithTimer(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.Idle.Action = "suspend"
	cfg.Server.Idle.Timeout = "100ms"
	manager := NewManager(cfg)
	if manager == nil {
		t.Fatal("Failed to create IdleManager")
	}
	defer manager.Stop()

	manager.EnableTestMode()

	// Start a request - should cancel any idle timer
	manager.RequestStarted()

	// Verify active requests increased
	activeRequests, _ := manager.GetStats()
	if activeRequests != 1 {
		t.Errorf("Expected 1 active request, got %d", activeRequests)
	}

	// Start another request while first is still active
	manager.RequestStarted()

	activeRequests, _ = manager.GetStats()
	if activeRequests != 2 {
		t.Errorf("Expected 2 active requests, got %d", activeRequests)
	}

	// Finish both requests
	manager.RequestFinished()
	manager.RequestFinished()

	activeRequests, _ = manager.GetStats()
	if activeRequests != 0 {
		t.Errorf("Expected 0 active requests, got %d", activeRequests)
	}
}
