package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfigDefaults(t *testing.T) {
	// Test loading configuration with only defaults
	// Set required environment variable
	os.Setenv("NAVIGATOR_RAILS_ROOT", "/tmp")
	defer os.Unsetenv("NAVIGATOR_RAILS_ROOT")
	
	config, err := LoadConfig("")
	if err != nil {
		t.Fatalf("Failed to load default config: %v", err)
	}

	// Check default values
	if config.Server.Listen != ":3000" {
		t.Errorf("Expected default listen address ':3000', got '%s'", config.Server.Listen)
	}
	if config.Server.URLPrefix != "/showcase" {
		t.Errorf("Expected default URL prefix '/showcase', got '%s'", config.Server.URLPrefix)
	}
	if config.Manager.MaxPuma != 10 {
		t.Errorf("Expected default max puma 10, got %d", config.Manager.MaxPuma)
	}
	if config.Manager.IdleTimeout != 5*time.Minute {
		t.Errorf("Expected default idle timeout 5m, got %v", config.Manager.IdleTimeout)
	}
	if config.Logging.Level != "info" {
		t.Errorf("Expected default log level 'info', got '%s'", config.Logging.Level)
	}
}

func TestLoadConfigFromYAML(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "navigator.yaml")

	configContent := `
server:
  listen: ":8080"
  url_prefix: "/test"

rails:
  root: "/tmp/test-rails"
  showcases: "custom/showcases.yml"
  db_path: "custom-db"
  storage: "custom-storage"

manager:
  max_puma: 20
  idle_timeout: "10m"

auth:
  htpasswd_file: "/tmp/htpasswd"

logging:
  level: "debug"
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Load config from file
	config, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("Failed to load config from YAML: %v", err)
	}

	// Verify values were loaded correctly
	if config.Server.Listen != ":8080" {
		t.Errorf("Expected listen ':8080', got '%s'", config.Server.Listen)
	}
	if config.Server.URLPrefix != "/test" {
		t.Errorf("Expected URL prefix '/test', got '%s'", config.Server.URLPrefix)
	}
	if config.Rails.Root != "/tmp/test-rails" {
		t.Errorf("Expected rails root '/tmp/test-rails', got '%s'", config.Rails.Root)
	}
	if config.Rails.Showcases != "custom/showcases.yml" {
		t.Errorf("Expected showcases 'custom/showcases.yml', got '%s'", config.Rails.Showcases)
	}
	if config.Manager.MaxPuma != 20 {
		t.Errorf("Expected max puma 20, got %d", config.Manager.MaxPuma)
	}
	if config.Manager.IdleTimeout != 10*time.Minute {
		t.Errorf("Expected idle timeout 10m, got %v", config.Manager.IdleTimeout)
	}
	if config.Auth.HtpasswdFile != "/tmp/htpasswd" {
		t.Errorf("Expected htpasswd file '/tmp/htpasswd', got '%s'", config.Auth.HtpasswdFile)
	}
	if config.Logging.Level != "debug" {
		t.Errorf("Expected log level 'debug', got '%s'", config.Logging.Level)
	}
}

func TestLoadConfigWithEnvironmentVariables(t *testing.T) {
	// Set environment variables
	oldEnvVars := map[string]string{}
	envVars := map[string]string{
		"NAVIGATOR_RAILS_ROOT":             "/env/rails/root",
		"NAVIGATOR_SERVER_LISTEN":          ":9000",
		"NAVIGATOR_SERVER_URL_PREFIX":      "/env-prefix",
		"NAVIGATOR_MANAGER_MAX_PUMA":       "15",
		"NAVIGATOR_MANAGER_IDLE_TIMEOUT":   "15m",
		"NAVIGATOR_AUTH_HTPASSWD_FILE":     "/env/htpasswd",
		"NAVIGATOR_LOGGING_LEVEL":          "warn",
	}

	// Save old values and set new ones
	for key, value := range envVars {
		if oldValue, exists := os.LookupEnv(key); exists {
			oldEnvVars[key] = oldValue
		}
		os.Setenv(key, value)
	}

	// Cleanup function
	defer func() {
		for key := range envVars {
			if oldValue, exists := oldEnvVars[key]; exists {
				os.Setenv(key, oldValue)
			} else {
				os.Unsetenv(key)
			}
		}
	}()

	// Load config (no file, should use env vars and defaults)
	config, err := LoadConfig("")
	if err != nil {
		t.Fatalf("Failed to load config with env vars: %v", err)
	}

	// Verify environment variables override defaults
	if config.Rails.Root != "/env/rails/root" {
		t.Errorf("Expected rails root '/env/rails/root', got '%s'", config.Rails.Root)
	}
	if config.Server.Listen != ":9000" {
		t.Errorf("Expected listen ':9000', got '%s'", config.Server.Listen)
	}
	if config.Server.URLPrefix != "/env-prefix" {
		t.Errorf("Expected URL prefix '/env-prefix', got '%s'", config.Server.URLPrefix)
	}
	if config.Manager.MaxPuma != 15 {
		t.Errorf("Expected max puma 15, got %d", config.Manager.MaxPuma)
	}
	if config.Manager.IdleTimeout != 15*time.Minute {
		t.Errorf("Expected idle timeout 15m, got %v", config.Manager.IdleTimeout)
	}
	if config.Auth.HtpasswdFile == "" {
		t.Error("Expected htpasswd file to be set from environment variable")
	}
	if config.Logging.Level != "warn" {
		t.Errorf("Expected log level 'warn', got '%s'", config.Logging.Level)
	}
}

func TestValidateAndResolvePaths(t *testing.T) {
	tmpDir := t.TempDir()
	
	config := &Config{
		Rails: RailsConfig{
			Root:      tmpDir,
			DbPath:    "db",
			Storage:   "storage",
		},
		Auth: AuthConfig{
			HtpasswdFile: "htpasswd",
		},
	}

	err := config.validateAndResolvePaths()
	if err != nil {
		t.Fatalf("Failed to validate and resolve paths: %v", err)
	}

	// Check that relative paths were resolved
	if !filepath.IsAbs(config.Rails.DbPath) {
		t.Errorf("Expected absolute db path, got '%s'", config.Rails.DbPath)
	}
	if !filepath.IsAbs(config.Rails.Storage) {
		t.Errorf("Expected absolute storage path, got '%s'", config.Rails.Storage)
	}
	if !filepath.IsAbs(config.Auth.HtpasswdFile) {
		t.Errorf("Expected absolute htpasswd path, got '%s'", config.Auth.HtpasswdFile)
	}

	// Check that paths are relative to Rails root
	expectedDbPath := filepath.Join(tmpDir, "db")
	if config.Rails.DbPath != expectedDbPath {
		t.Errorf("Expected db path '%s', got '%s'", expectedDbPath, config.Rails.DbPath)
	}
}

func TestValidateAndResolvePathsDefaultRoot(t *testing.T) {
	config := &Config{
		Rails: RailsConfig{
			Root: "", // Empty root should default to "."
		},
	}

	err := config.validateAndResolvePaths()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify that empty root was replaced with "."
	expectedRoot, _ := filepath.Abs(".")
	if config.Rails.Root != expectedRoot {
		t.Errorf("Expected root to be '%s', got '%s'", expectedRoot, config.Rails.Root)
	}
}

func TestGetShowcasesPath(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		railsRoot   string
		showcases   string
		expected    string
	}{
		{
			name:      "relative path",
			railsRoot: tmpDir,
			showcases: "config/showcases.yml",
			expected:  filepath.Join(tmpDir, "config/showcases.yml"),
		},
		{
			name:      "absolute path",
			railsRoot: tmpDir,
			showcases: "/absolute/showcases.yml",
			expected:  "/absolute/showcases.yml",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := &Config{
				Rails: RailsConfig{
					Root:      test.railsRoot,
					Showcases: test.showcases,
				},
			}

			result := config.GetShowcasesPath()
			if result != test.expected {
				t.Errorf("Expected showcases path '%s', got '%s'", test.expected, result)
			}
		})
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "bad.yaml")

	// Create invalid YAML file
	badContent := `
server:
  listen: ":8080"
  invalid yaml: [unclosed bracket
`

	if err := os.WriteFile(configFile, []byte(badContent), 0644); err != nil {
		t.Fatalf("Failed to create bad config file: %v", err)
	}

	// Should return error for invalid YAML
	_, err := LoadConfig(configFile)
	if err == nil {
		t.Error("Expected error for invalid YAML, but got none")
	}
}

func TestLoadConfigNonexistentFile(t *testing.T) {
	// Should return error for nonexistent file when explicitly specified
	_, err := LoadConfig("/nonexistent/config.yaml")
	if err == nil {
		t.Error("Expected error for nonexistent config file, but got none")
	}
}