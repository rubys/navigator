package auth

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rubys/navigator/internal/config"
	"github.com/tg123/go-htpasswd"
)

// BasicAuth represents HTTP basic authentication configuration
type BasicAuth struct {
	File     *htpasswd.File
	Realm    string
	Exclude  []string
	filename string       // Path to htpasswd file for reload checks
	mtime    time.Time    // Last modification time of htpasswd file
	mu       sync.RWMutex // Protects concurrent access to File, filename, and mtime
}

// LoadAuthFile loads an htpasswd file for authentication
func LoadAuthFile(filename, realm string, exclude []string) (*BasicAuth, error) {
	if filename == "" {
		return nil, nil
	}

	// Use go-htpasswd library to load the file
	htFile, err := htpasswd.New(filename, htpasswd.DefaultSystems, nil)
	if err != nil {
		return nil, err
	}

	// Get initial mtime
	var mtime time.Time
	if stat, err := os.Stat(filename); err == nil {
		mtime = stat.ModTime()
	}

	auth := &BasicAuth{
		File:     htFile,
		Realm:    realm,
		Exclude:  exclude,
		filename: filename,
		mtime:    mtime,
	}

	return auth, nil
}

// CheckAuth checks basic authentication credentials
func (a *BasicAuth) CheckAuth(r *http.Request) bool {
	if a == nil || a.File == nil {
		slog.Debug("Auth check: no auth configured",
			"path", r.URL.Path)
		return true // No auth configured
	}

	username, password, ok := r.BasicAuth()
	if !ok {
		slog.Debug("Auth check: no basic auth credentials",
			"path", r.URL.Path)
		return false
	}

	// Trim whitespace from username to handle malformed htpasswd entries
	username = strings.TrimSpace(username)

	// First attempt: check with current cached file
	a.mu.RLock()
	matched := a.File.Match(username, password)
	currentMtime := a.mtime
	filename := a.filename
	a.mu.RUnlock()

	slog.Debug("Auth check result (cached)",
		"path", r.URL.Path,
		"username", username,
		"matched", matched)

	// If authentication failed, check if htpasswd file has been modified
	if !matched && filename != "" {
		stat, err := os.Stat(filename)
		if err == nil && stat.ModTime().After(currentMtime) {
			slog.Info("htpasswd file modified, reloading",
				"file", filename,
				"old_mtime", currentMtime,
				"new_mtime", stat.ModTime())

			// Upgrade to write lock for reload
			a.mu.Lock()

			// Double-check that another goroutine hasn't already reloaded
			if stat.ModTime().After(a.mtime) {
				// Reload the htpasswd file
				htFile, err := htpasswd.New(filename, htpasswd.DefaultSystems, nil)
				if err != nil {
					slog.Error("Failed to reload htpasswd file",
						"file", filename,
						"error", err)
					a.mu.Unlock()
					return false
				}

				// Update the cached file and mtime
				a.File = htFile
				a.mtime = stat.ModTime()

				slog.Info("htpasswd file reloaded successfully",
					"file", filename,
					"mtime", a.mtime)
			}

			// Retry authentication with the reloaded file
			matched = a.File.Match(username, password)
			a.mu.Unlock()

			slog.Debug("Auth check result (after reload)",
				"path", r.URL.Path,
				"username", username,
				"matched", matched)
		}
	}

	return matched
}

// RequireAuth sends an authentication challenge
func (a *BasicAuth) RequireAuth(w http.ResponseWriter) {
	if a == nil {
		return
	}

	realm := a.Realm
	if realm == "" {
		realm = "Restricted"
	}

	w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, realm))
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}

// ShouldExcludeFromAuth checks if a path should be excluded from authentication
func ShouldExcludeFromAuth(path string, cfg *config.Config) bool {
	// Check simple exclusion paths first (from YAML auth.public_paths)
	for _, excludePath := range cfg.Auth.PublicPaths {
		// Handle glob patterns like *.css
		if strings.HasPrefix(excludePath, "*") {
			if strings.HasSuffix(path, excludePath[1:]) {
				slog.Debug("Auth exclusion: glob pattern match",
					"path", path,
					"pattern", excludePath)
				return true
			}
		} else if strings.Contains(excludePath, "*") {
			// Handle patterns like /path/*.ext
			if matched, _ := filepath.Match(excludePath, path); matched {
				slog.Debug("Auth exclusion: filepath pattern match",
					"path", path,
					"pattern", excludePath)
				return true
			}
		} else {
			// Check for prefix match (paths ending with /)
			if strings.HasSuffix(excludePath, "/") {
				if strings.HasPrefix(path, excludePath) {
					slog.Debug("Auth exclusion: prefix match",
						"path", path,
						"prefix", excludePath)
					return true
				}
			} else {
				// Exact match
				if path == excludePath {
					slog.Debug("Auth exclusion: exact match",
						"path", path,
						"match", excludePath)
					return true
				}
			}
		}
	}

	// Check regex auth patterns from the config file
	for _, authPattern := range cfg.Auth.AuthPatterns {
		if authPattern.Pattern.MatchString(path) && authPattern.Action == "off" {
			slog.Debug("Auth exclusion: regex pattern match",
				"path", path,
				"pattern", authPattern.Pattern.String(),
				"action", authPattern.Action)
			return true
		}
	}

	// Location-specific auth patterns removed - use Routes.ReverseProxies instead

	slog.Debug("Auth required: no exclusion matched",
		"path", path)
	return false
}

// IsEnabled checks if authentication is enabled
func (a *BasicAuth) IsEnabled() bool {
	return a != nil && a.File != nil && a.Realm != "off"
}
