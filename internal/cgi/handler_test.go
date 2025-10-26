package cgi

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rubys/navigator/internal/config"
)

func TestNewHandler(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name         string
		cfg          *config.CGIScriptConfig
		createScript bool
		executable   bool
		wantError    bool
	}{
		{
			name: "Valid script",
			cfg: &config.CGIScriptConfig{
				Path:   "/test",
				Script: filepath.Join(tmpDir, "test.sh"),
			},
			createScript: true,
			executable:   true,
			wantError:    false,
		},
		{
			name: "Script not found",
			cfg: &config.CGIScriptConfig{
				Path:   "/test",
				Script: "/nonexistent/script.sh",
			},
			wantError: true,
		},
		{
			name: "Empty script path",
			cfg: &config.CGIScriptConfig{
				Path:   "/test",
				Script: "",
			},
			wantError: true,
		},
		{
			name: "Script is directory",
			cfg: &config.CGIScriptConfig{
				Path:   "/test",
				Script: tmpDir,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create script file if needed
			if tt.createScript {
				content := "#!/bin/sh\necho 'Hello World'\n"
				if err := os.WriteFile(tt.cfg.Script, []byte(content), 0755); err != nil {
					t.Fatalf("Failed to create test script: %v", err)
				}
			}

			handler, err := NewHandler(tt.cfg, nil, nil)

			if tt.wantError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.wantError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tt.wantError && handler == nil {
				t.Error("Expected handler but got nil")
			}
		})
	}
}

func TestHandler_ServeHTTP(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a simple CGI script that returns status and body
	scriptPath := filepath.Join(tmpDir, "test.cgi")
	scriptContent := `#!/bin/sh
echo "Content-Type: text/plain"
echo "Status: 200 OK"
echo ""
echo "Hello from CGI"
echo "Method: $REQUEST_METHOD"
echo "Query: $QUERY_STRING"
`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	cfg := &config.CGIScriptConfig{
		Path:   "/test",
		Script: scriptPath,
		Env: map[string]string{
			"TEST_VAR": "test_value",
		},
	}

	handler, err := NewHandler(cfg, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	tests := []struct {
		name             string
		method           string
		path             string
		query            string
		wantStatus       int
		wantBodyContains []string
	}{
		{
			name:       "GET request",
			method:     "GET",
			path:       "/test",
			query:      "foo=bar",
			wantStatus: 200,
			wantBodyContains: []string{
				"Hello from CGI",
				"Method: GET",
				"Query: foo=bar",
			},
		},
		{
			name:       "POST request",
			method:     "POST",
			path:       "/test",
			wantStatus: 200,
			wantBodyContains: []string{
				"Hello from CGI",
				"Method: POST",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path+"?"+tt.query, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d", rec.Code, tt.wantStatus)
			}

			body := rec.Body.String()
			for _, want := range tt.wantBodyContains {
				if !strings.Contains(body, want) {
					t.Errorf("Body does not contain %q\nBody: %s", want, body)
				}
			}
		})
	}
}

func TestHandler_ReloadConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test config file
	configPath := filepath.Join(tmpDir, "config.yml")
	if err := os.WriteFile(configPath, []byte("test: config\n"), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	// Create a CGI script that modifies the config file
	scriptPath := filepath.Join(tmpDir, "update.cgi")
	scriptContent := `#!/bin/sh
sleep 0.1
echo "updated: true" >> ` + configPath + `
echo "Content-Type: text/plain"
echo ""
echo "Config updated"
`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	reloadTriggered := false
	reloadConfigPath := ""

	cfg := &config.CGIScriptConfig{
		Path:         "/update",
		Script:       scriptPath,
		ReloadConfig: configPath,
	}

	handler, err := NewHandler(
		cfg,
		func() string { return configPath },
		func(path string) {
			reloadTriggered = true
			reloadConfigPath = path
		},
	)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	req := httptest.NewRequest("POST", "/update", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("Status = %d, want 200", rec.Code)
	}

	if !reloadTriggered {
		t.Error("Expected reload to be triggered but it wasn't")
	}

	if reloadConfigPath != configPath {
		t.Errorf("Reload config path = %q, want %q", reloadConfigPath, configPath)
	}
}

func TestHandler_AccessControl(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a simple CGI script
	scriptPath := filepath.Join(tmpDir, "test.cgi")
	scriptContent := `#!/bin/sh
echo "Content-Type: text/plain"
echo ""
echo "Access granted"
`
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	tests := []struct {
		name           string
		allowedUsers   []string
		username       string
		password       string
		provideAuth    bool
		wantStatus     int
		wantBodyPrefix string
	}{
		{
			name:           "No allowed_users list - all authenticated users allowed",
			allowedUsers:   nil,
			username:       "testuser",
			password:       "testpass",
			provideAuth:    true,
			wantStatus:     200,
			wantBodyPrefix: "Access granted",
		},
		{
			name:           "Empty allowed_users list - all authenticated users allowed",
			allowedUsers:   []string{},
			username:       "testuser",
			password:       "testpass",
			provideAuth:    true,
			wantStatus:     200,
			wantBodyPrefix: "Access granted",
		},
		{
			name:           "User in allowed list - access granted",
			allowedUsers:   []string{"alice", "bob", "charlie"},
			username:       "bob",
			password:       "testpass",
			provideAuth:    true,
			wantStatus:     200,
			wantBodyPrefix: "Access granted",
		},
		{
			name:           "User not in allowed list - access denied",
			allowedUsers:   []string{"alice", "bob"},
			username:       "charlie",
			password:       "testpass",
			provideAuth:    true,
			wantStatus:     403,
			wantBodyPrefix: "Forbidden",
		},
		{
			name:           "No credentials with allowed_users - unauthorized",
			allowedUsers:   []string{"alice"},
			username:       "",
			password:       "",
			provideAuth:    false,
			wantStatus:     401,
			wantBodyPrefix: "Unauthorized",
		},
		{
			name:           "First user in list allowed",
			allowedUsers:   []string{"alice", "bob"},
			username:       "alice",
			password:       "testpass",
			provideAuth:    true,
			wantStatus:     200,
			wantBodyPrefix: "Access granted",
		},
		{
			name:           "Last user in list allowed",
			allowedUsers:   []string{"alice", "bob", "charlie"},
			username:       "charlie",
			password:       "testpass",
			provideAuth:    true,
			wantStatus:     200,
			wantBodyPrefix: "Access granted",
		},
		{
			name:           "Case sensitive username check - denied",
			allowedUsers:   []string{"alice"},
			username:       "Alice",
			password:       "testpass",
			provideAuth:    true,
			wantStatus:     403,
			wantBodyPrefix: "Forbidden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.CGIScriptConfig{
				Path:         "/test",
				Script:       scriptPath,
				AllowedUsers: tt.allowedUsers,
			}

			handler, err := NewHandler(cfg, nil, nil)
			if err != nil {
				t.Fatalf("Failed to create handler: %v", err)
			}

			req := httptest.NewRequest("GET", "/test", nil)
			if tt.provideAuth {
				req.SetBasicAuth(tt.username, tt.password)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d", rec.Code, tt.wantStatus)
			}

			body := strings.TrimSpace(rec.Body.String())
			if !strings.HasPrefix(body, tt.wantBodyPrefix) {
				t.Errorf("Body = %q, want prefix %q", body, tt.wantBodyPrefix)
			}
		})
	}
}

func TestHandler_IsUserAllowed(t *testing.T) {
	tests := []struct {
		name         string
		allowedUsers []string
		username     string
		wantAllowed  bool
	}{
		{
			name:         "No allowed_users list",
			allowedUsers: nil,
			username:     "anyone",
			wantAllowed:  true,
		},
		{
			name:         "Empty allowed_users list",
			allowedUsers: []string{},
			username:     "anyone",
			wantAllowed:  true,
		},
		{
			name:         "User in allowed list",
			allowedUsers: []string{"alice", "bob"},
			username:     "alice",
			wantAllowed:  true,
		},
		{
			name:         "User not in allowed list",
			allowedUsers: []string{"alice", "bob"},
			username:     "charlie",
			wantAllowed:  false,
		},
		{
			name:         "Single user allowed list - match",
			allowedUsers: []string{"alice"},
			username:     "alice",
			wantAllowed:  true,
		},
		{
			name:         "Single user allowed list - no match",
			allowedUsers: []string{"alice"},
			username:     "bob",
			wantAllowed:  false,
		},
		{
			name:         "Case sensitive check",
			allowedUsers: []string{"alice"},
			username:     "Alice",
			wantAllowed:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{
				AllowedUsers: tt.allowedUsers,
			}

			got := h.IsUserAllowed(tt.username)
			if got != tt.wantAllowed {
				t.Errorf("IsUserAllowed(%q) = %v, want %v", tt.username, got, tt.wantAllowed)
			}
		})
	}
}
