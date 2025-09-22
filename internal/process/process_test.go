package process

import (
	"sync"
	"testing"
	"time"

	"github.com/rubys/navigator/internal/config"
)

func TestNewManager(t *testing.T) {
	cfg := &config.Config{}
	manager := NewManager(cfg)

	if manager == nil {
		t.Fatal("NewManager should return a non-nil manager")
	}

	if manager.config != cfg {
		t.Error("Manager should store the provided config")
	}

	if len(manager.processes) != 0 {
		t.Error("New manager should start with empty process list")
	}
}

func TestManagedProcessBasics(t *testing.T) {
	// Test basic ManagedProcess creation
	mp := &ManagedProcess{
		Name:        "test-process",
		Command:     "echo",
		Args:        []string{"hello", "world"},
		WorkingDir:  "/tmp",
		Env:         map[string]string{"TEST": "value"},
		AutoRestart: true,
		StartDelay:  time.Second,
		Running:     false,
	}

	// Test basic fields
	if mp.Name != "test-process" {
		t.Errorf("Name = %q, expected %q", mp.Name, "test-process")
	}

	if mp.Command != "echo" {
		t.Errorf("Command = %q, expected %q", mp.Command, "echo")
	}

	if mp.Running {
		t.Error("New ManagedProcess should not be running")
	}
}

func TestExecuteHooks(t *testing.T) {
	tests := []struct {
		name        string
		hooks       []config.HookConfig
		env         map[string]string
		hookType    string
		expectError bool
	}{
		{
			name:        "Empty hooks",
			hooks:       []config.HookConfig{},
			env:         map[string]string{},
			hookType:    "test",
			expectError: false,
		},
		{
			name: "Simple successful hook",
			hooks: []config.HookConfig{
				{
					Command: "echo",
					Args:    []string{"test"},
					Timeout: "5s",
				},
			},
			env:         map[string]string{"TEST": "value"},
			hookType:    "test",
			expectError: false,
		},
		{
			name: "Hook with no command",
			hooks: []config.HookConfig{
				{
					Command: "",
					Args:    []string{"test"},
				},
			},
			env:         map[string]string{},
			hookType:    "test",
			expectError: false,
		},
		{
			name: "Hook with invalid timeout",
			hooks: []config.HookConfig{
				{
					Command: "echo",
					Args:    []string{"test"},
					Timeout: "invalid",
				},
			},
			env:         map[string]string{},
			hookType:    "test",
			expectError: false, // Should still succeed with no timeout
		},
		{
			name: "Hook with non-existent command",
			hooks: []config.HookConfig{
				{
					Command: "non-existent-command-12345",
					Args:    []string{},
					Timeout: "1s",
				},
			},
			env:         map[string]string{},
			hookType:    "test",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ExecuteHooks(tt.hooks, tt.env, tt.hookType)
			if (err != nil) != tt.expectError {
				t.Errorf("ExecuteHooks() error = %v, expectError = %v", err, tt.expectError)
			}
		})
	}
}

func TestExecuteServerHooks(t *testing.T) {
	tests := []struct {
		name        string
		hooks       []config.HookConfig
		hookType    string
		expectError bool
	}{
		{
			name:        "Empty server hooks",
			hooks:       []config.HookConfig{},
			hookType:    "start",
			expectError: false,
		},
		{
			name: "Valid server hook",
			hooks: []config.HookConfig{
				{
					Command: "echo",
					Args:    []string{"server", "starting"},
					Timeout: "2s",
				},
			},
			hookType:    "start",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ExecuteServerHooks(tt.hooks, tt.hookType)
			if (err != nil) != tt.expectError {
				t.Errorf("ExecuteServerHooks() error = %v, expectError = %v", err, tt.expectError)
			}
		})
	}
}

func TestManagerStartStopCycle(t *testing.T) {
	// Test starting and stopping processes
	cfg := createTestConfig()
	manager := NewManager(cfg)

	// Test starting processes (should handle empty config gracefully)
	err := manager.StartManagedProcesses()
	if err != nil {
		t.Errorf("StartManagedProcesses should not error: %v", err)
	}

	// Test stopping processes
	manager.StopManagedProcesses()
	// Should not panic or error
}

func TestManagerUpdateConfig(t *testing.T) {
	cfg := createTestConfig()
	manager := NewManager(cfg)

	// Test updating with new config
	newCfg := &config.Config{
		ManagedProcesses: []config.ManagedProcessConfig{
			{
				Name:        "test-updated",
				Command:     "echo",
				Args:        []string{"updated"},
				WorkingDir:  "/tmp",
				AutoRestart: false,
				StartDelay:  "2s",
			},
		},
	}

	// Should not panic
	manager.UpdateManagedProcesses(newCfg)
}

func TestExecuteTenantHooks(t *testing.T) {
	defaultHooks := []config.HookConfig{
		{
			Command: "echo",
			Args:    []string{"default", "hook"},
			Timeout: "2s",
		},
	}

	specificHooks := []config.HookConfig{
		{
			Command: "echo",
			Args:    []string{"specific", "hook"},
			Timeout: "2s",
		},
	}

	env := map[string]string{"TENANT": "test-tenant"}

	err := ExecuteTenantHooks(defaultHooks, specificHooks, env, "test-tenant", "start")
	if err != nil {
		t.Errorf("ExecuteTenantHooks should not error: %v", err)
	}
}

func TestCreateLogWriter(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		stream    string
		logConfig config.LogConfig
	}{
		{
			name:      "Text format",
			source:    "test-app",
			stream:    "stdout",
			logConfig: config.LogConfig{Format: "text"},
		},
		{
			name:      "JSON format",
			source:    "test-app",
			stream:    "stderr",
			logConfig: config.LogConfig{Format: "json"},
		},
		{
			name:      "Default format",
			source:    "test-app",
			stream:    "stdout",
			logConfig: config.LogConfig{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			writer := CreateLogWriter(tt.source, tt.stream, tt.logConfig)
			if writer == nil {
				t.Error("CreateLogWriter should return a non-nil writer")
			}

			// Test writing to it
			_, err := writer.Write([]byte("test log message\n"))
			if err != nil {
				t.Errorf("Writer should not error on write: %v", err)
			}
		})
	}
}

func BenchmarkExecuteHooks(b *testing.B) {
	hooks := []config.HookConfig{
		{
			Command: "echo",
			Args:    []string{"benchmark", "test"},
			Timeout: "1s",
		},
	}
	env := map[string]string{"BENCH": "true"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExecuteHooks(hooks, env, "benchmark")
	}
}

func TestWebAppManager(t *testing.T) {
	cfg := &config.Config{}
	cfg.Applications.Pools.MaxSize = 5
	cfg.Applications.Pools.StartPort = 4000

	appManager := NewAppManager(cfg)
	if appManager == nil {
		t.Fatal("NewAppManager should return a non-nil manager")
	}

	// Test getting a non-existent app
	_, exists := appManager.GetApp("non-existent")
	if exists {
		t.Error("GetApp should return false for non-existent app")
	}

	// Test cleanup
	appManager.Cleanup()
}

func TestManagedProcessLifecycle(t *testing.T) {
	cfg := &config.Config{
		ManagedProcesses: []config.ManagedProcessConfig{
			{
				Name:        "test-echo",
				Command:     "echo",
				Args:        []string{"hello", "world"},
				WorkingDir:  "/tmp",
				Env:         map[string]string{"TEST": "value"},
				AutoRestart: false,
				StartDelay:  "100ms",
			},
		},
	}

	manager := NewManager(cfg)

	// Test starting processes
	err := manager.StartManagedProcesses()
	if err != nil {
		t.Fatalf("StartManagedProcesses failed: %v", err)
	}

	// Give processes time to start
	time.Sleep(200 * time.Millisecond)

	// Verify process was created
	if len(manager.processes) != 1 {
		t.Errorf("Expected 1 process, got %d", len(manager.processes))
	}

	proc := manager.processes[0]
	if proc.Name != "test-echo" {
		t.Errorf("Process name = %q, expected %q", proc.Name, "test-echo")
	}

	// Test stopping processes
	manager.StopManagedProcesses()

	// Give processes time to stop
	time.Sleep(100 * time.Millisecond)
}

func TestManagedProcessAutoRestart(t *testing.T) {
	// Create a process that will fail quickly for testing auto-restart
	cfg := &config.Config{
		ManagedProcesses: []config.ManagedProcessConfig{
			{
				Name:        "test-failing",
				Command:     "false", // Command that exits with error
				Args:        []string{},
				AutoRestart: true,
				StartDelay:  "0s",
			},
		},
	}

	manager := NewManager(cfg)

	err := manager.StartManagedProcesses()
	if err != nil {
		t.Fatalf("StartManagedProcesses failed: %v", err)
	}

	// Give time for process to fail and restart attempt
	time.Sleep(1 * time.Second)

	// Verify process exists (restart logic should be working)
	if len(manager.processes) != 1 {
		t.Errorf("Expected 1 process after restart, got %d", len(manager.processes))
	}

	// Stop the auto-restart cycle quickly to avoid long test times
	manager.StopManagedProcesses()
}

func TestManagedProcessInvalidCommand(t *testing.T) {
	cfg := &config.Config{
		ManagedProcesses: []config.ManagedProcessConfig{
			{
				Name:    "test-invalid",
				Command: "non-existent-command-12345",
				Args:    []string{},
			},
		},
	}

	manager := NewManager(cfg)

	// Should not return error even if process fails to start
	err := manager.StartManagedProcesses()
	if err != nil {
		t.Fatalf("StartManagedProcesses should not fail for invalid command: %v", err)
	}

	// Give time for start attempt
	time.Sleep(100 * time.Millisecond)

	manager.StopManagedProcesses()
}

func TestWebAppManagerIntegration(t *testing.T) {
	cfg := &config.Config{
		Applications: config.Applications{
			Pools: config.Pools{
				MaxSize:   5,
				StartPort: 4000,
				Timeout:   "5m",
			},
			Tenants: []config.Tenant{
				{
					Name: "test-tenant",
					Root: "/tmp",
					Var:  map[string]interface{}{"database": "test_db"},
				},
			},
			Env: map[string]string{
				"DATABASE_URL": "sqlite:///${database}.db",
				"PORT":         "${port}",
			},
		},
	}

	appManager := NewAppManager(cfg)

	// Test getting non-existent tenant
	_, err := appManager.GetOrStartApp("non-existent-tenant")
	if err == nil {
		t.Error("Expected error for non-existent tenant")
	}

	// Test basic app manager functionality without actually starting processes
	// (since we don't have a real web app to start in tests)

	// Test port allocation (use the module function)
	port, err := findAvailablePort(appManager.minPort, appManager.maxPort)
	if err != nil {
		t.Fatalf("findAvailablePort failed: %v", err)
	}
	if port < appManager.minPort || port > appManager.maxPort {
		t.Errorf("Port %d is outside expected range [%d, %d]", port, appManager.minPort, appManager.maxPort)
	}

	// Test cleanup
	appManager.Cleanup()
}

func TestUpdateManagedProcessesIntegration(t *testing.T) {
	// Initial config with one process
	cfg := &config.Config{
		ManagedProcesses: []config.ManagedProcessConfig{
			{
				Name:    "initial-process",
				Command: "echo",
				Args:    []string{"initial"},
			},
		},
	}

	manager := NewManager(cfg)
	manager.StartManagedProcesses()
	time.Sleep(100 * time.Millisecond)

	if len(manager.processes) != 1 {
		t.Errorf("Expected 1 initial process, got %d", len(manager.processes))
	}

	// Updated config with different processes
	newCfg := &config.Config{
		ManagedProcesses: []config.ManagedProcessConfig{
			{
				Name:    "new-process",
				Command: "echo",
				Args:    []string{"new"},
			},
			{
				Name:    "another-process",
				Command: "echo",
				Args:    []string{"another"},
			},
		},
	}

	// Test configuration update
	manager.UpdateManagedProcesses(newCfg)
	time.Sleep(200 * time.Millisecond)

	if len(manager.processes) != 3 {
		t.Errorf("Expected 3 processes after update (1 stopped + 2 new), got %d", len(manager.processes))
	}

	manager.StopManagedProcesses()
}

func TestProcessManagerConcurrency(t *testing.T) {
	cfg := &config.Config{
		ManagedProcesses: []config.ManagedProcessConfig{
			{
				Name:    "concurrent-test",
				Command: "echo",
				Args:    []string{"concurrent"},
			},
		},
	}

	manager := NewManager(cfg)

	// Test concurrent access
	var wg sync.WaitGroup

	// Start multiple goroutines accessing the manager
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			manager.StartManagedProcesses()
		}()
	}

	wg.Wait()

	// Concurrent calls might create multiple processes, but should eventually stabilize
	// This tests that the manager doesn't crash under concurrent access
	if len(manager.processes) == 0 {
		t.Error("Expected at least some processes to be created")
	}

	manager.StopManagedProcesses()
}

func TestWebAppPortAllocation(t *testing.T) {
	// Test port allocation (independent of config)
	port1, err1 := findAvailablePort(4000, 4099)
	if err1 != nil {
		t.Fatalf("findAvailablePort failed: %v", err1)
	}
	port2, err2 := findAvailablePort(4000, 4099)
	if err2 != nil {
		t.Fatalf("findAvailablePort failed: %v", err2)
	}

	// Ports should be in valid range
	if port1 < 4000 || port1 > 4099 {
		t.Errorf("Port1 %d out of range", port1)
	}
	if port2 < 4000 || port2 > 4099 {
		t.Errorf("Port2 %d out of range", port2)
	}

	// Should get different ports (most of the time)
	// Note: This test might occasionally have the same port due to timing
}

func TestExecuteHooksTimeout(t *testing.T) {
	hooks := []config.HookConfig{
		{
			Command: "sleep",
			Args:    []string{"10"}, // Long sleep to trigger timeout
			Timeout: "100ms",
		},
	}

	start := time.Now()
	err := ExecuteHooks(hooks, map[string]string{}, "timeout-test")
	duration := time.Since(start)

	// Should timeout and return error
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}

	// Should complete in reasonable time (not wait for full 10 seconds)
	if duration > 2*time.Second {
		t.Errorf("Hook took too long: %v", duration)
	}
}

func TestWebAppGetApp(t *testing.T) {
	cfg := &config.Config{
		Applications: config.Applications{
			Pools: config.Pools{
				MaxSize:   5,
				StartPort: 4000,
				Timeout:   "5m",
			},
			Tenants: []config.Tenant{
				{
					Name: "get-app-test",
					Root: "/tmp",
				},
			},
		},
	}

	appManager := NewAppManager(cfg)

	// Test getting non-existent app
	app, exists := appManager.GetApp("non-existent")
	if exists {
		t.Error("GetApp should return false for non-existent app")
	}
	if app != nil {
		t.Error("GetApp should return nil for non-existent app")
	}

	// Test updating config
	newCfg := &config.Config{
		Applications: config.Applications{
			Pools: config.Pools{
				MaxSize:   10,
				StartPort: 4500,
				Timeout:   "10m",
			},
		},
	}

	// Should not panic
	appManager.UpdateConfig(newCfg)

	if appManager.config != newCfg {
		t.Error("UpdateConfig should update the config reference")
	}
}

// Helper functions for testing
func createTestConfig() *config.Config {
	return &config.Config{
		ManagedProcesses: []config.ManagedProcessConfig{
			{
				Name:        "test-redis",
				Command:     "echo",
				Args:        []string{"redis-server"},
				WorkingDir:  "/tmp",
				Env:         map[string]string{"REDIS_PORT": "6379"},
				AutoRestart: true,
				StartDelay:  "1s",
			},
		},
	}
}