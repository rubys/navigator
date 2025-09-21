package utils

import (
	"os"
	"strings"
	"testing"

	"github.com/rubys/navigator/internal/config"
)

func TestGenerateRequestID(t *testing.T) {
	// Test that GenerateRequestID returns non-empty string
	id := GenerateRequestID()
	if id == "" {
		t.Error("GenerateRequestID should return non-empty string")
	}

	// Test that multiple calls return different IDs
	id2 := GenerateRequestID()
	if id == id2 {
		t.Error("GenerateRequestID should return different IDs on multiple calls")
	}

	// Test ID length (should be reasonable length)
	if len(id) < 8 {
		t.Errorf("GenerateRequestID returned short ID: %q (length %d)", id, len(id))
	}
}

func TestExtractTenantName(t *testing.T) {
	tests := []struct {
		path     string
		expected string
		description string
	}{
		{"/2025/boston/users", "boston-users", "Standard tenant path - skips year, joins tenant parts"},
		{"/dashboard/admin", "admin", "Single segment tenant - not enough parts"},
		{"/", "", "Root path"},
		{"", "", "Empty path"},
		{"/single", "", "Single path segment - not enough parts"},
		{"/app/tenant/deep/path", "tenant-deep", "Deep path with tenant - skips first, joins next two"},
		{"/showcase/2025/livermore/district", "livermore-district", "Showcase pattern"},
		{"/2025/adelaide", "adelaide", "Year and single tenant"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			result := ExtractTenantName(tt.path)
			if result != tt.expected {
				t.Errorf("ExtractTenantName(%q) = %q, expected %q",
					tt.path, result, tt.expected)
			}
		})
	}
}

func TestWritePIDFile(t *testing.T) {
	// Test with temporary file
	tmpFile := "/tmp/test-navigator.pid"
	defer os.Remove(tmpFile)

	err := WritePIDFile(tmpFile)
	if err != nil {
		t.Errorf("WritePIDFile should not error: %v", err)
	}

	// Check file exists
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Error("PID file should exist after WritePIDFile")
	}
}

func TestRemovePIDFile(t *testing.T) {
	// Create a test PID file
	tmpFile := "/tmp/test-navigator-remove.pid"
	err := os.WriteFile(tmpFile, []byte("12345"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test PID file: %v", err)
	}

	// Test removing it
	RemovePIDFile(tmpFile)

	// Check file is gone
	if _, err := os.Stat(tmpFile); !os.IsNotExist(err) {
		t.Error("PID file should be removed after RemovePIDFile")
	}

	// Test removing non-existent file (should not panic)
	RemovePIDFile("/non/existent/file.pid")
}

func TestGetDefaultMaintenancePage(t *testing.T) {
	page := GetDefaultMaintenancePage()

	if page == "" {
		t.Error("GetDefaultMaintenancePage should return non-empty string")
	}

	// Should contain HTML
	if !strings.Contains(page, "<html>") {
		t.Error("Maintenance page should contain HTML")
	}

	// Should contain 503 status info
	if !strings.Contains(page, "503") {
		t.Error("Maintenance page should mention 503 status")
	}
}

func TestGetPidFilePath(t *testing.T) {
	tests := []struct {
		name     string
		tenant   *config.Tenant
		expected string
	}{
		{
			name:     "Nil tenant",
			tenant:   nil,
			expected: "",
		},
		{
			name: "Tenant with PIDFILE env var",
			tenant: &config.Tenant{
				Name: "test-app",
				Root: "/tmp/test",
				Env:  map[string]string{"PIDFILE": "pids/test-app.pid"},
			},
			expected: "pids/test-app.pid",
		},
		{
			name: "Tenant without PIDFILE env var",
			tenant: &config.Tenant{
				Name: "test-app",
				Root: "/tmp/test",
				Env:  map[string]string{"OTHER": "value"},
			},
			expected: "",
		},
		{
			name: "Tenant with nil Env",
			tenant: &config.Tenant{
				Name: "test-app",
				Root: "/tmp/test",
				Env:  nil,
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPidFilePath(tt.tenant)
			if result != tt.expected {
				t.Errorf("GetPidFilePath() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func BenchmarkGenerateRequestID(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateRequestID()
	}
}

func BenchmarkExtractTenantName(b *testing.B) {
	path := "/2025/boston/users/123/profile"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExtractTenantName(path)
	}
}

func TestRequestIDUniqueness(t *testing.T) {
	// Generate multiple IDs and check for uniqueness
	ids := make(map[string]bool)
	numIDs := 1000

	for i := 0; i < numIDs; i++ {
		id := GenerateRequestID()
		if ids[id] {
			t.Errorf("Duplicate request ID generated: %q", id)
		}
		ids[id] = true
	}

	if len(ids) != numIDs {
		t.Errorf("Expected %d unique IDs, got %d", numIDs, len(ids))
	}
}