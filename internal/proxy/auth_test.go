package proxy

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadHtpasswd(t *testing.T) {
	tmpDir := t.TempDir()
	htpasswdFile := filepath.Join(tmpDir, "htpasswd")

	// Create a test htpasswd file with different hash types
	htpasswdContent := `user1:$apr1$salt123$somehashedpassword
user2:$2y$10$somehashedpassword
user3:{SHA}somehashedpassword
user4:somecryptpassword
# This is a comment
invalidline
`

	if err := os.WriteFile(htpasswdFile, []byte(htpasswdContent), 0644); err != nil {
		t.Fatalf("Failed to create htpasswd file: %v", err)
	}

	// Load the htpasswd file
	auth, err := LoadHtpasswd(htpasswdFile)
	if err != nil {
		t.Fatalf("Failed to load htpasswd file: %v", err)
	}

	if auth == nil {
		t.Fatal("Expected auth object, got nil")
	}

	// Check that users were loaded (exact number depends on implementation)
	// We mainly want to ensure the file was parsed without error
}

func TestLoadHtpasswdNonexistent(t *testing.T) {
	_, err := LoadHtpasswd("/nonexistent/file")
	if err == nil {
		t.Error("Expected error for nonexistent htpasswd file")
	}
}

func TestHtpasswdAuthRequireAuth(t *testing.T) {
	tmpDir := t.TempDir()
	htpasswdFile := filepath.Join(tmpDir, "htpasswd")

	// Create minimal htpasswd file
	if err := os.WriteFile(htpasswdFile, []byte("user1:password"), 0644); err != nil {
		t.Fatalf("Failed to create htpasswd file: %v", err)
	}

	auth, err := LoadHtpasswd(htpasswdFile)
	if err != nil {
		t.Fatalf("Failed to load htpasswd: %v", err)
	}

	// Create test request and response
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	// RequireAuth should set WWW-Authenticate header and 401 status
	auth.RequireAuth(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	authHeader := rec.Header().Get("WWW-Authenticate")
	if !strings.Contains(authHeader, "Basic") {
		t.Errorf("Expected WWW-Authenticate header with Basic, got: %s", authHeader)
	}
}

func TestHtpasswdAuthenticateNoHeader(t *testing.T) {
	tmpDir := t.TempDir()
	htpasswdFile := filepath.Join(tmpDir, "htpasswd")

	// Create minimal htpasswd file
	if err := os.WriteFile(htpasswdFile, []byte("user1:password"), 0644); err != nil {
		t.Fatalf("Failed to create htpasswd file: %v", err)
	}

	auth, err := LoadHtpasswd(htpasswdFile)
	if err != nil {
		t.Fatalf("Failed to load htpasswd: %v", err)
	}

	// Request without Authorization header
	req := httptest.NewRequest("GET", "/", nil)

	if auth.Authenticate(req) {
		t.Error("Expected authentication to fail without Authorization header")
	}
}

func TestHtpasswdAuthenticateInvalidHeader(t *testing.T) {
	tmpDir := t.TempDir()
	htpasswdFile := filepath.Join(tmpDir, "htpasswd")

	if err := os.WriteFile(htpasswdFile, []byte("user1:password"), 0644); err != nil {
		t.Fatalf("Failed to create htpasswd file: %v", err)
	}

	auth, err := LoadHtpasswd(htpasswdFile)
	if err != nil {
		t.Fatalf("Failed to load htpasswd: %v", err)
	}

	// Test various invalid headers
	invalidHeaders := []string{
		"Bearer token123",           // Wrong auth type
		"Basic",                     // Missing credentials
		"Basic invalidbase64",       // Invalid base64
		"Basic dXNlcjE=",           // Valid base64 but missing colon (user1)
	}

	for _, header := range invalidHeaders {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", header)

		if auth.Authenticate(req) {
			t.Errorf("Expected authentication to fail for header: %s", header)
		}
	}
}

func TestIsPublicPath(t *testing.T) {
	// Create router with minimal config for testing
	router := &Router{
		URLPrefix: "/showcase",
	}

	publicPaths := []string{
		"/assets/app.css",
		"/assets/app.js", 
		"/packs/application.js",
		"/favicon.ico",
		"/image.png",
		"/logo.jpg",
		"/studios/",
		"/studios",
		"/cable",
		"/public/something.txt",
		"/password/reset",
	}

	privatePaths := []string{
		"/admin/users",
		"/2025/raleigh/disney/admin",
		"/private/data",
		"/api/secret",
	}

	for _, path := range publicPaths {
		if !router.isPublicPath(path) {
			t.Errorf("Expected path '%s' to be public", path)
		}
	}

	for _, path := range privatePaths {
		if router.isPublicPath(path) {
			t.Errorf("Expected path '%s' to be private", path)
		}
	}
}

func TestIsAssetPath(t *testing.T) {
	router := &Router{}

	assetPaths := []string{
		"/assets/application.css",
		"/assets/app.js",
		"/packs/application.js",
		"/styles.css",
		"/script.js",
		"/image.png",
		"/photo.jpg",
		"/favicon.ico",
	}

	nonAssetPaths := []string{
		"/admin",
		"/users/profile",
		"/api/data",
		"/home",
	}

	for _, path := range assetPaths {
		if !router.isAssetPath(path) {
			t.Errorf("Expected path '%s' to be an asset", path)
		}
	}

	for _, path := range nonAssetPaths {
		if router.isAssetPath(path) {
			t.Errorf("Expected path '%s' to not be an asset", path)
		}
	}
}

func TestIsCacheableAsset(t *testing.T) {
	router := &Router{}

	cacheablePaths := []string{
		"/assets/app.css",
		"/assets/app.js",
		"/packs/app.js", 
		"/favicon.ico",
		"/logo.png",
		"/photo.jpg",
		"/icon.gif",
		"/drawing.svg",
		"/styles.css",
		"/script.js",
	}

	nonCacheablePaths := []string{
		"/api/data",
		"/admin/users",
		"/home",
		"/login",
	}

	for _, path := range cacheablePaths {
		if !router.isCacheableAsset(path) {
			t.Errorf("Expected path '%s' to be cacheable", path)
		}
	}

	for _, path := range nonCacheablePaths {
		if router.isCacheableAsset(path) {
			t.Errorf("Expected path '%s' to not be cacheable", path)
		}
	}
}

func TestIsLongTermAsset(t *testing.T) {
	router := &Router{}

	longTermAssets := []string{
		"/assets/application-abc123.css",
		"/assets/app-def456.js",
		"/assets/image_123.png",
		"/assets/style-v2.css",
	}

	shortTermAssets := []string{
		"/assets/application.css",
		"/assets/app.js",
		"/favicon.ico",
		"/styles.css",
		"/other/file-123.css", // Not in /assets/
	}

	for _, path := range longTermAssets {
		if !router.isLongTermAsset(path) {
			t.Errorf("Expected path '%s' to be long-term cacheable", path)
		}
	}

	for _, path := range shortTermAssets {
		if router.isLongTermAsset(path) {
			t.Errorf("Expected path '%s' to not be long-term cacheable", path)
		}
	}
}