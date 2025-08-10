package proxy

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/tg123/go-htpasswd"
)

// HtpasswdAuth handles HTTP Basic authentication using htpasswd file
type HtpasswdAuth struct {
	htpasswd *htpasswd.File
	realm    string
	mu       sync.RWMutex
}

// LoadHtpasswd loads an htpasswd file
func LoadHtpasswd(path string) (*HtpasswdAuth, error) {
	htpasswdFile, err := htpasswd.New(path, htpasswd.DefaultSystems, nil)
	if err != nil {
		return nil, err
	}

	auth := &HtpasswdAuth{
		htpasswd: htpasswdFile,
		realm:    "Showcase",
	}

	return auth, nil
}

// Authenticate checks if the request has valid credentials
func (h *HtpasswdAuth) Authenticate(r *http.Request) bool {
	// Get Authorization header
	auth := r.Header.Get("Authorization")
	if auth == "" {
		log.Printf("No Authorization header found")
		return false
	}

	// Check if it's Basic auth
	if !strings.HasPrefix(auth, "Basic ") {
		authPreview := auth
		if len(auth) > 10 {
			authPreview = auth[:10]
		}
		log.Printf("Not Basic auth: %s", authPreview)
		return false
	}

	// Decode credentials
	encoded := strings.TrimPrefix(auth, "Basic ")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		log.Printf("Failed to decode Basic auth: %v", err)
		return false
	}

	// Split username and password
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		log.Printf("Invalid credential format")
		return false
	}

	username := parts[0]
	password := parts[1]
	log.Printf("Attempting authentication for user: %s", username)

	// Check credentials
	result := h.CheckPassword(username, password)
	log.Printf("Authentication result for user %s: %v", username, result)
	return result
}

// CheckPassword verifies a username/password combination
func (h *HtpasswdAuth) CheckPassword(username, password string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	log.Printf("Checking password for user: %s", username)

	// Use go-htpasswd library for verification
	match := h.htpasswd.Match(username, password)
	log.Printf("Password verification result for user %s: %v", username, match)

	return match
}

// RequireAuth sends a 401 response requesting authentication
func (h *HtpasswdAuth) RequireAuth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, h.realm))
	http.Error(w, "Authorization Required", http.StatusUnauthorized)
}

// SetRealm sets the authentication realm
func (h *HtpasswdAuth) SetRealm(realm string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.realm = realm
}

// ListUsers returns all usernames
func (h *HtpasswdAuth) ListUsers() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// go-htpasswd doesn't provide a direct method to list users
	// For now, return an empty slice or implement file parsing if needed
	// The main functionality (authentication) works without this
	return []string{}
}
