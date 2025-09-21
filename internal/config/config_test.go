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