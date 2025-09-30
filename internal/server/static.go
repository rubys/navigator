package server

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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

	// Set content type and serve the file
	SetContentType(w, fsPath)
	http.ServeFile(w, r, fsPath)
	slog.Debug("Serving static file", "path", path, "fsPath", fsPath)
	return true
}

// hasStaticExtension checks if the path has a common static file extension
func (s *StaticFileHandler) hasStaticExtension(path string) bool {
	ext := strings.TrimPrefix(filepath.Ext(path), ".")
	if ext == "" {
		return false
	}

	// Common static file extensions
	staticExts := []string{"js", "css", "png", "jpg", "jpeg", "gif", "svg", "ico", "pdf", "txt", "xml", "json", "woff", "woff2", "ttf", "eot"}
	for _, staticExt := range staticExts {
		if ext == staticExt {
			return true
		}
	}
	return false
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

	// Try static directories first
	if staticDir := s.findMatchingStaticDir(path); staticDir != nil {
		return s.tryStaticDirFiles(w, r, staticDir, extensions, path)
	}

	// Fallback to public directory
	return s.tryPublicDirFiles(w, r, extensions, path)
}

// getTryFileExtensions returns the configured try_files extensions
func (s *StaticFileHandler) getTryFileExtensions() []string {
	if len(s.config.Server.TryFiles) > 0 {
		return s.config.Server.TryFiles
	}
	if s.config.Static.TryFiles.Enabled && len(s.config.Static.TryFiles.Suffixes) > 0 {
		return s.config.Static.TryFiles.Suffixes
	}
	// Default extensions
	return []string{".html", ".htm", ".txt", ".xml", ".json"}
}

// findMatchingStaticDir finds the best matching static directory for a path
func (s *StaticFileHandler) findMatchingStaticDir(path string) *config.StaticDir {
	var bestStaticDir *config.StaticDir
	bestStaticDirLen := 0

	slog.Debug("Static directory matching", "path", path, "numDirectories", len(s.config.Static.Directories))
	for i, staticDir := range s.config.Static.Directories {
		hasPrefix := strings.HasPrefix(path, staticDir.Path)
		isLonger := len(staticDir.Path) > bestStaticDirLen
		slog.Debug("Checking static directory",
			"index", i,
			"staticPath", staticDir.Path,
			"dir", staticDir.Dir,
			"hasPrefix", hasPrefix,
			"pathLen", len(staticDir.Path),
			"bestLen", bestStaticDirLen,
			"isLonger", isLonger)
		if hasPrefix && isLonger {
			slog.Debug("New best match found", "staticPath", staticDir.Path, "dir", staticDir.Dir)
			bestStaticDir = &staticDir
			bestStaticDirLen = len(staticDir.Path)
		}
	}

	return bestStaticDir
}

// tryStaticDirFiles attempts to serve files from a static directory
func (s *StaticFileHandler) tryStaticDirFiles(w http.ResponseWriter, r *http.Request, staticDir *config.StaticDir, extensions []string, path string) bool {
	slog.Debug("Found matching static directory", "path", path, "staticPath", staticDir.Path, "dir", staticDir.Dir)

	// Remove the URL prefix to get the relative path
	relativePath := strings.TrimPrefix(path, staticDir.Path)
	if relativePath == "" {
		relativePath = "/"
	}
	if relativePath[0] != '/' {
		relativePath = "/" + relativePath
	}

	// Use server public directory as base
	publicDir := s.getPublicDir()

	// Try each extension
	for _, ext := range extensions {
		fsPath := filepath.Join(publicDir, staticDir.Dir, relativePath+ext)
		slog.Debug("tryFiles checking static", "fsPath", fsPath)
		if info, err := os.Stat(fsPath); err == nil && !info.IsDir() {
			return s.serveFile(w, r, fsPath, path+ext)
		}
	}
	return false
}

// tryPublicDirFiles attempts to serve files from the public directory
func (s *StaticFileHandler) tryPublicDirFiles(w http.ResponseWriter, r *http.Request, extensions []string, path string) bool {
	slog.Debug("No static directory match found, using fallback", "path", path)

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

	// Serve the file
	http.ServeFile(w, r, fsPath)
	slog.Info("Serving file via tryFiles", "requestPath", requestPath, "fsPath", fsPath)
	return true
}

// ServeFallback serves a fallback file when no tenants are configured
func (s *StaticFileHandler) ServeFallback(w http.ResponseWriter, r *http.Request) {
	// Check if static fallback is configured
	if s.config.Static.TryFiles.Fallback != "" {
		fallbackPath := s.config.Static.TryFiles.Fallback

		// Build the filesystem path
		publicDir := s.getPublicDir()
		fsPath := filepath.Join(publicDir, fallbackPath)

		// Check if the fallback file exists
		if info, err := os.Stat(fsPath); err == nil && !info.IsDir() {
			if recorder, ok := w.(*ResponseRecorder); ok {
				recorder.SetMetadata("response_type", "static-fallback")
				recorder.SetMetadata("file_path", fsPath)
			}

			SetContentType(w, fsPath)
			http.ServeFile(w, r, fsPath)
			slog.Info("Serving static fallback", "path", r.URL.Path, "fallback", fallbackPath, "fsPath", fsPath)
			return
		}
	}

	// No fallback configured or file not found
	http.NotFound(w, r)
}
