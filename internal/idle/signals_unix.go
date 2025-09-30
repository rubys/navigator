//go:build unix

package idle

import (
	"log/slog"
	"os"
	"syscall"
)

// sendSuspendSignal sends SIGTSTP to suspend the machine
func sendSuspendSignal() error {
	return syscall.Kill(os.Getpid(), syscall.SIGTSTP)
}

// sendStopSignal sends SIGTERM to stop the machine gracefully
func sendStopSignal() error {
	return syscall.Kill(os.Getpid(), syscall.SIGTERM)
}

// suspendMachine sends SIGTSTP to trigger machine suspension (Unix-specific)
func (m *Manager) suspendMachine() {
	slog.Info("Suspending machine due to inactivity",
		"timeout", m.idleTimeout,
		"lastActivity", m.lastActivity)

	// Skip signal sending in test mode
	if m.testMode {
		slog.Info("Test mode: skipping SIGTSTP signal")
		return
	}

	// Send SIGTSTP to self to trigger machine suspension
	if err := sendSuspendSignal(); err != nil {
		slog.Error("Failed to suspend machine", "error", err)
	}
}

// stopMachine sends SIGTERM to stop the machine gracefully (Unix-specific)
func (m *Manager) stopMachine() {
	slog.Info("Stopping machine due to inactivity",
		"timeout", m.idleTimeout,
		"lastActivity", m.lastActivity)

	// Skip signal sending in test mode
	if m.testMode {
		slog.Info("Test mode: skipping SIGTERM signal")
		return
	}

	// Send SIGTERM to self to trigger graceful shutdown
	if err := sendStopSignal(); err != nil {
		slog.Error("Failed to stop machine", "error", err)
	}
}