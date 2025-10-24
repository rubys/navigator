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
  root_path: /showcase
  static:
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
    - path: /showcase/test-app/
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

	if config.Server.Static.PublicDir != "public" {
		t.Errorf("Expected public_dir public, got %s", config.Server.Static.PublicDir)
	}

	// RootPath should be normalized with trailing slash
	if config.Server.RootPath != "/showcase/" {
		t.Errorf("Expected root_path /showcase/, got %s", config.Server.RootPath)
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
    - path: "/showcase/test-tenant/"
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

func TestAuthSectionParsing(t *testing.T) {
	testConfig := `
auth:
  enabled: true
  realm: "Test Realm"
  htpasswd: "/path/to/htpasswd"
  public_paths:
    - "/showcase/studios/"
    - "/showcase/docs/"
    - "*.css"
    - "*.js"
    - "/server/public/"
  exclude_patterns:
    - pattern: "^/showcase/?$"
      description: "Root showcase path"

server:
  listen: 3000
  hostname: localhost

applications:
  pools:
    max_size: 10
    timeout: 5m
    start_port: 4000
`

	tmpFile, err := os.CreateTemp("", "navigator-auth-test-*.yml")
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

	// Verify auth section sets public_paths in Auth.PublicPaths
	expectedPublicPaths := []string{"/showcase/studios/", "/showcase/docs/", "*.css", "*.js", "/server/public/"}
	if len(config.Auth.PublicPaths) != len(expectedPublicPaths) {
		t.Errorf("Expected %d auth public paths, got %d", len(expectedPublicPaths), len(config.Auth.PublicPaths))
	}

	for i, expected := range expectedPublicPaths {
		if i >= len(config.Auth.PublicPaths) || config.Auth.PublicPaths[i] != expected {
			t.Errorf("Expected auth public_paths[%d] to be %s, got %s", i, expected, config.Auth.PublicPaths[i])
		}
	}

	// Verify authentication file path is set from auth section
	if config.Auth.HTPasswd != "/path/to/htpasswd" {
		t.Errorf("Expected authentication file /path/to/htpasswd, got %s", config.Auth.HTPasswd)
	}
}

func TestAuthSectionDisabled(t *testing.T) {
	testConfig := `
auth:
  enabled: false
  public_paths:
    - "/should/not/be/used/"

server:
  listen: 3000

applications:
  pools:
    max_size: 10
`

	tmpFile, err := os.CreateTemp("", "navigator-auth-disabled-test-*.yml")
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

	// When auth is disabled, auth public paths should be empty
	if len(config.Auth.PublicPaths) != 0 {
		t.Errorf("Expected auth public_paths to be empty when auth disabled, got %v", config.Auth.PublicPaths)
	}
}

func TestServerConfigWithTryFiles(t *testing.T) {
	testConfig := `
server:
  listen: 3000
  static:
    public_dir: /Users/test/public
    try_files:
      - "index.html"
      - ".html"
      - ".htm"
    allowed_extensions:
      - html
      - css
      - js

applications:
  pools:
    max_size: 10
`

	tmpFile, err := os.CreateTemp("", "navigator-server-test-*.yml")
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

	// Verify try_files configuration at server level
	expectedTryFiles := []string{"index.html", ".html", ".htm"}
	if len(config.Server.Static.TryFiles) != len(expectedTryFiles) {
		t.Errorf("Expected %d try_files, got %d", len(expectedTryFiles), len(config.Server.Static.TryFiles))
	}

	for i, expected := range expectedTryFiles {
		if i >= len(config.Server.Static.TryFiles) || config.Server.Static.TryFiles[i] != expected {
			t.Errorf("Expected try_files[%d] to be %s, got %s", i, expected, config.Server.Static.TryFiles[i])
		}
	}

	// Verify allowed extensions at server level
	expectedExtensions := []string{"html", "css", "js"}
	if len(config.Server.Static.AllowedExtensions) != len(expectedExtensions) {
		t.Errorf("Expected %d extensions, got %d", len(expectedExtensions), len(config.Server.Static.AllowedExtensions))
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
	if config.Server.Static.PublicDir == "" {
		t.Log("Public dir is empty - this may be expected behavior")
	}
}

func TestServerConfigWithCacheControl(t *testing.T) {
	// Test config with server-level cache control
	testConfig := `
server:
  listen: 3000
  hostname: localhost
  static:
    public_dir: public
    cache_control:
      default: "public, max-age=86400"
      overrides:
        - path: "/assets/"
          max_age: "public, max-age=86400"
        - path: "/docs/"
          max_age: "public, max-age=3600"
        - path: "/images/"
          max_age: "public, max-age=1800"
        - path: "/temp/"
          max_age: "no-cache"
    allowed_extensions:
      - html
      - css
      - js
    try_files:
      - ".html"
      - ".htm"

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

	tmpFile, err := os.CreateTemp("", "navigator-cache-test-*.yml")
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

	// Verify cache control configuration
	if config.Server.Static.CacheControl.Default != "public, max-age=86400" {
		t.Errorf("Expected default cache control 'public, max-age=86400', got %s", config.Server.Static.CacheControl.Default)
	}

	if len(config.Server.Static.CacheControl.Overrides) != 4 {
		t.Errorf("Expected 4 cache overrides, got %d", len(config.Server.Static.CacheControl.Overrides))
	}

	// Verify allowed extensions at server level
	expectedExtensions := []string{"html", "css", "js"}
	if len(config.Server.Static.AllowedExtensions) != len(expectedExtensions) {
		t.Errorf("Expected %d extensions, got %d", len(expectedExtensions), len(config.Server.Static.AllowedExtensions))
	}
	for i, ext := range expectedExtensions {
		if i >= len(config.Server.Static.AllowedExtensions) || config.Server.Static.AllowedExtensions[i] != ext {
			t.Errorf("Expected extension %s at index %d, got %s", ext, i, config.Server.Static.AllowedExtensions[i])
		}
	}

	// Verify try_files configuration at server level
	expectedTryFiles := []string{".html", ".htm"}
	if len(config.Server.Static.TryFiles) != len(expectedTryFiles) {
		t.Errorf("Expected %d try_files, got %d", len(expectedTryFiles), len(config.Server.Static.TryFiles))
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
  static:
    public_dir: public
    cache_control:
      default: "no-cache"
    allowed_extensions:
      - html
      - css
      - js
      - png
      - jpg
      - svg
      - ico
    try_files: []

auth:
  enabled: false

applications:
  tenants: []

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

	// Verify server configuration
	if config.Server.Listen != "3000" {
		t.Errorf("Expected listen port 3000, got %s", config.Server.Listen)
	}

	if config.Server.Hostname != "localhost" {
		t.Errorf("Expected hostname localhost, got %s", config.Server.Hostname)
	}

	// RootPath should be correctly parsed from YAML
	if config.Server.RootPath != "/" {
		t.Errorf("Expected root_path /, got %s", config.Server.RootPath)
	}

	if config.Server.Static.PublicDir != "public" {
		t.Errorf("Expected public_dir public, got %s", config.Server.Static.PublicDir)
	}

	// Verify maintenance mode characteristics
	if len(config.Applications.Tenants) != 0 {
		t.Errorf("Expected 0 tenants for maintenance mode, got %d", len(config.Applications.Tenants))
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

	// Verify cache control is set to no-cache
	if config.Server.Static.CacheControl.Default != "no-cache" {
		t.Errorf("Expected cache control 'no-cache', got %s", config.Server.Static.CacheControl.Default)
	}
}

func TestAllDurationsAsStrings(t *testing.T) {
	// Test configuration with all possible duration fields to ensure consistency
	testConfig := `
server:
  listen: 3000
  hostname: localhost
  static:
    public_dir: public
    cache_control:
      default: "public, max-age=86400"
  idle:
    action: suspend
    timeout: "20m"

routes:
  fly:
    replay: []

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

func TestRoutesWithRedirects(t *testing.T) {
	// Test configuration with both redirects and rewrites
	testConfig := `
server:
  listen: 3000
  hostname: localhost

routes:
  redirects:
    - from: "^/(showcase)?$"
      to: "/showcase/studios/"
    - from: "^/old-path$"
      to: "/new-path"
  rewrites:
    - from: "^/assets/(.*)"
      to: "/showcase/assets/$1"
    - from: "^/([^/]+\\.(gif|png|jpg))$"
      to: "/showcase/$1"

applications:
  pools:
    max_size: 5
    timeout: "5m"
    start_port: 4000
  tenants: []
`

	tmpFile, err := os.CreateTemp("", "navigator-routes-test-*.yml")
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
		t.Fatalf("Failed to load config with routes: %v", err)
	}

	// Verify redirects are loaded correctly
	if len(config.Routes.Redirects) != 2 {
		t.Errorf("Expected 2 redirects, got %d", len(config.Routes.Redirects))
	}

	if len(config.Routes.Redirects) > 0 {
		redirect1 := config.Routes.Redirects[0]
		if redirect1.From != "^/(showcase)?$" {
			t.Errorf("Expected redirect from '^/(showcase)?$', got '%s'", redirect1.From)
		}
		if redirect1.To != "/showcase/studios/" {
			t.Errorf("Expected redirect to '/showcase/studios/', got '%s'", redirect1.To)
		}
	}

	// Verify rewrites are loaded correctly
	if len(config.Routes.Rewrites) != 2 {
		t.Errorf("Expected 2 rewrites, got %d", len(config.Routes.Rewrites))
	}

	if len(config.Routes.Rewrites) > 0 {
		rewrite1 := config.Routes.Rewrites[0]
		if rewrite1.From != "^/assets/(.*)" {
			t.Errorf("Expected rewrite from '^/assets/(.*)', got '%s'", rewrite1.From)
		}
		if rewrite1.To != "/showcase/assets/$1" {
			t.Errorf("Expected rewrite to '/showcase/assets/$1', got '%s'", rewrite1.To)
		}
	}

	// Verify rewrite rules are converted correctly
	// Redirects should become rewrite rules with "redirect" flag
	// Rewrites should become rewrite rules with "last" flag
	redirectCount := 0
	rewriteCount := 0

	for _, rule := range config.Server.RewriteRules {
		if rule.Flag == "redirect" {
			redirectCount++
		} else if rule.Flag == "last" {
			rewriteCount++
		}
	}

	if redirectCount != 2 {
		t.Errorf("Expected 2 redirect rewrite rules, got %d", redirectCount)
	}
	if rewriteCount != 2 {
		t.Errorf("Expected 2 internal rewrite rules, got %d", rewriteCount)
	}
}

func TestLoggingConfigurationParsing(t *testing.T) {
	tests := []struct {
		name           string
		config         string
		expectedFormat string
		expectedFile   string
	}{
		{
			name: "JSON logging format",
			config: `
server:
  listen: 3000
logging:
  format: json
applications:
  pools:
    max_size: 5
`,
			expectedFormat: "json",
			expectedFile:   "",
		},
		{
			name: "Text logging format",
			config: `
server:
  listen: 3000
logging:
  format: text
applications:
  pools:
    max_size: 5
`,
			expectedFormat: "text",
			expectedFile:   "",
		},
		{
			name: "JSON logging with file output",
			config: `
server:
  listen: 3000
logging:
  format: json
  file: "/var/log/navigator.log"
applications:
  pools:
    max_size: 5
`,
			expectedFormat: "json",
			expectedFile:   "/var/log/navigator.log",
		},
		{
			name: "Default logging format (empty)",
			config: `
server:
  listen: 3000
applications:
  pools:
    max_size: 5
`,
			expectedFormat: "",
			expectedFile:   "",
		},
		{
			name: "Logging section without format",
			config: `
server:
  listen: 3000
logging:
  file: "/tmp/navigator.log"
applications:
  pools:
    max_size: 5
`,
			expectedFormat: "",
			expectedFile:   "/tmp/navigator.log",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "navigator-logging-test-*.yml")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.WriteString(tt.config); err != nil {
				t.Fatalf("Failed to write test config: %v", err)
			}
			tmpFile.Close()

			config, err := LoadConfig(tmpFile.Name())
			if err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			// Verify logging format
			if config.Logging.Format != tt.expectedFormat {
				t.Errorf("Expected logging format %q, got %q", tt.expectedFormat, config.Logging.Format)
			}

			// Verify logging file
			if config.Logging.File != tt.expectedFile {
				t.Errorf("Expected logging file %q, got %q", tt.expectedFile, config.Logging.File)
			}
		})
	}
}

func TestLoggingWithVectorConfiguration(t *testing.T) {
	testConfig := `
server:
  listen: 3000
logging:
  format: json
  file: "/var/log/navigator.log"
  vector:
    enabled: true
    socket: "/tmp/vector.sock"
    config: "/etc/vector/vector.toml"
applications:
  pools:
    max_size: 5
`

	tmpFile, err := os.CreateTemp("", "navigator-vector-test-*.yml")
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

	// Verify logging configuration
	if config.Logging.Format != "json" {
		t.Errorf("Expected logging format json, got %s", config.Logging.Format)
	}

	if config.Logging.File != "/var/log/navigator.log" {
		t.Errorf("Expected logging file /var/log/navigator.log, got %s", config.Logging.File)
	}

	// Verify Vector configuration
	if !config.Logging.Vector.Enabled {
		t.Error("Expected Vector to be enabled")
	}

	if config.Logging.Vector.Socket != "/tmp/vector.sock" {
		t.Errorf("Expected Vector socket /tmp/vector.sock, got %s", config.Logging.Vector.Socket)
	}

	if config.Logging.Vector.Config != "/etc/vector/vector.toml" {
		t.Errorf("Expected Vector config /etc/vector/vector.toml, got %s", config.Logging.Vector.Config)
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

// TestTenantNameExtractionFromPath tests the critical tenant name extraction logic
func TestTenantNameExtractionFromPath(t *testing.T) {
	testConfig := `
server:
  listen: 3000
  static:
    public_dir: public

applications:
  pools:
    max_size: 10
    timeout: 5m
    start_port: 4000
  tenants:
    - path: /showcase/2025/raleigh/shimmer-shine/
      root: /app/shimmer-shine
    - path: /showcase/2025/boston/april/
      root: /app/boston
    - path: /showcase/test-simple/
      root: /app/simple
`

	tmpFile, err := os.CreateTemp("", "navigator-tenant-test-*.yml")
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

	// Verify tenant names are extracted correctly from paths
	if len(config.Applications.Tenants) != 3 {
		t.Fatalf("Expected 3 tenants, got %d", len(config.Applications.Tenants))
	}

	expectedTenants := []struct {
		name string
		root string
	}{
		{"2025/raleigh/shimmer-shine", "/app/shimmer-shine"},
		{"2025/boston/april", "/app/boston"},
		{"test-simple", "/app/simple"},
	}

	for i, expected := range expectedTenants {
		if config.Applications.Tenants[i].Name != expected.name {
			t.Errorf("Tenant %d: expected name %q, got %q", i, expected.name, config.Applications.Tenants[i].Name)
		}
		if config.Applications.Tenants[i].Root != expected.root {
			t.Errorf("Tenant %d: expected root %q, got %q", i, expected.root, config.Applications.Tenants[i].Root)
		}
	}
}
