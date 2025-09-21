package process

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/rubys/navigator/internal/config"
)

// WebApp represents a web application instance
type WebApp struct {
	URL              string
	Process          *exec.Cmd
	Tenant           *config.Tenant
	Port             int
	StartTime        time.Time
	LastActivity     time.Time
	mutex            sync.Mutex
	cancel           context.CancelFunc
	wsConnections    map[string]interface{}
	wsConnectionsMux sync.RWMutex
}

// AppManager manages web application processes
type AppManager struct {
	apps        map[string]*WebApp
	config      *config.Config
	mutex       sync.RWMutex
	idleTimeout time.Duration
	minPort     int // Minimum port for web apps
	maxPort     int // Maximum port for web apps
}

// NewAppManager creates a new application manager
func NewAppManager(cfg *config.Config) *AppManager {
	idleTimeout := config.DefaultIdleTimeout

	// Parse idle timeout from config
	if cfg.Applications.Pools.Timeout != "" {
		if duration, err := time.ParseDuration(cfg.Applications.Pools.Timeout); err == nil {
			idleTimeout = duration
		}
	}

	startPort := cfg.Applications.Pools.StartPort
	if startPort == 0 {
		startPort = config.DefaultStartPort
	}

	return &AppManager{
		apps:        make(map[string]*WebApp),
		config:      cfg,
		idleTimeout: idleTimeout,
		minPort:     startPort,
		maxPort:     startPort + config.MaxPortRange,
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
		app.mutex.Unlock()
		return app, nil
	}

	// Start new app
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Double-check after acquiring write lock
	if app, exists := m.apps[tenantName]; exists {
		app.LastActivity = time.Now()
		return app, nil
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
	port, err := findAvailablePort(m.minPort, m.maxPort)
	if err != nil {
		return nil, fmt.Errorf("no available ports: %w", err)
	}

	app = &WebApp{
		URL:           fmt.Sprintf("http://localhost:%d", port),
		Tenant:        tenant,
		Port:          port,
		StartTime:     time.Now(),
		LastActivity:  time.Now(),
		wsConnections: make(map[string]interface{}),
	}

	// Start the Rails application
	if err := m.startWebApp(app, tenant); err != nil {
		return nil, err
	}

	m.apps[tenantName] = app

	// Start idle cleanup goroutine for this app
	go m.monitorAppIdleTimeout(tenantName)

	return app, nil
}

// startWebApp starts a web application process
func (m *AppManager) startWebApp(app *WebApp, tenant *config.Tenant) error {
	// Clean up any existing PID file first
	if pidfile, ok := tenant.Env["PIDFILE"]; ok {
		cleanupPidFile(pidfile)
	}

	// Determine runtime command (e.g., "ruby", "python", "node")
	runtime := tenant.Runtime
	if runtime == "" {
		// Check framework-specific runtime
		if tenant.Framework != "" && m.config.Applications.Runtime != nil {
			runtime = m.config.Applications.Runtime[tenant.Framework]
		}
	}
	if runtime == "" {
		runtime = "ruby" // Default to Ruby
	}

	// Determine server command (e.g., "bin/rails", "manage.py", "server.js")
	server := tenant.Server
	if server == "" {
		// Check framework-specific server
		if tenant.Framework != "" && m.config.Applications.Server != nil {
			server = m.config.Applications.Server[tenant.Framework]
		}
	}
	if server == "" {
		server = "bin/rails" // Default to Rails
	}

	// Determine command arguments
	args := tenant.Args
	if len(args) == 0 {
		// Check framework-specific args
		if tenant.Framework != "" && m.config.Applications.Args != nil {
			args = m.config.Applications.Args[tenant.Framework]
		}
	}
	if len(args) == 0 {
		// Default Rails server args
		args = []string{"server", "-b", "0.0.0.0", "-p", strconv.Itoa(app.Port)}
	} else {
		// Replace {{port}} placeholder in args
		for i, arg := range args {
			args[i] = strings.ReplaceAll(arg, "{{port}}", strconv.Itoa(app.Port))
		}
	}

	// Create command
	ctx, cancel := context.WithCancel(context.Background())
	app.cancel = cancel

	cmd := exec.CommandContext(ctx, runtime, append([]string{server}, args...)...)

	// Set working directory
	if tenant.Root != "" {
		cmd.Dir = tenant.Root
	}

	// Set environment
	cmd.Env = os.Environ()

	// Add PORT environment variable
	cmd.Env = append(cmd.Env, fmt.Sprintf("PORT=%d", app.Port))

	// Add tenant-specific environment variables
	for key, value := range tenant.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Create log writers for the app output
	tenantName := tenant.Name
	stdout := CreateLogWriter(tenantName, "stdout", m.config.Logging)
	stderr := CreateLogWriter(tenantName, "stderr", m.config.Logging)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	app.Process = cmd

	slog.Info("Starting web app",
		"tenant", tenantName,
		"port", app.Port,
		"runtime", runtime,
		"server", server,
		"args", args,
		"dir", cmd.Dir)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start web app: %w", err)
	}

	// Execute tenant start hooks
	if err := ExecuteTenantHooks(m.config.Applications.Hooks.Start, tenant.Hooks.Start,
		tenant.Env, tenantName, "start"); err != nil {
		slog.Error("Failed to execute tenant start hooks", "tenant", tenantName, "error", err)
	}

	// Wait for Rails to be ready
	readyCtx, readyCancel := context.WithTimeout(context.Background(), config.RailsStartupTimeout)
	defer readyCancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-readyCtx.Done():
			// Give Rails more time but don't fail
			slog.Warn("Rails startup timeout reached, continuing anyway",
				"tenant", tenantName,
				"timeout", config.RailsStartupTimeout)
			return nil
		case <-ticker.C:
			// Try to connect to the Rails app
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", app.Port), 100*time.Millisecond)
			if err == nil {
				conn.Close()
				slog.Info("Web app is ready", "tenant", tenantName, "port", app.Port)
				return nil
			}
		}
	}
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
		hasConnections := len(app.wsConnections) > 0
		app.mutex.Unlock()

		// Don't stop if there are active WebSocket connections
		if hasConnections {
			continue
		}

		if idleTime > m.idleTimeout {
			slog.Info("Stopping idle web app",
				"tenant", tenantName,
				"idleTime", idleTime.Round(time.Second))

			m.mutex.Lock()
			delete(m.apps, tenantName)
			m.mutex.Unlock()

			// Execute tenant stop hooks
			if app.Tenant != nil {
				ExecuteTenantHooks(m.config.Applications.Hooks.Stop, app.Tenant.Hooks.Stop,
					app.Tenant.Env, tenantName, "stop")
			}

			// Stop the process
			if app.cancel != nil {
				app.cancel()
			}

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

	// Update idle timeout if changed
	idleTimeout := config.DefaultIdleTimeout
	if newConfig.Applications.Pools.Timeout != "" {
		if duration, err := time.ParseDuration(newConfig.Applications.Pools.Timeout); err == nil {
			idleTimeout = duration
		}
	}
	m.idleTimeout = idleTimeout

	// Update port range if changed
	startPort := newConfig.Applications.Pools.StartPort
	if startPort == 0 {
		startPort = config.DefaultStartPort
	}
	m.minPort = startPort
	m.maxPort = startPort + config.MaxPortRange

	slog.Info("Updated AppManager configuration",
		"idleTimeout", m.idleTimeout,
		"minPort", m.minPort,
		"maxPort", m.maxPort)
}

// Cleanup stops all running web applications
func (m *AppManager) Cleanup() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	slog.Info("Cleaning up all web applications")

	for tenantName, app := range m.apps {
		slog.Info("Stopping web app", "tenant", tenantName)

		// Execute tenant stop hooks
		if app.Tenant != nil {
			ExecuteTenantHooks(m.config.Applications.Hooks.Stop, app.Tenant.Hooks.Stop,
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
		process.Signal(syscall.SIGKILL)
	}

	// Remove PID file
	if err := os.Remove(pidfilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error removing PID file %s: %v", pidfilePath, err)
	}

	return nil
}

// findAvailablePort finds an available port in the specified range
func findAvailablePort(minPort, maxPort int) (int, error) {
	for port := minPort; port <= maxPort; port++ {
		// Try to listen on the port
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			// Port is available
			listener.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available ports in range %d-%d", minPort, maxPort)
}

// ParseURL safely parses a URL string
func (app *WebApp) ParseURL() (*url.URL, error) {
	return url.Parse(app.URL)
}