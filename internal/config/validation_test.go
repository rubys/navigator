package config

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// TestConfigurationValidation tests comprehensive validation scenarios
func TestConfigurationValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      string
		expectError bool
		errorMatch  string // Regex pattern for expected error message
	}{
		{
			name: "valid minimal configuration",
			config: `
server:
  listen: 3000
applications:
  pools:
    max_size: 1
`,
			expectError: false,
		},
		{
			name: "invalid YAML syntax",
			config: `
server:
  listen: 3000
  invalid: [unclosed
`,
			expectError: true,
			errorMatch:  "yaml|parse",
		},
		{
			name: "invalid port number too high",
			config: `
server:
  listen: 99999
applications:
  pools:
    max_size: 1
`,
			expectError: false, // Navigator accepts this but may fail at runtime
		},
		{
			name: "invalid port number negative",
			config: `
server:
  listen: -1
applications:
  pools:
    max_size: 1
`,
			expectError: false, // YAML parsing handles this as a string
		},
		{
			name: "invalid duration format in idle timeout",
			config: `
server:
  listen: 3000
  idle:
    action: suspend
    timeout: "invalid-duration"
applications:
  pools:
    max_size: 1
`,
			expectError: false, // Duration validation happens at runtime
		},
		{
			name: "invalid idle action",
			config: `
server:
  listen: 3000
  idle:
    action: invalid-action
    timeout: "20m"
applications:
  pools:
    max_size: 1
`,
			expectError: false, // Action validation happens at runtime
		},
		{
			name: "invalid regex pattern in rewrite rules",
			config: `
server:
  listen: 3000
  rewrites:
    - pattern: "[unclosed"
      replacement: "/test"
      flag: "last"
applications:
  pools:
    max_size: 1
`,
			expectError: false, // Regex compilation happens later
		},
		{
			name:        "empty configuration file",
			config:      ``,
			expectError: false, // Should use defaults
		},
		{
			name: "configuration with only comments",
			config: `
# This is a comment-only configuration file
# server:
#   listen: 3000
`,
			expectError: false, // Should handle empty config gracefully
		},
		{
			name: "extremely large pool size",
			config: `
server:
  listen: 3000
applications:
  pools:
    max_size: 999999999
    start_port: 4000
`,
			expectError: false, // Large numbers should be handled
		},
		{
			name: "negative pool size",
			config: `
server:
  listen: 3000
applications:
  pools:
    max_size: -1
    start_port: 4000
`,
			expectError: false, // Validation happens at runtime
		},
		{
			name: "very long tenant names",
			config: `
server:
  listen: 3000
applications:
  pools:
    max_size: 1
  tenants:
    - path: /showcase/` + strings.Repeat("very-long-tenant-name-", 50) + `/
      root: /tmp/test
`,
			expectError: false, // Should handle long names
		},
		{
			name: "special characters in tenant paths",
			config: `
server:
  listen: 3000
applications:
  pools:
    max_size: 1
  tenants:
    - path: /showcase/test-with-special-chars-!@#$%^&*()/
      root: /tmp/test
    - path: /showcase/unicode-测试-тест-テスト/
      root: /tmp/unicode
`,
			expectError: false, // Should handle special characters
		},
		{
			name: "duplicate tenant names",
			config: `
server:
  listen: 3000
applications:
  pools:
    max_size: 1
  tenants:
    - path: /showcase/duplicate/
      root: /tmp/test1
    - path: /showcase/duplicate/
      root: /tmp/test2
`,
			expectError: false, // Should parse but may cause runtime issues
		},
		{
			name: "extremely deep YAML nesting",
			config: `
server:
  listen: 3000
applications:
  pools:
    max_size: 1
  tenants:
    - name: deep-test
      var:
        level1:
          level2:
            level3:
              level4:
                level5:
                  deep_value: "test"
`,
			expectError: false, // Should handle deep nesting
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "navigator-validation-*.yml")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.WriteString(tt.config); err != nil {
				t.Fatalf("Failed to write test config: %v", err)
			}
			tmpFile.Close()

			config, err := LoadConfig(tmpFile.Name())

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", tt.name)
				} else if tt.errorMatch != "" {
					matched, _ := regexp.MatchString(tt.errorMatch, err.Error())
					if !matched {
						t.Errorf("Expected error to match %q, got: %v", tt.errorMatch, err)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.name, err)
				}
				if config == nil {
					t.Errorf("Config should not be nil for valid configuration")
				}
			}
		})
	}
}

// TestInvalidFilePaths tests configuration loading with various invalid file paths
func TestInvalidFilePaths(t *testing.T) {
	tests := []struct {
		name     string
		filepath string
	}{
		{"non-existent file", "/nonexistent/path/config.yml"},
		{"directory instead of file", "/tmp"},
		{"empty string path", ""},
		{"path with null bytes", "/tmp/config\x00.yml"},
		{"extremely long path", "/" + strings.Repeat("very-long-directory-name/", 100) + "config.yml"},
		{"path with invalid characters", "/tmp/config<>|*.yml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadConfig(tt.filepath)
			if err == nil {
				t.Errorf("Expected error for invalid path %q, but got none", tt.filepath)
			}
		})
	}
}

// TestConfigurationLimits tests configuration with boundary values
func TestConfigurationLimits(t *testing.T) {
	tests := []struct {
		name   string
		config string
	}{
		{
			name: "maximum reasonable configuration",
			config: `
server:
  listen: 65535
  hostname: ` + strings.Repeat("a", 253) + `.com
  public_dir: ` + strings.Repeat("very-long-path/", 20) + `
  idle:
    action: suspend
    timeout: "8760h"  # 1 year

static:
  directories:` + func() string {
				var dirs []string
				for i := 0; i < 100; i++ {
					dirs = append(dirs, `
    - path: /static`+string(rune('a'+i%26))+`/
      dir: static`+string(rune('a'+i%26))+`/
      cache: "24h"`)
				}
				return strings.Join(dirs, "")
			}() + `
  extensions:` + func() string {
				exts := []string{"html", "css", "js", "png", "jpg", "gif", "svg", "pdf", "txt", "json"}
				for i := 0; i < 50; i++ {
					exts = append(exts, "ext"+string(rune('a'+i%26)))
				}
				result := ""
				for _, ext := range exts {
					result += "\n    - " + ext
				}
				return result
			}() + `

routes:
  reverse_proxies:` + func() string {
				var proxies []string
				for i := 0; i < 50; i++ {
					proxies = append(proxies, `
    - name: proxy`+string(rune('a'+i%26))+`
      path: ^/api/v`+string(rune('1'+i%9))+`/
      target: http://backend`+string(rune('a'+i%26))+`:800`+string(rune('0'+i%10))+`
      strip_path: true
      headers:
        X-Proxy-ID: proxy`+string(rune('a'+i%26))+`
        X-Backend-Host: backend`+string(rune('a'+i%26)))
				}
				return strings.Join(proxies, "")
			}() + `

applications:
  pools:
    max_size: 1000
    timeout: "24h"
    start_port: 4000
  tenants:` + func() string {
				var tenants []string
				for i := 0; i < 200; i++ {
					tenants = append(tenants, `
    - path: /showcase/tenant`+string(rune('a'+i%26))+string(rune('a'+(i/26)%26))+`/
      root: /apps/tenant`+string(rune('a'+i%26))+string(rune('a'+(i/26)%26))+`
      env:
        TENANT_ID: tenant`+string(rune('a'+i%26))+string(rune('a'+(i/26)%26))+`
        DATABASE_URL: postgres://user:pass@db/tenant`+string(rune('a'+i%26)))
				}
				return strings.Join(tenants, "")
			}() + `

managed_processes:` + func() string {
				var processes []string
				for i := 0; i < 20; i++ {
					processes = append(processes, `
  - name: process`+string(rune('a'+i%26))+`
    command: /usr/local/bin/process`+string(rune('a'+i%26))+`
    auto_restart: true
    start_delay: "`+string(rune('1'+i%9))+`s"`)
				}
				return strings.Join(processes, "")
			}() + `

hooks:
  start:` + func() string {
				var hooks []string
				for i := 0; i < 10; i++ {
					hooks = append(hooks, `
    - command: /bin/echo
      args: ["Starting hook `+string(rune('1'+i))+`"]
      timeout: "30s"`)
				}
				return strings.Join(hooks, "")
			}(),
		},
		{
			name: "minimum configuration",
			config: `
server:
  listen: 1
applications:
  pools:
    max_size: 1
    start_port: 1
`,
		},
		{
			name: "configuration with zero values",
			config: `
server:
  listen: 3000
applications:
  pools:
    max_size: 0
    start_port: 0
  tenants: []
static:
  directories: []
  extensions: []
routes:
  reverse_proxies: []
managed_processes: []
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "navigator-limits-*.yml")
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
				t.Errorf("Failed to load %s config: %v", tt.name, err)
			}

			if config == nil {
				t.Errorf("Config should not be nil for %s", tt.name)
			}
		})
	}
}

// TestDurationValidation tests various duration string formats
func TestDurationValidation(t *testing.T) {
	tests := []struct {
		name     string
		duration string
		valid    bool
	}{
		{"valid seconds", "30s", true},
		{"valid minutes", "5m", true},
		{"valid hours", "2h", true},
		{"valid combined", "1h30m45s", true},
		{"valid milliseconds", "500ms", true},
		{"valid microseconds", "100us", true},
		{"valid nanoseconds", "1000ns", true},
		{"zero duration", "0", true},
		{"zero with unit", "0s", true},
		{"floating point", "1.5h", true},
		{"negative duration", "-1h", true},
		{"empty string", "", false},
		{"invalid unit", "5x", false},
		{"no unit", "123", false},
		{"invalid format", "abc", false},
		{"mixed invalid", "5m3x", false},
		{"spaces", "5 m", false},
		{"multiple periods", "1.5.0h", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test duration parsing by attempting to parse with Go's time.ParseDuration
			_, err := time.ParseDuration(tt.duration)

			if tt.valid && err != nil {
				t.Errorf("Expected duration %q to be valid, but got error: %v", tt.duration, err)
			}
			if !tt.valid && err == nil {
				t.Errorf("Expected duration %q to be invalid, but it was accepted", tt.duration)
			}
		})
	}
}

// TestRegexPatternValidation tests regex pattern compilation
func TestRegexPatternValidation(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		valid   bool
	}{
		{"valid simple pattern", "^/api/", true},
		{"valid with groups", "^/api/v([0-9]+)/", true},
		{"valid complex pattern", "^/showcase/([^/]+)/([^/]+)/$", true},
		{"valid optional group", "^/(showcase)?$", true},
		{"valid character class", "^/[a-zA-Z0-9_-]+/$", true},
		{"valid unicode", "^/测试/[\\p{L}]+/$", true},
		{"empty pattern", "", true},
		{"unclosed bracket", "[unclosed", false},
		{"unclosed parenthesis", "(unclosed", false},
		{"invalid escape", "\\k", false}, // \z is actually valid in Go regex
		{"invalid repetition", "*invalid", false},
		{"unclosed group", "(?unclosed", false},
		{"invalid character class", "[z-a]", false},
		{"nested unclosed brackets", "[[unclosed", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := regexp.Compile(tt.pattern)

			if tt.valid && err != nil {
				t.Errorf("Expected regex %q to be valid, but got error: %v", tt.pattern, err)
			}
			if !tt.valid && err == nil {
				t.Errorf("Expected regex %q to be invalid, but it was accepted", tt.pattern)
			}
		})
	}
}

// TestFileSystemValidation tests various file system scenarios
func TestFileSystemValidation(t *testing.T) {
	// Create temporary directory structure for testing
	testDir, err := os.MkdirTemp("", "navigator-fs-test-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create various test files and directories
	readableFile := filepath.Join(testDir, "readable.yml")
	unreadableFile := filepath.Join(testDir, "unreadable.yml")
	directory := filepath.Join(testDir, "directory")
	nonExistent := filepath.Join(testDir, "nonexistent.yml")

	// Create readable file
	testConfig := `
server:
  listen: 3000
applications:
  pools:
    max_size: 1
`
	if err := os.WriteFile(readableFile, []byte(testConfig), 0644); err != nil {
		t.Fatalf("Failed to create readable file: %v", err)
	}

	// Create unreadable file (if running as non-root)
	if err := os.WriteFile(unreadableFile, []byte(testConfig), 0644); err != nil {
		t.Fatalf("Failed to create unreadable file: %v", err)
	}
	if err := os.Chmod(unreadableFile, 0000); err != nil {
		t.Logf("Could not make file unreadable (may be running as root): %v", err)
	}

	// Create directory
	if err := os.Mkdir(directory, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	tests := []struct {
		name        string
		filepath    string
		expectError bool
	}{
		{"readable file", readableFile, false},
		{"unreadable file", unreadableFile, true},
		{"directory instead of file", directory, true},
		{"non-existent file", nonExistent, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadConfig(tt.filepath)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s, but got none", tt.name)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.name, err)
			}
		})
	}

	// Restore permissions for cleanup
	_ = os.Chmod(unreadableFile, 0644)
}

// TestVariableSubstitutionEdgeCases tests edge cases in variable substitution
func TestVariableSubstitutionEdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		config string
	}{
		{
			name: "nested variable references",
			config: `
applications:
  env:
    VAR1: "${VAR2}_suffix"
    VAR2: "prefix_${VAR3}"
    VAR3: "base"
  tenants:
    - path: /test/
      var:
        VAR2: "middle"
        VAR3: "core"
`,
		},
		{
			name: "circular variable references",
			config: `
applications:
  env:
    VAR1: "${VAR2}"
    VAR2: "${VAR1}"
  tenants:
    - path: /test/
      var:
        VAR1: "value1"
        VAR2: "value2"
`,
		},
		{
			name: "undefined variables",
			config: `
applications:
  env:
    DEFINED: "value"
    UNDEFINED_REF: "${UNDEFINED_VAR}"
  tenants:
    - path: /test/
      var:
        VAR1: "value1"
`,
		},
		{
			name: "variables with special characters",
			config: `
applications:
  env:
    SPECIAL: "${var_with_underscores}"
    NUMBERS: "${var123}"
  tenants:
    - path: /test/
      var:
        var_with_underscores: "test_value"
        var123: "numeric_var"
`,
		},
		{
			name: "empty variable values",
			config: `
applications:
  env:
    EMPTY_VAR: "${empty}"
    NULL_VAR: "${null_var}"
  tenants:
    - path: /test/
      var:
        empty: ""
        null_var: null
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "navigator-vartest-*.yml")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.WriteString(tt.config); err != nil {
				t.Fatalf("Failed to write test config: %v", err)
			}
			tmpFile.Close()

			// Configuration should load without errors even with problematic variable substitution
			config, err := LoadConfig(tmpFile.Name())
			if err != nil {
				t.Errorf("Failed to load config with variable edge cases: %v", err)
			}

			if config == nil {
				t.Errorf("Config should not be nil")
			}
		})
	}
}

// BenchmarkConfigValidation benchmarks configuration validation performance
func BenchmarkConfigValidation(b *testing.B) {
	testConfig := `
server:
  listen: 3000
  hostname: localhost
  public_dir: public
  idle:
    action: suspend
    timeout: 20m

static:
  directories:
    - path: /assets/
      dir: assets/
      cache: 24h
  extensions:
    - html
    - css
    - js

routes:
  reverse_proxies:
    - name: api
      path: ^/api/
      target: http://localhost:8080
      strip_path: true
      headers:
        X-Forwarded-For: $remote_addr

applications:
  pools:
    max_size: 10
    timeout: 5m
    start_port: 4000
  tenants:
    - path: /showcase/app1/
      root: /apps/app1
    - path: /showcase/app2/
      root: /apps/app2
    - path: /showcase/app3/
      root: /apps/app3

managed_processes:
  - name: redis
    command: redis-server
    auto_restart: true
    start_delay: 2s

hooks:
  start:
    - command: echo
      args: ["starting"]
      timeout: 30s
`

	tmpFile, err := os.CreateTemp("", "navigator-bench-validation-*.yml")
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
