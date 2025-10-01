//go:build integration
// +build integration

package process

import (
	"testing"
	"time"

	"github.com/rubys/navigator/internal/config"
)

// TestManagedProcessAutoRestart tests auto-restart functionality
// This is an integration test because it waits up to 11 seconds for restart attempts
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
