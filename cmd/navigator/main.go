package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/rubys/navigator/internal/config"
	"github.com/rubys/navigator/internal/logger"
	"github.com/rubys/navigator/internal/manager"
	"github.com/rubys/navigator/internal/proxy"
	"github.com/rubys/navigator/internal/server"
)

func main() {
	// Command-line flags
	var (
		railsRoot    = flag.String("rails-root", "", "Rails application root directory (required)")
		listenAddr   = flag.String("listen", ":3000", "Address to listen on")
		showcasesFile = flag.String("showcases", "config/tenant/showcases.yml", "Path to showcases.yml relative to rails-root")
		dbPath       = flag.String("db-path", "db", "Database directory path")
		storagePath  = flag.String("storage", "storage", "Storage directory path")
		htpasswdFile = flag.String("htpasswd", "", "Path to htpasswd file")
		urlPrefix    = flag.String("url-prefix", "/showcase", "URL prefix to strip from requests")
		maxPuma      = flag.Int("max-puma", 10, "Maximum number of concurrent Puma processes")
		idleTimeout  = flag.Duration("idle-timeout", 5*time.Minute, "Idle timeout before stopping Puma process")
		logLevel     = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	)
	
	flag.Parse()

	// Initialize structured logging
	logger.Init(*logLevel)

	// Validate required flags
	if *railsRoot == "" {
		logger.Fatal("Error: -rails-root is required")
	}

	// Resolve absolute paths
	absRailsRoot, err := filepath.Abs(*railsRoot)
	if err != nil {
		logger.Fatalf("Error resolving rails-root path: %v", err)
	}

	// Load showcases configuration
	showcasesPath := filepath.Join(absRailsRoot, *showcasesFile)
	logger.WithField("path", showcasesPath).Info("Loading showcases configuration")
	showcases, err := config.LoadShowcases(showcasesPath)
	if err != nil {
		logger.WithField("path", showcasesPath).Fatalf("Failed to load showcases: %v", err)
	}

	// Set up absolute paths
	if !filepath.IsAbs(*dbPath) {
		*dbPath = filepath.Join(absRailsRoot, *dbPath)
	}
	if !filepath.IsAbs(*storagePath) {
		*storagePath = filepath.Join(absRailsRoot, *storagePath)
	}
	if *htpasswdFile != "" && !filepath.IsAbs(*htpasswdFile) {
		*htpasswdFile = filepath.Join(absRailsRoot, *htpasswdFile)
	}

	// Create Puma manager
	pumaManager := manager.NewPumaManager(manager.Config{
		RailsRoot:    absRailsRoot,
		DbPath:       *dbPath,
		StoragePath:  *storagePath,
		MaxProcesses: *maxPuma,
		IdleTimeout:  *idleTimeout,
		Region:       os.Getenv("FLY_REGION"),
		AppName:      os.Getenv("FLY_APP_NAME"),
	})

	// Create router using chi
	router := proxy.NewRouter(proxy.RouterConfig{
		Manager:      pumaManager,
		Showcases:    showcases,
		RailsRoot:    absRailsRoot,
		DbPath:       *dbPath,
		StoragePath:  *storagePath,
		URLPrefix:    *urlPrefix,
		HtpasswdFile: *htpasswdFile,
	})

	// Create and start server
	srv := server.NewChiServer(router, *listenAddr)
	
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
		"listen_addr": *listenAddr,
		"rails_root": absRailsRoot,
		"db_path": *dbPath,
		"storage_path": *storagePath,
		"url_prefix": *urlPrefix,
		"tenants": len(showcases.GetAllTenants()),
		"log_level": *logLevel,
	}).Info("Starting Navigator server")
	
	if err := srv.Start(); err != nil {
		logger.WithField("error", err).Fatal("Server failed to start")
	}
}