package process

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/rubys/navigator/internal/config"
	"github.com/rubys/navigator/internal/logging"
	"github.com/rubys/navigator/internal/utils"
)

// WebApp represents a web application instance
type WebApp struct {
	URL              string
	Process          *exec.Cmd
	Tenant           *config.Tenant
	Port             int
	StartTime        time.Time
	LastActivity     time.Time
	Starting         bool // True while app is starting up
	mutex            sync.Mutex
	cancel           context.CancelFunc
	wsConnections    map[string]interface{}
	wsConnectionsMux sync.RWMutex
	activeWebSockets int32 // Atomic counter for active WebSocket connections
}

// GetActiveWebSocketsPtr returns a pointer to the atomic WebSocket counter
// This allows external packages to track WebSocket connections using atomic operations
func (w *WebApp) GetActiveWebSocketsPtr() *int32 {
	return &w.activeWebSockets
}

// GetActiveWebSocketCount returns the current number of active WebSocket connections
func (w *WebApp) GetActiveWebSocketCount() int32 {
	return atomic.LoadInt32(&w.activeWebSockets)
}

// AppManager manages web application processes
type AppManager struct {
	apps           map[string]*WebApp
	config         *config.Config
	processStarter *ProcessStarter
	portAllocator  *PortAllocator
	mutex          sync.RWMutex
	idleTimeout    time.Duration
}

// NewAppManager creates a new application manager
func NewAppManager(cfg *config.Config) *AppManager {
	// Parse idle timeout from config
	idleTimeout := utils.ParseDurationWithDefault(cfg.Applications.Pools.Timeout, config.DefaultIdleTimeout)

	startPort := cfg.Applications.Pools.StartPort
	if startPort == 0 {
		startPort = config.DefaultStartPort
	}

	return &AppManager{
		apps:           make(map[string]*WebApp),
		config:         cfg,
		processStarter: NewProcessStarter(cfg),
		portAllocator:  NewPortAllocator(startPort, startPort+config.MaxPortRange),
		idleTimeout:    idleTimeout,
	}
}

// GetOrStartApp gets an existing app or starts a new one
func (m *AppManager) GetOrStartApp(tenantName string) (*WebApp, error) {
	m.mutex.RLock()
	app, exists := m.apps[tenantName]
	m.mutex.RUnlock()

	if exists {
		app.mutex.Lock()
		app.LastActivity = time.Now()
		isStarting := app.Starting
		app.mutex.Unlock()

		if !isStarting {
			return app, nil
		}

		// Wait if app is still starting (with timeout)
		timeout := time.NewTimer(config.RailsStartupTimeout)
		defer timeout.Stop()

		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-timeout.C:
				return nil, fmt.Errorf("timeout waiting for app %s to start", tenantName)
			case <-ticker.C:
				app.mutex.Lock()
				isStarting := app.Starting
				app.mutex.Unlock()
				if !isStarting {
					return app, nil
				}
			}
		}
	}

	// Start new app
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Double-check after acquiring write lock
	if app, exists := m.apps[tenantName]; exists {
		app.mutex.Lock()
		app.LastActivity = time.Now()
		isStarting := app.Starting
		app.mutex.Unlock()

		if !isStarting {
			return app, nil
		}

		// Another goroutine is starting the app, wait for it
		m.mutex.Unlock()
		timeout := time.NewTimer(config.RailsStartupTimeout)
		defer timeout.Stop()

		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-timeout.C:
				m.mutex.Lock() // Re-acquire lock before returning
				return nil, fmt.Errorf("timeout waiting for app %s to start", tenantName)
			case <-ticker.C:
				app.mutex.Lock()
				isStarting := app.Starting
				app.mutex.Unlock()
				if !isStarting {
					m.mutex.Lock() // Re-acquire lock before returning
					return app, nil
				}
			}
		}
	}

	// Find tenant configuration
	var tenant *config.Tenant
	for i := range m.config.Applications.Tenants {
		if m.config.Applications.Tenants[i].Name == tenantName {
			tenant = &m.config.Applications.Tenants[i]
			break
		}
	}

	if tenant == nil {
		return nil, fmt.Errorf("tenant %s not found", tenantName)
	}

	// Find an available port
	port, err := m.portAllocator.FindAvailablePort()
	if err != nil {
		return nil, fmt.Errorf("no available ports: %w", err)
	}

	app = &WebApp{
		URL:           fmt.Sprintf("http://localhost:%d", port),
		Tenant:        tenant,
		Port:          port,
		StartTime:     time.Now(),
		LastActivity:  time.Now(),
		Starting:      true, // Mark as starting
		wsConnections: make(map[string]interface{}),
	}

	// Register app immediately so other requests can see it's starting
	m.apps[tenantName] = app

	// Start the web application (this will clear Starting flag when ready)
	if err := m.processStarter.StartWebApp(app, tenant); err != nil {
		// Clean up on error
		delete(m.apps, tenantName)
		return nil, err
	}

	// Start idle cleanup goroutine for this app
	go m.monitorAppIdleTimeout(tenantName)

	return app, nil
}

// monitorAppIdleTimeout monitors and stops idle apps
func (m *AppManager) monitorAppIdleTimeout(tenantName string) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		m.mutex.RLock()
		app, exists := m.apps[tenantName]
		m.mutex.RUnlock()

		if !exists {
			return // App was removed
		}

		app.mutex.Lock()
		idleTime := time.Since(app.LastActivity)
		app.mutex.Unlock()

		// Don't stop if there are active WebSocket connections
		activeWS := app.GetActiveWebSocketCount()
		if activeWS > 0 {
			slog.Debug("App has active WebSocket connections, skipping idle check",
				"tenant", tenantName,
				"activeWebSockets", activeWS,
				"idleTime", idleTime)
			continue
		}

		if idleTime > m.idleTimeout {
			logging.LogWebAppIdle(tenantName, idleTime.Round(time.Second).String())

			// Execute tenant stop hooks before removing from registry
			if app.Tenant != nil {
				_ = ExecuteTenantHooks(m.config.Applications.Hooks.Stop, app.Tenant.Hooks.Stop,
					app.Tenant.Env, tenantName, "stop")
			}

			// Stop the process
			if app.cancel != nil {
				app.cancel()
			}

			// Remove from registry only after fully stopped
			m.mutex.Lock()
			delete(m.apps, tenantName)
			m.mutex.Unlock()

			// Clean up PID file
			if app.Tenant != nil {
				if pidfile, ok := app.Tenant.Env["PIDFILE"]; ok {
					if err := os.Remove(pidfile); err != nil && !os.IsNotExist(err) {
						slog.Warn("Error removing PID file", "file", pidfile, "error", err)
					}
				}
			}

			return // Exit the monitoring goroutine
		}
	}
}

// RegisterWebSocketConnection registers a new WebSocket connection for an app
func (app *WebApp) RegisterWebSocketConnection(connID string, conn interface{}) {
	app.wsConnectionsMux.Lock()
	defer app.wsConnectionsMux.Unlock()
	app.wsConnections[connID] = conn
	slog.Debug("Registered WebSocket connection", "app", app.Tenant.Name, "connID", connID, "total", len(app.wsConnections))
}

// UnregisterWebSocketConnection removes a WebSocket connection from an app
func (app *WebApp) UnregisterWebSocketConnection(connID string) {
	app.wsConnectionsMux.Lock()
	defer app.wsConnectionsMux.Unlock()
	delete(app.wsConnections, connID)
	slog.Debug("Unregistered WebSocket connection", "app", app.Tenant.Name, "connID", connID, "remaining", len(app.wsConnections))
}

// UpdateConfig updates the AppManager configuration after a reload
func (m *AppManager) UpdateConfig(newConfig *config.Config) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.config = newConfig
	m.processStarter = NewProcessStarter(newConfig)

	// Update idle timeout if changed
	m.idleTimeout = utils.ParseDurationWithDefault(newConfig.Applications.Pools.Timeout, config.DefaultIdleTimeout)

	// Update port range if changed
	startPort := newConfig.Applications.Pools.StartPort
	if startPort == 0 {
		startPort = config.DefaultStartPort
	}
	m.portAllocator = NewPortAllocator(startPort, startPort+config.MaxPortRange)

	slog.Info("Updated AppManager configuration",
		"idleTimeout", m.idleTimeout,
		"portRange", fmt.Sprintf("%d-%d", startPort, startPort+config.MaxPortRange))
}

// Cleanup stops all running web applications
func (m *AppManager) Cleanup() {
	m.CleanupWithContext(context.Background())
}

// CleanupWithContext stops all running web applications with context deadline
func (m *AppManager) CleanupWithContext(ctx context.Context) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	logging.LogCleanup("web applications")

	done := make(chan struct{})
	go func() {
		defer close(done)

		for tenantName, app := range m.apps {
			logging.LogWebAppStop(tenantName)

			// Execute tenant stop hooks
			if app.Tenant != nil {
				_ = ExecuteTenantHooks(m.config.Applications.Hooks.Stop, app.Tenant.Hooks.Stop,
					app.Tenant.Env, tenantName, "stop")
			}

			// Clean up PID file
			if app.Tenant != nil {
				if pidfile, ok := app.Tenant.Env["PIDFILE"]; ok {
					if err := os.Remove(pidfile); err != nil && !os.IsNotExist(err) {
						slog.Warn("Error removing PID file", "file", pidfile, "error", err)
					}
				}
			}

			if app.cancel != nil {
				app.cancel()
			}
		}

		// Clear the apps map
		m.apps = make(map[string]*WebApp)

		// Give processes a moment to exit cleanly
		time.Sleep(500 * time.Millisecond)
	}()

	// Wait for cleanup or context timeout
	select {
	case <-done:
		logging.LogCleanupComplete("web applications")
	case <-ctx.Done():
		slog.Warn("Context deadline exceeded during web app cleanup")
	}
}

// GetApp returns a web app by tenant name if it exists and is running
func (m *AppManager) GetApp(tenantName string) (*WebApp, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	app, exists := m.apps[tenantName]
	return app, exists
}

// Helper functions

// cleanupPidFile checks for and removes stale PID file
func cleanupPidFile(pidfilePath string) error {
	if pidfilePath == "" {
		return nil
	}

	// Check if PID file exists
	data, err := os.ReadFile(pidfilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No PID file, nothing to clean up
		}
		return fmt.Errorf("error reading PID file %s: %v", pidfilePath, err)
	}

	// Parse PID
	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		slog.Warn("Invalid PID in file", "file", pidfilePath, "pid", pidStr)
		// Remove invalid PID file
		os.Remove(pidfilePath)
		return nil
	}

	// Try to kill the process
	process, err := os.FindProcess(pid)
	if err == nil {
		// Send SIGTERM
		err = process.Signal(syscall.SIGTERM)
		if err == nil {
			slog.Info("Killed stale process", "pid", pid, "file", pidfilePath)
			// Give it a moment to exit cleanly
			time.Sleep(100 * time.Millisecond)
		}
		// Try SIGKILL if needed
		if err := process.Signal(syscall.SIGKILL); err != nil {
			slog.Debug("Failed to send SIGKILL to stale process", "pid", pid, "error", err)
		}
	}

	// Remove PID file
	if err := os.Remove(pidfilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error removing PID file %s: %v", pidfilePath, err)
	}

	return nil
}

// ParseURL safely parses a URL string
func (app *WebApp) ParseURL() (*url.URL, error) {
	return url.Parse(app.URL)
}
