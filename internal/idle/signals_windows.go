//go:build windows

package idle

import (
	"log/slog"
	"os"
)

// suspendMachine is not supported on Windows
func (m *Manager) suspendMachine() {
	slog.Warn("Machine suspension is not supported on Windows",
		"timeout", m.idleTimeout,
		"lastActivity", m.lastActivity)
}

// stopMachine attempts to exit gracefully on Windows
func (m *Manager) stopMachine() {
	slog.Info("Stopping machine due to inactivity",
		"timeout", m.idleTimeout,
		"lastActivity", m.lastActivity)

	// Skip signal sending in test mode
	if m.testMode {
		slog.Info("Test mode: skipping exit call")
		return
	}

	// On Windows, we exit with code 0 for graceful shutdown
	// This works differently than Unix signals but achieves similar result
	os.Exit(0)
}