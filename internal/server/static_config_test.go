package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/rubys/navigator/internal/config"
)

// TestServerTryFilesWithFileResolution tests the server.try_files configuration
// with actual file resolution and fallback behavior
func TestServerTryFilesWithFileResolution(t *testing.T) {
	// Create temporary directory structure for testing
	tempDir, err := os.MkdirTemp("", "navigator-tryfiles-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	studiosDir := filepath.Join(tempDir, "studios")
	if err := os.MkdirAll(studiosDir, 0755); err != nil {
		t.Fatalf("Failed to create studios dir: %v", err)
	}

	docsDir := filepath.Join(tempDir, "docs")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatalf("Failed to create docs dir: %v", err)
	}

	// Create various test files
	indexContent := "<html><body>Index Page</body></html>"
	htmlContent := "<html><body>HTML Page</body></html>"

	// studios/index.html
	if err := os.WriteFile(filepath.Join(studiosDir, "index.html"), []byte(indexContent), 0644); err != nil {
		t.Fatalf("Failed to write studios/index.html: %v", err)
	}

	// studios/raleigh.html
	if err := os.WriteFile(filepath.Join(studiosDir, "raleigh.html"), []byte(htmlContent), 0644); err != nil {
		t.Fatalf("Failed to write studios/raleigh.html: %v", err)
	}

	// docs/guide.html
	if err := os.WriteFile(filepath.Join(docsDir, "guide.html"), []byte(htmlContent), 0644); err != nil {
		t.Fatalf("Failed to write docs/guide.html: %v", err)
	}

	// Create config with server-level try_files
	cfg := &config.Config{}
	cfg.Server.Static.PublicDir = tempDir
	cfg.Server.Static.TryFiles = []string{"index.html", ".html", ".htm"}
	cfg.Server.Static.AllowedExtensions = []string{"html", "htm"}

	handler := NewStaticFileHandler(cfg)

	tests := []struct {
		name         string
		requestPath  string
		expectServed bool
		expectedFile string
		description  string
	}{
		{
			name:         "Try index.html suffix",
			requestPath:  "/studios/",
			expectServed: true,
			expectedFile: filepath.Join(studiosDir, "index.html"),
			description:  "Should find index.html when requesting directory path",
		},
		{
			name:         "Try .html suffix",
			requestPath:  "/studios/raleigh",
			expectServed: true,
			expectedFile: filepath.Join(studiosDir, "raleigh.html"),
			description:  "Should find .html file when requesting path without extension",
		},
		{
			name:         "File not found - should not serve",
			requestPath:  "/studios/nonexistent",
			expectServed: false,
			description:  "Should return false when file doesn't exist with any suffix",
		},
		{
			name:         "Path with extension - should skip try_files",
			requestPath:  "/docs/guide.html",
			expectServed: false,
			description:  "TryFiles should skip paths that already have extension (use ServeStatic instead)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.requestPath, nil)
			recorder := httptest.NewRecorder()
			respRecorder := NewResponseRecorder(recorder, nil)

			served := handler.TryFiles(respRecorder, req)

			if served != tt.expectServed {
				t.Errorf("%s: expected served=%v, got %v", tt.description, tt.expectServed, served)
			}

			if served && recorder.Code != http.StatusOK {
				t.Errorf("%s: expected status 200, got %d", tt.description, recorder.Code)
			}
		})
	}
}

// TestServerAllowedExtensions tests the server.allowed_extensions configuration
func TestServerAllowedExtensions(t *testing.T) {
	// Create temporary directory with various file types
	tempDir, err := os.MkdirTemp("", "navigator-extensions-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files with different extensions
	testFiles := map[string]string{
		"allowed.html": "<html>Allowed</html>",
		"allowed.css":  "body { color: red; }",
		"allowed.js":   "console.log('allowed');",
		"blocked.txt":  "This should be blocked",
		"blocked.xml":  "<xml>Blocked</xml>",
	}

	for filename, content := range testFiles {
		if err := os.WriteFile(filepath.Join(tempDir, filename), []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	tests := []struct {
		name              string
		allowedExtensions []string
		requestPath       string
		expectServed      bool
		description       string
	}{
		{
			name:              "Allowed extension - html",
			allowedExtensions: []string{"html", "css", "js"},
			requestPath:       "/allowed.html",
			expectServed:      true,
			description:       "Should serve .html file when in allowed_extensions",
		},
		{
			name:              "Allowed extension - css",
			allowedExtensions: []string{"html", "css", "js"},
			requestPath:       "/allowed.css",
			expectServed:      true,
			description:       "Should serve .css file when in allowed_extensions",
		},
		{
			name:              "Blocked extension - txt",
			allowedExtensions: []string{"html", "css", "js"},
			requestPath:       "/blocked.txt",
			expectServed:      false,
			description:       "Should NOT serve .txt file when not in allowed_extensions",
		},
		{
			name:              "Blocked extension - xml",
			allowedExtensions: []string{"html", "css", "js"},
			requestPath:       "/blocked.xml",
			expectServed:      false,
			description:       "Should NOT serve .xml file when not in allowed_extensions",
		},
		{
			name:              "Empty allowed_extensions - allow all",
			allowedExtensions: []string{},
			requestPath:       "/blocked.txt",
			expectServed:      true,
			description:       "Should serve any file when allowed_extensions is empty",
		},
		{
			name:              "Empty allowed_extensions - allow all (xml)",
			allowedExtensions: []string{},
			requestPath:       "/blocked.xml",
			expectServed:      true,
			description:       "Should serve any file when allowed_extensions is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.Server.Static.PublicDir = tempDir
			cfg.Server.Static.AllowedExtensions = tt.allowedExtensions

			handler := NewStaticFileHandler(cfg)

			req := httptest.NewRequest(http.MethodGet, tt.requestPath, nil)
			recorder := httptest.NewRecorder()
			respRecorder := NewResponseRecorder(recorder, nil)

			served := handler.ServeStatic(respRecorder, req)

			if served != tt.expectServed {
				t.Errorf("%s: expected served=%v, got %v", tt.description, tt.expectServed, served)
			}
		})
	}
}

// TestServerCacheControl tests the server.cache_control configuration
func TestServerCacheControl(t *testing.T) {
	// Create temporary directory with test files
	tempDir, err := os.MkdirTemp("", "navigator-cache-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create directory structure
	assetsDir := filepath.Join(tempDir, "assets")
	imagesDir := filepath.Join(tempDir, "images")
	docsDir := filepath.Join(tempDir, "docs")

	for _, dir := range []string{assetsDir, imagesDir, docsDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
	}

	// Create test files
	testFiles := map[string]string{
		filepath.Join(assetsDir, "app.js"):   "console.log('app');",
		filepath.Join(imagesDir, "logo.png"): "PNG_DATA",
		filepath.Join(docsDir, "guide.html"): "<html>Guide</html>",
		filepath.Join(tempDir, "index.html"): "<html>Index</html>",
	}

	for filename, content := range testFiles {
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Create config with cache control
	cfg := &config.Config{}
	cfg.Server.Static.PublicDir = tempDir
	cfg.Server.Static.CacheControl.Default = "1h"
	cfg.Server.Static.CacheControl.Overrides = []config.CacheControlOverride{
		{Path: "/assets/", MaxAge: "24h"},
		{Path: "/images/", MaxAge: "12h"},
		{Path: "/docs/", MaxAge: "30m"},
	}

	handler := NewStaticFileHandler(cfg)

	tests := []struct {
		name           string
		requestPath    string
		expectedMaxAge string
		description    string
	}{
		{
			name:           "Assets path - 24h cache",
			requestPath:    "/assets/app.js",
			expectedMaxAge: "86400", // 24h in seconds
			description:    "Should apply 24h cache for /assets/ path",
		},
		{
			name:           "Images path - 12h cache",
			requestPath:    "/images/logo.png",
			expectedMaxAge: "43200", // 12h in seconds
			description:    "Should apply 12h cache for /images/ path",
		},
		{
			name:           "Docs path - 30m cache",
			requestPath:    "/docs/guide.html",
			expectedMaxAge: "1800", // 30m in seconds
			description:    "Should apply 30m cache for /docs/ path",
		},
		{
			name:           "Root path - default cache",
			requestPath:    "/index.html",
			expectedMaxAge: "3600", // 1h in seconds
			description:    "Should apply default cache for unmatched paths",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.requestPath, nil)
			recorder := httptest.NewRecorder()
			respRecorder := NewResponseRecorder(recorder, nil)

			served := handler.ServeStatic(respRecorder, req)

			if !served {
				t.Fatalf("%s: file was not served", tt.description)
			}

			cacheControl := recorder.Header().Get("Cache-Control")
			expectedHeader := "public, max-age=" + tt.expectedMaxAge

			if cacheControl != expectedHeader {
				t.Errorf("%s: expected Cache-Control=%q, got %q",
					tt.description, expectedHeader, cacheControl)
			}
		})
	}
}

// TestServerTryFilesWithAllowedExtensions tests that try_files works with various file types
// NOTE: Currently, try_files does NOT enforce allowed_extensions - it serves any file found
// This matches the behavior where allowed_extensions only applies to ServeStatic (exact matches)
func TestServerTryFilesWithAllowedExtensions(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "navigator-tryfiles-ext-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files with different extensions
	contentDir := filepath.Join(tempDir, "content")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatalf("Failed to create content dir: %v", err)
	}

	testFiles := map[string]string{
		filepath.Join(contentDir, "page.html"): "<html>Page</html>",
		filepath.Join(contentDir, "data.json"): `{"key": "value"}`,
		filepath.Join(contentDir, "info.txt"):  "Info text",
	}

	for filename, content := range testFiles {
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	cfg := &config.Config{}
	cfg.Server.Static.PublicDir = tempDir
	cfg.Server.Static.TryFiles = []string{".html", ".json", ".txt"}
	cfg.Server.Static.AllowedExtensions = []string{"html", "json"} // Note: try_files doesn't enforce this

	handler := NewStaticFileHandler(cfg)

	tests := []struct {
		name         string
		requestPath  string
		expectServed bool
		description  string
	}{
		{
			name:         "Try .html suffix",
			requestPath:  "/content/page",
			expectServed: true,
			description:  "Should serve .html file via try_files",
		},
		{
			name:         "Try .json suffix",
			requestPath:  "/content/data",
			expectServed: true,
			description:  "Should serve .json file via try_files",
		},
		{
			name:         "Try .txt suffix",
			requestPath:  "/content/info",
			expectServed: true,
			description:  "Should serve .txt file via try_files (allowed_extensions not enforced in try_files)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.requestPath, nil)
			recorder := httptest.NewRecorder()
			respRecorder := NewResponseRecorder(recorder, nil)

			served := handler.TryFiles(respRecorder, req)

			if served != tt.expectServed {
				t.Errorf("%s: expected served=%v, got %v", tt.description, tt.expectServed, served)
			}
		})
	}
}

// TestServerTryFilesDisabled tests that try_files is disabled when not configured
func TestServerTryFilesDisabled(t *testing.T) {
	// Create temporary directory with test file
	tempDir, err := os.MkdirTemp("", "navigator-tryfiles-disabled-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test file
	htmlFile := filepath.Join(tempDir, "page.html")
	if err := os.WriteFile(htmlFile, []byte("<html>Page</html>"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Config WITHOUT try_files
	cfg := &config.Config{}
	cfg.Server.Static.PublicDir = tempDir
	cfg.Server.Static.TryFiles = []string{} // Empty = disabled

	handler := NewStaticFileHandler(cfg)

	// Request without extension should NOT be served when try_files is disabled
	req := httptest.NewRequest(http.MethodGet, "/page", nil)
	recorder := httptest.NewRecorder()
	respRecorder := NewResponseRecorder(recorder, nil)

	served := handler.TryFiles(respRecorder, req)

	if served {
		t.Error("Expected try_files to be disabled (not serve), but file was served")
	}

	// Request WITH extension should still work via ServeStatic
	req2 := httptest.NewRequest(http.MethodGet, "/page.html", nil)
	recorder2 := httptest.NewRecorder()
	respRecorder2 := NewResponseRecorder(recorder2, nil)

	served2 := handler.ServeStatic(respRecorder2, req2)

	if !served2 {
		t.Error("Expected ServeStatic to serve exact file match even when try_files is disabled")
	}
}
