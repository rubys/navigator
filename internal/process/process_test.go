package process

import (
	"fmt"
	"os"
	"strings"
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
		return
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
		_ = ExecuteHooks(hooks, env, "benchmark")
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

// Note: TestManagedProcessAutoRestart moved to process_integration_test.go because it takes 11s+ to run

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

	// Test port allocation (use the port allocator)
	port, err := appManager.portAllocator.FindAvailablePort()
	if err != nil {
		t.Fatalf("FindAvailablePort failed: %v", err)
	}
	if port < 4000 || port > 4099 {
		t.Errorf("Port %d is outside expected range [%d, %d]", port, 4000, 4099)
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
	_ = manager.StartManagedProcesses()
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
			_ = manager.StartManagedProcesses()
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
	// Test port allocation (using PortAllocator)
	allocator := NewPortAllocator(4000, 4099)
	port1, err1 := allocator.FindAvailablePort()
	if err1 != nil {
		t.Fatalf("FindAvailablePort failed: %v", err1)
	}
	port2, err2 := allocator.FindAvailablePort()
	if err2 != nil {
		t.Fatalf("FindAvailablePort failed: %v", err2)
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

func TestStartWebApp(t *testing.T) {
	cfg := &config.Config{
		Applications: config.Applications{
			Pools: config.Pools{
				MaxSize:   5,
				StartPort: 4000,
				Timeout:   "5m",
			},
			Tenants: []config.Tenant{
				{
					Name:      "test-webapp",
					Root:      "/tmp",
					Runtime:   "echo", // Use echo to avoid actually starting a web server
					Server:    "test",
					Args:      []string{"testing", "{{port}}"},
					Framework: "test",
					Env: map[string]string{
						"TEST_VAR": "test_value",
						"PIDFILE":  "/tmp/test.pid",
					},
				},
			},
			Runtime: map[string]string{
				"test": "echo",
			},
			Server: map[string]string{
				"test": "test-server",
			},
			Args: map[string][]string{
				"test": {"default", "args", "{{port}}"},
			},
		},
	}

	appManager := NewAppManager(cfg)

	// Create a test app
	app := &WebApp{
		Port:          4001,
		StartTime:     time.Now(),
		LastActivity:  time.Now(),
		wsConnections: make(map[string]interface{}),
	}

	tenant := &cfg.Applications.Tenants[0]

	// Test starting a web app
	err := appManager.processStarter.StartWebApp(app, tenant)
	if err != nil {
		t.Errorf("StartWebApp should not error with echo command: %v", err)
	}

	// Cleanup
	if app.cancel != nil {
		app.cancel()
	}
}

func TestMonitorAppIdleTimeout(t *testing.T) {
	cfg := &config.Config{
		Applications: config.Applications{
			Pools: config.Pools{
				Timeout: "100ms", // Very short timeout for testing
			},
			Tenants: []config.Tenant{
				{
					Name: "idle-test",
					Root: "/tmp",
				},
			},
		},
	}

	appManager := NewAppManager(cfg)

	// Create a test app
	app := &WebApp{
		Port:          4002,
		StartTime:     time.Now(),
		LastActivity:  time.Now().Add(-200 * time.Millisecond), // Make it look idle
		wsConnections: make(map[string]interface{}),
	}

	// Add app to manager
	appManager.mutex.Lock()
	appManager.apps["idle-test"] = app
	appManager.mutex.Unlock()

	// Test monitoring (this will run in background)
	go appManager.monitorAppIdleTimeout("idle-test")

	// Give some time for monitoring to potentially detect idle state
	time.Sleep(50 * time.Millisecond)

	// The test app should still exist (since echo command finishes quickly)
	appManager.mutex.RLock()
	_, exists := appManager.apps["idle-test"]
	appManager.mutex.RUnlock()

	if !exists {
		t.Log("App was removed due to idle timeout (this is expected behavior)")
	}

	// Cleanup
	appManager.Cleanup()
}

func TestWebSocketConnectionManagement(t *testing.T) {
	cfg := &config.Config{
		Applications: config.Applications{
			Pools: config.Pools{
				MaxSize:   5,
				StartPort: 4000,
			},
			Tenants: []config.Tenant{
				{
					Name: "websocket-test",
					Root: "/tmp",
				},
			},
		},
	}

	appManager := NewAppManager(cfg)

	// Create a test app
	app := &WebApp{
		Port:          4003,
		StartTime:     time.Now(),
		LastActivity:  time.Now(),
		wsConnections: make(map[string]interface{}),
		Tenant:        &cfg.Applications.Tenants[0],
	}

	appManager.mutex.Lock()
	appManager.apps["websocket-test"] = app
	appManager.mutex.Unlock()

	// Test registering WebSocket connections
	connectionID := "test-connection-1"
	app.RegisterWebSocketConnection(connectionID, "test-connection-object")

	// Verify connection was registered (this exercises the WebSocket registration code)
	app.wsConnectionsMux.RLock()
	_, exists := app.wsConnections[connectionID]
	app.wsConnectionsMux.RUnlock()

	if !exists {
		t.Error("WebSocket connection should be registered")
	}

	// Test unregistering WebSocket connections
	app.UnregisterWebSocketConnection(connectionID)

	// Verify connection was unregistered
	app.wsConnectionsMux.RLock()
	_, exists = app.wsConnections[connectionID]
	app.wsConnectionsMux.RUnlock()

	if exists {
		t.Error("WebSocket connection should be unregistered")
	}

	// Cleanup
	appManager.Cleanup()
}

func TestCleanupPidFile(t *testing.T) {
	// Create a temporary PID file
	pidFile := "/tmp/test-navigator.pid"

	// Create the file
	err := os.WriteFile(pidFile, []byte("12345"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test PID file: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(pidFile); os.IsNotExist(err) {
		t.Fatal("Test PID file should exist")
	}

	// Test cleanup
	_ = cleanupPidFile(pidFile)

	// Verify file was removed
	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Error("PID file should be removed after cleanup")
		// Clean up if test failed
		os.Remove(pidFile)
	}

	// Test cleanup of non-existent file (should not error)
	_ = cleanupPidFile("/tmp/non-existent-pid.pid")
}

func TestStartWebAppDefaultValues(t *testing.T) {
	cfg := &config.Config{
		Applications: config.Applications{
			Pools: config.Pools{
				MaxSize:   5,
				StartPort: 4000,
			},
			Tenants: []config.Tenant{
				{
					Name: "default-test",
					Root: "/tmp",
					// No runtime, server, or args specified - should use defaults
				},
			},
		},
	}

	appManager := NewAppManager(cfg)

	app := &WebApp{
		Port:          4004,
		StartTime:     time.Now(),
		LastActivity:  time.Now(),
		wsConnections: make(map[string]interface{}),
	}

	tenant := &cfg.Applications.Tenants[0]

	// Test with no runtime/server specified (should use defaults)
	// We'll replace the default runtime with echo to avoid starting Ruby
	originalTenant := *tenant
	tenant.Runtime = "echo"
	tenant.Server = "test"
	tenant.Args = []string{"default", "test"}

	err := appManager.processStarter.StartWebApp(app, tenant)
	if err != nil {
		t.Errorf("StartWebApp should not error with default values: %v", err)
	}

	// Cleanup
	if app.cancel != nil {
		app.cancel()
	}

	// Restore original tenant
	*tenant = originalTenant
}

func TestStartWebAppWithFrameworkConfig(t *testing.T) {
	cfg := &config.Config{
		Applications: config.Applications{
			Pools: config.Pools{
				MaxSize:   5,
				StartPort: 4000,
			},
			Tenants: []config.Tenant{
				{
					Name:      "framework-test",
					Root:      "/tmp",
					Framework: "custom",
					// Runtime, server, and args will come from framework config
				},
			},
			Runtime: map[string]string{
				"custom": "echo",
			},
			Server: map[string]string{
				"custom": "framework-server",
			},
			Args: map[string][]string{
				"custom": {"framework", "args", "{{port}}"},
			},
		},
	}

	appManager := NewAppManager(cfg)

	app := &WebApp{
		Port:          4005,
		StartTime:     time.Now(),
		LastActivity:  time.Now(),
		wsConnections: make(map[string]interface{}),
	}

	tenant := &cfg.Applications.Tenants[0]

	// Test with framework configuration
	err := appManager.processStarter.StartWebApp(app, tenant)
	if err != nil {
		t.Errorf("StartWebApp should not error with framework config: %v", err)
	}

	// Cleanup
	if app.cancel != nil {
		app.cancel()
	}
}

func TestLoggingComponents(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		stream    string
		logConfig config.LogConfig
	}{
		{
			name:      "File logging with text format",
			source:    "test-app",
			stream:    "stdout",
			logConfig: config.LogConfig{Format: "text", File: "/tmp/test-navigator.log"},
		},
		{
			name:      "File logging with JSON format",
			source:    "test-app",
			stream:    "stderr",
			logConfig: config.LogConfig{Format: "json", File: "/tmp/test-navigator-json.log"},
		},
		{
			name:      "Console logging with app template",
			source:    "test-app",
			stream:    "stdout",
			logConfig: config.LogConfig{Format: "text", File: "/tmp/{{app}}.log"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test createFileWriter
			writer := CreateLogWriter(tt.source, tt.stream, tt.logConfig)
			if writer == nil {
				t.Error("CreateLogWriter should return a non-nil writer")
			}

			// Test writing to the logger
			testMessage := "Test log message for " + tt.name + "\n"
			_, err := writer.Write([]byte(testMessage))
			if err != nil {
				t.Errorf("Writer should not error on write: %v", err)
			}

			// Clean up test files
			if tt.logConfig.File != "" {
				// Handle template expansion
				filename := tt.logConfig.File
				if strings.Contains(filename, "{{app}}") {
					filename = strings.ReplaceAll(filename, "{{app}}", tt.source)
				}
				os.Remove(filename)
			}
		})
	}
}

func TestVectorLogging(t *testing.T) {
	// Test NewVectorWriter (even though it returns nil in current implementation)
	vectorWriter := NewVectorWriter("/tmp/test-vector.sock")

	// Should return a writer (even if it's a no-op)
	if vectorWriter == nil {
		// This is expected behavior in current implementation
		t.Log("VectorWriter returns nil (expected in current implementation)")
	} else {
		// If implementation changes to return a writer, test it
		_, err := vectorWriter.Write([]byte("test vector message\n"))
		if err != nil {
			t.Errorf("VectorWriter should not error on write: %v", err)
		}

		// Test Close method
		err = vectorWriter.Close()
		if err != nil {
			t.Errorf("VectorWriter Close should not error: %v", err)
		}
	}
}

func TestJSONLogWriter(t *testing.T) {
	// Test JSON log writer with various sources and streams
	tests := []struct {
		source string
		stream string
	}{
		{"json-test-app", "stdout"},
		{"json-test-app", "stderr"},
		{"another-app", "stdout"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s-%s", tt.source, tt.stream), func(t *testing.T) {
			cfg := config.LogConfig{Format: "json"}
			writer := CreateLogWriter(tt.source, tt.stream, cfg)

			if writer == nil {
				t.Error("CreateLogWriter should return a non-nil writer for JSON format")
			}

			// Test writing JSON log entries
			testMessages := []string{
				"Simple log message",
				"Log message with special characters: !@#$%^&*()",
				"Multi\nline\nlog\nmessage",
				"Log message with emoji: 🚀 Navigator logging test",
			}

			for _, msg := range testMessages {
				_, err := writer.Write([]byte(msg + "\n"))
				if err != nil {
					t.Errorf("JSON writer should not error on write: %v", err)
				}
			}
		})
	}
}

func TestFileLogWriter(t *testing.T) {
	// Test file-based logging
	tempDir := t.TempDir()
	logFile := tempDir + "/test-file-logging.log"

	cfg := config.LogConfig{
		Format: "text",
		File:   logFile,
	}

	writer := CreateLogWriter("file-test-app", "stdout", cfg)
	if writer == nil {
		t.Error("CreateLogWriter should return a non-nil writer for file logging")
	}

	// Write test messages
	testMessages := []string{
		"File logging test message 1",
		"File logging test message 2",
		"File logging test message 3",
	}

	for _, msg := range testMessages {
		_, err := writer.Write([]byte(msg + "\n"))
		if err != nil {
			t.Errorf("File writer should not error on write: %v", err)
		}
	}

	// Verify file was created and contains content
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Error("Log file should be created")
	} else {
		// Read file content
		content, err := os.ReadFile(logFile)
		if err != nil {
			t.Errorf("Should be able to read log file: %v", err)
		} else {
			contentStr := string(content)
			for _, msg := range testMessages {
				if !strings.Contains(contentStr, msg) {
					t.Errorf("Log file should contain message: %s", msg)
				}
			}
		}
	}
}

func TestTemplateLogFileNames(t *testing.T) {
	// Test log file template expansion
	tempDir := t.TempDir()

	tests := []struct {
		name         string
		source       string
		template     string
		expectedFile string
	}{
		{
			name:         "App template expansion",
			source:       "my-app",
			template:     tempDir + "/{{app}}.log",
			expectedFile: tempDir + "/my-app.log",
		},
		{
			name:         "No template",
			source:       "my-app",
			template:     tempDir + "/static.log",
			expectedFile: tempDir + "/static.log",
		},
		{
			name:         "Multiple app references",
			source:       "test-app",
			template:     tempDir + "/{{app}}-{{app}}.log",
			expectedFile: tempDir + "/test-app-test-app.log",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.LogConfig{
				Format: "text",
				File:   tt.template,
			}

			writer := CreateLogWriter(tt.source, "stdout", cfg)
			if writer == nil {
				t.Error("CreateLogWriter should return a non-nil writer")
			}

			// Write a test message
			_, err := writer.Write([]byte("Template test message\n"))
			if err != nil {
				t.Errorf("Writer should not error: %v", err)
			}

			// Verify the expected file was created
			if _, err := os.Stat(tt.expectedFile); os.IsNotExist(err) {
				t.Errorf("Expected file %s should be created", tt.expectedFile)
			}
		})
	}
}

func TestMultiDestinationLogging(t *testing.T) {
	// Test logging to both console and file simultaneously
	tempDir := t.TempDir()
	logFile := tempDir + "/multi-dest.log"

	cfg := config.LogConfig{
		Format: "json",
		File:   logFile,
	}

	writer := CreateLogWriter("multi-dest-app", "stdout", cfg)
	if writer == nil {
		t.Error("CreateLogWriter should return a non-nil writer")
	}

	// Write test messages that should go to both destinations
	testMessage := "Multi-destination logging test"
	_, err := writer.Write([]byte(testMessage + "\n"))
	if err != nil {
		t.Errorf("Writer should not error: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Error("Log file should be created for multi-destination logging")
	}

	// Verify file contains the message
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Errorf("Should be able to read log file: %v", err)
	} else if !strings.Contains(string(content), testMessage) {
		t.Error("Log file should contain the test message")
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
