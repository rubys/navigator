package process

import (
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