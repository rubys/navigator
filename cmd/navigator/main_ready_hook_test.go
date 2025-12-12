package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rubys/navigator/internal/config"
	"github.com/rubys/navigator/internal/idle"
	"github.com/rubys/navigator/internal/process"
)

func TestReadyHookExecutesAfterReload(t *testing.T) {
	// This test verifies that ready hooks execute after configuration reload

	// Create a temporary directory for test files
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test-config.yml")
	hookOutputFile := filepath.Join(tempDir, "hook-output.txt")
	hookScript := filepath.Join(tempDir, "ready-hook.sh")

	// Create a hook script that writes to a file
	hookContent := `#!/bin/bash
echo "Ready hook executed at $(date)" > "` + hookOutputFile + `"
`
	err := os.WriteFile(hookScript, []byte(hookContent), 0755)
	if err != nil {
		t.Fatalf("Failed to create hook script: %v", err)
	}

	// Create config with ready hook
	configContent := `
server:
  listen: "3004"
  hostname: "localhost"
applications:
  tenants: []
hooks:
  server:
    ready:
      - command: "` + hookScript + `"
        timeout: "5s"
logging:
  format: "text"
`
	err = os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Load initial config
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify ready hook is configured
	if len(cfg.Hooks.Ready) != 1 {
		t.Fatalf("Expected 1 ready hook, got %d", len(cfg.Hooks.Ready))
	}

	// Create managers
	appManager := process.NewAppManager(cfg)
	processManager := process.NewManager(cfg)
	idleManager := idle.NewManager(cfg, "", time.Time{}, nil)

	// Create lifecycle
	lifecycle := &ServerLifecycle{
		configFile:     configFile,
		cfg:            cfg,
		appManager:     appManager,
		processManager: processManager,
		basicAuth:      nil,
		idleManager:    idleManager,
	}

	// Verify hook output file doesn't exist yet
	if _, err := os.Stat(hookOutputFile); err == nil {
		t.Fatal("Hook output file should not exist before reload")
	}

	// Trigger reload - this should execute ready hooks asynchronously
	lifecycle.handleReload()

	// Wait for async hook to execute (hooks run in goroutine)
	// Give it up to 2 seconds to complete
	time.Sleep(2 * time.Second)

	// Verify hook was executed by checking if output file exists
	if _, err := os.Stat(hookOutputFile); os.IsNotExist(err) {
		t.Error("Ready hook did not execute - output file not found")
	}

	// Verify output file has content
	content, err := os.ReadFile(hookOutputFile)
	if err != nil {
		t.Fatalf("Failed to read hook output file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Ready hook output file is empty")
	}

	t.Logf("Ready hook output: %s", string(content))
}

func TestReadyHookExecutesOnInitialStart(t *testing.T) {
	// This test verifies that ready hooks still execute on initial start
	// (existing behavior should be preserved)

	// Create a temporary directory for test files
	tempDir := t.TempDir()
	hookOutputFile := filepath.Join(tempDir, "initial-hook-output.txt")
	hookScript := filepath.Join(tempDir, "initial-ready-hook.sh")

	// Create a hook script
	hookContent := `#!/bin/bash
echo "Initial ready hook executed" > "` + hookOutputFile + `"
`
	err := os.WriteFile(hookScript, []byte(hookContent), 0755)
	if err != nil {
		t.Fatalf("Failed to create hook script: %v", err)
	}

	// Create config with ready hook
	configContent := `
server:
  listen: "3005"
  hostname: "localhost"
applications:
  tenants: []
hooks:
  server:
    ready:
      - command: "` + hookScript + `"
        timeout: "5s"
logging:
  format: "text"
`
	configFile := filepath.Join(tempDir, "test-config-initial.yml")
	err = os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Load config
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Execute ready hooks (simulating initial start)
	err = process.ExecuteServerHooks(cfg.Hooks.Ready, "ready")
	if err != nil {
		t.Fatalf("Failed to execute ready hooks: %v", err)
	}

	// Wait for hook to complete
	time.Sleep(1 * time.Second)

	// Verify hook was executed
	if _, err := os.Stat(hookOutputFile); os.IsNotExist(err) {
		t.Error("Ready hook did not execute on initial start - output file not found")
	}

	// Verify output file has content
	content, err := os.ReadFile(hookOutputFile)
	if err != nil {
		t.Fatalf("Failed to read hook output file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Ready hook output file is empty")
	}

	t.Logf("Initial ready hook output: %s", string(content))
}
