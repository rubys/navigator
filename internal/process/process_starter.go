package process

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/rubys/navigator/internal/config"
	"github.com/rubys/navigator/internal/logging"
)

// ProcessStarter handles starting web application processes
type ProcessStarter struct {
	config *config.Config
}

// NewProcessStarter creates a new process starter
func NewProcessStarter(cfg *config.Config) *ProcessStarter {
	return &ProcessStarter{
		config: cfg,
	}
}

// StartWebApp starts a web application process
func (ps *ProcessStarter) StartWebApp(app *WebApp, tenant *config.Tenant) error {
	// Clean up any existing PID file first
	if pidfile, ok := tenant.Env["PIDFILE"]; ok {
		_ = cleanupPidFile(pidfile)
	}

	// Determine runtime, server, and args
	runtime := ps.getRuntime(tenant)
	server := ps.getServer(tenant)
	args := ps.getArgs(tenant, app.Port)

	// Create command with context
	ctx, cancel := context.WithCancel(context.Background())
	app.cancel = cancel

	cmd := exec.CommandContext(ctx, runtime, append([]string{server}, args...)...)

	// Setup command environment and working directory
	ps.setupCommand(cmd, tenant, app.Port)

	// Create log writers for the app output
	tenantName := tenant.Name
	stdout := CreateLogWriter(tenantName, "stdout", ps.config.Logging)
	stderr := CreateLogWriter(tenantName, "stderr", ps.config.Logging)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	app.Process = cmd

	logging.LogWebAppStart(tenantName, app.Port, runtime, server, args)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start web app: %w", err)
	}

	// Execute tenant start hooks
	if err := ExecuteTenantHooks(ps.config.Applications.Hooks.Start, tenant.Hooks.Start,
		tenant.Env, tenantName, "start"); err != nil {
		slog.Error("Failed to execute tenant start hooks", "tenant", tenantName, "error", err)
	}

	// Wait for app to be ready
	return ps.waitForReady(app, tenantName, runtime)
}

// getRuntime determines the runtime command (e.g., "ruby", "python", "node")
func (ps *ProcessStarter) getRuntime(tenant *config.Tenant) string {
	runtime := tenant.Runtime
	if runtime == "" {
		// Check framework-specific runtime
		if tenant.Framework != "" && ps.config.Applications.Runtime != nil {
			runtime = ps.config.Applications.Runtime[tenant.Framework]
		}
	}
	if runtime == "" {
		runtime = "ruby" // Default to Ruby
	}
	return runtime
}

// getServer determines the server command (e.g., "bin/rails", "manage.py", "server.js")
func (ps *ProcessStarter) getServer(tenant *config.Tenant) string {
	server := tenant.Server
	if server == "" {
		// Check framework-specific server
		if tenant.Framework != "" && ps.config.Applications.Server != nil {
			server = ps.config.Applications.Server[tenant.Framework]
		}
	}
	if server == "" {
		server = "bin/rails" // Default to Rails
	}
	return server
}

// getArgs determines command arguments with port substitution
func (ps *ProcessStarter) getArgs(tenant *config.Tenant, port int) []string {
	args := tenant.Args
	if len(args) == 0 {
		// Check framework-specific args
		if tenant.Framework != "" && ps.config.Applications.Args != nil {
			args = ps.config.Applications.Args[tenant.Framework]
		}
	}
	if len(args) == 0 {
		// Default Rails server args
		args = []string{"server", "-b", "0.0.0.0", "-p", strconv.Itoa(port)}
	} else {
		// Replace {{port}} placeholder in args
		portStr := strconv.Itoa(port)
		result := make([]string, len(args))
		for i, arg := range args {
			result[i] = strings.ReplaceAll(arg, "{{port}}", portStr)
		}
		return result
	}
	return args
}

// setupCommand configures the command's working directory and environment
func (ps *ProcessStarter) setupCommand(cmd *exec.Cmd, tenant *config.Tenant, port int) {
	// Set working directory
	if tenant.Root != "" {
		cmd.Dir = tenant.Root
	}

	// Set environment
	cmd.Env = os.Environ()

	// Add PORT environment variable
	cmd.Env = append(cmd.Env, fmt.Sprintf("PORT=%d", port))

	// Add tenant-specific environment variables
	for key, value := range tenant.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}
}

// waitForReady waits for the web app to be ready to accept connections
func (ps *ProcessStarter) waitForReady(app *WebApp, tenantName, runtime string) error {
	// Skip readiness check if in test mode with echo command
	if os.Getenv("NAVIGATOR_TEST_SKIP_READINESS") == "true" || runtime == "echo" {
		slog.Debug("Skipping readiness check for test", "tenant", tenantName)
		return nil
	}

	// Wait for app to be ready
	readyCtx, readyCancel := context.WithTimeout(context.Background(), config.RailsStartupTimeout)
	defer readyCancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-readyCtx.Done():
			// Give app more time but don't fail
			slog.Warn("App startup timeout reached, continuing anyway",
				"tenant", tenantName,
				"timeout", config.RailsStartupTimeout)
			return nil
		case <-ticker.C:
			// Try to connect to the app
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", app.Port), 100*time.Millisecond)
			if err == nil {
				conn.Close()
				logging.LogWebAppReady(tenantName, app.Port)
				return nil
			}
		}
	}
}
