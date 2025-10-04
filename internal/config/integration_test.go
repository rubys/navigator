package config

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestParseAuthConfig_NoWarningsForGlobPatterns(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug, // Capture all log levels
	}))
	slog.SetDefault(logger)

	// Create config with glob patterns
	yamlConfig := YAMLConfig{
		Auth: struct {
			Enabled     bool     `yaml:"enabled"`
			Realm       string   `yaml:"realm"`
			HTPasswd    string   `yaml:"htpasswd"`
			PublicPaths []string `yaml:"public_paths"`
		}{
			Enabled:  true,
			HTPasswd: "/etc/htpasswd",
			PublicPaths: []string{
				"*.css",
				"*.js",
				"*.png",
				"*.jpg",
				"*.gif",
				"*.woff",
				"*.woff2",
				"/assets/",
				"/favicon.ico",
			},
		},
	}

	// Parse the configuration
	parser := NewConfigParser(&yamlConfig)
	config, err := parser.Parse()
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Verify glob patterns were preserved in Auth.PublicPaths
	if len(config.Auth.PublicPaths) != 9 {
		t.Errorf("Expected 9 auth public paths, got %d", len(config.Auth.PublicPaths))
	}

	// Check for any warnings about invalid patterns
	logOutput := buf.String()
	if strings.Contains(logOutput, "Invalid auth exclude pattern") {
		t.Errorf("Unexpected warning about invalid auth exclude pattern in logs:\n%s", logOutput)
	}
	if strings.Contains(logOutput, "error parsing regexp") {
		t.Errorf("Unexpected regex parsing error in logs:\n%s", logOutput)
	}
	if strings.Contains(logOutput, "missing argument to repetition operator") {
		t.Errorf("Unexpected regex repetition operator error in logs:\n%s", logOutput)
	}

	// Verify the glob patterns are present as-is
	expectedPatterns := []string{
		"*.css", "*.js", "*.png", "*.jpg", "*.gif",
		"*.woff", "*.woff2", "/assets/", "/favicon.ico",
	}
	for i, expected := range expectedPatterns {
		if i >= len(config.Auth.PublicPaths) || config.Auth.PublicPaths[i] != expected {
			t.Errorf("Auth.PublicPaths[%d] = %q, want %q", i, config.Auth.PublicPaths[i], expected)
		}
	}
}
