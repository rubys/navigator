package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	showcasesConfig "github.com/rubys/navigator/internal/config"
	"github.com/rubys/navigator/internal/logger"
	"github.com/rubys/navigator/internal/manager"
	"github.com/rubys/navigator/internal/proxy"
	"github.com/rubys/navigator/internal/server"
	"github.com/spf13/cobra"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the Navigator server",
	Long: `Start the Navigator server to serve multi-tenant Rails applications.

The server will:
  • Load tenant configuration from showcases.yml
  • Start the HTTP/2 server with caching and compression
  • Manage Puma processes dynamically based on demand
  • Handle authentication if htpasswd file is configured
  • Provide automatic process recovery on failures

Configuration is loaded from (in order of precedence):
  • Command-line flags
  • Environment variables (NAVIGATOR_*)
  • Config file (auto-loads config/navigator.yml if present)

Examples:
  navigator serve                          # Uses current directory, auto-loads config/navigator.yml
  navigator serve --root /path/to/app      # Specify different root directory
  navigator serve --config custom.yaml     # Use custom config file
  navigator serve --listen :8080 --log-level debug`,
	Run: func(cmd *cobra.Command, args []string) {
		// Load configuration - flag binding should work with global Viper instance
		cfg, err := LoadConfig(cfgFile)
		if err != nil {
			fmt.Printf("Error loading configuration: %v\n", err)
			os.Exit(1)
		}
		runServer(cfg)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	// No longer marking root as required since it has a default value
}

// runServer starts the Navigator server with the given configuration
func runServer(cfg *Config) {
	// Initialize structured logging
	logger.Init(cfg.Logging.Level)

	// Load showcases configuration
	showcasesPath := cfg.GetShowcasesPath()
	logger.WithField("path", showcasesPath).Info("Loading showcases configuration")
	showcases, err := showcasesConfig.LoadShowcases(showcasesPath)
	if err != nil {
		logger.WithField("path", showcasesPath).Fatalf("Failed to load showcases: %v", err)
	}

	// Create Puma manager
	pumaManager := manager.NewPumaManager(manager.Config{
		RailsRoot:    cfg.Rails.Root,
		DbPath:       cfg.Rails.DbPath,
		StoragePath:  cfg.Rails.Storage,
		MaxProcesses: cfg.Manager.MaxPuma,
		IdleTimeout:  cfg.Manager.IdleTimeout,
		Region:       os.Getenv("FLY_REGION"),
		AppName:      os.Getenv("FLY_APP_NAME"),
	})

	// Create router using chi
	router := proxy.NewRouter(proxy.RouterConfig{
		Manager:      pumaManager,
		Showcases:    showcases,
		RailsRoot:    cfg.Rails.Root,
		DbPath:       cfg.Rails.DbPath,
		StoragePath:  cfg.Rails.Storage,
		URLPrefix:    cfg.Server.URLPrefix,
		HtpasswdFile: cfg.Auth.HtpasswdFile,
	})

	// Create and start server
	srv := server.NewChiServer(router, cfg.Server.Listen)

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		logger.Info("Shutdown signal received")

		// Create shutdown context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Shutdown server
		if err := srv.Shutdown(ctx); err != nil {
			logger.WithField("error", err).Error("Server shutdown error")
		}

		// Stop all Puma processes
		logger.Info("Stopping all Puma processes")
		pumaManager.StopAll()

		os.Exit(0)
	}()

	// Start server
	logger.WithFields(map[string]interface{}{
		"listen_addr":  cfg.Server.Listen,
		"rails_root":   cfg.Rails.Root,
		"db_path":      cfg.Rails.DbPath,
		"storage_path": cfg.Rails.Storage,
		"url_prefix":   cfg.Server.URLPrefix,
		"tenants":      len(showcases.GetAllTenants()),
		"log_level":    cfg.Logging.Level,
		"max_puma":     cfg.Manager.MaxPuma,
		"idle_timeout": cfg.Manager.IdleTimeout,
	}).Info("Starting Navigator server")

	if err := srv.Start(); err != nil {
		logger.WithField("error", err).Fatal("Server failed to start")
	}
}
