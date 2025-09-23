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
		"locations", len(cfg.Locations),
		"tenants", len(cfg.Applications.Tenants),
		"standaloneServers", len(cfg.StandaloneServers))

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
	if cfg.Server.Authentication != "" {
		basicAuth, err = auth.LoadAuthFile(
			cfg.Server.Authentication,
			"Restricted", // Default realm
			cfg.Server.AuthExclude,
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

	// Create HTTP handler with the new server package
	handler := server.CreateHandler(cfg, appManager, basicAuth, idleManager)

	// Create HTTP server
	addr := fmt.Sprintf(":%s", cfg.Server.Listen)
	srv := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)

	// Start server in goroutine
	serverErrors := make(chan error, 1)
	go func() {
		slog.Info("Navigator starting", "address", addr)

		// Execute server ready hooks
		process.ExecuteServerHooks(cfg.Hooks.Ready, "ready")

		serverErrors <- srv.ListenAndServe()
	}()

	// Handle signals and server errors
	for {
		select {
		case err := <-serverErrors:
			if err != http.ErrServerClosed {
				slog.Error("Server failed", "error", err)
			}
			return

		case sig := <-sigChan:
			switch sig {
			case syscall.SIGHUP:
				slog.Info("Received SIGHUP, reloading configuration")
				newBasicAuth, reloadSuccess := handleConfigReload(cfg, configFile, appManager, processManager, basicAuth, idleManager)
				if reloadSuccess {
					// Update handler after successful config reload (auth may have changed)
					basicAuth = newBasicAuth
					newHandler := server.CreateHandler(cfg, appManager, basicAuth, idleManager)
					srv.Handler = newHandler
				}
				// Continue the loop to keep the server running

			case syscall.SIGTERM, syscall.SIGINT:
				slog.Info("Received shutdown signal", "signal", sig)

				// Stop idle manager
				idleManager.Stop()

				// Graceful shutdown
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				// Shutdown server
				if err := srv.Shutdown(ctx); err != nil {
					slog.Error("Server shutdown failed", "error", err)
				}

				// Stop all applications
				appManager.Cleanup()

				// Stop managed processes
				processManager.StopManagedProcesses()

				// Log shutdown completion and exit
				slog.Info("Navigator shutdown complete")
				return
			}
		}
	}
}

func initLogger() {
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

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, opts))
	slog.SetDefault(logger)
}

func setupLogging(cfg *config.Config) {
	// Check if JSON logging is configured
	if cfg.Logging.Format == "json" {
		// Get current log level from existing logger
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

		// Switch to JSON handler
		opts := &slog.HandlerOptions{
			Level: logLevel,
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
			fmt.Println("Navigator v1.0.0")
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
	fmt.Println()
	fmt.Println("Default config file: config/navigator.yml")
	fmt.Println()
	fmt.Println("Signals:")
	fmt.Println("  SIGHUP   Reload configuration without restart")
	fmt.Println("  SIGTERM  Graceful shutdown")
	fmt.Println("  SIGINT   Immediate shutdown")
}

func handleConfigReload(oldConfig *config.Config, configFile string, appManager *process.AppManager, processManager *process.Manager, currentAuth *auth.BasicAuth, idleManager *idle.Manager) (*auth.BasicAuth, bool) {
	// Reload configuration
	newConfig, err := config.LoadConfig(configFile)
	if err != nil {
		slog.Error("Failed to reload configuration", "error", err)
		return nil, false
	}

	// Update configuration in managers
	appManager.UpdateConfig(newConfig)
	processManager.UpdateManagedProcesses(newConfig)

	// Reload auth if configured
	var newAuth *auth.BasicAuth
	if newConfig.Server.Authentication != "" {
		newAuth, err = auth.LoadAuthFile(
			newConfig.Server.Authentication,
			"Restricted", // Default realm
			newConfig.Server.AuthExclude,
		)
		if err != nil {
			slog.Warn("Failed to reload auth file", "file", newConfig.Server.Authentication, "error", err)
			// Keep existing auth on error
			newAuth = currentAuth
		} else {
			slog.Info("Reloaded authentication", "file", newConfig.Server.Authentication)
		}
	} else {
		// Auth disabled in new config
		newAuth = nil
	}

	// Update the main config
	config.UpdateConfig(oldConfig, newConfig)

	// Update logging format if changed
	setupLogging(newConfig)

	slog.Info("Configuration reloaded successfully")
	return newAuth, true
}