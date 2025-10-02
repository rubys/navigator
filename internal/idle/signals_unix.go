//go:build unix

package idle

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"
)

// suspendMachine uses Fly API to suspend the machine (Unix-specific)
func (m *Manager) suspendMachine() {
	slog.Info("Suspending machine due to inactivity",
		"timeout", m.idleTimeout,
		"lastActivity", m.lastActivity)

	// Skip API call in test mode
	if m.testMode {
		slog.Info("Test mode: skipping machine suspend API call")
		return
	}

	if err := m.performFlyAction("suspend"); err != nil {
		slog.Error("Failed to suspend machine", "error", err)
	}
}

// stopMachine uses Fly API to stop the machine (Unix-specific)
func (m *Manager) stopMachine() {
	slog.Info("Stopping machine due to inactivity",
		"timeout", m.idleTimeout,
		"lastActivity", m.lastActivity)

	// Skip API call in test mode
	if m.testMode {
		slog.Info("Test mode: skipping machine stop API call")
		return
	}

	if err := m.performFlyAction("stop"); err != nil {
		slog.Error("Failed to stop machine", "error", err)
	}
}

// performFlyAction calls the Fly.io FLAPS API to suspend or stop the machine
func (m *Manager) performFlyAction(action string) error {
	appName := os.Getenv("FLY_APP_NAME")
	machineID := os.Getenv("FLY_MACHINE_ID")

	if appName == "" || machineID == "" {
		return fmt.Errorf("missing FLY_APP_NAME or FLY_MACHINE_ID environment variables")
	}

	// Create HTTP client with Unix socket transport
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return net.Dial("unix", "/.fly/api")
			},
		},
		Timeout: 10 * time.Second,
	}

	// Create request for the appropriate action
	url := fmt.Sprintf("http://flaps/v1/apps/%s/machines/%s/%s", appName, machineID, action)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create %s request: %w", action, err)
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute %s request: %w", action, err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read %s response: %w", action, err)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		slog.Info("Machine idle action requested successfully",
			"action", action,
			"status", resp.StatusCode,
			"response", string(body))
		return nil
	}

	return fmt.Errorf("%s request failed with status %d: %s", action, resp.StatusCode, string(body))
}
