package manager

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/rubys/navigator/internal/config"
)

func TestPortPool(t *testing.T) {
	t.Run("NewPortPool creates pool with correct ports", func(t *testing.T) {
		pool := NewPortPool(4000, 5)
		
		if pool.basePort != 4000 {
			t.Errorf("Expected basePort 4000, got %d", pool.basePort)
		}
		if pool.maxPorts != 5 {
			t.Errorf("Expected maxPorts 5, got %d", pool.maxPorts)
		}
		if len(pool.available) != 5 {
			t.Errorf("Expected 5 available ports, got %d", len(pool.available))
		}
		
		// Check port order
		for i, port := range pool.available {
			expected := 4000 + i
			if port != expected {
				t.Errorf("Expected port %d at index %d, got %d", expected, i, port)
			}
		}
	})

	t.Run("Get allocates ports sequentially", func(t *testing.T) {
		pool := NewPortPool(4000, 3)
		
		// Get first port
		port1, err := pool.Get()
		if err != nil {
			t.Fatalf("Unexpected error getting port: %v", err)
		}
		if port1 != 4000 {
			t.Errorf("Expected first port 4000, got %d", port1)
		}
		if len(pool.available) != 2 {
			t.Errorf("Expected 2 available ports after allocation, got %d", len(pool.available))
		}
		if !pool.inUse[4000] {
			t.Error("Port 4000 should be marked as in use")
		}

		// Get second port
		port2, err := pool.Get()
		if err != nil {
			t.Fatalf("Unexpected error getting port: %v", err)
		}
		if port2 != 4001 {
			t.Errorf("Expected second port 4001, got %d", port2)
		}
	})

	t.Run("Get returns error when no ports available", func(t *testing.T) {
		pool := NewPortPool(4000, 1)
		
		// Allocate the only port
		_, err := pool.Get()
		if err != nil {
			t.Fatalf("Unexpected error getting port: %v", err)
		}
		
		// Try to get another port
		_, err = pool.Get()
		if err == nil {
			t.Error("Expected error when no ports available")
		}
	})

	t.Run("Release returns port to pool", func(t *testing.T) {
		pool := NewPortPool(4000, 2)
		
		// Allocate port
		port, err := pool.Get()
		if err != nil {
			t.Fatalf("Unexpected error getting port: %v", err)
		}
		
		// Release port
		pool.Release(port)
		
		if pool.inUse[port] {
			t.Error("Port should not be marked as in use after release")
		}
		if len(pool.available) != 2 {
			t.Errorf("Expected 2 available ports after release, got %d", len(pool.available))
		}
	})

	t.Run("Release ignores ports not in use", func(t *testing.T) {
		pool := NewPortPool(4000, 2)
		
		// Release port that was never allocated
		pool.Release(5000)
		
		// Should still have all original ports
		if len(pool.available) != 2 {
			t.Errorf("Expected 2 available ports, got %d", len(pool.available))
		}
	})
}

func TestPumaManager_NewPumaManager(t *testing.T) {
	cfg := Config{
		RailsRoot:    "/tmp/test",
		MaxProcesses: 5,
		IdleTimeout:  time.Minute,
	}
	
	manager := NewPumaManager(cfg)
	
	if manager.Config.RailsRoot != cfg.RailsRoot {
		t.Errorf("Expected RailsRoot %s, got %s", cfg.RailsRoot, manager.Config.RailsRoot)
	}
	if manager.portPool.basePort != 4000 {
		t.Errorf("Expected basePort 4000, got %d", manager.portPool.basePort)
	}
	if manager.portPool.maxPorts != 5 {
		t.Errorf("Expected maxPorts 5, got %d", manager.portPool.maxPorts)
	}
	if len(manager.processes) != 0 {
		t.Errorf("Expected empty processes map, got %d processes", len(manager.processes))
	}
	
	// Clean up
	manager.StopAll()
}

func TestPumaProcess_Touch(t *testing.T) {
	process := &PumaProcess{
		LastUsed: time.Now().Add(-time.Hour),
	}
	
	oldTime := process.LastUsed
	time.Sleep(10 * time.Millisecond) // Ensure time difference
	
	process.Touch()
	
	if !process.LastUsed.After(oldTime) {
		t.Error("Touch should update LastUsed time")
	}
}

func TestPumaProcess_IsHealthy(t *testing.T) {
	t.Run("Returns false for nil command", func(t *testing.T) {
		process := &PumaProcess{}
		
		if process.IsHealthy() {
			t.Error("Process with nil Cmd should not be healthy")
		}
	})

	t.Run("Returns false for nil process", func(t *testing.T) {
		process := &PumaProcess{
			Cmd: &exec.Cmd{},
		}
		
		if process.IsHealthy() {
			t.Error("Process with nil Process should not be healthy")
		}
	})
}

func TestPumaManager_GetProcess(t *testing.T) {
	manager := NewPumaManager(Config{MaxProcesses: 5})
	defer manager.StopAll()
	
	// Test getting non-existent process
	process := manager.GetProcess("nonexistent")
	if process != nil {
		t.Error("Expected nil for non-existent process")
	}
	
	// Add a mock process
	tenant := &config.Tenant{Label: "test", Scope: "test"}
	mockProcess := &PumaProcess{
		Tenant:   tenant,
		Port:     4000,
		stopChan: make(chan struct{}),
	}
	manager.processes["test"] = mockProcess
	
	// Test getting existing process
	process = manager.GetProcess("test")
	if process != mockProcess {
		t.Error("Expected to get the mock process")
	}
}

func TestPumaManager_ListProcesses(t *testing.T) {
	manager := NewPumaManager(Config{MaxProcesses: 5})
	defer manager.StopAll()
	
	// Test empty list
	processes := manager.ListProcesses()
	if len(processes) != 0 {
		t.Errorf("Expected empty list, got %d processes", len(processes))
	}
	
	// Add mock processes
	tenant1 := &config.Tenant{Label: "test1", Scope: "test1"}
	tenant2 := &config.Tenant{Label: "test2", Scope: "test2"}
	process1 := &PumaProcess{Tenant: tenant1, Port: 4000, stopChan: make(chan struct{})}
	process2 := &PumaProcess{Tenant: tenant2, Port: 4001, stopChan: make(chan struct{})}
	
	manager.processes["test1"] = process1
	manager.processes["test2"] = process2
	
	// Test list with processes
	processes = manager.ListProcesses()
	if len(processes) != 2 {
		t.Errorf("Expected 2 processes, got %d", len(processes))
	}
	
	// Check that both processes are in the list
	found1, found2 := false, false
	for _, p := range processes {
		if p.Tenant.Label == "test1" {
			found1 = true
		}
		if p.Tenant.Label == "test2" {
			found2 = true
		}
	}
	if !found1 || !found2 {
		t.Error("Not all processes found in list")
	}
}

func TestPumaManager_Stop(t *testing.T) {
	manager := NewPumaManager(Config{MaxProcesses: 5})
	defer manager.StopAll()
	
	t.Run("Stop non-existent process returns nil", func(t *testing.T) {
		err := manager.Stop("nonexistent")
		if err != nil {
			t.Errorf("Expected nil error for non-existent process, got %v", err)
		}
	})
	
	t.Run("Stop removes process from manager", func(t *testing.T) {
		// Add mock process
		tenant := &config.Tenant{Label: "test", Scope: "test"}
		mockProcess := &PumaProcess{
			Tenant:   tenant,
			Port:     4000,
			stopChan: make(chan struct{}),
		}
		manager.processes["test"] = mockProcess
		manager.portPool.inUse[4000] = true
		
		// Stop process
		err := manager.Stop("test")
		if err != nil {
			t.Errorf("Unexpected error stopping process: %v", err)
		}
		
		// Verify process removed
		if manager.GetProcess("test") != nil {
			t.Error("Process should be removed from manager")
		}
		
		// Verify port released
		if manager.portPool.inUse[4000] {
			t.Error("Port should be released")
		}
	})
}

// TestPumaManagerIntegration tests manager functionality with mock processes
func TestPumaManagerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	// Create temporary directory for test Rails app
	tempDir, err := os.MkdirTemp("", "navigator-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create necessary directories
	os.MkdirAll(filepath.Join(tempDir, "tmp", "pids"), 0755)
	os.MkdirAll(filepath.Join(tempDir, "log"), 0755)
	
	cfg := Config{
		RailsRoot:    tempDir,
		MaxProcesses: 2,
		IdleTimeout:  100 * time.Millisecond, // Short timeout for testing
	}
	
	manager := NewPumaManager(cfg)
	defer manager.StopAll()
	
	// Test that manager starts with empty state
	if len(manager.ListProcesses()) != 0 {
		t.Error("Manager should start with no processes")
	}
	
	// Test port allocation sequence
	port1, err := manager.portPool.Get()
	if err != nil {
		t.Fatalf("Failed to get first port: %v", err)
	}
	port2, err := manager.portPool.Get()
	if err != nil {
		t.Fatalf("Failed to get second port: %v", err)
	}
	
	if port1 != 4000 || port2 != 4001 {
		t.Errorf("Expected ports 4000, 4001, got %d, %d", port1, port2)
	}
	
	// Release ports for cleanup
	manager.portPool.Release(port1)
	manager.portPool.Release(port2)
	
	// Test that we can't allocate more ports than maximum
	for i := 0; i < cfg.MaxProcesses; i++ {
		_, err := manager.portPool.Get()
		if err != nil {
			t.Fatalf("Should be able to allocate port %d", i)
		}
	}
	
	// Next allocation should fail
	_, err = manager.portPool.Get()
	if err == nil {
		t.Error("Should not be able to allocate more ports than maximum")
	}
}

// Benchmarks for performance testing
func BenchmarkPortPool_Get(b *testing.B) {
	pool := NewPortPool(4000, 1000)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i < 1000 {
			pool.Get()
		} else {
			// Release and get to avoid exhausting pool
			port := 4000 + (i % 1000)
			pool.Release(port)
			pool.Get()
		}
	}
}

func BenchmarkPumaManager_GetProcess(b *testing.B) {
	manager := NewPumaManager(Config{MaxProcesses: 100})
	defer manager.StopAll()
	
	// Add some processes
	for i := 0; i < 10; i++ {
		tenant := &config.Tenant{
			Label: fmt.Sprintf("tenant%d", i),
			Scope: fmt.Sprintf("test%d", i),
		}
		process := &PumaProcess{Tenant: tenant, Port: 4000 + i}
		manager.processes[tenant.Label] = process
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		label := fmt.Sprintf("tenant%d", i%10)
		manager.GetProcess(label)
	}
}