package auth

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/rubys/navigator/internal/config"
	"github.com/tg123/go-htpasswd"
)

// BasicAuth represents HTTP basic authentication configuration
type BasicAuth struct {
	File    *htpasswd.File
	Realm   string
	Exclude []string
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
	if a == nil || a.File == nil {
		return true // No auth configured
	}

	username, password, ok := r.BasicAuth()
	if !ok {
		return false
	}

	// Use go-htpasswd library to match the password
	return a.File.Match(username, password)
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
				return true
			}
		} else if strings.Contains(excludePath, "*") {
			// Handle patterns like /path/*.ext
			if matched, _ := filepath.Match(excludePath, path); matched {
				return true
			}
		} else {
			// Check for prefix match (paths ending with /)
			if strings.HasSuffix(excludePath, "/") {
				if strings.HasPrefix(path, excludePath) {
					return true
				}
			} else {
				// Exact match
				if path == excludePath {
					return true
				}
			}
		}
	}

	// Check regex auth patterns from the config file
	for _, authPattern := range cfg.Auth.AuthPatterns {
		if authPattern.Pattern.MatchString(path) && authPattern.Action == "off" {
			return true
		}
	}

	// Location-specific auth patterns removed - use Routes.ReverseProxies instead

	return false
}

// IsEnabled checks if authentication is enabled
func (a *BasicAuth) IsEnabled() bool {
	return a != nil && a.File != nil && a.Realm != "off"
}
