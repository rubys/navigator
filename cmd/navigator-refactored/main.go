// This is an example of how the refactored main.go would look
// NOTE: This is not a complete implementation - just showing the new structure

package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/rubys/navigator/internal/auth"
	"github.com/rubys/navigator/internal/config"
	"github.com/rubys/navigator/internal/idle"
	"github.com/rubys/navigator/internal/process"
	"github.com/rubys/navigator/internal/server"
	"github.com/rubys/navigator/internal/utils"
)

var (
	version   = "dev"     // Set via -ldflags at build time
	commit    = "none"    // Git commit hash
	buildTime = "unknown" // Build timestamp
)

func main() {
	// Initialize basic logger
	initLogger()

	// Handle command line arguments
	if err := handleCommandLineArgs(); err != nil {
		slog.Error("Command failed", "error", err)
		os.Exit(1)
	}

	// Determine config file path
	configFile := "config/navigator.yml"
	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") {
		configFile = os.Args[1]
	}

	// Load configuration
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}
	slog.Info("Loaded configuration",
		"tenants", len(cfg.Applications.Tenants),
		"reverseProxies", len(cfg.Routes.ReverseProxies))

	// Setup logging format based on configuration
	setupLogging(cfg)

	// Write PID file
	if err := utils.WritePIDFile(config.NavigatorPIDFile); err != nil {
		slog.Error("Failed to write PID file", "error", err)
		os.Exit(1)
	}
	defer utils.RemovePIDFile(config.NavigatorPIDFile)

	// Create managers
	processManager := process.NewManager(cfg)
	appManager := process.NewAppManager(cfg)
	idleManager := idle.NewManager(cfg)

	// Load authentication if configured
	var basicAuth *auth.BasicAuth
	if cfg.Auth.Enabled && cfg.Auth.HTPasswd != "" {
		realm := cfg.Auth.Realm
		if realm == "" {
			realm = "Restricted" // Default realm
		}
		basicAuth, err = auth.LoadAuthFile(
			cfg.Auth.HTPasswd,
			realm,
			cfg.Auth.PublicPaths,
		)
		if err != nil {
			slog.Error("Failed to load auth file", "error", err)
			os.Exit(1)
		}
	}

	// Start managed processes
	if err := processManager.StartManagedProcesses(); err != nil {
		slog.Error("Failed to start managed processes", "error", err)
	}

	// Execute server start hooks
	if err := process.ExecuteServerHooks(cfg.Hooks.Start, "start"); err != nil {
		slog.Error("Failed to execute start hooks", "error", err)
		os.Exit(1)
	}

	// Create and run server lifecycle
	lifecycle := &ServerLifecycle{
		configFile:     configFile,
		cfg:            cfg,
		appManager:     appManager,
		processManager: processManager,
		basicAuth:      basicAuth,
		idleManager:    idleManager,
	}

	if err := lifecycle.Run(); err != nil {
		slog.Error("Server lifecycle failed", "error", err)
		os.Exit(1)
	}
}

func getLogLevel() slog.Level {
	logLevel := slog.LevelInfo
	if lvl := os.Getenv("LOG_LEVEL"); lvl != "" {
		switch strings.ToLower(lvl) {
		case "debug":
			logLevel = slog.LevelDebug
		case "info":
			logLevel = slog.LevelInfo
		case "warn", "warning":
			logLevel = slog.LevelWarn
		case "error":
			logLevel = slog.LevelError
		}
	}
	return logLevel
}

func initLogger() {
	opts := &slog.HandlerOptions{
		Level: getLogLevel(),
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, opts))
	slog.SetDefault(logger)
}

func setupLogging(cfg *config.Config) {
	// Check if JSON logging is configured
	if cfg.Logging.Format == "json" {
		// Switch to JSON handler
		opts := &slog.HandlerOptions{
			Level: getLogLevel(),
		}
		jsonLogger := slog.New(slog.NewJSONHandler(os.Stdout, opts))
		slog.SetDefault(jsonLogger)

		// Log the format switch (like the original navigator)
		slog.Info("Switched to JSON logging format")
	}
}

func handleCommandLineArgs() error {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-s":
			if len(os.Args) > 2 && os.Args[2] == "reload" {
				return utils.SendReloadSignal(config.NavigatorPIDFile)
			}
			return fmt.Errorf("option -s requires 'reload'")

		case "--help", "-h":
			printHelp()
			os.Exit(0)

		case "--version", "-v":
			if version != "dev" {
				fmt.Printf("Navigator %s\n", version)
			} else if commit != "none" {
				fmt.Printf("Navigator %s (commit: %s)\n", version, commit[:8])
			} else {
				fmt.Printf("Navigator %s (built: %s)\n", version, buildTime)
			}
			os.Exit(0)
		}
	}
	return nil
}

func printHelp() {
	fmt.Println("Navigator - Web application server")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  navigator [config-file]     Start server with optional config file")
	fmt.Println("  navigator -s reload         Reload configuration of running server")
	fmt.Println("  navigator --help            Show this help message")
	fmt.Println("  navigator --version         Show version information")
	fmt.Println()
	fmt.Println("Default config file: config/navigator.yml")
	fmt.Println()
	fmt.Println("Signals:")
	fmt.Println("  SIGHUP   Reload configuration without restart")
	fmt.Println("  SIGTERM  Graceful shutdown")
	fmt.Println("  SIGINT   Immediate shutdown")
}

// ServerLifecycle manages the HTTP server lifecycle and signal handling
type ServerLifecycle struct {
	configFile     string
	cfg            *config.Config
	appManager     *process.AppManager
	processManager *process.Manager
	basicAuth      *auth.BasicAuth
	idleManager    *idle.Manager
	srv            *http.Server
}

// Run starts the server and handles signals until shutdown
func (l *ServerLifecycle) Run() error {
	// Create HTTP handler
	handler := server.CreateHandler(l.cfg, l.appManager, l.basicAuth, l.idleManager)

	// Create HTTP server
	addr := fmt.Sprintf(":%s", l.cfg.Server.Listen)
	l.srv = &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)

	// Start server in goroutine
	serverErrors := make(chan error, 1)
	go func() {
		slog.Info("Navigator starting", "version", version, "address", addr)

		// Execute server ready hooks
		if err := process.ExecuteServerHooks(l.cfg.Hooks.Ready, "ready"); err != nil {
			slog.Error("Failed to execute ready hooks", "error", err)
		}

		serverErrors <- l.srv.ListenAndServe()
	}()

	// Handle signals and server errors
	for {
		select {
		case err := <-serverErrors:
			if err != http.ErrServerClosed {
				slog.Error("Server failed", "error", err)
				return err
			}
			return nil

		case sig := <-sigChan:
			switch sig {
			case syscall.SIGHUP:
				l.handleReload()

			case syscall.SIGTERM, syscall.SIGINT:
				return l.handleShutdown(sig)
			}
		}
	}
}

// handleReload reloads configuration without restarting the server
func (l *ServerLifecycle) handleReload() {
	slog.Info("Received SIGHUP, reloading configuration")

	// Load new configuration
	newConfig, err := config.LoadConfig(l.configFile)
	if err != nil {
		slog.Error("Failed to reload configuration", "error", err)
		return
	}

	// Update configuration in all managers
	l.appManager.UpdateConfig(newConfig)
	l.processManager.UpdateManagedProcesses(newConfig)
	l.idleManager.UpdateConfig(newConfig)

	// Update logging format if changed
	setupLogging(newConfig)

	// Replace config
	l.cfg = newConfig

	// Execute server start hooks BEFORE loading auth
	// This is important because hooks may update the htpasswd file
	if err := process.ExecuteServerHooks(newConfig.Hooks.Start, "start"); err != nil {
		slog.Error("Failed to execute start hooks after reload", "error", err)
	}

	// Reload auth if configured (AFTER hooks execute, since they may update htpasswd)
	if newConfig.Auth.Enabled && newConfig.Auth.HTPasswd != "" {
		realm := newConfig.Auth.Realm
		if realm == "" {
			realm = "Restricted"
		}
		newAuth, err := auth.LoadAuthFile(
			newConfig.Auth.HTPasswd,
			realm,
			newConfig.Auth.PublicPaths,
		)
		if err != nil {
			slog.Warn("Failed to reload auth file", "file", newConfig.Auth.HTPasswd, "error", err)
		} else {
			l.basicAuth = newAuth
			slog.Info("Reloaded authentication", "file", newConfig.Auth.HTPasswd)
		}
	} else {
		l.basicAuth = nil
	}

	// Update server handler if server is running (AFTER auth is loaded)
	if l.srv != nil {
		newHandler := server.CreateHandler(l.cfg, l.appManager, l.basicAuth, l.idleManager)
		l.srv.Handler = newHandler
	}

	slog.Info("Configuration reloaded successfully")
}

// handleShutdown performs graceful shutdown with context propagation
func (l *ServerLifecycle) handleShutdown(sig os.Signal) error {
	slog.Info("Received shutdown signal", "signal", sig)

	// Stop idle manager
	l.idleManager.Stop()

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown server
	if err := l.srv.Shutdown(ctx); err != nil {
		slog.Error("Server shutdown failed", "error", err)
	}

	// Stop all applications with context
	l.appManager.CleanupWithContext(ctx)

	// Stop managed processes with context
	l.processManager.StopManagedProcessesWithContext(ctx)

	slog.Info("Navigator shutdown complete")
	return nil
}
