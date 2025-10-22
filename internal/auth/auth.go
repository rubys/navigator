package auth

import (
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/rubys/navigator/internal/config"
	"github.com/tg123/go-htpasswd"
)

// BasicAuth represents HTTP basic authentication configuration
type BasicAuth struct {
	File    *htpasswd.File
	Realm   string
	Exclude []string
	mu      sync.RWMutex // Protects concurrent access to File
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

	auth := &BasicAuth{
		File:    htFile,
		Realm:   realm,
		Exclude: exclude,
	}

	return auth, nil
}

// CheckAuth checks basic authentication credentials
func (a *BasicAuth) CheckAuth(r *http.Request) bool {
	// Lock for concurrent access to htpasswd.File
	// The go-htpasswd library may not be thread-safe
	if a != nil {
		a.mu.RLock()
		defer a.mu.RUnlock()
	}

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

	// Use go-htpasswd library to match the password
	matched := a.File.Match(username, password)

	slog.Debug("Auth check result",
		"path", r.URL.Path,
		"username", username,
		"matched", matched)

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
