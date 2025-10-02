package process

import (
	"sync/atomic"
	"testing"

	"github.com/rubys/navigator/internal/config"
)

// TestWebAppWebSocketTracking tests that WebSocket connections are properly tracked
func TestWebAppWebSocketTracking(t *testing.T) {
	app := &WebApp{
		Tenant:    &config.Tenant{Name: "test-tenant"},
		readyChan: make(chan struct{}),
	}

	// Test initial count is zero
	if count := app.GetActiveWebSocketCount(); count != 0 {
		t.Errorf("Initial WebSocket count should be 0, got %d", count)
	}

	// Test incrementing connections
	wsPtr := app.GetActiveWebSocketsPtr()
	atomic.AddInt32(wsPtr, 1)

	if count := app.GetActiveWebSocketCount(); count != 1 {
		t.Errorf("WebSocket count should be 1 after increment, got %d", count)
	}

	// Test multiple connections
	atomic.AddInt32(wsPtr, 1)
	atomic.AddInt32(wsPtr, 1)

	if count := app.GetActiveWebSocketCount(); count != 3 {
		t.Errorf("WebSocket count should be 3 after adding 2 more, got %d", count)
	}

	// Test decrementing connections
	atomic.AddInt32(wsPtr, -1)

	if count := app.GetActiveWebSocketCount(); count != 2 {
		t.Errorf("WebSocket count should be 2 after decrement, got %d", count)
	}

	// Test decrementing to zero
	atomic.AddInt32(wsPtr, -2)

	if count := app.GetActiveWebSocketCount(); count != 0 {
		t.Errorf("WebSocket count should be 0 after decrementing all, got %d", count)
	}

	t.Logf("WebSocket tracking test passed with final count: %d", app.GetActiveWebSocketCount())
}

// TestWebAppWebSocketPointerConsistency tests that the pointer returned is stable
func TestWebAppWebSocketPointerConsistency(t *testing.T) {
	app := &WebApp{
		Tenant:    &config.Tenant{Name: "test-tenant"},
		readyChan: make(chan struct{}),
	}

	ptr1 := app.GetActiveWebSocketsPtr()
	ptr2 := app.GetActiveWebSocketsPtr()

	if ptr1 != ptr2 {
		t.Error("GetActiveWebSocketsPtr should return the same pointer")
	}

	// Verify modifications through either pointer work
	atomic.AddInt32(ptr1, 1)
	if *ptr2 != 1 {
		t.Error("Modifications through ptr1 should be visible through ptr2")
	}

	t.Logf("Pointer consistency test passed")
}

// TestWebAppWebSocketConcurrency tests concurrent WebSocket tracking
func TestWebAppWebSocketConcurrency(t *testing.T) {
	app := &WebApp{
		Tenant:    &config.Tenant{Name: "test-tenant"},
		readyChan: make(chan struct{}),
	}

	wsPtr := app.GetActiveWebSocketsPtr()

	// Simulate concurrent WebSocket connections
	done := make(chan bool)
	connections := 100

	// Add connections concurrently
	for i := 0; i < connections; i++ {
		go func() {
			atomic.AddInt32(wsPtr, 1)
			done <- true
		}()
	}

	// Wait for all additions
	for i := 0; i < connections; i++ {
		<-done
	}

	// Verify count
	if count := app.GetActiveWebSocketCount(); count != int32(connections) {
		t.Errorf("Expected %d connections, got %d", connections, count)
	}

	// Remove connections concurrently
	for i := 0; i < connections; i++ {
		go func() {
			atomic.AddInt32(wsPtr, -1)
			done <- true
		}()
	}

	// Wait for all removals
	for i := 0; i < connections; i++ {
		<-done
	}

	// Verify count is back to zero
	if count := app.GetActiveWebSocketCount(); count != 0 {
		t.Errorf("Expected 0 connections after removal, got %d", count)
	}

	t.Logf("Concurrency test passed with %d concurrent operations", connections*2)
}

// TestWebAppWebSocketIntegration tests integration with AppManager
func TestWebAppWebSocketIntegration(t *testing.T) {
	cfg := &config.Config{}
	cfg.Applications.Pools.Timeout = "5m"
	cfg.Applications.Tenants = []config.Tenant{
		{Name: "test-tenant"},
	}

	manager := NewAppManager(cfg)

	// Create an app manually to test WebSocket tracking
	app := &WebApp{
		Tenant:        &cfg.Applications.Tenants[0],
		Port:          4000,
		wsConnections: make(map[string]interface{}),
		readyChan:     make(chan struct{}),
	}

	manager.mutex.Lock()
	manager.apps["test-tenant"] = app
	manager.mutex.Unlock()

	// Test that we can track WebSockets on the app
	wsPtr := app.GetActiveWebSocketsPtr()
	if wsPtr == nil {
		t.Fatal("GetActiveWebSocketsPtr returned nil")
	}

	// Add some WebSocket connections
	atomic.AddInt32(wsPtr, 5)

	if count := app.GetActiveWebSocketCount(); count != 5 {
		t.Errorf("Expected 5 WebSocket connections, got %d", count)
	}

	// Retrieve app from manager and verify counter is accessible
	manager.mutex.RLock()
	retrievedApp, exists := manager.apps["test-tenant"]
	manager.mutex.RUnlock()

	if !exists {
		t.Fatal("App not found in manager")
	}

	if retrievedApp.GetActiveWebSocketCount() != 5 {
		t.Errorf("Retrieved app should have 5 WebSocket connections, got %d",
			retrievedApp.GetActiveWebSocketCount())
	}

	t.Logf("Integration test passed with %d active WebSockets", retrievedApp.GetActiveWebSocketCount())
}

// TestWebAppShouldTrackWebSockets tests the WebSocket tracking decision logic
func TestWebAppShouldTrackWebSockets(t *testing.T) {
	tests := []struct {
		name          string
		tenantSetting *bool
		globalSetting bool
		want          bool
	}{
		{
			name:          "tenant nil, global true",
			tenantSetting: nil,
			globalSetting: true,
			want:          true,
		},
		{
			name:          "tenant nil, global false",
			tenantSetting: nil,
			globalSetting: false,
			want:          false,
		},
		{
			name:          "tenant true, global false",
			tenantSetting: boolPtr(true),
			globalSetting: false,
			want:          true,
		},
		{
			name:          "tenant false, global true",
			tenantSetting: boolPtr(false),
			globalSetting: true,
			want:          false,
		},
		{
			name:          "tenant true, global true",
			tenantSetting: boolPtr(true),
			globalSetting: true,
			want:          true,
		},
		{
			name:          "tenant false, global false",
			tenantSetting: boolPtr(false),
			globalSetting: false,
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &WebApp{
				Tenant: &config.Tenant{
					Name:            "test-tenant",
					TrackWebSockets: tt.tenantSetting,
				},
				readyChan: make(chan struct{}),
			}

			got := app.ShouldTrackWebSockets(tt.globalSetting)
			if got != tt.want {
				t.Errorf("ShouldTrackWebSockets() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function for creating bool pointers
func boolPtr(b bool) *bool {
	return &b
}
