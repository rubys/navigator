package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary test config file
	testConfig := `
server:
  listen: 3000
  hostname: localhost
  public_dir: public
  idle:
    action: suspend
    timeout: 20m

applications:
  pools:
    max_size: 10
    timeout: 5m
    start_port: 4000
  tenants:
    - name: test-app
      root: /tmp/test-app
      env:
        PORT: "${PORT}"
        RAILS_ENV: development
`

	// Write test config to temporary file
	tmpFile, err := os.CreateTemp("", "navigator-test-*.yml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(testConfig); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}
	tmpFile.Close()

	// Test loading the config
	config, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify basic server config
	if config.Server.Listen != "3000" {
		t.Errorf("Expected listen port 3000, got %s", config.Server.Listen)
	}

	if config.Server.Hostname != "localhost" {
		t.Errorf("Expected hostname localhost, got %s", config.Server.Hostname)
	}

	if config.Server.PublicDir != "public" {
		t.Errorf("Expected public_dir public, got %s", config.Server.PublicDir)
	}

	// Verify idle config
	if config.Server.Idle.Action != "suspend" {
		t.Errorf("Expected idle action suspend, got %s", config.Server.Idle.Action)
	}

	if config.Server.Idle.Timeout != "20m" {
		t.Errorf("Expected idle timeout 20m, got %s", config.Server.Idle.Timeout)
	}

	// Verify applications config
	if config.Applications.Pools.MaxSize != 10 {
		t.Errorf("Expected max_size 10, got %d", config.Applications.Pools.MaxSize)
	}

	if len(config.Applications.Tenants) != 1 {
		t.Errorf("Expected 1 tenant, got %d", len(config.Applications.Tenants))
	}

	if config.Applications.Tenants[0].Name != "test-app" {
		t.Errorf("Expected tenant name test-app, got %s", config.Applications.Tenants[0].Name)
	}
}

func TestParseYAMLWithVariableSubstitution(t *testing.T) {
	testConfig := `
applications:
  env:
    DATABASE_URL: "postgres://user:pass@localhost/app_${tenant}"
    SECRET_KEY: "${secret}"
  tenants:
    - name: "test-tenant"
      var:
        tenant: "test"
        secret: "test-secret-123"
`

	tmpFile, err := os.CreateTemp("", "navigator-var-test-*.yml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(testConfig); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}
	tmpFile.Close()

	config, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Variable substitution should happen during tenant processing
	// This test validates the config structure is correct
	if len(config.Applications.Tenants) != 1 {
		t.Errorf("Expected 1 tenant, got %d", len(config.Applications.Tenants))
	}

	tenant := config.Applications.Tenants[0]
	if tenant.Name != "test-tenant" {
		t.Errorf("Expected tenant name test-tenant, got %s", tenant.Name)
	}
}

func TestInvalidConfigFile(t *testing.T) {
	// Test with non-existent file
	_, err := LoadConfig("non-existent-file.yml")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}

	// Test with invalid YAML
	invalidConfig := `
server:
  listen: 3000
  invalid_yaml: [unclosed
`

	tmpFile, err := os.CreateTemp("", "navigator-invalid-*.yml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(invalidConfig); err != nil {
		t.Fatalf("Failed to write invalid config: %v", err)
	}
	tmpFile.Close()

	_, err = LoadConfig(tmpFile.Name())
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}

func TestDefaultValues(t *testing.T) {
	// Test with minimal config
	minimalConfig := `
server:
  listen: 3000
`

	tmpFile, err := os.CreateTemp("", "navigator-minimal-*.yml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(minimalConfig); err != nil {
		t.Fatalf("Failed to write minimal config: %v", err)
	}
	tmpFile.Close()

	config, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load minimal config: %v", err)
	}

	// Check that default values are set appropriately
	if config.Server.Listen != "3000" {
		t.Errorf("Expected listen port 3000, got %s", config.Server.Listen)
	}

	// Public dir should have a reasonable default or be empty
	// (depending on implementation)
	if config.Server.PublicDir == "" {
		t.Log("Public dir is empty - this may be expected behavior")
	}
}

func TestStaticConfigWithDurations(t *testing.T) {
	// Test config with static directories and cache durations
	testConfig := `
server:
  listen: 3000
  hostname: localhost
  public_dir: public

static:
  directories:
    - path: "/assets/"
      root: "assets/"
      cache: "24h"
    - path: "/docs/"
      root: "docs/"
      cache: "1h"
    - path: "/images/"
      root: "images/"
      cache: "30m"
    - path: "/temp/"
      root: "temp/"
      cache: ""  # Empty string for no cache
  extensions:
    - html
    - css
    - js
  try_files:
    enabled: true
    suffixes:
      - ".html"
      - ".htm"
    fallback: "/404.html"

routes:
  rewrites:
    - from: "^/old-path$"
      to: "/new-path"
    - from: "^/maintenance$"
      to: "/503.html"

applications:
  pools:
    max_size: 5
    timeout: "10m"
    start_port: 4000
  tenants: []
`

	tmpFile, err := os.CreateTemp("", "navigator-static-test-*.yml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(testConfig); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}
	tmpFile.Close()

	config, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify static directories are parsed correctly
	if len(config.Static.Directories) != 4 {
		t.Errorf("Expected 4 static directories, got %d", len(config.Static.Directories))
	}

	// Check first directory with 24h cache
	dir1 := config.Static.Directories[0]
	if dir1.Path != "/assets/" {
		t.Errorf("Expected path /assets/, got %s", dir1.Path)
	}
	if dir1.Prefix != "assets/" {
		t.Errorf("Expected prefix assets/, got %s", dir1.Prefix)
	}
	if dir1.Cache != "24h" {
		t.Errorf("Expected cache 24h, got %s", dir1.Cache)
	}

	// Check second directory with 1h cache
	dir2 := config.Static.Directories[1]
	if dir2.Cache != "1h" {
		t.Errorf("Expected cache 1h, got %s", dir2.Cache)
	}

	// Check third directory with 30m cache
	dir3 := config.Static.Directories[2]
	if dir3.Cache != "30m" {
		t.Errorf("Expected cache 30m, got %s", dir3.Cache)
	}

	// Check fourth directory with empty cache
	dir4 := config.Static.Directories[3]
	if dir4.Cache != "" {
		t.Errorf("Expected empty cache, got %s", dir4.Cache)
	}

	// Verify static extensions
	expectedExtensions := []string{"html", "css", "js"}
	if len(config.Static.Extensions) != len(expectedExtensions) {
		t.Errorf("Expected %d extensions, got %d", len(expectedExtensions), len(config.Static.Extensions))
	}
	for i, ext := range expectedExtensions {
		if i >= len(config.Static.Extensions) || config.Static.Extensions[i] != ext {
			t.Errorf("Expected extension %s at index %d, got %s", ext, i, config.Static.Extensions[i])
		}
	}

	// Verify try_files configuration
	if !config.Static.TryFiles.Enabled {
		t.Error("Expected try_files to be enabled")
	}
	if config.Static.TryFiles.Fallback != "/404.html" {
		t.Errorf("Expected fallback /404.html, got %s", config.Static.TryFiles.Fallback)
	}

	// Verify routes configuration
	if len(config.Routes.Rewrites) != 2 {
		t.Errorf("Expected 2 rewrites, got %d", len(config.Routes.Rewrites))
	}

	// Verify timeout strings are preserved
	if config.Applications.Pools.Timeout != "10m" {
		t.Errorf("Expected pools timeout 10m, got %s", config.Applications.Pools.Timeout)
	}
}

func TestMaintenanceModeConfig(t *testing.T) {
	// Test maintenance mode configuration
	maintenanceConfig := `
server:
  listen: 3000
  hostname: localhost
  root_path: /
  public_dir: public

auth:
  enabled: false

applications:
  tenants: []

static:
  directories:
    - path: /
      root: .
      cache: 0
  extensions:
    - html
    - css
    - js
    - png
    - jpg
    - svg
    - ico
  try_files:
    enabled: true
    suffixes: []
    fallback: /503.html

routes:
  rewrites:
    - from: "^.*$"
      to: /503.html

pools:
  max_size: 1
  idle_timeout: 60
  start_port: 4000
`

	tmpFile, err := os.CreateTemp("", "navigator-maintenance-test-*.yml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(maintenanceConfig); err != nil {
		t.Fatalf("Failed to write maintenance config: %v", err)
	}
	tmpFile.Close()

	config, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load maintenance config: %v", err)
	}

	// Verify maintenance mode characteristics
	if len(config.Applications.Tenants) != 0 {
		t.Errorf("Expected 0 tenants for maintenance mode, got %d", len(config.Applications.Tenants))
	}

	// Verify static fallback is configured
	if config.Static.TryFiles.Fallback != "/503.html" {
		t.Errorf("Expected fallback /503.html, got %s", config.Static.TryFiles.Fallback)
	}

	// Verify rewrite rules are present
	if len(config.Routes.Rewrites) != 1 {
		t.Errorf("Expected 1 rewrite rule, got %d", len(config.Routes.Rewrites))
	}

	if len(config.Routes.Rewrites) > 0 {
		rewrite := config.Routes.Rewrites[0]
		if rewrite.From != "^.*$" {
			t.Errorf("Expected rewrite from ^.*$, got %s", rewrite.From)
		}
		if rewrite.To != "/503.html" {
			t.Errorf("Expected rewrite to /503.html, got %s", rewrite.To)
		}
	}

	// Verify static directories have cache values as strings
	if len(config.Static.Directories) > 0 {
		dir := config.Static.Directories[0]
		// Cache should be "0" (string) not 0 (int)
		if dir.Cache != "0" {
			t.Errorf("Expected cache as string '0', got %v (%T)", dir.Cache, dir.Cache)
		}
	}
}

func TestAllDurationsAsStrings(t *testing.T) {
	// Test configuration with all possible duration fields to ensure consistency
	testConfig := `
server:
  listen: 3000
  hostname: localhost
  public_dir: public
  idle:
    action: suspend
    timeout: "20m"
  sticky_sessions:
    enabled: true
    cookie_name: "_navigator_session"
    cookie_max_age: "2h"
    cookie_secure: true

static:
  directories:
    - path: "/assets/"
      root: "assets/"
      cache: "24h"
    - path: "/temp/"
      root: "temp/"
      cache: "0"  # String zero
  extensions:
    - html
    - css

applications:
  pools:
    max_size: 10
    timeout: "5m"
    start_port: 4000
  tenants:
    - name: test-app
      root: /tmp/test

managed_processes:
  - name: redis
    command: redis-server
    start_delay: "2s"
  - name: worker
    command: background-worker
    start_delay: "5s"

hooks:
  start:
    - command: /bin/echo
      args: ["starting"]
      timeout: "10s"
  ready:
    - command: /bin/echo
      args: ["ready"]
      timeout: "15s"
`

	tmpFile, err := os.CreateTemp("", "navigator-durations-test-*.yml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(testConfig); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}
	tmpFile.Close()

	config, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config with all durations: %v", err)
	}

	// Verify all duration fields are preserved as strings

	// Server idle timeout
	if config.Server.Idle.Timeout != "20m" {
		t.Errorf("Expected server idle timeout '20m', got '%s'", config.Server.Idle.Timeout)
	}

	// Sticky session cookie max age
	if config.Server.StickySession.CookieMaxAge != "2h" {
		t.Errorf("Expected cookie max age '2h', got '%s'", config.Server.StickySession.CookieMaxAge)
	}

	// Static cache durations
	if len(config.Static.Directories) >= 1 {
		if config.Static.Directories[0].Cache != "24h" {
			t.Errorf("Expected static cache '24h', got '%s'", config.Static.Directories[0].Cache)
		}
	}
	if len(config.Static.Directories) >= 2 {
		if config.Static.Directories[1].Cache != "0" {
			t.Errorf("Expected static cache '0', got '%s'", config.Static.Directories[1].Cache)
		}
	}

	// Application pool timeout
	if config.Applications.Pools.Timeout != "5m" {
		t.Errorf("Expected pools timeout '5m', got '%s'", config.Applications.Pools.Timeout)
	}

	// Managed process start delays
	if len(config.ManagedProcesses) >= 2 {
		redis := config.ManagedProcesses[0]
		if redis.StartDelay != "2s" {
			t.Errorf("Expected redis start_delay '2s', got '%s'", redis.StartDelay)
		}

		worker := config.ManagedProcesses[1]
		if worker.StartDelay != "5s" {
			t.Errorf("Expected worker start_delay '5s', got '%s'", worker.StartDelay)
		}
	}

	// Hook timeouts
	if len(config.Hooks.Start) >= 1 {
		if config.Hooks.Start[0].Timeout != "10s" {
			t.Errorf("Expected start hook timeout '10s', got '%s'", config.Hooks.Start[0].Timeout)
		}
	}
	if len(config.Hooks.Ready) >= 1 {
		if config.Hooks.Ready[0].Timeout != "15s" {
			t.Errorf("Expected ready hook timeout '15s', got '%s'", config.Hooks.Ready[0].Timeout)
		}
	}
}

func BenchmarkLoadConfig(b *testing.B) {
	testConfig := `
server:
  listen: 3000
  hostname: localhost
  public_dir: public
  idle:
    action: suspend
    timeout: 20m

applications:
  pools:
    max_size: 10
    timeout: 5m
    start_port: 4000
  tenants:
    - name: test-app-1
      root: /tmp/test-app-1
    - name: test-app-2
      root: /tmp/test-app-2
    - name: test-app-3
      root: /tmp/test-app-3
`

	tmpFile, err := os.CreateTemp("", "navigator-bench-*.yml")
	if err != nil {
		b.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(testConfig); err != nil {
		b.Fatalf("Failed to write test config: %v", err)
	}
	tmpFile.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := LoadConfig(tmpFile.Name())
		if err != nil {
			b.Fatalf("Failed to load config: %v", err)
		}
	}
}