package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rubys/navigator/internal/config"
)

// StaticFileHandler handles serving static files
type StaticFileHandler struct {
	config *config.Config
}

// NewStaticFileHandler creates a new static file handler
func NewStaticFileHandler(cfg *config.Config) *StaticFileHandler {
	return &StaticFileHandler{
		config: cfg,
	}
}

// getPublicDir returns the configured public directory or the default
func (s *StaticFileHandler) getPublicDir() string {
	if s.config.Server.PublicDir != "" {
		return s.config.Server.PublicDir
	}
	return config.DefaultPublicDir
}

// stripRootPath removes the configured root path prefix from the URL path
func (s *StaticFileHandler) stripRootPath(path string) string {
	rootPath := s.config.Server.RootPath
	if rootPath != "" && strings.HasPrefix(path, rootPath) {
		path = strings.TrimPrefix(path, rootPath)
		if path == "" {
			return "/"
		}
	}
	return path
}

// ServeStatic attempts to serve a static file
func (s *StaticFileHandler) ServeStatic(w http.ResponseWriter, r *http.Request) bool {
	path := r.URL.Path

	slog.Debug("Checking static file",
		"path", path,
		"publicDir", s.config.Server.PublicDir,
		"rootPath", s.config.Server.RootPath)

	// Strip the root path if configured (e.g., "/showcase" prefix)
	strippedPath := s.stripRootPath(path)
	if strippedPath != path {
		slog.Debug("Stripping root path", "originalPath", path, "rootPath", s.config.Server.RootPath)
		path = strippedPath
		slog.Debug("Path after stripping", "newPath", path)
	}

	// Check if file has a static extension
	if !s.hasStaticExtension(path) {
		return false
	}

	// Use server-level public directory
	fsPath := filepath.Join(s.getPublicDir(), path)

	// Check if file exists
	slog.Debug("Checking file existence", "fsPath", fsPath, "originalPath", path)
	if info, err := os.Stat(fsPath); os.IsNotExist(err) || info.IsDir() {
		slog.Debug("File not found or is directory", "fsPath", fsPath, "err", err)
		return false
	}

	// Set response metadata for logging
	if recorder, ok := w.(*ResponseRecorder); ok {
		recorder.SetMetadata("response_type", "static")
		recorder.SetMetadata("file_path", fsPath)
	}

	// Set content type and cache control headers
	SetContentType(w, fsPath)
	s.setCacheControl(w, r.URL.Path)

	// Serve the file
	http.ServeFile(w, r, fsPath)
	slog.Debug("Serving static file", "path", path, "fsPath", fsPath)
	return true
}

// hasStaticExtension checks if the path has a static file extension
func (s *StaticFileHandler) hasStaticExtension(path string) bool {
	ext := strings.TrimPrefix(filepath.Ext(path), ".")
	if ext == "" {
		return false
	}

	// Use configured allowed extensions
	if len(s.config.Server.AllowedExtensions) > 0 {
		for _, allowedExt := range s.config.Server.AllowedExtensions {
			if ext == allowedExt {
				return true
			}
		}
		return false
	}

	// If no extensions configured, allow all files
	return true
}

// TryFiles attempts to find and serve files with different extensions
func (s *StaticFileHandler) TryFiles(w http.ResponseWriter, r *http.Request) bool {
	path := r.URL.Path

	slog.Debug("tryFiles checking", "path", path)

	// Only try files for paths that don't already have an extension
	if filepath.Ext(path) != "" {
		slog.Debug("tryFiles skipping - path has extension")
		return false
	}

	// Get try_files extensions
	extensions := s.getTryFileExtensions()
	if len(extensions) == 0 {
		slog.Debug("tryFiles disabled - no suffixes configured")
		return false
	}

	// Skip paths that match tenant paths
	// (those should be handled by web app proxy, not public directory fallback)
	for _, tenant := range s.config.Applications.Tenants {
		if strings.HasPrefix(path, tenant.Path) {
			slog.Debug("tryFiles skipping - matches tenant path", "tenantPath", tenant.Path)
			return false
		}
	}

	// Try files in public directory
	return s.tryPublicDirFiles(w, r, extensions, path)
}

// getTryFileExtensions returns the configured try_files extensions
func (s *StaticFileHandler) getTryFileExtensions() []string {
	return s.config.Server.TryFiles
}

// tryPublicDirFiles attempts to serve files from the public directory
func (s *StaticFileHandler) tryPublicDirFiles(w http.ResponseWriter, r *http.Request, extensions []string, path string) bool {
	slog.Debug("Trying files in public directory", "path", path)

	publicDir := s.getPublicDir()
	strippedPath := s.stripRootPath(path)

	// Try each extension
	for _, ext := range extensions {
		fsPath := filepath.Join(publicDir, strippedPath+ext)
		slog.Debug("tryFiles checking", "fsPath", fsPath)
		if info, err := os.Stat(fsPath); err == nil && !info.IsDir() {
			return s.serveFile(w, r, fsPath, strippedPath+ext)
		}
	}

	return false
}

// serveFile serves a specific file with appropriate headers
func (s *StaticFileHandler) serveFile(w http.ResponseWriter, r *http.Request, fsPath, requestPath string) bool {
	// Set metadata for static file response
	if recorder, ok := w.(*ResponseRecorder); ok {
		recorder.SetMetadata("response_type", "static")
		recorder.SetMetadata("file_path", fsPath)
	}

	// Set appropriate content type
	SetContentType(w, fsPath)

	// Set cache control headers
	s.setCacheControl(w, r.URL.Path)

	// Serve the file
	http.ServeFile(w, r, fsPath)
	slog.Info("Serving file via tryFiles", "requestPath", requestPath, "fsPath", fsPath)
	return true
}

// setCacheControl sets Cache-Control headers based on configuration
func (s *StaticFileHandler) setCacheControl(w http.ResponseWriter, path string) {
	// Find the most specific cache control override
	var maxAge string
	bestMatchLen := 0

	for _, override := range s.config.Server.CacheControl.Overrides {
		if strings.HasPrefix(path, override.Path) && len(override.Path) > bestMatchLen {
			maxAge = override.MaxAge
			bestMatchLen = len(override.Path)
		}
	}

	// Use default if no override matched
	if maxAge == "" {
		maxAge = s.config.Server.CacheControl.Default
	}

	// Set Cache-Control header if configured
	if maxAge != "" && maxAge != "0" && maxAge != "0s" {
		// Parse duration and convert to seconds
		if duration, err := time.ParseDuration(maxAge); err == nil {
			seconds := int(duration.Seconds())
			w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", seconds))
		} else {
			// If not a duration, assume it's already in seconds
			w.Header().Set("Cache-Control", "public, max-age="+maxAge)
		}
	}
}

// ServeFallback serves a 404 response when no tenants are configured
func (s *StaticFileHandler) ServeFallback(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}
