package idle

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/rubys/navigator/internal/config"
	"github.com/rubys/navigator/internal/process"
)

// Manager tracks active requests and handles machine idle actions
type Manager struct {
	enabled        bool
	action         string // "suspend" or "stop"
	idleTimeout    time.Duration
	activeRequests int64
	lastActivity   time.Time
	mutex          sync.RWMutex
	timer          *time.Timer
	config         *config.Config
	idleActioned   bool       // Track if idle action was performed
	resuming       bool       // Track if resume hooks are currently running
	resumeCond     *sync.Cond // Condition variable to wait for resume completion
	testMode       bool       // Prevents actual signal sending during tests
}

// NewManager creates a new idle manager
func NewManager(cfg *config.Config) *Manager {
	m := &Manager{
		config:       cfg,
		lastActivity: time.Now(),
	}

	// Initialize condition variable
	m.resumeCond = sync.NewCond(&m.mutex)

	// Configure idle settings
	if cfg.Server.Idle.Action != "" && (cfg.Server.Idle.Action == "suspend" || cfg.Server.Idle.Action == "stop") {
		m.enabled = true
		m.action = cfg.Server.Idle.Action

		// Parse idle timeout
		if cfg.Server.Idle.Timeout != "" {
			if duration, err := time.ParseDuration(cfg.Server.Idle.Timeout); err == nil {
				m.idleTimeout = duration
			} else {
				m.idleTimeout = config.DefaultIdleTimeout
			}
		} else {
			m.idleTimeout = config.DefaultIdleTimeout
		}

		slog.Info("Machine idle management enabled",
			"action", m.action,
			"timeout", m.idleTimeout)
	}

	return m
}

// RequestStarted increments the active request counter
func (m *Manager) RequestStarted() {
	if !m.enabled {
		return
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Wait for resume hooks to complete if they're running
	for m.resuming {
		m.resumeCond.Wait()
	}

	// If this is the first request after idle action, execute resume hooks
	if m.idleActioned {
		m.idleActioned = false
		m.resuming = true

		// Execute resume hooks asynchronously
		go func() {
			slog.Info("Executing server resume hooks")
			if err := process.ExecuteServerHooks(m.config.Hooks.Resume, "resume"); err != nil {
				slog.Error("Failed to execute resume hooks", "error", err)
			}

			m.mutex.Lock()
			m.resuming = false
			m.resumeCond.Broadcast() // Wake up any waiting requests
			m.mutex.Unlock()
		}()
	}

	m.activeRequests++
	m.lastActivity = time.Now()

	// Cancel any pending idle timer
	if m.timer != nil {
		m.timer.Stop()
		m.timer = nil
	}
}

// RequestFinished decrements the active request counter and starts idle timer if needed
func (m *Manager) RequestFinished() {
	if !m.enabled {
		return
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.activeRequests > 0 {
		m.activeRequests--
	}

	m.lastActivity = time.Now()

	// If no more active requests, start idle timer
	if m.activeRequests == 0 && m.timer == nil {
		m.timer = time.AfterFunc(m.idleTimeout, m.handleIdle)
		slog.Debug("Started idle timer",
			"timeout", m.idleTimeout,
			"action", m.action)
	}
}

// handleIdle performs the configured idle action
func (m *Manager) handleIdle() {
	m.mutex.Lock()
	// Check again if there are no active requests
	if m.activeRequests > 0 {
		m.mutex.Unlock()
		return
	}

	// Check if enough time has passed since last activity
	if time.Since(m.lastActivity) < m.idleTimeout {
		// Reschedule
		remaining := m.idleTimeout - time.Since(m.lastActivity)
		m.timer = time.AfterFunc(remaining, m.handleIdle)
		m.mutex.Unlock()
		return
	}

	action := m.action
	m.idleActioned = true // Mark that idle action was performed
	m.mutex.Unlock()

	// Execute idle hooks
	slog.Info("Executing server idle hooks before machine idle action", "action", action)
	if err := process.ExecuteServerHooks(m.config.Hooks.Idle, "idle"); err != nil {
		slog.Error("Failed to execute idle hooks", "error", err)
	}

	// Perform the idle action
	switch action {
	case "suspend":
		m.suspendMachine()
	case "stop":
		m.stopMachine()
	default:
		slog.Warn("Unknown idle action", "action", action)
	}
}

// suspendMachine and stopMachine are implemented in platform-specific files:
// - signals_unix.go for Unix/Linux/macOS
// - signals_windows.go for Windows

// Stop cancels any pending idle timer
func (m *Manager) Stop() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.timer != nil {
		m.timer.Stop()
		m.timer = nil
	}
}

// Suspend suspends the machine immediately (for external trigger)
func (m *Manager) Suspend() error {
	if !m.enabled || m.action != "suspend" {
		return fmt.Errorf("machine suspension not enabled")
	}

	m.mutex.Lock()
	m.idleActioned = true
	m.mutex.Unlock()

	// Execute idle hooks before suspension
	process.ExecuteServerHooks(m.config.Hooks.Idle, "idle")

	m.suspendMachine()
	return nil
}

// IsEnabled returns whether idle management is enabled
func (m *Manager) IsEnabled() bool {
	return m.enabled
}

// GetStats returns current idle manager statistics
func (m *Manager) GetStats() (activeRequests int64, lastActivity time.Time) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.activeRequests, m.lastActivity
}

// UpdateConfig updates the idle manager configuration after a reload
func (m *Manager) UpdateConfig(newConfig *config.Config) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.config = newConfig

	// Re-configure idle settings from new config
	if newConfig.Server.Idle.Action != "" && (newConfig.Server.Idle.Action == "suspend" || newConfig.Server.Idle.Action == "stop") {
		m.enabled = true
		m.action = newConfig.Server.Idle.Action

		// Parse idle timeout
		if newConfig.Server.Idle.Timeout != "" {
			if duration, err := time.ParseDuration(newConfig.Server.Idle.Timeout); err == nil {
				m.idleTimeout = duration
			} else {
				m.idleTimeout = config.DefaultIdleTimeout
			}
		} else {
			m.idleTimeout = config.DefaultIdleTimeout
		}

		slog.Debug("Updated idle manager configuration",
			"action", m.action,
			"timeout", m.idleTimeout)
	} else {
		m.enabled = false
		// Cancel any pending idle timer if idle management is disabled
		if m.timer != nil {
			m.timer.Stop()
			m.timer = nil
		}
	}
}

// EnableTestMode prevents actual signal sending for testing
func (m *Manager) EnableTestMode() {
	m.testMode = true
}
