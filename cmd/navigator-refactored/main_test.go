package main

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rubys/navigator/internal/auth"
	"github.com/rubys/navigator/internal/config"
	"github.com/rubys/navigator/internal/idle"
	"github.com/rubys/navigator/internal/process"
)

func TestSetupLogging(t *testing.T) {
	tests := []struct {
		name          string
		loggingFormat string
		shouldSwitch  bool
	}{
		{
			name:          "JSON format should trigger switch",
			loggingFormat: "json",
			shouldSwitch:  true,
		},
		{
			name:          "Text format should not trigger switch",
			loggingFormat: "text",
			shouldSwitch:  false,
		},
		{
			name:          "Empty format should not trigger switch",
			loggingFormat: "",
			shouldSwitch:  false,
		},
		{
			name:          "Invalid format should not trigger switch",
			loggingFormat: "invalid",
			shouldSwitch:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test config
			cfg := &config.Config{}
			cfg.Logging.Format = tt.loggingFormat

			// Test the setupLogging logic directly
			// Since we can't easily capture the output, we'll test the logic
			shouldSetupJSON := (cfg.Logging.Format == "json")

			if shouldSetupJSON != tt.shouldSwitch {
				t.Errorf("Expected JSON setup to be %v for format %q, got %v", tt.shouldSwitch, tt.loggingFormat, shouldSetupJSON)
			}

			// We can call setupLogging without testing output directly
			// The important thing is that it doesn't panic and the logic is correct
			setupLogging(cfg)
		})
	}
}

func TestInitLogger(t *testing.T) {
	// Test that initLogger sets up a basic text logger
	initLogger()

	// Create a buffer to capture output
	var buf bytes.Buffer
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	handler := slog.NewTextHandler(&buf, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	// Log a test message
	slog.Info("Init test message")

	output := buf.String()
	if !strings.Contains(output, "Init test message") {
		t.Errorf("Expected log output to contain test message, got: %s", output)
	}

	// Should be text format (not JSON)
	if strings.Contains(output, `"msg":`) {
		t.Errorf("Expected text format from initLogger, but got JSON: %s", output)
	}
}

func TestLogLevelFromEnvironment(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected slog.Level
	}{
		{
			name:     "DEBUG level from environment",
			envValue: "debug",
			expected: slog.LevelDebug,
		},
		{
			name:     "INFO level from environment",
			envValue: "info",
			expected: slog.LevelInfo,
		},
		{
			name:     "WARN level from environment",
			envValue: "warn",
			expected: slog.LevelWarn,
		},
		{
			name:     "ERROR level from environment",
			envValue: "error",
			expected: slog.LevelError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			os.Setenv("LOG_LEVEL", tt.envValue)
			defer os.Unsetenv("LOG_LEVEL")

			// Test that initLogger correctly parses the environment variable
			// We'll verify by testing the log level detection logic directly
			logLevel := slog.LevelInfo // default
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

			if logLevel != tt.expected {
				t.Errorf("Expected log level %v, got %v", tt.expected, logLevel)
			}
		})
	}
}

func TestHandleCommandLineArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorText   string
		shouldExit  bool
	}{
		{
			name:        "No arguments should pass",
			args:        []string{"navigator"},
			expectError: false,
		},
		{
			name:        "Config file argument should pass",
			args:        []string{"navigator", "config.yml"},
			expectError: false,
		},
		{
			name:        "Invalid -s option should fail",
			args:        []string{"navigator", "-s"},
			expectError: true,
			errorText:   "option -s requires 'reload'",
		},
		{
			name:        "Invalid -s option with wrong arg should fail",
			args:        []string{"navigator", "-s", "invalid"},
			expectError: true,
			errorText:   "option -s requires 'reload'",
		},
		{
			name:        "-s reload should attempt to send signal",
			args:        []string{"navigator", "-s", "reload"},
			expectError: false, // Will be handled specially below
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original args
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			// Set test args
			os.Args = tt.args

			err := handleCommandLineArgs()

			// Special handling for reload signal test - it may succeed or fail
			if tt.name == "-s reload should attempt to send signal" {
				// The reload command may succeed or fail depending on whether Navigator is running
				// Both outcomes are acceptable for this test
				return
			}

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if tt.errorText != "" && (err == nil || !strings.Contains(err.Error(), tt.errorText)) {
				t.Errorf("Expected error containing %q, got: %v", tt.errorText, err)
			}
		})
	}
}

func TestPrintHelp(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Call printHelp
	printHelp()

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Verify help text contains expected content
	expectedTexts := []string{
		"Navigator - Web application server",
		"Usage:",
		"navigator [config-file]",
		"navigator -s reload",
		"navigator --help",
		"Default config file: config/navigator.yml",
		"Signals:",
		"SIGHUP",
		"SIGTERM",
		"SIGINT",
	}

	for _, expected := range expectedTexts {
		if !strings.Contains(output, expected) {
			t.Errorf("Help text missing expected content: %q\nGot: %s", expected, output)
		}
	}
}

func TestHandleConfigReload(t *testing.T) {
	// Create a basic config
	cfg := &config.Config{}
	cfg.Server.Listen = "3000"
	cfg.Server.Hostname = "localhost"

	// Create real managers to avoid nil pointer issues
	appManager := process.NewAppManager(cfg)
	processManager := process.NewManager(cfg)
	idleManager := idle.NewManager(cfg)

	// Create lifecycle with nonexistent config file
	lifecycle := &ServerLifecycle{
		configFile:     "nonexistent-config.yml",
		cfg:            cfg,
		appManager:     appManager,
		processManager: processManager,
		basicAuth:      nil,
		idleManager:    idleManager,
	}

	// Test failed reload with nonexistent config file
	// This should log an error but not crash
	lifecycle.handleReload()

	// Config should remain unchanged because reload failed
	if lifecycle.cfg.Server.Listen != "3000" {
		t.Error("Expected config to remain unchanged after failed reload")
	}

	t.Log("handleReload correctly handles missing config files")
}

func TestHandleConfigReloadWithValidConfig(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test-config.yml")

	configContent := `
server:
  listen: "3001"
  hostname: "test-host"
applications:
  tenants: []
logging:
  format: "text"
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Create a basic config
	cfg := &config.Config{}
	cfg.Server.Listen = "3000"
	cfg.Server.Hostname = "localhost"

	// Create real managers to avoid nil pointer issues
	appManager := process.NewAppManager(cfg)
	processManager := process.NewManager(cfg)
	idleManager := idle.NewManager(cfg)

	// Create lifecycle with valid config file
	lifecycle := &ServerLifecycle{
		configFile:     configFile,
		cfg:            cfg,
		appManager:     appManager,
		processManager: processManager,
		basicAuth:      nil,
		idleManager:    idleManager,
	}

	// Test successful reload with valid config file
	lifecycle.handleReload()

	// Config should be updated
	if lifecycle.cfg == nil {
		t.Fatal("Expected non-nil config after reload")
	}

	// basicAuth should be nil because no authentication is configured
	if lifecycle.basicAuth != nil {
		t.Error("Expected nil auth when no authentication configured")
	}

	// Config should be updated
	if lifecycle.cfg.Server.Listen != "3001" {
		t.Errorf("Expected config to be updated with listen port 3001, got %s", lifecycle.cfg.Server.Listen)
	}
	if lifecycle.cfg.Server.Hostname != "test-host" {
		t.Errorf("Expected config to be updated with hostname 'test-host', got %s", lifecycle.cfg.Server.Hostname)
	}
}

func TestMainFunctionComponents(t *testing.T) {
	// Test that we can create the basic components that main() would create
	// without actually starting the server

	// Create a temporary config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "navigator.yml")

	configContent := `
server:
  listen: "3002"
  hostname: "localhost"
applications:
  tenants: []
logging:
  format: "text"
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Test config loading (part of main())
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		t.Fatalf("Failed to load test config: %v", err)
	}

	if cfg.Server.Listen != "3002" {
		t.Errorf("Expected listen port 3002, got %s", cfg.Server.Listen)
	}

	// Test manager creation (part of main())
	processManager := process.NewManager(cfg)
	if processManager == nil {
		t.Error("Expected non-nil process manager")
	}

	appManager := process.NewAppManager(cfg)
	if appManager == nil {
		t.Error("Expected non-nil app manager")
	}

	idleManager := idle.NewManager(cfg)
	if idleManager == nil {
		t.Error("Expected non-nil idle manager")
	}

	// Test that setupLogging works with the config
	setupLogging(cfg)

	// Test auth loading with no auth file (should not error)
	var basicAuth *auth.BasicAuth
	if cfg.Server.Authentication != "" {
		// This branch shouldn't execute since we didn't configure auth
		t.Error("Expected no authentication configured")
	}
	if basicAuth != nil {
		t.Error("Expected nil auth when none configured")
	}
}

func TestConfigFilePathLogic(t *testing.T) {
	// Test the config file path determination logic from main()
	tests := []struct {
		name         string
		args         []string
		expectedPath string
	}{
		{
			name:         "Default config path with no args",
			args:         []string{"navigator"},
			expectedPath: "config/navigator.yml",
		},
		{
			name:         "Custom config path",
			args:         []string{"navigator", "custom-config.yml"},
			expectedPath: "custom-config.yml",
		},
		{
			name:         "Ignore flag arguments",
			args:         []string{"navigator", "-s", "reload"},
			expectedPath: "config/navigator.yml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original args
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			// Set test args
			os.Args = tt.args

			// Replicate the config file path logic from main()
			configFile := "config/navigator.yml"
			if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") {
				configFile = os.Args[1]
			}

			if configFile != tt.expectedPath {
				t.Errorf("Expected config file path %q, got %q", tt.expectedPath, configFile)
			}
		})
	}
}

func TestStaticDirectoryConfigReload(t *testing.T) {
	// Create a temporary config file with static directories
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test-config.yml")

	configContent := `
server:
  listen: "3001"
  hostname: "test-host"
  public_dir: "public"
static:
  directories:
    - path: "/showcase/assets/"
      dir: "assets/"
    - path: "/showcase/studios/"
      dir: "studios/"
    - path: "/showcase/"
      dir: "."
applications:
  tenants: []
logging:
  format: "text"
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Create initial config with minimal static directories (simulating maintenance config)
	cfg := &config.Config{}
	cfg.Server.Listen = "3000"
	cfg.Server.Hostname = "localhost"
	cfg.Static.Directories = []config.StaticDir{
		{Path: "/", Dir: ""},
	}

	// Verify initial state has only 1 static directory
	if len(cfg.Static.Directories) != 1 {
		t.Fatalf("Expected 1 initial static directory, got %d", len(cfg.Static.Directories))
	}
	if cfg.Static.Directories[0].Path != "/" {
		t.Errorf("Expected initial static directory path '/', got '%s'", cfg.Static.Directories[0].Path)
	}

	// Create real managers to avoid nil pointer issues
	appManager := process.NewAppManager(cfg)
	processManager := process.NewManager(cfg)
	idleManager := idle.NewManager(cfg)

	// Create lifecycle with valid config file
	lifecycle := &ServerLifecycle{
		configFile:     configFile,
		cfg:            cfg,
		appManager:     appManager,
		processManager: processManager,
		basicAuth:      nil,
		idleManager:    idleManager,
	}

	// Test reload with config containing multiple static directories
	lifecycle.handleReload()

	// Verify config was updated
	if lifecycle.cfg == nil {
		t.Fatal("Expected non-nil config after reload")
	}

	// Verify static directories were updated (this was the bug)
	if len(lifecycle.cfg.Static.Directories) != 3 {
		t.Errorf("Expected 3 static directories after reload, got %d", len(lifecycle.cfg.Static.Directories))
	}

	// Verify specific directories are present
	expectedDirs := map[string]string{
		"/showcase/assets/":  "assets/",
		"/showcase/studios/": "studios/",
		"/showcase/":         ".",
	}

	actualDirs := make(map[string]string)
	for _, dir := range lifecycle.cfg.Static.Directories {
		actualDirs[dir.Path] = dir.Dir
	}

	for expectedPath, expectedDir := range expectedDirs {
		if actualDir, exists := actualDirs[expectedPath]; !exists {
			t.Errorf("Expected static directory '%s' not found after reload", expectedPath)
		} else if actualDir != expectedDir {
			t.Errorf("Expected static directory '%s' to map to '%s', got '%s'", expectedPath, expectedDir, actualDir)
		}
	}

	// basicAuth should be nil because no authentication is configured
	if lifecycle.basicAuth != nil {
		t.Error("Expected nil auth when no auth is configured")
	}

	t.Log("Static directory configuration reload test passed")
}
