package server

import (
	"testing"

	"github.com/rubys/navigator/internal/config"
)

func TestHandler_getPublicDir(t *testing.T) {
	tests := []struct {
		name      string
		publicDir string
		expected  string
	}{
		{
			name:      "configured public directory",
			publicDir: "dist",
			expected:  "dist",
		},
		{
			name:      "empty public directory returns default",
			publicDir: "",
			expected:  config.DefaultPublicDir,
		},
		{
			name:      "custom public path",
			publicDir: "/var/www/public",
			expected:  "/var/www/public",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.Server.Static.PublicDir = tt.publicDir

			staticHandler := NewStaticFileHandler(cfg)

			result := staticHandler.getPublicDir()
			if result != tt.expected {
				t.Errorf("getPublicDir() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Test to verify constants are properly defined
func TestConstantsDefined(t *testing.T) {
	// Verify DefaultPublicDir is defined
	if config.DefaultPublicDir == "" {
		t.Error("DefaultPublicDir constant is not defined")
	}

	// Verify it has expected value
	if config.DefaultPublicDir != "public" {
		t.Errorf("DefaultPublicDir = %v, want 'public'", config.DefaultPublicDir)
	}

	// Verify other important constants
	if config.DefaultListenPort == 0 {
		t.Error("DefaultListenPort constant is not defined")
	}

	if config.DefaultStartPort == 0 {
		t.Error("DefaultStartPort constant is not defined")
	}

	// Verify stream constants
	if config.StreamStdout == "" {
		t.Error("StreamStdout constant is not defined")
	}

	if config.StreamStderr == "" {
		t.Error("StreamStderr constant is not defined")
	}

	// Verify header constants
	if config.HeaderRequestID == "" {
		t.Error("HeaderRequestID constant is not defined")
	}

	// Verify static file extensions are defined
	if len(config.StaticFileExtensions) == 0 {
		t.Error("StaticFileExtensions not defined")
	}

	// Check that common extensions are included
	commonExtensions := []string{"js", "css", "png", "jpg", "svg", "ico"}
	for _, ext := range commonExtensions {
		found := false
		for _, staticExt := range config.StaticFileExtensions {
			if staticExt == ext {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected extension %q not found in StaticFileExtensions", ext)
		}
	}
}

// Test MIME type mappings
func TestMIMETypes(t *testing.T) {
	expectedMIMEs := map[string]string{
		".html": "text/html; charset=utf-8",
		".css":  "text/css; charset=utf-8",
		".js":   "application/javascript",
		".json": "application/json",
		".png":  "image/png",
		".jpg":  "image/jpeg",
	}

	for ext, expectedMIME := range expectedMIMEs {
		if mime, ok := config.MIMETypes[ext]; !ok {
			t.Errorf("MIME type for %q not defined", ext)
		} else if mime != expectedMIME {
			t.Errorf("MIME type for %q = %v, want %v", ext, mime, expectedMIME)
		}
	}
}
