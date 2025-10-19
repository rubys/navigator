package server

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/rubys/navigator/internal/config"
	"github.com/rubys/navigator/internal/logging"
	"github.com/rubys/navigator/internal/utils"
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
	if s.config.Server.Static.PublicDir != "" {
		return s.config.Server.Static.PublicDir
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

	logging.LogStaticFileCheck(r.Method, path)

	// Strip the root path if configured (e.g., "/showcase" prefix)
	strippedPath := s.stripRootPath(path)
	if strippedPath != path {
		logging.LogStaticFileStripRoot(path, s.config.Server.RootPath, strippedPath)
		path = strippedPath
	}

	// Check if file has a static extension
	if !s.hasStaticExtension(path) {
		return false
	}

	// Use server-level public directory
	fsPath := filepath.Join(s.getPublicDir(), path)

	// Check if file exists
	logging.LogStaticFileExistenceCheck(fsPath, path)
	if info, err := os.Stat(fsPath); os.IsNotExist(err) || info.IsDir() {
		logging.LogStaticFileNotFound(fsPath, err)
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
	logging.LogStaticFileServe(path, fsPath)
	return true
}

// hasStaticExtension checks if the path has a static file extension
func (s *StaticFileHandler) hasStaticExtension(path string) bool {
	ext := strings.TrimPrefix(filepath.Ext(path), ".")
	if ext == "" {
		return false
	}

	// Use configured allowed extensions
	if len(s.config.Server.Static.AllowedExtensions) > 0 {
		for _, allowedExt := range s.config.Server.Static.AllowedExtensions {
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

	logging.LogTryFilesCheck(path)

	// Only try files for paths that don't already have an extension
	if filepath.Ext(path) != "" {
		logging.LogTryFilesSkipExtension()
		return false
	}

	// Get try_files extensions
	extensions := s.getTryFileExtensions()
	if len(extensions) == 0 {
		logging.LogTryFilesDisabled()
		return false
	}

	// Try files in public directory FIRST
	// This allows prerendered HTML files to be served statically even when
	// their paths match tenant prefixes (e.g., /showcase/studios/millbrae.html)
	if s.tryPublicDirFiles(w, r, extensions, path) {
		return true // Found and served static file
	}

	// No static file found - skip if this is a tenant path
	// (let tenant handle dynamic requests)
	for _, tenant := range s.config.Applications.Tenants {
		if strings.HasPrefix(path, tenant.Path) {
			logging.LogTryFilesSkipTenant(tenant.Path)
			return false
		}
	}

	return false
}

// getTryFileExtensions returns the configured try_files extensions
func (s *StaticFileHandler) getTryFileExtensions() []string {
	return s.config.Server.Static.TryFiles
}

// tryPublicDirFiles attempts to serve files from the public directory
func (s *StaticFileHandler) tryPublicDirFiles(w http.ResponseWriter, r *http.Request, extensions []string, path string) bool {
	logging.LogTryFilesSearching(path)

	publicDir := s.getPublicDir()
	strippedPath := s.stripRootPath(path)

	// Try each extension
	for _, ext := range extensions {
		fsPath := filepath.Join(publicDir, strippedPath+ext)
		logging.LogTryFilesCheckingPath(fsPath)
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
	logging.LogTryFilesServe(requestPath, fsPath)
	return true
}

// setCacheControl sets Cache-Control headers based on configuration
func (s *StaticFileHandler) setCacheControl(w http.ResponseWriter, path string) {
	// Find the most specific cache control override
	var maxAge string
	bestMatchLen := 0

	for _, override := range s.config.Server.Static.CacheControl.Overrides {
		if strings.HasPrefix(path, override.Path) && len(override.Path) > bestMatchLen {
			maxAge = override.MaxAge
			bestMatchLen = len(override.Path)
		}
	}

	// Use default if no override matched
	if maxAge == "" {
		maxAge = s.config.Server.Static.CacheControl.Default
	}

	// Set Cache-Control header if configured
	if maxAge != "" && maxAge != "0" && maxAge != "0s" {
		// Parse duration and convert to seconds
		duration := utils.ParseDurationWithDefault(maxAge, 0)
		if duration > 0 {
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
