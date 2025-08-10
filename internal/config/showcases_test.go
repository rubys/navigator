package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadShowcases(t *testing.T) {
	// Create a temporary showcases.yml file for testing
	tmpDir := t.TempDir()
	showcasesFile := filepath.Join(tmpDir, "showcases.yml")

	showcasesContent := `"2025":
  test-studio:
    :name: "Test Studio"
    :region: "test-region"
    :events:
      event1:
        :name: "Test Event 1"
        :date: "2025-01-01"
      event2:
        :name: "Test Event 2" 
        :date: "2025-02-01"
`

	if err := os.WriteFile(showcasesFile, []byte(showcasesContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Load the showcases
	showcases, err := LoadShowcases(showcasesFile)
	if err != nil {
		t.Fatalf("Failed to load showcases: %v", err)
	}

	// Verify basic structure
	if showcases == nil {
		t.Fatal("Expected showcases to not be nil")
	}

	if len(showcases.Years) == 0 {
		t.Fatal("Expected years to be loaded")
	}

	// Should have index tenant plus the test tenants
	expectedTenantCount := 3 // index + event1 + event2
	if len(showcases.Tenants) < expectedTenantCount {
		t.Errorf("Expected at least %d tenants, got %d", expectedTenantCount, len(showcases.Tenants))
	}

	// Check that index tenant exists
	indexTenant := showcases.GetTenant("index")
	if indexTenant == nil {
		t.Error("Expected index tenant to exist")
	}

	// Check that multi-event tenants were created
	tenant1 := showcases.GetTenant("2025-test-studio-event1")
	if tenant1 == nil {
		t.Error("Expected 2025-test-studio-event1 tenant to exist")
	} else {
		if tenant1.Scope != "2025/test-studio/event1" {
			t.Errorf("Expected scope '2025/test-studio/event1', got '%s'", tenant1.Scope)
		}
	}
}

func TestGetTenantByPath(t *testing.T) {
	// Create a simple in-memory showcases structure for testing
	showcases := &Showcases{
		Tenants: []*Tenant{
			{
				Label: "index",
				Scope: "",
			},
			{
				Label: "2025-test-studio",
				Scope: "2025/test-studio",
			},
			{
				Label: "2025-test-studio-event1",
				Scope: "2025/test-studio/event1",
			},
		},
	}

	tests := []struct {
		path     string
		expected string
	}{
		{"", "index"},
		{"index", "index"},
		{"2025/test-studio", "2025-test-studio"},
		{"2025/test-studio/", "2025-test-studio"},
		{"2025/test-studio/event1", "2025-test-studio-event1"},
		{"2025/test-studio/event1/", "2025-test-studio-event1"},
		{"2025/test-studio/event1/somepage", "2025-test-studio-event1"},
		{"nonexistent/path", ""},
	}

	for _, test := range tests {
		tenant := showcases.GetTenantByPath(test.path)
		var actualLabel string
		if tenant != nil {
			actualLabel = tenant.Label
		}

		if actualLabel != test.expected {
			t.Errorf("GetTenantByPath(%q): expected %q, got %q",
				test.path, test.expected, actualLabel)
		}
	}
}

func TestGetDatabasePath(t *testing.T) {
	tenant := &Tenant{
		Label: "test-tenant",
	}

	dbRoot := "/test/db"
	expected := "/test/db/test-tenant.sqlite3"

	result := tenant.GetDatabasePath(dbRoot)
	if result != expected {
		t.Errorf("Expected database path %q, got %q", expected, result)
	}
}

func TestGetEnvironment(t *testing.T) {
	tenant := &Tenant{
		Label:  "test-tenant",
		Owner:  "Test Owner",
		Scope:  "test/scope",
		Logo:   "test-logo.png",
		Locale: "en_US",
	}

	env := tenant.GetEnvironment("/rails/root", "/db/path", "/storage/path")

	// Check that required environment variables are present
	expectedVars := map[string]string{
		"RAILS_APP_DB":    "test-tenant",
		"RAILS_APP_OWNER": "Test Owner",
		"RAILS_APP_SCOPE": "test/scope",
		"SHOWCASE_LOGO":   "test-logo.png",
		"RAILS_LOCALE":    "en_US",
		"RAILS_ENV":       "production",
	}

	envMap := make(map[string]string)
	for _, envVar := range env {
		// Split on first = to handle values with =
		parts := []string{}
		if idx := len(envVar); idx > 0 {
			if eqIdx := findFirst(envVar, '='); eqIdx != -1 {
				parts = []string{envVar[:eqIdx], envVar[eqIdx+1:]}
			}
		}
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	for key, expectedValue := range expectedVars {
		if actualValue, exists := envMap[key]; !exists {
			t.Errorf("Expected environment variable %s to be set", key)
		} else if actualValue != expectedValue {
			t.Errorf("Expected %s=%s, got %s=%s", key, expectedValue, key, actualValue)
		}
	}
}

// Helper function to find first occurrence of character
func findFirst(s string, ch rune) int {
	for i, c := range s {
		if c == ch {
			return i
		}
	}
	return -1
}
